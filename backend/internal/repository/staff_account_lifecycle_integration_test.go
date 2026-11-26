package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
)

// TestAuthorizedStaffAccountLifecyclePostgresIntegration exercises successful
// School Admin staff lifecycle operations. All writes, including the temporary
// Platform Admin bootstrap (when needed), run inside a caller-owned transaction
// and are rolled back so the test never leaves a permanent administrator or
// staff account in school_management_test.
func TestAuthorizedStaffAccountLifecyclePostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped staff lifecycle test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id, school_id
		  FROM users
		 WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load staff lifecycle actor: %v", err)
	}

	// Reuse an already bootstrapped Platform Admin if the dedicated test DB has
	// one. Otherwise create the first administrator only inside this transaction.
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND r.is_system = TRUE
		   AND p.permission_key = 'platform.school_admins.create'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		var anyPlatformAdminExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1
			          FROM principal_roles pr
			          JOIN roles r ON r.id = pr.role_id
			         WHERE r.principal_type = 'platform_admin'
			           AND r.scope_type = 'platform'
			           AND r.role_key = 'platform_admin'
			    )
		`).Scan(&anyPlatformAdminExists); checkErr != nil {
			t.Fatalf("check existing Platform Admin bootstrap state: %v", checkErr)
		}
		if anyPlatformAdminExists {
			t.Skip("an existing Platform Admin is not active with the required onboarding permission")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("staff-lifecycle-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}

	// V13 provisions the School Admin system role when the integration fixture
	// creates its school. V14 independently requires a Platform Admin actor for
	// this reserved role assignment.
	if err := tx.QueryRow(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		SELECT $1, r.id, $2
		  FROM roles r
		 WHERE r.school_id = $3
		   AND r.principal_type = 'staff'
		   AND r.scope_type = 'school'
		   AND r.role_key = 'school_admin'
		   AND r.is_system = TRUE
		RETURNING id
	`, actorPrincipalID, platformAdminPrincipalID, schoolID).Scan(new(int64)); err != nil {
		t.Fatalf("assign temporary School Admin role to test actor: %v", err)
	}

	// pgx.Tx implements the same repository DB contract as pgxpool.Pool.
	// Repository Begin calls become nested savepoints, so successful method
	// commits remain inside this outer rollback boundary.
	repo := NewUserRepository(tx)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	username := "lifecycle-" + suffix
	passwordHash := "initial-integration-password-hash"

	created, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID, username, passwordHash)
	if err != nil {
		t.Fatalf("authorized staff account creation failed: %v", err)
	}
	if created == nil || created.PrincipalID <= 0 || created.SchoolID != schoolID ||
		created.Username != username || created.Status != "active" || !created.MustChangePassword {
		t.Fatalf("unexpected created staff account: %+v", created)
	}

    if _, err := tx.Exec(ctx, `
        INSERT INTO staff_profiles
            (principal_id, school_id, employee_code, full_name, designation, mobile_number, employment_status)
        VALUES ($1, $2, $3, 'Lifecycle Teacher', 'Teacher', '9876543210', 'active')
    `, created.PrincipalID, schoolID, "LIFE-"+suffix); err != nil {
        t.Fatalf("create active employee profile for lifecycle fixture: %v", err)
    }

	// Duplicate employee codes must fail with a stable domain error and roll
	// back the newly inserted user/principal as well as the profile row.
	duplicateEmployeeUsername := "dup-" + suffix[len(suffix)-12:]
	duplicateEmployeeProfile := models.StaffProfile{
		EmployeeCode: "LIFE-" + suffix,
		FullName: "Duplicate Employee Code",
		Designation: "Teacher",
		MobileNumber: "9876543211",
	}
	if _, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
		duplicateEmployeeUsername, "duplicate-employee-password-hash", duplicateEmployeeProfile); !errors.Is(err, ErrEmployeeCodeTaken) {
		t.Fatalf("duplicate employee code error = %v, want ErrEmployeeCodeTaken", err)
	}
	var duplicateEmployeeUsers, duplicateEmployeeProfiles int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE school_id = $1 AND username = $2
	`, schoolID, duplicateEmployeeUsername).Scan(&duplicateEmployeeUsers); err != nil {
		t.Fatalf("count user rows after duplicate employee code rejection: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM staff_profiles sp
		JOIN users u ON u.principal_id = sp.principal_id
		WHERE u.school_id = $1 AND u.username = $2
	`, schoolID, duplicateEmployeeUsername).Scan(&duplicateEmployeeProfiles); err != nil {
		t.Fatalf("count profile rows after duplicate employee code rejection: %v", err)
	}
	if duplicateEmployeeUsers != 0 || duplicateEmployeeProfiles != 0 {
		t.Fatalf("duplicate employee code left partial onboarding state: users=%d profiles=%d",
			duplicateEmployeeUsers, duplicateEmployeeProfiles)
	}

	// Force the audit write to fail and prove the user/principal insert rolls
	// back with it. This guards the all-or-nothing account provisioning contract.
	auditFailureSuffix := suffix[len(suffix)-8:]
	triggerName := "test_reject_staff_created_audit_" + auditFailureSuffix
	functionName := "test_reject_staff_created_audit_fn_" + auditFailureSuffix
	createFailureTrigger := fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger
		LANGUAGE plpgsql AS 'BEGIN
		    IF NEW.action = ''staff.account.created'' THEN
		        RAISE EXCEPTION ''forced staff audit failure'' USING ERRCODE = ''P0001'';
		    END IF;
		    RETURN NEW;
		END;';
		CREATE TRIGGER %s
		BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION %s();
	`, functionName, triggerName, functionName)
	if _, err := tx.Exec(ctx, createFailureTrigger); err != nil {
		t.Fatalf("install scoped staff audit failure trigger: %v", err)
	}
	auditFailureUsername := "audit-fail-" + auditFailureSuffix
	if _, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID,
		auditFailureUsername, "audit-failure-password-hash"); err == nil {
		t.Fatal("staff creation must fail when its required audit insert fails")
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events", triggerName)); err != nil {
		t.Fatalf("remove scoped staff audit failure trigger: %v", err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("DROP FUNCTION %s()", functionName)); err != nil {
		t.Fatalf("remove scoped staff audit failure function: %v", err)
	}
	var rolledBackUsers, failedAuditEvents int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM users WHERE school_id = $1 AND username = $2
	`, schoolID, auditFailureUsername).Scan(&rolledBackUsers); err != nil {
		t.Fatalf("verify failed staff creation left no user: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE actor_principal_id = $1 AND school_id = $2
		   AND action = 'staff.account.created'
		   AND metadata->>'username' = $3
	`, actorPrincipalID, schoolID, auditFailureUsername).Scan(&failedAuditEvents); err != nil {
		t.Fatalf("verify failed staff creation left no audit event: %v", err)
	}
	if rolledBackUsers != 0 || failedAuditEvents != 0 {
		t.Fatalf("failed staff creation was not atomic: users=%d audit_events=%d",
			rolledBackUsers, failedAuditEvents)
	}

	// The repository must translate the global case-insensitive unique
	// constraint into its stable domain error, not leak a raw PostgreSQL error.
	if _, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID, username,
		"duplicate-integration-password-hash"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate staff username error = %v, want ErrUsernameTaken", err)
	}

	listed, err := repo.ListStaffAccounts(ctx, actorPrincipalID, schoolID)
	if err != nil {
		t.Fatalf("authorized staff account listing failed: %v", err)
	}
	found := false
	for _, user := range listed {
		if user.PrincipalID == created.PrincipalID {
			found = true
			if user.Username != username || user.Status != "active" || !user.MustChangePassword {
				t.Fatalf("listed staff account differs from created account: %+v", user)
			}
		}
	}
	if !found {
		t.Fatalf("new staff account principal %d was absent from the same-school listing", created.PrincipalID)
	}

	// A caller cannot switch the legacy account listing to another school by
	// supplying a forged school ID, even when the actor is a valid School Admin.
	if foreignListing, err := repo.ListStaffAccounts(ctx, actorPrincipalID, schoolID+1); !errors.Is(err, ErrRolePermissionDenied) || foreignListing != nil {
		t.Fatalf("forged foreign-school account listing = %+v, err=%v; want nil listing and ErrRolePermissionDenied", foreignListing, err)
	}

    // Username changes must also roll back token invalidation and session
    // revocation when their required audit event cannot be written.
    usernameFailureSuffix := suffix[len(suffix)-8:]
    usernameTrigger := "test_reject_staff_updated_audit_" + usernameFailureSuffix
    usernameFunction := "test_reject_staff_updated_audit_fn_" + usernameFailureSuffix
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
        BEGIN
            IF NEW.action = 'staff.account.updated' THEN
                RAISE EXCEPTION 'forced staff username audit failure';
            END IF;
            RETURN NEW;
        END;
        $$;
        CREATE TRIGGER %s BEFORE INSERT ON audit_events
        FOR EACH ROW EXECUTE FUNCTION %s();
    `, usernameFunction, usernameTrigger, usernameFunction)); err != nil {
        t.Fatalf("install username update audit failure trigger: %v", err)
    }
    usernameSessionHash := integrationTokenHash("staff-username-audit-failure-" + suffix)
    if err := NewPrincipalRefreshSessionRepository(tx).CreatePrincipalInTx(ctx, tx, created.PrincipalID, usernameSessionHash, time.Now().Add(time.Hour)); err != nil {
        t.Fatalf("create refresh session before rejected username update: %v", err)
    }
    if _, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, "audit-fail-"+suffix); err == nil {
        t.Fatal("username update unexpectedly succeeded when its audit insert failed")
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", usernameTrigger, usernameFunction)); err != nil {
        t.Fatalf("remove username update audit failure trigger: %v", err)
    }
    var preservedUsername, preservedPrincipalStatus string
    var preservedTokenVersion int64
    var usernameSessionStillActive bool
    if err := tx.QueryRow(ctx, `
        SELECT u.username, ap.status, ap.token_version,
               EXISTS (SELECT 1 FROM refresh_sessions rs WHERE rs.token_hash = $2 AND rs.revoked_at IS NULL)
          FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
         WHERE u.principal_id = $1
    `, created.PrincipalID, usernameSessionHash).Scan(&preservedUsername, &preservedPrincipalStatus, &preservedTokenVersion, &usernameSessionStillActive); err != nil {
        t.Fatalf("read account state after rejected username update: %v", err)
    }
    if preservedUsername != username || preservedPrincipalStatus != "active" || preservedTokenVersion != 1 || !usernameSessionStillActive {
        t.Fatalf("failed username update left partial side effects: username=%q principal=%q token_version=%d session_active=%v",
            preservedUsername, preservedPrincipalStatus, preservedTokenVersion, usernameSessionStillActive)
    }
    var failedUsernameAuditCount int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
         WHERE actor_principal_id = $1 AND school_id = $2
           AND action = 'staff.account.updated' AND resource_id = $3
    `, actorPrincipalID, schoolID, fmt.Sprint(created.ID)).Scan(&failedUsernameAuditCount); err != nil {
        t.Fatalf("count username update audit events after injected failure: %v", err)
    }
    if failedUsernameAuditCount != 0 {
        t.Fatalf("failed username update created %d success audit events", failedUsernameAuditCount)
    }

	renamedUsername := "renamed-" + suffix
	updated, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, renamedUsername)
	if err != nil {
		t.Fatalf("authorized staff username update failed: %v", err)
	}
	if updated.Username != renamedUsername {
		t.Fatalf("updated username = %q, want %q", updated.Username, renamedUsername)
	}

	// Repository callers must never be able to replace a credential with an empty hash.
	if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID, created.PrincipalID, ""); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("empty staff password hash reset error = %v, want ErrInvalidUser", err)
	}
	if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID, actorPrincipalID, "self-reset-hash"); !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("self staff password reset error = %v, want ErrInvalidUser", err)
	}
	var unchangedResetHash string
	if err := tx.QueryRow(ctx, "SELECT password_hash FROM users WHERE principal_id = $1", created.PrincipalID).Scan(&unchangedResetHash); err != nil {
		t.Fatalf("read password hash after rejected reset attempts: %v", err)
	}
	if unchangedResetHash != "integration-test-password-hash" {
		t.Fatalf("rejected empty/self reset changed target password hash: %q", unchangedResetHash)
	}

	// A School Admin password reset must atomically replace the credential,
	// force a password change, invalidate access tokens, revoke refresh sessions,
	// and record the actor/target audit event.
	resetSessionHash := integrationTokenHash("staff-lifecycle-reset-" + suffix)
	if err := NewPrincipalRefreshSessionRepository(pool).CreatePrincipalInTx(ctx, tx, created.PrincipalID, resetSessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create target refresh session before password reset: %v", err)
	}
	const resetPasswordHash = "school-admin-reset-temporary-password-hash"
	if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID, created.PrincipalID, resetPasswordHash); err != nil {
		t.Fatalf("authorized same-school staff password reset failed: %v", err)
	}
	var actualResetPasswordHash string
	var resetMustChangePassword, resetSessionRevoked bool
	var resetTokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT u.password_hash, u.must_change_password, ap.token_version,
		       EXISTS (
		           SELECT 1 FROM refresh_sessions rs
		            WHERE rs.token_hash = $2 AND rs.revoked_at IS NOT NULL
	       )
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID, resetSessionHash).Scan(
		&actualResetPasswordHash, &resetMustChangePassword, &resetTokenVersion, &resetSessionRevoked,
	); err != nil {
		t.Fatalf("read staff password reset state: %v", err)
	}
	if actualResetPasswordHash != resetPasswordHash || !resetMustChangePassword ||
		resetTokenVersion != 3 || !resetSessionRevoked {
		t.Fatalf("password reset state mismatch: hash=%q must_change=%v token_version=%d session_revoked=%v",
			actualResetPasswordHash, resetMustChangePassword, resetTokenVersion, resetSessionRevoked)
	}
	var resetAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM audit_events
		 WHERE actor_principal_id = $1 AND school_id = $2
		   AND action = 'staff.password.reset' AND resource_type = 'staff_user'
		   AND resource_id = $3 AND outcome = 'success'
	`, actorPrincipalID, schoolID, fmt.Sprint(created.ID)).Scan(&resetAuditCount); err != nil {
		t.Fatalf("count staff password reset audit events: %v", err)
	}
	if resetAuditCount != 1 {
		t.Fatalf("expected exactly one staff password reset audit event, got %d", resetAuditCount)
	}

    // Password reset must roll back credential replacement, token-version
    // invalidation and refresh-session revocation if its audit write fails.
    resetFailureSuffix := suffix[len(suffix)-8:]
    resetFailureTrigger := "test_reject_staff_reset_audit_" + resetFailureSuffix
    resetFailureFunction := "test_reject_staff_reset_audit_fn_" + resetFailureSuffix
    resetFailureSession := integrationTokenHash("staff-reset-audit-failure-" + suffix)
    if err := NewPrincipalRefreshSessionRepository(pool).CreatePrincipalInTx(ctx, tx, created.PrincipalID, resetFailureSession, time.Now().Add(time.Hour)); err != nil {
        t.Fatalf("create refresh session before rejected password reset: %v", err)
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
        BEGIN
            IF NEW.action = 'staff.password.reset' THEN
                RAISE EXCEPTION 'forced staff password reset audit failure';
            END IF;
            RETURN NEW;
        END;
        $$;
        CREATE TRIGGER %s BEFORE INSERT ON audit_events
        FOR EACH ROW EXECUTE FUNCTION %s();
    `, resetFailureFunction, resetFailureTrigger, resetFailureFunction)); err != nil {
        t.Fatalf("install password reset audit failure trigger: %v", err)
    }
    if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID, created.PrincipalID, "must-not-be-committed-hash"); err == nil {
        t.Fatal("password reset unexpectedly succeeded when its audit insert failed")
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", resetFailureTrigger, resetFailureFunction)); err != nil {
        t.Fatalf("remove password reset audit failure trigger: %v", err)
    }
    var preservedPasswordHash string
    var preservedMustChange bool
    var preservedResetTokenVersion int64
    var resetSessionStillActive bool
    if err := tx.QueryRow(ctx, `
        SELECT u.password_hash, u.must_change_password, ap.token_version,
               EXISTS (SELECT 1 FROM refresh_sessions rs WHERE rs.token_hash = $2 AND rs.revoked_at IS NULL)
          FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
         WHERE u.principal_id = $1
    `, created.PrincipalID, resetFailureSession).Scan(&preservedPasswordHash, &preservedMustChange, &preservedResetTokenVersion, &resetSessionStillActive); err != nil {
        t.Fatalf("read state after rejected password reset: %v", err)
    }
    if preservedPasswordHash != resetPasswordHash || !preservedMustChange || preservedResetTokenVersion != 3 || !resetSessionStillActive {
        t.Fatalf("failed password reset left partial side effects: password_hash=%q must_change=%v token_version=%d session_active=%v",
            preservedPasswordHash, preservedMustChange, preservedResetTokenVersion, resetSessionStillActive)
    }

    // Disable must likewise preserve active access and sessions when its
    // mandatory audit insert fails.
    disableFailureSuffix := suffix[len(suffix)-8:]
    disableFailureTrigger := "test_reject_staff_disable_audit_" + disableFailureSuffix
    disableFailureFunction := "test_reject_staff_disable_audit_fn_" + disableFailureSuffix
    disableFailureSession := integrationTokenHash("staff-disable-audit-failure-" + suffix)
    if err := NewPrincipalRefreshSessionRepository(pool).CreatePrincipalInTx(ctx, tx, created.PrincipalID, disableFailureSession, time.Now().Add(time.Hour)); err != nil {
        t.Fatalf("create refresh session before rejected account disable: %v", err)
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
        BEGIN
            IF NEW.action = 'staff.account.disabled' THEN
                RAISE EXCEPTION 'forced staff disable audit failure';
            END IF;
            RETURN NEW;
        END;
        $$;
        CREATE TRIGGER %s BEFORE INSERT ON audit_events
        FOR EACH ROW EXECUTE FUNCTION %s();
    `, disableFailureFunction, disableFailureTrigger, disableFailureFunction)); err != nil {
        t.Fatalf("install account disable audit failure trigger: %v", err)
    }
    if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID); err == nil {
        t.Fatal("account disable unexpectedly succeeded when its audit insert failed")
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", disableFailureTrigger, disableFailureFunction)); err != nil {
        t.Fatalf("remove account disable audit failure trigger: %v", err)
    }
    var preservedAccountStatus string
    var preservedDisableTokenVersion int64
    var disableSessionStillActive bool
    if err := tx.QueryRow(ctx, `
        SELECT ap.status, ap.token_version,
               EXISTS (SELECT 1 FROM refresh_sessions rs WHERE rs.token_hash = $2 AND rs.revoked_at IS NULL)
          FROM auth_principals ap WHERE ap.id = $1
    `, created.PrincipalID, disableFailureSession).Scan(&preservedAccountStatus, &preservedDisableTokenVersion, &disableSessionStillActive); err != nil {
        t.Fatalf("read state after rejected account disable: %v", err)
    }
    if preservedAccountStatus != "active" || preservedDisableTokenVersion != 3 || !disableSessionStillActive {
        t.Fatalf("failed account disable left partial side effects: status=%q token_version=%d session_active=%v",
            preservedAccountStatus, preservedDisableTokenVersion, disableSessionStillActive)
    }

	// Seed a separate live refresh session in the same outer transaction before disable.
	// The repository must serialize with refresh rotation and revoke this session
	// atomically with principal deactivation.
	sessionHash := integrationTokenHash("staff-lifecycle-disable-" + suffix)
	sessions := NewPrincipalRefreshSessionRepository(pool)
	if err := sessions.CreatePrincipalInTx(ctx, tx, created.PrincipalID, sessionHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create target refresh session before disable: %v", err)
	}

	if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID); err != nil {
		t.Fatalf("authorized staff account disable failed: %v", err)
	}
	var sessionRevoked bool
	if err := tx.QueryRow(ctx, `
		SELECT revoked_at IS NOT NULL
		  FROM refresh_sessions
		 WHERE token_hash = $1
	`, sessionHash).Scan(&sessionRevoked); err != nil {
		t.Fatalf("read target refresh session after disable: %v", err)
	}
	if !sessionRevoked {
		t.Fatal("successful staff disable must revoke every active refresh session")
	}

	var userStatus, principalStatus string
	if err := tx.QueryRow(ctx, `
		SELECT u.status, ap.status
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(&userStatus, &principalStatus); err != nil {
		t.Fatalf("read disabled staff account state: %v", err)
	}
	if userStatus != "disabled" || principalStatus != "disabled" {
		t.Fatalf("disable status mismatch: users=%q principal=%q", userStatus, principalStatus)
	}

    // A disabled employee whose employment itself is inactive must not
    // regain login access merely because an administrator requests re-enable.
    if _, err := tx.Exec(ctx, `
        UPDATE staff_profiles SET employment_status = 'inactive'
         WHERE principal_id = $1 AND school_id = $2
    `, created.PrincipalID, schoolID); err != nil {
        t.Fatalf("mark disabled employee employment inactive: %v", err)
    }
    if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, "inactive-employee-must-not-enable"); !errors.Is(err, ErrStaffProfileInactive) {
        t.Fatalf("re-enable inactive employee error = %v, want ErrStaffProfileInactive", err)
    }
    var inactiveEnableStatus, inactiveEnablePrincipalStatus string
    if err := tx.QueryRow(ctx, `
        SELECT u.status, ap.status
          FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
         WHERE u.principal_id = $1
    `, created.PrincipalID).Scan(&inactiveEnableStatus, &inactiveEnablePrincipalStatus); err != nil {
        t.Fatalf("read account after inactive employee re-enable rejection: %v", err)
    }
    if inactiveEnableStatus != "disabled" || inactiveEnablePrincipalStatus != "disabled" {
        t.Fatalf("inactive employee was re-enabled: user=%q principal=%q", inactiveEnableStatus, inactiveEnablePrincipalStatus)
    }
    if _, err := tx.Exec(ctx, `
        UPDATE staff_profiles SET employment_status = 'active'
         WHERE principal_id = $1 AND school_id = $2
    `, created.PrincipalID, schoolID); err != nil {
        t.Fatalf("restore active employment before re-enable lifecycle checks: %v", err)
    }

    // A failed enable audit must not reactivate the principal or replace
    // the password. The subsequent successful enable below proves retryability.
    enableFailureSuffix := suffix[len(suffix)-8:]
    enableFailureTrigger := "test_reject_staff_enable_audit_" + enableFailureSuffix
    enableFailureFunction := "test_reject_staff_enable_audit_fn_" + enableFailureSuffix
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$
        BEGIN
            IF NEW.action = 'staff.account.enabled' THEN
                RAISE EXCEPTION 'forced staff enable audit failure';
            END IF;
            RETURN NEW;
        END;
        $$;
        CREATE TRIGGER %s BEFORE INSERT ON audit_events
        FOR EACH ROW EXECUTE FUNCTION %s();
    `, enableFailureFunction, enableFailureTrigger, enableFailureFunction)); err != nil {
        t.Fatalf("install account enable audit failure trigger: %v", err)
    }
    if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, "must-not-be-committed-enable-hash"); err == nil {
        t.Fatal("account enable unexpectedly succeeded when its audit insert failed")
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", enableFailureTrigger, enableFailureFunction)); err != nil {
        t.Fatalf("remove account enable audit failure trigger: %v", err)
    }
    var failedEnableUserStatus, failedEnablePrincipalStatus, failedEnablePasswordHash string
    var failedEnableTokenVersion int64
    if err := tx.QueryRow(ctx, `
        SELECT u.status, ap.status, u.password_hash, ap.token_version
          FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
         WHERE u.principal_id = $1
    `, created.PrincipalID).Scan(&failedEnableUserStatus, &failedEnablePrincipalStatus, &failedEnablePasswordHash, &failedEnableTokenVersion); err != nil {
        t.Fatalf("read state after rejected account enable: %v", err)
    }
    if failedEnableUserStatus != "disabled" || failedEnablePrincipalStatus != "disabled" ||
        failedEnablePasswordHash != resetPasswordHash || failedEnableTokenVersion != 4 {
        t.Fatalf("failed account enable left partial side effects: user=%q principal=%q password_hash=%q token_version=%d",
            failedEnableUserStatus, failedEnablePrincipalStatus, failedEnablePasswordHash, failedEnableTokenVersion)
    }


	// Re-enable is also strictly school-scoped; a different school must not
	// reactivate this disabled principal or replace its credential.
	if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID+1, created.PrincipalID,
		"cross-school-enable-takeover-hash"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("cross-school staff account re-enable error = %v, want ErrUserNotFound", err)
	}
	var crossSchoolEnableUserStatus, crossSchoolEnablePrincipalStatus, crossSchoolEnablePasswordHash string
	var crossSchoolEnableTokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT u.status, ap.status, u.password_hash, ap.token_version
		  FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(
		&crossSchoolEnableUserStatus, &crossSchoolEnablePrincipalStatus,
		&crossSchoolEnablePasswordHash, &crossSchoolEnableTokenVersion,
	); err != nil {
		t.Fatalf("read account after cross-school re-enable attempt: %v", err)
	}
	if crossSchoolEnableUserStatus != "disabled" || crossSchoolEnablePrincipalStatus != "disabled" ||
		crossSchoolEnablePasswordHash != resetPasswordHash || crossSchoolEnableTokenVersion != 4 {
		t.Fatalf("cross-school re-enable changed target state: user=%q principal=%q token_version=%d",
			crossSchoolEnableUserStatus, crossSchoolEnablePrincipalStatus, crossSchoolEnableTokenVersion)
	}

	const replacementPasswordHash = "replacement-temporary-password-hash"
	if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, created.PrincipalID, replacementPasswordHash); err != nil {
		t.Fatalf("authorized staff account re-enable failed: %v", err)
	}
	var finalUsername, finalUserStatus, finalPrincipalStatus, finalPasswordHash string
	var mustChangePassword bool
	var tokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT u.username, u.status, ap.status, u.password_hash,
		       u.must_change_password, ap.token_version
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(
		&finalUsername, &finalUserStatus, &finalPrincipalStatus, &finalPasswordHash,
		&mustChangePassword, &tokenVersion,
	); err != nil {
		t.Fatalf("read re-enabled staff account state: %v", err)
	}
	if finalUsername != renamedUsername || finalUserStatus != "active" ||
		finalPrincipalStatus != "active" || finalPasswordHash != replacementPasswordHash ||
		!mustChangePassword || tokenVersion != 5 {
		t.Fatalf("unexpected re-enabled staff state: username=%q user=%q principal=%q must_change=%v token_version=%d",
			finalUsername, finalUserStatus, finalPrincipalStatus, mustChangePassword, tokenVersion)
	}


	// Lifecycle operations must not cross school boundaries, even when the
	// caller is otherwise a fully authorized School Admin.
	if _, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID+1, created.PrincipalID,
		"cross-school-rename-"+suffix[len(suffix)-8:]); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("cross-school staff username update error = %v, want ErrUserNotFound", err)
	}
	if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID+1, created.PrincipalID,
		"cross-school-password-takeover-hash"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("cross-school staff password reset error = %v, want ErrUserNotFound", err)
	}
	if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID+1, created.PrincipalID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("cross-school staff account disable error = %v, want ErrUserNotFound", err)
	}
	var isolatedUsername, isolatedUserStatus, isolatedPrincipalStatus, isolatedPasswordHash string
	var isolatedTokenVersion int64
	if err := tx.QueryRow(ctx, `
		SELECT u.username, u.status, ap.status, u.password_hash, ap.token_version
		  FROM users u JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.principal_id = $1
	`, created.PrincipalID).Scan(
		&isolatedUsername, &isolatedUserStatus, &isolatedPrincipalStatus, &isolatedPasswordHash, &isolatedTokenVersion,
	); err != nil {
		t.Fatalf("read staff state after cross-school lifecycle attempts: %v", err)
	}
	if isolatedUsername != renamedUsername || isolatedUserStatus != "active" || isolatedPrincipalStatus != "active" ||
		isolatedPasswordHash != replacementPasswordHash || isolatedTokenVersion != 5 {
		t.Fatalf("cross-school lifecycle attempt changed target state: username=%q user=%q principal=%q token_version=%d",
			isolatedUsername, isolatedUserStatus, isolatedPrincipalStatus, isolatedTokenVersion)
	}

	var auditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM audit_events
		 WHERE actor_principal_id = $1
		   AND school_id = $2
		   AND resource_type = 'staff_user'
		   AND resource_id = $3
		   AND action IN (
		       'staff.account.created',
		       'staff.account.updated',
		       'staff.account.disabled',
		       'staff.account.enabled'
		   )
		   AND outcome = 'success'
	`, actorPrincipalID, schoolID, fmt.Sprint(created.ID)).Scan(&auditCount); err != nil {
		t.Fatalf("count staff lifecycle audit events: %v", err)
	}
	if auditCount != 4 {
		t.Fatalf("expected 4 staff lifecycle audit events, got %d", auditCount)
	}

	// An active School Admin must not be disableable by another School Admin.
	// Keep the generated username comfortably below the database's 32-char limit.
	protectedAdminUsername := "padmin-" + suffix[len(suffix)-8:]
	var protectedAdminUserID, protectedAdminPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id
	`, schoolID, protectedAdminUsername, "protected-admin-password-hash").Scan(
		&protectedAdminUserID, &protectedAdminPrincipalID,
	); err != nil {
		t.Fatalf("create second School Admin target: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
		    (principal_id, method, secret_ciphertext, encryption_key_version, verified_at)
		VALUES ($1, 'totp', $2, 1, NOW())
	`, protectedAdminPrincipalID, []byte("protected-school-admin-test-ciphertext")); err != nil {
		t.Fatalf("create verified MFA fixture for protected School Admin: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, (
		    SELECT id FROM roles
		     WHERE school_id = $2 AND principal_type = 'staff'
		       AND scope_type = 'school' AND role_key = 'school_admin' AND is_system = TRUE
		), $3)
	`, protectedAdminPrincipalID, schoolID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign protected School Admin role: %v", err)
	}
	// School Admin lifecycle protections are consistent: another School Admin
	// cannot be disabled or have their password reset by a peer administrator.
	protectedAdminSessionHash := integrationTokenHash("protected-admin-reset-" + suffix)
	if err := NewPrincipalRefreshSessionRepository(pool).CreatePrincipalInTx(
		ctx, tx, protectedAdminPrincipalID, protectedAdminSessionHash, time.Now().Add(time.Hour),
	); err != nil {
		t.Fatalf("create protected School Admin refresh session: %v", err)
	}
	if _, err := repo.UpdateStaffAccount(ctx, actorPrincipalID, schoolID, protectedAdminPrincipalID,
		"renamed-peer-admin-"+suffix[len(suffix)-8:]); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("renaming another active School Admin error = %v, want ErrUserNotFound", err)
	}
	if err := repo.ResetStaffPassword(ctx, actorPrincipalID, schoolID, protectedAdminPrincipalID,
		"attempted-admin-takeover-password-hash"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("resetting another active School Admin password error = %v, want ErrUserNotFound", err)
	}
	if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, protectedAdminPrincipalID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("disabling another active School Admin error = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, protectedAdminPrincipalID, models.StaffProfile{}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("updating another active School Admin profile error = %v, want ErrUserNotFound", err)
	}
	var protectedProfileCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM staff_profiles WHERE principal_id = $1 AND school_id = $2`,
		protectedAdminPrincipalID, schoolID).Scan(&protectedProfileCount); err != nil {
		t.Fatalf("count protected School Admin profiles after rejected update: %v", err)
	}
	if protectedProfileCount != 0 {
		t.Fatalf("rejected peer-admin profile update created %d staff profiles", protectedProfileCount)
	}
	var protectedUserStatus, protectedPrincipalStatus, protectedPasswordHash string
	var protectedTokenVersion int64
	var protectedMustChangePassword, protectedSessionActive bool
	if err := tx.QueryRow(ctx, `
		SELECT u.status, ap.status, ap.token_version, u.password_hash, u.must_change_password,
		       EXISTS (SELECT 1 FROM refresh_sessions rs
		                WHERE rs.token_hash = $3 AND rs.revoked_at IS NULL)
		  FROM users u
		  JOIN auth_principals ap ON ap.id = u.principal_id
		 WHERE u.id = $1 AND ap.id = $2
	`, protectedAdminUserID, protectedAdminPrincipalID, protectedAdminSessionHash).Scan(
		&protectedUserStatus, &protectedPrincipalStatus, &protectedTokenVersion,
		&protectedPasswordHash, &protectedMustChangePassword, &protectedSessionActive,
	); err != nil {
		t.Fatalf("verify protected School Admin state: %v", err)
	}
	if protectedUserStatus != "active" || protectedPrincipalStatus != "active" ||
		protectedTokenVersion != 1 || protectedPasswordHash != "protected-admin-password-hash" ||
		!protectedMustChangePassword || !protectedSessionActive {
		t.Fatalf("rejected School Admin lifecycle operation changed target state: user=%q principal=%q token_version=%d password_changed=%v must_change=%v session_active=%v",
			protectedUserStatus, protectedPrincipalStatus, protectedTokenVersion,
			protectedPasswordHash != "protected-admin-password-hash", protectedMustChangePassword, protectedSessionActive)
	}
	var protectedAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE actor_principal_id = $1 AND school_id = $2
		   AND action IN ('staff.account.disabled', 'staff.password.reset')
		   AND resource_type = 'staff_user'
		   AND resource_id = $3 AND outcome = 'success'
	`, actorPrincipalID, schoolID, fmt.Sprint(protectedAdminUserID)).Scan(&protectedAuditCount); err != nil {
		t.Fatalf("check rejected School Admin disable audit side effects: %v", err)
	}
	if protectedAuditCount != 0 {
		t.Fatalf("rejected School Admin disable must not create a success audit event; got %d", protectedAuditCount)
	}

	// Staff account lifecycle permissions are intentionally non-delegable.
	// Verify that the platform cannot create user-override ceilings for them;
	// the DENY checks in the repository remain defense-in-depth, not a way to
	// bypass the platform's delegation policy.
	roleRepo := NewRoleRepository(tx)
	for _, permissionKey := range []string{"users.create", "users.update", "users.disable", "users.password_reset"} {
		if _, err := roleRepo.GrantSchoolPermissionDelegation(
			ctx, platformAdminPrincipalID, schoolID, permissionKey, false, true,
		); !errors.Is(err, ErrRoleNotFound) {
			t.Fatalf("delegation for non-delegable %q error = %v, want ErrRoleNotFound", permissionKey, err)
		}
	}
}
