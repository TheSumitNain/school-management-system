package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/repository"

	"github.com/gin-gonic/gin"
)

type fakeRoleManagementService struct {
	roles []repository.SchoolRole
	permissions []repository.SchoolPermission
	created *repository.SchoolRole
	actorID int64
	schoolID int64
	name string
}

func (f *fakeRoleManagementService) ListSchoolRoles(context.Context, int64, int64) ([]repository.SchoolRole, error) {
	return f.roles, nil
}

func (f *fakeRoleManagementService) ListSchoolPermissions(context.Context) ([]repository.SchoolPermission, error) {
	return f.permissions, nil
}

func (f *fakeRoleManagementService) CreateCustomRole(_ context.Context, actorID, schoolID int64, name string) (*repository.SchoolRole, error) {
	f.actorID, f.schoolID, f.name = actorID, schoolID, name
	return f.created, nil
}

func (f *fakeRoleManagementService) GrantSchoolPermissionDelegation(context.Context, int64, int64, string, bool, bool) (*repository.SchoolPermissionDelegation, error) {
	return &repository.SchoolPermissionDelegation{ID: 6, SchoolID: 8, PermissionKey: "attendance.read", CanAssignToRoles: true}, nil
}

func (f *fakeRoleManagementService) RevokeSchoolPermissionDelegation(context.Context, int64, int64) error {
	return nil
}

func TestRoleHandlerCreateRoleUsesAuthenticatedSchool(t *testing.T) {
	gin.SetMode(gin.TestMode)
	role := &repository.SchoolRole{ID: 3, RoleKey: "custom_library", Name: "Library"}
	service := &fakeRoleManagementService{created: role}
	handler := NewRoleHandler(service)
	router := gin.New()
	router.POST("/school/roles", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, handler.CreateRole)

	req := httptest.NewRequest(http.MethodPost, "/school/roles", strings.NewReader(`{"name":"Library"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if service.actorID != 31 || service.schoolID != 8 || service.name != "Library" {
		t.Fatalf("unexpected custom role service input: %+v", service)
	}
}

func TestRoleHandlerListsSchoolRolesAndPermissionCatalogue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeRoleManagementService{
		roles: []repository.SchoolRole{{ID: 1, RoleKey: "school_admin", Name: "School Admin", IsSystem: true}},
		permissions: []repository.SchoolPermission{{Key: "students.read_basic", Description: "Read basic student fields", Delegable: true}},
	}
	handler := NewRoleHandler(service)
	router := gin.New()
	router.GET("/school/roles", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}, handler.ListRoles)
	router.GET("/school/permissions", handler.ListPermissions)

	rolesRec := httptest.NewRecorder()
	router.ServeHTTP(rolesRec, httptest.NewRequest(http.MethodGet, "/school/roles", nil))
	if rolesRec.Code != http.StatusOK || !strings.Contains(rolesRec.Body.String(), "school_admin") {
		t.Fatalf("role list response = %d %s", rolesRec.Code, rolesRec.Body.String())
	}

	permissionsRec := httptest.NewRecorder()
	router.ServeHTTP(permissionsRec, httptest.NewRequest(http.MethodGet, "/school/permissions", nil))
	if permissionsRec.Code != http.StatusOK || !strings.Contains(permissionsRec.Body.String(), "students.read_basic") {
		t.Fatalf("permission list response = %d %s", permissionsRec.Code, permissionsRec.Body.String())
	}
}
