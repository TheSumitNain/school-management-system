package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"school-management-system/internal/middleware"
	"school-management-system/internal/models"
	"school-management-system/internal/services"
)

// AuthService authenticates a globally unique staff username. School
// selection is deliberately absent from the V4 login request.
type AuthService interface {
	AuthenticateStaff(ctx context.Context, username, password string) (*models.AuthPrincipal, error)
}
type StaffPasswordService interface {
	ChangePassword(ctx context.Context, principalID int64, currentPassword, newPassword string) error
	ResetPassword(ctx context.Context, actorPrincipalID, schoolID, targetPrincipalID int64, newPassword string) error
}

type PlatformAdminAuthService interface {
	Authenticate(ctx context.Context, email, password string) (*models.AuthPrincipal, error)
}

type TokenService interface {
	GenerateAccessToken(principalID, tokenVersion int64, principalType string) (string, error)
}

type SessionService interface {
	Issue(ctx context.Context, principal *models.AuthPrincipal) (string, error)
	Rotate(ctx context.Context, refreshToken string) (*models.AuthPrincipal, string, error)
	Revoke(ctx context.Context, refreshToken string) error
}

type LoginIPRateLimiter interface {
	AllowIPAttempt(ctx context.Context, clientIP string) (bool, error)
}

type MFAService interface {
	BeginAuthentication(ctx context.Context, principal *models.AuthPrincipal, accountName string) (*services.MFAStart, error)
	CompleteEnrollment(ctx context.Context, challengeToken, code string) (int64, int64, []string, error)
	VerifyLoginChallenge(ctx context.Context, challengeToken, code string) (int64, int64, error)
	BeginRecoveryCodeRegeneration(ctx context.Context, principalID, tokenVersion int64) (*services.MFAStart, error)
	RegenerateRecoveryCodes(ctx context.Context, challengeToken, code string) ([]string, error)
}
type MFAPrincipalLoader interface {
	LoadPrincipalForAuthentication(ctx context.Context, principalID int64) (*models.AuthPrincipal, error)
}
type AuthHandler struct {
	authService AuthService
	platformAdminAuthService PlatformAdminAuthService
	tokenService TokenService
	sessionService SessionService
	ipRateLimiter LoginIPRateLimiter
	mfaService MFAService
	principalLoader MFAPrincipalLoader
	staffPasswordService StaffPasswordService
}

func NewAuthHandler(authService AuthService, tokenService TokenService, sessionService SessionService, ipRateLimiters ...LoginIPRateLimiter) *AuthHandler {
	h := &AuthHandler{authService: authService, tokenService: tokenService, sessionService: sessionService}
	if len(ipRateLimiters) > 0 {
		h.ipRateLimiter = ipRateLimiters[0]
	}
	return h
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type platformAdminLoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}
type mfaVerifyRequest struct {
	ChallengeToken string `json:"challenge_token"`
	Code string `json:"code"`
}

type mfaRecoveryCodeRegenerationRequest struct {
	ChallengeToken string `json:"challenge_token"`
	Code string `json:"code"`
}

type principalResponse struct {
	ID       int64  `json:"principal_id"`
	Type     string `json:"principal_type"`
	Status   string `json:"status"`
	SchoolID int64  `json:"school_id,omitempty"`
	Username string `json:"username,omitempty"`
	MustChangePassword bool `json:"must_change_password,omitempty"`
}

func (h *AuthHandler) SetStaffPasswordService(service StaffPasswordService) {
	h.staffPasswordService = service
}

func (h *AuthHandler) SetPlatformAdminAuthService(service PlatformAdminAuthService) {
	h.platformAdminAuthService = service
}

