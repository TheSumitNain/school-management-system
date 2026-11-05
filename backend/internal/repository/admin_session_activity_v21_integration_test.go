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

	// Reuse an authorized administrator when the integration database already
	// has one. On a clean database, bootstrap a temporary first administrator
	// inside this rollback-scoped transaction; never leave a permanent account.
	var principalID int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT ap.id
			FROM auth_principals ap
			JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id=pr.role_id
			JOIN role_permissions rp ON rp.role_id=r.id
			JOIN permissions p ON p.id=rp.permission_id
			WHERE ap.principal_type='platform_admin' AND ap.status='active'
			  AND r.scope_type='platform' AND r.role_key='platform_admin'
			  AND p.permission_key='platform.school_admins.create'
			  AND p.scope_type='platform' AND p.is_active=TRUE
			  AND EXISTS (
			      SELECT 1 FROM principal_mfa_credentials mc
			       WHERE mc.principal_id=ap.id AND mc.method='totp' AND mc.verified_at IS NOT NULL
			  )
			LIMIT 1
		), 0)
	`).Scan(&principalID); err != nil {
		t.Fatalf("resolve authorized Platform Admin fixture: %v", err)
	}
	if principalID == 0 {
		var bootstrapExists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1 FROM principal_roles pr
			        JOIN roles r ON r.id=pr.role_id
			        WHERE r.principal_type='platform_admin'
			          AND r.scope_type='platform' AND r.role_key='platform_admin'
			    )
		`).Scan(&bootstrapExists); err != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", err)
		}
		if bootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.school_admins.create")
		}
		var bootstrapErr error
		principalID, bootstrapErr = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, "idle-v21-"+suffix+"@example.invalid", "integration-password-hash",
		)
		if bootstrapErr != nil {
			t.Fatalf("create rollback-scoped Platform Admin fixture: %v", bootstrapErr)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO principal_mfa_credentials
			    (principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
			VALUES ($1,'totp',decode('01','hex'),1,NOW())
		`, principalID); err != nil {
			t.Fatalf("create temporary verified Platform Admin MFA fixture: %v", err)
		}
	}

	// The production login issuance path must initialize the V21 activity
	// timestamp on the first refresh-session row, not only after a rotation.
	issuedTokenSum := sha256.Sum256([]byte("admin-idle-v21-issued-"+suffix))
	issuedTokenHash := hex.EncodeToString(issuedTokenSum[:])
	issuedFamily, err := NewPrincipalRefreshSessionRepository(tx).CreatePrincipalForVersionWithFamily(
		ctx, principalID, 1, issuedTokenHash, time.Now().Add(24*time.Hour),
	)
	if err != nil { t.Fatalf("create first session through production family issuer: %v",err) }
	var activityInitialized bool
	if err := tx.QueryRow(ctx,`
		SELECT last_activity_at > clock_timestamp()-INTERVAL '1 minute'
		FROM refresh_sessions WHERE token_hash=$1 AND family_id=$2::uuid
	`,issuedTokenHash,issuedFamily).Scan(&activityInitialized); err != nil { t.Fatalf("verify initial activity timestamp: %v",err) }
	if !activityInitialized { t.Fatal("new refresh-session family must initialize last_activity_at at login") }

	createFamily := func(label string) string {
		t.Helper()
		var familyID string
		tokenSum := sha256.Sum256([]byte("admin-idle-v21-" + suffix + "-" + label))
		tokenHash := hex.EncodeToString(tokenSum[:])
		err := tx.QueryRow(ctx, `
			INSERT INTO refresh_sessions (principal_id,token_hash,expires_at)
			VALUES ($1,$2,NOW()+INTERVAL '1 day')
			RETURNING family_id::text
		`, principalID, tokenHash).Scan(&familyID)
		if err != nil { t.Fatalf("create %s device family: %v", label, err) }
		return familyID
	}
	expiredFamily := createFamily("expired")
	activeFamily := createFamily("active")

	tracker := NewPrincipalAuthorizationRepository(tx)
	isAdmin, valid, err := tracker.CheckAndTouchAdminSession(ctx, principalID, activeFamily)
	if err != nil { t.Fatalf("touch active device family: %v", err) }
	if !isAdmin || !valid { t.Fatalf("active admin family should pass: isAdmin=%v valid=%v",isAdmin,valid) }

	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`, expiredFamily); err != nil { t.Fatalf("age expired device family: %v", err) }

	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, principalID, expiredFamily)
	if err != nil { t.Fatalf("check expired device family: %v", err) }
	if !isAdmin || valid { t.Fatalf("idle family must be rejected: isAdmin=%v valid=%v",isAdmin,valid) }

	var expiredActive, otherActive int
	if err := tx.QueryRow(ctx, `
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
	if err := tx.QueryRow(ctx, `
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
	if err := tx.QueryRow(ctx, `
		INSERT INTO refresh_sessions (principal_id,token_hash,expires_at)
		VALUES ($1,$2,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`, principalID, refreshTokenHash).Scan(&refreshFamily); err != nil { t.Fatalf("create refresh-expiry family: %v",err) }
	if _, err := tx.Exec(ctx, `
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`, refreshFamily); err != nil { t.Fatalf("age refresh-expiry family: %v",err) }
	refreshRepo := NewPrincipalRefreshSessionRepository(tx)
	if _, err := refreshRepo.RotatePrincipal(ctx,refreshTokenHash,newTokenHash,time.Now().Add(24*time.Hour)); !errors.Is(err,ErrAdminSessionIdleExpired) {
		t.Fatalf("idle admin refresh must return ErrAdminSessionIdleExpired, got %v",err)
	}
	var refreshFamilyActive int
	if err := tx.QueryRow(ctx,`SELECT COUNT(*) FROM refresh_sessions WHERE family_id=$1::uuid AND revoked_at IS NULL`,refreshFamily).Scan(&refreshFamilyActive); err != nil { t.Fatalf("verify idle refresh family revocation: %v",err) }
	if refreshFamilyActive != 0 { t.Fatalf("idle refresh family still has %d active sessions",refreshFamilyActive) }

	// Ordinary staff sessions do not use the administrative idle-timeout rule.
	var staffPrincipalID int64
	if err := tx.QueryRow(ctx,`SELECT principal_id FROM users WHERE id=$1`,userID).Scan(&staffPrincipalID); err != nil { t.Fatalf("load integration staff principal: %v",err) }

	// Promote this fixture staff account to the exact system School Admin role
	// using the authorized Platform Admin resolved above.
	var schoolID, schoolAdminRoleID int64
	if err := tx.QueryRow(ctx,`SELECT school_id FROM users WHERE id=$1`,userID).Scan(&schoolID); err != nil { t.Fatalf("load School Admin school: %v",err) }
	if err := tx.QueryRow(ctx,`
		SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		  AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE
	`,schoolID).Scan(&schoolAdminRoleID); err != nil { t.Fatalf("load School Admin system role: %v",err) }
	if _, err := tx.Exec(ctx,`
		INSERT INTO principal_mfa_credentials
		    (principal_id,method,secret_ciphertext,encryption_key_version,verified_at)
		VALUES($1,'totp',decode('01','hex'),1,NOW())
	`,staffPrincipalID); err != nil { t.Fatalf("create School Admin MFA fixture: %v",err) }
	provisioningActorID := principalID
	if _, err := tx.Exec(ctx,`
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`,staffPrincipalID,schoolAdminRoleID,provisioningActorID); err != nil { t.Fatalf("assign exact School Admin system role: %v",err) }

	schoolAdminTokenSum := sha256.Sum256([]byte("school-admin-idle-v21-"+suffix))
	schoolAdminTokenHash := hex.EncodeToString(schoolAdminTokenSum[:])
	var schoolAdminFamily string
	if err := tx.QueryRow(ctx,`
		INSERT INTO refresh_sessions(user_id,principal_id,token_hash,expires_at)
		VALUES($1,$2,$3,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`,userID,staffPrincipalID,schoolAdminTokenHash).Scan(&schoolAdminFamily); err != nil { t.Fatalf("create School Admin family: %v",err) }
	if _, err := tx.Exec(ctx,`
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`,schoolAdminFamily); err != nil { t.Fatalf("age School Admin family: %v",err) }
	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx,staffPrincipalID,schoolAdminFamily)
	if err != nil { t.Fatalf("check School Admin idle timeout: %v",err) }
	if !isAdmin || valid { t.Fatalf("active system School Admin role must enforce idle timeout: isAdmin=%v valid=%v",isAdmin,valid) }
	schoolAdminRefreshSum := sha256.Sum256([]byte("school-admin-refresh-idle-v21-"+suffix))
	schoolAdminRefreshHash := hex.EncodeToString(schoolAdminRefreshSum[:])
	schoolAdminRefreshReplacementSum := sha256.Sum256([]byte("school-admin-refresh-idle-v21-replacement-"+suffix))
	schoolAdminRefreshReplacementHash := hex.EncodeToString(schoolAdminRefreshReplacementSum[:])
	var schoolAdminRefreshFamily string
	if err := tx.QueryRow(ctx,`
		INSERT INTO refresh_sessions(user_id,principal_id,token_hash,expires_at)
		VALUES($1,$2,$3,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`,userID,staffPrincipalID,schoolAdminRefreshHash).Scan(&schoolAdminRefreshFamily); err != nil { t.Fatalf("create School Admin refresh family: %v",err) }
	if _, err := tx.Exec(ctx,`
		UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes'
		WHERE family_id=$1::uuid AND revoked_at IS NULL
	`,schoolAdminRefreshFamily); err != nil { t.Fatalf("age School Admin refresh family: %v",err) }
	if _, err := refreshRepo.RotatePrincipal(ctx,schoolAdminRefreshHash,schoolAdminRefreshReplacementHash,time.Now().Add(24*time.Hour)); !errors.Is(err,ErrAdminSessionIdleExpired) {
		t.Fatalf("School Admin refresh must enforce idle timeout: %v",err)
	}
	var ordinaryUserID, ordinaryStaffPrincipalID int64
	if err := tx.QueryRow(ctx,`
		INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,'integration-password-hash','active')
		RETURNING id,principal_id
	`,schoolID,"o"+suffix).Scan(&ordinaryUserID,&ordinaryStaffPrincipalID); err != nil {
		t.Fatalf("create separate ordinary staff fixture: %v",err)
	}
	staffTokenSum := sha256.Sum256([]byte("staff-idle-v21-" + suffix))
	staffTokenHash := hex.EncodeToString(staffTokenSum[:])
	staffReplacementSum := sha256.Sum256([]byte("staff-idle-v21-replacement-" + suffix))
	staffReplacementHash := hex.EncodeToString(staffReplacementSum[:])
	var staffFamily string
	if err := tx.QueryRow(ctx,`
		INSERT INTO refresh_sessions(user_id,principal_id,token_hash,expires_at)
		VALUES($1,$2,$3,NOW()+INTERVAL '1 day') RETURNING family_id::text
	`,ordinaryUserID,ordinaryStaffPrincipalID,staffTokenHash).Scan(&staffFamily); err != nil { t.Fatalf("create ordinary staff family: %v",err) }
	if _, err := tx.Exec(ctx,`UPDATE refresh_sessions SET last_activity_at=NOW()-INTERVAL '31 minutes' WHERE family_id=$1::uuid AND revoked_at IS NULL`,staffFamily); err != nil { t.Fatalf("age ordinary staff family: %v",err) }
	rotatedPrincipal, err := refreshRepo.RotatePrincipal(ctx,staffTokenHash,staffReplacementHash,time.Now().Add(24*time.Hour))
	if err != nil { t.Fatalf("ordinary staff refresh must not be subject to admin idle timeout: %v",err) }
	if rotatedPrincipal == nil || rotatedPrincipal.ID != ordinaryStaffPrincipalID { t.Fatalf("ordinary staff refresh returned wrong principal: %+v",rotatedPrincipal) }

	// Ordinary staff are not subject to the admin idle timeout, but every
	// protected request must still prove that its refresh-session family is
	// active and belongs to the same principal.
	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, ordinaryStaffPrincipalID, staffFamily)
	if err != nil { t.Fatalf("validate ordinary staff family: %v", err) }
	if isAdmin || !valid { t.Fatalf("ordinary staff family should remain valid beyond admin idle window: isAdmin=%v valid=%v", isAdmin, valid) }
	if _, err := tx.Exec(ctx, `UPDATE refresh_sessions SET revoked_at=NOW() WHERE family_id=$1::uuid AND revoked_at IS NULL`, staffFamily); err != nil {
		t.Fatalf("revoke ordinary staff family: %v", err)
	}
	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, ordinaryStaffPrincipalID, staffFamily)
	if err != nil { t.Fatalf("validate revoked ordinary staff family: %v", err) }
	if isAdmin || valid { t.Fatalf("revoked ordinary staff family must be rejected: isAdmin=%v valid=%v", isAdmin, valid) }


	isAdmin, valid, err = tracker.CheckAndTouchAdminSession(ctx, principalID, activeFamily)
	if err != nil { t.Fatalf("recheck other device family: %v", err) }
	if !isAdmin || !valid { t.Fatalf("other device family must remain usable: isAdmin=%v valid=%v",isAdmin,valid) }
}


