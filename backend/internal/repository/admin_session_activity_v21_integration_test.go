package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

// TestPersistentAdminSessionsV25PostgresIntegration proves that Platform Admin
// and School Admin session families remain valid after long inactivity, while
// protected-request validation and transactional refresh rotation still work.
func TestPersistentAdminSessionsV25PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin persistent-session fixture transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var platformAdminID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		FROM auth_principals ap
		JOIN platform_admin_accounts pa ON pa.principal_id=ap.id
		WHERE ap.principal_type='platform_admin' AND ap.status='active'
		ORDER BY ap.id
		LIMIT 1
	`).Scan(&platformAdminID)
	if err != nil {
		if err.Error() != "no rows in result set" {
			t.Fatalf("find active Platform Admin fixture: %v", err)
		}
		var anyPlatformAdmin bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)`).Scan(&anyPlatformAdmin); err != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", err)
		}
		if anyPlatformAdmin {
			t.Skip("Platform Admin accounts exist but none is active")
		}
		platformAdminID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, "persistent-v25-"+suffix+"@example.invalid", "integration-password-hash",
		)
		if err != nil {
			t.Fatalf("create rollback-scoped Platform Admin fixture: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM principal_mfa_credentials WHERE principal_id=$1 AND method='totp'`, platformAdminID); err != nil {
		t.Fatalf("clear temporary Platform Admin MFA fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
		    (principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES ($1,'totp',decode('01','hex'),1,NOW())
	`, platformAdminID); err != nil {
		t.Fatalf("create verified Platform Admin MFA fixture: %v", err)
	}

	var schoolAdminID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id,school_id FROM users WHERE id=$1
	`, userID).Scan(&schoolAdminID, &schoolID); err != nil {
		t.Fatalf("load School Admin staff fixture: %v", err)
	}
	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		WHERE school_id=$1 AND principal_type='staff'
		  AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		SELECT $1,$2,$3
		WHERE NOT EXISTS (
			SELECT 1 FROM principal_roles
			WHERE principal_id=$1 AND role_id=$2 AND revoked_at IS NULL
		)
	`, schoolAdminID, schoolAdminRoleID, platformAdminID); err != nil {
		t.Fatalf("assign School Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM principal_mfa_credentials WHERE principal_id=$1 AND method='totp'`, schoolAdminID); err != nil {
		t.Fatalf("clear temporary School Admin MFA fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
		    (principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES ($1,'totp',decode('01','hex'),1,NOW())
	`, schoolAdminID); err != nil {
		t.Fatalf("create verified School Admin MFA fixture: %v", err)
	}

	tracker := NewPrincipalAuthorizationRepository(tx)
	refreshRepo := NewPrincipalRefreshSessionRepository(tx)
	assertPersistentFamily := func(label string, principalID int64, wantAdmin bool) {
		t.Helper()
		raw := "persistent-v25-" + suffix + "-" + label
		sum := sha256.Sum256([]byte(raw))
		oldHash := hex.EncodeToString(sum[:])
		replacementSum := sha256.Sum256([]byte(raw + "-replacement"))
		newHash := hex.EncodeToString(replacementSum[:])

		familyID, err := refreshRepo.CreatePrincipalForVersionWithFamily(ctx, principalID, 1, oldHash, time.Time{})
		if err != nil {
			t.Fatalf("create %s persistent session family: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE refresh_sessions
			SET last_activity_at=clock_timestamp()-INTERVAL '45 days'
			WHERE family_id=$1::uuid AND revoked_at IS NULL
		`, familyID); err != nil {
			t.Fatalf("age %s session activity: %v", label, err)
		}

		isAdmin, valid, err := tracker.CheckAndTouchAdminSession(ctx, principalID, familyID)
		if err != nil {
			t.Fatalf("validate inactive-for-45-days %s family: %v", label, err)
		}
		if isAdmin != wantAdmin || !valid {
			t.Fatalf("%s family should remain valid after 45 days: isAdmin=%v valid=%v", label, isAdmin, valid)
		}

		principal, err := refreshRepo.RotatePrincipal(ctx, oldHash, newHash, time.Time{})
		if err != nil {
			t.Fatalf("rotate %s persistent session after 45 days: %v", label, err)
		}
		if principal == nil || principal.ID != principalID {
			t.Fatalf("%s refresh returned wrong principal: %+v", label, principal)
		}

		var activeCount, infiniteExpiryCount int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FILTER (WHERE revoked_at IS NULL),
			       COUNT(*) FILTER (WHERE revoked_at IS NULL AND expires_at='infinity'::timestamptz)
			FROM refresh_sessions WHERE family_id=$1::uuid
		`, familyID).Scan(&activeCount, &infiniteExpiryCount); err != nil {
			t.Fatalf("verify %s family after rotation: %v", label, err)
		}
		if activeCount != 1 || infiniteExpiryCount != 1 {
			t.Fatalf("%s family must have one active non-expiring replacement; active=%d infinite=%d", label, activeCount, infiniteExpiryCount)
		}

		var idleAuditCount int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM audit_events
			WHERE actor_principal_id=$1 AND action='auth.admin_session_idle_timeout'
			  AND resource_id=$2
		`, principalID, familyID).Scan(&idleAuditCount); err != nil {
			t.Fatalf("check obsolete idle-timeout audit for %s: %v", label, err)
		}
		if idleAuditCount != 0 {
			t.Fatalf("%s session unexpectedly generated an idle-timeout audit", label)
		}
	}

	assertPersistentFamily("platform-admin", platformAdminID, true)
	assertPersistentFamily("school-admin", schoolAdminID, true)
}
