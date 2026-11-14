package repository

import (
	"context"
	"encoding/json"
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
	AdmissionNumber *string `json:"admission_number,omitempty"`
	AdmissionDate *time.Time `json:"admission_date,omitempty"`
	Status string `json:"status"`
	EnrolledAt time.Time `json:"enrolled_at"`
	EndedAt *time.Time `json:"ended_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	basicRosterOnly bool
}

// MarshalJSON limits subject-only teachers to the approved minimal roster fields.
// Administrators and class-in-charge teachers retain the complete enrollment view.
func (v StudentEnrollment) MarshalJSON() ([]byte, error) {
	if v.basicRosterOnly {
		return json.Marshal(struct {
			StudentName string `json:"student_name"`
			RollNumber *string `json:"roll_number,omitempty"`
			ClassNameSnapshot string `json:"class_name_snapshot"`
			SectionNameSnapshot string `json:"section_name_snapshot"`
		}{v.StudentName, v.RollNumber, v.ClassNameSnapshot, v.SectionNameSnapshot})
	}
	type enrollmentJSON StudentEnrollment
	return json.Marshal(enrollmentJSON(v))
}

func scanStudentEnrollment(row interface{ Scan(...any) error }) (*StudentEnrollment, error) {
	var v StudentEnrollment
	err := row.Scan(&v.ID, &v.SchoolID, &v.StudentID, &v.AcademicYearID, &v.ClassID,
		&v.SectionID, &v.RollNumber, &v.ClassNameSnapshot, &v.SectionNameSnapshot,
		&v.AdmissionNumber, &v.AdmissionDate, &v.Status, &v.EnrolledAt, &v.EndedAt, &v.CreatedAt, &v.UpdatedAt)
	if err != nil { return nil, err }
	return &v, nil
}

func scanStudentEnrollmentWithName(row interface{ Scan(...any) error }) (*StudentEnrollment, error) {
	var v StudentEnrollment
	var isSchoolAdmin, isClassIncharge bool
	err := row.Scan(&v.ID, &v.SchoolID, &v.StudentID, &v.AcademicYearID, &v.ClassID,
		&v.SectionID, &v.RollNumber, &v.ClassNameSnapshot, &v.SectionNameSnapshot,
		&v.AdmissionNumber, &v.AdmissionDate, &v.Status, &v.EnrolledAt, &v.EndedAt, &v.CreatedAt, &v.UpdatedAt, &v.StudentName,
		&isSchoolAdmin, &isClassIncharge)
	if err != nil { return nil, err }
	v.basicRosterOnly = !isSchoolAdmin && !isClassIncharge
	return &v, nil
}

// nextAvailableSectionRollNumber assigns a monotonically increasing numeric
// roll number within an active section. Callers must hold a FOR UPDATE lock on
// the destination section row before invoking it so concurrent admissions
// cannot calculate the same value.
func nextAvailableSectionRollNumber(ctx context.Context, tx pgx.Tx, schoolID, sectionID int64) (*string, error) {
	var value string
	err := tx.QueryRow(ctx, `SELECT (COALESCE(MAX(CASE WHEN btrim(roll_number) ~ '^[0-9]+$'
			THEN btrim(roll_number)::numeric END), 0) + 1)::text
		FROM student_enrollments
		WHERE school_id=$1 AND section_id=$2 AND status='active'`, schoolID, sectionID).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("allocate next section roll number: %w", err)
	}
	if len(value) > 30 {
		return nil, ErrInvalidAcademicInput
	}
	return &value, nil
}

type StudentAdmissionResult struct {
	Student StudentProfile `json:"student"`
	Enrollment StudentEnrollment `json:"enrollment"`
}

func normalizeEnrollmentAdmission(number string, admissionDate time.Time) (string, error) {
	number = strings.TrimSpace(number)
	if number == "" || len(number) > 50 || admissionDate.IsZero() {
		return "", ErrInvalidAcademicInput
	}
	return number, nil
}

func validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn time.Time) error {
	within := func(value time.Time) bool {
		date := value.Format("2006-01-02")
		return date >= yearStartsOn.Format("2006-01-02") && date <= yearEndsOn.Format("2006-01-02")
	}
	if !within(admissionDate) || !within(enrolledAt) {
		return ErrInvalidAcademicInput
	}
	return nil
}

// CreateStudentAdmission creates the permanent student identity and its first
// academic-year enrollment atomically. The legacy student-level admission
// number is deliberately NULL; the enrollment is the source of truth.
func (r *AcademicRepository) CreateStudentAdmission(
	ctx context.Context,
	actorID, schoolID int64,
	fullName, admissionNumber string,
	yearID, sectionID int64,
	rollNumber *string,
	admissionDate time.Time,
) (*StudentAdmissionResult, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	fullName = strings.TrimSpace(fullName)
	if fullName == "" || len(fullName) > 200 || yearID <= 0 || sectionID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil {
		return nil, err
	}
	if rollNumber != nil {
		value := strings.TrimSpace(*rollNumber)
		if value == "" {
			rollNumber = nil
		} else if len(value) > 30 {
			return nil, ErrInvalidAcademicInput
		} else {
			rollNumber = &value
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin combined student admission: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "students.create"); err != nil {
		return nil, err
	}
	var yearActive bool
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on
		FROM academic_years WHERE id=$1 AND school_id=$2`, yearID, schoolID).
		Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load first enrollment academic year: %w", err)
	}
	if !yearActive {
		return nil, ErrInvalidAcademicInput
	}
	if err = validateEnrollmentDateRange(admissionDate, admissionDate, yearStartsOn, yearEndsOn); err != nil {
		return nil, err
	}
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections
		WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE
		FOR UPDATE`, sectionID, schoolID, yearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load first enrollment section: %w", err)
	}
	if rollNumber == nil {
		rollNumber, err = nextAvailableSectionRollNumber(ctx, tx, schoolID, sectionID)
		if err != nil {
			return nil, err
		}
	}

	student, err := scanStudentProfile(tx.QueryRow(ctx, `INSERT INTO students
		(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING `+studentProfileColumns, schoolID, fullName))
	if err != nil {
		return nil, academicWriteError(err)
	}
	// The API's admission_number is the first enrollment's year-scoped value;
	// the legacy students column intentionally remains NULL.
	student.AdmissionNumber = admissionNumber
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events
		(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.created','student',$3,'success',
		jsonb_build_object('initial_academic_year_id',$4::bigint))`,
		actorID, schoolID, fmt.Sprint(student.ID), yearID); err != nil {
		return nil, fmt.Errorf("audit student creation: %w", err)
	}

	enrollment, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING `+enrollmentColumns,
		schoolID, student.ID, yearID, classID, sectionID, rollNumber, admissionNumber, admissionDate))
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events
		(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrolled','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'academic_year_id',$5::bigint,
		'class_id',$6::bigint,'section_id',$7::bigint,'admission_number',$8::text,'admission_date',$9::date))`,
		actorID, schoolID, fmt.Sprint(enrollment.ID), student.ID, yearID, classID, sectionID,
		admissionNumber, admissionDate.Format("2006-01-02")); err != nil {
		return nil, fmt.Errorf("audit first student enrollment: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit combined student admission: %w", err)
	}
	return &StudentAdmissionResult{Student: *student, Enrollment: *enrollment}, nil
}

