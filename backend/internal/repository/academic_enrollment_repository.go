package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type StudentEnrollment struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	StudentID int64 `json:"student_id"`
	StudentName string `json:"student_name,omitempty"`
	AcademicYearID int64 `json:"academic_year_id"`
	ClassID int64 `json:"class_id"`
	SectionID int64 `json:"section_id"`
	RollNumber *string `json:"roll_number,omitempty"`
	ClassNameSnapshot string `json:"class_name_snapshot"`
	SectionNameSnapshot string `json:"section_name_snapshot"`
	Status string `json:"status"`
	EnrolledAt time.Time `json:"enrolled_at"`
	EndedAt *time.Time `json:"ended_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func scanStudentEnrollment(row interface{ Scan(...any) error }) (*StudentEnrollment, error) {
	var v StudentEnrollment
	err := row.Scan(&v.ID, &v.SchoolID, &v.StudentID, &v.AcademicYearID, &v.ClassID,
		&v.SectionID, &v.RollNumber, &v.ClassNameSnapshot, &v.SectionNameSnapshot,
		&v.Status, &v.EnrolledAt, &v.EndedAt, &v.CreatedAt, &v.UpdatedAt)
	if err != nil { return nil, err }
	return &v, nil
}

func scanStudentEnrollmentWithName(row interface{ Scan(...any) error }) (*StudentEnrollment, error) {
	var v StudentEnrollment
	err := row.Scan(&v.ID, &v.SchoolID, &v.StudentID, &v.AcademicYearID, &v.ClassID,
		&v.SectionID, &v.RollNumber, &v.ClassNameSnapshot, &v.SectionNameSnapshot,
		&v.Status, &v.EnrolledAt, &v.EndedAt, &v.CreatedAt, &v.UpdatedAt, &v.StudentName)
	if err != nil { return nil, err }
	return &v, nil
}

const enrollmentColumns = `id,school_id,student_id,academic_year_id,class_id,section_id,roll_number,
	class_name_snapshot,section_name_snapshot,status,enrolled_at,ended_at,created_at,updated_at`

func (r *AcademicRepository) ListStudentEnrollments(ctx context.Context, actorID, schoolID, yearID, studentID int64) ([]StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if yearID < 0 || studentID < 0 { return nil, ErrInvalidAcademicInput }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_basic"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `SELECT e.id,e.school_id,e.student_id,e.academic_year_id,e.class_id,e.section_id,e.roll_number,
		e.class_name_snapshot,e.section_name_snapshot,e.status,e.enrolled_at,e.ended_at,e.created_at,e.updated_at,st.full_name
		FROM student_enrollments e
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		WHERE e.school_id=$1 AND ($2=0 OR e.academic_year_id=$2) AND ($3=0 OR e.student_id=$3)
		  AND (
			EXISTS (
				SELECT 1
				FROM principal_roles pr
				JOIN roles r ON r.id=pr.role_id
				WHERE pr.principal_id=$4 AND pr.revoked_at IS NULL
				  AND r.school_id=$1 AND r.scope_type='school'
				  AND r.principal_type='staff' AND r.role_key='school_admin'
				  AND r.is_system=TRUE
			)
			OR EXISTS (
				SELECT 1 FROM class_incharge_assignments cia
				WHERE cia.school_id=e.school_id
				  AND cia.academic_year_id=e.academic_year_id
				  AND cia.section_id=e.section_id
				  AND cia.teacher_principal_id=$4
				  AND cia.revoked_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM teacher_subject_assignments tsa
				WHERE tsa.school_id=e.school_id
				  AND tsa.academic_year_id=e.academic_year_id
				  AND tsa.section_id=e.section_id
				  AND tsa.teacher_principal_id=$4
				  AND tsa.revoked_at IS NULL
			)
		  )
		ORDER BY e.enrolled_at DESC,e.id DESC`, schoolID, yearID, studentID, actorID)
	if err != nil { return nil, fmt.Errorf("list student enrollments: %w", err) }
	defer rows.Close()
	items := make([]StudentEnrollment, 0)
	for rows.Next() {
		v, scanErr := scanStudentEnrollmentWithName(rows)
		if scanErr != nil { return nil, fmt.Errorf("scan student enrollment: %w", scanErr) }
		items = append(items, *v)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate student enrollments: %w", err) }
	if studentID > 0 && len(items) == 0 {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`, studentID, schoolID).Scan(&exists); err != nil { return nil, fmt.Errorf("check student: %w", err) }
		if !exists { return nil, ErrAcademicNotFound }
	}
	return items, nil
}

