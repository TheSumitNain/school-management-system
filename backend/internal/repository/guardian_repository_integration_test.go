package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
)

// TestGuardianRepositoryPostgresIntegration exercises guardian contact CRUD,
// student relationship history, live delegated RBAC, audit events, and tenant
// isolation. All application writes are contained by an outer transaction.
func TestGuardianRepositoryPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped guardian repository test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id, school_id FROM users WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	// Resolve a Platform Admin capable of granting school permission
	// delegations. A bootstrap created here exists only inside this transaction.
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
	if errors.Is(err, pgx.ErrNoRows) {
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
		if bootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx,
			fmt.Sprintf("guardian-repository-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'school_admin'
		   AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign School Admin fixture role: %v", err)
	}

	roleRepo := NewRoleRepository(tx)
	readDelegation, err := roleRepo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "guardians.read", true, false,
	)
	if err != nil {
		t.Fatalf("delegate guardians.read to school roles: %v", err)
	}
	manageDelegation, err := roleRepo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "guardians.manage", true, false,
	)
	if err != nil {
		t.Fatalf("delegate guardians.manage to school roles: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	customRole, err := roleRepo.CreateCustomRole(
		ctx, actorPrincipalID, schoolID, "guardian_test_"+suffix, "Guardian Integration Role",
	)
	if err != nil {
		t.Fatalf("create guardian integration role: %v", err)
	}
	if err := roleRepo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, readDelegation.ID, "guardians.read",
	); err != nil {
		t.Fatalf("grant guardians.read to custom role: %v", err)
	}
	if err := roleRepo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, manageDelegation.ID, "guardians.manage",
	); err != nil {
		t.Fatalf("grant guardians.manage to custom role: %v", err)
	}
	var targetPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, schoolID, "gtrg-"+suffix).Scan(&targetPrincipalID); err != nil {
		t.Fatalf("create guardian permission target staff: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, customRole.ID); err != nil {
		t.Fatalf("assign guardian integration role to target staff: %v", err)
	}

	repo := NewAcademicRepository(tx)

	// Even if a Platform Admin delegates guardian permissions to a custom
	// role, a Teacher-only principal must not gain family/contact access.
	var teacherRoleID, teacherPrincipalID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id=$1 AND principal_type='staff' AND scope_type='school'
		   AND role_key='teacher' AND is_system=TRUE
	`, schoolID).Scan(&teacherRoleID); err != nil {
		t.Fatalf("load Teacher system role: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, schoolID, "tguard-"+suffix).Scan(&teacherPrincipalID); err != nil {
		t.Fatalf("create Teacher boundary fixture: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, teacherPrincipalID, teacherRoleID); err != nil {
		t.Fatalf("assign Teacher system role: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, teacherPrincipalID, customRole.ID); err != nil {
		t.Fatalf("assign guardian custom role to Teacher: %v", err)
	}
	if _, _, err := repo.ListGuardians(ctx, teacherPrincipalID, schoolID, "", "active", 10, 0); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("Teacher with delegated guardians.read must be denied; got %v", err)
	}
	if err := authorizeAcademic(ctx, tx, teacherPrincipalID, schoolID, "students.read_sensitive"); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("Teacher with a custom role must not read sensitive student profiles; got %v", err)
	}

	mobile := "+911234567890"
	email := "guardian-" + suffix + "@example.invalid"
	guardian, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, 0,
		"  Guardian One  ", &mobile, &email, "active")
	if err != nil {
		t.Fatalf("create guardian contact: %v", err)
	}
	if guardian.ID <= 0 || guardian.FullName != "Guardian One" ||
		guardian.MobileNumber == nil || *guardian.MobileNumber != mobile ||
		guardian.EmailAddress == nil || *guardian.EmailAddress != email ||
		guardian.Status != "active" {
		t.Fatalf("unexpected created guardian: %+v", guardian)
	}

	updated, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, "inactive")
	if err != nil {
		t.Fatalf("update guardian contact: %v", err)
	}
	if updated.ID != guardian.ID || updated.FullName != "Guardian One Updated" || updated.Status != "inactive" {
		t.Fatalf("unexpected updated guardian: %+v", updated)
	}
	updated, err = repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, "active")
	if err != nil {
		t.Fatalf("reactivate guardian contact: %v", err)
	}

	got, err := repo.GetGuardian(ctx, targetPrincipalID, schoolID, guardian.ID)
	if err != nil || got.ID != guardian.ID || got.FullName != "Guardian One Updated" {
		t.Fatalf("get guardian: guardian=%+v err=%v", got, err)
	}
	guardians, total, err := repo.ListGuardians(ctx, targetPrincipalID, schoolID,
		"Guardian One Updated", "active", 10, 0)
	if err != nil || total != 1 || len(guardians) != 1 || guardians[0].ID != guardian.ID {
		t.Fatalf("search guardians: total=%d items=%+v err=%v", total, guardians, err)
	}

	var studentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, 'Guardian Integration Student') RETURNING id
	`, schoolID, "GUARDIAN-"+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create same-school student fixture: %v", err)
	}
	var otherSchoolID, foreignStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active') RETURNING id
	`, "Guardian Isolation "+suffix, "GI-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second-school fixture: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, 'Foreign Guardian Student') RETURNING id
	`, otherSchoolID, "FOREIGN-"+suffix).Scan(&foreignStudentID); err != nil {
		t.Fatalf("create foreign-school student fixture: %v", err)
	}
	if _, err := repo.GetGuardian(ctx, targetPrincipalID, otherSchoolID, guardian.ID); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("cross-school guardian read error = %v, want access denied", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		foreignStudentID, guardian.ID, "father", true, true); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("cross-school student link error = %v, want not found", err)
	}

	firstLink, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, guardian.ID, "father", true, true)
	if err != nil {
		t.Fatalf("link guardian to student: %v", err)
	}
	if firstLink.ID <= 0 || !firstLink.IsPrimaryContact || !firstLink.IsEmergencyContact ||
		firstLink.RelationshipType != "father" || firstLink.GuardianName != "Guardian One Updated" {
		t.Fatalf("unexpected first student-guardian link: %+v", firstLink)
	}

	var relatedStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, NULL, 'Student Sharing Guardian') RETURNING id
	`, schoolID).Scan(&relatedStudentID); err != nil {
		t.Fatalf("create related-student fixture: %v", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		relatedStudentID, guardian.ID, "father", true, false); err != nil {
		t.Fatalf("link shared guardian to related student: %v", err)
	}
	relatedStudents, err := repo.ListStudentRelatedStudents(ctx, targetPrincipalID, schoolID, studentID)
	if err != nil || len(relatedStudents) != 1 || relatedStudents[0].StudentID != relatedStudentID ||
		len(relatedStudents[0].SharedGuardianNames) != 1 || relatedStudents[0].SharedGuardianNames[0] != guardian.FullName {
		t.Fatalf("list students sharing active guardian: items=%+v err=%v", relatedStudents, err)
	}

	secondMobile := "+919876543210"
	secondGuardian, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, 0,
		"Guardian Two", &secondMobile, nil, "active")
	if err != nil {
		t.Fatalf("create second guardian contact: %v", err)
	}
	secondLink, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, secondGuardian.ID, "mother", true, false)
	if err != nil {
		t.Fatalf("link second primary guardian: %v", err)
	}
	if !secondLink.IsPrimaryContact {
		t.Fatalf("second link should be primary: %+v", secondLink)
	}

	history, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(history) != 2 {
		t.Fatalf("list complete student guardian history: count=%d err=%v", len(history), err)
	}
	for _, link := range history {
		if link.ID == firstLink.ID && link.IsPrimaryContact {
			t.Fatal("previous primary contact was not cleared when new primary was selected")
		}
	}
	active, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, false)
	if err != nil || len(active) != 2 {
		t.Fatalf("list active student guardian links: count=%d err=%v", len(active), err)
	}

	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "relationship ended"); err != nil {
		t.Fatalf("revoke student guardian link: %v", err)
	}
	active, err = repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, false)
	if err != nil || len(active) != 1 || active[0].ID != secondLink.ID {
		t.Fatalf("active links after revocation: items=%+v err=%v", active, err)
	}
	history, err = repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(history) != 2 {
		t.Fatalf("historical links after revocation: count=%d err=%v", len(history), err)
	}
	var revokedFound bool
	for _, link := range history {
		if link.ID == firstLink.ID {
			revokedFound = link.RevokedAt != nil
		}
	}
	if !revokedFound {
		t.Fatalf("revoked link missing from immutable history: %+v", history)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "duplicate revoke"); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("duplicate link revocation error = %v, want not found", err)
	}

	var guardianAuditCount, linkCreateAuditCount, linkRevokeAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE action IN ('school.guardian.created', 'school.guardian.updated')),
		  COUNT(*) FILTER (WHERE action = 'school.student_guardian_link_created'),
		  COUNT(*) FILTER (WHERE action = 'school.student_guardian_link_revoked')
		FROM audit_events
		WHERE school_id = $1
		  AND actor_principal_id = $2
		  AND (
		    (resource_type = 'guardian' AND resource_id IN ($3, $4))
		    OR resource_type = 'student_guardian_link'
		  )
	`, schoolID, targetPrincipalID, fmt.Sprint(guardian.ID),
		fmt.Sprint(secondGuardian.ID)).Scan(
		&guardianAuditCount, &linkCreateAuditCount, &linkRevokeAuditCount,
	); err != nil {
		t.Fatalf("count guardian audit events: %v", err)
	}
	if guardianAuditCount != 4 || linkCreateAuditCount != 2 || linkRevokeAuditCount != 1 {
		t.Fatalf("guardian audit counts = contacts %d, link creates %d, link revokes %d; want 4, 2, 1",
			guardianAuditCount, linkCreateAuditCount, linkRevokeAuditCount)
	}

	target := &models.AuthPrincipal{
		ID: targetPrincipalID, Type: "staff", Status: "active",
		TokenVersion: 1, SchoolID: schoolID,
	}
	effective, err := loadEffectivePermissions(ctx, tx, target)
	if err != nil {
		t.Fatalf("resolve guardian actor live permissions: %v", err)
	}
	for _, key := range []string{"guardians.read", "guardians.manage"} {
		found := false
		for _, permission := range effective {
			if permission == key {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected live delegated permission %q, got %v", key, effective)
		}
	}
}
