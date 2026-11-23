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
	ListQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error)
	CreateQualification(context.Context,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	ListCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error)
	CreateCertification(context.Context,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	ListExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error)
	CreateExperience(context.Context,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	UpdateQualification(context.Context,int64,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	UpdateCertification(context.Context,int64,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	UpdateExperience(context.Context,int64,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	ListAuditEvents(context.Context,int64,int64,int64)([]repository.StaffAuditEvent,error)
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
		c.JSON(http.StatusBadRequest,gin.H{"status":"error","message":"Check the highlighted staff record fields","field_errors":gin.H{}})
	default:c.JSON(http.StatusInternalServerError,gin.H{"status":"error","message":"Unable to process staff record"})
	}
}
func(h *StaffRecordsHandler) ListQualifications(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListQualifications(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","qualifications":items})
}
func(h *StaffRecordsHandler) CreateQualification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffQualification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification request"});return}
	saved,err:=h.service.CreateQualification(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	c.JSON(201,gin.H{"status":"success","qualification":saved})
}
func(h *StaffRecordsHandler) ListCertifications(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListCertifications(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","certifications":items})
}
func(h *StaffRecordsHandler) CreateCertification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffCertification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid certification request"});return}
	saved,err:=h.service.CreateCertification(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	c.JSON(201,gin.H{"status":"success","certification":saved})
}
func(h *StaffRecordsHandler) ListExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListExperience(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","experience":items})
}
func(h *StaffRecordsHandler) CreateExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	var item repository.StaffExperience;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid experience request"});return}
	saved,err:=h.service.CreateExperience(c.Request.Context(),actor,school,staff,item);if err!=nil{h.fail(c,err);return}
	c.JSON(201,gin.H{"status":"success","experience":saved})
}


func staffRecordID(c *gin.Context)(int64,bool){v,e:=strconv.ParseInt(c.Param("recordID"),10,64);return v,e==nil&&v>0}

func(h *StaffRecordsHandler) UpdateQualification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification record ID"});return}
	var item repository.StaffQualification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid qualification request"});return}
	saved,err:=h.service.UpdateQualification(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","qualification":saved})
}
func(h *StaffRecordsHandler) UpdateCertification(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid certification record ID"});return}
	var item repository.StaffCertification;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid certification request"});return}
	saved,err:=h.service.UpdateCertification(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","certification":saved})
}
func(h *StaffRecordsHandler) UpdateExperience(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	record,ok:=staffRecordID(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid experience record ID"});return}
	var item repository.StaffExperience;if err:=c.ShouldBindJSON(&item);err!=nil{c.JSON(400,gin.H{"status":"error","message":"Invalid experience request"});return}
	saved,err:=h.service.UpdateExperience(c.Request.Context(),actor,school,staff,record,item);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","experience":saved})
}

func(h *StaffRecordsHandler) ListAuditEvents(c *gin.Context){
	actor,school,ok:=staffRecordActor(c);if !ok{c.JSON(401,gin.H{"status":"error","message":"School staff authentication required"});return}
	staff,ok:=staffRecordTarget(c);if !ok{c.JSON(400,gin.H{"status":"error","message":"Invalid staff principal ID"});return}
	items,err:=h.service.ListAuditEvents(c.Request.Context(),actor,school,staff);if err!=nil{h.fail(c,err);return}
	c.JSON(200,gin.H{"status":"success","events":items})
}

func (h *StaffRecordsHandler) ListQualificationsReport(c *gin.Context) {
	actor, school, ok := staffRecordActor(c)
	if !ok { c.JSON(http.StatusUnauthorized, gin.H{"status":"error","message":"School staff authentication required"}); return }
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
	page, err := h.service.ListQualificationsReport(c.Request.Context(),actor,school,filter)
	if err != nil { h.fail(c,err); return }
	c.JSON(http.StatusOK,gin.H{"status":"success","report":page})
}
