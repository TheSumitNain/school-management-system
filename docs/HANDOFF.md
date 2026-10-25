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
- **Private GitHub repository:** `https://github.com/TheSumitNain/school-management-system`
- **Branch:** `main`
- **Last confirmed GitHub commit:** `e264400` — `feat: add PostgreSQL connection and school migration`
- **Last confirmed repository state:** `git status` reported `nothing to commit, working tree clean`; local `main` was up to date with `origin/main`.
- **Windows environment:** CMD / VS Code.
- **Known tools:** Go `1.27.1 windows/amd64`, Git `2.41.0.windows.1`, Node `v22.19.0`, npm `10.9.0`, PostgreSQL client `18.6`, GitHub CLI `2.103.2`.
- **Backend directory:** `D:\school-management-system\backend`
- **Backend server port:** `8081` (port 8080 was occupied by Apache/httpd).
- **Health endpoint:** `http://localhost:8081/health`
- **Health response previously verified:** `{"message":"School Management System API is running","status":"success"}` (JSON key order may differ).

## 3. Current repository structure

```text
D:\school-management-system
├── backend
│   ├── cmd
│   │   ├── api
│   │   │   └── main.go
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
│   ├── .env                 # local only; ignored by Git; never commit
│   ├── .env.example         # tracked template; no real secret
│   ├── go.mod
│   └── go.sum
├── docs
│   ├── API.md
│   ├── ARCHITECTURE.md
│   ├── DATABASE.md
│   ├── REQUIREMENTS.md
│   ├── SECURITY.md
│   └── TESTING.md
├── .gitignore
├── README.md
└── PROGRESS.md
```

A new `docs/HANDOFF.md` should be added from this handoff document and committed. Documentation sync is still a pending task.

## 4. Current backend implementation

### `backend/internal/config/config.go`

Current behavior:
- Loads `.env` using `godotenv.Load()` (the error is ignored, so real environment variables can be used when `.env` is absent).
- Defaults `APP_ENV` to `development`.
- Defaults `APP_PORT` to `8081`.
- Loads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME`, `DB_PASSWORD`.

The `Config` struct has fields:
`AppEnv`, `AppPort`, `DBHost`, `DBPort`, `DBUser`, `DBName`, `DBPassword`.

### `backend/internal/database/connection.go`

Current behavior:
- Validates that DB host, port, user, name and password are present.
- Constructs a PostgreSQL URL using `url.UserPassword` and `net.JoinHostPort`.
- Sets `sslmode=disable` for local development.
- Creates a `pgxpool.Pool`, pings PostgreSQL, and closes the pool if ping fails.
- Returns a pool on success.

**Production follow-ups:** configure TLS/appropriate `sslmode`, stricter numeric port validation, timeouts/pool limits, and safe environment-specific configuration. `sslmode=disable` is only suitable for the local development setup, not production.

### `backend/cmd/api/main.go`

Current behavior:
- Loads config.
- Uses Gin release mode if `APP_ENV == "production"`.
- Connects to PostgreSQL with a 5-second context timeout.
- Logs successful PostgreSQL connection.
- Uses `gin.Default()` and exposes `GET /health`.
- Starts server at `":" + cfg.AppPort`.
- Gin currently warns that all proxies are trusted; configure trusted proxies explicitly before production.
- Health endpoint currently checks only that the process is responding; consider adding a separate readiness endpoint that checks DB health.

### `backend/cmd/migrate/main.go`

Current behavior:
- Loads the same config (which reads `.env`).
- Builds a PostgreSQL URL using `url.UserPassword`.
- Resolves the `migrations` directory using `filepath.Abs("migrations")`.
- Creates a golang-migrate instance with file source and PostgreSQL driver.
- Runs `m.Up()`, treating `migrate.ErrNoChange` as success.
- Logs `Database migrations completed successfully`.

**Important Windows fix:** the source URL was changed to:
```go
migrationsPath, err := filepath.Abs("migrations")
if err != nil {
    log.Fatal("Could not locate migrations directory")
}
migrationsPath = filepath.ToSlash(migrationsPath)
sourceURL := "file:" + migrationsPath
```
The earlier `file://...` / `url.URL` construction failed on Windows. The `file:` form above was verified to work from `D:\school-management-system\backend`.

**Usage:** run from `D:\school-management-system\backend`:
```cmd
go run ./cmd/migrate
```
The current code expects the working directory to be `backend`, because it resolves `migrations` relative to the current directory. If it is run from the repo root, it may not find the migration folder. A later improvement could resolve paths robustly independent of current working directory.

## 5. Database migration state

PostgreSQL connection settings are local in `backend/.env`:
- `DB_HOST=localhost`
- `DB_PORT=5432`
- `DB_USER=postgres`
- `DB_NAME=school_management`
- `DB_PASSWORD` is configured locally and must remain secret.

Never ask the user to paste the password into chat. Never put the real password into `.env.example`, Git, documentation, shell commands, or commits.

### Applied migration v1

Files:
- `backend/migrations/000001_create_schools_table.up.sql`
- `backend/migrations/000001_create_schools_table.down.sql`

