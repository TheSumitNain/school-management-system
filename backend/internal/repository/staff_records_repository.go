package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrStaffRecordAccessDenied = errors.New("staff record access denied")
	ErrInvalidStaffRecord = errors.New("invalid staff record")
)

type StaffQualification struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	QualificationType string `json:"qualification_type"`
	QualificationName string `json:"qualification_name"`
	Specialization string `json:"specialization,omitempty"`
	Institution string `json:"institution"`
	BoardUniversity string `json:"board_university,omitempty"`
	CompletionYear *int `json:"completion_year,omitempty"`
	Score *float64 `json:"score,omitempty"`
	ScoreScale string `json:"score_scale,omitempty"`
	CredentialReference string `json:"credential_reference,omitempty"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StaffCertification struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	CertificateName string `json:"certificate_name"`
	IssuingOrganization string `json:"issuing_organization"`
	IssueDate string `json:"issue_date,omitempty"`
	ExpiryDate string `json:"expiry_date,omitempty"`
	CredentialID string `json:"credential_id,omitempty"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StaffExperience struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	EmployerName string `json:"employer_name"`
	Designation string `json:"designation"`
	EmploymentType string `json:"employment_type"`
	StartDate string `json:"start_date"`
	EndDate string `json:"end_date,omitempty"`
	IsCurrent bool `json:"is_current"`
	Description string `json:"description,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func authorizeStaffRecord(ctx context.Context, db repositoryDB, actorID, schoolID, staffID int64, permission string) error {
	if actorID <= 0 || schoolID <= 0 || staffID <= 0 {
		return ErrStaffRecordAccessDenied
	}
	var allowed bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM auth_principals ap
			JOIN users actor ON actor.principal_id = ap.id
			JOIN schools s ON s.id = actor.school_id
			JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
			JOIN roles r ON r.id = pr.role_id
			JOIN role_permissions rp ON rp.role_id = r.id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE ap.id = $1 AND ap.principal_type = 'staff' AND ap.status = 'active'
			  AND actor.status = 'active' AND actor.school_id = $2 AND s.status = 'active'
			  AND r.school_id = $2 AND r.scope_type = 'school' AND r.principal_type = 'staff'
			  AND r.role_key = 'school_admin' AND r.is_system = TRUE
			  AND p.permission_key = $4 AND p.scope_type = 'school' AND p.is_active = TRUE
			  AND NOT EXISTS (
			      SELECT 1 FROM user_permission_overrides denied
			      WHERE denied.principal_id = ap.id AND denied.school_id = $2
			        AND denied.permission_id = p.id AND denied.effect = 'deny'
			  )
			  AND EXISTS (
			      SELECT 1 FROM staff_profiles target
			      JOIN users target_user ON target_user.principal_id = target.principal_id
			                                AND target_user.school_id = target.school_id
			      JOIN auth_principals target_principal ON target_principal.id = target.principal_id
			      WHERE target.principal_id = $3 AND target.school_id = $2
			        AND target_principal.principal_type = 'staff'
			  )
		)
	`, actorID, schoolID, staffID, permission).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("authorize staff profile records: %w", err)
	}
	if !allowed {
		return ErrStaffRecordAccessDenied
	}
	return nil
}

