package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/middleware"
	"github.com/gin-gonic/gin"
)

type captureStaffAttendanceBulkService struct {
	StaffAttendanceManagementService
	calls int
	actorID int64
	schoolID int64
	staffIDs []int64
	day time.Time
	status string
	remarks *string
}

func (s *captureStaffAttendanceBulkService) MarkBulk(_ context.Context, actorID, schoolID int64, staffIDs []int64, day time.Time, status string, remarks *string) (int, error) {
	s.calls++
	s.actorID, s.schoolID = actorID, schoolID
	s.staffIDs = append([]int64(nil), staffIDs...)
	s.day, s.status, s.remarks = day, status, remarks
	return len(staffIDs), nil
}

func TestStaffAttendanceBulkHandlerMapsAuthenticatedContextAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffAttendanceBulkService{}
	router := gin.New()
	router.POST("/staff/attendance/bulk", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(101))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		NewStaffAttendanceHandler(service).MarkBulk(c)
	})

	recorder := httptest.NewRecorder()
	body := `{"staff_principal_ids":[201,202],"attendance_date":"2026-11-25","status":"present","remarks":"Morning register"}`
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/attendance/bulk", strings.NewReader(body)))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s want=%d", recorder.Code, recorder.Body.String(), http.StatusCreated)
	}
	if service.calls != 1 || service.actorID != 101 || service.schoolID != 10 {
		t.Fatalf("calls=%d actor=%d school=%d", service.calls, service.actorID, service.schoolID)
	}
	if len(service.staffIDs) != 2 || service.staffIDs[0] != 201 || service.staffIDs[1] != 202 {
		t.Fatalf("staff IDs=%v want=[201 202]", service.staffIDs)
	}
	if service.day.Format("2006-01-02") != "2026-11-25" || service.status != "present" {
		t.Fatalf("day=%s status=%q", service.day.Format("2006-01-02"), service.status)
	}
	if service.remarks == nil || *service.remarks != "Morning register" {
		t.Fatalf("remarks=%v want Morning register", service.remarks)
	}
	if !strings.Contains(recorder.Body.String(), `"created":2`) {
		t.Fatalf("response does not report created count: %s", recorder.Body.String())
	}
}

func TestStaffAttendanceBulkHandlerRejectsInvalidDateBeforeServiceCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffAttendanceBulkService{}
	router := gin.New()
	router.POST("/staff/attendance/bulk", func(c *gin.Context) {
		c.Set(middleware.ContextPrincipalIDKey, int64(101))
		c.Set(middleware.ContextSchoolIDKey, int64(10))
		NewStaffAttendanceHandler(service).MarkBulk(c)
	})

	recorder := httptest.NewRecorder()
	body := `{"staff_principal_ids":[201],"attendance_date":"2026-02-30","status":"present"}`
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/attendance/bulk", strings.NewReader(body)))

	if recorder.Code != http.StatusBadRequest || service.calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s; want 400 and no service call", recorder.Code, service.calls, recorder.Body.String())
	}
}

func TestStaffAttendanceBulkHandlerRequiresAuthenticatedSchoolContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &captureStaffAttendanceBulkService{}
	router := gin.New()
	router.POST("/staff/attendance/bulk", NewStaffAttendanceHandler(service).MarkBulk)

	recorder := httptest.NewRecorder()
	body := `{"staff_principal_ids":[201],"attendance_date":"2026-11-25","status":"present"}`
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/staff/attendance/bulk", strings.NewReader(body)))

	if recorder.Code != http.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s; want 401 and no service call", recorder.Code, service.calls, recorder.Body.String())
	}
}

