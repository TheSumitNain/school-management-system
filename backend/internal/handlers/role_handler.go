package handlers

import (
	"context"
	"errors"
	"net/http"

	"school-management-system/internal/middleware"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type RoleManagementService interface {
	ListSchoolRoles(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolRole, error)
	ListSchoolPermissions(ctx context.Context) ([]repository.SchoolPermission, error)
	CreateCustomRole(ctx context.Context, actorPrincipalID, schoolID int64, name string) (*repository.SchoolRole, error)
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
