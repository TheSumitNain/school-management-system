package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestChangeStaffPasswordPostgresIntegration verifies the password change's
// single-transaction effects: credential replacement, temporary-flag clearing,
// access-token version increment, refresh-session revocation, and audit append.
func TestChangeStaffPasswordPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV15 bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='users'
			  AND column_name='must_change_password'
		)
	`).Scan(&hasV15); err != nil {
		t.Fatalf("detect V15 staff password state: %v", err)
	}
	if !hasV15 {
		t.Skip("V15 is not applied; apply migrations to school_management_test first")
	}

	var principalID int64
	var oldHash string
	var oldTokenVersion int64
	var mustChange bool
	if err := pool.QueryRow(ctx, `
		SELECT u.principal_id, u.password_hash, ap.token_version, u.must_change_password
		FROM users u JOIN auth_principals ap ON ap.id=u.principal_id
		WHERE u.id=$1
	`, userID).Scan(&principalID, &oldHash, &oldTokenVersion, &mustChange); err != nil {
		t.Fatalf("load staff password fixture: %v", err)
	}
	if !mustChange {
		t.Fatal("new staff fixture must start with must_change_password=true")
	}

	refreshHash := integrationTokenHash("staff-password-change-session")
	sessionRepo := NewPrincipalRefreshSessionRepository(pool)
	if err := sessionRepo.CreatePrincipal(ctx, principalID, refreshHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create refresh session before password change: %v", err)
	}

	repo := NewUserRepository(pool)
	if err := repo.ChangeStaffPassword(ctx, principalID, oldHash, "integration-new-password-hash"); err != nil {
		t.Fatalf("change staff password: %v", err)
	}

	var storedHash string
	var stillMustChange bool
	var tokenVersion int64
	if err := pool.QueryRow(ctx, `
		SELECT u.password_hash, u.must_change_password, ap.token_version
		FROM users u JOIN auth_principals ap ON ap.id=u.principal_id
		WHERE u.id=$1
	`, userID).Scan(&storedHash, &stillMustChange, &tokenVersion); err != nil {
		t.Fatalf("verify changed staff credential: %v", err)
	}
	if storedHash != "integration-new-password-hash" || stillMustChange || tokenVersion != oldTokenVersion+1 {
		t.Fatalf("password state mismatch: hash_changed=%v must_change=%v token_version=%d want=%d",
			storedHash == "integration-new-password-hash", stillMustChange, tokenVersion, oldTokenVersion+1)
	}

	var revoked bool
	if err := pool.QueryRow(ctx, `
		SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash=$1
	`, refreshHash).Scan(&revoked); err != nil {
		t.Fatalf("verify refresh-session revocation: %v", err)
	}
	if !revoked {
		t.Fatal("password change must revoke every existing refresh session")
	}

	var auditCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		WHERE actor_principal_id=$1 AND action='staff.password.changed'
		  AND resource_type='staff_user' AND resource_id=$2
	`, principalID, fmt.Sprint(userID)).Scan(&auditCount); err != nil {
		t.Fatalf("verify staff password change audit event: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("password change audit count = %d, want 1", auditCount)
	}
}
