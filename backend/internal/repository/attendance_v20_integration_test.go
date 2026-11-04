package repository

import (
	"context"
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
