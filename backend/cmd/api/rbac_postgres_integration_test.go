package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/config"
	"school-management-system/internal/handlers"
	"school-management-system/internal/repository"
	"school-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// TestRBACHTTPAuthorizationUsesPostgresState exercises the real Gin router,
// authentication middleware, PostgreSQL principal/permission repositories,
// role service, and role handler. It proves default-deny behavior, a successful
// database-backed School Admin role grant, and immediate token-version revocation.
// All database writes are contained in one transaction and rolled back.
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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

	// Keep every fixture—including a possible first Platform Admin bootstrap—
	// inside this transaction. The router repositories use the same pgx.Tx,
	// so HTTP requests observe the uncommitted fixture and rollback is complete.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped HTTP/RBAC integration transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	var schoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, "RBAC HTTP Integration School "+suffix, "RBAC-HTTP-"+suffix).Scan(&schoolID); err != nil {
		t.Fatalf("create integration school: %v", err)
	}

	var principalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status, must_change_password)
		VALUES ($1, $2, $3, 'active', FALSE)
		RETURNING principal_id
	`, schoolID, "rbac-http-"+suffix, "integration-test-password-hash").Scan(&principalID); err != nil {
		t.Fatalf("create integration staff principal: %v", err)
	}

	var tokenVersion int64
	if err := tx.QueryRow(ctx,
		`SELECT token_version FROM auth_principals WHERE id = $1`, principalID,
	).Scan(&tokenVersion); err != nil {
		t.Fatalf("load integration principal token version: %v", err)
	}

	gin.SetMode(gin.TestMode)
	tokenService, err := services.NewTokenService(
		"rbac-postgres-integration-secret-material-32-bytes-minimum",
		"school-management-rbac-integration-test",
		5,
	)
	if err != nil {
		t.Fatalf("create real JWT service for integration test: %v", err)
	}
	accessToken, err := tokenService.GenerateAccessToken(principalID, tokenVersion, "staff")
	if err != nil {
		t.Fatalf("sign integration access token: %v", err)
	}
	principalRepo := repository.NewPrincipalAuthorizationRepository(tx)
	roleHandler := handlers.NewRoleHandler(services.NewRoleService(repository.NewRoleRepository(tx)))
	router, err := setupRouterWithHandlers(
		fakeDatabase{}, nil, tokenService,
		principalRepo, principalRepo,
		nil, roleHandler,
	)
	if err != nil {
		t.Fatalf("setup PostgreSQL-backed API router: %v", err)
	}

	request := func(method, path, bearerToken string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+bearerToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	postJSON := func(path, bearerToken, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+bearerToken)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// The real HS256 validator must reject a token whose signature is altered.
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 || parts[2] == "" {
		t.Fatal("generated access token has an invalid JWT structure")
	}
	// Mutate the first base64url character so the decoded signature bytes
	// necessarily change; the final character may contain unused padding bits.
	if parts[2][0] == 'A' {
		parts[2] = "B" + parts[2][1:]
	} else {
		parts[2] = "A" + parts[2][1:]
	}
	invalidSignature := request(http.MethodGet, "/auth/me", strings.Join(parts, "."))
	if invalidSignature.Code != http.StatusUnauthorized {
		t.Fatalf("JWT with altered signature must be rejected: status=%d body=%s; want 401",
			invalidSignature.Code, invalidSignature.Body.String())
	}

	identity := request(http.MethodGet, "/auth/me", accessToken)
	if identity.Code != http.StatusOK {
		t.Fatalf("active database principal should access /auth/me: status=%d body=%s",
			identity.Code, identity.Body.String())
	}

	denied := request(http.MethodGet, "/school/roles", accessToken)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("staff principal without role grants must be denied role administration: status=%d body=%s; want 403",
			denied.Code, denied.Body.String())
	}

	// Resolve an existing Platform Admin authorized to provision a School
	// Admin. On a fresh CI database, bootstrap one inside this transaction only.
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND p.permission_key = 'platform.school_admins.create'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		   AND EXISTS (
		       SELECT 1
		         FROM principal_roles delegate_pr
		         JOIN roles delegate_role ON delegate_role.id = delegate_pr.role_id
		         JOIN role_permissions delegate_rp ON delegate_rp.role_id = delegate_role.id
		         JOIN permissions delegate_p ON delegate_p.id = delegate_rp.permission_id
		        WHERE delegate_pr.principal_id = ap.id
		          AND delegate_pr.revoked_at IS NULL
		          AND delegate_role.role_key = 'platform_admin'
		          AND delegate_role.scope_type = 'platform'
		          AND delegate_p.permission_key = 'platform.permissions.delegate'
		          AND delegate_p.is_active = TRUE
		   )
		   AND EXISTS (
		       SELECT 1 FROM principal_mfa_credentials mfa
		        WHERE mfa.principal_id = ap.id
		          AND mfa.method = 'totp'
		          AND mfa.verified_at IS NOT NULL
		   )
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if errors.Is(err, pgx.ErrNoRows) {
		var anyBootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1
			          FROM principal_roles pr
			          JOIN roles r ON r.id = pr.role_id
			         WHERE r.principal_type = 'platform_admin'
			           AND r.scope_type = 'platform'
			           AND r.role_key = 'platform_admin'
			    )
		`).Scan(&anyBootstrapExists); checkErr != nil {
			t.Fatalf("check existing Platform Admin bootstrap state: %v", checkErr)
		}
		if anyBootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.school_admins.create")
		}
		platformAdminPrincipalID, err = repository.NewPlatformAdminRepository(pool).BootstrapFirstInTx(
			ctx, tx,
			fmt.Sprintf("rbac-http-%s@example.invalid", suffix),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}
	// A newly bootstrapped test-only Platform Admin needs a verified MFA
	// fixture to pass the same mandatory-MFA middleware as production users.
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
		    (principal_id, method, secret_ciphertext, encryption_key_version, verified_at)
		SELECT $1, 'totp', $2, 1, NOW()
		 WHERE NOT EXISTS (
		     SELECT 1 FROM principal_mfa_credentials
		      WHERE principal_id = $1 AND method = 'totp' AND verified_at IS NOT NULL
		 )
	`, platformAdminPrincipalID, []byte("integration-platform-admin-mfa-ciphertext")); err != nil {
		t.Fatalf("create verified MFA fixture for Platform Admin: %v", err)
	}
	var platformTokenVersion int64
	if err := tx.QueryRow(ctx,
		`SELECT token_version FROM auth_principals WHERE id = $1`, platformAdminPrincipalID,
	).Scan(&platformTokenVersion); err != nil {
		t.Fatalf("load Platform Admin token version: %v", err)
	}
	platformAccessToken, err := tokenService.GenerateAccessToken(
		platformAdminPrincipalID, platformTokenVersion, "platform_admin",
	)
	if err != nil {
		t.Fatalf("sign Platform Admin integration access token: %v", err)
	}

	// School Admin authentication requires verified MFA. A non-secret test
	// ciphertext is sufficient for this repository-level identity fixture;
	// no MFA verification or encryption implementation is under test here.
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_mfa_credentials
		    (principal_id, method, secret_ciphertext, encryption_key_version, verified_at)
		VALUES ($1, 'totp', $2, 1, NOW())
	`, principalID, []byte("integration-test-ciphertext")); err != nil {
		t.Fatalf("create verified MFA fixture for School Admin: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		  FROM roles
		 WHERE school_id = $1
		   AND principal_type = 'staff'
		   AND scope_type = 'school'
		   AND role_key = 'school_admin'
		   AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load provisioned School Admin role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, principalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign School Admin role through database authorization trigger: %v", err)
	}

	// Permissions are loaded from PostgreSQL on every request. The role grant
	// above must now permit the same token to reach the real role handler.
	allowed := request(http.MethodGet, "/school/roles", accessToken)
	if allowed.Code != http.StatusOK {
		t.Fatalf("staff principal with database-granted School Admin role should list school roles: status=%d body=%s; want 200",
			allowed.Code, allowed.Body.String())
	}
	if !strings.Contains(allowed.Body.String(), "school_admin") {
		t.Fatalf("successful role response does not contain the School Admin role: %s", allowed.Body.String())
	}

	// Exercise a real authorized HTTP write through the role handler, service,
	// and transaction-backed PostgreSQL repository.
	customRoleName := "HTTP Integration Custom Role " + suffix
	createdRole := postJSON("/school/roles", accessToken,
		fmt.Sprintf(`{"name":%q}`, customRoleName))
	if createdRole.Code != http.StatusCreated {
		t.Fatalf("School Admin custom-role creation should reach PostgreSQL and return 201: status=%d body=%s",
			createdRole.Code, createdRole.Body.String())
	}
	if !strings.Contains(createdRole.Body.String(), customRoleName) {
		t.Fatalf("custom-role creation response does not include the created role name: %s", createdRole.Body.String())
	}
	var createdRolePayload struct {
		Role struct {
			ID int64 `json:"id"`
		} `json:"role"`
	}
	if err := json.Unmarshal(createdRole.Body.Bytes(), &createdRolePayload); err != nil || createdRolePayload.Role.ID <= 0 {
		t.Fatalf("decode created custom-role ID: payload=%s error=%v", createdRole.Body.String(), err)
	}

	// Platform Admin grants a bounded permission delegation over HTTP; the
	// School Admin then consumes that exact delegation to grant the permission
	// to the custom role over the same real HTTP/PostgreSQL path.
	delegationResponse := postJSON(
		fmt.Sprintf("/platform/schools/%d/permission-delegations", schoolID),
		platformAccessToken,
		`{"permission_key":"students.read_basic","can_assign_to_roles":true,"can_override_users":false}`,
	)
	if delegationResponse.Code != http.StatusCreated {
		t.Fatalf("Platform Admin permission delegation should return 201: status=%d body=%s",
			delegationResponse.Code, delegationResponse.Body.String())
	}
	var delegationPayload struct {
		Delegation struct {
			ID int64 `json:"id"`
		} `json:"delegation"`
	}
	if err := json.Unmarshal(delegationResponse.Body.Bytes(), &delegationPayload); err != nil || delegationPayload.Delegation.ID <= 0 {
		t.Fatalf("decode permission delegation ID: payload=%s error=%v", delegationResponse.Body.String(), err)
	}
	grantResponse := postJSON(
		fmt.Sprintf("/school/roles/%d/permissions", createdRolePayload.Role.ID),
		accessToken,
		fmt.Sprintf(`{"permission_key":"students.read_basic","delegation_id":%d}`, delegationPayload.Delegation.ID),
	)
	if grantResponse.Code != http.StatusCreated {
		t.Fatalf("School Admin custom-role permission grant should return 201: status=%d body=%s",
			grantResponse.Code, grantResponse.Body.String())
	}
	permissionRevoke := request(
		http.MethodDelete,
		fmt.Sprintf("/school/roles/%d/permissions/students.read_basic", createdRolePayload.Role.ID),
		accessToken,
	)
	if permissionRevoke.Code != http.StatusOK {
		t.Fatalf("School Admin custom-role permission revoke should return 200: status=%d body=%s",
			permissionRevoke.Code, permissionRevoke.Body.String())
	}
	delegationRevoke := request(
		http.MethodPost,
		fmt.Sprintf("/platform/permission-delegations/%d/revoke", delegationPayload.Delegation.ID),
		platformAccessToken,
	)
	if delegationRevoke.Code != http.StatusOK {
		t.Fatalf("Platform Admin permission delegation revoke should return 200: status=%d body=%s",
			delegationRevoke.Code, delegationRevoke.Body.String())
	}

	// Exercise staff-role assignment and revocation through the real HTTP
	// handler using a same-school target staff principal.
	var targetStaffPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status, must_change_password)
		VALUES ($1, $2, $3, 'active', FALSE)
		RETURNING principal_id
	`, schoolID, "http-target-"+suffix, "integration-test-password-hash").Scan(&targetStaffPrincipalID); err != nil {
		t.Fatalf("create HTTP target staff principal: %v", err)
	}
	assignRoleResponse := request(
		http.MethodPost,
		fmt.Sprintf("/school/staff/%d/roles/%d", targetStaffPrincipalID, createdRolePayload.Role.ID),
		accessToken,
	)
	if assignRoleResponse.Code != http.StatusCreated {
		t.Fatalf("School Admin staff-role assignment should return 201: status=%d body=%s",
			assignRoleResponse.Code, assignRoleResponse.Body.String())
	}
	listAssignments := request(
		http.MethodGet,
		fmt.Sprintf("/school/staff/%d/roles", targetStaffPrincipalID),
		accessToken,
	)
	if listAssignments.Code != http.StatusOK || !strings.Contains(listAssignments.Body.String(), customRoleName) {
		t.Fatalf("School Admin should list the assigned staff role: status=%d body=%s",
			listAssignments.Code, listAssignments.Body.String())
	}
	revokeRoleResponse := request(
		http.MethodDelete,
		fmt.Sprintf("/school/staff/%d/roles/%d", targetStaffPrincipalID, createdRolePayload.Role.ID),
		accessToken,
	)
	if revokeRoleResponse.Code != http.StatusOK {
		t.Fatalf("School Admin staff-role revocation should return 200: status=%d body=%s",
			revokeRoleResponse.Code, revokeRoleResponse.Body.String())
	}

	var foreignSchoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, "HTTP Foreign School "+suffix, "HTTP-FOR-"+suffix).Scan(&foreignSchoolID); err != nil {
		t.Fatalf("create foreign-school HTTP fixture: %v", err)
	}
	var foreignStaffPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status, must_change_password)
		VALUES ($1, $2, $3, 'active', FALSE)
		RETURNING principal_id
	`, foreignSchoolID, "http-foreign-"+suffix, "integration-test-password-hash").Scan(&foreignStaffPrincipalID); err != nil {
		t.Fatalf("create foreign-school staff principal: %v", err)
	}
	foreignRoleAssignment := request(
		http.MethodPost,
		fmt.Sprintf("/school/staff/%d/roles/%d", foreignStaffPrincipalID, createdRolePayload.Role.ID),
		accessToken,
	)
	if foreignRoleAssignment.Code != http.StatusNotFound {
		t.Fatalf("School A admin assigning a role to School B staff must return 404: status=%d body=%s",
			foreignRoleAssignment.Code, foreignRoleAssignment.Body.String())
	}

	// Verify the delegated per-user override HTTP path, including exact
	// delegation linkage and the matching revocation route.
	overrideDelegationResponse := postJSON(
		fmt.Sprintf("/platform/schools/%d/permission-delegations", schoolID),
		platformAccessToken,
		`{"permission_key":"homework.create","can_assign_to_roles":false,"can_override_users":true}`,
	)
	if overrideDelegationResponse.Code != http.StatusCreated {
		t.Fatalf("Platform Admin override delegation should return 201: status=%d body=%s",
			overrideDelegationResponse.Code, overrideDelegationResponse.Body.String())
	}
	var overrideDelegationPayload struct {
		Delegation struct {
			ID int64 `json:"id"`
		} `json:"delegation"`
	}
	if err := json.Unmarshal(overrideDelegationResponse.Body.Bytes(), &overrideDelegationPayload); err != nil ||
		overrideDelegationPayload.Delegation.ID <= 0 {
		t.Fatalf("decode override delegation ID: payload=%s error=%v",
			overrideDelegationResponse.Body.String(), err)
	}
	setOverrideResponse := postJSON(
		fmt.Sprintf("/school/staff/%d/permission-overrides", targetStaffPrincipalID),
		accessToken,
		fmt.Sprintf(`{"permission_key":"homework.create","effect":"allow","reason":"HTTP integration test","delegation_id":%d}`,
			overrideDelegationPayload.Delegation.ID),
	)
	if setOverrideResponse.Code != http.StatusOK {
		t.Fatalf("School Admin staff permission override should return 200: status=%d body=%s",
			setOverrideResponse.Code, setOverrideResponse.Body.String())
	}
	foreignOverrideResponse := postJSON(
		fmt.Sprintf("/school/staff/%d/permission-overrides", foreignStaffPrincipalID),
		accessToken,
		fmt.Sprintf(`{"permission_key":"homework.create","effect":"allow","reason":"cross-school HTTP test","delegation_id":%d}`,
			overrideDelegationPayload.Delegation.ID),
	)
	if foreignOverrideResponse.Code != http.StatusConflict {
		t.Fatalf("School A admin overriding School B staff must be rejected with 409: status=%d body=%s",
			foreignOverrideResponse.Code, foreignOverrideResponse.Body.String())
	}
	deleteOverrideResponse := request(
		http.MethodDelete,
		fmt.Sprintf("/school/staff/%d/permission-overrides/homework.create", targetStaffPrincipalID),
		accessToken,
	)
	if deleteOverrideResponse.Code != http.StatusOK {
		t.Fatalf("School Admin staff permission override deletion should return 200: status=%d body=%s",
			deleteOverrideResponse.Code, deleteOverrideResponse.Body.String())
	}
	revokeOverrideDelegation := request(
		http.MethodPost,
		fmt.Sprintf("/platform/permission-delegations/%d/revoke", overrideDelegationPayload.Delegation.ID),
		platformAccessToken,
	)
	if revokeOverrideDelegation.Code != http.StatusOK {
		t.Fatalf("Platform Admin override delegation revoke should return 200: status=%d body=%s",
			revokeOverrideDelegation.Code, revokeOverrideDelegation.Body.String())
	}

	// HTTP success responses must correspond to durable-in-transaction audit
	// events, while rejected cross-school writes must leave no audit or data
	// side effects. The outer transaction is rolled back at test completion.
	assertAuditEvent := func(action, resourceType string, resourceID int64) {
		t.Helper()
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			  FROM audit_events
			 WHERE actor_principal_id = $1
			   AND school_id = $2
			   AND action = $3
			   AND resource_type = $4
			   AND resource_id = $5
			   AND outcome = 'success'
		`, principalID, schoolID, action, resourceType, fmt.Sprint(resourceID)).Scan(&count); err != nil {
			t.Fatalf("count audit event %s: %v", action, err)
		}
		if count != 1 {
			t.Fatalf("expected exactly one %s audit event for %s/%d; got %d",
				action, resourceType, resourceID, count)
		}
	}
	assertPlatformAuditEvent := func(action string, resourceID int64) {
		t.Helper()
		var count int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			  FROM audit_events
			 WHERE actor_principal_id = $1
			   AND school_id = $2
			   AND action = $3
			   AND resource_type = 'permission_delegation'
			   AND resource_id = $4
			   AND outcome = 'success'
		`, platformAdminPrincipalID, schoolID, action, fmt.Sprint(resourceID)).Scan(&count); err != nil {
			t.Fatalf("count Platform Admin audit event %s: %v", action, err)
		}
		if count != 1 {
			t.Fatalf("expected exactly one %s audit event for delegation %d; got %d",
				action, resourceID, count)
		}
	}
	assertAuditEvent("school.role.created", "school_role", createdRolePayload.Role.ID)
	assertPlatformAuditEvent("platform.permission_delegation.granted", delegationPayload.Delegation.ID)
	assertPlatformAuditEvent("platform.permission_delegation.revoked", delegationPayload.Delegation.ID)
	assertAuditEvent("school.custom_role.permission_granted", "school_role", createdRolePayload.Role.ID)
	assertAuditEvent("school.custom_role.permission_revoked", "school_role", createdRolePayload.Role.ID)
	assertAuditEvent("school.staff.role_assigned", "staff_principal", targetStaffPrincipalID)
	assertAuditEvent("school.staff.role_revoked", "staff_principal", targetStaffPrincipalID)
	assertPlatformAuditEvent("platform.permission_delegation.granted", overrideDelegationPayload.Delegation.ID)
	assertPlatformAuditEvent("platform.permission_delegation.revoked", overrideDelegationPayload.Delegation.ID)
	assertAuditEvent("school.staff.permission_override.set", "user_permission_override", targetStaffPrincipalID)
	assertAuditEvent("school.staff.permission_override.removed", "user_permission_override", targetStaffPrincipalID)

	var foreignAssignmentExists, foreignOverrideExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM principal_roles
		     WHERE principal_id = $1 AND role_id = $2 AND revoked_at IS NULL
		 )
	`, foreignStaffPrincipalID, createdRolePayload.Role.ID).Scan(&foreignAssignmentExists); err != nil {
		t.Fatalf("check foreign-school role assignment side effect: %v", err)
	}
	if foreignAssignmentExists {
		t.Fatal("rejected cross-school role assignment unexpectedly created a principal_roles row")
	}
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM user_permission_overrides
		     WHERE principal_id = $1 AND school_id = $2
		       AND delegation_id = $3
		 )
	`, foreignStaffPrincipalID, foreignSchoolID, overrideDelegationPayload.Delegation.ID).Scan(&foreignOverrideExists); err != nil {
		t.Fatalf("check foreign-school permission override side effect: %v", err)
	}
	if foreignOverrideExists {
		t.Fatal("rejected cross-school permission override unexpectedly created a user_permission_overrides row")
	}
	var foreignAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM audit_events
		 WHERE actor_principal_id = $1
		   AND school_id = $2
		   AND resource_id = $3
		   AND action IN ('school.staff.role_assigned', 'school.staff.permission_override.set')
		   AND outcome = 'success'
	`, principalID, schoolID, fmt.Sprint(foreignStaffPrincipalID)).Scan(&foreignAuditCount); err != nil {
		t.Fatalf("check rejected cross-school write audit side effects: %v", err)
	}
	if foreignAuditCount != 0 {
		t.Fatalf("rejected cross-school writes must not emit success audit events; got %d", foreignAuditCount)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE auth_principals SET token_version = token_version + 1 WHERE id = $1`, principalID,
	); err != nil {
		t.Fatalf("revoke integration access token by advancing token version: %v", err)
	}
	stale := request(http.MethodGet, "/auth/me", accessToken)
	if stale.Code != http.StatusUnauthorized {
		t.Fatalf("request with stale token version must be rejected using current PostgreSQL state: status=%d body=%s; want 401",
			stale.Code, stale.Body.String())
	}
}
