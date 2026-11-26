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
type staffDirectoryPageService interface {
	ListDirectoryPage(ctx context.Context, actorPrincipalID, schoolID int64, filter models.StaffDirectoryFilter) (*models.StaffDirectoryPage, error)
}

type staffDirectorySummaryService interface {
	GetDirectorySummary(ctx context.Context, actorPrincipalID, schoolID int64) (*models.StaffDirectorySummary, error)
}

type staffProfileUpdateService interface {
    UpdateProfile(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, profile models.StaffProfile) (*models.User, error)
}

func parseStaffJoiningDate(value string) (*time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return nil, true
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01-02") != value {
		return nil, false
	}
	return &parsed, true
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
		StaffType string `json:"staff_type"`
		JoiningDate string `json:"joining_date"`
	}
	if err := c.ShouldBindJSON(&request); err != nil ||
		strings.TrimSpace(request.Username) == "" || request.InitialPassword == "" ||
		strings.TrimSpace(request.EmployeeCode) == "" || strings.TrimSpace(request.FullName) == "" ||
		strings.TrimSpace(request.Designation) == "" || strings.TrimSpace(request.MobileNumber) == "" || strings.TrimSpace(request.StaffType) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "username, initial_password, staff_type, full_name, employee_code, designation and mobile_number are required"})
		return
	}
	joiningDate, validJoiningDate := parseStaffJoiningDate(request.JoiningDate)
	if !validJoiningDate {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "joining_date must use YYYY-MM-DD"})
		return
	}
	profileService, ok := h.service.(staffProfileCreateService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Employee profile creation is unavailable"})
		return
	}
	profile := models.StaffProfile{StaffType: request.StaffType, EmployeeCode: request.EmployeeCode, FullName: request.FullName,
		Designation: request.Designation, MobileNumber: request.MobileNumber, Email: request.Email,
		JoiningDate: joiningDate, EmploymentStatus: "active"}
	user, err := profileService.CreateWithProfile(c.Request.Context(), actorID, schoolID,
		request.Username, request.InitialPassword, profile)
	switch {
	case err == nil && user == nil:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to create staff account"})
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
                "employment_status": user.EmploymentStatus,
                "staff_type": user.StaffType, "profile_complete": user.ProfileComplete,
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
	if c.Query("limit") != "" || c.Query("offset") != "" || c.Query("search") != "" || c.Query("employment_status") != "" || c.Query("account_status") != "" || c.Query("profile_status") != "" {
		h.ListDirectoryPage(c)
		return
	}
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
            "employment_status": user.EmploymentStatus, "staff_type": user.StaffType, "profile_complete": user.ProfileComplete,
				"protected_school_admin": user.ProtectedSchoolAdmin,
		})
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "staff": staff})
}

func (h *StaffHandler) GetDirectorySummary(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff directory summary is unavailable"})
		return
	}
	summaryService, ok := h.service.(staffDirectorySummaryService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff directory summary is unavailable"})
		return
	}
	summary, err := summaryService.GetDirectorySummary(c.Request.Context(), actorID, schoolID)
	switch {
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff directory summary is not authorized"})
	case err != nil || summary == nil:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to load staff directory summary"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "success", "summary": summary})
	}
}

