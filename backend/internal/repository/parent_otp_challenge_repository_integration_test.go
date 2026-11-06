package repository

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestParentOTPChallengeRepositoryPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	repo := NewParentOTPChallengeRepository(pool)

	unique := strconv.FormatInt(time.Now().UnixNano(), 10)
	mobileHash := integrationTokenHash("parent-otp-mobile-" + unique)
	ipHash := integrationTokenHash("parent-otp-ip-" + unique)
	otpHash := integrationTokenHash("parent-otp-code-" + unique)

	challengeID, err := repo.CreatePreAccountChallenge(
		ctx, mobileHash, ipHash, otpHash, time.Now().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("create pre-account challenge: %v", err)
	}
	if challengeID <= 0 {
		t.Fatalf("challenge ID = %d, want positive ID", challengeID)
	}

	var principalID pgtype.Int8
	var storedMobileHash, storedOTPHash string
	var attemptCount int16
	if err := pool.QueryRow(ctx, `
		SELECT principal_id, mobile_lookup_hash, otp_hash, attempt_count
		  FROM parent_otp_challenges WHERE id=$1
	`, challengeID).Scan(&principalID, &storedMobileHash, &storedOTPHash, &attemptCount); err != nil {
		t.Fatalf("read persisted pre-account challenge: %v", err)
	}
	if principalID.Valid {
		t.Fatal("pre-account challenge unexpectedly reserved a principal")
	}
	if storedMobileHash != mobileHash || storedOTPHash != otpHash || attemptCount != 0 {
		t.Fatal("persisted challenge did not retain the expected HMAC hashes and zero attempts")
	}

	var mobileSends, ipSends int16
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type='mobile' AND subject_hash=$1
	`, mobileHash).Scan(&mobileSends); err != nil {
		t.Fatalf("read mobile rate bucket: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type='ip' AND subject_hash=$1
	`, ipHash).Scan(&ipSends); err != nil {
		t.Fatalf("read IP rate bucket: %v", err)
	}
	if mobileSends != 1 || ipSends != 1 {
		t.Fatalf("send counts mobile=%d IP=%d, want 1 each", mobileSends, ipSends)
	}

	_, err = repo.CreatePreAccountChallenge(
		ctx, mobileHash, ipHash, integrationTokenHash("parent-otp-second-"+unique), time.Now().Add(5*time.Minute),
	)
	if !errors.Is(err, ErrParentOTPResendWait) {
		t.Fatalf("immediate resend error = %v, want ErrParentOTPResendWait", err)
	}
	var challengeCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM parent_otp_challenges
		 WHERE mobile_lookup_hash=$1 AND consumed_at IS NULL
	`, mobileHash).Scan(&challengeCount); err != nil {
		t.Fatalf("count active challenges: %v", err)
	}
	if challengeCount != 1 {
		t.Fatalf("active challenge count = %d, want 1", challengeCount)
	}

	// Move only this test's send timestamps past the cooldown while keeping
	// them inside the rolling 15-minute window.
	for send := 2; send <= parentOTPMobileSendLimit; send++ {
		if _, err := pool.Exec(ctx, `
			UPDATE parent_otp_rate_limits
			   SET window_started_at=window_started_at-INTERVAL '61 seconds',
			    last_sent_at=last_sent_at-INTERVAL '61 seconds'
			 WHERE subject_hash IN ($1, $2)
		`, mobileHash, ipHash); err != nil {
			t.Fatalf("age rate-limit timestamps before send %d: %v", send, err)
		}
		_, err := repo.CreatePreAccountChallenge(
			ctx, mobileHash, ipHash,
			integrationTokenHash("parent-otp-code-"+unique+"-"+strconv.Itoa(send)),
			time.Now().Add(5*time.Minute),
		)
		if err != nil {
			t.Fatalf("create allowed challenge %d: %v", send, err)
		}
	}

	if _, err := pool.Exec(ctx, `
		UPDATE parent_otp_rate_limits
		   SET window_started_at=window_started_at-INTERVAL '61 seconds',
		       last_sent_at=last_sent_at-INTERVAL '61 seconds'
		 WHERE subject_hash IN ($1, $2)
	`, mobileHash, ipHash); err != nil {
		t.Fatalf("age rate-limit timestamps before over-limit send: %v", err)
	}
	_, err = repo.CreatePreAccountChallenge(
		ctx, mobileHash, ipHash,
		integrationTokenHash("parent-otp-code-over-limit-"+unique),
		time.Now().Add(5*time.Minute),
	)
	if !errors.Is(err, ErrParentOTPRateLimited) {
		t.Fatalf("fourth mobile send error = %v, want ErrParentOTPRateLimited", err)
	}

	// Delivery-failure cleanup consumes the challenge but does not refund
	// either durable rate-limit reservation.
	invalidateMobileHash := integrationTokenHash("parent-otp-invalidate-mobile-" + unique)
	invalidateIPHash := integrationTokenHash("parent-otp-invalidate-ip-" + unique)
	invalidateOTPHash := integrationTokenHash("parent-otp-invalidate-code-" + unique)
	undeliveredID, err := repo.CreatePreAccountChallenge(
		ctx, invalidateMobileHash, invalidateIPHash, invalidateOTPHash,
		time.Now().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("create challenge for delivery-failure invalidation: %v", err)
	}
	if err := repo.InvalidatePreAccountChallenge(ctx, undeliveredID); err != nil {
		t.Fatalf("invalidate undelivered challenge: %v", err)
	}

	var consumedAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT consumed_at FROM parent_otp_challenges WHERE id=$1`,
		undeliveredID,
	).Scan(&consumedAt); err != nil {
		t.Fatalf("read invalidated challenge: %v", err)
	}
	if consumedAt.IsZero() {
		t.Fatal("undelivered challenge remains active after invalidation")
	}

	var invalidatedMobileSends, invalidatedIPSends int16
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type='mobile' AND subject_hash=$1
	`, invalidateMobileHash).Scan(&invalidatedMobileSends); err != nil {
		t.Fatalf("read invalidated mobile rate bucket: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type='ip' AND subject_hash=$1
	`, invalidateIPHash).Scan(&invalidatedIPSends); err != nil {
		t.Fatalf("read invalidated IP rate bucket: %v", err)
	}
	if invalidatedMobileSends != 1 || invalidatedIPSends != 1 {
		t.Fatalf("delivery-failure cleanup refunded rate limits: mobile=%d IP=%d",
			invalidatedMobileSends, invalidatedIPSends)
	}

	// A failed guess increments the durable counter. The fifth attempt remains
	// usable when correct; a wrong fifth attempt permanently consumes the row.
	verifyMobileHash := integrationTokenHash("parent-otp-verify-mobile-" + unique)
	verifyOTPHash := integrationTokenHash("parent-otp-verify-code-" + unique)
	wrongOTPHash := integrationTokenHash("parent-otp-verify-wrong-" + unique)
	verifyID, err := repo.CreatePreAccountChallenge(
		ctx, verifyMobileHash, integrationTokenHash("parent-otp-verify-ip-"+unique),
		verifyOTPHash, time.Now().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("create challenge for verification: %v", err)
	}
	for attempt := 1; attempt <= 4; attempt++ {
		verified, verifyErr := repo.VerifyPreAccountChallenge(ctx, verifyMobileHash, wrongOTPHash)
		if verifyErr != nil || verified {
			t.Fatalf("wrong attempt %d: verified=%v error=%v", attempt, verified, verifyErr)
		}
	}
	verified, err := repo.VerifyPreAccountChallenge(ctx, verifyMobileHash, verifyOTPHash)
	if err != nil || !verified {
		t.Fatalf("correct fifth attempt: verified=%v error=%v", verified, err)
	}
	var verifyAttempts int16
	var verifyConsumed bool
	if err := pool.QueryRow(ctx, `
		SELECT attempt_count, consumed_at IS NOT NULL
		  FROM parent_otp_challenges WHERE id=$1
	`, verifyID).Scan(&verifyAttempts, &verifyConsumed); err != nil {
		t.Fatalf("read successfully verified challenge: %v", err)
	}
	if verifyAttempts != 4 || !verifyConsumed {
		t.Fatalf("verified challenge attempts=%d consumed=%v, want 4/true", verifyAttempts, verifyConsumed)
	}
	if verified, err := repo.VerifyPreAccountChallenge(ctx, verifyMobileHash, verifyOTPHash); err != nil || verified {
		t.Fatalf("replayed OTP: verified=%v error=%v, want false/nil", verified, err)
	}

	exhaustedMobileHash := integrationTokenHash("parent-otp-exhaust-mobile-" + unique)
	exhaustedOTPHash := integrationTokenHash("parent-otp-exhaust-code-" + unique)
	exhaustedID, err := repo.CreatePreAccountChallenge(
		ctx, exhaustedMobileHash, integrationTokenHash("parent-otp-exhaust-ip-"+unique),
		exhaustedOTPHash, time.Now().Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("create challenge for attempt exhaustion: %v", err)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		verified, verifyErr := repo.VerifyPreAccountChallenge(ctx, exhaustedMobileHash, wrongOTPHash)
		if verifyErr != nil || verified {
			t.Fatalf("exhaustion attempt %d: verified=%v error=%v", attempt, verified, verifyErr)
		}
	}
	if verified, err := repo.VerifyPreAccountChallenge(ctx, exhaustedMobileHash, exhaustedOTPHash); err != nil || verified {
		t.Fatalf("correct OTP after exhaustion: verified=%v error=%v, want false/nil", verified, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT attempt_count, consumed_at IS NOT NULL
		  FROM parent_otp_challenges WHERE id=$1
	`, exhaustedID).Scan(&verifyAttempts, &verifyConsumed); err != nil {
		t.Fatalf("read exhausted challenge: %v", err)
	}
	if verifyAttempts != 5 || !verifyConsumed {
		t.Fatalf("exhausted challenge attempts=%d consumed=%v, want 5/true", verifyAttempts, verifyConsumed)
	}

	// Two simultaneous submissions of one valid code must yield exactly one
	// success. SELECT FOR UPDATE serializes both attempts on the challenge row.
	concurrentMobileHash := integrationTokenHash("parent-otp-concurrent-mobile-" + unique)
	concurrentOTPHash := integrationTokenHash("parent-otp-concurrent-code-" + unique)
	if _, err := repo.CreatePreAccountChallenge(
		ctx, concurrentMobileHash, integrationTokenHash("parent-otp-concurrent-ip-"+unique),
		concurrentOTPHash, time.Now().Add(5*time.Minute),
	); err != nil {
		t.Fatalf("create challenge for concurrent verification: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errorsFound := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, verifyErr := repo.VerifyPreAccountChallenge(ctx, concurrentMobileHash, concurrentOTPHash)
			results <- ok
			errorsFound <- verifyErr
		}()
	}
	wg.Wait()
	close(results)
	close(errorsFound)
	successes := 0
	for ok := range results {
		if ok {
			successes++
		}
	}
	for verifyErr := range errorsFound {
		if verifyErr != nil {
			t.Fatalf("concurrent verification error: %v", verifyErr)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent successful verifications = %d, want exactly 1", successes)
	}

}
