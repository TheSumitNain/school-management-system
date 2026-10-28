package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeTokenValidator struct {
	claims *services.TokenClaims
	err    error
	token  string
}

func (f *fakeTokenValidator) ValidateAccessToken(
	tokenString string,
) (*services.TokenClaims, error) {
	f.token = tokenString
	return f.claims, f.err
}

func setupMiddlewareRouter(
	validator AccessTokenValidator,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET(
		"/protected",
		AuthMiddleware(validator),
		func(c *gin.Context) {
			userID, _ := c.Get(ContextUserIDKey)
			schoolID, _ := c.Get(ContextSchoolIDKey)

			c.JSON(http.StatusOK, gin.H{
				"user_id":   userID,
				"school_id": schoolID,
			})
		},
	)

	return router
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	validator := &fakeTokenValidator{
		claims: &services.TokenClaims{
			SchoolID: 2,
		},
	}

	validator.claims.Subject = "10"

	router := setupMiddlewareRouter(validator)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	if validator.token != "valid-token" {
		t.Fatalf("expected validator to receive token, got %q", validator.token)
	}

	expectedBody := `{"school_id":2,"user_id":10}`
	if rec.Body.String() != expectedBody {
		t.Fatalf("expected body %s, got %s",
			expectedBody, rec.Body.String())
	}
}

func TestAuthMiddlewareMissingAuthorization(t *testing.T) {
	router := setupMiddlewareRouter(&fakeTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddlewareMalformedAuthorization(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{name: "missing token", header: "Bearer"},
		{name: "missing scheme", header: "valid-token"},
		{name: "wrong scheme", header: "Basic valid-token"},
		{name: "extra parts", header: "Bearer first second"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := setupMiddlewareRouter(&fakeTokenValidator{})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.header)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d",
					http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	validator := &fakeTokenValidator{
		err: errors.New("invalid token"),
	}

	router := setupMiddlewareRouter(validator)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddlewareNilClaims(t *testing.T) {
	router := setupMiddlewareRouter(&fakeTokenValidator{})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddlewareInvalidClaims(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		schoolID int64
	}{
		{name: "invalid user ID", subject: "abc", schoolID: 2},
		{name: "zero user ID", subject: "0", schoolID: 2},
		{name: "negative user ID", subject: "-1", schoolID: 2},
		{name: "zero school ID", subject: "10", schoolID: 0},
		{name: "negative school ID", subject: "10", schoolID: -2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := &services.TokenClaims{SchoolID: tt.schoolID}
			claims.Subject = tt.subject

			router := setupMiddlewareRouter(&fakeTokenValidator{
				claims: claims,
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer valid-token")

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d",
					http.StatusUnauthorized, rec.Code)
			}
		})
	}
}

func TestAuthMiddlewareNilValidator(t *testing.T) {
	router := setupMiddlewareRouter(nil)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d",
			http.StatusInternalServerError, rec.Code)
	}
}
