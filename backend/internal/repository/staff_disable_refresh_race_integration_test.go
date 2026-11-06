package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestStaffDisableSerializesWithPrincipalRefreshRotationPostgresIntegration
// runs real repository operations concurrently. Rotation is paused after it
// obtains the principal advisory lock; staff disable must wait, then revoke
// the replacement session. The outer disable transaction is rolled back so
// this test leaves no audit rows or administrator accounts behind.
func TestStaffDisableSerializesWithPrincipalRefreshRotationPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, "SELECT principal_id,school_id FROM users WHERE id=$1", actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	shortSuffix := suffix[len(suffix)-8:]
	var targetUserID, targetPrincipalID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (school_id,username,password_hash,status) VALUES ($1,$2,$3,'active') RETURNING id,principal_id",
		schoolID, "race-"+shortSuffix, "race-target-password-hash").Scan(&targetUserID, &targetPrincipalID); err != nil {
		t.Fatalf("create target staff principal: %v", err)
	}
	sessions := NewPrincipalRefreshSessionRepository(pool)
	oldHash := integrationTokenHash("staff-disable-refresh-race-old-" + suffix)
	newHash := integrationTokenHash("staff-disable-refresh-race-new-" + suffix)
	if err := sessions.CreatePrincipal(ctx, targetPrincipalID, oldHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create initial target refresh session: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupTx, err := pool.Begin(cleanupCtx)
		if err != nil {
			t.Errorf("begin race fixture cleanup: %v", err)
			return
		}
		defer func() { _ = cleanupTx.Rollback(cleanupCtx) }()
		_, err = cleanupTx.Exec(cleanupCtx, "UPDATE refresh_sessions SET revoked_at=COALESCE(revoked_at,NOW()) WHERE principal_id=$1", targetPrincipalID)
		if err == nil {
			_, err = cleanupTx.Exec(cleanupCtx, "DELETE FROM refresh_sessions WHERE principal_id=$1", targetPrincipalID)
		}
		if err == nil {
			_, err = cleanupTx.Exec(cleanupCtx, "DELETE FROM users WHERE id=$1", targetUserID)
		}
		if err == nil {
			_, err = cleanupTx.Exec(cleanupCtx, "DELETE FROM auth_principals WHERE id=$1", targetPrincipalID)
		}
		if err == nil {
			err = cleanupTx.Commit(cleanupCtx)
		}
		if err != nil {
			t.Errorf("clean up race fixtures transactionally: %v", err)
		}
	})

	// Prepare an authorized School Admin actor inside an outer transaction.
	// DisableStaffAccount's nested transaction commits only to this savepoint;
	// rolling back the outer transaction also rolls back its audit event.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped School Admin fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, "SELECT ap.id FROM platform_admin_accounts pa JOIN auth_principals ap ON ap.id=pa.principal_id JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL JOIN roles r ON r.id=pr.role_id JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE ap.principal_type='platform_admin' AND ap.status='active' AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE AND p.permission_key='platform.school_admins.create' AND p.scope_type='platform' AND p.is_active=TRUE LIMIT 1").Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM platform_admin_accounts) OR EXISTS(SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id WHERE r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin')").Scan(&exists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if exists {
			t.Skip("existing Platform Admin is not eligible to assign the School Admin system role")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx, tx,
			fmt.Sprintf("staff-refresh-race-%d@example.invalid", time.Now().UnixNano()), "integration-test-password-hash")
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO principal_mfa_credentials (principal_id,method,secret_ciphertext,encryption_key_version,verified_at) VALUES ($1,'totp',$2,1,NOW())",
		actorPrincipalID, []byte("race-school-admin-mfa-fixture")); err != nil {
		t.Fatalf("create School Admin MFA fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO principal_roles (principal_id,role_id,assigned_by) SELECT $1,r.id,$2 FROM roles r WHERE r.school_id=$3 AND r.principal_type='staff' AND r.scope_type='school' AND r.role_key='school_admin' AND r.is_system=TRUE",
		actorPrincipalID, platformAdminPrincipalID, schoolID); err != nil {
		t.Fatalf("assign School Admin role to race actor: %v", err)
	}

	functionName := "test_staff_refresh_race_fn_" + shortSuffix
	triggerName := "test_staff_refresh_race_" + shortSuffix
	const barrierKey int64 = 73482917364829179
	createFunction := fmt.Sprintf("CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN IF NEW.token_hash = ''%s'' THEN PERFORM pg_advisory_xact_lock(%d); END IF; RETURN NEW; END;'", functionName, newHash, barrierKey)
	if _, err := pool.Exec(ctx, createFunction); err != nil {
		t.Fatalf("create refresh insert barrier function: %v", err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON refresh_sessions FOR EACH ROW EXECUTE FUNCTION %s()", triggerName, functionName)); err != nil {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
		t.Fatalf("create refresh insert barrier trigger: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON refresh_sessions", triggerName))
		_, _ = pool.Exec(cleanupCtx, fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
	})

	barrierConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire refresh barrier connection: %v", err)
	}
	defer barrierConn.Release()
	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_lock($1)", barrierKey); err != nil {
		t.Fatalf("hold refresh insert barrier: %v", err)
	}
	barrierHeld := true
	defer func() {
		if barrierHeld {
			_, _ = barrierConn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", barrierKey)
		}
	}()

	results := make(chan operationResult, 2)
	go func() {
		_, rotateErr := sessions.RotatePrincipal(ctx, oldHash, newHash, time.Now().Add(time.Hour))
		results <- operationResult{name: "refresh rotation", err: rotateErr}
	}()
	waitForAdvisoryLockWaiters(t, ctx, pool, results, 1, "refresh rotation insert barrier")

	userRepo := NewUserRepository(tx)
	go func() {
		disableErr := userRepo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID)
		results <- operationResult{name: "staff disable", err: disableErr}
	}()
	waitForAdvisoryLockWaiters(t, ctx, pool, results, 2, "staff disable waiting for refresh principal lock")

	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", barrierKey); err != nil {
		t.Fatalf("release refresh insert barrier: %v", err)
	}
	barrierHeld = false

	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("%s failed: %v", result.name, result.err)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for refresh/disable race: %v", ctx.Err())
		}
	}

	var userStatus, principalStatus string
	var tokenVersion int64
	var activeSessions int
	var replacementRevoked bool
	if err := tx.QueryRow(ctx, "SELECT u.status,ap.status,ap.token_version,(SELECT COUNT(*) FROM refresh_sessions rs WHERE rs.principal_id=ap.id AND rs.revoked_at IS NULL),(SELECT COALESCE(BOOL_AND(rs.revoked_at IS NOT NULL),FALSE) FROM refresh_sessions rs WHERE rs.token_hash=$2) FROM users u JOIN auth_principals ap ON ap.id=u.principal_id WHERE u.id=$1",
		targetUserID, newHash).Scan(&userStatus, &principalStatus, &tokenVersion, &activeSessions, &replacementRevoked); err != nil {
		t.Fatalf("verify final disabled staff and refresh-session state: %v", err)
	}
	if userStatus != "disabled" || principalStatus != "disabled" || tokenVersion != 2 {
		t.Fatalf("staff disable did not win final state: user=%q principal=%q token_version=%d", userStatus, principalStatus, tokenVersion)
	}
	if activeSessions != 0 || !replacementRevoked {
		t.Fatalf("refresh rotation left an active session after disable: active_sessions=%d replacement_revoked=%v", activeSessions, replacementRevoked)
	}
}


