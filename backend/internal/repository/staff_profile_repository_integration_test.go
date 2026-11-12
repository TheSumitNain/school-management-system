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
    if user == nil || !user.ProfileComplete || user.EmployeeCode != profile.EmployeeCode || user.FullName != profile.FullName {
        t.Fatalf("created staff profile not returned: %+v", user)
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

    profile.EmploymentStatus = "inactive"
    updated, err := repo.UpdateStaffProfile(ctx, actorPrincipalID, schoolID, user.PrincipalID, profile)
    if err != nil { t.Fatalf("mark employee profile inactive: %v", err) }
    if updated.Status != "disabled" || updated.EmploymentStatus != "inactive" {
        t.Fatalf("inactive employee did not disable login: %+v", updated)
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
}
