package repository

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMFARecoveryCodeRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	var principalID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id FROM users WHERE id=$1`, userID).Scan(&principalID); err != nil { t.Fatalf("load principal: %v", err) }
	if _, err := pool.Exec(ctx, `INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at,last_accepted_step)
VALUES($1,'totp',$2,1,NOW(),NULL)
ON CONFLICT (principal_id) DO UPDATE SET verified_at=NOW(),last_accepted_step=NULL`, principalID, make([]byte, 64)); err != nil { t.Fatalf("prepare credential: %v", err) }
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_mfa_challenges WHERE principal_id=$1`, principalID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principal_mfa_recovery_codes WHERE principal_id=$1`, principalID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principal_mfa_credentials WHERE principal_id=$1`, principalID)
	})
	repo := NewMFARepository(pool)
	challenges := NewMFAChallengeRepository(pool)
	initial := make([]string, 10)
	for i := range initial { initial[i] = integrationTokenHash("recovery-initial-" + string(rune('a'+i))) }
	if err := repo.ReplaceRecoveryCodeHashes(ctx, principalID, 100, initial); err != nil { t.Fatalf("store initial set: %v", err) }

	first := integrationTokenHash("recovery-login-first")
	if err := challenges.Create(ctx, principalID, MFAChallengeLogin, first, time.Now().Add(5*time.Minute)); err != nil { t.Fatal(err) }
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, first, initial[0]); err != nil { t.Fatalf("consume first code: %v", err) }
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, first, initial[1]); !errors.Is(err, ErrMFAEnrollmentInvalid) { t.Fatalf("consumed challenge accepted another code: %v", err) }

	second := integrationTokenHash("recovery-login-second")
	if err := challenges.Create(ctx, principalID, MFAChallengeLogin, second, time.Now().Add(5*time.Minute)); err != nil { t.Fatal(err) }
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, second, initial[0]); !errors.Is(err, ErrMFARecoveryCodeInvalid) { t.Fatalf("used code accepted again: %v", err) }
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, second, initial[1]); err != nil { t.Fatalf("unused code rejected: %v", err) }

	replacement := make([]string, 10)
	for i := range replacement { replacement[i] = integrationTokenHash("recovery-replacement-" + string(rune('a'+i))) }
	if err := repo.ReplaceRecoveryCodeHashes(ctx, principalID, 101, replacement); err != nil { t.Fatalf("replace set: %v", err) }
	var oldUnused, newUnused int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM principal_mfa_recovery_codes WHERE principal_id=$1 AND code_hash=ANY($2) AND used_at IS NULL`, principalID, initial).Scan(&oldUnused); err != nil { t.Fatal(err) }
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM principal_mfa_recovery_codes WHERE principal_id=$1 AND code_hash=ANY($2) AND used_at IS NULL`, principalID, replacement).Scan(&newUnused); err != nil { t.Fatal(err) }
	if oldUnused != 0 || newUnused != 10 { t.Fatalf("after replace: old unused=%d new unused=%d", oldUnused, newUnused) }

	raceChallenge := integrationTokenHash("recovery-race-single-challenge")
	if err := challenges.Create(ctx, principalID, MFAChallengeLogin, raceChallenge, time.Now().Add(5*time.Minute)); err != nil { t.Fatal(err) }
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.ConsumeLoginRecoveryCode(ctx, principalID, raceChallenge, replacement[0])
			if err == nil { successes.Add(1) } else if !errors.Is(err, ErrMFARecoveryCodeInvalid) && !errors.Is(err, ErrMFAEnrollmentInvalid) { t.Errorf("unexpected error: %v", err) }
		}()
	}
	wg.Wait()
	if successes.Load() != 1 { t.Fatalf("concurrent uses succeeded %d times; want 1", successes.Load()) }
}
