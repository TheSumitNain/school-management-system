package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type StudentMedicalDetailsVersion struct {
	ID int64 `json:"id"`
	VersionNumber int `json:"version_number"`
	BloodGroup *string `json:"blood_group,omitempty"`
	Allergies *string `json:"allergies,omitempty"`
	KnownConditions *string `json:"known_conditions,omitempty"`
	OngoingMedications *string `json:"ongoing_medications,omitempty"`
	EmergencyMedicalNotes *string `json:"emergency_medical_notes,omitempty"`
	ChangedByPrincipalID int64 `json:"changed_by_principal_id"`
	ChangedAt time.Time `json:"changed_at"`
}

type StudentMedicalDetails struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	StudentID int64 `json:"student_id"`
	VersionNumber int `json:"version_number"`
	BloodGroup *string `json:"blood_group,omitempty"`
	Allergies *string `json:"allergies,omitempty"`
	KnownConditions *string `json:"known_conditions,omitempty"`
	OngoingMedications *string `json:"ongoing_medications,omitempty"`
	EmergencyMedicalNotes *string `json:"emergency_medical_notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	History []StudentMedicalDetailsVersion `json:"history"`
}

type StudentPhysicalMeasurement struct {
	ID int64 `json:"id"`
	HeightCM *float64 `json:"height_cm,omitempty"`
	WeightKG *float64 `json:"weight_kg,omitempty"`
	MeasuredOn time.Time `json:"measured_on"`
	VisionNotes *string `json:"vision_notes,omitempty"`
	PhysicalNotes *string `json:"physical_notes,omitempty"`
	RecordedByPrincipalID int64 `json:"recorded_by_principal_id"`
	CreatedAt time.Time `json:"created_at"`
}

type StudentPhysicalDetails struct {
	SchoolID int64 `json:"school_id"`
	StudentID int64 `json:"student_id"`
	ID int64 `json:"id"`
	HeightCM *float64 `json:"height_cm,omitempty"`
	WeightKG *float64 `json:"weight_kg,omitempty"`
	MeasuredOn *time.Time `json:"measured_on,omitempty"`
	VisionNotes *string `json:"vision_notes,omitempty"`
	PhysicalNotes *string `json:"physical_notes,omitempty"`
	RecordedByPrincipalID int64 `json:"recorded_by_principal_id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	History []StudentPhysicalMeasurement `json:"history"`
}

func normalizeBloodGroup(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(*value))
	if normalized == "" {
		return nil, nil
	}
	switch normalized {
	case "A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-":
		return &normalized, nil
	default:
		return nil, ErrInvalidAcademicInput
	}
}

func auditStudentHealthRead(ctx context.Context, tx pgx.Tx, actorID, schoolID, studentID int64, section string) error {
	action := "school.student." + section + "_details.read"
	resourceType := "student_" + section + "_details"
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,$4,$5,'success',jsonb_build_object('section',$6::text))
	`, actorID, schoolID, action, resourceType, fmt.Sprint(studentID), section)
	if err != nil {
		return fmt.Errorf("audit student %s details read: %w", section, err)
	}
	return nil
}

func loadStudentMedicalDetails(ctx context.Context, tx pgx.Tx, schoolID, studentID int64) (*StudentMedicalDetails, error) {
	item := &StudentMedicalDetails{SchoolID: schoolID, StudentID: studentID, History: []StudentMedicalDetailsVersion{}}
	err := tx.QueryRow(ctx, `
		SELECT id,school_id,student_id,version_number,blood_group,allergies,known_conditions,
		       ongoing_medications,emergency_medical_notes,created_at,updated_at
		  FROM student_medical_details
		 WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.VersionNumber, &item.BloodGroup,
		&item.Allergies, &item.KnownConditions, &item.OngoingMedications,
		&item.EmergencyMedicalNotes, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load current student medical details: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT id,version_number,blood_group,allergies,known_conditions,ongoing_medications,
		       emergency_medical_notes,changed_by_principal_id,changed_at
		  FROM student_medical_details_history
		 WHERE school_id=$1 AND student_id=$2
		 ORDER BY version_number DESC
	`, schoolID, studentID)
	if err != nil {
		return nil, fmt.Errorf("load student medical history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var version StudentMedicalDetailsVersion
		if err := rows.Scan(&version.ID, &version.VersionNumber, &version.BloodGroup, &version.Allergies,
			&version.KnownConditions, &version.OngoingMedications, &version.EmergencyMedicalNotes,
			&version.ChangedByPrincipalID, &version.ChangedAt); err != nil {
			return nil, fmt.Errorf("scan student medical history: %w", err)
		}
		item.History = append(item.History, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student medical history: %w", err)
	}
	return item, nil
}

func (r *AcademicRepository) GetStudentMedicalDetails(ctx context.Context, actorID, schoolID, studentID int64) (*StudentMedicalDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student medical read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`, studentID, schoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check student for medical details: %w", err)
	}
	if !exists {
		return nil, ErrAcademicNotFound
	}
	item, err := loadStudentMedicalDetails(ctx, tx, schoolID, studentID)
	if err != nil {
		return nil, err
	}
	if err := auditStudentHealthRead(ctx, tx, actorID, schoolID, studentID, "medical"); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student medical read audit: %w", err)
	}
	return item, nil
}

