# School Management System — Project Handoff

> **Purpose:** This document is the source of truth for continuing the existing School Management System project. Update it after meaningful milestones and keep it committed to GitHub.
>
> **Last documented milestone:** `fec3b8a` — `feat(auth): add refresh token rotation and revocation`
>
> **Important:** Verify the current Git status and source files before assuming that this document or the local repository reflects the latest state.

## 1. Project Overview

- **Product:** Commercial multi-tenant School Management System.
- **Planning target:** 25–30 schools, approximately 400–500 students per school.
- **Frontend:** React + TypeScript + Vite.
- **Backend:** Go + Gin.
- **Database:** PostgreSQL.
- **Architecture:** Modular monolith with a shared PostgreSQL database and shared schema.
- **Tenant isolation:** Must be enforced by the backend on every applicable protected request.
- **Initial scope:** Foundation, authentication, RBAC, school onboarding, and core school-management workflows.
- **Excluded from initial scope:** Fee/payment gateway, native parent mobile app, live bus GPS tracking, and advanced AI analytics.
- **Working style:** Explain in Hindi/Hinglish, use a friendly tone, work on one small task at a time, provide exact commands and complete replacement files when appropriate, and wait for the user's output before proceeding.

The school and student counts are planning targets, not verified capacity limits.

## 2. Repository and Local Environment

- **Local project root:** `D:\school-management-system`
- **GitHub repository:** https://github.com/TheSumitNain/school-management-system
- **Branch:** `main`
- **Backend directory:** `D:\school-management-system\backend`
- **Go module:** `school-management-system`, located in `backend\go.mod`
- **Environment:** Windows CMD and VS Code
- **Default API port:** `8081`
- **Local database:** `school_management`

### Known endpoints

- `GET http://localhost:8081/health`
- `GET http://localhost:8081/ready`
- `POST http://localhost:8081/auth/login`
- `POST http://localhost:8081/auth/refresh`
- `POST http://localhost:8081/auth/logout`
- `GET http://localhost:8081/auth/me`

### Git checkpoint

The last documented authentication milestone is:

`fec3b8a feat(auth): add refresh token rotation and revocation`

This commit was reported pushed to `origin/main`. The last reported Git status after that push was clean and synchronized.

Documentation edits may now create uncommitted changes. Always check the actual state before committing or pushing.

Never commit `.env` files, passwords, JWT signing secrets, real student data, or other sensitive credentials. Never ask the user to paste real secret values into chat.

## 3. Project Structure

This is the known structure. Verify the actual filesystem before assuming that every listed file is present.

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

`backend\.env` is a local development configuration file and must remain untracked. Its presence in this illustrative structure does not mean it is committed to Git.

## 4. Backend Foundation

### 4.1 Configuration

File: `backend/internal/config/config.go`

Previously implemented configuration includes:

- Loading `.env` for local development while allowing externally supplied environment variables.
- `APP_ENV`, defaulting to development.
- `APP_PORT`, defaulting to `8081`.
- PostgreSQL connection settings.
- JWT issuer, signing-secret validation, and access-token expiry configuration.
- `REFRESH_TOKEN_EXPIRY_DAYS`, defaulting to 7 and accepting values from 1 through 30.
- Environment-specific database TLS settings.

Review the actual source and tests before changing or making further claims about configuration behavior.

Production secret management, TLS, certificate validation, and deployment-specific validation still require review.

### 4.2 PostgreSQL Connection

File: `backend/internal/database/connection.go`

The implemented foundation uses `pgxpool.Pool`, validates database configuration, establishes the connection, pings the database, and closes the pool if the initial ping fails.

The local development configuration may use `sslmode=disable`. This is not a production security recommendation. Production deployment must configure and verify TLS and certificate validation.

Database pool sizing, operational timeouts, recovery behavior, and integration tests remain areas for review.

### 4.3 API Server

File: `backend/cmd/api/main.go`

Implemented server foundation includes:

- Loading configuration.
- Connecting to PostgreSQL.
- Configuring Gin and HTTP server timeouts.
- Explicitly configuring trusted proxies.
- Health and readiness endpoints.
- Login, refresh, and logout routes.
- A protected `/auth/me` route.
- Graceful-shutdown handling.

Previously documented HTTP timeouts:

- `ReadHeaderTimeout`: 5 seconds.
- `ReadTimeout`: 15 seconds.
- `WriteTimeout`: 30 seconds.
- `IdleTimeout`: 60 seconds.

