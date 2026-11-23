package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStaffSubstitutionV61SerializesSlotValidationPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var triggerExists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM pg_trigger
			 WHERE tgname = 'trg_lock_v61_staff_substitution_slot'
			   AND NOT tgisinternal
		)
	`).Scan(&triggerExists)
	if err != nil {
		t.Fatalf("inspect V61 substitution slot-lock trigger: %v", err)
	}
	if !triggerExists {
		t.Fatal("V61 substitution slot-lock trigger is missing")
	}

	var definition string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_functiondef('public.lock_v61_staff_substitution_slot()'::regprocedure)
	`).Scan(&definition); err != nil {
		t.Fatalf("read V61 substitution slot-lock function: %v", err)
	}
	lockIndex := strings.Index(definition, "pg_advisory_xact_lock")
	rowLockIndex := strings.Index(definition, "FOR UPDATE")
	if rowLockIndex < 0 || lockIndex < 0 || rowLockIndex > lockIndex {
		t.Fatalf("V61 must lock the timetable entry row before acquiring the shared slot advisory lock; function: %s", definition)
	}
	if !strings.Contains(definition, "v_day_of_week") || !strings.Contains(definition, "v_period_id") {
		t.Fatalf("V61 lock key must use the authoritative timetable weekday and period: %s", definition)
	}
}
