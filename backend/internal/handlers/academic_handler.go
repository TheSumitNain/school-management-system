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
	ListEligibleTeachers(context.Context,int64,int64)([]repository.EligibleTeacher,error)
	ListTeacherSubjectAssignments(context.Context,int64,int64,int64,int64)([]repository.TeacherSubjectAssignment,error)
	CreateTeacherSubjectAssignment(context.Context,int64,int64,int64,int64,int64,int64)(*repository.TeacherSubjectAssignment,error)
	RevokeTeacherSubjectAssignment(context.Context,int64,int64,int64)error
	ListClassInchargeAssignments(context.Context,int64,int64,int64)([]repository.ClassInchargeAssignment,error)
	CreateClassInchargeAssignment(context.Context,int64,int64,int64,int64,int64)(*repository.ClassInchargeAssignment,error)
	RevokeClassInchargeAssignment(context.Context,int64,int64,int64)error
	ListStudentEnrollments(context.Context,int64,int64,int64,int64)([]repository.StudentEnrollment,error)
	CreateStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,string,time.Time,time.Time)(*repository.StudentEnrollment,error)
	PromoteStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,string,time.Time,time.Time)(*repository.StudentEnrollment,error)
	CreateStudentAdmission(context.Context,int64,int64,string,string,int64,int64,*string,time.Time)(*repository.StudentAdmissionResult,error)
	ListStudents(context.Context,int64,int64,string,string,int,int)([]repository.StudentProfile,int64,error)
	GetStudent(context.Context,int64,int64,int64)(*repository.StudentProfile,error)
	CreateStudent(context.Context,int64,int64,string,string)(*repository.StudentProfile,error)
	UpdateStudent(context.Context,int64,int64,int64,string)(*repository.StudentProfile,error)
	ArchiveStudent(context.Context,int64,int64,int64)(*repository.StudentProfile,error)
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
	var req struct{AcademicYearID int64 `json:"academic_year_id"`;ClassID int64 `json:"class_id"`;Name string `json:"name"`;Medium *string `json:"medium"`;DisplayOrder int `json:"display_order"`;IsActive *bool `json:"is_active"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return};active:=true;if req.IsActive!=nil{active=*req.IsActive}
	var v *repository.AcademicSection
	var err error
	if saver,ok:=h.service.(interface{SaveSectionWithMedium(context.Context,int64,int64,int64,int64,int64,string,*string,int,bool)(*repository.AcademicSection,error)});ok{
		v,err=saver.SaveSectionWithMedium(c.Request.Context(),a,s,id,req.AcademicYearID,req.ClassID,req.Name,req.Medium,req.DisplayOrder,active)
	}else if req.Medium==nil||strings.TrimSpace(*req.Medium)==""{
		v,err=h.service.SaveSection(c.Request.Context(),a,s,id,req.AcademicYearID,req.ClassID,req.Name,req.DisplayOrder,active)
	}else{err=repository.ErrInvalidAcademicInput}
	if err!=nil{academicError(c,err,"Section");return};code:=http.StatusOK;if id==0{code=http.StatusCreated};c.JSON(code,gin.H{"status":"success","section":v})
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


func queryPositiveID(c *gin.Context,key string)(int64,bool){
	raw:=c.Query(key);if raw==""{return 0,true};v,err:=strconv.ParseInt(raw,10,64);return v,err==nil&&v>0
}

func(h *AcademicHandler) ListEligibleTeachers(c *gin.Context) {
	a, s, ok := h.ready(c)
	if !ok { return }
	items, err := h.service.ListEligibleTeachers(c.Request.Context(), a, s)
	if err != nil { academicError(c, err, "Teacher"); return }
	c.JSON(http.StatusOK, gin.H{"status": "success", "teachers": items})
}

func(h *AcademicHandler) ListTeacherSubjectAssignments(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return}
	yearID,valid:=queryPositiveID(c,"academic_year_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid academic_year_id"});return}
	sectionID,valid:=queryPositiveID(c,"section_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid section_id"});return}
	items,err:=h.service.ListTeacherSubjectAssignments(c.Request.Context(),a,s,yearID,sectionID);if err!=nil{academicError(c,err,"Teacher assignment");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","assignments":items})
}
func(h *AcademicHandler) CreateTeacherSubjectAssignment(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return}
	var req struct{AcademicYearID int64 `json:"academic_year_id"`;SectionID int64 `json:"section_id"`;SubjectID int64 `json:"subject_id"`;TeacherPrincipalID int64 `json:"teacher_principal_id"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	item,err:=h.service.CreateTeacherSubjectAssignment(c.Request.Context(),a,s,req.AcademicYearID,req.SectionID,req.SubjectID,req.TeacherPrincipalID)
	if err!=nil{academicError(c,err,"Teacher assignment");return}
	c.JSON(http.StatusCreated,gin.H{"status":"success","assignment":item})
}
func(h *AcademicHandler) RevokeTeacherSubjectAssignment(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id,valid:=academicID(c,"assignmentID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid assignment ID"});return}
	if err:=h.service.RevokeTeacherSubjectAssignment(c.Request.Context(),a,s,id);err!=nil{academicError(c,err,"Teacher assignment");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Teacher assignment revoked"})
}
func(h *AcademicHandler) ListClassInchargeAssignments(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};yearID,valid:=queryPositiveID(c,"academic_year_id");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid academic_year_id"});return}
	items,err:=h.service.ListClassInchargeAssignments(c.Request.Context(),a,s,yearID);if err!=nil{academicError(c,err,"Class in-charge assignment");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","assignments":items})
}
func(h *AcademicHandler) CreateClassInchargeAssignment(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};var req struct{AcademicYearID int64 `json:"academic_year_id"`;SectionID int64 `json:"section_id"`;TeacherPrincipalID int64 `json:"teacher_principal_id"`}
	if c.ShouldBindJSON(&req)!=nil{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid request body"});return}
	item,err:=h.service.CreateClassInchargeAssignment(c.Request.Context(),a,s,req.AcademicYearID,req.SectionID,req.TeacherPrincipalID);if err!=nil{academicError(c,err,"Class in-charge assignment");return}
	c.JSON(http.StatusCreated,gin.H{"status":"success","assignment":item})
}
func(h *AcademicHandler) RevokeClassInchargeAssignment(c *gin.Context){
	a,s,ok:=h.ready(c);if !ok{return};id,valid:=academicID(c,"assignmentID");if !valid{c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid assignment ID"});return}
	if err:=h.service.RevokeClassInchargeAssignment(c.Request.Context(),a,s,id);err!=nil{academicError(c,err,"Class in-charge assignment");return}
	c.JSON(http.StatusOK,gin.H{"status":"success","message":"Class in-charge assignment revoked"})
}

func parseEnrollmentDate(raw string) (time.Time, error) {
	value, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil { return time.Time{}, repository.ErrInvalidAcademicInput }
	return value, nil
}

func (h *AcademicHandler) ListStudentEnrollments(c *gin.Context) {
	actor, school, ok := h.ready(c); if !ok { return }
	yearID, valid := queryPositiveID(c, "academic_year_id"); if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid academic_year_id"}); return }
	items, err := h.service.ListStudentEnrollments(c.Request.Context(), actor, school, yearID, 0)
	if err != nil { academicError(c, err, "Student enrollment"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","enrollments":items})
}

func (h *AcademicHandler) ListStudentEnrollmentHistory(c *gin.Context) {
	actor, school, ok := h.ready(c); if !ok { return }
	studentID, valid := academicID(c, "studentID"); if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid student ID"}); return }
	items, err := h.service.ListStudentEnrollments(c.Request.Context(), actor, school, 0, studentID)
	if err != nil { academicError(c, err, "Student enrollment"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","enrollments":items})
}

func (h *AcademicHandler) CreateStudentAdmission(c *gin.Context) {
	actor, school, ok := h.ready(c); if !ok { return }
	var req struct {
		FullName string `json:"full_name"`
		AdmissionNumber string `json:"admission_number"`
		AcademicYearID int64 `json:"academic_year_id"`
		SectionID int64 `json:"section_id"`
		RollNumber *string `json:"roll_number"`
		AdmissionDate string `json:"admission_date"`
	}
	if c.ShouldBindJSON(&req) != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid request body"}); return }
	admissionDate, err := parseEnrollmentDate(req.AdmissionDate)
	if err != nil { academicError(c, err, "Student admission"); return }
	result, err := h.service.CreateStudentAdmission(c.Request.Context(), actor, school,
		req.FullName, req.AdmissionNumber, req.AcademicYearID, req.SectionID, req.RollNumber, admissionDate)
	if err != nil { academicError(c, err, "Student admission"); return }
	c.JSON(http.StatusCreated, gin.H{"status":"success","student":result.Student,"enrollment":result.Enrollment})
}

func (h *AcademicHandler) CreateStudentEnrollment(c *gin.Context) {
	actor, school, ok := h.ready(c); if !ok { return }
	var req struct {
		StudentID int64 `json:"student_id"`
		AcademicYearID int64 `json:"academic_year_id"`
		SectionID int64 `json:"section_id"`
		RollNumber *string `json:"roll_number"`
		AdmissionNumber string `json:"admission_number"`
		AdmissionDate string `json:"admission_date"`
		EnrolledAt string `json:"enrolled_at"`
	}
	if c.ShouldBindJSON(&req) != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid request body"}); return }
	admissionDate, err := parseEnrollmentDate(req.AdmissionDate); if err != nil { academicError(c, err, "Student enrollment"); return }
	enrolledAt, err := parseEnrollmentDate(req.EnrolledAt); if err != nil { academicError(c, err, "Student enrollment"); return }
	item, err := h.service.CreateStudentEnrollment(c.Request.Context(), actor, school, req.StudentID, req.AcademicYearID, req.SectionID, req.RollNumber, req.AdmissionNumber, admissionDate, enrolledAt)
	if err != nil { academicError(c, err, "Student enrollment"); return }
	c.JSON(http.StatusCreated, gin.H{"status":"success","enrollment":item})
}

func (h *AcademicHandler) PromoteStudentEnrollment(c *gin.Context) {
	actor, school, ok := h.ready(c); if !ok { return }
	sourceID, valid := academicID(c, "enrollmentID"); if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid enrollment ID"}); return }
	var req struct {
		AcademicYearID int64 `json:"academic_year_id"`
		SectionID int64 `json:"section_id"`
		RollNumber *string `json:"roll_number"`
		AdmissionNumber string `json:"admission_number"`
		AdmissionDate string `json:"admission_date"`
		EnrolledAt string `json:"enrolled_at"`
	}
	if c.ShouldBindJSON(&req) != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid request body"}); return }
	admissionDate, err := parseEnrollmentDate(req.AdmissionDate); if err != nil { academicError(c, err, "Student promotion"); return }
	enrolledAt, err := parseEnrollmentDate(req.EnrolledAt); if err != nil { academicError(c, err, "Student promotion"); return }
	item, err := h.service.PromoteStudentEnrollment(c.Request.Context(), actor, school, sourceID, req.AcademicYearID, req.SectionID, req.RollNumber, req.AdmissionNumber, admissionDate, enrolledAt)
	if err != nil { academicError(c, err, "Student promotion"); return }
	c.JSON(http.StatusCreated, gin.H{"status":"success","enrollment":item})
}

func (h *AcademicHandler) ListStudents(c *gin.Context) {
	actor, school, ok := h.ready(c)
	if !ok { return }
	limit := 50
	offset := 0
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid limit"}); return }
		limit = value
	}
	if raw := c.Query("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid offset"}); return }
		offset = value
	}
	items, total, err := h.service.ListStudents(c.Request.Context(), actor, school,
		c.Query("q"), c.Query("status"), limit, offset)
	if err != nil { academicError(c, err, "Student"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","students":items,"pagination":gin.H{
		"limit":limit,"offset":offset,"total":total,
	}})
}

