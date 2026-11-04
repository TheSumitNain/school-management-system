package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestLoginSessionIssuanceRejectsStalePrincipalVersionPostgresIntegration
// models the critical interleaving where password authentication observes
// token_version 1, a password change commits, and login then attempts to store
// its refresh session. The stale login must fail even though the principal
// remains active. A login authenticated against the new version must succeed.
// All database writes are enclosed in one rollback-only transaction.
func TestLoginSessionIssuanceRejectsStalePrincipalVersionPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped login/password race test: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	username := "login-race-" + suffix[len(suffix)-10:]
	var userID, principalID, initialTokenVersion int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		SELECT school_id, $2, $3, 'active'
		  FROM users
		 ORDER BY id
		 LIMIT 1
		RETURNING id, principal_id
	`, username, "login-race-old-password-hash").Scan(&userID, &principalID); err != nil {
		t.Fatalf("create rollback-scoped login race staff account: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT token_version FROM auth_principals WHERE id = $1
	`, principalID).Scan(&initialTokenVersion); err != nil {
		t.Fatalf("read initial login race token version: %v", err)
	}
	if initialTokenVersion <= 0 {
		t.Fatalf("invalid initial token version: %d", initialTokenVersion)
	}

	sessions := NewPrincipalRefreshSessionRepository(pool)
	initialSessionHash := integrationTokenHash("login-password-race-initial-" + suffix)
	if err := sessions.CreatePrincipalInTx(ctx, tx, principalID, initialSessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create initial refresh session: %v", err)
	}

	if err := NewUserRepository(tx).ChangeStaffPassword(
		ctx, principalID, "login-race-old-password-hash", "login-race-new-password-hash",
	); err != nil {
		t.Fatalf("commit password change inside rollback boundary: %v", err)
	}

	var currentTokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT token_version FROM auth_principals WHERE id = $1
	`, principalID).Scan(&currentTokenVersion); err != nil {
		t.Fatalf("read token version after password change: %v", err)
	}
	if currentTokenVersion != initialTokenVersion+1 {
		t.Fatalf("token version after password change = %d, want %d", currentTokenVersion, initialTokenVersion+1)
	}

	staleSessionHash := integrationTokenHash("login-password-race-stale-" + suffix)
	err = sessions.CreatePrincipalForVersionInTx(
		ctx, tx, principalID, initialTokenVersion, staleSessionHash, time.Now().Add(time.Hour),
	)
	if !errors.Is(err, ErrPrincipalVersionChanged) {
		t.Fatalf("stale login session creation error = %v, want ErrPrincipalVersionChanged", err)
	}
	var staleSessionCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_sessions WHERE token_hash = $1
	`, staleSessionHash).Scan(&staleSessionCount); err != nil {
		t.Fatalf("verify stale login did not persist a refresh session: %v", err)
	}
	if staleSessionCount != 0 {
		t.Fatalf("stale login persisted %d refresh sessions", staleSessionCount)
	}

	currentSessionHash := integrationTokenHash("login-password-race-current-" + suffix)
	if err := sessions.CreatePrincipalForVersionInTx(
		ctx, tx, principalID, currentTokenVersion, currentSessionHash, time.Now().Add(time.Hour),
	); err != nil {
		t.Fatalf("current-version login session creation failed: %v", err)
	}
	var currentSessionCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_sessions
		 WHERE token_hash = $1 AND revoked_at IS NULL
	`, currentSessionHash).Scan(&currentSessionCount); err != nil {
		t.Fatalf("verify current-version refresh session: %v", err)
	}
	if currentSessionCount != 1 {
		t.Fatalf("current-version login created %d active sessions, want 1", currentSessionCount)
	}

	var targetUserStillExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND principal_id = $2)
	`, userID, principalID).Scan(&targetUserStillExists); err != nil {
		t.Fatalf("verify rollback-scoped staff fixture: %v", err)
	}
	if !targetUserStillExists {
		t.Fatal("login/password race test unexpectedly lost its staff fixture")
	}
}
