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
