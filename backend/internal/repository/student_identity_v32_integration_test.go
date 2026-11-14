package repository

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestStudentIdentityAdmissionDecouplingV32PostgresIntegration verifies that
// permanent student identity no longer requires a school-wide admission
// number, while preserving existing legacy values and school ownership.
func TestStudentIdentityAdmissionDecouplingV32PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var nullable, legacyIndex bool
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT is_nullable = 'YES'
			   FROM information_schema.columns
			  WHERE table_schema = 'public'
			    AND table_name = 'students'
			    AND column_name = 'admission_number'),
			to_regclass('public.ix_students_school_legacy_admission_number') IS NOT NULL
			  AND to_regclass('public.ux_students_school_admission_number') IS NULL
	`).Scan(&nullable, &legacyIndex)
	if err != nil {
		t.Fatalf("inspect V32 student identity schema: %v", err)
	}
	if !nullable || !legacyIndex {
		t.Fatal("V32 must make legacy admission_number nullable and replace its unique index with a non-unique lookup index")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V32 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load integration school: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	// New permanent student identities do not need an admission number before
	// their first academic-year enrollment is created.
	var studentOne, studentTwo int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, NULL, $2) RETURNING id
	`, schoolID, "V32 New Student One "+suffix).Scan(&studentOne); err != nil {
		t.Fatalf("insert student without legacy admission number: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, NULL, $2) RETURNING id
	`, schoolID, "V32 New Student Two "+suffix).Scan(&studentTwo); err != nil {
		t.Fatalf("insert second student without legacy admission number: %v", err)
	}
	if studentOne == studentTwo {
		t.Fatal("student identities unexpectedly share an ID")
	}

	// Legacy values remain readable and are not rewritten by V32.
	legacyNumber := "V32-LEGACY-" + suffix
	var legacyStudent int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, legacyNumber, "V32 Legacy Student "+suffix).Scan(&legacyStudent); err != nil {
		t.Fatalf("insert legacy student admission number: %v", err)
	}
	var preserved string
	if err := tx.QueryRow(ctx, `SELECT admission_number FROM students WHERE id=$1`, legacyStudent).Scan(&preserved); err != nil {
		t.Fatalf("read preserved legacy admission number: %v", err)
	}
	if preserved != legacyNumber {
		t.Fatalf("legacy admission number = %q, want %q", preserved, legacyNumber)
	}
	var duplicateLegacyID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3) RETURNING id
	`, schoolID, "  "+legacyNumber+"  ", "V32 Duplicate Legacy Student "+suffix).Scan(&duplicateLegacyID); err != nil {
		t.Fatalf("same legacy admission value must be allowed after V32: %v", err)
	}
	if duplicateLegacyID == legacyStudent {
		t.Fatal("duplicate legacy student unexpectedly reused the original student ID")
	}
}
