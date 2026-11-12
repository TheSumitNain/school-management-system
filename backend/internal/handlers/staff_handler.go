package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Update(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, username string) (*models.User, error)
	Enable(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, temporaryPassword string) error
}

type staffProfileCreateService interface {
	CreateWithProfile(ctx context.Context, actorPrincipalID, schoolID int64, username, initialPassword string, profile models.StaffProfile) (*models.User, error)
}
type staffProfileUpdateService interface {
    UpdateProfile(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, profile models.StaffProfile) (*models.User, error)
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
        EmployeeCode string `json:"employee_code"`
        FullName string `json:"full_name"`
        Designation string `json:"designation"`
        MobileNumber string `json:"mobile_number"`
        Email string `json:"email"`
        JoiningDate string `json:"joining_date"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Username) == "" || request.InitialPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "username and initial_password are required"})
		return
	}
    var user *models.User
    var err error
    hasProfile := strings.TrimSpace(request.EmployeeCode) != "" || strings.TrimSpace(request.FullName) != "" ||
        strings.TrimSpace(request.Designation) != "" || strings.TrimSpace(request.MobileNumber) != "" ||
        strings.TrimSpace(request.Email) != "" || strings.TrimSpace(request.JoiningDate) != ""
    if hasProfile {
        profileService, ok := h.service.(staffProfileCreateService)
        if !ok {
            c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Employee profile creation is unavailable"})
            return
        }
        var joiningDate *time.Time
        if strings.TrimSpace(request.JoiningDate) != "" {
            parsed, parseErr := time.Parse("2006-01-02", request.JoiningDate)
            if parseErr != nil {
                c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"joining_date must use YYYY-MM-DD"})
                return
            }
            joiningDate = &parsed
        }
        profile := models.StaffProfile{EmployeeCode:request.EmployeeCode, FullName:request.FullName, Designation:request.Designation,
            MobileNumber:request.MobileNumber, Email:request.Email, JoiningDate:joiningDate, EmploymentStatus:"active"}
        user, err = profileService.CreateWithProfile(c.Request.Context(), actorID, schoolID, request.Username, request.InitialPassword, profile)
    } else {
        user, err = h.service.Create(c.Request.Context(), actorID, schoolID, request.Username, request.InitialPassword)
    }
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
                "employee_code": user.EmployeeCode, "full_name": user.FullName, "designation": user.Designation,
                "mobile_number": user.MobileNumber, "email": user.Email, "joining_date": user.JoiningDate,
                "employment_status": user.EmploymentStatus, "profile_complete": user.ProfileComplete,
			},
		})
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Enter valid employee details and a username containing 3–32 lowercase letters, digits, dots, underscores, or hyphens"})
	case errors.Is(err, services.ErrInvalidPassword):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Initial password must meet the password policy"})
	case errors.Is(err, services.ErrCompromisedPassword):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "This password has appeared in known data breaches. Choose a different password."})
	case errors.Is(err, services.ErrPasswordBreachCheckUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Password safety verification is temporarily unavailable. Try again shortly."})
    case errors.Is(err, repository.ErrEmployeeCodeTaken):
        c.JSON(http.StatusConflict, gin.H{"status":"error","message":"This employee code is already used in this school"})
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
            "employee_code": user.EmployeeCode, "full_name": user.FullName, "designation": user.Designation,
            "mobile_number": user.MobileNumber, "email": user.Email, "joining_date": user.JoiningDate,
            "employment_status": user.EmploymentStatus, "profile_complete": user.ProfileComplete,
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


// Update changes an active staff account's login username.
func (h *StaffHandler) Update(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff account update is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	var request struct {
		Username string `json:"username"`
        EmployeeCode string `json:"employee_code"`
        FullName string `json:"full_name"`
        Designation string `json:"designation"`
        MobileNumber string `json:"mobile_number"`
        Email string `json:"email"`
        JoiningDate string `json:"joining_date"`
        EmploymentStatus string `json:"employment_status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff update request"})
		return
	}
    hasProfile := strings.TrimSpace(request.EmployeeCode) != "" || strings.TrimSpace(request.FullName) != "" ||
        strings.TrimSpace(request.Designation) != "" || strings.TrimSpace(request.MobileNumber) != "" ||
        strings.TrimSpace(request.Email) != "" || strings.TrimSpace(request.JoiningDate) != "" ||
        strings.TrimSpace(request.EmploymentStatus) != ""
    var user *models.User
    if hasProfile {
        updater, ok := h.service.(staffProfileUpdateService)
        if !ok {
            c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Employee profile update is unavailable"})
            return
        }
        var joiningDate *time.Time
        if strings.TrimSpace(request.JoiningDate) != "" {
            parsed, parseErr := time.Parse("2006-01-02", request.JoiningDate)
            if parseErr != nil {
                c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"joining_date must use YYYY-MM-DD"})
                return
            }
            joiningDate = &parsed
        }
        profile := models.StaffProfile{EmployeeCode:request.EmployeeCode, FullName:request.FullName, Designation:request.Designation,
            MobileNumber:request.MobileNumber, Email:request.Email, JoiningDate:joiningDate, EmploymentStatus:request.EmploymentStatus}
        user, err = updater.UpdateProfile(c.Request.Context(), actorID, schoolID, targetID, profile)
    } else {
        if strings.TrimSpace(request.Username) == "" {
            c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"username or employee profile fields are required"})
            return
        }
        user, err = h.service.Update(c.Request.Context(), actorID, schoolID, targetID, request.Username)
    }
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"staff": gin.H{
				"id": user.ID,
				"principal_id": user.PrincipalID,
				"school_id": user.SchoolID,
				"username": user.Username,
				"status": user.Status,
				"must_change_password": user.MustChangePassword,
			},
			"message": "Staff username updated; active sessions revoked",
		})
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff account update request"})
    case errors.Is(err, repository.ErrEmployeeCodeTaken):
        c.JSON(http.StatusConflict, gin.H{"status":"error","message":"This employee code is already used in this school"})
	case errors.Is(err, repository.ErrUsernameTaken):
		c.JSON(http.StatusConflict, gin.H{"status": "error", "message": "This username is already in use"})
	case errors.Is(err, repository.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Active staff account not found in this school"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account update is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to update staff account"})
	}
}

// Enable reactivates a disabled account with a new temporary password.
func (h *StaffHandler) Enable(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff account re-enable is unavailable"})
		return
	}
	targetID, err := strconv.ParseInt(c.Param("principalID"), 10, 64)
	if err != nil || targetID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid target principal ID"})
		return
	}
	var request struct {
		TemporaryPassword string `json:"temporary_password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.TemporaryPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "temporary_password is required"})
		return
	}
	err = h.service.Enable(c.Request.Context(), actorID, schoolID, targetID, request.TemporaryPassword)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"message": "Staff account enabled with a temporary password; password change is required at next login",
		})
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff account re-enable request"})
	case errors.Is(err, services.ErrInvalidPassword):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Temporary password must meet the password policy"})
	case errors.Is(err, services.ErrCompromisedPassword):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "This password has appeared in known data breaches. Choose a different password."})
	case errors.Is(err, services.ErrPasswordBreachCheckUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Password safety verification is temporarily unavailable. Try again shortly."})
	case errors.Is(err, repository.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Disabled staff account not found in this school"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account re-enable is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to enable staff account"})
	}
}
