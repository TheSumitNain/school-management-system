# School Management System — Project Handoff

> **Purpose:** This is the source-of-truth handoff for continuing the project in a new ChatGPT conversation. Update it after meaningful milestones and keep it committed to the GitHub repository.

## 1. Project Overview

- **Product:** Commercial multi-tenant School Management System.
- **Initial target:** 25–30 schools, approximately 400–500 students per school (roughly 10,000–15,000 students).
- **Frontend:** React + TypeScript + Vite.
- **Backend:** Go + Gin.
- **Database:** PostgreSQL.
- **Initial architecture:** Modular monolith with a shared database and shared schema.
- **Initial payment scope:** Fee/payment gateway excluded for now.
- **Future scope:** Parent mobile app, live bus tracking, advanced AI/analytics and additional roles/features.
- **Working style:** Explain in Hindi/Hinglish, use a friendly tone, work on one small task at a time, provide exact commands/code and wait for the user's result before proceeding.

These student and school counts are planning targets, not verified capacity limits.

## 2. Repository and Local Environment

- **Local project root:** `D:\school-management-system`
- **GitHub repository:** https://github.com/TheSumitNain/school-management-system
- **Branch:** `main`
- **Environment:** Windows CMD and VS Code.
- **Backend directory:** `D:\school-management-system\backend`
- **Default API port:** `8081`
- **Health endpoint:** `http://localhost:8081/health`
- **Readiness endpoint:** `http://localhost:8081/ready`
- **Database:** `school_management`

The current Git status must be checked before committing. Do not assume the latest commit, remote state or working-tree state without verifying it.

Never commit `.env` files, passwords, signing secrets or real student data. Never ask the user to paste secret values into chat.

## 3. Current Repository Structure

```text
D:\school-management-system
├── backend
│   ├── cmd
│   │   ├── api
│   │   │   ├── main.go
│   │   │   └── main_test.go
│   │   └── migrate
│   │       └── main.go
│   ├── internal
│   │   ├── config
│   │   │   └── config.go
│   │   ├── database
│   │   │   └── connection.go
│   │   ├── models
│   │   │   └── user.go
│   │   ├── repository
│   │   │   ├── school_repository.go
│   │   │   └── user_repository.go
│   │   └── services
│   │       ├── auth_service.go
│   │       ├── auth_service_test.go
│   │       ├── password.go
│   │       └── password_test.go
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
└── CHANGELOG.md (to be created)
```

This structure describes the expected files from the current implementation. Verify the actual filesystem before assuming every file is present.

## 4. Backend Foundation

### 4.1 Configuration

File: `backend/internal/config/config.go`

Previously implemented behavior:

- Loads `.env` using `godotenv.Load()` while allowing externally supplied environment variables.
- Defaults `APP_ENV` to `development`.
- Defaults `APP_PORT` to `8081`.
- Loads database host, port, username, database name and password.

The current configuration implementation should be reviewed before making changes. Verify required values, production-specific settings and error handling.

### 4.2 PostgreSQL Connection

File: `backend/internal/database/connection.go`

Implemented behavior includes:

- Validates required database connection settings.
- Builds a PostgreSQL connection URL safely.
- Uses `pgxpool.Pool`.
- Pings PostgreSQL during connection setup.
- Closes the pool if the initial ping fails.
- Uses `sslmode=disable` for the current local development setup.

**Production warning:** `sslmode=disable` must not be treated as production-ready. Configure and verify TLS/certificate validation for the actual production environment. Connection-pool limits and operational timeouts also need review.

### 4.3 API Server

File: `backend/cmd/api/main.go`

Implemented behavior:

- Loads configuration.
- Enables Gin release mode when `APP_ENV == "production"`.
- Connects to PostgreSQL with a 5-second context timeout.
- Configures Gin trusted proxies explicitly.
- Configures HTTP server timeouts:
  - `ReadHeaderTimeout`: 5 seconds.
  - `ReadTimeout`: 15 seconds.
  - `WriteTimeout`: 30 seconds.
  - `IdleTimeout`: 60 seconds.
