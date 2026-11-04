> **2026-11-04 latest verified RBAC HTTP checkpoint:** `d5e0c8d` adds a real-router PostgreSQL integration success path. Push run `41764671472` and Draft PR run `41764679462` both passed V1–V16 migrations on ephemeral PostgreSQL 18, the full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1`, and `go vet ./...`. The test verifies 200 for active identity, 403 with no grants, 200 after a database-authorized School Admin role grant, and 401 after token-version revocation. It uses a controlled token validator, not real JWT signature validation. All test writes, including a possible temporary first Platform Admin bootstrap, are rolled back.

> **2026-11-04 latest remote checkpoint:** `9e74c88` adds the first HTTP/PostgreSQL RBAC integration slice. Push run `41762250818` and Draft PR run `41762256758` both passed migrations V1–V16 on ephemeral PostgreSQL 18, the full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1`, and `go vet ./...`. The new test uses a controlled token validator with real PostgreSQL-backed principal and permission repositories; real JWT signature validation and a granted-role success path remain pending.

> **2026-11-03 approved delivery direction:** Phase 1 is the complete web product for Platform Admin, School Admin and school staff. Phase 2 Parent/Driver mobile interfaces begin only after the web product is complete. Parent work is paused; preserve existing Parent/Guardian schema and OTP foundation without extending it. Parent authentication/OTP provider remains undecided. Database design must remain extensible for future modules without speculative migrations. Active branch checkpoint before this documentation update: `c79b3a5959e0d125dae0326298b554f7b2f90133` (`fix/v4-auth-security-hardening`).

> **Current implementation reality (2026-11-03):** The repository has a Go/Gin backend foundation but no React frontend directory yet. Authentication, staff account lifecycle, role/delegation/override repository workflows, and 14 role-administration route permission gates have automated coverage. Full business APIs, end-to-end HTTP-to-PostgreSQL authorization, tenant/resource isolation, class/section/subject assignment scope, and dashboards remain pending. The latest CI run on commit `2c81dfc` passed the full PostgreSQL 18 integration suite and `go vet ./...`.

> **Migration safety:** The user's direct PostgreSQL query reported `school_management_test.schema_migrations` at `version=12, dirty=false` (2026-11-02), so V11 and V12 are included in that cumulative test schema. A dedicated V12 Parent–Guardian integration run was reported PASS; a later verbose attempt skipped integration tests because `RUN_POSTGRES_INTEGRATION_TESTS=1` was omitted. Re-run with the flag after relevant changes. Do not apply migrations to `school_management` without separate explicit approval.

# Project Progress
>
> ## Current Stage
>
> **Current milestone:** V4 RBAC/security hardening is continuing on `fix/v4-auth-security-hardening`. Parent OTP/registration implementation remains paused by decision. The latest verified remote code checkpoint is `9e74c88`; push and Draft PR CI both passed V1–V16 on ephemeral PostgreSQL 18, the full Go integration suite, and `go vet ./...`. The first HTTP/PostgreSQL RBAC integration slice now verifies database-backed principal identity, default-deny role access, and token-version revocation. The user's persistent databases were not modified by CI.
The project has a working authentication foundation, including login, JWT authentication middleware, a protected identity endpoint, refresh-token rotation, replay detection, family-wide logout revocation, family-scoped transaction advisory locking, and per-user serialization of concurrent rotations across token families.

The last recorded remote documentation checkpoint before the current local verification was:

- **Commit:** `cb6af3b` — `docs: finalize phase 1 RBAC and account decisions`
- **Full commit hash in ZIP root:** `cb6af3b5abc5249fba420e2699d16e87b1d14664`
- **Branch:** `main`
- **Repository:** https://github.com/TheSumitNain/school-management-system
- **Source snapshot:** supplied ZIP `school-management-system-audit(1).zip`; it contains source and docs but no `.git` directory.
- **Local working-tree state:** not established by the ZIP. The user previously reported an untracked `APPLY-DOCS.txt`; inspect `git status -sb` in the actual local repository before committing. Do not assume the working tree is clean.

The refresh-session work includes opt-in PostgreSQL integration tests for rotation, replay/family revocation, family-wide logout, idempotency, expiry, and concurrent refresh requests.

