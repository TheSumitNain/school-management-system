package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrAcademicAccessDenied = errors.New("academic operation is not authorized")
	ErrAcademicNotFound = errors.New("academic resource not found")
	ErrAcademicConflict = errors.New("academic resource conflicts with an existing record")
	ErrCurriculumRemovalConfirmationRequired = errors.New("curriculum subject removal requires explicit confirmation")
	ErrCurriculumRemovalImpactChanged = errors.New("curriculum removal impact changed; review the affected student count again")
	ErrCurriculumSubjectRuleConflict = errors.New("curriculum edit would invalidate active student subject choices")
	ErrSectionDeactivationConfirmationRequired = errors.New("section deactivation requires dependency preview and confirmation")
	ErrClassDeactivationConfirmationRequired = errors.New("class deactivation requires dependency preview and confirmation")
	ErrSubjectDeactivationConfirmationRequired = errors.New("subject deactivation requires dependency preview and confirmation")
	ErrInactiveSubjectInCurriculum = errors.New("new enrollment cannot use a curriculum containing an inactive subject")
	ErrInactiveSubjectSelection = errors.New("inactive subjects cannot be newly selected by a student")
	ErrInactiveSubjectAssignment = errors.New("inactive subjects cannot receive new teacher assignments")
	ErrAcademicImpactChanged = errors.New("academic dependency impact changed; review the impact again")
	ErrAcademicAdmissionNumberConflict = errors.New("academic admission number already exists in this school year")
	ErrAcademicRollNumberConflict = errors.New("academic roll number already exists in this active section")
	ErrStudentRegistrationNumberConflict = errors.New("student school registration number already exists in this school")
	ErrInvalidStudentContactEmail = errors.New("invalid student contact email")
	ErrInvalidStudentDateOfBirth = errors.New("invalid student date of birth")
	ErrInvalidStudentMeasurementDate = errors.New("invalid student measurement date")
	ErrPromotionBeforeAttendance = errors.New("promotion date precedes recorded attendance")
	ErrAdmissionAfterEnrollmentDate = errors.New("admission date is after enrollment start date")
	ErrInvalidAcademicInput = errors.New("invalid academic input")
)

type AcademicYear struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	Name string `json:"name"`
	StartsOn time.Time `json:"starts_on"`
	EndsOn time.Time `json:"ends_on"`
	Status string `json:"status"`
	IsCurrent bool `json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SectionDeactivationStudent struct {
	StudentID int64 `json:"student_id"`
	FullName string `json:"full_name"`
	AdmissionNumber string `json:"admission_number"`
	EnrollmentStatus string `json:"enrollment_status"`
}

type SectionDeactivationImpact struct {
	SectionID int64 `json:"section_id"`
	SectionName string `json:"section_name"`
	ClassName string `json:"class_name"`
	AcademicYearName string `json:"academic_year_name"`
	ActiveEnrollmentCount int64 `json:"active_enrollment_count"`
	HistoricalEnrollmentCount int64 `json:"historical_enrollment_count"`
	ActiveCurriculumCount int64 `json:"active_curriculum_count"`
	ActiveTeacherAssignmentCount int64 `json:"active_teacher_assignment_count"`
	ActiveClassInchargeCount int64 `json:"active_class_incharge_count"`
	AffectedStudents []SectionDeactivationStudent `json:"affected_students"`
	ImpactFingerprint string `json:"impact_fingerprint"`
}

type ClassDeactivationStudent struct {
	StudentID int64 `json:"student_id"`
	FullName string `json:"full_name"`
	AdmissionNumber string `json:"admission_number"`
	SectionName string `json:"section_name"`
	AcademicYearName string `json:"academic_year_name"`
}
type ClassDeactivationImpact struct {
	ClassID int64 `json:"class_id"`
	ClassName string `json:"class_name"`
	ActiveSectionCount int64 `json:"active_section_count"`
	InactiveSectionCount int64 `json:"inactive_section_count"`
	ActiveEnrollmentCount int64 `json:"active_enrollment_count"`
	HistoricalEnrollmentCount int64 `json:"historical_enrollment_count"`
	ActiveCurriculumCount int64 `json:"active_curriculum_count"`
	ActiveTeacherAssignmentCount int64 `json:"active_teacher_assignment_count"`
	ActiveClassInchargeCount int64 `json:"active_class_incharge_count"`
	AffectedStudents []ClassDeactivationStudent `json:"affected_students"`
	ImpactFingerprint string `json:"impact_fingerprint"`
}


type SubjectDeactivationStudent struct {
	StudentID int64 `json:"student_id"`
	FullName string `json:"full_name"`
	AdmissionNumber string `json:"admission_number"`
	ClassName string `json:"class_name"`
	SectionName string `json:"section_name"`
	AcademicYearName string `json:"academic_year_name"`
}
type SubjectDeactivationImpact struct {
	SubjectID int64 `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	CurriculumSubjectCount int64 `json:"curriculum_subject_count"`
	ActiveTeacherAssignmentCount int64 `json:"active_teacher_assignment_count"`
	ActiveStudentSelectionCount int64 `json:"active_student_selection_count"`
	HistoricalStudentSelectionCount int64 `json:"historical_student_selection_count"`
	AffectedStudents []SubjectDeactivationStudent `json:"affected_students"`
	ImpactFingerprint string `json:"impact_fingerprint"`
}

