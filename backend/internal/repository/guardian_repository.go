package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	RelationshipType string `json:"relationship_type"`
	IsPrimaryContact bool `json:"is_primary_contact"`
	IsEmergencyContact bool `json:"is_emergency_contact"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func authorizeGuardian(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if db == nil || actorID <= 0 || schoolID <= 0 || permission == "" { return ErrGuardianAccessDenied }
	var allowed bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id=$1 AND ap.principal_type='staff' AND ap.status='active'
			  AND u.status='active' AND u.school_id=$2 AND s.status='active'
			  AND r.school_id=$2 AND r.scope_type='school' AND r.principal_type='staff'
			  AND p.permission_key=$3 AND p.scope_type='school' AND p.is_active=TRUE
			  AND NOT EXISTS (
				SELECT 1 FROM user_permission_overrides o
				WHERE o.principal_id=ap.id AND o.school_id=$2
				  AND o.permission_id=p.id AND o.effect='deny'
			  )
		)`, actorID, schoolID, permission).Scan(&allowed)
	if err != nil { return fmt.Errorf("authorize guardian operation: %w", err) }
	if !allowed { return ErrGuardianAccessDenied }
	return nil
}

func guardianWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": return ErrGuardianConflict
		case "23503", "23514", "22001", "P0001":
			if strings.Contains(strings.ToLower(pgErr.Message), "revocation") ||
				strings.Contains(strings.ToLower(pgErr.Message), "same-school") { return ErrGuardianAccessDenied }
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
		AND ($3='' OR full_name ILIKE '%'||$3||'%' OR COALESCE(mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(email_address,'') ILIKE '%'||$3||'%')`,
		schoolID, status, search).Scan(&total); err != nil { return nil, 0, fmt.Errorf("count guardians: %w", err) }
	rows, err := r.db.Query(ctx, `SELECT id,school_id,full_name,mobile_number,email_address,status,created_at,updated_at
		FROM guardians WHERE school_id=$1 AND ($2='' OR status=$2)
		AND ($3='' OR full_name ILIKE '%'||$3||'%' OR COALESCE(mobile_number,'') ILIKE '%'||$3||'%' OR COALESCE(email_address,'') ILIKE '%'||$3||'%')
		ORDER BY LOWER(full_name),id LIMIT $4 OFFSET $5`, schoolID,status,search,limit,offset)
	if err != nil { return nil, 0, fmt.Errorf("list guardians: %w", err) }; defer rows.Close()
	items:=make([]Guardian,0)
	for rows.Next(){var v Guardian;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.Status,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,0,fmt.Errorf("scan guardian: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,0,fmt.Errorf("iterate guardians: %w",err)}
	return items,total,nil
}

func (r *AcademicRepository) GetGuardian(ctx context.Context, actorID, schoolID, guardianID int64) (*Guardian,error) {
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	if err:=authorizeGuardian(ctx,r.db,actorID,schoolID,"guardians.read");err!=nil{return nil,err}
	var v Guardian
	err:=r.db.QueryRow(ctx,`SELECT id,school_id,full_name,mobile_number,email_address,status,created_at,updated_at FROM guardians WHERE school_id=$1 AND id=$2`,schoolID,guardianID).
		Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.Status,&v.CreatedAt,&v.UpdatedAt)
	if err!=nil{return nil,guardianWriteError(err)}
	return &v,nil
}

func (r *AcademicRepository) SaveGuardian(ctx context.Context, actorID,schoolID,guardianID int64,fullName string,mobile,email *string,status string)(*Guardian,error){
	if r==nil||r.db==nil{return nil,ErrGuardianAccessDenied}
	fullName=strings.TrimSpace(fullName);status=strings.TrimSpace(status)
	if fullName==""||len([]rune(fullName))>200||(status!="active"&&status!="inactive"&&status!="archived"){return nil,ErrInvalidGuardianInput}
	if mobile!=nil{v:=strings.TrimSpace(*mobile);if v==""{mobile=nil}else if len(v)>30{return nil,ErrInvalidGuardianInput}else{mobile=&v}}
	if email!=nil{v:=strings.TrimSpace(*email);if v==""{email=nil}else if len(v)>254||!strings.Contains(v,"@"){return nil,ErrInvalidGuardianInput}else{email=&v}}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin guardian save: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeGuardian(ctx,tx,actorID,schoolID,"guardians.manage");err!=nil{return nil,err}
	var v Guardian
	if guardianID==0{
		err=tx.QueryRow(ctx,`INSERT INTO guardians(school_id,full_name,mobile_number,email_address,status)
			VALUES($1,$2,$3,$4,$5) RETURNING id,school_id,full_name,mobile_number,email_address,status,created_at,updated_at`,
			schoolID,fullName,mobile,email,status).Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.Status,&v.CreatedAt,&v.UpdatedAt)
	}else{
		err=tx.QueryRow(ctx,`UPDATE guardians SET full_name=$3,mobile_number=$4,email_address=$5,status=$6,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 RETURNING id,school_id,full_name,mobile_number,email_address,status,created_at,updated_at`,
			schoolID,guardianID,fullName,mobile,email,status).Scan(&v.ID,&v.SchoolID,&v.FullName,&v.MobileNumber,&v.EmailAddress,&v.Status,&v.CreatedAt,&v.UpdatedAt)
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
	rows,err:=r.db.Query(ctx,`SELECT sg.id,sg.student_id,sg.guardian_id,g.full_name,g.mobile_number,g.email_address,
		sg.relationship_type,sg.is_primary_contact,sg.is_emergency_contact,sg.created_at,sg.revoked_at
		FROM student_guardians sg JOIN guardians g ON g.id=sg.guardian_id AND g.school_id=sg.school_id
		JOIN students s ON s.id=sg.student_id AND s.school_id=sg.school_id
		WHERE sg.school_id=$1 AND sg.student_id=$2 AND ($3=TRUE OR sg.revoked_at IS NULL)
		ORDER BY sg.revoked_at NULLS FIRST,sg.is_primary_contact DESC,LOWER(g.full_name),sg.id`,schoolID,studentID,includeHistory)
	if err!=nil{return nil,fmt.Errorf("list student guardians: %w",err)};defer rows.Close()
	items:=make([]StudentGuardian,0)
	for rows.Next(){var v StudentGuardian;if err:=rows.Scan(&v.ID,&v.StudentID,&v.GuardianID,&v.GuardianName,&v.MobileNumber,&v.EmailAddress,&v.RelationshipType,&v.IsPrimaryContact,&v.IsEmergencyContact,&v.CreatedAt,&v.RevokedAt);err!=nil{return nil,fmt.Errorf("scan student guardian: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate student guardians: %w",err)}
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
		sg.relationship_type,sg.is_primary_contact,sg.is_emergency_contact,sg.created_at,sg.revoked_at
		FROM student_guardians sg JOIN guardians g ON g.id=sg.guardian_id AND g.school_id=sg.school_id
		WHERE sg.id=$1 AND sg.school_id=$2`,v.ID,schoolID).
		Scan(&v.ID,&v.StudentID,&v.GuardianID,&v.GuardianName,&v.MobileNumber,&v.EmailAddress,&v.RelationshipType,&v.IsPrimaryContact,&v.IsEmergencyContact,&v.CreatedAt,&v.RevokedAt)
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
