package repository

import (
	"context"
	"fmt"
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

func TestTimetablePeriodOverlapRejectedPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin rollback-scoped timetable test: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load school fixture: %v", err)
	}
	suffix := time.Now().UnixNano()
	var yearID int64
	if err := tx.QueryRow(ctx, `INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active') RETURNING id`,
		schoolID, fmt.Sprintf("timetable-overlap-%d", suffix)).Scan(&yearID); err != nil {
		t.Fatalf("create academic year fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,1,'Period 1',TIME '09:00',TIME '09:40')`, schoolID, yearID); err != nil {
		t.Fatalf("create first timetable period: %v", err)
	}
	savepoint, err := tx.Begin(ctx)
	if err != nil { t.Fatalf("begin overlap savepoint: %v", err) }
	_, overlapErr := tx.Exec(ctx, `INSERT INTO timetable_periods(school_id,academic_year_id,period_number,name,starts_at,ends_at)
		VALUES($1,$2,2,'Period 2',TIME '09:30',TIME '10:10')`, schoolID, yearID)
	if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil {
		t.Fatalf("rollback overlap savepoint: %v", rollbackErr)
	}
	if overlapErr == nil {
		t.Fatal("overlapping timetable periods were accepted")
	}
}
