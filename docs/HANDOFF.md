# School Management System — Project Handoff

> **Purpose:** This file is the source-of-truth handoff for continuing work in a new ChatGPT conversation. Update it at every meaningful milestone and keep it committed to the private GitHub repository.

## 1. Project at a glance

- **Product:** Commercial, multi-tenant School Management System.
- **Initial target:** 25–30 schools, approximately 400–500 students per school (roughly 10,000–15,000 students).
- **Frontend:** React + TypeScript + Vite.
- **Backend:** Go + Gin.
- **Database:** PostgreSQL.
- **Initial architecture:** Modular monolith; do not introduce microservices without a clear need.
- **Initial payment scope:** Fee/payment gateway excluded for now.
- **Later/future scope:** Native parent app, live bus tracking, advanced AI/analytics and additional roles/features.
- **User's working style:** Explain in Hindi/Hinglish, friendly tone, one small task at a time. Give exact commands/code, explain the reason, then wait for the user to run it and share the result before moving on. Avoid large multi-step instructions.

## 2. Repository and local environment

- **Local project root:** `D:\school-management-system`
- **GitHub repository:** `https://github.com/TheSumitNain/school-management-system`
- **Branch:** `main`
- **Last known commit:** `33905bc` — `docs: document initial database migration` (current GitHub HEAD must be verified before the next commit).
- **Current working tree:** Modified: `PROGRESS.md`, `README.md`, `backend/cmd/api/main.go`, `docs/API.md`, `docs/ARCHITECTURE.md`, `docs/DATABASE.md`, `docs/HANDOFF.md`, `docs/REQUIREMENTS.md`, `docs/SECURITY.md`, and `docs/TESTING.md`. Untracked: `backend/cmd/api/main_test.go`. Changes are not yet committed. Verify the latest test results and Git remote state before committing or pushing.
- **Environment:** Windows CMD / VS Code.
- **Known tools:** Go, Git, Node.js/npm, and PostgreSQL.
- **Backend directory:** `D:\school-management-system\backend`
- **Backend server port:** `8081` by default; port `8080` was occupied.
- **Health endpoint:** `http://localhost:8081/health`
- **Readiness endpoint:** `http://localhost:8081/ready`
- **Health and readiness:** Both endpoints have been manually verified. Automated tests cover their successful and database-failure behavior.

Never commit `.env` files, passwords, signing secrets, or real student data. Never ask the user to paste secret values into chat.

## 3. Current repository structure

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
│   │   └── database
│   │       └── connection.go
│   ├── migrations
│   │   ├── 000001_create_schools_table.up.sql
│   │   └── 000001_create_schools_table.down.sql
│   ├── .env                 # local only; ignored by Git
│   ├── .env.example         # tracked template; no real secrets
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
└── PROGRESS.md
```

`CHANGELOG.md` has not been created yet. Decide whether to add it when synchronizing the remaining documentation.


A new `docs/HANDOFF.md` should be added from this handoff document and committed. Documentation sync is still a pending task.

## 4. Current backend implementation

### `backend/internal/config/config.go`

Previously verified behavior:

- Loads `.env` using `godotenv.Load()` and allows environment variables to be supplied externally.
- Defaults `APP_ENV` to `development`.
- Defaults `APP_PORT` to `8081`.
- Loads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME`, and `DB_PASSWORD`.

The `Config` struct contains:
`AppEnv`, `AppPort`, `DBHost`, `DBPort`, `DBUser`, `DBName`, `DBPassword`.

**Follow-up:** Recheck the current source before changing configuration behavior. Validate required values and production-specific settings, and ensure errors do not expose secrets.

### `backend/internal/database/connection.go`

Previously verified behavior:

- Validates that the database host, port, user, name, and password are present.
- Constructs a PostgreSQL connection URL using `url.UserPassword` and `net.JoinHostPort`.
- Uses `sslmode=disable` for the local development setup.
- Creates a `pgxpool.Pool`, pings PostgreSQL, and closes the pool if the initial ping fails.
- Returns the pool after a successful connection.

**Production follow-ups:**

- Configure TLS and an appropriate `sslmode`.
- Validate the database port and other configuration values.
- Configure connection-pool limits and suitable timeouts.
- Verify production-specific configuration and error handling.

The current `sslmode=disable` setup must not be treated as production-ready.

### `backend/cmd/api/main.go`

Current implementation:

- Loads configuration using `config.Load()`.
- Enables Gin release mode when `APP_ENV == "production"`.
- Connects to PostgreSQL with a 5-second context timeout.
- Logs successful database connection.
- Creates a Gin router using `gin.Default()`.
- Calls `router.SetTrustedProxies(nil)` to avoid trusting forwarded client IP headers from arbitrary proxies.
- Configures the HTTP server with:
  - `ReadHeaderTimeout`: 5 seconds
  - `ReadTimeout`: 15 seconds
  - `WriteTimeout`: 30 seconds
  - `IdleTimeout`: 60 seconds
