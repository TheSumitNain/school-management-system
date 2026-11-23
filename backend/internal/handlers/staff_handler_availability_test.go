package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestStaffAccountHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name    string
		handler func(*StaffHandler, *gin.Context)
	}{
		{"create staff", (*StaffHandler).Create},
		{"list staff", (*StaffHandler).List},
		{"disable staff", (*StaffHandler).Disable},
		{"update staff", (*StaffHandler).Update},
		{"enable staff", (*StaffHandler).Enable},
	}

	for _, method := range methods {
		for _, receiver := range []struct {
			name    string
			handler *StaffHandler
		}{
			{name: "nil receiver", handler: nil},
			{name: "nil service", handler: NewStaffHandler(nil)},
		} {
			t.Run(method.name+" "+receiver.name, func(t *testing.T) {
				router := gin.New()
				router.Any("/staff/:principalID", func(c *gin.Context) {
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
