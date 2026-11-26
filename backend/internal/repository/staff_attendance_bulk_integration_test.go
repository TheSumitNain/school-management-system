package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"school-management-system/internal/schooltime"
	"github.com/jackc/pgx/v5/pgxpool"
)

type staffAttendanceIntegrationFixture struct {
	pool *pgxpool.Pool
	repo *StaffAttendanceRepository
	ctx context.Context
	schoolID int64
	actorID int64
	staffIDs []int64
}

func newStaffAttendanceIntegrationFixture(t *testing.T) *staffAttendanceIntegrationFixture {
	t.Helper()
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	var actorID, schoolID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, userID).Scan(&actorID, &schoolID); err != nil { t.Fatalf("load attendance actor: %v", err) }
	platformAdminID := ensurePlatformAdminLifecycleActor(t, pool)
	var roleID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM roles WHERE school_id=$1 AND role_key='school_admin' AND principal_type='staff' AND scope_type='school' AND is_system=TRUE`, schoolID).Scan(&roleID); err != nil { t.Fatalf("load School Admin role: %v", err) }
	if _, err := pool.Exec(ctx, `INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, actorID, roleID, platformAdminID); err != nil { t.Fatalf("grant School Admin role: %v", err) }
	suffix := fmt.Sprint(time.Now().UnixNano())
	if _, err := pool.Exec(ctx, `INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number) VALUES($1,$2,$3,'Attendance Admin','Administrator','9876543210')`, actorID, schoolID, "ATT-ADMIN-"+suffix); err != nil { t.Fatalf("create attendance actor profile: %v", err) }
	f := &staffAttendanceIntegrationFixture{pool: pool, repo: NewStaffAttendanceRepository(pool), ctx: ctx, schoolID: schoolID, actorID: actorID}
	for i := 0; i < 2; i++ {
		var staffID int64
		username := fmt.Sprintf("att-%06d-%d", time.Now().UnixNano()%1000000, i)
		if err := pool.QueryRow(ctx, `INSERT INTO users(school_id,username,password_hash,status) VALUES($1,$2,'integration-password-hash','active') RETURNING principal_id`, schoolID, username).Scan(&staffID); err != nil { t.Fatalf("create attendance staff account: %v", err) }
		if _, err := pool.Exec(ctx, `INSERT INTO staff_profiles(principal_id,school_id,employee_code,full_name,designation,mobile_number) VALUES($1,$2,$3,$4,'Teacher','9876543211')`, staffID, schoolID, fmt.Sprintf("ATT-%d-%d", time.Now().UnixNano(), i), fmt.Sprintf("Attendance Staff %d", i)); err != nil { t.Fatalf("create attendance staff profile: %v", err) }
		f.staffIDs = append(f.staffIDs, staffID)
	}
	return f
}

func TestStaffAttendanceBulkMarkIsAtomicAndAuditedPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil { t.Fatalf("parse school date: %v", err) }
	created, err := f.repo.MarkBulk(f.ctx, f.actorID, f.schoolID, f.staffIDs, day, "present", nil)
	if err != nil { t.Fatalf("bulk attendance mark: %v", err) }
	if created != len(f.staffIDs) { t.Fatalf("created=%d want=%d", created, len(f.staffIDs)) }
	var records, audits int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=ANY($2::bigint[]) AND attendance_date=$3::date`, f.schoolID, f.staffIDs, schooltime.TodayDate()).Scan(&records); err != nil { t.Fatalf("count bulk attendance records: %v", err) }
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.staff_attendance.marked' AND metadata->>'bulk'='true' AND metadata->>'attendance_date'=$3`, f.schoolID, f.actorID, schooltime.TodayDate()).Scan(&audits); err != nil { t.Fatalf("count bulk attendance audit events: %v", err) }
	if records != len(f.staffIDs) || audits != len(f.staffIDs) { t.Fatalf("bulk records=%d audits=%d want=%d each", records, audits, len(f.staffIDs)) }
}

