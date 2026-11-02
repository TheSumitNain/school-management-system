package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

// TestPrincipalRefreshSessionCreateInTxPostgresIntegration proves that a
// principal-bound refresh session can be inserted through a caller-owned
// transaction, without committing that transaction internally.
func TestPrincipalRefreshSessionCreateInTxPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var principalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type, status)
		VALUES ('parent', 'active')
		RETURNING id
	`).Scan(&principalID); err != nil {
		t.Fatalf("create parent principal: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		VALUES ($1, '+919876543210', NOW())
	`, principalID); err != nil {
		t.Fatalf("create verified parent account: %v", err)
	}

	sum := sha256.Sum256([]byte("parent-refresh-session-transaction-integration"))
	tokenHash := hex.EncodeToString(sum[:])
	repository := NewPrincipalRefreshSessionRepository(pool)

	if err := repository.CreatePrincipalInTx(ctx, tx, principalID, tokenHash, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("create parent refresh session in caller transaction: %v", err)
	}

	var sessions int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM refresh_sessions
		WHERE principal_id = $1 AND token_hash = $2
	`, principalID, tokenHash).Scan(&sessions); err != nil {
		t.Fatalf("verify refresh session inside transaction: %v", err)
	}
	if sessions != 1 {
		t.Fatalf("expected one uncommitted refresh session, got %d", sessions)
	}
}
