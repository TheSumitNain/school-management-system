package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type CurriculumOptionalGroupInput struct {
	Name          string `json:"name"`
	MinSelections int    `json:"min_selections"`
	MaxSelections int    `json:"max_selections"`
}

type CurriculumSubjectInput struct {
	SubjectID     int64  `json:"subject_id"`
	OptionalGroup string `json:"optional_group,omitempty"`
	IsOptional    bool   `json:"is_optional,omitempty"`
}

type CurriculumRemovalConfirmation struct {
	SubjectID int64 `json:"subject_id"`
	AffectedStudents int `json:"affected_students"`
}

type CurriculumRemovalImpact struct {
	SubjectID int64 `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	AffectedStudents int `json:"affected_students"`
}

type CurriculumOptionalGroup struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	MinSelections int    `json:"min_selections"`
	MaxSelections int    `json:"max_selections"`
}

type CurriculumSubject struct {
	ID                int64   `json:"id"`
	SubjectID         int64   `json:"subject_id"`
	Name              string  `json:"name"`
	Code              *string `json:"code,omitempty"`
	OptionalGroupID   *int64  `json:"optional_group_id,omitempty"`
	OptionalGroupName *string `json:"optional_group_name,omitempty"`
	MasterActive      bool    `json:"master_active"`
	IsCompulsory      bool    `json:"is_compulsory"`
	IsOptional        bool    `json:"is_optional"`
}

type ClassCurriculum struct {
	ID             int64                     `json:"id"`
	SchoolID       int64                     `json:"school_id"`
	AcademicYearID int64                     `json:"academic_year_id"`
	ClassID        int64                     `json:"class_id"`
	SectionID      *int64                    `json:"section_id,omitempty"`
	VersionNumber  int                       `json:"version_number"`
	Status         string                    `json:"status"`
	CreatedAt      time.Time                 `json:"created_at"`
	Groups         []CurriculumOptionalGroup `json:"optional_groups"`
	Subjects       []CurriculumSubject       `json:"subjects"`
}

type StudentEnrollmentCurriculum struct {
	EnrollmentID               int64                            `json:"enrollment_id"`
	StudentID                  int64                            `json:"student_id"`
	StudentStatus              string                           `json:"student_status"`
	EnrollmentStatus           string                           `json:"enrollment_status"`
	AcademicYearID             int64                            `json:"academic_year_id"`
	ClassID                    int64                            `json:"class_id"`
	SectionID                  int64                            `json:"section_id"`
	Curriculum                 *ClassCurriculum                 `json:"curriculum,omitempty"`
	AvailableCurriculum        *ClassCurriculum                 `json:"available_curriculum,omitempty"`
	Assigned                   bool                             `json:"assigned"`
	SelectedOptionalSubjectIDs []int64                          `json:"selected_optional_subject_ids"`
}

func loadClassCurriculum(ctx context.Context, db repositoryDB, curriculumID, schoolID int64) (*ClassCurriculum, error) {
	var item ClassCurriculum
	err := db.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, class_id, section_id, version_number, status, created_at
		  FROM class_curricula
		 WHERE id=$1 AND school_id=$2
	`, curriculumID, schoolID).Scan(
		&item.ID, &item.SchoolID, &item.AcademicYearID, &item.ClassID, &item.SectionID,
		&item.VersionNumber, &item.Status, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAcademicNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load class curriculum: %w", err)
	}
	item.Groups = make([]CurriculumOptionalGroup, 0)
	groupRows, err := db.Query(ctx, `
		SELECT id, name, min_selections, max_selections
		  FROM class_curriculum_optional_groups
		 WHERE curriculum_id=$1 AND school_id=$2
           AND EXISTS (SELECT 1 FROM class_curriculum_subjects cs WHERE cs.optional_group_id=class_curriculum_optional_groups.id AND cs.curriculum_id=class_curriculum_optional_groups.curriculum_id AND cs.school_id=class_curriculum_optional_groups.school_id AND cs.is_active=TRUE)
		 ORDER BY lower(name), id
	`, curriculumID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list curriculum optional groups: %w", err)
	}
	for groupRows.Next() {
		var group CurriculumOptionalGroup
		if err := groupRows.Scan(&group.ID, &group.Name, &group.MinSelections, &group.MaxSelections); err != nil {
			groupRows.Close()
			return nil, fmt.Errorf("scan curriculum optional group: %w", err)
		}
		item.Groups = append(item.Groups, group)
	}
	if err := groupRows.Err(); err != nil {
		groupRows.Close()
		return nil, fmt.Errorf("iterate curriculum optional groups: %w", err)
	}
	groupRows.Close()

	item.Subjects = make([]CurriculumSubject, 0)
	subjectRows, err := db.Query(ctx, `
		SELECT cs.id, cs.subject_id, cs.subject_name_snapshot, cs.subject_code_snapshot,
		       cs.optional_group_id, g.name, cs.is_optional, s.is_active
		  FROM class_curriculum_subjects cs
		  JOIN subjects s ON s.id=cs.subject_id AND s.school_id=cs.school_id
		  LEFT JOIN class_curriculum_optional_groups g
		    ON g.id=cs.optional_group_id AND g.curriculum_id=cs.curriculum_id AND g.school_id=cs.school_id
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2 AND cs.is_active=TRUE
		 ORDER BY (cs.optional_group_id IS NOT NULL), lower(COALESCE(g.name,'')), lower(cs.subject_name_snapshot), cs.id
	`, curriculumID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list curriculum subjects: %w", err)
	}
	for subjectRows.Next() {
		var subject CurriculumSubject
		if err := subjectRows.Scan(&subject.ID, &subject.SubjectID, &subject.Name, &subject.Code,
			&subject.OptionalGroupID, &subject.OptionalGroupName, &subject.IsOptional, &subject.MasterActive); err != nil {
			subjectRows.Close()
			return nil, fmt.Errorf("scan curriculum subject: %w", err)
		}
		subject.IsCompulsory = !subject.IsOptional
		item.Subjects = append(item.Subjects, subject)
	}
	if err := subjectRows.Err(); err != nil {
		subjectRows.Close()
		return nil, fmt.Errorf("iterate curriculum subjects: %w", err)
	}
	subjectRows.Close()
	return &item, nil
}

