package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
)

func TestStaffRecordsHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name string
		handler func(*StaffRecordsHandler, *gin.Context)
	}{
		{"list qualifications", (*StaffRecordsHandler).ListQualifications},
		{"create qualification", (*StaffRecordsHandler).CreateQualification},
		{"list certifications", (*StaffRecordsHandler).ListCertifications},
		{"create certification", (*StaffRecordsHandler).CreateCertification},
		{"list experience", (*StaffRecordsHandler).ListExperience},
		{"create experience", (*StaffRecordsHandler).CreateExperience},
		{"update qualification", (*StaffRecordsHandler).UpdateQualification},
		{"update certification", (*StaffRecordsHandler).UpdateCertification},
		{"update experience", (*StaffRecordsHandler).UpdateExperience},
		{"list audit events", (*StaffRecordsHandler).ListAuditEvents},
		{"qualifications report", (*StaffRecordsHandler).ListQualificationsReport},
		{"experience report", (*StaffRecordsHandler).ListExperienceReport},
	}
	for _, method := range methods {
		t.Run(method.name+" nil receiver", func(t *testing.T) {
			router := gin.New()
			router.Any("/staff/:principalID/records/:recordID", func(c *gin.Context) {
                c.Set(middleware.ContextPrincipalIDKey, int64(100))
                c.Set(middleware.ContextSchoolIDKey, int64(10))
				method.handler(nil, c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/42/records/7", nil))
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
		})
		t.Run(method.name+" nil service", func(t *testing.T) {
			handler := NewStaffRecordsHandler(nil)
			router := gin.New()
			router.Any("/staff/:principalID/records/:recordID", func(c *gin.Context) {
                c.Set(middleware.ContextPrincipalIDKey, int64(100))
                c.Set(middleware.ContextSchoolIDKey, int64(10))
				method.handler(handler, c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/42/records/7", nil))
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
		})
	}
}


func TestStaffRecordsUnauthenticatedRequestDoesNotExposeServiceAvailability(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/staff/:principalID/records", NewStaffRecordsHandler(nil).ListQualifications)
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/42/records", nil))
    if recorder.Code != http.StatusUnauthorized {
        t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
    }
}


type nilStaffRecordsResultService struct { StaffRecordsManagementService }

func (nilStaffRecordsResultService) CreateQualification(context.Context,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error){return nil,nil}
func (nilStaffRecordsResultService) CreateCertification(context.Context,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error){return nil,nil}
func (nilStaffRecordsResultService) CreateExperience(context.Context,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error){return nil,nil}
func (nilStaffRecordsResultService) UpdateQualification(context.Context,int64,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error){return nil,nil}
func (nilStaffRecordsResultService) UpdateCertification(context.Context,int64,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error){return nil,nil}
func (nilStaffRecordsResultService) UpdateExperience(context.Context,int64,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error){return nil,nil}
func (nilStaffRecordsResultService) ListQualificationsReport(context.Context,int64,int64,repository.StaffQualificationReportFilter)(*repository.StaffQualificationReportPage,error){return nil,nil}
func (nilStaffRecordsResultService) ListExperienceReport(context.Context,int64,int64,repository.StaffExperienceReportFilter)(*repository.StaffExperienceReportPage,error){return nil,nil}

func TestStaffRecordsHandlersRejectNilSuccessfulResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler:=NewStaffRecordsHandler(nilStaffRecordsResultService{})
	auth:=func(c *gin.Context){c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()}
	tests:=[]struct{name,method,path,body string; handler func(*StaffRecordsHandler,*gin.Context)}{
		{"create qualification",http.MethodPost,"/staff/42/qualifications",`{}`,(*StaffRecordsHandler).CreateQualification},
		{"create certification",http.MethodPost,"/staff/42/certifications",`{}`,(*StaffRecordsHandler).CreateCertification},
		{"create experience",http.MethodPost,"/staff/42/experience",`{}`,(*StaffRecordsHandler).CreateExperience},
		{"update qualification",http.MethodPut,"/staff/42/qualifications/7",`{}`,(*StaffRecordsHandler).UpdateQualification},
		{"update certification",http.MethodPut,"/staff/42/certifications/7",`{}`,(*StaffRecordsHandler).UpdateCertification},
		{"update experience",http.MethodPut,"/staff/42/experience/7",`{}`,(*StaffRecordsHandler).UpdateExperience},
		{"qualifications report",http.MethodGet,"/staff/qualifications/report","",(*StaffRecordsHandler).ListQualificationsReport},
		{"experience report",http.MethodGet,"/staff/experience/report","",(*StaffRecordsHandler).ListExperienceReport},
	}
	for _,test:=range tests{t.Run(test.name,func(t *testing.T){
		router:=gin.New();router.Handle(test.method,test.path,auth,func(c *gin.Context){test.handler(handler,c)})
		req:=httptest.NewRequest(test.method,test.path,strings.NewReader(test.body));if test.body!=""{req.Header.Set("Content-Type","application/json")}
		recorder:=httptest.NewRecorder();router.ServeHTTP(recorder,req)
		if recorder.Code!=http.StatusInternalServerError{t.Fatalf("nil successful service result must return 500: status=%d body=%s",recorder.Code,recorder.Body.String())}
	})}
}
