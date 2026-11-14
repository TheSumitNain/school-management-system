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

func TestGuardianRoutesRequireDedicatedPermissions(t *testing.T) {
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
		method string
		path string
		permission string
		body string
	}{
		{http.MethodGet, "/school/guardians", "guardians.read", ""},
		{http.MethodPost, "/school/guardians", "guardians.manage", `{"full_name":"Guardian One"}`},
		{http.MethodGet, "/school/guardians/1", "guardians.read", ""},
		{http.MethodPatch, "/school/guardians/1", "guardians.manage", `{"full_name":"Guardian Updated","status":"active"}`},
		{http.MethodGet, "/school/students/1/guardians", "guardians.read", ""},
		{http.MethodPost, "/school/students/1/guardians", "guardians.manage", `{"guardian_id":2,"relationship_type":"mother"}`},
		{http.MethodPatch, "/school/student-guardian-links/1", "guardians.manage", `{"relationship_type":"mother","is_primary_contact":true,"is_emergency_contact":false}`},
		{http.MethodPost, "/school/student-guardian-links/1/revoke", "guardians.manage", `{"reason":"Relationship corrected"}`},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			build := func(permissions []string) *gin.Engine {
				router, err := setupRouterWithHandlers(
					fakeDatabase{}, nil, validator,
					fakePrincipalLoader{principal: principal},
					fakePermissionLoader{permissions: permissions},
					nil, nil, nil, nil, handlers.NewGuardianHandler(nil), fakeAdminSessionActivityTracker{adminPrincipalID: -1, familyID: "00000000-0000-4000-8000-000000000031"},
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
			if got:=request(nil);got.Code!=http.StatusForbidden{
				t.Fatalf("without %s: status=%d body=%s",test.permission,got.Code,got.Body.String())
			}
			if got:=request([]string{"students.read_basic"});got.Code!=http.StatusForbidden{
				t.Fatalf("Teacher/basic student permission must not expose guardian data: status=%d body=%s",got.Code,got.Body.String())
			}
			if got:=request([]string{test.permission});got.Code!=http.StatusServiceUnavailable{
				t.Fatalf("authorized route should reach handler: status=%d body=%s",got.Code,got.Body.String())
			}
		})
	}
}
