# School Management System

A commercial, multi-tenant School Management System built incrementally with production-oriented architecture, security practices and automated testing.

## Technology Stack

- **Frontend:** React + TypeScript + Vite (planned; not yet implemented)
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Migrations:** golang-migrate
- **Password hashing:** bcrypt
- **Authentication:** JWT access tokens with initial authentication middleware

## Initial Target

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students in total

These are planning targets, not verified capacity limits. Production capacity will require load testing and monitoring.

## Project Principles

- Enforce school-level data isolation on the backend.
- Use version-controlled database migrations.
- Never commit secrets, credentials or real student data.
- Test implemented behavior before moving to the next feature.
- Document architecture, API behavior, database changes and security decisions.
- Implement features incrementally and verify each milestone.
- Keep implemented functionality clearly separated from planned functionality.

## Current Status

**Stage:** Backend foundation and initial JWT authentication.

### Implemented

- Go + Gin API server.
- Environment-based application configuration.
- PostgreSQL connection pool using pgxpool.
- Health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Versioned database migrations for `schools`, `users` and `refresh_sessions`.
- User and school repository operations.
- bcrypt password hashing and verification, including password-length validation.
- Authentication service operations for user creation and credential verification.
- JWT access-token generation and validation.
- JWT issuer, expiry and required-claim validation.
- `POST /auth/login` endpoint.
- Generic unauthorized responses for invalid login credentials.
- Login response DTO that excludes password hashes.
- Authentication middleware that validates Bearer access tokens and extracts authenticated user and school IDs.
- Protected `GET /auth/me` endpoint returning the authenticated user's ID and school ID.
- Unit tests for configuration, password handling, authentication services and token services.
- Login-handler tests, authentication middleware tests and API route tests.
- Development-only seed command with environment safeguards.
- Documentation for requirements, architecture, database design, API behavior, security, testing, progress and handoff.

### Implemented Authentication Endpoints

#### `POST /auth/login`

Accepts a school ID, username and password. On successful authentication, returns an access token and basic user information. Invalid credentials return an unauthorized response.

#### `GET /auth/me`

Requires an `Authorization: Bearer <access-token>` header. The authentication middleware validates the token and the endpoint returns the authenticated user's ID and school ID.

This endpoint currently returns identity IDs only; fetching the complete user profile remains pending.

See `docs/API.md` for the documented request and response contracts.

### Verification

The following commands were previously reported successful from the backend directory after the authentication middleware and `/auth/me` changes:

```cmd
go test -count=1 ./...
go vet ./...
```

The reported test run included API, configuration, handler, middleware and service packages. Some packages had no test files.

`git diff --check` was also previously reported successful.

The local database migrations through version 2 were previously applied, and the relevant tables were verified.

These are historical results. Re-run the checks against the current working tree before committing. A previous successful test run does not guarantee that subsequent changes pass.

### Not Yet Implemented

- Refresh-token issuance, rotation, replay protection and complete session lifecycle endpoints.
- Logout endpoint and complete HTTP-level session revocation workflow.
- Full user-profile retrieval from `GET /auth/me`.
- Access-token revocation enforcement.
- User roles and role-based access control (RBAC).
- Complete school-level tenant authorization and cross-school isolation tests.
- Public registration and production school-onboarding workflows.
- MFA, password-reset and invitation workflows.
- Login rate limiting, lockout enforcement and security-event monitoring.
- Student, teacher, class, subject and attendance APIs.
- React frontend, dashboards and parent-facing interfaces.
- PostgreSQL repository integration tests and automated migration tests.
- Production deployment, monitoring, backup and recovery procedures.
- Comprehensive security, end-to-end and load testing.

**Important:** This project is under active development and is not production-ready. Authentication components have been implemented incrementally; they have not yet been validated as a complete production authentication and authorization system.

## Documentation

- `PROGRESS.md` — implementation progress and next steps.
- `docs/REQUIREMENTS.md` — product requirements.
- `docs/ARCHITECTURE.md` — architecture and component boundaries.
- `docs/DATABASE.md` — schema and migration decisions.
- `docs/API.md` — API conventions and implemented endpoints.
- `docs/SECURITY.md` — security requirements and implementation status.
- `docs/TESTING.md` — testing strategy and verification.
- `docs/HANDOFF.md` — handoff instructions and source of truth for continuing work in a new chat.
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

Run the API server using the project's configured development workflow.

Keep local environment files and credentials out of Git. Never share secrets in chat or commit them to the repository.

Review migration changes carefully before applying them to any production database. Use a separate, backed-up database for production migration testing.
