# Changelog

All notable changes to the School Management System are documented in this file.

This project is under active development. Verification results describe the latest reported local checks and should be rerun whenever relevant code changes.

## [Unreleased]
### Database — V13 new-school system-role provisioning (2026-11-03)

- Added an additive V13 migration that provisions the fixed School Admin and Teacher system roles whenever a new school is inserted.
- The School Admin role receives exactly the approved 51 permissions; the Teacher role receives exactly the approved 7 permissions.
- Direct runtime creation of system roles and direct modification of system-role permission bundles remain blocked; only the nested school-insert trigger may provision the reserved roles.
- Added PostgreSQL integration coverage for exact role bundles, rejection of direct system-role writes, and atomic rollback when a required permission is inactive.
- User confirmed V13 is applied on school_management_test (cumulative migration version 13, dirty=false); the V13 school-system-role provisioning integration test passed. This does not establish the school_management state.

### Database verification clarification — 2026-11-03

- The user later confirmed `school_management_test.schema_migrations` at cumulative version 13, dirty=false. This supersedes the earlier version-12 checkpoint and does not establish the state of school_management.
- A dedicated `TestParentGuardianAuthorizationV12PostgresIntegration` run was reported PASS. A later verbose attempt skipped integration tests because `RUN_POSTGRES_INTEGRATION_TESTS=1` was omitted; re-run with the flag after relevant changes.
- Older changelog entries below retain the status as it was recorded at their original checkpoint.

### Product direction — web-first delivery (2026-11-03)

- Approved Phase 1 as the complete web application for Platform Admin, School Admin and school staff.
- Deferred Parent and Driver mobile applications/interfaces until Phase 1 web completion. Parent access is intended to be read-only initially; authentication and OTP provider remain undecided for Phase 2.
- Paused further Parent-specific implementation while preserving existing Parent/Guardian schema and OTP foundation.
- Confirmed future-ready database principles: shared identity/role boundaries, strict school ownership, student/enrollment history separation, guardian relationship history, transport extensibility, auditability and shared backend APIs for web/mobile. No speculative schema or migration execution is authorized.
- Set delivery order: reconcile existing backend/database foundation, establish frontend/API foundation, then deliver Platform Admin, School Admin and staff modules with server-side authorization and tests.


### Parent OTP request and delivery foundation — 2026-11-02

- Added a provider-independent Parent OTP request service with secure six-digit generation, keyed mobile/IP/OTP hashes, five-minute expiry, repository-enforced resend/rate limits, and delivery through an injected provider interface.
- The service never returns or logs the OTP. Production must not use the test-only fake delivery provider; production SMS provider configuration and India TRAI/DLT registration remain deployment requirements.
- If delivery fails, the pre-account challenge is invalidated without refunding the reserved mobile/IP rate-limit counters.
- If challenge invalidation also fails, the service returns safe delivery/cleanup sentinels without exposing provider or database error details; unit tests cover this failure path.
- User-confirmed verification on commit `c78e770`: Parent OTP focused unit tests, full `go test -count=1 -p 1 ./...`, and `go vet ./...` passed on Windows. Windows temporary test-executable cleanup warnings appeared after successful package results.
- User-confirmed PostgreSQL integration test on commit `7105762`: pre-account challenge persistence, rate limits, failed-delivery invalidation, and non-refunded rate-limit counters passed on `school_management_test`.

### Database — V12 Parent–Guardian authorization foundation

- Added migration V12 for school-authorized guardian portal eligibility, authenticated Parent-to-Guardian links, and revocation history.
- Added same-school and exact verified-mobile checks before a Parent principal can be linked to a guardian record.
- Added irreversible revocation history for official student–guardian relationships while allowing a later re-link as a new row.
- Extended Parent OTP storage to support pre-account challenges using a keyed mobile HMAC without creating or reserving an unauthenticated principal.
- Added persistent keyed-HMAC mobile/IP OTP rate-limit buckets and append-only audit events for relationship changes.
- Added PostgreSQL integration-test source for link validation, active relationship visibility, revocation, pre-account OTP uniqueness/immutability, and rate-limit storage.
- V12 is source-only at this checkpoint. It has **not** been applied or PostgreSQL-tested. Parent OTP request/delivery service exists, but OTP verification, Parent account/session creation, Parent auth handlers/routes, and business-resource authorization remain pending.


