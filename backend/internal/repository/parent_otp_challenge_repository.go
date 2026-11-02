package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrParentOTPRateLimited = errors.New("parent OTP send rate limit exceeded")
	ErrParentOTPResendWait  = errors.New("parent OTP resend cooldown has not elapsed")
)

const (
	parentOTPSendWindow = 15 * time.Minute
	parentOTPResendWait = 60 * time.Second
	parentOTPMobileSendLimit = 3
	parentOTPIPSendLimit = 10
)

// ParentOTPChallengeRepository persists pre-account OTP challenges and their
// send limits. It stores only keyed hashes; callers must never pass raw mobile,
// IP, or OTP values to this repository.
type ParentOTPChallengeRepository struct {
	db *pgxpool.Pool
}

func NewParentOTPChallengeRepository(db *pgxpool.Pool) *ParentOTPChallengeRepository {
	return &ParentOTPChallengeRepository{db: db}
}

// CreatePreAccountChallenge atomically reserves both mobile and IP send
// budgets, supersedes any prior unconsumed challenge for the mobile, and
// inserts the new challenge. mobileHash, ipHash, and otpHash must be lowercase
// SHA-256 HMAC hex strings. Delivery must happen only after this succeeds.
func (r *ParentOTPChallengeRepository) CreatePreAccountChallenge(
	ctx context.Context,
	mobileHash, ipHash, otpHash string,
	expiresAt time.Time,
) (int64, error) {
	if r == nil || r.db == nil || !isSHA256Hash(mobileHash) ||
		!isSHA256Hash(ipHash) || !isSHA256Hash(otpHash) ||
		!expiresAt.After(time.Now()) || expiresAt.After(time.Now().Add(6*time.Minute)) {
		return 0, fmt.Errorf("invalid parent OTP challenge data")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin parent OTP challenge: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Every request acquires locks in the same order, avoiding mobile/IP
	// lock-order inversions when many clients share either identifier.
	for _, lockKey := range []string{"mobile:" + mobileHash, "ip:" + ipHash} {
		if _, err := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtextextended('parent-otp-rate:' || $1, 0))`,
			lockKey,
		); err != nil {
			return 0, fmt.Errorf("lock parent OTP rate bucket: %w", err)
		}
	}

	var windowStarted time.Time
	if err := tx.QueryRow(ctx,
		`SELECT date_bin($1::interval, NOW(), TIMESTAMPTZ '2000-01-01 00:00:00+00')`,
		parentOTPSendWindow.String(),
	).Scan(&windowStarted); err != nil {
		return 0, fmt.Errorf("calculate parent OTP rate window: %w", err)
	}

	mobileCount, mobileLastSent, mobileBlocked, err := lockParentOTPRateBucket(ctx, tx, "mobile", mobileHash, windowStarted)
	if err != nil {
		return 0, err
	}
	ipCount, _, ipBlocked, err := lockParentOTPRateBucket(ctx, tx, "ip", ipHash, windowStarted)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	if mobileBlocked || ipBlocked || mobileCount >= parentOTPMobileSendLimit || ipCount >= parentOTPIPSendLimit {
		return 0, ErrParentOTPRateLimited
	}
	if mobileLastSent.Valid && now.Sub(mobileLastSent.Time) < parentOTPResendWait {
		return 0, ErrParentOTPResendWait
	}

	if _, err := tx.Exec(ctx,
		`UPDATE parent_otp_challenges
		    SET consumed_at = NOW()
		  WHERE mobile_lookup_hash = $1 AND consumed_at IS NULL`,
		mobileHash,
	); err != nil {
		return 0, fmt.Errorf("supersede previous parent OTP challenge: %w", err)
	}

	var challengeID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO parent_otp_challenges
		    (principal_id, mobile_lookup_hash, otp_hash, expires_at)
		  VALUES (NULL, $1, $2, $3)
		  RETURNING id`,
		mobileHash, otpHash, expiresAt,
	).Scan(&challengeID); err != nil {
		return 0, fmt.Errorf("insert parent OTP challenge: %w", err)
	}

	if err := updateParentOTPRateBucket(ctx, tx, "mobile", mobileHash, windowStarted); err != nil {
		return 0, err
	}
	if err := updateParentOTPRateBucket(ctx, tx, "ip", ipHash, windowStarted); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit parent OTP challenge: %w", err)
	}
	return challengeID, nil
}

func lockParentOTPRateBucket(
	ctx context.Context, tx pgx.Tx, subjectType, subjectHash string, windowStarted time.Time,
) (int16, pgtype.Timestamptz, bool, error) {
	if _, err := tx.Exec(ctx,
		`INSERT INTO parent_otp_rate_limits
		    (subject_type, subject_hash, window_started_at, send_count)
		  VALUES ($1, $2, $3, 0)
		  ON CONFLICT (subject_type, subject_hash, window_started_at) DO NOTHING`,
		subjectType, subjectHash, windowStarted,
	); err != nil {
		return 0, pgtypeTimestamp{}, false, fmt.Errorf("initialize parent OTP rate bucket: %w", err)
	}

	var count int16
	var lastSent pgtypeTimestamp
	var blockedUntil pgtypeTimestamp
	if err := tx.QueryRow(ctx,
		`SELECT send_count, last_sent_at, blocked_until
		   FROM parent_otp_rate_limits
		  WHERE subject_type=$1 AND subject_hash=$2 AND window_started_at=$3
		  FOR UPDATE`,
		subjectType, subjectHash, windowStarted,
	).Scan(&count, &lastSent, &blockedUntil); err != nil {
		return 0, pgtypeTimestamp{}, false, fmt.Errorf("lock parent OTP rate bucket: %w", err)
	}
	return count, lastSent, blockedUntil.Valid && blockedUntil.Time.After(time.Now()), nil
}

func updateParentOTPRateBucket(ctx context.Context, tx pgx.Tx, subjectType, subjectHash string, windowStarted time.Time) error {
	if _, err := tx.Exec(ctx,
		`UPDATE parent_otp_rate_limits
		    SET send_count = send_count + 1,
		        last_sent_at = NOW(),
		        updated_at = NOW()
		  WHERE subject_type=$1 AND subject_hash=$2 AND window_started_at=$3`,
		subjectType, subjectHash, windowStarted,
	); err != nil {
		return fmt.Errorf("increment parent OTP rate bucket: %w", err)
	}
	return nil
}