func (r *AcademicRepository) SaveStudentMedicalDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
	bloodGroup, allergies, knownConditions, ongoingMedications, emergencyNotes *string,
) (*StudentMedicalDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	var err error
	if bloodGroup, err = normalizeBloodGroup(bloodGroup); err != nil {
		return nil, err
	}
	if allergies, err = normalizeAddressText(allergies, 2000); err != nil {
		return nil, err
	}
	if knownConditions, err = normalizeAddressText(knownConditions, 2000); err != nil {
		return nil, err
	}
	if ongoingMedications, err = normalizeAddressText(ongoingMedications, 2000); err != nil {
		return nil, err
	}
	if emergencyNotes, err = normalizeAddressText(emergencyNotes, 2000); err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student medical save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,
		studentID, schoolID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load student for medical details: %w", err)
	}
	if !active {
		return nil, ErrInvalidAcademicInput
	}

	existing, err := loadStudentMedicalCurrentForUpdate(ctx, tx, schoolID, studentID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_medical_details
				(school_id,student_id,version_number,blood_group,allergies,known_conditions,ongoing_medications,emergency_medical_notes)
			VALUES($1,$2,1,$3,$4,$5,$6,$7)
		`, schoolID, studentID, bloodGroup, allergies, knownConditions, ongoingMedications, emergencyNotes); err != nil {
			return nil, academicWriteError(err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_medical_details_history
				(school_id,student_id,version_number,blood_group,allergies,known_conditions,
				 ongoing_medications,emergency_medical_notes,changed_by_principal_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, schoolID, studentID, existing.VersionNumber, existing.BloodGroup, existing.Allergies,
			existing.KnownConditions, existing.OngoingMedications, existing.EmergencyMedicalNotes, actorID); err != nil {
			return nil, fmt.Errorf("preserve prior student medical version: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_medical_details SET
				version_number=version_number+1,
				blood_group=$3,allergies=$4,known_conditions=$5,
				ongoing_medications=$6,emergency_medical_notes=$7,updated_at=NOW()
			WHERE school_id=$1 AND student_id=$2
		`, schoolID, studentID, bloodGroup, allergies, knownConditions, ongoingMedications, emergencyNotes); err != nil {
			return nil, academicWriteError(err)
		}
	}
	var versionNumber int
	if err := tx.QueryRow(ctx, `SELECT version_number FROM student_medical_details WHERE school_id=$1 AND student_id=$2`,
		schoolID, studentID).Scan(&versionNumber); err != nil {
		return nil, fmt.Errorf("load saved medical version number: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.medical_details.updated','student_medical_details',$3,'success',
		       jsonb_build_object(
		           'version_number',$4::integer,
		           'has_blood_group',$5::boolean,
		           'has_allergies',$6::boolean,
		           'has_known_conditions',$7::boolean,
		           'has_ongoing_medications',$8::boolean,
		           'has_emergency_notes',$9::boolean
		       ))
	`, actorID, schoolID, fmt.Sprint(studentID), versionNumber, bloodGroup != nil,
		allergies != nil, knownConditions != nil, ongoingMedications != nil, emergencyNotes != nil); err != nil {
		return nil, fmt.Errorf("audit student medical update: %w", err)
	}
	item, err := loadStudentMedicalDetails(ctx, tx, schoolID, studentID)
	if err != nil {
		return nil, fmt.Errorf("read saved student medical details: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student medical details: %w", err)
	}
	return item, nil
}

func loadStudentMedicalCurrentForUpdate(ctx context.Context, tx pgx.Tx, schoolID, studentID int64) (*StudentMedicalDetails, error) {
	var item StudentMedicalDetails
	err := tx.QueryRow(ctx, `
		SELECT id,school_id,student_id,version_number,blood_group,allergies,known_conditions,
		       ongoing_medications,emergency_medical_notes,created_at,updated_at
		  FROM student_medical_details
		 WHERE school_id=$1 AND student_id=$2
		 FOR UPDATE
	`, schoolID, studentID).Scan(
		&item.ID, &item.SchoolID, &item.StudentID, &item.VersionNumber, &item.BloodGroup,
		&item.Allergies, &item.KnownConditions, &item.OngoingMedications,
		&item.EmergencyMedicalNotes, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock current student medical details: %w", err)
	}
	return &item, nil
}

func loadStudentPhysicalDetails(ctx context.Context, tx pgx.Tx, schoolID, studentID int64) (*StudentPhysicalDetails, error) {
	item := &StudentPhysicalDetails{SchoolID: schoolID, StudentID: studentID, History: []StudentPhysicalMeasurement{}}
	rows, err := tx.Query(ctx, `
		SELECT id,height_cm,weight_kg,measured_on,vision_notes,physical_notes,recorded_by_principal_id,created_at
		  FROM student_physical_details
		 WHERE school_id=$1 AND student_id=$2
		 ORDER BY measured_on DESC,created_at DESC,id DESC
	`, schoolID, studentID)
	if err != nil {
		return nil, fmt.Errorf("load student physical measurement history: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var measurement StudentPhysicalMeasurement
		if err := rows.Scan(&measurement.ID, &measurement.HeightCM, &measurement.WeightKG,
			&measurement.MeasuredOn, &measurement.VisionNotes, &measurement.PhysicalNotes,
			&measurement.RecordedByPrincipalID, &measurement.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan student physical measurement: %w", err)
		}
		item.History = append(item.History, measurement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student physical measurements: %w", err)
	}
	if len(item.History) > 0 {
		current := item.History[0]
		item.ID = current.ID
		item.HeightCM = current.HeightCM
		item.WeightKG = current.WeightKG
		item.MeasuredOn = &current.MeasuredOn
		item.VisionNotes = current.VisionNotes
		item.PhysicalNotes = current.PhysicalNotes
		item.RecordedByPrincipalID = current.RecordedByPrincipalID
		item.CreatedAt = &current.CreatedAt
	}
	return item, nil
}

func (r *AcademicRepository) GetStudentPhysicalDetails(ctx context.Context, actorID, schoolID, studentID int64) (*StudentPhysicalDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student physical read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`, studentID, schoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check student for physical details: %w", err)
	}
	if !exists {
		return nil, ErrAcademicNotFound
	}
	item, err := loadStudentPhysicalDetails(ctx, tx, schoolID, studentID)
	if err != nil {
		return nil, err
	}
	if err := auditStudentHealthRead(ctx, tx, actorID, schoolID, studentID, "physical"); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student physical read audit: %w", err)
	}
	return item, nil
}

