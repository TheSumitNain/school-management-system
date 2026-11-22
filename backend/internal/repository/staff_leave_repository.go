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
 ErrStaffLeaveAccessDenied = errors.New("staff leave access denied")
 ErrStaffLeaveNotFound = errors.New("staff leave record not found")
 ErrStaffLeaveConflict = errors.New("staff leave request conflicts with another request")
 ErrInvalidStaffLeave = errors.New("invalid staff leave data")
)

type StaffLeaveType struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 IsPaid bool `json:"is_paid"`
 RequiresApproval bool `json:"requires_approval"`
 IsActive bool `json:"is_active"`
}
type StaffLeaveRequest struct {
 ID int64 `json:"id"`
 StaffPrincipalID int64 `json:"staff_principal_id"`
 EmployeeCode string `json:"employee_code,omitempty"`
 StaffName string `json:"staff_name,omitempty"`
 LeaveTypeID int64 `json:"leave_type_id"`
 LeaveTypeName string `json:"leave_type_name,omitempty"`
 StartDate string `json:"start_date"`
 EndDate string `json:"end_date"`
 DurationType string `json:"duration_type"`
 HalfDaySession *string `json:"half_day_session,omitempty"`
 Reason string `json:"reason"`
 Status string `json:"status"`
 DecisionComment *string `json:"decision_comment,omitempty"`
 DecidedByUsername *string `json:"decided_by_username,omitempty"`
 DecidedAt *time.Time `json:"decided_at,omitempty"`
 CreatedAt time.Time `json:"created_at"`
}
type SchoolHoliday struct {
 ID int64 `json:"id"`
 HolidayDate string `json:"holiday_date"`
 Name string `json:"name"`
 Description *string `json:"description,omitempty"`
 IsActive bool `json:"is_active"`
}
type StaffLeaveRepository struct{ db repositoryDB }
func NewStaffLeaveRepository(db repositoryDB)*StaffLeaveRepository{return &StaffLeaveRepository{db:db}}