### Database — V11 student and guardian foundation

- Added stable school-scoped student records with normalized, school-unique admission numbers.
- Added separate school-scoped guardian contact records; contact mobile numbers and email addresses are intentionally not unique and do not authenticate a Parent account.
- Added a many-to-many student/guardian mapping with same-school composite foreign keys, duplicate-link prevention, and immutable school ownership.
- Added PostgreSQL integration coverage for multiple guardians per student, one guardian linked to multiple students, duplicate admission/link rejection, cross-school link rejection, and school-scope immutability.
- Kept academic-year enrollment, class/section placement, and authenticated parent-to-guardian/child relationships out of V11.
- User applied V11 to the dedicated `school_management_test` database; `schema_migrations` reported version `11`, `dirty = false`.
- User ran `go test -count=1 -p 1 ./...` with `RUN_POSTGRES_INTEGRATION_TESTS=1`; all packages with tests passed. `go vet ./...` completed without diagnostics.
- The MFA recovery-code PostgreSQL integration test passed twice consecutively after commit `79c2717` made its fixtures unique per run. Its append-only recovery-code history constraint remains intact.

### Database — V4 identity/RBAC migration draft

- Added a draft additive migration for shared principals, globally unique staff usernames, account subtype tables, permission catalogue, system roles, role assignments, bounded overrides/delegations, principal-bound refresh sessions, MFA/parent OTP storage, and append-only audit events.
- Preserved existing user IDs and refresh-token family identifiers in the migration design.
- Added `docs/V4-IMPLEMENTATION-NOTES.md` documenting observed v3 database state, migration scope, compatibility behavior, and blockers.
- Migration is **not applied and not PostgreSQL-tested**; Go principal-aware authentication, RBAC enforcement, and automated migration tests remain required.
- Added a V4 username preflight that validates normalized legacy usernames, detects global collisions before changing data, reports affected user IDs, and only then canonicalizes case/outer whitespace. It does not silently rename invalid or colliding accounts.
- Added database guardrails for custom-role insertion: runtime inserts cannot create system/platform/global roles, must be school-scoped staff roles created by an active staff principal in that school, and cannot change a role's system classification or creator afterward. Exact `roles.create_custom` authorization remains an API responsibility.

### Documentation — Approved Phase 1 account and RBAC decisions

- Finalized Phase 1 core roles: Platform Admin, School Admin, Teacher, and Parent.
- Confirmed one-school-only staff accounts and globally unique, case-insensitive staff usernames; staff working at multiple schools must receive separate accounts.
- Confirmed School Admin staff provisioning, temporary initial passwords, staff self-service password change using the current password, and admin reset without the old password with session revocation, forced next-login change, and audit logging.
- Confirmed School Admin may create school-scoped custom roles only within an authorized permission delegation ceiling; additional operational roles remain future scope.
- Confirmed separate teaching and class-in-charge assignments, field-limited teacher student data, and class-specific attendance/timetable/homework/notices access.
- Confirmed one verified parent mobile/OTP account can access multiple officially linked children across schools; parent selects a child, server verifies the link on every request, and student login accounts are excluded from Phase 1.
- Added `docs/PHASE1-RBAC.md` as the detailed approved permission catalogue and authorization reference. These are requirements only; RBAC implementation remains pending.

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
- Family-wide logout with idempotent revocation and unknown-token no-op behavior.
- Shared transaction-scoped PostgreSQL advisory locking between refresh rotation and logout, plus per-user locking across concurrent rotations.
- `POST /auth/logout` endpoint for family-wide refresh-session revocation.
- Unit and handler tests for implemented authentication behavior.
- PostgreSQL refresh-session repository integration tests covering rotation, replay detection, token-family revocation, family-wide logout, idempotency, expiry, concurrent refresh, and concurrent logout/refresh behavior.
- Development-only seed command with environment safeguards.
- Documentation covering requirements, architecture, database design, API conventions, security, testing, progress and handoff.

