package repository

import (
	"context"
	"testing"
	"time"
)

func TestStudentGuardianRelationshipHistoryV36PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.ux_student_guardians_active_pair') IS NOT NULL
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_constraint
		        WHERE conrelid='public.student_guardians'::regclass
		          AND conname='student_guardians_unique'
		   )
	`).Scan(&ready); err != nil {
		t.Fatalf("inspect V36 active relationship index: %v", err)
	}
	if !ready {
		t.Fatal("V36 active-only guardian relationship uniqueness is missing")
	}
}
