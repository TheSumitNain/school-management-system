package repository

import (
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"
)

func TestStaffDirectoryPaginationFiltersAndSchoolIsolationPostgresIntegration(t *testing.T) {
	fixture := newStaffLeaveIntegrationFixture(t)
	repo := NewUserRepository(fixture.pool)
	suffix := fmt.Sprint(time.Now().UnixNano())
	names := []string{"Directory Alpha", "Directory Beta", "Directory Gamma", "Directory Delta"}
	staffTypes := []string{"teaching", "non_teaching", "teaching", "non_teaching"}
	for i, name := range names {
		username := fmt.Sprintf("directory-%s-%d", suffix, i)
		var principalID int64
		if err := fixture.pool.QueryRow(fixture.ctx, `
			INSERT INTO users (school_id, username, password_hash, status)
			VALUES ($1, $2, 'integration-password-hash', 'active')
			RETURNING principal_id
		`, fixture.schoolID, username).Scan(&principalID); err != nil {
			t.Fatalf("create directory account %q: %v", name, err)
		}
		if _, err := fixture.pool.Exec(fixture.ctx, `
			INSERT INTO staff_profiles
				(principal_id, school_id, employee_code, full_name, designation, mobile_number, employment_status)
			VALUES ($1, $2, $3, $4, 'Office Staff', '9876543220', 'active')
		`, principalID, fixture.schoolID, fmt.Sprintf("DIR-%s-%02d", suffix, i), name); err != nil {
			t.Fatalf("create directory profile %q: %v", name, err)
		}
		if _, err := fixture.pool.Exec(fixture.ctx, `
			UPDATE staff_profiles SET staff_type=$3 WHERE principal_id=$1 AND school_id=$2
		`, principalID, fixture.schoolID, staffTypes[i]); err != nil {
			t.Fatalf("set directory staff type %q: %v", name, err)
		}
	}
	// A legacy login without a profile must be filterable as incomplete.
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
	`, fixture.schoolID, "dir-leg-"+suffix); err != nil {
		t.Fatalf("create legacy account: %v", err)
	}

	var inactivePrincipalID int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, fixture.schoolID, "dir-off-"+suffix).Scan(&inactivePrincipalID); err != nil {
		t.Fatalf("create inactive directory account: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO staff_profiles
			(principal_id, school_id, employee_code, full_name, designation, mobile_number, employment_status)
		VALUES ($1, $2, $3, 'Directory Inactive', 'Office Staff', '9876543221', 'active')
	`, inactivePrincipalID, fixture.schoolID, "DIR-OFF-"+suffix); err != nil {
		t.Fatalf("create inactive directory profile: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE staff_profiles SET staff_type='teaching' WHERE principal_id=$1 AND school_id=$2
	`, inactivePrincipalID, fixture.schoolID); err != nil {
		t.Fatalf("set inactive staff type: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		UPDATE staff_profiles SET employment_status='inactive' WHERE principal_id=$1 AND school_id=$2
	`, inactivePrincipalID, fixture.schoolID); err != nil {
		t.Fatalf("inactivate directory employee: %v", err)
	}

	page, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("list first filtered page: %v", err)
	}
	if page.Total != 8 || len(page.Staff) != 2 || page.Limit != 2 || page.Offset != 1 {
		t.Fatalf("unexpected page metadata: total=%d rows=%d limit=%d offset=%d", page.Total, len(page.Staff), page.Limit, page.Offset)
	}
	if page.Staff[0].FullName > page.Staff[1].FullName {
		t.Fatalf("page is not in stable employee-name order: %q before %q", page.Staff[0].FullName, page.Staff[1].FullName)
	}

	searchPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Search: "Directory Beta", Limit: 50})
	if err != nil {
		t.Fatalf("search directory: %v", err)
	}
	if searchPage.Total != 1 || len(searchPage.Staff) != 1 || searchPage.Staff[0].FullName != "Directory Beta" {
		t.Fatalf("search did not return exact matching employee: %+v", searchPage)
	}

	incompletePage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{ProfileStatus: "incomplete", Limit: 50})
	if err != nil {
		t.Fatalf("filter incomplete profiles: %v", err)
	}
	if incompletePage.Total != 1 || len(incompletePage.Staff) != 1 || incompletePage.Staff[0].ProfileComplete {
		t.Fatalf("incomplete-profile filter mismatch: %+v", incompletePage)
	}

	activePage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{EmploymentStatus: "active", Limit: 50})
	if err != nil || activePage.Total != 6 {
		t.Fatalf("active-employment filter total=%v err=%v, want 6", activePage, err)
	}
	inactivePage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{EmploymentStatus: "inactive", Limit: 50})
	if err != nil || inactivePage.Total != 1 || inactivePage.Staff[0].FullName != "Directory Inactive" {
		t.Fatalf("inactive-employment filter mismatch: page=%+v err=%v", inactivePage, err)
	}
	disabledPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{AccountStatus: "disabled", Limit: 50})
	if err != nil || disabledPage.Total != 1 || disabledPage.Staff[0].PrincipalID != inactivePrincipalID {
		t.Fatalf("disabled-account filter mismatch: page=%+v err=%v", disabledPage, err)
	}

	teachingPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{StaffType: "teaching", Limit: 50})
	if err != nil || teachingPage.Total != 3 { t.Fatalf("teaching filter mismatch: %+v %v", teachingPage, err) }
	nonTeachingPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{StaffType: "non_teaching", Limit: 50})
	if err != nil || nonTeachingPage.Total != 2 { t.Fatalf("non-teaching filter mismatch: %+v %v", nonTeachingPage, err) }
	unclassifiedPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{StaffType: "unclassified", Limit: 50})
	if err != nil || unclassifiedPage.Total < 1 { t.Fatalf("unclassified filter mismatch: %+v %v", unclassifiedPage, err) }
	legacyFound := false
	for _, user := range unclassifiedPage.Staff { if user.Username == "dir-leg-"+suffix { legacyFound = true } }
	if !legacyFound { t.Fatalf("unclassified filter omitted legacy account: %+v", unclassifiedPage.Staff) }

	foreignPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.staffPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Limit: 50})
	if err == nil || foreignPage != nil {
		t.Fatalf("ordinary staff unexpectedly read directory: page=%+v err=%v", foreignPage, err)
	}

	// A valid School Admin must not be able to change the tenant scope by
	// supplying another school ID to the repository boundary.
	forgedSchoolPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID+1,
		models.StaffDirectoryFilter{Limit: 50})
	if err == nil || forgedSchoolPage != nil {
		t.Fatalf("School Admin unexpectedly read a forged foreign-school directory: page=%+v err=%v", forgedSchoolPage, err)
	}
}


