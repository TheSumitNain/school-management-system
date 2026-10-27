# Changelog

All notable changes to the School Management System are documented in this file.

This project is under active development. Entries describe implementation completed so far; final verification of the current working tree is still required before committing or releasing.

## [Unreleased]

### Added

- Initial Go backend using Gin.
- Environment-based application configuration with validation.
- PostgreSQL connection pool using pgxpool.
- Database migration command and versioned SQL migration files.
- Migration v1 for the `schools` table.
- Migration v2 for `users` and `refresh_sessions`.
- Health endpoint at `GET /health`.
- Database-readiness endpoint at `GET /ready`.
- HTTP server timeouts and graceful shutdown handling.
- Explicit Gin trusted-proxy configuration.
- User and school repository operations.
- Password hashing and verification using bcrypt.
- Password-length validation and password service tests.
- Authentication service for user creation and credential verification.
- JWT access-token generation and validation.
- JWT issuer, expiry and required-claim validation.
- Initial HTTP login handler and `POST /auth/login` endpoint.
- Login response DTO that excludes password hashes.
- Tests for configuration, authentication services, token services and login handlers.
- Development-only seed command with environment safeguards.
- Documentation covering requirements, architecture, database design, API conventions, security, testing, progress and handoff.

### Changed

- Expanded configuration validation for JWT-related settings.
- Updated documentation to distinguish implemented functionality from planned work.
- Documented the initial shared-database architecture for multiple schools.
- Updated authentication documentation to reflect the access-token service and login endpoint.

### Security

- Added bcrypt-based password handling.
- Added initial JWT access-token generation and validation.
- Added generic unauthorized responses for invalid login credentials.
- Kept password hashes out of login responses.
- Added safeguards to the development seed command.
- Documented remaining work for refresh-token lifecycle management, protected routes, RBAC, tenant isolation and production security hardening.

### Pending

- Refresh-token issuance, rotation, replay protection and complete session lifecycle.
- HTTP logout endpoint and complete session-revocation workflow.
- Authentication middleware and protected `GET /auth/me` endpoint.
- Public registration and production school-onboarding workflows.
- User roles, RBAC and permission enforcement.
- School-level tenant authorization and automated isolation tests.
- PostgreSQL repository integration tests and migration tests.
- Student, teacher, class, subject and attendance APIs.
- React frontend and dashboard implementation.
- Production deployment, monitoring, backup, recovery, security testing and load testing.

## Verification Notes

- `go test -count=1 ./...` and `go vet ./...` passed in an earlier verification run.
- Local migrations through version 2 were previously applied and relevant tables verified.
- These checks must be rerun against the current working tree before the milestone is committed.
- The project is not yet production-ready.

## Changelog Guidelines

- Add entries when meaningful changes are completed.
- Describe verified implementation rather than planned functionality as completed.
- Keep unreleased work under `[Unreleased]`.
- Add a dated or versioned section when preparing a release.
- Do not record a change as complete until its implementation has been verified.
- Do not claim production readiness until required security, testing and operational controls are in place.
