package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// StudentProfile contains the permanent student identity and the admission
// number presented for the current academic year. The number is sourced from
// the current-year enrollment; a legacy student-level value is used only when
// the student has no current-year enrollment.
type StudentProfile struct {
	ID              int64     `json:"id"`
	SchoolID        int64     `json:"school_id"`
	AdmissionNumber string    `json:"admission_number"`
	FullName        string    `json:"full_name"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

const studentProfileColumns = `id, school_id, COALESCE(admission_number, ''), full_name, status, created_at, updated_at`

func scanStudentProfile(row interface{ Scan(...any) error }) (*StudentProfile, error) {
	var student StudentProfile
	if err := row.Scan(
		&student.ID, &student.SchoolID, &student.AdmissionNumber,
		&student.FullName, &student.Status, &student.CreatedAt, &student.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &student, nil
}

// ListStudents returns a bounded, school-scoped student directory. Search is
// limited to admission number and student name; guardian/contact data is not
// joined into this basic listing.
func (r *AcademicRepository) ListStudents(
	ctx context.Context, actorID, schoolID int64, search, status string, limit, offset int,
) ([]StudentProfile, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, ErrAcademicAccessDenied
	}
	search = strings.TrimSpace(search)
	status = strings.TrimSpace(strings.ToLower(status))
	if len(search) > 100 || (status != "" && status != "active" && status != "inactive" && status != "archived") ||
		limit < 1 || limit > 100 || offset < 0 {
		return nil, 0, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("begin student directory read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM students st
		LEFT JOIN LATERAL (
			SELECT e.id, e.admission_number
			FROM student_enrollments e
			JOIN academic_years ay
			  ON ay.id=e.academic_year_id AND ay.school_id=e.school_id AND ay.is_current=TRUE
			WHERE e.school_id=st.school_id AND e.student_id=st.id
			ORDER BY e.id DESC
			LIMIT 1
		) current_enrollment ON TRUE
		WHERE st.school_id = $1
		  AND ($2 = '' OR st.status = $2)
		  AND ($3 = '' OR
		       COALESCE(current_enrollment.admission_number,
		                CASE WHEN current_enrollment.id IS NULL THEN st.admission_number END, '')
		         ILIKE '%' || $3 || '%' OR st.full_name ILIKE '%' || $3 || '%')
	`, schoolID, status, search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count students: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT st.id, st.school_id,
		       COALESCE(current_enrollment.admission_number,
		                CASE WHEN current_enrollment.id IS NULL THEN st.admission_number END, ''),
		       st.full_name, st.status, st.created_at, st.updated_at
		FROM students st
		LEFT JOIN LATERAL (
			SELECT e.id, e.admission_number
			FROM student_enrollments e
			JOIN academic_years ay
			  ON ay.id=e.academic_year_id AND ay.school_id=e.school_id AND ay.is_current=TRUE
			WHERE e.school_id=st.school_id AND e.student_id=st.id
			ORDER BY e.id DESC
			LIMIT 1
		) current_enrollment ON TRUE
		WHERE st.school_id = $1
		  AND ($2 = '' OR st.status = $2)
		  AND ($3 = '' OR
		       COALESCE(current_enrollment.admission_number,
		                CASE WHEN current_enrollment.id IS NULL THEN st.admission_number END, '')
		         ILIKE '%' || $3 || '%' OR st.full_name ILIKE '%' || $3 || '%')
		ORDER BY LOWER(st.full_name), st.id
		LIMIT $4 OFFSET $5
	`, schoolID, status, search, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list students: %w", err)
	}
	items := make([]StudentProfile, 0)
	for rows.Next() {
		student, scanErr := scanStudentProfile(rows)
		if scanErr != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("scan student: %w", scanErr)
		}
		items = append(items, *student)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, fmt.Errorf("iterate students: %w", err)
	}
	rows.Close()
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.directory.read','student_collection',$3,'success',
		       jsonb_build_object('section','student_directory','result_count',$4::integer))
	`, actorID, schoolID, fmt.Sprint(schoolID), len(items)); err != nil {
		return nil, 0, fmt.Errorf("audit student directory read: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("commit student directory read audit: %w", err)
	}
	return items, total, nil
}

