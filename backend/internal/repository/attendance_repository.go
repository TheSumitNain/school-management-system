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
	ErrAttendanceAccessDenied = errors.New("attendance operation is not authorized")
	ErrAttendanceNotFound = errors.New("attendance record not found")
	ErrAttendanceConflict = errors.New("attendance already exists for this enrollment and date")
	ErrInvalidAttendanceInput = errors.New("invalid attendance data")
)

type AttendanceRecord struct {
	ID int64 `json:"id"`
	SchoolID int64 `json:"school_id"`
	EnrollmentID int64 `json:"enrollment_id"`
	StudentID int64 `json:"student_id"`
	AdmissionNumber string `json:"admission_number"`
	StudentName string `json:"student_name"`
	AcademicYearID int64 `json:"academic_year_id"`
	ClassName string `json:"class_name"`
	SectionName string `json:"section_name"`
	RollNumber *string `json:"roll_number,omitempty"`
	AttendanceDate time.Time `json:"attendance_date"`
	Status string `json:"status"`
	Remarks *string `json:"remarks,omitempty"`
	MarkedByPrincipalID int64 `json:"marked_by_principal_id"`
	UpdatedByPrincipalID int64 `json:"updated_by_principal_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const attendanceColumns = `a.id,a.school_id,a.enrollment_id,e.student_id,st.admission_number,st.full_name,
	e.academic_year_id,e.class_name_snapshot,e.section_name_snapshot,e.roll_number,
	a.attendance_date,a.status,a.remarks,a.marked_by_principal_id,a.updated_by_principal_id,a.created_at,a.updated_at`

func scanAttendance(row interface{ Scan(...any) error }) (*AttendanceRecord,error) {
	var v AttendanceRecord
	err:=row.Scan(&v.ID,&v.SchoolID,&v.EnrollmentID,&v.StudentID,&v.AdmissionNumber,&v.StudentName,
		&v.AcademicYearID,&v.ClassName,&v.SectionName,&v.RollNumber,&v.AttendanceDate,&v.Status,
		&v.Remarks,&v.MarkedByPrincipalID,&v.UpdatedByPrincipalID,&v.CreatedAt,&v.UpdatedAt)
	if err!=nil{return nil,err};return &v,nil
}

func attendancePermission(ctx context.Context, db repositoryDB, actorID, schoolID int64, permission string) error {
	if err:=authorizeGuardian(ctx,db,actorID,schoolID,permission);err!=nil{return ErrAttendanceAccessDenied}
	return nil
}

func attendanceSectionAllowed(ctx context.Context, db repositoryDB, actorID, schoolID, sectionID int64) (bool,error) {
	var allowed bool
	err:=db.QueryRow(ctx,`SELECT EXISTS (
		SELECT 1 FROM class_incharge_assignments a
		JOIN auth_principals ap ON ap.id=a.teacher_principal_id
		JOIN users u ON u.principal_id=ap.id AND u.school_id=a.school_id
		WHERE a.school_id=$1 AND a.section_id=$2 AND a.teacher_principal_id=$3
		  AND a.revoked_at IS NULL AND ap.status='active' AND u.status='active'
		  AND EXISTS (
			SELECT 1 FROM principal_roles tpr
			JOIN roles tr ON tr.id=tpr.role_id
			WHERE tpr.principal_id=ap.id AND tpr.revoked_at IS NULL
			  AND tr.school_id=a.school_id AND tr.scope_type='school'
			  AND tr.principal_type='staff' AND tr.role_key='teacher'
			  AND tr.is_system=TRUE
		  )
	)`,schoolID,sectionID,actorID).Scan(&allowed)
	if err!=nil{return false,fmt.Errorf("check attendance section assignment: %w",err)}
	if allowed{return true,nil}
	err=db.QueryRow(ctx,`SELECT EXISTS (
		SELECT 1 FROM principal_roles pr JOIN roles r ON r.id=pr.role_id
		WHERE pr.principal_id=$1 AND pr.revoked_at IS NULL
		  AND r.school_id=$2 AND r.scope_type='school' AND r.principal_type='staff'
		  AND r.role_key='school_admin' AND r.is_system=TRUE
	)`,actorID,schoolID).Scan(&allowed)
	if err!=nil{return false,fmt.Errorf("check attendance administrator role: %w",err)}
	return allowed,nil
}

func attendanceWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err,&pgErr) {
		switch pgErr.Code {
		case "23505": return ErrAttendanceConflict
		case "23503","23514","22001","22007","P0001": return ErrInvalidAttendanceInput
		}
	}
	if errors.Is(err,pgx.ErrNoRows){return ErrAttendanceNotFound}
	return err
}

func (r *AcademicRepository) ListAttendance(ctx context.Context,actorID,schoolID,yearID,sectionID int64,day time.Time)([]AttendanceRecord,error) {
	if r==nil||r.db==nil{return nil,ErrAttendanceAccessDenied}
	if yearID<0||sectionID<0{return nil,ErrInvalidAttendanceInput}
	if err:=attendancePermission(ctx,r.db,actorID,schoolID,"attendance.read");err!=nil{return nil,err}
	if sectionID>0 {
		allowed,err:=attendanceSectionAllowed(ctx,r.db,actorID,schoolID,sectionID);if err!=nil{return nil,err};if !allowed{return nil,ErrAttendanceAccessDenied}
	}
	rows,err:=r.db.Query(ctx,`SELECT `+attendanceColumns+`
		FROM attendance_records a
		JOIN student_enrollments e ON e.id=a.enrollment_id AND e.school_id=a.school_id
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		JOIN sections sec ON sec.id=e.section_id AND sec.school_id=e.school_id
		WHERE a.school_id=$1 AND ($2=0 OR e.academic_year_id=$2)
		  AND ($3=0 OR e.section_id=$3) AND ($4::date IS NULL OR a.attendance_date=$4)
		  AND (EXISTS (SELECT 1 FROM principal_roles pr JOIN roles ro ON ro.id=pr.role_id
		       WHERE pr.principal_id=$5 AND pr.revoked_at IS NULL AND ro.school_id=$1
		         AND ro.scope_type='school' AND ro.role_key='school_admin' AND ro.is_system=TRUE)
		       OR EXISTS (SELECT 1 FROM class_incharge_assignments ci
		       JOIN auth_principals cap ON cap.id=ci.teacher_principal_id
		       JOIN users cu ON cu.principal_id=cap.id AND cu.school_id=ci.school_id
		       WHERE ci.school_id=$1 AND ci.section_id=e.section_id
		         AND ci.teacher_principal_id=$5 AND ci.revoked_at IS NULL
		         AND cap.status='active' AND cu.status='active'
		         AND EXISTS (
				SELECT 1 FROM principal_roles tpr
				JOIN roles tr ON tr.id=tpr.role_id
				WHERE tpr.principal_id=cap.id AND tpr.revoked_at IS NULL
				  AND tr.school_id=ci.school_id AND tr.scope_type='school'
				  AND tr.principal_type='staff' AND tr.role_key='teacher'
				  AND tr.is_system=TRUE
			 )))
		ORDER BY a.attendance_date DESC,sec.display_order,e.section_id,e.roll_number,st.full_name,a.id
		LIMIT 1000`,schoolID,yearID,sectionID,nullAttendanceDate(day),actorID)
	if err!=nil{return nil,fmt.Errorf("list attendance: %w",err)}
	defer rows.Close();items:=make([]AttendanceRecord,0)
	for rows.Next(){v,scanErr:=scanAttendance(rows);if scanErr!=nil{return nil,fmt.Errorf("scan attendance: %w",scanErr)};items=append(items,*v)}
	if err:=rows.Err();err!=nil{return nil,fmt.Errorf("iterate attendance: %w",err)}
	return items,nil
}

func nullAttendanceDate(day time.Time) any { if day.IsZero(){return nil};return day.Format("2006-01-02") }

func (r *AcademicRepository) MarkAttendance(ctx context.Context,actorID,schoolID,enrollmentID int64,day time.Time,status string,remarks *string)(*AttendanceRecord,error) {
	if r==nil||r.db==nil{return nil,ErrAttendanceAccessDenied}
	status=strings.TrimSpace(status);if enrollmentID<=0||day.IsZero()||!validAttendanceStatus(status){return nil,ErrInvalidAttendanceInput}
	if remarks!=nil{v:=strings.TrimSpace(*remarks);if len([]rune(v))>500{return nil,ErrInvalidAttendanceInput};if v==""{remarks=nil}else{remarks=&v}}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin attendance mark: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	var sectionID int64
	err=tx.QueryRow(ctx,`SELECT section_id FROM student_enrollments WHERE id=$1 AND school_id=$2`,enrollmentID,schoolID).Scan(&sectionID)
	if errors.Is(err,pgx.ErrNoRows){return nil,ErrAttendanceNotFound};if err!=nil{return nil,fmt.Errorf("load attendance enrollment: %w",err)}
	var isPast bool
	if err=tx.QueryRow(ctx,`SELECT $1::date < CURRENT_DATE`,day.Format("2006-01-02")).Scan(&isPast);err!=nil{return nil,fmt.Errorf("classify attendance date: %w",err)}
	permission:="attendance.mark";if isPast{permission="attendance.correct_past"}
	if err=attendancePermission(ctx,tx,actorID,schoolID,permission);err!=nil{return nil,err}
	allowed,err:=attendanceSectionAllowed(ctx,tx,actorID,schoolID,sectionID);if err!=nil{return nil,err};if !allowed{return nil,ErrAttendanceAccessDenied}
	var id int64
	err=tx.QueryRow(ctx,`INSERT INTO attendance_records
		(school_id,enrollment_id,attendance_date,status,remarks,marked_by_principal_id,updated_by_principal_id)
		VALUES($1,$2,$3,$4,$5,$6,$6) RETURNING id`,schoolID,enrollmentID,day.Format("2006-01-02"),status,remarks,actorID).Scan(&id)
	if err!=nil{return nil,attendanceWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.attendance.marked','attendance_record',$3,'success',
		jsonb_build_object('enrollment_id',$4::bigint,'attendance_date',$5::date,'status',$6::text))`,actorID,schoolID,fmt.Sprint(id),enrollmentID,day.Format("2006-01-02"),status);err!=nil{return nil,fmt.Errorf("audit attendance mark: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit attendance mark: %w",err)}
	return r.getAttendance(ctx,schoolID,id)
}