func (r *UserRepository) ListStaffQualifications(ctx context.Context, actorID, schoolID, staffID int64) ([]StaffQualification, error) {
	if r == nil || r.db == nil { return nil, ErrStaffRecordAccessDenied }
	if err := authorizeStaffRecord(ctx, r.db, actorID, schoolID, staffID, "users.read"); err != nil { return nil, err }
	rows, err := r.db.Query(ctx, `
		SELECT id, staff_principal_id, qualification_type, qualification_name,
		       COALESCE(specialization,''), institution, COALESCE(board_university,''),
		       completion_year, score::float8, COALESCE(score_scale,''),
		       COALESCE(credential_reference,''), verification_status, created_at, updated_at
		FROM staff_qualifications
		WHERE school_id=$1 AND staff_principal_id=$2
		ORDER BY completion_year DESC NULLS LAST, id DESC
	`, schoolID, staffID)
	if err != nil { return nil, fmt.Errorf("list staff qualifications: %w", err) }
	defer rows.Close()
	items := make([]StaffQualification, 0)
	for rows.Next() {
		var item StaffQualification
		if err := rows.Scan(&item.ID, &item.StaffPrincipalID, &item.QualificationType, &item.QualificationName,
			&item.Specialization, &item.Institution, &item.BoardUniversity, &item.CompletionYear,
			&item.Score, &item.ScoreScale, &item.CredentialReference, &item.VerificationStatus,
			&item.CreatedAt, &item.UpdatedAt); err != nil { return nil, fmt.Errorf("scan staff qualification: %w", err) }
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate staff qualifications: %w", err) }
	return items, nil
}

func (r *UserRepository) CreateStaffQualification(ctx context.Context, actorID, schoolID, staffID int64, item StaffQualification) (*StaffQualification, error) {
	if r == nil || r.db == nil { return nil, ErrStaffRecordAccessDenied }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin staff qualification creation: %w", err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err := authorizeStaffRecord(ctx, tx, actorID, schoolID, staffID, "users.update"); err != nil { return nil, err }
	var saved StaffQualification
	err = tx.QueryRow(ctx, `
		INSERT INTO staff_qualifications
		(school_id,staff_principal_id,qualification_type,qualification_name,specialization,institution,board_university,completion_year,score,score_scale,credential_reference)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,NULLIF($7,''),$8,$9,NULLIF($10,''),NULLIF($11,''))
		RETURNING id,staff_principal_id,qualification_type,qualification_name,COALESCE(specialization,''),institution,
		          COALESCE(board_university,''),completion_year,score::float8,COALESCE(score_scale,''),
		          COALESCE(credential_reference,''),verification_status,created_at,updated_at
	`, schoolID,staffID,item.QualificationType,item.QualificationName,item.Specialization,item.Institution,
		item.BoardUniversity,item.CompletionYear,item.Score,item.ScoreScale,item.CredentialReference).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.QualificationType,&saved.QualificationName,&saved.Specialization,&saved.Institution,
		&saved.BoardUniversity,&saved.CompletionYear,&saved.Score,&saved.ScoreScale,&saved.CredentialReference,
		&saved.VerificationStatus,&saved.CreatedAt,&saved.UpdatedAt)
	if err != nil { return nil, fmt.Errorf("insert staff qualification: %w", err) }
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.qualification.created','staff_qualification',$3,'success',jsonb_build_object('record_type','qualification','staff_principal_id',$4::bigint))`,
		actorID,schoolID,fmt.Sprint(saved.ID),staffID); err != nil { return nil,fmt.Errorf("audit staff qualification creation: %w",err) }
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff qualification creation: %w",err)}
	return &saved,nil
}

func (r *UserRepository) ListStaffCertifications(ctx context.Context, actorID, schoolID, staffID int64) ([]StaffCertification, error) {
	if r == nil || r.db == nil { return nil, ErrStaffRecordAccessDenied }
	if err := authorizeStaffRecord(ctx,r.db,actorID,schoolID,staffID,"users.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`
		SELECT id,staff_principal_id,certificate_name,issuing_organization,
		       COALESCE(issue_date::text,''),COALESCE(expiry_date::text,''),COALESCE(credential_id,''),
		       verification_status,created_at,updated_at
		FROM staff_certifications WHERE school_id=$1 AND staff_principal_id=$2
		ORDER BY issue_date DESC NULLS LAST,id DESC
	`,schoolID,staffID)
	if err!=nil{return nil,fmt.Errorf("list staff certifications: %w",err)}
	defer rows.Close()
	items:=make([]StaffCertification,0)
	for rows.Next(){var item StaffCertification
		if err:=rows.Scan(&item.ID,&item.StaffPrincipalID,&item.CertificateName,&item.IssuingOrganization,&item.IssueDate,&item.ExpiryDate,&item.CredentialID,&item.VerificationStatus,&item.CreatedAt,&item.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan staff certification: %w",err)}
		items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate staff certifications: %w",err)}
	return items,nil
}

func (r *UserRepository) CreateStaffCertification(ctx context.Context, actorID, schoolID, staffID int64, item StaffCertification) (*StaffCertification,error) {
	if r==nil||r.db==nil{return nil,ErrStaffRecordAccessDenied}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin staff certification creation: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err:=authorizeStaffRecord(ctx,tx,actorID,schoolID,staffID,"users.update");err!=nil{return nil,err}
	var saved StaffCertification
	err=tx.QueryRow(ctx,`
		INSERT INTO staff_certifications(school_id,staff_principal_id,certificate_name,issuing_organization,issue_date,expiry_date,credential_id)
		VALUES($1,$2,$3,$4,NULLIF($5,'')::date,NULLIF($6,'')::date,NULLIF($7,''))
		RETURNING id,staff_principal_id,certificate_name,issuing_organization,COALESCE(issue_date::text,''),COALESCE(expiry_date::text,''),
		          COALESCE(credential_id,''),verification_status,created_at,updated_at
	`,schoolID,staffID,item.CertificateName,item.IssuingOrganization,item.IssueDate,item.ExpiryDate,item.CredentialID).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.CertificateName,&saved.IssuingOrganization,&saved.IssueDate,&saved.ExpiryDate,&saved.CredentialID,&saved.VerificationStatus,&saved.CreatedAt,&saved.UpdatedAt)
	if err!=nil{return nil,fmt.Errorf("insert staff certification: %w",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.certification.created','staff_certification',$3,'success',jsonb_build_object('record_type','certification','staff_principal_id',$4::bigint))`,actorID,schoolID,fmt.Sprint(saved.ID),staffID);err!=nil{return nil,fmt.Errorf("audit staff certification creation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff certification creation: %w",err)}
	return &saved,nil
}

