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
	ErrGuardianAccessDenied = errors.New("guardian operation is not authorized")
	ErrGuardianNotFound = errors.New("guardian record not found")
	ErrGuardianConflict = errors.New("guardian record conflicts with an existing relationship")
	ErrInvalidGuardianInput = errors.New("invalid guardian data")
)

type Guardian struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	FullName string `json:"full_name"`
	MobileNumber *string `json:"mobile_number,omitempty"`
	EmailAddress *string `json:"email_address,omitempty"`
	AlternateMobileNumber *string `json:"alternate_mobile_number,omitempty"`
	Occupation *string `json:"occupation,omitempty"`
	Education *string `json:"education,omitempty"`
	WorkplaceName *string `json:"workplace_name,omitempty"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StudentGuardian struct {
	ID int64 `json:"id"`
	StudentID int64 `json:"student_id"`
	GuardianID int64 `json:"guardian_id"`
	GuardianName string `json:"guardian_name"`
	MobileNumber *string `json:"mobile_number,omitempty"`
	EmailAddress *string `json:"email_address,omitempty"`
	AlternateMobileNumber *string `json:"alternate_mobile_number,omitempty"`
	Occupation *string `json:"occupation,omitempty"`
	Education *string `json:"education,omitempty"`
	WorkplaceName *string `json:"workplace_name,omitempty"`
	RelationshipType string `json:"relationship_type"`
	IsPrimaryContact bool `json:"is_primary_contact"`
	IsEmergencyContact bool `json:"is_emergency_contact"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type StudentRelatedStudent struct {
	StudentID int64 `json:"student_id"`
	FullName string `json:"full_name"`
	AdmissionNumber *string `json:"admission_number,omitempty"`
	ClassName *string `json:"class_name,omitempty"`
	SectionName *string `json:"section_name,omitempty"`
	AcademicYearName *string `json:"academic_year_name,omitempty"`
	SharedGuardianNames []string `json:"shared_guardian_names"`
}

func authorizeGuardian(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if db == nil || actorID <= 0 || schoolID <= 0 || permission == "" {
		return ErrGuardianAccessDenied
	}

	// Keep repository authorization aligned with the same live RBAC evaluator
	// used by AuthMiddleware. A role-permission join alone would incorrectly
	// ignore delegation revocation, user ALLOW overrides, and exact delegation
	// identity, potentially denying valid grants or accepting stale role grants.
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
		return ErrGuardianAccessDenied
	}
	if err != nil {
		return fmt.Errorf("load guardian actor: %w", err)
	}
	if principal.Type != "staff" || principal.Status != "active" ||
		principal.SchoolID != schoolID || userStatus != "active" || schoolStatus != "active" {
		return ErrGuardianAccessDenied
	}
	if strings.HasPrefix(permission, "guardians.") {
		teacherOnly, roleErr := isTeacherWithoutSchoolAdmin(ctx, db, actorID, schoolID)
		if roleErr != nil {
			return fmt.Errorf("enforce teacher guardian-data boundary: %w", roleErr)
		}
		if teacherOnly {
			return ErrGuardianAccessDenied
		}
	}

	effective, err := loadEffectivePermissions(ctx, db, &principal)
	if err != nil {
		return fmt.Errorf("authorize guardian operation: %w", err)
	}
	for _, key := range effective {
		if key == permission {
			return nil
		}
	}
	return ErrGuardianAccessDenied
}

func guardianWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": return ErrGuardianConflict
		case "23503", "23514", "22001", "P0001":
			message := strings.ToLower(pgErr.Message)
			if strings.Contains(message, "active parent link") {
				return ErrGuardianConflict
			}
			if strings.Contains(message, "revocation") || strings.Contains(message, "same-school") {
				return ErrGuardianAccessDenied
			}
			return ErrInvalidGuardianInput
		}
	}
	if errors.Is(err, pgx.ErrNoRows) { return ErrGuardianNotFound }
	return err
}

