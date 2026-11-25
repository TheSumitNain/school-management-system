package repository

import (
	"context"
	"errors"
	"fmt"
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
		username := fmt.Sprintf("attendance-staff-%d-%d", time.Now().UnixNano(), i)
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
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON audit_events; DROP FUNCTION IF EXISTS %s();", triggerName, functionName)) })
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
