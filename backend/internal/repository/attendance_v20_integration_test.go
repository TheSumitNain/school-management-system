package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

// The integration suite applies every migration to a disposable PostgreSQL
// database. This verifies that V20's schema and enforcement objects are present.
func TestAttendanceV20SchemaPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ready bool
	err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.attendance_records') IS NOT NULL
		   AND to_regclass('public.ux_attendance_records_enrollment_date') IS NOT NULL
		   AND EXISTS (SELECT 1 FROM pg_trigger WHERE tgname='trg_validate_v20_attendance_record' AND NOT tgisinternal)
		   AND EXISTS (SELECT 1 FROM pg_proc WHERE proname='validate_v20_attendance_record')
	`).Scan(&ready)
	if err != nil { t.Fatalf("inspect V20 attendance schema: %v", err) }
	if !ready { t.Fatal("V20 attendance schema, unique index, or validation trigger is missing") }
}


func TestAttendanceUsesAsiaKolkataCalendarPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var definition string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_functiondef('public.validate_v20_attendance_record()'::regprocedure)
	`).Scan(&definition); err != nil {
		t.Fatalf("load attendance validation function: %v", err)
	}
	if !strings.Contains(definition, "Asia/Kolkata") {
		t.Fatal("attendance database trigger must validate future dates using the Asia/Kolkata school calendar")
	}
	if strings.Contains(definition, "NEW.attendance_date > CURRENT_DATE") {
		t.Fatal("attendance database trigger must not use the PostgreSQL session CURRENT_DATE")
	}
}



func TestAdminIdleTimeoutV21SchemaPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var ready bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema='public' AND table_name='refresh_sessions'
			  AND column_name='last_activity_at' AND is_nullable='NO'
		) AND to_regclass('public.ix_refresh_sessions_active_family_activity') IS NOT NULL
	`).Scan(&ready)
	if err != nil { t.Fatalf("inspect V21 admin idle-timeout schema: %v", err) }
	if !ready { t.Fatal("V21 refresh-session activity column or index is missing") }
}
