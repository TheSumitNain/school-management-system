package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type StudentBasicDetails struct {
	ID                     int64      `json:"id"`
	SchoolID               int64      `json:"school_id"`
	StudentID              int64      `json:"student_id"`
	DateOfBirth            *time.Time `json:"date_of_birth,omitempty"`
	PlaceOfBirth           *string    `json:"place_of_birth,omitempty"`
	Gender                 *string    `json:"gender,omitempty"`
	Nationality            *string    `json:"nationality,omitempty"`
	Religion               *string    `json:"religion,omitempty"`
	SocialCategory         *string    `json:"social_category,omitempty"`
	EWSStatus              *bool      `json:"ews_status,omitempty"`
	SchoolRegistrationNumber *string  `json:"school_registration_number,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func (r *AcademicRepository) GetStudentBasicDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
) (*StudentBasicDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`,
		studentID, schoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check student for basic details: %w", err)
	}
	if !exists {
		return nil, ErrAcademicNotFound
	}
	var item StudentBasicDetails
	err := r.db.QueryRow(ctx, `
		SELECT id,school_id,student_id,date_of_birth,place_of_birth,gender,nationality,
		       religion,social_category,ews_status,school_registration_number,created_at,updated_at
		  FROM student_basic_details WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.DateOfBirth, &item.PlaceOfBirth,
		&item.Gender, &item.Nationality, &item.Religion, &item.SocialCategory,
		&item.EWSStatus, &item.SchoolRegistrationNumber, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return &StudentBasicDetails{SchoolID: schoolID, StudentID: studentID}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load student basic details: %w", err)
	}
	return &item, nil
}

func (r *AcademicRepository) SaveStudentBasicDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
	dateOfBirth *time.Time, placeOfBirth, gender, nationality, religion, socialCategory *string,
	ewsStatus *bool, registrationNumber *string,
) (*StudentBasicDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	var err error
	if placeOfBirth, err = normalizeAddressText(placeOfBirth, 120); err != nil { return nil, err }
	if nationality, err = normalizeAddressText(nationality, 80); err != nil { return nil, err }
	if religion, err = normalizeAddressText(religion, 80); err != nil { return nil, err }
	if registrationNumber, err = normalizeAddressText(registrationNumber, 50); err != nil { return nil, err }
	if gender != nil {
		value := strings.ToLower(strings.TrimSpace(*gender))
		if value == "" { gender = nil } else if value != "female" && value != "male" && value != "other" && value != "not_specified" {
			return nil, ErrInvalidAcademicInput
		} else { gender = &value }
	}
	if socialCategory != nil {
		value := strings.ToLower(strings.TrimSpace(*socialCategory))
		if value == "" { socialCategory = nil } else if value != "general" && value != "obc" && value != "sc" && value != "st" && value != "other" && value != "not_specified" {
			return nil, ErrInvalidAcademicInput
		} else { socialCategory = &value }
	}
	if dateOfBirth != nil && (dateOfBirth.After(time.Now().UTC()) || dateOfBirth.Before(time.Now().UTC().AddDate(-130, 0, 0))) {
		return nil, ErrInvalidAcademicInput
	}

	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin student basic details save: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil { return nil, err }
	var active bool
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,
		studentID, schoolID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load student for basic details: %w", err)
	}
	if !active { return nil, ErrInvalidAcademicInput }

	if _, err := tx.Exec(ctx, `
		INSERT INTO student_basic_details
			(school_id,student_id,date_of_birth,place_of_birth,gender,nationality,religion,
			 social_category,ews_status,school_registration_number,created_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		ON CONFLICT(student_id) DO UPDATE SET
			date_of_birth=EXCLUDED.date_of_birth,
			place_of_birth=EXCLUDED.place_of_birth,
			gender=EXCLUDED.gender,
			nationality=EXCLUDED.nationality,
			religion=EXCLUDED.religion,
			social_category=EXCLUDED.social_category,
			ews_status=EXCLUDED.ews_status,
			school_registration_number=EXCLUDED.school_registration_number,
			updated_by_principal_id=EXCLUDED.updated_by_principal_id,
			updated_at=NOW()
	`, schoolID, studentID, dateOfBirth, placeOfBirth, gender, nationality, religion,
		socialCategory, ewsStatus, registrationNumber, actorID); err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		SELECT $1,$2,'school.student.basic_details.updated','student_basic_details',bd.id::text,'success',
		       jsonb_build_object(
		           'has_date_of_birth',bd.date_of_birth IS NOT NULL,
		           'has_place_of_birth',bd.place_of_birth IS NOT NULL,
		           'has_gender',bd.gender IS NOT NULL,
		           'has_nationality',bd.nationality IS NOT NULL,
		           'has_religion',bd.religion IS NOT NULL,
		           'has_social_category',bd.social_category IS NOT NULL,
		           'has_ews_status',bd.ews_status IS NOT NULL,
		           'has_school_registration_number',bd.school_registration_number IS NOT NULL
		       )
		  FROM student_basic_details bd WHERE bd.school_id=$2 AND bd.student_id=$3
	`, actorID, schoolID, studentID); err != nil {
		return nil, fmt.Errorf("audit student basic details: %w", err)
	}
	var item StudentBasicDetails
	if err := tx.QueryRow(ctx, `
		SELECT id,school_id,student_id,date_of_birth,place_of_birth,gender,nationality,
		       religion,social_category,ews_status,school_registration_number,created_at,updated_at
		  FROM student_basic_details WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.DateOfBirth, &item.PlaceOfBirth,
		&item.Gender, &item.Nationality, &item.Religion, &item.SocialCategory,
		&item.EWSStatus, &item.SchoolRegistrationNumber, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("read saved student basic details: %w", err)
	}
	if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student basic details: %w", err) }
	return &item, nil
}
