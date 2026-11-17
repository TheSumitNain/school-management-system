package repository

import (
	"context"
	"testing"
	"time"
)

// TestStudentEnrollmentDirectoryIndexesV48PostgresIntegration verifies that
// both all-years and selected-year directory ordering have matching indexes.
func TestStudentEnrollmentDirectoryIndexesV48PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var allYears, selectedYear bool
	if err := pool.QueryRow(ctx, `
		SELECT
			to_regclass('public.ix_student_enrollments_school_enrolled_at') IS NOT NULL,
			to_regclass('public.ix_student_enrollments_school_year_enrolled_at') IS NOT NULL
	`).Scan(&allYears, &selectedYear); err != nil {
		t.Fatalf("inspect enrollment directory indexes: %v", err)
	}
	if !allYears || !selectedYear {
		t.Fatalf("enrollment directory indexes missing: all-years=%v selected-year=%v", allYears, selectedYear)
	}
}
