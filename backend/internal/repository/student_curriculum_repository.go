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
	IsCompulsory      bool    `json:"is_compulsory"`
}

type ClassCurriculum struct {
	ID             int64                     `json:"id"`
	SchoolID       int64                     `json:"school_id"`
	AcademicYearID int64                     `json:"academic_year_id"`
	ClassID        int64                     `json:"class_id"`
	VersionNumber  int                       `json:"version_number"`
	Status         string                    `json:"status"`
	CreatedAt      time.Time                 `json:"created_at"`
	Groups         []CurriculumOptionalGroup `json:"optional_groups"`
	Subjects       []CurriculumSubject       `json:"subjects"`
}

type StudentSubjectSelectionHistory struct {
	CurriculumVersion int        `json:"curriculum_version"`
	SubjectName       string     `json:"subject_name"`
	SubjectCode       *string    `json:"subject_code,omitempty"`
	SelectedAt        time.Time  `json:"selected_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
}

type StudentEnrollmentCurriculum struct {
	EnrollmentID               int64                            `json:"enrollment_id"`
	StudentID                  int64                            `json:"student_id"`
	StudentStatus              string                           `json:"student_status"`
	EnrollmentStatus           string                           `json:"enrollment_status"`
	AcademicYearID             int64                            `json:"academic_year_id"`
	ClassID                    int64                            `json:"class_id"`
	Curriculum                 *ClassCurriculum                 `json:"curriculum,omitempty"`
	AvailableCurriculum        *ClassCurriculum                 `json:"available_curriculum,omitempty"`
	Assigned                   bool                             `json:"assigned"`
	SelectedOptionalSubjectIDs []int64                          `json:"selected_optional_subject_ids"`
	SelectionHistory           []StudentSubjectSelectionHistory `json:"selection_history"`
}

func loadClassCurriculum(ctx context.Context, db repositoryDB, curriculumID, schoolID int64) (*ClassCurriculum, error) {
	var item ClassCurriculum
	err := db.QueryRow(ctx, `
		SELECT id, school_id, academic_year_id, class_id, version_number, status, created_at
		  FROM class_curricula
		 WHERE id=$1 AND school_id=$2
	`, curriculumID, schoolID).Scan(
		&item.ID, &item.SchoolID, &item.AcademicYearID, &item.ClassID,
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
		       cs.optional_group_id, g.name
		  FROM class_curriculum_subjects cs
		  LEFT JOIN class_curriculum_optional_groups g
		    ON g.id=cs.optional_group_id AND g.curriculum_id=cs.curriculum_id AND g.school_id=cs.school_id
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2
		 ORDER BY (cs.optional_group_id IS NOT NULL), lower(COALESCE(g.name,'')), lower(cs.subject_name_snapshot), cs.id
	`, curriculumID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("list curriculum subjects: %w", err)
	}
	for subjectRows.Next() {
		var subject CurriculumSubject
		if err := subjectRows.Scan(&subject.ID, &subject.SubjectID, &subject.Name, &subject.Code,
			&subject.OptionalGroupID, &subject.OptionalGroupName); err != nil {
			subjectRows.Close()
			return nil, fmt.Errorf("scan curriculum subject: %w", err)
		}
		subject.IsCompulsory = subject.OptionalGroupID == nil
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
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if schoolID <= 0 || academicYearID <= 0 || classID <= 0 || len(subjects) == 0 || len(subjects) > 150 || len(groups) > 30 {
		return nil, ErrInvalidAcademicInput
	}
	groupByName := make(map[string]CurriculumOptionalGroupInput, len(groups))
	groupCounts := make(map[string]int, len(groups))
	for _, group := range groups {
		group.Name = strings.TrimSpace(group.Name)
		key := strings.ToLower(group.Name)
		if group.Name == "" || len([]rune(group.Name)) > 100 || group.MinSelections < 0 ||
			group.MaxSelections < 1 || group.MinSelections > group.MaxSelections || key == "" {
			return nil, ErrInvalidAcademicInput
		}
		if _, exists := groupByName[key]; exists {
			return nil, ErrInvalidAcademicInput
		}
		groupByName[key] = group
	}
	seenSubjects := make(map[int64]struct{}, len(subjects))
	normalizedSubjects := make([]CurriculumSubjectInput, 0, len(subjects))
	for _, subject := range subjects {
		if subject.SubjectID <= 0 {
			return nil, ErrInvalidAcademicInput
		}
		if _, exists := seenSubjects[subject.SubjectID]; exists {
			return nil, ErrInvalidAcademicInput
		}
		seenSubjects[subject.SubjectID] = struct{}{}
		subject.OptionalGroup = strings.TrimSpace(subject.OptionalGroup)
		if subject.OptionalGroup != "" {
			key := strings.ToLower(subject.OptionalGroup)
			if _, exists := groupByName[key]; !exists {
				return nil, ErrInvalidAcademicInput
			}
			groupCounts[key]++
		}
		normalizedSubjects = append(normalizedSubjects, subject)
	}
	for key, group := range groupByName {
		count := groupCounts[key]
		if count == 0 || group.MinSelections > count || group.MaxSelections > count {
			return nil, ErrInvalidAcademicInput
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin class curriculum save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "subjects.manage"); err != nil {
		return nil, err
	}
	var lockedClassID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM classes WHERE id=$1 AND school_id=$2 AND is_active=TRUE FOR UPDATE
	`, classID, schoolID).Scan(&lockedClassID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock curriculum class: %w", err)
	}
	var lockedYearID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM academic_years WHERE id=$1 AND school_id=$2 AND status <> 'closed' FOR UPDATE
	`, academicYearID, schoolID).Scan(&lockedYearID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock curriculum academic year: %w", err)
	}
	subjectIDs := make([]int64, 0, len(normalizedSubjects))
	for _, subject := range normalizedSubjects {
		subjectIDs = append(subjectIDs, subject.SubjectID)
	}
	var activeSubjectCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM subjects WHERE school_id=$1 AND is_active=TRUE AND id=ANY($2::bigint[])
	`, schoolID, subjectIDs).Scan(&activeSubjectCount); err != nil {
		return nil, fmt.Errorf("validate curriculum subjects: %w", err)
	}
	if activeSubjectCount != len(subjectIDs) {
		return nil, ErrAcademicNotFound
	}
	var version int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number),0)+1
		  FROM class_curricula
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3
	`, schoolID, academicYearID, classID).Scan(&version); err != nil {
		return nil, fmt.Errorf("allocate curriculum version: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE class_curricula SET status='retired'
		 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND status='active'
	`, schoolID, academicYearID, classID); err != nil {
		return nil, academicWriteError(err)
	}
	var curriculumID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO class_curricula(school_id,academic_year_id,class_id,version_number,status,created_by_principal_id)
		VALUES($1,$2,$3,$4,'active',$5) RETURNING id
	`, schoolID, academicYearID, classID, version, actorID).Scan(&curriculumID); err != nil {
		return nil, academicWriteError(err)
	}
	groupIDs := make(map[string]int64, len(groupByName))
	for key, group := range groupByName {
		var groupID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO class_curriculum_optional_groups(school_id,curriculum_id,name,min_selections,max_selections)
			VALUES($1,$2,$3,$4,$5) RETURNING id
		`, schoolID, curriculumID, group.Name, group.MinSelections, group.MaxSelections).Scan(&groupID); err != nil {
			return nil, academicWriteError(err)
		}
		groupIDs[key] = groupID
	}
	for _, subject := range normalizedSubjects {
		var groupID *int64
		if subject.OptionalGroup != "" {
			id := groupIDs[strings.ToLower(subject.OptionalGroup)]
			groupID = &id
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO class_curriculum_subjects
				(school_id,curriculum_id,subject_id,optional_group_id,subject_name_snapshot,subject_code_snapshot)
			SELECT $1,$2,id,$3,name,subject_code FROM subjects
			 WHERE id=$4 AND school_id=$1 AND is_active=TRUE
		`, schoolID, curriculumID, groupID, subject.SubjectID); err != nil {
			return nil, academicWriteError(err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.class_curriculum.created','class_curriculum',$3,'success',
		       jsonb_build_object('academic_year_id',$4::bigint,'class_id',$5::bigint,'version',$6::integer,
		                          'subject_count',$7::integer,'optional_group_count',$8::integer))
	`, actorID, schoolID, fmt.Sprint(curriculumID), academicYearID, classID, version, len(subjects), len(groups)); err != nil {
		return nil, fmt.Errorf("audit class curriculum save: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit class curriculum save: %w", err)
	}
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
		                          'selection_history_count',$5::integer,
		                          'has_curriculum',$6::boolean))
	`, actorID, schoolID, fmt.Sprint(enrollmentID), len(item.SelectedOptionalSubjectIDs),
		len(item.SelectionHistory), item.Curriculum != nil); err != nil {
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
	item.SelectionHistory = make([]StudentSubjectSelectionHistory, 0)
	if err := r.db.QueryRow(ctx, `
		SELECT e.academic_year_id,e.class_id,e.status,st.status
		  FROM student_enrollments e
		  JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		 WHERE e.id=$1 AND e.school_id=$2
	`, enrollmentID, schoolID).Scan(&item.AcademicYearID, &item.ClassID, &item.EnrollmentStatus, &item.StudentStatus); err != nil {
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
			 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND status='active'
		`, schoolID, item.AcademicYearID, item.ClassID).Scan(&curriculumID)
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
			 WHERE school_id=$1 AND academic_year_id=$2 AND class_id=$3 AND status='active'
		`, schoolID, item.AcademicYearID, item.ClassID).Scan(&activeCurriculumID)
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
			 WHERE enrollment_curriculum_id=$1 AND revoked_at IS NULL
			 ORDER BY curriculum_subject_id
		`, assignmentID)
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
	historyRows, err := r.db.Query(ctx, `
		SELECT cc.version_number,s.subject_name_snapshot,s.subject_code_snapshot,s.selected_at,s.revoked_at
		  FROM student_enrollment_curricula ec
		  JOIN class_curricula cc ON cc.id=ec.curriculum_id AND cc.school_id=ec.school_id
		  JOIN student_enrollment_subject_selections s
		    ON s.enrollment_curriculum_id=ec.id AND s.curriculum_id=ec.curriculum_id AND s.school_id=ec.school_id
		 WHERE ec.enrollment_id=$1 AND ec.school_id=$2
		 ORDER BY s.selected_at DESC,s.id DESC
	`, enrollmentID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("load student subject selection history: %w", err)
	}
	for historyRows.Next() {
		var history StudentSubjectSelectionHistory
		if err := historyRows.Scan(&history.CurriculumVersion, &history.SubjectName, &history.SubjectCode, &history.SelectedAt, &history.RevokedAt); err != nil {
			historyRows.Close()
			return nil, fmt.Errorf("scan student subject selection history: %w", err)
		}
		item.SelectionHistory = append(item.SelectionHistory, history)
	}
	if err := historyRows.Err(); err != nil {
		historyRows.Close()
		return nil, fmt.Errorf("iterate student subject selection history: %w", err)
	}
	historyRows.Close()
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
	var academicYearID, classID int64
	var enrollmentStatus string
	if err := tx.QueryRow(ctx, `
		SELECT academic_year_id,class_id,status FROM student_enrollments
		 WHERE id=$1 AND school_id=$2 AND student_id=$3 FOR UPDATE
	`, enrollmentID, schoolID, studentID).Scan(&academicYearID, &classID, &enrollmentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock student enrollment for subject selection: %w", err)
	}
	if enrollmentStatus != "active" {
		return nil, ErrAcademicNotFound
	}
	var curriculumYearID, curriculumClassID int64
	var curriculumStatus string
	if err := tx.QueryRow(ctx, `
		SELECT academic_year_id,class_id,status FROM class_curricula
		 WHERE id=$1 AND school_id=$2 FOR UPDATE
	`, curriculumID, schoolID).Scan(&curriculumYearID, &curriculumClassID, &curriculumStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAcademicNotFound
		}
		return nil, fmt.Errorf("lock student curriculum: %w", err)
	}
	if curriculumYearID != academicYearID || curriculumClassID != classID {
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
		groupID int64
		name string
		code *string
		min int
		max int
	}
	options := make(map[int64]option)
	groupCounts := make(map[int64]int)
	groupBounds := make(map[int64]struct{ min, max int })
	rows, err := tx.Query(ctx, `
		SELECT cs.id,cs.optional_group_id,cs.subject_name_snapshot,cs.subject_code_snapshot,
		       COALESCE(g.min_selections,0),COALESCE(g.max_selections,0)
		  FROM class_curriculum_subjects cs
		  LEFT JOIN class_curriculum_optional_groups g
		    ON g.id=cs.optional_group_id AND g.curriculum_id=cs.curriculum_id AND g.school_id=cs.school_id
		 WHERE cs.curriculum_id=$1 AND cs.school_id=$2
	`, curriculumID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("load curriculum subject selection rules: %w", err)
	}
	for rows.Next() {
		var id int64
		var groupID *int64
		var subjectName string
		var subjectCode *string
		var min, max int
		if err := rows.Scan(&id, &groupID, &subjectName, &subjectCode, &min, &max); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan curriculum subject selection rule: %w", err)
		}
		if groupID != nil {
			options[id] = option{groupID: *groupID, name: subjectName, code: subjectCode, min: min, max: max}
			groupBounds[*groupID] = struct{ min, max int }{min: min, max: max}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate curriculum subject selection rules: %w", err)
	}
	rows.Close()
	for id := range selected {
		option, exists := options[id]
		if !exists {
			return nil, ErrInvalidAcademicInput
		}
		groupCounts[option.groupID]++
	}
	for groupID, bounds := range groupBounds {
		count := groupCounts[groupID]
		if count < bounds.min || count > bounds.max {
			return nil, ErrInvalidAcademicInput
		}
	}
	if hasActiveAssignment && activeCurriculumID != curriculumID {
		if _, err := tx.Exec(ctx, `
			UPDATE student_enrollment_subject_selections SET revoked_at=NOW()
			 WHERE enrollment_curriculum_id=$1 AND revoked_at IS NULL
		`, activeAssignmentID); err != nil {
			return nil, academicWriteError(err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE student_enrollment_curricula SET status='replaced',revoked_at=NOW()
			 WHERE id=$1 AND status='active'
		`, activeAssignmentID); err != nil {
			return nil, academicWriteError(err)
		}
		hasActiveAssignment = false
	} else if hasActiveAssignment {
		if _, err := tx.Exec(ctx, `
			UPDATE student_enrollment_subject_selections SET revoked_at=NOW()
			 WHERE enrollment_curriculum_id=$1 AND revoked_at IS NULL
		`, activeAssignmentID); err != nil {
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
