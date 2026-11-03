package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestFirstPlatformAdminBootstrapV14PostgresIntegration verifies that the
// database permits exactly one NULL-actor assignment to the platform_admin
// system role and rejects a second bootstrap attempt.
func TestFirstPlatformAdminBootstrapV14PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var hasV14 bool
	if err := pool.QueryRow(ctx, `
		SELECT POSITION('7414' IN pg_get_functiondef(
			'public.validate_principal_role_assignment()'::regprocedure
		)) > 0
	`).Scan(&hasV14); err != nil {
		t.Fatalf("detect V14 Platform Admin bootstrap guard: %v", err)
	}
	if !hasV14 {
		t.Skip("V14 is not applied; apply migrations to school_management_test before running this integration test")
	}

	var alreadyBootstrapped bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
		    OR EXISTS (
		        SELECT 1 FROM principal_roles pr
		        JOIN roles r ON r.id = pr.role_id
		        WHERE r.role_key = 'platform_admin'
		          AND r.principal_type = 'platform_admin'
		          AND r.scope_type = 'platform'
		    )
	`).Scan(&alreadyBootstrapped); err != nil {
		t.Fatalf("check existing Platform Admin bootstrap: %v", err)
	}
	if alreadyBootstrapped {
		t.Skip("a Platform Admin already exists in school_management_test; one-time bootstrap test requires an unbootstrapped test database")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin V14 integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	t.Run("repository bootstrap transaction and audit", func(t *testing.T) {
		bootstrapTx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin repository bootstrap transaction: %v", err)
		}
		defer func() { _ = bootstrapTx.Rollback(ctx) }()

		email := fmt.Sprintf("bootstrap-repository-%d@example.invalid", time.Now().UnixNano())
		const passwordHash = "integration-test-password-hash"
		repo := NewPlatformAdminRepository(pool)
		principalID, err := repo.bootstrapFirstInTx(ctx, bootstrapTx, email, passwordHash)
		if err != nil {
			t.Fatalf("run repository first-admin bootstrap: %v", err)
		}
		if _, err := bootstrapTx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
			t.Fatalf("validate bootstrapped principal subtype: %v", err)
		}

		var storedPrincipalID int64
		if err := bootstrapTx.QueryRow(ctx, `
			SELECT principal_id FROM platform_admin_accounts WHERE LOWER(BTRIM(email)) = $1
		`, email).Scan(&storedPrincipalID); err != nil {
			t.Fatalf("load bootstrapped Platform Admin: %v", err)
		}
		var roleCount, auditCount int
		if err := bootstrapTx.QueryRow(ctx, `
			SELECT COUNT(*) FROM principal_roles pr
			JOIN roles r ON r.id = pr.role_id
			WHERE pr.principal_id = $1 AND pr.revoked_at IS NULL
			  AND r.role_key = 'platform_admin' AND r.scope_type = 'platform'
		`, principalID).Scan(&roleCount); err != nil {
			t.Fatalf("verify Platform Admin role assignment: %v", err)
		}
		if err := bootstrapTx.QueryRow(ctx, `
			SELECT COUNT(*) FROM audit_events
			WHERE action = 'platform.admin.bootstrapped'
			  AND resource_id = $1 AND metadata->>'email' = $2
		`, fmt.Sprint(principalID), email).Scan(&auditCount); err != nil {
			t.Fatalf("verify bootstrap audit event: %v", err)
		}
		if storedPrincipalID != principalID || roleCount != 1 || auditCount != 1 {
			t.Fatalf("bootstrap mismatch: principal=%d stored=%d roles=%d audit=%d", principalID, storedPrincipalID, roleCount, auditCount)
		}
		if _, err := repo.bootstrapFirstInTx(ctx, bootstrapTx, "second-"+email, passwordHash); err != ErrPlatformAdminAlreadyBootstrapped {
			t.Fatalf("expected second bootstrap rejection, got %v", err)
		}
	})

	if _, err := tx.Exec(ctx, "SAVEPOINT first_platform_admin"); err != nil {
		t.Fatalf("create first bootstrap savepoint: %v", err)
	}

	createAccount := func(label string) (int64, error) {
		var principalID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO auth_principals (principal_type, status)
			VALUES ('platform_admin', 'active')
			RETURNING id
		`).Scan(&principalID); err != nil {
			return 0, err
		}
		email := fmt.Sprintf("v14-%s-%d@example.invalid", label, time.Now().UnixNano())
		if _, err := tx.Exec(ctx, `
			INSERT INTO platform_admin_accounts
			    (principal_id, email, password_hash, must_change_password)
			VALUES ($1, $2, $3, FALSE)
		`, principalID, email, "integration-test-hash"); err != nil {
			return 0, err
		}
		return principalID, nil
	}

	firstID, err := createAccount("first")
	if err != nil {
		t.Fatalf("create first Platform Admin subtype: %v", err)
	}
	var roleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE role_key = 'platform_admin'
		   AND principal_type = 'platform_admin'
		   AND scope_type = 'platform'
		   AND is_system = TRUE
	`).Scan(&roleID); err != nil {
		t.Fatalf("load Platform Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, NULL)
	`, firstID, roleID); err != nil {
		t.Fatalf("first Platform Admin bootstrap assignment rejected: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE"); err != nil {
		t.Fatalf("validate first Platform Admin subtype: %v", err)
	}

	if _, err := tx.Exec(ctx, "SAVEPOINT second_platform_admin"); err != nil {
		t.Fatalf("create second bootstrap savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET CONSTRAINTS ct_principal_subtype_principal, ct_principal_subtype_platform DEFERRED"); err != nil {
		t.Fatalf("defer subtype constraints for second account: %v", err)
	}
	secondID, err := createAccount("second")
	if err != nil {
		t.Fatalf("create second Platform Admin subtype: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, NULL)
	`, secondID, roleID); err == nil {
		t.Fatal("second NULL-actor Platform Admin assignment unexpectedly succeeded")
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT second_platform_admin"); err != nil {
		t.Fatalf("rollback second bootstrap attempt: %v", err)
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT first_platform_admin"); err != nil {
		t.Fatalf("rollback first bootstrap test data: %v", err)
	}
}
