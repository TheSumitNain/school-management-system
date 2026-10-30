package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"school-management-system/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func newPostgresIntegrationRepository(
	t *testing.T,
) (*RefreshSessionRepository, *pgxpool.Pool, int64) {
	t.Helper()

	if os.Getenv("RUN_POSTGRES_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	if err := godotenv.Load("..\\..\\.env"); err != nil {
		t.Fatalf("load backend .env file: %v", err)
	}

	cfg := config.Load()

	// Never allow integration tests to target the development database.
	if cfg.DBName != "school_management_test" {
		t.Fatalf(
			"unsafe test database %q; expected school_management_test",
			cfg.DBName,
		)
	}

	if err := cfg.ValidateDatabase(); err != nil {
		t.Fatalf("invalid database configuration: %v", err)
	}

	dbURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:   "/" + cfg.DBName,
	}

	query := dbURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dbURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL.String())
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	// Register pool cleanup first; row cleanup below runs before this.
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var schoolID int64
	err = pool.QueryRow(
		ctx,
		`INSERT INTO schools (name, code, status)
		 VALUES ($1, $2, 'active') RETURNING id`,
		"Integration Test School "+suffix,
		"IT-"+suffix,
	).Scan(&schoolID)
	if err != nil {
		t.Fatalf("create test school: %v", err)
	}

	var userID int64
	err = pool.QueryRow(
		ctx,
		`INSERT INTO users (school_id, username, password_hash, status)
		 VALUES ($1, $2, $3, 'active') RETURNING id`,
		schoolID,
		"test-user-"+suffix,
		"integration-test-password-hash",
	).Scan(&userID)
	if err != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM schools WHERE id = $1`, schoolID)
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schools WHERE id = $1`, schoolID)
	})

	return NewRefreshSessionRepository(pool), pool, userID
}