Verify the current source before relying on these values as final production settings.

### 4.4 Health and Readiness

`GET /health`

- Checks whether the API process is responding.
- Does not independently establish database readiness.

`GET /ready`

- Checks database readiness.
- Returns success when the database ping succeeds.
- Returns `503 Service Unavailable` when the readiness check fails.

### 4.5 Migration Runner

File: `backend/cmd/migrate/main.go`

Run migrations from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

**Caution:** Running the migration command changes the target database schema. Confirm the database target and current migration state before executing it. Never run a down migration casually or against production.

Making migration path resolution independent of the working directory and adding automated migration tests against a dedicated database remain follow-up tasks.

## 5. Authentication Implementation

The current backend includes password hashing, user and school repositories, an authentication service, JWT access-token generation and validation, login, authentication middleware, a protected identity endpoint, and a refresh-token lifecycle.

This is a meaningful authentication foundation, but it is **not a complete production authentication system**.

### 5.1 User Model and Database

Known model file:

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

The current user schema does not yet implement the complete role and permission model.

### 5.2 Password Service

File: `backend/internal/services/password.go`

Implemented behavior includes:

- bcrypt hashing with cost 12.
- Password-hash verification.
- Rejection of passwords shorter than 8 bytes or longer than 72 bytes.
- Rejection of whitespace-only passwords.

Tests: `backend/internal/services/password_test.go`.

The production policy still requires review. Do not claim that a minimum 12-character policy, compromised-password checks, or a finalized Argon2id policy is implemented unless the code and tests enforce it.

### 5.3 User Repository

File: `backend/internal/repository/user_repository.go`

Implemented operations include:

- Finding a user by school ID and case-insensitive username.
- Creating a user with a password hash.
- Handling duplicate school/username conflicts.

PostgreSQL repository integration and concurrency tests remain pending.

### 5.4 School Repository

File: `backend/internal/repository/school_repository.go`

Provides school lookup by ID.

The authentication flow checks school status before authenticating the user. Full tenant authorization across future application endpoints is not yet implemented.

### 5.5 Authentication Service

File: `backend/internal/services/auth_service.go`

Implemented operations include:

- Creating a user for an existing active school.
- Authenticating by school ID, username, and password.
- Rejecting inactive schools and disabled users.
- Returning a generic invalid-credentials error for unknown users and incorrect passwords.

Tests: `backend/internal/services/auth_service_test.go`.

User creation is an internal service operation, not a public self-registration endpoint.

### 5.6 JWT Access-Token Service

File: `backend/internal/services/token_service.go`

Implemented functionality includes:

- Generating JWT access tokens.
- Validating access tokens.
- Signing with HS256.
- Checking issuer and expiration.
- Validating the subject and school-related identity claims according to the implementation.

Tests: `backend/internal/services/token_service_test.go`.

Verify the actual code and tests before claiming additional properties such as immediate token revocation, clock-skew guarantees, or complete session invalidation.

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

File: `backend/internal/middleware/auth.go`

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

Tests: `backend/internal/middleware/auth_test.go`.

The middleware establishes identity. It does not implement RBAC, permission enforcement, immediate access-token revocation, or complete tenant isolation.

Do not assume all future routes are protected automatically. Every protected route must be deliberately wired to the appropriate middleware and authorization checks.

### 5.9 Protected Identity Endpoint

`GET /auth/me`

- Requires a valid access token.
- Uses the authentication middleware.
- Returns authenticated identity information, including user and school IDs.

The endpoint currently returns identity data rather than a complete profile retrieved from PostgreSQL. Full profile lookup and additional account-state checks remain pending.

### 5.10 Development Seed Command

File: `backend/cmd/devseed/main.go`

A development command exists for creating a test user through the authentication service. The known implementation uses school ID `1`.

Review its environment guard, seed credentials, school-ID assumptions, and deployment packaging before treating it as production-safe. It must not become a public API endpoint or run against a production database.

## 6. Refresh-Token Lifecycle

The refresh-token implementation was committed in:

`fec3b8a feat(auth): add refresh token rotation and revocation`

Known files include:

- `backend/internal/repository/refresh_session_repository.go`
- `backend/internal/services/session_service.go`
- `backend/internal/services/session_service_test.go`
- `backend/internal/handlers/auth_handler.go`
- `backend/internal/handlers/auth_handler_test.go`
- `backend/migrations/000003_add_refresh_token_families.up.sql`
- `backend/migrations/000003_add_refresh_token_families.down.sql`

