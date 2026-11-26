package repository

import (
    "context"
    "errors"
    "fmt"
    "strings"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgconn"
)

func TestStaffProfileInactivationUsesNonBlockingRefreshLockPostgresIntegration(t *testing.T) {
    _, pool, _ := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    var definition string
    if err := pool.QueryRow(ctx, `
        SELECT pg_get_functiondef('public.disable_staff_login_for_inactive_profile()'::regprocedure)
    `).Scan(&definition); err != nil {
        t.Fatalf("read profile-inactivation trigger definition: %v", err)
    }
    normalized := strings.ToLower(strings.Join(strings.Fields(definition), " "))
    if !strings.Contains(normalized, "pg_try_advisory_xact_lock") ||
        !strings.Contains(normalized, "errcode = '40001'") ||
        !strings.Contains(normalized, "staff_profile_auth_lifecycle_serialization") {
        t.Fatalf("profile inactivation must fail fast with a retryable serialization error rather than wait while holding the profile row: %s", definition)
    }
}

func TestStaffProfileInactivationFailsFastDuringRefreshLockPostgresIntegration(t *testing.T) {
    _, pool, actorUserID := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    var schoolID int64
    if err := pool.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, actorUserID).Scan(&schoolID); err != nil {
        t.Fatalf("read integration school: %v", err)
    }
    suffix := fmt.Sprintf("%d", time.Now().UnixNano())
    var userID, principalID int64
    if err := pool.QueryRow(ctx, `
        INSERT INTO users (school_id, username, password_hash, status)
        VALUES ($1, $2, 'integration-test-hash', 'active')
        RETURNING id, principal_id
    `, schoolID, "v69-"+suffix).Scan(&userID, &principalID); err != nil {
        t.Fatalf("create staff fixture: %v", err)
    }
    t.Cleanup(func() {
        cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cleanupCancel()
        tx, err := pool.Begin(cleanupCtx)
        if err != nil { t.Errorf("begin V69 fixture cleanup: %v", err); return }
        defer func() { _ = tx.Rollback(cleanupCtx) }()
        for _, query := range []string{
            `DELETE FROM staff_profiles WHERE principal_id=$1`,
            `DELETE FROM refresh_sessions WHERE principal_id=$1`,
            `DELETE FROM users WHERE id=$1`,
            `DELETE FROM auth_principals WHERE id=$1`,
        } {
            arg := principalID
            if strings.Contains(query, "users WHERE") { arg = userID }
            if _, err := tx.Exec(cleanupCtx, query, arg); err != nil {
                t.Errorf("clean V69 fixture with %q: %v", query, err)
                return
            }
        }
        if err := tx.Commit(cleanupCtx); err != nil { t.Errorf("commit V69 fixture cleanup: %v", err) }
    })

    if _, err := pool.Exec(ctx, `
        INSERT INTO staff_profiles
            (principal_id, school_id, employee_code, full_name, designation, mobile_number, employment_status)
        VALUES ($1, $2, $3, 'V69 Integration Staff', 'Teacher', '9876543210', 'active')
    `, principalID, schoolID, "V69-"+suffix); err != nil {
        t.Fatalf("create staff profile fixture: %v", err)
    }

    lockTx, err := pool.Begin(ctx)
    if err != nil { t.Fatalf("begin refresh-lock transaction: %v", err) }
    defer func() { _ = lockTx.Rollback(context.Background()) }()
    if _, err := lockTx.Exec(ctx,
        `SELECT pg_advisory_xact_lock(hashtextextended('refresh-principal:' || $1::bigint::text,0))`,
        principalID,
    ); err != nil {
        t.Fatalf("hold refresh principal lock: %v", err)
    }

    _, err = pool.Exec(ctx, `
        UPDATE staff_profiles SET employment_status='inactive', updated_at=NOW()
        WHERE principal_id=$1
    `, principalID)
    var pgErr *pgconn.PgError
    if !strings.Contains(fmt.Sprintf("%v", err), "conflicts with an in-flight authentication change") ||
        !asPostgresCode(err, &pgErr) || pgErr.Code != "40001" {
        t.Fatalf("profile inactivation should fail fast with SQLSTATE 40001 while refresh lock is held; got %v", err)
    }

    var employmentStatus, principalStatus string
    if err := pool.QueryRow(ctx, `
        SELECT sp.employment_status, ap.status
          FROM staff_profiles sp JOIN auth_principals ap ON ap.id=sp.principal_id
         WHERE sp.principal_id=$1
    `, principalID).Scan(&employmentStatus, &principalStatus); err != nil {
        t.Fatalf("read state after rejected inactivation: %v", err)
    }
    if employmentStatus != "active" || principalStatus != "active" {
        t.Fatalf("rejected profile update partially changed state: employment=%q principal=%q", employmentStatus, principalStatus)
    }
}

func asPostgresCode(err error, target **pgconn.PgError) bool {
    if err == nil { return false }
    return errors.As(err, target)
}
