package repository

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type StudentContactDetails struct {
	ID             int64     `json:"id"`
	SchoolID       int64     `json:"school_id"`
	StudentID      int64     `json:"student_id"`
	PrimaryPhone   *string   `json:"primary_phone,omitempty"`
	AlternatePhone *string   `json:"alternate_phone,omitempty"`
	EmailAddress   *string   `json:"email_address,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (r *AcademicRepository) GetStudentContactDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
) (*StudentContactDetails, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if studentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`,
		studentID, schoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check student for contact details: %w", err)
	}
	if !exists {
		return nil, ErrAcademicNotFound
	}

	var item StudentContactDetails
	err := r.db.QueryRow(ctx, `
		SELECT id,school_id,student_id,primary_phone,alternate_phone,email_address,created_at,updated_at
		  FROM student_contact_details
		 WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.PrimaryPhone,
		&item.AlternatePhone, &item.EmailAddress, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &StudentContactDetails{SchoolID: schoolID, StudentID: studentID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load student contact details: %w", err)
	}
	return &item, nil
}

func (r *AcademicRepository) SaveStudentContactDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
	primaryPhone, alternatePhone, emailAddress *string,
) (*StudentContactDetails, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if studentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	var err error
	if primaryPhone, err = normalizeAddressText(primaryPhone, 30); err != nil {
		return nil, err
	}
	if alternatePhone, err = normalizeAddressText(alternatePhone, 30); err != nil {
		return nil, err
	}
	if emailAddress, err = normalizeAddressText(emailAddress, 254); err != nil {
		return nil, err
	}
	if emailAddress != nil {
		parsed, parseErr := mail.ParseAddress(*emailAddress)
		if parseErr != nil || parsed.Address != *emailAddress {
			return nil, ErrInvalidAcademicInput
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student contact details save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	var studentActive bool
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,
		studentID, schoolID).Scan(&studentActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load student for contact details: %w", err)
	}
	if !studentActive {
		return nil, ErrInvalidAcademicInput
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO student_contact_details
			(school_id,student_id,primary_phone,alternate_phone,email_address)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(student_id) DO UPDATE SET
			primary_phone=EXCLUDED.primary_phone,
			alternate_phone=EXCLUDED.alternate_phone,
			email_address=EXCLUDED.email_address,
			updated_at=NOW()
	`, schoolID, studentID, primaryPhone, alternatePhone, emailAddress); err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		SELECT $1,$2,'school.student.contact_details.updated','student_contact_details',cd.id::text,'success',
		       jsonb_build_object(
		           'has_primary_phone',cd.primary_phone IS NOT NULL,
		           'has_alternate_phone',cd.alternate_phone IS NOT NULL,
		           'has_email',cd.email_address IS NOT NULL
		       )
		  FROM student_contact_details cd
		 WHERE cd.school_id=$2 AND cd.student_id=$3
	`, actorID, schoolID, studentID); err != nil {
		return nil, fmt.Errorf("audit student contact details update: %w", err)
	}
	var item StudentContactDetails
	if err := tx.QueryRow(ctx, `
		SELECT id,school_id,student_id,primary_phone,alternate_phone,email_address,created_at,updated_at
		  FROM student_contact_details
		 WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.PrimaryPhone,
		&item.AlternatePhone, &item.EmailAddress, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("read saved student contact details: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student contact details: %w", err)
	}
	return &item, nil
}