func (h *AuthHandler) SetMFAService(mfa MFAService, loader MFAPrincipalLoader) {
	h.mfaService = mfa
	h.principalLoader = loader
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.ipRateLimiter != nil {
		allowed, err := h.ipRateLimiter.AllowIPAttempt(c.Request.Context(), c.ClientIP())
		if err != nil {
			writeAuthError(c, http.StatusInternalServerError, "Unable to process login")
			return
		}
		if !allowed {
			c.Header("Retry-After", "900")
			writeAuthError(c, http.StatusTooManyRequests, "Too many login requests from this network. Try again in 15 minutes.")
			return
		}
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeAuthError(c, http.StatusBadRequest, "Invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeAuthError(c, http.StatusBadRequest, "username and password are required")
		return
	}
	if h.authService == nil || h.tokenService == nil || h.sessionService == nil {
		writeAuthError(c, http.StatusInternalServerError, "Authentication service unavailable")
		return
	}
	principal, err := h.authService.AuthenticateStaff(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrLoginLocked) {
			c.Header("Retry-After", "900")
			writeAuthError(c, http.StatusTooManyRequests, "Too many failed login attempts. Try again in 15 minutes.")
			return
		}
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeAuthError(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		if errors.Is(err, services.ErrUserInactive) || errors.Is(err, services.ErrSchoolInactive) {
			writeAuthError(c, http.StatusForbidden, "Account or school is inactive")
			return
		}
		writeAuthError(c, http.StatusInternalServerError, "Unable to log in")
		return
	}
	if !validStaffPrincipal(principal) {
		writeAuthError(c, http.StatusInternalServerError, "Unable to log in")
		return
	}
	if h.mfaService == nil {
		writeAuthError(c,http.StatusServiceUnavailable,"Multi-factor authentication is unavailable")
		return
	}
	if h.mfaService != nil {
		mfaStart, mfaErr := h.mfaService.BeginAuthentication(c.Request.Context(), principal, principal.Username)
		if mfaErr != nil {
			if errors.Is(mfaErr, services.ErrSessionPrincipalVersionChanged) {
				writeAuthError(c, http.StatusUnauthorized, "Credentials changed during sign-in. Please sign in again.")
				return
			}
			writeAuthError(c, http.StatusInternalServerError, "Unable to start multi-factor authentication")
			return
		}
		if mfaStart.Required {
			c.JSON(http.StatusAccepted, gin.H{"status":"mfa_required","challenge_token":mfaStart.ChallengeToken,
				"enrollment_required":mfaStart.EnrollmentRequired,"secret":mfaStart.Secret,
				"provisioning_uri":mfaStart.ProvisioningURI,"expires_at":mfaStart.ExpiresAt})
			return
		}
	}
	accessToken, refreshToken, err := issueSessionTokens(c.Request.Context(), h.tokenService, h.sessionService, principal)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSessionPrincipalVersionChanged):
			writeAuthError(c, http.StatusUnauthorized, "Credentials changed during sign-in. Please sign in again.")
		case errors.Is(err, services.ErrRefreshSessionInactive):
			writeAuthError(c, http.StatusForbidden, "Account or school is inactive")
		default:
			writeAuthError(c, http.StatusInternalServerError, "Unable to log in")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success", "access_token": accessToken, "refresh_token": refreshToken,
		"token_type": "Bearer", "principal": toPrincipalResponse(principal),
	})
}