type AcademicClass struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	Name string `json:"name"`
	DisplayOrder int `json:"display_order"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AcademicSection struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	AcademicYearID int64 `json:"academic_year_id"`
	ClassID int64 `json:"class_id"`
	ClassName string `json:"class_name,omitempty"`
	Name string `json:"name"`
	Medium *string `json:"medium,omitempty"`
	DisplayOrder int `json:"display_order"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AcademicSubject struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	Name string `json:"name"`
	Code *string `json:"subject_code,omitempty"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AcademicRepository struct { db repositoryDB }

func NewAcademicRepository(db repositoryDB) *AcademicRepository { return &AcademicRepository{db: db} }

func isTeacherWithoutSchoolAdmin(ctx context.Context, db repositoryDB, principalID, schoolID int64) (bool, error) {
	var isTeacher, isSchoolAdmin bool
	err := db.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM principal_roles pr
				JOIN roles r ON r.id = pr.role_id
				WHERE pr.principal_id = $1 AND pr.revoked_at IS NULL
				  AND r.school_id = $2 AND r.scope_type = 'school'
				  AND r.principal_type = 'staff' AND r.role_key = 'teacher'
				  AND r.is_system = TRUE
			),
			EXISTS (
				SELECT 1 FROM principal_roles pr
				JOIN roles r ON r.id = pr.role_id
				WHERE pr.principal_id = $1 AND pr.revoked_at IS NULL
				  AND r.school_id = $2 AND r.scope_type = 'school'
				  AND r.principal_type = 'staff' AND r.role_key = 'school_admin'
				  AND r.is_system = TRUE
			)
	`, principalID, schoolID).Scan(&isTeacher, &isSchoolAdmin)
	if err != nil {
		return false, fmt.Errorf("check teacher role boundary: %w", err)
	}
	return isTeacher && !isSchoolAdmin, nil
}

func authorizeAcademic(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if db == nil || actorID <= 0 || schoolID <= 0 || permission == "" {
		return ErrAcademicAccessDenied
	}

	// Use the same live RBAC evaluator as AuthMiddleware and the guardian
	// repository. A direct role-permission join would bypass exact delegation
	// identity, revoked delegation checks, and ALLOW/DENY user overrides.
	var principal models.AuthPrincipal
	var userStatus, schoolStatus string
	err := db.QueryRow(ctx, `
		SELECT ap.id, ap.principal_type, ap.status, u.school_id, u.status, s.status
		FROM auth_principals ap
		JOIN users u ON u.principal_id = ap.id
		JOIN schools s ON s.id = u.school_id
		WHERE ap.id = $1
	`, actorID).Scan(
		&principal.ID, &principal.Type, &principal.Status,
		&principal.SchoolID, &userStatus, &schoolStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAcademicAccessDenied
	}
	if err != nil {
		return fmt.Errorf("load academic actor: %w", err)
	}
	if principal.Type != "staff" || principal.Status != "active" ||
		principal.SchoolID != schoolID || userStatus != "active" || schoolStatus != "active" {
		return ErrAcademicAccessDenied
	}
	if strings.HasPrefix(permission, "students.") && permission != "students.read_basic" {
		teacherOnly, roleErr := isTeacherWithoutSchoolAdmin(ctx, db, actorID, schoolID)
		if roleErr != nil {
			return fmt.Errorf("enforce teacher student-data boundary: %w", roleErr)
		}
		if teacherOnly {
			return ErrAcademicAccessDenied
		}
	}

	effective, err := loadEffectivePermissions(ctx, db, &principal)
	if err != nil {
		return fmt.Errorf("authorize academic operation: %w", err)
	}
	for _, key := range effective {
		if key == permission {
			return nil
		}
	}
	return ErrAcademicAccessDenied
}

func academicWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			switch pgErr.ConstraintName {
			case "ux_student_enrollments_school_year_admission_number":
				return ErrAcademicAdmissionNumberConflict
			case "ux_student_enrollments_active_roll":
				return ErrAcademicRollNumberConflict
			case "ux_student_basic_details_school_registration_number":
				return ErrStudentRegistrationNumberConflict
			case "student_enrollments_admission_not_after_enrollment":
				return ErrAdmissionAfterEnrollmentDate
			default:
				return ErrAcademicConflict
			}
		case "23P01": return ErrAcademicConflict
		case "P0001":
			message := strings.ToLower(pgErr.Message)
			if strings.Contains(message, "assignment actor") { return ErrAcademicAccessDenied }
			if strings.Contains(message, "subject choice rules cannot change") ||
				strings.Contains(message, "optional group limits would invalidate") ||
				strings.Contains(message, "active student subject selections must be revoked") {
				return ErrCurriculumSubjectRuleConflict
			}
			return ErrInvalidAcademicInput
		case "23503", "23514", "22001", "22007":
			if strings.Contains(strings.ToLower(pgErr.Message), "assignment actor") { return ErrAcademicAccessDenied }
			return ErrInvalidAcademicInput
		}
	}
	if errors.Is(err, pgx.ErrNoRows) { return ErrAcademicNotFound }
	return err
}

func (r *AcademicRepository) ListAcademicYears(ctx context.Context, actorID, schoolID int64) ([]AcademicYear, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "academic_years.read"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `SELECT id, school_id, name, starts_on, ends_on, status, is_current, created_at, updated_at
		FROM academic_years WHERE school_id = $1 ORDER BY starts_on DESC, id DESC`, schoolID)
	if err != nil { return nil, fmt.Errorf("list academic years: %w", err) }
	defer rows.Close()
	items := make([]AcademicYear, 0)
	for rows.Next() { var v AcademicYear; if err := rows.Scan(&v.ID,&v.SchoolID,&v.Name,&v.StartsOn,&v.EndsOn,&v.Status,&v.IsCurrent,&v.CreatedAt,&v.UpdatedAt); err != nil { return nil, fmt.Errorf("scan academic year: %w",err) }; items=append(items,v) }
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate academic years: %w",err) }
	return items,nil
}

