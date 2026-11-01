> **Newest handoff — 2026-11-01:** Read `docs/CURRENT-BUILD-FIX-HANDOFF-20261101.md` before following older handoff notes. The V4 principal-auth cutover candidate now updates `main.go`, staff authentication, live principal/permission loaders, and affected tests. Full Go tests are not verified in the review environment (Go 1.27.1 unavailable); run the exact commands in that handoff on Windows. V4 migration has not been run and must not be run. Do not commit/push.

# School Management System — Project Handoff

> **Purpose:** This document is the primary handoff reference for continuing the existing School Management System project. Keep it synchronized with the source code, tests, database migrations, and other project documentation.
>
> **Last confirmed pushed repository checkpoint:** `cb6af3b` — `docs: finalize phase 1 RBAC and account decisions`
>
> **Previous code/test milestone:** `a705f9a` — `test(auth): add PostgreSQL refresh session integration tests`
>
> **Branch:** `main`
>
> **Repository:** https://github.com/TheSumitNain/school-management-system
>
> **Git status:** The supplied ZIP has no `.git` metadata, so its working-tree state cannot be determined. The latest prior local report included an untracked `APPLY-DOCS.txt`; inspect `git status -sb` in the actual local repository before committing.
>
> **Important:** These details describe the last reported state. Check the actual repository before assuming that no newer changes exist.

## 0.1 Latest V4 Work (Current)

The supplied archive root records commit `cb6af3b5abc5249fba420e2699d16e87b1d14664` (`cb6af3b`, `docs: finalize phase 1 RBAC and account decisions`). The uploaded ZIP has been inspected and its commit hash matches the user's reported checkpoint.

The user verified the development database: migration v3, `dirty = false`; 1 school; 1 user; 4 refresh sessions; 0 active and 0 expired. Username duplicate and canonical-format preflight checks returned no rows. Exact existing columns, indexes, and FK names were inspected.

A V4 SQL draft now exists at `backend/migrations/000004_add_principal_rbac_foundation.up.sql`. Read `docs/V4-IMPLEMENTATION-NOTES.md` before touching it. **Do not run V4 yet:** it has not been PostgreSQL-tested, and the Go code still uses user-ID JWT subjects and does not enforce RBAC or principal token versions. The existing staff account's role is unknown and must not be guessed.

The verified current refresh-session code uses family advisory locks, per-user cross-family advisory locks, replay detection, and family-wide revocation. Preserve all of these during the principal-aware session refactor.

## 1. Project Overview

- **Product:** Commercial multi-tenant School Management System.
- **Planning target:** 25–30 schools, approximately 400–500 students per school.
- **Frontend:** React + TypeScript + Vite.
- **Backend:** Go + Gin.
- **Database:** PostgreSQL.
- **Database access:** pgx / pgxpool.
- **Architecture:** Modular monolith with a shared PostgreSQL database and shared schema initially.
- **Tenant isolation:** Must be enforced by the backend for every applicable protected request.
- **Initial priorities:** Backend foundation, authentication, RBAC, school onboarding, and core school-management workflows.
- **Excluded from initial scope:** Fee/payment gateway, native parent mobile app, live bus GPS tracking, and advanced AI analytics.
- **Working style:** Explain in Hindi/Hinglish, use a friendly tone, work on one small task at a time, provide exact Windows CMD commands and complete replacement files when appropriate, and wait for the user's output before proceeding.

The school and student counts are planning targets, not verified capacity limits.

## 1.1 Approved Phase 1 Identity and RBAC Decisions (Important)

The user has finalized the Phase 1 role and account rules. The authoritative detailed matrix is `docs/PHASE1-RBAC.md`.

- Core roles: Platform Admin (global), School Admin (exactly one school), Teacher (exactly one school), Parent (verified linked children; links may span schools). Student login accounts are excluded from Phase 1.
- Every staff account is single-school. A person employed by multiple schools must receive separate accounts, credentials, and sessions at each school. Staff usernames are globally unique and case-insensitive. Current code does not yet meet global uniqueness; it currently requires school ID at login and has school-scoped username uniqueness.
- School Admin provisions staff with a temporary initial password. Staff must change it at first login and provide the current password for self-service changes. School Admin may reset same-school staff passwords without old password; reset revokes sessions/access, forces password change at next login, and is audited.
- Platform Admin provisions School Admin accounts. School Admin cannot create another School Admin or Platform Admin.
- School Admin may create school-scoped custom roles from platform-defined permissions, within an explicit delegation ceiling. Future roles (Clerk, Manager, Librarian, Gate Keeper, Helper, Bus Conductor, etc.) are not Phase 1 implementations.
- Staff may hold multiple roles only within their one school. Role permissions combine with per-user allow/deny overrides; explicit deny beats role allow, explicit authorized allow can add a permission, and default is deny. Permission changes apply on the next protected request.
- Teaching assignments and class-in-charge assignments are separate. Subject teachers see only student name, roll number, class, and section for assigned class/subject, plus permitted teaching data. Teaching assignment alone does not grant attendance access.
- Class-in-charge teachers may access attendance, same-day correction within the configured cutoff, basic student details, timetable, homework, and applicable notices only for their exact class/section. Past attendance correction is School Admin-only.
- Parent uses registered mobile + OTP; one verified account can access all officially linked children across schools. Parent selects a child, but every API request verifies the relationship. Parent access is read-only; no student login in Phase 1.

