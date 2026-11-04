package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"school-management-system/internal/models"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeTokenValidator struct {
	claims *services.TokenClaims
	err    error
	token  string
}

func (f *fakeTokenValidator) ValidateAccessToken(tokenString string) (*services.TokenClaims, error) {
	f.token = tokenString
	return f.claims, f.err
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

func setupMiddlewareRouter(validator AccessTokenValidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	principal := &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 1, SchoolID: 2, Username: "tester"}
	router.GET("/protected", AuthMiddleware(validator, fakePrincipalLoader{principal: principal}, fakePermissionLoader{permissions: []string{"students.read_basic"}}), func(c *gin.Context) {
		principalID, _ := c.Get(ContextPrincipalIDKey)
		schoolID, _ := c.Get(ContextSchoolIDKey)
		c.JSON(http.StatusOK, gin.H{"principal_id": principalID, "school_id": schoolID})
	})
	return router
}

func validClaims() *services.TokenClaims {
	claims := &services.TokenClaims{PrincipalType: "staff", TokenVersion: 1}
	claims.Subject = "10"
	return claims
}

func TestAuthMiddlewareValidToken(t *testing.T) {
	validator := &fakeTokenValidator{claims: validClaims()}
	router := setupMiddlewareRouter(validator)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
	if validator.token != "valid-token" {
		t.Fatalf("expected validator to receive token, got %q", validator.token)
	}
	if rec.Body.String() != `{"principal_id":10,"school_id":2}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestAuthMiddlewareMissingAuthorization(t *testing.T) {
	router := setupMiddlewareRouter(&fakeTokenValidator{})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareMalformedAuthorization(t *testing.T) {
	for _, header := range []string{"Bearer", "valid-token", "Basic valid-token", "Bearer first second"} {
		t.Run(header, func(t *testing.T) {
			router := setupMiddlewareRouter(&fakeTokenValidator{})
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", header)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	router := setupMiddlewareRouter(&fakeTokenValidator{err: errors.New("invalid token")})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareRejectsStalePrincipalVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	claims := validClaims()
	principal := &models.AuthPrincipal{ID: 10, Type: "staff", Status: "active", TokenVersion: 2, SchoolID: 2}
	router.GET("/protected", AuthMiddleware(&fakeTokenValidator{claims: claims}, fakePrincipalLoader{principal: principal}, fakePermissionLoader{}), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for stale token version, got %d", rec.Code)
	}
}

func TestAuthMiddlewareNilDependencies(t *testing.T) {
	router := setupMiddlewareRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		permission     string
		contextValue   any
		setContext     bool
		expectedStatus int
	}{
		{
			name:           "granted permission continues",
			permission:     "students.read_basic",
			contextValue:   map[string]struct{}{"students.read_basic": {}},
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing permission is forbidden",
			permission:     "students.read_sensitive",
			contextValue:   map[string]struct{}{"students.read_basic": {}},
			setContext:     true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing authorization context fails closed",
			permission:     "students.read_basic",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unexpected authorization context type fails closed",
			permission:     "students.read_basic",
			contextValue:   []string{"students.read_basic"},
			setContext:     true,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "empty permission policy fails closed",
			contextValue:   map[string]struct{}{"students.read_basic": {}},
			setContext:     true,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/protected", func(c *gin.Context) {
				if tt.setContext {
					c.Set(ContextPermissionsKey, tt.contextValue)
				}
				c.Next()
			}, RequirePermission(tt.permission), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/protected", nil))

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d; body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}


func TestAuthMiddlewareRestrictsTemporaryPasswordStaff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	validator := &fakeTokenValidator{claims: validClaims()}
	principal := &models.AuthPrincipal{
		ID: 10, Type: "staff", Status: "active", TokenVersion: 1,
		SchoolID: 2, Username: "tester", MustChangePassword: true,
	}
	permissions := fakePermissionLoader{permissions: []string{"students.read_basic"}}
	loader := fakePrincipalLoader{principal: principal}
	router := gin.New()
	auth := AuthMiddleware(validator, loader, permissions)

	router.GET("/protected", auth, func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/auth/password/change", auth, func(c *gin.Context) {
		value, exists := c.Get(ContextMustChangePasswordKey)
		if !exists || value != true {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	})
	router.GET("/auth/me", auth, func(c *gin.Context) { c.Status(http.StatusOK) })

	request := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	if rec := request(http.MethodGet, "/protected"); rec.Code != http.StatusForbidden {
		t.Fatalf("temporary-password staff access status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(http.MethodPost, "/auth/password/change"); rec.Code != http.StatusOK {
		t.Fatalf("password-change endpoint status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if rec := request(http.MethodGet, "/auth/me"); rec.Code != http.StatusOK {
		t.Fatalf("identity endpoint status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}


type fakeAdminSessionTracker struct {
	isAdmin bool
	valid bool
	err error
	called int
	familyID string
	principalID int64
}
func (f *fakeAdminSessionTracker) CheckAndTouchAdminSession(_ context.Context, principalID int64, familyID string) (bool,bool,error) {
	f.called++; f.principalID=principalID; f.familyID=familyID
	return f.isAdmin,f.valid,f.err
}

func TestAuthMiddlewareRequiresValidSessionForAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal:=&models.AuthPrincipal{ID:99,Type:"platform_admin",Status:"active",TokenVersion:4}
	tracker:=&fakeAdminSessionTracker{isAdmin:true,valid:true}
	claims:=&services.TokenClaims{PrincipalType:"platform_admin",TokenVersion:4}
	claims.Subject="99"
	router:=gin.New()
	router.GET("/protected",AuthMiddleware(&fakeTokenValidator{claims:claims},fakePrincipalLoader{principal:principal},fakePermissionLoader{},tracker),func(c *gin.Context){c.Status(http.StatusOK)})
	req:=httptest.NewRequest(http.MethodGet,"/protected",nil);req.Header.Set("Authorization","Bearer token")
	rec:=httptest.NewRecorder();router.ServeHTTP(rec,req)
	if rec.Code!=http.StatusUnauthorized { t.Fatalf("admin token without sid must be rejected, got %d",rec.Code) }
	if tracker.called!=1 || tracker.principalID!=99 { t.Fatalf("tracker was not called for admin principal: %+v",tracker) }
}

func TestAuthMiddlewareAcceptsActiveAdminSessionFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal:=&models.AuthPrincipal{ID:99,Type:"platform_admin",Status:"active",TokenVersion:4}
	tracker:=&fakeAdminSessionTracker{isAdmin:true,valid:true}
	claims:=&services.TokenClaims{PrincipalType:"platform_admin",TokenVersion:4,SessionID:"a1b2c3d4-e5f6-4789-8123-abcdef012345"}
	claims.Subject="99"
	router:=gin.New()
	router.GET("/protected",AuthMiddleware(&fakeTokenValidator{claims:claims},fakePrincipalLoader{principal:principal},fakePermissionLoader{},tracker),func(c *gin.Context){c.Status(http.StatusOK)})
	req:=httptest.NewRequest(http.MethodGet,"/protected",nil);req.Header.Set("Authorization","Bearer token")
	rec:=httptest.NewRecorder();router.ServeHTTP(rec,req)
	if rec.Code!=http.StatusOK { t.Fatalf("valid admin session should pass, got %d: %s",rec.Code,rec.Body.String()) }
	if tracker.familyID!=claims.SessionID { t.Fatalf("tracker family = %q, want %q",tracker.familyID,claims.SessionID) }
}

func TestAuthMiddlewareRejectsExpiredAdminSessionFamily(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal:=&models.AuthPrincipal{ID:99,Type:"platform_admin",Status:"active",TokenVersion:4}
	tracker:=&fakeAdminSessionTracker{isAdmin:true,valid:false}
	claims:=&services.TokenClaims{PrincipalType:"platform_admin",TokenVersion:4,SessionID:"a1b2c3d4-e5f6-4789-8123-abcdef012345"}
	claims.Subject="99"
	router:=gin.New()
	router.GET("/protected",AuthMiddleware(&fakeTokenValidator{claims:claims},fakePrincipalLoader{principal:principal},fakePermissionLoader{},tracker),func(c *gin.Context){c.Status(http.StatusOK)})
	req:=httptest.NewRequest(http.MethodGet,"/protected",nil);req.Header.Set("Authorization","Bearer token")
	rec:=httptest.NewRecorder();router.ServeHTTP(rec,req)
	if rec.Code!=http.StatusUnauthorized { t.Fatalf("expired admin session should fail, got %d",rec.Code) }
}