func TestRefreshSessionRepositoryPostgresIntegration(t *testing.T) {
	repo, pool, userID := newPostgresIntegrationRepository(t)

	t.Run("rotation and replay revoke the family", func(t *testing.T) {
		ctx := context.Background()
		oldHash := integrationTokenHash("rotation-old")
		newHash := integrationTokenHash("rotation-new")

		if err := repo.Create(ctx, userID, oldHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create original session: %v", err)
		}

		user, err := repo.Rotate(ctx, oldHash, newHash, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatalf("rotate refresh token: %v", err)
		}
		if user.ID != userID || user.Status != "active" {
			t.Fatalf("unexpected user returned from rotation: %+v", user)
		}

		var oldRevoked, replacementRevoked bool
		err = pool.QueryRow(ctx, `
			SELECT
				(SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $1),
				(SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $2)
		`, oldHash, newHash).Scan(&oldRevoked, &replacementRevoked)
		if err != nil {
			t.Fatalf("check rotation state: %v", err)
		}
		if !oldRevoked || replacementRevoked {
			t.Fatalf("unexpected state after rotation: old revoked=%v, replacement revoked=%v",
				oldRevoked, replacementRevoked)
		}

		_, err = repo.Rotate(ctx, oldHash, integrationTokenHash("replay-new"),
			time.Now().Add(time.Hour))
		if !errors.Is(err, ErrRefreshTokenReuse) {
			t.Fatalf("expected reuse error, got %v", err)
		}

		err = pool.QueryRow(ctx,
			`SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $1`,
			newHash,
		).Scan(&replacementRevoked)
		if err != nil {
			t.Fatalf("check family revocation: %v", err)
		}
		if !replacementRevoked {
			t.Fatal("expected replay detection to revoke the replacement in the same family")
		}
	})

	t.Run("logout revocation is idempotent", func(t *testing.T) {
		ctx := context.Background()
		hash := integrationTokenHash("logout-token")

		if err := repo.Create(ctx, userID, hash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create logout session: %v", err)
		}

		if err := repo.Revoke(ctx, hash); err != nil {
			t.Fatalf("revoke token: %v", err)
		}
		if err := repo.Revoke(ctx, hash); err != nil {
			t.Fatalf("revoke token a second time: %v", err)
		}

		_, err := repo.Rotate(ctx, hash, integrationTokenHash("logout-replacement"),
			time.Now().Add(time.Hour))
		if !errors.Is(err, ErrRefreshTokenReuse) {
			t.Fatalf("expected revoked token to be rejected, got %v", err)
		}
	})

	t.Run("logout revokes entire rotated token family", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		oldHash := integrationTokenHash("logout-family-old")
		newHash := integrationTokenHash("logout-family-new")

		if err := repo.Create(ctx, userID, oldHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create original session: %v", err)
		}

		if _, err := repo.Rotate(ctx, oldHash, newHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("rotate refresh token: %v", err)
		}

		if err := repo.Revoke(ctx, newHash); err != nil {
			t.Fatalf("logout using replacement token: %v", err)
		}

		var oldRevoked, newRevoked bool
		err := pool.QueryRow(ctx, `
			SELECT
				(SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $1),
				(SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $2)
		`, oldHash, newHash).Scan(&oldRevoked, &newRevoked)
		if err != nil {
			t.Fatalf("check family revocation: %v", err)
		}

		if !oldRevoked || !newRevoked {
			t.Fatalf("expected entire family revoked: old=%v replacement=%v",
				oldRevoked, newRevoked)
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		ctx := context.Background()
		hash := integrationTokenHash("expired-token")

		if err := repo.Create(ctx, userID, hash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create session: %v", err)
		}

		_, err := pool.Exec(ctx,
			`UPDATE refresh_sessions
			 SET created_at = NOW() - INTERVAL '2 minutes',
			     expires_at = NOW() - INTERVAL '1 minute'
			 WHERE token_hash = $1`,
			hash,
		)
		if err != nil {
			t.Fatalf("expire session: %v", err)
		}

		_, err = repo.Rotate(ctx, hash, integrationTokenHash("expired-replacement"),
			time.Now().Add(time.Hour))
		if !errors.Is(err, ErrRefreshTokenInvalid) {
			t.Fatalf("expected invalid-token error, got %v", err)
		}
	})

	t.Run("concurrent refresh detects reuse", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		oldHash := integrationTokenHash("concurrent-old")
		if err := repo.Create(ctx, userID, oldHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create concurrent session: %v", err)
		}

		type result struct {
			err error
		}

		results := make(chan result, 2)
		var start sync.WaitGroup
		start.Add(1)

		for i := 0; i < 2; i++ {
			go func(i int) {
				start.Wait()
				_, err := repo.Rotate(
					ctx,
					oldHash,
					integrationTokenHash(fmt.Sprintf("concurrent-replacement-%d", i)),
					time.Now().Add(time.Hour),
				)
				results <- result{err: err}
			}(i)
		}

		start.Done()

		successes, reuses := 0, 0
		for i := 0; i < 2; i++ {
			select {
			case result := <-results:
				switch {
				case result.err == nil:
					successes++
				case errors.Is(result.err, ErrRefreshTokenReuse):
					reuses++
				default:
					t.Fatalf("unexpected concurrent rotation error: %v", result.err)
				}
			case <-ctx.Done():
				t.Fatal("timed out waiting for concurrent refresh requests")
			}
		}

		if successes != 1 || reuses != 1 {
			t.Fatalf("expected one success and one reuse detection; got successes=%d, reuses=%d",
				successes, reuses)
		}
	})

	t.Run("concurrent logout and refresh leave no active token in the family", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		oldHash := integrationTokenHash("concurrent-logout-refresh-old")
		newHash := integrationTokenHash("concurrent-logout-refresh-new")

		if err := repo.Create(ctx, userID, oldHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create original session: %v", err)
		}

		type operationResult struct {
			name string
			err  error
		}

		start := make(chan struct{})
		results := make(chan operationResult, 2)

		go func() {
			<-start
			_, err := repo.Rotate(ctx, oldHash, newHash, time.Now().Add(time.Hour))
			results <- operationResult{name: "refresh", err: err}
		}()

		go func() {
			<-start
			err := repo.Revoke(ctx, oldHash)
			results <- operationResult{name: "logout", err: err}
		}()

		close(start)

		for i := 0; i < 2; i++ {
			select {
			case result := <-results:
				switch result.name {
				case "logout":
					if result.err != nil {
						t.Fatalf("logout failed: %v", result.err)
					}
				case "refresh":
					if result.err != nil && !errors.Is(result.err, ErrRefreshTokenReuse) {
						t.Fatalf("unexpected refresh error: %v", result.err)
					}
				default:
					t.Fatalf("unknown operation result: %q", result.name)
				}
			case <-ctx.Done():
				t.Fatal("timed out waiting for concurrent logout and refresh")
			}
		}

		var activeSessions int
		err := pool.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM refresh_sessions
			WHERE family_id = (
				SELECT family_id
				FROM refresh_sessions
				WHERE token_hash = $1
			)
			AND revoked_at IS NULL
		`, oldHash).Scan(&activeSessions)
		if err != nil {
			t.Fatalf("count active sessions after concurrent operations: %v", err)
		}

		if activeSessions != 0 {
			t.Fatalf("expected no active sessions after concurrent logout and refresh; got %d",
				activeSessions)
		}
	})
}

func integrationTokenHash(value string) string {
	sum := sha256.Sum256([]byte("postgres-integration-test:" + value))
	return hex.EncodeToString(sum[:])
}
