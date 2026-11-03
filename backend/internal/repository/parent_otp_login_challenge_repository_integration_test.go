package repository

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"testing"
	"time"
)

// TestParentOTPLoginChallengeRepositoryPostgresIntegration verifies that
// principal-bound challenges are persisted, rate-limited, isolated by
// principal/mobile, and consumed atomically by PostgreSQL.
func TestParentOTPLoginChallengeRepositoryPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV12 bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.parent_otp_challenges') IS NOT NULL
		   AND to_regclass('public.parent_otp_rate_limits') IS NOT NULL
		   AND to_regclass('public.parent_accounts') IS NOT NULL
	`).Scan(&hasV12); err != nil {
		t.Fatalf("detect Parent OTP login schema: %v", err)
	}
	if !hasV12 {
		t.Fatal("V12 Parent OTP schema is missing; apply migrations to school_management_test first")
	}

	randomSuffix, err := rand.Int(rand.Reader, big.NewInt(1_000_000_000))
	if err != nil {
		t.Fatalf("generate unique Parent fixture mobile suffix: %v", err)
	}
	suffix := fmt.Sprintf("%09d", randomSuffix.Int64())
	mobileA := "+919" + suffix
	mobileB := "+918" + suffix
	mobileHashA := integrationTokenHash("parent-login-mobile-a-" + suffix)
	mobileHashB := integrationTokenHash("parent-login-mobile-b-" + suffix)
	ipHashA := integrationTokenHash("parent-login-ip-a-" + suffix)
	ipHashB := integrationTokenHash("parent-login-ip-b-" + suffix)
	otpHashA := integrationTokenHash("parent-login-otp-a-" + suffix)
	otpHashB := integrationTokenHash("parent-login-otp-b-" + suffix)
	wrongOTPHash := integrationTokenHash("parent-login-otp-wrong-" + suffix)

	var principalIDs []int64
	rateLimitHashes := []string{mobileHashA, mobileHashB, ipHashA, ipHashB}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if len(principalIDs) > 0 {
			if _, err := pool.Exec(cleanupCtx,
				`DELETE FROM parent_otp_challenges WHERE principal_id = ANY($1::BIGINT[])`,
				principalIDs,
			); err != nil {
				t.Errorf("clean up Parent OTP challenge fixtures: %v", err)
			}
			if _, err := pool.Exec(cleanupCtx,
				`DELETE FROM parent_accounts WHERE principal_id = ANY($1::BIGINT[])`,
				principalIDs,
			); err != nil {
				t.Errorf("clean up Parent account fixtures: %v", err)
			}
			if _, err := pool.Exec(cleanupCtx,
				`DELETE FROM auth_principals WHERE id = ANY($1::BIGINT[])`,
				principalIDs,
			); err != nil {
				t.Errorf("clean up Parent principal fixtures: %v", err)
			}
		}
		if _, err := pool.Exec(cleanupCtx,
			`DELETE FROM parent_otp_rate_limits WHERE subject_hash = ANY($1::TEXT[])`,
			rateLimitHashes,
		); err != nil {
			t.Errorf("clean up Parent OTP rate-limit fixtures: %v", err)
		}
	})

	createVerifiedParent := func(mobile string) int64 {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin Parent fixture transaction: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var principalID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO auth_principals (principal_type, status)
			VALUES ('parent', 'active')
			RETURNING id
		`).Scan(&principalID); err != nil {
			t.Fatalf("create Parent principal: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO parent_accounts (principal_id, mobile_number, mobile_verified_at)
			VALUES ($1, $2, NOW())
		`, principalID, mobile); err != nil {
			t.Fatalf("create verified Parent account: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit Parent fixture: %v", err)
		}
		principalIDs = append(principalIDs, principalID)
		return principalID
	}

	principalA := createVerifiedParent(mobileA)
	principalB := createVerifiedParent(mobileB)
	repo := NewParentOTPChallengeRepository(pool)

	challengeID, err := repo.CreatePrincipalChallenge(
		ctx, principalA, mobileA, mobileHashA, ipHashA, otpHashA,
		time.Now().Add(parentOTPChallengeTTL),
	)
	if err != nil {
		t.Fatalf("create principal-bound Parent OTP challenge: %v", err)
	}
	if challengeID <= 0 {
		t.Fatalf("challenge ID = %d, want positive ID", challengeID)
	}

	var storedPrincipalID int64
	var storedMobileHash, storedOTPHash string
	var attempts int16
	if err := pool.QueryRow(ctx, `
		SELECT principal_id, mobile_lookup_hash, otp_hash, attempt_count
		  FROM parent_otp_challenges WHERE id = $1
	`, challengeID).Scan(&storedPrincipalID, &storedMobileHash, &storedOTPHash, &attempts); err != nil {
		t.Fatalf("read persisted principal-bound challenge: %v", err)
	}
	if storedPrincipalID != principalA || storedMobileHash != mobileHashA ||
		storedOTPHash != otpHashA || attempts != 0 {
		t.Fatalf("persisted challenge identity/hash mismatch: principal=%d mobileHash=%q attempts=%d",
			storedPrincipalID, storedMobileHash, attempts)
	}

	var mobileSends, ipSends int16
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type = 'mobile' AND subject_hash = $1
	`, mobileHashA).Scan(&mobileSends); err != nil {
		t.Fatalf("read Parent mobile send bucket: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT SUM(send_count)::SMALLINT FROM parent_otp_rate_limits
		 WHERE subject_type = 'ip' AND subject_hash = $1
	`, ipHashA).Scan(&ipSends); err != nil {
		t.Fatalf("read Parent IP send bucket: %v", err)
	}
	if mobileSends != 1 || ipSends != 1 {
		t.Fatalf("send counts mobile=%d IP=%d, want 1 each", mobileSends, ipSends)
	}

	// A challenge cannot be verified through a different Parent principal.
	if verified, err := repo.VerifyPrincipalChallenge(ctx, principalB, mobileHashA, otpHashA); err != nil || verified {
		t.Fatalf("cross-principal verification: verified=%v error=%v, want false/nil", verified, err)
	}

	// A wrong guess is durable, but a later correct guess can still succeed.
	if verified, err := repo.VerifyPrincipalChallenge(ctx, principalA, mobileHashA, wrongOTPHash); err != nil || verified {
		t.Fatalf("wrong OTP: verified=%v error=%v, want false/nil", verified, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT attempt_count FROM parent_otp_challenges WHERE id = $1
	`, challengeID).Scan(&attempts); err != nil {
		t.Fatalf("read failed OTP attempt count: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempt count = %d, want 1", attempts)
	}

	if verified, err := repo.VerifyPrincipalChallenge(ctx, principalA, mobileHashA, otpHashA); err != nil || !verified {
		t.Fatalf("correct OTP: verified=%v error=%v, want true/nil", verified, err)
	}
	var consumed bool
	if err := pool.QueryRow(ctx, `
		SELECT consumed_at IS NOT NULL FROM parent_otp_challenges WHERE id = $1
	`, challengeID).Scan(&consumed); err != nil {
		t.Fatalf("read consumed Parent challenge: %v", err)
	}
	if !consumed {
		t.Fatal("successfully verified Parent challenge was not consumed")
	}
	if verified, err := repo.VerifyPrincipalChallenge(ctx, principalA, mobileHashA, otpHashA); err != nil || verified {
		t.Fatalf("replayed Parent OTP: verified=%v error=%v, want false/nil", verified, err)
	}

	// The mobile cooldown applies to principal-bound sends as it does to
	// registration OTP sends; a rejected resend must not create a new row.
	if _, err := repo.CreatePrincipalChallenge(
		ctx, principalA, mobileA, mobileHashA, ipHashA,
		integrationTokenHash("parent-login-otp-resend-" + suffix),
		time.Now().Add(parentOTPChallengeTTL),
	); !errors.Is(err, ErrParentOTPResendWait) {
		t.Fatalf("immediate resend error = %v, want ErrParentOTPResendWait", err)
	}

	// Delivery failure cleanup is scoped to both challenge and principal.
	undeliveredID, err := repo.CreatePrincipalChallenge(
		ctx, principalB, mobileB, mobileHashB, ipHashB, otpHashB,
		time.Now().Add(parentOTPChallengeTTL),
	)
	if err != nil {
		t.Fatalf("create challenge for delivery-failure cleanup: %v", err)
	}
	if err := repo.InvalidatePrincipalChallenge(ctx, principalA, undeliveredID); err == nil {
		t.Fatal("expected another Parent principal to be unable to invalidate this challenge")
	}
	if err := repo.InvalidatePrincipalChallenge(ctx, principalB, undeliveredID); err != nil {
		t.Fatalf("invalidate undelivered principal-bound challenge: %v", err)
	}
	if verified, err := repo.VerifyPrincipalChallenge(ctx, principalB, mobileHashB, otpHashB); err != nil || verified {
		t.Fatalf("verify invalidated challenge: verified=%v error=%v, want false/nil", verified, err)
	}
}

