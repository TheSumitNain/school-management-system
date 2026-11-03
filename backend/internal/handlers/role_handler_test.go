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
	delegation *repository.SchoolPermissionDelegation
	delegationID int64
	roleID int64
	grantedDelegationID int64
	grantPermissionKey string
	permissionKey string
	canAssignToRoles bool
	canOverrideUsers bool
	err error
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

func (f *fakeRoleManagementService) GrantSchoolPermissionDelegation(_ context.Context, actorID, schoolID int64, permissionKey string, canAssignToRoles, canOverrideUsers bool) (*repository.SchoolPermissionDelegation, error) {
	f.actorID, f.schoolID, f.permissionKey = actorID, schoolID, permissionKey
	f.canAssignToRoles, f.canOverrideUsers = canAssignToRoles, canOverrideUsers
	if f.delegation == nil {
		f.delegation = &repository.SchoolPermissionDelegation{ID: 6, SchoolID: schoolID, PermissionKey: permissionKey, CanAssignToRoles: canAssignToRoles, CanOverrideUsers: canOverrideUsers}
	}
	return f.delegation, f.err
}

func (f *fakeRoleManagementService) RevokeSchoolPermissionDelegation(_ context.Context, actorID, delegationID int64) error {
	f.actorID, f.delegationID = actorID, delegationID
	return f.err
}

func (f *fakeRoleManagementService) ListSchoolPermissionDelegations(context.Context, int64, int64) ([]repository.SchoolPermissionDelegation, error) {
	return []repository.SchoolPermissionDelegation{}, nil
}
func (f *fakeRoleManagementService) GrantCustomRolePermission(_ context.Context, actorID, schoolID, roleID, delegationID int64, permissionKey string) error {
	f.actorID, f.schoolID, f.roleID, f.grantedDelegationID, f.grantPermissionKey = actorID, schoolID, roleID, delegationID, permissionKey
	return f.err
}
func (f *fakeRoleManagementService) RevokeCustomRolePermission(_ context.Context, actorID, schoolID, roleID int64, permissionKey string) error {
	f.actorID, f.schoolID, f.roleID, f.grantPermissionKey = actorID, schoolID, roleID, permissionKey
	return f.err
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

func TestRoleHandlerPlatformAdminDelegationGrantAndRevoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeRoleManagementService{}
	handler := NewRoleHandler(service)
	router := gin.New()
	authenticate := func(c *gin.Context) {
		c.Set("auth_principal_id", int64(99))
		c.Next()
	}
	router.POST("/platform/schools/:schoolID/permission-delegations", authenticate, handler.GrantSchoolPermissionDelegation)
	router.POST("/platform/permission-delegations/:delegationID/revoke", authenticate, handler.RevokeSchoolPermissionDelegation)

	req := httptest.NewRequest(http.MethodPost, "/platform/schools/8/permission-delegations",
		strings.NewReader(`{"permission_key":"attendance.read","can_assign_to_roles":true,"can_override_users":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || service.actorID != 99 || service.schoolID != 8 ||
		service.permissionKey != "attendance.read" || !service.canAssignToRoles || service.canOverrideUsers {
		t.Fatalf("unexpected delegation grant: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/platform/permission-delegations/6/revoke", nil))
	if rec.Code != http.StatusOK || service.actorID != 99 || service.delegationID != 6 {
		t.Fatalf("unexpected delegation revoke: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
}

func TestRoleHandlerDelegationRejectsInvalidRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeRoleManagementService{}
	router := gin.New()
	router.POST("/platform/schools/:schoolID/permission-delegations", func(c *gin.Context) {
		c.Set("auth_principal_id", int64(99))
		c.Next()
	}, NewRoleHandler(service).GrantSchoolPermissionDelegation)

	req := httptest.NewRequest(http.MethodPost, "/platform/schools/8/permission-delegations",
		strings.NewReader(`{"permission_key":"attendance.read","can_assign_to_roles":false,"can_override_users":false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid delegation to be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRoleHandlerCustomRolePermissionGrantAndRevoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeRoleManagementService{}
	router := gin.New()
	authenticate := func(c *gin.Context) {
		c.Set("auth_principal_id", int64(31))
		c.Set("auth_school_id", int64(8))
		c.Next()
	}
	handler := NewRoleHandler(service)
	router.POST("/school/roles/:roleID/permissions", authenticate, handler.GrantCustomRolePermission)
	router.DELETE("/school/roles/:roleID/permissions/:permissionKey", authenticate, handler.RevokeCustomRolePermission)

	req := httptest.NewRequest(http.MethodPost, "/school/roles/14/permissions",
		strings.NewReader(`{"permission_key":"attendance.read","delegation_id":73}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || service.actorID != 31 || service.schoolID != 8 ||
		service.roleID != 14 || service.grantedDelegationID != 73 || service.grantPermissionKey != "attendance.read" {
		t.Fatalf("unexpected custom role permission grant: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/school/roles/14/permissions/attendance.read", nil))
	if rec.Code != http.StatusOK || service.actorID != 31 || service.schoolID != 8 ||
		service.roleID != 14 || service.grantPermissionKey != "attendance.read" {
		t.Fatalf("unexpected custom role permission revoke: status=%d service=%+v body=%s", rec.Code, service, rec.Body.String())
	}
}
