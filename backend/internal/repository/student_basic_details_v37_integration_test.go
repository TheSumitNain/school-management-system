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
	var birthDateConstraint string
	if err := pool.QueryRow(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='student_basic_details'::regclass AND conname='student_basic_details_birth_date'`).Scan(&birthDateConstraint); err != nil {
		t.Fatalf("read birth-date constraint: %v", err)
	}
	if !strings.Contains(birthDateConstraint, "Asia/Kolkata") {
		t.Fatalf("birth-date constraint must use the Asia/Kolkata calendar date, got %q", birthDateConstraint)
	}

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

	repository := NewAcademicRepository(tx)
	birthDate := time.Date(2018, 4, 12, 0, 0, 0, 0, time.UTC)
	placeOfBirth := "Ambala"
	nationality := "Indian"
	religion := "Not specified"
	category := "general"
	ews := false
	repoRegistration := "SR-API-" + suffix
	basic, err := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&birthDate, &placeOfBirth, ptrString("female"), &nationality, &religion, &category, &ews, &repoRegistration)
	if err != nil {
		t.Fatalf("save student basic details through repository: %v", err)
	}
	if _, duplicateErr := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&birthDate, nil, nil, nil, nil, nil, nil, &registration); !errors.Is(duplicateErr, ErrStudentRegistrationNumberConflict) {
		t.Fatalf("duplicate school registration number error = %v, want field-specific conflict", duplicateErr)
	}
	if basic.DateOfBirth == nil || basic.DateOfBirth.Format("2006-01-02") != "2018-04-12" ||
		basic.Gender == nil || *basic.Gender != "female" || basic.EWSStatus == nil || *basic.EWSStatus ||
		basic.SchoolRegistrationNumber == nil || *basic.SchoolRegistrationNumber != repoRegistration {
		t.Fatalf("unexpected repository basic details: %+v", basic)
	}
	loadedBasic, err := repository.GetStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID)
	if err != nil || loadedBasic.ID != basic.ID || loadedBasic.Nationality == nil || *loadedBasic.Nationality != nationality {
		t.Fatalf("read student basic details through repository: item=%+v err=%v", loadedBasic, err)
	}
	var readAuditCount int
	var readAuditMetadata string
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),COALESCE(string_agg(metadata::text,' '),'')
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.basic_details.read'
		   AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(secondStudentID)).Scan(&readAuditCount, &readAuditMetadata); err != nil {
		t.Fatalf("inspect basic profile read audit: %v", err)
	}
	if readAuditCount != 1 || !strings.Contains(readAuditMetadata, "basic_details") ||
		strings.Contains(readAuditMetadata, nationality) || strings.Contains(readAuditMetadata, placeOfBirth) {
		t.Fatalf("basic profile read audit must record section only: count=%d metadata=%s", readAuditCount, readAuditMetadata)
	}
	var auditCount int
	var auditMetadata string
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),COALESCE(string_agg(metadata::text,' '),'')
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.basic_details.updated'
		   AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(basic.ID)).Scan(&auditCount, &auditMetadata); err != nil {
		t.Fatalf("inspect basic profile audit: %v", err)
	}
	if auditCount != 1 || strings.Contains(auditMetadata, placeOfBirth) || strings.Contains(auditMetadata, nationality) {
		t.Fatalf("basic profile audit must record field presence only: count=%d metadata=%s", auditCount, auditMetadata)
	}

	// A lost response followed by an identical retry must not rewrite the
	// basic-details timestamp or append another sensitive-profile audit event.
	unchangedBasic, err := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&birthDate, &placeOfBirth, ptrString("female"), &nationality, &religion, &category, &ews, &repoRegistration)
	if err != nil || unchangedBasic == nil || unchangedBasic.ID != basic.ID {
		t.Fatalf("retry identical student basic details: item=%+v err=%v", unchangedBasic, err)
	}
	var retryAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.basic_details.updated' AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(basic.ID)).Scan(&retryAuditCount); err != nil {
		t.Fatalf("count basic details audit events after identical retry: %v", err)
	}
	if !unchangedBasic.UpdatedAt.Equal(loadedBasic.UpdatedAt) || retryAuditCount != auditCount || retryAuditCount != 1 {
		t.Fatalf("identical basic-details retry changed timestamp/audit count: before=%s after=%s audits before=%d after=%d",
			loadedBasic.UpdatedAt, unchangedBasic.UpdatedAt, auditCount, retryAuditCount)
	}
	if _, err := repository.GetStudentBasicDetails(ctx, actorPrincipalID, schoolID+999999, secondStudentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school basic details read error = %v, want access denied", err)
	}

	if _, err := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID+999999, secondStudentID,
		&birthDate, &placeOfBirth, ptrString("female"), &nationality, &religion, &category, &ews, &repoRegistration); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school basic details update error = %v, want access denied", err)
	}

	// The basic-details VARCHAR limits count Unicode characters, not UTF-8 bytes.
	maxPlace := strings.Repeat("अ", 120)
	maxNationality := strings.Repeat("न", 80)
	maxReligion := strings.Repeat("ध", 80)
	maxRegistration := strings.Repeat("प", 50)
	unicodeBasic, err := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&birthDate, &maxPlace, nil, &maxNationality, &maxReligion, nil, nil, &maxRegistration)
	if err != nil {
		t.Fatalf("save basic details at Unicode character limits: %v", err)
	}
	if unicodeBasic.PlaceOfBirth == nil || len([]rune(*unicodeBasic.PlaceOfBirth)) != 120 ||
		unicodeBasic.Nationality == nil || len([]rune(*unicodeBasic.Nationality)) != 80 ||
		unicodeBasic.Religion == nil || len([]rune(*unicodeBasic.Religion)) != 80 ||
		unicodeBasic.SchoolRegistrationNumber == nil || len([]rune(*unicodeBasic.SchoolRegistrationNumber)) != 50 {
		t.Fatalf("basic-details fields must preserve exact Unicode limits: %+v", unicodeBasic)
	}
	futureBirthDate := schooltime.Today().AddDate(0, 0, 1)
	if _, saveErr := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&futureBirthDate, nil, nil, nil, nil, nil, nil, nil); !errors.Is(saveErr, ErrInvalidStudentDateOfBirth) {
		t.Fatalf("future date of birth error = %v, want invalid input", saveErr)
	}
	tooOldBirthDate := schooltime.Today().AddDate(-131, 0, 0)
	if _, saveErr := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
		&tooOldBirthDate, nil, nil, nil, nil, nil, nil, nil); !errors.Is(saveErr, ErrInvalidStudentDateOfBirth) {
		t.Fatalf("date of birth older than 130 years error = %v, want invalid input", saveErr)
	}

	for _, field := range []struct {
		name string
		place, nationality, religion, registration *string
	}{
		{name: "place of birth", place: ptrString(strings.Repeat("अ", 121))},
		{name: "nationality", nationality: ptrString(strings.Repeat("न", 81))},
		{name: "religion", religion: ptrString(strings.Repeat("ध", 81))},
		{name: "school registration number", registration: ptrString(strings.Repeat("प", 51))},
	} {
		_, saveErr := repository.SaveStudentBasicDetails(ctx, actorPrincipalID, schoolID, secondStudentID,
			nil, field.place, nil, field.nationality, field.religion, nil, nil, field.registration)
		if !errors.Is(saveErr, ErrInvalidAcademicInput) {
			t.Fatalf("%s over-limit Unicode input error = %v, want invalid input", field.name, saveErr)
		}
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
		 VALUES($1,$2,(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date + 1,$3,$3)`, schoolID, secondStudentID, actorPrincipalID)
	assertRejected("school registration number blank",
		`UPDATE student_basic_details SET school_registration_number='   ' WHERE id=$1`, detailID)
	assertRejected("basic details delete",
		`DELETE FROM student_basic_details WHERE id=$1`, detailID)
	assertRejected("basic details student identity rewrite",
		`UPDATE student_basic_details SET student_id=$2 WHERE id=$1`, detailID, secondStudentID)
	assertRejected("basic details creator rewrite",
		`UPDATE student_basic_details SET created_by_principal_id=created_by_principal_id+1 WHERE id=$1`, detailID)
}
