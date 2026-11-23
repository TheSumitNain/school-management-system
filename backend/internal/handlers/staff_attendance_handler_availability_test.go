package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
