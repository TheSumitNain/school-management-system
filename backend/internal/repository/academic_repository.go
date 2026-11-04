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
	ErrAcademicAccessDenied = errors.New("academic operation is not authorized")
	ErrAcademicNotFound = errors.New("academic resource not found")
	ErrAcademicConflict = errors.New("academic resource conflicts with an existing record")
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
	Name string `json:"name"`
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

func authorizeAcademic(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if db == nil || actorID <= 0 || schoolID <= 0 || permission == "" { return ErrAcademicAccessDenied }
	var allowed bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users u ON u.principal_id = ap.id
			JOIN schools s ON s.id = u.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles ro ON ro.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = ro.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff' AND ap.status = 'active'
			  AND u.status = 'active' AND u.school_id = $2 AND s.status = 'active'
			  AND ro.school_id = $2 AND ro.scope_type = 'school' AND ro.principal_type = 'staff'
			  AND p.permission_key = $3 AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
				SELECT 1 FROM user_permission_overrides o
				WHERE o.principal_id = ap.id AND o.school_id = $2
				  AND o.permission_id = p.id AND o.effect = 'deny'
			  )
		)`, actorID, schoolID, permission).Scan(&allowed)
	if err != nil { return fmt.Errorf("authorize academic operation: %w", err) }
	if !allowed { return ErrAcademicAccessDenied }
	return nil
}

func academicWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01": return ErrAcademicConflict
		case "23503", "23514", "22001", "22007": return ErrInvalidAcademicInput
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
	if name == "" || len(name)>80 || !endsOn.After(startsOn) || (status!="planned" && status!="active" && status!="closed") || (current && status!="active") { return nil,ErrInvalidAcademicInput }
	tx,err:=r.db.Begin(ctx); if err!=nil{return nil,fmt.Errorf("begin academic year save: %w",err)}; defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"academic_years.manage"); err!=nil{return nil,err}
	if id>0 {
		if _,err=tx.Exec(ctx,`UPDATE academic_years SET is_current=FALSE, updated_at=NOW() WHERE school_id=$1 AND id=$2 AND $3=TRUE`,schoolID,id,current); err!=nil{return nil,fmt.Errorf("clear current academic year: %w",err)}
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
func (r *AcademicRepository) SaveClass(ctx context.Context, actorID, schoolID, id int64, name string, order int, active bool) (*AcademicClass, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 100 || order < 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin class save: %w", err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err = authorizeAcademic(ctx, tx, actorID, schoolID, "classes.manage"); err != nil { return nil, err }
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
	rows,err:=r.db.Query(ctx,`SELECT id,school_id,academic_year_id,class_id,name,display_order,is_active,created_at,updated_at FROM sections WHERE school_id=$1 AND ($2=0 OR academic_year_id=$2) ORDER BY academic_year_id,class_id,display_order,id`,schoolID,yearID);if err!=nil{return nil,fmt.Errorf("list sections: %w",err)};defer rows.Close()
	items:=make([]AcademicSection,0);for rows.Next(){var v AcademicSection;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan section: %w",err)};items=append(items,v)};if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate sections: %w",err)};return items,nil
}
func (r *AcademicRepository) SaveSection(ctx context.Context, actorID, schoolID, id, yearID, classID int64, name string, order int, active bool) (*AcademicSection, error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 40 || order < 0 || yearID <= 0 || classID <= 0 { return nil, ErrInvalidAcademicInput }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin section save: %w",err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"classes.manage"); err!=nil { return nil,err }
	var v AcademicSection
	if id==0 {
		err=tx.QueryRow(ctx,`INSERT INTO sections(school_id,academic_year_id,class_id,name,display_order,is_active)
			VALUES($1,$2,$3,$4,$5,$6) RETURNING id,school_id,academic_year_id,class_id,name,display_order,is_active,created_at,updated_at`,
			schoolID,yearID,classID,name,order,active).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	} else {
		err=tx.QueryRow(ctx,`UPDATE sections SET name=$4,display_order=$5,is_active=$6,updated_at=NOW()
			WHERE school_id=$1 AND id=$2 AND academic_year_id=$3 AND class_id=$7
			RETURNING id,school_id,academic_year_id,class_id,name,display_order,is_active,created_at,updated_at`,
			schoolID,id,yearID,name,order,active,classID).Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.ClassID,&v.Name,&v.DisplayOrder,&v.IsActive,&v.CreatedAt,&v.UpdatedAt)
	}
	if err!=nil { return nil,academicWriteError(err) }
	action:="school.section.updated";if id==0{action="school.section.created"}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,$3,'section',$4,'success',jsonb_build_object('name',$5::text,'is_active',$6::boolean))`,
		actorID,schoolID,action,fmt.Sprint(v.ID),v.Name,v.IsActive);err!=nil{return nil,fmt.Errorf("audit section save: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit section save: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) ListSubjects(ctx context.Context,actorID,schoolID int64)([]AcademicSubject,error){
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"subjects.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT id,school_id,name,subject_code,is_active,created_at,updated_at FROM subjects WHERE school_id=$1 ORDER BY lower(name),id`,schoolID);if err!=nil{return nil,fmt.Errorf("list subjects: %w",err)};defer rows.Close()
	items:=make([]AcademicSubject,0);for rows.Next(){var v AcademicSubject;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.Name,&v.Code,&v.IsActive,&v.CreatedAt,&v.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan subject: %w",err)};items=append(items,v)};if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate subjects: %w",err)};return items,nil
}
func (r *AcademicRepository) SaveSubject(ctx context.Context, actorID, schoolID, id int64, name string, code *string, active bool) (*AcademicSubject,error) {
	if r == nil || r.db == nil { return nil, ErrAcademicAccessDenied }
	name=strings.TrimSpace(name)
	if code!=nil { value:=strings.TrimSpace(*code); if value=="" { code=nil } else if len(value)>40 { return nil,ErrInvalidAcademicInput } else { code=&value } }
	if name==""||len(name)>120{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin subject save: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"subjects.manage");err!=nil{return nil,err}
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

