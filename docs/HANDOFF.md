# School Management System — Project Handoff

> **Purpose:** This is the source-of-truth handoff for continuing the project in a new ChatGPT conversation. Update it after meaningful milestones and keep it committed to the GitHub repository.

## 1. Project Overview

- **Product:** Commercial multi-tenant School Management System.
- **Initial planning target:** 25–30 schools, approximately 400–500 students per school (roughly 10,000–15,000 students).
- **Frontend:** React + TypeScript + Vite.
- **Backend:** Go + Gin.
- **Database:** PostgreSQL.
- **Initial architecture:** Modular monolith with a shared database and shared schema.
- **Initial payment scope:** Fee/payment gateway excluded for now.
- **Future scope:** Parent mobile app, live bus tracking, advanced AI/analytics and additional roles/features.
- **Working style:** Explain in Hindi/Hinglish, use a friendly tone, work on one small task at a time, provide exact commands/code and wait for the user's result before proceeding.

The school and student counts are planning targets, not verified capacity limits.

## 2. Repository and Local Environment

- **Local project root:** `D:\school-management-system`
- **GitHub repository:** https://github.com/TheSumitNain/school-management-system
- **Branch:** `main`
- **Environment:** Windows CMD and VS Code.
- **Backend directory:** `D:\school-management-system\backend`
- **Go module:** `school-management-system`, located in `backend\go.mod`.
- **Default API port:** `8081`
- **Health endpoint:** `http://localhost:8081/health`
- **Readiness endpoint:** `http://localhost:8081/ready`
- **Login endpoint:** `POST http://localhost:8081/auth/login`
- **Protected identity endpoint:** `GET http://localhost:8081/auth/me`
- **Local database:** `school_management`

The last reported repository state was `main...origin/main` before the current authentication middleware and documentation updates were committed. The current Git state has not been verified in this handoff update.

Always verify the current Git status, latest commit, and remote synchronization before committing. Never assume the working tree is clean without checking.

Never commit `.env` files, passwords, JWT signing secrets, or real student data. Never ask the user to paste secret values into chat.

## 3. Current Repository Structure

The following structure reflects the known project files. Verify the actual filesystem before assuming every file is present.

```text
D:\school-management-system
├── backend
│   ├── cmd
│   │   ├── api
│   │   │   ├── main.go
│   │   │   └── main_test.go
│   │   ├── devseed
│   │   │   └── main.go
│   │   └── migrate
│   │       └── main.go
│   ├── internal
│   │   ├── config
│   │   │   ├── config.go
│   │   │   └── config_test.go
│   │   ├── database
│   │   │   └── connection.go
│   │   ├── handlers
│   │   │   ├── auth_handler.go
│   │   │   └── auth_handler_test.go
│   │   ├── middleware
│   │   │   ├── auth.go
│   │   │   └── auth_test.go
│   │   ├── models
│   │   │   └── user.go
│   │   ├── repository
│   │   │   ├── school_repository.go
│   │   │   └── user_repository.go
│   │   └── services
│   │       ├── auth_service.go
│   │       ├── auth_service_test.go
│   │       ├── password.go
│   │       ├── password_test.go
│   │       ├── token_service.go
│   │       └── token_service_test.go
│   ├── migrations
│   │   ├── 000001_create_schools_table.up.sql
│   │   ├── 000001_create_schools_table.down.sql
│   │   ├── 000002_create_auth_tables.up.sql
│   │   └── 000002_create_auth_tables.down.sql
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
├── .gitignore
├── README.md
├── PROGRESS.md
└── CHANGELOG.md
```

The local `backend\.env` file is shown only to document the expected development setup. It must remain untracked by Git.

## 4. Backend Foundation

### 4.1 Configuration

File: `backend/internal/config/config.go`

Previously implemented configuration includes:

