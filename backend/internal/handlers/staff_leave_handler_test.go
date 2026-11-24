package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
)

func TestStaffLeaveHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name    string
		handler func(*StaffLeaveHandler, *gin.Context)
	}{
		{"list leave types", (*StaffLeaveHandler).ListTypes},
		{"save leave type", (*StaffLeaveHandler).SaveType},
		{"list leave requests", (*StaffLeaveHandler).ListRequests},
		{"create leave request", (*StaffLeaveHandler).CreateRequest},
		{"approve leave request", (*StaffLeaveHandler).Approve},
		{"reject leave request", (*StaffLeaveHandler).Reject},
		{"cancel leave request", (*StaffLeaveHandler).Cancel},
		{"list holidays", (*StaffLeaveHandler).ListHolidays},
		{"save holiday", (*StaffLeaveHandler).SaveHoliday},
	}

	for _, method := range methods {
		t.Run(method.name+" nil receiver", func(t *testing.T) {
			router := gin.New()
			router.Any("/staff/:requestID/:typeID/:holidayID", func(c *gin.Context) {
                c.Set(middleware.ContextPrincipalIDKey, int64(100))
                c.Set(middleware.ContextSchoolIDKey, int64(10))
				method.handler(nil, c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/42/1/2", nil))
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unavailable response must contain exactly one valid JSON object: %v; body=%s", err, recorder.Body.String())
			}
			if body["message"] != "Staff leave and holiday service is unavailable" {
				t.Fatalf("unexpected unavailable response body: %#v", body)
			}
		})

		t.Run(method.name+" nil service", func(t *testing.T) {
			handler := NewStaffLeaveHandler(nil)
			router := gin.New()
			router.Any("/staff/:requestID/:typeID/:holidayID", func(c *gin.Context) {
                c.Set(middleware.ContextPrincipalIDKey, int64(100))
                c.Set(middleware.ContextSchoolIDKey, int64(10))
				method.handler(handler, c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/42/1/2", nil))
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("unavailable response must contain exactly one valid JSON object: %v; body=%s", err, recorder.Body.String())
			}
			if body["message"] != "Staff leave and holiday service is unavailable" {
				t.Fatalf("unexpected unavailable response body: %#v", body)
			}
		})
	}
}


func TestStaffLeaveUnauthenticatedRequestDoesNotExposeServiceAvailability(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/staff/leave-types", NewStaffLeaveHandler(nil).ListTypes)
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/leave-types", nil))
    if recorder.Code != http.StatusUnauthorized {
        t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
    }
}


type nilStaffLeaveListService struct { StaffLeaveManagementService }
func (nilStaffLeaveListService) ListTypes(context.Context,int64,int64)([]repository.StaffLeaveType,error){return nil,nil}
func (nilStaffLeaveListService) ListRequests(context.Context,int64,int64,string,int64,string,string,string)([]repository.StaffLeaveRequest,error){return nil,nil}
func (nilStaffLeaveListService) ListHolidays(context.Context,int64,int64,string,string,bool)([]repository.SchoolHoliday,error){return nil,nil}

func TestStaffLeaveListHandlersReturnEmptyArrays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h:=NewStaffLeaveHandler(nilStaffLeaveListService{})
	r:=gin.New()
	auth:=func(c *gin.Context){c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()}
	r.GET("/types",auth,h.ListTypes)
	r.GET("/requests",auth,h.ListRequests)
	r.GET("/holidays",auth,h.ListHolidays)
	for _,path:=range []string{"/types","/requests","/holidays"} {
		rec:=httptest.NewRecorder();r.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,path,nil))
		if rec.Code!=http.StatusOK || !strings.Contains(rec.Body.String(),`"data":[]`) {t.Fatalf("%s: status=%d body=%s",path,rec.Code,rec.Body.String())}
	}
}