func (r *AcademicRepository) ListStudentEnrollments(ctx context.Context, actorID, schoolID, yearID, studentID int64) ([]StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if yearID < 0 || studentID < 0 { return nil, ErrInvalidAcademicInput }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_basic"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `SELECT e.id,e.school_id,e.student_id,e.academic_year_id,e.class_id,e.section_id,e.roll_number,
		e.class_name_snapshot,e.section_name_snapshot,e.admission_number,e.admission_date,e.status,e.enrolled_at,e.ended_at,e.created_at,e.updated_at,st.full_name,
		EXISTS (SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
		 WHERE pr.principal_id=$4 AND pr.revoked_at IS NULL AND r.school_id=e.school_id
		 AND r.scope_type='school' AND r.principal_type='staff' AND r.role_key='school_admin' AND r.is_system=TRUE),
		EXISTS (SELECT 1 FROM class_incharge_assignments cia
		 WHERE cia.school_id=e.school_id AND cia.academic_year_id=e.academic_year_id
		 AND cia.section_id=e.section_id AND cia.teacher_principal_id=$4 AND cia.revoked_at IS NULL)
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
			OR (
				EXISTS (
					SELECT 1 FROM principal_roles tpr
					JOIN roles tr ON tr.id=tpr.role_id
					WHERE tpr.principal_id=$4 AND tpr.revoked_at IS NULL
					  AND tr.school_id=e.school_id AND tr.scope_type='school'
					  AND tr.principal_type='staff' AND tr.role_key='teacher'
					  AND tr.is_system=TRUE
				)
				AND (
					EXISTS (
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

func (r *AcademicRepository) CreateStudentEnrollment(ctx context.Context, actorID, schoolID, studentID, yearID, sectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if studentID <= 0 || yearID <= 0 || sectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil { return nil, err }
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
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on FROM academic_years WHERE id=$1 AND school_id=$2`, yearID, schoolID).Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	if err = validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn); err != nil { return nil, err }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE FOR UPDATE`, sectionID, schoolID, yearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment section: %w", err)
	}
	if rollNumber == nil {
		rollNumber, err = nextAvailableSectionRollNumber(ctx, tx, schoolID, sectionID)
		if err != nil { return nil, err }
	}
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+enrollmentColumns,
		schoolID, studentID, yearID, classID, sectionID, rollNumber, admissionNumber, admissionDate, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrolled','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'academic_year_id',$5::bigint,'class_id',$6::bigint,'section_id',$7::bigint,'admission_number',$8::text,'admission_date',$9::date))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, yearID, classID, sectionID, admissionNumber, admissionDate.Format("2006-01-02")); err != nil { return nil, fmt.Errorf("audit student enrollment: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student enrollment: %w", err) }
	return v, nil
}