- Loading `.env` for local development while allowing externally supplied environment variables.
- Defaulting `APP_ENV` to `development`.
- Defaulting `APP_PORT` to `8081`.
- Loading PostgreSQL connection settings.
- Configuring the JWT issuer and access-token expiry.
- Validating JWT signing-secret length and token-expiry settings.
- Separating local development database TLS settings from production requirements.

Review the actual configuration code before changing behavior. Production TLS, environment validation, error handling, and secret management still require deployment-specific verification.

### 4.2 PostgreSQL Connection

File: `backend/internal/database/connection.go`

Previously implemented behavior includes:

- Validating required database settings.
- Building PostgreSQL connection configuration.
- Using `pgxpool.Pool`.
- Pinging PostgreSQL during connection setup.
- Closing the pool if the initial ping fails.

The local development configuration uses `sslmode=disable` by default. This is not a production security recommendation. Configure and verify TLS and certificate validation for production.

Connection-pool limits, operational timeouts, and database availability behavior require further review.

### 4.3 API Server

File: `backend/cmd/api/main.go`

Implemented foundation includes:

- Loading application configuration.
- Connecting to PostgreSQL.
- Configuring Gin and HTTP server timeouts.
- Explicitly configuring trusted proxies.
- Registering health and readiness endpoints.
- Registering `POST /auth/login`.
- Registering protected `GET /auth/me`.
- Attempting graceful shutdown.

Previously documented HTTP timeouts:

- `ReadHeaderTimeout`: 5 seconds.
- `ReadTimeout`: 15 seconds.
- `WriteTimeout`: 30 seconds.
- `IdleTimeout`: 60 seconds.

Verify these settings against the current source before changing or documenting them as final production settings.

#### `GET /health`

- Checks whether the API process is responding.
- Does not independently establish database readiness.

#### `GET /ready`

- Checks database readiness.
- Returns success when the database ping succeeds.
- Returns `503 Service Unavailable` when the readiness check fails.

#### `POST /auth/login`

- Accepts `school_id`, `username`, and `password`.
- Authenticates the user through the authentication service.
- Returns an access token and sanitized user data on success.
- Rejects invalid credentials.
- Does not return the password hash.

#### `GET /auth/me`

- Requires an `Authorization: Bearer <access-token>` header.
- Uses the authentication middleware to validate the access token.
- Extracts the authenticated user's ID and school ID from validated claims.
- Returns the authenticated identity IDs.
- Does not yet fetch and return the complete user profile.
- Does not itself enforce role-based permissions or complete tenant authorization.

The middleware is currently applied to `/auth/me`; do not assume all future protected routes are automatically secured.

#### Remaining server hardening

- Verify graceful-shutdown timeout and force-close behavior.
- Review startup/shutdown error handling and resource cleanup.
- Verify trusted-proxy configuration for the deployment.
- Add deployment-specific health and readiness checks.

### 4.4 API and Middleware Tests

Known files:

- `backend/cmd/api/main_test.go`
- `backend/internal/handlers/auth_handler_test.go`
- `backend/internal/middleware/auth_test.go`

Reported test coverage includes API health/readiness behavior, login-handler behavior, middleware authorization-header validation, and `/auth/me` route behavior.

Middleware tests cover valid tokens, missing or malformed authorization headers, invalid tokens, invalid identity claims, nil claims, and a nil validator.

Route tests cover valid, missing, and invalid access tokens on `/auth/me`.

Review the current test files before claiming coverage beyond these cases.

### 4.5 Migration Runner

File: `backend/cmd/migrate/main.go`

The migration runner uses the PostgreSQL configuration and migration files. It handles the no-change migration result.

Run migrations from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

Making migration path resolution independent of the working directory and testing migrations against a dedicated database remain follow-up tasks.

## 5. Authentication Foundation

The initial authentication implementation includes password hashing, user authentication, JWT access-token generation and validation, a login handler, authentication middleware, and a protected identity endpoint.

This is **not yet a complete production authentication system**.

### 5.1 User Model