type nilStaffLeaveWriteResultService struct { StaffLeaveManagementService }
func (nilStaffLeaveWriteResultService) SaveType(context.Context,int64,int64,int64,repository.StaffLeaveType)(*repository.StaffLeaveType,error){return nil,nil}
func (nilStaffLeaveWriteResultService) CreateRequest(context.Context,int64,int64,repository.StaffLeaveRequest)(*repository.StaffLeaveRequest,error){return nil,nil}
func (nilStaffLeaveWriteResultService) SaveHoliday(context.Context,int64,int64,int64,repository.SchoolHoliday)(*repository.SchoolHoliday,error){return nil,nil}

func TestStaffLeaveHandlersRejectNilSuccessfulWriteResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h:=NewStaffLeaveHandler(nilStaffLeaveWriteResultService{})
	auth:=func(c *gin.Context){c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()}
	tests:=[]struct{name,method,path,body string; handler func(*StaffLeaveHandler,*gin.Context)}{
		{"save leave type",http.MethodPost,"/staff/types",`{}`,(*StaffLeaveHandler).SaveType},
		{"create leave request",http.MethodPost,"/staff/requests",`{}`,(*StaffLeaveHandler).CreateRequest},
		{"save holiday",http.MethodPost,"/staff/holidays",`{}`,(*StaffLeaveHandler).SaveHoliday},
	}
	for _,tt:=range tests{t.Run(tt.name,func(t *testing.T){
		r:=gin.New();r.Handle(tt.method,tt.path,auth,func(c *gin.Context){tt.handler(h,c)})
		req:=httptest.NewRequest(tt.method,tt.path,strings.NewReader(tt.body));req.Header.Set("Content-Type","application/json")
		rec:=httptest.NewRecorder();r.ServeHTTP(rec,req)
		if rec.Code!=http.StatusInternalServerError{t.Fatalf("nil successful service result must return 500: status=%d body=%s",rec.Code,rec.Body.String())}
	})}
}


type captureStaffLeaveListService struct {
	StaffLeaveManagementService
	actorID, schoolID, staffID int64
	status, onDate, fromDate, toDate string
	called bool
}

func (s *captureStaffLeaveListService) ListRequests(_ context.Context, actorID, schoolID int64, status string, staffID int64, onDate, fromDate, toDate string) ([]repository.StaffLeaveRequest, error) {
	s.called = true
	s.actorID, s.schoolID, s.staffID = actorID, schoolID, staffID
	s.status, s.onDate, s.fromDate, s.toDate = status, onDate, fromDate, toDate
	return []repository.StaffLeaveRequest{}, nil
}

func TestStaffLeaveListRequestsPassesTrimmedFiltersToService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffLeaveListService{}
	handler := NewStaffLeaveHandler(service)
	router := gin.New()
	router.GET("/staff/requests", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(100))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		c.Next()
	}, handler.ListRequests)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/staff/requests?status=%20pending%20&staff_principal_id=%20321%20&on_date=%202026-11-12%20&from_date=%202026-11-01%20&to_date=%202026-11-30%20", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !service.called {
		t.Fatal("leave list service was not called")
	}
	if service.actorID != 100 || service.schoolID != 10 || service.staffID != 321 {
		t.Fatalf("identity filters actor=%d school=%d staff=%d", service.actorID, service.schoolID, service.staffID)
	}
	if service.status != "pending" || service.onDate != "2026-11-12" || service.fromDate != "2026-11-01" || service.toDate != "2026-11-30" {
		t.Fatalf("unexpected forwarded filters: status=%q onDate=%q fromDate=%q toDate=%q", service.status, service.onDate, service.fromDate, service.toDate)
	}
}

func TestStaffLeaveListRequestsRejectsInvalidStaffPrincipalIDBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffLeaveListService{}
	handler := NewStaffLeaveHandler(service)
	router := gin.New()
	router.GET("/staff/requests", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(100))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		c.Next()
	}, handler.ListRequests)

	for _, value := range []string{"0", "-1", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/requests?staff_principal_id="+value, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if service.called {
				t.Fatal("leave list service must not be called for an invalid staff principal ID")
			}
		})
	}
}
