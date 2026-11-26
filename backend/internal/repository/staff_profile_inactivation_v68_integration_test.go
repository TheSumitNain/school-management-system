package repository

import (
    "context"
    "strings"
    "testing"
    "time"
)

// TestStaffProfileInactivationSerializesWithRefreshRotationPostgresIntegration
// protects the DB trigger's shared lock contract with refresh-session rotation.
func TestStaffProfileInactivationSerializesWithRefreshRotationPostgresIntegration(t *testing.T) {
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
    if !strings.Contains(normalized, "pg_advisory_xact_lock") ||
        !strings.Contains(normalized, "refresh-principal:") {
        t.Fatalf("profile inactivation must serialize on the refresh-principal lock; definition: %s", definition)
    }
}