func (h *StaffHandler) ListDirectoryPage(c *gin.Context) {
	actorValue, actorExists := c.Get(middleware.ContextPrincipalIDKey)
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get(middleware.ContextSchoolIDKey)
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 || !schoolExists || !schoolOK || schoolID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "School staff authentication required"})
		return
	}
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff directory is unavailable"})
		return
	}
	pageService, ok := h.service.(staffDirectoryPageService)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "Staff directory pagination is unavailable"})
		return
	}
	parseQueryInt := func(key string, fallback int) (int, bool) {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" { return fallback, true }
		value, err := strconv.Atoi(raw)
		return value, err == nil
	}
	limit, limitOK := parseQueryInt("limit", 50)
	offset, offsetOK := parseQueryInt("offset", 0)
	if !limitOK || !offsetOK {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "limit and offset must be valid integers"})
		return
	}
	filter := models.StaffDirectoryFilter{
		Search: c.Query("search"),
		EmploymentStatus: c.Query("employment_status"),
		AccountStatus: c.Query("account_status"),
		ProfileStatus: c.Query("profile_status"),
		Limit: limit,
		Offset: offset,
	}
	page, err := pageService.ListDirectoryPage(c.Request.Context(), actorID, schoolID, filter)
	switch {
	case errors.Is(err, services.ErrInvalidStaffAccount):
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff directory filters or pagination"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff directory access is not authorized"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to load staff directory"})
	case page == nil:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to load staff directory"})
	default:
		staff := make([]gin.H, 0, len(page.Staff))
		for _, user := range page.Staff {
			staff = append(staff, gin.H{
				"id": user.ID, "principal_id": user.PrincipalID, "school_id": user.SchoolID,
				"username": user.Username, "status": user.Status, "must_change_password": user.MustChangePassword,
				"created_at": user.CreatedAt, "updated_at": user.UpdatedAt,
				"employee_code": user.EmployeeCode, "full_name": user.FullName, "designation": user.Designation,
				"mobile_number": user.MobileNumber, "email": user.Email, "joining_date": user.JoiningDate,
				"employment_status": user.EmploymentStatus, "staff_type": user.StaffType, "profile_complete": user.ProfileComplete,
				"protected_school_admin": user.ProtectedSchoolAdmin,
			})
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "staff": staff, "total": page.Total, "limit": page.Limit, "offset": page.Offset})
	}
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
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Staff account not found in this school"})
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
        StaffType string `json:"staff_type"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid staff update request"})
		return
	}
    hasProfile := strings.TrimSpace(request.EmployeeCode) != "" || strings.TrimSpace(request.FullName) != "" ||
        strings.TrimSpace(request.Designation) != "" || strings.TrimSpace(request.MobileNumber) != "" ||
        strings.TrimSpace(request.Email) != "" || strings.TrimSpace(request.JoiningDate) != "" ||
        strings.TrimSpace(request.EmploymentStatus) != "" || strings.TrimSpace(request.StaffType) != ""
    var user *models.User
    if hasProfile {
        updater, ok := h.service.(staffProfileUpdateService)
        if !ok {
            c.JSON(http.StatusServiceUnavailable, gin.H{"status":"error","message":"Employee profile update is unavailable"})
            return
        }
        joiningDate, validJoiningDate := parseStaffJoiningDate(request.JoiningDate)
        if !validJoiningDate {
            c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"joining_date must use YYYY-MM-DD"})
            return
        }
        profile := models.StaffProfile{EmployeeCode:request.EmployeeCode, FullName:request.FullName, Designation:request.Designation,
            MobileNumber:request.MobileNumber, Email:request.Email, JoiningDate:joiningDate, EmploymentStatus:request.EmploymentStatus, StaffType:request.StaffType}
        user, err = updater.UpdateProfile(c.Request.Context(), actorID, schoolID, targetID, profile)
    } else {
        if strings.TrimSpace(request.Username) == "" {
            c.JSON(http.StatusBadRequest, gin.H{"status":"error","message":"username or employee profile fields are required"})
            return
        }
        user, err = h.service.Update(c.Request.Context(), actorID, schoolID, targetID, request.Username)
    }
	switch {
	case err == nil && user == nil:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to update staff account"})
	case err == nil:
        message := "Employee profile updated. Login access follows the employment status."
        if !hasProfile { message = "Staff username updated; active sessions revoked" }
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"staff": gin.H{
				"id": user.ID,
				"principal_id": user.PrincipalID,
				"school_id": user.SchoolID,
				"username": user.Username,
				"status": user.Status,
				"must_change_password": user.MustChangePassword,
                "employee_code": user.EmployeeCode,
                "full_name": user.FullName,
                "designation": user.Designation,
                "mobile_number": user.MobileNumber,
                "email": user.Email,
                "joining_date": user.JoiningDate,
                "employment_status": user.EmploymentStatus,
                "staff_type": user.StaffType,
                "profile_complete": user.ProfileComplete,
			},
			"message": message,
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
    case errors.Is(err, repository.ErrStaffProfileIncomplete):
        c.JSON(http.StatusConflict, gin.H{"status":"error","message":"Complete the employee profile before enabling this login account"})
    case errors.Is(err, repository.ErrStaffProfileInactive):
        c.JSON(http.StatusConflict, gin.H{"status":"error","message":"Reactivate the employee profile before enabling this login account"})
	case errors.Is(err, repository.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "message": "Disabled staff account not found in this school"})
	case errors.Is(err, repository.ErrRolePermissionDenied):
		c.JSON(http.StatusForbidden, gin.H{"status": "error", "message": "Staff account re-enable is not authorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Unable to enable staff account"})
	}
}
