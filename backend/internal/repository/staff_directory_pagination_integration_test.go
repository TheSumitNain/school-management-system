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
	}
	// A legacy login without a profile must be filterable as incomplete.
	if _, err := fixture.pool.Exec(fixture.ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
	`, fixture.schoolID, "dir-leg-"+suffix); err != nil {
		t.Fatalf("create legacy account: %v", err)
	}

	page, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.adminPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("list first filtered page: %v", err)
	}
	if page.Total != 7 || len(page.Staff) != 2 || page.Limit != 2 || page.Offset != 1 {
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

	foreignPage, err := repo.ListStaffDirectoryPage(fixture.ctx, fixture.staffPrincipalID, fixture.schoolID,
		models.StaffDirectoryFilter{Limit: 50})
	if err == nil || foreignPage != nil {
		t.Fatalf("ordinary staff unexpectedly read directory: page=%+v err=%v", foreignPage, err)
	}
}