func (h *AuthHandler) PlatformAdminLogin(c *gin.Context) {
	if h.ipRateLimiter!=nil {
		allowed,err:=h.ipRateLimiter.AllowIPAttempt(c.Request.Context(),c.ClientIP())
		if err!=nil { writeAuthError(c,http.StatusInternalServerError,"Unable to process login"); return }
		if !allowed { c.Header("Retry-After","900"); writeAuthError(c,http.StatusTooManyRequests,"Too many login requests from this network. Try again in 15 minutes."); return }
	}
	var req platformAdminLoginRequest
	if err:=c.ShouldBindJSON(&req); err!=nil { writeAuthError(c,http.StatusBadRequest,"Invalid request body"); return }
	req.Email=strings.TrimSpace(req.Email)
	if req.Email=="" || req.Password=="" { writeAuthError(c,http.StatusBadRequest,"email and password are required"); return }
	if h.platformAdminAuthService==nil || h.mfaService==nil {
		writeAuthError(c,http.StatusServiceUnavailable,"Platform Admin authentication is unavailable"); return
	}
	principal,err:=h.platformAdminAuthService.Authenticate(c.Request.Context(),req.Email,req.Password)
	if err!=nil {
		if errors.Is(err,services.ErrLoginLocked) { c.Header("Retry-After","900"); writeAuthError(c,http.StatusTooManyRequests,"Too many failed login attempts. Try again in 15 minutes."); return }
		if errors.Is(err,services.ErrInvalidCredentials) { writeAuthError(c,http.StatusUnauthorized,"Invalid email or password"); return }
		writeAuthError(c,http.StatusInternalServerError,"Unable to log in"); return
	}
	if !validPrincipal(principal) || principal.Type!="platform_admin" { writeAuthError(c,http.StatusInternalServerError,"Unable to log in"); return }
	mfaStart,err:=h.mfaService.BeginAuthentication(c.Request.Context(),principal,req.Email)
	if err!=nil {
		if errors.Is(err, services.ErrSessionPrincipalVersionChanged) {
			writeAuthError(c, http.StatusUnauthorized, "Credentials changed during sign-in. Please sign in again.")
			return
		}
		writeAuthError(c,http.StatusInternalServerError,"Unable to start multi-factor authentication"); return
	}
	if !mfaStart.Required { writeAuthError(c,http.StatusForbidden,"Platform Admin MFA policy is not enabled"); return }
	c.JSON(http.StatusAccepted,gin.H{"status":"mfa_required","challenge_token":mfaStart.ChallengeToken,
		"enrollment_required":mfaStart.EnrollmentRequired,"secret":mfaStart.Secret,
		"provisioning_uri":mfaStart.ProvisioningURI,"expires_at":mfaStart.ExpiresAt})
}

func (h *AuthHandler) CompleteMFAEnrollment(c *gin.Context) {
	var req mfaVerifyRequest
	if err:=c.ShouldBindJSON(&req); err!=nil || strings.TrimSpace(req.ChallengeToken)=="" || strings.TrimSpace(req.Code)=="" {
		writeAuthError(c,http.StatusBadRequest,"challenge_token and code are required"); return
	}
	principalID, expectedTokenVersion, recoveryCodes, err := h.mfaService.CompleteEnrollment(c.Request.Context(), req.ChallengeToken, req.Code)
	if err != nil { writeMFAError(c, err); return }
	h.issueMFASession(c, principalID, expectedTokenVersion, recoveryCodes)
}

func (h *AuthHandler) VerifyMFA(c *gin.Context) {
	var req mfaVerifyRequest
	if err:=c.ShouldBindJSON(&req); err!=nil || strings.TrimSpace(req.ChallengeToken)=="" || strings.TrimSpace(req.Code)=="" {
		writeAuthError(c,http.StatusBadRequest,"challenge_token and code are required"); return
	}
	principalID, expectedTokenVersion, err:=h.mfaService.VerifyLoginChallenge(c.Request.Context(),req.ChallengeToken,req.Code)
	if err!=nil { writeMFAError(c,err); return }
	h.issueMFASession(c, principalID, expectedTokenVersion, nil)
}

