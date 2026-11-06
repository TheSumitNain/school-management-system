package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPlatformAdminPasswordRecoveryPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	actorID := ensurePlatformAdminLifecycleActor(t, pool)
	lifecycle := NewPlatformAdminLifecycleRepository(pool)
	email := fmt.Sprintf("password-recovery-%d@example.invalid", time.Now().UnixNano())
	admin, err := lifecycle.Create(ctx, actorID, email, "old-password-hash")
	if err != nil {
		t.Fatalf("create Platform Admin fixture: %v", err)
	}

	var tokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, admin.PrincipalID).Scan(&tokenVersion); err != nil {
		t.Fatalf("load initial token version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO principal_mfa_credentials
    (principal_id, method, secret_ciphertext, encryption_key_version, verified_at)
VALUES ($1, 'totp', $2, 1, NOW())`, admin.PrincipalID, make([]byte, 64)); err != nil {
		t.Fatalf("prepare verified Platform Admin MFA fixture: %v", err)
	}

	sessionRepo := NewPrincipalRefreshSessionRepository(pool)
	sessionHash := integrationTokenHash(email + "-recovery-session")
	if _, err := sessionRepo.CreatePrincipalForVersionWithFamily(ctx, admin.PrincipalID, tokenVersion, sessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create active refresh session: %v", err)
	}

	repo := NewPlatformAdminRepository(pool)
	recoveredID, err := repo.RecoverPasswordByEmail(ctx, email, "recovered-password-hash")
	if err != nil {
		t.Fatalf("recover Platform Admin password: %v", err)
	}
	if recoveredID != admin.PrincipalID {
		t.Fatalf("recovered principal ID = %d, want %d", recoveredID, admin.PrincipalID)
	}

	newHash, err := repo.GetPasswordHashByPrincipalID(ctx, admin.PrincipalID)
	if err != nil || newHash != "recovered-password-hash" {
		t.Fatalf("recovered password hash = %q, err=%v", newHash, err)
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
		t.Fatalf("password recovery left %d active refresh sessions", activeSessions)
	}

	var auditCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM audit_events
WHERE action='platform.admin.password.recovered' AND resource_id=$1
  AND actor_principal_id IS NULL AND outcome='success'`, fmt.Sprint(admin.PrincipalID)).Scan(&auditCount); err != nil {
		t.Fatalf("verify password-recovery audit event: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one password-recovery audit event, got %d", auditCount)
	}

	if _, err := repo.RecoverPasswordByEmail(ctx, "missing-platform-admin@example.invalid", "unused-hash"); !errors.Is(err, ErrPlatformAdminNotFound) {
		t.Fatalf("unknown Platform Admin recovery error = %v, want not found", err)
	}
}