These are approved requirements, not implemented functionality. Do not begin migrations by blindly dropping `users.school_id` or adding universal multi-school staff memberships. First design a safe model for global Platform Admin identity, strictly one-school staff accounts, globally unique usernames, and parent-child links across schools while preserving current user IDs and refresh sessions.

## 2. Repository and Local Environment

- **Local project root:** `D:\school-management-system`
- **GitHub repository:** https://github.com/TheSumitNain/school-management-system
- **Branch:** `main`
- **Backend directory:** `D:\school-management-system\backend`
- **Go module:** `school-management-system`, located in `backend\go.mod`
- **Environment:** Windows CMD and VS Code
- **Default API port:** `8081`
- **Development database:** `school_management`
- **Dedicated integration-test database:** `school_management_test`

### Known endpoints

- `GET http://localhost:8081/health`
- `GET http://localhost:8081/ready`
- `POST http://localhost:8081/auth/login`
- `POST http://localhost:8081/auth/refresh`
- `POST http://localhost:8081/auth/logout`
- `GET http://localhost:8081/auth/me`

Verify the actual route registration and request/response structures before changing API documentation or relying on undocumented behavior.

### Latest Git checkpoint

- **Last confirmed pushed repository commit:** `cb6af3b`
- **Message:** `docs: reconcile authentication testing status`
- **Branch:** `main`
- **Remote:** `origin/main`
- **Last reported status:** local modifications existed in refresh-session repository/test and documentation files; verify with `git status -sb`.

The supplied source ZIP root records full commit `cb6af3b5abc5249fba420e2699d16e87b1d14664`. It includes the refresh-session cross-family concurrency implementation and regression test. The ZIP has no `.git` metadata; verify the actual local repository and remote before committing. The latest user-reported local status previously included an untracked `APPLY-DOCS.txt`; inspect it rather than deleting or confusing it with the archive `APPLY-CHANGES.txt`.

Never commit `.env` files, passwords, JWT signing secrets, real student data, or other sensitive credentials. Never ask the user to paste real secret values into chat.

## 3. Project Structure

Known project structure:

```text
D:\school-management-system
├── backend
│   ├── cmd
│   │   ├── api
│   │   ├── devseed
│   │   └── migrate
│   ├── internal
│   │   ├── config
│   │   ├── database
│   │   ├── handlers
│   │   ├── middleware
│   │   ├── models
│   │   ├── repository
│   │   └── services
│   ├── migrations
│   │   ├── 000001_create_schools_table.up.sql
│   │   ├── 000001_create_schools_table.down.sql
│   │   ├── 000002_create_auth_tables.up.sql
│   │   ├── 000002_create_auth_tables.down.sql
│   │   ├── 000003_add_refresh_token_families.up.sql
│   │   └── 000003_add_refresh_token_families.down.sql
│   ├── .env
│   ├── .env.example
│   ├── go.mod
│   └── go.sum
├── docs
│   ├── API.md
│   ├── ARCHITECTURE.md
│   ├── DATABASE.md
│   ├── HANDOFF.md
│   ├── REQUIREMENTS.md
│   ├── SECURITY.md
│   └── TESTING.md
├── README.md
├── PROGRESS.md
├── CHANGELOG.md
└── .gitignore
```

Verify the actual filesystem before assuming every listed file is present.

`backend\.env` is a local development configuration file and must remain untracked. Its presence in this illustrative structure does not mean it is committed to Git.

## 4. Backend Foundation

### 4.1 Configuration

Known file:

`backend/internal/config/config.go`

Implemented configuration includes:

- Environment-based configuration.
- `.env` loading for local development.
- Application environment, defaulting to development.
- Application port, defaulting to `8081`.
- PostgreSQL connection settings.
- Database configuration validation.
- JWT signing-secret validation and access-token configuration.
- `REFRESH_TOKEN_EXPIRY_DAYS`, defaulting to 7, with an allowed range of 1–30 days.
- Environment-specific database TLS settings.

The existing configuration uses environment variables. The `godotenv.Load()` behavior does not overwrite variables already present in the process environment.

Production secret management, TLS, certificate validation, and deployment-specific configuration validation still require review.

### 4.2 PostgreSQL Connection

Known file:

`backend/internal/database/connection.go`

The backend foundation uses `pgxpool.Pool`, validates database configuration, establishes the connection, pings the database, and closes the pool if the initial ping fails.

Local development may use `sslmode=disable`. This is not a production security recommendation. Production must use the approved TLS configuration and certificate validation.

