# Testing Documentation

## 1. Purpose

This document describes the testing strategy for the School Management System.

Testing must verify application correctness, security, database behavior, and reliability as the system grows.

The project is in its initial development stage. Tests documented as future requirements must not be treated as completed tests.

## 2. Current Testing Status

The backend has automated Go tests for selected packages and components.

Current test areas include:

- Password-service behavior, including staff current-password change and School Admin temporary-password reset.
- Authentication-service behavior and temporary-password state propagation.
- JWT access-token generation and validation.
- Authentication HTTP handler behavior, including staff password change/reset.
- Middleware enforcement for temporary-password staff sessions.
- API server behavior and selected configuration validation.
- Refresh-token session-service behavior and concurrency safeguards.
- V15/V16 PostgreSQL integration coverage for password lifecycle and exact delegation revocation.
- Router/middleware authorization tests for all 14 Platform Admin and school role-administration routes, including AND semantics for multi-permission routes.
- HTTP/PostgreSQL integration coverage for database-backed staff identity, default-deny role authorization, and token-version revocation using the real Gin router and principal/permission repositories.

An earlier source-only CI run on commit `4351ce9` passed the unit suite and vet but did not enable PostgreSQL integration tests. The latest full database-backed verification is recorded immediately below.

```bash
go test -count=1 ./...
go vet ./...
```

The latest GitHub Actions push run `41762250818` and Draft PR run `41762256758` on commit `9e74c88` applied migrations V1–V16 to a fresh ephemeral PostgreSQL 18 database, ran the full suite with `RUN_POSTGRES_INTEGRATION_TESTS=1`, and passed `go vet ./...`. Both runs include the 14 role-administration route permission tests and the first HTTP/PostgreSQL RBAC integration test. The user's persistent `school_management_test` and `school_management` databases were not changed. Repeat the workflow after relevant code changes.

The current test suite does not establish that every application behavior is tested or that the application is production-ready.

Important remaining gaps include PostgreSQL coverage for repository paths outside the currently tested authentication, staff lifecycle, parent foundation, and role-administration workflows; additional concurrency/failure-mode testing; full HTTP-to-PostgreSQL end-to-end verification (the current 14-route tests use controlled token/principal/permission loaders); business-resource tenant/assignment isolation; migration/rollback testing; and tests for future school business modules.

## 3. Testing Principles

- Test observable behavior rather than implementation details alone.
- Cover successful operations and expected failures.
- Test security boundaries explicitly.
- Keep tests repeatable and independent where practical.
- Use isolated test data and avoid production data.
- Test database constraints as well as application-level validation.
- Do not claim production readiness based only on unit-test success.
- Add regression tests when fixing a bug.
- Distinguish automated tests from manual verification.
- Record test results only after the commands have actually completed.
- Never treat a planned test as completed evidence.

## 4. Test Categories

### 4.1 Unit tests

Unit tests verify individual functions and business rules without requiring a complete running application.

Current examples include password-service tests, authentication-service tests using fake repositories, JWT token-service tests, HTTP handler tests, and refresh-token service tests.

Future unit tests should cover:

- Input validation and boundary conditions.
- User and school status rules.
- Authorization decisions.
- Attendance and other business rules.
- Error handling and unexpected failures.
- Token expiry, replay, and session-state edge cases.

### 4.2 Repository and database integration tests

Integration tests verify behavior against an actual PostgreSQL database.

They should cover:

- Successful inserts, reads, updates, and deletes.
- Foreign-key constraints.
- Unique constraints and duplicate records.
- Transaction commit and rollback.
- Query filtering and pagination.
- Database errors and connection failures.
- Migration behavior.
- Refresh-token rotation and revocation transactions.
- Revocation of active sessions when a revoked refresh token is reused.

Tests requiring PostgreSQL must use a dedicated test database or isolated test schema. They must never modify production data.

The repository has opt-in PostgreSQL integration tests in `backend/internal/repository/`. They require `RUN_POSTGRES_INTEGRATION_TESTS=1` and must use the dedicated `school_management_test` database when run locally. Coverage includes V15 staff password defaults and transactional password-change invalidation, V16 exact-delegation revocation/regrant behavior, rollback-scoped role/delegation/custom-permission/staff-role/ALLOW-DENY override repository workflows, and staff account create/list/update/disable/re-enable lifecycle. GitHub Actions applies V1–V16 to a fresh ephemeral PostgreSQL 18 database and runs the full integration suite; commit `64acf1b` and run `41657762673` passed, including `go vet ./...`. The user's persistent `school_management_test` remains unchanged. Never point tests or migrations at `school_management`.