The user reported the full backend test suite and `go vet` passing after the concurrency changes. The dedicated PostgreSQL integration run included six refresh-session lifecycle subtests plus the cross-family inactive-user race regression test. `git diff --cached --check` passed and the `cb6af3b` commit was pushed. These are historical user-reported results; rerun them in the actual local repository before relying on them.

**Important:** Passing tests does not establish production readiness. RBAC, tenant authorization, account-security controls, deployment hardening, and other production requirements remain outstanding.

## Project Overview

- **Product:** Commercial multi-tenant School Management System
- **Initial planning target:** 25–30 schools, approximately 400–500 students per school
- **Frontend:** React + TypeScript + Vite
- **Backend:** Go + Gin
- **Database:** PostgreSQL
- **Database access:** pgx / pgxpool
- **Architecture:** Modular monolith using a shared database and shared schema initially
- **Tenant isolation:** Must be enforced by the backend
- **Repository:** https://github.com/TheSumitNain/school-management-system
- **Main branch:** `main`
- **Default local API port:** `8081`
- **Local repository:** `D:\school-management-system`
- **Backend directory:** `D:\school-management-system\backend`
- **Go module:** `school-management-system`

The school and student counts are planning targets, not verified capacity limits.

## Completed

### Planning and Documentation

- [x] Initial product requirements documented
- [x] Architecture principles documented
- [x] Database design principles documented
- [x] API conventions and endpoint documentation created
- [x] Security requirements documented
- [x] Testing strategy documented
- [x] Project handoff documentation created
- [x] Progress and changelog files maintained
- [x] Documentation consistency audit against supplied `cb6af3b` source snapshot; RBAC decisions documented as approved requirements

### Backend Foundation

- [x] Go module and Gin API scaffold
- [x] Environment-based configuration
- [x] Configuration validation
- [x] PostgreSQL connection pool using pgxpool
- [x] Database connectivity and readiness check
- [x] `GET /health`
- [x] `GET /ready`
- [x] HTTP server timeouts
- [x] Graceful shutdown handling
- [x] Gin trusted proxies explicitly configured
- [x] Configurable application port, default `8081`
- [x] Development and production configuration distinctions

### Database and Migrations

- [x] Local `school_management` database configured
- [x] Version-controlled migration tooling
- [x] Migration v1 for the `schools` table
- [x] Migration v2 for `users` and `refresh_sessions`
- [x] Migration v3 for refresh-token family identifiers
- [x] Migration v3 applied to the development database and previously reported clean (`dirty = false`)
- [x] Migration v3 applied to the dedicated `school_management_test` database and reported clean (`dirty = false`)
- [x] User and school repository operations implemented
- [x] Refresh-session repository integration tests against PostgreSQL
- [ ] Automated migration tests against a dedicated clean test database
- [ ] Production migration and recovery procedure fully validated

**Migration safety:** Always verify the target database and migration state before deployment. Never run destructive down migrations against production without an approved backup and recovery plan.

### Authentication

- [x] User model and repository operations
- [x] School repository lookup
- [x] bcrypt password hashing and verification
- [x] Password validation and password-service tests
- [x] Authentication service for creating users in an active school
- [x] Authentication service for verifying credentials
- [x] Authentication service unit tests
- [x] JWT access-token generation and validation
- [x] JWT issuer, expiry and required-claim validation
- [x] HTTP login handler
- [x] `POST /auth/login`
- [x] Login response excludes password hashes
- [x] Invalid credentials return an unauthorized response
- [x] Login handler tests
- [x] Development-only seed command with environment safeguards
- [x] JWT authentication middleware for protected routes
- [x] Bearer access-token validation
- [x] Validated user ID and school ID stored in Gin request context
- [x] Missing or invalid authentication rejected by middleware
- [x] Invalid token identity claims rejected
- [x] Protected `GET /auth/me`
- [x] `/auth/me` returns authenticated user ID and school ID
- [x] Opaque refresh-token generation
- [x] SHA-256 hashes stored for refresh tokens rather than raw tokens
- [x] Configurable refresh-token expiry through `REFRESH_TOKEN_EXPIRY_DAYS`
- [x] Default refresh-token expiry of 7 days, with configuration validation
- [x] `POST /auth/refresh`
- [x] Refresh-token rotation
- [x] Token-family tracking for replay detection
- [x] Refresh-token family revocation on detected replay
- [x] `POST /auth/logout`
- [x] Session-service unit tests and authentication-handler tests
- [x] PostgreSQL integration coverage for refresh-session lifecycle
- [x] Concurrent refresh-token reuse detection tested against PostgreSQL
- [x] Cross-family inactive-user race regression test included: concurrent rotations across separate token families are serialized per user so an inactive-user revocation cannot be bypassed by a competing family rotation

