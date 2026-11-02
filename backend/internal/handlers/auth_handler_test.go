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

type fakeSessionService struct {
	refreshToken string
	principal    *models.AuthPrincipal
	err          error
	issuedUserID int64
	rotatedToken string
	revokedToken string
}

func (f *fakeSessionService) Issue(_ context.Context, userID int64) (string, error) {
	f.issuedUserID = userID
	return f.refreshToken, f.err
}

func (f *fakeSessionService) Rotate(_ context.Context, refreshToken string) (*models.AuthPrincipal, string, error) {
	f.rotatedToken = refreshToken
	return f.principal, f.refreshToken, f.err
}

func (f *fakeSessionService) Revoke(_ context.Context, refreshToken string) error {
	f.revokedToken = refreshToken
	return f.err
}

type fakeMFAService struct{}
func (fakeMFAService) BeginAuthentication(context.Context,int64,string) (*services.MFAStart,error) {
	return &services.MFAStart{Required:false},nil
}
func (fakeMFAService) CompleteEnrollment(context.Context,string,string) (int64,[]string,error) { return 0,nil,services.ErrMFAInvalid }
func (fakeMFAService) VerifyLoginChallenge(context.Context,string,string) (int64,error) { return 0,services.ErrMFAInvalid }
func (fakeMFAService) BeginRecoveryCodeRegeneration(context.Context,int64) (*services.MFAStart,error) { return nil,services.ErrMFAInvalid }
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

	router := setupTestRouter(authService, tokenService, &fakeSessionService{refreshToken: "test-refresh-token"})

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