func TestStaffAttendanceBulkMarkRollsBackWhenAuditFailsPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil { t.Fatalf("parse school date: %v", err) }
	triggerName := fmt.Sprintf("reject_attendance_audit_%d", time.Now().UnixNano())
	functionName := triggerName + "_fn"
	if _, err := f.pool.Exec(f.ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $attendance_audit_failure$
		BEGIN
			IF NEW.action = 'school.staff_attendance.marked' THEN RAISE EXCEPTION 'forced attendance audit failure'; END IF;
			RETURN NEW;
		END;
		$attendance_audit_failure$;
	`, functionName)); err != nil { t.Fatalf("create attendance audit failure function: %v", err) }
	if _, err := f.pool.Exec(f.ctx, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION %s()", triggerName, functionName)); err != nil { t.Fatalf("create attendance audit failure trigger: %v", err) }
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_events", triggerName))
		_, _ = f.pool.Exec(context.Background(), fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
	})
	_, err = f.repo.MarkBulk(f.ctx, f.actorID, f.schoolID, f.staffIDs, day, "present", nil)
	if err == nil { t.Fatal("bulk attendance mark succeeded despite forced audit failure") }
	var records int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=ANY($2::bigint[]) AND attendance_date=$3::date`, f.schoolID, f.staffIDs, schooltime.TodayDate()).Scan(&records); err != nil { t.Fatalf("count records after failed bulk mark: %v", err) }
	if records != 0 { t.Fatalf("records after failed audit=%d want=0", records) }
}

func TestStaffAttendanceBulkMarkRejectsDuplicateTargetsBeforeWritesPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil { t.Fatalf("parse school date: %v", err) }
	_, err = f.repo.MarkBulk(f.ctx, f.actorID, f.schoolID, []int64{f.staffIDs[0], f.staffIDs[0]}, day, "present", nil)
	if !errors.Is(err, ErrInvalidStaffAttendance) { t.Fatalf("duplicate target error=%v want ErrInvalidStaffAttendance", err) }
	var records int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND attendance_date=$2::date`, f.schoolID, schooltime.TodayDate()).Scan(&records); err != nil { t.Fatalf("count records after duplicate-target rejection: %v", err) }
	if records != 0 { t.Fatalf("records after duplicate-target rejection=%d want=0", records) }
}

func TestStaffAttendanceBulkMarkRollsBackWhenAnyTargetIsInvalidPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil { t.Fatalf("parse school date: %v", err) }

	_, err = f.repo.MarkBulk(f.ctx, f.actorID, f.schoolID, []int64{f.staffIDs[0], 9_999_999_999}, day, "present", nil)
	if !errors.Is(err, ErrStaffAttendanceNotFound) {
		t.Fatalf("invalid target error=%v want ErrStaffAttendanceNotFound", err)
	}
	var records int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND attendance_date=$2::date`, f.schoolID, schooltime.TodayDate()).Scan(&records); err != nil {
		t.Fatalf("count records after invalid-target rollback: %v", err)
	}
	if records != 0 { t.Fatalf("records after invalid-target rollback=%d want=0", records) }
}

func TestStaffAttendanceConcurrentBulkMarksSerializeThroughUniqueConstraintPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil { t.Fatalf("parse school date: %v", err) }

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, markErr := f.repo.MarkBulk(f.ctx, f.actorID, f.schoolID, f.staffIDs, day, "present", nil)
			errs <- markErr
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes, conflicts := 0, 0
	for markErr := range errs {
		switch {
		case markErr == nil:
			successes++
		case errors.Is(markErr, ErrStaffAttendanceConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent bulk mark error: %v", markErr)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent results successes=%d conflicts=%d; want 1 each", successes, conflicts)
	}

	var records, audits int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=ANY($2::bigint[]) AND attendance_date=$3::date`, f.schoolID, f.staffIDs, schooltime.TodayDate()).Scan(&records); err != nil {
		t.Fatalf("count records after concurrent bulk marks: %v", err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.staff_attendance.marked' AND metadata->>'bulk'='true' AND metadata->>'attendance_date'=$3`, f.schoolID, f.actorID, schooltime.TodayDate()).Scan(&audits); err != nil {
		t.Fatalf("count audits after concurrent bulk marks: %v", err)
	}
	if records != len(f.staffIDs) || audits != len(f.staffIDs) {
		t.Fatalf("records=%d audits=%d want=%d each; losing transaction must leave no partial rows", records, audits, len(f.staffIDs))
	}
}

