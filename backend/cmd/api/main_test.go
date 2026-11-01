package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type fakeDatabase struct {
	err error
}

type fakePrincipalLoader struct {
	principal *models.AuthPrincipal
	err       error
}

func (f fakePrincipalLoader) LoadPrincipalForAuthentication(_ context.Context, _ int64) (*models.AuthPrincipal, error) {
	return f.principal, f.err
}

type fakePermissionLoader struct {
	permissions []string
	err         error
}

func (f fakePermissionLoader) LoadEffectivePermissions(_ context.Context, _ *models.AuthPrincipal) ([]string, error) {
	return f.permissions, f.err
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

	router, err := setupRouter(fakeDatabase{}, nil, nil, nil, nil)
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

	router, err := setupRouter(fakeDatabase{}, nil, nil, nil, nil)
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
	}, nil, nil, nil, nil)

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
			PrincipalType: "staff",
			TokenVersion:  1,
		},
	}

	router, err := setupRouter(fakeDatabase{}, nil, validator, fakePrincipalLoader{principal: &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "tester"}}, fakePermissionLoader{})
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

	expectedBody := `{"principal_id":10,"principal_type":"staff","school_id":2,"status":"success","username":"tester"}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s", expectedBody, rec.Body.String())
	}
}

func TestAuthMeWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{}, fakePrincipalLoader{}, fakePermissionLoader{})
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

	router, err := setupRouter(fakeDatabase{}, nil, fakeTokenValidator{}, fakePrincipalLoader{}, fakePermissionLoader{})
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