Database pool sizing, operational timeouts, recovery behavior, and failure scenarios require further review.

### 4.3 API Server

Known file:

`backend/cmd/api/main.go`

The implemented server foundation includes:

- Configuration loading.
- PostgreSQL connection setup.
- Gin configuration.
- HTTP server timeouts.
- Explicit trusted-proxy configuration.
- Health and readiness endpoints.
- Login, refresh, and logout routes.
- A protected `/auth/me` route.
- Graceful shutdown.

Previously documented HTTP timeouts:

- `ReadHeaderTimeout`: 5 seconds.
- `ReadTimeout`: 15 seconds.
- `WriteTimeout`: 30 seconds.
- `IdleTimeout`: 60 seconds.

Verify the current source before treating these values as final production settings.

### 4.4 Health and Readiness

`GET /health`

- Checks whether the API process responds.
- Does not independently establish database readiness.

`GET /ready`

- Checks database readiness.
- Returns success when the database ping succeeds.
- Returns `503 Service Unavailable` when the readiness check fails.

### 4.5 Migration Runner

Known file:

`backend/cmd/migrate/main.go`

Run migrations from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

**Caution:** The migration command changes the target database schema. Confirm the database target and current migration state before executing it. Never run a destructive down migration casually or against production.

Migration path resolution independent of the working directory and automated migration tests against a dedicated clean database remain follow-up tasks.

## 5. Authentication Implementation

The backend has a working initial authentication foundation consisting of password hashing, user and school repositories, an authentication service, JWT access-token generation and validation, login, authentication middleware, a protected identity endpoint, and a refresh-token lifecycle.

This is a meaningful authentication foundation, but it is **not a complete production authentication system**.

### 5.1 User Model and Database

Known file:

`backend/internal/models/user.go`

The user model includes:

- `ID`
- `SchoolID`
- `Username`
- `PasswordHash`
- `Status`
- `CreatedAt`
- `UpdatedAt`

The password hash is excluded from JSON serialization.

The current user schema does not implement the complete role and permission model.

### 5.2 Password Service

Known file:

`backend/internal/services/password.go`

Implemented behavior includes:

- bcrypt hashing with cost 12.
- Password-hash verification.
- Rejection of passwords shorter than 8 bytes or longer than 72 bytes.
- Rejection of whitespace-only passwords.

Tests are in:

`backend/internal/services/password_test.go`

The production policy still requires review. Do not claim that a minimum 12-character admin password policy, compromised-password checks, or a finalized Argon2id policy is implemented unless the source code and tests enforce it.

### 5.3 User Repository

Known file:

`backend/internal/repository/user_repository.go`

Implemented operations include:

- Finding a user by school ID and case-insensitive username.
- Creating a user with a password hash.
- Handling duplicate school/username conflicts.

The broader repository test suite passes. Additional user-repository integration coverage and migration tests may still be needed.

### 5.4 School Repository

Known file:

`backend/internal/repository/school_repository.go`

Provides school lookup by ID.

The authentication flow checks school status before authenticating the user. Full tenant authorization across future application endpoints is not implemented yet.

### 5.5 Authentication Service

Known file:

`backend/internal/services/auth_service.go`

Implemented operations include:

- Creating a user for an existing active school.
- Authenticating by school ID, username, and password.
- Rejecting inactive schools and disabled users.
- Returning a generic invalid-credentials error for unknown users and incorrect passwords.

Tests are in:

`backend/internal/services/auth_service_test.go`

User creation is an internal service operation, not a public self-registration endpoint.

### 5.6 JWT Access-Token Service

Known file:

`backend/internal/services/token_service.go`

Implemented functionality includes:

- Generating JWT access tokens.
- Validating access tokens.
- Signing with HS256.
- Checking issuer and expiration.
- Validating subject and school-related identity claims according to the implementation.

Tests are in:

`backend/internal/services/token_service_test.go`

Verify the current source before claiming additional guarantees such as immediate token revocation, clock-skew handling, or complete session invalidation.

### 5.7 Login Endpoint

`POST /auth/login`

The request uses `school_id`, `username`, and `password`.

Example with placeholder values:

```json
{
  "school_id": 1,
  "username": "devtest",
  "password": "your-test-password"
}
```

Successful responses include an access token, a refresh token, Bearer token type, and sanitized user data. Password hashes must never be returned.

Previously reported manual checks showed valid development credentials succeeding and an invalid password returning `401 Unauthorized`.

### 5.8 Authentication Middleware

Known file:

`backend/internal/middleware/auth.go`

Implemented behavior includes:

- Requiring a Bearer authorization header.
- Validating access tokens through an injected token-validator interface.
- Rejecting missing, malformed, invalid, or unusable tokens.
- Rejecting invalid identity values, including non-positive user or school IDs.
- Storing authenticated user ID and school ID in Gin request context.
- Returning `401 Unauthorized` for invalid authentication.
- Returning `500 Internal Server Error` if the token validator is unavailable due to server misconfiguration.

Known context keys:

- `auth_user_id`
- `auth_school_id`

Tests are in:

`backend/internal/middleware/auth_test.go`

The middleware establishes identity. It does not implement RBAC, permission enforcement, immediate access-token revocation, or complete tenant isolation.

Every future protected route must deliberately use the appropriate middleware and authorization checks. Do not assume all routes are protected automatically.

### 5.9 Protected Identity Endpoint

`GET /auth/me`

- Requires a valid access token.
- Uses the authentication middleware.
- Returns authenticated identity information, including user and school IDs.

The endpoint currently returns identity data rather than a complete profile retrieved from PostgreSQL. Full profile lookup and additional account-state checks remain pending.

### 5.10 Development Seed Command

Known file:

`backend/cmd/devseed/main.go`

A development command exists for creating a test user through the authentication service. The known implementation uses school ID `1`.

Review its environment guard, seed credentials, school-ID assumptions, and deployment packaging before treating it as production-safe. It must not become a public API endpoint or run against a production database.

## 6. Refresh-Token Lifecycle

The refresh-token lifecycle implementation was previously committed in:

Earlier authentication work was followed by the PostgreSQL integration-test milestone:

`cb6af3b docs: finalize phase 1 RBAC and account decisions`

Known files include:

- `backend/internal/repository/refresh_session_repository.go`
- `backend/internal/repository/refresh_session_repository_integration_test.go`
- `backend/internal/services/session_service.go`
- `backend/internal/services/session_service_test.go`
- `backend/internal/handlers/auth_handler.go`
- `backend/internal/handlers/auth_handler_test.go`
- `backend/migrations/000003_add_refresh_token_families.up.sql`
- `backend/migrations/000003_add_refresh_token_families.down.sql`

Verify exact file names and source code before making further implementation changes.

### 6.1 Implemented Behavior

- Refresh tokens are opaque, cryptographically random tokens generated using 32 random bytes.
- Only SHA-256 hashes of refresh tokens are persisted.
- Refresh-token expiry is configurable using `REFRESH_TOKEN_EXPIRY_DAYS`, defaulting to 7 days with a supported range of 1–30 days.
- `POST /auth/refresh` rotates a presented refresh token and returns a new access/refresh token pair.
- `POST /auth/logout` revokes every active refresh session in the supplied token family; repeated logout is idempotent and unknown tokens are a no-op.
- Migration v3 adds `family_id` to group refresh-token rotation chains.
- Reuse of a revoked token revokes active sessions in the associated token family.
- Refresh rotation and logout acquire the same transaction-scoped PostgreSQL advisory lock derived from the family ID. Rotation also acquires a per-user transaction advisory lock to serialize concurrent rotations across the same user's token families. It performs a fresh `FOR UPDATE` lookup after acquiring both locks.

### 6.2 PostgreSQL Integration Tests — Completed

File:

`backend/internal/repository/refresh_session_repository_integration_test.go`

The test helper:

- Requires `RUN_POSTGRES_INTEGRATION_TESTS=1`.
- Refuses to proceed unless the configured database name is `school_management_test`.
- Loads local configuration for the test.
- Creates a dedicated test school and user.
- Cleans up the inserted user and school.
- Connects to PostgreSQL through `pgxpool`.

The dedicated database is separate from `school_management`, which is the development database.

The integration tests cover:

- Refresh-token rotation.
- Rejection of a replayed refresh token.
- Revocation of the replacement token when replay is detected.
- Family-wide logout revocation and idempotency.
- Rejection of an expired refresh token.
- Concurrent refresh attempts against the same token, expecting one successful rotation and one reuse-detection result.

**Last reported result:** PostgreSQL repository integration tests passed. Reported coverage includes rotation, replay-triggered family revocation, family-wide logout, idempotency, expiry rejection, and concurrent refresh reuse detection.