func (r *AcademicRepository) SaveAcademicYear(ctx context.Context, actorID, schoolID, id int64, name string, startsOn, endsOn time.Time, status string, current bool) (*AcademicYear,error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name = strings.TrimSpace(name); status = strings.TrimSpace(status)
	if len([]rune(name))<4 || len([]rune(name))>80 || !endsOn.After(startsOn) || (status!="planned" && status!="active" && status!="closed") || (current && status!="active") { return nil,ErrInvalidAcademicInput }
	tx,err:=r.db.Begin(ctx); if err!=nil{return nil,fmt.Errorf("begin academic year save: %w",err)}; defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"academic_years.manage"); err!=nil{return nil,err}
	if id>0 {
		if _,err=tx.Exec(ctx,`UPDATE academic_years SET is_current=FALSE, updated_at=NOW() WHERE school_id=$1 AND id<>$2 AND $3=TRUE`,schoolID,id,current); err!=nil{return nil,fmt.Errorf("clear current academic year: %w",err)}
	} else if current {
		if _,err=tx.Exec(ctx,`UPDATE academic_years SET is_current=FALSE, updated_at=NOW() WHERE school_id=$1 AND is_current=TRUE`,schoolID); err!=nil{return nil,fmt.Errorf("clear current academic year: %w",err)}
	}
	var v AcademicYear
	if id==0 {
		err=tx.QueryRow(ctx,`INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
			VALUES($1,$2,$3,$4,$5,$6) RETURNING id,school_id,name,starts_on,ends_on,status,is_current,created_at,updated_at`,
			schoolID,name,startsOn,endsOn,status,current).Scan(&v.ID,&v.SchoolID,&v.Name,&v.StartsOn,&v.EndsOn,&v.Status,&v.IsCurrent,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err=tx.QueryRow(ctx,`UPDATE academic_years SET name=$3,starts_on=$4,ends_on=$5,status=$6,is_current=$7,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 AND status <> 'closed'
			RETURNING id,school_id,name,starts_on,ends_on,status,is_current,created_at,updated_at`,
			schoolID,id,name,startsOn,endsOn,status,current).Scan(&v.ID,&v.SchoolID,&v.Name,&v.StartsOn,&v.EndsOn,&v.Status,&v.IsCurrent,&v.CreatedAt,&v.UpdatedAt)
	}
	if err!=nil{return nil,academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'academic_year',$4,'success',jsonb_build_object('name',$5::text,'status',$6::text,'is_current',$7::boolean))`,
		actorID,schoolID,map[bool]string{true:"school.academic_year.created",false:"school.academic_year.updated"}[id==0],fmt.Sprint(v.ID),v.Name,v.Status,v.IsCurrent);err!=nil{return nil,fmt.Errorf("audit academic year save: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit academic year save: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) ListClasses(ctx context.Context, actorID,schoolID int64)([]AcademicClass,error){
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"classes.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT id,school_id,name,display_order,is_active,created_at,updated_at FROM classes WHERE school_id=$1 ORDER BY display_order,id`,schoolID);if err!=nil{return nil,fmt.Errorf("list classes: %w",err)};defer rows.Close()
	items:=make([]AcademicClass,0);for rows.Next(){var v AcademicClass;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan class: %w",err)};items=append(items,v)};if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate classes: %w",err)};return items,nil
}
func classDeactivationFingerprint(ctx context.Context, tx pgx.Tx, schoolID, classID int64) (string,error) {
	var fingerprint string
	err:=tx.QueryRow(ctx,`
		SELECT md5(concat_ws('|',
			COALESCE((SELECT c.name || ':' || c.display_order::text || ':' || c.is_active::text FROM classes c WHERE c.school_id=$1 AND c.id=$2),''),
			COALESCE((SELECT string_agg(s.id::text || ':' || s.is_active::text, ',' ORDER BY s.id) FROM sections s WHERE s.school_id=$1 AND s.class_id=$2),''),
			COALESCE((SELECT string_agg(e.id::text || ':' || e.status, ',' ORDER BY e.id) FROM student_enrollments e WHERE e.school_id=$1 AND e.class_id=$2),''),
			COALESCE((SELECT string_agg(cc.id::text || ':' || cc.status, ',' ORDER BY cc.id) FROM class_curricula cc WHERE cc.school_id=$1 AND cc.class_id=$2),''),
			COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=$1 AND s.class_id=$2),''),
			COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM class_incharge_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=$1 AND s.class_id=$2),'')
		))
	`,schoolID,classID).Scan(&fingerprint)
	if err!=nil{return "",fmt.Errorf("calculate class dependency fingerprint: %w",err)}
	return fingerprint,nil
}
func lockClassDeactivationDependencies(ctx context.Context, tx pgx.Tx, schoolID, classID int64) error {
	for _,query:=range []string{
		`SELECT id FROM sections WHERE school_id=$1 AND class_id=$2 ORDER BY id FOR UPDATE`,
		`SELECT id FROM student_enrollments WHERE school_id=$1 AND class_id=$2 ORDER BY id FOR UPDATE`,
		`SELECT id FROM class_curricula WHERE school_id=$1 AND class_id=$2 ORDER BY id FOR UPDATE`,
		`SELECT a.id FROM teacher_subject_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=$1 AND s.class_id=$2 ORDER BY a.id FOR UPDATE OF a`,
		`SELECT a.id FROM class_incharge_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=$1 AND s.class_id=$2 ORDER BY a.id FOR UPDATE OF a`,
	}{
		rows,err:=tx.Query(ctx,query,schoolID,classID);if err!=nil{return fmt.Errorf("lock class dependency rows: %w",err)}
		for rows.Next(){var id int64;if err:=rows.Scan(&id);err!=nil{rows.Close();return fmt.Errorf("scan locked class dependency: %w",err)}}
		if err:=rows.Err();err!=nil{rows.Close();return fmt.Errorf("iterate locked class dependencies: %w",err)}
		rows.Close()
	}
	return nil
}
func (r *AcademicRepository) PreviewClassDeactivation(ctx context.Context, actorID, schoolID, classID int64) (*ClassDeactivationImpact,error) {
	if r==nil||r.db==nil||classID<=0{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin class deactivation preview: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"classes.manage");err!=nil{return nil,err}
	var impact ClassDeactivationImpact
	err=tx.QueryRow(ctx,`
		SELECT c.id,c.name,
			(SELECT COUNT(*) FROM sections s WHERE s.school_id=c.school_id AND s.class_id=c.id AND s.is_active=TRUE),
			(SELECT COUNT(*) FROM sections s WHERE s.school_id=c.school_id AND s.class_id=c.id AND s.is_active=FALSE),
			(SELECT COUNT(*) FROM student_enrollments e WHERE e.school_id=c.school_id AND e.class_id=c.id AND e.status='active'),
			(SELECT COUNT(*) FROM student_enrollments e WHERE e.school_id=c.school_id AND e.class_id=c.id AND e.status<>'active'),
			(SELECT COUNT(*) FROM class_curricula cc WHERE cc.school_id=c.school_id AND cc.class_id=c.id AND cc.status='active'),
			(SELECT COUNT(*) FROM teacher_subject_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=c.school_id AND s.class_id=c.id AND a.revoked_at IS NULL),
			(SELECT COUNT(*) FROM class_incharge_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=c.school_id AND s.class_id=c.id AND a.revoked_at IS NULL),
			md5(concat_ws('|',
				COALESCE((SELECT c.name || ':' || c.display_order::text || ':' || c.is_active::text FROM classes class_row WHERE class_row.school_id=c.school_id AND class_row.id=c.id),''),
				COALESCE((SELECT string_agg(s.id::text || ':' || s.is_active::text, ',' ORDER BY s.id) FROM sections s WHERE s.school_id=c.school_id AND s.class_id=c.id),''),
				COALESCE((SELECT string_agg(e.id::text || ':' || e.status, ',' ORDER BY e.id) FROM student_enrollments e WHERE e.school_id=c.school_id AND e.class_id=c.id),''),
				COALESCE((SELECT string_agg(cc.id::text || ':' || cc.status, ',' ORDER BY cc.id) FROM class_curricula cc WHERE cc.school_id=c.school_id AND cc.class_id=c.id),''),
				COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=c.school_id AND s.class_id=c.id),''),
				COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM class_incharge_assignments a JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id WHERE a.school_id=c.school_id AND s.class_id=c.id),'')
			))
		FROM classes c WHERE c.id=$1 AND c.school_id=$2
	`,classID,schoolID).Scan(&impact.ClassID,&impact.ClassName,&impact.ActiveSectionCount,&impact.InactiveSectionCount,
		&impact.ActiveEnrollmentCount,&impact.HistoricalEnrollmentCount,&impact.ActiveCurriculumCount,
		&impact.ActiveTeacherAssignmentCount,&impact.ActiveClassInchargeCount,&impact.ImpactFingerprint)
	if err!=nil{return nil,academicWriteError(err)}
	rows,err:=tx.Query(ctx,`
		SELECT DISTINCT st.id,st.full_name,COALESCE(e.admission_number,''),s.name,ay.name
		FROM student_enrollments e
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		JOIN sections s ON s.id=e.section_id AND s.school_id=e.school_id
		JOIN academic_years ay ON ay.id=e.academic_year_id AND ay.school_id=e.school_id
		WHERE e.school_id=$1 AND e.class_id=$2 AND e.status='active'
		ORDER BY st.full_name,st.id,s.name
	`,schoolID,classID)
	if err!=nil{return nil,fmt.Errorf("load affected class students: %w",err)}
	impact.AffectedStudents=make([]ClassDeactivationStudent,0)
	for rows.Next(){var item ClassDeactivationStudent;if err:=rows.Scan(&item.StudentID,&item.FullName,&item.AdmissionNumber,&item.SectionName,&item.AcademicYearName);err!=nil{rows.Close();return nil,fmt.Errorf("scan affected class student: %w",err)};impact.AffectedStudents=append(impact.AffectedStudents,item)}
	if err:=rows.Err();err!=nil{rows.Close();return nil,fmt.Errorf("iterate affected class students: %w",err)}
	rows.Close()
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit class deactivation preview: %w",err)}
	return &impact,nil
}
func (r *AcademicRepository) SaveClassWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, classID int64, name string, order int, expectedFingerprint string) (*AcademicClass,error) {
	if r==nil||r.db==nil||classID<=0||strings.TrimSpace(expectedFingerprint)==""{return nil,ErrInvalidAcademicInput}
	name=strings.TrimSpace(name);if len([]rune(name))<2||len([]rune(name))>100||order<0{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin confirmed class deactivation: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"classes.manage");err!=nil{return nil,err}
	var lockedID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM classes WHERE id=$1 AND school_id=$2 AND is_active=TRUE FOR UPDATE`,classID,schoolID).Scan(&lockedID);err!=nil{return nil,academicWriteError(err)}
	if err=lockClassDeactivationDependencies(ctx,tx,schoolID,classID);err!=nil{return nil,err}
	currentFingerprint,err:=classDeactivationFingerprint(ctx,tx,schoolID,classID);if err!=nil{return nil,err}
	if currentFingerprint!=expectedFingerprint{return nil,ErrAcademicImpactChanged}
	var item AcademicClass
	err=tx.QueryRow(ctx,`UPDATE classes SET name=$3,display_order=$4,is_active=FALSE,updated_at=NOW()
		WHERE school_id=$1 AND id=$2 RETURNING id,school_id,name,display_order,is_active,created_at,updated_at`,
		schoolID,classID,name,order).Scan(&item.ID,&item.SchoolID,&item.Name,&item.DisplayOrder,&item.IsActive,&item.CreatedAt,&item.UpdatedAt)
	if err!=nil{return nil,academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.class.deactivated','class',$3,'success',jsonb_build_object('name',$4::text,'impact_fingerprint',$5::text,'is_active',FALSE))`,
		actorID,schoolID,fmt.Sprint(item.ID),item.Name,currentFingerprint);err!=nil{return nil,fmt.Errorf("audit confirmed class deactivation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit confirmed class deactivation: %w",err)}
	return &item,nil
}