func (h *AcademicHandler) GetStudent(c *gin.Context) {
	actor, school, ok := h.ready(c)
	if !ok { return }
	studentID, valid := academicID(c, "studentID")
	if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid student ID"}); return }
	item, err := h.service.GetStudent(c.Request.Context(), actor, school, studentID)
	if err != nil { academicError(c, err, "Student"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","student":item})
}

func (h *AcademicHandler) CreateStudent(c *gin.Context) {
	actor, school, ok := h.ready(c)
	if !ok { return }
	var req struct {
		AdmissionNumber string `json:"admission_number"`
		FullName string `json:"full_name"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid request body"})
		return
	}
	item, err := h.service.CreateStudent(c.Request.Context(), actor, school, req.AdmissionNumber, req.FullName)
	if err != nil { academicError(c, err, "Student"); return }
	c.JSON(http.StatusCreated, gin.H{"status":"success","student":item})
}

func (h *AcademicHandler) UpdateStudent(c *gin.Context) {
	actor, school, ok := h.ready(c)
	if !ok { return }
	studentID, valid := academicID(c, "studentID")
	if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid student ID"}); return }
	var req struct { FullName string `json:"full_name"` }
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid request body"})
		return
	}
	item, err := h.service.UpdateStudent(c.Request.Context(), actor, school, studentID, req.FullName)
	if err != nil { academicError(c, err, "Student"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","student":item})
}

func (h *AcademicHandler) ArchiveStudent(c *gin.Context) {
	actor, school, ok := h.ready(c)
	if !ok { return }
	studentID, valid := academicID(c, "studentID")
	if !valid { c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid student ID"}); return }
	item, err := h.service.ArchiveStudent(c.Request.Context(), actor, school, studentID)
	if err != nil { academicError(c, err, "Student"); return }
	c.JSON(http.StatusOK, gin.H{"status":"success","student":item})
}
