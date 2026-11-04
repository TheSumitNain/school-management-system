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

## Approved Product Delivery Direction (2026-11-03)

**Phase 1 is the complete web-based school management product.** It includes:
- Platform Admin dashboard for platform operations and school onboarding.
- School Admin dashboard for complete school administration.
- Role-based dashboards and workflows for school staff, including teachers and future school-scoped operational roles as their modules are delivered.

**Phase 2 follows after the web product is complete:** mobile applications/interfaces for Parents, Drivers and other approved user groups. Parent access is intended to be read-only initially; its authentication and OTP provider will be decided when Phase 2 is planned. Parent-specific implementation is paused in the meantime. Existing Parent/Guardian schema and OTP foundation must be preserved, not deleted or expanded during Phase 1 without a separately approved requirement.

The database must be designed incrementally but with future compatibility in mind: stable student identity separate from academic enrollment, school-scoped ownership, extensible roles/permissions, guardian relationships, transport-ready concepts, audit history and API contracts reusable by web and mobile clients. Future-proofing does not authorize speculative tables or premature migrations. Every schema change requires source review, additive migration design, tests on the dedicated test database and explicit approval before applying migrations.

## Current Status

**Stage:** Principal-aware authentication, RBAC, academic foundation, student admission/profile APIs, and school-scoped Guardian Management backend. The React frontend is not yet implemented, and most school business modules remain pending. Guardian permissions use explicit Platform Admin delegation; built-in School Admin and Teacher role bundles remain unchanged. Active branch: `fix/v4-auth-security-hardening`. This is not production-ready.

### Implemented

- Go + Gin API server.
- Environment-based application configuration.
- PostgreSQL connection pool using pgxpool.
- Health and database-readiness endpoints.
- HTTP server timeouts and graceful shutdown handling.
- Versioned PostgreSQL migrations for shared principals, RBAC, account subtypes, MFA/OTP foundations, audit history, student/guardian records and new-school system-role provisioning.
- User and school repository operations.
- bcrypt password hashing and verification, including password-length validation.
- Authentication service operations for user creation and credential verification.
- Principal-aware JWT access-token generation and validation; tokens carry principal identity/type and token version, while permissions are loaded live from PostgreSQL.
- Staff login by globally unique username and a separate Platform Admin email login.
- Principal-aware refresh sessions with rotation, replay/family revocation, and preserved transaction-lock concurrency protections.
- Database-backed principal status/token-version validation and effective system-role permission loading on protected requests.
- RequirePermission middleware and authenticated /auth/me route.
- Mandatory TOTP MFA enrollment/verification and recovery-code flows for Platform Admin and School Admin.
- Failed-login lockout and login IP rate-limiting components.
- Transactional Platform Admin school onboarding, with V13 automatic School Admin/Teacher system-role provisioning for new schools.
- One-time first Platform Admin CLI bootstrap source in V14; migration and repository integration verification are still pending.
- `POST /auth/login` endpoint that issues access and refresh tokens.
- `POST /auth/refresh` endpoint for refresh-token rotation and replay detection.
- `POST /auth/logout` endpoint to revoke every active refresh session in the supplied token family.
- Generic unauthorized responses for invalid login credentials.
- Login response DTO that excludes password hashes.
- Authentication middleware that validates Bearer access tokens, reloads principal status/token version and effective permissions, and places canonical principal context in the request.
- Protected `GET /auth/me` endpoint returning principal ID/type and staff-only school/username context.
- Unit and handler tests for implemented authentication behavior.
- Authentication middleware and API route tests.
- Development-only seed command with environment safeguards.
- School-scoped Guardian Management APIs for guardian contacts and student relationships, including audited creation/update, primary/emergency contacts, history-preserving revocation, dedicated RBAC permissions, and V19 controlled delegation.
- Documentation for requirements, architecture, database design, API behavior, security, testing, progress, and handoff.

### Authentication Endpoints

