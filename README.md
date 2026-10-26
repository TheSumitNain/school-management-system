# School Management System

A multi-tenant School Management System designed for commercial use.

## Technology Stack

- Frontend: React + TypeScript + Vite
- Backend: Go + Gin
- Database: PostgreSQL

## Initial Target

- 25–30 schools
- Approximately 400–500 students per school
- Approximately 10,000–15,000 students across all schools

These are planning targets, not verified system capacity limits.

## Project Principles

- School-level data isolation must be enforced by the backend.
- Automated testing is required for implemented features.
- Database schema changes must use version-controlled migrations.
- Secrets and credentials must never be committed to Git.
- API behavior, architecture and security requirements must be documented.
- Features must be implemented incrementally and verified before moving forward.

## Project Status

**Current stage:** Backend foundation and API reliability improvements.

### Implemented

- Go + Gin API server with configurable application environment and port.
- PostgreSQL connection and health/readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Initial `schools` database migration.
- Automated tests for API health and database readiness behavior.

### In Progress

- Documentation review and synchronization.
- Production configuration and security hardening.
- Database and migration test coverage.

### Not Yet Implemented

- Authentication and role-based access control (RBAC).
- School-level tenant authorization and data isolation.
- Student, teacher, class, subject, and attendance APIs.
- Frontend dashboards and user workflows.

See `PROGRESS.md` for current progress and `docs/` for architecture, database, security, API, testing, and handoff documentation.