func installStaffAttendanceAuditFailure(t *testing.T, f *staffAttendanceIntegrationFixture, action string) {
	t.Helper()
	triggerName := fmt.Sprintf("reject_attendance_audit_%d", time.Now().UnixNano())
	functionName := triggerName + "_fn"
	if _, err := f.pool.Exec(f.ctx, fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $attendance_audit_failure$
		BEGIN
			IF NEW.action = %q THEN RAISE EXCEPTION 'forced attendance audit failure'; END IF;
			RETURN NEW;
		END;
		$attendance_audit_failure$;
	`, functionName, action)); err != nil {
		t.Fatalf("create attendance audit failure function for %s: %v", action, err)
	}
	if _, err := f.pool.Exec(f.ctx, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION %s()", triggerName, functionName)); err != nil {
		t.Fatalf("create attendance audit failure trigger for %s: %v", action, err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_events", triggerName))
		_, _ = f.pool.Exec(context.Background(), fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
	})
}

func TestStaffAttendanceSingleMarkRollsBackWhenAuditFailsPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil {
		t.Fatalf("parse school date: %v", err)
	}
	installStaffAttendanceAuditFailure(t, f, "school.staff_attendance.marked")

	_, err = f.repo.Mark(f.ctx, f.actorID, f.schoolID, f.staffIDs[0], day, "present", nil)
	if err == nil {
		t.Fatal("single attendance mark succeeded despite forced audit failure")
	}
	var records, audits int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=$2 AND attendance_date=$3::date`, f.schoolID, f.staffIDs[0], schooltime.TodayDate()).Scan(&records); err != nil {
		t.Fatalf("count attendance rows after failed mark: %v", err)
	}
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.staff_attendance.marked' AND resource_type='staff_attendance_record'`, f.schoolID, f.actorID).Scan(&audits); err != nil {
		t.Fatalf("count attendance audits after failed mark: %v", err)
	}
	if records != 0 || audits != 0 {
		t.Fatalf("failed single mark left records=%d audits=%d; want both zero", records, audits)
	}
}

func TestStaffAttendanceUpdateRollsBackWhenAuditFailsPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil {
		t.Fatalf("parse school date: %v", err)
	}
	remarks := "original attendance remark"
	created, err := f.repo.Mark(f.ctx, f.actorID, f.schoolID, f.staffIDs[0], day, "present", &remarks)
	if err != nil {
		t.Fatalf("create baseline attendance record: %v", err)
	}
	if created == nil || created.ID == nil {
		t.Fatal("baseline attendance mark returned no record ID")
	}
	installStaffAttendanceAuditFailure(t, f, "school.staff_attendance.updated")

	updatedRemarks := "attempted correction"
	_, err = f.repo.Update(f.ctx, f.actorID, f.schoolID, *created.ID, "absent", &updatedRemarks)
	if err == nil {
		t.Fatal("attendance update succeeded despite forced audit failure")
	}
	var status string
	var persistedRemarks *string
	if err := f.pool.QueryRow(f.ctx, `SELECT status, remarks FROM staff_attendance_records WHERE id=$1 AND school_id=$2`, *created.ID, f.schoolID).Scan(&status, &persistedRemarks); err != nil {
		t.Fatalf("read attendance row after failed update: %v", err)
	}
	if status != "present" || persistedRemarks == nil || *persistedRemarks != remarks {
		t.Fatalf("failed update changed attendance row: status=%q remarks=%v; want status=present and original remark", status, persistedRemarks)
	}
	var audits int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.staff_attendance.updated' AND resource_id=$3`, f.schoolID, f.actorID, fmt.Sprint(*created.ID)).Scan(&audits); err != nil {
		t.Fatalf("count update audit events after failed update: %v", err)
	}
	if audits != 0 {
		t.Fatalf("failed update wrote %d audit events; want zero", audits)
	}
}

