package repository

import (
    "context"
    "strings"
    "testing"
    "time"
)

// TestStudentLocationHierarchyParentLockV47PostgresIntegration verifies that
// V47 strengthens the hierarchy trigger's parent lock so parent deactivation
// cannot race with insertion of an active child location.
func TestStudentLocationHierarchyParentLockV47PostgresIntegration(t *testing.T) {
    _, pool, _ := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    var functionDefinition string
    if err := pool.QueryRow(ctx, `
        SELECT pg_get_functiondef('public.guard_student_location_hierarchy()'::regprocedure)
    `).Scan(&functionDefinition); err != nil {
        t.Fatalf("read student location hierarchy trigger definition: %v", err)
    }
    normalized := strings.ToUpper(functionDefinition)
    if !strings.Contains(normalized, "FOR SHARE") {
        t.Fatal("student location hierarchy trigger must lock its parent FOR SHARE")
    }
    if strings.Contains(normalized, "FOR KEY SHARE") {
        t.Fatal("student location hierarchy trigger still uses the weaker FOR KEY SHARE lock")
    }
}