Command:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 -v ./internal/repository
```

The test output reported `PASS`. An earlier run emitted a Windows temporary executable cleanup warning after the test completed; a subsequent full backend test run completed successfully.

The integration tests improve confidence in refresh-session behavior, but they do not prove every possible database failure or concurrency scenario is safe.

### 6.3 Latest Full Verification

After the integration-test file was added, the user reported:

```cmd
go test -count=1 ./...
```

Result: PASS.

The reported output included successful API, config, handler, middleware, repository, and service tests. Some packages reported no test files.

The user also reported:

```cmd
go vet ./...
```

Result: PASS, with no diagnostics.

These results are historical. Rerun the checks after future code changes.

### 6.4 Important Security Limitation

Logout and refresh-token revocation do not automatically invalidate already-issued stateless access JWTs.

An access token may remain valid until expiration unless the API implements an additional revocation mechanism or checks server-side session state. Do not claim immediate access-token revocation is implemented.

### 6.5 Remaining Refresh-Token Work

- Add further tests for transaction failures and refresh/logout interleavings not covered by the current suite.
- Verified concurrent rotations across separate token families for the same user: the per-user advisory lock prevents a later rotation from leaving a replacement session active after an inactive-user revocation.
- Verify behavior for expired, unknown, revoked, and malformed refresh tokens.
- Review database failure handling and cleanup behavior.
- Add more tests for relevant edge cases identified during review.
- Design immediate access-token revocation if required by the final security design.
- Confirm that password changes and account deactivation invalidate the intended sessions.

Do not reimplement refresh-token rotation from scratch. Extend the existing implementation only where review identifies a genuine gap.

## 7. Database and Migration State

Database connection settings are stored locally in `backend/.env`. Never expose its contents.

### 7.1 Migration v1 — Schools

Files:

- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

The `schools` table includes a generated ID, school name, unique school code, status, and timestamps.

Known status values include:

- `pending_setup`
- `active`
- `suspended`

The down migration drops the table and its data. Do not execute destructive migrations without reviewing their impact and having an appropriate recovery plan.

### 7.2 Migration v2 — Authentication Tables

Files:

- `backend/migrations/000002_create_auth_tables.up.sql`
- `backend/migrations/000002_create_auth_tables.down.sql`

This migration creates the `users` and `refresh_sessions` tables.

The users table associates a user with a school. Username uniqueness is enforced within a school, with case-insensitive duplicate prevention.

The refresh-session table stores refresh-token hashes, expiry and revocation timestamps, and associated user information.

### 7.3 Migration v3 — Refresh Token Families

Files:

- `backend/migrations/000003_add_refresh_token_families.up.sql`
- `backend/migrations/000003_add_refresh_token_families.down.sql`

The up migration adds a `family_id` UUID with a generated default and an index to group refresh-token rotations.

**Last reported local database state:** Migration version `3`, `dirty=false`, for both the development database and the dedicated test database.

Verify the current migration state before future database changes or deployment. Do not rerun migrations merely because this handoff mentions them.

### 7.4 Database Work Remaining

- Automated migration tests against a clean database.
- Migration rollback tests in a disposable environment.
- Database connection and pool behavior tests.
- Review of transaction and concurrency edge cases.
- Query-performance and indexing review as the schema expands.
- A decision about how application code or database triggers maintain `updated_at`.
- Production migration and recovery procedure validation.

Never run destructive migration tests against the development database containing data the user needs or against production.

## 8. Architecture and Product Decisions

### 8.1 Multi-Tenant Architecture

- Start with a shared PostgreSQL database and shared schema.
- Include `school_id` in appropriate school-owned tables.
- Enforce tenant authorization on the server for every applicable protected request.
- Never treat a client-supplied school ID as proof of authorization.
- Consider PostgreSQL Row-Level Security later as defense in depth.
- Separate databases per school are not part of the initial plan.

These are architecture decisions and requirements. Complete tenant-isolation enforcement remains pending.

### 8.2 Roles and Permissions

Roles discussed for the product include:

- Platform Admin.
- School Admin.
- Teacher.
- Parent.
- Student.
- Future roles such as Driver, Librarian, Gatekeeper, Helper, Bus Conductor, and Clerk/Manager.

The Phase 1 role list, scope, and permission matrix are approved and documented in `docs/PHASE1-RBAC.md`. RBAC tables, policy evaluation, and endpoint enforcement remain unimplemented.

The product should support role-based defaults and configurable individual permissions by authorized administrators. Permission changes should apply on subsequent protected API requests, including for existing sessions.

### 8.3 Account and Login Decisions

- Platform Admin creates or approves schools.
- School Admin accounts are created or invited by an authorized Platform Admin.
- A School Admin can activate the account and set a password through a secure invitation workflow.
- Platform Admin may create a School Admin account for a school that requests a complete setup.
- Teacher accounts are created by authorized school staff.
- Teacher email is optional and is not the intended login identifier.
- Teacher login should use a generated username or school-specific ID.
- Parent login is planned around mobile OTP.
- One parent account may access multiple officially linked children.

Invitation and parent OTP workflows are planned, not yet implemented.

### 8.4 Parent Features — Initial Scope

Planned parent features include:

- Daily attendance.
- Timetable.
- Homework and assignments.
- Exam results and report cards.
- Notices and announcements.
- Student profile and basic details.

Parent-child authorization must be enforced by the backend before these features expose student data.

### 8.5 Academic Data Model Principles

- `schools` is the tenant root.
- Student identity should be separate from school-year enrollment and class placement.
- Parent-child relationships may be many-to-many.
- Subjects and teacher assignments should use normalized relational tables.
- The default academic structure is Classes 1–12, customizable by each school.
- Consider whether login identities and school memberships need separate tables before expanding the schema.
- Explain foreign keys gradually and use a small ER diagram before adding many tables.

### 8.6 Attendance — Initial Scope

Planned Phase 1 attendance includes:

- Daily class/section-wise attendance.
- Assigned teachers and School Admins can mark same-day attendance.
- Same-day cutoff is configurable by each school.
- Only School Admin can modify past attendance.
- Statuses: Present, Absent, Late, Leave.

The authorization, cutoff, and audit rules must be enforced by the backend when attendance endpoints are implemented.

### 8.7 Excluded From Initial Scope

- Fee/payment gateway.
- Native parent mobile app.
- Live bus GPS tracking.
- Advanced AI analytics.

These may be considered later.

## 9. Security Requirements

The following are product requirements, not proof of completed implementation.

### 9.1 Administrative Authentication

- Platform Admin and School Admin must use MFA at every login.
- No trusted-device exemption is planned for these administrative roles.
- Admin idle timeout target: 30 minutes.
- Platform Admin must not have public self-registration.
- School Admin accounts must be created or invited by an authorized Platform Admin.

MFA and idle-session enforcement remain pending.

### 9.2 Passwords and Sessions

- Planned minimum admin password length: 12 characters.
- Passphrases should be supported.
- Common and compromised passwords should be blocked.
- Forced periodic password expiry is not planned.
- Evaluate Argon2id against the existing bcrypt implementation before finalizing production password storage.
- Password-reset tokens should be single-use and expire after 30 minutes.
- Password changes and account deactivation should invalidate the intended sessions.
- MFA backup codes should be single-use.

These controls must not be described as implemented until the code and tests enforce them.

### 9.3 Login Abuse Protection

Planned controls:

- Five failed login attempts trigger a 15-minute lockout.
- Login and account/IP rate limiting.
- Generic authentication errors.
- Security-event logging and monitoring.
- MFA attempt protection.

These controls remain pending unless separately verified in the current code.

### 9.4 Authorization and Tenant Isolation

- Enforce permissions on every protected API request.
- Enforce school and record scope in the backend.
- Teachers should access only assigned classes and subjects.
- Parents should access only verified linked children.
- Drivers, when implemented, should access only assigned routes.
- `pending_setup` and `suspended` schools must not have normal application access.
- Audit sensitive permission and security changes.

The current authentication middleware establishes identity; it does not fulfill these complete authorization requirements.

### 9.5 Invitation Security

Planned invitation behavior:

- Single-use tokens.
- Expiry after 24 hours.
- Only token hashes stored server-side.
- Authorized account creation.
- Audit events for sensitive actions.

The invitation workflow remains pending.

### 9.6 Deployment Security

Before production deployment:

- Verify secret management.
- Enforce database TLS and certificate validation.
- Review CORS, trusted proxies, HTTPS, and secure transport.
- Review logging and avoid recording passwords, tokens, or sensitive personal data.
- Establish backups, restoration procedures, monitoring, and incident response.
- Conduct a security review and verify dependency updates.

Do not describe the system as production-ready while these controls remain incomplete.

## 10. Testing and Verification

### 10.1 Last Reported Full Backend Verification

The following commands completed successfully after the integration-test file was added. Run them from `D:\school-management-system\backend`.

```cmd
go test -count=1 ./...
go vet ./...
```

**Reported result:** Both commands passed.

The test output included:

- `cmd/api`
- `internal/config`
- `internal/handlers`
- `internal/middleware`
- `internal/repository`
- `internal/services`

The following packages reported no test files:

- `cmd/devseed`
- `cmd/migrate`
- `internal/database`
- `internal/models`

### 10.2 PostgreSQL Integration-Test Results

The repository integration-test command completed successfully:

```cmd
go test -count=1 -v ./internal/repository
```

Reported scenarios passed:

- [x] Rotation and replay revoke the token family.
- [x] Logout revocation is idempotent.
- [x] Expired refresh tokens are rejected.
- [x] Concurrent refresh requests detect reuse.

The integration test requires `RUN_POSTGRES_INTEGRATION_TESTS=1` and refuses to proceed unless `DB_NAME=school_management_test`.

### 10.3 Manual Authentication Checks Reported

- `/health` returned success.
- `/ready` confirmed API and database readiness.
- Login returned an access token and refresh token.
- Refresh rotated the refresh token.
- Reuse of an old refresh token was rejected.
- Family-wide revocation behavior was observed after replay.
- Logout returned success in a manual test; the automated PostgreSQL integration test separately verifies logout revocation and idempotency.

These checks do not establish production readiness.

### 10.4 Remaining Testing Gaps

- Automated migration tests against a clean database.
- Additional refresh-token failure and concurrency scenarios.
- Database connection and pool behavior tests.
- Shutdown and timeout edge cases.
- RBAC and tenant-isolation tests.
- Cross-school access-denial tests.
- Password reset, invitations, MFA, lockout, and rate-limit tests.
- Frontend and end-to-end tests.
- Security and deployment verification.
- Load and performance testing.

Passing the current tests does not establish production readiness.

## 11. Documentation Maintenance

Keep the following documents synchronized:

- `README.md`
- `PROGRESS.md`
- `CHANGELOG.md`
- `docs/REQUIREMENTS.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/API.md`
- `docs/SECURITY.md`
- `docs/TESTING.md`
- `docs/HANDOFF.md`

Rules:

1. Separate implemented features from planned requirements.
2. Update `PROGRESS.md` after a meaningful milestone.
3. Update `CHANGELOG.md` when a feature or significant documentation change is completed.
4. Keep API request/response documentation aligned with actual handlers.
5. Keep database documentation aligned with migration files.
6. Record known limitations and unverified behavior.
7. Never claim that tests passed unless their output was observed.
8. Keep the handoff current so a new conversation can continue without repeating completed work.
9. Do not mark the documentation audit complete until the relevant files have been reviewed against the implementation.

## 12. Git and Delivery Workflow

Before committing:

1. Check the current branch and working tree.
2. Review tracked and untracked files.
3. Inspect the complete diff and ensure no credentials are included.
4. Run `gofmt` only on modified Go files.
5. Run backend tests.
6. Run `go vet`.
7. Run `git diff --check`.
8. Verify `.env` files are ignored and not tracked.
9. Review the staged diff.
10. Commit with a meaningful message.
11. Push only after verifying the intended files are included.
12. Confirm remote synchronization and a clean working tree.

Useful commands from the project root:

```cmd
cd /d D:\school-management-system
git status -sb
git log -1 --oneline
git diff --check
git check-ignore backend/.env
git ls-files backend/.env
```

`git ls-files backend/.env` should print nothing if that file is not tracked. Never display or paste `.env` contents.

Backend verification:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 ./...
go vet ./...
```

