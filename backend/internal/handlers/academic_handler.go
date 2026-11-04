package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type AcademicManagementService interface {
	ListAcademicYears(context.Context,int64,int64)([]repository.AcademicYear,error)
	SaveAcademicYear(context.Context,int64,int64,int64,string,time.Time,time.Time,string,bool)(*repository.AcademicYear,error)
	ListClasses(context.Context,int64,int64)([]repository.AcademicClass,error)
	SaveClass(context.Context,int64,int64,int64,string,int,bool)(*repository.AcademicClass,error)
	ListSections(context.Context,int64,int64,int64)([]repository.AcademicSection,error)
	SaveSection(context.Context,int64,int64,int64,int64,int64,string,int,bool)(*repository.AcademicSection,error)
	ListSubjects(context.Context,int64,int64)([]repository.AcademicSubject,error)
	SaveSubject(context.Context,int64,int64,int64,string,*string,bool)(*repository.AcademicSubject,error)
}

type AcademicHandler struct { service AcademicManagementService }
func NewAcademicHandler(service AcademicManagementService)*AcademicHandler{return &AcademicHandler{service:service}}

func academicActor(c *gin.Context)(int64,int64,bool){
	a,ao:=c.Get(middleware.ContextPrincipalIDKey);s,so:=c.Get(middleware.ContextSchoolIDKey)
	actor,aok:=a.(int64);school,sok:=s.(int64)
	return actor,school,ao&&so&&aok&&sok&&actor>0&&school>0
}
func academicID(c *gin.Context,key string)(int64,bool){v,e:=strconv.ParseInt(c.Param(key),10,64);return v,e==nil&&v>0}
func academicError(c *gin.Context,err error,resource string){
	switch{
	case errors.Is(err,repository.ErrAcademicAccessDenied):c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Academic operation is not authorized"})
	case errors.Is(err,repository.ErrAcademicNotFound):c.JSON(http.StatusNotFound,gin.H{"status":"error","message":resource+" not found"})
	case errors.Is(err,repository.ErrAcademicConflict):c.JSON(http.StatusConflict,gin.H{"status":"error","message":resource+" conflicts with an existing record"})
	case errors.Is(err,repository.ErrInvalidAcademicInput):c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid academic data"})
	default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process "+strings.ToLower(resource)+" request"})
	}
}
func(h *AcademicHandler) ready(c *gin.Context)(int64,int64,bool){
	actor,school,ok:=academicActor(c);if !ok{c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"School staff authentication required"});return 0,0,false}
	if h==nil||h.service==nil{c.JSON(http.StatusServiceUnavailable,gin.H{"status":"error","message":"Academic management is unavailable"});return 0,0,false}
	return actor,school,true
}

func(h *AcademicHandler) ListAcademicYears(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};items,err:=h.service.ListAcademicYears(c.Request.Context(),a,s);if err!=nil{academicError(c,err,"Academic year");return};c.JSON(http.StatusOK,gin.H{"status":"success","academic_years":items})
}
func(h *AcademicHandler) SaveAcademicYear(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64
	if c.Param("yearID")!=""{var valid bool;id,valid=academicID(c,"yearID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid academic year ID"});return}}
	var req struct{Name string `json:"name"`;StartsOn string `json:"starts_on"`;EndsOn string `json:"ends_on"`;Status string `json:"status"`;IsCurrent bool `json:"is_current"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	start,e1:=time.Parse("2006-01-02",req.StartsOn);end,e2:=time.Parse("2006-01-02",req.EndsOn)
	if e1!=nil||e2!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Dates must use YYYY-MM-DD"});return}
	v,err:=h.service.SaveAcademicYear(c.Request.Context(),a,s,id,req.Name,start,end,req.Status,req.IsCurrent);if err!=nil{academicError(c,err,"Academic year");return}
	code:=http.StatusOK;if id==0{code=http.StatusCreated};c.JSON(code,gin.H{"status":"success","academic_year":v})
}
func(h *AcademicHandler) ListClasses(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};items,err:=h.service.ListClasses(c.Request.Context(),a,s);if err!=nil{academicError(c,err,"Class");return};c.JSON(http.StatusOK,gin.H{"status":"success","classes":items})
}
func(h *AcademicHandler) SaveClass(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64;if c.Param("classID")!=""{var valid bool;id,valid=academicID(c,"classID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid class ID"});return}}
	var req struct{Name string `json:"name"`;DisplayOrder int `json:"display_order"`;IsActive *bool `json:"is_active"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return};active:=true;if req.IsActive!=nil{active=*req.IsActive}
	v,err:=h.service.SaveClass(c.Request.Context(),a,s,id,req.Name,req.DisplayOrder,active);if err!=nil{academicError(c,err,"Class");return};code:=http.StatusOK;if id==0{code=http.StatusCreated};c.JSON(code,gin.H{"status":"success","class":v})
}
func(h *AcademicHandler) ListSections(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};yearID:=int64(0);if c.Query("academic_year_id")!=""{v,e:=strconv.ParseInt(c.Query("academic_year_id"),10,64);if e!=nil||v<=0{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid academic_year_id"});return};yearID=v}
	items,err:=h.service.ListSections(c.Request.Context(),a,s,yearID);if err!=nil{academicError(c,err,"Section");return};c.JSON(http.StatusOK,gin.H{"status":"success","sections":items})
}
func(h *AcademicHandler) SaveSection(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64;if c.Param("sectionID")!=""{var valid bool;id,valid=academicID(c,"sectionID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid section ID"});return}}
	var req struct{AcademicYearID int64 `json:"academic_year_id"`;ClassID int64 `json:"class_id"`;Name string `json:"name"`;DisplayOrder int `json:"display_order"`;IsActive *bool `json:"is_active"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return};active:=true;if req.IsActive!=nil{active=*req.IsActive}
	v,err:=h.service.SaveSection(c.Request.Context(),a,s,id,req.AcademicYearID,req.ClassID,req.Name,req.DisplayOrder,active);if err!=nil{academicError(c,err,"Section");return};code:=http.StatusOK;if id==0{code=http.StatusCreated};c.JSON(code,gin.H{"status":"success","section":v})
}
func(h *AcademicHandler) ListSubjects(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};items,err:=h.service.ListSubjects(c.Request.Context(),a,s);if err!=nil{academicError(c,err,"Subject");return};c.JSON(http.StatusOK,gin.H{"status":"success","subjects":items})
}
func(h *AcademicHandler) SaveSubject(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var id int64;if c.Param("subjectID")!=""{var valid bool;id,valid=academicID(c,"subjectID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid subject ID"});return}}
	var req struct{Name string `json:"name"`;Code *string `json:"subject_code"`;IsActive *bool `json:"is_active"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return};active:=true;if req.IsActive!=nil{active=*req.IsActive}
	v,err:=h.service.SaveSubject(c.Request.Context(),a,s,id,req.Name,req.Code,active);if err!=nil{academicError(c,err,"Subject");return};code:=http.StatusOK;if id==0{code=http.StatusCreated};c.JSON(code,gin.H{"status":"success","subject":v})
}