func (r *AcademicRepository) PromoteStudentEnrollment(ctx context.Context, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if sourceEnrollmentID <= 0 || destinationYearID <= 0 || destinationSectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil { return nil, err }
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
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on FROM academic_years WHERE id=$1 AND school_id=$2`, destinationYearID, schoolID).Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	if err = validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn); err != nil { return nil, err }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE FOR UPDATE`, destinationSectionID, schoolID, destinationYearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination section: %w", err)
	}
	if rollNumber == nil {
		rollNumber, err = nextAvailableSectionRollNumber(ctx, tx, schoolID, destinationSectionID)
		if err != nil { return nil, err }
	}
	if _, err = tx.Exec(ctx, `UPDATE student_enrollments SET status='promoted',ended_at=$3,updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND status='active'`, sourceEnrollmentID, schoolID, enrolledAt); err != nil { return nil, academicWriteError(err) }
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+enrollmentColumns,
		schoolID, studentID, destinationYearID, classID, destinationSectionID, rollNumber, admissionNumber, admissionDate, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.promoted','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'source_enrollment_id',$5::bigint,'source_academic_year_id',$6::bigint,'destination_academic_year_id',$7::bigint,'destination_class_id',$8::bigint,'destination_section_id',$9::bigint,'admission_number',$10::text,'admission_date',$11::date))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, sourceEnrollmentID, sourceYearID, destinationYearID, classID, destinationSectionID, admissionNumber, admissionDate.Format("2006-01-02")); err != nil { return nil, fmt.Errorf("audit student promotion: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student promotion: %w", err) }
	return v, nil
}
`, schoolID, sectionID).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("allocate next section roll number: %w", err)
	}
	if len(value) > 30 {
		return nil, ErrInvalidAcademicInput
	}
	return &value, nil
}

const enrollmentColumns = `id,school_id,student_id,academic_year_id,class_id,section_id,roll_number,
	class_name_snapshot,section_name_snapshot,admission_number,admission_date,status,enrolled_at,ended_at,created_at,updated_at`

type StudentAdmissionResult struct {
	Student StudentProfile `json:"student"`
	Enrollment StudentEnrollment `json:"enrollment"`
}

func normalizeEnrollmentAdmission(number string, admissionDate time.Time) (string, error) {
	number = strings.TrimSpace(number)
	if number == "" || len(number) > 50 || admissionDate.IsZero() {
		return "", ErrInvalidAcademicInput
	}
	return number, nil
}

func validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn time.Time) error {
	within := func(value time.Time) bool {
		date := value.Format("2006-01-02")
		return date >= yearStartsOn.Format("2006-01-02") && date <= yearEndsOn.Format("2006-01-02")
	}
	if !within(admissionDate) || !within(enrolledAt) {
		return ErrInvalidAcademicInput
	}
	return nil
}

// CreateStudentAdmission creates the permanent student identity and its first
// academic-year enrollment atomically. The legacy student-level admission
// number is deliberately NULL; the enrollment is the source of truth.
func (r *AcademicRepository) CreateStudentAdmission(
	ctx context.Context,
	actorID, schoolID int64,
	fullName, admissionNumber string,
	yearID, sectionID int64,
	rollNumber *string,
	admissionDate time.Time,
) (*StudentAdmissionResult, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	fullName = strings.TrimSpace(fullName)
	if fullName == "" || len(fullName) > 200 || yearID <= 0 || sectionID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil {
		return nil, err
	}
	if rollNumber != nil {
		value := strings.TrimSpace(*rollNumber)
		if value == "" {
			rollNumber = nil
		} else if len(value) > 30 {
			return nil, ErrInvalidAcademicInput
		} else {
			rollNumber = &value
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin combined student admission: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "students.create"); err != nil {
		return nil, err
	}
	var yearActive bool
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on
		FROM academic_years WHERE id=$1 AND school_id=$2`, yearID, schoolID).
		Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load first enrollment academic year: %w", err)
	}
	if !yearActive {
		return nil, ErrInvalidAcademicInput
	}
	if err = validateEnrollmentDateRange(admissionDate, admissionDate, yearStartsOn, yearEndsOn); err != nil {
		return nil, err
	}
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections
		WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE`,
		sectionID, schoolID, yearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load first enrollment section: %w", err)
	}

	student, err := scanStudentProfile(tx.QueryRow(ctx, `INSERT INTO students
		(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING `+studentProfileColumns, schoolID, fullName))
	if err != nil {
		return nil, academicWriteError(err)
	}
	// The API's admission_number is the first enrollment's year-scoped value;
	// the legacy students column intentionally remains NULL.
	student.AdmissionNumber = admissionNumber
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events
		(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.created','student',$3,'success',
		jsonb_build_object('initial_academic_year_id',$4::bigint))`,
		actorID, schoolID, fmt.Sprint(student.ID), yearID); err != nil {
		return nil, fmt.Errorf("audit student creation: %w", err)
	}

	enrollment, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING `+enrollmentColumns,
		schoolID, student.ID, yearID, classID, sectionID, rollNumber, admissionNumber, admissionDate))
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events
		(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrolled','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'academic_year_id',$5::bigint,
		'class_id',$6::bigint,'section_id',$7::bigint,'admission_number',$8::text,'admission_date',$9::date))`,
		actorID, schoolID, fmt.Sprint(enrollment.ID), student.ID, yearID, classID, sectionID,
		admissionNumber, admissionDate.Format("2006-01-02")); err != nil {
		return nil, fmt.Errorf("audit first student enrollment: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit combined student admission: %w", err)
	}
	return &StudentAdmissionResult{Student: *student, Enrollment: *enrollment}, nil
}