**Known limitations:**

- `/auth/me` returns the authenticated identity; it does not yet provide a complete user profile.
- Authentication middleware establishes identity but does not itself enforce roles, permissions or complete tenant authorization.
- PostgreSQL integration tests cover the listed refresh-session scenarios, but they do not prove every possible database failure or concurrency scenario is safe.
- Immediate revocation of already-issued access tokens requires a separately designed and implemented strategy.
- Authentication endpoints do not mean that public registration or the full school-onboarding workflow exists.

## Latest Verification Results

The following checks were reported after the PostgreSQL integration test work was completed. These are historical results; rerun them after any code changes.

### Repository integration tests

Command, run from `D:\school-management-system\backend`:

```bat
go test -count=1 -v ./internal/repository
```

**Result:** PASS. A repeated run with `go test -count=20 ./internal/repository` was also reported successful.

Verified scenarios:

- [x] Refresh-token rotation revokes the old session
- [x] Reuse of a rotated token is detected
- [x] Replay detection revokes the replacement token in the same family
- [x] Logout revokes the complete refresh-token family
- [x] Logout revocation is idempotent
- [x] Expired refresh tokens are rejected
- [x] Concurrent refresh requests produce one successful rotation and one reuse-detection result
- [x] Cross-family inactive-user race regression is covered by a dedicated PostgreSQL-backed repository test

### Full backend tests

Command:

```bat
go test -count=1 ./...
```

**Result:** PASS.

Verified packages:

- [x] `cmd/api`
- [x] `internal/config`
- [x] `internal/handlers`
- [x] `internal/middleware`
- [x] `internal/repository`
- [x] `internal/services`

The `cmd/devseed`, `cmd/migrate`, `internal/database`, and `internal/models` packages reported no test files.

### Static analysis

Command:

```bat
go vet ./...
```

**Result:** PASS, with no reported diagnostics.

### Git verification

- [x] Integration-test file staged and reviewed
- [x] `git diff --cached --check` completed without whitespace errors
- [x] Latest user-reported pushed code checkpoint represented by supplied ZIP: `cb6af3b` (`docs: finalize phase 1 RBAC and account decisions`)
- [ ] Verify actual local Git status and remote alignment before committing documentation; the ZIP has no `.git` metadata

These results describe the last verified state. Repeat relevant checks after future code changes.

## Approved Phase 1 Product and Authorization Decisions (Implementation Pending)

The user has finalized the Phase 1 account and RBAC direction. The detailed authoritative matrix is `docs/PHASE1-RBAC.md`. Key decisions:

- Phase 1 web roles: Platform Admin (global), School Admin (one school), and school staff (one school; Teacher plus approved school-scoped roles). Parent and Driver mobile experiences are Phase 2.
- Every staff account is school-specific. A person employed by multiple schools receives separate accounts, passwords, and sessions for each school. Staff usernames are globally unique and case-insensitive.
- School Admin provisions staff with a temporary initial password; staff must change it at first login and must enter the current password for self-service changes. School Admin can reset same-school staff passwords without the old password; reset revokes sessions/access, requires a change at next login, and is audited.
- Platform Admin provisions School Admin accounts. School Admin cannot create another School Admin or Platform Admin.
- School Admin may create school-scoped custom roles from the platform permission catalogue, only within an explicit delegation ceiling. Operational roles (Clerk, Manager, Librarian, Gate Keeper, Helper, Bus Conductor, etc.) are future scope, not Phase 1 implementations.
- Multiple roles per staff account are supported within that account's single school. Role permissions plus per-user allow/deny overrides are supported; explicit deny overrides role allow, explicit authorized allow can add a permission otherwise absent, and default is deny.
- Teaching assignment and class-in-charge assignment are separate. Subject teachers see only basic student fields (name, roll number, class, section) for assigned class/subject and relevant permitted teaching data. A teaching assignment alone never grants attendance access.
- A class-in-charge teacher can access attendance, same-day attendance correction within the configured cutoff, basic student details, timetable, homework, and applicable notices only for the specific class/section in charge. Past attendance correction is School Admin-only.
- Future Parent mobile target: one verified Parent identity may access officially linked children across schools, read-only. Authentication and OTP provider are deferred to Phase 2; no Parent implementation is active now.
- Permissions are checked server-side on each protected request; permission changes apply to the next request. Tenant, assignment, field-level, account-status, and school-status checks cannot be bypassed by overrides.

