package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRefreshTokenInvalid    = errors.New("invalid or expired refresh token")
	ErrRefreshTokenReuse      = errors.New("refresh token reuse detected")
	ErrRefreshSessionInactive = errors.New("user or school is inactive")
)

// RefreshSessionRepository persists refresh-token hashes, never raw tokens.
type RefreshSessionRepository struct {
	db *pgxpool.Pool
}

func NewRefreshSessionRepository(db *pgxpool.Pool) *RefreshSessionRepository {
	return &RefreshSessionRepository{db: db}
}

func (r *RefreshSessionRepository) Create(
	ctx context.Context,
	userID int64,
	tokenHash string,
	expiresAt time.Time,
) error {
	if userID <= 0 || !isSHA256Hash(tokenHash) || !expiresAt.After(time.Now()) {
		return fmt.Errorf("invalid refresh session data")
	}

	const query = `
		INSERT INTO refresh_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	if _, err := r.db.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

// Rotate atomically revokes the presented token and inserts its replacement.
// Reuse of a previously revoked token revokes every still-active token in its family.
func (r *RefreshSessionRepository) Rotate(
	ctx context.Context,
	oldTokenHash string,
	newTokenHash string,
	newExpiresAt time.Time,
) (*models.User, error) {
	if !isSHA256Hash(oldTokenHash) || !isSHA256Hash(newTokenHash) ||
		oldTokenHash == newTokenHash || !newExpiresAt.After(time.Now()) {
		return nil, ErrRefreshTokenInvalid
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin refresh-token rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize operations within one rotation family. The first lookup only
	// discovers the immutable family ID; all security decisions use the fresh,
	// row-locked lookup after the family lock is acquired.
	var familyID string
	if err := tx.QueryRow(
		ctx,
		`SELECT family_id FROM refresh_sessions WHERE token_hash = $1`,
		oldTokenHash,
	).Scan(&familyID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenInvalid
	} else if err != nil {
		return nil, fmt.Errorf("find refresh-token family: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		familyID,
	); err != nil {
		return nil, fmt.Errorf("lock refresh-token family: %w", err)
	}

	const lookup = `
		SELECT rs.id, rs.user_id, rs.family_id, (rs.expires_at > NOW()),
		       (rs.revoked_at IS NOT NULL),
		       u.school_id, u.username, u.status, s.status
		FROM refresh_sessions AS rs
		JOIN users AS u ON u.id = rs.user_id
		JOIN schools AS s ON s.id = u.school_id
		WHERE rs.token_hash = $1
		FOR UPDATE OF rs
	`

	var sessionID, userID, schoolID int64
	var currentFamilyID string
	var notExpired, revoked bool
	var username, userStatus, schoolStatus string

	err = tx.QueryRow(ctx, lookup, oldTokenHash).Scan(
		&sessionID,
		&userID,
		&currentFamilyID,
		&notExpired,
		&revoked,
		&schoolID,
		&username,
		&userStatus,
		&schoolStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh session: %w", err)
	}

	if currentFamilyID != familyID {
		return nil, fmt.Errorf("refresh-token family changed unexpectedly")
	}

	if revoked {
		const revokeFamily = `
			UPDATE refresh_sessions
			SET revoked_at = COALESCE(revoked_at, NOW())
			WHERE family_id = $1 AND revoked_at IS NULL
		`
		if _, err := tx.Exec(ctx, revokeFamily, familyID); err != nil {
			return nil, fmt.Errorf("revoke refresh-token family after reuse: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit refresh-token reuse response: %w", err)
		}
		return nil, ErrRefreshTokenReuse
	}

	if !notExpired {
		return nil, ErrRefreshTokenInvalid
	}

	if userStatus != "active" || schoolStatus != "active" {
		const revokeUserSessions = `
			UPDATE refresh_sessions
			SET revoked_at = COALESCE(revoked_at, NOW())
			WHERE user_id = $1 AND revoked_at IS NULL
		`
		if _, err := tx.Exec(ctx, revokeUserSessions, userID); err != nil {
			return nil, fmt.Errorf("revoke inactive user's sessions: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit inactive-session revocation: %w", err)
		}
		return nil, ErrRefreshSessionInactive
	}

	const revokeOld = `
		UPDATE refresh_sessions
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`
	result, err := tx.Exec(ctx, revokeOld, sessionID)
	if err != nil {
		return nil, fmt.Errorf("revoke previous refresh token: %w", err)
	}
	if result.RowsAffected() != 1 {
		return nil, ErrRefreshTokenInvalid
	}

	const insertReplacement = `
		INSERT INTO refresh_sessions (user_id, token_hash, expires_at, family_id)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertReplacement, userID, newTokenHash, newExpiresAt, familyID); err != nil {
		return nil, fmt.Errorf("store replacement refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh-token rotation: %w", err)
	}

	return &models.User{
		ID:       userID,
		SchoolID: schoolID,
		Username: username,
		Status:   userStatus,
	}, nil
}

// Revoke invalidates a refresh token. Unknown tokens are intentionally a no-op,
// allowing logout to be safely retried without revealing token state.
func (r *RefreshSessionRepository) Revoke(
	ctx context.Context,
	tokenHash string,
) error {
	if !isSHA256Hash(tokenHash) {
		return nil
	}

	const query = `
		UPDATE refresh_sessions
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	if _, err := r.db.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	return nil
}

func isSHA256Hash(value string) bool {
	if len(value) != 64 || strings.Trim(value, "0123456789abcdef") != "" {
		return false
	}
	return true
}
