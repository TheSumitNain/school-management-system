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
	user *models.User
	err  error
}

func (f fakeAuthService) Authenticate(
	ctx context.Context,
	schoolID int64,
	username string,
	password string,
) (*models.User, error) {
	return f.user, f.err
}

type fakeTokenService struct {
	token string
	err   error
}

func (f fakeTokenService) GenerateAccessToken(
	userID, schoolID int64,
) (string, error) {
	return f.token, f.err
}

type fakeSessionService struct {
	refreshToken string
	user         *models.User
	err          error
	issuedUserID int64
	rotatedToken string
	revokedToken string
}

func (f *fakeSessionService) Issue(_ context.Context, userID int64) (string, error) {
	f.issuedUserID = userID
	return f.refreshToken, f.err
}

func (f *fakeSessionService) Rotate(_ context.Context, refreshToken string) (*models.User, string, error) {
	f.rotatedToken = refreshToken
	return f.user, f.refreshToken, f.err
}

func (f *fakeSessionService) Revoke(_ context.Context, refreshToken string) error {
	f.revokedToken = refreshToken
	return f.err
}

func setupTestRouter(authService AuthService, tokenService TokenService, sessionService SessionService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(authService, tokenService, sessionService)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)

	return router
}

func TestLoginSuccess(t *testing.T) {
	authService := fakeAuthService{
		user: &models.User{
			ID:       10,
			SchoolID: 2,
			Username: "testuser",
			Status:   "active",
		},
	}
	tokenService := fakeTokenService{
		token: "test-access-token",
	}

	router := setupTestRouter(authService, tokenService, &fakeSessionService{refreshToken: "test-refresh-token"})

	body := `{"school_id":2,"username":"testuser","password":"StrongPass123"}`
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

func TestLoginInvalidRequest(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{},
		fakeTokenService{token: "unused"},
		&fakeSessionService{refreshToken: "test-refresh-token"},
	)

	body := `{"school_id":0,"username":"","password":""}`
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

	body := `{"school_id":2,"username":"testuser","password":"WrongPassword"}`
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
			user: &models.User{
				ID:       10,
				SchoolID: 2,
				Username: "testuser",
				Status:   "active",
			},
		},
		fakeTokenService{err: errors.New("token generation failed")},
		&fakeSessionService{},
	)

	body := `{"school_id":2,"username":"testuser","password":"StrongPass123"}`
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
		user:         &models.User{ID: 10, SchoolID: 2, Username: "testuser", Status: "active"},
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