func (r *UserRepository) ListStaffExperience(ctx context.Context, actorID, schoolID, staffID int64) ([]StaffExperience,error) {
	if r==nil||r.db==nil{return nil,ErrStaffRecordAccessDenied}
	if err:=authorizeStaffRecord(ctx,r.db,actorID,schoolID,staffID,"users.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`
		SELECT id,staff_principal_id,employer_name,designation,employment_type,start_date::text,
		       COALESCE(end_date::text,''),is_current,COALESCE(description,''),created_at,updated_at
		FROM staff_experience WHERE school_id=$1 AND staff_principal_id=$2
		ORDER BY start_date DESC,id DESC
	`,schoolID,staffID)
	if err!=nil{return nil,fmt.Errorf("list staff experience: %w",err)}
	defer rows.Close()
	items:=make([]StaffExperience,0)
	for rows.Next(){var item StaffExperience
		if err:=rows.Scan(&item.ID,&item.StaffPrincipalID,&item.EmployerName,&item.Designation,&item.EmploymentType,&item.StartDate,&item.EndDate,&item.IsCurrent,&item.Description,&item.CreatedAt,&item.UpdatedAt);err!=nil{return nil,fmt.Errorf("scan staff experience: %w",err)}
		items=append(items,item)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate staff experience: %w",err)}
	return items,nil
}

func (r *UserRepository) CreateStaffExperience(ctx context.Context, actorID, schoolID, staffID int64, item StaffExperience)(*StaffExperience,error){
	if r==nil||r.db==nil{return nil,ErrStaffRecordAccessDenied}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin staff experience creation: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err:=authorizeStaffRecord(ctx,tx,actorID,schoolID,staffID,"users.update");err!=nil{return nil,err}
	var saved StaffExperience
	err=tx.QueryRow(ctx,`
		INSERT INTO staff_experience(school_id,staff_principal_id,employer_name,designation,employment_type,start_date,end_date,is_current,description)
		VALUES($1,$2,$3,$4,$5,$6::date,NULLIF($7,'')::date,$8,NULLIF($9,''))
		RETURNING id,staff_principal_id,employer_name,designation,employment_type,start_date::text,COALESCE(end_date::text,''),
		          is_current,COALESCE(description,''),created_at,updated_at
	`,schoolID,staffID,item.EmployerName,item.Designation,item.EmploymentType,item.StartDate,item.EndDate,item.IsCurrent,item.Description).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.EmployerName,&saved.Designation,&saved.EmploymentType,&saved.StartDate,&saved.EndDate,&saved.IsCurrent,&saved.Description,&saved.CreatedAt,&saved.UpdatedAt)
	if err!=nil{return nil,fmt.Errorf("insert staff experience: %w",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.experience.created','staff_experience',$3,'success',jsonb_build_object('record_type','experience','staff_principal_id',$4::bigint))`,actorID,schoolID,fmt.Sprint(saved.ID),staffID);err!=nil{return nil,fmt.Errorf("audit staff experience creation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff experience creation: %w",err)}
	return &saved,nil
}