func (r *AcademicRepository) ListClassCurricula(ctx context.Context, actorID, schoolID, academicYearID, classID int64) ([]ClassCurriculum, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if academicYearID <= 0 || classID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "subjects.read"); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id FROM class_curricula
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3
		 ORDER BY version_number DESC
	`, schoolID, academicYearID, classID)
	if err != nil {
		return nil, fmt.Errorf("list class curricula: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan class curriculum ID: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate class curricula: %w", err)
	}
	items := make([]ClassCurriculum, 0, len(ids))
	for _, id := range ids {
		item, err := loadClassCurriculum(ctx, r.db, id, schoolID)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *AcademicRepository) SaveClassCurriculum(
	ctx context.Context, actorID, schoolID, academicYearID, classID int64,
	groups []CurriculumOptionalGroupInput, subjects []CurriculumSubjectInput,
) (*ClassCurriculum, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "subjects.manage"); err != nil { return nil, err }
	// New curriculum writes must always be scoped to an explicit section.
	return nil, ErrInvalidAcademicInput
}

func (r *AcademicRepository) ListSectionClassCurricula(ctx context.Context, actorID, schoolID, academicYearID, classID, sectionID int64) ([]ClassCurriculum, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if academicYearID <= 0 || classID <= 0 || sectionID <= 0 { return nil, ErrInvalidAcademicInput }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "subjects.read"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `
		SELECT id FROM class_curricula
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND section_id=$4 AND status='active'
		 ORDER BY version_number DESC
	`, schoolID, academicYearID, classID, sectionID)
	if err != nil { return nil, fmt.Errorf("list section curricula: %w", err) }
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() { var id int64; if err := rows.Scan(&id); err != nil { return nil, fmt.Errorf("scan section curriculum ID: %w", err) }; ids = append(ids, id) }
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate section curricula: %w", err) }
	rows.Close()
	items := make([]ClassCurriculum, 0, len(ids))
	for _, id := range ids { item, err := loadClassCurriculum(ctx, r.db, id, schoolID); if err != nil { return nil, err }; items = append(items, *item) }
	return items, nil
}

func (r *AcademicRepository) SaveSectionClassCurriculum(ctx context.Context, actorID, schoolID, academicYearID, classID, sectionID int64, groups []CurriculumOptionalGroupInput, subjects []CurriculumSubjectInput) (*ClassCurriculum, error) {
	return r.SaveSectionClassCurriculumWithConfirmations(ctx, actorID, schoolID, academicYearID, classID, sectionID, groups, subjects, nil)
}

func (r *AcademicRepository) PreviewSectionCurriculumRemoval(ctx context.Context, actorID, schoolID, academicYearID, classID, sectionID int64, subjectIDs []int64) ([]CurriculumRemovalImpact, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if academicYearID <= 0 || classID <= 0 || sectionID <= 0 || len(subjectIDs) == 0 || len(subjectIDs) > 150 { return nil, ErrInvalidAcademicInput }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "subjects.manage"); err != nil { return nil, err }
	seen := make(map[int64]struct{}, len(subjectIDs))
	for _, id := range subjectIDs { if id <= 0 { return nil, ErrInvalidAcademicInput }; if _, ok := seen[id]; ok { return nil, ErrInvalidAcademicInput }; seen[id] = struct{}{} }
	var curriculumID int64
	err := r.db.QueryRow(ctx, `
		SELECT id FROM class_curricula
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND section_id=$4 AND status='active'
	`, schoolID, academicYearID, classID, sectionID).Scan(&curriculumID)
	if errors.Is(err, pgx.ErrNoRows) { return []CurriculumRemovalImpact{}, nil }
	if err != nil { return nil, fmt.Errorf("load curriculum for removal preview: %w", err) }
	return curriculumRemovalImpacts(ctx, r.db, schoolID, curriculumID, subjectIDs)
}

func curriculumRemovalImpacts(ctx context.Context, db repositoryDB, schoolID, curriculumID int64, subjectIDs []int64) ([]CurriculumRemovalImpact, error) {
	rows, err := db.Query(ctx, `
		SELECT cs.subject_id, cs.subject_name_snapshot,
		       CASE WHEN cs.is_optional
		            THEN COUNT(DISTINCT e.student_id) FILTER (WHERE ec.curriculum_id=cs.curriculum_id AND s.id IS NOT NULL AND st.id IS NOT NULL)
		            ELSE COUNT(DISTINCT e.student_id) FILTER (WHERE st.id IS NOT NULL AND (ec.id IS NULL OR ec.curriculum_id=cs.curriculum_id))
		       END::integer AS affected_students
		  FROM class_curriculum_subjects cs
		  JOIN class_curricula cc
		    ON cc.id=cs.curriculum_id AND cc.school_id=cs.school_id
		  LEFT JOIN student_enrollments e
		    ON e.school_id=cc.school_id AND e.academic_year_id=cc.academic_year_id
		   AND e.class_id=cc.class_id AND e.section_id=cc.section_id AND e.status='active'
		  LEFT JOIN students st
		    ON st.id=e.student_id AND st.school_id=e.school_id AND st.status='active'
		  LEFT JOIN student_enrollment_curricula ec
		    ON ec.enrollment_id=e.id AND ec.school_id=e.school_id AND ec.status='active'
		  LEFT JOIN student_enrollment_subject_selections s
		    ON s.enrollment_curriculum_id=ec.id AND s.curriculum_id=cs.curriculum_id
		   AND s.curriculum_subject_id=cs.id AND s.school_id=cs.school_id AND s.revoked_at IS NULL
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2 AND cs.is_active=TRUE
		   AND cs.subject_id=ANY($3::bigint[])
		 GROUP BY cs.subject_id, cs.subject_name_snapshot, cs.is_optional
		 ORDER BY lower(cs.subject_name_snapshot), cs.subject_id
	`, curriculumID, schoolID, subjectIDs)
	if err != nil { return nil, fmt.Errorf("calculate curriculum removal impact: %w", err) }
	defer rows.Close()
	items := make([]CurriculumRemovalImpact, 0)
	for rows.Next() {
		var item CurriculumRemovalImpact
		if err := rows.Scan(&item.SubjectID, &item.SubjectName, &item.AffectedStudents); err != nil { return nil, fmt.Errorf("scan curriculum removal impact: %w", err) }
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate curriculum removal impact: %w", err) }
	return items, nil
}

func (r *AcademicRepository) SaveSectionClassCurriculumWithConfirmations(ctx context.Context, actorID, schoolID, academicYearID, classID, sectionID int64, groups []CurriculumOptionalGroupInput, subjects []CurriculumSubjectInput, confirmations []CurriculumRemovalConfirmation) (*ClassCurriculum, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if schoolID <= 0 || academicYearID <= 0 || classID <= 0 || sectionID <= 0 || len(subjects) == 0 || len(subjects) > 150 || len(groups) > 30 { return nil, ErrInvalidAcademicInput }

	groupByName := make(map[string]CurriculumOptionalGroupInput, len(groups))
	groupCounts := make(map[string]int, len(groups))
	for _, group := range groups {
		group.Name = strings.TrimSpace(group.Name)
		key := strings.ToLower(group.Name)
		if len([]rune(group.Name)) < 2 || len([]rune(group.Name)) > 100 || group.MinSelections < 0 || group.MaxSelections < 1 || group.MinSelections > group.MaxSelections || key == "" { return nil, ErrInvalidAcademicInput }
		if _, exists := groupByName[key]; exists { return nil, ErrInvalidAcademicInput }
		groupByName[key] = group
	}
	seenSubjects := make(map[int64]struct{}, len(subjects))
	normalizedSubjects := make([]CurriculumSubjectInput, 0, len(subjects))
	for _, subject := range subjects {
		if subject.SubjectID <= 0 { return nil, ErrInvalidAcademicInput }
		if _, exists := seenSubjects[subject.SubjectID]; exists { return nil, ErrInvalidAcademicInput }
		seenSubjects[subject.SubjectID] = struct{}{}
		subject.OptionalGroup = strings.TrimSpace(subject.OptionalGroup)
		if subject.OptionalGroup != "" {
			subject.IsOptional = true
			key := strings.ToLower(subject.OptionalGroup)
			if _, exists := groupByName[key]; !exists { return nil, ErrInvalidAcademicInput }
			groupCounts[key]++
		}
		normalizedSubjects = append(normalizedSubjects, subject)
	}
	for key, group := range groupByName {
		count := groupCounts[key]
		if count == 0 || group.MinSelections > count || group.MaxSelections > count { return nil, ErrInvalidAcademicInput }
	}

	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin editable class curriculum save: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "subjects.manage"); err != nil { return nil, err }
	var lockedClassID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM classes WHERE id=$1 AND school_id=$2 AND is_active=TRUE FOR UPDATE`, classID, schoolID).Scan(&lockedClassID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }; return nil, fmt.Errorf("lock curriculum class: %w", err)
	}
	var lockedYearID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM academic_years WHERE id=$1 AND school_id=$2 AND status <> 'closed' FOR UPDATE`, academicYearID, schoolID).Scan(&lockedYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }; return nil, fmt.Errorf("lock curriculum academic year: %w", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND class_id=$4 AND is_active=TRUE FOR UPDATE
	`, sectionID, schoolID, academicYearID, classID).Scan(&sectionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }; return nil, fmt.Errorf("lock curriculum section: %w", err)
	}

	subjectIDs := make([]int64, 0, len(normalizedSubjects))
	for _, subject := range normalizedSubjects { subjectIDs = append(subjectIDs, subject.SubjectID) }
	var activeSubjectCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM subjects WHERE school_id=$1 AND is_active=TRUE AND id=ANY($2::bigint[])`, schoolID, subjectIDs).Scan(&activeSubjectCount); err != nil { return nil, fmt.Errorf("validate curriculum subjects: %w", err) }
	if activeSubjectCount != len(subjectIDs) { return nil, ErrAcademicNotFound }

	var curriculumID int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM class_curricula
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND section_id=$4 AND status='active'
		 FOR UPDATE
	`, schoolID, academicYearID, classID, sectionID).Scan(&curriculumID)
	created := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !created { return nil, fmt.Errorf("lock existing section curriculum: %w", err) }
	if created {
		if len(confirmations) != 0 { return nil, ErrInvalidAcademicInput }
		if err := tx.QueryRow(ctx, `
			INSERT INTO class_curricula(school_id,academic_year_id,class_id,section_id,version_number,status,created_by_principal_id)
			VALUES($1,$2,$3,$4,1,'active',$5) RETURNING id
		`, schoolID, academicYearID, classID, sectionID, actorID).Scan(&curriculumID); err != nil { return nil, academicWriteError(err) }
	}

	type existingSubject struct { id int64; active bool }
	existing := make(map[int64]existingSubject)
	if !created {
		rows, err := tx.Query(ctx, `
			SELECT id,subject_id,is_active
			  FROM class_curriculum_subjects WHERE curriculum_id=$1 AND school_id=$2 FOR UPDATE
		`, curriculumID, schoolID)
		if err != nil { return nil, fmt.Errorf("lock existing curriculum subjects: %w", err) }
		for rows.Next() {
			var item existingSubject
			var subjectID int64
			if err := rows.Scan(&item.id,&subjectID,&item.active); err != nil { rows.Close(); return nil, fmt.Errorf("scan existing curriculum subject: %w", err) }
			existing[subjectID] = item
		}
		if err := rows.Err(); err != nil { rows.Close(); return nil, fmt.Errorf("iterate existing curriculum subjects: %w", err) }
		rows.Close()
	}

	removedIDs := make([]int64, 0)
	removedCurriculumSubjectIDs := make([]int64, 0)
	for subjectID, item := range existing {
		if _, keep := seenSubjects[subjectID]; !keep {
			removedCurriculumSubjectIDs = append(removedCurriculumSubjectIDs, item.id)
			if item.active {
				removedIDs = append(removedIDs, subjectID)
			}
		}
	}
	if len(removedIDs) > 0 {
		impacts, err := curriculumRemovalImpacts(ctx, tx, schoolID, curriculumID, removedIDs)
		if err != nil { return nil, err }
		confirmed := make(map[int64]int, len(confirmations))
		for _, item := range confirmations {
			if item.SubjectID <= 0 || item.AffectedStudents < 0 { return nil, ErrInvalidAcademicInput }
			if _, duplicate := confirmed[item.SubjectID]; duplicate { return nil, ErrInvalidAcademicInput }
			confirmed[item.SubjectID] = item.AffectedStudents
		}
		if len(confirmed) != len(impacts) { return nil, ErrCurriculumRemovalConfirmationRequired }
		for _, impact := range impacts {
			count, ok := confirmed[impact.SubjectID]
			if !ok { return nil, ErrCurriculumRemovalConfirmationRequired }
			if count != impact.AffectedStudents { return nil, ErrCurriculumRemovalImpactChanged }
		}
	} else if len(confirmations) > 0 {
		return nil, ErrInvalidAcademicInput
	}
	if len(removedCurriculumSubjectIDs) > 0 {
		// Confirmed active removals and cleanup of legacy soft-deactivated
		// subjects permanently remove current subject rows and linked choices.
		if _, err := tx.Exec(ctx, `
			DELETE FROM student_enrollment_subject_selections
			 WHERE school_id=$1 AND curriculum_subject_id=ANY($2::bigint[])
		`, schoolID, removedCurriculumSubjectIDs); err != nil { return nil, academicWriteError(err) }
		if _, err := tx.Exec(ctx, `
			DELETE FROM class_curriculum_subjects
			 WHERE curriculum_id=$1 AND school_id=$2 AND id=ANY($3::bigint[])
		`, curriculumID, schoolID, removedCurriculumSubjectIDs); err != nil { return nil, academicWriteError(err) }
	}

	groupIDs := make(map[string]int64, len(groupByName))
	for key, group := range groupByName {
		var groupID int64
		groupErr := tx.QueryRow(ctx, `
			SELECT id FROM class_curriculum_optional_groups
			 WHERE curriculum_id=$1 AND school_id=$2 AND lower(btrim(name))=$3
			 FOR UPDATE
		`, curriculumID, schoolID, key).Scan(&groupID)
		if errors.Is(groupErr, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx, `
				INSERT INTO class_curriculum_optional_groups(school_id,curriculum_id,name,min_selections,max_selections)
				VALUES($1,$2,$3,$4,$5) RETURNING id
			`, schoolID, curriculumID, group.Name, group.MinSelections, group.MaxSelections).Scan(&groupID); err != nil { return nil, academicWriteError(err) }
		} else if groupErr != nil { return nil, fmt.Errorf("load curriculum optional group: %w", groupErr)
		} else if _, err := tx.Exec(ctx, `
			UPDATE class_curriculum_optional_groups SET name=$3,min_selections=$4,max_selections=$5
			 WHERE id=$1 AND school_id=$2
		`, groupID, schoolID, group.Name, group.MinSelections, group.MaxSelections); err != nil { return nil, academicWriteError(err) }
		groupIDs[key] = groupID
	}

	for _, subject := range normalizedSubjects {
		var groupID *int64
		if subject.OptionalGroup != "" { id := groupIDs[strings.ToLower(subject.OptionalGroup)]; groupID = &id }
		if old, ok := existing[subject.SubjectID]; ok {
			if _, err := tx.Exec(ctx, `
				UPDATE class_curriculum_subjects
				   SET optional_group_id=$3,is_optional=$4,is_active=TRUE
				 WHERE id=$1 AND school_id=$2
			`, old.id, schoolID, groupID, subject.IsOptional); err != nil { return nil, academicWriteError(err) }
		} else {
			if _, err := tx.Exec(ctx, `
				INSERT INTO class_curriculum_subjects
					(school_id,curriculum_id,subject_id,optional_group_id,is_optional,subject_name_snapshot,subject_code_snapshot)
				SELECT $1,$2,id,$3,$4,name,subject_code FROM subjects
				 WHERE id=$5 AND school_id=$1 AND is_active=TRUE
			`, schoolID, curriculumID, groupID, subject.IsOptional, subject.SubjectID); err != nil { return nil, academicWriteError(err) }
		}
	}

	// Groups removed in the editor are deleted after their subjects have been
	// ungrouped or removed. The database guard permits deletion only when no
	// curriculum subject still references the group.
	if _, err := tx.Exec(ctx, `
		DELETE FROM class_curriculum_optional_groups g
		 WHERE g.curriculum_id=$1 AND g.school_id=$2
		   AND NOT EXISTS (
		       SELECT 1 FROM class_curriculum_subjects cs
		        WHERE cs.optional_group_id=g.id
		          AND cs.curriculum_id=g.curriculum_id
		          AND cs.school_id=g.school_id
		   )
	`, curriculumID, schoolID); err != nil { return nil, academicWriteError(err) }

	action := "school.class_curriculum.created"
	if !created { action = "school.class_curriculum.updated" }
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'class_curriculum',$4,'success',
		       jsonb_build_object('academic_year_id',$5::bigint,'class_id',$6::bigint,'section_id',$7::bigint,
		                          'subject_count',$8::integer,'optional_group_count',$9::integer,
		                          'removed_subject_count',$10::integer))
	`, actorID, schoolID, action, fmt.Sprint(curriculumID), academicYearID, classID, sectionID,
		len(subjects), len(groups), len(removedCurriculumSubjectIDs)); err != nil {
		return nil, fmt.Errorf("audit editable class curriculum save: %w", err)
	}
	if err := tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit editable class curriculum save: %w", err) }
	return loadClassCurriculum(ctx, r.db, curriculumID, schoolID)
}

func (r *AcademicRepository) GetStudentEnrollmentCurriculum(ctx context.Context, actorID, schoolID, enrollmentID int64) (*StudentEnrollmentCurriculum, error) {
	item, err := r.loadStudentEnrollmentCurriculum(ctx, actorID, schoolID, enrollmentID)
	if err != nil {
		return nil, err
	}
	// Student subject choices and their history are personal academic records.
	// Do not return them unless their dedicated read audit is persisted.
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student enrollment curriculum read audit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_basic"); err != nil {
		return nil, err
	}
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "subjects.read"); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.enrollment_curriculum.read','student_enrollment',$3,'success',
		       jsonb_build_object('selected_optional_subject_count',$4::integer,
		                          'has_curriculum',$5::boolean))
	`, actorID, schoolID, fmt.Sprint(enrollmentID), len(item.SelectedOptionalSubjectIDs),
		item.Curriculum != nil); err != nil {
		return nil, fmt.Errorf("audit student enrollment curriculum read: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student enrollment curriculum read audit: %w", err)
	}
	return item, nil
}

