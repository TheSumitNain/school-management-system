package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"school-management-system/internal/middleware"
	"school-management-system/internal/models"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type StaffAccountManagementService interface {
	Create(ctx context.Context, actorPrincipalID, schoolID int64, username, initialPassword string) (*models.User, error)
	List(ctx context.Context, actorPrincipalID, schoolID int64) ([]models.User, error)
	Disable(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64) error
}

type StaffHandler struct {
	service StaffAccountManagementService
}

func NewStaffHandler(service StaffAccountManagementService) *StaffHandler {
	return &StaffHandler{service: service}
}

func (h *StaffHandler) Create(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff account creation is unavailable"})
		return
	}
	var request struct {
		Username string `json:"username"`
		InitialPassword string `json:"initial_password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Username) == "" || request.InitialPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "username and initial_password are required"})
		return
	}
	user, err := h.service.Create(c.Request.Context(), actorID, schoolID, request.Username, request.InitialPassword)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{
			"status": "success",
			"staff": gin.H{
				"id": user.ID,
				"principal_id": user.PrincipalID,
				"school_id": user.SchoolID,
				"username": user.Username,
				"status": user.Status,
				"must_change_password": user.MustChangePassword,
			},
		})
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Username must be 3–32 lowercase letters, digits, dots, underscores, or hyphens"})
	case errors.Is(err, services.ErrInvalidPassword):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Initial password must meet the password policy"})
	case errors.Is(err, repository.ErrUsernameTaken):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "This username is already in use"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account creation is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to create staff account"})
	}
}

func (h *StaffHandler) List(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff account listing is unavailable"})
		return
	}
	users, err := h.service.List(c.Request.Context(), actorID, schoolID)
	if errors.Is(err, repository.ErrRolePermissionDenied) {
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account listing is not authorized"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to list school staff"})
		return
	}
	staff := make([]gin.H, 0, len(users))
	for _, user := range users {
		staff = append(staff, gin.H{
			"id": user.ID,
			"principal_id": user.PrincipalID,
			"school_id": user.SchoolID,
			"username": user.Username,
			"status": user.Status,
			"must_change_password": user.MustChangePassword,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "staff": staff})
}

func (h *StaffHandler) Disable(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff account disable is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	err = h.service.Disable(c.Request.Context(), actorID, schoolID, targetID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Staff account disabled and active sessions revoked"})
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff account disable request"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account disable is not authorized"})
	case errors.Is(err, repository.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Active staff account not found in this school"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to disable staff account"})
	}
}