func (r *AcademicRepository) GetStudent(
	ctx context.Context, actorID, schoolID, studentID int64,
) (*StudentProfile, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if studentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student profile read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	student, err := scanStudentProfile(tx.QueryRow(ctx, `
		SELECT st.id, st.school_id,
		       COALESCE(current_enrollment.admission_number,
		                CASE WHEN current_enrollment.id IS NULL THEN st.admission_number END, ''),
		       st.full_name, st.status, st.created_at, st.updated_at
		FROM students st
		LEFT JOIN LATERAL (
			SELECT e.id, e.admission_number
			FROM student_enrollments e
			JOIN academic_years ay
			  ON ay.id=e.academic_year_id AND ay.school_id=e.school_id AND ay.is_current=TRUE
			WHERE e.school_id=st.school_id AND e.student_id=st.id
			ORDER BY e.id DESC
			LIMIT 1
		) current_enrollment ON TRUE
		WHERE st.id = $1 AND st.school_id = $2
		`, studentID, schoolID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAcademicNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get student: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.profile.read','student',$3,'success',
		       jsonb_build_object('section','student_profile'))
	`, actorID, schoolID, fmt.Sprint(student.ID)); err != nil {
		return nil, fmt.Errorf("audit student profile read: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student profile read audit: %w", err)
	}
	return student, nil
}

func (r *AcademicRepository) CreateStudent(
	ctx context.Context, actorID, schoolID int64, admissionNumber, fullName string,
) (*StudentProfile, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	admissionNumber = strings.TrimSpace(admissionNumber)
	fullName = strings.TrimSpace(fullName)
	if admissionNumber == "" || len(admissionNumber) > 50 || fullName == "" || len(fullName) > 200 {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student admission: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.create"); err != nil {
		return nil, err
	}
	student, err := scanStudentProfile(tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, $3)
		RETURNING `+studentProfileColumns,
		schoolID, admissionNumber, fullName,
	))
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.student.created', 'student', $3, 'success',
			jsonb_build_object('admission_number', $4::text))
	`, actorID, schoolID, fmt.Sprint(student.ID), student.AdmissionNumber); err != nil {
		return nil, fmt.Errorf("audit student admission: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student admission: %w", err)
	}
	return student, nil
}

func (r *AcademicRepository) UpdateStudent(
	ctx context.Context, actorID, schoolID, studentID int64, fullName string,
) (*StudentProfile, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	fullName = strings.TrimSpace(fullName)
	if studentID <= 0 || fullName == "" || len(fullName) > 200 {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	student, err := scanStudentProfile(tx.QueryRow(ctx, `
		UPDATE students SET full_name = $3, updated_at = NOW()
		WHERE id = $1 AND school_id = $2 AND status <> 'archived'
		RETURNING `+studentProfileColumns,
		studentID, schoolID, fullName,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAcademicNotFound
	}
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.student.updated', 'student', $3, 'success', '{}'::jsonb)
	`, actorID, schoolID, fmt.Sprint(student.ID)); err != nil {
		return nil, fmt.Errorf("audit student update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student update: %w", err)
	}
	return student, nil
}

func (r *AcademicRepository) ArchiveStudent(
	ctx context.Context, actorID, schoolID, studentID int64,
) (*StudentProfile, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if studentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student archive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.archive"); err != nil {
		return nil, err
	}
	student, err := scanStudentProfile(tx.QueryRow(ctx, `
		UPDATE students SET status = 'archived', updated_at = NOW()
		WHERE id = $1 AND school_id = $2 AND status <> 'archived'
		RETURNING `+studentProfileColumns,
		studentID, schoolID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAcademicNotFound
	}
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata)
		VALUES ($1, $2, 'school.student.archived', 'student', $3, 'success', '{}'::jsonb)
	`, actorID, schoolID, fmt.Sprint(student.ID)); err != nil {
		return nil, fmt.Errorf("audit student archive: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student archive: %w", err)
	}
	return student, nil
}