func (r *AcademicRepository) loadStudentEnrollmentCurriculum(ctx context.Context, actorID, schoolID, enrollmentID int64) (*StudentEnrollmentCurriculum, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if enrollmentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "subjects.read"); err != nil {
		return nil, err
	}
	var studentID int64
	if err := r.db.QueryRow(ctx, `
		SELECT student_id FROM student_enrollments WHERE id=$1 AND school_id=$2
	`, enrollmentID, schoolID).Scan(&studentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("find enrollment student for subject view: %w", err)
	}
	visibleEnrollments, err := r.ListStudentEnrollments(ctx, actorID, schoolID, 0, studentID)
	if err != nil {
		return nil, err
	}
	visible := false
	for _, enrollment := range visibleEnrollments {
		if enrollment.ID == enrollmentID {
			visible = true
			break
		}
	}
	if !visible {
		return nil, ErrAcademicNotFound
	}
	var item StudentEnrollmentCurriculum
	item.EnrollmentID = enrollmentID
	item.StudentID = studentID
	item.SelectedOptionalSubjectIDs = make([]int64, 0)
	if err := r.db.QueryRow(ctx, `
		SELECT e.academic_year_id,e.class_id,e.section_id,e.status,st.status
		  FROM student_enrollments e
		  JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		 WHERE e.id=$1 AND e.school_id=$2
	`, enrollmentID, schoolID).Scan(&item.AcademicYearID, &item.ClassID, &item.SectionID, &item.EnrollmentStatus, &item.StudentStatus); err != nil {
		return nil, fmt.Errorf("load student enrollment curriculum context: %w", err)
	}
	var assignmentID, curriculumID int64
	err = r.db.QueryRow(ctx, `
		SELECT id,curriculum_id FROM student_enrollment_curricula
		 WHERE enrollment_id=$1 AND school_id=$2 AND status='active'
	`, enrollmentID, schoolID).Scan(&assignmentID, &curriculumID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx, `
			SELECT id FROM class_curricula
			 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND section_id=$4 AND status='active'
		`, schoolID, item.AcademicYearID, item.ClassID, item.SectionID).Scan(&curriculumID)
		if errors.Is(err, pgx.ErrNoRows) {
			return &item, nil
		}
		if err != nil {
			return nil, fmt.Errorf("load active class curriculum: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("load assigned enrollment curriculum: %w", err)
	} else {
		item.Assigned = true
	}
	item.Curriculum, err = loadClassCurriculum(ctx, r.db, curriculumID, schoolID)
	if err != nil {
		return nil, err
	}
	if item.Assigned {
		var activeCurriculumID int64
		activeErr := r.db.QueryRow(ctx, `
			SELECT id FROM class_curricula
			 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND section_id=$4 AND status='active'
		`, schoolID, item.AcademicYearID, item.ClassID, item.SectionID).Scan(&activeCurriculumID)
		if activeErr != nil && !errors.Is(activeErr, pgx.ErrNoRows) {
			return nil, fmt.Errorf("load current class curriculum version: %w", activeErr)
		}
		if activeErr == nil && activeCurriculumID != curriculumID {
			item.AvailableCurriculum, err = loadClassCurriculum(ctx, r.db, activeCurriculumID, schoolID)
			if err != nil {
				return nil, err
			}
		}
		rows, err := r.db.Query(ctx, `
			SELECT curriculum_subject_id
			  FROM student_enrollment_subject_selections
			 WHERE enrollment_curriculum_id=$1 AND school_id=$2 AND revoked_at IS NULL
			 ORDER BY curriculum_subject_id
		`, assignmentID, schoolID)
		if err != nil {
			return nil, fmt.Errorf("load selected optional subjects: %w", err)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan selected optional subject: %w", err)
			}
			item.SelectedOptionalSubjectIDs = append(item.SelectedOptionalSubjectIDs, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate selected optional subjects: %w", err)
		}
		rows.Close()
	}
	return &item, nil
}

