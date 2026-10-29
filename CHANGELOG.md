# Changelog

All notable changes to the School Management System are documented in this file.

This project is under active development. Verification notes describe previously reported results and must be rechecked against the final working tree before committing or releasing.

## [Unreleased]

### Added

- Initial Go backend using Gin.
- Environment-based application configuration with validation.
- PostgreSQL connection pool using pgxpool.
- Database migration command and versioned SQL migration files.
- Migration v1 for the `schools` table.
- Migration v2 for `users` and `refresh_sessions`.
- Migration v3 adding refresh-token family identifiers.
- Health endpoint at `GET /health`.
- Database-readiness endpoint at `GET /ready`.
- HTTP server timeouts and graceful shutdown handling.
- Explicit Gin trusted-proxy configuration.
- User and school repository operations.
- Password hashing and verification using bcrypt.
- Password-length validation and password service tests.
- Authentication service for user creation and credential verification.
- JWT access-token generation and validation, including issuer, expiry and required-claim checks.
- `POST /auth/login` endpoint and a response DTO that excludes password hashes.
- JWT authentication middleware for protected routes.
- Bearer access-token validation and authenticated identity extraction in middleware.
- Protected `GET /auth/me` endpoint returning the authenticated user's ID and school ID.
- Opaque refresh-token generation, with SHA-256 hashes persisted instead of raw tokens.
- Refresh-token expiry configuration using `REFRESH_TOKEN_EXPIRY_DAYS`.
- `POST /auth/refresh` endpoint for refresh-token rotation.
- Replay detection that revokes active sessions in a reused token family.
- `POST /auth/logout` endpoint for refresh-session revocation.
- Unit, handler, middleware and API route tests for implemented authentication behavior.
- Development-only seed command with environment safeguards.
- Documentation covering requirements, architecture, database design, API conventions, security, testing, progress and handoff.

### Changed

- Expanded configuration validation for JWT-related settings.
- Updated API documentation to describe `/auth/me`, refresh-token rotation and logout behavior.
- Updated security documentation to distinguish implemented authentication behavior from remaining security gaps.
- Updated progress and handoff documentation to reflect the refresh-token implementation and latest reported verification.
- Documented the shared-database architecture for multiple schools.
- Updated authentication documentation to distinguish identity validation from full authorization.

### Security

- Added bcrypt-based password handling.
- Added JWT access-token generation and validation.
- Added generic unauthorized responses for invalid login credentials.
- Kept password hashes out of login responses.
- Added authentication middleware that validates Bearer access tokens on the protected identity endpoint.
- Rejects missing or malformed authorization headers, invalid tokens and invalid identity claims.
- Stores only refresh-token hashes in the database.
- Rotates refresh tokens and revokes active sessions in a token family when a revoked token is replayed.
- Added safeguards to the development seed command.
- Documented remaining work for RBAC, tenant isolation, MFA, rate limiting and production security hardening.

### Pending

- PostgreSQL repository integration tests and concurrent refresh-token rotation/replay tests.
- Independent logout verification using a fresh authentication session.
- Immediate access-token revocation; access JWTs remain valid until expiry after logout.
- Full user-profile retrieval from `GET /auth/me`.
- Production school-onboarding, invitation and account-activation workflows.
- User roles, RBAC and permission enforcement.
- Complete school-level tenant authorization and automated cross-school isolation tests.
- Automated migration tests against a disposable test database.
- Student, teacher, class, subject and attendance APIs.
- React frontend and dashboard implementation.
- Mandatory MFA for Platform Admin and School Admin.
- Password-reset workflows, login rate limiting, lockout enforcement and security-event monitoring.
- Production deployment, monitoring, backup, recovery, security testing and load testing.

## Verification Notes

- `go test -count=1 ./...` was reported successful after the refresh-token implementation.
- `go vet ./...` was reported successful after the refresh-token implementation.
- Manual checks reportedly covered login, refresh-token rotation, replay-triggered family revocation and refresh rejection after logout.
- Migration v3 was reported applied to the local development database, with migration version 3 and `dirty=false`.
- `git diff --check` has been run during the documentation updates; only Windows LF-to-CRLF warnings were reported, with no whitespace errors.
- These are previously reported results. Re-run tests and inspect the actual migration state before committing.
- PostgreSQL integration tests, concurrent replay tests and a clean-session logout test remain necessary.
- The project is not production-ready.

## Changelog Guidelines

- Add entries when meaningful changes are completed.
- Describe verified implementation rather than planned functionality as completed.
- Keep unreleased work under `[Unreleased]`.
- Add a dated or versioned section when preparing a release.
- Do not record a change as complete until its implementation has been verified.
- Do not claim production readiness until required security, testing and operational controls are in place.