These are approved requirements, not proof of complete RBAC enforcement. The V4 SQL draft corrective audit items and Go principal-auth cutover work remain documented in repository history. The current branch includes V15 staff temporary-password enforcement and V16 exact delegation-ID linkage. Custom-role grants and ALLOW overrides are resolved only while the exact referenced delegation remains active. V15/V16 migrations and integration tests passed in GitHub Actions against ephemeral PostgreSQL 18; this does not establish that they have been applied to the user's persistent `school_management_test`. Role/permission administration HTTP routes now exist and all 14 route permission gates have router/middleware tests. Full HTTP-to-PostgreSQL end-to-end verification and business-resource/tenant/teacher authorization routes remain pending.

## 2026-11-03 Backend/Database Source Audit

- [x] Confirmed there is no frontend directory in the current repository.
- [x] Confirmed the API currently exposes authentication/MFA, `/auth/me`, `/health` and `/ready`; Platform Admin school onboarding and school business APIs are not yet wired.
- [x] Confirmed authentication middleware reloads current principal/token version and permissions; `RequirePermission` exists and fails closed.
- [x] Identified a school-onboarding schema blocker: V4 seeds school-scoped `school_admin` and `teacher` system roles only for schools existing at migration time; the current role guard blocks ordinary runtime creation of system roles.
- [x] Add V13 source migration for secure new-school system-role provisioning and a PostgreSQL integration test.
- [ ] Review/apply V13 only on `school_management_test` after user approval, then run `TestSchoolSystemRoleProvisioningV13PostgresIntegration` with the integration flag.
- [ ] Reconcile the actual `school_management_test` migration version/dirty flag; historical docs conflict about V4–V12 application status. Do not run migrations until this is resolved.

## Approved Web-First Delivery Plan — 2026-11-03

- [x] Confirm Platform Admin, School Admin and complete school staff web dashboards as Phase 1.
- [x] Defer Parent, Driver and other approved mobile-first interfaces to Phase 2 after web completion.
- [x] Pause Parent-specific implementation and preserve existing source/schema foundations.
- [x] Require future-ready but incremental database design; no speculative migrations.
- [x] Reconcile current backend/database modules and identify Platform Admin onboarding prerequisites.
- [ ] Verify V13 migration and new-school system-role provisioning on PostgreSQL before Platform Admin school creation APIs.
- [ ] Build the frontend foundation and start the Platform Admin workflow.

## In Progress

- [x] Audit project documentation against the latest reported implementation and test coverage
- [x] Cross-check README, progress, changelog, requirements, architecture, database, API, security, testing and handoff documentation for consistency
- [ ] Review remaining authentication security edge cases against the current local working tree
- [ ] Re-review refresh-token integration-test isolation and cleanup after any test changes
- [ ] Add migration tests against a dedicated clean database
- [x] Verify that `backend/.env` is ignored by Git and is not tracked
- [ ] Run the full verification workflow after each implementation milestone

## Not Yet Implemented or Verified

### Authentication and Account Security

- [ ] Complete user-profile retrieval
- [ ] Production school onboarding and secure invitation workflow
- [ ] Password-reset workflow and one-time reset tokens
- [x] Mandatory TOTP MFA for Platform Admin and School Admin (V8–V10 source and current auth flow)
- [x] MFA recovery codes, one-time recovery workflow and audit history (broader threat-model review remains pending)
- [x] Failed-login lockout: 5 failed attempts followed by a 15-minute lock
- [x] Identifier-based login lockout and IP-based login rate limiting
- [ ] Security-event logging and audit trail
- [ ] Account deactivation and complete session invalidation
- [ ] Admin idle-session timeout of 30 minutes
- [x] Current 12–72 byte password policy and local common-password denylist; breached-password intelligence remains pending
- [x] Password changes/resets immediately invalidate access JWTs through token-version increments; broader administrative event revocation policy remains pending

### Authorization and Multi-Tenancy