Configuration, API wiring, and documentation were also updated.

### 6.1 Implemented Behavior

- Refresh tokens are opaque, cryptographically random tokens generated using 32 random bytes.
- Only SHA-256 hashes of refresh tokens are persisted.
- Refresh-token expiry is configurable using `REFRESH_TOKEN_EXPIRY_DAYS`, defaulting to 7 days with a supported range of 1–30 days.
- `POST /auth/refresh` rotates a presented refresh token and returns a new access/refresh token pair.
- `POST /auth/logout` revokes the presented refresh session.
- Migration v3 adds `family_id` to group refresh-token rotation chains.
- Reuse of a revoked token is intended to revoke active sessions in the associated token family.

### 6.2 Reported Verification

The following checks were reported from the user's local development environment:

- Login returned access and refresh tokens.
- Refreshing the initial token returned a new token pair.
- Replaying the old token was rejected.
- The subsequently rotated token was also rejected after replay, consistent with family-wide revocation.
- Logout returned success in a manual test, and a subsequent refresh attempt was rejected. That sequence may already have involved a revoked token family, so a separate clean-session logout test should be run for stronger confirmation.
- Unit and handler tests passed as part of the backend test suite.

These are reported checks, not proof of production-grade concurrency safety.

### 6.3 Important Security Limitation

Logout and refresh-token revocation do **not** automatically invalidate already issued stateless access JWTs.

An access token may remain valid until its expiration unless the API implements an additional revocation mechanism or checks server-side session state. Do not claim immediate access-token revocation is implemented.

### 6.4 Remaining Refresh-Token Work

- Add PostgreSQL integration tests for rotation, expiry, logout, and replay detection.
- Test concurrent refresh attempts against the same token.
- Verify transaction and locking behavior to prevent inconsistent concurrent rotations.
- Verify behavior for expired, unknown, revoked, and malformed refresh tokens.
- Add immediate access-token revocation if required by the final security design.
- Confirm that password changes and account deactivation invalidate the intended sessions.

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

### 7.3 Migration v3 — Refresh Token Families

Files:

- `backend/migrations/000003_add_refresh_token_families.up.sql`
- `backend/migrations/000003_add_refresh_token_families.down.sql`

The up migration adds a `family_id` UUID with a generated default and an index to group refresh-token rotations.

**Last reported local database state:** Migration version `3`, `dirty=false`. The migration was reported successfully applied to the user's local development database.

Verify the current migration state before future database changes or deployment. Do not rerun migrations merely because this handoff mentions them.

### 7.4 Known Table Responsibilities

`users`:

- Stores school-associated login identities.
- Stores password hashes, never plaintext passwords.
- Tracks account status and timestamps.

`refresh_sessions`:

- Stores the user association.
- Stores the unique refresh-token hash.
- Tracks expiry and revocation timestamps.
- Stores a family ID for refresh-token rotation chains.

Verify the actual SQL migrations before changing constraints, indexes, or field definitions.

### 7.5 Database Work Remaining

- PostgreSQL repository integration tests.
- Automated up-migration tests against a clean, dedicated test database.
- Migration rollback testing in a disposable environment.
- Database connection-pool tests.
- Refresh-session concurrency tests.
- A decision about whether application code or database triggers maintain `updated_at`.
- Additional indexes and query-performance review as the schema expands.

Never run destructive migration tests against the development database containing data the user needs, and never do so against production.

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

The final role list, scope, and permission matrix must be documented before implementing RBAC.

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
- Rate limiting.
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

### 10.1 Last Reported Backend Verification

After the refresh-token changes, these commands were reported successful from `D:\school-management-system\backend`:

```cmd
go test -count=1 ./...
go vet ./...
```

The reported test run included passing API, configuration, handler, middleware, and service packages. Some packages had no test files, including repository and database packages.

`git diff --check` was also reported without whitespace errors. Windows emitted a line-ending warning that LF may be replaced by CRLF. This warning alone is not a code failure.

These are historical results. Rerun the commands against the current working tree before committing new code or documentation.

### 10.2 Manual Authentication Checks Reported