func (h *AuthHandler) BeginMFARecoveryCodeRegeneration(c *gin.Context) {
	value, exists := c.Get("auth_principal_id")
	principalID, ok := value.(int64)
	if !exists || !ok || principalID <= 0 {
		writeAuthError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	if h.mfaService == nil {
		writeAuthError(c, http.StatusServiceUnavailable, "Multi-factor authentication is unavailable")
		return
	}
	tokenVersionValue, tokenVersionExists := c.Get(middleware.ContextTokenVersionKey)
	tokenVersion, tokenVersionOK := tokenVersionValue.(int64)
	if !tokenVersionExists || !tokenVersionOK || tokenVersion <= 0 {
		writeAuthError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	challenge, err := h.mfaService.BeginRecoveryCodeRegeneration(c.Request.Context(), principalID, tokenVersion)
	if err != nil { writeMFAError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"status": "challenge_required", "challenge_token": challenge.ChallengeToken, "expires_at": challenge.ExpiresAt})
}

func (h *AuthHandler) RegenerateMFARecoveryCodes(c *gin.Context) {
	var req mfaRecoveryCodeRegenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ChallengeToken) == "" || strings.TrimSpace(req.Code) == "" {
		writeAuthError(c, http.StatusBadRequest, "challenge_token and code are required")
		return
	}
	if h.mfaService == nil {
		writeAuthError(c, http.StatusServiceUnavailable, "Multi-factor authentication is unavailable")
		return
	}
	codes, err := h.mfaService.RegenerateRecoveryCodes(c.Request.Context(), strings.TrimSpace(req.ChallengeToken), strings.TrimSpace(req.Code))
	if err != nil { writeMFAError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"status": "success", "recovery_codes": codes})
}

func (h *AuthHandler) issueMFASession(c *gin.Context, principalID, expectedTokenVersion int64, recoveryCodes []string) {
	if h.principalLoader==nil || h.tokenService==nil || h.sessionService==nil {
		writeAuthError(c,http.StatusInternalServerError,"Authentication service unavailable"); return
	}
	principal,err:=h.principalLoader.LoadPrincipalForAuthentication(c.Request.Context(),principalID)
	if err!=nil || !validPrincipal(principal) || expectedTokenVersion <= 0 || principal.TokenVersion != expectedTokenVersion {
		writeAuthError(c,http.StatusUnauthorized,"Authentication could not be completed"); return
	}
	accessToken,refreshToken,err:=issueSessionTokens(c.Request.Context(),h.tokenService,h.sessionService,principal)
	if err!=nil {
		if errors.Is(err, services.ErrSessionPrincipalVersionChanged) {
			writeAuthError(c, http.StatusUnauthorized, "Credentials changed during sign-in. Please sign in again.")
			return
		}
		if errors.Is(err, services.ErrRefreshSessionInactive) {
			writeAuthError(c, http.StatusForbidden, "Account or school is inactive")
			return
		}
		writeAuthError(c,http.StatusInternalServerError,"Unable to complete authentication")
		return
	}
	response := gin.H{"status":"success","access_token":accessToken,"refresh_token":refreshToken,"token_type":"Bearer","principal":toPrincipalResponse(principal)}
	if len(recoveryCodes) > 0 { response["recovery_codes"] = recoveryCodes }
	c.JSON(http.StatusOK, response)
}

func writeMFAError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrSessionPrincipalVersionChanged) {
		writeAuthError(c, http.StatusUnauthorized, "Credentials changed. Please sign in again.")
		return
	}
	if errors.Is(err,services.ErrMFAInvalid) { writeAuthError(c,http.StatusUnauthorized,"Invalid or expired MFA challenge or code"); return }
	writeAuthError(c,http.StatusInternalServerError,"Unable to complete multi-factor authentication")
}

type changeStaffPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword string `json:"new_password"`
}