func (r *AcademicRepository) ListStudentEnrollments(ctx context.Context, actorID, schoolID, yearID, studentID int64) ([]StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if yearID < 0 || studentID < 0 { return nil, ErrInvalidAcademicInput }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_basic"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `SELECT e.id,e.school_id,e.student_id,e.academic_year_id,e.class_id,e.section_id,e.roll_number,
		e.class_name_snapshot,e.section_name_snapshot,e.admission_number,e.admission_date,e.status,e.enrolled_at,e.ended_at,e.created_at,e.updated_at,st.full_name,
		EXISTS (SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
		 WHERE pr.principal_id=$4 AND pr.revoked_at IS NULL AND r.school_id=e.school_id
		 AND r.scope_type='school' AND r.principal_type='staff' AND r.role_key='school_admin' AND r.is_system=TRUE),
		EXISTS (SELECT 1 FROM class_incharge_assignments cia
		 WHERE cia.school_id=e.school_id AND cia.academic_year_id=e.academic_year_id
		 AND cia.section_id=e.section_id AND cia.teacher_principal_id=$4 AND cia.revoked_at IS NULL)
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
			OR (
				EXISTS (
					SELECT 1 FROM principal_roles tpr
					JOIN roles tr ON tr.id=tpr.role_id
					WHERE tpr.principal_id=$4 AND tpr.revoked_at IS NULL
					  AND tr.school_id=e.school_id AND tr.scope_type='school'
					  AND tr.principal_type='staff' AND tr.role_key='teacher'
					  AND tr.is_system=TRUE
				)
				AND (
					EXISTS (
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

func (r *AcademicRepository) CreateStudentEnrollment(ctx context.Context, actorID, schoolID, studentID, yearID, sectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if studentID <= 0 || yearID <= 0 || sectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil { return nil, err }
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
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on FROM academic_years WHERE id=$1 AND school_id=$2`, yearID, schoolID).Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	if err = validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn); err != nil { return nil, err }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE`, sectionID, schoolID, yearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load enrollment section: %w", err)
	}
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+enrollmentColumns,
		schoolID, studentID, yearID, classID, sectionID, rollNumber, admissionNumber, admissionDate, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrolled','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'academic_year_id',$5::bigint,'class_id',$6::bigint,'section_id',$7::bigint,'admission_number',$8::text,'admission_date',$9::date))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, yearID, classID, sectionID, admissionNumber, admissionDate.Format("2006-01-02")); err != nil { return nil, fmt.Errorf("audit student enrollment: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student enrollment: %w", err) }
	return v, nil
}

