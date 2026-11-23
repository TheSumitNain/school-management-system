package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
