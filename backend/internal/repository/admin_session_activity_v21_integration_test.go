package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestAdminSessionActivityV21PostgresIntegration verifies that activity is
// tracked per device family, expired families are revoked and audited, and
// another device belonging to the same administrator remains active.
func TestAdminSessionActivityV21PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
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
		tokenSum := sha256.Sum256([]byte("admin-idle-v21-" + suffix + "-" + label))
		tokenHash := hex.EncodeToString(tokenSum[:])
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

	// Refresh rotation must enforce the same idle timeout inside its existing
	// family/principal-locked transaction and revoke only the expired family.
	refreshTokenSum := sha256.Sum256([]byte("admin-idle-v21-refresh-" + suffix))
	refreshTokenHash := hex.EncodeToString(refreshTokenSum[:])
	newTokenSum := sha256.Sum256([]byte("admin-idle-v21-refresh-replacement-" + suffix))
	newTokenHash := hex.EncodeToString(newTokenSum[:])
	var refreshFamily string
	if err := pool.QueryRow(ctx, `
		INSERT INTO refresh_sessions (principal_id,token_hash,expires_at)
		VALUES ($1,$2,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`, principalID, refreshTokenHash).Scan(&refreshFamily); err != nil { t.Fatalf("create refresh-expiry family: %v",err) }
	if _, err := pool.Exec(ctx, `
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`, refreshFamily); err != nil { t.Fatalf("age refresh-expiry family: %v",err) }
	refreshRepo := NewPrincipalRefreshSessionRepository(pool)
	if _, err := refreshRepo.RotatePrincipal(ctx,refreshTokenHash,newTokenHash,time.Now().Add(24*time.Hour)); !errors.Is(err,ErrAdminSessionIdleExpired) {
		t.Fatalf("idle admin refresh must return ErrAdminSessionIdleExpired, got %v",err)
	}
	var refreshFamilyActive int
	if err := pool.QueryRow(ctx,`SELECT COUNT(*) FROM refresh_sessions WHERE family_id=$1::uuid AND revoked_at IS NULL`,refreshFamily).Scan(&refreshFamilyActive); err != nil { t.Fatalf("verify idle refresh family revocation: %v",err) }
	if refreshFamilyActive != 0 { t.Fatalf("idle refresh family still has %d active sessions",refreshFamilyActive) }

	// Ordinary staff sessions do not use the administrative idle-timeout rule.
	var staffPrincipalID int64
	if err := pool.QueryRow(ctx,`SELECT principal_id FROM users WHERE id=$1`,userID).Scan(&staffPrincipalID); err != nil { t.Fatalf("load integration staff principal: %v",err) }
	staffTokenSum := sha256.Sum256([]byte("staff-idle-v21-" + suffix))
	staffTokenHash := hex.EncodeToString(staffTokenSum[:])
	staffReplacementSum := sha256.Sum256([]byte("staff-idle-v21-replacement-" + suffix))
	staffReplacementHash := hex.EncodeToString(staffReplacementSum[:])
	var staffFamily string
	if err := pool.QueryRow(ctx,`
		INSERT INTO refresh_sessions(user_id,principal_id,token_hash,expires_at)
		VALUES($1,$2,$3,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`,userID,staffPrincipalID,staffTokenHash).Scan(&staffFamily); err != nil { t.Fatalf("create ordinary staff family: %v",err) }
	if _, err := pool.Exec(ctx,`UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes' WHERE family_id=$1::uuid AND revoked_at IS NULL`,staffFamily); err != nil { t.Fatalf("age ordinary staff family: %v",err) }
	rotatedPrincipal, err := refreshRepo.RotatePrincipal(ctx,staffTokenHash,staffReplacementHash,time.Now().Add(24*time.Hour))
	if err != nil { t.Fatalf("ordinary staff refresh must not be subject to admin idle timeout: %v",err) }
	if rotatedPrincipal == nil || rotatedPrincipal.ID != staffPrincipalID { t.Fatalf("ordinary staff refresh returned wrong principal: %+v",rotatedPrincipal) }

	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, principalID, activeFamily)
	if err != nil { t.Fatalf("recheck other device family: %v", err) }
	if !isAdmin || !valid { t.Fatalf("other device family must remain usable: isAdmin=%v valid=%v",isAdmin,valid) }
}


