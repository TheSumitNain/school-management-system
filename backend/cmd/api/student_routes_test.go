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

func TestStudentRoutesEnforceLivePermissions(t *testing.T) {
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
		{http.MethodGet, "/school/students", "students.read_basic", ""},
		{http.MethodPost, "/school/students", "students.create", `{"admission_number":"ADM-001","full_name":"Student One"}`},
		{http.MethodGet, "/school/students/1", "students.read_basic", ""},
		{http.MethodPatch, "/school/students/1", "students.update", `{"full_name":"Student Updated"}`},
		{http.MethodPost, "/school/students/1/archive", "students.archive", ""},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			build := func(permissions []string) *gin.Engine {
				router, err := setupRouterWithHandlers(
					fakeDatabase{}, nil, validator,
					fakePrincipalLoader{principal: principal},
					fakePermissionLoader{permissions: permissions},
					nil, nil, nil, handlers.NewAcademicHandler(nil),
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
				t.Fatalf("without %s: status=%d body=%s", test.permission, got.Code, got.Body.String())
			}
			if got := request([]string{test.permission}); got.Code != http.StatusServiceUnavailable {
				t.Fatalf("authorized route should reach handler: status=%d body=%s", got.Code, got.Body.String())
			}
		})
	}
}