- `/health` returned success.
- `/ready` confirmed API and database readiness.
- Login returned an access token and refresh token.
- Refresh rotated the refresh token.
- Reuse of an old refresh token was rejected.
- Family-wide revocation behavior was observed after replay.
- Logout returned success in a manual test; perform a separate clean-session logout test for stronger independent confirmation.

Do not claim untested behavior merely because a related endpoint exists.

### 10.3 Testing Gaps

- PostgreSQL repository integration tests.
- Refresh-token rotation and replay concurrency tests.
- Automated migration tests against a clean database.
- Database connection and pool behavior tests.
- Shutdown and timeout edge cases.
- Additional JWT and HTTP authentication tests.
- RBAC and tenant-isolation tests.
- Cross-school access-denial tests.
- Password reset, invitations, MFA, lockout, and rate-limit tests.
- Frontend and end-to-end tests.
- Security and deployment verification.

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
4. Keep API request/response documentation aligned with the actual handlers.
5. Keep database documentation aligned with the migration files.
6. Record known limitations and unverified behavior.
7. Never claim that tests passed unless their output was observed.
8. Keep the handoff current so a new conversation does not restart completed work.

## 12. Git and Delivery Workflow

Before committing:

1. Check the current branch and working tree.
2. Review tracked and untracked files.
3. Inspect the full diff and ensure no credentials are included.
4. Run `gofmt` only on modified Go files.
5. Run backend tests.
6. Run `go vet`.
7. Run `git diff --check`.
8. Verify that `.env` files are ignored and not tracked.
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
- Unit and handler tests for the implemented authentication foundation.
- Reported backend test and vet success for the refresh-token milestone.

### 13.2 Immediate Documentation Task

1. Replace the stale contents of this file with this corrected version.
2. Review `PROGRESS.md` and confirm it reflects commit `fec3b8a` and the implemented refresh-token lifecycle.
3. Review `README.md`, `CHANGELOG.md`, `docs/API.md`, `docs/DATABASE.md`, `docs/SECURITY.md`, and `docs/TESTING.md` for contradictions.
4. Review `docs/ARCHITECTURE.md` and `docs/REQUIREMENTS.md` for consistency with current decisions.
5. Inspect the complete Git diff.
6. Rerun verification before committing documentation changes.
7. Commit and push only after the reviewed documentation and tests are satisfactory.

### 13.3 Recommended Engineering Priorities

After the documentation audit:

1. Review the current authentication code and refresh-token repository for correctness and transaction/concurrency behavior.
2. Add PostgreSQL integration tests for refresh-token rotation, replay, expiry, and logout.
3. Design and implement RBAC and a documented permission matrix.
4. Implement server-side tenant authorization and cross-school denial tests.
5. Complete user-profile retrieval and account/session-state checks.
6. Implement admin MFA, login rate limiting, temporary lockouts, and security-event monitoring.
7. Implement secure school onboarding, invitations, and password-reset workflows.
8. Build academic, student, attendance, and parent features only after their authorization model is established.
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
> The last documented milestone is commit `fec3b8a`, `feat(auth): add refresh token rotation and revocation`. The refresh-token implementation includes opaque random tokens, SHA-256 hash storage, rotation, logout, and token-family replay handling. Migration v3 was reported applied locally with migration version 3 and `dirty=false`. Backend `go test -count=1 ./...` and `go vet ./...` were reported successful after this milestone. Verify the current local state before relying on those results.
>
> The backend has PostgreSQL connectivity, health/readiness endpoints, migrations for `schools`, `users`, and `refresh_sessions`, password hashing, user/school repositories, an authentication service, JWT access-token generation/validation, `POST /auth/login`, authentication middleware, `GET /auth/me`, `POST /auth/refresh`, and `POST /auth/logout`.
>
> Do not reimplement completed work from scratch. Inspect the current source and documentation, identify genuine gaps, and continue from the existing repository.
>
> Remaining work includes PostgreSQL integration and concurrency tests, full profile lookup, access-token revocation design, RBAC, complete tenant authorization, admin MFA, rate limiting, lockouts, invitations, password reset, and production hardening. Do not claim these are completed.
>
> First review the current Git status and documentation. Keep `README.md`, `PROGRESS.md`, `CHANGELOG.md`, and all `docs/*.md` files synchronized. Then verify tests, `go vet`, `git diff --check`, and secret exclusion before committing.
>
> Never ask me to paste real `.env` secrets. Do not claim the system is production-ready until outstanding security, testing, deployment, backup, and monitoring requirements have been addressed.
