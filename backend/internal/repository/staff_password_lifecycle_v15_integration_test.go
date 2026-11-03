package repository

import (
	"context"
	"fmt"
	"testing"
)

// TestStaffPasswordLifecycleV15PostgresIntegration verifies that V15 preserves
// existing accounts while new staff accounts default to a required password
// change. The shared fixture is created after migrations and therefore exercises
// the production INSERT default.
func TestStaffPasswordLifecycleV15PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasColumn bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'users'
			  AND column_name = 'must_change_password'
			  AND is_nullable = 'NO'
		)
	`).Scan(&hasColumn); err != nil {
		t.Fatalf("detect V15 staff password lifecycle column: %v", err)
	}
	if !hasColumn {
		t.Skip("V15 is not applied; apply migrations to school_management_test before running this integration test")
	}

	var defaultValue string
	if err := pool.QueryRow(ctx, `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'users'
		  AND column_name = 'must_change_password'
	`).Scan(&defaultValue); err != nil {
		t.Fatalf("read staff password lifecycle default: %v", err)
	}
	if defaultValue != "true" && defaultValue != "true::boolean" {
		t.Fatalf("must_change_password default = %q, want true", defaultValue)
	}

	var required bool
	if err := pool.QueryRow(ctx, `
		SELECT must_change_password FROM users WHERE id = $1
	`, userID).Scan(&required); err != nil {
		t.Fatalf("read newly provisioned staff password state: %v", err)
	}
	if !required {
		t.Fatal("new staff account must require a password change")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin existing-account compatibility transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE users SET must_change_password = FALSE WHERE id = $1
	`, userID); err != nil {
		t.Fatalf("mark existing account as already using a permanent password: %v", err)
	}
	newUsername := fmt.Sprintf("v15-%d", userID)
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		SELECT school_id, $2, password_hash, status
		FROM users WHERE id = $1
	`, userID, newUsername); err != nil {
		t.Fatalf("insert a new staff account using database defaults: %v", err)
	}
	var newAccountRequired bool
	if err := tx.QueryRow(ctx, `
		SELECT must_change_password FROM users WHERE username = $1
	`, newUsername).Scan(&newAccountRequired); err != nil {
		t.Fatalf("read second staff account password state: %v", err)
	}
	if !newAccountRequired {
		t.Fatal("new staff account did not inherit must_change_password=true")
	}
	var existingStillFalse bool
	if err := tx.QueryRow(ctx, `
		SELECT NOT must_change_password FROM users WHERE id = $1
	`, userID).Scan(&existingStillFalse); err != nil {
		t.Fatalf("verify existing account retained its permanent-password state: %v", err)
	}
	if !existingStillFalse {
		t.Fatal("changing the default for new accounts must not alter existing account state")
	}
}