After running `gofmt`, inspect the resulting diff to ensure formatting did not introduce unrelated changes.

## 13. Current Status and Next Steps

### 13.1 Completed Foundation

- PostgreSQL connection foundation.
- Health and readiness endpoints.
- Migrations for schools and authentication tables.
- Password hashing and user authentication service.
- JWT access-token generation and validation.
- Login endpoint.
- Authentication middleware.
- Protected `/auth/me` endpoint.
- Refresh-token storage and rotation.
- Refresh-token logout/revocation flow.
- Migration v3 for refresh-token families.
- PostgreSQL refresh-session integration tests for rotation, replay, logout, expiry, and concurrency.
- Reported full backend test-suite success.
- Reported `go vet` success.
- Code/test milestone `a705f9a`, documentation checkpoint `d85e1ef`, and latest concurrency-hardening checkpoint `cb6af3b` were reported committed and pushed.

### 13.2 V4 Migration Draft and Application Cutover

The supplied `cb6af3b` archive contains the current authentication source and the approved RBAC/account documentation. A V4 schema migration draft has now been added, with implementation notes in `docs/V4-IMPLEMENTATION-NOTES.md`. The migration is not applied or PostgreSQL-tested. The source application still uses user-ID JWT subjects and has no RBAC enforcement; do not treat the schema draft as a completed security feature.

