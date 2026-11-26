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
	"school-management-system/internal/services"
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

func (nilStaffRecordsResultService) ListQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error){return nil,nil}
func (nilStaffRecordsResultService) ListCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error){return nil,nil}
func (nilStaffRecordsResultService) ListExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error){return nil,nil}
func (nilStaffRecordsResultService) ListAuditEvents(context.Context,int64,int64,int64,repository.StaffAuditEventFilter)(*repository.StaffAuditEventPage,error){return nil,nil}

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
		router:=gin.New();routePath:=strings.Replace(test.path,"/42","/:principalID",1);routePath=strings.Replace(routePath,"/7","/:recordID",1);router.Handle(test.method,routePath,auth,func(c *gin.Context){test.handler(handler,c)})
		req:=httptest.NewRequest(test.method,test.path,strings.NewReader(test.body));if test.body!=""{req.Header.Set("Content-Type","application/json")}
		recorder:=httptest.NewRecorder();router.ServeHTTP(recorder,req)
		if recorder.Code!=http.StatusInternalServerError{t.Fatalf("nil successful service result must return 500: status=%d body=%s",recorder.Code,recorder.Body.String())}
	})}
}


func TestStaffRecordsListHandlersReturnEmptyArraysForNilResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewStaffRecordsHandler(nilStaffRecordsResultService{})
	tests := []struct {
		name, path, key string
		handler func(*StaffRecordsHandler, *gin.Context)
	}{
		{"qualifications", "/staff/42/qualifications", "qualifications", (*StaffRecordsHandler).ListQualifications},
		{"certifications", "/staff/42/certifications", "certifications", (*StaffRecordsHandler).ListCertifications},
		{"experience", "/staff/42/experience", "experience", (*StaffRecordsHandler).ListExperience},
		{"audit events", "/staff/42/history", "events", (*StaffRecordsHandler).ListAuditEvents},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/staff/:principalID/:resource", func(c *gin.Context) {
				c.Set(middleware.ContextPrincipalIDKey, int64(100))
				c.Set(middleware.ContextSchoolIDKey, int64(10))
				tt.handler(handler, c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"`+tt.key+`":[]`) {
				t.Fatalf("nil list must serialize as an empty JSON array for %q: %s", tt.key, recorder.Body.String())
			}
		})
	}
}

func TestStaffRecordsHandlerReturnsStructuredFieldValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewStaffRecordsHandler(nil)
	router.POST("/validation", func(c *gin.Context) {
		handler.fail(c, &services.StaffRecordFieldValidationError{
			FieldErrors: map[string]string{"qualification_name": "Enter a valid qualification name."},
		})
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/validation", strings.NewReader("{}")))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"qualification_name":"Enter a valid qualification name."`) {
		t.Fatalf("field error missing from response: %s", recorder.Body.String())
	}
}

func TestStaffAuditHandlerRejectsInvalidPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewStaffRecordsHandler(nilStaffRecordsResultService{})
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(100))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		c.Next()
	}
	for _, query := range []string{"limit=abc", "limit=0", "limit=101", "offset=abc", "offset=-1", "offset=1000001"} {
		t.Run(query, func(t *testing.T) {
			router := gin.New()
			router.GET("/school/staff/:principalID/audit-events", auth, handler.ListAuditEvents)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/school/staff/42/audit-events?"+query, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("invalid audit pagination %q status=%d want=%d body=%s", query, recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
		})
	}
}

