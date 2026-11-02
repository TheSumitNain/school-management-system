package repository

import (
	"context"
	"errors"
	"strconv"
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
}
