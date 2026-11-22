package repository

import (
	"context"
	"testing"
	"time"
)

func TestStaffSubstitutionV60SchemaPostgresIntegration(t *testing.T) {
	_,pool,_:=newPostgresIntegrationRepository(t)
	ctx,cancel:=context.WithTimeout(context.Background(),10*time.Second)
	defer cancel()
	var ready bool
	err:=pool.QueryRow(ctx,`
		SELECT to_regclass('public.staff_substitutions') IS NOT NULL
		  AND to_regclass('public.ux_staff_substitutions_entry_date_active') IS NOT NULL
		  AND to_regclass('public.ux_staff_substitutions_substitute_period_active') IS NOT NULL
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_validate_v60_staff_substitution' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_timetable_substitution_update' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_substitution_staff_lifecycle' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_substitution_login_lifecycle' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_substitution_principal_lifecycle' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_substitution_assignment_lifecycle' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='trg_guard_v60_staff_substitution_history' AND NOT tgisinternal)
		  AND EXISTS(SELECT 1 FROM pg_constraint WHERE conname='staff_substitutions_period_scope_fkey')
		  AND (SELECT count(*)=2 FROM public.permissions WHERE permission_key IN ('staff.substitutions.read','staff.substitutions.manage'))
	`).Scan(&ready)
	if err!=nil{t.Fatalf("inspect V60 staff substitution schema: %v",err)}
	if !ready{t.Fatal("V60 substitution table, uniqueness guards, validation triggers, timetable history guard, or permissions are missing")}
}
