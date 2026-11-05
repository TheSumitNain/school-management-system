package repository

import (
	"context"
	"errors"
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

	var year1, year2, classLKG, classUKG, section1, section2, subjectID, subjectAltID, studentID int64
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
		INSERT INTO subjects (school_id, name, subject_code)
		VALUES ($1, 'Science', $2) RETURNING id
	`, schoolID, "SCI-"+suffix).Scan(&subjectAltID); err != nil {
		t.Fatalf("create alternate school subject: %v", err)
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
	// Give the same teacher a separate current-year section so attendance
	// behavior can be exercised against an enrollment whose date range includes
	// the database's current date.
	if _, err := tx.Exec(ctx, `
		INSERT INTO class_incharge_assignments
			(school_id, academic_year_id, section_id,
			 teacher_principal_id, assigned_by_principal_id)
		VALUES ($1, $2, $3, $4, $5)
	`, schoolID, year1, section1, teacherPrincipalID, actorPrincipalID); err != nil {
		t.Fatalf("assign current-year class in-charge teacher: %v", err)
	}
	attendanceRepo := NewAcademicRepository(tx)
	var attendanceDay time.Time
	if err := tx.QueryRow(ctx, `SELECT CURRENT_DATE`).Scan(&attendanceDay); err != nil {
		t.Fatalf("load PostgreSQL current date for attendance fixture: %v", err)
	}
	attendance, err := attendanceRepo.MarkAttendance(
		ctx, teacherPrincipalID, schoolID, enrollment1, attendanceDay, "present", nil,
	)
	if err != nil {
		t.Fatalf("class-in-charge teacher could not mark current-section attendance: %v", err)
	}
	if attendance == nil || attendance.Status != "present" || attendance.SectionName != "A" {
		t.Fatalf("unexpected marked attendance record: %+v", attendance)
	}
	attendanceItems, err := attendanceRepo.ListAttendance(
		ctx, teacherPrincipalID, schoolID, year1, section1, attendanceDay,
	)
	if err != nil || len(attendanceItems) != 1 || attendanceItems[0].ID != attendance.ID {
		t.Fatalf("class-in-charge teacher attendance listing: items=%+v err=%v", attendanceItems, err)
	}
	updatedAttendance, err := attendanceRepo.UpdateAttendance(
		ctx, teacherPrincipalID, schoolID, attendance.ID, "late", nil,
	)
	if err != nil || updatedAttendance == nil || updatedAttendance.Status != "late" {
		t.Fatalf("class-in-charge teacher same-day attendance update: record=%+v err=%v", updatedAttendance, err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE class_incharge_assignments SET revoked_at = NOW()
		 WHERE school_id = $1 AND section_id = $2 AND teacher_principal_id = $3
		   AND revoked_at IS NULL
	`, schoolID, section1, teacherPrincipalID); err != nil {
		t.Fatalf("revoke current-year class-in-charge assignment: %v", err)
	}
	if _, err := attendanceRepo.ListAttendance(
		ctx, teacherPrincipalID, schoolID, year1, section1, attendanceDay,
	); !errors.Is(err, ErrAttendanceAccessDenied) {
		t.Fatalf("revoked class-in-charge attendance listing error = %v, want ErrAttendanceAccessDenied", err)
	}
	if _, err := attendanceRepo.MarkAttendance(
		ctx, teacherPrincipalID, schoolID, enrollment1, attendanceDay, "present", nil,
	); !errors.Is(err, ErrAttendanceAccessDenied) {
		t.Fatalf("revoked class-in-charge attendance marking error = %v, want ErrAttendanceAccessDenied", err)
	}
	if _, err := attendanceRepo.UpdateAttendance(
		ctx, teacherPrincipalID, schoolID, attendance.ID, "absent", nil,
	); !errors.Is(err, ErrAttendanceAccessDenied) {
		t.Fatalf("revoked class-in-charge attendance update error = %v, want ErrAttendanceAccessDenied", err)
	}

	// Basic enrollment reads must be constrained to the teacher's currently
	// assigned sections, even when the teacher has school-level read_basic.
	teacherAcademicRepo := NewAcademicRepository(tx)
	teacherEnrollments, err := teacherAcademicRepo.ListStudentEnrollments(ctx, teacherPrincipalID, schoolID, 0, 0)
	if err != nil {
		t.Fatalf("list teacher's assigned-section enrollments: %v", err)
	}
	if len(teacherEnrollments) != 1 || teacherEnrollments[0].SectionID != section2 {
		t.Fatalf("teacher enrollment scope = %+v; want only assigned section %d", teacherEnrollments, section2)
	}
	if teacherEnrollments[0].StudentName != "V17 Promotion Student" ||
		teacherEnrollments[0].RollNumber == nil || *teacherEnrollments[0].RollNumber != "01" ||
		teacherEnrollments[0].ClassNameSnapshot != "UKG" ||
		teacherEnrollments[0].SectionNameSnapshot != "A" {
		t.Fatalf("teacher roster did not return the approved minimal student fields: %+v", teacherEnrollments[0])
	}
	unassignedHistory, err := teacherAcademicRepo.ListStudentEnrollments(ctx, teacherPrincipalID, schoolID, 0, studentID)
	if err != nil {
		t.Fatalf("read history for unassigned student section: %v", err)
	}
	if len(unassignedHistory) != 1 || unassignedHistory[0].SectionID != section2 {
		t.Fatalf("teacher student history exposed unassigned sections: %+v", unassignedHistory)
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
		_, err := tx.Exec(ctx, `UPDATE teacher_subject_assignments SET subject_id = $1 WHERE school_id = $2 AND section_id = $3 AND teacher_principal_id = $4 AND revoked_at IS NULL`, subjectAltID, schoolID, section2, teacherPrincipalID)
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

	// Revoking both assignment types must immediately remove the teacher's
	// minimal roster access, even while the Teacher role itself remains active.
	rosterAfterAssignmentRevocation, err := teacherAcademicRepo.ListStudentEnrollments(
		ctx, teacherPrincipalID, schoolID, year2, studentID,
	)
	if err != nil {
		t.Fatalf("read roster after assignment revocation: %v", err)
	}
	if len(rosterAfterAssignmentRevocation) != 0 {
		t.Fatalf("revoked assignments retained teacher roster access: %+v", rosterAfterAssignmentRevocation)
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

	// Exercise the production repository/service-facing operations against the
	// same PostgreSQL schema and rollback-scoped School Admin fixture.
	academicRepo := NewAcademicRepository(tx)
	apiYear, err := academicRepo.SaveAcademicYear(ctx, actorPrincipalID, schoolID, 0,
		"API Academic "+suffix, time.Date(2028, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2029, 3, 31, 0, 0, 0, 0, time.UTC), "planned", false)
	if err != nil { t.Fatalf("create academic year through repository: %v", err) }
	apiYear, err = academicRepo.SaveAcademicYear(ctx, actorPrincipalID, schoolID, apiYear.ID,
		apiYear.Name, apiYear.StartsOn, apiYear.EndsOn, "active", true)
	if err != nil || !apiYear.IsCurrent || apiYear.Status != "active" {
		t.Fatalf("activate current academic year through repository: year=%+v err=%v", apiYear, err)
	}
	apiClass, err := academicRepo.SaveClass(ctx, actorPrincipalID, schoolID, 0, "Pre-Nursery "+suffix, 5, true)
	if err != nil { t.Fatalf("create configurable class through repository: %v", err) }
	apiSection, err := academicRepo.SaveSection(ctx, actorPrincipalID, schoolID, 0, apiYear.ID, apiClass.ID, "B", 2, true)
	if err != nil { t.Fatalf("create year-specific section through repository: %v", err) }
	apiCode := "API-" + suffix
	apiSubject, err := academicRepo.SaveSubject(ctx, actorPrincipalID, schoolID, 0, "Environmental Studies "+suffix, &apiCode, true)
	if err != nil { t.Fatalf("create subject through repository: %v", err) }
	years, err := academicRepo.ListAcademicYears(ctx, actorPrincipalID, schoolID)
	if err != nil || len(years) < 3 { t.Fatalf("list academic years through repository: count=%d err=%v", len(years), err) }
	classes, err := academicRepo.ListClasses(ctx, actorPrincipalID, schoolID)
	if err != nil || len(classes) < 3 { t.Fatalf("list classes through repository: count=%d err=%v", len(classes), err) }
	sections, err := academicRepo.ListSections(ctx, actorPrincipalID, schoolID, apiYear.ID)
	if err != nil || len(sections) != 1 || sections[0].ID != apiSection.ID { t.Fatalf("list sections through repository: items=%+v err=%v", sections, err) }
	subjects, err := academicRepo.ListSubjects(ctx, actorPrincipalID, schoolID)
	if err != nil || len(subjects) < 3 || apiSubject.ID <= 0 { t.Fatalf("list subjects through repository: count=%d err=%v", len(subjects), err) }

	teacherAssignment, err := academicRepo.CreateTeacherSubjectAssignment(ctx, actorPrincipalID, schoolID,
		apiYear.ID, apiSection.ID, apiSubject.ID, teacherPrincipalID)
	if err != nil || teacherAssignment == nil || teacherAssignment.TeacherPrincipalID != teacherPrincipalID {
		t.Fatalf("create teacher subject assignment through repository: assignment=%+v err=%v", teacherAssignment, err)
	}
	teacherAssignments, err := academicRepo.ListTeacherSubjectAssignments(ctx, actorPrincipalID, schoolID, apiYear.ID, apiSection.ID)
	if err != nil || len(teacherAssignments) != 1 || teacherAssignments[0].ID != teacherAssignment.ID {
		t.Fatalf("list teacher subject assignments through repository: items=%+v err=%v", teacherAssignments, err)
	}
	if err := academicRepo.RevokeTeacherSubjectAssignment(ctx, actorPrincipalID, schoolID, teacherAssignment.ID); err != nil {
		t.Fatalf("revoke teacher subject assignment through repository: %v", err)
	}
	classIncharge, err := academicRepo.CreateClassInchargeAssignment(ctx, actorPrincipalID, schoolID,
		apiYear.ID, apiSection.ID, teacherPrincipalID)
	if err != nil || classIncharge == nil || classIncharge.TeacherPrincipalID != teacherPrincipalID {
		t.Fatalf("create class in-charge assignment through repository: assignment=%+v err=%v", classIncharge, err)
	}
	classIncharges, err := academicRepo.ListClassInchargeAssignments(ctx, actorPrincipalID, schoolID, apiYear.ID)
	if err != nil || len(classIncharges) != 1 || classIncharges[0].ID != classIncharge.ID {
		t.Fatalf("list class in-charge assignments through repository: items=%+v err=%v", classIncharges, err)
	}
	if err := academicRepo.RevokeClassInchargeAssignment(ctx, actorPrincipalID, schoolID, classIncharge.ID); err != nil {
		t.Fatalf("revoke class in-charge assignment through repository: %v", err)
	}

	// Exercise enrollment and promotion through the production repository.
	var apiStudentID int64
	if err := tx.QueryRow(ctx, `INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,$2,$3) RETURNING id`, schoolID, "API-ENROLL-"+suffix, "API Enrollment Student").Scan(&apiStudentID); err != nil {
		t.Fatalf("create API enrollment student: %v", err)
	}
	enrolledAt := time.Date(2028, 4, 1, 0, 0, 0, 0, time.UTC)
	apiEnrollment, err := academicRepo.CreateStudentEnrollment(ctx, actorPrincipalID, schoolID,
		apiStudentID, apiYear.ID, apiSection.ID, nil, enrolledAt)
	if err != nil || apiEnrollment == nil || apiEnrollment.ClassNameSnapshot != apiClass.Name || apiEnrollment.SectionNameSnapshot != "B" {
		t.Fatalf("create enrollment through repository: enrollment=%+v err=%v", apiEnrollment, err)
	}
	if _, err := academicRepo.CreateStudentEnrollment(ctx, actorPrincipalID, schoolID,
		apiStudentID, apiYear.ID, apiSection.ID, nil, enrolledAt); err == nil {
		t.Fatal("expected duplicate student/year enrollment to be rejected")
	}
	nextYear, err := academicRepo.SaveAcademicYear(ctx, actorPrincipalID, schoolID, 0,
		"API Next Academic "+suffix, time.Date(2029, 4, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2030, 3, 31, 0, 0, 0, 0, time.UTC), "active", false)
	if err != nil { t.Fatalf("create next academic year for promotion: %v", err) }
	nextSection, err := academicRepo.SaveSection(ctx, actorPrincipalID, schoolID, 0,
		nextYear.ID, apiClass.ID, "C", 3, true)
	if err != nil { t.Fatalf("create next-year section for promotion: %v", err) }
	promotedAt := time.Date(2029, 4, 1, 0, 0, 0, 0, time.UTC)
	promoted, err := academicRepo.PromoteStudentEnrollment(ctx, actorPrincipalID, schoolID,
		apiEnrollment.ID, nextYear.ID, nextSection.ID, nil, promotedAt)
	if err != nil || promoted == nil || promoted.AcademicYearID != nextYear.ID || promoted.SectionNameSnapshot != "C" {
		t.Fatalf("promote enrollment through repository: enrollment=%+v err=%v", promoted, err)
	}
	history, err := academicRepo.ListStudentEnrollments(ctx, actorPrincipalID, schoolID, 0, apiStudentID)
	if err != nil || len(history) != 2 {
		t.Fatalf("list student enrollment history: items=%+v err=%v", history, err)
	}
	var priorStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM student_enrollments WHERE id=$1`, apiEnrollment.ID).Scan(&priorStatus); err != nil || priorStatus != "promoted" {
		t.Fatalf("prior enrollment status=%q err=%v; want promoted", priorStatus, err)
	}

	// A class-in-charge assignment must not preserve attendance authority after
	// the principal's system Teacher role has been revoked.
	if _, err := tx.Exec(ctx, `
		UPDATE principal_roles SET revoked_at = NOW()
		 WHERE principal_id = $1 AND role_id = $2 AND revoked_at IS NULL
	`, teacherPrincipalID, teacherRoleID); err != nil {
		t.Fatalf("revoke Teacher role fixture: %v", err)
	}
	attendanceAllowed, err := attendanceSectionAllowed(ctx, tx, teacherPrincipalID, schoolID, section2)
	if err != nil {
		t.Fatalf("check attendance after Teacher role revocation: %v", err)
	}
	if attendanceAllowed {
		t.Fatal("active class-in-charge assignment retained attendance access after Teacher role revocation")
	}
	rosterAfterTeacherRoleRevocation, err := teacherAcademicRepo.ListStudentEnrollments(
		ctx, teacherPrincipalID, schoolID, year2, studentID,
	)
	if err != nil && !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("unexpected roster result after Teacher role revocation: %v", err)
	}
	if err == nil && len(rosterAfterTeacherRoleRevocation) != 0 {
		t.Fatalf("revoked Teacher role retained roster access: %+v", rosterAfterTeacherRoleRevocation)
	}

}
