package repository

import (
	"context"
	"testing"
)

// TestSchoolAdminGuardianPermissionsV39PostgresIntegration verifies that the
// V39 guardian grants are present on every existing School Admin system role,
// remain absent from Teacher roles, and retain the system-role immutability
// guard after the migration.
func TestSchoolAdminGuardianPermissionsV39PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV39 bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_proc
			 WHERE proname = 'provision_school_system_roles'
		)
		AND EXISTS (
			SELECT 1 FROM pg_trigger
			 WHERE tgname = 'trg_protect_system_role_permissions'
			   AND NOT tgisinternal
		)
		AND EXISTS (
			SELECT 1 FROM schema_migrations
			 WHERE version >= 39 AND dirty = FALSE
		)
	`).Scan(&hasV39); err != nil {
		t.Fatalf("detect V39 guardian permission migration: %v", err)
	}
	if !hasV39 {
		t.Skip("V39 guardian permission migration is not applied; apply migrations to school_management_test first")
	}

	var schoolAdminCount, missingAdminGrantCount, teacherGrantCount int
	if err := pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE r.role_key = 'school_admin'),
			COUNT(*) FILTER (
				WHERE r.role_key = 'school_admin'
				  AND (p.permission_key IS NULL OR rp.role_id IS NULL)
			),
			(
				SELECT COUNT(*)
				  FROM roles tr
				  JOIN role_permissions trp ON trp.role_id = tr.id
				  JOIN permissions tp ON tp.id = trp.permission_id
				 WHERE tr.is_system = TRUE
				   AND tr.scope_type = 'school'
				   AND tr.principal_type = 'staff'
				   AND tr.role_key = 'teacher'
				   AND tp.permission_key IN ('guardians.read', 'guardians.manage')
			)
		FROM roles r
		CROSS JOIN (VALUES ('guardians.read'), ('guardians.manage')) required(permission_key)
		LEFT JOIN permissions p
		  ON p.permission_key = required.permission_key
		 AND p.scope_type = 'school'
		 AND p.is_active = TRUE
		LEFT JOIN role_permissions rp
		  ON rp.role_id = r.id
		 AND rp.permission_id = p.id
		WHERE r.is_system = TRUE
		  AND r.scope_type = 'school'
		  AND r.principal_type = 'staff'
		  AND r.role_key = 'school_admin'
	`).Scan(&schoolAdminCount, &missingAdminGrantCount, &teacherGrantCount); err != nil {
		t.Fatalf("verify School Admin guardian grants: %v", err)
	}
	if schoolAdminCount == 0 {
		t.Fatal("expected at least one provisioned School Admin system role")
	}
	if missingAdminGrantCount != 0 {
		t.Fatalf("expected all School Admin roles to have both guardian permissions; missing grants: %d", missingAdminGrantCount)
	}
	if teacherGrantCount != 0 {
		t.Fatalf("expected no Teacher guardian grants, got %d", teacherGrantCount)
	}

	var triggerEnabled bool
	if err := pool.QueryRow(ctx, `
		SELECT tgenabled <> 'D'
		  FROM pg_trigger
		 WHERE tgname = 'trg_protect_system_role_permissions'
		   AND NOT tgisinternal
	`).Scan(&triggerEnabled); err != nil {
		t.Fatalf("verify system-role permission guard: %v", err)
	}
	if !triggerEnabled {
		t.Fatal("system-role permission immutability trigger must remain enabled")
	}
}