func (r *AcademicRepository) SaveStudentPhysicalDetails(
	ctx context.Context, actorID, schoolID, studentID int64,
	heightCM, weightKG *float64, measuredOn *time.Time, visionNotes, physicalNotes *string,
) (*StudentPhysicalDetails, error) {
	if r == nil || r.db == nil || studentID <= 0 {
		return nil, ErrAcademicAccessDenied
	}
	if measuredOn == nil {
		return nil, ErrInvalidAcademicInput
	}
	if heightCM != nil && (*heightCM < 20 || *heightCM > 250 ||
		math.Abs(*heightCM*100-math.Round(*heightCM*100)) > 1e-8) {
		return nil, ErrInvalidAcademicInput
	}
	if weightKG != nil && (*weightKG < 1 || *weightKG > 300 ||
		math.Abs(*weightKG*100-math.Round(*weightKG*100)) > 1e-8) {
		return nil, ErrInvalidAcademicInput
	}
	var err error
	if visionNotes, err = normalizeAddressText(visionNotes, 1000); err != nil {
		return nil, err
	}
	if physicalNotes, err = normalizeAddressText(physicalNotes, 2000); err != nil {
		return nil, err
	}
	if heightCM == nil && weightKG == nil && visionNotes == nil && physicalNotes == nil {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student physical save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	var active bool
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,
		studentID, schoolID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load student for physical details: %w", err)
	}
	if !active {
		return nil, ErrInvalidAcademicInput
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_physical_details
			(school_id,student_id,height_cm,weight_kg,measured_on,vision_notes,physical_notes,recorded_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
	`, schoolID, studentID, heightCM, weightKG, measuredOn, visionNotes, physicalNotes, actorID); err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.physical_details.recorded','student_physical_details',$3,'success',
		       jsonb_build_object(
		           'has_height',$4::boolean,
		           'has_weight',$5::boolean,
		           'has_measurement_date',$6::boolean,
		           'has_vision_notes',$7::boolean,
		           'has_physical_notes',$8::boolean
		       ))
	`, actorID, schoolID, fmt.Sprint(studentID), heightCM != nil, weightKG != nil,
		true, visionNotes != nil, physicalNotes != nil); err != nil {
		return nil, fmt.Errorf("audit student physical measurement: %w", err)
	}
	item, err := loadStudentPhysicalDetails(ctx, tx, schoolID, studentID)
	if err != nil {
		return nil, fmt.Errorf("read saved student physical details: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student physical details: %w", err)
	}
	return item, nil
}
