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

// TestStaffProfileRepositoryPostgresIntegration covers atomic profile creation,
// legacy-compatible listing, profile lifecycle, explicit login re-enable and
// employee-code uniqueness against PostgreSQL.
func TestStaffProfileRepositoryPostgresIntegration(t *testing.T) {
    _, pool, actorUserID := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    tx, err := pool.Begin(ctx)
    if err != nil { t.Fatalf("begin rollback-scoped staff profile test: %v", err) }
    defer func() { _ = tx.Rollback(ctx) }()

    var actorPrincipalID, schoolID int64
    if err := tx.QueryRow(ctx, `
        SELECT principal_id, school_id FROM users WHERE id = $1
    `, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
        t.Fatalf("load School Admin fixture identity: %v", err)
    }

    var platformAdminPrincipalID int64
    err = tx.QueryRow(ctx, `
        SELECT ap.id
          FROM platform_admin_accounts pa
          JOIN auth_principals ap ON ap.id = pa.principal_id
          JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
          JOIN roles r ON r.id = pr.role_id
          JOIN role_permissions rp ON rp.role_id = r.id
          JOIN permissions p ON p.id = rp.permission_id
         WHERE ap.principal_type = 'platform_admin' AND ap.status = 'active'
           AND r.role_key = 'platform_admin' AND r.scope_type = 'platform' AND r.is_system = TRUE
           AND p.permission_key = 'platform.permissions.delegate' AND p.scope_type = 'platform' AND p.is_active = TRUE
         LIMIT 1
    `).Scan(&platformAdminPrincipalID)
    if errors.Is(err, pgx.ErrNoRows) {
        var anyBootstrapExists bool
        if checkErr := tx.QueryRow(ctx, `
            SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
                OR EXISTS (SELECT 1 FROM principal_roles pr JOIN roles r ON r.id = pr.role_id
                           WHERE r.principal_type = 'platform_admin' AND r.scope_type = 'platform' AND r.role_key = 'platform_admin')
        `).Scan(&anyBootstrapExists); checkErr != nil {
            t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
        }
        if anyBootstrapExists { t.Skip("Platform Admin bootstrap exists but no active administrator can delegate permissions") }
        platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
            ctx, tx, fmt.Sprintf("staff-profile-%d@example.invalid", time.Now().UnixNano()), "integration-test-password-hash")
    }
    if err != nil { t.Fatalf("resolve Platform Admin fixture: %v", err) }

    var schoolAdminRoleID int64
    if err := tx.QueryRow(ctx, `
        SELECT id FROM roles
         WHERE school_id = $1 AND principal_type = 'staff' AND scope_type = 'school'
           AND role_key = 'school_admin' AND is_system = TRUE
    `, schoolID).Scan(&schoolAdminRoleID); err != nil {
        t.Fatalf("load School Admin role: %v", err)
    }
    if _, err := tx.Exec(ctx, `
        INSERT INTO principal_roles (principal_id, role_id, assigned_by)
        VALUES ($1, $2, $3)
    `, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
        t.Fatalf("assign School Admin fixture role: %v", err)
    }

    repo := NewUserRepository(tx)
    suffix := fmt.Sprintf("%d", time.Now().UnixNano())
    profile := models.StaffProfile{
        StaffType: "teaching",
        EmployeeCode: "EMP-" + suffix,
        FullName: "Integration Teacher",
        Designation: "Mathematics Teacher",
        MobileNumber: "9876543210",
        Email: "integration@example.invalid",
        EmploymentStatus: "active",
    }
    user, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
        "staffprof-"+suffix, "integration-test-password-hash", profile)
    if err != nil { t.Fatalf("create account and employee profile atomically: %v", err) }
    if user == nil || !user.ProfileComplete || user.StaffType != profile.StaffType || user.EmployeeCode != profile.EmployeeCode || user.FullName != profile.FullName {
        t.Fatalf("created staff profile not returned: %+v", user)
    }
    var teacherRoleAssignments int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*)
          FROM principal_roles pr
          JOIN roles r ON r.id = pr.role_id
         WHERE pr.principal_id = $1 AND pr.revoked_at IS NULL
           AND r.school_id = $2 AND r.role_key = 'teacher'
           AND r.principal_type = 'staff' AND r.scope_type = 'school'
           AND r.is_system = TRUE
    `, user.PrincipalID, schoolID).Scan(&teacherRoleAssignments); err != nil {
        t.Fatalf("count default Teacher role assignments: %v", err)
    }
    if teacherRoleAssignments != 1 {
        t.Fatalf("teaching staff onboarding created %d active Teacher role assignments, want 1", teacherRoleAssignments)
    }
    var teacherRoleAuditCount int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
         WHERE actor_principal_id = $1 AND school_id = $2
           AND action = 'school.staff.role_assigned'
           AND resource_id = $3
           AND metadata->>'role_key' = 'teacher'
           AND metadata->>'assignment_source' = 'staff_onboarding'
    `, actorPrincipalID, schoolID, fmt.Sprint(user.PrincipalID)).Scan(&teacherRoleAuditCount); err != nil {
        t.Fatalf("count default Teacher role audit events: %v", err)
    }
    if teacherRoleAuditCount != 1 {
        t.Fatalf("teaching staff onboarding created %d Teacher role audit events, want 1", teacherRoleAuditCount)
    }

    // The profile update must not cross a tenant boundary, even when the
    // caller knows the target principal identifier.
    crossSchoolProfile := profile
    crossSchoolProfile.FullName = "Cross-School Mutation"
    if _, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID+1, user.PrincipalID, crossSchoolProfile); !errors.Is(err, ErrUserNotFound) {
        t.Fatalf("cross-school profile update error = %v, want ErrUserNotFound", err)
    }
    var nameAfterCrossSchoolAttempt string
    if err := tx.QueryRow(ctx, `
        SELECT full_name FROM staff_profiles WHERE principal_id = $1
    `, user.PrincipalID).Scan(&nameAfterCrossSchoolAttempt); err != nil {
        t.Fatalf("read profile after cross-school update attempt: %v", err)
    }
    if nameAfterCrossSchoolAttempt != profile.FullName {
        t.Fatalf("cross-school profile update changed employee data: name=%q", nameAfterCrossSchoolAttempt)
    }

    // A School Admin cannot create a staff account by supplying another
    // school's ID, even when the target school exists and the caller has a
    // valid users.create permission in their own school.
    forgedSchoolUsername := "staff-forged-school-" + suffix
    forgedSchoolProfile := profile
    forgedSchoolProfile.EmployeeCode = "EMP-FORGED-" + suffix
    if _, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID+1,
        forgedSchoolUsername, "integration-test-password-hash", forgedSchoolProfile); !errors.Is(err, ErrRolePermissionDenied) {
        t.Fatalf("cross-school staff account/profile creation error = %v, want ErrRolePermissionDenied", err)
    }
    var forgedSchoolUserCount int
    if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE username=$1", forgedSchoolUsername).Scan(&forgedSchoolUserCount); err != nil {
        t.Fatalf("count account after forged-school creation attempt: %v", err)
    }
    if forgedSchoolUserCount != 0 {
        t.Fatalf("forged-school account creation left %d login accounts", forgedSchoolUserCount)
    }

    // Optional email may be omitted during initial onboarding as well as cleared later.
    blankEmailProfile := profile
    blankEmailProfile.StaffType = "non_teaching"
    blankEmailProfile.EmployeeCode = "EMP-BLANK-" + suffix
    blankEmailProfile.Email = ""
    blankEmailUser, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
        "staffblank-"+suffix, "integration-test-password-hash", blankEmailProfile)
    if err != nil {
        t.Fatalf("create employee profile without optional email: %v", err)
    }
    var blankEmailIsNull bool
    if err := tx.QueryRow(ctx, "SELECT email IS NULL FROM staff_profiles WHERE principal_id = $1",
        blankEmailUser.PrincipalID).Scan(&blankEmailIsNull); err != nil {
        t.Fatalf("verify omitted optional email: %v", err)
    }
    if !blankEmailIsNull || blankEmailUser.Email != "" {
        t.Fatalf("omitted optional email was not represented as empty in API and NULL in storage: is_null=%v email=%q", blankEmailIsNull, blankEmailUser.Email)
    }
    var nonTeachingTeacherRoles int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*)
          FROM principal_roles pr
          JOIN roles r ON r.id = pr.role_id
         WHERE pr.principal_id = $1 AND pr.revoked_at IS NULL
           AND r.school_id = $2 AND r.role_key = 'teacher'
           AND r.principal_type = 'staff' AND r.scope_type = 'school'
           AND r.is_system = TRUE
    `, blankEmailUser.PrincipalID, schoolID).Scan(&nonTeachingTeacherRoles); err != nil {
        t.Fatalf("count Teacher roles for non-teaching employee: %v", err)
    }
    if nonTeachingTeacherRoles != 0 {
        t.Fatalf("non-teaching employee received %d active Teacher roles, want 0", nonTeachingTeacherRoles)
    }

    // Account + employee-profile onboarding is one transaction: an audit
    // failure must not leave either the login account or the profile behind.
    createFailureUsername := "profile-audit-fail-" + suffix
    createFailureEmployeeCode := "EMP-AUDIT-FAIL-" + suffix
    if _, err := tx.Exec(ctx, `
        CREATE OR REPLACE FUNCTION public.test_staff_profile_create_audit_failure()
        RETURNS trigger LANGUAGE plpgsql AS $body$
        BEGIN
            IF NEW.action = 'staff.account.created' THEN
                RAISE EXCEPTION 'forced staff profile creation audit failure';
            END IF;
            RETURN NEW;
        END;
        $body$;
        CREATE TRIGGER trg_test_staff_profile_create_audit_failure
        BEFORE INSERT ON public.audit_events
        FOR EACH ROW EXECUTE FUNCTION public.test_staff_profile_create_audit_failure();
    `); err != nil {
        t.Fatalf("install profile creation audit failure trigger: %v", err)
    }
    failedCreateProfile := profile
    failedCreateProfile.EmployeeCode = createFailureEmployeeCode
    if _, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
        createFailureUsername, "profile-audit-failure-password-hash", failedCreateProfile); err == nil {
        t.Fatal("staff account/profile creation unexpectedly succeeded when its audit insert failed")
    }
    if _, err := tx.Exec(ctx, `
        DROP TRIGGER trg_test_staff_profile_create_audit_failure ON public.audit_events;
        DROP FUNCTION public.test_staff_profile_create_audit_failure();
    `); err != nil {
        t.Fatalf("remove profile creation audit failure trigger: %v", err)
    }
    var failedCreateUsers, failedCreateProfiles, failedCreateAudits int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM users WHERE school_id=$1 AND username=$2
    `, schoolID, createFailureUsername).Scan(&failedCreateUsers); err != nil {
        t.Fatalf("count account after profile creation audit failure: %v", err)
    }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM staff_profiles WHERE school_id=$1 AND employee_code=$2
    `, schoolID, createFailureEmployeeCode).Scan(&failedCreateProfiles); err != nil {
        t.Fatalf("count profile after profile creation audit failure: %v", err)
    }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
         WHERE actor_principal_id=$1 AND school_id=$2
           AND action='staff.account.created' AND metadata->>'username'=$3
    `, actorPrincipalID, schoolID, createFailureUsername).Scan(&failedCreateAudits); err != nil {
        t.Fatalf("count audit after profile creation audit failure: %v", err)
    }
    var failedCreateRoleAssignments int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*)
          FROM principal_roles pr
          JOIN users u ON u.principal_id = pr.principal_id
         WHERE u.school_id = $1 AND u.username = $2 AND pr.revoked_at IS NULL
    `, schoolID, createFailureUsername).Scan(&failedCreateRoleAssignments); err != nil {
        t.Fatalf("count role assignments after failed profile creation: %v", err)
    }
    if failedCreateUsers != 0 || failedCreateProfiles != 0 || failedCreateAudits != 0 || failedCreateRoleAssignments != 0 {
        t.Fatalf("profile creation audit failure left partial state: users=%d profiles=%d audits=%d role_assignments=%d",
            failedCreateUsers, failedCreateProfiles, failedCreateAudits, failedCreateRoleAssignments)
    }

    // A failed audit write for the automatically assigned Teacher role must
    // roll back the new login, employee profile, role assignment and audit row.
    roleAuditFailureUsername := "teacher-role-audit-fail-" + suffix
    roleAuditFailureCode := "EMP-ROLE-AUDIT-FAIL-" + suffix
    if _, err := tx.Exec(ctx, `
        CREATE OR REPLACE FUNCTION public.test_staff_teacher_role_audit_failure()
        RETURNS trigger LANGUAGE plpgsql AS $body$
        BEGIN
            IF NEW.action = 'school.staff.role_assigned'
               AND NEW.metadata->>'assignment_source' = 'staff_onboarding' THEN
                RAISE EXCEPTION 'forced default Teacher role audit failure';
            END IF;
            RETURN NEW;
        END;
        $body$;
        CREATE TRIGGER trg_test_staff_teacher_role_audit_failure
        BEFORE INSERT ON public.audit_events
        FOR EACH ROW EXECUTE FUNCTION public.test_staff_teacher_role_audit_failure();
    `); err != nil {
        t.Fatalf("install Teacher role audit failure trigger: %v", err)
    }
    roleAuditFailureProfile := profile
    roleAuditFailureProfile.EmployeeCode = roleAuditFailureCode
    if _, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
        roleAuditFailureUsername, "teacher-role-audit-failure-password-hash", roleAuditFailureProfile); err == nil {
        t.Fatal("teaching staff creation unexpectedly succeeded when default role audit failed")
    }
    if _, err := tx.Exec(ctx, `
        DROP TRIGGER trg_test_staff_teacher_role_audit_failure ON public.audit_events;
        DROP FUNCTION public.test_staff_teacher_role_audit_failure();
    `); err != nil {
        t.Fatalf("remove Teacher role audit failure trigger: %v", err)
    }
    var roleAuditFailureUsers, roleAuditFailureProfiles, roleAuditFailureAssignments, roleAuditFailureEvents int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM users WHERE school_id = $1 AND username = $2
    `, schoolID, roleAuditFailureUsername).Scan(&roleAuditFailureUsers); err != nil {
        t.Fatalf("count login after Teacher role audit failure: %v", err)
    }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM staff_profiles WHERE school_id = $1 AND employee_code = $2
    `, schoolID, roleAuditFailureCode).Scan(&roleAuditFailureProfiles); err != nil {
        t.Fatalf("count profile after Teacher role audit failure: %v", err)
    }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM principal_roles pr
          JOIN users u ON u.principal_id = pr.principal_id
         WHERE u.school_id = $1 AND u.username = $2 AND pr.revoked_at IS NULL
    `, schoolID, roleAuditFailureUsername).Scan(&roleAuditFailureAssignments); err != nil {
        t.Fatalf("count role assignment after Teacher role audit failure: %v", err)
    }
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
         WHERE actor_principal_id = $1 AND school_id = $2
           AND action = 'school.staff.role_assigned'
           AND metadata->>'assignment_source' = 'staff_onboarding'
    `, actorPrincipalID, schoolID).Scan(&roleAuditFailureEvents); err != nil {
        t.Fatalf("count Teacher role audit events after forced failure: %v", err)
    }
    if roleAuditFailureUsers != 0 || roleAuditFailureProfiles != 0 ||
        roleAuditFailureAssignments != 0 || roleAuditFailureEvents != 1 {
        t.Fatalf("Teacher role audit failure left partial state: users=%d profiles=%d assignments=%d onboarding_role_audits=%d (want 0,0,0,1 existing)",
            roleAuditFailureUsers, roleAuditFailureProfiles, roleAuditFailureAssignments, roleAuditFailureEvents)
    }

    // Updating an existing profile to another employee's code must be
    // rejected without partially changing the profile or recording success.
    duplicateCodeUpdate := profile
    duplicateCodeUpdate.EmployeeCode = blankEmailProfile.EmployeeCode
    duplicateCodeUpdate.FullName = "Must Not Replace Existing Employee"
    if _, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, duplicateCodeUpdate); !errors.Is(err, ErrEmployeeCodeTaken) {
        t.Fatalf("duplicate employee code profile update error = %v, want ErrEmployeeCodeTaken", err)
    }
    var preservedCode, preservedName string
    if err := tx.QueryRow(ctx, `
        SELECT employee_code, full_name FROM staff_profiles WHERE principal_id = $1
    `, user.PrincipalID).Scan(&preservedCode, &preservedName); err != nil {
        t.Fatalf("read profile after duplicate employee code update: %v", err)
    }
    if preservedCode != profile.EmployeeCode || preservedName != profile.FullName {
        t.Fatalf("duplicate employee code update partially changed profile: code=%q name=%q", preservedCode, preservedName)
    }
    var duplicateUpdateAuditCount int
    if err := tx.QueryRow(ctx, `
        SELECT COUNT(*) FROM audit_events
         WHERE actor_principal_id = $1 AND school_id = $2
           AND action = 'staff.profile.updated' AND resource_id = $3
    `, actorPrincipalID, schoolID, fmt.Sprint(user.ID)).Scan(&duplicateUpdateAuditCount); err != nil {
        t.Fatalf("count audit events after duplicate employee code update: %v", err)
    }
    if duplicateUpdateAuditCount != 0 {
        t.Fatalf("rejected duplicate employee code update created %d success audit events", duplicateUpdateAuditCount)
    }

    listed, err := repo.ListStaffAccounts(ctx, actorPrincipalID, schoolID)
    if err != nil { t.Fatalf("list school staff profiles: %v", err) }
    found := false
    for _, item := range listed {
        if item.PrincipalID == user.PrincipalID {
            found = item.ProfileComplete && item.EmployeeCode == profile.EmployeeCode && item.MobileNumber == profile.MobileNumber
        }
    }
    if !found { t.Fatal("created employee profile missing from staff listing") }

    duplicate := profile
    duplicate.FullName = "Duplicate Employee Code"
    if _, err := repo.CreateStaffAccountWithProfile(ctx, actorPrincipalID, schoolID,
        "staffdup-"+suffix, "integration-test-password-hash", duplicate); !errors.Is(err, ErrEmployeeCodeTaken) {
        t.Fatalf("duplicate employee code error = %v, want ErrEmployeeCodeTaken", err)
    }
    var duplicateCount int
    if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE username = $1`, "staffdup-"+suffix).Scan(&duplicateCount); err != nil {
        t.Fatalf("verify failed profile creation rollback: %v", err)
    }
    if duplicateCount != 0 { t.Fatalf("failed profile creation left %d orphan login accounts", duplicateCount) }

    profileSessionHash := integrationTokenHash("staff-profile-inactive-" + suffix)
    sessions := NewPrincipalRefreshSessionRepository(tx)
    if err := sessions.CreatePrincipalInTx(ctx, tx, user.PrincipalID, profileSessionHash, time.Now().Add(time.Hour)); err != nil {
        t.Fatalf("create refresh session before profile deactivation: %v", err)
    }

    profile.EmploymentStatus = "inactive"
    updated, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, profile)
    if err != nil { t.Fatalf("mark employee profile inactive: %v", err) }
    if updated.Status != "disabled" || updated.EmploymentStatus != "inactive" {
        t.Fatalf("inactive employee did not disable login: %+v", updated)
    }
    var profileSessionRevoked bool
    if err := tx.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM refresh_sessions WHERE token_hash = $1`, profileSessionHash).Scan(&profileSessionRevoked); err != nil {
        t.Fatalf("read refresh session after employee inactivation: %v", err)
    }
    if !profileSessionRevoked { t.Fatal("inactivating employee profile did not revoke active refresh sessions") }
    if _, err := repo.GetStaffPrincipalByUsername(ctx, user.Username); !errors.Is(err, ErrStaffInactive) {
        t.Fatalf("inactive employee authentication lookup error = %v, want ErrStaffInactive", err)
    }
    if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, user.PrincipalID, "must-not-enable-hash"); !errors.Is(err, ErrStaffProfileInactive) {
        t.Fatalf("inactive employee login enable error = %v, want ErrStaffProfileInactive", err)
    }

    profile.EmploymentStatus = "active"
    updated, err = repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, profile)
    if err != nil { t.Fatalf("reactivate employee profile: %v", err) }
    if updated.Status != "disabled" || updated.EmploymentStatus != "active" {
        t.Fatalf("reactivating employment unexpectedly enabled login: %+v", updated)
    }
    if _, err := repo.GetStaffPrincipalByUsername(ctx, user.Username); !errors.Is(err, ErrStaffInactive) {
        t.Fatalf("reactivated employee authentication lookup error = %v, want ErrStaffInactive until explicit login enable", err)
    }

    // An audit failure must roll back the employee profile mutation.
    failFunction := "test_fail_staff_profile_audit_" + suffix
    failTrigger := "test_fail_staff_profile_audit_trigger_" + suffix
    if _, err := tx.Exec(ctx, fmt.Sprintf(`
        CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $staff_profile_audit_failure$
        BEGIN
            IF NEW.action = 'staff.profile.updated' THEN
                RAISE EXCEPTION 'injected staff profile audit failure';
            END IF;
            RETURN NEW;
        END;
        $staff_profile_audit_failure$;
        CREATE TRIGGER %s BEFORE INSERT ON audit_events
        FOR EACH ROW EXECUTE FUNCTION %s();
    `, failFunction, failTrigger, failFunction)); err != nil {
        t.Fatalf("install staff profile audit failure trigger: %v", err)
    }
    failedUpdate := profile
    failedUpdate.FullName = "Must Roll Back"
    if _, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, failedUpdate); err == nil {
        t.Fatal("profile update unexpectedly succeeded when its audit insert failed")
    }
    var nameAfterAuditFailure string
    if err := tx.QueryRow(ctx, `
        SELECT full_name FROM staff_profiles WHERE principal_id = $1
    `, user.PrincipalID).Scan(&nameAfterAuditFailure); err != nil {
        t.Fatalf("read profile after injected audit failure: %v", err)
    }
    if nameAfterAuditFailure != profile.FullName {
        t.Fatalf("profile update was not rolled back with audit failure: name=%q want=%q", nameAfterAuditFailure, profile.FullName)
    }
    if _, err := tx.Exec(ctx, fmt.Sprintf("DROP TRIGGER %s ON audit_events; DROP FUNCTION %s();", failTrigger, failFunction)); err != nil {
        t.Fatalf("remove staff profile audit failure trigger: %v", err)
    }

    // Email is optional: clearing it in the profile editor must persist SQL NULL,
    // not an empty string that violates the staff_profiles email constraint.
    profile.Email = ""
    updated, err = repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, profile)
    if err != nil {
        t.Fatalf("clear optional employee email: %v", err)
    }
    var emailIsNull bool
    if err := tx.QueryRow(ctx, `
        SELECT email IS NULL FROM staff_profiles WHERE principal_id = $1
    `, user.PrincipalID).Scan(&emailIsNull); err != nil {
        t.Fatalf("verify cleared optional employee email: %v", err)
    }
    if !emailIsNull || updated.Email != "" {
        t.Fatalf("cleared optional email was not represented as empty in API and NULL in storage: is_null=%v email=%q", emailIsNull, updated.Email)
    }

    if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, user.PrincipalID, "integration-temporary-password-hash"); err != nil {
        t.Fatalf("explicitly enable active employee login: %v", err)
    }
    var finalStatus string
    var mustChange bool
    if err := tx.QueryRow(ctx, `
        SELECT status, must_change_password FROM users WHERE principal_id = $1
    `, user.PrincipalID).Scan(&finalStatus, &mustChange); err != nil {
        t.Fatalf("read explicitly enabled account: %v", err)
    }
    if finalStatus != "active" || !mustChange {
        t.Fatalf("explicit enable did not activate temporary-password account: status=%q must_change=%v", finalStatus, mustChange)
    }

    legacy, err := repo.CreateStaffAccount(ctx, actorPrincipalID, schoolID,
        "legacy-"+suffix, "integration-test-password-hash")
    if err != nil { t.Fatalf("create profile-incomplete legacy account: %v", err) }
    if err := repo.DisableStaffAccount(ctx, actorPrincipalID, schoolID, legacy.PrincipalID); err != nil {
        t.Fatalf("disable profile-incomplete legacy account: %v", err)
    }
    if err := repo.EnableStaffAccount(ctx, actorPrincipalID, schoolID, legacy.PrincipalID, "legacy-temporary-password-hash"); !errors.Is(err, ErrStaffProfileIncomplete) {
        t.Fatalf("profile-incomplete login enable error = %v, want ErrStaffProfileIncomplete", err)
    }
}
