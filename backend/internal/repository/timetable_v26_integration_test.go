package repository

import (
	"context"
	"testing"
	"time"
)

func TestTimetableV26SchemaPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var ready bool
	err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.timetable_periods') IS NOT NULL
		   AND to_regclass('public.timetable_entries') IS NOT NULL
		   AND to_regclass('public.ux_timetable_entries_section_slot') IS NOT NULL
		   AND to_regclass('public.ux_timetable_entries_teacher_slot') IS NOT NULL
		   AND EXISTS (
		       SELECT 1 FROM pg_trigger
		        WHERE tgname = 'trg_validate_v26_timetable_entry'
		          AND NOT tgisinternal
		   )
		   AND EXISTS (
		       SELECT 1 FROM pg_trigger
		        WHERE tgname = 'trg_guard_v26_teacher_assignment_schedule'
		          AND NOT tgisinternal
		   )
		   AND EXISTS (
		       SELECT 1 FROM pg_constraint
		        WHERE conname = 'timetable_entries_assignment_scope_fkey'
		   )
	`).Scan(&ready)
	if err != nil {
		t.Fatalf("inspect V26 timetable schema: %v", err)
	}
	if !ready {
		t.Fatal("V26 timetable tables, conflict indexes, assignment scope, or validation triggers are missing")
	}
}