func (r *AcademicRepository) SaveClass(ctx context.Context, actorID, schoolID, id int64, name string, order int, active bool) (*AcademicClass, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name = strings.TrimSpace(name)
	if len([]rune(name)) < 2 || len([]rune(name)) > 100 || order < 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin class save: %w", err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "classes.manage"); err != nil { return nil, err }
	if id>0 && !active { var wasActive bool; checkErr:=tx.QueryRow(ctx, `SELECT is_active FROM classes WHERE id=$1 AND school_id=$2`,id,schoolID).Scan(&wasActive); if checkErr!=nil { return nil,academicWriteError(checkErr) }; if wasActive { return nil,ErrClassDeactivationConfirmationRequired } }
	var v AcademicClass
	if id == 0 {
		err = tx.QueryRow(ctx, `INSERT INTO classes(school_id,name,display_order,is_active) VALUES($1,$2,$3,$4)
			RETURNING id,school_id,name,display_order,is_active,created_at,updated_at`, schoolID,name,order,active).
			Scan(&v.ID,&v.SchoolID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err = tx.QueryRow(ctx, `UPDATE classes SET name=$3,display_order=$4,is_active=$5,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 RETURNING id,school_id,name,display_order,is_active,created_at,updated_at`,
			schoolID,id,name,order,active).Scan(&v.ID,&v.SchoolID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	}
	if err != nil { return nil, academicWriteError(err) }
	action := "school.class.updated"; if id == 0 { action = "school.class.created" }
	if _,err = tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'class',$4,'success',jsonb_build_object('name',$5::text,'is_active',$6::boolean))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),v.Name,v.IsActive); err != nil { return nil,fmt.Errorf("audit class save: %w",err) }
	if err=tx.Commit(ctx); err!=nil { return nil,fmt.Errorf("commit class save: %w",err) }
	return &v,nil
}

