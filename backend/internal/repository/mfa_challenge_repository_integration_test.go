package repository

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMFAChallengeRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	var principalID int64
	if err := pool.QueryRow(ctx,
		`SELECT principal_id FROM users WHERE id = $1`, userID,
	).Scan(&principalID); err != nil {
		t.Fatalf("load fixture principal: %v", err)
	}
	var tokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, principalID).Scan(&tokenVersion); err != nil {
		t.Fatalf("load fixture token version: %v", err)
	}
	repo := NewMFAChallengeRepository(pool)
	attemptHash := mfaLoginAttemptHash(principalID)
	_, _ = pool.Exec(ctx, `DELETE FROM auth_login_attempts WHERE identifier_hash=$1`, attemptHash)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM auth_mfa_challenges WHERE principal_id = $1`, principalID)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM auth_login_attempts WHERE identifier_hash=$1`, attemptHash)
	})

	t.Run("new challenge supersedes previous challenge", func(t *testing.T) {
		oldHash := integrationTokenHash("mfa-challenge-old")
		newHash := integrationTokenHash("mfa-challenge-new")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeEnrollment, oldHash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create old challenge: %v", err)
		}
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeEnrollment, newHash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create replacement challenge: %v", err)
		}
		if _, err := repo.Get(ctx, oldHash); !errors.Is(err, ErrMFAChallengeUnavailable) {
			t.Fatalf("superseded challenge should be unavailable, got %v", err)
		}
		if _, err := repo.Get(ctx, newHash); err != nil {
			t.Fatalf("replacement challenge should be available: %v", err)
		}
	})

	t.Run("fifth failed attempt consumes challenge", func(t *testing.T) {
		hash := integrationTokenHash("mfa-challenge-five-failures")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, hash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create login challenge: %v", err)
		}
		for i := 1; i <= maxMFAChallengeAttempts; i++ {
			err := repo.RecordFailure(ctx, hash)
			if i < maxMFAChallengeAttempts && err != nil {
				t.Fatalf("record failure %d: %v", i, err)
			}
			if i == maxMFAChallengeAttempts && !errors.Is(err, ErrMFALoginLocked) {
				t.Fatalf("fifth failure error = %v, want principal-wide MFA lockout", err)
			}
		}
		if _, err := repo.Get(ctx, hash); !errors.Is(err, ErrMFALoginLocked) {
			t.Fatalf("locked challenge should return principal-wide lockout, got %v", err)
		}
		if err := repo.RecordFailure(ctx, hash); !errors.Is(err, ErrMFAChallengeUnavailable) {
			t.Fatalf("sixth failure should be rejected, got %v", err)
		}
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin,
			integrationTokenHash("mfa-challenge-after-lockout"), time.Now().Add(5*time.Minute)); !errors.Is(err, ErrMFALoginLocked) {
			t.Fatalf("new challenge must not reset principal MFA lockout, got %v", err)
		}
	})

	t.Run("challenge can be consumed exactly once", func(t *testing.T) {
		hash := integrationTokenHash("mfa-challenge-consume-once")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, hash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create challenge: %v", err)
		}
		gotPrincipal, purpose, err := repo.Consume(ctx, hash)
		if err != nil {
			t.Fatalf("consume challenge: %v", err)
		}
		if gotPrincipal != principalID || purpose != MFAChallengeLogin {
			t.Fatalf("unexpected consumed challenge identity: principal=%d purpose=%s", gotPrincipal, purpose)
		}
		if _, _, err := repo.Consume(ctx, hash); !errors.Is(err, ErrMFAChallengeUnavailable) {
			t.Fatalf("second consume should fail, got %v", err)
		}
	})

	t.Run("TOTP replay is rejected across distinct challenges", func(t *testing.T) {
		credential := make([]byte, 64)
		if _, err := pool.Exec(ctx, "INSERT INTO principal_mfa_credentials (principal_id,method,secret_ciphertext,encryption_key_version,verified_at) VALUES ($1,'totp',$2,1,NOW())", principalID, credential); err != nil {
			t.Fatalf("insert MFA credential: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), "DELETE FROM principal_mfa_credentials WHERE principal_id=$1", principalID)
		})
		mfaRepo := NewMFARepository(pool)
		if _, err := pool.Exec(ctx, `
			INSERT INTO auth_login_attempts
			    (identifier_hash,failed_attempts,window_started_at,last_attempt_at,updated_at)
			VALUES ($1,4,NOW(),NOW(),NOW())
			ON CONFLICT (identifier_hash) DO UPDATE
			SET failed_attempts=4,window_started_at=NOW(),locked_until=NULL,last_attempt_at=NOW(),updated_at=NOW()
		`, attemptHash); err != nil {
			t.Fatalf("seed pre-lockout MFA failures: %v", err)
		}
		first := integrationTokenHash("mfa-replay-first")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, first, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := mfaRepo.ConsumeLoginChallenge(ctx, principalID, first, tokenVersion, 12345); err != nil {
			t.Fatalf("first TOTP use: %v", err)
		}
		var mfaFailuresRemain bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM auth_login_attempts WHERE identifier_hash=$1)`, attemptHash).Scan(&mfaFailuresRemain); err != nil {
			t.Fatalf("check MFA failure reset after successful verification: %v", err)
		}
		if mfaFailuresRemain {
			t.Fatal("successful MFA verification did not clear principal-wide MFA failures")
		}
		second := integrationTokenHash("mfa-replay-second")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, second, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := mfaRepo.ConsumeLoginChallenge(ctx, principalID, second, tokenVersion, 12345); !errors.Is(err, ErrMFAEnrollmentInvalid) {
			t.Fatalf("replay should fail, got %v", err)
		}
		if _, err := repo.Get(ctx, second); err != nil {
			t.Fatalf("rejected replay should roll back challenge consumption: %v", err)
		}
		if err := mfaRepo.ConsumeLoginChallenge(ctx, principalID, second, tokenVersion, 12346); err != nil {
			t.Fatalf("next TOTP step should succeed: %v", err)
		}
		_, _ = pool.Exec(ctx, "DELETE FROM principal_mfa_credentials WHERE principal_id=$1", principalID)
	})

	t.Run("stale principal version cannot create or consume a challenge", func(t *testing.T) {
		hash := integrationTokenHash("mfa-stale-principal-version")
		if err := repo.Create(ctx, principalID, tokenVersion+1, MFAChallengeLogin, hash, time.Now().Add(5*time.Minute)); !errors.Is(err, ErrPrincipalVersionChanged) {
			t.Fatalf("stale challenge creation error = %v, want ErrPrincipalVersionChanged", err)
		}
		mfaRepo := NewMFARepository(pool)
		liveHash := integrationTokenHash("mfa-live-challenge-wrong-version")
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, liveHash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create live challenge: %v", err)
		}
		if err := mfaRepo.ConsumeLoginChallenge(ctx, principalID, liveHash, tokenVersion+1, 12347); !errors.Is(err, ErrMFAEnrollmentInvalid) {
			t.Fatalf("challenge consumption with stale version error = %v, want ErrMFAEnrollmentInvalid", err)
		}
		if _, err := repo.Get(ctx, liveHash); err != nil {
			t.Fatalf("stale-version attempt must not consume the live challenge: %v", err)
		}
	})

	t.Run("invalid purpose and expired expiry are rejected", func(t *testing.T) {
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengePurpose("other"), integrationTokenHash("mfa-invalid-purpose"), time.Now().Add(time.Minute)); err == nil {
			t.Fatal("expected invalid purpose to be rejected")
		}
		if err := repo.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, integrationTokenHash("mfa-expired"), time.Now().Add(-time.Minute)); err == nil {
			t.Fatal("expected expired challenge to be rejected")
		}
	})
}
