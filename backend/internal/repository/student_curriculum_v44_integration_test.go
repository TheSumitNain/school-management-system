package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestStudentCurriculumV44PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin rollback-scoped V44 test: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, actorPrincipalID int64
	if err := tx.QueryRow(ctx, `SELECT school_id, principal_id FROM users WHERE id=$1`, userID).Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load integration actor: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	var yearID, classID, sectionID, secondSectionID, subjectID, standaloneSubjectID, compulsorySubjectID, studentID, enrollmentID, secondStudentID, secondEnrollmentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active') RETURNING id
	`, schoolID, "V44-"+suffix).Scan(&yearID); err != nil { t.Fatalf("create academic year: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id
	`, schoolID, "V44 Class "+suffix).Scan(&classID); err != nil { t.Fatalf("create class: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,'A') RETURNING id
	`, schoolID, yearID, classID).Scan(&sectionID); err != nil { t.Fatalf("create section: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,'B') RETURNING id
	`, schoolID, yearID, classID).Scan(&secondSectionID); err != nil { t.Fatalf("create second section: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO subjects(school_id,name,subject_code) VALUES($1,$2,$3) RETURNING id
	`, schoolID, "V44 Optional "+suffix, "OPT-"+suffix).Scan(&subjectID); err != nil { t.Fatalf("create optional subject: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO subjects(school_id,name,subject_code) VALUES($1,$2,$3) RETURNING id
	`, schoolID, "V44 Standalone "+suffix, "STO-"+suffix).Scan(&standaloneSubjectID); err != nil { t.Fatalf("create standalone optional subject: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO subjects(school_id,name,subject_code) VALUES($1,$2,$3) RETURNING id
	`, schoolID, "V44 Compulsory "+suffix, "CMP-"+suffix).Scan(&compulsorySubjectID); err != nil { t.Fatalf("create compulsory subject: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,$2,'V44 Curriculum Student') RETURNING id
	`, schoolID, "V44-STU-"+suffix).Scan(&studentID); err != nil { t.Fatalf("create student: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollments
			(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date)
		VALUES($1,$2,$3,$4,$5,'01',$6,DATE '2026-04-01') RETURNING id
	`, schoolID, studentID, yearID, classID, sectionID, "V44-ENR-"+suffix).Scan(&enrollmentID); err != nil {
		t.Fatalf("create active enrollment: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,$2,'V44 Different Section Student') RETURNING id
	`, schoolID, "V44-STU-B-"+suffix).Scan(&secondStudentID); err != nil { t.Fatalf("create second student: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollments
			(school_id,student_id,academic_year_id,class_id,section_id,roll_number,admission_number,admission_date)
		VALUES($1,$2,$3,$4,$5,'02',$6,DATE '2026-04-01') RETURNING id
	`, schoolID, secondStudentID, yearID, classID, secondSectionID, "V44-ENR-B-"+suffix).Scan(&secondEnrollmentID); err != nil {
		t.Fatalf("create second-section enrollment: %v", err)
	}

	var curriculumID, groupID, optionalCurriculumSubjectID, standaloneCurriculumSubjectID, compulsoryCurriculumSubjectID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curricula(school_id,academic_year_id,class_id,section_id,version_number,created_by_principal_id)
		VALUES($1,$2,$3,$4,1,$5) RETURNING id
	`, schoolID, yearID, classID, sectionID, actorPrincipalID).Scan(&curriculumID); err != nil { t.Fatalf("create class curriculum: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_optional_groups(school_id,curriculum_id,name,min_selections,max_selections)
		VALUES($1,$2,'Language',1,1) RETURNING id
	`, schoolID, curriculumID).Scan(&groupID); err != nil { t.Fatalf("create optional group: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_subjects(school_id,curriculum_id,subject_id,optional_group_id,subject_name_snapshot,subject_code_snapshot)
		VALUES($1,$2,$3,$4,'Optional Language','OPT-SNAPSHOT') RETURNING id
	`, schoolID, curriculumID, subjectID, groupID).Scan(&optionalCurriculumSubjectID); err != nil { t.Fatalf("create optional curriculum subject: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_subjects(school_id,curriculum_id,subject_id,is_optional,subject_name_snapshot)
		VALUES($1,$2,$3,TRUE,'Standalone Optional') RETURNING id
	`, schoolID, curriculumID, standaloneSubjectID).Scan(&standaloneCurriculumSubjectID); err != nil { t.Fatalf("create standalone optional curriculum subject: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_subjects(school_id,curriculum_id,subject_id,subject_name_snapshot)
		VALUES($1,$2,$3,'Compulsory Language') RETURNING id
	`, schoolID, curriculumID, compulsorySubjectID).Scan(&compulsoryCurriculumSubjectID); err != nil { t.Fatalf("create compulsory curriculum subject: %v", err) }

	var assignmentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollment_curricula(school_id,enrollment_id,curriculum_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4) RETURNING id
	`, schoolID, enrollmentID, curriculumID, actorPrincipalID).Scan(&assignmentID); err != nil { t.Fatalf("assign curriculum to active enrollment: %v", err) }
	var selectionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollment_subject_selections
			(school_id,enrollment_curriculum_id,curriculum_id,curriculum_subject_id,
			 subject_name_snapshot,subject_code_snapshot,selected_by_principal_id)
		VALUES($1,$2,$3,$4,'Optional Language','OPT-SNAPSHOT',$5) RETURNING id
	`, schoolID, assignmentID, curriculumID, optionalCurriculumSubjectID, actorPrincipalID).Scan(&selectionID); err != nil {
		t.Fatalf("save optional subject choice: %v", err)
	}
	var standaloneSelectionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_enrollment_subject_selections
			(school_id,enrollment_curriculum_id,curriculum_id,curriculum_subject_id,
			 subject_name_snapshot,selected_by_principal_id)
		VALUES($1,$2,$3,$4,'Standalone Optional',$5) RETURNING id
	`, schoolID, assignmentID, curriculumID, standaloneCurriculumSubjectID, actorPrincipalID).Scan(&standaloneSelectionID); err != nil {
		t.Fatalf("save standalone optional subject choice: %v", err)
	}

	impacts, err := curriculumRemovalImpacts(ctx, tx, schoolID, curriculumID, []int64{subjectID, compulsorySubjectID})
	if err != nil {
		t.Fatalf("preview affected students for optional and compulsory subject removal: %v", err)
	}
	if len(impacts) != 2 {
		t.Fatalf("curriculum removal impacts = %d, want 2", len(impacts))
	}
	for _, impact := range impacts {
		if impact.AffectedStudents != 1 || len(impact.Students) != 1 {
			t.Fatalf("subject %d affected count/list = %d/%d, want 1/1", impact.SubjectID, impact.AffectedStudents, len(impact.Students))
		}
		student := impact.Students[0]
		if student.StudentID != studentID || student.AdmissionNumber != "V44-ENR-"+suffix || student.FullName != "V44 Curriculum Student" {
			t.Fatalf("subject %d affected student = %+v, want the enrolled student details", impact.SubjectID, student)
		}
	}

	// Give the integration actor the production School Admin grant so the
	// read path is exercised with both students.read_basic and subjects.read.
	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id=$1 AND principal_type='staff'
		   AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load system School Admin role: %v", err)
	}
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id=pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id=pr.role_id
		  JOIN role_permissions rp ON rp.role_id=r.id
		  JOIN permissions p ON p.id=rp.permission_id
		 WHERE ap.principal_type='platform_admin' AND ap.status='active'
		   AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
		   AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if err != nil {
		var bootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS(SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
			              WHERE r.principal_type='platform_admin' AND r.scope_type='platform'
			                AND r.role_key='platform_admin')
		`).Scan(&bootstrapExists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if bootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("curriculum-v44-%s@example.invalid", suffix), "integration-test-password-hash",
		)
		if err != nil {
			t.Fatalf("create rollback-scoped Platform Admin fixture: %v", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("grant School Admin role to fixture actor: %v", err)
	}

	academicRepo := &AcademicRepository{db: tx}
	if _, err := academicRepo.SaveSectionClassCurriculumWithConfirmations(
		ctx, actorPrincipalID, schoolID, yearID, classID, sectionID,
		[]CurriculumOptionalGroupInput{{Name: "Empty group", MinSelections: 1, MaxSelections: 1}},
		[]CurriculumSubjectInput{{SubjectID: compulsorySubjectID}},
		nil,
	); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("saving an optional group with no assigned subjects returned %v, want invalid academic input", err)
	}

	curriculumView, err := academicRepo.GetStudentEnrollmentCurriculum(ctx, actorPrincipalID, schoolID, enrollmentID)
	if err != nil {
		t.Fatalf("read student enrollment curriculum: %v", err)
	}

	// Curriculum and selected-subject data must remain isolated even when a
	// caller supplies another school's scope with valid local record IDs.
	foreignSchoolID := schoolID + 999999
	if _, err := academicRepo.GetStudentEnrollmentCurriculum(ctx, actorPrincipalID, foreignSchoolID, enrollmentID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school enrollment curriculum read error = %v, want access denied", err)
	}
	if _, err := academicRepo.SaveStudentEnrollmentSubjects(ctx, actorPrincipalID, foreignSchoolID,
		enrollmentID, curriculumID, []int64{optionalCurriculumSubjectID}); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school enrollment subject update error = %v, want access denied", err)
	}
	if _, err := academicRepo.ListClassCurricula(ctx, actorPrincipalID, foreignSchoolID, yearID, classID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school class curriculum list error = %v, want access denied", err)
	}
	if _, err := academicRepo.SaveClassCurriculum(ctx, actorPrincipalID, foreignSchoolID, yearID, classID,
		nil, []CurriculumSubjectInput{{SubjectID: subjectID}}); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school class curriculum save error = %v, want access denied", err)
	}

	if curriculumView.EnrollmentID != enrollmentID || curriculumView.StudentID != studentID ||
		!curriculumView.Assigned || len(curriculumView.SelectedOptionalSubjectIDs) != 2 {
		t.Fatalf("student enrollment curriculum view = %+v; expected assigned curriculum with two current optional choices and no history field", curriculumView)
	}
	var curriculumReadAuditCount, auditedSelectionCount int
	var hasHistoryAuditField bool
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::integer,
		       COALESCE(MAX((metadata->>'selected_optional_subject_count')::integer),-1),
		       COALESCE(BOOL_OR(metadata ? 'selection_history_count'),false)
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.enrollment_curriculum.read'
		   AND resource_type='student_enrollment' AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(enrollmentID)).Scan(
		&curriculumReadAuditCount, &auditedSelectionCount, &hasHistoryAuditField,
	); err != nil {
		t.Fatalf("read student enrollment curriculum audit: %v", err)
	}
	if curriculumReadAuditCount != 1 || auditedSelectionCount != 2 || hasHistoryAuditField {
		t.Fatalf("student enrollment curriculum audit count=%d selected=%d contains_history_field=%t; want 1, 2, false",
			curriculumReadAuditCount, auditedSelectionCount, hasHistoryAuditField)
	}

	expectRejected := func(label string, operation func() error) {
		t.Helper()
		savepoint, saveErr := tx.Begin(ctx)
		if saveErr != nil { t.Fatalf("begin savepoint for %s: %v", label, saveErr) }
		opErr := operation()
		if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil { t.Fatalf("rollback savepoint for %s: %v", label, rollbackErr) }
		if opErr == nil { t.Fatalf("%s unexpectedly succeeded", label) }
	}
	expectRejected("assigning a different section's curriculum", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO student_enrollment_curricula(school_id,enrollment_id,curriculum_id,assigned_by_principal_id)
			VALUES($1,$2,$3,$4)
		`, schoolID, secondEnrollmentID, curriculumID, actorPrincipalID)
		return err
	})
	expectRejected("editing curriculum subject after assignment", func() error {
		_, err := tx.Exec(ctx, `UPDATE class_curriculum_subjects SET subject_name_snapshot='Changed' WHERE id=$1`, optionalCurriculumSubjectID)
		return err
	})
	expectRejected("selecting a compulsory subject as optional", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO student_enrollment_subject_selections
				(school_id,enrollment_curriculum_id,curriculum_id,curriculum_subject_id,
				 subject_name_snapshot,selected_by_principal_id)
			VALUES($1,$2,$3,$4,'Compulsory Language',$5)
		`, schoolID, assignmentID, curriculumID, compulsoryCurriculumSubjectID, actorPrincipalID)
		return err
	})
	expectRejected("replacing curriculum before closing optional choices", func() error {
		_, err := tx.Exec(ctx, `
			UPDATE student_enrollment_curricula SET status='replaced',revoked_at=NOW() WHERE id=$1
		`, assignmentID)
		return err
	})

	// An already-selected optional subject remains preservable after its master
	// subject is deactivated. The student may also remove that choice, but may
	// not select it again after removal.
	if _, err := tx.Exec(ctx, `UPDATE subjects SET is_active=FALSE WHERE id=$1`, standaloneSubjectID); err != nil {
		t.Fatalf("deactivate selected optional master subject: %v", err)
	}
	preserved, err := academicRepo.SaveStudentEnrollmentSubjects(ctx, actorPrincipalID, schoolID,
		enrollmentID, curriculumID, []int64{optionalCurriculumSubjectID, standaloneCurriculumSubjectID})
	if err != nil {
		t.Fatalf("preserve already-selected inactive optional subject: %v", err)
	}
	if len(preserved.SelectedOptionalSubjectIDs) != 2 {
		t.Fatalf("selected subjects after preserving inactive choice = %v, want both existing choices", preserved.SelectedOptionalSubjectIDs)
	}
	removed, err := academicRepo.SaveStudentEnrollmentSubjects(ctx, actorPrincipalID, schoolID,
		enrollmentID, curriculumID, []int64{optionalCurriculumSubjectID})
	if err != nil {
		t.Fatalf("remove previously selected inactive optional subject: %v", err)
	}
	if len(removed.SelectedOptionalSubjectIDs) != 1 || removed.SelectedOptionalSubjectIDs[0] != optionalCurriculumSubjectID {
		t.Fatalf("selected subjects after removing inactive choice = %v, want only %d", removed.SelectedOptionalSubjectIDs, optionalCurriculumSubjectID)
	}
	if _, err := academicRepo.SaveStudentEnrollmentSubjects(ctx, actorPrincipalID, schoolID,
		enrollmentID, curriculumID, []int64{optionalCurriculumSubjectID, standaloneCurriculumSubjectID}); !errors.Is(err, ErrInactiveSubjectSelection) {
		t.Fatalf("reselecting removed inactive optional subject error = %v, want ErrInactiveSubjectSelection", err)
	}

	// Subject choices are current state, not history. Remove both rows before
	// replacing the enrollment's curriculum assignment.
	if _, err := tx.Exec(ctx, `DELETE FROM student_enrollment_subject_selections WHERE enrollment_curriculum_id=$1`, assignmentID); err != nil {
		t.Fatalf("delete current optional subject choices: %v", err)
	}
	var remainingChoices int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*)::integer FROM student_enrollment_subject_selections WHERE enrollment_curriculum_id=$1`, assignmentID).Scan(&remainingChoices); err != nil {
		t.Fatalf("count current optional subject choices after deletion: %v", err)
	}
	if remainingChoices != 0 {
		t.Fatalf("current optional subject choices remaining = %d, want 0", remainingChoices)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_enrollment_curricula SET status='replaced',revoked_at=NOW() WHERE id=$1
	`, assignmentID); err != nil { t.Fatalf("replace curriculum after revoking choices: %v", err) }
	expectRejected("deleting curriculum assignment history", func() error {
		_, err := tx.Exec(ctx, `DELETE FROM student_enrollment_curricula WHERE id=$1`, assignmentID)
		return err
	})

	_, err = academicRepo.SaveSectionClassCurriculumWithConfirmations(
		ctx, actorPrincipalID, schoolID, yearID, classID, sectionID,
		[]CurriculumOptionalGroupInput{{Name: "Language", MinSelections: 1, MaxSelections: 1}},
		[]CurriculumSubjectInput{{SubjectID: subjectID, IsOptional: true, OptionalGroup: "Language"}},
		[]CurriculumRemovalConfirmation{
			{SubjectID: standaloneSubjectID, AffectedStudents: 0, StudentIDs: []int64{}},
			{SubjectID: compulsorySubjectID, AffectedStudents: 1, StudentIDs: []int64{secondStudentID}},
		},
	)
	if !errors.Is(err, ErrCurriculumRemovalImpactChanged) {
		t.Fatalf("same-count confirmation with a different student ID returned %v, want stale impact", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE students SET status='inactive' WHERE id=$1`, studentID); err != nil { t.Fatalf("make student inactive: %v", err) }
	expectRejected("assigning curriculum to inactive student", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO student_enrollment_curricula(school_id,enrollment_id,curriculum_id,assigned_by_principal_id)
			VALUES($1,$2,$3,$4)
		`, schoolID, enrollmentID, curriculumID, actorPrincipalID)
		return err
	})
}
