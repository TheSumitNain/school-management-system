package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeAuthService struct {
	principal *models.AuthPrincipal
	err       error
}
type fakeLoginIPRateLimiter struct {
	allowed bool
	err     error
	clientIP string
	calls   int
}

func (f *fakeLoginIPRateLimiter) AllowIPAttempt(_ context.Context, clientIP string) (bool, error) {
	f.calls++
	f.clientIP = clientIP
	return f.allowed, f.err
}


func (f fakeAuthService) AuthenticateStaff(_ context.Context, _, _ string) (*models.AuthPrincipal, error) {
	return f.principal, f.err
}

type fakeTokenService struct {
	token string
	err   error
}

func (f fakeTokenService) GenerateAccessToken(
	principalID, tokenVersion int64, principalType string,
) (string, error) {
	return f.token, f.err
}

func (f fakeTokenService) GenerateAccessTokenForSession(principalID, tokenVersion int64, principalType, sessionID string) (string,error) {
	if sessionID=="" { return "",errors.New("session id required") }
	return f.token,f.err
}

type fakeSessionService struct {
	refreshToken string
	principal    *models.AuthPrincipal
	err          error
	issuedUserID int64
	issuedTokenVersion int64
	rotatedToken string
	revokedToken string
}

func (f *fakeSessionService) Issue(_ context.Context, principal *models.AuthPrincipal) (string, error) {
	if principal != nil {
		f.issuedUserID = principal.ID
		f.issuedTokenVersion = principal.TokenVersion
	}
	return f.refreshToken, f.err
}

func (f *fakeSessionService) IssueWithFamily(ctx context.Context, principal *models.AuthPrincipal) (string,string,error) {
	token,err:=f.Issue(ctx,principal)
	if err!=nil { return "", "", err }
	return token,"a1b2c3d4-e5f6-4789-8123-abcdef012345",nil
}

func (f *fakeSessionService) Rotate(_ context.Context, refreshToken string) (*models.AuthPrincipal, string, error) {
	f.rotatedToken = refreshToken
	return f.principal, f.refreshToken, f.err
}

func (f *fakeSessionService) RotateWithFamily(ctx context.Context, refreshToken string) (*models.AuthPrincipal,string,string,error) {
	principal,replacement,err:=f.Rotate(ctx,refreshToken)
	if err!=nil { return nil,"","",err }
	return principal,replacement,"a1b2c3d4-e5f6-4789-8123-abcdef012345",nil
}

func (f *fakeSessionService) Revoke(_ context.Context, refreshToken string) error {
	f.revokedToken = refreshToken
	return f.err
}

type fakeMFAService struct{}
func (fakeMFAService) BeginAuthentication(context.Context,*models.AuthPrincipal,string) (*services.MFAStart,error) {
	return &services.MFAStart{Required:false},nil
}
func (fakeMFAService) CompleteEnrollment(context.Context,string,string) (int64,int64,[]string,error) { return 0,0,nil,services.ErrMFAInvalid }
func (fakeMFAService) VerifyLoginChallenge(context.Context,string,string) (int64,int64,error) { return 0,0,services.ErrMFAInvalid }
func (fakeMFAService) BeginRecoveryCodeRegeneration(context.Context,int64,int64) (*services.MFAStart,error) { return nil,services.ErrMFAInvalid }
func (fakeMFAService) RegenerateRecoveryCodes(context.Context,string,string) ([]string,error) { return nil,services.ErrMFAInvalid }

type fakeMFAPrincipalLoader struct{}
func (fakeMFAPrincipalLoader) LoadPrincipalForAuthentication(context.Context,int64) (*models.AuthPrincipal,error) {
	return nil,models.ErrPrincipalNotAuthenticated
}

func setupTestRouter(authService AuthService, tokenService TokenService, sessionService SessionService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(authService, tokenService, sessionService)
	handler.SetMFAService(fakeMFAService{},fakeMFAPrincipalLoader{})
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)

	return router
}

func TestLoginSuccess(t *testing.T) {
	authService := fakeAuthService{
		principal: &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "testuser"},
	}
	tokenService := fakeTokenService{
		token: "test-access-token",
	}

	sessionService := &fakeSessionService{refreshToken: "test-refresh-token"}
	router := setupTestRouter(authService, tokenService, sessionService)

	body := `{"username":"testuser","password":"StrongPass123"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"access_token":"test-access-token"`) {
		t.Fatalf("expected access token in response, got %s", rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"token_type":"Bearer"`) {
		t.Fatalf("expected Bearer token type, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"refresh_token":"test-refresh-token"`) {
		t.Fatalf("expected refresh token in response, got %s", rec.Body.String())
	}

	if strings.Contains(rec.Body.String(), "PasswordHash") {
		t.Fatal("response must not expose password hash")
	}
	if sessionService.issuedUserID != 10 || sessionService.issuedTokenVersion != 1 {
		t.Fatalf("login refresh session was not bound to authenticated principal version: principal=%d version=%d",
			sessionService.issuedUserID, sessionService.issuedTokenVersion)
	}
}

