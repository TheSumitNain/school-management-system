package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/handlers"
	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Attendance authorization has a date-sensitive repository check as well as
// route-level permission gates. These tests ensure the route accepts either
// valid permission and never accidentally requires both.
func TestAttendanceRoutesEnforceExpectedPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{
		ID: 31, Type: "staff", Status: "active", TokenVersion: 1,
		SchoolID: 8, Username: "school.admin",
	}
	tests := []struct {
		name string
		method string
		path string
		body string
		acceptedPermissions []string
	}{
		{
			name: "list requires attendance read",
			method: http.MethodGet,
			path: "/school/attendance",
			acceptedPermissions: []string{"attendance.read"},
		},
		{
			name: "mark accepts mark or past correction permission",
			method: http.MethodPost,
			path: "/school/attendance",
			body: `{"enrollment_id":1,"attendance_date":"2026-11-15","status":"present"}`,
			acceptedPermissions: []string{"attendance.mark", "attendance.correct_past"},
		},
		{
			name: "update accepts same day or past correction permission",
			method: http.MethodPatch,
			path: "/school/attendance/1",
			body: `{"status":"absent"}`,
			acceptedPermissions: []string{"attendance.update_same_day", "attendance.correct_past"},
		},
		{
			name: "correction endpoint accepts same day or past correction permission",
			method: http.MethodPatch,
			path: "/school/attendance/1/correct",
			body: `{"status":"absent"}`,
			acceptedPermissions: []string{"attendance.update_same_day", "attendance.correct_past"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			build := func(permissions []string) *gin.Engine {
				router, err := setupRouterWithHandlers(
					fakeDatabase{}, nil, validator,
					fakePrincipalLoader{principal: principal},
					fakePermissionLoader{permissions: permissions},
					nil, nil,
					handlers.NewAttendanceHandler(nil),
					fakeAdminSessionActivityTracker{adminPrincipalID: -1, familyID: "00000000-0000-4000-8000-000000000031"},
				)
				if err != nil { t.Fatalf("setup router: %v", err) }
				return router
			}
			request := func(permissions []string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
				req.Header.Set("Authorization", "Bearer valid-test-token")
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				build(permissions).ServeHTTP(rec, req)
				return rec
			}
			if got := request(nil); got.Code != http.StatusForbidden {
				t.Fatalf("without attendance permission: status=%d body=%s", got.Code, got.Body.String())
			}
			for _, permission := range test.acceptedPermissions {
				if got := request([]string{permission}); got.Code != http.StatusServiceUnavailable {
					t.Fatalf("with %s: route should pass authorization and reach unavailable test handler; status=%d body=%s",
						permission, got.Code, got.Body.String())
				}
			}
			if len(test.acceptedPermissions) > 1 {
				for _, permission := range test.acceptedPermissions {
					if got := request([]string{test.acceptedPermissions[0], test.acceptedPermissions[1]}); got.Code != http.StatusServiceUnavailable {
						t.Fatalf("with both permissions (including %s): status=%d body=%s", permission, got.Code, got.Body.String())
					}
				}
			}
		})
	}
}