- Starts the API on the configured port, defaulting to `8081`.

#### `GET /health`

- Returns HTTP `200 OK` while the process can serve the request.
- Returns JSON with status `success` and message `School Management System API is running`.
- Does not check PostgreSQL availability.

#### `GET /ready`

- Creates a request-scoped context with a 2-second timeout.
- Calls the database ping method.
- Returns HTTP `200 OK` with status `success` when the database ping succeeds.
- Returns HTTP `503 Service Unavailable` with status `error` when the database ping fails.

#### Shutdown behavior

- Listens for `os.Interrupt` and `SIGTERM`.
- Attempts graceful HTTP server shutdown with a 10-second timeout.
- Logs an error if graceful shutdown fails.
- Closes the database pool through `defer` when `main` returns.

**Remaining hardening:**

- Handle shutdown timeout by explicitly forcing server closure if needed.
- Avoid `log.Fatal` paths that bypass deferred cleanup when cleanup is required.
- Review startup and shutdown behavior on supported deployment platforms.
- Verify all configuration and database settings before production deployment.

### `backend/cmd/api/main_test.go`

Automated API tests currently cover:

- `/health`: HTTP status, JSON content type, and expected response body.
- `/ready`: successful database ping and expected response.
- `/ready`: failed database ping, HTTP `503`, and expected response.

The tests use Go's `testing` package, `net/http/httptest`, and a fake database pinger. These test cases do not require a live PostgreSQL connection.

Run the tests from the `backend` directory:

```cmd
go test ./...
```

The latest reported run passed for all current Go packages. Configuration, database integration, migration, and shutdown behavior still need additional dedicated tests.

### `backend/cmd/migrate/main.go`

Previously verified behavior:

- Loads the application configuration.
- Builds a PostgreSQL connection URL.
- Resolves the migration directory using `filepath.Abs("migrations")`.
- Uses `golang-migrate` with a file source and PostgreSQL driver.
- Treats `migrate.ErrNoChange` as a successful outcome.
- Logs `Database migrations completed successfully`.

Run the migration command from `D:\school-management-system\backend`:

```cmd
go run ./cmd/migrate
```

The migration directory is resolved relative to the current working directory. Running the command from another directory may fail to locate the migrations.

**Follow-up:** Make path resolution independent of the current working directory and test migration behavior against a dedicated test database before relying on it in automated deployment.


## 5. Database migration state

PostgreSQL connection settings are stored locally in `backend/.env`. The database name is `school_management`; connection credentials must remain secret.

Never ask the user to paste the database password into chat. Never put real credentials into `.env.example`, Git, documentation, shell commands, or commits.

### Applied migration v1

Files:

- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

The `up` migration creates the following table:

