package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type fakeDatabase struct {
	err error
}

type fakeTokenValidator struct {
	claims *services.TokenClaims
	err    error
}

func (f fakeTokenValidator) ValidateAccessToken(
	tokenString string,
) (*services.TokenClaims, error) {
	if tokenString != "valid-test-token" {
		return nil, errors.New("invalid token")
	}

	return f.claims, f.err
}

func (f fakeDatabase) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf(
			"expected JSON content type, got %q",
			rec.Header().Get("Content-Type"),
		)
	}

	expectedBody := `{"message":"School Management System API is running","status":"success"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestReadyEndpointWhenDatabaseIsAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expectedBody := `{"message":"API and database are ready","status":"success"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestReadyEndpointWhenDatabaseFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{
		err: errors.New("database unavailable"),
	}, nil, nil)

	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			rec.Code,
		)
	}

	expectedBody := `{"message":"Database is not ready","status":"error"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf(
			"expected body %s, got %s",
			expectedBody,
			rec.Body.String(),
		)
	}
}

func TestAuthMeWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validator := fakeTokenValidator{
		claims: &services.TokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "10",
			},
			SchoolID: 2,
		},
	}

	router, err := setupRouter(fakeDatabase{}, nil, validator)
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer valid-test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	expectedBody := `{"school_id":2,"status":"success","user_id":10}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, rec.Body.String())
	}
}

func TestAuthMeWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{})
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMeWithInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{})
	if err != nil {
		t.Fatalf("failed to setup router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-test-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
