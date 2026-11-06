package repository

import (
	"context"
	"testing"
	"time"
)

func TestParentOTPLoginChallengeRepositoryRejectsInvalidIdentity(t *testing.T) {
	repo := NewParentOTPChallengeRepository(nil)
	hash := integrationTokenHash("principal-bound-test")
	if _, err := repo.CreatePrincipalChallenge(context.Background(), 0, "+919876543210", hash, hash, hash, time.Now().Add(time.Minute)); err == nil {
		t.Fatal("expected zero principal ID to be rejected")
	}
	if _, err := repo.VerifyPrincipalChallenge(context.Background(), 0, hash, hash); err == nil {
		t.Fatal("expected zero principal ID to be rejected")
	}
	if err := repo.InvalidatePrincipalChallenge(context.Background(), 0, 1); err == nil {
		t.Fatal("expected zero principal ID to be rejected")
	}
}
