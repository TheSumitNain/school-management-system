package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/models"
	"school-management-system/internal/services"
)

// ParentAuthenticationService is the HTTP boundary for existing Parent
// account OTP login. RequestOTP must keep unknown/ineligible mobiles
// indistinguishable from eligible accounts.
type ParentAuthenticationService interface {
	RequestOTP(ctx context.Context, mobileE164, clientIP string) error
	VerifyOTP(ctx context.Context, mobileE164, code string) (*models.AuthPrincipal, error)
}

// ParentTokenService issues access tokens bound to the refresh-session family.
type ParentTokenService interface {
	GenerateAccessTokenForSession(principalID, tokenVersion int64, principalType, sessionID string) (string, error)
}

// ParentSessionService creates family-bound refresh sessions. Revoke is used
// to clean up the refresh token if access-token signing fails after issuance.
type ParentSessionService interface {
	IssueWithFamily(ctx context.Context, principal *models.AuthPrincipal) (refreshToken, familyID string, err error)
	Revoke(ctx context.Context, refreshToken string) error
}

// ParentAuthHandler issues the same principal-based, family-bound credentials
// used by staff and Platform Admin authentication.
type ParentAuthHandler struct {
	authentication ParentAuthenticationService
	tokens         ParentTokenService
	sessions       ParentSessionService
}

func NewParentAuthHandler(
	authentication ParentAuthenticationService,
	tokens ParentTokenService,
	sessions ParentSessionService,
) *ParentAuthHandler {
	return &ParentAuthHandler{
		authentication: authentication,
		tokens:         tokens,
		sessions:       sessions,
	}
}

type parentOTPRequest struct {
	Mobile string `json:"mobile_e164"`
}

type parentOTPVerifyRequest struct {
	Mobile string `json:"mobile_e164"`
	Code   string `json:"code"`
}

// RequestOTP always returns the same accepted response for a syntactically
// valid E.164 mobile, regardless of account eligibility or delivery outcome.
// Operational failures are logged without attaching the raw error to Gin's
// context, because the default Gin logger may serialize context errors.
func (h *ParentAuthHandler) RequestOTP(c *gin.Context) {
	var req parentOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAuthError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	mobile, err := services.NormalizeParentMobile(req.Mobile)
	if err != nil {
		writeAuthError(c, http.StatusBadRequest, "mobile_e164 must be a valid E.164 mobile number")
		return
	}
	if h == nil || h.authentication == nil {
		c.JSON(http.StatusAccepted, gin.H{
			"status": "accepted",
			"message": "If the mobile number is eligible, an OTP has been sent.",
		})
		return
	}
	if err := h.authentication.RequestOTP(c.Request.Context(), mobile, c.ClientIP()); err != nil {
		// Do not log the underlying error: provider and database errors may
		// contain sensitive operational details or personal information.
		log.Print("Parent OTP request failed")
	}
	c.JSON(http.StatusAccepted, gin.H{
		"status": "accepted",
		"message": "If the mobile number is eligible, an OTP has been sent.",
	})
}

// VerifyOTP consumes a one-time challenge and then issues the canonical
// principal-based JWT and refresh session. Both credentials carry the same
// immutable refresh-family ID so middleware can reject revoked sessions.
func (h *ParentAuthHandler) VerifyOTP(c *gin.Context) {
	var req parentOTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAuthError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	mobile, err := services.NormalizeParentMobile(req.Mobile)
	if err != nil || strings.TrimSpace(req.Code) == "" {
		writeAuthError(c, http.StatusUnauthorized, "Invalid or expired OTP")
		return
	}
	if h == nil || h.authentication == nil || h.tokens == nil || h.sessions == nil {
		writeAuthError(c, http.StatusServiceUnavailable, "Parent authentication is unavailable")
		return
	}
	principal, err := h.authentication.VerifyOTP(c.Request.Context(), mobile, strings.TrimSpace(req.Code))
	if err != nil || !validParentLoginPrincipal(principal) {
		writeAuthError(c, http.StatusUnauthorized, "Invalid or expired OTP")
		return
	}

	refreshToken, familyID, err := h.sessions.IssueWithFamily(c.Request.Context(), principal)
	if err != nil {
		if errors.Is(err, services.ErrSessionPrincipalVersionChanged) ||
			errors.Is(err, services.ErrRefreshSessionInactive) {
			writeAuthError(c, http.StatusUnauthorized, "Invalid or expired OTP")
			return
		}
		writeAuthError(c, http.StatusInternalServerError, "Unable to complete authentication")
		return
	}
	accessToken, err := h.tokens.GenerateAccessTokenForSession(
		principal.ID, principal.TokenVersion, principal.Type, familyID,
	)
	if err != nil {
		// Avoid leaving an active refresh credential behind when its matching
		// access token could not be created.
		_ = h.sessions.Revoke(c.Request.Context(), refreshToken)
		writeAuthError(c, http.StatusInternalServerError, "Unable to complete authentication")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"access_token": accessToken,
		"refresh_token": refreshToken,
		"token_type": "Bearer",
		"principal": toPrincipalResponse(principal),
	})
}

func validParentLoginPrincipal(principal *models.AuthPrincipal) bool {
	return principal != nil &&
		principal.ID > 0 &&
		principal.Type == "parent" &&
		principal.Status == "active" &&
		principal.TokenVersion > 0 &&
		principal.MobileVerified
}