func TestStaffDirectorySummaryMatchesFilteredCountsAndEnforcesSchoolAdminPostgresIntegration(t *testing.T) {
	fixture := newStaffLeaveIntegrationFixture(t)
	repo := NewUserRepository(fixture.pool)
	summary, err := repo.GetStaffDirectorySummary(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID)
	if err != nil || summary == nil {
		t.Fatalf("get staff directory summary: summary=%+v err=%v", summary, err)
	}
	all, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{Limit: 1})
	if err != nil {
		t.Fatalf("count all staff: %v", err)
	}
	active, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{EmploymentStatus: "active", Limit: 1})
	if err != nil {
		t.Fatalf("count active employment: %v", err)
	}
	enabled, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{AccountStatus: "enabled", Limit: 1})
	if err != nil {
		t.Fatalf("count enabled accounts: %v", err)
	}
	incomplete, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{ProfileStatus: "incomplete", Limit: 1})
	if err != nil {
		t.Fatalf("count incomplete profiles: %v", err)
	}
	activeDisabled, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{EmploymentStatus: "active", AccountStatus: "disabled", Limit: 1})
	if err != nil {
		t.Fatalf("count active employees with disabled login: %v", err)
	}
	inactive, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID, models.StaffDirectoryFilter{EmploymentStatus: "inactive", Limit: 1})
	if err != nil {
		t.Fatalf("count inactive employment: %v", err)
	}
	var mustChange int64
	if err := fixture.pool.QueryRow(fixture.ctx, `SELECT COUNT(*) FROM users WHERE school_id=$1 AND must_change_password=TRUE`, fixture.schoolID).Scan(&mustChange); err != nil {
		t.Fatalf("count required password changes: %v", err)
	}
	if summary.Total != all.Total || summary.ActiveEmployment != active.Total || summary.LoginEnabled != enabled.Total ||
		summary.ProfileIncomplete != incomplete.Total || summary.ActiveEmploymentDisabledLogin != activeDisabled.Total ||
		summary.MustChangePassword != mustChange || summary.InactiveEmployment != inactive.Total {
		t.Fatalf("summary disagrees with directory filters: summary=%+v all=%d active=%d enabled=%d incomplete=%d active-disabled=%d must-change=%d inactive=%d",
			summary, all.Total, active.Total, enabled.Total, incomplete.Total, activeDisabled.Total, mustChange, inactive.Total)
	}
	if denied, err := repo.GetStaffDirectorySummary(fixture.ctx, fixture.staffPrincipalID, fixture.schoolID); err == nil || denied != nil {
		t.Fatalf("ordinary staff unexpectedly read directory summary: summary=%+v err=%v", denied, err)
	}

	if forgedSummary, err := repo.GetStaffDirectorySummary(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID+1); err == nil || forgedSummary != nil {
		t.Fatalf("School Admin unexpectedly read a forged foreign-school directory summary: summary=%+v err=%v", forgedSummary, err)
	}

	// Repository reads must honor the same live DENY override as AuthMiddleware,
	// even when the principal still has the School Admin system-role grant.
	var usersReadPermissionID int64
	if err := fixture.pool.QueryRow(fixture.ctx, `
		SELECT id FROM permissions WHERE permission_key='users.read' AND scope_type='school'
	`).Scan(&usersReadPermissionID); err != nil {
		t.Fatalf("load users.read permission for DENY regression: %v", err)
	}
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO user_permission_overrides
			(principal_id, school_id, permission_id, effect, reason, granted_by)
		VALUES ($1,$2,$3,'deny','staff directory DENY regression',$4)
		ON CONFLICT (principal_id, school_id, permission_id)
		DO UPDATE SET effect='deny', reason=EXCLUDED.reason, granted_by=EXCLUDED.granted_by
	`, fixture.adminPrincipalID, fixture.schoolID, usersReadPermissionID, fixture.adminPrincipalID); err != nil {
		t.Fatalf("apply users.read DENY override: %v", err)
	}
	if denied, err := repo.GetStaffDirectorySummary(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID); err == nil || denied != nil {
		t.Fatalf("staff directory summary ignored users.read DENY override: summary=%+v err=%v", denied, err)
	}
	if denied, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Limit: 10}); err == nil || denied != nil {
		t.Fatalf("staff directory page ignored users.read DENY override: page=%+v err=%v", denied, err)
	}
}
