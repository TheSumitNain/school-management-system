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
			IF NEW.action = '%s' THEN RAISE EXCEPTION 'forced attendance audit failure'; END IF;
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

func TestStaffAttendanceDailyAndMonthlyReadsAreSchoolScopedPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil {
		t.Fatalf("parse school date: %v", err)
	}
	if _, err := f.repo.Mark(f.ctx, f.actorID, f.schoolID, f.staffIDs[0], day, "present", nil); err != nil {
		t.Fatalf("mark attendance for read verification: %v", err)
	}

	daily, err := f.repo.ListDaily(f.ctx, f.actorID, f.schoolID, day)
	if err != nil {
		t.Fatalf("list daily attendance: %v", err)
	}
	foundDaily := false
	for _, item := range daily {
		if item.StaffPrincipalID == f.staffIDs[0] {
			foundDaily = true
			if item.Status == nil || *item.Status != "present" {
				t.Fatalf("daily attendance status=%v want present", item.Status)
			}
		}
	}
	if !foundDaily {
		t.Fatal("daily attendance omitted the active staff member with a marked record")
	}

    dailyPage1, dailyTotal, err := f.repo.ListDailyPage(f.ctx, f.actorID, f.schoolID, day, 1, 0)
    if err != nil || dailyTotal != 3 || len(dailyPage1) != 1 { t.Fatalf("daily first page rows=%d total=%d err=%v; want 1 and 3",len(dailyPage1),dailyTotal,err) }
    dailyPage2, dailyTotal2, err := f.repo.ListDailyPage(f.ctx, f.actorID, f.schoolID, day, 1, 1)
    if err != nil || dailyTotal2 != dailyTotal || len(dailyPage2) != 1 || dailyPage1[0].StaffPrincipalID == dailyPage2[0].StaffPrincipalID {
        t.Fatalf("daily second page rows=%d total=%d first=%d err=%v; expected distinct row and stable total",len(dailyPage2),dailyTotal2,dailyPage1[0].StaffPrincipalID,err)
    }
    filteredDaily, filteredTotal, err := f.repo.ListDailyFilteredPage(f.ctx, f.actorID, f.schoolID, day,
        StaffAttendanceDailyFilter{Search: "Attendance Staff 1", Status: "unmarked"}, 10, 0)
    if err != nil || filteredTotal != 1 || len(filteredDaily) != 1 ||
        filteredDaily[0].StaffPrincipalID != f.staffIDs[1] || filteredDaily[0].Status != nil {
        t.Fatalf("search + unmarked daily filter rows=%+v total=%d err=%v; want only the matching unmarked staff", filteredDaily, filteredTotal, err)
    }
    markedDaily, markedTotal, err := f.repo.ListDailyFilteredPage(f.ctx, f.actorID, f.schoolID, day,
        StaffAttendanceDailyFilter{Status: "present"}, 10, 0)
    if err != nil || markedTotal != 1 || len(markedDaily) != 1 ||
        markedDaily[0].StaffPrincipalID != f.staffIDs[0] || markedDaily[0].Status == nil || *markedDaily[0].Status != "present" {
        t.Fatalf("present daily filter rows=%+v total=%d err=%v; want only the marked-present staff", markedDaily, markedTotal, err)
    }

	monthly, err := f.repo.ListMonthly(f.ctx, f.actorID, f.schoolID, day)
	if err != nil {
		t.Fatalf("list monthly attendance: %v", err)
	}
	foundMonthly := false
	for _, item := range monthly {
		if item.StaffPrincipalID == f.staffIDs[0] {
			foundMonthly = true
			if item.PresentDays != 1 || item.AbsentDays != 0 {
				t.Fatalf("monthly totals present=%d absent=%d; want 1 and 0", item.PresentDays, item.AbsentDays)
			}
		}
	}
	if !foundMonthly {
		t.Fatal("monthly attendance omitted the active staff member with a marked record")
	}

    monthlyPage1, monthlyTotal, err := f.repo.ListMonthlyPage(f.ctx, f.actorID, f.schoolID, day, 1, 0)
    if err != nil || monthlyTotal != 3 || len(monthlyPage1) != 1 { t.Fatalf("monthly first page rows=%d total=%d err=%v; want 1 and 3",len(monthlyPage1),monthlyTotal,err) }
    monthlyPage2, monthlyTotal2, err := f.repo.ListMonthlyPage(f.ctx, f.actorID, f.schoolID, day, 1, 1)
    if err != nil || monthlyTotal2 != monthlyTotal || len(monthlyPage2) != 1 || monthlyPage1[0].StaffPrincipalID == monthlyPage2[0].StaffPrincipalID {
        t.Fatalf("monthly second page rows=%d total=%d first=%d err=%v; expected distinct row and stable total",len(monthlyPage2),monthlyTotal2,monthlyPage1[0].StaffPrincipalID,err)
    }

	otherSchoolID := f.schoolID + 9_000_000
	if _, err := f.repo.ListDaily(f.ctx, f.actorID, otherSchoolID, day); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("cross-school daily read error=%v want ErrStaffAttendanceAccessDenied", err)
	}
	if _, err := f.repo.ListMonthly(f.ctx, f.actorID, otherSchoolID, day); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("cross-school monthly read error=%v want ErrStaffAttendanceAccessDenied", err)
	}

	// Read isolation alone is insufficient: a caller with legitimate
	// home-school permissions must not forge another school ID on writes.
	if _, err := f.repo.Mark(f.ctx, f.actorID, otherSchoolID, f.staffIDs[1], day, "absent", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("cross-school single mark error=%v want access denied", err)
	}
	if count, err := f.repo.MarkBulk(f.ctx, f.actorID, otherSchoolID, f.staffIDs, day.AddDate(0, 0, -1), "absent", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) || count != 0 {
		t.Fatalf("cross-school bulk mark count=%d error=%v want zero and access denied", count, err)
	}
	var attendanceID int64
	if err := f.pool.QueryRow(f.ctx, `SELECT id FROM staff_attendance_records WHERE school_id=$1 AND staff_principal_id=$2 AND attendance_date=$3::date`, f.schoolID, f.staffIDs[0], day.Format("2006-01-02")).Scan(&attendanceID); err != nil {
		t.Fatalf("load existing attendance for cross-school update: %v", err)
	}
	if _, err := f.repo.Update(f.ctx, f.actorID, otherSchoolID, attendanceID, "absent", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("cross-school attendance update error=%v want access denied", err)
	}
	var forgedWriteCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*) FROM staff_attendance_records WHERE school_id=$1 AND attendance_date=$2::date AND status='absent'`, f.schoolID, day.AddDate(0, 0, -1).Format("2006-01-02")).Scan(&forgedWriteCount); err != nil {
		t.Fatalf("count records after rejected cross-school writes: %v", err)
	}
	if forgedWriteCount != 0 {
		t.Fatalf("cross-school bulk mark created %d attendance records", forgedWriteCount)
	}
}

func TestStaffAttendanceMonthlyRetainsInactiveStaffHistoryPostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil {
		t.Fatalf("parse school date: %v", err)
	}
	if _, err := f.repo.Mark(f.ctx, f.actorID, f.schoolID, f.staffIDs[0], day, "present", nil); err != nil {
		t.Fatalf("mark attendance before staff inactivation: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `UPDATE staff_profiles SET employment_status='inactive' WHERE principal_id=$1 AND school_id=$2`, f.staffIDs[0], f.schoolID); err != nil {
		t.Fatalf("inactivate staff profile after attendance: %v", err)
	}

	daily, err := f.repo.ListDaily(f.ctx, f.actorID, f.schoolID, day)
	if err != nil {
		t.Fatalf("list current daily attendance after inactivation: %v", err)
	}
	for _, item := range daily {
		if item.StaffPrincipalID == f.staffIDs[0] {
			t.Fatal("inactive employee remained in the current daily attendance roster")
		}
	}

	monthly, err := f.repo.ListMonthly(f.ctx, f.actorID, f.schoolID, day)
	if err != nil {
		t.Fatalf("list monthly attendance history after inactivation: %v", err)
	}
	for _, item := range monthly {
		if item.StaffPrincipalID == f.staffIDs[0] {
			if item.PresentDays != 1 {
				t.Fatalf("inactive employee monthly present days=%d want=1", item.PresentDays)
			}
			return
		}
	}
	t.Fatal("monthly attendance omitted inactive employee with retained attendance history")
}



// A School Admin's attendance permissions are bound to the actor's own school.
// Passing a different school ID to any attendance read/write path must not
// bypass that boundary, even when the staff or record ID belongs to the home school.
func TestStaffAttendanceOperationsRejectForeignSchoolScopePostgresIntegration(t *testing.T) {
	f := newStaffAttendanceIntegrationFixture(t)
	suffix := fmt.Sprint(time.Now().UnixNano())
	var foreignSchoolID int64
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO schools(name, code, status)
		VALUES($1, $2, 'active') RETURNING id
	`, "Attendance Isolation School "+suffix, "AI-"+suffix).Scan(&foreignSchoolID); err != nil {
		t.Fatalf("create foreign school fixture: %v", err)
	}
	if foreignSchoolID == f.schoolID {
		t.Fatal("foreign school fixture unexpectedly matches actor school")
	}

	day, err := time.Parse("2006-01-02", schooltime.TodayDate())
	if err != nil {
		t.Fatalf("parse school-local attendance date: %v", err)
	}
	repo := f.repo
	if _, err := repo.ListDaily(f.ctx, f.actorID, foreignSchoolID, day); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("foreign-school daily read error=%v; want access denied", err)
	}
	month := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
	if _, err := repo.ListMonthly(f.ctx, f.actorID, foreignSchoolID, month); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("foreign-school monthly read error=%v; want access denied", err)
	}
	if _, err := repo.Mark(f.ctx, f.actorID, foreignSchoolID, f.staffIDs[0], day, "present", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("foreign-school mark error=%v; want access denied", err)
	}
	if _, err := repo.MarkBulk(f.ctx, f.actorID, foreignSchoolID, f.staffIDs, day, "present", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("foreign-school bulk mark error=%v; want access denied", err)
	}

	// Create a legitimate home-school record, then try to update it while
	// supplying the foreign school ID. The update must be denied as well.
	created, err := repo.Mark(f.ctx, f.actorID, f.schoolID, f.staffIDs[0], day, "present", nil)
	if err != nil {
		t.Fatalf("create home-school attendance fixture: %v", err)
	}
	if created == nil || created.ID == nil {
		t.Fatal("home-school attendance fixture returned no record ID")
	}
	if _, err := repo.Update(f.ctx, f.actorID, foreignSchoolID, *created.ID, "late", nil); !errors.Is(err, ErrStaffAttendanceAccessDenied) {
		t.Fatalf("foreign-school update error=%v; want access denied", err)
	}

	var foreignRecords int
	if err := f.pool.QueryRow(f.ctx, `
		SELECT COUNT(*) FROM staff_attendance_records
		WHERE school_id=$1 AND attendance_date=$2::date
	`, foreignSchoolID, schooltime.TodayDate()).Scan(&foreignRecords); err != nil {
		t.Fatalf("count foreign-school attendance records: %v", err)
	}
	if foreignRecords != 0 {
		t.Fatalf("foreign-school attendance records=%d after denied operations; want 0", foreignRecords)
	}
}