File: `backend/internal/models/user.go`

The model includes:

- `ID`
- `SchoolID`
- `Username`
- `PasswordHash`
- `Status`
- `CreatedAt`
- `UpdatedAt`

The password hash is excluded from JSON serialization.

The current users schema does not yet contain a role column.

### 5.2 Password Service

File: `backend/internal/services/password.go`

Implemented behavior includes:

- bcrypt hashing with cost 12.
- Password hash verification.
- Rejection of passwords shorter than 8 bytes or longer than 72 bytes.
- Rejection of whitespace-only passwords.

Tests: `backend/internal/services/password_test.go`.

The current implementation must be reconciled with the stronger planned production password policy. Do not claim a 12-character minimum or compromised-password checks are implemented until the code and tests enforce them.

### 5.3 User Repository

File: `backend/internal/repository/user_repository.go`

Implemented operations include:

- Finding a user by school ID and case-insensitive username.
- Creating a user with a password hash.
- Handling duplicate school/username conflicts.

PostgreSQL repository integration tests remain pending.

### 5.4 School Repository

File: `backend/internal/repository/school_repository.go`

Implements school lookup by ID.

The authentication flow depends on checking the school's current status before authenticating its users.

### 5.5 Authentication Service

File: `backend/internal/services/auth_service.go`

Implemented internal operations include:

- Creating a user for an existing active school.
- Authenticating a user using school ID, username, and password.
- Rejecting inactive schools and disabled users.
- Returning a generic invalid-credentials error for unknown users and incorrect passwords.

Tests: `backend/internal/services/auth_service_test.go`.

These are internal service-layer operations. User creation is not exposed as a public registration endpoint.

### 5.6 JWT Access-Token Service

File: `backend/internal/services/token_service.go`

Implemented functionality includes:

- Generating JWT access tokens.
- Validating access tokens.
- Signing with HS256.
- Checking issuer and expiration.
- Validating the token subject and school-related claims according to the service implementation.

The service uses configuration for the signing secret, issuer, and access-token expiry.

Tests: `backend/internal/services/token_service_test.go`.

Review the actual implementation and tests before making claims about additional claims, clock-skew handling, or revocation behavior.

### 5.7 Login Handler

Files:

- `backend/internal/handlers/auth_handler.go`
- `backend/internal/handlers/auth_handler_test.go`

Implemented endpoint:

`POST /auth/login`

Request example:

```json
{
  "school_id": 1,
  "username": "devtest",
  "password": "your-test-password"
}
```

These values are placeholders, not real credentials.

Successful responses include:

- A success status.
- `access_token`.
- `token_type` set to `Bearer`.
- A sanitized user object containing user ID, school ID, username, and status.

The response does not include the user's password hash.

Previously reported manual verification:

- Valid development credentials returned a successful response and an access token.
- An invalid password returned HTTP `401 Unauthorized`.

Inactive users and inactive schools are handled as invalid credentials by the current authentication service, generally resulting in `401` rather than a distinct `403`.

### 5.8 Authentication Middleware

File: `backend/internal/middleware/auth.go`

Implemented behavior includes:

- Requiring a Bearer authorization header.
- Validating the access token through an injected token-validator interface.
- Rejecting missing, malformed, invalid, or unusable tokens.
- Rejecting missing or invalid identity values, including a non-positive user ID or school ID.
- Storing the authenticated user ID and school ID in Gin request context.
- Returning a generic `401 Unauthorized` response for invalid authentication.
- Returning `500 Internal Server Error` if the token validator is unavailable due to server misconfiguration.

Context keys:

- `auth_user_id`
- `auth_school_id`

Tests: `backend/internal/middleware/auth_test.go`.

The middleware establishes authenticated identity only. It does not implement RBAC, permission checks, account/session revocation, or complete tenant isolation.

### 5.9 Protected Identity Endpoint

`GET /auth/me` is registered in `backend/cmd/api/main.go` and protected by the authentication middleware.

