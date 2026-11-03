package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type RoleManagementService interface {
	ListSchoolRoles(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolRole, error)
	ListSchoolPermissions(ctx context.Context) ([]repository.SchoolPermission, error)
	CreateCustomRole(ctx context.Context, actorPrincipalID, schoolID int64, name string) (*repository.SchoolRole, error)
	GrantSchoolPermissionDelegation(ctx context.Context, actorPrincipalID, schoolID int64, permissionKey string, canAssignToRoles, canOverrideUsers bool) (*repository.SchoolPermissionDelegation, error)
	RevokeSchoolPermissionDelegation(ctx context.Context, actorPrincipalID, delegationID int64) error
}

type RoleHandler struct {
	service RoleManagementService
}

func NewRoleHandler(service RoleManagementService) *RoleHandler {
	return &RoleHandler{service: service}
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Role management is unavailable"})
		return
	}
	roles, err := h.service.ListSchoolRoles(c.Request.Context(), actorID, schoolID)
	if err != nil {
		if errors.Is(err, repository.ErrRolePermissionDenied) {
			c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "School role access is not authorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list school roles"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "roles": roles})
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission catalogue is unavailable"})
		return
	}
	permissions, err := h.service.ListSchoolPermissions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list school permissions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "permissions": permissions})
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Role management is unavailable"})
		return
	}
	var request struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request body"})
		return
	}
	role, err := h.service.CreateCustomRole(c.Request.Context(), actorID, schoolID, request.Name)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"status": "success", "role": role})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Role name must contain 2–120 characters"})
	case errors.Is(err, repository.ErrRoleNameTaken):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "A role with this name already exists"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Custom role creation is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to create custom role"})
	}
}

func roleActorContext(c *gin.Context) (int64, int64, bool) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolID, schoolOK := schoolValue.(int64)
	return actorID, schoolID, actorExists && schoolExists && actorOK && schoolOK && actorID > 0 && schoolID > 0
}

func (h *RoleHandler) GrantSchoolPermissionDelegation(c *gin.Context) {
	actorValue, exists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, ok := actorValue.(int64)
	if !exists || !ok || actorID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Platform Admin authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission delegation is unavailable"})
		return
	}
	schoolID, err := strconv.ParseInt(c.Param("schoolID"), 10, 64)
	if err != nil || schoolID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid school ID"})
		return
	}
	var request struct {
		PermissionKey string `json:"permission_key"`
		CanAssignToRoles bool `json:"can_assign_to_roles"`
		CanOverrideUsers bool `json:"can_override_users"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request body"})
		return
	}
	if request.PermissionKey == "" || (!request.CanAssignToRoles && !request.CanOverrideUsers) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "A permission and at least one delegation capability are required"})
		return
	}
	delegation, err := h.service.GrantSchoolPermissionDelegation(
		c.Request.Context(), actorID, schoolID, request.PermissionKey,
		request.CanAssignToRoles, request.CanOverrideUsers,
	)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"status": "success", "delegation": delegation})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid delegation request"})
	case errors.Is(err, repository.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "School or delegable permission not found"})
	case errors.Is(err, repository.ErrDelegationAlreadyActive):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "An active delegation already exists for this school permission"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission delegation is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to grant permission delegation"})
	}
}

func (h *RoleHandler) RevokeSchoolPermissionDelegation(c *gin.Context) {
	actorValue, exists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, ok := actorValue.(int64)
	if !exists || !ok || actorID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Platform Admin authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission delegation is unavailable"})
		return
	}
	delegationID, err := strconv.ParseInt(c.Param("delegationID"), 10, 64)
	if err != nil || delegationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid delegation ID"})
		return
	}
	err = h.service.RevokeSchoolPermissionDelegation(c.Request.Context(), actorID, delegationID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Permission delegation revoked"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid delegation request"})
	case errors.Is(err, repository.ErrDelegationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Active permission delegation not found"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission delegation is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to revoke permission delegation"})
	}
}
