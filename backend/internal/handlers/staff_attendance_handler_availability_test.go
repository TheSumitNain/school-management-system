package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/repository"

	"school-management-system/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestStaffAttendanceHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name string
		handler func(*StaffAttendanceHandler, *gin.Context)
	}{
		{"list daily", (*StaffAttendanceHandler).ListDaily},
		{"list monthly", (*StaffAttendanceHandler).ListMonthly},
		{"mark attendance", (*StaffAttendanceHandler).Mark},
		{"bulk mark attendance", (*StaffAttendanceHandler).MarkBulk},
		{"update attendance", (*StaffAttendanceHandler).Update},
	}
	for _, method := range methods {
		for _, receiver := range []struct {
			name string
			handler *StaffAttendanceHandler
		}{
			{"nil receiver", nil},
			{"nil service", NewStaffAttendanceHandler(nil)},
		} {
			t.Run(method.name+" "+receiver.name, func(t *testing.T) {
				router := gin.New()
				router.Any("/staff/:recordID", func(c *gin.Context) {
					c.Set(middleware.ContextPrincipalIDKey, int64(100))
					c.Set(middleware.ContextSchoolIDKey, int64(10))
					method.handler(receiver.handler, c)
				})
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/200", nil))
				if recorder.Code != http.StatusServiceUnavailable {
					t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
				}
			})
		}
	}
}


func TestStaffAttendanceUnauthenticatedRequestDoesNotExposeServiceAvailability(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    router.GET("/staff/attendance", NewStaffAttendanceHandler(nil).ListDaily)
    recorder := httptest.NewRecorder()
    router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/staff/attendance", nil))
    if recorder.Code != http.StatusUnauthorized {
        t.Fatalf("status=%d want=%d body=%s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
    }
}

type nilStaffAttendanceListService struct { StaffAttendanceManagementService }
func (nilStaffAttendanceListService) ListDaily(context.Context,int64,int64,time.Time)([]repository.StaffAttendanceRecord,error){return nil,nil}
func (nilStaffAttendanceListService) ListMonthly(context.Context,int64,int64,time.Time)([]repository.StaffAttendanceMonthlySummary,error){return nil,nil}
func (nilStaffAttendanceListService) ListDailyPage(context.Context,int64,int64,time.Time,int,int)([]repository.StaffAttendanceRecord,int,error){return nil,0,nil}
func (nilStaffAttendanceListService) ListMonthlyPage(context.Context,int64,int64,time.Time,int,int)([]repository.StaffAttendanceMonthlySummary,int,error){return nil,0,nil}

func TestStaffAttendanceListHandlersReturnEmptyArrays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r:=gin.New()
	h:=NewStaffAttendanceHandler(nilStaffAttendanceListService{})
	auth:=func(c *gin.Context){c.Set(middleware.ContextPrincipalIDKey,int64(100));c.Set(middleware.ContextSchoolIDKey,int64(10));c.Next()}
	r.GET("/daily",auth,h.ListDaily)
	r.GET("/monthly",auth,h.ListMonthly)
	for _,path:=range []string{"/daily","/monthly"} {
		rec:=httptest.NewRecorder();r.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,path,nil))
		if rec.Code!=http.StatusOK || !strings.Contains(rec.Body.String(),`"data":[]`) {t.Fatalf("%s: status=%d body=%s",path,rec.Code,rec.Body.String())}
	}
}

type nilStaffAttendanceWriteResultService struct {
	StaffAttendanceManagementService
}

func (nilStaffAttendanceWriteResultService) Mark(context.Context, int64, int64, int64, time.Time, string, *string) (*repository.StaffAttendanceRecord, error) {
	return nil, nil
}

func (nilStaffAttendanceWriteResultService) Update(context.Context, int64, int64, int64, string, *string) (*repository.StaffAttendanceRecord, error) {
	return nil, nil
}

func TestStaffAttendanceWriteHandlersRejectNilSuccessfulResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		method string
		path string
		body string
		wantStatus int
		wantMessage string
	}{
		{name: "mark", method: http.MethodPost, path: "/staff/attendance", body: `{"staff_principal_id":200,"attendance_date":"2026-11-23","status":"present"}`, wantStatus: http.StatusInternalServerError, wantMessage: "Unable to mark staff attendance"},
		{name: "update", method: http.MethodPut, path: "/staff/attendance/300", body: `{"status":"present"}`, wantStatus: http.StatusInternalServerError, wantMessage: "Unable to update staff attendance"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			handler := NewStaffAttendanceHandler(nilStaffAttendanceWriteResultService{})
			router.Any("/staff/attendance", func(c *gin.Context) {
				c.Set(middleware.ContextPrincipalIDKey, int64(100))
				c.Set(middleware.ContextSchoolIDKey, int64(10))
				if c.Request.Method == http.MethodPost {
					handler.Mark(c)
					return
				}
				handler.ListDaily(c)
			})
			router.PUT("/staff/attendance/:recordID", func(c *gin.Context) {
				c.Set(middleware.ContextPrincipalIDKey, int64(100))
				c.Set(middleware.ContextSchoolIDKey, int64(10))
				handler.Update(c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if recorder.Code != tt.wantStatus || !strings.Contains(recorder.Body.String(), tt.wantMessage) {
				t.Fatalf("status=%d body=%s; want status=%d and message %q", recorder.Code, recorder.Body.String(), tt.wantStatus, tt.wantMessage)
			}
		})
	}
}