// ChangeStaffPassword is the only authenticated operation available to a
// staff principal while a temporary password is active.
func (h *AuthHandler) ChangeStaffPassword(c *gin.Context) {
	if h.staffPasswordService == nil {
		writeAuthError(c, http.StatusServiceUnavailable, "Password change is unavailable")
		return
	}
	value, exists := c.Get("auth_principal_id")
	principalID, ok := value.(int64)
	if !exists || !ok || principalID <= 0 {
		writeAuthError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	principalType, _ := c.Get("auth_principal_type")
	if principalType != "staff" {
		writeAuthError(c, http.StatusForbidden, "Only staff accounts can use this password-change endpoint")
		return
	}
	var req changeStaffPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil ||
		strings.TrimSpace(req.CurrentPassword) == "" ||
		strings.TrimSpace(req.NewPassword) == "" {
		writeAuthError(c, http.StatusBadRequest, "current_password and new_password are required")
		return
	}
	err := h.staffPasswordService.ChangePassword(
		c.Request.Context(), principalID, req.CurrentPassword, req.NewPassword,
	)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"message": "Password changed successfully. Sign in again with your new password.",
		})
	case errors.Is(err, services.ErrInvalidCredentials):
		writeAuthError(c, http.StatusUnauthorized, "Current password is incorrect")
	case errors.Is(err, services.ErrInvalidPassword):
		writeAuthError(c, http.StatusBadRequest, "New password must meet the password policy")
	case errors.Is(err, services.ErrPasswordUnchanged):
		writeAuthError(c, http.StatusBadRequest, "New password must differ from the current password")
	case errors.Is(err, services.ErrPasswordChangeConflict):
		writeAuthError(c, http.StatusConflict, "Password changed concurrently. Sign in again.")
	default:
		writeAuthError(c, http.StatusInternalServerError, "Unable to change password")
	}
}

type resetStaffPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// ResetStaffPassword lets a School Admin reset a staff member in the same
// school. Authorization is checked by middleware and independently by SQL.
func (h *AuthHandler) ResetStaffPassword(c *gin.Context) {
	if h.staffPasswordService == nil {
		writeAuthError(c, http.StatusServiceUnavailable, "Password reset is unavailable")
		return
	}
	actorValue, actorExists := c.Get("auth_principal_id")
	actorID, actorOK := actorValue.(int64)
	schoolValue, schoolExists := c.Get("auth_school_id")
	schoolID, schoolOK := schoolValue.(int64)
	if !actorExists || !actorOK || actorID <= 0 ||
		!schoolExists || !schoolOK || schoolID <= 0 {
		writeAuthError(c, http.StatusUnauthorized, "School staff authentication required")
		return
	}
	targetID, err := strconv.ParseInt(strings.TrimSpace(c.Param("principalID")), 10, 64)
	if err != nil || targetID <= 0 {
		writeAuthError(c, http.StatusBadRequest, "A valid staff principal ID is required")
		return
	}
	var req resetStaffPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NewPassword) == "" {
		writeAuthError(c, http.StatusBadRequest, "new_password is required")
		return
	}
	err = h.staffPasswordService.ResetPassword(c.Request.Context(), actorID, schoolID, targetID, req.NewPassword)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"message": "Staff password reset successfully. The staff member must change it at next login.",
		})
	case errors.Is(err, services.ErrInvalidPassword):
		writeAuthError(c, http.StatusBadRequest, "New password must meet the password policy")
	case errors.Is(err, services.ErrStaffPasswordTargetNotFound):
		writeAuthError(c, http.StatusNotFound, "Staff account was not found in this school")
	case errors.Is(err, services.ErrInvalidRegistration):
		writeAuthError(c, http.StatusBadRequest, "Invalid password reset request")
	default:
		writeAuthError(c, http.StatusInternalServerError, "Unable to reset staff password")
	}
}

// Refresh rotates the opaque refresh token and issues a principal-subject JWT.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		writeAuthError(c, http.StatusBadRequest, "refresh_token is required")
		return
	}
	if h.sessionService == nil || h.tokenService == nil {
		writeAuthError(c, http.StatusInternalServerError, "Authentication service unavailable")
		return
	}

	principal, replacement, familyID, err := rotateSessionTokens(c.Request.Context(), h.tokenService, h.sessionService, strings.TrimSpace(req.RefreshToken))
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefreshToken), errors.Is(err, services.ErrRefreshTokenReuse):
			writeAuthError(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		case errors.Is(err, services.ErrRefreshSessionInactive):
			writeAuthError(c, http.StatusForbidden, "Account or school is inactive")
		default:
			writeAuthError(c, http.StatusInternalServerError, "Unable to refresh session")
		}
		return
	}
	if !validPrincipal(principal) {
		writeAuthError(c, http.StatusInternalServerError, "Unable to refresh session")
		return
	}
	accessToken, err := generateSessionAccessToken(h.tokenService, principal, familyID)
	if err != nil {
		writeAuthError(c, http.StatusInternalServerError, "Unable to refresh session")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success", "access_token": accessToken, "refresh_token": replacement,
		"token_type": "Bearer", "principal": toPrincipalResponse(principal),
	})
}