### 4.3 HTTP tests

HTTP tests should exercise the Gin router and verify:

- HTTP status codes.
- Response bodies and content types.
- Request validation.
- Error responses.
- Health and readiness behavior.
- Login, refresh, and logout handler behavior.
- Authentication middleware behavior.
- Authorization behavior when RBAC-protected routes are implemented.

### 4.4 Security tests

Security tests must verify that unauthorized access is denied and sensitive data is not exposed.

The security test plan is detailed in `SECURITY.md`.

### 4.5 Load and performance tests

Load tests should use realistic request patterns and representative data volumes.

Measure:

- Request throughput.
- Response latency, including p95 and p99.
- Error rate.
- CPU and memory usage.
- PostgreSQL connection usage.
- Slow queries and lock contention.

The planning target is approximately 25–30 schools and 400–500 students per school, or around 10,000–15,000 students overall. This is a planning target, not a verified capacity guarantee. Capacity claims require reproducible load-test results.

## 5. Running Backend Tests

Run commands from the backend directory:

```bat
cd /d D:\school-management-system\backend
```

Run all Go tests:

```bash
go test -count=1 ./...
```

The `-count=1` flag disables reuse of cached test results for this run.

Run static analysis:

```bash
go vet ./...
```

Format Go source files when necessary:

```bash
gofmt -w cmd internal
```

Run formatting only when Go source files have changed. Review the resulting diff before committing.

## 6. Current Test Areas

### 6.1 Password service

Verify:

- Valid passwords can be hashed.
- Password verification succeeds for the matching password.
- An incorrect password fails verification.
- Passwords outside the supported byte-length range are rejected.
- Whitespace-only passwords are rejected.
- The original password is not stored as plaintext.
- Password hashes are not exposed in API responses.

The current password implementation uses bcrypt with cost 12 and an input range of 8–72 bytes. A separate 12-character administrator password policy and compromised-password checks remain outstanding.

### 6.2 Authentication service

Verify:

- Authentication succeeds for valid credentials when the account and school are active.
- Inactive or suspended schools cannot authenticate users.
- Disabled users cannot authenticate.
- Invalid credentials are rejected.
- Repository errors are handled safely.
- Duplicate usernames are handled correctly at the repository/database boundary.

### 6.3 JWT token service

The token service has an initial automated test suite.

Verify:

- Valid access tokens can be generated and validated.
- Invalid tokens are rejected.
- Expired tokens are rejected.
- Invalid signatures are rejected.
- Issuer and required claims are validated.
- Invalid token configuration is rejected where covered by the implementation.

The exact cases covered must be confirmed against the actual test files.

### 6.4 Login handler

The login handler has an initial automated test suite.

Verify:

- Valid requests return the expected success response.
- Invalid credentials return an unauthorized response.
- Malformed requests return an appropriate client error.
- Inactive accounts and schools are handled as specified.
- Token-generation failures do not expose sensitive internal details.
- Password hashes and other sensitive fields are not included in responses.

The exact cases covered must be confirmed against the actual test files.

### 6.5 Refresh-token lifecycle

Refresh-token rotation, revocation, and replay handling have been implemented in the current backend foundation. Service-level tests and opt-in PostgreSQL repository integration tests cover selected lifecycle scenarios.

The opt-in PostgreSQL integration suite uses `RUN_POSTGRES_INTEGRATION_TESTS=1` and refuses to run unless the configured database is `school_management_test`. The reported scenarios cover rotation, replay-triggered family revocation, family-wide logout and idempotency, expiry, and concurrent refresh requests.

The test plan includes:

- Generated refresh tokens are opaque and cryptographically random.
- Only the SHA-256 hash is persisted.
- Refresh-token expiry configuration is validated.
- Expired or invalid refresh tokens are rejected.
- Successful refresh rotates the token.
- Reuse of a revoked token triggers family revocation.
- Logout forwards the supplied refresh token for family-wide revocation; repeated logout is idempotent and unknown tokens are a no-op.
- Inactive-user or inactive-school behavior is handled safely.
- Cross-family inactive-user race: concurrent rotations in separate token families are serialized, and the replacement from the other family is revoked.
- Refresh failures do not expose raw tokens or sensitive internals.

