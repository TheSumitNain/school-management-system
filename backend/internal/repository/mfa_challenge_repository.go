package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MFAChallengePurpose string

const (
	MFAChallengeEnrollment MFAChallengePurpose = "enrollment"
	MFAChallengeLogin      MFAChallengePurpose = "login"
	MFAChallengeRecoveryRegeneration MFAChallengePurpose = "recovery_regeneration"
	maxMFAChallengeAttempts                     = 5
)

var ErrMFAChallengeUnavailable = errors.New("MFA challenge is invalid, expired, consumed, or locked")

type MFAChallenge struct {
	PrincipalID int64
	Purpose     MFAChallengePurpose
	ExpiresAt   time.Time
	Attempts    int
	ConsumedAt  *time.Time
}

type MFAChallengeRepository struct {
	db *pgxpool.Pool
}

func NewMFAChallengeRepository(db *pgxpool.Pool) *MFAChallengeRepository {
	return &MFAChallengeRepository{db: db}
}

// Create atomically replaces any previous unconsumed challenge for the same
// principal and purpose. The raw bearer token is never stored.
func (r *MFAChallengeRepository) Create(
	ctx context.Context,
	principalID int64,
	purpose MFAChallengePurpose,
	challengeHash string,
	expiresAt time.Time,
) error {
	if principalID <= 0 || !validMFAChallengePurpose(purpose) ||
		!isSHA256Hash(challengeHash) || !expiresAt.After(time.Now()) ||
		expiresAt.After(time.Now().Add(15*time.Minute)) {
		return fmt.Errorf("invalid MFA challenge data")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin MFA challenge creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	lockKey := fmt.Sprintf("mfa-challenge:%d:%s", principalID, purpose)
	if _, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey,
	); err != nil {
		return fmt.Errorf("lock MFA challenge creation: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE auth_mfa_challenges
		    SET consumed_at = NOW()
		  WHERE principal_id = $1 AND purpose = $2 AND consumed_at IS NULL`,
		principalID, string(purpose),
	); err != nil {
		return fmt.Errorf("supersede previous MFA challenge: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO auth_mfa_challenges (principal_id, challenge_hash, purpose, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		principalID, challengeHash, string(purpose), expiresAt,
	); err != nil {
		return fmt.Errorf("store MFA challenge: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit MFA challenge: %w", err)
	}
	return nil
}

func (r *MFAChallengeRepository) Get(
	ctx context.Context,
	challengeHash string,
) (*MFAChallenge, error) {
	if !isSHA256Hash(challengeHash) {
		return nil, ErrMFAChallengeUnavailable
	}
	var challenge MFAChallenge
	var consumedAt *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT principal_id, purpose, expires_at, attempts, consumed_at
		   FROM auth_mfa_challenges WHERE challenge_hash = $1`,
		challengeHash,
	).Scan(&challenge.PrincipalID, &challenge.Purpose, &challenge.ExpiresAt,
		&challenge.Attempts, &consumedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMFAChallengeUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("load MFA challenge: %w", err)
	}
	challenge.ConsumedAt = consumedAt
	if challenge.ConsumedAt != nil || !challenge.ExpiresAt.After(time.Now()) ||
		challenge.Attempts >= maxMFAChallengeAttempts {
		return nil, ErrMFAChallengeUnavailable
	}
	return &challenge, nil
}

// RecordFailure atomically counts a rejected code and permanently consumes
// the challenge on its fifth failed attempt.
func (r *MFAChallengeRepository) RecordFailure(ctx context.Context, challengeHash string) error {
	if !isSHA256Hash(challengeHash) {
		return ErrMFAChallengeUnavailable
	}
	var attempts int
	err := r.db.QueryRow(ctx,
		`UPDATE auth_mfa_challenges
		    SET attempts = attempts + 1,
		        consumed_at = CASE WHEN attempts + 1 >= $2 THEN NOW() ELSE consumed_at END
		  WHERE challenge_hash = $1
		    AND consumed_at IS NULL
		    AND expires_at > NOW()
		    AND attempts < $2
		  RETURNING attempts`,
		challengeHash, maxMFAChallengeAttempts,
	).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMFAChallengeUnavailable
	}
	if err != nil {
		return fmt.Errorf("record MFA challenge failure: %w", err)
	}
	return nil
}

// Consume succeeds once, and only while the challenge is live and below its
// failed-attempt limit. Call only after validating the supplied TOTP code.
func (r *MFAChallengeRepository) Consume(
	ctx context.Context,
	challengeHash string,
) (int64, MFAChallengePurpose, error) {
	if !isSHA256Hash(challengeHash) {
		return 0, "", ErrMFAChallengeUnavailable
	}
	var principalID int64
	var purpose MFAChallengePurpose
	err := r.db.QueryRow(ctx,
		`UPDATE auth_mfa_challenges
		    SET consumed_at = NOW()
		  WHERE challenge_hash = $1
		    AND consumed_at IS NULL
		    AND expires_at > NOW()
		    AND attempts < $2
		  RETURNING principal_id, purpose`,
		challengeHash, maxMFAChallengeAttempts,
	).Scan(&principalID, &purpose)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrMFAChallengeUnavailable
	}
	if err != nil {
		return 0, "", fmt.Errorf("consume MFA challenge: %w", err)
	}
	return principalID, purpose, nil
}

func validMFAChallengePurpose(purpose MFAChallengePurpose) bool {
	return purpose == MFAChallengeEnrollment || purpose == MFAChallengeLogin || purpose == MFAChallengeRecoveryRegeneration
}
