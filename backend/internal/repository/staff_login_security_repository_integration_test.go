package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// This integration test uses the same strict school_management_test guard as
// the refresh-session integration suite. Apply migrations through V6 first.
func TestStaffLoginSecurityRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var principalID int64
	if err := pool.QueryRow(ctx,
		`SELECT principal_id FROM users WHERE id = $1`, userID,
	).Scan(&principalID); err != nil {
		t.Fatalf("load integration principal: %v", err)
	}

	var hasV6 bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('public.auth_login_ip_attempts') IS NOT NULL`,
	).Scan(&hasV6); err != nil {
		t.Fatalf("detect V6 login IP rate-limit schema: %v", err)
	}
	if !hasV6 {
		t.Fatal("V6 schema is missing; apply migrations to school_management_test before running this integration test")
	}

	repo := NewStaffLoginSecurityRepository(pool)
	username := fmt.Sprintf("login-security-%d", time.Now().UnixNano())
	identifierHash := loginIdentifierHash(username)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM auth_login_attempts WHERE identifier_hash = $1`, identifierHash)
	})

	t.Run("username advisory lock serializes database sessions", func(t *testing.T) {
		release, err := repo.AcquireAttemptLock(ctx, username)
		if err != nil {
			t.Fatalf("acquire username lock: %v", err)
		}
		defer release()

		otherConn, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire second database connection: %v", err)
		}
		var acquired bool
		err = otherConn.QueryRow(ctx,
			`SELECT pg_try_advisory_lock(hashtextextended($1, 0))`,
			identifierHash,
		).Scan(&acquired)
		if err != nil {
			otherConn.Release()
			t.Fatalf("probe username advisory lock: %v", err)
		}
		if acquired {
			_, _ = otherConn.Exec(ctx,
				`SELECT pg_advisory_unlock(hashtextextended($1, 0))`,
				identifierHash,
			)
			otherConn.Release()
			t.Fatal("a second database session acquired a username lock that should be held")
		}
		otherConn.Release()

		release()
		checkConn, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire verification connection: %v", err)
		}
		defer checkConn.Release()
		if err := checkConn.QueryRow(ctx,
			`SELECT pg_try_advisory_lock(hashtextextended($1, 0))`,
			identifierHash,
		).Scan(&acquired); err != nil {
			t.Fatalf("verify username lock release: %v", err)
		}
		if !acquired {
			t.Fatal("username advisory lock remained held after release")
		}
		if _, err := checkConn.Exec(ctx,
			`SELECT pg_advisory_unlock(hashtextextended($1, 0))`,
			identifierHash,
		); err != nil {
			t.Fatalf("release verification advisory lock: %v", err)
		}
	})

	t.Run("fifth failure locks username and success clears lockout", func(t *testing.T) {
		for i := 1; i <= 4; i++ {
			if err := repo.RecordFailure(ctx, username); err != nil {
				t.Fatalf("record failure %d: %v", i, err)
			}
		}
		locked, err := repo.IsLocked(ctx, username)
		if err != nil {
			t.Fatalf("check lockout after four failures: %v", err)
		}
		if locked {
			t.Fatal("username must not be locked before the fifth failure")
		}

		if err := repo.RecordFailure(ctx, username); err != nil {
			t.Fatalf("record fifth failure: %v", err)
		}
		locked, err = repo.IsLocked(ctx, username)
		if err != nil {
			t.Fatalf("check lockout after fifth failure: %v", err)
		}
		if !locked {
			t.Fatal("expected username to be locked after five failures")
		}

		if err := repo.RecordSuccess(ctx, username, principalID); err != nil {
			t.Fatalf("record successful login: %v", err)
		}
		locked, err = repo.IsLocked(ctx, username)
		if err != nil {
			t.Fatalf("check lockout after successful login: %v", err)
		}
		if locked {
			t.Fatal("successful authentication must clear the username lockout")
		}
	})

	t.Run("IP limit is shared and atomic under concurrent requests", func(t *testing.T) {
		clientIP := "198.51.100.21"
		ipHash, err := loginIPHash(clientIP)
		if err != nil {
			t.Fatalf("hash integration client IP: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(),
				`DELETE FROM auth_login_ip_attempts WHERE ip_hash = $1`, ipHash)
		})

		const requests = 40
		var wg sync.WaitGroup
		var mu sync.Mutex
		allowed := 0
		errs := make(chan error, requests)
		for i := 0; i < requests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ok, callErr := repo.AllowIPAttempt(ctx, clientIP)
				if callErr != nil {
					errs <- callErr
					return
				}
				if ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		close(errs)
		for callErr := range errs {
			t.Errorf("record concurrent IP attempt: %v", callErr)
		}
		if allowed != 30 {
			t.Fatalf("allowed concurrent requests = %d, want exactly 30", allowed)
		}

		ok, err := repo.AllowIPAttempt(ctx, clientIP)
		if err != nil {
			t.Fatalf("check blocked IP: %v", err)
		}
		if ok {
			t.Fatal("31st and subsequent requests from a blocked IP must be denied")
		}
	})

	t.Run("invalid client IP is rejected", func(t *testing.T) {
		if _, err := repo.AllowIPAttempt(ctx, "not-an-ip"); err == nil {
			t.Fatal("expected invalid IP to be rejected")
		}
	})
}
