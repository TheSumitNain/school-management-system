package repository

import (
	"context"
	"strconv"
	"testing"
	"time"
)

// TestSchoolSystemRoleProvisioningV13PostgresIntegration verifies that each
// newly inserted school receives exactly the approved School Admin and Teacher
// system roles, and that direct runtime system-role mutations remain blocked.
func TestSchoolSystemRoleProvisioningV13PostgresIntegration(t *testing.T) {
	_, pool, _ := newPostgresIntegrationRepository(t)
	ctx := context.Background()

	var hasV13 bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regprocedure('public.provision_school_system_roles()') IS NOT NULL
		   AND EXISTS (
		       SELECT 1 FROM pg_trigger
		        WHERE tgname = 'trg_provision_school_system_roles'
		          AND NOT tgisinternal
		   )
	`).Scan(&hasV13); err != nil {
		t.Fatalf("detect V13 school role provisioning: %v", err)
	}
	if !hasV13 {
		t.Skip("V13 school system-role provisioning is not applied; apply migrations to school_management_test first")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin V13 integration transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	schoolCode := "V13-" + suffix
	var schoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'pending_setup')
		RETURNING id
	`, "V13 Provisioning "+suffix, schoolCode).Scan(&schoolID); err != nil {
		t.Fatalf("create school and provision system roles: %v", err)
	}

	expected := map[string]map[string]bool{
		"school_admin": {
			"school.settings.read": true,
			"school.settings.update": true,
			"school.audit.read": true,
			"academic_years.read": true,
			"academic_years.manage": true,
			"classes.read": true,
			"classes.manage": true,
			"subjects.read": true,
			"subjects.manage": true,
			"teacher_assignments.read": true,
			"teacher_assignments.manage": true,
			"class_incharge_assignments.read": true,
			"class_incharge_assignments.manage": true,
			"students.read_basic": true,
			"students.read_sensitive": true,
			"students.create": true,
			"students.update": true,
			"students.archive": true,
			"attendance.read": true,
			"attendance.mark": true,
			"attendance.update_same_day": true,
			"attendance.correct_past": true,
			"timetable.read": true,
			"timetable.manage": true,
			"homework.read": true,
			"homework.create": true,
			"homework.update": true,
			"homework.delete": true,
			"exams.read": true,
			"exams.manage": true,
			"results.read": true,
			"results.enter": true,
			"results.update": true,
			"results.publish": true,
			"notices.read": true,
			"notices.create": true,
			"notices.update": true,
			"notices.publish": true,
			"notices.delete": true,
			"users.read": true,
			"users.create": true,
			"users.update": true,
			"users.disable": true,
			"users.password_reset": true,
			"roles.read": true,
			"roles.create_custom": true,
			"roles.update_custom": true,
			"roles.assign": true,
			"permissions.read": true,
			"permissions.assign": true,
			"permissions.override": true,
		},
		"teacher": {
			"students.read_basic": true,
			"attendance.read": true,
			"attendance.mark": true,
			"attendance.update_same_day": true,
			"timetable.read": true,
			"homework.read": true,
			"notices.read": true,
		},
	}

	actual := make(map[string]map[string]bool)
	rows, err := tx.Query(ctx, `
		SELECT r.role_key, p.permission_key
		  FROM roles r
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE r.school_id = $1
		   AND r.is_system = TRUE
		   AND r.role_key IN ('school_admin', 'teacher')
		 ORDER BY r.role_key, p.permission_key
	`, schoolID)
	if err != nil {
		t.Fatalf("load provisioned role permissions: %v", err)
	}
	for rows.Next() {
		var roleKey, permissionKey string
		if err := rows.Scan(&roleKey, &permissionKey); err != nil {
			rows.Close()
			t.Fatalf("scan provisioned role permission: %v", err)
		}
		if actual[roleKey] == nil {
			actual[roleKey] = make(map[string]bool)
		}
		actual[roleKey][permissionKey] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatalf("iterate provisioned role permissions: %v", err)
	}
	rows.Close()

	var provisionedRoleCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM roles
		 WHERE school_id = $1 AND is_system = TRUE
		   AND role_key IN ('school_admin', 'teacher')
	`, schoolID).Scan(&provisionedRoleCount); err != nil {
		t.Fatalf("count provisioned system roles: %v", err)
	}
	if provisionedRoleCount != 2 {
		t.Fatalf("expected exactly 2 school system roles, got %d", provisionedRoleCount)
	}

	if len(actual) != len(expected) {
		t.Fatalf("expected exactly %d provisioned system roles, got %#v", len(expected), actual)
	}
	for roleKey, expectedPermissions := range expected {
		got := actual[roleKey]
		if len(got) != len(expectedPermissions) {
			t.Fatalf("role %q: expected %d permissions, got %d (%v)", roleKey, len(expectedPermissions), len(got), got)
		}
		for permissionKey := range expectedPermissions {
			if !got[permissionKey] {
				t.Errorf("role %q is missing permission %q", roleKey, permissionKey)
			}
		}
	}

	expectFailure := func(name string, operation func() error) {
		t.Helper()
		if _, err := tx.Exec(ctx, "SAVEPOINT v13_expected_failure"); err != nil {
			t.Fatalf("%s: create savepoint: %v", name, err)
		}
		operationErr := operation()
		if operationErr == nil {
			_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT v13_expected_failure")
			_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT v13_expected_failure")
			t.Fatalf("%s: expected database rejection", name)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT v13_expected_failure"); err != nil {
			t.Fatalf("%s: rollback expected failure: %v", name, err)
		}
		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT v13_expected_failure"); err != nil {
			t.Fatalf("%s: release expected failure: %v", name, err)
		}
	}

	expectFailure("direct system-role insertion", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO roles
				(school_id, principal_type, scope_type, role_key, name, is_system)
			VALUES ($1, 'staff', 'school', 'forged_system', 'Forged System Role', TRUE)
		`, schoolID)
		return err
	})

	expectFailure("direct system-role permission insertion", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT r.id, p.id
			  FROM roles r
			  JOIN permissions p ON p.permission_key = 'guardians.read'
			 WHERE r.school_id = $1
			   AND r.role_key = 'school_admin'
			   AND r.is_system = TRUE
		`, schoolID)
		return err
	})

expectFailure("direct system-role update", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE roles SET name = 'Modified School Admin'
			 WHERE school_id = $1 AND role_key = 'school_admin' AND is_system = TRUE
		`, schoolID)
		return err
	})

	expectFailure("direct system-role deletion", func() error {
		_, err := tx.Exec(ctx, `
			DELETE FROM roles
			 WHERE school_id = $1 AND role_key = 'teacher' AND is_system = TRUE
		`, schoolID)
		return err
	})

	expectFailure("direct system-role permission deletion", func() error {
		_, err := tx.Exec(ctx, `
			DELETE FROM role_permissions rp
			 USING roles r, permissions p
			 WHERE rp.role_id = r.id AND rp.permission_id = p.id
			   AND r.school_id = $1 AND r.role_key = 'school_admin'
			   AND r.is_system = TRUE AND p.permission_key = 'school.settings.read'
		`, schoolID)
		return err
	})

	if _, err := tx.Exec(ctx, "SAVEPOINT v13_missing_permission"); err != nil {
		t.Fatalf("create missing-permission savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE permissions SET is_active = FALSE WHERE permission_key = 'students.create'`,
	); err != nil {
		t.Fatalf("deactivate permission inside test savepoint: %v", err)
	}
	_, missingPermissionErr := tx.Exec(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'pending_setup')
	`, "V13 Incomplete "+suffix, "V13X-"+suffix)
	if missingPermissionErr == nil {
		t.Fatal("expected school creation to fail when a required system-role permission is inactive")
	}
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT v13_missing_permission"); err != nil {
		t.Fatalf("rollback missing-permission savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT v13_missing_permission"); err != nil {
		t.Fatalf("release missing-permission savepoint: %v", err)
	}

	var incompleteSchoolCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM schools WHERE code = $1
	`, "V13X-"+suffix).Scan(&incompleteSchoolCount); err != nil {
		t.Fatalf("check failed school insert rollback: %v", err)
	}
	if incompleteSchoolCount != 0 {
		t.Fatalf("expected failed school provisioning to roll back the school row, found %d", incompleteSchoolCount)
	}
}