The current endpoint returns:

- `status`
- `user_id`
- `school_id`

A valid access token is required. Missing or invalid authentication is rejected.

The endpoint currently returns identity IDs rather than a complete profile retrieved from PostgreSQL. Full profile lookup and additional account-state checks remain follow-up tasks.

### 5.10 Development Seed Command

File: `backend/cmd/devseed/main.go`

A development-only command exists for creating a test user through the authentication service.

It requires development environment configuration and development seed credentials. The current implementation uses school ID `1`.

Review its environment guard, school-ID behavior, command documentation, and deployment packaging before treating it as production-safe. It must not be exposed as a public API endpoint or run against a production database.

### 5.11 Authentication Not Yet Implemented

- Refresh-token issuance, rotation, and refresh HTTP endpoint.
- Refresh-token replay protection and complete session lifecycle.
- Logout HTTP endpoint.
- Complete session invalidation after password changes or account deactivation.
- Full user-profile retrieval from `/auth/me`.
- Access-token revocation enforcement.
- RBAC and permission enforcement.
- Complete tenant authorization and cross-school authorization tests.
- Login rate limiting, lockout enforcement, and security-event monitoring.
- MFA and recovery workflows.
- Password-reset and invitation workflows.
- PostgreSQL repository integration tests.
- Production authentication and deployment security review.

The existence of a `refresh_sessions` table does not mean refresh-token flows are operational.

## 6. Database and Migration State

Connection settings are stored locally in `backend/.env`. Credentials must remain secret.

### 6.1 Migration v1 — Schools

Files:

- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

The `schools` table includes:

- `id`: generated primary key.
- `name`: school name.
- `code`: unique school code.
- `status`: `pending_setup`, `active`, or `suspended`.
- `created_at` and `updated_at`: timestamps.

The down migration drops the table and its data. Never run destructive migrations casually in production.

### 6.2 Migration v2 — Authentication Tables

Files:

- `backend/migrations/000002_create_auth_tables.up.sql`
- `backend/migrations/000002_create_auth_tables.down.sql`

The migration creates `users` and `refresh_sessions`.

#### `users`

- `id`: generated primary key.
- `school_id`: foreign key to `schools(id)` with `ON DELETE RESTRICT`.
- `username`: username scoped to a school.
- `password_hash`: stored password hash.
- `status`: `active` or `disabled`.
- `created_at` and `updated_at`: timestamps.
- A unique index on `(school_id, LOWER(username))` prevents case-insensitive duplicate usernames within a school.

#### `refresh_sessions`

- `id`: generated primary key.
- `user_id`: foreign key to `users(id)` with `ON DELETE CASCADE`.
- `token_hash`: unique hash of a refresh token.
- `expires_at`: session expiry timestamp.
- `revoked_at`: optional revocation timestamp.
- `created_at`: creation timestamp.
- Indexes support user lookup and expiration-time queries.

Refresh-session storage exists, but the HTTP refresh/logout flow is pending.

#### Local migration verification

The user previously reported successful migration execution and verified the presence of `schools`, `users`, and `refresh_sessions` in the local database.

Recheck the migration version and dirty state before relying on this database for further work or deployment.

### 6.3 `updated_at` Behavior

The columns default to `NOW()` on insert. They do not automatically update when a row changes. Decide whether application code or a database trigger will maintain them before implementing broader CRUD operations.

### 6.4 Migration Safety

- Version schema changes using migration files.
- Test migrations against a clean, dedicated test database.
- Review down migrations before execution.
- Never run destructive migrations in production without an approved backup and recovery plan.

## 7. Architecture and Product Decisions

### 7.1 Multi-Tenant Architecture

- Start with a shared PostgreSQL database and shared schema.
- Include `school_id` in appropriate school-owned tables.
- Enforce tenant authorization on the server.
- Never rely on client-provided school IDs as proof of authorization.
- Consider PostgreSQL Row-Level Security later as defense in depth after application authorization tests exist.
- Separate databases per school are not part of the initial plan.

