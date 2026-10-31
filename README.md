# School Management System

A commercial, multi-tenant School Management System built incrementally with production-oriented architecture, security practices, and automated testing.

## Technology Stack

- **Frontend:** React + TypeScript + Vite (planned; not yet implemented)
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Migrations:** golang-migrate
- **Password hashing:** bcrypt
- **Authentication:** JWT access tokens and opaque refresh tokens with rotation and refresh-session revocation

## Initial Target

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students in total

These are planning targets, not verified capacity limits. Production capacity will require load testing and monitoring.

## Project Principles

- Enforce school-level data isolation on the backend.
- Use version-controlled database migrations.
- Never commit secrets, credentials, or real student data.
- Test implemented behavior before moving to the next feature.
- Document architecture, API behavior, database changes, and security decisions.
- Implement features incrementally and verify each milestone.
- Keep implemented functionality clearly separated from planned functionality.

## Current Status

**Stage:** Backend foundation and initial authentication, including JWT access tokens and refresh-token rotation/revocation. Phase 1 role, permission, staff-account, teacher-scope, and parent-access requirements are now approved and documented; RBAC implementation and complete tenant authorization remain pending.

### Implemented

- Go + Gin API server.
- Environment-based application configuration.
- PostgreSQL connection pool using pgxpool.
- Health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Versioned migrations for `schools`, `users`, and `refresh_sessions`, including refresh-token family metadata.
- User and school repository operations.
- bcrypt password hashing and verification, including password-length validation.
- Authentication service operations for user creation and credential verification.
- JWT access-token generation and validation, including issuer, expiry, and required-claim checks.
- `POST /auth/login` endpoint that issues access and refresh tokens.
- `POST /auth/refresh` endpoint for refresh-token rotation and replay detection.
- `POST /auth/logout` endpoint to revoke every active refresh session in the supplied token family.
- Generic unauthorized responses for invalid login credentials.
- Login response DTO that excludes password hashes.
- Authentication middleware that validates Bearer access tokens and extracts authenticated user and school IDs.
- Protected `GET /auth/me` endpoint returning the authenticated user's ID and school ID.
- Unit and handler tests for implemented authentication behavior.
- Authentication middleware and API route tests.
- Development-only seed command with environment safeguards.
- Documentation for requirements, architecture, database design, API behavior, security, testing, progress, and handoff.

### Authentication Endpoints

#### `POST /auth/login`

Accepts a school ID, username, and password. Successful authentication returns an access token, a refresh token, and basic user information. Invalid credentials return an unauthorized response.

#### `POST /auth/refresh`

Exchanges a refresh token for a new access token and replacement refresh token. Refresh tokens are opaque, cryptographically random values; only their SHA-256 hashes are persisted.

A reused revoked refresh token triggers revocation of active sessions in its token family. Refresh and logout operations serialize on a PostgreSQL transaction advisory lock scoped to the token family. Rotation additionally acquires a per-user transaction advisory lock to serialize rotations across that user's token families. Rotation performs a fresh row-locked lookup after acquiring the locks. Opt-in PostgreSQL repository integration tests cover refresh-session rotation, replay/family revocation, family-wide logout, idempotency, expiry, and concurrent refresh behavior. These tests do not cover every possible database failure or concurrency interleaving.

#### `POST /auth/logout`

Revokes all active refresh sessions in the token family associated with the supplied refresh token. Repeating logout is safe; an unknown token is a no-op. Already-issued access JWTs remain valid until expiry because access-token revocation enforcement has not been implemented.

#### `GET /auth/me`

Requires an `Authorization: Bearer <access-token>` header. The middleware validates the token, and the endpoint returns the authenticated user's ID and school ID.

This endpoint currently returns identity IDs only; complete user-profile retrieval remains pending.

See `docs/API.md` for request and response contracts.

## Database Migrations

The repository includes versioned migrations for:

- **Version 1:** Schools.
- **Version 2:** Users and refresh sessions.
- **Version 3:** Refresh-token family metadata.

According to the latest development verification, migration version 3 has been applied to the local development database, with the migration state reported as `dirty=false`. Verify the actual database state before running migrations or making further schema changes.

## Verification

After the refresh-token implementation, the following commands were reported successful from the backend directory:

```cmd
go test -count=1 ./...
go vet ./...
```

The reported test run covered the API, configuration, handlers, middleware, repositories, and services. The opt-in PostgreSQL repository integration tests must be enabled explicitly against the dedicated `school_management_test` database. Some packages have no test files.

Manual checks also reportedly verified login, refresh-token rotation, replay-triggered family revocation, and rejection of a refresh token after logout. The automated repository integration tests separately cover refresh-session lifecycle behavior. Independent end-to-end HTTP verification should still be performed.

These are previously reported results, not a guarantee that the current working tree has been reverified. Re-run the tests before committing.

## Not Yet Implemented

- Automated migration tests against a disposable test database.
- Complete user-profile retrieval from `GET /auth/me`.
- Access-token revocation enforcement.
- RBAC tables, role assignments, custom roles, permission overrides, and server-side permission enforcement (requirements approved; implementation pending).
- Global case-insensitive username uniqueness for staff accounts and the corresponding login-contract migration (current code remains school-ID scoped).
- Complete school/class/subject/parent-child resource authorization and cross-school/cross-class isolation tests.
- Production school onboarding, invitations, and account activation workflows.
- Mandatory MFA for Platform Admin and School Admin.
- Password-reset workflows, login rate limiting, lockout enforcement, and security-event monitoring.
- Student, teacher, class, subject, and attendance APIs.
- React frontend, dashboards, and parent-facing interfaces.
- Production deployment, monitoring, backup, and recovery procedures.
- Comprehensive security, end-to-end, and load testing.

**Important:** This project is under active development and is not production-ready. Authentication components have been implemented incrementally, but the complete authentication and authorization system has not yet been validated as production-ready.

## Documentation

- `PROGRESS.md` — implementation progress and next steps.
- `docs/REQUIREMENTS.md` — product requirements.
- `docs/ARCHITECTURE.md` — architecture and component boundaries.
- `docs/DATABASE.md` — schema and migration decisions.
- `docs/API.md` — API conventions and endpoint contracts.
- `docs/SECURITY.md` — security requirements and implementation status.
- `docs/TESTING.md` — testing strategy and verification.
- `docs/HANDOFF.md` — handoff instructions for continuing in a new chat.
- `docs/PHASE1-RBAC.md` — approved Phase 1 roles, permission catalogue, teacher scopes, staff account rules, and parent access model.
- `CHANGELOG.md` — milestone history.

## Development

Run backend commands from the backend directory:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 ./...
go vet ./...
```

Run database migrations from the same directory:

```cmd
go run ./cmd/migrate
```

Use the project's configured development workflow to start the API server.

Keep local environment files and credentials out of Git. Never share secrets in chat or commit them to the repository.

Review migration changes carefully before applying them to any production database. Test production migration procedures against a separate, backed-up database.