- Starts the API on the configured port, defaulting to `8081`.
- Attempts graceful shutdown with a 10-second timeout.

#### `GET /health`

- Returns HTTP `200 OK` while the process can serve requests.
- Returns JSON indicating API status.
- Does not verify PostgreSQL availability.

#### `GET /ready`

- Pings the database using a request-scoped context with a 2-second timeout.
- Returns HTTP `200 OK` when the database ping succeeds.
- Returns HTTP `503 Service Unavailable` when the database ping fails.

#### Remaining hardening

- Explicitly handle graceful-shutdown timeout and force server closure if necessary.
- Review startup/shutdown error handling and deferred resource cleanup.
- Verify deployment-platform behavior before production release.

### 4.4 API Tests

File: `backend/cmd/api/main_test.go`

Tests cover:

- `/health` status, content type and response body.
- `/ready` when the database ping succeeds.
- `/ready` when the database ping fails.

These tests use Go's testing package, `net/http/httptest` and a fake database pinger.

### 4.5 Migration Runner

File: `backend/cmd/migrate/main.go`

Previously implemented behavior:

- Loads application configuration.
- Creates the PostgreSQL connection URL.
- Uses `golang-migrate`.
- Treats `migrate.ErrNoChange` as a successful outcome.
- Logs successful migration completion.

The migration directory is currently resolved relative to the working directory. Run the migration command from `D:\school-management-system\backend`:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

Making migration path resolution independent of the working directory and testing migrations against a dedicated database remain follow-up tasks.

## 5. Initial Authentication Foundation

The initial authentication foundation has been added, but complete HTTP/JWT authentication has not been implemented.

### 5.1 User Model

File: `backend/internal/models/user.go`

The `User` model contains:

- `ID`
- `SchoolID`
- `Username`
- `PasswordHash`
- `Status`
- `CreatedAt`
- `UpdatedAt`

The password hash uses `json:"-"` so it is excluded from JSON serialization.

### 5.2 Password Service

File: `backend/internal/services/password.go`

Implemented behavior:

- Uses bcrypt with cost 12.
- Provides password hashing and password verification.
- Rejects passwords shorter than 8 bytes or longer than 72 bytes.
- Rejects whitespace-only passwords.

Tests are in `backend/internal/services/password_test.go`.

**Security follow-up:** The current minimum is 8 bytes. Reconcile this with the stronger password policy in `docs/SECURITY.md` before exposing registration or password-change endpoints. Do not claim the stronger policy is implemented until the code and tests enforce it.

### 5.3 User Repository

File: `backend/internal/repository/user_repository.go`

Implemented operations:

- Find a user by school ID and case-insensitive username.
- Create a user with a password hash.
- Translate duplicate school/username conflicts into a repository error.

Repository-level PostgreSQL integration tests have not yet been implemented.

### 5.4 School Repository

File: `backend/internal/repository/school_repository.go`

Implements lookup of a school by ID and reports a not-found error when no matching school exists.

### 5.5 Authentication Service

File: `backend/internal/services/auth_service.go`

Implemented internal operations:

- Create a user for an existing active school.
- Authenticate a user using school ID, username and password.
- Reject inactive schools and disabled users.
- Return a generic invalid-credentials error for unknown users and incorrect passwords.

Service tests are in `backend/internal/services/auth_service_test.go`. They use fake repositories rather than a live PostgreSQL database.

**Important:** These are internal service operations, not public HTTP endpoints.

### 5.6 Not Yet Implemented

- JWT access-token issuance and validation.
- Refresh-token issuance and rotation.
- HTTP registration/login/refresh/logout endpoints.
- Authentication middleware.
- Session revocation and revoked-access-token enforcement in this project.
- RBAC and permission enforcement.
- School-level tenant authorization.
- PostgreSQL integration tests for authentication repositories.

