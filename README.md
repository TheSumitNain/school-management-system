# School Management System

A commercial, multi-tenant School Management System built incrementally with production-oriented architecture, security and testing practices.

## Technology Stack

- **Frontend:** React + TypeScript + Vite (planned)
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Migrations:** golang-migrate
- **Password hashing:** bcrypt

## Initial Target

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students in total

These are planning targets, not verified capacity limits.

## Project Principles

- Enforce school-level data isolation on the backend.
- Use version-controlled database migrations.
- Never commit secrets, credentials or real student data.
- Test implemented behavior before moving to the next feature.
- Document architecture, API behavior, database changes and security decisions.
- Implement features incrementally and verify each milestone.

## Current Status

**Stage:** Backend foundation and initial authentication foundation.

### Implemented

- Go + Gin API server.
- Environment-based application configuration.
- PostgreSQL connection pool.
- Health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Initial `schools` database migration.
- Initial `users` and `refresh_sessions` migration.
- User and school repository operations.
- bcrypt password hashing and verification.
- Initial authentication service operations.
- Unit tests for password handling and authentication service behavior.

### Verification

The following commands completed successfully in the backend directory:

```cmd
go test -count=1 ./...
go vet ./...
```

The local database migrations through version 2 were applied and the relevant tables were verified.

### Not Yet Implemented

- JWT access and refresh token issuance and validation.
- HTTP login, registration and logout endpoints.
- Refresh-token rotation and session revocation workflows in this project.
- Role-based access control (RBAC).
- School-level tenant authorization and isolation tests.
- Student, teacher, class, subject and attendance APIs.
- React frontend and dashboards.
- Production deployment, monitoring, backup and recovery procedures.

Passing the current tests does not mean the application is production-ready.

## Documentation

- `PROGRESS.md` — implementation progress and next steps.
- `docs/REQUIREMENTS.md` — product requirements.
- `docs/ARCHITECTURE.md` — architecture and boundaries.
- `docs/DATABASE.md` — schema and migration decisions.
- `docs/API.md` — API conventions and implemented endpoints.
- `docs/SECURITY.md` — security requirements and implementation status.
- `docs/TESTING.md` — testing strategy and verification.
- `docs/HANDOFF.md` — source of truth for continuing work in a new chat.
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

Keep local environment files and credentials out of Git. Review migration changes carefully before applying them to any production database.