**Remaining verification:**

- Expand PostgreSQL integration coverage for remaining repository paths and concurrency/failure modes not covered by the existing refresh-session, auth, staff lifecycle, parent foundation, and role administration tests.
- Add further transaction-failure and family-revocation edge cases against PostgreSQL.

- Test additional simultaneous refresh/logout interleavings and database failure scenarios.
- Test replay behavior across the HTTP API.
- Independently verify the HTTP logout flow, family-wide refresh-session revocation, and rejection of refresh tokens after logout.
- Test refresh attempts after expiry, logout, account disabling, and school deactivation.

Service-level tests alone do not establish database transaction correctness or concurrency safety.

### 6.6 Authentication middleware and protected identity endpoint

Verify:

- A valid Bearer access token is accepted.
- Missing or malformed authorization headers are rejected.
- Invalid, expired, or incorrectly signed tokens are rejected.
- Invalid user or school identity claims are rejected.
- The authenticated identity is extracted from validated claims.
- `GET /auth/me` rejects unauthenticated requests.
- The response does not expose password hashes, raw tokens, or unrelated sensitive fields.
- Middleware configuration failures are handled without exposing internal details.

The current middleware establishes token-based identity. It does not independently establish RBAC or complete tenant isolation.

## 7. Health and Readiness Tests

### Health endpoint

`GET /health`

Verify:

- A successful response is returned while the API is running.
- The status code and response body match the implementation.

Expected successful response:

```json
{
  "status": "success",
  "message": "School Management System API is running"
}
```

### Readiness endpoint

`GET /ready`

Verify:

- The endpoint succeeds when the database is available.
- The endpoint returns the intended unavailable status when the database check fails.
- Database failure information is not exposed inappropriately.

The current implementation uses a database readiness check and returns `503 Service Unavailable` when the database is not ready.

Re-run relevant tests whenever server initialization or database connection handling changes.

## 8. Authentication and Authorization Tests

The login, refresh, logout, and protected identity HTTP endpoints are implemented. The complete authentication and authorization security suite is not yet finished.

Test:

- Login with valid and invalid credentials.
- Malformed login, refresh, and logout requests.
- Access-token signature and expiry validation.
- Refresh-token rotation.
- Reuse of an old refresh token.
- Refresh-token revocation after logout.
- Concurrent refresh requests.
- Disabled-user and inactive-school behavior.
- Missing or malformed authorization headers.
- Invalid token claims.
- Rate limiting for repeated failed login attempts.

Once RBAC is implemented, verify that unauthenticated requests are denied and each role can perform only its authorized operations.

Registration, secure user provisioning, MFA, password reset, and authentication rate limiting remain planned work and require their own tests when implemented.

## 9. Multi-Tenant Isolation Tests

Cross-school data leakage is a critical risk for this product.

Create test data for at least two schools and verify:

- A user from School A cannot read School B's records.
- A user from School A cannot update or delete School B's records.
- List endpoints return only records authorized for the user's school.
- Related-record lookups cannot bypass school restrictions.
- Client-supplied school identifiers cannot override the authenticated school context.
- Administrative operations enforce the intended scope.
- Background jobs, exports, reports, and other data-processing workflows preserve tenant boundaries.

Repeat these checks for every school-owned module as it is introduced.

Automated tenant-isolation tests remain future work until the corresponding authorization mechanisms and business endpoints exist.

## 10. Database Migration Tests

For each migration:

1. Apply it to a clean test database.
2. Verify that the expected tables, constraints, and indexes exist.
3. Verify that the application can use the resulting schema.
4. Test rollback behavior where the down migration is safe and supported.
5. Test the migration against a database with the expected previous schema.
6. Check how failures are reported and recovered from.

The current migration history includes:

- `000001_create_schools_table`
- `000002_create_auth_tables`
- `000003_add_refresh_token_families`

Do not test destructive rollback procedures against production data.

## 11. Test Data and Environment

