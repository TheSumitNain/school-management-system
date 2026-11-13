package repository

import (
    "context"
    "testing"
    "time"
)

// TestStudentEnrollmentAdmissionV31SchemaPostgresIntegration verifies the
// additive, year-scoped admission identity schema on the disposable PostgreSQL
// database used by the repository integration suite.
func TestStudentEnrollmentAdmissionV31SchemaPostgresIntegration(t *testing.T) {
    _, pool, _ := newPostgresIntegrationRepository(t)
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var ready bool
    err := pool.QueryRow(ctx, `
        SELECT
            (
                SELECT COUNT(*) = 2
                  FROM information_schema.columns
                 WHERE table_schema = 'public'
                   AND table_name = 'student_enrollments'
                   AND column_name IN ('admission_number', 'admission_date')
                   AND is_nullable = 'YES'
            )
            AND to_regclass('public.ux_student_enrollments_school_year_admission_number') IS NOT NULL
            AND EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = 'public.student_enrollments'::regclass
                   AND conname = 'student_enrollments_admission_number_nonempty'
                   AND contype = 'c'
            )
            AND EXISTS (
                SELECT 1
                  FROM pg_constraint
                 WHERE conrelid = 'public.student_enrollments'::regclass
                   AND conname = 'student_enrollments_admission_identity_pair'
                   AND contype = 'c'
            )
    `).Scan(&ready)
    if err != nil {
        t.Fatalf("inspect V31 enrollment admission schema: %v", err)
    }
    if !ready {
        t.Fatal("V31 enrollment admission columns, constraints, or year-scoped unique index are missing")
    }
}
