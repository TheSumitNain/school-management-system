package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestStaffAccountRepositoryV17PostgresIntegration exercises the staff account
// repository against the dedicated school_management_test database. The
// fixture actor is deliberately an ordinary staff principal, so every
// management operation must be denied unless the repository independently
// verifies the School Admin role and permission.
func TestStaffAccountRepositoryV17PostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, `
		SELECT principal_id, school_id
		  FROM users
		 WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load ordinary staff fixture: %v", err)
	}

	repo := NewUserRepository(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	targetUsername := "staff-target-" + suffix
	targetPasswordHash := "integration-test-target-password-hash"

	var targetUserID, targetPrincipalID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id
	`, schoolID, targetUsername, targetPasswordHash).Scan(&targetUserID, &targetPrincipalID); err != nil {
		t.Fatalf("create same-school target staff account: %v", err)
	}

	var secondSchoolID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, "Staff Account Isolation "+suffix, "SAI-"+suffix).Scan(&secondSchoolID); err != nil {
		t.Fatalf("create second integration school: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM refresh_sessions WHERE principal_id = $1`, targetPrincipalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE principal_id = $1`, targetPrincipalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_principals WHERE id = $1`, targetPrincipalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schools WHERE id = $1`, secondSchoolID)
	})

	var otherSchoolUserID, otherSchoolPrincipalID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id
	`, secondSchoolID, "other-"+suffix, targetPasswordHash).Scan(
		&otherSchoolUserID, &otherSchoolPrincipalID,
	); err != nil {
		t.Fatalf("create other-school staff account: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, otherSchoolUserID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_principals WHERE id = $1`, otherSchoolPrincipalID)
	})

	t.Run("ordinary staff cannot create or list accounts", func(t *testing.T) {
		if _, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID,
			"unauthorized-"+suffix, "not-a-real-password-hash"); !errors.Is(err, ErrRolePermissionDenied) {
			t.Fatalf("CreateStaffAccount error = %v, want ErrRolePermissionDenied", err)
		}

		var unauthorizedCount int
		if err := pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM users WHERE username = $1
		`, "unauthorized-"+suffix).Scan(&unauthorizedCount); err != nil {
			t.Fatalf("check unauthorized account creation: %v", err)
		}
		if unauthorizedCount != 0 {
			t.Fatalf("unauthorized creation left %d user rows", unauthorizedCount)
		}

		if _, err := repo.ListStaffAccounts(ctx, actorPrincipalID, schoolID); !errors.Is(err, ErrRolePermissionDenied) {
			t.Fatalf("ListStaffAccounts error = %v, want ErrRolePermissionDenied", err)
		}
	})

	t.Run("ordinary staff cannot disable same-school or cross-school accounts", func(t *testing.T) {
		sessionHash := integrationTokenHash("staff-account-disable-" + suffix)
		sessions := NewPrincipalRefreshSessionRepository(pool)
		if err := sessions.CreatePrincipal(ctx, targetPrincipalID, sessionHash, time.Now().Add(time.Hour)); err != nil {
			t.Fatalf("create target refresh session: %v", err)
		}

		if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("same-school unauthorized disable error = %v, want ErrUserNotFound", err)
		}
		if err := repo.DisableStaffAccount(ctx, actorPrincipalID, secondSchoolID, otherSchoolPrincipalID); !errors.Is(err, ErrUserNotFound) {
			t.Fatalf("cross-school disable error = %v, want ErrUserNotFound", err)
		}

		var userStatus, principalStatus string
		var tokenVersion int64
		var sessionActive bool
		if err := pool.QueryRow(ctx, `
			SELECT u.status, ap.status, ap.token_version,
			       EXISTS (
			           SELECT 1 FROM refresh_sessions rs
			            WHERE rs.token_hash = $3 AND rs.revoked_at IS NULL
			       )
			  FROM users u
			  JOIN auth_principals ap ON ap.id = u.principal_id
			 WHERE u.id = $1 AND ap.id = $2
		`, targetUserID, targetPrincipalID, sessionHash).Scan(
			&userStatus, &principalStatus, &tokenVersion, &sessionActive,
		); err != nil {
			t.Fatalf("verify unauthorized disable had no side effects: %v", err)
		}
		if userStatus != "active" || principalStatus != "active" || tokenVersion != 1 || !sessionActive {
			t.Fatalf("unauthorized disable changed target state: user=%q principal=%q token_version=%d session_active=%v",
				userStatus, principalStatus, tokenVersion, sessionActive)
		}
	})

	t.Run("staff usernames are globally unique case-insensitively", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO users (school_id, username, password_hash, status)
			VALUES ($1, $2, $3, 'active')
		`, secondSchoolID, "  "+upperASCII(targetUsername), targetPasswordHash)
		if err == nil {
			t.Fatal("expected case-insensitive duplicate username to be rejected across schools")
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != "ux_users_username_global" {
			t.Fatalf("duplicate username error = %v, want unique violation ux_users_username_global", err)
		}

		var originalSchool int64
		if err := pool.QueryRow(ctx,
			`SELECT school_id FROM users WHERE principal_id = $1`, targetPrincipalID,
		).Scan(&originalSchool); err != nil {
			t.Fatalf("verify original username owner: %v", err)
		}
		if originalSchool != schoolID {
			t.Fatalf("duplicate attempt changed original school: got %d, want %d", originalSchool, schoolID)
		}
	})

	// Confirm the integration fixture is using the current V4 principal model.
	var principalType string
	if err := pool.QueryRow(ctx,
		`SELECT principal_type FROM auth_principals WHERE id = $1`, actorPrincipalID,
	).Scan(&principalType); err != nil {
		t.Fatalf("verify fixture principal: %v", err)
	}
	if principalType != "staff" {
		t.Fatalf("fixture principal type = %q, want staff", principalType)
	}

	// Ensure the repository's normal no-row contract remains distinguishable
	// from a database failure for an unknown principal identifier.
	if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, 9_223_372_036_854_775_000); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("disable unknown principal error = %v, want ErrUserNotFound", err)
	}

	// The fixture actor must still resolve to its own user row after the tests.
	var actorStillExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND principal_id = $2)
	`, actorUserID, actorPrincipalID).Scan(&actorStillExists); err != nil {
		t.Fatalf("verify actor fixture remains intact: %v", err)
	}
	if !actorStillExists {
		t.Fatal("staff account tests unexpectedly modified the actor fixture")
	}

}

func upperASCII(value string) string {
	bytes := []byte(value)
	for i, b := range bytes {
		if b >= 'a' && b <= 'z' {
			bytes[i] = b - ('a' - 'A')
		}
	}
	return string(bytes)
}


func TestStaffAccountUpdateAndEnableAuthorizationPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx,
		`SELECT principal_id, school_id FROM users WHERE id = $1`, actorUserID,
	).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load ordinary staff actor: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var activePrincipalID, disabledPrincipalID int64
	var activeUserID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'existing-test-hash', 'active')
		RETURNING id, principal_id
	`, schoolID, "active-"+suffix).Scan(&activeUserID, &activePrincipalID); err != nil {
		t.Fatalf("create active target account: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'disabled-test-hash', 'disabled')
		RETURNING principal_id
	`, schoolID, "disabled-"+suffix).Scan(&disabledPrincipalID); err != nil {
		t.Fatalf("create disabled target account: %v", err)
	}

	repo := NewUserRepository(pool)
	if _, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, activePrincipalID, "renamed-"+suffix); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unauthorized UpdateStaffAccount error = %v, want ErrUserNotFound", err)
	}
	if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, disabledPrincipalID, "valid-test-password-hash"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("unauthorized EnableStaffAccount error = %v, want ErrUserNotFound", err)
	}

	var activeUsername, disabledUserStatus, disabledPrincipalStatus string
	if err := pool.QueryRow(ctx, `
		SELECT username FROM users WHERE id = $1
	`, activeUserID).Scan(&activeUsername); err != nil {
		t.Fatalf("verify active username after denied update: %v", err)
	}
	if activeUsername != "active-"+suffix {
		t.Fatalf("unauthorized update changed username to %q", activeUsername)
	}
	if err := pool.QueryRow(ctx, `
		SELECT u.status, ap.status
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, disabledPrincipalID).Scan(&disabledUserStatus, &disabledPrincipalStatus); err != nil {
		t.Fatalf("verify disabled account after denied re-enable: %v", err)
	}
	if disabledUserStatus != "disabled" || disabledPrincipalStatus != "disabled" {
		t.Fatalf("unauthorized re-enable changed account state: user=%q principal=%q",
			disabledUserStatus, disabledPrincipalStatus)
	}
}