- [x] Core principal RBAC schema, live permission resolver, RequirePermission, and exact delegation-linked custom-role/ALLOW resolution; role administration and resource-level enforcement remain pending
- [x] Prepare V4 identity/RBAC SQL migration draft; user later reported cumulative V4–V12 applied on `school_management_test` (version 12, `dirty=false`)
- [ ] Implement permission checks on protected endpoints
- [x] Live permissions are reloaded from PostgreSQL on every protected request; permission changes apply on the next request
- [ ] Implement school-level tenant authorization
- [ ] Ensure client-provided school IDs never independently grant access
- [ ] Add cross-school authorization and data-isolation tests
- [ ] Restrict teachers to assigned classes and subjects
- [ ] Design and implement Parent access to verified linked children (Phase 2 only; paused)
- [ ] Implement authorized role and school-membership management

### Database and School Features

- [ ] Complete production database schema
- [ ] Add automated migration tests against a clean database
- [ ] Student APIs
- [ ] Teacher and staff APIs
- [ ] Class and subject APIs
- [ ] Daily attendance APIs
- [ ] Parent-child portal authorization (Phase 2; existing foundation preserved)
- [ ] Timetable, homework, exam results and notices
- [ ] Additional school workflows and reporting

### Frontend and Operations

- [ ] React frontend implementation
- [ ] Login and authentication integration
- [ ] Role-specific dashboards
- [ ] Parent/Driver mobile interfaces (Phase 2, after complete web delivery)
- [ ] CI/CD and deployment configuration
- [ ] Production TLS and certificate verification
- [ ] Production database connection-pool tuning
- [ ] Centralized logging, monitoring and alerting
- [ ] Backup and recovery procedures
- [ ] Security testing and load testing
- [ ] Production readiness review

## Next Steps

1. Reconcile the current V4 principal/authentication code and SQL migration drafts against the approved Phase 1 web roles; preserve refresh-token concurrency guarantees.
2. Complete the database/backend module blueprint against the actual existing migrations, models, repositories, handlers and tests. Identify additive schema gaps before implementing each module.
3. Complete permission enforcement, staff account lifecycle, school onboarding and school/resource tenant isolation.
4. Establish the React + TypeScript + Vite frontend foundation and typed API integration.
5. Deliver Platform Admin dashboard and school onboarding.
6. Deliver School Admin settings and school staff management.
7. Deliver academic setup, student/enrollment management, attendance, timetable, exams/results, homework, notices and reports one module at a time.
8. Run full backend, PostgreSQL integration, frontend build, authorization and tenant-isolation verification after relevant milestones.
9. Plan Parent/Driver mobile apps only after Phase 1 web completion; keep Parent work paused until then.
10. Complete production security, deployment, monitoring, backup and recovery checks before release.

## Working Rules

- Work on one small task at a time.
- Explain code in Hindi/Hinglish, step by step.
- Wait for command output before proceeding.
- Do not mark features complete without verifying implementation and relevant tests.
- Keep `PROGRESS.md` and `docs/HANDOFF.md` synchronized at meaningful milestones.
- Never commit `.env` files, passwords, signing secrets or real student data.
- Never share secrets in chat.
- Never trust client-provided identifiers for authorization.
- Do not run destructive production migrations without an approved backup and recovery plan.
- Do not claim production readiness until security, testing, deployment, monitoring and recovery requirements have been addressed.

## V4 Corrective Source Audit — 2026-11-01/02

- [x] C1–C17 source-level migration review and corrective changes are recorded in `docs/V4-IMPLEMENTATION-NOTES.md`
- [x] Principal role assignment, permission delegation, permission catalogue, MFA recovery-code, and parent OTP history guard source changes are present
- [x] Corrected PL/pgSQL dollar-quote delimiters in the five identified V4 trigger functions
- [ ] Re-run the corrected V4 source through a clean PostgreSQL 18 v3-to-v4 upgrade and trigger failure-case suite after syncing the latest branch