func (r *AcademicRepository) SaveStudentEnrollmentSubjects(
	ctx context.Context, actorID, schoolID, enrollmentID, curriculumID int64, selectedSubjectIDs []int64,
) (*StudentEnrollmentCurriculum, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if enrollmentID <= 0 || curriculumID <= 0 || len(selectedSubjectIDs) > 150 {
		return nil, ErrInvalidAcademicInput
	}
	selected := make(map[int64]struct{}, len(selectedSubjectIDs))
	for _, id := range selectedSubjectIDs {
		if id <= 0 {
			return nil, ErrInvalidAcademicInput
		}
		if _, exists := selected[id]; exists {
			return nil, ErrInvalidAcademicInput
		}
		selected[id] = struct{}{}
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student subject selection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	// The successful response reloads the enrollment curriculum through the
	// same visibility rules as a read. Check that permission before writing to
	// avoid a predictable post-commit 403 for callers who lack read access.
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.read_basic"); err != nil {
		return nil, err
	}
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "subjects.read"); err != nil {
		return nil, err
	}
	var studentID int64
	if err := tx.QueryRow(ctx, `
		SELECT student_id FROM student_enrollments WHERE id=$1 AND school_id=$2
	`, enrollmentID, schoolID).Scan(&studentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("load student for subject selection: %w", err)
	}
	var studentActive bool
	if err := tx.QueryRow(ctx, `
		SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE
	`, studentID, schoolID).Scan(&studentActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock student for subject selection: %w", err)
	}
	if !studentActive {
		return nil, ErrAcademicNotFound
	}
	var academicYearID, classID, sectionID int64
	var enrollmentStatus string
	if err := tx.QueryRow(ctx, `
		SELECT academic_year_id,class_id,section_id,status FROM student_enrollments
		 WHERE id=$1 AND school_id=$2 AND student_id=$3 FOR UPDATE
	`, enrollmentID, schoolID, studentID).Scan(&academicYearID, &classID, &sectionID, &enrollmentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock student enrollment for subject selection: %w", err)
	}
	if enrollmentStatus != "active" {
		return nil, ErrAcademicNotFound
	}
	var curriculumYearID, curriculumClassID, curriculumSectionID int64
	var curriculumStatus string
	if err := tx.QueryRow(ctx, `
		SELECT academic_year_id,class_id,COALESCE(section_id,0),status FROM class_curricula
		 WHERE id=$1 AND school_id=$2 FOR UPDATE
	`, curriculumID, schoolID).Scan(&curriculumYearID, &curriculumClassID, &curriculumSectionID, &curriculumStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock student curriculum: %w", err)
	}
	if curriculumYearID != academicYearID || curriculumClassID != classID || curriculumSectionID != sectionID {
		return nil, ErrInvalidAcademicInput
	}
	var activeAssignmentID, activeCurriculumID int64
	assignmentErr := tx.QueryRow(ctx, `
		SELECT id,curriculum_id FROM student_enrollment_curricula
		 WHERE enrollment_id=$1 AND school_id=$2 AND status='active' FOR UPDATE
	`, enrollmentID, schoolID).Scan(&activeAssignmentID, &activeCurriculumID)
	hasActiveAssignment := assignmentErr == nil
	if assignmentErr != nil && !errors.Is(assignmentErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock active enrollment curriculum: %w", assignmentErr)
	}
	if curriculumStatus != "active" && (!hasActiveAssignment || activeCurriculumID != curriculumID) {
		return nil, ErrAcademicNotFound
	}
	type option struct {
		isOptional bool
		masterActive bool
		groupID int64
		name string
		code *string
		min int
		max int
	}
	options := make(map[int64]option)
	groupCounts := make(map[int64]int)
	groupBounds := make(map[int64]struct{ min, max int })
	currentSelected := make(map[int64]struct{})
	if hasActiveAssignment && activeCurriculumID == curriculumID {
		currentRows, currentErr := tx.Query(ctx, `SELECT curriculum_subject_id FROM student_enrollment_subject_selections WHERE enrollment_curriculum_id=$1 AND school_id=$2 AND revoked_at IS NULL`, activeAssignmentID, schoolID)
		if currentErr != nil { return nil, fmt.Errorf("load current subject choices before validation: %w", currentErr) }
		for currentRows.Next() { var currentID int64; if scanErr := currentRows.Scan(&currentID); scanErr != nil { currentRows.Close(); return nil, fmt.Errorf("scan current subject choice: %w", scanErr) }; currentSelected[currentID] = struct{}{} }
		if rowsErr := currentRows.Err(); rowsErr != nil { currentRows.Close(); return nil, fmt.Errorf("iterate current subject choices: %w", rowsErr) }
		currentRows.Close()
	}
	// Lock each master subject in shared mode before validating/assigning the
	// curriculum. Subject deactivation takes an exclusive row lock, so either
	// this enrollment commits first and invalidates its preview fingerprint, or
	// this request observes the inactive subject and refuses the assignment.
	var inactiveSubjectCount int
	subjectRows, subjectErr := tx.Query(ctx, `
		SELECT s.id,s.is_active
		  FROM class_curriculum_subjects cs
		  JOIN subjects s ON s.id=cs.subject_id AND s.school_id=cs.school_id
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2 AND cs.is_active=TRUE
		 ORDER BY s.id
		 FOR SHARE OF s
	`, curriculumID, schoolID)
	if subjectErr != nil { return nil, fmt.Errorf("lock curriculum master subjects: %w", subjectErr) }
	for subjectRows.Next() {
		var masterID int64
		var masterActive bool
		if scanErr := subjectRows.Scan(&masterID,&masterActive); scanErr != nil { subjectRows.Close(); return nil, fmt.Errorf("scan locked curriculum master subject: %w", scanErr) }
		if !masterActive { inactiveSubjectCount++ }
	}
	if rowsErr := subjectRows.Err(); rowsErr != nil { subjectRows.Close(); return nil, fmt.Errorf("iterate locked curriculum master subjects: %w", rowsErr) }
	subjectRows.Close()
	if (!hasActiveAssignment || activeCurriculumID != curriculumID) && inactiveSubjectCount > 0 {
		return nil, ErrInactiveSubjectInCurriculum
	}
	rows, err := tx.Query(ctx, `
		SELECT cs.id,cs.optional_group_id,cs.is_optional,cs.subject_name_snapshot,cs.subject_code_snapshot,
		       COALESCE(g.min_selections,0),COALESCE(g.max_selections,0),s.is_active
		  FROM class_curriculum_subjects cs
		  JOIN subjects s ON s.id=cs.subject_id AND s.school_id=cs.school_id
		  LEFT JOIN class_curriculum_optional_groups g
		    ON g.id=cs.optional_group_id AND g.curriculum_id=cs.curriculum_id AND g.school_id=cs.school_id
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2 AND cs.is_active=TRUE
	`, curriculumID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("load curriculum subject selection rules: %w", err)
	}
	for rows.Next() {
		var id int64
		var groupID *int64
		var isOptional bool
		var subjectName string
		var subjectCode *string
		var min, max int
		var masterActive bool
		if err := rows.Scan(&id, &groupID, &isOptional, &subjectName, &subjectCode, &min, &max, &masterActive); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan curriculum subject selection rule: %w", err)
		}
		if isOptional {
			item := option{isOptional: true, masterActive: masterActive, name: subjectName, code: subjectCode}
			if groupID != nil {
				item.groupID, item.min, item.max = *groupID, min, max
				groupBounds[*groupID] = struct{ min, max int }{min: min, max: max}
			}
			options[id] = item
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate curriculum subject selection rules: %w", err)
	}
	rows.Close()
	for id := range selected {
		option, exists := options[id]
		if !exists || !option.isOptional {
			return nil, ErrInvalidAcademicInput
		}
		if !option.masterActive {
			if _, wasAlreadySelected := currentSelected[id]; !wasAlreadySelected { return nil, ErrInactiveSubjectSelection }
		}
		if option.groupID != 0 { groupCounts[option.groupID]++ }
	}
	for groupID, bounds := range groupBounds {
		count := groupCounts[groupID]
		if count < bounds.min || count > bounds.max {
			return nil, ErrInvalidAcademicInput
		}
	}
	if hasActiveAssignment && activeCurriculumID != curriculumID {
		if _, err := tx.Exec(ctx, `
			DELETE FROM student_enrollment_subject_selections
			 WHERE enrollment_curriculum_id=$1 AND school_id=$2
		`, activeAssignmentID, schoolID); err != nil {
			return nil, academicWriteError(err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_enrollment_curricula SET status='replaced',revoked_at=NOW()
			 WHERE id=$1 AND school_id=$2 AND status='active'
		`, activeAssignmentID, schoolID); err != nil {
			return nil, academicWriteError(err)
		}
		hasActiveAssignment = false
	} else if hasActiveAssignment {
		if _, err := tx.Exec(ctx, `
			DELETE FROM student_enrollment_subject_selections
			 WHERE enrollment_curriculum_id=$1 AND school_id=$2
		`, activeAssignmentID, schoolID); err != nil {
			return nil, academicWriteError(err)
		}
	}
	if !hasActiveAssignment {
		if err := tx.QueryRow(ctx, `
			INSERT INTO student_enrollment_curricula(school_id,enrollment_id,curriculum_id,assigned_by_principal_id)
			VALUES($1,$2,$3,$4) RETURNING id
		`, schoolID, enrollmentID, curriculumID, actorID).Scan(&activeAssignmentID); err != nil {
			return nil, academicWriteError(err)
		}
	}
	for id := range selected {
		option := options[id]
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_enrollment_subject_selections
				(school_id,enrollment_curriculum_id,curriculum_id,curriculum_subject_id,
				 subject_name_snapshot,subject_code_snapshot,selected_by_principal_id)
			VALUES($1,$2,$3,$4,$5,$6,$7)
		`, schoolID, activeAssignmentID, curriculumID, id, option.name, option.code, actorID); err != nil {
			return nil, academicWriteError(err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student.subjects.updated','student_enrollment',$3,'success',
		       jsonb_build_object('curriculum_id',$4::bigint,'selected_optional_subject_count',$5::integer))
	`, actorID, schoolID, fmt.Sprint(enrollmentID), curriculumID, len(selected)); err != nil {
		return nil, fmt.Errorf("audit student subject selection: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student subject selection: %w", err)
	}
	return r.GetStudentEnrollmentCurriculum(ctx, actorID, schoolID, enrollmentID)
}
