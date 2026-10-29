# Changelog

All notable changes to the School Management System are documented in this file.

This project is under active development. Verification results describe the latest reported local checks and should be rerun whenever relevant code changes.

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
- `POST /auth/login` endpoint with a response DTO that excludes password hashes.
- JWT authentication middleware for protected routes.
- Bearer access-token validation and authenticated identity extraction in middleware.
- Protected `GET /auth/me` endpoint returning the authenticated user's ID and school ID.
- Opaque refresh-token generation using cryptographically secure randomness.
- SHA-256 hashing of refresh tokens before database persistence; raw refresh tokens are not stored.
- Refresh-token expiry configuration using `REFRESH_TOKEN_EXPIRY_DAYS`.
- `POST /auth/refresh` endpoint for refresh-token rotation.
- Refresh-token replay detection and revocation of active sessions belonging to a reused token family.
- `POST /auth/logout` endpoint for refresh-session revocation.
- Unit and handler tests for implemented authentication behavior.
- PostgreSQL refresh-session repository integration tests covering rotation, replay detection, token-family revocation, logout revocation, expired-token rejection and concurrent refresh behavior.
- Development-only seed command with environment safeguards.
- Documentation covering requirements, architecture, database design, API conventions, security, testing, progress and handoff.

### Changed

- Expanded configuration validation for JWT-related settings.
- Updated API documentation to describe `/auth/me`, refresh-token rotation and logout behavior.
- Updated security documentation to distinguish implemented authentication from remaining security gaps.
- Updated progress and handoff documentation to reflect the latest authentication implementation and verification.
- Documented the shared-database architecture for multiple schools.
- Clarified that authentication identifies the caller but does not yet provide complete role-based or school-level authorization.
- Added PostgreSQL integration-test coverage using a dedicated `school_management_test` database.
- Configured integration tests to run explicitly with `RUN_POSTGRES_INTEGRATION_TESTS=1` and a database safety check.

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

### Testing

- Backend Go test suite reportedly passes with `go test -count=1 ./...`.
- Backend static analysis reportedly passes with `go vet ./...`.
- PostgreSQL refresh-session repository integration tests reportedly pass against the dedicated `school_management_test` database.
- Integration tests cover refresh rotation and replay, token-family revocation, logout revocation, expired tokens and concurrent refresh behavior.
- Integration tests are opt-in and must not be run against the development database.
- `git diff --check` has been run during documentation updates; Windows LF-to-CRLF warnings were reported without whitespace errors.

### Pending

- Immediate access-token revocation; access JWTs remain valid until expiry after logout unless another enforcement mechanism is introduced.
- Full user-profile retrieval from `GET /auth/me`.
- Production school-onboarding, invitation and account-activation workflows.
- User roles, RBAC and permission enforcement.
- Complete school-level tenant authorization and automated cross-school isolation tests.
- Automated migration tests against a disposable test database.
- Additional authentication failure-path and database-error coverage where needed.
- Student, teacher, class, subject and attendance APIs.
- React frontend and dashboard implementation.
- Mandatory MFA for Platform Admin and School Admin.
- Password-reset workflows, login rate limiting, lockout enforcement and security-event monitoring.
- Production deployment, monitoring, backup, recovery, security testing and load testing.

## Latest Reported Git Checkpoint

- Latest repository checkpoint: `e88e94e` — `docs: update authentication progress and handoff`
- Previous code milestone: `a705f9a` — `test(auth): add PostgreSQL refresh session integration tests`
- Branch: `main`
- Remote: `origin`
- Last reported Git status: clean and synchronized with `origin/main`.
- `e88e94e` updates documentation only; it does not change application code or constitute a new test run.

## Changelog Guidelines

- Add entries when meaningful changes are completed.
- Describe verified implementation rather than planned functionality as completed.
- Keep unreleased work under `[Unreleased]`.
- Add a dated or versioned section when preparing a release.
- Re-run relevant tests after code changes and before recording verification results.
- Do not claim production readiness until required security, testing and operational controls are in place.
