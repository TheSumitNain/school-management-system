package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrParentOTPRateLimited = errors.New("parent OTP send rate limit exceeded")
	ErrParentOTPResendWait  = errors.New("parent OTP resend cooldown has not elapsed")
)

const (
	parentOTPChallengeTTL = 5 * time.Minute
	parentOTPResendWait   = 60 * time.Second
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
// inserts the new challenge. Rate limits use a rolling 15-minute window.
// mobileHash, ipHash, and otpHash must be lowercase SHA-256 HMAC hex strings.
// Delivery must happen only after this succeeds.
func (r *ParentOTPChallengeRepository) CreatePreAccountChallenge(
	ctx context.Context,
	mobileHash, ipHash, otpHash string,
	expiresAt time.Time,
) (int64, error) {
	now := time.Now()
	if r == nil || r.db == nil || !isSHA256Hash(mobileHash) ||
		!isSHA256Hash(ipHash) || !isSHA256Hash(otpHash) ||
		!expiresAt.After(now) || expiresAt.After(now.Add(parentOTPChallengeTTL)) {
		return 0, fmt.Errorf("invalid parent OTP challenge data")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin parent OTP challenge: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Every request acquires locks in the same order. This serializes sends
	// sharing either identifier and prevents concurrent limit bypass.
	for _, lockKey := range []string{"mobile:" + mobileHash, "ip:" + ipHash} {
		if _, err := tx.Exec(ctx,
			`SELECT pg_advisory_xact_lock(hashtextextended('parent-otp-rate:' || $1, 0))`,
			lockKey,
		); err != nil {
			return 0, fmt.Errorf("lock parent OTP rate bucket: %w", err)
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

	if err := recordParentOTPSend(ctx, tx, "mobile", mobileHash); err != nil {
		return 0, err
	}
	if err := recordParentOTPSend(ctx, tx, "ip", ipHash); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit parent OTP challenge: %w", err)
	}
	return challengeID, nil
}

func readParentOTPRateState(
	ctx context.Context, tx pgx.Tx, subjectType, subjectHash string,
) (int16, pgtype.Timestamptz, bool, error) {
	var count int16
	var lastSent pgtype.Timestamptz
	var blocked bool
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(send_count), 0)::SMALLINT,
		        MAX(last_sent_at),
		        COALESCE(BOOL_OR(blocked_until > NOW()), FALSE)
		   FROM parent_otp_rate_limits
		  WHERE subject_type=$1
		    AND subject_hash=$2
		    AND window_started_at > NOW() - INTERVAL '15 minutes'`,
		subjectType, subjectHash,
	).Scan(&count, &lastSent, &blocked)
	if err != nil {
		return 0, pgtype.Timestamptz{}, false, fmt.Errorf("read parent OTP rate state: %w", err)
	}
	return count, lastSent, blocked, nil
}

func recordParentOTPSend(ctx context.Context, tx pgx.Tx, subjectType, subjectHash string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO parent_otp_rate_limits
		    (subject_type, subject_hash, window_started_at, send_count, last_sent_at)
		  VALUES ($1, $2, NOW(), 1, NOW())`,
		subjectType, subjectHash,
	)
	if err != nil {
		return fmt.Errorf("record parent OTP send: %w", err)
	}
	return nil
}