func (r *AcademicRepository) CreateStudentEnrollment(ctx context.Context, actorID, schoolID, studentID, yearID, sectionID int64, rollNumber *string, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if studentID <= 0 || yearID <= 0 || sectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	if rollNumber != nil {
		value := strings.TrimSpace(*rollNumber)
		if value == "" { rollNumber = nil } else if len(value) > 30 { return nil, ErrInvalidAcademicInput } else { rollNumber = &value }
	}
	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin student enrollment: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "students.create"); err != nil { return nil, err }
	var studentActive, yearActive bool
	if err = tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`, studentID, schoolID).Scan(&studentActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load student for enrollment: %w", err)
	}
	if !studentActive { return nil, ErrInvalidAcademicInput }
	if err = tx.QueryRow(ctx, `SELECT status='active' FROM academic_years WHERE id=$1 AND school_id=$2`, yearID, schoolID).Scan(&yearActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE`, sectionID, schoolID, yearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment section: %w", err)
	}
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+enrollmentColumns,
		schoolID, studentID, yearID, classID, sectionID, rollNumber, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrolled','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'academic_year_id',$5::bigint,'class_id',$6::bigint,'section_id',$7::bigint))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, yearID, classID, sectionID); err != nil { return nil, fmt.Errorf("audit student enrollment: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student enrollment: %w", err) }
	return v, nil
}

func (r *AcademicRepository) PromoteStudentEnrollment(ctx context.Context, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID int64, rollNumber *string, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if sourceEnrollmentID <= 0 || destinationYearID <= 0 || destinationSectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	if rollNumber != nil {
		value := strings.TrimSpace(*rollNumber)
		if value == "" { rollNumber = nil } else if len(value) > 30 { return nil, ErrInvalidAcademicInput } else { rollNumber = &value }
	}
	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin student promotion: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil { return nil, err }
	var studentID, sourceYearID int64
	var sourceEnrolledAt time.Time
	err = tx.QueryRow(ctx, `SELECT student_id,academic_year_id,enrolled_at FROM student_enrollments
		WHERE id=$1 AND school_id=$2 AND status='active' FOR UPDATE`, sourceEnrollmentID, schoolID).Scan(&studentID, &sourceYearID, &sourceEnrolledAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("lock source enrollment: %w", err)
	}
	if sourceYearID == destinationYearID || enrolledAt.Before(sourceEnrolledAt) { return nil, ErrInvalidAcademicInput }
	var yearActive bool
	if err = tx.QueryRow(ctx, `SELECT status='active' FROM academic_years WHERE id=$1 AND school_id=$2`, destinationYearID, schoolID).Scan(&yearActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE`, destinationSectionID, schoolID, destinationYearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination section: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE student_enrollments SET status='promoted',ended_at=$3,updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND status='active'`, sourceEnrollmentID, schoolID, enrolledAt); err != nil { return nil, academicWriteError(err) }
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+enrollmentColumns,
		schoolID, studentID, destinationYearID, classID, destinationSectionID, rollNumber, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.promoted','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'source_enrollment_id',$5::bigint,'source_academic_year_id',$6::bigint,'destination_academic_year_id',$7::bigint,'destination_class_id',$8::bigint,'destination_section_id',$9::bigint))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, sourceEnrollmentID, sourceYearID, destinationYearID, classID, destinationSectionID); err != nil { return nil, fmt.Errorf("audit student promotion: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student promotion: %w", err) }
	return v, nil
}