func (r *AcademicRepository) ListSections(ctx context.Context,actorID,schoolID,yearID int64)([]AcademicSection,error){
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"classes.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT s.id,s.school_id,s.academic_year_id,s.class_id,c.name,s.name,s.medium,s.display_order,s.is_active,s.created_at,s.updated_at
		FROM sections s JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		WHERE s.school_id=$1 AND ($2=0 OR s.academic_year_id=$2)
		ORDER BY s.academic_year_id,c.display_order,c.id,s.display_order,s.id`,schoolID,yearID);if err!=nil{return nil,fmt.Errorf("list sections: %w",err)};defer rows.Close()
	items:=make([]AcademicSection,0);for rows.Next(){var v AcademicSection;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.ClassName,&v.Name,&v.Medium,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan section: %w",err)};items=append(items,v)};if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate sections: %w",err)};return items,nil
}
func lockSectionDeactivationDependencies(ctx context.Context, tx pgx.Tx, schoolID, sectionID int64) error {
	for _, query := range []string{
		`SELECT id FROM student_enrollments WHERE school_id=$1 AND section_id=$2 FOR UPDATE`,
		`SELECT id FROM class_curricula WHERE school_id=$1 AND section_id=$2 FOR UPDATE`,
		`SELECT id FROM teacher_subject_assignments WHERE school_id=$1 AND section_id=$2 FOR UPDATE`,
		`SELECT id FROM class_incharge_assignments WHERE school_id=$1 AND section_id=$2 FOR UPDATE`,
	} {
		rows, err := tx.Query(ctx, query, schoolID, sectionID)
		if err != nil { return fmt.Errorf("lock section dependency rows: %w", err) }
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil { rows.Close(); return fmt.Errorf("scan locked section dependency: %w", err) }
		}
		if err := rows.Err(); err != nil { rows.Close(); return fmt.Errorf("iterate locked section dependencies: %w", err) }
		rows.Close()
	}
	return nil
}

func sectionDeactivationFingerprint(ctx context.Context, tx pgx.Tx, schoolID, sectionID int64) (string, error) {
	var fingerprint string
	err := tx.QueryRow(ctx, `
		SELECT md5(concat_ws('|',
			COALESCE((SELECT s.name || ':' || COALESCE(s.medium,'') || ':' || s.display_order::text || ':' || s.is_active::text FROM sections s WHERE s.school_id=$1 AND s.id=$2),''),
			COALESCE((SELECT string_agg(e.id::text || ':' || e.status, ',' ORDER BY e.id) FROM student_enrollments e WHERE e.school_id=$1 AND e.section_id=$2), ''),
			COALESCE((SELECT string_agg(cc.id::text || ':' || cc.status, ',' ORDER BY cc.id) FROM class_curricula cc WHERE cc.school_id=$1 AND cc.section_id=$2), ''),
			COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a WHERE a.school_id=$1 AND a.section_id=$2), ''),
			COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM class_incharge_assignments a WHERE a.school_id=$1 AND a.section_id=$2), '')
		))
	`, schoolID, sectionID).Scan(&fingerprint)
	if err != nil { return "", fmt.Errorf("calculate section dependency fingerprint: %w", err) }
	return fingerprint, nil
}

func (r *AcademicRepository) PreviewSectionDeactivation(ctx context.Context, actorID, schoolID, sectionID int64) (*SectionDeactivationImpact, error) {
	if r == nil || r.db == nil || sectionID <= 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx)
	if err != nil { return nil, fmt.Errorf("begin section deactivation preview: %w", err) }
	defer func() { _ = tx.Rollback(ctx) }()
	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "classes.manage"); err != nil { return nil, err }

	var impact SectionDeactivationImpact
	err = tx.QueryRow(ctx, `
		SELECT s.id, s.name, c.name, ay.name,
			(SELECT COUNT(*) FROM student_enrollments e WHERE e.school_id=s.school_id AND e.section_id=s.id AND e.status='active'),
			(SELECT COUNT(*) FROM student_enrollments e WHERE e.school_id=s.school_id AND e.section_id=s.id AND e.status<>'active'),
			(SELECT COUNT(*) FROM class_curricula cc WHERE cc.school_id=s.school_id AND cc.section_id=s.id AND cc.status='active'),
			(SELECT COUNT(*) FROM teacher_subject_assignments a WHERE a.school_id=s.school_id AND a.section_id=s.id AND a.revoked_at IS NULL),
			(SELECT COUNT(*) FROM class_incharge_assignments a WHERE a.school_id=s.school_id AND a.section_id=s.id AND a.revoked_at IS NULL),
			md5(concat_ws('|',
				COALESCE((SELECT s.name || ':' || COALESCE(s.medium,'') || ':' || s.display_order::text || ':' || s.is_active::text FROM sections section_row WHERE section_row.school_id=s.school_id AND section_row.id=s.id),''),
				COALESCE((SELECT string_agg(e.id::text || ':' || e.status, ',' ORDER BY e.id) FROM student_enrollments e WHERE e.school_id=s.school_id AND e.section_id=s.id), ''),
				COALESCE((SELECT string_agg(cc.id::text || ':' || cc.status, ',' ORDER BY cc.id) FROM class_curricula cc WHERE cc.school_id=s.school_id AND cc.section_id=s.id), ''),
				COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a WHERE a.school_id=s.school_id AND a.section_id=s.id), ''),
				COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM class_incharge_assignments a WHERE a.school_id=s.school_id AND a.section_id=s.id), '')
			))
		FROM sections s
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN academic_years ay ON ay.id=s.academic_year_id AND ay.school_id=s.school_id
		WHERE s.id=$1 AND s.school_id=$2
	`, sectionID, schoolID).Scan(&impact.SectionID, &impact.SectionName, &impact.ClassName, &impact.AcademicYearName,
		&impact.ActiveEnrollmentCount, &impact.HistoricalEnrollmentCount, &impact.ActiveCurriculumCount,
		&impact.ActiveTeacherAssignmentCount, &impact.ActiveClassInchargeCount, &impact.ImpactFingerprint)
	if err != nil { return nil, academicWriteError(err) }
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT st.id, st.full_name, COALESCE(e.admission_number,''), e.status
		FROM student_enrollments e
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		WHERE e.school_id=$1 AND e.section_id=$2 AND e.status='active'
		ORDER BY st.full_name, st.id
	`, schoolID, sectionID)
	if err != nil { return nil, fmt.Errorf("load affected section students: %w", err) }
	defer rows.Close()
	impact.AffectedStudents = make([]SectionDeactivationStudent, 0)
	for rows.Next() {
		var student SectionDeactivationStudent
		if err := rows.Scan(&student.StudentID, &student.FullName, &student.AdmissionNumber, &student.EnrollmentStatus); err != nil {
			return nil, fmt.Errorf("scan affected section student: %w", err)
		}
		impact.AffectedStudents = append(impact.AffectedStudents, student)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate affected section students: %w", err) }
	if err = tx.Commit(ctx); err != nil { return nil, fmt.Errorf("commit section deactivation preview: %w", err) }
	return &impact, nil
}

