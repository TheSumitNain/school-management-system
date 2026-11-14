package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type GuardianManagementService interface {
	ListGuardians(context.Context,int64,int64,string,string,int,int)([]repository.Guardian,int64,error)
	GetGuardian(context.Context,int64,int64,int64)(*repository.Guardian,error)
	SaveGuardian(context.Context,int64,int64,int64,string,*string,*string,string)(*repository.Guardian,error)
	ListStudentGuardians(context.Context,int64,int64,int64,bool)([]repository.StudentGuardian,error)
	ListStudentRelatedStudents(context.Context,int64,int64,int64)([]repository.StudentRelatedStudent,error)
	LinkStudentGuardian(context.Context,int64,int64,int64,int64,string,bool,bool)(*repository.StudentGuardian,error)
	RevokeStudentGuardianLink(context.Context,int64,int64,int64,string)error
}

type GuardianHandler struct { service GuardianManagementService }
func NewGuardianHandler(service GuardianManagementService)*GuardianHandler{return &GuardianHandler{service:service}}

func guardianActor(c *gin.Context)(int64,int64,bool){
	a,ao:=c.Get(middleware.ContextPrincipalIDKey);s,so:=c.Get(middleware.ContextSchoolIDKey)
	actor,aok:=a.(int64);school,sok:=s.(int64)
	return actor,school,ao&&so&&aok&&sok&&actor>0&&school>0
}
func guardianID(c *gin.Context,key string)(int64,bool){v,e:=strconv.ParseInt(c.Param(key),10,64);return v,e==nil&&v>0}
func guardianError(c *gin.Context,err error,resource string){
	switch{
	case errors.Is(err,repository.ErrGuardianAccessDenied):c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Guardian operation is not authorized"})
	case errors.Is(err,repository.ErrGuardianNotFound):c.JSON(http.StatusNotFound,gin.H{"status":"error","message":resource+" not found"})
	case errors.Is(err,repository.ErrGuardianConflict):c.JSON(http.StatusConflict,gin.H{"status":"error","message":resource+" conflicts with an existing record"})
	case errors.Is(err,repository.ErrInvalidGuardianInput):c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid guardian data"})
	default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process "+strings.ToLower(resource)+" request"})
	}
}
func(h *GuardianHandler) ready(c *gin.Context)(int64,int64,bool){
	a,s,ok:=guardianActor(c);if !ok{c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"School staff authentication required"});return 0,0,false}
	if h==nil||h.service==nil{c.JSON(http.StatusServiceUnavailable,gin.H{"status":"error","message":"Guardian management is unavailable"});return 0,0,false}
	return a,s,true
}

func(h *GuardianHandler) ListGuardians(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return}
	limit:=50;offset:=0
	if raw:=c.Query("limit");raw!=""{v,e:=strconv.Atoi(raw);if e!=nil||v<1||v>100{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"limit must be between 1 and 100"});return};limit=v}
	if raw:=c.Query("offset");raw!=""{v,e:=strconv.Atoi(raw);if e!=nil||v<0{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"offset must be zero or greater"});return};offset=v}
	items,total,err:=h.service.ListGuardians(c.Request.Context(),a,s,c.Query("q"),c.Query("status"),limit,offset)
	if err!=nil{guardianError(c,err,"Guardian");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","guardians":items,"total":total,"limit":limit,"offset":offset})
}

func(h *GuardianHandler) GetGuardian(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id,valid:=guardianID(c,"guardianID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid guardian ID"});return}
	v,err:=h.service.GetGuardian(c.Request.Context(),a,s,id);if err!=nil{guardianError(c,err,"Guardian");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","guardian":v})
}

func(h *GuardianHandler) SaveGuardian(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id:=int64(0)
	if c.Param("guardianID")!=""{var valid bool;id,valid=guardianID(c,"guardianID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid guardian ID"});return}}
	var req struct{FullName string `json:"full_name"`;MobileNumber *string `json:"mobile_number"`;EmailAddress *string `json:"email_address"`;Status string `json:"status"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	if req.Status==""&&id==0{req.Status="active"}
	if req.Status==""{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"status is required when updating a guardian"});return}
	v,err:=h.service.SaveGuardian(c.Request.Context(),a,s,id,req.FullName,req.MobileNumber,req.EmailAddress,req.Status)
	if err!=nil{guardianError(c,err,"Guardian");return}
	code:=http.StatusOK;if id==0{code=http.StatusCreated}
	c.JSON(code,gin.H{"status":"success","guardian":v})
}

func(h *GuardianHandler) ListStudentRelatedStudents(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return}
	student,valid:=guardianID(c,"studentID")
	if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid student ID"});return}
	items,err:=h.service.ListStudentRelatedStudents(c.Request.Context(),a,s,student)
	if err!=nil{guardianError(c,err,"Related student");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","related_students":items})
}

func(h *GuardianHandler) ListStudentGuardians(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};student,valid:=guardianID(c,"studentID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid student ID"});return}
	history:=strings.EqualFold(c.Query("include_history"),"true")
	items,err:=h.service.ListStudentGuardians(c.Request.Context(),a,s,student,history)
	if err!=nil{guardianError(c,err,"Student guardian relationship");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","student_guardians":items})
}

func(h *GuardianHandler) LinkStudentGuardian(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};student,valid:=guardianID(c,"studentID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid student ID"});return}
	var req struct{GuardianID int64 `json:"guardian_id"`;RelationshipType string `json:"relationship_type"`;IsPrimaryContact bool `json:"is_primary_contact"`;IsEmergencyContact bool `json:"is_emergency_contact"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	v,err:=h.service.LinkStudentGuardian(c.Request.Context(),a,s,student,req.GuardianID,req.RelationshipType,req.IsPrimaryContact,req.IsEmergencyContact)
	if err!=nil{guardianError(c,err,"Student guardian relationship");return}
	c.JSON(http.StatusCreated,gin.H{"status":"success","student_guardian":v})
}

func(h *GuardianHandler) RevokeStudentGuardianLink(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id,valid:=guardianID(c,"linkID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid relationship ID"});return}
	var req struct{Reason string `json:"reason"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	if err:=h.service.RevokeStudentGuardianLink(c.Request.Context(),a,s,id,req.Reason);err!=nil{guardianError(c,err,"Student guardian relationship");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Guardian relationship revoked"})
}
