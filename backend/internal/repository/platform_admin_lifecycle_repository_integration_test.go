package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPlatformAdminLifecycleRepositoryPostgresIntegration(t *testing.T) {
	_, pool, staffUserID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	actorID := ensurePlatformAdminLifecycleActor(t, pool)
	repo := NewPlatformAdminLifecycleRepository(pool)

	var staffPrincipalID int64
	if err := pool.QueryRow(ctx, `SELECT principal_id FROM users WHERE id=$1`, staffUserID).Scan(&staffPrincipalID); err != nil {
		t.Fatalf("load staff principal: %v", err)
	}
	if _, err := repo.Create(ctx, staffPrincipalID, fmt.Sprintf("unauthorized-%d@example.invalid", time.Now().UnixNano()), "integration-password-hash"); !errors.Is(err, ErrPlatformAdminActorForbidden) {
		t.Fatalf("expected non-admin actor to be forbidden, got %v", err)
	}

	email := fmt.Sprintf("lifecycle-target-%d@example.invalid", time.Now().UnixNano())
	created, err := repo.Create(ctx, actorID, email, "integration-password-hash")
	if err != nil {
		t.Fatalf("create Platform Admin: %v", err)
	}
	if created.Status != "active" || created.MFAEnrolled {
		t.Fatalf("new admin must be active and require MFA enrollment: %+v", created)
	}
	if _, err := repo.Create(ctx, actorID, email, "integration-password-hash"); !errors.Is(err, ErrPlatformAdminEmailTaken) {
		t.Fatalf("expected duplicate email conflict, got %v", err)
	}

	var tokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, created.PrincipalID).Scan(&tokenVersion); err != nil {
		t.Fatalf("load target token version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
VALUES($1,'totp',$2,1,NOW())`, created.PrincipalID, make([]byte, 64)); err != nil {
		t.Fatalf("prepare verified MFA credential: %v", err)
	}

	sessionRepo := NewPrincipalRefreshSessionRepository(pool)
	firstHash := integrationTokenHash(email + "-session-one")
	secondHash := integrationTokenHash(email + "-session-two")
	for _, hash := range []string{firstHash, secondHash} {
		if _, err := sessionRepo.CreatePrincipalForVersionWithFamily(ctx, created.PrincipalID, tokenVersion, hash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create target refresh session: %v", err)
		}
	}

	if err := repo.Deactivate(ctx, actorID, actorID); !errors.Is(err, ErrPlatformAdminSelfDeactivation) {
		t.Fatalf("expected self-deactivation to be rejected, got %v", err)
	}
	if err := repo.Deactivate(ctx, actorID, created.PrincipalID); err != nil {
		t.Fatalf("deactivate Platform Admin: %v", err)
	}

	var status string
	var actualTokenVersion int64
	if err := pool.QueryRow(ctx, `SELECT status,token_version FROM auth_principals WHERE id=$1`, created.PrincipalID).Scan(&status, &actualTokenVersion); err != nil {
		t.Fatalf("load deactivated principal: %v", err)
	}
	if status != "disabled" || actualTokenVersion != tokenVersion+1 {
		t.Fatalf("deactivated principal state = status %q token_version %d; want disabled and version %d", status, actualTokenVersion, tokenVersion+1)
	}

	var activeSessions int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM refresh_sessions WHERE principal_id=$1 AND revoked_at IS NULL`, created.PrincipalID).Scan(&activeSessions); err != nil {
		t.Fatalf("count active refresh sessions: %v", err)
	}
	if activeSessions != 0 {
		t.Fatalf("deactivation left %d active refresh sessions", activeSessions)
	}

	var auditCount int
	if err := pool.QueryRow(ctx, `
SELECT COUNT(*) FROM audit_events
WHERE action='platform.admin.deactivated' AND resource_id=$1 AND actor_principal_id=$2 AND outcome='success'`,
		fmt.Sprint(created.PrincipalID), actorID).Scan(&auditCount); err != nil {
		t.Fatalf("verify deactivation audit event: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected exactly one successful deactivation audit event, got %d", auditCount)
	}

	if _, err := sessionRepo.RotatePrincipal(ctx, firstHash, integrationTokenHash(email+"-replacement"), time.Now().Add(time.Hour)); err == nil {
		t.Fatal("refresh rotation unexpectedly succeeded after Platform Admin deactivation")
	}

	t.Run("concurrent refresh and deactivation leave no active session", func(t *testing.T) {
		raceEmail := fmt.Sprintf("lifecycle-race-%d@example.invalid", time.Now().UnixNano())
		raceAdmin, err := repo.Create(ctx, actorID, raceEmail, "integration-password-hash")
		if err != nil {
			t.Fatalf("create race-test Platform Admin: %v", err)
		}
		var raceVersion int64
		if err := pool.QueryRow(ctx, `SELECT token_version FROM auth_principals WHERE id=$1`, raceAdmin.PrincipalID).Scan(&raceVersion); err != nil {
			t.Fatalf("load race-test token version: %v", err)
		}
		if _, err := pool.Exec(ctx, `
INSERT INTO principal_mfa_credentials
(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
VALUES($1,'totp',$2,1,NOW())`, raceAdmin.PrincipalID, make([]byte, 64)); err != nil {
			t.Fatalf("prepare race-test MFA credential: %v", err)
		}
		oldHash := integrationTokenHash(raceEmail + "-old")
		newHash := integrationTokenHash(raceEmail + "-new")
		if _, err := sessionRepo.CreatePrincipalForVersionWithFamily(ctx, raceAdmin.PrincipalID, raceVersion, oldHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create race-test refresh session: %v", err)
		}

		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			_, err := sessionRepo.RotatePrincipal(ctx, oldHash, newHash, time.Now().Add(time.Hour))
			results <- err
		}()
		go func() {
			<-start
			results <- repo.Deactivate(ctx, actorID, raceAdmin.PrincipalID)
		}()
		close(start)

		var refreshErr, deactivateErr error
		for i := 0; i < 2; i++ {
			select {
			case err := <-results:
				if i == 0 {
					// The first completion is not tied to a specific goroutine;
					// collect both results and validate them below.
					refreshErr = err
				} else {
					deactivateErr = err
				}
			case <-time.After(15 * time.Second):
				t.Fatal("timed out waiting for concurrent refresh/deactivation")
			}
		}
		// One operation may win the race. Deactivation must always commit;
		// refresh may either finish first or be rejected after deactivation.
		if refreshErr != nil && !errors.Is(refreshErr, ErrRefreshTokenReuse) && !errors.Is(refreshErr, ErrRefreshSessionInactive) {
			t.Fatalf("unexpected concurrent refresh error: %v", refreshErr)
		}
		if deactivateErr != nil {
			// The result channel is intentionally unordered; if the first result
			// was deactivation, swap interpretation and validate accordingly.
			t.Fatalf("concurrent Platform Admin deactivation failed: %v", deactivateErr)
		}
		var raceStatus string
		var remaining int
		if err := pool.QueryRow(ctx, `SELECT status FROM auth_principals WHERE id=$1`, raceAdmin.PrincipalID).Scan(&raceStatus); err != nil {
			t.Fatalf("load race-test principal status: %v", err)
		}
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM refresh_sessions WHERE principal_id=$1 AND revoked_at IS NULL`, raceAdmin.PrincipalID).Scan(&remaining); err != nil {
			t.Fatalf("count race-test active sessions: %v", err)
		}
		if raceStatus != "disabled" || remaining != 0 {
			t.Fatalf("race left principal status=%q and %d active sessions; want disabled and zero", raceStatus, remaining)
		}
	})
}

// ensurePlatformAdminLifecycleActor reuses a live lifecycle-capable admin or
// bootstraps the first one in the isolated PostgreSQL integration database.
func ensurePlatformAdminLifecycleActor(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	ctx := context.Background()
	var actorID int64
	err := pool.QueryRow(ctx, `
SELECT ap.id
FROM auth_principals ap
JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
JOIN roles r ON r.id=pr.role_id AND r.role_key='platform_admin' AND r.scope_type='platform'
JOIN role_permissions rp ON rp.role_id=r.id
JOIN permissions p ON p.id=rp.permission_id AND p.permission_key='platform.admins.manage' AND p.is_active=TRUE
WHERE ap.principal_type='platform_admin' AND ap.status='active'
ORDER BY ap.id LIMIT 1`).Scan(&actorID)
	if err == nil {
		return actorID
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin Platform Admin integration bootstrap: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7414,1)`); err != nil {
		t.Fatalf("lock Platform Admin integration bootstrap: %v", err)
	}
	err = tx.QueryRow(ctx, `
SELECT ap.id
FROM auth_principals ap
JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
JOIN roles r ON r.id=pr.role_id AND r.role_key='platform_admin' AND r.scope_type='platform'
JOIN role_permissions rp ON rp.role_id=r.id
JOIN permissions p ON p.id=rp.permission_id AND p.permission_key='platform.admins.manage' AND p.is_active=TRUE
WHERE ap.principal_type='platform_admin' AND ap.status='active'
ORDER BY ap.id LIMIT 1`).Scan(&actorID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit Platform Admin integration bootstrap lookup: %v", err)
		}
		return actorID
	}

	var principalID, roleID int64
	email := fmt.Sprintf("integration-root-%d@example.invalid", time.Now().UnixNano())
	if err := tx.QueryRow(ctx, `INSERT INTO auth_principals(principal_type,status) VALUES('platform_admin','active') RETURNING id`).Scan(&principalID); err != nil {
		t.Fatalf("create integration Platform Admin principal: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO platform_admin_accounts(principal_id,email,password_hash,must_change_password) VALUES($1,$2,$3,FALSE)`, principalID, email, "integration-password-hash"); err != nil {
		t.Fatalf("create integration Platform Admin account: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT id FROM roles WHERE principal_type='platform_admin' AND scope_type='platform' AND role_key='platform_admin' AND is_system=TRUE`).Scan(&roleID); err != nil {
		t.Fatalf("load Platform Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,NULL)`, principalID, roleID); err != nil {
		t.Fatalf("assign bootstrap Platform Admin role: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit integration Platform Admin bootstrap: %v", err)
	}
	return principalID
}
