package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestStudentArchiveWinsQueuedAttendanceMarkPostgresIntegration proves that
// when archive is already queued on the student row, a concurrent attendance
// mark cannot slip through after the student has been archived.
func TestStudentArchiveWinsQueuedAttendanceMarkPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var schoolID, actorID int64
	if err := pool.QueryRow(ctx, `
		SELECT u.school_id, u.principal_id FROM users u WHERE u.id=$1
	`, userID).Scan(&schoolID, &actorID); err != nil {
		t.Fatalf("load School Admin integration actor: %v", err)
	}
	var schoolAdminRoleID int64
	if err := pool.QueryRow(ctx, `
		SELECT id FROM roles
		WHERE school_id=$1 AND principal_type='staff' AND scope_type='school'
		  AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin role: %v", err)
	}
	platformAdminID := ensurePlatformAdminLifecycleActor(t, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`, actorID, schoolAdminRoleID, platformAdminID); err != nil {
		t.Fatalf("grant School Admin role to integration actor: %v", err)
	}

	suffix := fmt.Sprint(time.Now().UnixNano())
	var yearID, classID, sectionID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
		VALUES($1,$2,CURRENT_DATE-30,CURRENT_DATE+30,'active',FALSE)
		RETURNING id
	`, schoolID, "Archive Attendance Race Year "+suffix).Scan(&yearID); err != nil {
		t.Fatalf("create archive race academic year: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id
	`, schoolID, "Archive Attendance Race Class "+suffix).Scan(&classID); err != nil {
		t.Fatalf("create archive race class: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,$4) RETURNING id
	`, schoolID, yearID, classID, "Race "+suffix).Scan(&sectionID); err != nil {
		t.Fatalf("create archive race section: %v", err)
	}
	var today time.Time
	if err := pool.QueryRow(ctx, "SELECT CURRENT_DATE").Scan(&today); err != nil {
		t.Fatalf("read database current date: %v", err)
	}
	repo := NewAcademicRepository(pool)
	admission, err := repo.CreateStudentAdmission(ctx, actorID, schoolID,
		"Archive Attendance Race Student "+suffix, "ARCHIVE-RACE-"+suffix,
		yearID, sectionID, nil, today)
	if err != nil || admission == nil {
		t.Fatalf("create archive race student and enrollment: admission=%+v err=%v", admission, err)
	}

	// Hold the permanent student row first. Queue archive before attendance so
	// PostgreSQL's row-lock queue gives archive the first opportunity to commit.
	blockerConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire student-row blocker connection: %v", err)
	}
	defer blockerConn.Release()
	blockerTx, err := blockerConn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin student-row blocker transaction: %v", err)
	}
	defer func() { _ = blockerTx.Rollback(ctx) }()
	var lockedStudentID int64
	if err := blockerTx.QueryRow(ctx, `
		SELECT id FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE
	`, admission.Student.ID, schoolID).Scan(&lockedStudentID); err != nil {
		t.Fatalf("lock student row before competing operations: %v", err)
	}

	archiveResult := make(chan error, 1)
	go func() {
		_, archiveErr := repo.ArchiveStudent(ctx, actorID, schoolID, admission.Student.ID)
		archiveResult <- archiveErr
	}()
	waitForStudentOperationLock(t, ctx, pool, "UPDATE students SET status = 'archived'", archiveResult, "archive")

	attendanceResult := make(chan error, 1)
	go func() {
		_, attendanceErr := repo.MarkAttendance(ctx, actorID, schoolID, admission.Enrollment.ID, today, "present", nil)
		attendanceResult <- attendanceErr
	}()
	waitForStudentOperationLock(t, ctx, pool, "FOR UPDATE OF st", attendanceResult, "attendance mark")

	if err := blockerTx.Commit(ctx); err != nil {
		t.Fatalf("release student row after both operations are queued: %v", err)
	}
	if err := <-archiveResult; err != nil {
		t.Fatalf("archive should win the queued student-row lock: %v", err)
	}
	if err := <-attendanceResult; !errors.Is(err, ErrAttendanceNotFound) {
		t.Fatalf("attendance mark after queued archive error = %v, want ErrAttendanceNotFound", err)
	}

	var studentStatus, enrollmentStatus string
	if err := pool.QueryRow(ctx, `
		SELECT st.status,e.status
		FROM students st
		JOIN student_enrollments e ON e.student_id=st.id AND e.school_id=st.school_id
		WHERE st.id=$1 AND st.school_id=$2 AND e.id=$3
	`, admission.Student.ID, schoolID, admission.Enrollment.ID).Scan(&studentStatus, &enrollmentStatus); err != nil {
		t.Fatalf("verify archived student and withdrawn enrollment: %v", err)
	}
	if studentStatus != "archived" || enrollmentStatus != "withdrawn" {
		t.Fatalf("post-race student/enrollment status = %q/%q; want archived/withdrawn", studentStatus, enrollmentStatus)
	}
	var attendanceCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM attendance_records
		WHERE school_id=$1 AND enrollment_id=$2
	`, schoolID, admission.Enrollment.ID).Scan(&attendanceCount); err != nil {
		t.Fatalf("count attendance records after archive race: %v", err)
	}
	if attendanceCount != 0 {
		t.Fatalf("archive race left %d attendance records; want none", attendanceCount)
	}
}

func waitForStudentOperationLock(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queryFragment string,
	result <-chan error,
	operation string,
) {
	t.Helper()
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE wait_event_type='Lock' AND position($1 in query)>0
			)
		`, queryFragment).Scan(&waiting); err != nil {
			t.Fatalf("observe %s lock wait: %v", operation, err)
		}
		if waiting {
			return
		}
		select {
		case err := <-result:
			t.Fatalf("%s returned before waiting on the held student row: %v", operation, err)
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			t.Fatalf("%s never waited for the held student row: %v", operation, ctx.Err())
		}
	}
}
