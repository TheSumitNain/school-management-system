# Changelog

All notable changes to the School Management System are documented in this file.

This project is currently under active development. Entries describe the work completed or verified at the time of recording.

## [Unreleased]

### Added

- Initial Go backend using Gin.
- PostgreSQL connection-pool foundation.
- Database migration command and versioned SQL migration files.
- Initial `schools` database table.
- Health endpoint at `GET /health`.
- Readiness endpoint at `GET /ready`.
- HTTP server timeouts and graceful shutdown foundation.
- Initial user model and password service using bcrypt.
- Initial authentication service for user creation and credential verification.
- User and refresh-session database tables.
- Initial unit tests for selected authentication and password-service behavior.
- Project documentation covering requirements, architecture, database design, API, security, testing, and handoff.

### Changed

- Expanded project documentation to distinguish implemented functionality from planned work.
- Documented the initial shared-database architecture for multiple schools.

### Security

- Added initial password-hashing and credential-verification functionality.
- Documented pending work for JWT authentication, refresh-token lifecycle, RBAC, tenant isolation, and production security hardening.

### Pending

- HTTP registration and login endpoints.
- JWT access-token issuance and validation.
- Refresh-token rotation and logout flow.
- Authentication middleware and RBAC.
- Complete tenant-isolation enforcement.
- School business modules and frontend implementation.
- Production deployment configuration and load testing.

## Changelog Guidelines

- Add entries when meaningful changes are completed.
- Describe verified implementation rather than planned functionality as completed.
- Keep unreleased work under `[Unreleased]`.
- Add a dated or versioned section when preparing a release.
- Do not record a change as complete until it has been verified.
