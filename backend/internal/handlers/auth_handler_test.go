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

func setupTestRouter(authService AuthService, tokenService TokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := NewAuthHandler(authService, tokenService)
	router.POST("/auth/login", handler.Login)

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

	router := setupTestRouter(authService, tokenService)

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

	if strings.Contains(rec.Body.String(), "PasswordHash") {
		t.Fatal("response must not expose password hash")
	}
}

func TestLoginInvalidRequest(t *testing.T) {
	router := setupTestRouter(
		fakeAuthService{},
		fakeTokenService{token: "unused"},
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