Next engineering sequence:

1. Review and statically inspect the V4 migration and its forward-only rollback warning.
2. Build and test the principal-aware Go auth/session model, preserving family-level and cross-family locking, replay detection, and revocation.
3. Implement database-backed principal status/token-version validation and RBAC evaluation on every protected request.
4. Add migration integration tests on a disposable PostgreSQL 18 database, including compatibility with existing staff login/session creation.
5. Explicitly confirm the existing staff account's intended role; do not guess it.
6. Only after all tests pass, back up and apply V4 to the intended development database.
7. Update docs and changelog from verified implementation/test results, then review and commit/push the exact diff.

### 13.3 Recommended Engineering Priorities

1. Verify the actual local Git state and confirm it matches the supplied `cb6af3b` source checkpoint; preserve the untracked `APPLY-DOCS.txt`.
2. Review the V4 migration draft and complete Go principal-aware login, JWT claims, refresh-session rotation/revocation, principal status/token-version checks, and authorization middleware.
3. Implement RBAC evaluation, custom roles, bounded permission overrides, and next-request permission updates with automated tests.
4. Add migration integration tests against a disposable PostgreSQL 18 database. Do not apply V4 to the development database until those tests pass.
5. Explicitly confirm the existing staff account's role and provision its principal-role assignment through a controlled, audited operation.
6. Complete teacher teaching/class-in-charge assignment enforcement, parent-child linking and OTP, staff password lifecycle, mandatory admin MFA, login rate limiting, lockout, and audit workflows.
7. Implement secure school onboarding, invitations, and password-reset workflows.
8. Build academic, student, attendance, timetable, homework, results, and parent features only after authorization enforcement is in place.
9. Perform production deployment, monitoring, backup, recovery, and security reviews.

Do not reimplement refresh-token rotation from scratch. Inspect the existing code and extend it only where verification identifies a real gap.

## 14. New-Chat Continuation Prompt

Copy the following into a new ChatGPT conversation when needed:

> Continue my existing commercial School Management System project using `docs/HANDOFF.md` as the primary handoff reference in `https://github.com/TheSumitNain/school-management-system`.
>
> Local project root: `D:\school-management-system`. I use Windows CMD and VS Code. The stack is React + TypeScript + Vite, Go + Gin, and PostgreSQL. The planning target is 25–30 schools with approximately 400–500 students per school.
>
> Explain in Hindi/Hinglish with a friendly tone. Work on one small task at a time. Provide exact commands and complete replacement code when appropriate. Explain why each step is needed and wait for my output before proceeding. Do not assume a command succeeded without seeing the result.
>
> The latest user-reported repository checkpoint represented by the supplied ZIP is `cb6af3b`, `docs: finalize phase 1 RBAC and account decisions`; the previous code/test milestone was `a705f9a`, `test(auth): add PostgreSQL refresh session integration tests`. The ZIP root records full hash `cb6af3b5abc5249fba420e2699d16e87b1d14664`. The refresh-token implementation includes opaque random tokens, SHA-256 hash storage, rotation, logout, token-family replay handling, family-wide advisory locking, and per-user cross-family serialization. The user reported `go test -count=1 ./...`, the opt-in PostgreSQL refresh-session integration suite (including the cross-family inactive-user race regression), and `go vet ./...` passing after the concurrency changes. Verify the current local state before relying on those results.
>
> The backend includes PostgreSQL connectivity, health/readiness endpoints, migrations for `schools`, `users`, and `refresh_sessions`, password hashing, user/school repositories, an authentication service, JWT access-token generation/validation, `POST /auth/login`, authentication middleware, `GET /auth/me`, `POST /auth/refresh`, and `POST /auth/logout`.
>
> Do not reimplement completed work from scratch. Inspect the current source and documentation, identify genuine gaps, and continue from the existing repository.
>
> Remaining work includes additive identity/RBAC schema design, global username migration, authorization enforcement, full profile lookup, immediate access-token revocation, parent OTP, staff provisioning/password lifecycle, admin MFA, rate limiting, lockouts, invitations, and production hardening. Do not claim these are completed.
>
> First check Git status and finish synchronizing the documentation. Then verify tests, `go vet`, `git diff --check`, and secret exclusion before committing.
>
> Never ask me to paste real `.env` secrets. Do not claim the system is production-ready until outstanding security, testing, deployment, backup, and monitoring requirements have been addressed.

## 15. Latest Repository Checkpoint — 2026-11-01 (V4 C1–C14)

The V4 C14 source checkpoint is ecaf9af211289f52a46f79ebffa0f41f62b06d5a — fix(db): prevent permission catalogue deletion. Subsequent documentation commits record this checkpoint in the handoff and progress files.

- V4 C1–C11: prior source-level corrective audit items are documented in docs/V4-IMPLEMENTATION-NOTES.md.
- V4 C12: principal_roles assignment identity/provenance is immutable; rows cannot be deleted; new assignments must start active; revocation cannot be reversed. Re-grants require a new row.
- V4 C13: permission delegation identity, capabilities and grantor provenance are immutable; rows cannot be deleted; new delegations must start active. Combined with C6, revocation is irreversible and restoring authority requires a new row.
- V4 C14: permission catalogue rows cannot be deleted, including unused keys. Obsolete permissions must be deactivated while retaining stable keys.
- The SQL changes are in backend/migrations/000004_add_principal_rbac_foundation.up.sql; C12–C14 are also recorded in docs/V4-IMPLEMENTATION-NOTES.md and CHANGELOG.md.
- All C1–C14 migration findings are source-level review/corrections only. PostgreSQL 18 migration execution and integration verification have NOT been completed. DO NOT apply V4 to the development database.
- The 2026-11-01 Go principal-auth build-fix candidate is described in docs/CURRENT-BUILD-FIX-HANDOFF-20261101.md. Full Go tests and vet were not verified in that review environment; use the user's Windows Go 1.27.1 environment and inspect actual command output.
- The effective-permission resolver intentionally excludes custom-role grants and ALLOW overrides until exact delegation-ID linkage is implemented and checked. Explicit DENY overrides remain fail-closed.
- No Phase 1 business-resource routes currently establish full student, attendance, teacher-assignment, parent-child, or tenant/resource authorization. Do not call the RBAC system complete or production-ready.
- The latest GitHub changes are not yet confirmed pulled into the user's local working tree. The last reported local status had modified and untracked files; preserve them. Do not clean, reset, overwrite, or commit those files. After checking local status, fast-forward pull from origin main if safe.

### Next engineering sequence

1. User pulls the latest GitHub main without disturbing local modifications and confirms git status -sb.
2. Review and verify the Go principal-auth build-fix candidate on Windows with Go 1.27.1; run go test -count=1 -p 1 ./... and go vet ./... from backend.
3. Continue source audit from C15; inspect before deciding the next finding. Do not assume a C15 issue without evidence.
4. Add disposable PostgreSQL 18 migration tests from a v3 snapshot, including upgrade, constraint/trigger behavior, and rollback/forward-recovery checks.
5. Only after the Go cutover and migration tests pass, plan a backed-up, controlled V4 application. The user must explicitly approve applying V4 to the development database.

