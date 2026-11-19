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
		additionalPermissions []string
		body string
	}{
		{method: http.MethodGet, path: "/school/students", permission: "students.read_sensitive"},
		{method: http.MethodPost, path: "/school/students", permission: "students.create", body: `{"admission_number":"ADM-001","full_name":"Student One"}`},
		{method: http.MethodGet, path: "/school/students/1", permission: "students.read_sensitive"},
		{method: http.MethodPatch, path: "/school/students/1", permission: "students.update", body: `{"full_name":"Student Updated"}`},
		{method: http.MethodPost, path: "/school/students/1/archive", permission: "students.archive"},
		{method: http.MethodGet, path: "/school/enrollments", permission: "students.read_basic"},
		{method: http.MethodGet, path: "/school/students/1/enrollments", permission: "students.read_basic"},
		{method: http.MethodPost, path: "/school/enrollments", permission: "students.create", body: `{"student_id":1,"academic_year_id":2,"section_id":3}`},
		{method: http.MethodPost, path: "/school/enrollments/1/promote", permission: "students.update", body: `{"academic_year_id":2,"section_id":3}`},
		{method: http.MethodGet, path: "/school/students/1/addresses", permission: "students.read_sensitive"},
		{method: http.MethodGet, path: "/school/students/1/basic-details", permission: "students.read_sensitive"},
		{method: http.MethodGet, path: "/school/students/1/contact-details", permission: "students.read_sensitive"},
		{method: http.MethodGet, path: "/school/students/1/medical-details", permission: "students.read_sensitive"},
		{method: http.MethodGet, path: "/school/students/1/physical-details", permission: "students.read_sensitive"},
		{method: http.MethodGet, path: "/school/locations", permission: "students.read_sensitive"},
		{method: http.MethodPost, path: "/school/locations", permission: "students.update", body: `{"level":"state","name":"Test State","country_code":"IN"}`},
		{method: http.MethodPost, path: "/school/locations/1/deactivate", permission: "students.update"},
		{method: http.MethodPut, path: "/school/students/1/addresses/current", permission: "students.update", additionalPermissions: []string{"students.read_sensitive"}, body: `{"address_line_1":"House 1"}`},
		{method: http.MethodPut, path: "/school/students/1/basic-details", permission: "students.update", additionalPermissions: []string{"students.read_sensitive"}, body: `{}`},
		{method: http.MethodPut, path: "/school/students/1/contact-details", permission: "students.update", additionalPermissions: []string{"students.read_sensitive"}, body: `{}`},
		{method: http.MethodPut, path: "/school/students/1/medical-details", permission: "students.update", additionalPermissions: []string{"students.read_sensitive"}, body: `{}`},
		{method: http.MethodPut, path: "/school/students/1/physical-details", permission: "students.update", additionalPermissions: []string{"students.read_sensitive"}, body: `{"measured_on":"2026-11-15","height_cm":120}`},
		{method: http.MethodGet, path: "/school/class-curricula?academic_year_id=1&class_id=2&section_id=4", permission: "subjects.read"},
		{method: http.MethodPost, path: "/school/class-curricula", permission: "subjects.manage", body: `{"academic_year_id":1,"class_id":2,"section_id":4,"subjects":[{"subject_id":3}]}`},
		{method: http.MethodGet, path: "/school/enrollments/1/curriculum", permission: "subjects.read", additionalPermissions: []string{"students.read_basic"}},
		{method: http.MethodPut, path: "/school/enrollments/1/subjects", permission: "students.update", additionalPermissions: []string{"subjects.read", "students.read_basic"}, body: `{"curriculum_id":1,"selected_optional_subject_ids":[]}`},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			build := func(permissions []string) *gin.Engine {
				router, err := setupRouterWithHandlers(
					fakeDatabase{}, nil, validator,
					fakePrincipalLoader{principal: principal},
					fakePermissionLoader{permissions: permissions},
					nil, nil, nil, handlers.NewAcademicHandler(nil), fakeAdminSessionActivityTracker{adminPrincipalID: -1, familyID: "00000000-0000-4000-8000-000000000031"},
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
			requiredPermissions := append([]string{test.permission}, test.additionalPermissions...)
			for _, missingPermission := range requiredPermissions {
				permissions := make([]string, 0, len(requiredPermissions)-1)
				for _, candidate := range requiredPermissions {
					if candidate != missingPermission {
						permissions = append(permissions, candidate)
					}
				}
				if got := request(permissions); got.Code != http.StatusForbidden {
					t.Fatalf("without %s: status=%d body=%s", missingPermission, got.Code, got.Body.String())
				}
			}
			if test.permission == "students.read_sensitive" {
				if got := request([]string{"students.read_basic"}); got.Code != http.StatusForbidden {
					t.Fatalf("basic Teacher read permission must not expose student profiles: status=%d body=%s", got.Code, got.Body.String())
				}
			}
			permissions := append([]string{test.permission}, test.additionalPermissions...)
			if got := request(permissions); got.Code != http.StatusServiceUnavailable {
				t.Fatalf("authorized route should reach handler: status=%d body=%s", got.Code, got.Body.String())
			}
		})
	}
}


func TestStudentDirectoryRejectsInvalidAcademicYearFilter(t *testing.T) {
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
	router, err := setupRouterWithHandlers(
		fakeDatabase{}, nil, validator,
		fakePrincipalLoader{principal: principal},
		fakePermissionLoader{permissions: []string{"students.read_sensitive"}},
		nil, nil, nil, handlers.NewAcademicHandler(services.NewAcademicService(nil)),
		fakeAdminSessionActivityTracker{adminPrincipalID: -1, familyID: "00000000-0000-4000-8000-000000000031"},
	)
	if err != nil {
		t.Fatalf("setup router: %v", err)
	}
	for _, value := range []string{"not-a-number", "0", "-1"} {
		t.Run(value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/school/students?academic_year_id="+value, nil)
			req.Header.Set("Authorization", "Bearer valid-test-token")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("invalid academic_year_id %q: status=%d body=%s; want 400", value, rec.Code, rec.Body.String())
			}
		})
	}
}