The `up` migration creates:
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

**Warning:** down migration drops the table and all its data. Do not casually run a down migration in production.

### Verification performed

From `psql -h localhost -p 5432 -U postgres -d school_management`:
- `\dt` showed `public.schema_migrations` and `public.schools`.
- `\d schools` showed the expected columns, primary key, unique constraint on `code`, and status check constraint.
- `SELECT * FROM schema_migrations;` returned version `1`, dirty `false`.

`updated_at` currently defaults to `NOW()` on insert, but will **not automatically update** when a row changes. Before CRUD operations, choose and document whether this will be handled by a DB trigger or application code.

## 6. Migration tooling and Go module notes

- The CLI tool was added via Go tool directives.
- `go.mod` must contain only the CLI tool entry in the `tool` block to avoid ambiguity:
```go
tool (
    github.com/golang-migrate/migrate/v4/cmd/migrate
)
```
- Both `github.com/golang-migrate/migrate/v4` and `/cmd/migrate` were initially listed, causing `go tool migrate` to report ambiguity. The root module entry was removed from the tool block.
- `go tool migrate -version` returned `dev`, which is acceptable as the tool runs but does not expose a release version string.
- The CLI tool downloaded many transitive/optional drivers. This is expected for the CLI package; application runtime should only include dependencies actually imported by application packages.

## 7. Tests and verification completed

From `D:\school-management-system\backend`:
```cmd
gofmt -w cmd\migrate\main.go
go test ./...
```
Passed; packages reported `[no test files]`:
- `school-management-system/cmd/api`
- `school-management-system/cmd/migrate`
- `school-management-system/internal/config`
- `school-management-system/internal/database`

This means packages compile, but automated behavioral tests have not yet been written.

Other verified commands:
- `git diff --cached --check` produced no output.
- `git check-ignore backend/.env` returned `backend/.env`.
- `git status` after removing two accidental empty files named `git` and `main` returned:
  `On branch main`, `Your branch is up to date with 'origin/main'`, `nothing to commit, working tree clean`.
- Push of commit `e264400` to `origin/main` succeeded.

Windows may print harmless LF-to-CRLF warnings. Do not blindly change line endings unless needed.

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

1. Documentation sync is pending. Update `PROGRESS.md`, `docs/DATABASE.md`, `docs/ARCHITECTURE.md`, and add this `docs/HANDOFF.md`. Review `README.md`, `docs/API.md`, `docs/SECURITY.md`, `docs/TESTING.md` against actual current implementation.
2. Add meaningful unit/integration tests; current `go test ./...` only confirms packages compile.
3. Add database migration integration tests, including tenant isolation once tenant-owned tables exist.
4. Ensure migration runner handles working directory reliably.
5. Set explicit trusted proxies in Gin before production.
6. Production DB TLS; current `sslmode=disable` is local-only.
7. Decide automatic `updated_at` updates.
8. Validate environment variables at startup and fail with clear, non-secret error messages.
9. Consider graceful shutdown and HTTP server timeouts.
10. Add liveness/readiness distinction; readiness should verify DB availability.
11. Document migration creation/apply/rollback commands, including warning about destructive down migrations.
12. Revisit Go tool dependencies if `go.mod` became unusually large; do not remove dependencies without verifying build and tool setup.
13. Decide audit log retention and how to deliver invitations/password-reset links when email/mobile is optional.
14. Keep all actual secrets in local ignored `.env` or deployment secret manager, never in tracked files.

## 11. Exact next steps

Perform one step at a time, asking the user to run each command and share output before proceeding.

1. **Create/update documentation in the local repository**:
   - Add `docs/HANDOFF.md` using this content.
   - Update `PROGRESS.md` to state that database connection and migration v1 are complete and verified, current commit is `e264400`, Git tree was clean, and documentation sync is next.
   - Update `docs/DATABASE.md` with the exact `schools` schema and applied migration state.
   - Check architecture/security/API/testing docs against this handoff and update only what is stale.
2. Run `git diff --check`, `git status --short`, and `cd backend && go test ./...`.
3. Verify no `backend/.env` is staged or tracked (`git check-ignore backend/.env`; optionally `git ls-files backend/.env` should print nothing).
4. Commit and push documentation updates to `origin/main`.
5. Confirm clean `git status`.
6. Only then proceed to design the `users` and `school_memberships` tables, with a small relationship diagram and documented decisions before coding.

## 12. New-chat continuation prompt

Copy/paste the following into a new ChatGPT chat if needed:

> Continue my commercial School Management System project from `docs/HANDOFF.md` in the private repo `https://github.com/TheSumitNain/school-management-system`. Use Hindi/Hinglish, explain step by step, one small task at a time, and wait for my output before continuing. Local root is `D:\school-management-system`, branch `main`. Stack: React + TypeScript + Vite, Go + Gin, PostgreSQL. Initial target 25–30 schools × 400–500 students. Latest known commit is `e264400`; migration v1 is applied (schools table, schema_migrations version 1, dirty=false). The next task is documentation synchronization and committing it, then design `users` and `school_memberships` relationships before writing the next migration. Never ask me to share real `.env` secrets.