var _ = pgx.ErrNoRows


// Update methods preserve verification state and change only editable staff-supplied fields.
// The record ID, staff principal ID and school ID are always taken from the route/auth context.
func (r *UserRepository) UpdateStaffQualification(ctx context.Context, actorID, schoolID, staffID, recordID int64, item StaffQualification) (*StaffQualification, error) {
	if r == nil || r.db == nil || recordID <= 0 { return nil, ErrStaffRecordAccessDenied }
	tx, err := r.db.Begin(ctx); if err != nil { return nil, fmt.Errorf("begin staff qualification update: %w", err) }
	defer func(){ _ = tx.Rollback(ctx) }()
	if err = authorizeStaffRecord(ctx, tx, actorID, schoolID, staffID, "users.update"); err != nil { return nil, err }
	var saved StaffQualification
	err = tx.QueryRow(ctx, `
		UPDATE staff_qualifications SET qualification_type=$4, qualification_name=$5, specialization=NULLIF($6,''), institution=$7,
		 board_university=NULLIF($8,''), completion_year=$9, score=$10, score_scale=NULLIF($11,''), credential_reference=NULLIF($12,''), updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND staff_principal_id=$3
		RETURNING id,staff_principal_id,qualification_type,qualification_name,COALESCE(specialization,''),institution,
		 COALESCE(board_university,''),completion_year,score::float8,COALESCE(score_scale,''),COALESCE(credential_reference,''),
		 verification_status,created_at,updated_at
	`, recordID,schoolID,staffID,item.QualificationType,item.QualificationName,item.Specialization,item.Institution,
		item.BoardUniversity,item.CompletionYear,item.Score,item.ScoreScale,item.CredentialReference).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.QualificationType,&saved.QualificationName,&saved.Specialization,&saved.Institution,
		&saved.BoardUniversity,&saved.CompletionYear,&saved.Score,&saved.ScoreScale,&saved.CredentialReference,&saved.VerificationStatus,&saved.CreatedAt,&saved.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) { return nil, ErrInvalidStaffRecord }
	if err != nil { return nil, fmt.Errorf("update staff qualification: %w", err) }
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.qualification.updated','staff_qualification',$3,'success',jsonb_build_object('record_type','qualification','staff_principal_id',$4::bigint))`,actorID,schoolID,fmt.Sprint(recordID),staffID);err!=nil{return nil,fmt.Errorf("audit staff qualification update: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff qualification update: %w",err)}
	return &saved,nil
}