func (r *AcademicRepository) SaveSectionWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, id, yearID, classID int64, name string, medium *string, order int, expectedFingerprint string) (*AcademicSection, error) {
	if r == nil || r.db == nil || id <= 0 || strings.TrimSpace(expectedFingerprint)=="" { return nil, ErrInvalidAcademicInput }
	name = strings.TrimSpace(name)
	if medium != nil { value:=strings.ToLower(strings.TrimSpace(*medium)); if value=="" { medium=nil } else if value!="hindi" && value!="english" { return nil,ErrInvalidAcademicInput } else { medium=&value } }
	if name == "" || len(name) > 40 || order < 0 || yearID <= 0 || classID <= 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin confirmed section deactivation: %w",err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"classes.manage"); err!=nil { return nil,err }
	var lockedID int64
	if err=tx.QueryRow(ctx, `SELECT id FROM sections WHERE id=$1 AND school_id=$2 AND academic_year_id=$3 AND class_id=$4 AND is_active=TRUE FOR UPDATE`, id,schoolID,yearID,classID).Scan(&lockedID); err!=nil { return nil,academicWriteError(err) }
	if err = lockSectionDeactivationDependencies(ctx,tx,schoolID,id); err != nil { return nil,err }
	currentFingerprint, err := sectionDeactivationFingerprint(ctx,tx,schoolID,id)
	if err != nil { return nil,err }
	if currentFingerprint != expectedFingerprint { return nil,ErrAcademicImpactChanged }
	var v AcademicSection
	err=tx.QueryRow(ctx, `UPDATE sections SET name=$4,medium=$5,display_order=$6,is_active=FALSE,updated_at=NOW()
		WHERE school_id=$1 AND id=$2 AND academic_year_id=$3 AND class_id=$7
		RETURNING id,school_id,academic_year_id,class_id,name,medium,display_order,is_active,created_at,updated_at`,
		schoolID,id,yearID,name,medium,order,classID).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.Medium,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	if err!=nil { return nil,academicWriteError(err) }
	if _,err=tx.Exec(ctx, `INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.section.deactivated','section',$3,'success',jsonb_build_object('name',$4::text,'impact_fingerprint',$5::text,'is_active',FALSE))`,
		actorID,schoolID,fmt.Sprint(v.ID),v.Name,currentFingerprint); err!=nil { return nil,fmt.Errorf("audit confirmed section deactivation: %w",err) }
	if err=tx.Commit(ctx); err!=nil { return nil,fmt.Errorf("commit confirmed section deactivation: %w",err) }
	return &v,nil
}

func (r *AcademicRepository) SaveSection(ctx context.Context, actorID, schoolID, id, yearID, classID int64, name string, order int, active bool) (*AcademicSection, error) {
	return r.SaveSectionWithMedium(ctx,actorID,schoolID,id,yearID,classID,name,nil,order,active)
}
func (r *AcademicRepository) SaveSectionWithMedium(ctx context.Context, actorID, schoolID, id, yearID, classID int64, name string, medium *string, order int, active bool) (*AcademicSection, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name = strings.TrimSpace(name)
	if medium != nil { value:=strings.ToLower(strings.TrimSpace(*medium)); if value=="" { medium=nil } else if value!="hindi" && value!="english" { return nil,ErrInvalidAcademicInput } else { medium=&value } }
	if name == "" || len(name) > 40 || order < 0 || yearID <= 0 || classID <= 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin section save: %w",err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"classes.manage"); err!=nil { return nil,err }
	if id>0 && !active { var wasActive bool; checkErr:=tx.QueryRow(ctx, `SELECT is_active FROM sections WHERE id=$1 AND school_id=$2`,id,schoolID).Scan(&wasActive); if checkErr!=nil { return nil,academicWriteError(checkErr) }; if wasActive { return nil,ErrSectionDeactivationConfirmationRequired } }
	var v AcademicSection
	if id==0 {
		err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name,medium,display_order,is_active)
			VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,school_id,academic_year_id,class_id,name,medium,display_order,is_active,created_at,updated_at`,
			schoolID,yearID,classID,name,medium,order,active).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.Medium,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err=tx.QueryRow(ctx,`UPDATE sections SET name=$4,medium=$5,display_order=$6,is_active=$7,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 AND academic_year_id=$3 AND class_id=$8
			RETURNING id,school_id,academic_year_id,class_id,name,medium,display_order,is_active,created_at,updated_at`,
			schoolID,id,yearID,name,medium,order,active,classID).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.Medium,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	}
	if err!=nil { return nil,academicWriteError(err) }
	action:="school.section.updated";if id==0{action="school.section.created"}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'section',$4,'success',jsonb_build_object('name',$5::text,'medium',$6::text,'is_active',$7::boolean))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),v.Name,v.Medium,v.IsActive);err!=nil{return nil,fmt.Errorf("audit section save: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit section save: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) ListSubjects(ctx context.Context,actorID,schoolID int64)([]AcademicSubject,error){
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"subjects.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT id,school_id,name,subject_code,is_active,created_at,updated_at FROM subjects WHERE school_id=$1 ORDER BY lower(name),id`,schoolID);if err!=nil{return nil,fmt.Errorf("list subjects: %w",err)};defer rows.Close()
	items:=make([]AcademicSubject,0);for rows.Next(){var v AcademicSubject;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.Name,&v.Code,&v.IsActive,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan subject: %w",err)};items=append(items,v)};if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate subjects: %w",err)};return items,nil
}

