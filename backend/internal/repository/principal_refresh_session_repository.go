package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PrincipalRefreshSessionRepository struct {
	db *pgxpool.Pool
}

func NewPrincipalRefreshSessionRepository(db *pgxpool.Pool) *PrincipalRefreshSessionRepository {
	return &PrincipalRefreshSessionRepository{db: db}
}

// Create binds a refresh session directly to auth_principals.
// It validates the principal and serializes creation with token rotation.
func (r *PrincipalRefreshSessionRepository) CreatePrincipal(
	ctx context.Context,
	principalID int64,
	tokenHash string,
	expiresAt time.Time,
) error {
	if principalID <= 0 || !isSHA256Hash(tokenHash) || !expiresAt.After(time.Now()) {
		return fmt.Errorf("invalid refresh session data")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refresh-session creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Use the same principal-level lock as refresh-token rotation.
	if _, err = tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		principalID,
	); err != nil {
		return fmt.Errorf("lock principal refresh sessions: %w", err)
	}

	const lookup = `
SELECT ap.principal_type, ap.status, ap.token_version,
       u.id, COALESCE(u.school_id,0), COALESCE(u.username,''), COALESCE(s.status,''),
       (pa.principal_id IS NOT NULL),
       COALESCE(parent.mobile_verified_at IS NOT NULL,FALSE)
FROM auth_principals ap
LEFT JOIN users u ON u.principal_id=ap.id
LEFT JOIN schools s ON s.id=u.school_id
LEFT JOIN platform_admin_accounts pa ON pa.principal_id=ap.id
LEFT JOIN parent_accounts parent ON parent.principal_id=ap.id
WHERE ap.id=$1
FOR UPDATE OF ap`

	var principalType string
	var principalStatus string
	var username string
	var schoolStatus string
	var tokenVersion int64
	var userID pgtype.Int8
	var schoolID int64
	var hasPlatformAccount bool
	var mobileVerified bool
	var mfaVerified bool

	err = tx.QueryRow(ctx, lookup, principalID).Scan(
		&principalType,
		&principalStatus,
		&tokenVersion,
		&userID,
		&schoolID,
		&username,
		&schoolStatus,
		&hasPlatformAccount,
		&mobileVerified,
		&mfaVerified,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRefreshSessionInactive
	}
	if err != nil {
		return fmt.Errorf("load principal for refresh-session creation: %w", err)
	}

	active := principalStatus == "active" && tokenVersion > 0 && mfaVerified

	switch principalType {
	case "staff":
		active = active &&
			schoolID > 0 &&
			schoolStatus == "active" &&
			username != ""

	case "platform_admin":
		active = active && hasPlatformAccount

	case "parent":
		active = active && mobileVerified

	default:
		active = false
	}

	if !active {
		return ErrRefreshSessionInactive
	}

	const insert = `
INSERT INTO refresh_sessions (user_id, principal_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4)`

	if _, err = tx.Exec(ctx, insert, nullableInt64(userID), principalID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh-session creation: %w", err)
	}

	return nil
}

