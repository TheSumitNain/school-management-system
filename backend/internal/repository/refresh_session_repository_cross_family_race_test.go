package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type operationResult struct {
	name string
	err  error
}

func TestRefreshSessionRepositoryCrossFamilyInactiveRace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo, pool, userID := newPostgresIntegrationRepository(t)

	// Keep token hashes unique across repeated runs. A prior interrupted
	// integration run may have left rows behind in the dedicated test DB.
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	familyAOld := integrationTokenHash("cross-family-race-a-old-" + suffix)
	familyANew := integrationTokenHash("cross-family-race-a-new-" + suffix)
	familyBOld := integrationTokenHash("cross-family-race-b-old-" + suffix)
	familyBNew := integrationTokenHash("cross-family-race-b-new-" + suffix)

	expiresAt := time.Now().Add(time.Hour)

	if err := repo.Create(ctx, userID, familyAOld, expiresAt); err != nil {
		t.Fatalf("create family A session: %v", err)
	}

	if err := repo.Create(ctx, userID, familyBOld, expiresAt); err != nil {
		t.Fatalf("create family B session: %v", err)
	}

	functionName := "test_refresh_insert_barrier_fn_" + suffix
	triggerName := "test_refresh_insert_barrier_" + suffix

	const barrierKey int64 = 73482917364829173

	createFunction := fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger AS $$
		BEGIN
			IF NEW.token_hash = '%s' THEN
				PERFORM pg_advisory_xact_lock(%d);
			END IF;

			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`, functionName, familyBNew, barrierKey)

	if _, err := pool.Exec(ctx, createFunction); err != nil {
		t.Fatalf("create insert barrier function: %v", err)
	}

	createTrigger := fmt.Sprintf(`
		CREATE TRIGGER %s
		BEFORE INSERT ON refresh_sessions
		FOR EACH ROW
		EXECUTE FUNCTION %s()
	`, triggerName, functionName)

	if _, err := pool.Exec(ctx, createTrigger); err != nil {
		_, _ = pool.Exec(
			context.Background(),
			fmt.Sprintf(`DROP FUNCTION IF EXISTS %s()`, functionName),
		)

		t.Fatalf("create insert barrier trigger: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		_, _ = pool.Exec(
			cleanupCtx,
			fmt.Sprintf(
				`DROP TRIGGER IF EXISTS %s ON refresh_sessions`,
				triggerName,
			),
		)

		_, _ = pool.Exec(
			cleanupCtx,
			fmt.Sprintf(
				`DROP FUNCTION IF EXISTS %s()`,
				functionName,
			),
		)
	})

	barrierConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire barrier connection: %v", err)
	}
	defer barrierConn.Release()

	if _, err := barrierConn.Exec(
		ctx,
		`SELECT pg_advisory_lock($1)`,
		barrierKey,
	); err != nil {
		t.Fatalf("hold insert barrier lock: %v", err)
	}

	barrierHeld := true

	defer func() {
		if barrierHeld {
			_, _ = barrierConn.Exec(
				context.Background(),
				`SELECT pg_advisory_unlock($1)`,
				barrierKey,
			)
		}
	}()

	results := make(chan operationResult, 2)

	// Start family B.
	go func() {
		_, rotateErr := repo.Rotate(
			ctx,
			familyBOld,
			familyBNew,
			time.Now().Add(time.Hour),
		)

		results <- operationResult{
			name: "family B",
			err:  rotateErr,
		}
	}()

	// Family B must reach the trigger and wait for its advisory lock.
	waitForAdvisoryLockWaiters(
		t,
		ctx,
		pool,
		results,
		1,
		"family B",
	)

	// Disable the user while family B is blocked.
	if _, err := pool.Exec(
		ctx,
		`UPDATE users SET status = 'disabled' WHERE id = $1`,
		userID,
	); err != nil {
		t.Fatalf("disable test user: %v", err)
	}

	// Start family A. It must wait for family B's per-user advisory lock.
	go func() {
		_, rotateErr := repo.Rotate(
			ctx,
			familyAOld,
			familyANew,
			time.Now().Add(time.Hour),
		)

		results <- operationResult{
			name: "family A",
			err:  rotateErr,
		}
	}()

	waitForAdvisoryLockWaiters(
		t,
		ctx,
		pool,
		results,
		2,
		"family A",
	)

	// Release family B's trigger barrier.
	if _, err := barrierConn.Exec(
		ctx,
		`SELECT pg_advisory_unlock($1)`,
		barrierKey,
	); err != nil {
		t.Fatalf("release insert barrier lock: %v", err)
	}

	barrierHeld = false

	var familyAErr error
	var familyBErr error

	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			switch result.name {
			case "family A":
				familyAErr = result.err

			case "family B":
				familyBErr = result.err

			default:
				t.Fatalf(
					"unexpected operation result: %q",
					result.name,
				)
			}

		case <-ctx.Done():
			t.Fatalf(
				"timed out waiting for concurrent rotations: %v",
				ctx.Err(),
			)
		}
	}

	if !errors.Is(familyAErr, ErrRefreshSessionInactive) {
		t.Fatalf(
			"expected family A to return inactive-session error, got %v",
			familyAErr,
		)
	}

	if familyBErr != nil {
		t.Fatalf(
			"expected family B rotation to complete, got %v",
			familyBErr,
		)
	}

	var replacementActive bool

	err = pool.QueryRow(
		ctx,
		`
			SELECT revoked_at IS NULL
			FROM refresh_sessions
			WHERE token_hash = $1
		`,
		familyBNew,
	).Scan(&replacementActive)

	if err != nil {
		t.Fatalf(
			"check family B replacement state: %v",
			err,
		)
	}

	if replacementActive {
		t.Fatal(
			"family B replacement remained active after family A revoked all user sessions",
		)
	}

	t.Log(
		"PASS: family A waited for family B, then revoked its replacement after observing the inactive user",
	)
}

func waitForAdvisoryLockWaiters(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	results <-chan operationResult,
	minimum int,
	operation string,
) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Report an early Rotate error immediately instead of hiding it
		// behind a timeout.
		select {
		case result := <-results:
			t.Fatalf(
				"%s returned before the expected advisory lock wait; error: %v",
				result.name,
				result.err,
			)
		default:
		}

		var waiting int

		err := pool.QueryRow(
			ctx,
			`
				SELECT COUNT(*)
				FROM pg_locks
				WHERE locktype = 'advisory'
				  AND granted = false
			`,
		).Scan(&waiting)

		if err != nil {
			t.Fatalf(
				"inspect PostgreSQL advisory lock waiters: %v",
				err,
			)
		}

		if waiting >= minimum {
			return
		}

		select {
		case result := <-results:
			t.Fatalf(
				"%s returned before the expected advisory lock wait; error: %v",
				result.name,
				result.err,
			)

		case <-ctx.Done():
			t.Fatalf(
				"timed out waiting for %s: expected %d advisory lock waiters, observed %d: %v",
				operation,
				minimum,
				waiting,
				ctx.Err(),
			)

		case <-ticker.C:
		}
	}
}