### Changed

- Reconciled README and project documentation with the refresh-token family-wide logout and PostgreSQL advisory-lock implementation.
- Clarified migration constraints, destructive rollback behavior, and the distinction between implemented authentication and pending authorization controls.

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
- Serializes refresh rotation and logout within a token family using a PostgreSQL transaction advisory lock; rotations for the same user are also serialized across token families using a per-user lock.
- Rotates refresh tokens and revokes active sessions in a token family when a revoked token is replayed.
- Added safeguards to the development seed command.
- Documented remaining work for RBAC, tenant isolation, MFA, rate limiting and production security hardening.

### Testing

- Backend Go test suite reportedly passes with `go test -count=1 ./...`.
- Backend static analysis reportedly passes with `go vet ./...`.
- PostgreSQL refresh-session repository integration tests reportedly pass against the dedicated `school_management_test` database.
- Integration tests cover refresh rotation and replay, token-family revocation, family-wide logout/idempotency, expired tokens and concurrent refresh behavior.
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

### 2026-11-01 — Windows principal-auth verification

- User ran the prescribed `gofmt` command on the principal-auth cutover files; it completed without errors.
- User ran `go test -count=1 -p 1 ./...` from the backend directory. All packages with tests passed: `cmd/api`, `internal/config`, `internal/handlers`, `internal/middleware`, `internal/repository`, and `internal/services`. The remaining listed packages reported no test files.
- User ran `go vet ./...`; it completed without diagnostics.
- The reviewed local diff in the two Go repository files contained only formatting changes (import ordering and removal of extra blank lines).
- These results do not establish that opt-in PostgreSQL integration tests ran, and they do not verify or authorize execution of Migration V4.

## Latest Reported Git Checkpoint

- Latest user-reported pushed code checkpoint represented by supplied ZIP: `cb6af3b` — `docs: finalize phase 1 RBAC and account decisions`
- Full commit hash recorded by the ZIP root: `cb6af3b5abc5249fba420e2699d16e87b1d14664`
- The ZIP is a source snapshot without `.git`; verify local `git status -sb` and remote alignment before committing these documentation updates.
- Branch: `main`; remote: `origin`
- Later local modifications were reported. Verify the current working tree before committing; this changelog does not assert that it is clean.

## Changelog Guidelines

- Add entries when meaningful changes are completed.
- Describe verified implementation rather than planned functionality as completed.
- Keep unreleased work under `[Unreleased]`.
- Add a dated or versioned section when preparing a release.
- Re-run relevant tests after code changes and before recording verification results.
- Do not claim production readiness until required security, testing and operational controls are in place.

## 2026-11-01 — V4 principal-auth build-fix checkpoint

- Wired the API to principal-based refresh sessions and live principal/permission loaders.
- Added globally unique staff-principal authentication lookup and service method.
- Updated `/auth/me` to principal context and revised handler/middleware/API/session tests for principal-based interfaces.
- Added fail-closed permission resolution: system roles only, explicit DENY overrides, no custom-role or ALLOW override grants until exact delegation linkage is implemented.
- Added runtime allowlisting for the approved School Admin permission bundle.
- Full Go tests/vet remain unverified in the review environment because Go 1.27.1 and dependencies were unavailable. V4 migration remains unexecuted and unsafe to run. See `docs/CURRENT-BUILD-FIX-HANDOFF-20261101.md`.
- V4 C4 hardening: principal-role assignments now reject inactive/missing target principals; School Admin provisioning requires an active Platform Admin holding the `platform.school_admins.create` system-role grant; routine staff role assignment requires an active same-school School Admin with the `roles.assign` system-role grant and no explicit DENY override. Migration remains unapplied and PostgreSQL-unverified.
- V4 C5 hardening: user permission override writes now require the grantor to hold the active `permissions.override` system-role grant as a same-school School Admin; an explicit DENY override on that authority permission blocks the operation. Delegation-ceiling validation remains mandatory. Migration remains unapplied and PostgreSQL-unverified.
- V4 C6 hardening: delegation revocation is now one-way. Once `revoked_at` is set, the timestamp cannot be cleared or rewritten, preventing revoked delegation records from being reactivated. Migration remains unapplied and PostgreSQL-unverified.
- V4 C7 audit: confirmed that custom-role grants and user ALLOW overrides are intentionally excluded from effective permissions until each grant is linked to the exact delegation record that authorized it. Explicit DENY overrides remain effective. No schema change was made; enabling delegated grants is blocked pending delegation-ID linkage and revocation-aware resolution.
- V4 C8 hardening: added a statement-level `BEFORE TRUNCATE` trigger for `audit_events`, alongside existing UPDATE/DELETE rejection. Deployment must still enforce least-privilege application grants and separate table ownership; migration source cannot safely guess deployment role names. PostgreSQL execution remains pending.
- V4 C9 hardening: Parent role seeding now uses an explicit seven-key allowlist rather than inheriting every global permission. A migration assertion checks the exact Parent role count and rejects both missing and unexpected permission grants. PostgreSQL execution remains pending.
- V4 C10 hardening: School Admin bundle verification now checks the exact approved 51-key permission set in both directions, in addition to the count, and restricts the assertion to School Admin system roles. PostgreSQL execution remains pending.