// TestStaffPasswordChangeSerializesWithDisablePostgresIntegration verifies
// that a password change already holding the principal lock commits before a
// concurrent School Admin disable, which then invalidates the updated account.
// The disable transaction and its audit event are rolled back with the fixture.
func TestStaffPasswordChangeSerializesWithDisablePostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var actorPrincipalID, schoolID int64
	if err := pool.QueryRow(ctx, "SELECT principal_id,school_id FROM users WHERE id=$1", actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	shortSuffix := suffix[len(suffix)-8:]
	const oldPasswordHash = "password-disable-race-old-hash"
	const newPasswordHash = "password-disable-race-new-hash"
	var targetUserID, targetPrincipalID int64
	if err := pool.QueryRow(ctx, "INSERT INTO users (school_id,username,password_hash,status) VALUES ($1,$2,$3,'active') RETURNING id,principal_id",
		schoolID, "password-race-"+shortSuffix, oldPasswordHash).Scan(&targetUserID, &targetPrincipalID); err != nil {
		t.Fatalf("create target staff principal: %v", err)
	}

	sessionHash := integrationTokenHash("staff-password-disable-race-" + suffix)
	if err := NewPrincipalRefreshSessionRepository(pool).CreatePrincipal(ctx, targetPrincipalID, sessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create target refresh session: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		cleanupTx, err := pool.Begin(cleanupCtx)
		if err != nil {
			t.Errorf("begin password/disable fixture cleanup: %v", err)
			return
		}
		defer func() { _ = cleanupTx.Rollback(cleanupCtx) }()
		// Password-change audit events are append-only and retain the target
		// principal as their actor. Keep that identity fixture so audit history
		// remains intact; only remove its refresh-session credentials.
		_, err = cleanupTx.Exec(cleanupCtx, "DELETE FROM refresh_sessions WHERE principal_id=$1", targetPrincipalID)
		if err == nil {
			err = cleanupTx.Commit(cleanupCtx)
		}
		if err != nil {
			t.Errorf("clean up password/disable fixtures: %v", err)
		}
	})

	// Give the fixture actor a School Admin role and verified MFA only inside
	// the outer transaction used by DisableStaffAccount.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped School Admin fixture: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, "SELECT ap.id FROM platform_admin_accounts pa JOIN auth_principals ap ON ap.id=pa.principal_id JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL JOIN roles r ON r.id=pr.role_id JOIN role_permissions rp ON rp.role_id=r.id JOIN permissions p ON p.id=rp.permission_id WHERE ap.principal_type='platform_admin' AND ap.status='active' AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE AND p.permission_key='platform.school_admins.create' AND p.scope_type='platform' AND p.is_active=TRUE LIMIT 1").Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM platform_admin_accounts) OR EXISTS(SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id WHERE r.principal_type='platform_admin' AND r.scope_type='platform' AND r.role_key='platform_admin')").Scan(&exists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if exists {
			t.Skip("existing Platform Admin is not eligible to assign the School Admin system role")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx, tx,
			fmt.Sprintf("staff-password-disable-race-%d@example.invalid", time.Now().UnixNano()), "integration-test-password-hash")
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO principal_mfa_credentials (principal_id,method,secret_ciphertext,encryption_key_version,verified_at) VALUES ($1,'totp',$2,1,NOW())",
		actorPrincipalID, []byte("password-disable-race-mfa-fixture")); err != nil {
		t.Fatalf("create School Admin MFA fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO principal_roles (principal_id,role_id,assigned_by) SELECT $1,r.id,$2 FROM roles r WHERE r.school_id=$3 AND r.principal_type='staff' AND r.scope_type='school' AND r.role_key='school_admin' AND r.is_system=TRUE",
		actorPrincipalID, platformAdminPrincipalID, schoolID); err != nil {
		t.Fatalf("assign School Admin role to race actor: %v", err)
	}

	functionName := "test_staff_password_disable_fn_" + shortSuffix
	triggerName := "test_staff_password_disable_" + shortSuffix
	const barrierKey int64 = 73482917364829181
	createFunction := fmt.Sprintf("CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS 'BEGIN IF NEW.password_hash = ''%s'' THEN PERFORM pg_advisory_xact_lock(%d); END IF; RETURN NEW; END;'",
		functionName, newPasswordHash, barrierKey)
	if _, err := pool.Exec(ctx, createFunction); err != nil {
		t.Fatalf("create password update barrier function: %v", err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE OF password_hash ON users FOR EACH ROW EXECUTE FUNCTION %s()", triggerName, functionName)); err != nil {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
		t.Fatalf("create password update barrier trigger: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON users", triggerName))
		_, _ = pool.Exec(cleanupCtx, fmt.Sprintf("DROP FUNCTION IF EXISTS %s()", functionName))
	})

	barrierConn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire password update barrier connection: %v", err)
	}
	defer barrierConn.Release()
	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_lock($1)", barrierKey); err != nil {
		t.Fatalf("hold password update barrier: %v", err)
	}
	barrierHeld := true
	defer func() {
		if barrierHeld {
			_, _ = barrierConn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", barrierKey)
		}
	}()

	results := make(chan operationResult, 2)
	go func() {
		changeErr := NewUserRepository(pool).ChangeStaffPassword(ctx, targetPrincipalID, oldPasswordHash, newPasswordHash)
		results <- operationResult{name: "password change", err: changeErr}
	}()
	waitForAdvisoryLockWaiters(t, ctx, pool, results, 1, "password change update barrier")

	go func() {
		disableErr := NewUserRepository(tx).DisableStaffAccount(ctx, actorPrincipalID, schoolID, targetPrincipalID)
		results <- operationResult{name: "staff disable", err: disableErr}
	}()
	waitForAdvisoryLockWaiters(t, ctx, pool, results, 2, "staff disable waiting for password change principal lock")

	if _, err := barrierConn.Exec(ctx, "SELECT pg_advisory_unlock($1)", barrierKey); err != nil {
		t.Fatalf("release password update barrier: %v", err)
	}
	barrierHeld = false

	for i := 0; i < 2; i++ {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("%s failed: %v", result.name, result.err)
			}
		case <-ctx.Done():
			t.Fatalf("timed out waiting for password-change/disable race: %v", ctx.Err())
		}
	}

	var userStatus, principalStatus, storedPasswordHash string
	var tokenVersion int64
	var mustChangePassword bool
	var activeSessions int
	if err := tx.QueryRow(ctx, "SELECT u.status,ap.status,ap.token_version,u.password_hash,u.must_change_password,(SELECT COUNT(*) FROM refresh_sessions rs WHERE rs.principal_id=ap.id AND rs.revoked_at IS NULL) FROM users u JOIN auth_principals ap ON ap.id=u.principal_id WHERE u.id=$1",
		targetUserID).Scan(&userStatus, &principalStatus, &tokenVersion, &storedPasswordHash, &mustChangePassword, &activeSessions); err != nil {
		t.Fatalf("verify final password/disable state: %v", err)
	}
	if userStatus != "disabled" || principalStatus != "disabled" || tokenVersion != 3 {
		t.Fatalf("expected disable after password change: user=%q principal=%q token_version=%d", userStatus, principalStatus, tokenVersion)
	}
	if storedPasswordHash != newPasswordHash || mustChangePassword {
		t.Fatalf("password change did not commit before disable: hash_changed=%v must_change_password=%v", storedPasswordHash == newPasswordHash, mustChangePassword)
	}
	if activeSessions != 0 {
		t.Fatalf("expected no active refresh sessions after password change and disable; got %d", activeSessions)
	}
}