func TestStaffReportHandlersRejectMalformedPaginationAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewStaffRecordsHandler(nilStaffRecordsResultService{})
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(100))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		c.Next()
	}
	tests := []struct {
		name, path string
		handler func(*StaffRecordsHandler, *gin.Context)
	}{
		{"qualifications malformed limit", "/staff/qualifications/report?limit=abc", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications malformed offset", "/staff/qualifications/report?offset=abc", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications limit too small", "/staff/qualifications/report?limit=0", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications limit too large", "/staff/qualifications/report?limit=201", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications offset too large", "/staff/qualifications/report?offset=1000001", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications invalid record type", "/staff/qualifications/report?record_type=payroll", (*StaffRecordsHandler).ListQualificationsReport},
		{"qualifications invalid verification status", "/staff/qualifications/report?verification_status=approved", (*StaffRecordsHandler).ListQualificationsReport},
		{"experience malformed limit", "/staff/experience/report?limit=abc", (*StaffRecordsHandler).ListExperienceReport},
		{"experience malformed offset", "/staff/experience/report?offset=abc", (*StaffRecordsHandler).ListExperienceReport},
		{"experience limit too large", "/staff/experience/report?limit=201", (*StaffRecordsHandler).ListExperienceReport},
		{"experience offset too large", "/staff/experience/report?offset=1000001", (*StaffRecordsHandler).ListExperienceReport},
		{"experience invalid employment type", "/staff/experience/report?employment_type=casual", (*StaffRecordsHandler).ListExperienceReport},
		{"experience invalid current-only", "/staff/experience/report?current_only=yes", (*StaffRecordsHandler).ListExperienceReport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/staff/qualifications/report", auth, func(c *gin.Context) { tt.handler(handler, c) })
			router.GET("/staff/experience/report", auth, func(c *gin.Context) { tt.handler(handler, c) })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("malformed report query must return 400: status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}


type captureStaffReportFiltersService struct {
	StaffRecordsManagementService
	qualificationFilter repository.StaffQualificationReportFilter
	experienceFilter repository.StaffExperienceReportFilter
}

func (s *captureStaffReportFiltersService) ListQualificationsReport(_ context.Context, _, _ int64, filter repository.StaffQualificationReportFilter) (*repository.StaffQualificationReportPage, error) {
	s.qualificationFilter = filter
	return &repository.StaffQualificationReportPage{Rows: []repository.StaffQualificationReportRow{}, Counts: []repository.StaffQualificationReportCount{}, Total: 0, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (s *captureStaffReportFiltersService) ListExperienceReport(_ context.Context, _, _ int64, filter repository.StaffExperienceReportFilter) (*repository.StaffExperienceReportPage, error) {
	s.experienceFilter = filter
	return &repository.StaffExperienceReportPage{Rows: []repository.StaffExperienceReportRow{}, Counts: []repository.StaffExperienceReportCount{}, Total: 0, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func TestStaffReportHandlersForwardValidFiltersAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffReportFiltersService{}
	handler := NewStaffRecordsHandler(service)
	auth := func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(100))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		c.Next()
	}
	tests := []struct {
		name, path string
		handler func(*StaffRecordsHandler, *gin.Context)
	}{
		{"qualifications", "/staff/qualifications/report?record_type=certification&verification_status=pending&limit=17&offset=34", (*StaffRecordsHandler).ListQualificationsReport},
		{"experience", "/staff/experience/report?employment_type=contract&current_only=true&limit=23&offset=46", (*StaffRecordsHandler).ListExperienceReport},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/staff/qualifications/report", auth, func(c *gin.Context) { tt.handler(handler, c) })
			router.GET("/staff/experience/report", auth, func(c *gin.Context) { tt.handler(handler, c) })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("valid report query status=%d want=%d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), `"total":0`) {
				t.Fatalf("report response omitted total count: %s", recorder.Body.String())
			}
		})
	}
	if got := service.qualificationFilter; got.RecordType != "certification" || got.VerificationStatus != "pending" || got.Limit != 17 || got.Offset != 34 {
		t.Fatalf("qualification report filter=%+v want record_type=certification verification_status=pending limit=17 offset=34", got)
	}
	if got := service.experienceFilter; got.EmploymentType != "contract" || !got.CurrentOnly || got.Limit != 23 || got.Offset != 46 {
		t.Fatalf("experience report filter=%+v want employment_type=contract current_only=true limit=23 offset=46", got)
	}
}