The user subsequently pulled the V11 and recovery-test fix commits. On the dedicated `school_management_test` database, V11 is applied and clean (`schema_migrations.version = 11`, `dirty = false`). The full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` and `go vet ./...` both passed on the user's Windows environment. Do not apply migrations to `school_management` without explicit approval.

## V11 Student and Guardian Foundation — 2026-11-02

- [x] Added `000011_add_student_guardian_foundation.up.sql`
- [x] Added a guarded down migration that refuses rollback when V11 data exists
- [x] Added school-scoped `students` and `guardians` records
- [x] Added many-to-many `student_guardians` mapping with composite same-school foreign keys
- [x] Added immutable school ownership for student and guardian records
- [x] Added PostgreSQL integration test source for many-to-many links, duplicate prevention, cross-school isolation, normalized admission-number uniqueness, and school-scope immutability
- [x] Updated `docs/DATABASE.md` and `CHANGELOG.md`
- [x] Pull latest branch through recovery-code fixture fix `79c2717`
- [x] Apply V11 only to `school_management_test`; subsequent direct query reported cumulative migration version `12`, `dirty = false` after V12
- [x] Run full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` and `go vet ./...`; both passed
- [x] Run focused recovery-code integration test twice consecutively; both runs passed

V11 is **applied and PostgreSQL-verified on the dedicated test database**. This verification does not imply production readiness or complete parent authorization. It does not create academic-year enrollment, class/section placement, Parent login endpoints, or authenticated Parent-to-child API access. V12 now defines the reviewed Parent–Guardian verification/revocation relationship but remains source-only and unverified. The Parent OTP request/delivery service is implemented separately; OTP verification, Parent account/session issuance, and protected child-data routes remain pending.

## 2026-11-03 — Staff password lifecycle and exact delegation linkage

### Implemented in source on `fix/v4-auth-security-hardening`

- [x] V15 migration adds `users.must_change_password`; existing accounts default to permanent-password state and new staff inserts default to temporary-password state.
- [x] Staff login and MFA-completed session issuance carry the current temporary-password state from PostgreSQL.
- [x] Protected middleware restricts temporary-password staff sessions to `POST /auth/password/change` and `GET /auth/me`; refresh rotation preserves the database-backed restriction.
- [x] Staff self-service password change verifies the current password and password policy.
- [x] School Admin staff reset route requires `users.password_reset`; repository independently checks active same-school School Admin authority, active target staff/school, and explicit DENY.
- [x] Password change/reset transactions increment `auth_principals.token_version`, revoke active refresh sessions, and append audit events.
- [x] V16 migration adds exact `delegation_id` references to custom-role permission grants and staff permission overrides; migration fails safely if un-attributable legacy custom grants/overrides already exist.
- [x] Live permission resolver includes custom-role grants and ALLOW overrides only when their exact delegation row remains active; DENY continues to remove access.
- [x] Added unit, handler, middleware, and opt-in PostgreSQL integration-test source for V15/V16 behavior.
- [x] Updated `docs/SECURITY.md`, `docs/PHASE1-RBAC.md`, `docs/TESTING.md`, and `CHANGELOG.md`.

### Latest automated verification

