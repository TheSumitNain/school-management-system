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

// TestStudentLocationHierarchyParentLockV47PostgresIntegration verifies that
// V47 changes the hierarchy trigger's parent lock to FOR SHARE. This lock
// conflicts with the NO KEY UPDATE lock taken by parent deactivation.
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

func TestStudentLocationParentDeactivationWaitsForConcurrentChildV47PostgresIntegration(t *testing.T) {
    _, pool, _ := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    var parentID int64
    if err := pool.QueryRow(ctx, "INSERT INTO student_locations(school_id,parent_id,country_code,level,name) VALUES(NULL,NULL,'IN','state',$1) RETURNING id",
        fmt.Sprintf("V47 concurrent parent %d", time.Now().UnixNano())).Scan(&parentID); err != nil {
        t.Fatalf("create V47 parent location: %v", err)
    }

    childTx, err := pool.Begin(ctx)
    if err != nil { t.Fatalf("begin child transaction: %v", err) }
    defer func() { _ = childTx.Rollback(ctx) }()
    if _, err := childTx.Exec(ctx, "INSERT INTO student_locations(school_id,parent_id,country_code,level,name) VALUES(NULL,$1,'IN','district',$2)",
        parentID, fmt.Sprintf("V47 concurrent child %d", time.Now().UnixNano())); err != nil {
        t.Fatalf("insert child while retaining parent lock: %v", err)
    }

    deactivationConn, err := pool.Acquire(ctx)
    if err != nil { t.Fatalf("acquire deactivation connection: %v", err) }
    defer deactivationConn.Release()
    applicationName := fmt.Sprintf("v47-parent-deactivate-%d", time.Now().UnixNano())
    if _, err := deactivationConn.Exec(ctx, "SELECT set_config('application_name',$1,false)", applicationName); err != nil {
        t.Fatalf("label deactivation connection: %v", err)
    }
    deactivationTx, err := deactivationConn.Begin(ctx)
    if err != nil { t.Fatalf("begin parent deactivation transaction: %v", err) }
    defer func() { _ = deactivationTx.Rollback(ctx) }()

    deactivationResult := make(chan error, 1)
    go func() {
        _, updateErr := deactivationTx.Exec(ctx, "UPDATE student_locations SET is_active=FALSE WHERE id=$1", parentID)
        deactivationResult <- updateErr
    }()

    blocked := false
    for !blocked {
        select {
        case updateErr := <-deactivationResult:
            if updateErr == nil { t.Fatal("parent deactivation completed before the child transaction committed") }
            t.Fatalf("parent deactivation failed before the expected lock wait: %v", updateErr)
        default:
        }
        if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock')",
            applicationName).Scan(&blocked); err != nil {
            t.Fatalf("observe parent deactivation lock wait: %v", err)
        }
        if !blocked {
            select {
            case updateErr := <-deactivationResult:
                if updateErr == nil { t.Fatal("parent deactivation completed before the child transaction committed") }
                t.Fatalf("parent deactivation failed before expected lock wait: %v", updateErr)
            case <-time.After(10 * time.Millisecond):
            case <-ctx.Done():
                t.Fatalf("parent deactivation never waited for the child row lock: %v", ctx.Err())
            }
        }
    }

    if err := childTx.Commit(ctx); err != nil { t.Fatalf("commit child after parent deactivation is blocked: %v", err) }
    updateErr := <-deactivationResult
    var pgErr *pgconn.PgError
    if !errors.As(updateErr, &pgErr) || pgErr.Code != "P0001" {
        t.Fatalf("parent deactivation error = %v, want active-child hierarchy exception (P0001)", updateErr)
    }
    if err := deactivationTx.Rollback(ctx); err != nil { t.Fatalf("rollback rejected parent deactivation: %v", err) }

    var parentActive bool
    if err := pool.QueryRow(ctx, "SELECT is_active FROM student_locations WHERE id=$1", parentID).Scan(&parentActive); err != nil {
        t.Fatalf("read parent state after concurrent lifecycle operations: %v", err)
    }
    if !parentActive { t.Fatal("parent was deactivated despite the newly committed active child") }
}
