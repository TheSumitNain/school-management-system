package repository

import (
	"context"
	"strings"
	"testing"
)

// TestExactDelegationLinkageV16PostgresIntegration verifies that both grant
// tables store delegation IDs and that their database guards validate the
// exact referenced delegation rather than any active school/permission pair.
func TestExactDelegationLinkageV16PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasColumns bool
	if err := pool.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'role_permissions'
				  AND column_name = 'delegation_id'
			)
			AND EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = 'public'
				  AND table_name = 'user_permission_overrides'
				  AND column_name = 'delegation_id'
			)
	`).Scan(&hasColumns); err != nil {
		t.Fatalf("detect V16 exact delegation columns: %v", err)
	}
	if !hasColumns {
		t.Skip("V16 exact delegation linkage is not applied; apply migrations to school_management_test first")
	}

	var roleGuard, overrideGuard string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_functiondef('public.validate_role_permission_scope()'::regprocedure),
		       pg_get_functiondef('public.validate_user_permission_override()'::regprocedure)
	`).Scan(&roleGuard, &overrideGuard); err != nil {
		t.Fatalf("read V16 delegation guard definitions: %v", err)
	}

	for _, required := range []string{"NEW.delegation_id", "d.id", "d.can_assign_to_roles", "d.revoked_at IS NULL"} {
		if !strings.Contains(roleGuard, required) {
			t.Errorf("role permission guard is missing exact-delegation check %q", required)
		}
	}
	for _, required := range []string{"NEW.delegation_id", "d.id", "d.can_override_users", "d.revoked_at IS NULL"} {
		if !strings.Contains(overrideGuard, required) {
			t.Errorf("user override guard is missing exact-delegation check %q", required)
		}
	}
}