func (r *UserRepository) UpdateStaffCertification(ctx context.Context, actorID, schoolID, staffID, recordID int64, item StaffCertification) (*StaffCertification, error) {
	if r == nil || r.db == nil || recordID <= 0 { return nil, ErrStaffRecordAccessDenied }
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin staff certification update: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeStaffRecord(ctx,tx,actorID,schoolID,staffID,"users.update");err!=nil{return nil,err}
	var saved StaffCertification
	err=tx.QueryRow(ctx,`
		UPDATE staff_certifications SET certificate_name=$4,issuing_organization=$5,issue_date=NULLIF($6,'')::date,
		 expiry_date=NULLIF($7,'')::date,credential_id=NULLIF($8,''),updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND staff_principal_id=$3
		RETURNING id,staff_principal_id,certificate_name,issuing_organization,COALESCE(issue_date::text,''),COALESCE(expiry_date::text,''),
		 COALESCE(credential_id,''),verification_status,created_at,updated_at
	`,recordID,schoolID,staffID,item.CertificateName,item.IssuingOrganization,item.IssueDate,item.ExpiryDate,item.CredentialID).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.CertificateName,&saved.IssuingOrganization,&saved.IssueDate,&saved.ExpiryDate,&saved.CredentialID,&saved.VerificationStatus,&saved.CreatedAt,&saved.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows){return nil,ErrInvalidStaffRecord}
	if err!=nil{return nil,fmt.Errorf("update staff certification: %w",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.certification.updated','staff_certification',$3,'success',jsonb_build_object('record_type','certification','staff_principal_id',$4::bigint))`,actorID,schoolID,fmt.Sprint(recordID),staffID);err!=nil{return nil,fmt.Errorf("audit staff certification update: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff certification update: %w",err)}
	return &saved,nil
}

func (r *UserRepository) UpdateStaffExperience(ctx context.Context, actorID, schoolID, staffID, recordID int64, item StaffExperience) (*StaffExperience, error) {
	if r==nil||r.db==nil||recordID<=0{return nil,ErrStaffRecordAccessDenied}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin staff experience update: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err=authorizeStaffRecord(ctx,tx,actorID,schoolID,staffID,"users.update");err!=nil{return nil,err}
	var saved StaffExperience
	err=tx.QueryRow(ctx,`
		UPDATE staff_experience SET employer_name=$4,designation=$5,employment_type=$6,start_date=$7::date,
		 end_date=NULLIF($8,'')::date,is_current=$9,description=NULLIF($10,''),updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND staff_principal_id=$3
		RETURNING id,staff_principal_id,employer_name,designation,employment_type,start_date::text,COALESCE(end_date::text,''),
		 is_current,COALESCE(description,''),created_at,updated_at
	`,recordID,schoolID,staffID,item.EmployerName,item.Designation,item.EmploymentType,item.StartDate,item.EndDate,item.IsCurrent,item.Description).Scan(
		&saved.ID,&saved.StaffPrincipalID,&saved.EmployerName,&saved.Designation,&saved.EmploymentType,&saved.StartDate,&saved.EndDate,&saved.IsCurrent,&saved.Description,&saved.CreatedAt,&saved.UpdatedAt)
	if errors.Is(err,pgx.ErrNoRows){return nil,ErrInvalidStaffRecord}
	if err!=nil{return nil,fmt.Errorf("update staff experience: %w",err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.experience.updated','staff_experience',$3,'success',jsonb_build_object('record_type','experience','staff_principal_id',$4::bigint))`,actorID,schoolID,fmt.Sprint(recordID),staffID);err!=nil{return nil,fmt.Errorf("audit staff experience update: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff experience update: %w",err)}
	return &saved,nil
}

type StaffAuditEvent struct {
	ID int64 `json:"id"`
	Action string `json:"action"`
	ResourceType string `json:"resource_type"`
	Outcome string `json:"outcome"`
	ActorUsername string `json:"actor_username"`
	Metadata json.RawMessage `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *UserRepository) ListStaffAuditEvents(ctx context.Context, actorID, schoolID, staffID int64) ([]StaffAuditEvent, error) {
	if r == nil || r.db == nil {
		return nil, ErrStaffRecordAccessDenied
	}
	if err := authorizeStaffRecord(ctx, r.db, actorID, schoolID, staffID, "school.audit.read"); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT ae.id, ae.action, ae.resource_type, ae.outcome,
		       COALESCE(actor.username, 'System'), ae.metadata, ae.created_at
		FROM audit_events ae
		LEFT JOIN users actor
		  ON actor.principal_id = ae.actor_principal_id
		 AND actor.school_id = ae.school_id
		LEFT JOIN users target_user
		  ON target_user.school_id = ae.school_id
		 AND target_user.id::text = ae.resource_id
		 AND target_user.principal_id = $2
		WHERE ae.school_id = $1
		  AND (
		       (ae.resource_type = 'staff_principal' AND ae.resource_id = $2::text)
		    OR (ae.resource_type = 'staff_user' AND (
		           ae.metadata->>'principal_id' = $2::text
		        OR ae.metadata->>'target_principal_id' = $2::text
		        OR target_user.principal_id = $2
		    ))
		    OR (ae.resource_type IN ('teacher_subject_assignment','class_incharge_assignment')
		        AND ae.metadata->>'teacher_principal_id' = $2::text)
		    OR (ae.resource_type IN ('staff_qualification','staff_certification','staff_experience')
		        AND ae.metadata->>'staff_principal_id' = $2::text)
		  )
		ORDER BY ae.created_at DESC, ae.id DESC
		LIMIT 100
	`, schoolID, staffID)
	if err != nil {
		return nil, fmt.Errorf("list staff audit events: %w", err)
	}
	defer rows.Close()
	items := make([]StaffAuditEvent, 0)
	for rows.Next() {
		var item StaffAuditEvent
		if err := rows.Scan(&item.ID, &item.Action, &item.ResourceType, &item.Outcome, &item.ActorUsername, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan staff audit event: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staff audit events: %w", err)
	}
	return items, nil
}
