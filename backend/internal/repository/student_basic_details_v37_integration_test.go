package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestStudentBasicDetailsV37PostgresIntegration verifies optional basic-profile
// storage, tenant boundaries, normalized registration-number uniqueness and
// the protected record lifecycle. Fixtures are rollback-scoped.
func TestStudentBasicDetailsV37PostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.student_basic_details') IS NOT NULL
		   AND to_regprocedure('public.guard_student_basic_details()') IS NOT NULL
	`).Scan(&ready); err != nil {
		t.Fatalf("inspect V37 basic student profile schema: %v", err)
	}
	if !ready {
		t.Fatal("V37 student basic details schema is missing")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V37 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, actorPrincipalID int64
	if err := tx.QueryRow(ctx, `SELECT school_id,principal_id FROM users WHERE id=$1`, actorUserID).
		Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load V37 school actor: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var studentID, secondStudentID int64
	for _, target := range []struct {
		name string
		out  *int64
	}{
		{"V37 Basic Student " + suffix, &studentID},
		{"V37 Second Basic Student " + suffix, &secondStudentID},
	} {
		if err := tx.QueryRow(ctx, `
			INSERT INTO students(school_id,admission_number,full_name)
			VALUES($1,NULL,$2) RETURNING id
		`, schoolID, target.name).Scan(target.out); err != nil {
			t.Fatalf("create V37 student fixture: %v", err)
		}
	}

	registration := "SR-" + suffix
	var detailID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_basic_details
			(school_id,student_id,date_of_birth,place_of_birth,gender,nationality,religion,
			 social_category,ews_status,school_registration_number,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,DATE '2018-04-12','Ambala','female','Indian','Not specified',
		       'general',FALSE,$3,$4,$4)
		RETURNING id
	`, schoolID, studentID, registration, actorPrincipalID).Scan(&detailID); err != nil {
		t.Fatalf("insert optional student basic details: %v", err)
	}
	var savedRegistration, gender string
	if err := tx.QueryRow(ctx, `
		SELECT school_registration_number,gender
		  FROM student_basic_details WHERE id=$1 AND school_id=$2
	`, detailID, schoolID).Scan(&savedRegistration, &gender); err != nil {
		t.Fatalf("read student basic details: %v", err)
	}
	if savedRegistration != registration || gender != "female" {
		t.Fatalf("unexpected basic details readback: registration=%q gender=%q", savedRegistration, gender)
	}

	assertRejected := func(label, query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, `SAVEPOINT v37_rejected_write`); err != nil {
			t.Fatalf("%s: create savepoint: %v", label, err)
		}
		_, writeErr := tx.Exec(ctx, query, args...)
		if writeErr == nil {
			_, _ = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v37_rejected_write`)
			_, _ = tx.Exec(ctx, `RELEASE SAVEPOINT v37_rejected_write`)
			t.Fatalf("%s: expected database rejection", label)
		}
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v37_rejected_write`); err != nil {
			t.Fatalf("%s: rollback savepoint: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT v37_rejected_write`); err != nil {
			t.Fatalf("%s: release savepoint: %v", label, err)
		}
	}
	assertRejected("normalized school registration duplicate",
		`INSERT INTO student_basic_details(school_id,student_id,school_registration_number,created_by_principal_id,updated_by_principal_id)
		 VALUES($1,$2,$3,$4,$4)`, schoolID, secondStudentID, "  "+strings.ToLower(registration)+"  ", actorPrincipalID)
	assertRejected("invalid gender",
		`INSERT INTO student_basic_details(school_id,student_id,gender,created_by_principal_id,updated_by_principal_id)
		 VALUES($1,$2,'unknown',$3,$3)`, schoolID, secondStudentID, actorPrincipalID)
	assertRejected("future date of birth",
		`INSERT INTO student_basic_details(school_id,student_id,date_of_birth,created_by_principal_id,updated_by_principal_id)
		 VALUES($1,$2,CURRENT_DATE + 1,$3,$3)`, schoolID, secondStudentID, actorPrincipalID)
	assertRejected("school registration number blank",
		`UPDATE student_basic_details SET school_registration_number='   ' WHERE id=$1`, detailID)
	assertRejected("basic details delete",
		`DELETE FROM student_basic_details WHERE id=$1`, detailID)
	assertRejected("basic details student identity rewrite",
		`UPDATE student_basic_details SET student_id=$2 WHERE id=$1`, detailID, secondStudentID)
	assertRejected("basic details creator rewrite",
		`UPDATE student_basic_details SET created_by_principal_id=updated_by_principal_id WHERE id=$1`, detailID)
}
