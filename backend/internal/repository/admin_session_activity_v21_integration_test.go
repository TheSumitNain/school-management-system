package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestAdminSessionActivityV21PostgresIntegration verifies that activity is
// tracked per device family, expired families are revoked and audited, and
// another device belonging to the same administrator remains active.
func TestAdminSessionActivityV21PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin admin fixture transaction: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var principalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_principals (principal_type,status)
		VALUES ('platform_admin','active') RETURNING id
	`).Scan(&principalID); err != nil { t.Fatalf("create Platform Admin principal: %v", err) }
	if _, err := tx.Exec(ctx, `
		INSERT INTO platform_admin_accounts (principal_id,email,password_hash,must_change_password,mfa_enrolled_at)
		VALUES ($1,$2,'integration-password-hash',FALSE,NOW())
	`, principalID, "idle-v21-"+suffix+"@example.invalid"); err != nil { t.Fatalf("create Platform Admin account: %v", err) }
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
			(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES ($1,'totp',decode('01','hex'),1,NOW())
	`, principalID); err != nil { t.Fatalf("create verified MFA credential: %v", err) }
	if err := tx.Commit(ctx); err != nil { t.Fatalf("commit Platform Admin fixture: %v", err) }

	createFamily := func(label string) string {
		t.Helper()
		var familyID string
		tokenHash := HashRefreshToken("admin-idle-v21-" + suffix + "-" + label)
		err := pool.QueryRow(ctx, `
			INSERT INTO refresh_sessions (principal_id,token_hash,expires_at)
			VALUES ($1,$2,NOW()+INTERVAL '1 day')
			RETURNING family_id::text
		`, principalID, tokenHash).Scan(&familyID)
		if err != nil { t.Fatalf("create %s device family: %v", label, err) }
		return familyID
	}
	expiredFamily := createFamily("expired")
	activeFamily := createFamily("active")

	tracker := NewPrincipalAuthorizationRepository(pool)
	isAdmin, valid, err := tracker.CheckAndTouchAdminSession(ctx, principalID, activeFamily)
	if err != nil { t.Fatalf("touch active device family: %v", err) }
	if !isAdmin || !valid { t.Fatalf("active admin family should pass: isAdmin=%v valid=%v",isAdmin,valid) }

	if _, err := pool.Exec(ctx, `
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`, expiredFamily); err != nil { t.Fatalf("age expired device family: %v", err) }

	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, principalID, expiredFamily)
	if err != nil { t.Fatalf("check expired device family: %v", err) }
	if !isAdmin || valid { t.Fatalf("idle family must be rejected: isAdmin=%v valid=%v",isAdmin,valid) }

	var expiredActive, otherActive int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE family_id=$1::uuid AND revoked_at IS NULL),
		       COUNT(*) FILTER (WHERE family_id=$2::uuid AND revoked_at IS NULL)
		FROM refresh_sessions WHERE principal_id=$3
	`, expiredFamily, activeFamily, principalID).Scan(&expiredActive,&otherActive); err != nil {
		t.Fatalf("verify device-family isolation: %v", err)
	}
	if expiredActive != 0 || otherActive != 1 {
		t.Fatalf("expected expired family revoked and other family active; expired=%d active=%d",expiredActive,otherActive)
	}

	var auditCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		WHERE actor_principal_id=$1 AND action='auth.admin_session_idle_timeout'
		  AND resource_id=$2
	`, principalID, expiredFamily).Scan(&auditCount); err != nil { t.Fatalf("verify idle-timeout audit: %v", err) }
	if auditCount != 1 { t.Fatalf("expected exactly one idle-timeout audit event, got %d",auditCount) }

	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, principalID, activeFamily)
	if err != nil { t.Fatalf("recheck other device family: %v", err) }
	if !isAdmin || !valid { t.Fatalf("other device family must remain usable: isAdmin=%v valid=%v",isAdmin,valid) }
}
