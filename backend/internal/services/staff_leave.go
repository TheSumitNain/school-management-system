package services

import (
 "context"
 "errors"
 "strings"
 "time"

 "school-management-system/internal/repository"
)
var ErrInvalidStaffLeave = errors.New("invalid staff leave input")
type staffLeaveRepository interface {
 ListTypes(context.Context,int64,int64)([]repository.StaffLeaveType,error)
 SaveType(context.Context,int64,int64,int64,repository.StaffLeaveType)(*repository.StaffLeaveType,error)
 ListRequests(context.Context,int64,int64,string,int64)([]repository.StaffLeaveRequest,error)
 CreateRequest(context.Context,int64,int64,repository.StaffLeaveRequest)(*repository.StaffLeaveRequest,error)
 Decide(context.Context,int64,int64,int64,bool,*string)error
 Cancel(context.Context,int64,int64,int64)error
 ListHolidays(context.Context,int64,int64,string,string)([]repository.SchoolHoliday,error)
 SaveHoliday(context.Context,int64,int64,int64,repository.SchoolHoliday)(*repository.SchoolHoliday,error)
}
type StaffLeaveService struct{ repo staffLeaveRepository }
func NewStaffLeaveService(repo staffLeaveRepository)*StaffLeaveService{return &StaffLeaveService{repo:repo}}
func(s *StaffLeaveService) ListTypes(c context.Context,a,sc int64)([]repository.StaffLeaveType,error){if s==nil||s.repo==nil||a<=0||sc<=0{return nil,ErrInvalidStaffLeave};return s.repo.ListTypes(c,a,sc)}
func(s *StaffLeaveService) SaveType(c context.Context,a,sc,id int64,x repository.StaffLeaveType)(*repository.StaffLeaveType,error){x.Name=strings.TrimSpace(x.Name);if s==nil||s.repo==nil||a<=0||sc<=0||len([]rune(x.Name))<2||len([]rune(x.Name))>100{return nil,ErrInvalidStaffLeave};return s.repo.SaveType(c,a,sc,id,x)}
func(s *StaffLeaveService) ListRequests(c context.Context,a,sc int64,status string,staff int64)([]repository.StaffLeaveRequest,error){if s==nil||s.repo==nil||a<=0||sc<=0||staff<0{return nil,ErrInvalidStaffLeave};status=strings.TrimSpace(status);switch status{case "","pending","approved","rejected","cancelled":default:return nil,ErrInvalidStaffLeave};return s.repo.ListRequests(c,a,sc,status,staff)}
func(s *StaffLeaveService) CreateRequest(c context.Context,a,sc int64,x repository.StaffLeaveRequest)(*repository.StaffLeaveRequest,error){if s==nil||s.repo==nil||a<=0||sc<=0||x.StaffPrincipalID<=0||x.LeaveTypeID<=0{return nil,ErrInvalidStaffLeave};if _,e:=time.Parse("2006-01-02",x.StartDate);e!=nil{return nil,ErrInvalidStaffLeave};if _,e:=time.Parse("2006-01-02",x.EndDate);e!=nil{return nil,ErrInvalidStaffLeave};x.Reason=strings.TrimSpace(x.Reason);if x.Reason==""||len([]rune(x.Reason))>1000{return nil,ErrInvalidStaffLeave};return s.repo.CreateRequest(c,a,sc,x)}
func(s *StaffLeaveService) Decide(c context.Context,a,sc,id int64,approve bool,comment *string)error{if s==nil||s.repo==nil||a<=0||sc<=0||id<=0{return ErrInvalidStaffLeave};if !approve&&(comment==nil||strings.TrimSpace(*comment)==""){return ErrInvalidStaffLeave};return s.repo.Decide(c,a,sc,id,approve,comment)}
func(s *StaffLeaveService) Cancel(c context.Context,a,sc,id int64)error{if s==nil||s.repo==nil||a<=0||sc<=0||id<=0{return ErrInvalidStaffLeave};return s.repo.Cancel(c,a,sc,id)}
func(s *StaffLeaveService) ListHolidays(c context.Context,a,sc int64,from,to string)([]repository.SchoolHoliday,error){if s==nil||s.repo==nil||a<=0||sc<=0{return nil,ErrInvalidStaffLeave};f,e:=time.Parse("2006-01-02",from);if e!=nil{return nil,ErrInvalidStaffLeave};t,e:=time.Parse("2006-01-02",to);if e!=nil||t.Before(f)||t.Sub(f)>366*24*time.Hour{return nil,ErrInvalidStaffLeave};return s.repo.ListHolidays(c,a,sc,from,to)}
func(s *StaffLeaveService) SaveHoliday(c context.Context,a,sc,id int64,x repository.SchoolHoliday)(*repository.SchoolHoliday,error){if s==nil||s.repo==nil||a<=0||sc<=0{return nil,ErrInvalidStaffLeave};return s.repo.SaveHoliday(c,a,sc,id,x)}
