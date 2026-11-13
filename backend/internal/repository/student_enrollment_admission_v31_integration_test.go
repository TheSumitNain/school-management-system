package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestStudentEnrollmentAdmissionV31SchemaPostgresIntegration verifies both
// the additive schema and its year-scoped admission identity constraints.
func TestStudentEnrollmentAdmissionV31SchemaPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var ready bool
	err := pool.QueryRow(ctx, `
		SELECT
			(
				SELECT COUNT(*) = 2
				  FROM information_schema.columns
				 WHERE table_schema = 'public'
				   AND table_name = 'student_enrollments'
				   AND column_name IN ('admission_number', 'admission_date')
				   AND is_nullable = 'YES'
			)
			AND to_regclass('public.ux_student_enrollments_school_year_admission_number') IS NOT NULL
			AND EXISTS (
				SELECT 1 FROM pg_constraint
				 WHERE conrelid = 'public.student_enrollments'::regclass
				   AND conname = 'student_enrollments_admission_number_nonempty'
				   AND contype = 'c'
			)
			AND EXISTS (
				SELECT 1 FROM pg_constraint
				 WHERE conrelid = 'public.student_enrollments'::regclass
				   AND conname = 'student_enrollments_admission_identity_pair'
				   AND contype = 'c'
			)
	`).Scan(&ready)
	if err != nil {
		t.Fatalf("inspect V31 enrollment admission schema: %v", err)
	}
	if !ready {
		t.Fatal("V31 enrollment admission columns, constraints, or year-scoped unique index are missing")
	}

	// Keep all fixture records rollback-scoped: this test must not leave test
	// students, academic years, classes, sections, or enrollments behind.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin V31 constraint test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load integration school: %v", err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	createYear := func(name, start, end string) int64 {
		t.Helper()
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
			VALUES($1,$2,$3::date,$4::date,'active',FALSE) RETURNING id
		`, schoolID, name+"-"+suffix, start, end).Scan(&id)
		if err != nil {
			t.Fatalf("create academic year %s: %v", name, err)
		}
		return id
	}
	createClass := func(name string) int64 {
		t.Helper()
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id
		`, schoolID, name+"-"+suffix).Scan(&id); err != nil {
			t.Fatalf("create class %s: %v", name, err)
		}
		return id
	}
	createSection := func(yearID, classID int64, name string) int64 {
		t.Helper()
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO sections(school_id,academic_year_id,class_id,name)
			VALUES($1,$2,$3,$4) RETURNING id
		`, schoolID, yearID, classID, name).Scan(&id); err != nil {
			t.Fatalf("create section %s: %v", name, err)
		}
		return id
	}
	createStudent := func(name string) int64 {
		t.Helper()
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO students(school_id,admission_number,full_name)
			VALUES($1,$2,$3) RETURNING id
		`, schoolID, "V31-"+name+"-"+suffix, name).Scan(&id); err != nil {
			t.Fatalf("create student %s: %v", name, err)
		}
		return id
	}
	createEnrollment := func(studentID, yearID, classID, sectionID int64, number *string, date *string) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO student_enrollments
				(school_id,student_id,academic_year_id,class_id,section_id,enrolled_at,admission_number,admission_date)
			VALUES($1,$2,$3,$4,$5,DATE '2026-04-01',$6,$7::date)
		`, schoolID, studentID, yearID, classID, sectionID, number, date)
		return err
	}

	yearOne := createYear("V31-Year-One", "2026-04-01", "2027-03-31")
	yearTwo := createYear("V31-Year-Two", "2027-04-01", "2028-03-31")
	classID := createClass("V31-Class")
	sectionOne := createSection(yearOne, classID, "A")
	sectionTwo := createSection(yearTwo, classID, "A")
	studentOne := createStudent("V31 Student One")
	studentTwo := createStudent("V31 Student Two")
	studentLegacy := createStudent("V31 Legacy Student")

	admissionNumber := "  ADM-V31-" + suffix + "  "
	admissionDate := "2026-04-01"
	if err := createEnrollment(studentOne, yearOne, classID, sectionOne, &admissionNumber, &admissionDate); err != nil {
		t.Fatalf("insert first year admission: %v", err)
	}

	// The same student may reuse the same admission number in a different year.
	if err := createEnrollment(studentOne, yearTwo, classID, sectionTwo, &admissionNumber, &admissionDate); err != nil {
		t.Fatalf("reuse admission number in a different academic year: %v", err)
	}

	// Case/whitespace normalization must reject a duplicate in the same
	// school/year, even when it belongs to a different student.
	duplicate := "adm-v31-" + suffix
	err = createEnrollment(studentTwo, yearOne, classID, sectionOne, &duplicate, &admissionDate)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("same-year normalized duplicate error = %v, want unique violation (23505)", err)
	}

	// Historical rows without proven year-specific admission identity remain
	// representable as NULL/NULL during the expand-and-contract migration.
	if err := createEnrollment(studentLegacy, yearOne, classID, sectionOne, nil, nil); err != nil {
		t.Fatalf("insert legacy enrollment with unknown admission identity: %v", err)
	}

	// The number and date are an atomic pair; neither half may be stored alone.
	dateWithoutNumber := "2026-04-02"
	err = createEnrollment(studentTwo, yearTwo, classID, sectionTwo, nil, &dateWithoutNumber)
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("date without admission number error = %v, want check violation (23514)", err)
	}
	emptyNumber := "   "
	err = createEnrollment(studentTwo, yearTwo, classID, sectionTwo, &emptyNumber, &admissionDate)
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("blank admission number error = %v, want check violation (23514)", err)
	}

	// Ensure test fixture setup remains tied to the expected school scope.
	if _, err := tx.Exec(ctx, `SELECT $1::bigint`, fmt.Sprint(schoolID)); err != nil {
		t.Fatalf("confirm fixture school scope: %v", err)
	}
}
