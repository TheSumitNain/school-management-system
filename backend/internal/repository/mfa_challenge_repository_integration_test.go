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
	repo := NewMFAChallengeRepository(pool)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM auth_mfa_challenges WHERE principal_id = $1`, principalID)
	})

	t.Run("new challenge supersedes previous challenge", func(t *testing.T) {
		oldHash := integrationTokenHash("mfa-challenge-old")
		newHash := integrationTokenHash("mfa-challenge-new")
		if err := repo.Create(ctx, principalID, MFAChallengeEnrollment, oldHash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create old challenge: %v", err)
		}
		if err := repo.Create(ctx, principalID, MFAChallengeEnrollment, newHash, time.Now().Add(5*time.Minute)); err != nil {
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
		if err := repo.Create(ctx, principalID, MFAChallengeLogin, hash, time.Now().Add(5*time.Minute)); err != nil {
			t.Fatalf("create login challenge: %v", err)
		}
		for i := 1; i <= maxMFAChallengeAttempts; i++ {
			if err := repo.RecordFailure(ctx, hash); err != nil {
				t.Fatalf("record failure %d: %v", i, err)
			}
		}
		if _, err := repo.Get(ctx, hash); !errors.Is(err, ErrMFAChallengeUnavailable) {
			t.Fatalf("locked challenge should be unavailable, got %v", err)
		}
		if err := repo.RecordFailure(ctx, hash); !errors.Is(err, ErrMFAChallengeUnavailable) {
			t.Fatalf("sixth failure should be rejected, got %v", err)
		}
	})

	t.Run("challenge can be consumed exactly once", func(t *testing.T) {
		hash := integrationTokenHash("mfa-challenge-consume-once")
		if err := repo.Create(ctx, principalID, MFAChallengeLogin, hash, time.Now().Add(5*time.Minute)); err != nil {
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

	t.Run("invalid purpose and expired expiry are rejected", func(t *testing.T) {
		if err := repo.Create(ctx, principalID, MFAChallengePurpose("other"), integrationTokenHash("mfa-invalid-purpose"), time.Now().Add(time.Minute)); err == nil {
			t.Fatal("expected invalid purpose to be rejected")
		}
		if err := repo.Create(ctx, principalID, MFAChallengeLogin, integrationTokenHash("mfa-expired"), time.Now().Add(-time.Minute)); err == nil {
			t.Fatal("expected expired challenge to be rejected")
		}
	})
}