func validAttendanceStatus(status string) bool { return status=="present"||status=="absent"||status=="late"||status=="excused" }

func (r *AcademicRepository) UpdateAttendance(ctx context.Context,actorID,schoolID,recordID int64,status string,remarks *string)(*AttendanceRecord,error) {
	if r==nil||r.db==nil||recordID<=0{return nil,ErrInvalidAttendanceInput}
	status=strings.TrimSpace(status);if !validAttendanceStatus(status){return nil,ErrInvalidAttendanceInput}
	if remarks!=nil{v:=strings.TrimSpace(*remarks);if len([]rune(v))>500{return nil,ErrInvalidAttendanceInput};if v==""{remarks=nil}else{remarks=&v}}
	tx,err:=r.db.Begin(ctx);if err!=nil{return nil,fmt.Errorf("begin attendance update: %w",err)};defer func(){_=tx.Rollback(ctx)}()
	var sectionID int64;var day time.Time
	err=tx.QueryRow(ctx,`SELECT e.section_id,a.attendance_date FROM attendance_records a
		JOIN student_enrollments e ON e.id=a.enrollment_id AND e.school_id=a.school_id
		WHERE a.id=$1 AND a.school_id=$2 FOR UPDATE OF a`,recordID,schoolID).Scan(&sectionID,&day)
	if errors.Is(err,pgx.ErrNoRows){return nil,ErrAttendanceNotFound};if err!=nil{return nil,fmt.Errorf("load attendance for update: %w",err)}
	var isPast bool
	if err=tx.QueryRow(ctx,`SELECT $1::date < CURRENT_DATE`,day.Format("2006-01-02")).Scan(&isPast);err!=nil{return nil,fmt.Errorf("classify attendance date: %w",err)}
	permission:="attendance.update_same_day";if isPast{permission="attendance.correct_past"}
	if err=attendancePermission(ctx,tx,actorID,schoolID,permission);err!=nil{return nil,err}
	allowed,err:=attendanceSectionAllowed(ctx,tx,actorID,schoolID,sectionID);if err!=nil{return nil,err};if !allowed{return nil,ErrAttendanceAccessDenied}
	_,err=tx.Exec(ctx,`UPDATE attendance_records SET status=$3,remarks=$4,updated_by_principal_id=$5,updated_at=NOW()
		WHERE id=$1 AND school_id=$2`,recordID,schoolID,status,remarks,actorID)
	if err!=nil{return nil,attendanceWriteError(err)}
	if _,err=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.attendance.corrected','attendance_record',$3,'success',
		jsonb_build_object('attendance_date',$4::date,'status',$5::text))`,actorID,schoolID,fmt.Sprint(recordID),day.Format("2006-01-02"),status);err!=nil{return nil,fmt.Errorf("audit attendance correction: %w",err)}
	if err=tx.Commit(ctx);err!=nil{return nil,fmt.Errorf("commit attendance update: %w",err)}
	return r.getAttendance(ctx,schoolID,recordID)
}

func (r *AcademicRepository) getAttendance(ctx context.Context,schoolID,recordID int64)(*AttendanceRecord,error) {
	v,err:=scanAttendance(r.db.QueryRow(ctx,`SELECT `+attendanceColumns+`
		FROM attendance_records a JOIN student_enrollments e ON e.id=a.enrollment_id AND e.school_id=a.school_id
		JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
		WHERE a.school_id=$1 AND a.id=$2`,schoolID,recordID))
	if err!=nil{return nil,attendanceWriteError(err)};return v,nil
}