func leaveWriteError(err error)error{
 var p *pgconn.PgError
 if errors.As(err,&p){switch p.Code{case "23505","23P01":return ErrStaffLeaveConflict;case "23503","23514","22001","22007":return ErrInvalidStaffLeave}}
 if errors.Is(err,pgx.ErrNoRows){return ErrStaffLeaveNotFound}
 return err
}
func cleanOptional(v *string,max int)(*string,error){
 if v==nil{return nil,nil}; s:=strings.TrimSpace(*v);if s==""{return nil,nil};if len([]rune(s))>max{return nil,ErrInvalidStaffLeave};return &s,nil
}
func(r *StaffLeaveRepository) ListTypes(ctx context.Context,actor,school int64)([]StaffLeaveType,error){
 if r==nil||r.db==nil{return nil,ErrInvalidStaffLeave}
 if authorizeGuardian(ctx,r.db,actor,school,"staff.leave.read")!=nil{return nil,ErrStaffLeaveAccessDenied}
 rows,e:=r.db.Query(ctx,`SELECT id,name,is_paid,requires_approval,is_active FROM staff_leave_types WHERE school_id=$1 ORDER BY lower(name),id`,school);if e!=nil{return nil,e};defer rows.Close()
 out:=[]StaffLeaveType{};for rows.Next(){var x StaffLeaveType;if e=rows.Scan(&x.ID,&x.Name,&x.IsPaid,&x.RequiresApproval,&x.IsActive);e!=nil{return nil,e};out=append(out,x)};return out,rows.Err()
}
func(r *StaffLeaveRepository) SaveType(ctx context.Context,actor,school,id int64,x StaffLeaveType)(*StaffLeaveType,error){
 x.Name=strings.TrimSpace(x.Name);if r==nil||r.db==nil||actor<=0||school<=0||len([]rune(x.Name))<2||len([]rune(x.Name))>100{return nil,ErrInvalidStaffLeave}
 tx,e:=r.db.Begin(ctx);if e!=nil{return nil,e};defer func(){_=tx.Rollback(ctx)}()
 if authorizeGuardian(ctx,tx,actor,school,"staff.leave.manage_types")!=nil{return nil,ErrStaffLeaveAccessDenied}
 if id==0{e=tx.QueryRow(ctx,`INSERT INTO staff_leave_types(school_id,name,is_paid,requires_approval,is_active,created_by_principal_id,updated_by_principal_id) VALUES($1,$2,$3,$4,$5,$6,$6) RETURNING id`,school,x.Name,x.IsPaid,x.RequiresApproval,x.IsActive,actor).Scan(&x.ID)}else{e=tx.QueryRow(ctx,`UPDATE staff_leave_types SET name=$3,is_paid=$4,requires_approval=$5,is_active=$6,updated_by_principal_id=$7,updated_at=NOW() WHERE id=$1 AND school_id=$2 RETURNING id`,id,school,x.Name,x.IsPaid,x.RequiresApproval,x.IsActive,actor).Scan(&x.ID)}
 if e!=nil{return nil,leaveWriteError(e)}
 action:="staff.leave_type.created";if id!=0{action="staff.leave_type.updated"}
 if _,e=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,$2,$3,'staff_leave_type',$4,'success',jsonb_build_object('is_active',$5::boolean))`,actor,school,action,fmt.Sprint(x.ID),x.IsActive);e!=nil{return nil,e}
 if e=tx.Commit(ctx);e!=nil{return nil,e};return &x,nil
}
func(r *StaffLeaveRepository) ListRequests(ctx context.Context,actor,school int64,status string,staffID int64,onDate string)([]StaffLeaveRequest,error){
 if r==nil||r.db==nil{return nil,ErrInvalidStaffLeave};if authorizeGuardian(ctx,r.db,actor,school,"staff.leave.read")!=nil{return nil,ErrStaffLeaveAccessDenied}
 rows,e:=r.db.Query(ctx,`SELECT q.id,q.staff_principal_id,sp.employee_code,sp.full_name,q.leave_type_id,t.name,q.start_date::text,q.end_date::text,q.duration_type,q.half_day_session,q.reason,q.status,q.decision_comment,u.username,q.decided_at,q.created_at FROM staff_leave_requests q JOIN staff_profiles sp ON sp.principal_id=q.staff_principal_id AND sp.school_id=q.school_id JOIN staff_leave_types t ON t.id=q.leave_type_id AND t.school_id=q.school_id LEFT JOIN users u ON u.principal_id=q.decided_by_principal_id AND u.school_id=q.school_id WHERE q.school_id=$1 AND ($2='' OR q.status=$2) AND ($3=0 OR q.staff_principal_id=$3) AND (NULLIF($4,'') IS NULL OR daterange(q.start_date,q.end_date,'[]') @> NULLIF($4,'')::date) ORDER BY q.start_date DESC,q.id DESC LIMIT 1000`,school,status,staffID,onDate);if e!=nil{return nil,e};defer rows.Close()
 out:=[]StaffLeaveRequest{};for rows.Next(){var x StaffLeaveRequest;if e=rows.Scan(&x.ID,&x.StaffPrincipalID,&x.EmployeeCode,&x.StaffName,&x.LeaveTypeID,&x.LeaveTypeName,&x.StartDate,&x.EndDate,&x.DurationType,&x.HalfDaySession,&x.Reason,&x.Status,&x.DecisionComment,&x.DecidedByUsername,&x.DecidedAt,&x.CreatedAt);e!=nil{return nil,e};out=append(out,x)};return out,rows.Err()
}
func(r *StaffLeaveRepository) CreateRequest(ctx context.Context,actor,school int64,x StaffLeaveRequest)(*StaffLeaveRequest,error){
 if r==nil||r.db==nil||actor<=0||school<=0||x.StaffPrincipalID<=0||x.LeaveTypeID<=0{return nil,ErrInvalidStaffLeave}
 start,e:=time.Parse("2006-01-02",x.StartDate);if e!=nil{return nil,ErrInvalidStaffLeave};end,e:=time.Parse("2006-01-02",x.EndDate);if e!=nil||end.Before(start){return nil,ErrInvalidStaffLeave}
 x.DurationType=strings.TrimSpace(x.DurationType);x.Reason=strings.TrimSpace(x.Reason)
 if len([]rune(x.Reason))==0||len([]rune(x.Reason))>1000{return nil,ErrInvalidStaffLeave}
 if x.DurationType=="half_day"{if !start.Equal(end)||x.HalfDaySession==nil||(*x.HalfDaySession!="first_half"&&*x.HalfDaySession!="second_half"){return nil,ErrInvalidStaffLeave}}else if x.DurationType=="full_day"{x.HalfDaySession=nil}else{return nil,ErrInvalidStaffLeave}
 tx,e:=r.db.Begin(ctx);if e!=nil{return nil,e};defer func(){_=tx.Rollback(ctx)}()
 if authorizeGuardian(ctx,tx,actor,school,"staff.leave.manage_requests")!=nil{return nil,ErrStaffLeaveAccessDenied}
 // Serialize overlap checks per school/staff so simultaneous submissions cannot both pass.
 if _,e=tx.Exec(ctx,`SELECT pg_advisory_xact_lock(hashtextextended($1::text || ':' || $2::text,0))`,school,x.StaffPrincipalID);e!=nil{return nil,e}
 var active bool
 e=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM staff_profiles WHERE principal_id=$1 AND school_id=$2 AND employment_status='active') AND EXISTS(SELECT 1 FROM staff_leave_types WHERE id=$3 AND school_id=$2 AND is_active=TRUE)`,x.StaffPrincipalID,school,x.LeaveTypeID).Scan(&active)
 if e!=nil{return nil,e};if !active{return nil,ErrStaffLeaveNotFound}
 var overlap bool
 e=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM staff_leave_requests WHERE school_id=$1 AND staff_principal_id=$2 AND status IN ('pending','approved') AND daterange(start_date,end_date,'[]') && daterange($3::date,$4::date,'[]'))`,school,x.StaffPrincipalID,x.StartDate,x.EndDate).Scan(&overlap)
 if e!=nil{return nil,e};if overlap{return nil,ErrStaffLeaveConflict}
 e=tx.QueryRow(ctx,`INSERT INTO staff_leave_requests(school_id,staff_principal_id,leave_type_id,start_date,end_date,duration_type,half_day_session,reason,status,created_by_principal_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9) RETURNING id,created_at`,school,x.StaffPrincipalID,x.LeaveTypeID,x.StartDate,x.EndDate,x.DurationType,x.HalfDaySession,x.Reason,actor).Scan(&x.ID,&x.CreatedAt)
 if e!=nil{return nil,leaveWriteError(e)}
 if _,e=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,$2,'staff.leave.requested','staff_leave_request',$3,'success',jsonb_build_object('staff_principal_id',$4::bigint,'leave_type_id',$5::bigint,'start_date',$6::date,'end_date',$7::date))`,actor,school,fmt.Sprint(x.ID),x.StaffPrincipalID,x.LeaveTypeID,x.StartDate,x.EndDate);e!=nil{return nil,e}
 if e=tx.Commit(ctx);e!=nil{return nil,e};return &x,nil
}
func(r *StaffLeaveRepository) Decide(ctx context.Context,actor,school,id int64,approve bool,comment *string)(error){
 if r==nil||r.db==nil||actor<=0||school<=0||id<=0{return ErrInvalidStaffLeave}
 var e error;comment,e=cleanOptional(comment,1000);if e!=nil{return e};if !approve&&comment==nil{return ErrInvalidStaffLeave}
 tx,e:=r.db.Begin(ctx);if e!=nil{return e};defer func(){_=tx.Rollback(ctx)}()
 if authorizeGuardian(ctx,tx,actor,school,"staff.leave.decide")!=nil{return ErrStaffLeaveAccessDenied}
 var applicant int64
 e=tx.QueryRow(ctx,`SELECT staff_principal_id FROM staff_leave_requests WHERE id=$1 AND school_id=$2 AND status='pending' FOR UPDATE`,id,school).Scan(&applicant)
 if errors.Is(e,pgx.ErrNoRows){return ErrStaffLeaveNotFound};if e!=nil{return e};if applicant==actor{return ErrStaffLeaveAccessDenied}
 status:="approved";action:="staff.leave.approved";if !approve{status="rejected";action="staff.leave.rejected"}
 _,e=tx.Exec(ctx,`UPDATE staff_leave_requests SET status=$3,decision_comment=$4,decided_by_principal_id=$5,decided_at=NOW(),updated_at=NOW() WHERE id=$1 AND school_id=$2`,id,school,status,comment,actor);if e!=nil{return leaveWriteError(e)}
 if _,e=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,$2,$3,'staff_leave_request',$4,'success',jsonb_build_object('status',$5::text))`,actor,school,action,fmt.Sprint(id),status);e!=nil{return e}
 return tx.Commit(ctx)
}
func(r *StaffLeaveRepository) Cancel(ctx context.Context,actor,school,id int64)error{
 tx,e:=r.db.Begin(ctx);if e!=nil{return e};defer func(){_=tx.Rollback(ctx)}()
 if authorizeGuardian(ctx,tx,actor,school,"staff.leave.manage_requests")!=nil{return ErrStaffLeaveAccessDenied}
 tag,e:=tx.Exec(ctx,`UPDATE staff_leave_requests SET status='cancelled',updated_at=NOW() WHERE id=$1 AND school_id=$2 AND status='pending'`,id,school);if e!=nil{return e};if tag.RowsAffected()!=1{return ErrStaffLeaveNotFound}
 if _,e=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome) VALUES($1,$2,'staff.leave.cancelled','staff_leave_request',$3,'success')`,actor,school,fmt.Sprint(id));e!=nil{return e};return tx.Commit(ctx)
}
func(r *StaffLeaveRepository) ListHolidays(ctx context.Context,actor,school int64,from,to string)([]SchoolHoliday,error){
 if authorizeGuardian(ctx,r.db,actor,school,"staff.holidays.read")!=nil{return nil,ErrStaffLeaveAccessDenied}
 rows,e:=r.db.Query(ctx,`SELECT id,holiday_date::text,name,description,is_active FROM school_holidays WHERE school_id=$1 AND holiday_date BETWEEN $2::date AND $3::date AND is_active=TRUE ORDER BY holiday_date,name LIMIT 1000`,school,from,to);if e!=nil{return nil,e};defer rows.Close();out:=[]SchoolHoliday{}
 for rows.Next(){var x SchoolHoliday;if e=rows.Scan(&x.ID,&x.HolidayDate,&x.Name,&x.Description,&x.IsActive);e!=nil{return nil,e};out=append(out,x)};return out,rows.Err()
}
func(r *StaffLeaveRepository) SaveHoliday(ctx context.Context,actor,school,id int64,x SchoolHoliday)(*SchoolHoliday,error){
 if r==nil||r.db==nil||actor<=0||school<=0{return nil,ErrInvalidStaffLeave};if _,e:=time.Parse("2006-01-02",x.HolidayDate);e!=nil{return nil,ErrInvalidStaffLeave}
 x.Name=strings.TrimSpace(x.Name);if len([]rune(x.Name))<2||len([]rune(x.Name))>160{return nil,ErrInvalidStaffLeave};desc,e:=cleanOptional(x.Description,1000);if e!=nil{return nil,e};x.Description=desc
 tx,e:=r.db.Begin(ctx);if e!=nil{return nil,e};defer func(){_=tx.Rollback(ctx)}();if authorizeGuardian(ctx,tx,actor,school,"staff.holidays.manage")!=nil{return nil,ErrStaffLeaveAccessDenied}
 if id==0{e=tx.QueryRow(ctx,`INSERT INTO school_holidays(school_id,holiday_date,name,description,is_active,created_by_principal_id,updated_by_principal_id) VALUES($1,$2,$3,$4,$5,$6,$6) RETURNING id`,school,x.HolidayDate,x.Name,x.Description,x.IsActive,actor).Scan(&x.ID)}else{e=tx.QueryRow(ctx,`UPDATE school_holidays SET holiday_date=$3,name=$4,description=$5,is_active=$6,updated_by_principal_id=$7,updated_at=NOW() WHERE id=$1 AND school_id=$2 RETURNING id`,id,school,x.HolidayDate,x.Name,x.Description,x.IsActive,actor).Scan(&x.ID)}
 if e!=nil{return nil,leaveWriteError(e)};action:="school.holiday.created";if id!=0{action="school.holiday.updated"}
 if _,e=tx.Exec(ctx,`INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata) VALUES($1,$2,$3,'school_holiday',$4,'success',jsonb_build_object('holiday_date',$5::date,'is_active',$6::boolean))`,actor,school,action,fmt.Sprint(x.ID),x.HolidayDate,x.IsActive);e!=nil{return nil,e}
 if e=tx.Commit(ctx);e!=nil{return nil,e};return &x,nil
}