```sql
CREATE TABLE schools (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_setup'
        CHECK (status IN ('pending_setup', 'active', 'suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

The `down` migration is:

```sql
DROP TABLE IF EXISTS schools;
```

**Warning:** The down migration drops the table and its data. Do not casually run it in production.

### Previously verified migration state

The following was verified using `psql` against `school_management`:

- `public.schools` and `public.schema_migrations` existed.
- The `schools` table had the expected columns, primary key, unique `code` constraint, and status check constraint.
- Migration version was `1`, with `dirty = false`.

Recheck the database before relying on this state for future migrations or deployment.

### `updated_at` behavior

The column defaults to `NOW()` on insert. It does not automatically update when a row changes. Before implementing CRUD operations, decide whether application code or a database trigger will maintain it.

### Migration safety

- Use version-controlled migration files for schema changes.
- Test migrations against a clean, dedicated test database.
- Review destructive down migrations before execution.
- Never run destructive migrations against production without an approved backup and recovery plan.

## 6. Migration tooling and Go module notes

Previously verified details:

- The project uses `golang-migrate`.
- The CLI tool is configured through Go tool directives in `go.mod`.
- The tool block should contain the CLI entry rather than conflicting entries for both the root migration module and the CLI package.
- An earlier duplicate tool configuration caused `go tool migrate` to report ambiguity; the tool block was corrected.
- The CLI reported `dev` for its version in the earlier setup.

The migration runner uses `filepath.Abs("migrations")` and converts the resulting path to slash separators before constructing a `file:` source URL. This Windows-specific approach was previously verified from the `backend` directory.

Run migrations from:

```cmd
cd /d D:\school-management-system\backend
go run ./cmd/migrate
```

The current runner expects the working directory to be `backend`. Making migration path resolution independent of the working directory remains a follow-up task.

## 7. Tests and verification completed

### Automated API tests

File: `backend/cmd/api/main_test.go`

Current test cases cover:

- `GET /health`: HTTP status, JSON content type, and expected response body.
- `GET /ready`: successful database ping and expected response.
- `GET /ready`: failed database ping, HTTP `503 Service Unavailable`, and expected response body.

The tests use Go's standard `testing` package, `net/http/httptest`, and a fake database pinger.

### Latest reported test result

Command run from `D:\school-management-system\backend`:

```cmd
go test ./...
```

Reported output:

```text
ok      school-management-system/cmd/api        (cached)
?       school-management-system/cmd/migrate    [no test files]
?       school-management-system/internal/config        [no test files]
?       school-management-system/internal/database      [no test files]
```

This confirms that all current Go packages compile and the existing API tests pass. The API test result was served from Go's test cache, so this command did not necessarily rerun the tests.

To force test execution:

```cmd
go test -count=1 ./...
```

### Remaining verification

- Configuration validation tests.
- Database connection and pool integration tests.
- Migration tests against a dedicated test database.
- HTTP shutdown and timeout tests.
- Authentication, RBAC, and tenant-isolation tests when those features are implemented.
- Frontend and end-to-end tests when the frontend is available.

Passing the current test suite does not mean the application is production-ready.


## 8. Architecture and product decisions already discussed

### Tenant isolation
- Initial scale: 25–30 schools × 400–500 students.
- Start with a shared PostgreSQL database and shared schema, with `school_id` in appropriate school-owned tables and strict server-side tenant isolation.
- Do not rely on client-provided school IDs for authorization.
- Evaluate PostgreSQL Row-Level Security later as defense in depth, after clear authorization tests exist.
- Separate database per school is not the initial plan.

### Initial roles
- Platform Admin
- School Admin
- Teacher
- Parent
- Student, Driver and other staff roles can be added when requirements are finalized.

### Data model principles
- `schools` is the tenant root table.
- A `users` table represents login identities.
- A `school_memberships` relationship is a strong candidate for associating users with schools and roles. This supports the possibility of a person belonging to multiple schools without duplicating identity.
- Do not finalize `users`/`school_memberships` schema until roles, invitation flow, authentication identifiers, and account lifecycle have been checked against requirements.
- Student identity should be separated from school-year enrollment/class placement.
- Parent-child links may be many-to-many and should be verified.
- Subjects and teacher assignments should be normalized relational tables rather than storing multiple subjects in arbitrary JSON.
- User finds many foreign keys/tables hard to follow; explain relationships gradually and show a small ER diagram before writing each migration.

### Scope not included in initial version
- Fee/payment gateway.
- Native parent mobile app.
- Live bus GPS tracking.
- Advanced AI analytics.
These can be planned later.

## 9. Security decisions already finalized in previous planning

These requirements should be checked against `docs/SECURITY.md`; if the document does not contain them, update it.

- Platform Admin and School Admin authenticate using email/password and must perform MFA at every login; no trusted-device exemption.
- Access token expiry: 15 minutes.
- Refresh token expiry: 7 days, with rotation.
- Logout revokes refresh token and blocks use of the associated access token.
- Password changes and account deactivation invalidate sessions.
- Admin inactivity timeout: 30 minutes.
- Five consecutive failed logins cause a 15-minute temporary lock; account/IP rate limits; generic login errors; reset failed-attempt count on success; rate-limit and log MFA attempts.
- Password minimum length 12; allow passphrases; block common/compromised passwords; no routine forced expiry; hash with Argon2id; require changes if compromised.
- Admin password reset uses single-use link expiring in 30 minutes; revoke sessions; require MFA next login. If email is lost, use verified admin-assisted recovery.
- MFA backup codes are single-use. Recovery revokes sessions and is audited. Platform Admin recovery needs additional verification.
- No public Platform Admin registration. Platform Admin is created through secure setup; authorized Platform Admin creates others and invites School Admins; School Admin cannot self-elevate.
- Teacher accounts are created/invited by School Admin/authorized staff; teacher sets their password.
- School statuses: `pending_setup`, `active`, `suspended`. Pending/suspended schools cannot use normal access; preserve their data; only Platform Admin can reactivate.
- Invitations are single-use, expire in 24 hours, new invite invalidates old, and only a token hash is stored.
- Audit logs for permission changes, MFA reset, password resets, security events and suspension. Audit retention period still pending.
- Every protected API request enforces role and record scope; never trust client IDs. Permission changes must take effect on the next request, including existing sessions.
- Teacher access is limited to assigned classes/subjects; parents can access only verified linked children. Driver access later is limited to assigned routes.
- Teacher reset is initiated by School Admin, teacher sets own password. Delivery method is unresolved if email/mobile are optional.
- Passwords/secrets must not be committed.

## 10. Current known issues / follow-ups

1. Synchronize the remaining documentation with the current implementation, especially `README.md`, `docs/API.md`, `docs/ARCHITECTURE.md`, `docs/DATABASE.md`, and `docs/SECURITY.md`.
2. Review the complete Git diff and verify the current branch, remote status, and latest commit before committing documentation changes.
3. Run `git diff --check` and `go test -count=1 ./...` before committing.
4. Verify that `backend/.env` is ignored and not tracked. Never expose real credentials.
5. Add configuration validation and tests for invalid or missing settings.
6. Add PostgreSQL connection, pool, and migration integration tests using a dedicated test database.
7. Make migration path resolution independent of the current working directory.
8. Configure production database TLS; `sslmode=disable` is local-development-only.
9. Configure appropriate database connection-pool limits and operational timeouts.
10. Improve shutdown error handling, including forced closure if graceful shutdown times out.
11. Decide how `updated_at` values will be maintained on updates.
12. Finalize the database model and migration strategy before implementing school-owned CRUD APIs.
13. Implement authentication, followed by RBAC and school-level tenant authorization.
14. Decide audit-log retention and how invitations/password-reset links will be delivered if email or mobile is optional.
15. Add CI, deployment configuration, monitoring, backups, recovery procedures, and security/load testing before production release.

## 11. Exact next steps

Continue one small task at a time. Give exact commands or code, explain the reason, and wait for the user to share the result before proceeding.

1. Review the remaining documentation against the actual code. Start with `README.md`, `docs/API.md`, `docs/ARCHITECTURE.md`, `docs/DATABASE.md`, and `docs/SECURITY.md`. Update only stale or inaccurate content.
2. Review `git status --short` and `git diff` to understand every local change, including the API implementation, tests, and documentation.
3. Run the following checks from the appropriate directories:

   ```cmd
   git diff --check
   cd /d D:\school-management-system\backend
   go test -count=1 ./...
   ```

4. Confirm `backend/.env` is ignored and not tracked:

   ```cmd
   cd /d D:\school-management-system
   git check-ignore backend/.env
   git ls-files backend/.env
   ```

   The first command should report the ignored path; the second should print nothing if the file is not tracked.

5. Review and commit the verified changes with a meaningful commit message, then push to `origin/main` only after confirming the intended files are included.
6. Confirm the working tree is clean and the local branch is synchronized with the remote.
7. Before adding more migrations, finalize the `users` and `school_memberships` design. Show a small relationship diagram, explain each foreign key gradually, and document the decisions before writing migration files.
8. Continue with authentication, RBAC, tenant isolation, core APIs, frontend integration, and production hardening in manageable stages.

## 12. New-chat continuation prompt

Copy and paste this prompt into a new ChatGPT conversation if needed:

> Continue my commercial School Management System project from `docs/HANDOFF.md` in the GitHub repository `https://github.com/TheSumitNain/school-management-system`.
>
> My local project root is `D:\school-management-system`, using Windows CMD and VS Code. The stack is React + TypeScript + Vite, Go + Gin, and PostgreSQL. The initial target is 25–30 schools with approximately 400–500 students per school.
>
> Explain in Hindi/Hinglish, use a friendly tone, and work on one small task at a time. Give exact commands or code, explain why, and wait for my output before continuing. Do not assume that a command succeeded without seeing its result.
>
> The backend has a PostgreSQL connection pool, a `schools` table migration (version 1 was previously verified with `dirty = false`), `GET /health`, and `GET /ready`. The API includes HTTP server timeouts and graceful shutdown handling. Automated API tests exist in `backend/cmd/api/main_test.go` for health and readiness behavior.
>
> The latest reported test command was `go test ./...`; all packages compiled and the existing API tests passed, with the API result served from the Go test cache. Use `go test -count=1 ./...` when a fresh test run is required.
>
> Documentation updates are in progress. Verify the current Git status, commit, and remote state instead of assuming they match this document. The latest previously known commit was `33905bc` (`docs: document initial database migration`), but the current GitHub HEAD has not been confirmed.
>
> The next step is to review the remaining documentation and inspect the actual Git diff before committing. After documentation is synchronized, design the `users` and `school_memberships` tables with a small relationship diagram before creating the next migration.
>
> Never ask me to paste real `.env` secrets. Do not claim that the system is production-ready until the outstanding security, testing, configuration, deployment, backup, and monitoring requirements have been addressed.