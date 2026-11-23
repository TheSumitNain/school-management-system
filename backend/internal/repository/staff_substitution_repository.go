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
	ErrStaffSubstitutionAccessDenied = errors.New("staff substitution operation is not authorized")
	ErrStaffSubstitutionNotFound = errors.New("staff substitution not found")
	ErrStaffSubstitutionConflict = errors.New("staff substitution conflicts with existing coverage or timetable")
	ErrInvalidStaffSubstitution = errors.New("invalid staff substitution input")
)

type StaffSubstitution struct {
	ID int64 `json:"id"`
	TimetableEntryID int64 `json:"timetable_entry_id"`
	CoverDate string `json:"cover_date"`
	OriginalTeacherPrincipalID int64 `json:"original_teacher_principal_id"`
	OriginalTeacherName string `json:"original_teacher_name"`
	SubstituteTeacherPrincipalID int64 `json:"substitute_teacher_principal_id"`
	SubstituteTeacherName string `json:"substitute_teacher_name"`
	ApprovedLeaveRequestID *int64 `json:"approved_leave_request_id,omitempty"`
	Reason string `json:"reason"`
	Status string `json:"status"`
	SectionName string `json:"section_name"`
	ClassName string `json:"class_name"`
	SubjectName string `json:"subject_name"`
	PeriodNumber int `json:"period_number"`
	PeriodName string `json:"period_name"`
	StartsAt string `json:"starts_at"`
	EndsAt string `json:"ends_at"`
	Room *string `json:"room,omitempty"`
	CreatedByPrincipalID int64 `json:"created_by_principal_id"`
	CreatedAt time.Time `json:"created_at"`
	CancelledByPrincipalID *int64 `json:"cancelled_by_principal_id,omitempty"`
	CancelledAt *time.Time `json:"cancelled_at,omitempty"`
}

func substitutionWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err,&pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrStaffSubstitutionConflict
		case "23503","23514","22001","22007","P0001":
			return ErrInvalidStaffSubstitution
		}
	}
	if errors.Is(err,pgx.ErrNoRows) { return ErrStaffSubstitutionNotFound }
	return err
}

const staffSubstitutionSelect = `ss.id,ss.timetable_entry_id,ss.cover_date::text,
	ss.original_teacher_principal_id,ss.original_teacher_name_snapshot,
	ss.substitute_teacher_principal_id,ss.substitute_teacher_name_snapshot,
	ss.approved_leave_request_id,ss.reason,ss.status,
	ss.section_name_snapshot,ss.class_name_snapshot,ss.subject_name_snapshot,
	ss.period_number_snapshot,ss.period_name_snapshot,
	to_char(ss.period_starts_at_snapshot,'HH24:MI'),to_char(ss.period_ends_at_snapshot,'HH24:MI'),
	ss.room_snapshot,ss.created_by_principal_id,ss.created_at,ss.cancelled_by_principal_id,ss.cancelled_at`

func staffSubstitutionAuthorize(ctx context.Context,db repositoryDB,actorID,schoolID int64,permission string)error {
	err:=timetableAuthorize(ctx,db,actorID,schoolID,permission)
	if errors.Is(err,ErrTimetableAccessDenied){return ErrStaffSubstitutionAccessDenied}
	return err
}

func validateStaffSubstitutionDateRange(fromDate, toDate string) error {
	if fromDate == "" || toDate == "" {
		return ErrInvalidStaffSubstitution
	}
	from, err := time.Parse("2006-01-02", fromDate)
	if err != nil || from.Year() < 1 || from.Format("2006-01-02") != fromDate {
		return ErrInvalidStaffSubstitution
	}
	to, err := time.Parse("2006-01-02", toDate)
	if err != nil || to.Year() < 1 || to.Format("2006-01-02") != toDate || to.Before(from) {
		return ErrInvalidStaffSubstitution
	}
	return nil
}

func (r *AcademicRepository) ListStaffSubstitutions(ctx context.Context,actorID,schoolID int64,fromDate,toDate string,staffID int64)([]StaffSubstitution,error) {
	if r==nil||r.db==nil||actorID<=0||schoolID<=0||staffID<0 { return nil,ErrInvalidStaffSubstitution }
	if err:=staffSubstitutionAuthorize(ctx,r.db,actorID,schoolID,"staff.substitutions.read");err!=nil { return nil,err }
	if err:=validateStaffSubstitutionDateRange(fromDate,toDate);err!=nil { return nil,err }
	rows,err:=r.db.Query(ctx,`SELECT `+staffSubstitutionSelect+`
		FROM staff_substitutions ss
		WHERE ss.school_id=$1 AND ss.cover_date BETWEEN $2::date AND $3::date
		  AND ($4=0 OR ss.original_teacher_principal_id=$4 OR ss.substitute_teacher_principal_id=$4)
		ORDER BY ss.cover_date,ss.period_number_snapshot,ss.id LIMIT 1000`,schoolID,fromDate,toDate,staffID)
	if err!=nil{return nil,fmt.Errorf("list staff substitutions: %w",err)}
	defer rows.Close()
	items:=make([]StaffSubstitution,0)
	for rows.Next(){
		var v StaffSubstitution
		if err:=rows.Scan(&v.ID,&v.TimetableEntryID,&v.CoverDate,&v.OriginalTeacherPrincipalID,&v.OriginalTeacherName,
			&v.SubstituteTeacherPrincipalID,&v.SubstituteTeacherName,&v.ApprovedLeaveRequestID,&v.Reason,&v.Status,
			&v.SectionName,&v.ClassName,&v.SubjectName,&v.PeriodNumber,&v.PeriodName,&v.StartsAt,&v.EndsAt,
			&v.Room,&v.CreatedByPrincipalID,&v.CreatedAt,&v.CancelledByPrincipalID,&v.CancelledAt);err!=nil{return nil,fmt.Errorf("scan staff substitution: %w",err)}
		items=append(items,v)
	}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate staff substitutions: %w",err)}
	return items,nil
}

