package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

// V67 keeps the login-disable guard on the same Asia/Kolkata calendar
// used by the staff attendance and substitution lifecycle rules.
func TestStaffSubstitutionLoginGuardUsesIndiaCalendarPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var definition string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_functiondef('public.guard_v60_substitution_login_lifecycle()'::regprocedure)
	`).Scan(&definition); err != nil {
		t.Fatalf("read substitution login lifecycle guard: %v", err)
	}
	if !strings.Contains(definition, "Asia/Kolkata") {
		t.Fatalf("substitution login guard must use Asia/Kolkata calendar date; definition: %s", definition)
	}
	if strings.Contains(definition, "ss.cover_date>=CURRENT_DATE") ||
		strings.Contains(definition, "ss.cover_date >= CURRENT_DATE") {
		t.Fatalf("substitution login guard must not depend on database session CURRENT_DATE; definition: %s", definition)
	}
}
