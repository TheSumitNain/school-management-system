package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/models"
	"school-management-system/internal/services"
)

type AuthService interface {
	Authenticate(
		ctx context.Context,
		schoolID int64,
		username string,
		password string,
	) (*models.User, error)
}

type TokenService interface {
	GenerateAccessToken(userID, schoolID int64) (string, error)
}

type SessionService interface {
	Issue(ctx context.Context, userID int64) (string, error)
	Rotate(ctx context.Context, refreshToken string) (*models.User, string, error)
	Revoke(ctx context.Context, refreshToken string) error
}

type AuthHandler struct {
	authService    AuthService
	tokenService   TokenService
	sessionService SessionService
}

func NewAuthHandler(
	authService AuthService,
	tokenService TokenService,
	sessionService SessionService,
) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		tokenService:   tokenService,
		sessionService: sessionService,
	}
}

type loginRequest struct {
	SchoolID int64  `json:"school_id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type loginUserResponse struct {
	ID       int64  `json:"id"`
	SchoolID int64  `json:"school_id"`
	Username string `json:"username"`
	Status   string `json:"status"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body",
		})
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	if req.SchoolID <= 0 ||
		req.Username == "" ||
		req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "school_id, username and password are required",
		})
		return
	}

	user, err := h.authService.Authenticate(
		c.Request.Context(),
		req.SchoolID,
		req.Username,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid username or password",
			})
			return
		}

		if errors.Is(err, services.ErrUserInactive) ||
			errors.Is(err, services.ErrSchoolInactive) {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Account or school is inactive",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Unable to log in",
		})
		return
	}

	accessToken, err := h.tokenService.GenerateAccessToken(
		user.ID,
		user.SchoolID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Unable to log in",
		})
		return
	}

	refreshToken, err := h.sessionService.Issue(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Unable to log in",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"user": loginUserResponse{
			ID:       user.ID,
			SchoolID: user.SchoolID,
			Username: user.Username,
			Status:   user.Status,
		},
	})
}

// Refresh exchanges a valid refresh token for a new access token and one-time refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "refresh_token is required",
		})
		return
	}

	user, refreshToken, err := h.sessionService.Rotate(
		c.Request.Context(),
		req.RefreshToken,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefreshToken),
			errors.Is(err, services.ErrRefreshTokenReuse):
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid or expired refresh token",
			})
		case errors.Is(err, services.ErrRefreshSessionInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Account or school is inactive",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Unable to refresh session",
			})
		}
		return
	}

	accessToken, err := h.tokenService.GenerateAccessToken(user.ID, user.SchoolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Unable to refresh session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"user": loginUserResponse{
			ID:       user.ID,
			SchoolID: user.SchoolID,
			Username: user.Username,
			Status:   user.Status,
		},
	})
}

// Logout revokes the supplied refresh token. Repeating logout is safe.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "refresh_token is required",
		})
		return
	}

	if err := h.sessionService.Revoke(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Unable to log out",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Logged out successfully",
	})
}