- Keep test credentials separate from development and production credentials.
- Never commit secrets to test configuration.
- Use deterministic test fixtures where practical.
- Clean up test data safely.
- Avoid tests that depend on execution order.
- Use a dedicated database for tests that perform writes.
- Ensure tests cannot accidentally connect to production.
- Avoid printing tokens, password hashes, or credentials in test output.
- Make integration-test database configuration explicit and safe.

## 12. Before Committing Changes

From the backend directory, run:

```bash
go test -count=1 ./...
go vet ./...
```

From the repository root, review:

```bash
git diff --check
git status --short
```

Inspect changed files and verify that no secrets, generated binaries, or unintended files are included.

For changes involving database migrations, run relevant migration and database tests as well.

Documentation-only changes do not replace the need to verify any pending code changes in the working tree. Do not report a new test result until the relevant command has actually completed.

## 13. Before Production Release

The release checklist should include:

- [ ] All Go tests pass.
- [ ] Static analysis passes.
- [ ] Relevant PostgreSQL integration tests pass.
- [ ] HTTP integration tests pass.
- [ ] Refresh rotation, replay, concurrency, and logout behavior are verified.
- [ ] Authentication and authorization tests pass.
- [ ] Cross-school isolation tests pass for all school-owned resources.
- [ ] Database migrations have been tested.
- [ ] Dependency and vulnerability checks have been run.
- [ ] Load tests have been run for the expected workload.
- [ ] Backup restoration has been tested.
- [ ] Monitoring and error reporting have been reviewed.
- [ ] Production configuration and secret handling have been reviewed.

A checked item must be supported by an actual verification result.

## 14. Maintenance

Whenever a bug is fixed or functionality changes:

1. Add or update the relevant tests.
2. Run the affected tests.
3. Run the full backend test suite when appropriate.
4. Update `PROGRESS.md` with verified results.
5. Update this document if testing procedures or coverage change.
6. Keep completed testing evidence separate from planned work.


## 15. Approved Phase 1 RBAC and Account-Security Test Plan (Not Yet Implemented)

Use `docs/PHASE1-RBAC.md` as the source of truth for the permission catalogue and default role matrix. These are required future tests, not evidence of current coverage.

### 15.1 Staff identity and global username uniqueness

- Same username with different letter case is rejected even when attempted in different schools.
- A username resolves exactly one staff account; staff login does not need a school ID to disambiguate.
- A staff account from School A cannot authenticate into or access School B.
- A person with separate School A and School B accounts has independent credentials, sessions, disablement, and password resets.
- School Admin cannot create or modify another school's staff account.

### 15.2 Password lifecycle

- Initial temporary password requires change at first login.
- Staff self-service password change fails with an incorrect current password and succeeds with the correct current password.
- School Admin can reset a same-school staff password without the old password.
- Admin reset revokes all refresh sessions and immediately invalidates access tokens; next login requires password change.
- Cross-school admin password reset is denied.
- Audit event records actor, target, time, and result but never stores the password.

### 15.3 Teacher assignment and field-level tests

- A subject teacher sees only name, roll number, class, and section for students in assigned class/section and only permitted teaching-related data for assigned subject/class.
- Teaching assignment alone does not permit attendance read/mark/update.
- A class-in-charge teacher can read/mark attendance and same-day correct attendance only for the exact class/section in charge and within cutoff.
- Class-in-charge can read that class's basic student details, timetable, homework, and applicable notices.
- Class-in-charge cannot access another class, including another section in the same grade.
- Past attendance correction is denied to teachers and allowed only to authorized School Admins.
- Neither assignment grants guardian contacts, home address, fee/financial details, medical details, or other sensitive fields by default.
- Manipulating client-supplied school/class/section/subject/student IDs never bypasses backend checks.

### 15.4 Custom roles and permission overrides

- School Admin can create a custom role only within their own school.
- School Admin cannot create/assign Platform Admin or School Admin roles or platform permissions.
- Delegation ceiling prevents granting permissions the acting admin cannot delegate.
- Multiple roles combine as specified within a single staff account/school.
- Explicit user deny overrides role allow.
- Explicit allow is accepted only when the assigning admin is authorized to delegate that permission.
- Default deny applies when no effective permission exists.
- Permission changes take effect on the next protected request using an already-issued session.
- Overrides never bypass tenant, assignment, relationship, field-level, account, or school status checks.

### 15.5 Parent and child-link tests

