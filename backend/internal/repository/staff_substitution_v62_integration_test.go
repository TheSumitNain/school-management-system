package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStaffSubstitutionV62CanonicalPrincipalGuardPostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var triggerExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_trigger
			 WHERE tgname='trg_validate_v62_substitute_principal'
			   AND NOT tgisinternal
		)
	`).Scan(&triggerExists); err != nil {
		t.Fatalf("inspect V62 canonical principal guard: %v", err)
	}
	if !triggerExists {
		t.Fatal("V62 canonical principal validation trigger is missing")
	}

	var definition string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_functiondef('public.validate_v62_substitute_principal()'::regprocedure)
	`).Scan(&definition); err != nil {
		t.Fatalf("read V62 canonical principal guard: %v", err)
	}
	for _, required := range []string{"public.auth_principals", "principal_type", "status", "FOR UPDATE"} {
		if !strings.Contains(definition, required) {
			t.Fatalf("V62 guard must validate and lock canonical auth principal state; missing %q in: %s", required, definition)
		}
	}
}
