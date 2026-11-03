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

// TestExactDelegationRevocationV16PostgresIntegration proves that a revoked
// delegation does not reactivate a custom-role grant or an ALLOW override when
// a new delegation is issued for the same school/permission pair.
func TestExactDelegationRevocationV16PostgresIntegration(t *testing.T) {
	_, pool, fixtureUserID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV16 bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='role_permissions'
			  AND column_name='delegation_id'
		) AND EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='user_permission_overrides'
			  AND column_name='delegation_id'
		)
	`).Scan(&hasV16); err != nil {
		t.Fatalf("detect V16 exact delegation linkage: %v", err)
	}
	if !hasV16 {
		t.Skip("V16 is not applied; apply migrations to school_management_test before running this integration test")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin V16 delegation integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, schoolAdminPrincipalID int64
	if err := tx.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id=$1
	`, fixtureUserID).Scan(&schoolID, &schoolAdminPrincipalID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	var platformAdminID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		FROM auth_principals ap
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		WHERE ap.principal_type='platform_admin'
		  AND ap.status='active'
		  AND r.scope_type='platform'
		  AND r.role_key='platform_admin'
		  AND r.is_system=TRUE
		ORDER BY ap.id
		LIMIT 1
	`).Scan(&platformAdminID)
	if errors.Is(err, pgx.ErrNoRows) {
		var anyBootstrap bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1 FROM principal_roles pr
			        JOIN roles r ON r.id=pr.role_id
			        WHERE r.principal_type='platform_admin'
			          AND r.scope_type='platform'
			          AND r.role_key='platform_admin'
			    )
		`).Scan(&anyBootstrap); err != nil {
			t.Fatalf("check existing Platform Admin bootstrap: %v", err)
		}
		if anyBootstrap {
			t.Skip("Platform Admin bootstrap history exists but no active Platform Admin is available for delegation integration tests")
		}
		repo := NewPlatformAdminRepository(pool)
		platformAdminID, err = repo.bootstrapFirstInTx(
			ctx, tx,
			fmt.Sprintf("v16-delegation-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
		if err != nil {
			t.Fatalf("bootstrap transactional Platform Admin fixture: %v", err)
		}
	} else if err != nil {
		t.Fatalf("load active Platform Admin fixture: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		WHERE school_id=$1 AND scope_type='school'
		  AND principal_type='staff' AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1,$2,$3)
	`, schoolAdminPrincipalID, schoolAdminRoleID, platformAdminID); err != nil {
		t.Fatalf("assign School Admin fixture role: %v", err)
	}

	suffix := time.Now().UnixNano()
	var targetPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1,$2,$3,'active')
		RETURNING principal_id
	`, schoolID, fmt.Sprintf("v16-target-%d", suffix), "integration-test-password-hash").Scan(&targetPrincipalID); err != nil {
		t.Fatalf("create target staff principal: %v", err)
	}

	var customRoleID, rolePermissionID, overridePermissionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO roles
		    (school_id, principal_type, scope_type, role_key, name, is_system, created_by)
		VALUES ($1,'staff','school',$2,'V16 Delegation Test Role',FALSE,$3)
		RETURNING id
	`, schoolID, fmt.Sprintf("v16_custom_%d", suffix), schoolAdminPrincipalID).Scan(&customRoleID); err != nil {
		t.Fatalf("create custom role fixture: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id FROM permissions WHERE permission_key='students.read_basic'
	`).Scan(&rolePermissionID); err != nil {
		t.Fatalf("load delegable role permission: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id FROM permissions WHERE permission_key='homework.create'
	`).Scan(&overridePermissionID); err != nil {
		t.Fatalf("load delegable override permission: %v", err)
	}

	createDelegation := func(permissionID int64, assign, override bool) int64 {
		t.Helper()
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO role_permission_delegations
			    (school_id, permission_id, can_assign_to_roles, can_override_users, granted_by)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING id
		`, schoolID, permissionID, assign, override, platformAdminID).Scan(&id); err != nil {
			t.Fatalf("create exact permission delegation: %v", err)
		}
		return id
	}

	firstRoleDelegation := createDelegation(rolePermissionID, true, false)
	if _, err := tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, delegation_id)
		VALUES ($1,$2,$3)
	`, customRoleID, rolePermissionID, firstRoleDelegation); err != nil {
		t.Fatalf("grant custom-role permission using exact delegation: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1,$2,$3)
	`, targetPrincipalID, customRoleID, schoolAdminPrincipalID); err != nil {
		t.Fatalf("assign custom role to target staff: %v", err)
	}

	resolver := NewPrincipalAuthorizationRepository(pool)
	target := &models.AuthPrincipal{
		ID: targetPrincipalID, Type: "staff", Status: "active",
		TokenVersion: 1, SchoolID: schoolID,
	}
	assertPermission := func(permission string, want bool) {
		t.Helper()
		permissions, err := loadEffectivePermissions(ctx, tx, target)
		if err != nil {
			t.Fatalf("resolve effective permissions: %v", err)
		}
		found := false
		for _, value := range permissions {
			if value == permission {
				found = true
				break
			}
		}
		if found != want {
			t.Fatalf("permission %q present=%v, want %v (resolved=%v)", permission, found, want, permissions)
		}
	}
	_ = resolver // The transaction-aware helper is used so every fixture rolls back.

	assertPermission("students.read_basic", true)
	if _, err := tx.Exec(ctx, `
		UPDATE role_permission_delegations SET revoked_at=NOW() WHERE id=$1
	`, firstRoleDelegation); err != nil {
		t.Fatalf("revoke original custom-role delegation: %v", err)
	}
	assertPermission("students.read_basic", false)

	secondRoleDelegation := createDelegation(rolePermissionID, true, false)
	assertPermission("students.read_basic", false)
	if _, err := tx.Exec(ctx, `
		UPDATE role_permissions SET delegation_id=$3
		WHERE role_id=$1 AND permission_id=$2
	`, customRoleID, rolePermissionID, secondRoleDelegation); err != nil {
		t.Fatalf("explicitly rebind custom-role grant to new delegation: %v", err)
	}
	assertPermission("students.read_basic", true)

	firstOverrideDelegation := createDelegation(overridePermissionID, false, true)
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_permission_overrides
		    (principal_id, school_id, permission_id, effect, reason, granted_by, delegation_id)
		VALUES ($1,$2,$3,'allow','V16 integration test',$4,$5)
	`, targetPrincipalID, schoolID, overridePermissionID, schoolAdminPrincipalID, firstOverrideDelegation); err != nil {
		t.Fatalf("create exact-delegation ALLOW override: %v", err)
	}
	assertPermission("homework.create", true)

	if _, err := tx.Exec(ctx, `
		UPDATE role_permission_delegations SET revoked_at=NOW() WHERE id=$1
	`, firstOverrideDelegation); err != nil {
		t.Fatalf("revoke original user-override delegation: %v", err)
	}
	assertPermission("homework.create", false)

	secondOverrideDelegation := createDelegation(overridePermissionID, false, true)
	assertPermission("homework.create", false)
	if _, err := tx.Exec(ctx, `
		UPDATE user_permission_overrides SET delegation_id=$4
		WHERE principal_id=$1 AND school_id=$2 AND permission_id=$3
	`, targetPrincipalID, schoolID, overridePermissionID, secondOverrideDelegation); err != nil {
		t.Fatalf("explicitly rebind ALLOW override to new delegation: %v", err)
	}
	assertPermission("homework.create", true)
}
