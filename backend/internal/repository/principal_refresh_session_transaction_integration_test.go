package repository

import (
	"context"
	"crypto/sha256"
	"strconv"
	"encoding/hex"
	"testing"
	"time"
)

// TestPrincipalRefreshSessionCreateInTxPostgresIntegration proves that a
// principal-bound refresh session can be inserted through a caller-owned
// transaction, without committing that transaction internally.
func TestPrincipalRefreshSessionCreateInTxPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var schoolID, staffPrincipalID int64
	if err := pool.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &staffPrincipalID); err != nil {
		t.Fatalf("load integration school and staff principal: %v", err)
	}

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
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobile := "+9198765" + suffix[len(suffix)-5:]
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
		VALUES ($1, $2, NOW())
	`, principalID, mobile); err != nil {
		t.Fatalf("create verified parent account: %v", err)
	}

	var guardianID, studentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO guardians
			(school_id, full_name, mobile_number, mobile_number_e164,
			 parent_portal_enabled_at, parent_portal_enabled_by)
		VALUES ($1, $2, $3, $3, NOW(), $4)
		RETURNING id
	`, schoolID, "Refresh Transaction Guardian "+suffix, mobile, staffPrincipalID).Scan(&guardianID); err != nil {
		t.Fatalf("create eligible guardian: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "PARENT-REFRESH-"+suffix, "Refresh Transaction Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create active student: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_guardians
			(school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'mother')
	`, schoolID, studentID, guardianID); err != nil {
		t.Fatalf("create official guardian relationship: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO parent_guardian_links (principal_id, guardian_id, school_id, link_method)
		VALUES ($1, $2, $3, 'otp_verified')
	`, principalID, guardianID, schoolID); err != nil {
		t.Fatalf("create active Parent guardian link: %v", err)
	}

	sum := sha256.Sum256([]byte("parent-refresh-session-transaction-integration-" + suffix))
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
