package repository

import (
    "context"
    "fmt"
    "testing"
    "time"
)

// TestStaffProfileInactiveLoginGuardPostgresIntegration verifies the database
// boundary: changing an employee profile to inactive disables the canonical
// principal, and no repository path can reactivate that principal until the
// profile is active again.
func TestStaffProfileInactiveLoginGuardPostgresIntegration(t *testing.T) {
    _, pool, actorUserID := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    var schoolID int64
    if err := pool.QueryRow(ctx, `SELECT school_id FROM users WHERE id = $1`, actorUserID).Scan(&schoolID); err != nil {
        t.Fatalf("load integration school: %v", err)
    }

    suffix := fmt.Sprintf("%d", time.Now().UnixNano())
    var userID, principalID int64
    if err := pool.QueryRow(ctx, `
        INSERT INTO users (school_id, username, password_hash, status)
        VALUES ($1, $2, 'integration-test-hash', 'active')
        RETURNING id, principal_id
    `, schoolID, "profile-guard-"+suffix).Scan(&userID, &principalID); err != nil {
        t.Fatalf("create staff account fixture: %v", err)
    }
    t.Cleanup(func() {
        cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cleanupCancel()
        _, _ = pool.Exec(cleanupCtx, `DELETE FROM staff_profiles WHERE principal_id = $1`, principalID)
        _, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
        _, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_principals WHERE id = $1`, principalID)
    })

    if _, err := pool.Exec(ctx, `
        INSERT INTO staff_profiles
            (principal_id, school_id, employee_code, full_name, designation, mobile_number, employment_status)
        VALUES ($1, $2, $3, 'Integration Teacher', 'Teacher', '9876543210', 'active')
    `, principalID, schoolID, "EMP-"+suffix); err != nil {
        t.Fatalf("create active staff profile: %v", err)
    }

    if _, err := pool.Exec(ctx, `
        UPDATE staff_profiles SET employment_status = 'inactive', updated_at = NOW()
        WHERE principal_id = $1
    `, principalID); err != nil {
        t.Fatalf("inactivate staff profile: %v", err)
    }

    var principalStatus, accountStatus string
    if err := pool.QueryRow(ctx, `
        SELECT ap.status, u.status
          FROM auth_principals ap JOIN users u ON u.principal_id = ap.id
         WHERE ap.id = $1
    `, principalID).Scan(&principalStatus, &accountStatus); err != nil {
        t.Fatalf("read disabled staff state: %v", err)
    }
    if principalStatus != "disabled" || accountStatus != "disabled" {
        t.Fatalf("inactive profile left login enabled: principal=%q account=%q", principalStatus, accountStatus)
    }

    if _, err := pool.Exec(ctx, `
        UPDATE auth_principals SET status = 'active', token_version = token_version + 1
        WHERE id = $1
    `, principalID); err == nil {
        t.Fatal("database allowed reactivation while employee profile is inactive")
    }
}
