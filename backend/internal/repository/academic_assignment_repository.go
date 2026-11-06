package repository

import (
	"context"
	"fmt"
	"time"
)

type TeacherSubjectAssignment struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	AcademicYearID int64 `json:"academic_year_id"`
	SectionID int64 `json:"section_id"`
	SectionName string `json:"section_name"`
	ClassName string `json:"class_name"`
	SubjectID int64 `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	TeacherPrincipalID int64 `json:"teacher_principal_id"`
	TeacherUsername string `json:"teacher_username"`
	AssignedByPrincipalID int64 `json:"assigned_by_principal_id"`
	AssignedAt time.Time `json:"assigned_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type ClassInchargeAssignment struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	AcademicYearID int64 `json:"academic_year_id"`
	SectionID int64 `json:"section_id"`
	SectionName string `json:"section_name"`
	ClassName string `json:"class_name"`
	TeacherPrincipalID int64 `json:"teacher_principal_id"`
	TeacherUsername string `json:"teacher_username"`
	AssignedByPrincipalID int64 `json:"assigned_by_principal_id"`
	AssignedAt time.Time `json:"assigned_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

func (r *AcademicRepository) ListTeacherSubjectAssignments(ctx context.Context, actorID, schoolID, yearID, sectionID int64) ([]TeacherSubjectAssignment,error) {
	if r==nil||r.db==nil{return nil,ErrAcademicAccessDenied}
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"teacher_assignments.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT a.id,a.school_id,a.academic_year_id,a.section_id,s.name,c.name,
		a.subject_id,sub.name,a.teacher_principal_id,u.username,a.assigned_by_principal_id,a.assigned_at,a.revoked_at
		FROM teacher_subject_assignments a
		JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN subjects sub ON sub.id=a.subject_id AND sub.school_id=a.school_id
		JOIN users u ON u.principal_id=a.teacher_principal_id AND u.school_id=a.school_id
		WHERE a.school_id=$1 AND ($2=0 OR a.academic_year_id=$2) AND ($3=0 OR a.section_id=$3)
		ORDER BY a.academic_year_id,s.class_id,s.display_order,a.section_id,sub.name,a.id`,schoolID,yearID,sectionID)
	if err!=nil{return nil,fmt.Errorf("list teacher subject assignments: %w",err)}
	defer rows.Close()
	items:=make([]TeacherSubjectAssignment,0)
	for rows.Next(){var v TeacherSubjectAssignment;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.SectionID,&v.SectionName,&v.ClassName,&v.SubjectID,&v.SubjectName,&v.TeacherPrincipalID,&v.TeacherUsername,&v.AssignedByPrincipalID,&v.AssignedAt,&v.RevokedAt);err!=nil{return nil,fmt.Errorf("scan teacher subject assignment: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate teacher subject assignments: %w",err)}
	return items,nil
}

func (r *AcademicRepository) CreateTeacherSubjectAssignment(ctx context.Context,actorID,schoolID,yearID,sectionID,subjectID,teacherID int64) (*TeacherSubjectAssignment,error) {
	if r==nil||r.db==nil{return nil,ErrAcademicAccessDenied}
	if yearID<=0||sectionID<=0||subjectID<=0||teacherID<=0{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin teacher subject assignment: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"teacher_assignments.manage");err!=nil{return nil,err}
	var id int64
	err=tx.QueryRow(ctx,`INSERT INTO teacher_subject_assignments(school_id,academic_year_id,section_id,subject_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,schoolID,yearID,sectionID,subjectID,teacherID,actorID).Scan(&id)
	if err!=nil{return nil,academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.teacher_subject_assignment.created','teacher_subject_assignment',$3,'success',
		jsonb_build_object('academic_year_id',$4::bigint,'section_id',$5::bigint,'subject_id',$6::bigint,'teacher_principal_id',$7::bigint))`,
		actorID,schoolID,fmt.Sprint(id),yearID,sectionID,subjectID,teacherID);err!=nil{return nil,fmt.Errorf("audit teacher subject assignment: %w",err)}
	var item TeacherSubjectAssignment
	err=tx.QueryRow(ctx,`SELECT a.id,a.school_id,a.academic_year_id,a.section_id,s.name,c.name,
		a.subject_id,sub.name,a.teacher_principal_id,u.username,a.assigned_by_principal_id,a.assigned_at,a.revoked_at
		FROM teacher_subject_assignments a
		JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN subjects sub ON sub.id=a.subject_id AND sub.school_id=a.school_id
		JOIN users u ON u.principal_id=a.teacher_principal_id AND u.school_id=a.school_id
		WHERE a.id=$1 AND a.school_id=$2`,id,schoolID).Scan(
		&item.ID,&item.SchoolID,&item.AcademicYearID,&item.SectionID,&item.SectionName,&item.ClassName,
		&item.SubjectID,&item.SubjectName,&item.TeacherPrincipalID,&item.TeacherUsername,
		&item.AssignedByPrincipalID,&item.AssignedAt,&item.RevokedAt)
	if err!=nil{return nil,fmt.Errorf("load created teacher subject assignment: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit teacher subject assignment: %w",err)}
	return &item,nil
}

