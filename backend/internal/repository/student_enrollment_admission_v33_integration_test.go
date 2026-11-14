package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestEnrollmentAdmissionIdentityImmutableV33PostgresIntegration verifies
// admission number/date are immutable even while an enrollment is active,
// while the supported active -> promoted transition remains possible.
func TestEnrollmentAdmissionIdentityImmutableV33PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V33 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load integration school: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var yearID, classID, sectionID, studentID, enrollmentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active',FALSE) RETURNING id
	`, schoolID, "V33-Year-"+suffix).Scan(&yearID); err != nil {
		t.Fatalf("create V33 academic year: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id
	`, schoolID, "V33-Class-"+suffix).Scan(&classID); err != nil {
		t.Fatalf("create V33 class: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,'A') RETURNING id
	`, schoolID, yearID, classID).Scan(&sectionID); err != nil {
		t.Fatalf("create V33 section: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING id
	`, schoolID, "V33 Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create V33 student: %v", err)
	}
	admissionNumber := "V33-ADM-" + suffix
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollments
			(school_id,student_id,academic_year_id,class_id,section_id,enrolled_at,admission_number,admission_date)
		VALUES($1,$2,$3,$4,$5,DATE '2026-04-01',$6,DATE '2026-04-01') RETURNING id
	`, schoolID, studentID, yearID, classID, sectionID, admissionNumber).Scan(&enrollmentID); err != nil {
		t.Fatalf("create V33 enrollment: %v", err)
	}

	assertRejectedMutation := func(label, query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, `SAVEPOINT v33_immutable_mutation`); err != nil {
			t.Fatalf("%s: create savepoint: %v", label, err)
		}
		_, mutationErr := tx.Exec(ctx, query, args...)
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v33_immutable_mutation`); err != nil {
			t.Fatalf("%s: rollback savepoint: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT v33_immutable_mutation`); err != nil {
			t.Fatalf("%s: release savepoint: %v", label, err)
		}
		var pgErr *pgconn.PgError
		if !errors.As(mutationErr, &pgErr) || pgErr.Code != "P0001" {
			t.Fatalf("%s error = %v, want immutable-history exception (P0001)", label, mutationErr)
		}
	}

	assertRejectedMutation("mutate active admission number",
		`UPDATE student_enrollments SET admission_number='V33-MUTATED' WHERE id=$1`, enrollmentID)
	assertRejectedMutation("mutate active admission date",
		`UPDATE student_enrollments SET admission_date=DATE '2026-04-02' WHERE id=$1`, enrollmentID)

	// Closing an enrollment is still allowed when its identity is unchanged.
	if _, err := tx.Exec(ctx, `
		UPDATE student_enrollments SET status='promoted',ended_at=DATE '2027-03-31' WHERE id=$1
	`, enrollmentID); err != nil {
		t.Fatalf("close V33 enrollment without rewriting identity: %v", err)
	}
	var storedNumber string
	var storedDate time.Time
	if err := tx.QueryRow(ctx, `
		SELECT admission_number,admission_date FROM student_enrollments WHERE id=$1
	`, enrollmentID).Scan(&storedNumber, &storedDate); err != nil {
		t.Fatalf("read immutable V33 admission identity: %v", err)
	}
	if storedNumber != admissionNumber || storedDate.Format("2006-01-02") != "2026-04-01" {
		t.Fatalf("stored admission identity changed: number=%q date=%s", storedNumber, storedDate.Format("2006-01-02"))
	}
}