// Logout is idempotent at the session repository boundary. It revokes the
// refresh-token family; access tokens are rejected after expiry or when the
// principal token_version is incremented.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		writeAuthError(c, http.StatusBadRequest, "refresh_token is required")
		return
	}
	if h.sessionService == nil {
		writeAuthError(c, http.StatusInternalServerError, "Authentication service unavailable")
		return
	}
	if err := h.sessionService.Revoke(c.Request.Context(), strings.TrimSpace(req.RefreshToken)); err != nil {
		writeAuthError(c, http.StatusInternalServerError, "Unable to log out")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Logged out successfully"})
}

func validStaffPrincipal(p *models.AuthPrincipal) bool {
	return validPrincipal(p) && p.Type == "staff" && p.SchoolID > 0 && strings.TrimSpace(p.Username) != ""
}

func validPrincipal(p *models.AuthPrincipal) bool {
	if p == nil || p.ID <= 0 || p.TokenVersion <= 0 || p.Status != "active" {
		return false
	}
	switch p.Type {
	case "staff":
		return p.SchoolID > 0
	case "platform_admin":
		return true
	case "parent":
		return p.MobileVerified
	default:
		return false
	}
}

func toPrincipalResponse(p *models.AuthPrincipal) principalResponse {
	return principalResponse{ID: p.ID, Type: p.Type, Status: p.Status, SchoolID: p.SchoolID, Username: p.Username, MustChangePassword: p.MustChangePassword}
}

func writeAuthError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"status": "error", "message": message})
}


type sessionBoundAccessTokenGenerator interface {
	GenerateAccessTokenForSession(principalID, tokenVersion int64, principalType, sessionID string) (string, error)
}
type sessionFamilyIssuer interface {
	IssueWithFamily(ctx context.Context, principal *models.AuthPrincipal) (string, string, error)
}
type sessionFamilyRotator interface {
	RotateWithFamily(ctx context.Context, refreshToken string) (*models.AuthPrincipal, string, string, error)
}

func generateSessionAccessToken(tokens TokenService, principal *models.AuthPrincipal, familyID string) (string, error) {
	bound, ok := tokens.(sessionBoundAccessTokenGenerator)
	if !ok { return "", errors.New("session-bound access token generation unavailable") }
	return bound.GenerateAccessTokenForSession(principal.ID, principal.TokenVersion, principal.Type, familyID)
}
func issueSessionTokens(ctx context.Context, tokens TokenService, sessions SessionService, principal *models.AuthPrincipal) (string, string, error) {
	issuer, ok := sessions.(sessionFamilyIssuer)
	if !ok { return "", "", errors.New("session-family issuance unavailable") }
	refresh, familyID, err := issuer.IssueWithFamily(ctx, principal)
	if err != nil { return "", "", err }
	access, err := generateSessionAccessToken(tokens, principal, familyID)
	if err != nil {
		_ = sessions.Revoke(ctx, refresh)
		return "", "", err
	}
	return access, refresh, nil
}
func rotateSessionTokens(ctx context.Context, tokens TokenService, sessions SessionService, refresh string) (*models.AuthPrincipal, string, string, error) {
	rotator, ok := sessions.(sessionFamilyRotator)
	if !ok { return nil, "", "", errors.New("session-family rotation unavailable") }
	principal, replacement, familyID, err := rotator.RotateWithFamily(ctx, refresh)
	if err != nil { return nil, "", "", err }
	if _, err := generateSessionAccessToken(tokens, principal, familyID); err != nil { return nil, "", "", err }
	return principal, replacement, familyID, nil
}
