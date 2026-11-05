package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPlatformAdminLifecyclePreventsDeactivatingLastActiveAdmin(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	actorID := ensurePlatformAdminLifecycleActor(t, pool)
	repo := NewPlatformAdminLifecycleRepository(pool)

	stamp := time.Now().UnixNano()
	first, err := repo.Create(ctx, actorID, fmt.Sprintf("last-admin-first-%d@example.invalid", stamp), "integration-password-hash")
	if err != nil {
		t.Fatalf("create first fixture Platform Admin: %v", err)
	}
	second, err := repo.Create(ctx, actorID, fmt.Sprintf("last-admin-second-%d@example.invalid", stamp), "integration-password-hash")
	if err != nil {
		_ = repo.Deactivate(ctx, actorID, first.PrincipalID)
		t.Fatalf("create second fixture Platform Admin: %v", err)
	}

	// Keep exactly these two fixture admins active inside this transaction.
	// The transaction is rolled back after exercising the database trigger, so
	// unrelated integration fixtures are never permanently modified.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin last-admin guard transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7414, 2)`); err != nil {
		t.Fatalf("lock Platform Admin lifecycle: %v", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE auth_principals
   SET status='disabled', updated_at=NOW()
 WHERE principal_type='platform_admin'
   AND status='active'
   AND id NOT IN ($1,$2)`, first.PrincipalID, second.PrincipalID); err != nil {
		t.Fatalf("isolate two fixture Platform Admins: %v", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE auth_principals
   SET status='disabled', updated_at=NOW()
 WHERE id=$1 AND principal_type='platform_admin' AND status='active'`, second.PrincipalID); err != nil {
		t.Fatalf("deactivate one of two remaining Platform Admins: %v", err)
	}

	_, err = tx.Exec(ctx, `
UPDATE auth_principals
   SET status='disabled', updated_at=NOW()
 WHERE id=$1 AND principal_type='platform_admin' AND status='active'`, first.PrincipalID)
	if err == nil {
		t.Fatal("database allowed deactivation of the last active Platform Admin")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || !strings.Contains(pgErr.Message, "last active Platform Admin") {
		t.Fatalf("expected last-active-admin trigger rejection, got %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback last-admin guard transaction: %v", err)
	}

	// Remove the fixture accounts through the normal repository path.
	if err := repo.Deactivate(ctx, actorID, first.PrincipalID); err != nil {
		t.Fatalf("cleanup first fixture Platform Admin: %v", err)
	}
	if err := repo.Deactivate(ctx, actorID, second.PrincipalID); err != nil {
		t.Fatalf("cleanup second fixture Platform Admin: %v", err)
	}
}