func TestLoginRejectsPrincipalVersionChangeDuringSessionIssue(t *testing.T) {
	authService := fakeAuthService{
		principal: &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "testuser"},
	}
	sessionService := &fakeSessionService{err: services.ErrSessionPrincipalVersionChanged}
	router := setupTestRouter(authService, fakeTokenService{token: "test-access-token"}, sessionService)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"username":"testuser","password":"StrongPass123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("stale authenticated principal version status=%d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Credentials changed during sign-in") {
		t.Fatalf("unexpected stale login response: %s", rec.Body.String())
	}
}

func TestLoginRejectsRateLimitedClientIP(t *testing.T) {
	limiter := &fakeLoginIPRateLimiter{allowed: false}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAuthHandler(
		fakeAuthService{},
		fakeTokenService{},
		&fakeSessionService{},
		limiter,
	)
	router.POST("/auth/login", handler.Login)

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{malformed-json`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusTooManyRequests, rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("Retry-After = %q, want 900", rec.Header().Get("Retry-After"))
	}
	if limiter.calls != 1 || limiter.clientIP != "192.0.2.1" {
		t.Fatalf("rate limiter received unexpected client IP/call count: %+v", limiter)
	}
}

func TestLoginInvalidRequest(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{},
		fakeTokenService{token: "unused"},
		&fakeSessionService{refreshToken: "test-refresh-token"},
	)

	body := `{"username":"","password":""}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{err: services.ErrInvalidCredentials},
		fakeTokenService{},
		&fakeSessionService{},
	)

	body := `{"username":"testuser","password":"WrongPassword"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestLoginTokenGenerationFailure(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{
			principal: &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "testuser"},
		},
		fakeTokenService{err: errors.New("token generation failed")},
		&fakeSessionService{},
	)

	body := `{"username":"testuser","password":"StrongPass123"}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusInternalServerError, rec.Code, rec.Body.String())
	}

	if strings.Contains(rec.Body.String(), "token generation failed") {
		t.Fatal("response must not expose internal error details")
	}
}

func TestRefreshSuccess(t *testing.T) {
	sessions := &fakeSessionService{
		refreshToken: "replacement-refresh-token",
		principal:    &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "testuser"},
	}
	router := setupTestRouter(fakeAuthService{}, fakeTokenService{token: "new-access-token"}, sessions)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh",
		strings.NewReader(`{"refresh_token":"old-refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"access_token":"new-access-token"`) ||
		!strings.Contains(rec.Body.String(), `"refresh_token":"replacement-refresh-token"`) {
		t.Fatalf("expected rotated tokens, got %s", rec.Body.String())
	}
	if sessions.rotatedToken != "old-refresh-token" {
		t.Fatalf("expected old refresh token to be rotated, got %q", sessions.rotatedToken)
	}
}

func TestRefreshRejectsInvalidToken(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{},
		fakeTokenService{},
		&fakeSessionService{err: services.ErrInvalidRefreshToken},
	)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh",
		strings.NewReader(`{"refresh_token":"bad-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusUnauthorized, rec.Code, rec.Body.String())
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	sessions := &fakeSessionService{}
	router := setupTestRouter(fakeAuthService{}, fakeTokenService{}, sessions)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout",
		strings.NewReader(`{"refresh_token":"test-refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if sessions.revokedToken != "test-refresh-token" {
		t.Fatalf("expected logout to revoke the supplied token, got %q", sessions.revokedToken)
	}
}


func TestRefreshAccessTokenFailureRevokesRotatedRefreshSession(t *testing.T) {
	principal := &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "testuser"}
	tokenService := fakeTokenService{err: errors.New("signing unavailable")}
	sessionService := &fakeSessionService{refreshToken: "replacement-refresh-token", principal: principal}
	router := setupTestRouter(fakeAuthService{}, tokenService, sessionService)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"old-refresh-token"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when access-token signing fails, got %d: %s", rec.Code, rec.Body.String())
	}
	if sessionService.revokedToken != "replacement-refresh-token" {
		t.Fatalf("rotated refresh token was not revoked after access-token failure: got %q", sessionService.revokedToken)
	}
}
