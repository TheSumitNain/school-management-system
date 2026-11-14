package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestStudentContactDetailsV35PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.student_contact_details') IS NOT NULL
		   AND EXISTS (
		       SELECT 1 FROM information_schema.columns
		        WHERE table_schema='public' AND table_name='student_contact_details'
		          AND column_name='email_address' AND is_nullable='YES'
		   )
	`).Scan(&ready); err != nil {
		t.Fatalf("inspect V35 contact schema: %v", err)
	}
	if !ready {
		t.Fatal("V35 student contact details schema is missing")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V35 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID int64
	if err := tx.QueryRow(ctx, `SELECT school_id FROM users WHERE id=$1`, userID).Scan(&schoolID); err != nil {
		t.Fatalf("load integration school: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var studentOne, studentTwo int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING id
	`, schoolID, "V35 Contact One "+suffix).Scan(&studentOne); err != nil {
		t.Fatalf("create V35 student one: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING id
	`, schoolID, "V35 Contact Two "+suffix).Scan(&studentTwo); err != nil {
		t.Fatalf("create V35 student two: %v", err)
	}

	phone := "+91 98765 43210"
	email := "student-" + suffix + "@example.test"
	var contactID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_contact_details(school_id,student_id,primary_phone,email_address)
		VALUES($1,$2,$3,$4) RETURNING id
	`, schoolID, studentOne, phone, email).Scan(&contactID); err != nil {
		t.Fatalf("create V35 student contact: %v", err)
	}
	// Contact values are not unique; family members may share one number.
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_contact_details(school_id,student_id,primary_phone)
		VALUES($1,$2,$3)
	`, schoolID, studentTwo, phone); err != nil {
		t.Fatalf("allow shared household contact number: %v", err)
	}

	assertRejected := func(label, query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, `SAVEPOINT v35_rejected_write`); err != nil {
			t.Fatalf("%s: create savepoint: %v", label, err)
		}
		_, writeErr := tx.Exec(ctx, query, args...)
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v35_rejected_write`); err != nil {
			t.Fatalf("%s: rollback savepoint: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT v35_rejected_write`); err != nil {
			t.Fatalf("%s: release savepoint: %v", label, err)
		}
		var pgErr *pgconn.PgError
		if !errors.As(writeErr, &pgErr) {
			t.Fatalf("%s error = %v, want PostgreSQL constraint/trigger error", label, writeErr)
		}
	}
	assertRejected("duplicate contact row for student",
		`INSERT INTO student_contact_details(school_id,student_id) VALUES($1,$2)`, schoolID, studentOne)
	assertRejected("change contact student identity",
		`UPDATE student_contact_details SET student_id=$2 WHERE id=$1`, contactID, studentTwo)
	assertRejected("delete contact history",
		`DELETE FROM student_contact_details WHERE id=$1`, contactID)
	assertRejected("empty primary phone",
		`UPDATE student_contact_details SET primary_phone='   ' WHERE id=$1`, contactID)
}