// Rotate preserves family replay detection and cross-family principal locking.
// The replacement is inserted in the same transaction as revoking the old row.
func (r *PrincipalRefreshSessionRepository) RotatePrincipal(
	ctx context.Context,
	oldHash string,
	newHash string,
	newExpiry time.Time,
) (*models.AuthPrincipal, error) {
	if !isSHA256Hash(oldHash) ||
		!isSHA256Hash(newHash) ||
		oldHash == newHash ||
		!newExpiry.After(time.Now()) {
		return nil, ErrRefreshTokenInvalid
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin refresh-token rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var familyID string
	var principalID int64

	err = tx.QueryRow(
		ctx,
		`SELECT family_id, principal_id FROM refresh_sessions WHERE token_hash=$1`,
		oldHash,
	).Scan(&familyID, &principalID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("find refresh-token family: %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		familyID,
	); err != nil {
		return nil, fmt.Errorf("lock refresh-token family: %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
		principalID,
	); err != nil {
		return nil, fmt.Errorf("lock principal refresh sessions: %w", err)
	}

	const lookup = `
SELECT rs.id, rs.family_id, rs.principal_id,
       (rs.expires_at > NOW()),
       (rs.revoked_at IS NOT NULL),
       ap.principal_type, ap.status, ap.token_version,
       u.id,
       COALESCE(u.school_id,0),
       COALESCE(u.username,''),
       COALESCE(s.status,''),
       (pa.principal_id IS NOT NULL),
       COALESCE(parent.mobile_verified_at IS NOT NULL,FALSE),
       CASE
         WHEN ap.principal_type='platform_admin' OR
              (ap.principal_type='staff' AND EXISTS (
                SELECT 1 FROM principal_roles mpr JOIN roles mr ON mr.id=mpr.role_id
                WHERE mpr.principal_id=ap.id AND mpr.revoked_at IS NULL
                  AND mr.is_system=TRUE AND mr.role_key='school_admin'
                  AND mr.principal_type='staff' AND mr.scope_type='school'
                  AND mr.school_id=u.school_id
              ))
         THEN EXISTS (SELECT 1 FROM principal_mfa_credentials mc
                      WHERE mc.principal_id=ap.id AND mc.method='totp' AND mc.verified_at IS NOT NULL)
         ELSE TRUE
       END
FROM refresh_sessions rs
JOIN auth_principals ap ON ap.id=rs.principal_id
LEFT JOIN users u ON u.principal_id=ap.id
LEFT JOIN schools s ON s.id=u.school_id
LEFT JOIN platform_admin_accounts pa ON pa.principal_id=ap.id
LEFT JOIN parent_accounts parent ON parent.principal_id=ap.id
WHERE rs.token_hash=$1
FOR UPDATE OF rs, ap`

	var sessionID int64
	var currentFamily string
	var currentPrincipal int64
	var notExpired bool
	var revoked bool
	var principalType string
	var principalStatus string
	var username string
	var schoolStatus string
	var tokenVersion int64
	var userID pgtype.Int8
	var schoolID int64
	var hasPlatformAccount bool
	var mobileVerified bool

	err = tx.QueryRow(ctx, lookup, oldHash).Scan(
		&sessionID,
		&currentFamily,
		&currentPrincipal,
		&notExpired,
		&revoked,
		&principalType,
		&principalStatus,
		&tokenVersion,
		&userID,
		&schoolID,
		&username,
		&schoolStatus,
		&hasPlatformAccount,
		&mobileVerified,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRefreshTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("load refresh principal: %w", err)
	}

	if currentFamily != familyID || currentPrincipal != principalID {
		return nil, fmt.Errorf("refresh-session identity changed unexpectedly")
	}

	// A previously revoked token being reused revokes its entire family.
	if revoked {
		if _, err = tx.Exec(
			ctx,
			`UPDATE refresh_sessions
			 SET revoked_at=COALESCE(revoked_at,NOW())
			 WHERE family_id=$1 AND revoked_at IS NULL`,
			familyID,
		); err != nil {
			return nil, fmt.Errorf("revoke refresh family after reuse: %w", err)
		}

		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit refresh-token reuse: %w", err)
		}

		return nil, ErrRefreshTokenReuse
	}

	if !notExpired {
		return nil, ErrRefreshTokenInvalid
	}

	active := principalStatus == "active" && tokenVersion > 0

	switch principalType {
	case "staff":
		active = active &&
			schoolID > 0 &&
			schoolStatus == "active" &&
			username != ""

	case "platform_admin":
		active = active && hasPlatformAccount

	case "parent":
		active = active && mobileVerified

	default:
		active = false
	}

	if !active {
		if _, err = tx.Exec(
			ctx,
			`UPDATE refresh_sessions
			 SET revoked_at=COALESCE(revoked_at,NOW())
			 WHERE principal_id=$1 AND revoked_at IS NULL`,
			principalID,
		); err != nil {
			return nil, fmt.Errorf("revoke inactive principal sessions: %w", err)
		}

		if err = tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit inactive principal revocation: %w", err)
		}

		return nil, ErrRefreshSessionInactive
	}

	res, err := tx.Exec(
		ctx,
		`UPDATE refresh_sessions
		 SET revoked_at=NOW()
		 WHERE id=$1 AND revoked_at IS NULL`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("revoke previous refresh token: %w", err)
	}

	if res.RowsAffected() != 1 {
		return nil, ErrRefreshTokenInvalid
	}

	if _, err = tx.Exec(
		ctx,
		`INSERT INTO refresh_sessions
		 (user_id,principal_id,token_hash,expires_at,family_id)
		 VALUES ($1,$2,$3,$4,$5)`,
		nullableInt64(userID),
		principalID,
		newHash,
		newExpiry,
		familyID,
	); err != nil {
		return nil, fmt.Errorf("store replacement refresh token: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh-token rotation: %w", err)
	}

	return &models.AuthPrincipal{
		ID:             principalID,
		Type:           principalType,
		Status:         principalStatus,
		TokenVersion:   tokenVersion,
		SchoolID:       schoolID,
		Username:       username,
		MobileVerified: mobileVerified,
	}, nil
}

// Revoke revokes the entire token family and serializes with Rotate.
func (r *PrincipalRefreshSessionRepository) RevokePrincipal(
	ctx context.Context,
	tokenHash string,
) error {
	if !isSHA256Hash(tokenHash) {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin refresh-session revocation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var familyID string

	err = tx.QueryRow(
		ctx,
		`SELECT family_id FROM refresh_sessions WHERE token_hash=$1`,
		tokenHash,
	).Scan(&familyID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find refresh-token family for revocation: %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		familyID,
	); err != nil {
		return fmt.Errorf("lock refresh-token family for revocation: %w", err)
	}

	if _, err = tx.Exec(
		ctx,
		`UPDATE refresh_sessions
		 SET revoked_at=COALESCE(revoked_at,NOW())
		 WHERE family_id=$1 AND revoked_at IS NULL`,
		familyID,
	); err != nil {
		return fmt.Errorf("revoke refresh-token family: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh-token family revocation: %w", err)
	}

	return nil
}

// nullableInt64 returns SQL NULL for non-staff principals, which do not have
// a corresponding row in users.
func nullableInt64(value pgtype.Int8) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}
