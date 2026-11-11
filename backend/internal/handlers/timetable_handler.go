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

type TimetableManagementService interface {
	ListPeriods(context.Context,int64,int64,int64)([]repository.TimetablePeriod,error)
	SavePeriod(context.Context,int64,int64,int64,int64,int,string,string,string,bool)(*repository.TimetablePeriod,error)
	ListEntries(context.Context,int64,int64,int64,int64)([]repository.TimetableEntry,error)
	SaveEntry(context.Context,int64,int64,int64,int64,int64,int64,int64,int64,int64,int,*string)(*repository.TimetableEntry,error)
	DeleteEntry(context.Context,int64,int64,int64,int64)error
}

type TimetableHandler struct{service TimetableManagementService}
func NewTimetableHandler(service TimetableManagementService)*TimetableHandler{return &TimetableHandler{service:service}}

func(h *TimetableHandler) ready(c *gin.Context)(int64,int64,bool){
	v,ok:=c.Get(middleware.ContextPrincipalIDKey);actor,aok:=v.(int64)
	v,ok2:=c.Get(middleware.ContextSchoolIDKey);school,sok:=v.(int64)
	if !ok||!ok2||!aok||!sok||actor<=0||school<=0{c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"School staff authentication required"});return 0,0,false}
	if h==nil||h.service==nil{c.JSON(http.StatusServiceUnavailable,gin.H{"status":"error","message":"Timetable management is unavailable"});return 0,0,false}
	return actor,school,true
}
func timetablePathID(c *gin.Context,key string)(int64,bool){v,e:=strconv.ParseInt(c.Param(key),10,64);return v,e==nil&&v>0}
func timetableQueryID(c *gin.Context,key string)(int64,bool){raw:=strings.TrimSpace(c.Query(key));if raw==""{return 0,false};v,e:=strconv.ParseInt(raw,10,64);return v,e==nil&&v>0}
func timetableError(c *gin.Context,err error,resource string){
	switch{
	case errors.Is(err,repository.ErrTimetableAccessDenied):c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"You are not authorized to manage this timetable"})
	case errors.Is(err,repository.ErrTimetableNotFound):c.JSON(http.StatusNotFound,gin.H{"status":"error","message":resource+" not found"})
	case errors.Is(err,repository.ErrTimetableConflict):c.JSON(http.StatusConflict,gin.H{"status":"error","message":"This class or teacher already has a timetable entry in that period"})
	case errors.Is(err,repository.ErrInvalidTimetableInput):c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid timetable data; check the academic year, active assignment, period and time values"})
	default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process timetable request"})
	}
}

func(h *TimetableHandler) ListPeriods(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};year,valid:=timetableQueryID(c,"academic_year_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"academic_year_id is required"});return}
	items,err:=h.service.ListPeriods(c.Request.Context(),a,s,year);if err!=nil{timetableError(c,err,"Timetable period");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","periods":items})
}
func(h *TimetableHandler) SavePeriod(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64
	if c.Param("periodID")!=""{var valid bool;id,valid=timetablePathID(c,"periodID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid period ID"});return}}
	var req struct{AcademicYearID int64 `json:"academic_year_id"`;PeriodNumber int `json:"period_number"`;Name string `json:"name"`;StartsAt string `json:"starts_at"`;EndsAt string `json:"ends_at"`;IsBreak bool `json:"is_break"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid period payload"});return}
	item,err:=h.service.SavePeriod(c.Request.Context(),a,s,id,req.AcademicYearID,req.PeriodNumber,req.Name,req.StartsAt,req.EndsAt,req.IsBreak)
	if err!=nil{timetableError(c,err,"Timetable period");return};status:=http.StatusOK;if id==0{status=http.StatusCreated};c.JSON(status,gin.H{"status":"success","period":item})
}
func(h *TimetableHandler) ListEntries(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};year,valid:=timetableQueryID(c,"academic_year_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"academic_year_id is required"});return}
	section:=int64(0);if raw:=strings.TrimSpace(c.Query("section_id"));raw!=""{v,e:=strconv.ParseInt(raw,10,64);if e!=nil||v<=0{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid section_id"});return};section=v}
	items,err:=h.service.ListEntries(c.Request.Context(),a,s,year,section);if err!=nil{timetableError(c,err,"Timetable entry");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","entries":items})
}
func(h *TimetableHandler) SaveEntry(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64
	if c.Param("entryID")!=""{var valid bool;id,valid=timetablePathID(c,"entryID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid timetable entry ID"});return}}
	var req struct{AcademicYearID int64 `json:"academic_year_id"`;SectionID int64 `json:"section_id"`;SubjectID int64 `json:"subject_id"`;TeacherPrincipalID int64 `json:"teacher_principal_id"`;TeacherSubjectAssignmentID int64 `json:"teacher_subject_assignment_id"`;PeriodID int64 `json:"period_id"`;DayOfWeek int `json:"day_of_week"`;Room *string `json:"room"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid timetable entry payload"});return}
	item,err:=h.service.SaveEntry(c.Request.Context(),a,s,id,req.AcademicYearID,req.SectionID,req.SubjectID,req.TeacherPrincipalID,req.TeacherSubjectAssignmentID,req.PeriodID,req.DayOfWeek,req.Room)
	if err!=nil{timetableError(c,err,"Timetable entry");return};status:=http.StatusOK;if id==0{status=http.StatusCreated};c.JSON(status,gin.H{"status":"success","entry":item})
}
func(h *TimetableHandler) DeleteEntry(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id,valid:=timetablePathID(c,"entryID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid timetable entry ID"});return}
	year,valid:=timetableQueryID(c,"academic_year_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"academic_year_id is required"});return}
	if err:=h.service.DeleteEntry(c.Request.Context(),a,s,year,id);err!=nil{timetableError(c,err,"Timetable entry");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Timetable entry deleted"})
}
