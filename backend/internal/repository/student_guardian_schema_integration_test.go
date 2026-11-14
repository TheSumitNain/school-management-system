package repository

import (
	"context"
	"strconv"
	"testing"
	"time"
)

// TestStudentGuardianFoundationPostgresIntegration verifies the student/guardian
// relational constraints, composite tenant boundaries, and V32 legacy admission semantics.
func TestStudentGuardianFoundationPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin guardian schema integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	expectRejected := func(query string, args ...any) error {
		if _, err := tx.Exec(ctx, `SAVEPOINT expected_guardian_rejection`); err != nil { t.Fatalf("create expected-rejection savepoint: %v", err) }
		_, writeErr := tx.Exec(ctx, query, args...)
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT expected_guardian_rejection`); err != nil { t.Fatalf("rollback expected-rejection savepoint: %v", err) }
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT expected_guardian_rejection`); err != nil { t.Fatalf("release expected-rejection savepoint: %v", err) }
		return writeErr
	}

	var hasV11Schema bool
	if err := tx.QueryRow(ctx,
		`SELECT to_regclass('public.student_guardians') IS NOT NULL`,
	).Scan(&hasV11Schema); err != nil {
		t.Fatalf("detect V11 schema: %v", err)
	}
	if !hasV11Schema {
		t.Fatal("V11 student/guardian schema is missing; apply migrations to school_management_test first")
	}

	var schoolID1 int64
	if err := tx.QueryRow(ctx,
		`SELECT school_id FROM users WHERE id = $1`, userID,
	).Scan(&schoolID1); err != nil {
		t.Fatalf("load helper school: %v", err)
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	var schoolID2 int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO schools (name, code, status)
		 VALUES ($1, $2, 'active') RETURNING id`,
		"V11 Cross-School Test "+suffix,
		"V11-"+suffix,
	).Scan(&schoolID2); err != nil {
		t.Fatalf("create second test school: %v", err)
	}

	// This cleanup runs before the shared helper removes its school/user.


	admission := "V11-" + suffix
	var student1, student2, otherSchoolStudent int64
	for _, target := range []struct {
		schoolID int64
		name     string
		out      *int64
	}{
		{schoolID1, "Student One", &student1},
		{schoolID1, "Student Two", &student2},
		{schoolID2, "Other School Student", &otherSchoolStudent},
	} {
		if err := tx.QueryRow(ctx,
			`INSERT INTO students (school_id, admission_number, full_name)
			 VALUES ($1, $2, $3) RETURNING id`,
			target.schoolID, admission+"-"+target.name, target.name,
		).Scan(target.out); err != nil {
			t.Fatalf("create student %q: %v", target.name, err)
		}
	}

	// Shared mobile details are allowed in school contact records. These
	// records are not authenticated parent identities.
	var guardian1, guardian2 int64
	for _, target := range []struct {
		name string
		out  *int64
	}{
		{"Guardian One", &guardian1},
		{"Guardian Two", &guardian2},
	} {
		if err := tx.QueryRow(ctx,
			`INSERT INTO guardians (school_id, full_name, mobile_number)
			 VALUES ($1, $2, $3) RETURNING id`,
			schoolID1, target.name, "+911234567890",
		).Scan(target.out); err != nil {
			t.Fatalf("create guardian %q: %v", target.name, err)
		}
	}

	// Many-to-many: one student may have multiple guardians, and one guardian
	// may be linked to multiple students in the same school.
	for _, link := range []struct {
		studentID  int64
		guardianID int64
		relation   string
	}{
		{student1, guardian1, "father"},
		{student1, guardian2, "mother"},
		{student2, guardian1, "father"},
	} {
		if _, err := tx.Exec(ctx,
			`INSERT INTO student_guardians
			 (school_id, student_id, guardian_id, relationship_type)
			 VALUES ($1, $2, $3, $4)`,
			schoolID1, link.studentID, link.guardianID, link.relation,
		); err != nil {
			t.Fatalf("create student-guardian link: %v", err)
		}
	}

	if err := expectRejected(`INSERT INTO student_guardians (school_id, student_id, guardian_id, relationship_type) VALUES ($1, $2, $3, 'father')`, schoolID1, student1, guardian1); err == nil {
		t.Fatal("expected duplicate student-guardian pair to be rejected")
	}

	if err := expectRejected(`INSERT INTO student_guardians (school_id, student_id, guardian_id, relationship_type) VALUES ($1, $2, $3, 'guardian')`, schoolID2, otherSchoolStudent, guardian1); err == nil {
		t.Fatal("expected cross-school student-guardian link to be rejected")
	}

	// V32 makes the student-level admission value legacy-only and non-unique.
	// Year-specific uniqueness is enforced by student_enrollments (V31).
	var duplicateLegacyStudentID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO students (school_id, admission_number, full_name)
		 VALUES ($1, $2, 'Duplicate Legacy Admission') RETURNING id`,
		schoolID1, "  "+admission+"-Student One  ",
	).Scan(&duplicateLegacyStudentID); err != nil {
		t.Fatalf("same legacy admission value should be allowed on another permanent student: %v", err)
	}

	if err := expectRejected(`UPDATE students SET school_id = $1 WHERE id = $2`, schoolID2, student1); err == nil {
		t.Fatal("expected student school ownership to be immutable")
	}
	if err := expectRejected(`UPDATE guardians SET school_id = $1 WHERE id = $2`, schoolID2, guardian1); err == nil {
		t.Fatal("expected guardian school ownership to be immutable")
	}

	var linkCount int
	if err := tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM student_guardians
		 WHERE school_id = $1 AND student_id IN ($2, $3)`,
		schoolID1, student1, student2,
	).Scan(&linkCount); err != nil {
		t.Fatalf("count student-guardian links: %v", err)
	}
	if linkCount != 3 {
		t.Fatalf("expected 3 valid student-guardian links, got %d", linkCount)
	}
}
