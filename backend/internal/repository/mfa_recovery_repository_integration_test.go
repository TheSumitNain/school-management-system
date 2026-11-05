package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	if err := pool.QueryRow(ctx, `SELECT principal_id FROM users WHERE id=$1`, userID).Scan(&principalID); err != nil {
		t.Fatalf("load principal: %v", err)
	}
	var tokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, principalID).Scan(&tokenVersion); err != nil {
		t.Fatalf("load principal token version: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at,last_accepted_step)
VALUES($1,'totp',$2,1,NOW(),NULL)
ON CONFLICT (principal_id) DO UPDATE SET verified_at=NOW(),last_accepted_step=NULL`, principalID, make([]byte, 64)); err != nil {
		t.Fatalf("prepare credential: %v", err)
	}

	// Recovery-code hashes are globally unique and their history is append-only.
	// Use a fresh random namespace for every test run so reruns never collide
	// with immutable rows left by earlier runs.
	var runID [16]byte
	if _, err := rand.Read(runID[:]); err != nil {
		t.Fatalf("generate integration test namespace: %v", err)
	}
	runPrefix := hex.EncodeToString(runID[:])

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_mfa_challenges WHERE principal_id=$1`, principalID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principal_mfa_credentials WHERE principal_id=$1`, principalID)
	})
	repo := NewMFARepository(pool)
	challenges := NewMFAChallengeRepository(pool)
	initial := make([]string, 10)
	for i := range initial {
		initial[i] = integrationTokenHash(runPrefix + "-recovery-initial-" + string(rune('a'+i)))
	}
	initialChallenge := integrationTokenHash(runPrefix + "-recovery-regeneration-initial")
	if err := challenges.Create(ctx, principalID, tokenVersion, MFAChallengeRecoveryRegeneration, initialChallenge, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("create initial regeneration challenge: %v", err)
	}
	if err := repo.ReplaceRecoveryCodeHashes(ctx, principalID, initialChallenge, tokenVersion, 100, initial); err != nil {
		t.Fatalf("store initial set: %v", err)
	}

	first := integrationTokenHash(runPrefix + "-recovery-login-first")
	if err := challenges.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, first, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, first, tokenVersion+1, initial[0]); !errors.Is(err, ErrMFAEnrollmentInvalid) {
		t.Fatalf("stale token version accepted recovery code: %v", err)
	}
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, first, tokenVersion, initial[0]); err != nil {
		t.Fatalf("consume first code: %v", err)
	}
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, first, tokenVersion, initial[1]); !errors.Is(err, ErrMFAEnrollmentInvalid) {
		t.Fatalf("consumed challenge accepted another code: %v", err)
	}

	second := integrationTokenHash(runPrefix + "-recovery-login-second")
	if err := challenges.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, second, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, second, tokenVersion, initial[0]); !errors.Is(err, ErrMFARecoveryCodeInvalid) {
		t.Fatalf("used code accepted again: %v", err)
	}
	if err := repo.ConsumeLoginRecoveryCode(ctx, principalID, second, tokenVersion, initial[1]); err != nil {
		t.Fatalf("unused code rejected: %v", err)
	}

	replacement := make([]string, 10)
	for i := range replacement {
		replacement[i] = integrationTokenHash(runPrefix + "-recovery-replacement-" + string(rune('a'+i)))
	}
	replacementChallenge := integrationTokenHash(runPrefix + "-recovery-regeneration-replacement")
	if err := challenges.Create(ctx, principalID, tokenVersion, MFAChallengeRecoveryRegeneration, replacementChallenge, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("create replacement challenge: %v", err)
	}
	if err := repo.ReplaceRecoveryCodeHashes(ctx, principalID, replacementChallenge, tokenVersion, 101, replacement); err != nil {
		t.Fatalf("replace set: %v", err)
	}
	var oldUnused, newUnused int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM principal_mfa_recovery_codes WHERE principal_id=$1 AND code_hash=ANY($2) AND used_at IS NULL`, principalID, initial).Scan(&oldUnused); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM principal_mfa_recovery_codes WHERE principal_id=$1 AND code_hash=ANY($2) AND used_at IS NULL`, principalID, replacement).Scan(&newUnused); err != nil {
		t.Fatal(err)
	}
	if oldUnused != 0 || newUnused != 10 {
		t.Fatalf("after replace: old unused=%d new unused=%d", oldUnused, newUnused)
	}

	raceChallenge := integrationTokenHash(runPrefix + "-recovery-race-single-challenge")
	if err := challenges.Create(ctx, principalID, tokenVersion, MFAChallengeLogin, raceChallenge, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.ConsumeLoginRecoveryCode(ctx, principalID, raceChallenge, tokenVersion, replacement[0])
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrMFARecoveryCodeInvalid) && !errors.Is(err, ErrMFAEnrollmentInvalid) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("concurrent uses succeeded %d times; want 1", successes.Load())
	}
}

func TestMFARepositoryReencryptCredentialPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	var principalID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id FROM users WHERE id=$1`, userID).Scan(&principalID); err != nil {
		t.Fatalf("load principal: %v", err)
	}
	oldCiphertext := make([]byte, 64)
	newCiphertext := make([]byte, 64)
	staleCiphertext := make([]byte, 64)
	oldCiphertext[0], newCiphertext[0], staleCiphertext[0] = 1, 2, 3
	if _, err := pool.Exec(ctx, `INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
VALUES($1,'totp',$2,1,NOW())
ON CONFLICT (principal_id) DO UPDATE SET
 secret_ciphertext=EXCLUDED.secret_ciphertext,
 encryption_key_version=EXCLUDED.encryption_key_version,
 verified_at=NOW()`, principalID, oldCiphertext); err != nil {
		t.Fatalf("prepare MFA credential: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM auth_mfa_challenges WHERE principal_id=$1`, principalID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM principal_mfa_credentials WHERE principal_id=$1`, principalID)
	})

	repo := NewMFARepository(pool)
	if err := repo.ReencryptCredential(ctx, principalID, oldCiphertext, 1, newCiphertext, 2); err != nil {
		t.Fatalf("rotate credential: %v", err)
	}
	var storedCiphertext []byte
	var storedVersion int
	if err := pool.QueryRow(ctx, `SELECT secret_ciphertext,encryption_key_version
FROM principal_mfa_credentials WHERE principal_id=$1 AND method='totp'`, principalID).Scan(&storedCiphertext, &storedVersion); err != nil {
		t.Fatalf("load rotated credential: %v", err)
	}
	if storedVersion != 2 || string(storedCiphertext) != string(newCiphertext) {
		t.Fatalf("stored credential after rotation = version %d, ciphertext %v; want version 2 and new ciphertext", storedVersion, storedCiphertext)
	}

	// A stale concurrent worker must not overwrite the already-rotated value.
	if err := repo.ReencryptCredential(ctx, principalID, oldCiphertext, 1, staleCiphertext, 3); err != nil {
		t.Fatalf("stale compare-and-swap should be harmless: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT secret_ciphertext,encryption_key_version
FROM principal_mfa_credentials WHERE principal_id=$1 AND method='totp'`, principalID).Scan(&storedCiphertext, &storedVersion); err != nil {
		t.Fatalf("reload rotated credential: %v", err)
	}
	if storedVersion != 2 || string(storedCiphertext) != string(newCiphertext) {
		t.Fatalf("stale rotation overwrote credential: version %d", storedVersion)
	}
}
