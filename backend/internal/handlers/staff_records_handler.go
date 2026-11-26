package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type StaffRecordsManagementService interface {
	ListQualificationsReport(context.Context,int64,int64,repository.StaffQualificationReportFilter)(*repository.StaffQualificationReportPage,error)
	ListExperienceReport(context.Context,int64,int64,repository.StaffExperienceReportFilter)(*repository.StaffExperienceReportPage,error)
	ListQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error)
	CreateQualification(context.Context,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	ListCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error)
	CreateCertification(context.Context,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	ListExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error)
	CreateExperience(context.Context,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	UpdateQualification(context.Context,int64,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	UpdateCertification(context.Context,int64,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	UpdateExperience(context.Context,int64,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	ListAuditEvents(context.Context,int64,int64,int64,repository.StaffAuditEventFilter)(*repository.StaffAuditEventPage,error)
}

type StaffRecordsHandler struct { service StaffRecordsManagementService }
func NewStaffRecordsHandler(service StaffRecordsManagementService)*StaffRecordsHandler{return &StaffRecordsHandler{service:service}}

func staffRecordActor(c *gin.Context)(int64,int64,bool){
	a,ao:=c.Get(middleware.ContextPrincipalIDKey);s,so:=c.Get(middleware.ContextSchoolIDKey)
	actor,aok:=a.(int64);school,sok:=s.(int64)
	return actor,school,ao&&so&&aok&&sok&&actor>0&&school>0
}
func staffRecordTarget(c *gin.Context)(int64,bool){v,e:=strconv.ParseInt(c.Param("principalID"),10,64);return v,e==nil&&v>0}
func(h *StaffRecordsHandler) fail(c *gin.Context,err error){
	switch{
	case errors.Is(err,repository.ErrStaffRecordAccessDenied),errors.Is(err,repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden,gin.H{"status":"error","message":"Staff record access is not authorized"})
	case errors.Is(err,services.ErrInvalidStaffRecord),errors.Is(err,repository.ErrInvalidStaffRecord):
		var validation *services.StaffRecordFieldValidationError
		if errors.As(err, &validation) && len(validation.FieldErrors) > 0 {
			c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Check the highlighted staff record fields","field_errors":validation.FieldErrors})
			return
		}
		c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Check the submitted staff record","field_errors":gin.H{}})
	default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process staff record"})
	}
}
func staffRecordsServiceUnavailable(c *gin.Context) bool {
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff records service is unavailable"})
	return true
}
func (h *StaffRecordsHandler) unavailable(c *gin.Context) bool {
	if h == nil || h.service == nil { return staffRecordsServiceUnavailable(c) }
	return false
}

func(h *StaffRecordsHandler) ListQualifications(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListQualifications(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	if items == nil { items = []repository.StaffQualification{} }
	c.JSON(200,gin.H{"status":"success","qualifications":items})
}
func(h *StaffRecordsHandler) CreateQualification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffQualification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification request"});return}
	saved,err:=h.service.CreateQualification(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to save staff qualification"}); return }
	c.JSON(201,gin.H{"status":"success","qualification":saved})
}
func(h *StaffRecordsHandler) ListCertifications(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListCertifications(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	if items == nil { items = []repository.StaffCertification{} }
	c.JSON(200,gin.H{"status":"success","certifications":items})
}
func(h *StaffRecordsHandler) CreateCertification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffCertification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid certification request"});return}
	saved,err:=h.service.CreateCertification(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to save staff certification"}); return }
	c.JSON(201,gin.H{"status":"success","certification":saved})
}
func(h *StaffRecordsHandler) ListExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListExperience(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	if items == nil { items = []repository.StaffExperience{} }
	c.JSON(200,gin.H{"status":"success","experience":items})
}
func(h *StaffRecordsHandler) CreateExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffExperience;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid experience request"});return}
	saved,err:=h.service.CreateExperience(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to save staff experience"}); return }
	c.JSON(201,gin.H{"status":"success","experience":saved})
}


func staffRecordID(c *gin.Context)(int64,bool){v,e:=strconv.ParseInt(c.Param("recordID"),10,64);return v,e==nil&&v>0}

func(h *StaffRecordsHandler) UpdateQualification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification record ID"});return}
	var item repository.StaffQualification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification request"});return}
	saved,err:=h.service.UpdateQualification(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to update staff qualification"}); return }
	c.JSON(200,gin.H{"status":"success","qualification":saved})
}
func(h *StaffRecordsHandler) UpdateCertification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid certification record ID"});return}
	var item repository.StaffCertification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid certification request"});return}
	saved,err:=h.service.UpdateCertification(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to update staff certification"}); return }
	c.JSON(200,gin.H{"status":"success","certification":saved})
}
func(h *StaffRecordsHandler) UpdateExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid experience record ID"});return}
	var item repository.StaffExperience;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid experience request"});return}
	saved,err:=h.service.UpdateExperience(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	if saved == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to update staff experience"}); return }
	c.JSON(200,gin.H{"status":"success","experience":saved})
}

func(h *StaffRecordsHandler) ListAuditEvents(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	if h.unavailable(c) { return }
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	filter:=repository.StaffAuditEventFilter{Limit:50,Offset:0}
	if raw:=c.Query("limit");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil||value<1||value>100 { c.JSON(400,gin.H{"status":"error","message":"Invalid audit event limit"});return };filter.Limit=value }
	if raw:=c.Query("offset");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil||value<0||value>1000000 { c.JSON(400,gin.H{"status":"error","message":"Invalid audit event offset"});return };filter.Offset=value }
	page,err:=h.service.ListAuditEvents(c.Request.Context(),actor,school,staff,filter);if err!=nil{h.fail(c,err);return}
	if page==nil { page=&repository.StaffAuditEventPage{Events:[]repository.StaffAuditEvent{},Limit:filter.Limit,Offset:filter.Offset} }
	if page.Events==nil { page.Events=[]repository.StaffAuditEvent{} }
	c.JSON(200,gin.H{"status":"success","events":page.Events,"limit":page.Limit,"offset":page.Offset,"has_more":page.HasMore})
}

func (h *StaffRecordsHandler) ListQualificationsReport(c *gin.Context) {
	actor, school, ok := staffRecordActor(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
	if h.unavailable(c) { return }
	filter := repository.StaffQualificationReportFilter{
		RecordType: c.DefaultQuery("record_type","all"),
		VerificationStatus: c.Query("verification_status"),
		Limit: 50,
		Offset: 0,
	}
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil { c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid report limit"}); return }
		filter.Limit = value
	}
	if raw := c.Query("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil { c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid report offset"}); return }
		filter.Offset = value
	}
	if filter.Limit < 1 || filter.Limit > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Report limit must be between 1 and 200"})
		return
	}
	if filter.Offset < 0 || filter.Offset > 1000000 {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Report offset must be between 0 and 1000000"})
		return
	}
	if filter.RecordType != "all" && filter.RecordType != "qualification" && filter.RecordType != "certification" {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid qualification report record type"})
		return
	}
	if filter.VerificationStatus != "" && filter.VerificationStatus != "pending" && filter.VerificationStatus != "verified" && filter.VerificationStatus != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid verification status"})
		return
	}
	page, err := h.service.ListQualificationsReport(c.Request.Context(),actor,school,filter)
	if err != nil { h.fail(c,err); return }
	if page == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to load staff qualifications report"}); return }
	if page.Rows == nil { page.Rows = []repository.StaffQualificationReportRow{} }
	if page.Counts == nil { page.Counts = []repository.StaffQualificationReportCount{} }
	c.JSON(http.StatusOK,gin.H{"status":"success","report":page})
}

