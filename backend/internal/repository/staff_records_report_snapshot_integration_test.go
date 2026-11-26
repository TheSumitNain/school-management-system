package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"
	"school-management-system/internal/schooltime"

	"github.com/jackc/pgx/v5"
)

// Exercise the production pool-backed path: each report must establish its
// own read-only repeatable-read transaction before resolving permissions or
// reading aggregate counts and page rows.
func TestStaffReportsUsePoolBackedReadOnlySnapshotPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin report authorization fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorID, schoolID int64
	if err := tx.QueryRow(ctx, `SELECT principal_id, school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture: %v", err)
	}

	var hasRole bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM principal_roles pr
			JOIN roles r ON r.id=pr.role_id
			WHERE pr.principal_id=$1 AND pr.revoked_at IS NULL
			  AND r.school_id=$2 AND r.role_key='school_admin' AND r.is_system=TRUE
		)
	`, actorID, schoolID).Scan(&hasRole); err != nil {
		t.Fatalf("check School Admin fixture role: %v", err)
	}

	if !hasRole {
		var platformAdminID, schoolAdminRoleID int64
		err = tx.QueryRow(ctx, `
			SELECT ap.id
			FROM platform_admin_accounts pa
			JOIN auth_principals ap ON ap.id=pa.principal_id
			JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id=pr.role_id
			JOIN role_permissions rp ON rp.role_id=r.id
			JOIN permissions p ON p.id=rp.permission_id
			WHERE ap.principal_type='platform_admin' AND ap.status='active'
			  AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
			  AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
			LIMIT 1
		`).Scan(&platformAdminID)
		if errors.Is(err, pgx.ErrNoRows) {
			var anyBootstrapExists bool
			checkQuery := `SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
				OR EXISTS (
					SELECT 1 FROM principal_roles pr
					JOIN roles r ON r.id=pr.role_id
					WHERE r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin'
				)`
			if checkErr := tx.QueryRow(ctx, checkQuery).Scan(&anyBootstrapExists); checkErr != nil {
				t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
			}
			if anyBootstrapExists {
				t.Skip("Platform Admin bootstrap exists but no active administrator can delegate permissions")
			}
			platformAdminID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
				ctx,
				tx,
				fmt.Sprintf("staff-report-snapshot-%d@example.invalid", time.Now().UnixNano()),
				"integration-test-password-hash",
			)
		}
		if err != nil {
			t.Fatalf("resolve Platform Admin fixture: %v", err)
		}

		if err := tx.QueryRow(ctx, `
			SELECT id
			FROM roles
			WHERE school_id=$1 AND principal_type='staff' AND scope_type='school'
			  AND role_key='school_admin' AND is_system=TRUE
		`, schoolID).Scan(&schoolAdminRoleID); err != nil {
			t.Fatalf("load School Admin role: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO principal_roles(principal_id,role_id,assigned_by)
			VALUES($1,$2,$3)
			ON CONFLICT DO NOTHING
		`, actorID, schoolAdminRoleID, platformAdminID); err != nil {
			t.Fatalf("assign School Admin fixture role: %v", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit report authorization fixture: %v", err)
	}

	repo := NewUserRepository(pool)
	qualifications, err := repo.ListStaffQualificationReport(ctx, actorID, schoolID, StaffQualificationReportFilter{
		RecordType: "all",
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("list pool-backed qualifications report: %v", err)
	}
	var qualificationCount int64
	for _, item := range qualifications.Counts {
		qualificationCount += item.Count
	}
	if qualificationCount != qualifications.Total || int64(len(qualifications.Rows)) > qualifications.Total || len(qualifications.Rows) > 1 {
		t.Fatalf("qualifications report totals are inconsistent: %+v", qualifications)
	}

	experience, err := repo.ListStaffExperienceReport(ctx, actorID, schoolID, StaffExperienceReportFilter{
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("list pool-backed experience report: %v", err)
	}
	var experienceCount int64
	for _, item := range experience.Counts {
		experienceCount += item.Count
	}
	if experienceCount != experience.Total || int64(len(experience.Rows)) > experience.Total || len(experience.Rows) > 1 {
		t.Fatalf("experience report totals are inconsistent: %+v", experience)
	}

	// Exercise both production pool-backed attendance snapshot paths with a
	// real active staff member and a recorded mark.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	staff, err := repo.CreateStaffAccountWithProfile(ctx, actorID, schoolID, "att-snap-"+suffix, "integration-test-password-hash", models.StaffProfile{
		StaffType: "teaching", EmployeeCode: "AS-"+suffix, FullName: "Attendance Snapshot Teacher",
		Designation: "Teacher", MobileNumber: fmt.Sprintf("9%09d", time.Now().UnixNano()%1000000000), EmploymentStatus: "active",
	})
	if err != nil {
		t.Fatalf("create attendance snapshot staff fixture: %v", err)
	}
	attendanceRepo := NewStaffAttendanceRepository(pool)
	today := schooltime.Today()
	if _, err := attendanceRepo.Mark(ctx, actorID, schoolID, staff.PrincipalID, today, "present", nil); err != nil {
		t.Fatalf("mark attendance snapshot staff fixture: %v", err)
	}
	dailyRows, dailyTotal, err := attendanceRepo.ListDailyFilteredPage(ctx, actorID, schoolID, today, StaffAttendanceDailyFilter{Status: "all"}, 1, 0)
	if err != nil {
		t.Fatalf("list pool-backed daily attendance snapshot: %v", err)
	}
	if dailyTotal < 1 || len(dailyRows) != 1 {
		t.Fatalf("daily attendance snapshot total=%d rows=%+v want at least one row", dailyTotal, dailyRows)
	}
	monthlyRows, monthlyTotal, err := attendanceRepo.ListMonthlyPage(ctx, actorID, schoolID, today, 1, 0)
	if err != nil {
		t.Fatalf("list pool-backed monthly attendance snapshot: %v", err)
	}
	if monthlyTotal < 1 || len(monthlyRows) != 1 {
		t.Fatalf("monthly attendance snapshot total=%d rows=%+v want at least one row", monthlyTotal, monthlyRows)
	}

	leaveRepo := NewStaffLeaveRepository(pool)
	leaveType, err := leaveRepo.SaveType(ctx, actorID, schoolID, 0, StaffLeaveType{
		Name: "Snapshot Leave " + suffix, IsPaid: true, RequiresApproval: true, IsActive: true,
	})
	if err != nil {
		t.Fatalf("create leave type for pool-backed pagination: %v", err)
	}
	leaveDate := today.AddDate(0, 0, 7).Format("2006-01-02")
	if _, err := leaveRepo.CreateRequest(ctx, actorID, schoolID, StaffLeaveRequest{
		StaffPrincipalID: staff.PrincipalID, LeaveTypeID: leaveType.ID,
		StartDate: leaveDate, EndDate: leaveDate, DurationType: "full_day", Reason: "Snapshot pagination test",
	}); err != nil {
		t.Fatalf("create leave request for pool-backed pagination: %v", err)
	}
	leavePage, err := leaveRepo.ListRequestsPage(ctx, actorID, schoolID, "", 0, "", "", "", 1, 50)
	if err != nil {
		t.Fatalf("list pool-backed staff leave page snapshot: %v", err)
	}
	if leavePage.Total < 1 || len(leavePage.Items) < 1 {
		t.Fatalf("leave page snapshot total=%d items=%+v want at least one row", leavePage.Total, leavePage.Items)
	}

	substitutionRepo := NewAcademicRepository(pool)
	todayDate := today.Format("2006-01-02")
	substitutionPage, err := substitutionRepo.ListStaffSubstitutionsPage(ctx, actorID, schoolID, todayDate, todayDate, 0, 1, 50)
	if err != nil {
		t.Fatalf("list pool-backed staff substitution page snapshot: %v", err)
	}
	if substitutionPage.Total != int64(len(substitutionPage.Items)) {
		t.Fatalf("substitution page snapshot total=%d items=%d want matching empty/current page", substitutionPage.Total, len(substitutionPage.Items))
	}
}
