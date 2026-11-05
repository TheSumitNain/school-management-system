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

// TestRoleRepositoryAdministrationPostgresIntegration exercises the role,
// delegation, permission-grant, override, and staff-role repository workflows.
// Every write runs inside one outer transaction and is rolled back, including
// audit rows and a temporary Platform Admin bootstrap when one is required.
func TestRoleRepositoryAdministrationPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var hasV16 bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'role_permissions'
			  AND column_name = 'delegation_id'
		) AND EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'user_permission_overrides'
			  AND column_name = 'delegation_id'
		)
	`).Scan(&hasV16); err != nil {
		t.Fatalf("detect V16 exact delegation schema: %v", err)
	}
	if !hasV16 {
		t.Skip("V16 exact delegation linkage is not applied; apply migrations to school_management_test before running this integration test")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped role repository test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id, school_id
		  FROM users
		 WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	// Use an existing authorized Platform Admin when available. If this test
	// database has never been bootstrapped, create one only inside tx so the
	// deferred rollback leaves no permanent administrator behind.
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
		var anyBootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1
			          FROM principal_roles pr
			          JOIN roles r ON r.id = pr.role_id
		         WHERE r.principal_type = 'platform_admin'
		           AND r.scope_type = 'platform'
		           AND r.role_key = 'platform_admin'
		    )
		`).Scan(&anyBootstrapExists); checkErr != nil {
			t.Fatalf("check existing Platform Admin bootstrap state: %v", checkErr)
		}
		if anyBootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx,
			fmt.Sprintf("role-repository-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		  FROM roles
		 WHERE school_id = $1
		   AND principal_type = 'staff'
		   AND scope_type = 'school'
		   AND role_key = 'school_admin'
		   AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load V13-provisioned School Admin role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign temporary School Admin fixture role: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var targetPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING principal_id
	`, schoolID, "role-target-"+suffix, "integration-test-password-hash").Scan(&targetPrincipalID); err != nil {
		t.Fatalf("create target staff fixture: %v", err)
	}

	// pgx.Tx implements repositoryDB. Repository-owned Begin calls therefore
	// create savepoints, and their commits remain inside this rollback boundary.
	repo := NewRoleRepository(tx)

	customRoleKey := "custom_test_" + suffix
	customRole, err := repo.CreateCustomRole(ctx, actorPrincipalID, schoolID, customRoleKey, "Integration Custom Role")
	if err != nil {
		t.Fatalf("authorized custom role creation failed: %v", err)
	}
	if customRole == nil || customRole.ID <= 0 || customRole.RoleKey != customRoleKey || customRole.IsSystem {
		t.Fatalf("unexpected custom role: %+v", customRole)
	}
	if _, err := repo.CreateCustomRole(ctx, actorPrincipalID, schoolID, customRoleKey, "Duplicate Custom Role"); !errors.Is(err, ErrRoleNameTaken) {
		t.Fatalf("duplicate custom role error = %v, want ErrRoleNameTaken", err)
	}

	roles, err := repo.ListSchoolRoles(ctx, actorPrincipalID, schoolID)
	if err != nil {
		t.Fatalf("authorized school role listing failed: %v", err)
	}
	foundCustomRole := false
	for _, role := range roles {
		if role.ID == customRole.ID {
			foundCustomRole = true
		}
	}
	if !foundCustomRole {
		t.Fatalf("custom role %d missing from school role listing", customRole.ID)
	}

	permissions, err := repo.ListSchoolPermissions(ctx)
	if err != nil {
		t.Fatalf("school permission catalogue listing failed: %v", err)
	}
	foundBasicStudentPermission := false
	for _, permission := range permissions {
		if permission.Key == "students.read_basic" && permission.Delegable {
			foundBasicStudentPermission = true
		}
	}
	if !foundBasicStudentPermission {
		t.Fatal("active delegable students.read_basic permission missing from catalogue")
	}

	roleDelegation, err := repo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "students.read_basic", true, false,
	)
	if err != nil {
		t.Fatalf("authorized Platform Admin delegation grant failed: %v", err)
	}
	if roleDelegation.ID <= 0 || roleDelegation.PermissionKey != "students.read_basic" || !roleDelegation.CanAssignToRoles {
		t.Fatalf("unexpected role delegation: %+v", roleDelegation)
	}
	if _, err := repo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "students.read_basic", true, false,
	); !errors.Is(err, ErrDelegationAlreadyActive) {
		t.Fatalf("duplicate active delegation error = %v, want ErrDelegationAlreadyActive", err)
	}

	overrideDelegation, err := repo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "homework.create", false, true,
	)
	if err != nil {
		t.Fatalf("authorized user-override delegation grant failed: %v", err)
	}

	// A School Admin cannot add delegated authority to their own principal,
	// either through a role assignment or a direct permission override.
	if err := repo.AssignStaffRole(ctx, actorPrincipalID, schoolID, actorPrincipalID, customRole.ID); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("self role assignment error = %v, want ErrRolePermissionDenied", err)
	}
	if err := repo.SetStaffPermissionOverride(
		ctx, actorPrincipalID, schoolID, actorPrincipalID, overrideDelegation.ID,
		"homework.create", "allow", "self-escalation prevention test",
	); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("self permission override error = %v, want ErrRolePermissionDenied", err)
	}
	if err := repo.DeleteStaffPermissionOverride(
		ctx, actorPrincipalID, schoolID, actorPrincipalID, "homework.create",
	); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("self permission override deletion error = %v, want ErrRolePermissionDenied", err)
	}

	delegations, err := repo.ListSchoolPermissionDelegations(ctx, actorPrincipalID, schoolID)
	if err != nil {
		t.Fatalf("authorized school delegation listing failed: %v", err)
	}
	if len(delegations) != 2 {
		t.Fatalf("active delegation count = %d, want 2: %+v", len(delegations), delegations)
	}

	if err := repo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, roleDelegation.ID, "students.read_basic",
	); err != nil {
		t.Fatalf("authorized custom-role permission grant failed: %v", err)
	}
	if err := repo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, roleDelegation.ID, "students.read_basic",
	); !errors.Is(err, ErrRolePermissionAlreadyGranted) {
		t.Fatalf("duplicate custom-role permission error = %v, want ErrRolePermissionAlreadyGranted", err)
	}

	if err := repo.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, customRole.ID); err != nil {
		t.Fatalf("authorized staff role assignment failed: %v", err)
	}
	if err := repo.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, customRole.ID); !errors.Is(err, ErrRoleAssignmentExists) {
		t.Fatalf("duplicate staff role assignment error = %v, want ErrRoleAssignmentExists", err)
	}
	assignments, err := repo.ListStaffRoleAssignments(ctx, actorPrincipalID, schoolID, targetPrincipalID)
	if err != nil {
		t.Fatalf("authorized staff role assignment listing failed: %v", err)
	}
	if len(assignments) != 1 || assignments[0].RoleID != customRole.ID {
		t.Fatalf("unexpected staff role assignments: %+v", assignments)
	}

	target := &models.AuthPrincipal{
		ID: targetPrincipalID, Type: "staff", Status: "active",
		TokenVersion: 1, SchoolID: schoolID,
	}
	assertPermission := func(permissionKey string, want bool) {
		t.Helper()
		resolved, resolveErr := loadEffectivePermissions(ctx, tx, target)
		if resolveErr != nil {
			t.Fatalf("resolve effective permissions: %v", resolveErr)
		}
		found := false
		for _, item := range resolved {
			if item == permissionKey {
				found = true
				break
			}
		}
		if found != want {
			t.Fatalf("effective permission %q present=%v, want %v; resolved=%v", permissionKey, found, want, resolved)
		}
	}
	assertPermission("students.read_basic", true)

	if err := repo.RevokeCustomRolePermission(ctx, actorPrincipalID, schoolID, customRole.ID, "students.read_basic"); err != nil {
		t.Fatalf("authorized custom-role permission revoke failed: %v", err)
	}
	assertPermission("students.read_basic", false)
	if err := repo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, roleDelegation.ID, "students.read_basic",
	); err != nil {
		t.Fatalf("regrant custom-role permission with active delegation failed: %v", err)
	}
	assertPermission("students.read_basic", true)

	if err := repo.RevokeStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, customRole.ID); err != nil {
		t.Fatalf("authorized staff role revocation failed: %v", err)
	}
	assertPermission("students.read_basic", false)
	assignments, err = repo.ListStaffRoleAssignments(ctx, actorPrincipalID, schoolID, targetPrincipalID)
	if err != nil {
		t.Fatalf("list staff role assignments after revoke: %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("revoked staff role remains active: %+v", assignments)
	}
	if err := repo.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, schoolAdminRoleID); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("reserved School Admin assignment error = %v, want ErrRoleNotFound", err)
	}

	if err := repo.SetStaffPermissionOverride(
		ctx, actorPrincipalID, schoolID, targetPrincipalID, overrideDelegation.ID,
		"homework.create", "allow", "integration test allow",
	); err != nil {
		t.Fatalf("authorized ALLOW override failed: %v", err)
	}
	assertPermission("homework.create", true)
	overrides, err := repo.ListStaffPermissionOverrides(ctx, actorPrincipalID, schoolID, targetPrincipalID)
	if err != nil {
		t.Fatalf("authorized staff override listing failed: %v", err)
	}
	if len(overrides) != 1 || overrides[0].Effect != "allow" || overrides[0].DelegationID != overrideDelegation.ID {
		t.Fatalf("unexpected ALLOW override listing: %+v", overrides)
	}

	if err := repo.SetStaffPermissionOverride(
		ctx, actorPrincipalID, schoolID, targetPrincipalID, overrideDelegation.ID,
		"homework.create", "deny", "integration test explicit deny",
	); err != nil {
		t.Fatalf("authorized DENY override update failed: %v", err)
	}
	assertPermission("homework.create", false)
	overrides, err = repo.ListStaffPermissionOverrides(ctx, actorPrincipalID, schoolID, targetPrincipalID)
	if err != nil {
		t.Fatalf("list staff overrides after DENY update: %v", err)
	}
	if len(overrides) != 1 || overrides[0].Effect != "deny" {
		t.Fatalf("DENY override did not replace ALLOW: %+v", overrides)
	}
	if err := repo.DeleteStaffPermissionOverride(ctx, actorPrincipalID, schoolID, targetPrincipalID, "homework.create"); err != nil {
		t.Fatalf("authorized staff override deletion failed: %v", err)
	}
	assertPermission("homework.create", false)
	overrides, err = repo.ListStaffPermissionOverrides(ctx, actorPrincipalID, schoolID, targetPrincipalID)
	if err != nil {
		t.Fatalf("list staff overrides after deletion: %v", err)
	}
	if len(overrides) != 0 {
		t.Fatalf("deleted staff override remains: %+v", overrides)
	}

	// A School Admin from School A must not read or mutate School B's role
	// administration state, even when supplied with School B identifiers.
	var otherSchoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, "RBAC Isolation School "+suffix, "RBAC-ISO-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second-school isolation fixture: %v", err)
	}
	var otherSchoolPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING principal_id
	`, otherSchoolID, "oth-"+suffix, "integration-test-password-hash").Scan(&otherSchoolPrincipalID); err != nil {
		t.Fatalf("create second-school staff fixture: %v", err)
	}
	if _, err := repo.ListSchoolRoles(ctx, actorPrincipalID, otherSchoolID); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("School A actor listing School B roles error = %v, want ErrRolePermissionDenied", err)
	}
	if _, err := repo.CreateCustomRole(ctx, actorPrincipalID, otherSchoolID, "cross_school_"+suffix, "Cross School Role"); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("School A actor creating School B role error = %v, want ErrRolePermissionDenied", err)
	}
	crossSchoolAssignments, err := repo.ListStaffRoleAssignments(ctx, actorPrincipalID, schoolID, otherSchoolPrincipalID)
	if err != nil {
		t.Fatalf("list role assignments for foreign-school target: %v", err)
	}
	if len(crossSchoolAssignments) != 0 {
		t.Fatalf("School A actor received School B role assignments: %+v", crossSchoolAssignments)
	}
	if err := repo.AssignStaffRole(ctx, actorPrincipalID, schoolID, otherSchoolPrincipalID, customRole.ID); !errors.Is(err, ErrRoleNotFound) {
		t.Fatalf("School A actor assigning role to School B staff error = %v, want ErrRoleNotFound", err)
	}
	if err := repo.SetStaffPermissionOverride(
		ctx, actorPrincipalID, schoolID, otherSchoolPrincipalID, overrideDelegation.ID,
		"homework.create", "allow", "cross-school isolation test",
	); !errors.Is(err, ErrDelegationNotFound) {
		t.Fatalf("School A actor overriding School B staff error = %v, want ErrDelegationNotFound", err)
	}
	crossSchoolOverrides, err := repo.ListStaffPermissionOverrides(ctx, actorPrincipalID, schoolID, otherSchoolPrincipalID)
	if err != nil {
		t.Fatalf("list permission overrides for foreign-school target: %v", err)
	}
	if len(crossSchoolOverrides) != 0 {
		t.Fatalf("School A actor received School B permission overrides: %+v", crossSchoolOverrides)
	}

	if err := repo.RevokeSchoolPermissionDelegation(ctx, platformAdminPrincipalID, roleDelegation.ID); err != nil {
		t.Fatalf("authorized role delegation revoke failed: %v", err)
	}
	if err := repo.RevokeSchoolPermissionDelegation(ctx, platformAdminPrincipalID, overrideDelegation.ID); err != nil {
		t.Fatalf("authorized override delegation revoke failed: %v", err)
	}
	if err := repo.RevokeSchoolPermissionDelegation(ctx, platformAdminPrincipalID, roleDelegation.ID); !errors.Is(err, ErrDelegationNotFound) {
		t.Fatalf("duplicate delegation revoke error = %v, want ErrDelegationNotFound", err)
	}
	delegations, err = repo.ListSchoolPermissionDelegations(ctx, actorPrincipalID, schoolID)
	if err != nil {
		t.Fatalf("list delegations after revocation: %v", err)
	}
	if len(delegations) != 0 {
		t.Fatalf("revoked delegations remain active: %+v", delegations)
	}

	if _, err := repo.GrantSchoolPermissionDelegation(
		ctx, actorPrincipalID, schoolID, "students.read_basic", true, false,
	); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("School Admin delegation grant error = %v, want ErrRolePermissionDenied", err)
	}
	if _, err := repo.ListSchoolPermissionDelegations(ctx, targetPrincipalID, schoolID); !errors.Is(err, ErrRolePermissionDenied) {
		t.Fatalf("non-admin delegation listing error = %v, want ErrRolePermissionDenied", err)
	}

	var schoolAdminAuditCount, platformAdminAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE actor_principal_id = $1),
		    COUNT(*) FILTER (WHERE actor_principal_id = $2)
		  FROM audit_events
		 WHERE actor_principal_id IN ($1, $2)
		   AND school_id = $3
		   AND resource_id IN ($4, $5, $6, $7)
		   AND action IN (
		       'school.role.created',
		       'platform.permission_delegation.granted',
		       'platform.permission_delegation.revoked',
		       'school.custom_role.permission_granted',
		       'school.custom_role.permission_revoked',
		       'school.staff.permission_override.set',
		       'school.staff.permission_override.removed',
		       'school.staff.role_assigned',
		       'school.staff.role_revoked'
		   )
	`, actorPrincipalID, platformAdminPrincipalID, schoolID, fmt.Sprint(customRole.ID),
		fmt.Sprint(roleDelegation.ID), fmt.Sprint(overrideDelegation.ID),
		fmt.Sprint(targetPrincipalID)).Scan(&schoolAdminAuditCount, &platformAdminAuditCount); err != nil {
		t.Fatalf("count role administration audit events: %v", err)
	}
	if schoolAdminAuditCount != 9 || platformAdminAuditCount != 4 {
		t.Fatalf("role administration audit counts = School Admin %d, Platform Admin %d; want 9 and 4",
			schoolAdminAuditCount, platformAdminAuditCount)
	}
}
