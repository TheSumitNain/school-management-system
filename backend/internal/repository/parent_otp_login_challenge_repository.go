package repository

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreatePrincipalChallenge creates an OTP challenge bound to an existing,
// active, mobile-verified Parent principal. The canonical mobile is checked
// against parent_accounts inside the same transaction; only its HMAC is stored.
func (r *ParentOTPChallengeRepository) CreatePrincipalChallenge(
	ctx context.Context,
	principalID int64,
	mobileE164, mobileHash, ipHash, otpHash string,
	expiresAt time.Time,
) (int64, error) {
	now := time.Now()
	if r == nil || r.db == nil || principalID <= 0 ||
		!parentE164PatternForRepository(mobileE164) ||
		!isSHA256Hash(mobileHash) || !isSHA256Hash(ipHash) ||
		!isSHA256Hash(otpHash) || !expiresAt.After(now) ||
		expiresAt.After(now.Add(parentOTPChallengeTTL)) {
		return 0, fmt.Errorf("invalid principal-bound parent OTP challenge data")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin principal-bound parent OTP challenge: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM auth_principals ap
			  JOIN parent_accounts pa ON pa.principal_id = ap.id
			 WHERE ap.id = $1
			   AND ap.principal_type = 'parent'
			   AND ap.status = 'active'
			   AND pa.mobile_number = $2
			   AND pa.mobile_verified_at IS NOT NULL
		)
	`, principalID, mobileE164).Scan(&eligible); err != nil {
		return 0, fmt.Errorf("validate Parent identity for OTP challenge: %w", err)
	}
	if !eligible {
		return 0, fmt.Errorf("Parent identity is not eligible for OTP authentication")
	}

	// Match the established mobile-then-IP lock order used by registration OTP.
	for _, lockKey := range []string{"mobile:" + mobileHash, "ip:" + ipHash} {
		if _, err := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtextextended('parent-otp-rate:' || $1, 0))`,
			lockKey,
		); err != nil {
			return 0, fmt.Errorf("lock Parent OTP rate bucket: %w", err)
		}
	}

	mobileCount, mobileLastSent, mobileBlocked, err := readParentOTPRateState(ctx, tx, "mobile", mobileHash)
	if err != nil {
		return 0, err
	}
	ipCount, _, ipBlocked, err := readParentOTPRateState(ctx, tx, "ip", ipHash)
	if err != nil {
		return 0, err
	}
	if mobileBlocked || ipBlocked || mobileCount >= parentOTPMobileSendLimit || ipCount >= parentOTPIPSendLimit {
		return 0, ErrParentOTPRateLimited
	}
	if mobileLastSent.Valid && time.Since(mobileLastSent.Time) < parentOTPResendWait {
		return 0, ErrParentOTPResendWait
	}

	// A new challenge supersedes every active challenge for this Parent or
	// mobile, including any stale pre-account challenge for the same mobile.
	if _, err := tx.Exec(ctx, `
		UPDATE parent_otp_challenges
		   SET consumed_at = NOW()
		 WHERE consumed_at IS NULL
		   AND (principal_id = $1 OR mobile_lookup_hash = $2)
	`, principalID, mobileHash); err != nil {
		return 0, fmt.Errorf("supersede previous Parent OTP challenge: %w", err)
	}

	var challengeID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO parent_otp_challenges
		    (principal_id, mobile_lookup_hash, otp_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, principalID, mobileHash, otpHash, expiresAt).Scan(&challengeID); err != nil {
		return 0, fmt.Errorf("insert principal-bound Parent OTP challenge: %w", err)
	}

	if err := recordParentOTPSend(ctx, tx, "mobile", mobileHash); err != nil {
		return 0, err
	}
	if err := recordParentOTPSend(ctx, tx, "ip", ipHash); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit principal-bound Parent OTP challenge: %w", err)
	}
	return challengeID, nil
}

// VerifyPrincipalChallenge atomically verifies and consumes only the latest
// active challenge bound to the supplied Parent principal and mobile HMAC.
func (r *ParentOTPChallengeRepository) VerifyPrincipalChallenge(
	ctx context.Context, principalID int64, mobileHash, candidateOTPHash string,
) (bool, error) {
	if r == nil || r.db == nil || principalID <= 0 ||
		!isSHA256Hash(mobileHash) || !isSHA256Hash(candidateOTPHash) {
		return false, fmt.Errorf("invalid principal-bound Parent OTP verification data")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin principal-bound Parent OTP verification: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var challengeID int64
	var storedOTPHash string
	var expired bool
	var attemptCount int16
	err = tx.QueryRow(ctx, `
		SELECT id, otp_hash, (expires_at <= NOW()), attempt_count
		  FROM parent_otp_challenges
		 WHERE principal_id = $1
		   AND mobile_lookup_hash = $2
		   AND consumed_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT 1
		 FOR UPDATE
	`, principalID, mobileHash).Scan(&challengeID, &storedOTPHash, &expired, &attemptCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load principal-bound Parent OTP challenge: %w", err)
	}

	if expired || attemptCount >= parentOTPMaxAttempts {
		if _, err := tx.Exec(ctx,
			`UPDATE parent_otp_challenges SET consumed_at = NOW()
			  WHERE id = $1 AND consumed_at IS NULL`, challengeID); err != nil {
			return false, fmt.Errorf("consume expired Parent OTP challenge: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit expired Parent OTP challenge: %w", err)
		}
		return false, nil
	}

	if hmac.Equal([]byte(storedOTPHash), []byte(candidateOTPHash)) {
		tag, err := tx.Exec(ctx,
			`UPDATE parent_otp_challenges SET consumed_at = NOW()
			  WHERE id = $1 AND consumed_at IS NULL`, challengeID)
		if err != nil {
			return false, fmt.Errorf("consume verified Parent OTP challenge: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return false, fmt.Errorf("Parent OTP challenge changed during verification")
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit verified Parent OTP challenge: %w", err)
		}
		return true, nil
	}

	nextAttempt := attemptCount + 1
	consume := nextAttempt >= parentOTPMaxAttempts
	tag, err := tx.Exec(ctx, `
		UPDATE parent_otp_challenges
		   SET attempt_count = $2,
		       consumed_at = CASE WHEN $3::BOOLEAN THEN NOW() ELSE consumed_at END
		 WHERE id = $1 AND consumed_at IS NULL
	`, challengeID, nextAttempt, consume)
	if err != nil {
		return false, fmt.Errorf("record failed Parent OTP attempt: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("Parent OTP challenge changed during failed verification")
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit failed Parent OTP attempt: %w", err)
	}
	return false, nil
}

// InvalidatePrincipalChallenge consumes a challenge after delivery failure.
// The principal predicate prevents cleanup from touching another account's row.
func (r *ParentOTPChallengeRepository) InvalidatePrincipalChallenge(
	ctx context.Context, principalID, challengeID int64,
) error {
	if r == nil || r.db == nil || principalID <= 0 || challengeID <= 0 {
		return fmt.Errorf("invalid principal-bound Parent OTP challenge identity")
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE parent_otp_challenges
		   SET consumed_at = COALESCE(consumed_at, NOW())
		 WHERE id = $1 AND principal_id = $2
	`, challengeID, principalID)
	if err != nil {
		return fmt.Errorf("invalidate undelivered Parent OTP challenge: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("principal-bound Parent OTP challenge not found")
	}
	return nil
}