The existence of a `refresh_sessions` database table does not mean refresh-token flows have been implemented.

## 6. Database and Migration State

Connection settings are stored locally in `backend/.env`. Credentials must remain secret.

### 6.1 Migration v1 — Schools

Files:

- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

The `schools` table contains:

- `id`: generated primary key.
- `name`: school name.
- `code`: unique school code.
- `status`: `pending_setup`, `active` or `suspended`.
- `created_at` and `updated_at`: timestamps.

The down migration drops the `schools` table and its data. Do not run it casually in production.

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
- A unique index on `(school_id, LOWER(username))` prevents case-insensitive duplicate usernames within one school.

#### `refresh_sessions`

- `id`: generated primary key.
- `user_id`: foreign key to `users(id)` with `ON DELETE CASCADE`.
- `token_hash`: unique hash of a refresh token.
- `expires_at`: session expiry timestamp.
- `revoked_at`: optional revocation timestamp.
- `created_at`: creation timestamp.
- Indexes support user lookup and expiration-time queries.

#### Local migration verification

The user reported that migration execution completed successfully in the local development database and that `schools`, `users` and `refresh_sessions` were verified to exist.

Recheck the migration version and `dirty` state before relying on this database for later work or deployment.

#### Migration safety

- Version schema changes with migration files.
- Test migrations against a clean, dedicated test database.
- Review down migrations before execution.
- Never run destructive migrations in production without an approved backup and recovery plan.

### 6.3 `updated_at` Behavior

The columns default to `NOW()` on insert. They do not automatically update when a row changes. Decide whether application code or a database trigger will maintain them before implementing broader CRUD operations.

## 7. Architecture and Product Decisions

### 7.1 Multi-Tenant Architecture

- Start with a shared PostgreSQL database and shared schema.
- Include `school_id` in appropriate school-owned tables.
- Enforce tenant authorization on the server.
- Never rely on client-provided school IDs as proof of authorization.
- Consider PostgreSQL Row-Level Security later as defense in depth after application authorization tests exist.
- Separate databases per school are not part of the initial plan.

### 7.2 Roles

Roles discussed for the product:

- Platform Admin
- School Admin
- Teacher
- Parent
- Student
- Driver and other staff roles as requirements are finalized

These roles and their permissions are product-planning decisions. Complete RBAC enforcement is not yet implemented.

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

The following are previously discussed product security requirements. They are not all implemented in the current code.

- Platform Admin and School Admin should use email/password and MFA at every login, without trusted-device exemptions.
- Planned access-token expiry: 15 minutes.
- Planned refresh-token expiry: 7 days, with rotation.
- Logout should revoke refresh tokens and block use of the associated access token.
- Password changes and account deactivation should invalidate sessions.
- Planned admin inactivity timeout: 30 minutes.
- Implement rate limits, temporary lockouts, generic login errors and monitoring for failed login/MFA attempts.
- Strong password policy: minimum 12 characters, passphrase support and compromised-password checks.
- Evaluate Argon2id for password hashing against the current bcrypt implementation before finalizing the production password policy.
- Password reset links should be single-use and expire after 30 minutes.
- MFA backup codes should be single-use.
- Platform Admin must not have public self-registration.
- School Admin accounts must be created or invited by an authorized Platform Admin.
- Teacher accounts should be created or invited by authorized school staff.
- `pending_setup` and `suspended` schools must not have normal application access.
- Invitations should be single-use, expire after 24 hours and store only token hashes.
- Audit sensitive security and permission changes.
- Enforce permissions and record scope on every protected API request.
- Teacher access should be limited to assigned classes/subjects.
- Parents should access only verified linked children.
- Driver access, when implemented, should be limited to assigned routes.
- Keep credentials and signing secrets out of source control.

