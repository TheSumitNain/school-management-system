package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"strconv"
	"testing"

	"school-management-system/internal/handlers"
	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type fakeDatabase struct {
	err error
}

type fakePrincipalLoader struct {
	principal *models.AuthPrincipal
	err       error
}

func (f fakePrincipalLoader) LoadPrincipalForAuthentication(_ context.Context, _ int64) (*models.AuthPrincipal, error) {
	return f.principal, f.err
}

type fakePermissionLoader struct {
	permissions []string
	err         error
}

func (f fakePermissionLoader) LoadEffectivePermissions(_ context.Context, _ *models.AuthPrincipal) ([]string, error) {
	return f.permissions, f.err
}

type fakeAdminSessionActivityTracker struct {
	adminPrincipalID int64
	familyID string
}

func (f fakeAdminSessionActivityTracker) CheckAndTouchAdminSession(_ context.Context, principalID int64, familyID string) (bool, bool, error) {
	isAdmin := principalID == f.adminPrincipalID
	return isAdmin, !isAdmin || familyID == f.familyID, nil
}

type fakeTokenValidator struct {
	claims *services.TokenClaims
	err    error
}

func (f fakeTokenValidator) ValidateAccessToken(
	tokenString string,
) (*services.TokenClaims, error) {
	if tokenString != "valid-test-token" {
		return nil, errors.New("invalid token")
	}

	return f.claims, f.err
}

func (f fakeDatabase) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf(
			"expected JSON content type, got %q",
			rec.Header().Get("Content-Type"),
		)
	}

	expectedBody := `{"message":"School Management System API is running","status":"success"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestReadyEndpointWhenDatabaseIsAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expectedBody := `{"message":"API and database are ready","status":"success"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestReadyEndpointWhenDatabaseFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{
		err: errors.New("database unavailable"),
	}, nil, nil, nil, nil)

	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	expectedBody := `{"message":"Database is not ready","status":"error"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestAuthMeWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validator := fakeTokenValidator{
		claims: &services.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "10",
			},
			PrincipalType: "staff",
			TokenVersion:  1,
		},
	}

	router, err := setupRouter(fakeDatabase{}, nil, validator, fakePrincipalLoader{principal: &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "tester"}}, fakePermissionLoader{})
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer valid-test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	expectedBody := `{"must_change_password":false,"principal_id":10,"principal_type":"staff","school_id":2,"status":"success","username":"tester"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, rec.Body.String())
	}
}

func TestAuthMeWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{}, fakePrincipalLoader{}, fakePermissionLoader{})
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMeWithInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{}, fakePrincipalLoader{}, fakePermissionLoader{})
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMeForParentOmitsStaffOnlyFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validator := fakeTokenValidator{
		claims: &services.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: "42"},
			PrincipalType: "parent",
			TokenVersion: 1,
		},
	}
	router, err := setupRouter(
		fakeDatabase{},
		nil,
		validator,
		fakePrincipalLoader{principal: &models.AuthPrincipal{
			ID: 42, Type: "parent", Status: "active", TokenVersion: 1, MobileVerified: true,
		}},
		fakePermissionLoader{},
	)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer valid-test-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	expectedBody := `{"principal_id":42,"principal_type":"parent","status":"success"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected Parent identity response %s, got %s", expectedBody, rec.Body.String())
	}
}

func TestSchoolStaffListingRouteEnforcesLiveUsersReadPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{ID: 31, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 8, Username: "school.admin"}
	staffHandler := handlers.NewStaffHandler(nil)

	deniedRouter, err := setupRouterWithHandlers(
		fakeDatabase{}, nil, validator,
		fakePrincipalLoader{principal: principal},
		fakePermissionLoader{permissions: []string{}},
		nil, nil, staffHandler,
	)
	if err != nil {
		t.Fatalf("setup denied staff router: %v", err)
	}
	deniedReq := httptest.NewRequest(http.MethodGet, "/school/staff", nil)
	deniedReq.Header.Set("Authorization", "Bearer valid-test-token")
	deniedRec := httptest.NewRecorder()
	deniedRouter.ServeHTTP(deniedRec, deniedReq)
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("missing users.read permission: status=%d body=%s", deniedRec.Code, deniedRec.Body.String())
	}

	allowedRouter, err := setupRouterWithHandlers(
		fakeDatabase{}, nil, validator,
		fakePrincipalLoader{principal: principal},
		fakePermissionLoader{permissions: []string{"users.read"}},
		nil, nil, staffHandler,
	)
	if err != nil {
		t.Fatalf("setup allowed staff router: %v", err)
	}
	allowedReq := httptest.NewRequest(http.MethodGet, "/school/staff", nil)
	allowedReq.Header.Set("Authorization", "Bearer valid-test-token")
	allowedRec := httptest.NewRecorder()
	allowedRouter.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("authorized request should reach handler (nil service), got status=%d body=%s", allowedRec.Code, allowedRec.Body.String())
	}
}


func TestStaffUpdateAndEnableRoutesEnforceLiveUsersUpdatePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{ID: 31, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 8, Username: "school.admin"}
	staffHandler := handlers.NewStaffHandler(nil)

	for _, test := range []struct {
		method string
		path string
		body string
	}{
		{method: http.MethodPatch, path: "/school/staff/42", body: `{"username":"new.teacher"}`},
		{method: http.MethodPost, path: "/school/staff/42/enable", body: `{"temporary_password":"A-Strong-Temporary-Pass-123"}`},
	} {
		deniedRouter, err := setupRouterWithHandlers(
			fakeDatabase{}, nil, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: []string{}},
			nil, nil, staffHandler,
		)
		if err != nil {
			t.Fatalf("setup denied %s route: %v", test.method, err)
		}
		deniedReq := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		deniedReq.Header.Set("Authorization", "Bearer valid-test-token")
		deniedReq.Header.Set("Content-Type", "application/json")
		deniedRec := httptest.NewRecorder()
		deniedRouter.ServeHTTP(deniedRec, deniedReq)
		if deniedRec.Code != http.StatusForbidden {
			t.Fatalf("%s %s without users.update: status=%d body=%s",
				test.method, test.path, deniedRec.Code, deniedRec.Body.String())
		}

		allowedRouter, err := setupRouterWithHandlers(
			fakeDatabase{}, nil, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: []string{"users.update"}},
			nil, nil, staffHandler,
		)
		if err != nil {
			t.Fatalf("setup allowed %s route: %v", test.method, err)
		}
		allowedReq := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		allowedReq.Header.Set("Authorization", "Bearer valid-test-token")
		allowedReq.Header.Set("Content-Type", "application/json")
		allowedRec := httptest.NewRecorder()
		allowedRouter.ServeHTTP(allowedRec, allowedReq)
		if allowedRec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s with users.update should reach unavailable handler, got status=%d body=%s",
				test.method, test.path, allowedRec.Code, allowedRec.Body.String())
		}
	}
}

func TestStaffPasswordResetRouteRequiresDedicatedLivePermission(t *testing.T) {
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

	request := func(permissions []string) *httptest.ResponseRecorder {
		authHandler := handlers.NewAuthHandler(nil, nil, nil)
		router, err := setupRouterWithHandlers(
			fakeDatabase{}, authHandler, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: permissions},
			nil, nil,
		)
		if err != nil {
			t.Fatalf("setup password-reset router: %v", err)
		}
		req := httptest.NewRequest(
			http.MethodPost,
			"/school/staff/42/password/reset",
			strings.NewReader(`{"new_password":"A-Strong-Temporary-Pass-123"}`),
		)
		req.Header.Set("Authorization", "Bearer valid-test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if denied := request(nil); denied.Code != http.StatusForbidden {
		t.Fatalf("password reset without permission: status=%d body=%s; want 403",
			denied.Code, denied.Body.String())
	}
	if wrongPermission := request([]string{"users.update"}); wrongPermission.Code != http.StatusForbidden {
		t.Fatalf("password reset with users.update only: status=%d body=%s; want 403",
			wrongPermission.Code, wrongPermission.Body.String())
	}
	if allowed := request([]string{"users.password_reset"}); allowed.Code != http.StatusServiceUnavailable {
		t.Fatalf("password reset with dedicated permission should reach unavailable handler: status=%d body=%s; want 503",
			allowed.Code, allowed.Body.String())
	}
}

func TestRoleAdministrationRoutesEnforceLivePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		method      string
		path        string
		principalType string
		permissions []string
	}{
		{name: "platform delegation grant", method: http.MethodPost, path: "/platform/schools/8/permission-delegations", principalType: "platform_admin", permissions: []string{"platform.permissions.delegate"}},
		{name: "platform delegation revoke", method: http.MethodPost, path: "/platform/permission-delegations/6/revoke", principalType: "platform_admin", permissions: []string{"platform.permissions.delegate"}},
		{name: "school role list", method: http.MethodGet, path: "/school/roles", principalType: "staff", permissions: []string{"roles.read"}},
		{name: "custom role create", method: http.MethodPost, path: "/school/roles", principalType: "staff", permissions: []string{"roles.create_custom"}},
		{name: "permission catalogue", method: http.MethodGet, path: "/school/permissions", principalType: "staff", permissions: []string{"permissions.read"}},
		{name: "school delegation list", method: http.MethodGet, path: "/school/permission-delegations", principalType: "staff", permissions: []string{"permissions.assign"}},
		{name: "custom role permission grant", method: http.MethodPost, path: "/school/roles/14/permissions", principalType: "staff", permissions: []string{"permissions.assign", "roles.update_custom"}},
		{name: "custom role permission revoke", method: http.MethodDelete, path: "/school/roles/14/permissions/students.read_basic", principalType: "staff", permissions: []string{"permissions.assign", "roles.update_custom"}},
		{name: "staff override list", method: http.MethodGet, path: "/school/staff/42/permission-overrides", principalType: "staff", permissions: []string{"permissions.override"}},
		{name: "staff override set", method: http.MethodPost, path: "/school/staff/42/permission-overrides", principalType: "staff", permissions: []string{"permissions.override"}},
		{name: "staff override delete", method: http.MethodDelete, path: "/school/staff/42/permission-overrides/homework.create", principalType: "staff", permissions: []string{"permissions.override"}},
		{name: "staff role assignment list", method: http.MethodGet, path: "/school/staff/42/roles", principalType: "staff", permissions: []string{"roles.read"}},
		{name: "staff role assignment grant", method: http.MethodPost, path: "/school/staff/42/roles/14", principalType: "staff", permissions: []string{"roles.assign"}},
		{name: "staff role assignment revoke", method: http.MethodDelete, path: "/school/staff/42/roles/14", principalType: "staff", permissions: []string{"roles.assign"}},
	}

	buildRouter := func(principalType string, permissions []string) *gin.Engine {
		principal := &models.AuthPrincipal{
			ID: 31, Type: principalType, Status: "active", TokenVersion: 1,
			SchoolID: 8, Username: "school.admin",
		}
		if principalType == "platform_admin" {
			principal.ID = 32
			principal.SchoolID = 0
			principal.Username = ""
		}
		validator := fakeTokenValidator{claims: &services.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(principal.ID, 10)},
			PrincipalType: principalType,
			TokenVersion: 1,
			SessionID: "00000000-0000-4000-8000-000000000031",
		}}
		router, err := setupRouterWithHandlers(
			fakeDatabase{}, nil, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: permissions},
			nil, handlers.NewRoleHandler(nil), fakeAdminSessionActivityTracker{adminPrincipalID: 32, familyID: "00000000-0000-4000-8000-000000000031"},
		)
		if err != nil {
			t.Fatalf("setup %s router: %v", principalType, err)
		}
		return router
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := func(permissions []string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(test.method, test.path, nil)
				req.Header.Set("Authorization", "Bearer valid-test-token")
				rec := httptest.NewRecorder()
				buildRouter(test.principalType, permissions).ServeHTTP(rec, req)
				return rec
			}

			denied := request(nil)
			if denied.Code != http.StatusForbidden {
				t.Fatalf("without required permissions: status=%d body=%s; want 403",
					denied.Code, denied.Body.String())
			}

			if len(test.permissions) > 1 {
				partial := request(test.permissions[:1])
				if partial.Code != http.StatusForbidden {
					t.Fatalf("with only one of %v: status=%d body=%s; want 403",
						test.permissions, partial.Code, partial.Body.String())
				}
			}

			allowed := request(test.permissions)
			if allowed.Code != http.StatusServiceUnavailable {
				t.Fatalf("with all required permissions request should pass route authorization and reach nil handler service: status=%d body=%s; want 503",
					allowed.Code, allowed.Body.String())
			}
		})
	}
}



func TestAcademicRoutesEnforceResourcePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{ID: 31, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 8, Username: "school.admin"}
	handler := handlers.NewAcademicHandler(nil)
	tests := []struct {
		method string
		path string
		permission string
		body string
	}{
		{http.MethodGet, "/school/academic-years", "academic_years.read", ""},
		{http.MethodPost, "/school/academic-years", "academic_years.manage", `{"name":"2026-27","starts_on":"2026-04-01","ends_on":"2027-03-31","status":"active","is_current":true}`},
		{http.MethodGet, "/school/classes", "classes.read", ""},
		{http.MethodPost, "/school/classes", "classes.manage", `{"name":"LKG","display_order":1}`},
		{http.MethodGet, "/school/sections", "classes.read", ""},
		{http.MethodPost, "/school/sections", "classes.manage", `{"academic_year_id":1,"class_id":1,"name":"A"}`},
		{http.MethodGet, "/school/subjects", "subjects.read", ""},
		{http.MethodPost, "/school/subjects", "subjects.manage", `{"name":"Mathematics","subject_code":"MATH"}`},
		{http.MethodGet, "/school/teacher-assignments", "teacher_assignments.read", ""},
		{http.MethodPost, "/school/teacher-assignments", "teacher_assignments.manage", `{"academic_year_id":1,"section_id":1,"subject_id":1,"teacher_principal_id":1}`},
		{http.MethodDelete, "/school/teacher-assignments/1", "teacher_assignments.manage", ""},
		{http.MethodGet, "/school/class-incharge-assignments", "class_incharge_assignments.read", ""},
		{http.MethodPost, "/school/class-incharge-assignments", "class_incharge_assignments.manage", `{"academic_year_id":1,"section_id":1,"teacher_principal_id":1}`},
		{http.MethodDelete, "/school/class-incharge-assignments/1", "class_incharge_assignments.manage", ""},
		{http.MethodGet, "/school/enrollments", "students.read_basic", ""},
		{http.MethodPost, "/school/enrollments", "students.create", `{"student_id":1,"academic_year_id":1,"section_id":1,"enrolled_at":"2028-04-01"}`},
		{http.MethodGet, "/school/students/1/enrollments", "students.read_basic", ""},
		{http.MethodPost, "/school/enrollments/1/promote", "students.update", `{"academic_year_id":2,"section_id":2,"enrolled_at":"2029-04-01"}`},
		{http.MethodPatch, "/school/academic-years/1", "academic_years.manage", ""},
		{http.MethodPatch, "/school/classes/1", "classes.manage", ""},
		{http.MethodPatch, "/school/sections/1", "classes.manage", ""},
		{http.MethodPatch, "/school/subjects/1", "subjects.manage", ""},
		{http.MethodGet, "/school/students", "students.read_sensitive", ""},
		{http.MethodPost, "/school/students", "students.create", `{"admission_number":"ADM-1","full_name":"Test Student"}`},
		{http.MethodGet, "/school/students/1", "students.read_sensitive", ""},
		{http.MethodPatch, "/school/students/1", "students.update", `{"full_name":"Updated Student"}`},
		{http.MethodPost, "/school/students/1/archive", "students.archive", `{"reason":"left school"}`},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			deniedRouter, err := setupRouterWithHandlers(
				fakeDatabase{}, nil, validator, fakePrincipalLoader{principal: principal},
				fakePermissionLoader{}, nil, nil, nil, handler,
			)
			if err != nil { t.Fatalf("setup denied router: %v", err) }
			req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Authorization", "Bearer valid-test-token")
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			deniedRouter.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden { t.Fatalf("missing %s: status=%d body=%s", test.permission, rec.Code, rec.Body.String()) }

			allowedRouter, err := setupRouterWithHandlers(
				fakeDatabase{}, nil, validator, fakePrincipalLoader{principal: principal},
				fakePermissionLoader{permissions: []string{test.permission}}, nil, nil, nil, handler,
			)
			if err != nil { t.Fatalf("setup allowed router: %v", err) }
			req = httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			req.Header.Set("Authorization", "Bearer valid-test-token")
			req.Header.Set("Content-Type", "application/json")
			rec = httptest.NewRecorder()
			allowedRouter.ServeHTTP(rec, req)
			if rec.Code != http.StatusServiceUnavailable { t.Fatalf("authorized request should reach handler, got status=%d body=%s", rec.Code, rec.Body.String()) }
		})
	}
}


func TestGuardianRoutesEnforceDedicatedLivePermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{
		ID: 31, Type: "staff", Status: "active", TokenVersion: 1,
		SchoolID: 8, Username: "school.staff",
	}
	tests := []struct {
		method, path, permission, body string
	}{
		{http.MethodGet, "/school/guardians", "guardians.read", ""},
		{http.MethodPost, "/school/guardians", "guardians.manage", `{"full_name":"Guardian"}`},
		{http.MethodGet, "/school/guardians/9", "guardians.read", ""},
		{http.MethodPatch, "/school/guardians/9", "guardians.manage", `{"full_name":"Guardian","status":"active"}`},
		{http.MethodGet, "/school/students/12/guardians", "guardians.read", ""},
		{http.MethodPost, "/school/students/12/guardians", "guardians.manage", `{"guardian_id":9,"relationship_type":"mother"}`},
		{http.MethodPost, "/school/student-guardian-links/14/revoke", "guardians.manage", `{"reason":"relationship ended"}`},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			buildRouter := func(permissions []string) *gin.Engine {
				router, err := setupRouterWithHandlers(
					fakeDatabase{}, nil, validator,
					fakePrincipalLoader{principal: principal},
					fakePermissionLoader{permissions: permissions},
					nil, nil, nil, nil, handlers.NewGuardianHandler(nil),
				)
				if err != nil {
					t.Fatalf("setup guardian router: %v", err)
				}
				return router
			}
			request := func(permissions []string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
				req.Header.Set("Authorization", "Bearer valid-test-token")
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				buildRouter(permissions).ServeHTTP(rec, req)
				return rec
			}
			denied := request(nil)
			if denied.Code != http.StatusForbidden {
				t.Fatalf("without %s: status=%d body=%s", test.permission, denied.Code, denied.Body.String())
			}
			allowed := request([]string{test.permission})
			if allowed.Code != http.StatusServiceUnavailable {
				t.Fatalf("with %s request should reach unavailable handler: status=%d body=%s",
					test.permission, allowed.Code, allowed.Body.String())
			}
		})
	}
}


func TestAttendanceRoutesEnforceReadAndMarkPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff", TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{ID:31,Type:"staff",Status:"active",TokenVersion:1,SchoolID:8,Username:"school.staff"}
	tests := []struct{method,path,permission,body string}{
		{http.MethodGet,"/school/attendance","attendance.read",""},
		{http.MethodPost,"/school/attendance","attendance.mark",`{"enrollment_id":2,"attendance_date":"2026-11-04","status":"present"}`},
	}
	for _,test:=range tests {
		t.Run(test.method+" "+test.path,func(t *testing.T){
			request:=func(perms []string)*httptest.ResponseRecorder{
				router,err:=setupRouterWithHandlers(fakeDatabase{},nil,validator,
					fakePrincipalLoader{principal:principal},fakePermissionLoader{permissions:perms},
					nil,nil,nil,nil,nil,handlers.NewAttendanceHandler(nil))
				if err!=nil{t.Fatalf("setup router: %v",err)}
				req:=httptest.NewRequest(test.method,test.path,strings.NewReader(test.body))
				req.Header.Set("Authorization","Bearer valid-test-token")
				req.Header.Set("Content-Type","application/json")
				rec:=httptest.NewRecorder();router.ServeHTTP(rec,req);return rec
			}
			if got:=request(nil);got.Code!=http.StatusForbidden{t.Fatalf("without %s: status=%d body=%s",test.permission,got.Code,got.Body.String())}
			if got:=request([]string{test.permission});got.Code!=http.StatusServiceUnavailable{t.Fatalf("with %s should reach handler: status=%d body=%s",test.permission,got.Code,got.Body.String())}
		})
	}
}


func TestAttendanceCorrectionRouteEnforcesDedicatedPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator:=fakeTokenValidator{claims:&services.TokenClaims{RegisteredClaims:jwt.RegisteredClaims{Subject:"31"},PrincipalType:"staff",TokenVersion:1}}
	principal:=&models.AuthPrincipal{ID:31,Type:"staff",Status:"active",TokenVersion:1,SchoolID:8,Username:"school.staff"}
	routerFor:=func(perms []string)*gin.Engine{
		router,err:=setupRouterWithHandlers(fakeDatabase{},nil,validator,fakePrincipalLoader{principal:principal},
			fakePermissionLoader{permissions:perms},nil,nil,nil,nil,nil,handlers.NewAttendanceHandler(nil))
		if err!=nil{t.Fatalf("setup router: %v",err)}
		return router
	}
	request:=func(perms []string)*httptest.ResponseRecorder{
		req:=httptest.NewRequest(http.MethodPatch,"/school/attendance/7/correct",strings.NewReader(`{"status":"absent"}`))
		req.Header.Set("Authorization","Bearer valid-test-token");req.Header.Set("Content-Type","application/json")
		rec:=httptest.NewRecorder();routerFor(perms).ServeHTTP(rec,req);return rec
	}
	if got:=request(nil);got.Code!=http.StatusForbidden{t.Fatalf("without attendance.correct_past: %d %s",got.Code,got.Body.String())}
	if got:=request([]string{"attendance.correct_past"});got.Code!=http.StatusServiceUnavailable{t.Fatalf("with attendance.correct_past should reach handler: %d %s",got.Code,got.Body.String())}
}


func TestAttendanceSameDayUpdateRouteRequiresDedicatedPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "31"},
		PrincipalType: "staff",
		TokenVersion: 1,
	}}
	principal := &models.AuthPrincipal{
		ID: 31, Type: "staff", Status: "active", TokenVersion: 1,
		SchoolID: 8, Username: "school.staff",
	}
	request := func(permissions []string) *httptest.ResponseRecorder {
		router, err := setupRouterWithHandlers(
			fakeDatabase{}, nil, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: permissions},
			nil, nil, nil, nil, nil, handlers.NewAttendanceHandler(nil),
		)
		if err != nil {
			t.Fatalf("setup attendance router: %v", err)
		}
		req := httptest.NewRequest(
			http.MethodPatch,
			"/school/attendance/7",
			strings.NewReader(`{"status":"absent"}`),
		)
		req.Header.Set("Authorization", "Bearer valid-test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if denied := request(nil); denied.Code != http.StatusForbidden {
		t.Fatalf("same-day update without attendance.update_same_day: status=%d body=%s; want 403",
			denied.Code, denied.Body.String())
	}
	if wrongPermission := request([]string{"attendance.mark"}); wrongPermission.Code != http.StatusForbidden {
		t.Fatalf("same-day update with attendance.mark only: status=%d body=%s; want 403",
			wrongPermission.Code, wrongPermission.Body.String())
	}
	if allowed := request([]string{"attendance.update_same_day"}); allowed.Code != http.StatusServiceUnavailable {
		t.Fatalf("same-day update with dedicated permission should reach unavailable handler: status=%d body=%s; want 503",
			allowed.Code, allowed.Body.String())
	}
}

func TestMFARecoveryCodeRegenerationRequiresAuthenticatedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHandler := handlers.NewAuthHandler(nil, nil, nil)
	router, err := setupRouter(
		fakeDatabase{},
		authHandler,
		fakeTokenValidator{},
		fakePrincipalLoader{},
		fakePermissionLoader{},
	)
	if err != nil {
		t.Fatalf("setup router: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/mfa/recovery-codes/regenerate",
		strings.NewReader(`{"challenge_token":"opaque-challenge","code":"123456"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("recovery-code regeneration without access token must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPlatformSchoolCreationRouteRequiresBothDedicatedPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		principalID = int64(32)
		sessionID = "00000000-0000-4000-8000-000000000031"
	)
	principal := &models.AuthPrincipal{
		ID: principalID, Type: "platform_admin", Status: "active", TokenVersion: 1,
	}
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(principalID, 10)},
		PrincipalType: "platform_admin",
		TokenVersion: 1,
		SessionID: sessionID,
	}}
	request := func(permissions []string) *httptest.ResponseRecorder {
		router, err := setupRouterWithHandlers(
			fakeDatabase{}, nil, validator,
			fakePrincipalLoader{principal: principal},
			fakePermissionLoader{permissions: permissions},
			handlers.NewSchoolOnboardingHandler(nil), nil,
			fakeAdminSessionActivityTracker{adminPrincipalID: principalID, familyID: sessionID},
		)
		if err != nil {
			t.Fatalf("setup platform school onboarding router: %v", err)
		}
		req := httptest.NewRequest(
			http.MethodPost,
			"/platform/schools",
			strings.NewReader(`{"name":"Test School","code":"TEST","school_admin_username":"test.admin","school_admin_initial_password":"A-Strong-Temporary-Pass-123"}`),
		)
		req.Header.Set("Authorization", "Bearer valid-test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if denied := request(nil); denied.Code != http.StatusForbidden {
		t.Fatalf("without platform school permissions: status=%d body=%s; want 403",
			denied.Code, denied.Body.String())
	}
	if partial := request([]string{"platform.schools.create"}); partial.Code != http.StatusForbidden {
		t.Fatalf("with platform.schools.create only: status=%d body=%s; want 403",
			partial.Code, partial.Body.String())
	}
	if partial := request([]string{"platform.school_admins.create"}); partial.Code != http.StatusForbidden {
		t.Fatalf("with platform.school_admins.create only: status=%d body=%s; want 403",
			partial.Code, partial.Body.String())
	}
	if allowed := request([]string{"platform.schools.create", "platform.school_admins.create"}); allowed.Code != http.StatusServiceUnavailable {
		t.Fatalf("with both dedicated permissions request should reach unavailable onboarding handler: status=%d body=%s; want 503",
			allowed.Code, allowed.Body.String())
	}
}

