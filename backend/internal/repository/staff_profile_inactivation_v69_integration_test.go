package repository

import (
    "context"
    "strings"
    "testing"
    "time"
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