- V4 C11 hardening: Teacher system-role seeding is explicitly limited to school-scoped permissions and the migration asserts one Teacher system role per school with exactly the approved seven-key default bundle. Assignment-specific access remains an API responsibility. PostgreSQL execution remains pending.
- V4 C12 hardening: principal role assignment rows now preserve immutable identity/provenance, cannot be deleted, must be inserted active, and cannot be reactivated or have their revocation timestamp rewritten. Re-grants require a new history row. PostgreSQL execution remains pending.
- V4 C13 hardening: permission delegation rows now start active, preserve immutable identity/capability/grantor provenance, and cannot be deleted. Combined with C6's one-way revocation guard, restoring authority requires a new delegation record. PostgreSQL execution remains pending.
- V4 C14 hardening: permission catalogue rows can no longer be deleted, including unused keys not yet protected by foreign keys. Obsolete permissions must be deactivated while retaining their stable identity. PostgreSQL execution remains pending.
- V4 C15 hardening: MFA recovery-code rows must start unused, preserve immutable principal/hash/creation provenance, and cannot be deleted. A code can be consumed once and its used state cannot be cleared or rewritten; replacement codes require new rows. PostgreSQL execution remains pending.
- V4 C16 hardening: parent OTP challenges must start unused with zero attempts; challenge identity/hash/expiry are immutable; attempts cannot decrease; consumed challenges cannot be reactivated or deleted. The resend flow must retire expired unconsumed challenges before inserting a replacement. PostgreSQL execution remains pending.

### Fixed — V4 migration source audit C17

- Corrected malformed PL/pgSQL dollar-quote delimiters in five trigger functions (delimiter-only change; no business logic changes).
- Static full-file scan confirms all five opening/closing pairs are valid and no targeted malformed delimiters remain.
- PostgreSQL 18 execution and integration verification are still pending; Migration V4 remains unapplied.


### Authentication/RBAC hardening — 2026-11-03

- Added a one-time first Platform Admin bootstrap path through a manually invoked CLI; no public registration endpoint is exposed.
- Hardened bootstrap password entry to require an interactive terminal and read the password without echo. Plaintext is no longer accepted through an environment variable; the byte buffer is cleared immediately after hashing.
- Refactored the repository bootstrap transaction body for rollback-based PostgreSQL integration testing. Added assertions for principal/account creation, the Platform Admin system-role assignment, the audit event, rejection of a second bootstrap attempt, and the V14 system-role-only trigger condition.
- Tightened the V14 database exception to require the reserved Platform Admin role to be marked is_system = TRUE. Existing authenticated role-assignment rules remain in the replacement trigger.
- V14 has not been applied to PostgreSQL. New Go changes and the V14 integration test must be run on the user's machine after pulling the latest branch; migration application to school_management_test requires explicit approval.