func (r *AcademicRepository) PromoteStudentEnrollment(ctx context.Context, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*StudentEnrollment, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if sourceEnrollmentID <= 0 || destinationYearID <= 0 || destinationSectionID <= 0 || enrolledAt.IsZero() { return nil, ErrInvalidAcademicInput }
	admissionNumber, err := normalizeEnrollmentAdmission(admissionNumber, admissionDate)
	if err != nil { return nil, err }
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
	var yearStartsOn, yearEndsOn time.Time
	if err = tx.QueryRow(ctx, `SELECT status='active',starts_on,ends_on FROM academic_years WHERE id=$1 AND school_id=$2`, destinationYearID, schoolID).Scan(&yearActive, &yearStartsOn, &yearEndsOn); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination academic year: %w", err)
	}
	if !yearActive { return nil, ErrInvalidAcademicInput }
	if err = validateEnrollmentDateRange(admissionDate, enrolledAt, yearStartsOn, yearEndsOn); err != nil { return nil, err }
	var classID int64
	if err = tx.QueryRow(ctx, `SELECT class_id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND is_active=TRUE`, destinationSectionID, schoolID, destinationYearID).Scan(&classID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load destination section: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE student_enrollments SET status='promoted',ended_at=$3,updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND status='active'`, sourceEnrollmentID, schoolID, enrolledAt); err != nil { return nil, academicWriteError(err) }
	v, err := scanStudentEnrollment(tx.QueryRow(ctx, `INSERT INTO student_enrollments
		(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+enrollmentColumns,
		schoolID, studentID, destinationYearID, classID, destinationSectionID, rollNumber, admissionNumber, admissionDate, enrolledAt))
	if err != nil { return nil, academicWriteError(err) }
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.promoted','student_enrollment',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'source_enrollment_id',$5::bigint,'source_academic_year_id',$6::bigint,'destination_academic_year_id',$7::bigint,'destination_class_id',$8::bigint,'destination_section_id',$9::bigint,'admission_number',$10::text,'admission_date',$11::date))`,
		actorID, schoolID, fmt.Sprint(v.ID), studentID, sourceEnrollmentID, sourceYearID, destinationYearID, classID, destinationSectionID, admissionNumber, admissionDate.Format("2006-01-02")); err != nil { return nil, fmt.Errorf("audit student promotion: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit student promotion: %w", err) }
	return v, nil
}
