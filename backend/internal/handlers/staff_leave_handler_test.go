package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
