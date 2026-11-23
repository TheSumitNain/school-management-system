package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestStaffSubstitutionHandlersReturnUnavailableWithoutService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	methods := []struct {
		name string
		handler func(*StaffSubstitutionHandler, *gin.Context)
	}{
		{"list substitutions", (*StaffSubstitutionHandler).List},
		{"create substitution", (*StaffSubstitutionHandler).Create},
		{"cancel substitution", (*StaffSubstitutionHandler).Cancel},
	}
	for _, method := range methods {
		for _, receiver := range []struct {
			name string
			handler *StaffSubstitutionHandler
		}{
			{"nil receiver", nil},
			{"nil service", NewStaffSubstitutionHandler(nil)},
		} {
			t.Run(method.name+" "+receiver.name, func(t *testing.T) {
				router := gin.New()
				router.Any("/staff/:substitutionID", func(c *gin.Context) {
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
