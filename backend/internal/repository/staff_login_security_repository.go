package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StaffLoginSecurityRepository persists lockout state across API instances.
type StaffLoginSecurityRepository struct {
	db *pgxpool.Pool
}

func NewStaffLoginSecurityRepository(db *pgxpool.Pool) *StaffLoginSecurityRepository {
	return &StaffLoginSecurityRepository{db: db}
}

func loginIdentifierHash(username string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(username))))
	return hex.EncodeToString(sum[:])
}

// IsLocked checks the normalized username's active lockout window.
func (r *StaffLoginSecurityRepository) IsLocked(ctx context.Context, username string) (bool, error) {
	if strings.TrimSpace(username) == "" {
		return false, nil
	}
	const query = `
SELECT EXISTS (
    SELECT 1 FROM auth_login_attempts
     WHERE identifier_hash = $1 AND locked_until > NOW()
)`
	var locked bool
	if err := r.db.QueryRow(ctx, query, loginIdentifierHash(username)).Scan(&locked); err != nil {
		return false, fmt.Errorf("check staff login lockout: %w", err)
	}
	return locked, nil
}

// RecordFailure atomically increments the rolling 15-minute counter and audits
// the failure. The fifth failure starts a 15-minute lockout.
func (r *StaffLoginSecurityRepository) RecordFailure(ctx context.Context, username string) error {
	if strings.TrimSpace(username) == "" {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin staff login failure: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const upsert = `
INSERT INTO auth_login_attempts
    (identifier_hash, failed_attempts, window_started_at, last_attempt_at, updated_at)
VALUES ($1, 1, NOW(), NOW(), NOW())
ON CONFLICT (identifier_hash) DO UPDATE SET
    failed_attempts = CASE
        WHEN auth_login_attempts.window_started_at <= NOW() - INTERVAL '15 minutes' THEN 1
        ELSE LEAST(auth_login_attempts.failed_attempts + 1, 5)
    END,
    window_started_at = CASE
        WHEN auth_login_attempts.window_started_at <= NOW() - INTERVAL '15 minutes' THEN NOW()
        ELSE auth_login_attempts.window_started_at
    END,
    locked_until = CASE
        WHEN auth_login_attempts.window_started_at <= NOW() - INTERVAL '15 minutes' THEN NULL
        WHEN auth_login_attempts.failed_attempts >= 4 THEN COALESCE(auth_login_attempts.locked_until, NOW() + INTERVAL '15 minutes')
        ELSE NULL
    END,
    last_attempt_at = NOW(),
    updated_at = NOW()
RETURNING failed_attempts`
	var attempts int
	if err := tx.QueryRow(ctx, upsert, loginIdentifierHash(username)).Scan(&attempts); err != nil {
		return fmt.Errorf("record staff login failure: %w", err)
	}
	metadata, _ := json.Marshal(map[string]string{
		"identifier_hash": loginIdentifierHash(username),
		"reason":          "invalid_credentials",
	})
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (action, resource_type, outcome, metadata)
VALUES ('auth.staff_login', 'authentication', 'failure', $1::jsonb)`, string(metadata)); err != nil {
		return fmt.Errorf("audit staff login failure: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit staff login failure: %w", err)
	}
	return nil
}

// RecordSuccess clears the failure window and writes a successful login event.
func (r *StaffLoginSecurityRepository) RecordSuccess(ctx context.Context, username string, principalID int64) error {
	if strings.TrimSpace(username) == "" || principalID <= 0 {
		return fmt.Errorf("invalid successful staff login identity")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin successful staff login: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM auth_login_attempts WHERE identifier_hash = $1`, loginIdentifierHash(username)); err != nil {
		return fmt.Errorf("clear staff login failures: %w", err)
	}
	metadata, _ := json.Marshal(map[string]string{"identifier_hash": loginIdentifierHash(username)})
	if _, err := tx.Exec(ctx, `
INSERT INTO audit_events (actor_principal_id, action, resource_type, outcome, metadata)
VALUES ($1, 'auth.staff_login', 'authentication', 'success', $2::jsonb)`, principalID, string(metadata)); err != nil {
		return fmt.Errorf("audit successful staff login: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit successful staff login: %w", err)
	}
	return nil
}