func (h *StaffRecordsHandler) ListExperienceReport(c *gin.Context) {
	actor,school,ok:=staffRecordActor(c)
    if !ok { c.JSON(http.StatusUnauthorized,gin.H{"status":"error","message":"School staff authentication required"});return }
    if h.unavailable(c) { return }
    currentOnly:=c.Query("current_only")
    if currentOnly!="" && currentOnly!="true" && currentOnly!="false" {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid current-only filter"});return}
    filter:=repository.StaffExperienceReportFilter{EmploymentType:c.Query("employment_type"),CurrentOnly:currentOnly=="true",Limit:50}
    if raw:=c.Query("limit");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid report limit"});return};filter.Limit=value }
    if raw:=c.Query("offset");raw!="" { value,err:=strconv.Atoi(raw);if err!=nil {c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Invalid report offset"});return};filter.Offset=value }
    if filter.Limit < 1 || filter.Limit > 200 {
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Report limit must be between 1 and 200"}); return
    }
    if filter.Offset < 0 || filter.Offset > 1000000 {
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Report offset must be between 0 and 1000000"}); return
    }
    switch filter.EmploymentType {
    case "", "permanent", "contract", "part_time", "temporary", "internship", "other":
    default:
        c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"Invalid employment type"}); return
    }
    page,err:=h.service.ListExperienceReport(c.Request.Context(),actor,school,filter)
    if err!=nil {h.fail(c,err);return}
    if page == nil { c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to load staff experience report"}); return }
    if page.Rows == nil { page.Rows = []repository.StaffExperienceReportRow{} }
    if page.Counts == nil { page.Counts = []repository.StaffExperienceReportCount{} }
    c.JSON(http.StatusOK,gin.H{"status":"success","report":page})
}
