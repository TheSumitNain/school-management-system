package handlers

import (
 "context"
 "errors"
 "net/http"
 "strconv"
 "strings"

 "school-management-system/internal/repository"
 "school-management-system/internal/services"
 "github.com/gin-gonic/gin"
)
type StaffLeaveManagementService interface {
 ListTypes(context.Context,int64,int64)([]repository.StaffLeaveType,error)
 SaveType(context.Context,int64,int64,int64,repository.StaffLeaveType)(*repository.StaffLeaveType,error)
 ListRequests(context.Context,int64,int64,string,int64,string,string,string)([]repository.StaffLeaveRequest,error)
 CreateRequest(context.Context,int64,int64,repository.StaffLeaveRequest)(*repository.StaffLeaveRequest,error)
 Decide(context.Context,int64,int64,int64,bool,*string)error
 Cancel(context.Context,int64,int64,int64)error
 ListHolidays(context.Context,int64,int64,string,string,bool)([]repository.SchoolHoliday,error)
 SaveHoliday(context.Context,int64,int64,int64,repository.SchoolHoliday)(*repository.SchoolHoliday,error)
}
type StaffLeaveHandler struct{ service StaffLeaveManagementService }
func NewStaffLeaveHandler(s StaffLeaveManagementService)*StaffLeaveHandler{return &StaffLeaveHandler{service:s}}
func(h *StaffLeaveHandler) actor(c *gin.Context)(int64,int64,bool){return staffRecordActor(c)}
func(h *StaffLeaveHandler) fail(c *gin.Context,e error){
 switch{
 case errors.Is(e,repository.ErrStaffLeaveAccessDenied),errors.Is(e,repository.ErrRolePermissionDenied):
  c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"You are not authorized to manage staff leave or holidays"})
 case errors.Is(e,repository.ErrStaffLeaveNotFound):
  c.JSON(http.StatusNotFound,gin.H{"status":"error","message":"Staff leave record, leave type or employee not found"})
 case errors.Is(e,repository.ErrStaffLeaveConflict):
  c.JSON(http.StatusConflict,gin.H{"status":"error","message":"This leave request overlaps another pending or approved request, or the record already exists"})
 case errors.Is(e,repository.ErrInvalidStaffLeave),errors.Is(e,services.ErrInvalidStaffLeave):
  c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Check the leave or holiday fields"})
 default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process staff leave or holiday"})
 }
}
func(h *StaffLeaveHandler) ListTypes(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};v,e:=h.service.ListTypes(c,a,s);if e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","data":v})}
func(h *StaffLeaveHandler) SaveType(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};id:=int64(0);if c.Param("typeID")!=""{v,e:=strconv.ParseInt(c.Param("typeID"),10,64);if e!=nil||v<=0{c.JSON(400,gin.H{"status":"error","message":"Invalid leave type ID"});return};id=v};var x repository.StaffLeaveType;if c.ShouldBindJSON(&x)!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid leave type payload"});return};v,e:=h.service.SaveType(c,a,s,id,x);if e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","data":v})}
func(h *StaffLeaveHandler) ListRequests(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};staff:=int64(0);if raw:=c.Query("staff_principal_id");raw!=""{v,e:=strconv.ParseInt(raw,10,64);if e!=nil||v<=0{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return};staff=v};v,e:=h.service.ListRequests(c,a,s,c.Query("status"),staff,strings.TrimSpace(c.Query("on_date")),strings.TrimSpace(c.Query("from_date")),strings.TrimSpace(c.Query("to_date")));if e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","data":v})}
func(h *StaffLeaveHandler) CreateRequest(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};var x repository.StaffLeaveRequest;if c.ShouldBindJSON(&x)!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid leave request payload"});return};v,e:=h.service.CreateRequest(c,a,s,x);if e!=nil{h.fail(c,e);return};c.JSON(201,gin.H{"status":"success","data":v})}
func(h *StaffLeaveHandler) decision(c *gin.Context,approve bool){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};id,e:=strconv.ParseInt(c.Param("requestID"),10,64);if e!=nil||id<=0{c.JSON(400,gin.H{"status":"error","message":"Invalid leave request ID"});return};var body struct{Comment *string `json:"comment"`};if c.Request.ContentLength>0&&c.ShouldBindJSON(&body)!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid decision payload"});return};if e=h.service.Decide(c,a,s,id,approve,body.Comment);e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","message":"Leave request decision saved"})}
func(h *StaffLeaveHandler) Approve(c *gin.Context){h.decision(c,true)}
func(h *StaffLeaveHandler) Reject(c *gin.Context){h.decision(c,false)}
func(h *StaffLeaveHandler) Cancel(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};id,e:=strconv.ParseInt(c.Param("requestID"),10,64);if e!=nil||id<=0{c.JSON(400,gin.H{"status":"error","message":"Invalid leave request ID"});return};if e=h.service.Cancel(c,a,s,id);e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","message":"Leave request cancelled"})}
func(h *StaffLeaveHandler) ListHolidays(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};from,to:=strings.TrimSpace(c.Query("from")),strings.TrimSpace(c.Query("to"));includeInactive:=strings.EqualFold(strings.TrimSpace(c.Query("include_inactive")),"true");v,e:=h.service.ListHolidays(c,a,s,from,to,includeInactive);if e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","data":v})}
func(h *StaffLeaveHandler) SaveHoliday(c *gin.Context){a,s,ok:=h.actor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return};id:=int64(0);if c.Param("holidayID")!=""{v,e:=strconv.ParseInt(c.Param("holidayID"),10,64);if e!=nil||v<=0{c.JSON(400,gin.H{"status":"error","message":"Invalid holiday ID"});return};id=v};var x repository.SchoolHoliday;if c.ShouldBindJSON(&x)!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid holiday payload"});return};v,e:=h.service.SaveHoliday(c,a,s,id,x);if e!=nil{h.fail(c,e);return};c.JSON(200,gin.H{"status":"success","data":v})}
