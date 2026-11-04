package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestAcademicFoundationV17PostgresIntegration exercises configurable class
// names, year-specific sections, promotion history, teacher scope, and tenant FKs.
func TestAcademicFoundationAndAssignmentHistoryV18PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V17 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Expected constraint failures are isolated in nested transactions because
	// PostgreSQL aborts the current transaction after a failed statement.
	expectRejected := func(label string, operation func() error) {
		t.Helper()
		savepoint, savepointErr := tx.Begin(ctx)
		if savepointErr != nil {
			t.Fatalf("begin savepoint for %s: %v", label, savepointErr)
		}
		operationErr := operation()
		if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil {
			t.Fatalf("rollback savepoint for %s: %v", label, rollbackErr)
		}
		if operationErr == nil {
			t.Fatalf("%s unexpectedly succeeded", label)
		}
	}

	var schoolID, actorPrincipalID, teacherPrincipalID int64
	if err := tx.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load active staff actor fixture: %v", err)
	}

	var hasV17 bool
	if err := tx.QueryRow(ctx, `
		SELECT to_regclass('public.academic_years') IS NOT NULL
		   AND to_regclass('public.classes') IS NOT NULL
		   AND to_regclass('public.sections') IS NOT NULL
		   AND to_regclass('public.subjects') IS NOT NULL
		   AND to_regclass('public.student_enrollments') IS NOT NULL
		   AND to_regclass('public.teacher_subject_assignments') IS NOT NULL
		   AND to_regclass('public.class_incharge_assignments') IS NOT NULL
	`).Scan(&hasV17); err != nil {
		t.Fatalf("detect V17 academic schema: %v", err)
	}
	if !hasV17 {
		t.Fatal("V17 academic foundation is missing; apply migrations to school_management_test first")
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active') RETURNING principal_id
	`, schoolID, "v17-teacher-"+suffix, "integration-test-password-hash").Scan(&teacherPrincipalID); err != nil {
		t.Fatalf("create Teacher account fixture: %v", err)
	}
	// The principal_roles guard requires the assigning actor to be an
	// authorized School Admin. Grant that role to the fixture actor through a
	// rollback-scoped Platform Admin, matching the production authorization path.
	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'school_admin' AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load system School Admin role: %v", err)
	}
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND r.is_system = TRUE
		   AND p.permission_key = 'platform.permissions.delegate'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if err != nil {
		var bootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1 FROM principal_roles pr
			        JOIN roles r ON r.id = pr.role_id
			        WHERE r.principal_type = 'platform_admin'
			          AND r.scope_type = 'platform'
			          AND r.role_key = 'platform_admin'
			    )
		`).Scan(&bootstrapExists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if bootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("academic-v17-%s@example.invalid", suffix),
			"integration-test-password-hash",
		)
		if err != nil {
			t.Fatalf("create rollback-scoped Platform Admin fixture: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("grant School Admin role to fixture actor: %v", err)
	}

	var teacherRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'teacher' AND is_system = TRUE
	`, schoolID).Scan(&teacherRoleID); err != nil {
		t.Fatalf("load system Teacher role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, teacherPrincipalID, teacherRoleID, actorPrincipalID); err != nil {
		t.Fatalf("assign Teacher role to fixture: %v", err)
	}

	var year1, year2, classLKG, classUKG, section1, section2, subjectID, studentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years (school_id, name, starts_on, ends_on, status, is_current)
		VALUES ($1, $2, DATE '2026-04-01', DATE '2027-03-31', 'active', TRUE)
		RETURNING id
	`, schoolID, "2026-27-"+suffix).Scan(&year1); err != nil {
		t.Fatalf("create current academic year: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years (school_id, name, starts_on, ends_on, status)
		VALUES ($1, $2, DATE '2027-04-01', DATE '2028-03-31', 'planned')
		RETURNING id
	`, schoolID, "2027-28-"+suffix).Scan(&year2); err != nil {
		t.Fatalf("create next academic year: %v", err)
	}

	for _, item := range []struct {
		name string
		out  *int64
	}{
		{"LKG", &classLKG},
		{"UKG", &classUKG},
	} {
		if err := tx.QueryRow(ctx, `
			INSERT INTO classes (school_id, name, display_order)
			VALUES ($1, $2, 10)
			RETURNING id
		`, schoolID, item.name).Scan(item.out); err != nil {
			t.Fatalf("create configurable class %q: %v", item.name, err)
		}
	}
	// Class names are arbitrary school data, not a fixed enum. Duplicate names
	// are rejected after trimming and case-folding.
	expectRejected("normalized duplicate class name", func() error {
		_, err := tx.Exec(ctx, `INSERT INTO classes (school_id, name) VALUES ($1, $2)`, schoolID, "  lkg  ")
		return err
	})

	if err := tx.QueryRow(ctx, `
		INSERT INTO sections (school_id, academic_year_id, class_id, name)
		VALUES ($1, $2, $3, 'A') RETURNING id
	`, schoolID, year1, classLKG).Scan(&section1); err != nil {
		t.Fatalf("create LKG section for first year: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections (school_id, academic_year_id, class_id, name)
		VALUES ($1, $2, $3, 'A') RETURNING id
	`, schoolID, year2, classUKG).Scan(&section2); err != nil {
		t.Fatalf("create UKG section for next year: %v", err)
	}
	expectRejected("duplicate normalized section name", func() error {
		_, err := tx.Exec(ctx, `INSERT INTO sections (school_id, academic_year_id, class_id, name) VALUES ($1, $2, $3, ' a ')`, schoolID, year1, classLKG)
		return err
	})

	if err := tx.QueryRow(ctx, `
		INSERT INTO subjects (school_id, name, subject_code)
		VALUES ($1, 'Mathematics', $2) RETURNING id
	`, schoolID, "MATH-"+suffix).Scan(&subjectID); err != nil {
		t.Fatalf("create school subject: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "V17-"+suffix, "V17 Promotion Student").Scan(&studentID); err != nil {
		t.Fatalf("create student fixture: %v", err)
	}

	var enrollment1, enrollment2 int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollments
			(school_id, student_id, academic_year_id, class_id, section_id, roll_number)
		VALUES ($1, $2, $3, $4, $5, '01') RETURNING id
	`, schoolID, studentID, year1, classLKG, section1).Scan(&enrollment1); err != nil {
		t.Fatalf("enroll student in LKG: %v", err)
	}
	var historicalClassName, historicalSectionName string
	if err := tx.QueryRow(ctx, `
		SELECT class_name_snapshot, section_name_snapshot
		  FROM student_enrollments WHERE id = $1
	`, enrollment1).Scan(&historicalClassName, &historicalSectionName); err != nil {
		t.Fatalf("read enrollment-time class/section snapshots: %v", err)
	}
	if historicalClassName != "LKG" || historicalSectionName != "A" {
		t.Fatalf("enrollment snapshots = %q/%q, want LKG/A", historicalClassName, historicalSectionName)
	}
	if _, err := tx.Exec(ctx, `UPDATE classes SET name = 'Lower Kindergarten' WHERE id = $1`, classLKG); err != nil {
		t.Fatalf("rename configurable class: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT class_name_snapshot FROM student_enrollments WHERE id = $1
	`, enrollment1).Scan(&historicalClassName); err != nil {
		t.Fatalf("re-read historical class snapshot after class rename: %v", err)
	}
	if historicalClassName != "LKG" {
		t.Fatalf("class rename rewrote historical enrollment label to %q", historicalClassName)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_enrollments SET roll_number = '02' WHERE id = $1
	`, enrollment1); err != nil {
		t.Fatalf("allow roll number correction while enrollment is active: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_enrollments
		   SET status = 'promoted', ended_at = DATE '2027-03-31'
		 WHERE id = $1
	`, enrollment1); err != nil {
		t.Fatalf("close prior-year enrollment for promotion: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollments
			(school_id, student_id, academic_year_id, class_id, section_id, roll_number)
		VALUES ($1, $2, $3, $4, $5, '01') RETURNING id
	`, schoolID, studentID, year2, classUKG, section2).Scan(&enrollment2); err != nil {
		t.Fatalf("create next-year UKG enrollment without overwriting history: %v", err)
	}
	var historyCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_enrollments
		 WHERE student_id = $1 AND id IN ($2, $3)
	`, studentID, enrollment1, enrollment2).Scan(&historyCount); err != nil {
		t.Fatalf("count promotion history: %v", err)
	}
	if historyCount != 2 {
		t.Fatalf("promotion retained %d enrollment rows, want 2", historyCount)
	}
	expectRejected("historical enrollment deletion", func() error {
		_, err := tx.Exec(ctx, `DELETE FROM student_enrollments WHERE id = $1`, enrollment1)
		return err
	})
	expectRejected("closed enrollment roll-number update", func() error {
		_, err := tx.Exec(ctx, `UPDATE student_enrollments SET roll_number = '03' WHERE id = $1`, enrollment1)
		return err
	})
	expectRejected("historical enrollment identity rewrite", func() error {
		_, err := tx.Exec(ctx, `UPDATE student_enrollments SET class_id = $1 WHERE id = $2`, classLKG, enrollment1)
		return err
	})

	if _, err := tx.Exec(ctx, `
		INSERT INTO teacher_subject_assignments
			(school_id, academic_year_id, section_id, subject_id,
			 teacher_principal_id, assigned_by_principal_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, schoolID, year2, section2, subjectID, teacherPrincipalID, actorPrincipalID); err != nil {
		t.Fatalf("assign active same-school teacher to subject/section: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO class_incharge_assignments
			(school_id, academic_year_id, section_id,
			 teacher_principal_id, assigned_by_principal_id)
		VALUES ($1, $2, $3, $4, $5)
	`, schoolID, year2, section2, teacherPrincipalID, actorPrincipalID); err != nil {
		t.Fatalf("assign class in-charge teacher: %v", err)
	}
	expectRejected("second active class-in-charge", func() error {
		_, err := tx.Exec(ctx, `INSERT INTO class_incharge_assignments (school_id, academic_year_id, section_id, teacher_principal_id, assigned_by_principal_id) VALUES ($1, $2, $3, $4, $5)`, schoolID, year2, section2, teacherPrincipalID, actorPrincipalID)
		return err
	})

	// Academic sections may be deactivated after use, but must not be
	// reactivated once their class is inactive.
	if _, err := tx.Exec(ctx, `UPDATE classes SET is_active = FALSE WHERE id = $1`, classLKG); err != nil {
		t.Fatalf("deactivate historical LKG class: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sections SET is_active = FALSE WHERE id = $1`, section1); err != nil {
		t.Fatalf("deactivate historical LKG section: %v", err)
	}
	expectRejected("reactivate section under inactive class", func() error {
		_, err := tx.Exec(ctx, `UPDATE sections SET is_active = TRUE WHERE id = $1`, section1)
		return err
	})

	// Assignment records are append-only except for their one-way revocation.
	expectRejected("teacher subject assignment identity rewrite", func() error {
		_, err := tx.Exec(ctx, `UPDATE teacher_subject_assignments SET subject_id = $1 WHERE school_id = $2 AND section_id = $3 AND teacher_principal_id = $4 AND revoked_at IS NULL`, subjectID, schoolID, section2, teacherPrincipalID)
		return err
	})
	expectRejected("teacher subject assignment deletion", func() error {
		_, err := tx.Exec(ctx, `DELETE FROM teacher_subject_assignments WHERE school_id = $1 AND section_id = $2 AND teacher_principal_id = $3 AND revoked_at IS NULL`, schoolID, section2, teacherPrincipalID)
		return err
	})
	if _, err := tx.Exec(ctx, `
		UPDATE teacher_subject_assignments SET revoked_at = NOW()
		 WHERE school_id = $1 AND section_id = $2 AND teacher_principal_id = $3 AND revoked_at IS NULL
	`, schoolID, section2, teacherPrincipalID); err != nil {
		t.Fatalf("revoke teacher subject assignment: %v", err)
	}
	expectRejected("teacher assignment reactivation after revocation", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE teacher_subject_assignments SET revoked_at = NULL
			 WHERE school_id = $1 AND section_id = $2 AND teacher_principal_id = $3
		`, schoolID, section2, teacherPrincipalID)
		return err
	})
	expectRejected("class in-charge assignment identity rewrite", func() error {
		_, err := tx.Exec(ctx, `UPDATE class_incharge_assignments SET assigned_by_principal_id = $1 WHERE school_id = $2 AND section_id = $3 AND teacher_principal_id = $4 AND revoked_at IS NULL`, platformAdminPrincipalID, schoolID, section2, teacherPrincipalID)
		return err
	})
	if _, err := tx.Exec(ctx, `
		UPDATE class_incharge_assignments SET revoked_at = NOW()
		 WHERE school_id = $1 AND section_id = $2 AND teacher_principal_id = $3 AND revoked_at IS NULL
	`, schoolID, section2, teacherPrincipalID); err != nil {
		t.Fatalf("revoke class in-charge assignment: %v", err)
	}

	var otherSchoolID, foreignClassID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active') RETURNING id
	`, "V17 Isolation "+suffix, "V17-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second-school fixture: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO classes (school_id, name) VALUES ($1, 'Play Group') RETURNING id
	`, otherSchoolID).Scan(&foreignClassID); err != nil {
		t.Fatalf("create second-school configurable class: %v", err)
	}
	expectRejected("cross-school section class reference", func() error {
		_, err := tx.Exec(ctx, `INSERT INTO sections (school_id, academic_year_id, class_id, name) VALUES ($1, $2, $3, 'Foreign')`, schoolID, year1, foreignClassID)
		return err
	})
	expectRejected("class school ownership change", func() error {
		_, err := tx.Exec(ctx, `UPDATE classes SET school_id = $1 WHERE id = $2`, otherSchoolID, classLKG)
		return err
	})
}
