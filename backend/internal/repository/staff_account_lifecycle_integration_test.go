package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestAuthorizedStaffAccountLifecyclePostgresIntegration exercises successful
// School Admin staff lifecycle operations. All writes, including the temporary
// Platform Admin bootstrap (when needed), run inside a caller-owned transaction
// and are rolled back so the test never leaves a permanent administrator or
// staff account in school_management_test.
func TestAuthorizedStaffAccountLifecyclePostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped staff lifecycle test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id, school_id
		  FROM users
		 WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load staff lifecycle actor: %v", err)
	}

	// Reuse an already bootstrapped Platform Admin if the dedicated test DB has
	// one. Otherwise create the first administrator only inside this transaction.
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
		   AND p.permission_key = 'platform.school_admins.create'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		var anyPlatformAdminExists bool
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
		`).Scan(&anyPlatformAdminExists); checkErr != nil {
			t.Fatalf("check existing Platform Admin bootstrap state: %v", checkErr)
		}
		if anyPlatformAdminExists {
			t.Skip("an existing Platform Admin is not active with the required onboarding permission")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("staff-lifecycle-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}

	// V13 provisions the School Admin system role when the integration fixture
	// creates its school. V14 independently requires a Platform Admin actor for
	// this reserved role assignment.
	if err := tx.QueryRow(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		SELECT $1, r.id, $2
		  FROM roles r
		 WHERE r.school_id = $3
		   AND r.principal_type = 'staff'
		   AND r.scope_type = 'school'
		   AND r.role_key = 'school_admin'
		   AND r.is_system = TRUE
		RETURNING id
	`, actorPrincipalID, platformAdminPrincipalID, schoolID).Scan(new(int64)); err != nil {
		t.Fatalf("assign temporary School Admin role to test actor: %v", err)
	}

	// pgx.Tx implements the same repository DB contract as pgxpool.Pool.
	// Repository Begin calls become nested savepoints, so successful method
	// commits remain inside this outer rollback boundary.
	repo := NewUserRepository(tx)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	username := "lifecycle-" + suffix
	passwordHash := "initial-integration-password-hash"

	created, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID, username, passwordHash)
	if err != nil {
		t.Fatalf("authorized staff account creation failed: %v", err)
	}
	if created == nil || created.PrincipalID <= 0 || created.SchoolID != schoolID ||
		created.Username != username || created.Status != "active" || !created.MustChangePassword {
		t.Fatalf("unexpected created staff account: %+v", created)
	}

	listed, err := repo.ListStaffAccounts(ctx, actorPrincipalID, schoolID)
	if err != nil {
		t.Fatalf("authorized staff account listing failed: %v", err)
	}
	found := false
	for _, user := range listed {
		if user.PrincipalID == created.PrincipalID {
			found = true
			if user.Username != username || user.Status != "active" || !user.MustChangePassword {
				t.Fatalf("listed staff account differs from created account: %+v", user)
			}
		}
	}
	if !found {
		t.Fatalf("new staff account principal %d was absent from the same-school listing", created.PrincipalID)
	}

	renamedUsername := "renamed-" + suffix
	updated, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, renamedUsername)
	if err != nil {
		t.Fatalf("authorized staff username update failed: %v", err)
	}
	if updated.Username != renamedUsername {
		t.Fatalf("updated username = %q, want %q", updated.Username, renamedUsername)
	}

	if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID); err != nil {
		t.Fatalf("authorized staff account disable failed: %v", err)
	}
	var userStatus, principalStatus string
	if err := tx.QueryRow(ctx, `
		SELECT u.status, ap.status
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(&userStatus, &principalStatus); err != nil {
		t.Fatalf("read disabled staff account state: %v", err)
	}
	if userStatus != "disabled" || principalStatus != "disabled" {
		t.Fatalf("disable status mismatch: users=%q principal=%q", userStatus, principalStatus)
	}

	const replacementPasswordHash = "replacement-temporary-password-hash"
	if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, replacementPasswordHash); err != nil {
		t.Fatalf("authorized staff account re-enable failed: %v", err)
	}
	var finalUsername, finalUserStatus, finalPrincipalStatus, finalPasswordHash string
	var mustChangePassword bool
	var tokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT u.username, u.status, ap.status, u.password_hash,
		       u.must_change_password, ap.token_version
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(
		&finalUsername, &finalUserStatus, &finalPrincipalStatus, &finalPasswordHash,
		&mustChangePassword, &tokenVersion,
	); err != nil {
		t.Fatalf("read re-enabled staff account state: %v", err)
	}
	if finalUsername != renamedUsername || finalUserStatus != "active" ||
		finalPrincipalStatus != "active" || finalPasswordHash != replacementPasswordHash ||
		!mustChangePassword || tokenVersion != 4 {
		t.Fatalf("unexpected re-enabled staff state: username=%q user=%q principal=%q must_change=%v token_version=%d",
			finalUsername, finalUserStatus, finalPrincipalStatus, mustChangePassword, tokenVersion)
	}

	var auditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM audit_events
		 WHERE actor_principal_id = $1
		   AND school_id = $2
		   AND resource_type = 'staff_user'
		   AND resource_id = $3
		   AND action IN (
		       'staff.account.created',
		       'staff.account.updated',
		       'staff.account.disabled',
		       'staff.account.enabled'
		   )
		   AND outcome = 'success'
	`, actorPrincipalID, schoolID, fmt.Sprint(created.ID)).Scan(&auditCount); err != nil {
		t.Fatalf("count staff lifecycle audit events: %v", err)
	}
	if auditCount != 4 {
		t.Fatalf("expected 4 staff lifecycle audit events, got %d", auditCount)
	}
}
