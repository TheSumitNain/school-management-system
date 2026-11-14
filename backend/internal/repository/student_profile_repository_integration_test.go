package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestStudentProfileRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin student profile integration transaction: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, actorID int64
	if err := tx.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &actorID); err != nil {
		t.Fatalf("load school admin fixture: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'school_admin' AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load system School Admin role: %v", err)
	}
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND r.is_system = TRUE
		   AND p.permission_key = 'platform.permissions.delegate'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if err != nil {
		var bootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1 FROM principal_roles pr
			        JOIN roles r ON r.id = pr.role_id
			        WHERE r.principal_type = 'platform_admin'
			          AND r.scope_type = 'platform'
			          AND r.role_key = 'platform_admin'
			    )
		`).Scan(&bootstrapExists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if bootstrapExists { t.Skip("no active Platform Admin has platform.permissions.delegate") }
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("student-profile-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
		if err != nil { t.Fatalf("create rollback-scoped Platform Admin fixture: %v", err) }
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("grant School Admin role to fixture actor: %v", err)
	}

	repo := NewAcademicRepository(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	student, err := repo.CreateStudent(ctx, actorID, schoolID, " ADM-"+suffix+" ", "  Student One  ")
	if err != nil { t.Fatalf("create student admission: %v", err) }
	if student.AdmissionNumber != "ADM-"+suffix || student.FullName != "Student One" || student.Status != "active" {
		t.Fatalf("unexpected created student: %+v", student)
	}
	duplicateLegacy, err := repo.CreateStudent(ctx, actorID, schoolID, "adm-"+suffix, "Duplicate Legacy Admission")
	if err != nil || duplicateLegacy == nil || duplicateLegacy.AdmissionNumber != "adm-"+suffix {
		t.Fatalf("duplicate legacy student-level admission value should be retained without a conflict: student=%+v err=%v", duplicateLegacy, err)
	}
	got, err := repo.GetStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || got.ID != student.ID { t.Fatalf("get student: student=%+v err=%v", got, err) }

	// Address reads must return every saved hierarchy level, including names
	// needed to display a previously selected location after deactivation.
	state, err := repo.CreateStudentLocation(ctx, actorID, schoolID, nil, "IN", "state", "State "+suffix, nil)
	if err != nil { t.Fatalf("create address state: %v", err) }
	district, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &state.ID, "", "district", "District "+suffix, nil)
	if err != nil { t.Fatalf("create address district: %v", err) }
	subdistrict, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &district.ID, "", "subdistrict", "Block "+suffix, nil)
	if err != nil { t.Fatalf("create address block: %v", err) }
	localityType := "village"
	locality, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &subdistrict.ID, "", "locality", "Village "+suffix, &localityType)
	if err != nil { t.Fatalf("create address locality: %v", err) }
	line1 := "House 1"
	if _, err := repo.SaveStudentAddress(ctx, actorID, schoolID, student.ID, "current",
		&line1, nil, nil, nil, nil, &locality.ID, false); err != nil {
		t.Fatalf("save student address: %v", err)
	}
	if _, err := repo.DeactivateStudentLocation(ctx, actorID, schoolID, locality.ID); err != nil {
		t.Fatalf("deactivate saved locality: %v", err)
	}
	addresses, err := repo.ListStudentAddresses(ctx, actorID, schoolID, student.ID)
	if err != nil || len(addresses) != 1 {
		t.Fatalf("list saved address: addresses=%+v err=%v", addresses, err)
	}
	address := addresses[0]
	if address.StateLocationID == nil || *address.StateLocationID != state.ID ||
		address.DistrictLocationID == nil || *address.DistrictLocationID != district.ID ||
		address.SubdistrictLocationID == nil || *address.SubdistrictLocationID != subdistrict.ID ||
		address.StateLocationName == nil || *address.StateLocationName != state.Name ||
		address.DistrictLocationName == nil || *address.DistrictLocationName != district.Name ||
		address.SubdistrictLocationName == nil || *address.SubdistrictLocationName != subdistrict.Name ||
		address.LocationName == nil || *address.LocationName != locality.Name {
		t.Fatalf("saved address hierarchy was not fully returned: %+v", address)
	}

	updated, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Student One Updated")
	if err != nil || updated.FullName != "Student One Updated" {
		t.Fatalf("update student: student=%+v err=%v", updated, err)
	}
	items, total, err := repo.ListStudents(ctx, actorID, schoolID, "Updated", "active", 10, 0)
	if err != nil || total < 1 { t.Fatalf("search students: total=%d err=%v", total, err) }
	found := false
	for _, item := range items { if item.ID == student.ID { found = true } }
	if !found { t.Fatalf("student %d missing from school-scoped search: %+v", student.ID, items) }

	archived, err := repo.ArchiveStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || archived.Status != "archived" { t.Fatalf("archive student: student=%+v err=%v", archived, err) }
	if _, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Should Not Update"); !errors.Is(err, ErrAcademicNotFound) {
		t.Fatalf("update archived student error = %v, want not found", err)
	}

	var otherSchoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status) VALUES ($1, $2, 'active') RETURNING id
	`, "Student Profile Isolation "+suffix, "SPI-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create isolation school: %v", err)
	}
	if _, err := repo.GetStudent(ctx, actorID, otherSchoolID, student.ID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school student lookup error = %v, want access denied", err)
	}
}
