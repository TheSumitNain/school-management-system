package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

)

func TestStudentMedicalPhysicalDetailsV36PostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var ready bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.student_medical_details') IS NOT NULL
		   AND to_regclass('public.student_physical_details') IS NOT NULL
		   AND to_regprocedure('public.guard_student_health_profile()') IS NOT NULL
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
	if err := NewRoleRepository(tx).AssignStaffRole(ctx, platformAdminPrincipalID, schoolID, actorPrincipalID, schoolAdminRoleID); err != nil {
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
	if err != nil || loadedMedical.ID != medical.ID || loadedMedical.BloodGroup == nil || *loadedMedical.BloodGroup != "AB+" {
		t.Fatalf("read medical profile: item=%+v err=%v", loadedMedical, err)
	}

	height, weight := 132.5, 31.25
	measuredOn := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
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
	if err != nil || loadedPhysical.ID != physical.ID || loadedPhysical.HeightCM == nil || *loadedPhysical.HeightCM != height {
		t.Fatalf("read physical profile: item=%+v err=%v", loadedPhysical, err)
	}

	if _, err := repo.SaveStudentMedicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		ptrString("INVALID"), nil, nil, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("invalid blood group error = %v, want invalid input", err)
	}
	invalidHeight := 300.0
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&invalidHeight, nil, nil, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("invalid height error = %v, want invalid input", err)
	}
	invalidPrecision := 132.555
	if _, err := repo.SaveStudentPhysicalDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&invalidPrecision, nil, nil, nil, nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("measurement precision error = %v, want invalid input", err)
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
	assertRejected("medical profile delete",
		`DELETE FROM student_medical_details WHERE student_id=$1`, studentID)
	assertRejected("physical profile delete",
		`DELETE FROM student_physical_details WHERE student_id=$1`, studentID)
	assertRejected("medical profile ownership rewrite",
		`UPDATE student_medical_details SET student_id=$2 WHERE student_id=$1`, studentID, secondStudentID)

	var readMedical, readPhysical, updateMedical, updatePhysical int
	if err := tx.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE action='school.student.medical_details.read'),
		  COUNT(*) FILTER (WHERE action='school.student.physical_details.read'),
		  COUNT(*) FILTER (WHERE action='school.student.medical_details.updated'),
		  COUNT(*) FILTER (WHERE action='school.student.physical_details.updated')
		FROM audit_events
		WHERE school_id=$1 AND actor_principal_id=$2
		  AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(studentID)).Scan(
		&readMedical, &readPhysical, &updateMedical, &updatePhysical,
	); err != nil {
		t.Fatalf("count health profile audit events: %v", err)
	}
	if readMedical != 1 || readPhysical != 1 || updateMedical != 1 || updatePhysical != 1 {
		t.Fatalf("health audit counts medical read=%d physical read=%d medical update=%d physical update=%d; want 1 each",
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
			   )
		)
	`, schoolID, actorPrincipalID, fmt.Sprint(studentID), allergies, conditions, medications).Scan(&auditContainsHealthValue); err != nil {
		t.Fatalf("inspect health audit metadata: %v", err)
	}
	if auditContainsHealthValue {
		t.Fatal("health profile audit metadata must not contain medical values or notes")
	}

	// Ensure no unexpected open transaction state remains after nested repository operations.
	if err := tx.QueryRow(ctx, `SELECT 1`).Scan(new(int)); err != nil {
		t.Fatalf("outer transaction is unusable after health operations: %v", err)
	}
}

func ptrString(value string) *string { return &value }