These are architecture decisions and requirements. Complete tenant-isolation enforcement remains pending.

### 7.2 Roles

Roles discussed for the product include:

- Platform Admin.
- School Admin.
- Teacher.
- Parent.
- Student.
- Driver and other staff roles as requirements are finalized.

The final role list and permission matrix must be documented before RBAC implementation.

### 7.3 Data Model Principles

- `schools` is the tenant root table.
- Login identities and school memberships need a deliberate design.
- Consider whether `users` and `school_memberships` should be separate before expanding the schema.
- Student identity should be separate from school-year enrollment and class placement.
- Parent-child relationships may be many-to-many and need explicit modeling.
- Subjects and teacher assignments should use normalized relational tables rather than arbitrary JSON.
- Explain foreign keys gradually and use a small ER diagram before introducing additional tables.

### 7.4 Excluded From Initial Scope

- Fee/payment gateway.
- Native parent mobile app.
- Live bus GPS tracking.
- Advanced AI analytics.

These may be considered later.

## 8. Security Requirements and Decisions

The following are planned requirements, not proof of completed implementation:

- Platform Admin and School Admin should use email/password and MFA at every login, without trusted-device exemptions.
- Planned access-token expiry: 15 minutes, subject to current configuration.
- Planned refresh-token expiry: 7 days, subject to implementation and review.
- Refresh-token rotation and revocation must be implemented and tested.
- Logout must revoke the relevant refresh session and enforce the final access-token revocation design.
- Password changes and account deactivation should invalidate sessions.
- Planned admin inactivity timeout: 30 minutes.
- Implement rate limits, temporary lockouts, generic login errors, and monitoring for failed login/MFA attempts.
- Strong password policy: minimum 12 characters, passphrase support, and compromised-password checks.
- Evaluate Argon2id against the current bcrypt implementation before finalizing the production password policy.
- Password reset links should be single-use and expire after 30 minutes.
- MFA backup codes should be single-use.
- Platform Admin must not have public self-registration.
- School Admin accounts must be created or invited by an authorized Platform Admin.
- Teacher accounts should be created or invited by authorized school staff.
- `pending_setup` and `suspended` schools must not have normal application access.
- Invitations should be single-use, expire after 24 hours, and store only token hashes.
- Audit sensitive security and permission changes.
- Enforce permissions and record scope on every protected API request.
- Teacher access should be limited to assigned classes and subjects.
- Parents should access only verified linked children.
- Driver access, when implemented, should be limited to assigned routes.
- Keep credentials and signing secrets out of source control.

The current middleware validates token identity for `/auth/me`; it does not fulfill the complete security requirements above.

## 9. Testing and Verification

### 9.1 Latest Reported Verification

After adding authentication middleware and `/auth/me`, the following commands were reported successful from `D:\school-management-system\backend`:

```cmd
go test -count=1 ./...
go vet ./...
```

The reported test run showed the API, configuration, handlers, middleware, and services packages passing; some packages had no test files.

`git diff --check` was also reported to complete without whitespace errors. Git emitted line-ending warnings indicating that LF may be replaced by CRLF on Windows.

These are historical results. Re-run the checks against the final working tree before committing.

### 9.2 Testing Gaps

- PostgreSQL repository integration tests.
- Automated migration tests against a clean database.
- Dedicated database connection/pool tests.
- Shutdown and timeout edge cases.
- Further JWT and HTTP authentication tests.
- RBAC and tenant-isolation tests.
- Refresh-token rotation, logout, and revocation tests.
- Frontend and end-to-end tests.
- Security and deployment verification.

Passing the current tests does not establish production readiness.

## 10. Documentation Checklist

Keep these documents synchronized with the implementation:

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

Do not document planned functionality as implemented.

## 11. Current Git and Delivery Workflow

Before committing:

1. Check the current branch and working tree.
2. Review tracked and untracked files.
3. Inspect changes and ensure no credentials are included.
4. Run `gofmt` on modified Go source files.
5. Run `go test -count=1 ./...`.
6. Run `go vet ./...`.
7. Run `git diff --check`.
8. Verify `backend/.env` is ignored and not tracked.
9. Review the staged diff before committing.
10. Commit with a meaningful message.
11. Push only after verifying the intended files are included.
12. Confirm the branch is synchronized with the remote and the working tree is clean.

Useful commands from the project root:

```cmd
cd /d D:\school-management-system
git status --short
git status -sb
git diff --check
git check-ignore backend/.env
git ls-files backend/.env
```

The `git ls-files backend/.env` command should print nothing if the file is not tracked. Never display or paste `.env` contents.

Useful backend verification commands:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 ./...
go vet ./...
```

Run `gofmt` only on the relevant Go source files. Review the resulting diff afterward to make sure formatting did not introduce unrelated changes.

## 12. Next Steps

1. Save this updated handoff file.
2. Review `README.md` and synchronize it with the implemented authentication middleware and `/auth/me` endpoint.
3. Review `docs/ARCHITECTURE.md`, `docs/DATABASE.md`, and `docs/TESTING.md` for consistency with the current implementation.
4. Confirm that `docs/API.md`, `docs/SECURITY.md`, `PROGRESS.md`, and `CHANGELOG.md` accurately distinguish implemented features from pending work.
5. Inspect the current Git status and review all modified and untracked files.
6. Format modified Go source files.
7. Run `go test -count=1 ./...`, `go vet ./...`, and `git diff --check`.
8. Verify that `.env` files and other secrets are not tracked or staged.
9. Review the staged diff and commit the verified authentication milestone and documentation.
10. Push the commit and verify remote synchronization and a clean working tree.
11. Design and implement the refresh-token lifecycle in small, tested steps.
12. Implement RBAC and tenant authorization before adding sensitive school-management endpoints.
13. Add PostgreSQL integration tests and production hardening before expanding the application.

Do not add more features until the current milestone has been reviewed, documented, tested, and committed.

## 13. New-Chat Continuation Prompt

Copy and paste the following into a new ChatGPT conversation:

> Continue my commercial School Management System project using `docs/HANDOFF.md` as the handoff reference in the GitHub repository `https://github.com/TheSumitNain/school-management-system`.
>
> My local project root is `D:\school-management-system`. I use Windows CMD and VS Code. The stack is React + TypeScript + Vite, Go + Gin, and PostgreSQL. The initial planning target is 25–30 schools with around 400–500 students per school.
>
> Explain in Hindi/Hinglish with a friendly tone. Work on one small task at a time. Give exact commands/code, explain why, and wait for my output before continuing. Do not assume a command succeeded without seeing the result.
>
> The backend includes PostgreSQL connectivity, health/readiness endpoints, migrations for `schools`, `users`, and `refresh_sessions`, password hashing, user/school repositories, an authentication service, JWT access-token generation/validation, `POST /auth/login`, authentication middleware, and a protected `GET /auth/me` endpoint. Tests exist for the login handler, token service, middleware, and API routes.
>
> Previous `go test -count=1 ./...`, `go vet ./...`, and `git diff --check` runs were reported successful after the middleware and `/auth/me` changes. Rerun them against the current working tree before committing.
>
> Refresh-token HTTP flows, logout, full profile lookup, access-token revocation, RBAC, complete tenant authorization, MFA, rate limiting, and PostgreSQL repository integration tests remain pending. Do not describe these as completed.
>
> The immediate priority is to review and synchronize the documentation, inspect Git changes, format and test the code, verify secret exclusion, and commit/push the reviewed milestone.
>
> Never ask me to paste real `.env` secrets. Do not claim the system is production-ready until outstanding security, testing, deployment, backup, and monitoring requirements have been addressed.