func (r *AcademicRepository) CreateStaffSubstitution(ctx context.Context,actorID,schoolID,timetableEntryID,originalTeacherID,substituteTeacherID int64,coverDate string,leaveRequestID *int64,reason string)(*StaffSubstitution,error) {
	if r==nil||r.db==nil||actorID<=0||schoolID<=0{return nil,ErrInvalidStaffSubstitution}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin staff substitution: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err:=staffSubstitutionAuthorize(ctx,tx,actorID,schoolID,"staff.substitutions.manage");err!=nil{return nil,err}
	if timetableEntryID<=0||originalTeacherID<=0||substituteTeacherID<=0||originalTeacherID==substituteTeacherID{return nil,ErrInvalidStaffSubstitution}
	reason=strings.TrimSpace(reason)
	if len([]rune(reason))==0||len([]rune(reason))>1000{return nil,ErrInvalidStaffSubstitution}
	parsedCoverDate,err:=time.Parse("2006-01-02",coverDate)
	if err!=nil||parsedCoverDate.Year()<1||parsedCoverDate.Format("2006-01-02")!=coverDate{return nil,ErrInvalidStaffSubstitution}
	var periodID int64
	err=tx.QueryRow(ctx,`SELECT period_id FROM timetable_entries
		WHERE id=$1 AND school_id=$2 AND teacher_principal_id=$3`,timetableEntryID,schoolID,originalTeacherID).Scan(&periodID)
	if err!=nil{return nil,substitutionWriteError(err)}
	var leave any
	if leaveRequestID!=nil { if *leaveRequestID<=0{return nil,ErrInvalidStaffSubstitution};leave=*leaveRequestID }
	var id int64
	err=tx.QueryRow(ctx,`INSERT INTO staff_substitutions
		(school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,
		 substitute_teacher_principal_id,approved_leave_request_id,reason,created_by_principal_id)
		VALUES($1,$2,$3,$4::date,$5,$6,$7,$8,$9) RETURNING id`,
		schoolID,timetableEntryID,periodID,coverDate,originalTeacherID,substituteTeacherID,leave,reason,actorID).Scan(&id)
	if err!=nil{return nil,substitutionWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'staff.substitution.assigned','staff_substitution',$3,'success',
		jsonb_build_object('timetable_entry_id',$4::bigint,'cover_date',$5::date,
		'original_teacher_principal_id',$6::bigint,'substitute_teacher_principal_id',$7::bigint,
		'approved_leave_request_id',$8::bigint))`,actorID,schoolID,fmt.Sprint(id),timetableEntryID,coverDate,originalTeacherID,substituteTeacherID,leave);err!=nil{return nil,fmt.Errorf("audit staff substitution: %w",err)}
	var v StaffSubstitution
	err=tx.QueryRow(ctx,`SELECT `+staffSubstitutionSelect+` FROM staff_substitutions ss WHERE ss.school_id=$1 AND ss.id=$2`,schoolID,id).Scan(
		&v.ID,&v.TimetableEntryID,&v.CoverDate,&v.OriginalTeacherPrincipalID,&v.OriginalTeacherName,
		&v.SubstituteTeacherPrincipalID,&v.SubstituteTeacherName,&v.ApprovedLeaveRequestID,&v.Reason,&v.Status,
		&v.SectionName,&v.ClassName,&v.SubjectName,&v.PeriodNumber,&v.PeriodName,&v.StartsAt,&v.EndsAt,
		&v.Room,&v.CreatedByPrincipalID,&v.CreatedAt,&v.CancelledByPrincipalID,&v.CancelledAt)
	if err!=nil{return nil,fmt.Errorf("read created substitution: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit staff substitution: %w",err)}
	return &v,nil
}

func (r *AcademicRepository) CancelStaffSubstitution(ctx context.Context,actorID,schoolID,id int64)error {
	if r==nil||r.db==nil||actorID<=0||schoolID<=0||id<=0{return ErrInvalidStaffSubstitution}
	tx,err:=r.db.Begin(ctx);if err!=nil{return fmt.Errorf("begin substitution cancellation: %w",err)}
	defer func(){_=tx.Rollback(ctx)}()
	if err:=staffSubstitutionAuthorize(ctx,tx,actorID,schoolID,"staff.substitutions.manage");err!=nil{return err}
	tag,err:=tx.Exec(ctx,`UPDATE staff_substitutions SET status='cancelled',cancelled_by_principal_id=$3,cancelled_at=NOW(),updated_at=NOW()
		WHERE id=$1 AND school_id=$2 AND status='active'`,id,schoolID,actorID)
	if err!=nil{return substitutionWriteError(err)}
	if tag.RowsAffected()!=1{return ErrStaffSubstitutionNotFound}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome)
		VALUES($1,$2,'staff.substitution.cancelled','staff_substitution',$3,'success')`,actorID,schoolID,fmt.Sprint(id));err!=nil{return fmt.Errorf("audit substitution cancellation: %w",err)}
	return tx.Commit(ctx)
}
