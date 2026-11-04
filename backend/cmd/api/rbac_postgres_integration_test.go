package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/handlers"
	"school-management-system/internal/models"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// TestRBACHTTPAuthorizationUsesPostgresState exercises the real Gin router,
// authentication middleware, principal repository, permission repository,
// role service, and role handler against PostgreSQL. It deliberately creates
// a staff principal with no role assignments: identity inspection is allowed,
// while role administration must be denied by the live permission lookup.
func TestRBACHTTPAuthorizationUsesPostgresState(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	if err := godotenv.Load(filepath.Join("..", "..", ".env")); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("load backend .env file: %v", err)
	}
	cfg := config.Load()
	if cfg.DBName != "school_management_test" {
		t.Fatalf("unsafe integration database %q; expected school_management_test", cfg.DBName)
	}
	if err := cfg.ValidateDatabase(); err != nil {
		t.Fatalf("invalid test database configuration: %v", err)
	}

	dbURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host:   net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path:   "/" + cfg.DBName,
	}
	query := dbURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	dbURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL.String())
	if err != nil {
		t.Fatalf("connect to PostgreSQL integration database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping PostgreSQL integration database: %v", err)
	}

	var hasV4 bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.auth_principals') IS NOT NULL`).Scan(&hasV4); err != nil {
		t.Fatalf("detect V4 principal schema: %v", err)
	}
	if !hasV4 {
		t.Skip("V4 principal schema is not applied")
	}

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	var schoolID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, "RBAC HTTP Integration School "+suffix, "RBAC-HTTP-"+suffix).Scan(&schoolID); err != nil {
		t.Fatalf("create integration school: %v", err)
	}

	var userID, principalID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id, principal_id
	`, schoolID, "rbac-http-"+suffix, "integration-test-password-hash").Scan(&userID, &principalID); err != nil {
		_, _ = pool.Exec(ctx, `DELETE FROM schools WHERE id = $1`, schoolID)
		t.Fatalf("create integration staff principal: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx,
			`DELETE FROM refresh_sessions WHERE principal_id = $1 OR user_id = $2`,
			principalID, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM auth_principals WHERE id = $1`, principalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM schools WHERE id = $1`, schoolID)
	})

	var tokenVersion int64
	if err := pool.QueryRow(ctx,
		`SELECT token_version FROM auth_principals WHERE id = $1`, principalID,
	).Scan(&tokenVersion); err != nil {
		t.Fatalf("load integration principal token version: %v", err)
	}

	gin.SetMode(gin.TestMode)
	validator := fakeTokenValidator{claims: &services.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(principalID, 10)},
		PrincipalType:    "staff",
		TokenVersion:     tokenVersion,
	}}
	principalRepo := repository.NewPrincipalAuthorizationRepository(pool)
	roleHandler := handlers.NewRoleHandler(services.NewRoleService(repository.NewRoleRepository(pool)))
	router, err := setupRouterWithHandlers(
		fakeDatabase{}, nil, validator,
		principalRepo, principalRepo,
		nil, roleHandler,
	)
	if err != nil {
		t.Fatalf("setup PostgreSQL-backed API router: %v", err)
	}

	request := func(method, path string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer valid-test-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	identity := request(http.MethodGet, "/auth/me")
	if identity.Code != http.StatusOK {
		t.Fatalf("active database principal should access /auth/me: status=%d body=%s",
			identity.Code, identity.Body.String())
	}

	permissions, err := principalRepo.LoadEffectivePermissions(ctx, &models.AuthPrincipal{
		ID: principalID, Type: "staff", Status: "active", TokenVersion: tokenVersion, SchoolID: schoolID,
	})
	if err != nil {
		t.Fatalf("resolve effective permissions from PostgreSQL: %v", err)
	}
	if len(permissions) != 0 {
		t.Fatalf("new staff principal unexpectedly has permissions: %v", permissions)
	}

	denied := request(http.MethodGet, "/school/roles")
	if denied.Code != http.StatusForbidden {
		t.Fatalf("staff principal without roles must be denied role administration: status=%d body=%s; want 403",
			denied.Code, denied.Body.String())
	}

	if _, err := pool.Exec(ctx,
		`UPDATE auth_principals SET token_version = token_version + 1 WHERE id = $1`, principalID,
	); err != nil {
		t.Fatalf("revoke integration access token by advancing token version: %v", err)
	}
	stale := request(http.MethodGet, "/auth/me")
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("request with stale token version must be rejected using current PostgreSQL state: status=%d body=%s; want 401",
			stale.Code, stale.Body.String())
	}

}