- Parent account creation/linking occurs only after OTP verification and guardian matching.
- One parent can see all currently verified linked children across schools.
- Father and mother can use separate verified accounts to access the same child.
- Selecting a child does not bypass server-side link verification.
- Guessing another student's ID is denied.
- Removing one child link does not affect other child links.
- Parent can read only approved child profile, attendance, timetable, homework, published results, and applicable notices; write operations are denied in Phase 1.
- No student login account or student password flow is exposed in Phase 1.

### 15.6 Migration tests

- Upgrade from the existing v3 schema preserves user IDs and refresh-session foreign-key relationships.
- Global username migration detects cross-school case-insensitive duplicates and fails safely without silently renaming accounts.
- Fresh database installation produces the same target schema as an upgrade path.
- Rollback/forward-recovery procedures are reviewed and tested on disposable databases.
- Existing refresh-token rotation, replay detection, family-wide logout, and cross-family concurrency tests continue to pass after identity changes.


## 16. Current authentication/RBAC verification — 2026-11-03

The user reported the following successful local checks after pulling commit bea184f (before the subsequent V14 bootstrap hardening commits):

- go test -count=1 -p 1 ./... — PASS, exit code 0.
- go vet ./... — PASS, exit code 0.
- Focused TestPrincipalRefreshSessionCreateInTxPostgresIntegration assertions passed. That focused command emitted a Windows temporary test-executable cleanup/lock warning and returned exit code 1; the subsequent full suite completed with exit code 0.

The current branch has since added hidden terminal password input for the one-time Platform Admin bootstrap CLI, a caller-owned transaction helper for rollback-based PostgreSQL tests, and integration assertions for bootstrap contents, audit event, second-attempt rejection, and the V14 system-role-only trigger guard.

These newest changes are not yet locally verified. Pull the latest branch, run gofmt on changed Go files, then run:

```cmd
go test -count=1 -p 1 ./...
echo TEST_EXIT_CODE=%ERRORLEVEL%
go vet ./...
echo VET_EXIT_CODE=%ERRORLEVEL%
```

After explicit approval to apply V14 to school_management_test, run the focused integration test with the project's opt-in PostgreSQL integration environment enabled. Confirm it does not skip and exits with code 0. Do not run the bootstrap CLI against either database as part of migration verification. Never apply V14 to school_management without separate explicit approval.


## 17. HTTP/PostgreSQL RBAC integration — 2026-11-04

`TestRBACHTTPAuthorizationUsesPostgresState` starts the actual Gin router with the real `PrincipalAuthorizationRepository`, `RoleRepository`, `RoleService`, and `RoleHandler`, backed by the ephemeral PostgreSQL integration database. The principal repository and role repository use the same caller-owned transaction as the fixture, so all test writes are rolled back.

The test creates a temporary school and staff principal, then verifies:

- `GET /auth/me` returns 200 for the active principal loaded from PostgreSQL.
- `GET /school/roles` returns 403 while the principal has no role grants.
- A School Admin system-role assignment, inserted through the existing database authorization trigger and attributed to a Platform Admin with `platform.school_admins.create`, is picked up on the next protected request.
- The same `GET /school/roles` request then returns 200 from the real role handler and includes the School Admin role.
- Incrementing the principal's database `token_version` invalidates the already-issued test token; the next `GET /auth/me` returns 401.

The School Admin fixture includes a verified MFA credential because the live principal loader enforces MFA for School Admins. The staff fixture has no temporary-password gate. If the ephemeral database has never been bootstrapped, the test creates the first Platform Admin inside the same transaction using `BootstrapFirstInTx`; the deferred rollback removes the account, role grant, and audit event. No permanent Platform Admin is created.

The test uses the real `services.TokenService` rather than a fake validator. It generates an HS256 access token with the integration principal ID and current database token version, then passes that token through the actual authentication middleware. A request with one altered signature character must return `401 Unauthorized`; the unmodified token must authenticate successfully.

Cross-school tenant/resource isolation, teacher class/section/subject assignment-scope authorization, and HTTP write-path coverage for delegation/override/role assignment remain explicit follow-up work.

Verification: push run `41765751785` and Draft PR run `41765745170` on `ec55ea4` both passed V1–V16 migrations on ephemeral PostgreSQL 18, the full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1`, and `go vet ./...`. No persistent user database was modified.