#### `POST /auth/login`

Accepts a globally unique staff username and password (no school ID). Privileged School Admin login requires TOTP MFA; successful authentication after MFA returns principal-based access and refresh tokens. Invalid credentials return a generic unauthorized response.

#### `POST /auth/refresh`

Exchanges a refresh token for a new access token and replacement refresh token. Refresh tokens are opaque, cryptographically random values; only their SHA-256 hashes are persisted.

A reused revoked refresh token triggers revocation of active sessions in its token family. Refresh and logout operations serialize on a PostgreSQL transaction advisory lock scoped to the token family. Rotation additionally acquires a per-user transaction advisory lock to serialize rotations across that user's token families. Rotation performs a fresh row-locked lookup after acquiring the locks. Opt-in PostgreSQL repository integration tests cover refresh-session rotation, replay/family revocation, family-wide logout, idempotency, expiry, and concurrent refresh behavior. These tests do not cover every possible database failure or concurrency interleaving.

#### `POST /auth/logout`

Revokes all active refresh sessions in the token family associated with the supplied refresh token. Repeating logout is safe; an unknown token is a no-op. Already-issued access JWTs remain valid until expiry because access-token revocation enforcement has not been implemented.

#### `GET /auth/me`

Requires an `Authorization: Bearer <access-token>` header. The middleware validates the token, reloads current principal and authorization state, and returns principal ID/type plus staff-only school/username context.

This endpoint returns identity context only; complete user-profile retrieval and business-resource authorization remain pending.

Platform Admin uses POST /auth/platform-admin/login with email/password and must complete TOTP enrollment or verification through /auth/mfa/enroll/verify or /auth/mfa/verify. MFA recovery-code regeneration routes are also present. See docs/API.md for request and response contracts.

## Database Migrations

The repository includes migrations through V14. The V4 principal/RBAC schema and subsequent student/guardian, Parent–Guardian, school-role provisioning and other additive migrations are part of the current branch. The user confirmed school_management_test at cumulative version 13 with dirty=false; V13's school-role provisioning integration test passed. V14 first-Platform-Admin bootstrap is source-only and must not be applied until reviewed and explicitly approved. The state of school_management is not inferred; do not apply migrations there without separate explicit approval.

## Verification

On commit bea184f, the user confirmed from backend: go test -count=1 -p 1 ./... passed with exit code 0, and go vet ./... passed with exit code 0. The focused TestPrincipalRefreshSessionCreateInTxPostgresIntegration assertions passed, although that single command reported a Windows temporary executable cleanup/lock exit code of 1. The subsequent full suite completed with exit code 0. Additional V14 bootstrap hardening/test commits have since been added; rerun the suite on the latest pulled commit before treating those changes as verified. PostgreSQL V14 execution remains pending.

## Not Yet Implemented

- Automated migration tests against a disposable test database.
- Complete user-profile retrieval from `GET /auth/me`.
- Access-token revocation enforcement.
- Complete resource-level tenant isolation and assignment/field-level authorization for school business APIs; those APIs are not yet built.
- Safe custom-role permission grants and user ALLOW overrides. They intentionally fail closed until every grant is linked to the exact delegation record that authorized it.
- Staff password change/reset lifecycle, forced first-login password change, and immediate access-token invalidation after administrative security changes.
- Complete role/permission administration workflows and route-level guards for the future business modules.
- Final PostgreSQL V14 migration and end-to-end first Platform Admin bootstrap verification.
- Complete HTTP end-to-end security tests and production database least-privilege grants.
- Student, teacher, class, subject, and attendance APIs.
- React frontend and Platform Admin, School Admin, and school-staff dashboards (Phase 1).
- Parent/Driver mobile interfaces and Parent authentication flow (Phase 2; intentionally deferred).
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
- `docs/PHASE1-RBAC.md` — approved Phase 1 web roles, permission catalogue, teacher scopes, staff account rules, and future Parent mobile access model.
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