Before implementing endpoints, reconcile these requirements with the actual authentication design and document any deliberate decisions.

## 9. Testing and Verification

### 9.1 Latest Reported Verification

The following commands completed successfully from `D:\school-management-system\backend`:

```cmd
go test -count=1 ./...
go vet ./...
```

The test suite included API, configuration and password/authentication service tests. The reported output showed all packages passing or having no test files.

`git diff --check` also completed without reporting whitespace errors in the latest reported check.

### 9.2 Testing Gaps

- PostgreSQL repository integration tests.
- Automated migration tests against a clean database.
- Dedicated tests for database connection/pool behavior.
- Shutdown and timeout edge cases.
- JWT and HTTP authentication tests once implemented.
- RBAC and tenant-isolation tests.
- Frontend and end-to-end tests.

Passing current tests does not establish production readiness.

## 10. Documentation Checklist

Keep these documents synchronized with implementation:

- `README.md`
- `PROGRESS.md`
- `docs/REQUIREMENTS.md`
- `docs/ARCHITECTURE.md`
- `docs/DATABASE.md`
- `docs/API.md`
- `docs/SECURITY.md`
- `docs/TESTING.md`
- `docs/HANDOFF.md`
- `CHANGELOG.md`

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

Useful backend verification commands:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 ./...
go vet ./...
```

The `git ls-files backend/.env` command should print nothing if the file is not tracked. Never display or paste `.env` contents.

## 12. Next Steps

1. Synchronize `docs/API.md`, `docs/ARCHITECTURE.md`, `docs/DATABASE.md`, `docs/SECURITY.md` and `docs/TESTING.md` with the actual code.
2. Create `CHANGELOG.md`.
3. Review and format the Auth foundation source files.
4. Re-run tests and vet.
5. Check Git status, staged changes and secret exclusion.
6. Commit and push the documentation plus Auth-foundation milestone.
7. Verify a clean working tree and remote synchronization.
8. Continue with authentication design and JWT flows in small steps.
9. Add RBAC and tenant authorization after authentication is verified.
10. Add PostgreSQL integration tests and production hardening before expanding the school-management APIs.

Do not add more features until the current milestone has been reviewed, documented, tested and committed.

## 13. New-Chat Continuation Prompt

Copy and paste the following into a new ChatGPT conversation:

> Continue my commercial School Management System project using `docs/HANDOFF.md` as the handoff reference in the GitHub repository `https://github.com/TheSumitNain/school-management-system`.
>
> My local project root is `D:\school-management-system`. I use Windows CMD and VS Code. The stack is React + TypeScript + Vite, Go + Gin, and PostgreSQL. The initial planning target is 25–30 schools with around 400–500 students per school.
>
> Explain in Hindi/Hinglish with a friendly tone. Work on one small task at a time. Give exact commands/code, explain why, and wait for my output before continuing. Do not assume a command succeeded without seeing the result.
>
> The backend has a PostgreSQL connection pool, health/readiness endpoints, HTTP timeouts, graceful shutdown handling and migrations for `schools`, `users` and `refresh_sessions`. The initial Auth foundation includes a user model, bcrypt password service, user/school repositories and an authentication service with unit tests.
>
> The latest reported `go test -count=1 ./...` and `go vet ./...` commands completed successfully. Migration v2 was applied and its relevant tables were verified in the local database.
>
> JWT access/refresh token flows, HTTP authentication endpoints, RBAC, tenant authorization and PostgreSQL repository integration tests are not yet implemented. Do not describe these as completed.
>
> The immediate priority is to finish reviewing and synchronizing documentation, including `CHANGELOG.md`, then run final checks, review Git changes, commit and push the verified milestone. Verify the actual Git status and files rather than assuming this handoff reflects the latest state.
>
> Never ask me to paste real `.env` secrets. Do not claim the system is production-ready until outstanding security, testing, deployment, backup and monitoring requirements have been addressed.
