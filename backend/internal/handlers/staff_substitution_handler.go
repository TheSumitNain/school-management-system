package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type StaffSubstitutionManagementService interface {
	List(context.Context,int64,int64,string,string,int64)([]repository.StaffSubstitution,error)
	Create(context.Context,int64,int64,int64,int64,int64,string,*int64,string)(*repository.StaffSubstitution,error)
	Cancel(context.Context,int64,int64,int64)error
}

type StaffSubstitutionHandler struct { service StaffSubstitutionManagementService }

func NewStaffSubstitutionHandler(service StaffSubstitutionManagementService)*StaffSubstitutionHandler {
	return &StaffSubstitutionHandler{service:service}
}

func(h *StaffSubstitutionHandler) ready(c *gin.Context)(int64,int64,bool) {
	v,ok:=c.Get(middleware.ContextPrincipalIDKey);actor,aok:=v.(int64)
	v,ok2:=c.Get(middleware.ContextSchoolIDKey);school,sok:=v.(int64)
	if !ok||!ok2||!aok||!sok||actor<=0||school<=0 {
		c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"School staff authentication required"});return 0,0,false
	}
	if h==nil||h.service==nil {
		c.JSON(http.StatusServiceUnavailable,gin.H{"status":"error","message":"Staff substitution management is unavailable"});return 0,0,false
	}
	return actor,school,true
}

func staffSubstitutionError(c *gin.Context,err error) {
	switch {
	case errors.Is(err,repository.ErrStaffSubstitutionAccessDenied):
		c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"You are not authorized to manage staff substitutions"})
	case errors.Is(err,repository.ErrStaffSubstitutionNotFound):
		c.JSON(http.StatusNotFound,gin.H{"status":"error","message":"Staff substitution not found or already cancelled"})
	case errors.Is(err,repository.ErrStaffSubstitutionConflict):
		c.JSON(http.StatusConflict,gin.H{"status":"error","code":"staff_substitution_conflict","message":"The substitute already has a class or another coverage assignment in this period. Refresh the timetable and choose another substitute."})
	case errors.Is(err,repository.ErrInvalidStaffSubstitution),errors.Is(err,services.ErrInvalidStaffSubstitution):
		c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid substitution details. Check the timetable entry, date, staff, leave reference and reason."})
	default:
		c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process staff substitution"})
	}
}

func(h *StaffSubstitutionHandler) List(c *gin.Context) {
	actor,school,ok:=h.ready(c);if !ok{return}
	from:=strings.TrimSpace(c.Query("from_date"));to:=strings.TrimSpace(c.Query("to_date"))
	staffID:=int64(0)
	if raw:=strings.TrimSpace(c.Query("staff_principal_id"));raw!="" {
		v,err:=strconv.ParseInt(raw,10,64);if err!=nil||v<=0 {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid staff_principal_id"});return};staffID=v
	}
	items,err:=h.service.List(c.Request.Context(),actor,school,from,to,staffID)
	if err!=nil{staffSubstitutionError(c,err);return}
	c.JSON(http.StatusOK,gin.H{"status":"success","substitutions":items})
}

func(h *StaffSubstitutionHandler) Create(c *gin.Context) {
	actor,school,ok:=h.ready(c);if !ok{return}
	var req struct {
		TimetableEntryID int64 `json:"timetable_entry_id"`
		OriginalTeacherPrincipalID int64 `json:"original_teacher_principal_id"`
		SubstituteTeacherPrincipalID int64 `json:"substitute_teacher_principal_id"`
		CoverDate string `json:"cover_date"`
		ApprovedLeaveRequestID *int64 `json:"approved_leave_request_id"`
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&req)!=nil {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid substitution payload"});return}
	item,err:=h.service.Create(c.Request.Context(),actor,school,req.TimetableEntryID,req.OriginalTeacherPrincipalID,
		req.SubstituteTeacherPrincipalID,req.CoverDate,req.ApprovedLeaveRequestID,req.Reason)
	if err!=nil{staffSubstitutionError(c,err);return}
	if item==nil{c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to create staff substitution"});return}
	c.JSON(http.StatusCreated,gin.H{"status":"success","substitution":item})
}

func(h *StaffSubstitutionHandler) Cancel(c *gin.Context) {
	actor,school,ok:=h.ready(c);if !ok{return}
	id,err:=strconv.ParseInt(c.Param("substitutionID"),10,64)
	if err!=nil||id<=0 {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid substitution ID"});return}
	if err=h.service.Cancel(c.Request.Context(),actor,school,id);err!=nil{staffSubstitutionError(c,err);return}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Substitution cancelled; history has been retained"})
}
