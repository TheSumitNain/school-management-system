package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	var schoolID, actorPrincipalID int64
	if err := tx.QueryRow(ctx, `SELECT school_id,principal_id FROM users WHERE id=$1`, userID).Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load integration school actor: %v", err)
	}
	platformAdminPrincipalID := ensurePlatformAdminLifecycleActor(t, pool)
	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		  AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign School Admin fixture role: %v", err)
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

	repository := NewAcademicRepository(tx)
	contact, err := repository.GetStudentContactDetails(ctx, actorPrincipalID, schoolID, studentTwo)
	if err != nil || contact.StudentID != studentTwo || contact.PrimaryPhone == nil || *contact.PrimaryPhone != phone {
		t.Fatalf("read student contact details through repository: item=%+v err=%v", contact, err)
	}

	updatedPhone := "+91 98123 45678"
	updatedEmail := "updated-" + suffix + "@example.test"
	savedContact, err := repository.SaveStudentContactDetails(ctx, actorPrincipalID, schoolID, studentTwo,
		&updatedPhone, &phone, &updatedEmail)
	if err != nil {
		t.Fatalf("save student contact details through repository: %v", err)
	}
	if savedContact.StudentID != studentTwo || savedContact.PrimaryPhone == nil || *savedContact.PrimaryPhone != updatedPhone ||
		savedContact.AlternatePhone == nil || *savedContact.AlternatePhone != phone ||
		savedContact.EmailAddress == nil || *savedContact.EmailAddress != updatedEmail {
		t.Fatalf("unexpected saved student contact details: %+v", savedContact)
	}
	var updateAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.contact_details.updated'
		   AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(savedContact.ID)).Scan(&updateAuditCount); err != nil {
		t.Fatalf("inspect contact update audit: %v", err)
	}
	if updateAuditCount != 1 {
		t.Fatalf("contact update audit count = %d, want 1", updateAuditCount)
	}
	var readAuditCount int
	var readAuditMetadata string
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),COALESCE(string_agg(metadata::text,' '),'')
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.contact_details.read'
		   AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(studentTwo)).Scan(&readAuditCount, &readAuditMetadata); err != nil {
		t.Fatalf("inspect contact read audit: %v", err)
	}
	if readAuditCount != 1 || !strings.Contains(readAuditMetadata, "contact_details") ||
		strings.Contains(readAuditMetadata, phone) || strings.Contains(readAuditMetadata, email) {
		t.Fatalf("contact read audit must contain section only: count=%d metadata=%s", readAuditCount, readAuditMetadata)
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
