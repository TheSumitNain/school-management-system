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

type SchoolOnboardingService interface {
	CreateSchoolWithAdmin(ctx context.Context, actorPrincipalID int64, input services.SchoolOnboardingInput) (*services.SchoolOnboardingResult, error)
}

type SchoolOnboardingHandler struct {
	service SchoolOnboardingService
}

func NewSchoolOnboardingHandler(service SchoolOnboardingService) *SchoolOnboardingHandler {
	return &SchoolOnboardingHandler{service: service}
}

type createSchoolRequest struct {
	Name                 string `json:"name"`
	Code                 string `json:"code"`
	AdminUsername        string `json:"school_admin_username"`
	AdminInitialPassword string `json:"school_admin_initial_password"`
}

func (h *SchoolOnboardingHandler) Create(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "School onboarding is unavailable"})
		return
	}
	actorValue, exists := c.Get(middleware.ContextPrincipalIDKey)
	actorPrincipalID, ok := actorValue.(int64)
	if !exists || !ok || actorPrincipalID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "Authentication required"})
		return
	}

	var request createSchoolRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request body"})
		return
	}

	result, err := h.service.CreateSchoolWithAdmin(c.Request.Context(), actorPrincipalID, services.SchoolOnboardingInput{
		Name: request.Name,
		Code: request.Code,
		AdminUsername: request.AdminUsername,
		AdminInitialPassword: request.AdminInitialPassword,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidSchoolOnboarding):
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "School name, code or School Admin username is invalid"})
		case errors.Is(err, services.ErrInvalidPassword):
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "School Admin initial password must meet the password policy"})
		case errors.Is(err, services.ErrCompromisedPassword):
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "School Admin initial password has appeared in known data breaches. Choose a different password."})
		case errors.Is(err, services.ErrPasswordBreachCheckUnavailable):
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Password safety verification is temporarily unavailable. Try again shortly."})
		case errors.Is(err, repository.ErrSchoolCodeTaken):
			c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "School code is already in use"})
		case errors.Is(err, repository.ErrUsernameTaken):
			c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "School Admin username is already in use"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to create school"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"school": gin.H{
			"id": result.SchoolID,
			"name": result.SchoolName,
			"code": result.SchoolCode,
			"status": result.SchoolStatus,
		},
		"school_admin": gin.H{
			"principal_id": result.AdminPrincipalID,
			"username": result.AdminUsername,
		},
	})
}
