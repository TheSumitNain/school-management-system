package repository

import (
	"context"
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
	var yearID, classID, sectionID, subjectID, compulsorySubjectID, studentID, enrollmentID int64
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
		INSERT INTO subjects(school_id,name,subject_code) VALUES($1,$2,$3) RETURNING id
	`, schoolID, "V44 Optional "+suffix, "OPT-"+suffix).Scan(&subjectID); err != nil { t.Fatalf("create optional subject: %v", err) }
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

	var curriculumID, groupID, optionalCurriculumSubjectID, compulsoryCurriculumSubjectID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curricula(school_id,academic_year_id,class_id,version_number,created_by_principal_id)
		VALUES($1,$2,$3,1,$4) RETURNING id
	`, schoolID, yearID, classID, actorPrincipalID).Scan(&curriculumID); err != nil { t.Fatalf("create class curriculum: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_optional_groups(school_id,curriculum_id,name,min_selections,max_selections)
		VALUES($1,$2,'Language',1,1) RETURNING id
	`, schoolID, curriculumID).Scan(&groupID); err != nil { t.Fatalf("create optional group: %v", err) }
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curriculum_subjects(school_id,curriculum_id,subject_id,optional_group_id,subject_name_snapshot,subject_code_snapshot)
		VALUES($1,$2,$3,$4,'Optional Language','OPT-SNAPSHOT') RETURNING id
	`, schoolID, curriculumID, subjectID, groupID).Scan(&optionalCurriculumSubjectID); err != nil { t.Fatalf("create optional curriculum subject: %v", err) }
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

	academicRepo := &AcademicRepository{db: tx}
	curriculumView, err := academicRepo.GetStudentEnrollmentCurriculum(ctx, actorPrincipalID, schoolID, enrollmentID)
	if err != nil {
		t.Fatalf("read student enrollment curriculum: %v", err)
	}
	if curriculumView.EnrollmentID != enrollmentID || curriculumView.StudentID != studentID ||
		!curriculumView.Assigned || len(curriculumView.SelectedOptionalSubjectIDs) != 1 ||
		curriculumView.SelectedOptionalSubjectIDs[0] != optionalCurriculumSubjectID || len(curriculumView.SelectionHistory) != 1 {
		t.Fatalf("student enrollment curriculum view = %+v; expected assigned curriculum, one selected option and one history row", curriculumView)
	}
	var curriculumReadAuditCount, auditedSelectionCount, auditedHistoryCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::integer,
		       COALESCE(MAX((metadata->>'selected_optional_subject_count')::integer),-1),
		       COALESCE(MAX((metadata->>'selection_history_count')::integer),-1)
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.enrollment_curriculum.read'
		   AND resource_type='student_enrollment' AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(enrollmentID)).Scan(
		&curriculumReadAuditCount, &auditedSelectionCount, &auditedHistoryCount,
	); err != nil {
		t.Fatalf("read student enrollment curriculum audit: %v", err)
	}
	if curriculumReadAuditCount != 1 || auditedSelectionCount != 1 || auditedHistoryCount != 1 {
		t.Fatalf("student enrollment curriculum audit count=%d selected=%d history=%d; want 1, 1, 1",
			curriculumReadAuditCount, auditedSelectionCount, auditedHistoryCount)
	}

	expectRejected := func(label string, operation func() error) {
		t.Helper()
		savepoint, saveErr := tx.Begin(ctx)
		if saveErr != nil { t.Fatalf("begin savepoint for %s: %v", label, saveErr) }
		opErr := operation()
		if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil { t.Fatalf("rollback savepoint for %s: %v", label, rollbackErr) }
		if opErr == nil { t.Fatalf("%s unexpectedly succeeded", label) }
	}
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

	if _, err := tx.Exec(ctx, `UPDATE student_enrollment_subject_selections SET revoked_at=NOW() WHERE id=$1`, selectionID); err != nil {
		t.Fatalf("revoke optional subject choice: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE student_enrollment_curricula SET status='replaced',revoked_at=NOW() WHERE id=$1
	`, assignmentID); err != nil { t.Fatalf("replace curriculum after revoking choices: %v", err) }
	expectRejected("deleting curriculum assignment history", func() error {
		_, err := tx.Exec(ctx, `DELETE FROM student_enrollment_curricula WHERE id=$1`, assignmentID)
		return err
	})

	if _, err := tx.Exec(ctx, `UPDATE students SET status='inactive' WHERE id=$1`, studentID); err != nil { t.Fatalf("make student inactive: %v", err) }
	expectRejected("assigning curriculum to inactive student", func() error {
		_, err := tx.Exec(ctx, `
			INSERT INTO student_enrollment_curricula(school_id,enrollment_id,curriculum_id,assigned_by_principal_id)
			VALUES($1,$2,$3,$4)
		`, schoolID, enrollmentID, curriculumID, actorPrincipalID)
		return err
	})
}