func TestSchoolAdminIdleTimeoutV21PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var schoolID, schoolAdminPrincipalID int64
	if err := pool.QueryRow(ctx, `SELECT school_id,principal_id FROM users WHERE id=$1`, userID).Scan(&schoolID,&schoolAdminPrincipalID); err != nil {
		t.Fatalf("load fixture staff identity: %v",err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin School Admin fixture transaction: %v",err) }
	defer func() { _ = tx.Rollback(ctx) }()
	var platformPrincipalID int64
	if err := tx.QueryRow(ctx,`
		INSERT INTO auth_principals(principal_type,status) VALUES('platform_admin','active') RETURNING id
	`).Scan(&platformPrincipalID); err != nil { t.Fatalf("create Platform Admin actor: %v",err) }
	if _, err := tx.Exec(ctx,`
		INSERT INTO platform_admin_accounts(principal_id,email,password_hash,must_change_password,mfa_enrolled_at)
		VALUES($1,$2,'integration-password-hash',FALSE,NOW())
	`,platformPrincipalID,"school-admin-idle-actor-"+suffix+"@example.invalid"); err != nil { t.Fatalf("create Platform Admin actor account: %v",err) }
	if _, err := tx.Exec(ctx,`
		INSERT INTO principal_mfa_credentials(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES($1,'totp',decode('01','hex'),1,NOW())
	`,platformPrincipalID); err != nil { t.Fatalf("create Platform Admin MFA: %v",err) }

	if _, err := tx.Exec(ctx,`
		INSERT INTO principal_mfa_credentials(principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES($1,'totp',decode('01','hex'),1,NOW())
	`,schoolAdminPrincipalID); err != nil { t.Fatalf("create School Admin MFA: %v",err) }
	var roleID int64
	if err := tx.QueryRow(ctx,`
		SELECT id FROM roles WHERE school_id=$1 AND role_key='school_admin'
		  AND principal_type='staff' AND scope_type='school' AND is_system=TRUE
	`,schoolID).Scan(&roleID); err != nil { t.Fatalf("load School Admin system role: %v",err) }
	if _, err := tx.Exec(ctx,`
		INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)
`,schoolAdminPrincipalID,roleID,platformPrincipalID); err != nil { t.Fatalf("assign School Admin system role: %v",err) }
	if err := tx.Commit(ctx); err != nil { t.Fatalf("commit School Admin fixture: %v",err) }

	tokenSum := sha256.Sum256([]byte("school-admin-idle-v21-"+suffix))
	tokenHash := hex.EncodeToString(tokenSum[:])
	var familyID string
	if err := tx.QueryRow(ctx,`
		INSERT INTO refresh_sessions(user_id,principal_id,token_hash,expires_at)
		VALUES($1,$2,$3,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`,userID,schoolAdminPrincipalID,tokenHash).Scan(&familyID); err != nil { t.Fatalf("create School Admin refresh family: %v",err) }
	if _, err := tx.Exec(ctx,`
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`,familyID); err != nil { t.Fatalf("age School Admin session family: %v",err) }

	isAdmin, valid, err := NewPrincipalAuthorizationRepository(pool).CheckAndTouchAdminSession(ctx,schoolAdminPrincipalID,familyID)
	if err != nil { t.Fatalf("check School Admin session activity: %v",err) }
	if !isAdmin || valid { t.Fatalf("active School Admin system role must be idle-expired: isAdmin=%v valid=%v",isAdmin,valid) }
}
