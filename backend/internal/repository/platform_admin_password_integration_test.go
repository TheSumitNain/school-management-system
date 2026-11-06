package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPlatformAdminPasswordChangePostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	actorID := ensurePlatformAdminLifecycleActor(t, pool)
	lifecycle := NewPlatformAdminLifecycleRepository(pool)
	email := fmt.Sprintf("password-change-%d@example.invalid", time.Now().UnixNano())
	admin, err := lifecycle.Create(ctx, actorID, email, "old-password-hash")
	if err != nil {
		t.Fatalf("create Platform Admin fixture: %v", err)
	}

	repo := NewPlatformAdminRepository(pool)
	oldHash, err := repo.GetPasswordHashByPrincipalID(ctx, admin.PrincipalID)
	if err != nil || oldHash != "old-password-hash" {
		t.Fatalf("load Platform Admin password hash = %q, err=%v", oldHash, err)
	}
	var tokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, admin.PrincipalID).Scan(&tokenVersion); err != nil {
		t.Fatalf("load initial token version: %v", err)
	}
	// Platform Admin refresh sessions require verified mandatory MFA. The
	// integration fixture only needs the repository's verified-state predicate.
	if _, err := pool.Exec(ctx, `
INSERT INTO principal_mfa_credentials
    (principal_id, method, secret_ciphertext, encryption_key_version, verified_at)
VALUES ($1, 'totp', $2, 1, NOW())`, admin.PrincipalID, make([]byte, 64)); err != nil {
		t.Fatalf("prepare verified Platform Admin MFA fixture: %v", err)
	}

	sessionRepo := NewPrincipalRefreshSessionRepository(pool)
	sessionHash := integrationTokenHash(email + "-active-session")
	if _, err := sessionRepo.CreatePrincipalForVersionWithFamily(ctx, admin.PrincipalID, tokenVersion, sessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create active refresh session: %v", err)
	}

	if err := repo.ChangePassword(ctx, admin.PrincipalID, oldHash, "new-password-hash"); err != nil {
		t.Fatalf("change Platform Admin password: %v", err)
	}
	newHash, err := repo.GetPasswordHashByPrincipalID(ctx, admin.PrincipalID)
	if err != nil || newHash != "new-password-hash" {
		t.Fatalf("new password hash = %q, err=%v", newHash, err)
	}
	var actualVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, admin.PrincipalID).Scan(&actualVersion); err != nil {
		t.Fatalf("load updated token version: %v", err)
	}
	if actualVersion != tokenVersion+1 {
		t.Fatalf("token version = %d, want %d", actualVersion, tokenVersion+1)
	}
	var activeSessions int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM refresh_sessions WHERE principal_id=$1 AND revoked_at IS NULL`, admin.PrincipalID).Scan(&activeSessions); err != nil {
		t.Fatalf("count active refresh sessions: %v", err)
	}
	if activeSessions != 0 {
		t.Fatalf("password change left %d active refresh sessions", activeSessions)
	}
	var auditCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM audit_events
WHERE action='platform.admin.password.changed' AND resource_id=$1
  AND actor_principal_id=$2 AND outcome='success'`, fmt.Sprint(admin.PrincipalID), admin.PrincipalID).Scan(&auditCount); err != nil {
		t.Fatalf("verify password-change audit event: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one password-change audit event, got %d", auditCount)
	}

	if err := repo.ChangePassword(ctx, admin.PrincipalID, oldHash, "stale-password-hash"); !errors.Is(err, ErrPlatformAdminPasswordConflict) {
		t.Fatalf("stale expected hash error = %v, want conflict", err)
	}
	newHash, err = repo.GetPasswordHashByPrincipalID(ctx, admin.PrincipalID)
	if err != nil || newHash != "new-password-hash" {
		t.Fatalf("stale password change modified credential: hash=%q err=%v", newHash, err)
	}
}