func (r *AcademicRepository) ListGuardians(ctx context.Context, actorID, schoolID int64, search, status string, limit, offset int) ([]Guardian, int64, error) {
	if r == nil || r.db == nil { return nil, 0, ErrGuardianAccessDenied }
	if err := authorizeGuardian(ctx, r.db, actorID, schoolID, "guardians.read"); err != nil { return nil, 0, err }
	search = strings.TrimSpace(search); status = strings.TrimSpace(status)
	if limit < 1 || limit > 100 { limit = 50 }; if offset < 0 { offset = 0 }
	if status != "" && status != "active" && status != "inactive" && status != "archived" { return nil, 0, ErrInvalidGuardianInput }
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM guardians WHERE school_id=$1 AND ($2='' OR status=$2)
		AND ($3='' OR full_name ILIKE '%'||$3||'%' OR COALESCE(mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(alternate_mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(email_address,'') ILIKE '%'||$3||'%')`,
		schoolID, status, search).Scan(&total); err != nil { return nil, 0, fmt.Errorf("count guardians: %w", err) }
	rows, err := r.db.Query(ctx, `SELECT id,school_id,full_name,mobile_number,email_address,alternate_mobile_number,occupation,education,workplace_name,status,created_at,updated_at
		FROM guardians WHERE school_id=$1 AND ($2='' OR status=$2)
		AND ($3='' OR full_name ILIKE '%'||$3||'%' OR COALESCE(mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(alternate_mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(email_address,'') ILIKE '%'||$3||'%')
		ORDER BY LOWER(full_name),id LIMIT $4 OFFSET $5`, schoolID,status,search,limit,offset)
	if err != nil { return nil, 0, fmt.Errorf("list guardians: %w", err) }; defer rows.Close()
	items:=make([]Guardian,0)
	for rows.Next(){var v Guardian;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.Status,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,0,fmt.Errorf("scan guardian: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate guardians: %w",err)}
	return items,total,nil
}

func (r *AcademicRepository) GetGuardian(ctx context.Context, actorID, schoolID, guardianID int64) (*Guardian,error) {
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	if err:=authorizeGuardian(ctx,r.db,actorID,schoolID,"guardians.read");err!=nil{return nil,err}
	var v Guardian
	err:=r.db.QueryRow(ctx,`SELECT id,school_id,full_name,mobile_number,email_address,alternate_mobile_number,occupation,education,workplace_name,status,created_at,updated_at FROM guardians WHERE school_id=$1 AND id=$2`,schoolID,guardianID).
		Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.Status,&v.CreatedAt,&v.UpdatedAt)
	if err!=nil{return nil,guardianWriteError(err)}
	return &v,nil
}

func (r *AcademicRepository) SaveGuardian(ctx context.Context, actorID,schoolID,guardianID int64,fullName string,mobile,email,alternateMobile,occupation,education,workplace *string,status string)(*Guardian,error){
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	fullName=strings.TrimSpace(fullName);status=strings.TrimSpace(status)
	if fullName==""||len([]rune(fullName))>200||(status!="active"&&status!="inactive"&&status!="archived"){return nil,ErrInvalidGuardianInput}
	if mobile!=nil{v:=strings.TrimSpace(*mobile);if v==""{mobile=nil}else if len(v)>30{return nil,ErrInvalidGuardianInput}else{mobile=&v}}
	if email!=nil{v:=strings.TrimSpace(*email);if v==""{email=nil}else if len(v)>254||!strings.Contains(v,"@"){return nil,ErrInvalidGuardianInput}else{email=&v}}
	for _, field := range []*struct{ value **string; max int }{{&alternateMobile,30},{&occupation,120},{&education,120},{&workplace,160}} { if *field.value != nil { value:=strings.TrimSpace(**field.value); if value=="" { *field.value=nil } else if len([]rune(value))>field.max { return nil,ErrInvalidGuardianInput } else { *field.value=&value } } }
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin guardian save: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeGuardian(ctx,tx,actorID,schoolID,"guardians.manage");err!=nil{return nil,err}
	var v Guardian
	if guardianID==0{
		err=tx.QueryRow(ctx,`INSERT INTO guardians(school_id,full_name,mobile_number,email_address,alternate_mobile_number,occupation,education,workplace_name,status)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id,school_id,full_name,mobile_number,email_address,alternate_mobile_number,occupation,education,workplace_name,status,created_at,updated_at`,
			schoolID,fullName,mobile,email,alternateMobile,occupation,education,workplace,status).Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.Status,&v.CreatedAt,&v.UpdatedAt)
	}else{
		err=tx.QueryRow(ctx,`UPDATE guardians SET full_name=$3,mobile_number=$4,email_address=$5,alternate_mobile_number=$6,occupation=$7,education=$8,workplace_name=$9,status=$10,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 RETURNING id,school_id,full_name,mobile_number,email_address,alternate_mobile_number,occupation,education,workplace_name,status,created_at,updated_at`,
			schoolID,guardianID,fullName,mobile,email,alternateMobile,occupation,education,workplace,status).Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.Status,&v.CreatedAt,&v.UpdatedAt)
	}
	if err!=nil{return nil,guardianWriteError(err)}
	action:="school.guardian.updated";if guardianID==0{action="school.guardian.created"}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'guardian',$4,'success',jsonb_build_object('status',$5::text))`,actorID,schoolID,action,fmt.Sprint(v.ID),v.Status);err!=nil{return nil,fmt.Errorf("audit guardian save: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit guardian save: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) ListStudentGuardians(ctx context.Context,actorID,schoolID,studentID int64,includeHistory bool)([]StudentGuardian,error){
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	if err:=authorizeGuardian(ctx,r.db,actorID,schoolID,"guardians.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT sg.id,sg.student_id,sg.guardian_id,g.full_name,g.mobile_number,g.email_address,g.alternate_mobile_number,g.occupation,g.education,g.workplace_name,
		sg.relationship_type,sg.is_primary_contact,sg.is_emergency_contact,sg.created_at,sg.revoked_at
		FROM student_guardians sg JOIN guardians g ON g.id=sg.guardian_id AND g.school_id=sg.school_id
		JOIN students s ON s.id=sg.student_id AND s.school_id=sg.school_id
		WHERE sg.school_id=$1 AND sg.student_id=$2 AND ($3=TRUE OR sg.revoked_at IS NULL)
		ORDER BY sg.revoked_at NULLS FIRST,sg.is_primary_contact DESC,LOWER(g.full_name),sg.id`,schoolID,studentID,includeHistory)
	if err!=nil{return nil,fmt.Errorf("list student guardians: %w",err)};defer rows.Close()
	items:=make([]StudentGuardian,0)
	for rows.Next(){var v StudentGuardian;if err:=rows.Scan(&v.ID,&v.StudentID,&v.GuardianID,&v.GuardianName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.RelationshipType,&v.IsPrimaryContact,&v.IsEmergencyContact,&v.CreatedAt,&v.RevokedAt);err!=nil{return nil,fmt.Errorf("scan student guardian: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate student guardians: %w",err)}
	return items,nil
}

// ListStudentRelatedStudents derives other active students who share at
// least one active guardian link. It does not assert a biological sibling
// relationship; the UI must label these as students sharing a guardian.
func (r *AcademicRepository) ListStudentRelatedStudents(ctx context.Context, actorID, schoolID, studentID int64) ([]StudentRelatedStudent, error) {
	if r == nil || r.db == nil { return nil, ErrGuardianAccessDenied }
	if studentID <= 0 { return nil, ErrInvalidGuardianInput }
	if err := authorizeGuardian(ctx, r.db, actorID, schoolID, "guardians.read"); err != nil { return nil, err }
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_basic"); err != nil {
		if errors.Is(err, ErrAcademicAccessDenied) { return nil, ErrGuardianAccessDenied }
		return nil, err
	}
	var studentExists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`, studentID, schoolID).Scan(&studentExists); err != nil {
		return nil, fmt.Errorf("check student for related-student list: %w", err)
	}
	if !studentExists { return nil, ErrGuardianNotFound }

	rows, err := r.db.Query(ctx, `
		SELECT related_student.id, related_student.full_name,
		       enrollment.admission_number, enrollment.class_name_snapshot,
		       enrollment.section_name_snapshot, academic_year.name,
		       COALESCE(array_agg(DISTINCT shared_guardian.full_name ORDER BY shared_guardian.full_name), ARRAY[]::text[])
		  FROM student_guardians source_link
		  JOIN guardians shared_guardian
		    ON shared_guardian.id=source_link.guardian_id
		   AND shared_guardian.school_id=source_link.school_id
		   AND shared_guardian.status='active'
		  JOIN student_guardians related_link
		    ON related_link.guardian_id=shared_guardian.id
		   AND related_link.school_id=shared_guardian.school_id
		   AND related_link.revoked_at IS NULL
		  JOIN students related_student
		    ON related_student.id=related_link.student_id
		   AND related_student.school_id=related_link.school_id
		   AND related_student.status='active'
		  LEFT JOIN student_enrollments enrollment
		    ON enrollment.student_id=related_student.id
		   AND enrollment.school_id=related_student.school_id
		   AND enrollment.academic_year_id=(
		       SELECT id FROM academic_years WHERE school_id=$1 AND is_current=TRUE
		   )
		   AND enrollment.status='active'
		  LEFT JOIN academic_years academic_year
		    ON academic_year.id=enrollment.academic_year_id
		 WHERE source_link.school_id=$1
		   AND source_link.student_id=$2
		   AND source_link.revoked_at IS NULL
		   AND related_student.id<>$2
		 GROUP BY related_student.id,related_student.full_name,enrollment.admission_number,
		          enrollment.class_name_snapshot,enrollment.section_name_snapshot,academic_year.name
		 ORDER BY LOWER(related_student.full_name),related_student.id
	`, schoolID, studentID)
	if err != nil { return nil, fmt.Errorf("list students sharing guardian: %w", err) }
	defer rows.Close()
	items := make([]StudentRelatedStudent, 0)
	for rows.Next() {
		var item StudentRelatedStudent
		if err := rows.Scan(&item.StudentID,&item.FullName,&item.AdmissionNumber,&item.ClassName,
			&item.SectionName,&item.AcademicYearName,&item.SharedGuardianNames); err != nil {
			return nil, fmt.Errorf("scan related student: %w", err)
		}
		items = append(items,item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate related students: %w", err) }
	return items,nil
}

func (r *AcademicRepository) LinkStudentGuardian(ctx context.Context,actorID,schoolID,studentID,guardianID int64,relationship string,primary,emergency bool)(*StudentGuardian,error){
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	relationship=strings.TrimSpace(relationship)
	if studentID<=0||guardianID<=0||relationship==""||len([]rune(relationship))>40{return nil,ErrInvalidGuardianInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin student guardian link: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeGuardian(ctx,tx,actorID,schoolID,"guardians.manage");err!=nil{return nil,err}
	var active bool
	if err=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2 AND status='active')`,studentID,schoolID).Scan(&active);err!=nil{return nil,fmt.Errorf("validate student: %w",err)}
	if !active{return nil,ErrGuardianNotFound}
	if err=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM guardians WHERE id=$1 AND school_id=$2 AND status='active')`,guardianID,schoolID).Scan(&active);err!=nil{return nil,fmt.Errorf("validate guardian: %w",err)}
	if !active{return nil,ErrGuardianNotFound}
	// Serialize primary-contact selection for a student.
	if err=tx.QueryRow(ctx,`SELECT id FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,studentID,schoolID).Scan(new(int64));err!=nil{return nil,guardianWriteError(err)}
	if primary { if _,err=tx.Exec(ctx,`UPDATE student_guardians SET is_primary_contact=FALSE,updated_at=NOW() WHERE school_id=$1 AND student_id=$2 AND revoked_at IS NULL`,schoolID,studentID);err!=nil{return nil,fmt.Errorf("clear prior primary contact: %w",err)} }
	var v StudentGuardian
	err=tx.QueryRow(ctx,`INSERT INTO student_guardians(school_id,student_id,guardian_id,relationship_type,is_primary_contact,is_emergency_contact)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		schoolID,studentID,guardianID,relationship,primary,emergency).Scan(&v.ID)
	if err!=nil{return nil,guardianWriteError(err)}
	err=tx.QueryRow(ctx,`SELECT sg.id,sg.student_id,sg.guardian_id,g.full_name,g.mobile_number,g.email_address,
		g.alternate_mobile_number,g.occupation,g.education,g.workplace_name,
		sg.relationship_type,sg.is_primary_contact,sg.is_emergency_contact,sg.created_at,sg.revoked_at
		FROM student_guardians sg JOIN guardians g ON g.id=sg.guardian_id AND g.school_id=sg.school_id
		WHERE sg.id=$1 AND sg.school_id=$2`,v.ID,schoolID).
		Scan(&v.ID,&v.StudentID,&v.GuardianID,&v.GuardianName,&v.MobileNumber,&v.EmailAddress,&v.AlternateMobileNumber,&v.Occupation,&v.Education,&v.WorkplaceName,&v.RelationshipType,&v.IsPrimaryContact,&v.IsEmergencyContact,&v.CreatedAt,&v.RevokedAt)
	if err!=nil{return nil,fmt.Errorf("load created student guardian link: %w",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student_guardian_link_created','student_guardian_link',$3,'success',
		jsonb_build_object('student_id',$4::bigint,'guardian_id',$5::bigint,'relationship_type',$6::text))`,actorID,schoolID,fmt.Sprint(v.ID),studentID,guardianID,relationship);err!=nil{return nil,fmt.Errorf("audit student guardian link: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit student guardian link: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) RevokeStudentGuardianLink(ctx context.Context,actorID,schoolID,linkID int64,reason string)error{
	if r==nil||r.db==nil{return ErrGuardianAccessDenied}
	reason=strings.TrimSpace(reason);if linkID<=0||reason==""||len([]rune(reason))>500{return ErrInvalidGuardianInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return fmt.Errorf("begin guardian link revocation: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeGuardian(ctx,tx,actorID,schoolID,"guardians.manage");err!=nil{return err}
	tag,err:=tx.Exec(ctx,`UPDATE student_guardians SET revoked_at=NOW(),revoked_by_principal_id=$3,revocation_reason=$4,updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND revoked_at IS NULL`,linkID,schoolID,actorID,reason)
	if err!=nil{return guardianWriteError(err)};if tag.RowsAffected()!=1{return ErrGuardianNotFound}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit guardian link revocation: %w",err)}
	return nil
}
