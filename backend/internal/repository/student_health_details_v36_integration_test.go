package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/schooltime"
)

func TestStudentMedicalPhysicalDetailsV36PostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.student_medical_details') IS NOT NULL
		   AND to_regclass('public.student_physical_details') IS NOT NULL
		   AND to_regprocedure('public.guard_student_medical_details()') IS NOT NULL
		   AND to_regprocedure('public.guard_student_health_history()') IS NOT NULL
	`).Scan(&ready); err != nil {
		t.Fatalf("inspect V36 student health schema: %v", err)
	}
	if !ready {
		t.Fatal("V36 medical and physical profile schema is missing")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V36 integration test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `SELECT principal_id,school_id FROM users WHERE id=$1`, actorUserID).
		Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin actor: %v", err)
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
	var studentID, secondStudentID int64
	for _, target := range []struct {
		name string
		out  *int64
	}{
		{"V36 Health Student " + suffix, &studentID},
		{"V36 Health Second Student " + suffix, &secondStudentID},
	} {
		if err := tx.QueryRow(ctx, `
			INSERT INTO students(school_id,admission_number,full_name)
			VALUES($1,NULL,$2) RETURNING id
		`, schoolID, target.name).Scan(target.out); err != nil {
			t.Fatalf("create student fixture: %v", err)
		}
	}

	repo := NewAcademicRepository(tx)
	bloodGroup := "ab+"
	allergies := "V36 private allergy note " + suffix
	conditions := "V36 private condition note " + suffix
	medications := "V36 private medication note " + suffix
	emergencyNotes := "V36 private emergency note " + suffix
	medical, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&bloodGroup, &allergies, &conditions, &medications, &emergencyNotes)
	if err != nil {
		t.Fatalf("save medical profile: %v", err)
	}
	if medical.ID <= 0 || medical.BloodGroup == nil || *medical.BloodGroup != "AB+" ||
		medical.Allergies == nil || *medical.Allergies != allergies ||
		medical.KnownConditions == nil || *medical.KnownConditions != conditions ||
		medical.OngoingMedications == nil || *medical.OngoingMedications != medications ||
		medical.EmergencyMedicalNotes == nil || *medical.EmergencyMedicalNotes != emergencyNotes {
		t.Fatalf("unexpected saved medical profile: %+v", medical)
	}
	loadedMedical, err := repo.GetStudentMedicalDetails(ctx, actorPrincipalID, schoolID, studentID)
	if err != nil || loadedMedical.ID != medical.ID || loadedMedical.VersionNumber != 1 || loadedMedical.BloodGroup == nil || *loadedMedical.BloodGroup != "AB+" {
		t.Fatalf("read medical profile: item=%+v err=%v", loadedMedical, err)
	}
	updatedAllergies := "V36 updated private allergy note " + suffix
	updatedMedical, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&bloodGroup, &updatedAllergies, &conditions, &medications, &emergencyNotes)
	if err != nil {
		t.Fatalf("update medical profile: %v", err)
	}
	if updatedMedical.VersionNumber != 2 || len(updatedMedical.History) != 1 ||
		updatedMedical.History[0].VersionNumber != 1 || updatedMedical.History[0].Allergies == nil ||
		*updatedMedical.History[0].Allergies != allergies {
		t.Fatalf("medical profile update must preserve previous version: %+v", updatedMedical)
	}
	unchangedMedical, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&bloodGroup, &updatedAllergies, &conditions, &medications, &emergencyNotes)
	if err != nil {
		t.Fatalf("save unchanged medical profile: %v", err)
	}
	if unchangedMedical.VersionNumber != 2 || len(unchangedMedical.History) != 1 {
		t.Fatalf("saving unchanged medical details must not create a new version: %+v", unchangedMedical)
	}
	loadedMedical, err = repo.GetStudentMedicalDetails(ctx, actorPrincipalID, schoolID, studentID)
	if err != nil || loadedMedical.VersionNumber != 2 || len(loadedMedical.History) != 1 {
		t.Fatalf("read versioned medical profile: item=%+v err=%v", loadedMedical, err)
	}

	height, weight := 132.5, 31.25
	today := schooltime.Today()
	measuredOn := today.AddDate(0, 0, -1)
	vision := "V36 private vision note " + suffix
	physicalNotes := "V36 private physical note " + suffix
	physical, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&height, &weight, &measuredOn, &vision, &physicalNotes)
	if err != nil {
		t.Fatalf("save physical profile: %v", err)
	}
	if physical.ID <= 0 || physical.HeightCM == nil || *physical.HeightCM != height ||
		physical.WeightKG == nil || *physical.WeightKG != weight ||
		physical.MeasuredOn == nil || physical.VisionNotes == nil || *physical.VisionNotes != vision ||
		physical.PhysicalNotes == nil || *physical.PhysicalNotes != physicalNotes {
		t.Fatalf("unexpected saved physical profile: %+v", physical)
	}
	loadedPhysical, err := repo.GetStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, studentID)
	if err != nil || loadedPhysical.ID != physical.ID || loadedPhysical.HeightCM == nil || *loadedPhysical.HeightCM != height || len(loadedPhysical.History) != 1 {
		t.Fatalf("read physical profile: item=%+v err=%v", loadedPhysical, err)
	}
	unchangedPhysical, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&height, &weight, &measuredOn, &vision, &physicalNotes)
	if err != nil {
		t.Fatalf("retry unchanged physical measurement: %v", err)
	}
	if unchangedPhysical.ID != physical.ID || len(unchangedPhysical.History) != 1 {
		t.Fatalf("repeating the latest physical PUT must not create a duplicate measurement: %+v", unchangedPhysical)
	}
	nextHeight := 133.25
	nextMeasuredOn := today
	nextVision := "V36 next vision measurement " + suffix
	nextPhysicalNotes := "V36 next physical measurement " + suffix
	updatedPhysical, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&nextHeight, &weight, &nextMeasuredOn, &nextVision, &nextPhysicalNotes)
	if err != nil {
		t.Fatalf("record second physical measurement: %v", err)
	}
	if updatedPhysical.ID == physical.ID || updatedPhysical.HeightCM == nil || *updatedPhysical.HeightCM != nextHeight ||
		len(updatedPhysical.History) != 2 || updatedPhysical.History[1].ID != physical.ID ||
		updatedPhysical.History[1].HeightCM == nil || *updatedPhysical.History[1].HeightCM != height {
		t.Fatalf("physical measurement history did not preserve prior measurement: %+v", updatedPhysical)
	}
	unchangedPhysical, err = repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, studentID,
		&nextHeight, &weight, &nextMeasuredOn, &nextVision, &nextPhysicalNotes)
	if err != nil {
		t.Fatalf("save unchanged physical measurement: %v", err)
	}
	if unchangedPhysical.ID != updatedPhysical.ID || len(unchangedPhysical.History) != 2 {
		t.Fatalf("retrying an unchanged physical measurement must not append duplicate history: %+v", unchangedPhysical)
	}

	if _, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		ptrString("INVALID"), nil, nil, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("invalid blood group error = %v, want invalid input", err)
	}
	invalidHeight := 300.0
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&invalidHeight, nil, &measuredOn, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("invalid height error = %v, want invalid input", err)
	}
	invalidPrecision := 132.555
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&invalidPrecision, nil, &measuredOn, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("measurement precision error = %v, want invalid input", err)
	}
	invalidWeight := 0.5
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		nil, &invalidWeight, &measuredOn, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("invalid weight error = %v, want invalid input", err)
	}
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&height, nil, nil, nil, nil); !errors.Is(err, ErrInvalidStudentMeasurementDate) {
		t.Fatalf("missing measurement date error = %v, want measurement-date field error", err)
	}
	futureMeasurementDate := schooltime.Today().AddDate(0, 0, 1)
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&height, nil, &futureMeasurementDate, nil, nil); !errors.Is(err, ErrInvalidStudentMeasurementDate) {
		t.Fatalf("future measurement date error = %v, want invalid input", err)
	}
	futureMeasuredOn := today.AddDate(0, 0, 1)
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&height, nil, &futureMeasuredOn, nil, nil); !errors.Is(err, ErrInvalidStudentMeasurementDate) {
		t.Fatalf("future measurement date error = %v, want invalid input", err)
	}

	// Text limits are Unicode code-point limits, not UTF-8 byte or UTF-16 code-unit limits.
	maxMedicalText := strings.Repeat("अ", 2000)
	maxVisionText := strings.Repeat("अ", 1000)
	maxPhysicalText := strings.Repeat("अ", 2000)
	boundaryMedical, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		nil, &maxMedicalText, &maxMedicalText, &maxMedicalText, &maxMedicalText)
	if err != nil {
		t.Fatalf("save medical fields at 2,000 Unicode characters: %v", err)
	}
	if boundaryMedical.Allergies == nil || len([]rune(*boundaryMedical.Allergies)) != 2000 ||
		boundaryMedical.KnownConditions == nil || len([]rune(*boundaryMedical.KnownConditions)) != 2000 ||
		boundaryMedical.OngoingMedications == nil || len([]rune(*boundaryMedical.OngoingMedications)) != 2000 ||
		boundaryMedical.EmergencyMedicalNotes == nil || len([]rune(*boundaryMedical.EmergencyMedicalNotes)) != 2000 {
		t.Fatal("medical fields must preserve exactly 2,000 Unicode characters")
	}
	tooLongMedicalText := strings.Repeat("अ", 2001)
	for label, values := range map[string][4]*string{
		"allergies": {&tooLongMedicalText, &maxMedicalText, &maxMedicalText, &maxMedicalText},
		"known conditions": {&maxMedicalText, &tooLongMedicalText, &maxMedicalText, &maxMedicalText},
		"ongoing medications": {&maxMedicalText, &maxMedicalText, &tooLongMedicalText, &maxMedicalText},
		"emergency notes": {&maxMedicalText, &maxMedicalText, &maxMedicalText, &tooLongMedicalText},
	} {
		if _, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
			nil, values[0], values[1], values[2], values[3]); !errors.Is(err, ErrInvalidAcademicInput) {
			t.Fatalf("%s over-limit Unicode text error = %v, want invalid input", label, err)
		}
	}

	boundaryPhysical, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		nil, nil, &measuredOn, &maxVisionText, &maxPhysicalText)
	if err != nil {
		t.Fatalf("save physical fields at Unicode character limits: %v", err)
	}
	if boundaryPhysical.VisionNotes == nil || len([]rune(*boundaryPhysical.VisionNotes)) != 1000 ||
		boundaryPhysical.PhysicalNotes == nil || len([]rune(*boundaryPhysical.PhysicalNotes)) != 2000 {
		t.Fatal("physical fields must preserve 1,000/2,000 Unicode characters")
	}
	tooLongVisionText := strings.Repeat("अ", 1001)
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		nil, nil, &measuredOn, &tooLongVisionText, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("vision notes over-limit Unicode text error = %v, want invalid input", err)
	}
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		nil, nil, &measuredOn, nil, &tooLongMedicalText); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("physical notes over-limit Unicode text error = %v, want invalid input", err)
	}
	otherSchoolID := schoolID + 999999
	if _, err := repo.GetStudentMedicalDetails(ctx, actorPrincipalID, otherSchoolID, studentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school medical read error = %v, want access denied", err)
	}

	assertRejected := func(label, query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, `SAVEPOINT v36_rejected_write`); err != nil {
			t.Fatalf("%s: create savepoint: %v", label, err)
		}
		_, writeErr := tx.Exec(ctx, query, args...)
		if writeErr == nil {
			_, _ = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v36_rejected_write`)
			_, _ = tx.Exec(ctx, `RELEASE SAVEPOINT v36_rejected_write`)
			t.Fatalf("%s: expected database rejection", label)
		}
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v36_rejected_write`); err != nil {
			t.Fatalf("%s: rollback savepoint: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT v36_rejected_write`); err != nil {
			t.Fatalf("%s: release savepoint: %v", label, err)
		}
	}
	assertRejected("invalid blood group database constraint",
		`INSERT INTO student_medical_details(school_id,student_id,blood_group)
		 VALUES($1,$2,'INVALID')`, schoolID, secondStudentID)
	assertRejected("height range database constraint",
		`INSERT INTO student_physical_details(school_id,student_id,height_cm,measured_on,recorded_by_principal_id)
		 VALUES($1,$2,300,$3,$4)`, schoolID, secondStudentID, measuredOn, actorPrincipalID)
	assertRejected("medical profile delete",
		`DELETE FROM student_medical_details WHERE student_id=$1`, studentID)
	assertRejected("physical measurement delete",
		`DELETE FROM student_physical_details WHERE student_id=$1`, studentID)
	assertRejected("physical measurement update",
		`UPDATE student_physical_details SET height_cm=140 WHERE id=$1`, physical.ID)
	assertRejected("medical history update",
		`UPDATE student_medical_details_history SET allergies='rewrite' WHERE student_id=$1`, studentID)
	assertRejected("medical history delete",
		`DELETE FROM student_medical_details_history WHERE student_id=$1`, studentID)
	assertRejected("medical profile ownership rewrite",
		`UPDATE student_medical_details SET student_id=$2 WHERE student_id=$1`, studentID, secondStudentID)

	var readMedical, readPhysical, updateMedical, updatePhysical int
	if err := tx.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE action='school.student.medical_details.read'),
		  COUNT(*) FILTER (WHERE action='school.student.physical_details.read'),
		  COUNT(*) FILTER (WHERE action='school.student.medical_details.updated'),
		  COUNT(*) FILTER (WHERE action='school.student.physical_details.recorded')
		FROM audit_events
		WHERE school_id=$1 AND actor_principal_id=$2
		  AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(studentID)).Scan(
		&readMedical, &readPhysical, &updateMedical, &updatePhysical,
	); err != nil {
		t.Fatalf("count health profile audit events: %v", err)
	}
	if readMedical != 2 || readPhysical != 1 || updateMedical != 2 || updatePhysical != 2 {
		t.Fatalf("health audit counts medical read=%d physical read=%d medical update=%d physical record=%d; want 2,1,2,2",
			readMedical, readPhysical, updateMedical, updatePhysical)
	}

	var auditContainsHealthValue bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM audit_events
			 WHERE school_id=$1 AND actor_principal_id=$2 AND resource_id=$3
			   AND (
			     metadata::text LIKE '%'||$4||'%'
			     OR metadata::text LIKE '%'||$5||'%'
			     OR metadata::text LIKE '%'||$6||'%'
			     OR metadata::text LIKE '%'||$7||'%'
			     OR metadata::text LIKE '%'||$8||'%'
			   )
		)
	`, schoolID, actorPrincipalID, fmt.Sprint(studentID), allergies, conditions, medications, vision, physicalNotes).Scan(&auditContainsHealthValue); err != nil {
		t.Fatalf("inspect health audit metadata: %v", err)
	}
	if auditContainsHealthValue {
		t.Fatal("health profile audit metadata must not contain medical values or notes")
	}

	var teacherRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		  AND scope_type='school' AND role_key='teacher' AND is_system=TRUE
	`, schoolID).Scan(&teacherRoleID); err != nil {
		t.Fatalf("load Teacher role for restricted health access test: %v", err)
	}
	var teacherPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users(school_id,username,password_hash,status)
		VALUES($1,$2,'integration-test-hash','active') RETURNING principal_id
	`, schoolID, "v36t-"+suffix).Scan(&teacherPrincipalID); err != nil {
		t.Fatalf("create Teacher health access fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`, teacherPrincipalID, teacherRoleID, actorPrincipalID); err != nil {
		t.Fatalf("assign Teacher role for restricted health access test: %v", err)
	}
	if _, err := repo.GetStudentMedicalDetails(ctx, teacherPrincipalID, schoolID, studentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("Teacher medical read error = %v, want access denied", err)
	}
	if _, err := repo.GetStudentPhysicalDetails(ctx, teacherPrincipalID, schoolID, studentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("Teacher physical read error = %v, want access denied", err)
	}

	// Cross-tenant medical and physical data must be inaccessible for both
	// reads and writes, independently of the role-based checks above.
	foreignSchoolID := schoolID + 999999
	if _, err := repo.GetStudentMedicalDetails(ctx, actorPrincipalID, foreignSchoolID, studentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school medical read error = %v, want access denied", err)
	}
	if _, err := repo.GetStudentPhysicalDetails(ctx, actorPrincipalID, foreignSchoolID, studentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school physical read error = %v, want access denied", err)
	}
	if _, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, foreignSchoolID, studentID,
		&bloodGroup, &updatedAllergies, &conditions, &medications, &emergencyNotes); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school medical update error = %v, want access denied", err)
	}
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, foreignSchoolID, studentID,
		&nextHeight, &weight, &nextMeasuredOn, &nextVision, &nextPhysicalNotes); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school physical update error = %v, want access denied", err)
	}

	// Ensure no unexpected open transaction state remains after nested repository operations.
	if err := tx.QueryRow(ctx, `SELECT 1`).Scan(new(int)); err != nil {
		t.Fatalf("outer transaction is unusable after health operations: %v", err)
	}
}

func ptrString(value string) *string { return &value }