func (r *AcademicRepository) RevokeTeacherSubjectAssignment(ctx context.Context,actorID,schoolID,assignmentID int64)error{
	if r==nil||r.db==nil||assignmentID<=0{return ErrAcademicAccessDenied}
	tx,err:=r.db.Begin(ctx);if err!=nil{return fmt.Errorf("begin teacher subject assignment revocation: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"teacher_assignments.manage");err!=nil{return err}
	var yearID,sectionID,subjectID,teacherID int64
	err=tx.QueryRow(ctx,`UPDATE teacher_subject_assignments SET revoked_at=NOW()
		WHERE id=$1 AND school_id=$2 AND revoked_at IS NULL
		RETURNING academic_year_id,section_id,subject_id,teacher_principal_id`,assignmentID,schoolID).Scan(&yearID,&sectionID,&subjectID,&teacherID)
	if err!=nil{return academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.teacher_subject_assignment.revoked','teacher_subject_assignment',$3,'success',
		jsonb_build_object('academic_year_id',$4::bigint,'section_id',$5::bigint,'subject_id',$6::bigint,'teacher_principal_id',$7::bigint))`,
		actorID,schoolID,fmt.Sprint(assignmentID),yearID,sectionID,subjectID,teacherID);err!=nil{return fmt.Errorf("audit teacher subject assignment revocation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit teacher subject assignment revocation: %w",err)}
	return nil
}

func (r *AcademicRepository) ListClassInchargeAssignments(ctx context.Context,actorID,schoolID,yearID int64)([]ClassInchargeAssignment,error){
	if r==nil||r.db==nil{return nil,ErrAcademicAccessDenied}
	if err:=authorizeAcademic(ctx,r.db,actorID,schoolID,"class_incharge_assignments.read");err!=nil{return nil,err}
	rows,err:=r.db.Query(ctx,`SELECT a.id,a.school_id,a.academic_year_id,a.section_id,s.name,c.name,
		a.teacher_principal_id,u.username,a.assigned_by_principal_id,a.assigned_at,a.revoked_at
		FROM class_incharge_assignments a
		JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN users u ON u.principal_id=a.teacher_principal_id AND u.school_id=a.school_id
		WHERE a.school_id=$1 AND ($2=0 OR a.academic_year_id=$2)
		ORDER BY a.academic_year_id,s.class_id,s.display_order,a.section_id,a.id`,schoolID,yearID)
	if err!=nil{return nil,fmt.Errorf("list class in-charge assignments: %w",err)}
	defer rows.Close();items:=make([]ClassInchargeAssignment,0)
	for rows.Next(){var v ClassInchargeAssignment;if err:=rows.Scan(&v.ID,&v.SchoolID,&v.AcademicYearID,&v.SectionID,&v.SectionName,&v.ClassName,&v.TeacherPrincipalID,&v.TeacherUsername,&v.AssignedByPrincipalID,&v.AssignedAt,&v.RevokedAt);err!=nil{return nil,fmt.Errorf("scan class in-charge assignment: %w",err)};items=append(items,v)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate class in-charge assignments: %w",err)}
	return items,nil
}

func (r *AcademicRepository) CreateClassInchargeAssignment(ctx context.Context,actorID,schoolID,yearID,sectionID,teacherID int64)(*ClassInchargeAssignment,error){
	if r==nil||r.db==nil{return nil,ErrAcademicAccessDenied}
	if yearID<=0||sectionID<=0||teacherID<=0{return nil,ErrInvalidAcademicInput}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin class in-charge assignment: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"class_incharge_assignments.manage");err!=nil{return nil,err}
	var id int64
	err=tx.QueryRow(ctx,`INSERT INTO class_incharge_assignments(school_id,academic_year_id,section_id,teacher_principal_id,assigned_by_principal_id)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,schoolID,yearID,sectionID,teacherID,actorID).Scan(&id)
	if err!=nil{return nil,academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.class_incharge_assignment.created','class_incharge_assignment',$3,'success',
		jsonb_build_object('academic_year_id',$4::bigint,'section_id',$5::bigint,'teacher_principal_id',$6::bigint))`,
		actorID,schoolID,fmt.Sprint(id),yearID,sectionID,teacherID);err!=nil{return nil,fmt.Errorf("audit class in-charge assignment: %w",err)}
	var item ClassInchargeAssignment
	err=tx.QueryRow(ctx,`SELECT a.id,a.school_id,a.academic_year_id,a.section_id,s.name,c.name,
		a.teacher_principal_id,u.username,a.assigned_by_principal_id,a.assigned_at,a.revoked_at
		FROM class_incharge_assignments a
		JOIN sections s ON s.id=a.section_id AND s.school_id=a.school_id
		JOIN classes c ON c.id=s.class_id AND c.school_id=s.school_id
		JOIN users u ON u.principal_id=a.teacher_principal_id AND u.school_id=a.school_id
		WHERE a.id=$1 AND a.school_id=$2`,id,schoolID).Scan(
		&item.ID,&item.SchoolID,&item.AcademicYearID,&item.SectionID,&item.SectionName,&item.ClassName,
		&item.TeacherPrincipalID,&item.TeacherUsername,&item.AssignedByPrincipalID,&item.AssignedAt,&item.RevokedAt)
	if err!=nil{return nil,fmt.Errorf("load created class in-charge assignment: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit class in-charge assignment: %w",err)}
	return &item,nil
}

func (r *AcademicRepository) RevokeClassInchargeAssignment(ctx context.Context,actorID,schoolID,assignmentID int64)error{
	if r==nil||r.db==nil||assignmentID<=0{return ErrAcademicAccessDenied}
	tx,err:=r.db.Begin(ctx);if err!=nil{return fmt.Errorf("begin class in-charge assignment revocation: %w",err)}
	defer func(){_ = tx.Rollback(ctx)}()
	if err=authorizeAcademic(ctx,tx,actorID,schoolID,"class_incharge_assignments.manage");err!=nil{return err}
	var yearID,sectionID,teacherID int64
	err=tx.QueryRow(ctx,`UPDATE class_incharge_assignments SET revoked_at=NOW()
		WHERE id=$1 AND school_id=$2 AND revoked_at IS NULL
		RETURNING academic_year_id,section_id,teacher_principal_id`,assignmentID,schoolID).Scan(&yearID,&sectionID,&teacherID)
	if err!=nil{return academicWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.class_incharge_assignment.revoked','class_incharge_assignment',$3,'success',
		jsonb_build_object('academic_year_id',$4::bigint,'section_id',$5::bigint,'teacher_principal_id',$6::bigint))`,
		actorID,schoolID,fmt.Sprint(assignmentID),yearID,sectionID,teacherID);err!=nil{return fmt.Errorf("audit class in-charge assignment revocation: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return fmt.Errorf("commit class in-charge assignment revocation: %w",err)}
	return nil
}