- [x] GitHub Actions push workflow on commit `524d795`: ephemeral PostgreSQL 18, V1–V16 migrations, full PostgreSQL integration suite, handler/service tests for staff account creation and role/delegation/override/assignment APIs, and `go vet ./...` PASS.
- [x] GitHub Actions pull-request workflow on commit `fcffdbf`: PASS.
- [x] Apply V1–V16 to a fresh ephemeral PostgreSQL 18 database in GitHub Actions; migration sequence completed successfully.
- [x] Run the complete Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` against ephemeral PostgreSQL 18; all integration tests passed, including V15 password lifecycle and V16 exact delegation revocation/regrant.
- [x] Run `go vet ./...` in the same CI job; passed.
- [ ] Apply V15/V16 to the user's persistent `school_management_test` only after checking its current migration version and dirty flag; CI did not modify that database.
- [ ] Confirm Windows local working-copy synchronization without overwriting the user's untracked helper files or tracked local edits.

### Boundaries

- No migration has been applied to `school_management` or `school_management_test` by this work.
- No permanent Platform Admin account has been created.
- Parent OTP/registration production implementation remains paused and untouched.
- Draft PR #1 remains open and must not be merged without explicit approval.
- Custom-role/override resolution is implemented at the permission resolver and database-guard level. School role creation, permission delegation, custom-role permission grants, staff ALLOW/DENY overrides, and same-school staff role assignment list/assign/revoke HTTP workflows now exist. Rollback-scoped PostgreSQL integration coverage for role/delegation/override/assignment repository workflows and the staff account create/list/update/disable/re-enable lifecycle has passed in GitHub Actions. All 14 role-administration route permission gates now have router/middleware tests. Full HTTP-to-PostgreSQL end-to-end verification and business-resource/tenant/teacher-scope authorization routes remain pending.


## 2026-11-03 — Role repository integration coverage

- [x] Added a shared transaction-aware repository DB interface implemented by `pgxpool.Pool` and `pgx.Tx`; repository-owned transactions use nested savepoints when called inside a caller-owned transaction.
- [x] Added rollback-scoped PostgreSQL integration coverage for custom-role create/list, permission catalogue, Platform Admin permission delegation grant/list/revoke, custom-role permission grant/revoke, staff role assignment/list/revoke, and staff ALLOW/DENY override set/list/delete.
- [x] Verified duplicate role/delegation/permission/assignment rejection, reserved School Admin assignment protection, unauthorized actor rejection, and audit events attributed to the correct Platform Admin or School Admin actor.
- [x] Added explicit SQL casts for JSONB audit metadata parameters in role administration operations.
- [x] GitHub Actions run `41657762673` on commit `64acf1b`: V1–V16 applied to ephemeral PostgreSQL 18, full Go suite with `RUN_POSTGRES_INTEGRATION_TESTS=1` PASS, and `go vet ./...` PASS.
- [x] No migrations were applied to the user's persistent `school_management_test` or `school_management` databases.
- [x] Added router/middleware permission-gate tests for all 14 Platform Admin and school role-administration endpoints, including AND semantics for routes requiring multiple permissions. Push run `41658994787` and Draft PR run `41659008482` on commit `2c81dfc` passed the full PostgreSQL integration suite and `go vet ./...`.
- [ ] Extend HTTP-to-PostgreSQL integration coverage from the verified role-list authorization path to role/delegation/override/assignment write endpoints.
- [ ] Implement and verify business-resource, tenant, class/section/subject assignment, and teacher field-level authorization.


## 2026-11-04 — First HTTP/PostgreSQL RBAC integration slice

- [x] Added an integration test that exercises the actual Gin router and authentication middleware with PostgreSQL-backed principal and effective-permission repositories.
- [x] Verified active staff identity inspection returns 200, staff without role grants is denied `/school/roles` with 403, and a stale token version is rejected with 401.
- [x] Push CI run `41762250818` and Draft PR CI run `41762256758` passed V1–V16 migrations, full PostgreSQL integration tests, and `go vet ./...` on ephemeral PostgreSQL 18.
- [x] Add a database-backed allowed-role path that proves a granted School Admin role reaches the real role handler. Role assignment uses the existing database trigger; the fixture and optional first Platform Admin bootstrap are rolled back.
- [x] Replaced the controlled token validator in this HTTP integration path with the real `TokenService`; the test signs an HS256 token and exercises actual middleware signature verification.
- [x] Added a tampered-signature request assertion; the real middleware rejects it with 401 while the original valid token proceeds through PostgreSQL-backed authorization.
- [x] Push CI run `41765751785` and Draft PR run `41765745170` on `ec55ea4`: V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` all PASS.
- [ ] Continue tenant/resource isolation and teacher assignment-scope authorization tests.
- No persistent user database was changed by CI; PR #1 remains Draft and unmerged.


## 2026-11-04 — Database-granted role authorization success path

- [x] Principal authorization repository can now use a caller-owned `pgx.Tx`, allowing the HTTP test router and PostgreSQL fixture to share one transaction.
- [x] Added `PlatformAdminRepository.BootstrapFirstInTx` so a temporary first Platform Admin can be created inside the test transaction only when needed; the transaction rollback removes the temporary account, role assignment, and audit event.
- [x] Extended `TestRBACHTTPAuthorizationUsesPostgresState`: a staff principal without grants gets 403; after verified MFA fixture setup and a database-trigger-authorized School Admin role assignment, the same request gets 200 from the real role handler; token-version increment then makes `/auth/me` return 401.
- [x] Push run `41764671472` and Draft PR run `41764679462` on `d5e0c8d` passed PostgreSQL 18 V1–V16 migrations, the full integration-enabled Go suite, and `go vet ./...`.
- [ ] Replace the controlled test token validator with real JWT signing/validation in this HTTP integration path.
- [ ] Extend HTTP/database integration coverage to delegation/override/assignment writes and tenant/resource/teacher assignment-scope isolation.
- All test fixtures are transaction-scoped and rolled back; no persistent user database or permanent Platform Admin account was modified.


