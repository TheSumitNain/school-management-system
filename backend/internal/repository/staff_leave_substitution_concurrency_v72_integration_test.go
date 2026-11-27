package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

)

// TestStaffLeaveApprovalAndSubstitutionAssignmentShareSlotLockPostgresIntegration
// holds the real V60 slot key, starts both writes independently, then releases
// them together. Exactly one operation may commit: approved leave or cover.
func TestStaffLeaveApprovalAndSubstitutionAssignmentShareSlotLockPostgresIntegration(t *testing.T) {
	f := newStaffLeaveIntegrationFixture(t)
	ctx, cancel := context.WithTimeout(f.ctx, 45*time.Second)
	defer cancel()

	nowIST := time.Now().In(time.FixedZone("IST", 5*60*60+30*60))
	daysUntilMonday := (int(time.Monday) - int(nowIST.Weekday()) + 7) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	coverDate := nowIST.AddDate(0, 0, daysUntilMonday+14).Format("2006-01-02")
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var teacherRoleID int64
	if err := f.pool.QueryRow(ctx, `SELECT id FROM roles WHERE school_id=$1
		AND principal_type='staff' AND scope_type='school' AND role_key='teacher' AND is_system=TRUE`,
		f.schoolID).Scan(&teacherRoleID); err != nil {
		t.Fatalf("load Teacher role: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, f.staffPrincipalID, teacherRoleID, f.adminPrincipalID); err != nil {
		t.Fatalf("assign Teacher role to substitute: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE staff_profiles SET staff_type='teaching'
		WHERE principal_id=$1 AND school_id=$2`, f.staffPrincipalID, f.schoolID); err != nil {
		t.Fatalf("classify substitute as Teaching staff: %v", err)
	}

	var originalID int64
	if err := f.pool.QueryRow(ctx, `INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,'integration-password-hash','active') RETURNING principal_id`,
		f.schoolID, "lc-orig-"+suffix).Scan(&originalID); err != nil {
		t.Fatalf("create original teacher account: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO staff_profiles
		(principal_id,school_id,employee_code,full_name,designation,mobile_number,staff_type)
		VALUES($1,$2,$3,'Leave Cover Original','Teacher','9876543212','teaching')`,
		originalID, f.schoolID, "LC-ORIG-"+suffix); err != nil {
		t.Fatalf("create original teacher profile: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, originalID, teacherRoleID, f.adminPrincipalID); err != nil {
		t.Fatalf("assign Teacher role to original teacher: %v", err)
	}

	var yearID, classID, sectionID, subjectID, periodID, assignmentID, entryID int64
	if err := f.pool.QueryRow(ctx, `INSERT INTO academic_years
		(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,CURRENT_DATE-30,CURRENT_DATE+365,'active') RETURNING id`,
		f.schoolID, "Leave Substitution Race "+suffix).Scan(&yearID); err != nil {
		t.Fatalf("create academic year: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO classes(school_id,name)
		VALUES($1,$2) RETURNING id`, f.schoolID, "Leave Substitution Race "+suffix).Scan(&classID); err != nil {
		t.Fatalf("create class: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,'A') RETURNING id`, f.schoolID, yearID, classID).Scan(&sectionID); err != nil {
		t.Fatalf("create section: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO subjects(school_id,name)
		VALUES($1,$2) RETURNING id`, f.schoolID, "Leave Substitution Race "+suffix).Scan(&subjectID); err != nil {
		t.Fatalf("create subject: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO timetable_periods
		(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Leave Race Period',TIME '09:00',TIME '09:40') RETURNING id`,
		f.schoolID, yearID).Scan(&periodID); err != nil {
		t.Fatalf("create timetable period: %v", err)
	}
	for _, teacherID := range []int64{originalID, f.staffPrincipalID} {
		if _, err := f.pool.Exec(ctx, `INSERT INTO teacher_subject_assignments
			(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
			VALUES($1,$2,$3,$4,$5,$6)`, f.schoolID, yearID, sectionID, subjectID, teacherID, f.adminPrincipalID); err != nil {
			t.Fatalf("create matching teacher-subject assignment for %d: %v", teacherID, err)
		}
	}
	if err := f.pool.QueryRow(ctx, `SELECT id FROM teacher_subject_assignments
		WHERE school_id=$1 AND academic_year_id=$2 AND section_id=$3 AND subject_id=$4
		AND teacher_principal_id=$5 AND revoked_at IS NULL`,
		f.schoolID, yearID, sectionID, subjectID, originalID).Scan(&assignmentID); err != nil {
		t.Fatalf("load original teacher assignment: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO timetable_entries
		(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,
		 teacher_subject_assignment_id,period_id,day_of_week,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,1,$8,$8) RETURNING id`,
		f.schoolID, yearID, sectionID, subjectID, originalID, assignmentID, periodID, f.adminPrincipalID).Scan(&entryID); err != nil {
		t.Fatalf("create recurring timetable entry: %v", err)
	}

	leave := f.request(t, f.staffPrincipalID, coverDate, coverDate)
	lockText := fmt.Sprintf("%d:%d:1:%d", f.schoolID, f.staffPrincipalID, periodID)
	var lockKey int64
	if err := f.pool.QueryRow(ctx, "SELECT hashtextextended($1,0)", lockText).Scan(&lockKey); err != nil {
		t.Fatalf("compute exact V60 substitute slot lock: %v", err)
	}
	barrierConn, err := f.pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire slot-lock barrier connection: %v", err)
	}
	defer barrierConn.Release()
	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatalf("hold exact substitute slot lock: %v", err)
	}
	barrierHeld := true
	defer func() {
		if barrierHeld {
			_, _ = barrierConn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey)
		}
	}()

	type outcome struct {
		operation string
		err       error
	}
	results := make(chan outcome, 2)
	go func() {
		results <- outcome{operation: "leave approval", err: f.repo.Decide(ctx, f.adminPrincipalID, f.schoolID, leave.ID, true, nil)}
	}()
	go func() {
		_, createErr := (&AcademicRepository{db: f.pool}).CreateStaffSubstitution(
			ctx, f.adminPrincipalID, f.schoolID, entryID, originalID, f.staffPrincipalID,
			coverDate, nil, "Race with leave approval",
		)
		results <- outcome{operation: "substitution assignment", err: createErr}
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting int
		if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM pg_locks
			WHERE locktype='advisory' AND NOT granted AND objsubid=1
			AND classid=(($1::bigint >> 32) & 4294967295)::oid
			AND objid=($1::bigint & 4294967295)::oid`, lockKey).Scan(&waiting); err != nil {
			t.Fatalf("inspect waiters on shared substitution slot lock: %v", err)
		}
		if waiting >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("both writes did not wait on the exact shared slot lock; waiting=%d", waiting)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
		t.Fatalf("release shared slot lock: %v", err)
	}
	barrierHeld = false

	successes, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		select {
		case got := <-results:
			switch {
			case got.err == nil:
				successes++
			case errors.Is(got.err, ErrStaffLeaveConflict), errors.Is(got.err, ErrStaffSubstitutionConflict):
				conflicts++
			default:
				t.Fatalf("%s returned unexpected error: %v", got.operation, got.err)
			}
		case <-ctx.Done():
			t.Fatalf("timed out collecting concurrent leave/substitution results: %v", ctx.Err())
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent leave/substitution outcomes successes=%d conflicts=%d; want exactly one success and one conflict", successes, conflicts)
	}

	var approvedLeaves, activeCovers int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM staff_leave_requests
		WHERE id=$1 AND status='approved'`, leave.ID).Scan(&approvedLeaves); err != nil {
		t.Fatalf("verify leave outcome: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM staff_substitutions
		WHERE school_id=$1 AND timetable_entry_id=$2 AND cover_date=$3::date
		AND substitute_teacher_principal_id=$4 AND status='active'`,
		f.schoolID, entryID, coverDate, f.staffPrincipalID).Scan(&activeCovers); err != nil {
		t.Fatalf("verify substitution outcome: %v", err)
	}
	if approvedLeaves+activeCovers != 1 {
		t.Fatalf("race persisted approved leaves=%d and active covers=%d; exactly one must exist", approvedLeaves, activeCovers)
	}

}