func lockSubjectDeactivationDependencies(ctx context.Context, tx pgx.Tx, schoolID, subjectID int64) error {
	for _,query:=range []string{
		`SELECT id FROM class_curriculum_subjects WHERE school_id=$1 AND subject_id=$2 ORDER BY id FOR UPDATE`,
		`SELECT sel.id FROM student_enrollment_subject_selections sel
			JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
			WHERE sel.school_id=$1 AND cs.subject_id=$2 ORDER BY sel.id FOR UPDATE OF sel`,
		`SELECT id FROM teacher_subject_assignments WHERE school_id=$1 AND subject_id=$2 ORDER BY id FOR UPDATE`,
	}{
		rows,err:=tx.Query(ctx,query,schoolID,subjectID);if err!=nil{return fmt.Errorf("lock subject dependency rows: %w",err)}
		for rows.Next(){var id int64;if err:=rows.Scan(&id);err!=nil{rows.Close();return fmt.Errorf("scan locked subject dependency: %w",err)}}
		if err:=rows.Err();err!=nil{rows.Close();return fmt.Errorf("iterate locked subject dependencies: %w",err)}
		rows.Close()
	}
	return nil
}

func subjectDeactivationFingerprint(ctx context.Context, tx pgx.Tx, schoolID, subjectID int64) (string,error) {
	var fingerprint string
	err:=tx.QueryRow(ctx,`
		SELECT md5(concat_ws('|',
			COALESCE((SELECT sub.name || ':' || COALESCE(sub.subject_code,'') || ':' || sub.is_active::text FROM subjects sub WHERE sub.school_id=$1 AND sub.id=$2),''),
			COALESCE((SELECT string_agg(cs.id::text || ':' || cs.curriculum_id::text || ':' || cs.is_active::text, ',' ORDER BY cs.id) FROM class_curriculum_subjects cs WHERE cs.school_id=$1 AND cs.subject_id=$2),''),
			COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a WHERE a.school_id=$1 AND a.subject_id=$2),''),
			COALESCE((SELECT string_agg(sel.id::text || ':' || COALESCE(sel.revoked_at::text,'active'), ',' ORDER BY sel.id)
				FROM student_enrollment_subject_selections sel
				JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
				WHERE sel.school_id=$1 AND cs.subject_id=$2),'')
		))`,schoolID,subjectID).Scan(&fingerprint)
	if err!=nil{return "",fmt.Errorf("calculate subject deactivation fingerprint: %w",err)}
	return fingerprint,nil
}

func (r *AcademicRepository) PreviewSubjectDeactivation(ctx context.Context, actorID, schoolID, subjectID int64) (*SubjectDeactivationImpact,error) {
	if r==nil||r.db==nil||subjectID<=0{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin subject deactivation preview: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"subjects.manage");err!=nil{return nil,err}
	var impact SubjectDeactivationImpact
	err=tx.QueryRow(ctx,`
		SELECT s.id,s.name,
			(SELECT COUNT(*) FROM class_curriculum_subjects cs WHERE cs.school_id=s.school_id AND cs.subject_id=s.id),
			(SELECT COUNT(*) FROM teacher_subject_assignments a WHERE a.school_id=s.school_id AND a.subject_id=s.id AND a.revoked_at IS NULL),
			(SELECT COUNT(*) FROM student_enrollment_subject_selections sel
				JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
				WHERE sel.school_id=s.school_id AND cs.subject_id=s.id AND sel.revoked_at IS NULL),
			(SELECT COUNT(*) FROM student_enrollment_subject_selections sel
				JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
				WHERE sel.school_id=s.school_id AND cs.subject_id=s.id AND sel.revoked_at IS NOT NULL),
			md5(concat_ws('|',
				COALESCE((SELECT sub.name || ':' || COALESCE(sub.subject_code,'') || ':' || sub.is_active::text FROM subjects sub WHERE sub.school_id=s.school_id AND sub.id=s.id),''),
				COALESCE((SELECT string_agg(cs.id::text || ':' || cs.curriculum_id::text || ':' || cs.is_active::text, ',' ORDER BY cs.id) FROM class_curriculum_subjects cs WHERE cs.school_id=s.school_id AND cs.subject_id=s.id),''),
				COALESCE((SELECT string_agg(a.id::text || ':' || COALESCE(a.revoked_at::text,'active'), ',' ORDER BY a.id) FROM teacher_subject_assignments a WHERE a.school_id=s.school_id AND a.subject_id=s.id),''),
				COALESCE((SELECT string_agg(sel.id::text || ':' || COALESCE(sel.revoked_at::text,'active'), ',' ORDER BY sel.id)
					FROM student_enrollment_subject_selections sel
					JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
					WHERE sel.school_id=s.school_id AND cs.subject_id=s.id),'')
			))
		FROM subjects s WHERE s.id=$1 AND s.school_id=$2 AND s.is_active=TRUE
	`,subjectID,schoolID).Scan(&impact.SubjectID,&impact.SubjectName,&impact.CurriculumSubjectCount,&impact.ActiveTeacherAssignmentCount,&impact.ActiveStudentSelectionCount,&impact.HistoricalStudentSelectionCount,&impact.ImpactFingerprint)
	if err!=nil{return nil,academicWriteError(err)}
	rows,err:=tx.Query(ctx,`
		SELECT DISTINCT st.id,st.full_name,COALESCE(e.admission_number,''),cl.name,sec.name,ay.name
		FROM student_enrollment_subject_selections sel
		JOIN class_curriculum_subjects cs ON cs.id=sel.curriculum_subject_id AND cs.curriculum_id=sel.curriculum_id AND cs.school_id=sel.school_id
		JOIN student_enrollment_curricula ec ON ec.id=sel.enrollment_curriculum_id AND ec.curriculum_id=sel.curriculum_id AND ec.school_id=sel.school_id
		JOIN student_enrollments e ON e.id=ec.enrollment_id AND e.school_id=ec.school_id
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		JOIN classes cl ON cl.id=e.class_id AND cl.school_id=e.school_id
		JOIN sections sec ON sec.id=e.section_id AND sec.school_id=e.school_id
		JOIN academic_years ay ON ay.id=e.academic_year_id AND ay.school_id=e.school_id
		WHERE sel.school_id=$1 AND cs.subject_id=$2 AND sel.revoked_at IS NULL AND ec.status='active'
		ORDER BY st.full_name,st.id,cl.name,sec.name
	`,schoolID,subjectID)
	if err!=nil{return nil,fmt.Errorf("load affected subject students: %w",err)}
	impact.AffectedStudents=make([]SubjectDeactivationStudent,0)
	for rows.Next(){var item SubjectDeactivationStudent;if err:=rows.Scan(&item.StudentID,&item.FullName,&item.AdmissionNumber,&item.ClassName,&item.SectionName,&item.AcademicYearName);err!=nil{rows.Close();return nil,fmt.Errorf("scan affected subject student: %w",err)};impact.AffectedStudents=append(impact.AffectedStudents,item)}
	if err:=rows.Err();err!=nil{rows.Close();return nil,fmt.Errorf("iterate affected subject students: %w",err)}
	rows.Close()
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit subject deactivation preview: %w",err)}
	return &impact,nil
}