## 2026-11-04 — Real JWT verification and RBAC tenant-boundary tests

- [x] HTTP/PostgreSQL integration now uses the real HS256 `TokenService`, not a fake token validator. A genuinely altered signature is rejected with 401; valid signed tokens proceed through live principal and permission loading.
- [x] Expanded PostgreSQL role-repository integration tests with a second school. A School A admin cannot list/create School B roles, view School B staff role assignments or permission overrides, assign a School A role to School B staff, or apply School A permission overrides to School B staff.
- [x] CI push run `41766408925` and Draft PR run `41766414685` on `c7dd5ce`: V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` all PASS.
- [ ] Continue cross-school isolation coverage for future business-resource APIs and data repositories; no academic class/section/subject assignment model or student-resource API is currently available to test teacher assignment scope.
- All integration fixtures are rollback-scoped. No persistent user database was modified. PR #1 remains Draft and unmerged.


## 2026-11-04 — Authorized custom-role creation over real HTTP/PostgreSQL

- [x] Extended `TestRBACHTTPAuthorizationUsesPostgresState` with an authorized `POST /school/roles` request using the real HS256 access token.
- [x] The request passes live PostgreSQL principal/permission checks, reaches the actual role handler/service/repository, creates a custom role, and returns HTTP 201 with the created role. The write is contained in the existing rollback-only transaction.
- [x] Draft PR CI run `41766719434` on `d5f8340`: all V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` PASS.
- [ ] Continue HTTP-to-PostgreSQL coverage for delegation, permission override, and staff-role assignment write routes.
- No persistent user database was modified; PR #1 remains Draft and unmerged.


## 2026-11-04 — HTTP permission delegation and custom-role permission grant

- [x] Extended the real HTTP/PostgreSQL integration flow with a signed Platform Admin token and mandatory verified-MFA fixture, scoped to the rollback-only test transaction.
- [x] Platform Admin grants `students.read_basic` delegation through the actual `POST /platform/schools/:schoolID/permission-delegations` route; response is HTTP 201.
- [x] School Admin consumes the exact returned delegation ID to grant `students.read_basic` to the previously created custom role through `POST /school/roles/:roleID/permissions`; response is HTTP 201.
- [x] Push CI run `41767168878` and Draft PR run `41767176542` on `6bf8ca5`: V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` PASS.
- [ ] Continue HTTP/PostgreSQL write-path coverage for delegation revocation, custom-role permission revocation, staff permission overrides, and staff role assignment/revocation.
- All writes and MFA fixtures are rolled back; no persistent user database was modified. PR #1 remains Draft and unmerged.


## 2026-11-04 — HTTP revocation write paths

- [x] School Admin revokes the custom-role `students.read_basic` grant through the actual `DELETE /school/roles/:roleID/permissions/:permissionKey` route; HTTP 200 verified.
- [x] Platform Admin revokes the exact permission delegation through the actual `POST /platform/permission-delegations/:delegationID/revoke` route; HTTP 200 verified.
- [x] Push CI run `41767372453` and Draft PR run `41767378009` on `0b560fd`: V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` PASS.
- [ ] Continue HTTP/PostgreSQL write-path coverage for staff permission overrides and staff role assignment/revocation.
- All writes, test Platform Admin/MFA fixtures, and audit events remain inside the rollback-only integration transaction. No persistent user database was modified.


## 2026-11-04 — HTTP staff role and permission override lifecycle

- [x] Added same-school target staff fixture and verified School Admin role assignment over HTTP (201), assignment listing (200), and assignment revocation (200).
- [x] Added Platform Admin `homework.create` override delegation with `can_override_users=true`; School Admin sets an ALLOW override (200), deletes the override (200), and Platform Admin revokes the delegation (200).
- [x] Push CI run `41767598076` and Draft PR run `41767602833` on `d90ba9e`: V1–V16 migrations, full PostgreSQL integration-enabled Go suite, and `go vet ./...` PASS.
- [ ] Review remaining authorization boundaries and add tests for HTTP write failure cases, cross-school route targets, and audit/event outcomes.
- All test staff, Platform Admin/MFA fixtures, assignments, delegations, overrides, and audit events are rolled back. No persistent user database was modified.
