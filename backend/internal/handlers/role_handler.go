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
	ListSchoolPermissionDelegations(ctx context.Context, actorPrincipalID, schoolID int64) ([]repository.SchoolPermissionDelegation, error)
	GrantCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID, delegationID int64, permissionKey string) error
	RevokeCustomRolePermission(ctx context.Context, actorPrincipalID, schoolID, roleID int64, permissionKey string) error
	ListStaffPermissionOverrides(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffPermissionOverride, error)
	SetStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, delegationID int64, permissionKey, effect, reason string) error
	DeleteStaffPermissionOverride(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, permissionKey string) error
	ListStaffRoleAssignments(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) ([]repository.StaffRoleAssignment, error)
	AssignStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error
	RevokeStaffRole(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID, roleID int64) error
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

func (h *RoleHandler) ListSchoolPermissionDelegations(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission delegation is unavailable"})
		return
	}
	items, err := h.service.ListSchoolPermissionDelegations(c.Request.Context(), actorID, schoolID)
	if errors.Is(err, repository.ErrRolePermissionDenied) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission delegation access is not authorized"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list permission delegations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "delegations": items})
}

func (h *RoleHandler) GrantCustomRolePermission(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Role permission management is unavailable"})
		return
	}
	roleID, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || roleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid role ID"})
		return
	}
	var request struct {
		PermissionKey string `json:"permission_key"`
		DelegationID int64 `json:"delegation_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.PermissionKey == "" || request.DelegationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "A permission key and exact delegation ID are required"})
		return
	}
	err = h.service.GrantCustomRolePermission(c.Request.Context(), actorID, schoolID, roleID, request.DelegationID, request.PermissionKey)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Permission granted to custom role"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid role permission grant"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Role permission grant is not authorized"})
	case errors.Is(err, repository.ErrDelegationNotFound), errors.Is(err, repository.ErrRoleNotFound):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "The exact active delegation or custom role is not available"})
	case errors.Is(err, repository.ErrRolePermissionAlreadyGranted):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Permission is already granted to this role"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to grant permission to custom role"})
	}
}

func (h *RoleHandler) RevokeCustomRolePermission(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Role permission management is unavailable"})
		return
	}
	roleID, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || roleID <= 0 || c.Param("permissionKey") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid role ID or permission key"})
		return
	}
	err = h.service.RevokeCustomRolePermission(c.Request.Context(), actorID, schoolID, roleID, c.Param("permissionKey"))
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Custom role permission revoked"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid role permission revocation"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Role permission revocation is not authorized"})
	case errors.Is(err, repository.ErrDelegationNotFound), errors.Is(err, repository.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Custom role permission grant not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to revoke custom role permission"})
	}
}

func (h *RoleHandler) ListStaffPermissionOverrides(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission override management is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	items, err := h.service.ListStaffPermissionOverrides(c.Request.Context(), actorID, schoolID, targetID)
	if errors.Is(err, repository.ErrRolePermissionDenied) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission override access is not authorized"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list staff permission overrides"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "overrides": items})
}

func (h *RoleHandler) SetStaffPermissionOverride(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission override management is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	var request struct {
		PermissionKey string `json:"permission_key"`
		Effect string `json:"effect"`
		Reason string `json:"reason"`
		DelegationID int64 `json:"delegation_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.PermissionKey == "" ||
		(request.Effect != "allow" && request.Effect != "deny") ||
		request.Reason == "" || request.DelegationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Permission, allow/deny effect, reason, and exact delegation ID are required"})
		return
	}
	err = h.service.SetStaffPermissionOverride(c.Request.Context(), actorID, schoolID, targetID,
		request.DelegationID, request.PermissionKey, request.Effect, request.Reason)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Staff permission override saved"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid permission override"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission override is not authorized"})
	case errors.Is(err, repository.ErrDelegationNotFound):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "The exact active override delegation is not available"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to save staff permission override"})
	}
}

func (h *RoleHandler) DeleteStaffPermissionOverride(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Permission override management is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 || c.Param("permissionKey") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID or permission key"})
		return
	}
	err = h.service.DeleteStaffPermissionOverride(c.Request.Context(), actorID, schoolID, targetID, c.Param("permissionKey"))
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Staff permission override removed"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid permission override removal"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Permission override removal is not authorized"})
	case errors.Is(err, repository.ErrDelegationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Staff permission override not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to remove staff permission override"})
	}
}

func (h *RoleHandler) ListStaffRoleAssignments(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff role management is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	items, err := h.service.ListStaffRoleAssignments(c.Request.Context(), actorID, schoolID, targetID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "roles": items})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff role access is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list staff roles"})
	}
}

func (h *RoleHandler) AssignStaffRole(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff role management is unavailable"})
		return
	}
	targetID, targetErr := strconv.ParseInt(c.Param("principalID"), 10, 64)
	roleID, roleErr := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if targetErr != nil || roleErr != nil || targetID <= 0 || roleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID or role ID"})
		return
	}
	err := h.service.AssignStaffRole(c.Request.Context(), actorID, schoolID, targetID, roleID)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"status": "success", "message": "Staff role assigned"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff role assignment"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff role assignment is not authorized"})
	case errors.Is(err, repository.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Target staff account or assignable role not found"})
	case errors.Is(err, repository.ErrRoleAssignmentExists):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "Staff member already has this role"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to assign staff role"})
	}
}

func (h *RoleHandler) RevokeStaffRole(c *gin.Context) {
	actorID, schoolID, ok := roleActorContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff role management is unavailable"})
		return
	}
	targetID, targetErr := strconv.ParseInt(c.Param("principalID"), 10, 64)
	roleID, roleErr := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if targetErr != nil || roleErr != nil || targetID <= 0 || roleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID or role ID"})
		return
	}
	err := h.service.RevokeStaffRole(c.Request.Context(), actorID, schoolID, targetID, roleID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Staff role revoked"})
	case errors.Is(err, services.ErrInvalidCustomRole):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff role revocation"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff role revocation is not authorized"})
	case errors.Is(err, repository.ErrRoleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Active staff role assignment not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to revoke staff role"})
	}
}