func (r *AcademicRepository) SaveSubjectWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, subjectID int64, name string, code *string, fingerprint string) (*AcademicSubject,error) {
	if r==nil||r.db==nil||subjectID<=0||strings.TrimSpace(fingerprint)==""{return nil,ErrInvalidAcademicInput}
	name=strings.TrimSpace(name);if len([]rune(name))<2||len([]rune(name))>120{return nil,ErrInvalidAcademicInput}
	if code!=nil{v:=strings.TrimSpace(*code);if v==""{code=nil}else if len(v)>40{return nil,ErrInvalidAcademicInput}else{code=&v}}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin confirmed subject deactivation: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"subjects.manage");err!=nil{return nil,err}
	var lockedID int64
	if err=tx.QueryRow(ctx,`SELECT id FROM subjects WHERE id=$1 AND school_id=$2 AND is_active=TRUE FOR UPDATE`,subjectID,schoolID).Scan(&lockedID);err!=nil{return nil,academicWriteError(err)}
	if err=lockSubjectDeactivationDependencies(ctx,tx,schoolID,subjectID);err!=nil{return nil,err}
	current,err:=subjectDeactivationFingerprint(ctx,tx,schoolID,subjectID);if err!=nil{return nil,err}
	if current!=fingerprint{return nil,ErrAcademicImpactChanged}
	var item AcademicSubject
	err=tx.QueryRow(ctx,`UPDATE subjects SET name=$3,subject_code=$4,is_active=FALSE,updated_at=NOW()
		WHERE school_id=$1 AND id=$2 RETURNING id,school_id,name,subject_code,is_active,created_at,updated_at`,
		schoolID,subjectID,name,code).Scan(&item.ID,&item.SchoolID,&item.Name,&item.Code,&item.IsActive,&item.CreatedAt,&item.UpdatedAt)
	if err!=nil{return nil,academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.subject.deactivated','subject',$3,'success',jsonb_build_object('name',$4::text,'impact_fingerprint',$5::text,'is_active',FALSE))`,
		actorID,schoolID,fmt.Sprint(item.ID),item.Name,current);err!=nil{return nil,fmt.Errorf("audit confirmed subject deactivation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit confirmed subject deactivation: %w",err)}
	return &item,nil
}

func (r *AcademicRepository) SaveSubject(ctx context.Context, actorID, schoolID, id int64, name string, code *string, active bool) (*AcademicSubject,error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name=strings.TrimSpace(name)
	if code!=nil { value:=strings.TrimSpace(*code); if value=="" { code=nil } else if len(value)>40 { return nil,ErrInvalidAcademicInput } else { code=&value } }
	if len([]rune(name))<2||len([]rune(name))>120{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin subject save: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"subjects.manage");err!=nil{return nil,err}
	if id>0 && !active { var wasActive bool; checkErr:=tx.QueryRow(ctx, `SELECT is_active FROM subjects WHERE id=$1 AND school_id=$2`,id,schoolID).Scan(&wasActive); if checkErr!=nil{return nil,academicWriteError(checkErr)}; if wasActive{return nil,ErrSubjectDeactivationConfirmationRequired} }
	var v AcademicSubject
	if id==0 {
		err=tx.QueryRow(ctx,`INSERT INTO subjects(school_id,name,subject_code,is_active) VALUES($1,$2,$3,$4)
			RETURNING id,school_id,name,subject_code,is_active,created_at,updated_at`,schoolID,name,code,active).
			Scan(&v.ID,&v.SchoolID,&v.Name,&v.Code,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err=tx.QueryRow(ctx,`UPDATE subjects SET name=$3,subject_code=$4,is_active=$5,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 RETURNING id,school_id,name,subject_code,is_active,created_at,updated_at`,
			schoolID,id,name,code,active).Scan(&v.ID,&v.SchoolID,&v.Name,&v.Code,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	}
	if err!=nil{return nil,academicWriteError(err)}
	action:="school.subject.updated";if id==0{action="school.subject.created"}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'subject',$4,'success',jsonb_build_object('name',$5::text,'is_active',$6::boolean))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),v.Name,v.IsActive);err!=nil{return nil,fmt.Errorf("audit subject save: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit subject save: %w",err)}
	return &v,nil
}

