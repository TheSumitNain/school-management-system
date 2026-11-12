# Project Status

**Last reconciled:** 2026-11-11  
**Repository:** `TheSumitNain/school-management-system`  
**Reviewed branch:** `feature/school-attendance-management`  
**Reviewed branch HEAD before this status reconciliation:** `f448a871b574a60a6cc8da810adf9d2e648a3495`

## Staff Management — employee profile foundation in progress

- Migration V28 introduces a separate `staff_profiles` table with school-scoped unique employee codes, required full name/designation/mobile, optional email/joining date, and Active/Inactive employment status.
- V29 enforces the login safety invariant at the database identity boundary: inactive employee profiles disable the canonical staff principal, increment its token version and revoke active refresh sessions; a principal cannot be reactivated until the employee profile is Active.
- Backend work now supports profile-aware account creation in one transaction and profile-aware listing/update. The frontend Staff Management form and table expose the employee details, account status and employment status.
- Existing staff accounts are preserved without fabricated employee data and are shown as profile-incomplete until updated. Academic history and assignments are not deleted by profile changes.
- Latest implementation commit and CI must be reconciled after the current workflow run completes. Do not apply V28/V29 to a persistent local database until the user explicitly approves after a verified backup.

This file is the current implementation-status reference. Older checkpoint notes must not override it. Verification statements below distinguish repository inspection, user-reported local checks and real browser/database operations.

## Latest academic UI correction — section medium and date selection

- The shared date-picker parser now correctly recognizes ISO dates selected from the calendar, so selected academic-year start/end dates render visibly in the field. This fix is in commit `40a3d70506c6c1e07e181c40adedf0de5b5d45d2`.
- Migration V27 adds nullable section medium metadata restricted to `hindi` or `english`. Section names remain independently customizable. The section create/edit form exposes an optional Medium of instruction field, and the section list displays it.
- Backend persistence, validation, audit metadata, API handling and frontend form support have been updated. Frontend CI passed on the initial implementation commit; backend CI verification for the latest commits must be checked before local acceptance.
- Section-optional student enrollment/default-group behavior and stream metadata are design decisions still requiring implementation and verification. Do not treat those as delivered yet.

## Delivery scope and current priority

- **Phase 1:** Complete web product for Platform Admin, School Admin and school staff.
- **Immediate priority:** Complete the School Admin workspace and verify its end-to-end operational workflows.
- **Phase 2:** Parent and Driver experiences only after Phase 1 school web delivery.
- **Parent OTP/SMS:** Explicitly deferred. Do not implement parent OTP login, SMS delivery or parent-facing authentication now. Preserve existing guardian/parent relationship foundations without expanding this workstream until Phase 2 is approved.
- **Scale target:** 25–30 schools and 400–500 students per school; capacity is not yet load-tested.

## Current verification task — persistent authenticated sessions (implementation pushed; local acceptance pending)

The persistent-session/MFA policy has been implemented on this branch and pushed to GitHub. Backend GitHub Actions passed on code/test commit `b3af5904bd6e862aa261ee519a22effcacf2377a`, including application of all migrations to an ephemeral PostgreSQL database, backend tests with PostgreSQL integration, and `go vet`. Frontend CI passed on commit `6541584154c554e6ea662bdb9c78ac1ce27206a1`.

The frontend already restores a session at app startup by calling the refresh endpoint with the same-origin HttpOnly cookie; this was verified by source inspection, not by a fresh browser close/reopen test. Do not mark the task fully accepted until the user pulls the branch, reviews/applies migration V25 to the intended local database after a backup, and manually verifies browser restart/session restoration, fresh-login MFA, background refresh, cross-tab account/logout behavior and security revocation.

- The refresh cookie is persistent with a 400-day Max-Age and is renewed on each successful refresh. Browser close/reopen and laptop restart restore the session while the browser retains the cookie; browser-enforced retention limits still apply.
- Platform Admin and School Admin must complete mandatory MFA on every fresh login. MFA must not be repeated during background access-token refresh or restoration of an existing valid session.
- The 30-minute administrative idle timeout and server-side absolute refresh-session expiry have been removed. Protected requests and refresh continue to validate the active session family.
- A session remains valid until the user manually logs out or a security/account event requires revocation (for example, password/security reset, account disablement or confirmed compromise).
- Manual logout must revoke the session family. Preserve opaque refresh-token rotation, replay detection/family revocation, transaction/concurrency protections, HttpOnly + Secure cookie policy, cross-tab logout synchronization, principal token-version checks and live database-backed RBAC authorization.
- Backend session lifecycle, persistent-cookie policy, regression/integration tests and backend/frontend security documentation have been updated. Automated browser close/reopen and interactive MFA acceptance remain pending locally.
- Work on the existing `feature/school-attendance-management` branch. Do not create a new branch or merge PR #6; neither is part of this task.

## Future product direction (documented, not current implementation scope)

- **Self-service school registration is deferred until the School Admin dashboard and the Phase 1 school web experience are complete and stable.** Do not build public registration, email-verification onboarding, or self-service tenant creation during the current workstream.
- The current supported school-creation path remains Platform Admin-assisted onboarding. Complete and verify that path first, including its real PostgreSQL transaction and failure handling.
- When the deferred self-service phase is approved, schools should be able to register and create their own initial School Admin account without manual Platform Admin intervention. Email ownership verification will be required, but email verification alone must not be treated as proof that an institution is genuine. Abuse controls and any school-authenticity review policy must be designed before release.
- Platform Admin-assisted onboarding must remain available for assisted/special-case onboarding.
- Both onboarding entry points must call one shared, server-authoritative school-onboarding service so tenant creation, initial role provisioning, transaction boundaries, validation and audit behavior remain consistent.
- The public school website is also a future product capability and should use the same product domain (school-specific paths such as `/s/{school-slug}`); do not introduce a separate domain per school as a requirement.
- Record this direction in the roadmap only. Do not create placeholder registration routes, forms, mock flows or database changes before the deferred phase is explicitly started.

## Repository reality

- Backend: Go + Gin, PostgreSQL + pgx/pgxpool.
- Frontend: React + TypeScript + Vite. A working frontend source tree exists, including the shared application shell, session-aware login/MFA flow, theme support, Platform Admin screens and School Admin feature pages.
- Deployment: production deployment architecture and operational controls are not yet complete.

## Implemented backend capabilities

### Identity and session security
- Principal-aware staff and Platform Admin authentication.
- JWT access tokens with database-backed principal/token-version checks.
- Opaque refresh tokens, rotation, replay/family revocation, transaction-lock concurrency protections, and protected-request refresh-family validation.
- Mandatory TOTP MFA for Platform Admin and School Admin, recovery-code workflows, attempt lockout/auditing, and versioned TOTP encryption-key rotation.
- Staff and Platform Admin password lifecycle, temporary-password enforcement, token-version invalidation and session revocation.
- Login lockout, IP rate limiting and breached-password checks.

### RBAC and platform administration
- Live database-backed permission resolution on protected requests.
- System/custom roles, scoped delegation, exact-delegation-bound grants, user ALLOW/DENY overrides and audit events.
- Route-level permission middleware and repository/database guards against self-escalation.
- Platform Admin lifecycle and transactional school onboarding with school system-role provisioning.
- `GET /auth/me` returns current effective permission keys for Staff and Platform Admin principals as a frontend navigation hint; server authorization remains authoritative.

### Selected school workflows
- School-scoped staff account lifecycle and password reset.
- School-scoped system/custom role management, permission delegation and staff permission overrides.
- Academic years, classes, sections, subjects, teacher-subject assignments and class-in-charge assignments.
- Student records, enrollment, promotion/history and teacher roster restrictions.
- Attendance APIs with targeted permission and scope checks.
- Timetable current-year lookup, period definitions and weekly section schedules, including active teacher-assignment validation, period-overlap prevention, class/teacher double-booking constraints, Teacher-scoped reads and atomic audit writes.
- School-scoped guardian management APIs and parent/guardian eligibility foundations.

## Frontend implementation present on the reviewed branch

- Shared login and authenticated application shell with role/permission-aware navigation.
- Platform Admin account list and school onboarding form.
- School Admin pages for staff accounts, roles/permissions, academic structure (including teacher-subject and class-in-charge assignment management), students/enrollments, attendance and timetable period/schedule management. Teacher accounts with `timetable.read` can view their own entries for the current academic year without being granted broad `academic_years.read` access.
- Separate page CSS, shared design tokens and Light/Dark theme foundations.
- Typed API modules and TanStack Query-based server state for school administration.

The presence of a page or API client does not mean its full browser workflow has been accepted. Continue testing against the real local API and verify every permission boundary. Do not present mock data or frontend-only checks as completed backend workflows.

## Recent verification evidence

### Timetable backend — implementation pushed; frontend pending
- Migration V26 adds school/academic-year-scoped timetable periods and weekly section entries. Composite foreign keys prevent cross-school/year mismatches; database triggers require an active teacher-subject assignment and reject break-period scheduling.
- Database safeguards prevent overlapping period definitions, duplicate section slots and teacher double-booking. A teacher assignment cannot be revoked while timetable entries still depend on it.
- Timetable period and entry APIs enforce live `timetable.read` / `timetable.manage` permissions, restrict Teacher reads to their own scheduled entries, and write audit records atomically with mutations.
- PostgreSQL integration tests verify the V26 schema and overlapping-period rejection. GitHub Actions backend run for `b52513d51c2bc8ba26d54f13c914ddac78bc371b` passed migration application, the full backend test suite including PostgreSQL integration, and `go vet` on commit `b52513d51c2bc8ba26d54f13c914ddac78bc371b`. The timetable current-year endpoint and frontend workspace have since been added; frontend TypeScript/Vite CI passed on commit `f448a871b574a60a6cc8da810adf9d2e648a3495`. The latest backend CI for the current-year endpoint is still running. Manual browser/role acceptance remains pending.


### User-reported local checks
- Full backend: `go test -count=1 -p 1 ./...` passed.
- Backend static analysis: `go vet ./...` passed.
- School onboarding handler tests: `go test -count=1 ./internal/handlers` passed after the latest pull. Windows printed a temporary test executable cleanup warning (`go: unlinkat ... handlers.test.exe`); the test package itself passed.
- Frontend production build passed after the latest pull, including the student-search enrollment updates (Vite 7.3.7; 1,657 modules transformed).
- School onboarding UI required-field checks were manually exercised for school name, school code minimum length, username minimum length and password minimum length.
- Student enrollment selection now uses server-side search (name/admission number) instead of loading only the first 100 students; a student must be selected before submission. The user pulled these changes and confirmed the production build passes.
- Platform Admin login and MFA verification succeeded against the user's `school_management` database.

### Scope and limitations of that verification
- No school was created during onboarding UI validation. A successful real onboarding transaction and live duplicate-code/username checks have not been demonstrated in this workflow.
- The onboarding handler/service tests cover success/error mapping and validation paths, but do not by themselves prove a real database transaction in the user's persistent database.
- The initial Platform Admin list request returned one HTTP 500 followed by HTTP 200; the page subsequently loaded, but the first response's cause has not been established.
- These checks do not prove production readiness, exhaustive authorization coverage or load capacity.
- Never infer the migration version or contents of the user's persistent databases from disposable CI environments.

## Not yet complete

- End-to-end browser verification of all School Admin workflows, including staff creation/role assignment, academic setup, student enrollment and attendance.
- Full browser verification of Platform Admin onboarding, lifecycle and delegation workflows.
- Complete tenant/resource/assignment/field-level authorization coverage across every business module and route.
- Remaining academic/business modules, including timetable, examinations and results.
- Teacher/staff-specific operational screens and assignment-limited browser workflows beyond the new School Admin assignment-management screen.
- Exhaustive HTTP coverage with real JWT signature validation for every protected route.
- Production deployment hardening, least-privilege database grants, monitoring/alerting, backup/restore drills, penetration testing and load/capacity testing.
- Parent OTP/SMS delivery and Parent/Driver Phase 2 interfaces remain deferred and are not current work.

## Safe next sequence

Follow the mandatory [Phase 1 School Operations task-by-task build order](../product/PHASE1-WORKFLOW-ORDER.md). The immediate operational sequence is: verify school/academic foundations → complete/verify subject catalogue → create teacher/staff accounts → assign teachers to subjects/classes/sections → create students and enroll them → finish timetable acceptance using real data → complete attendance → examinations/results → notices and dashboard integration.

This sequence does not imply that existing modules must be rebuilt. Inspect current code and close missing implementation or acceptance gaps in place. Subject management already belongs to Academic Structure; do not create a duplicate subject module. Timetable implementation already exists and must be preserved while its real-data browser/role acceptance is completed. Student and teacher profile creation may be prepared in parallel after academic foundations, but downstream workflows must respect the dependency gates in the roadmap.

Before new business-module work, keep the current persistent-session/MFA acceptance task visible and complete its pending local verification. Fix verified integration defects with regression tests, reconcile remaining authorization gaps against actual routes/repositories, and record browser, responsive, accessibility, theme and session acceptance. Parent OTP/SMS and Parent/Driver Phase 2 work remain deferred until Phase 1 school web completion and explicit reprioritization.

## Git and database safety

- Prefer the agreed primary development branch workflow; do not create unnecessary branches or merge a pull request without explicit approval.
- Do not stage unrelated helper files or commit secrets.
- Do not run migrations against persistent databases without explicit approval.
- Keep `.env`, passwords, JWT secrets, TOTP encryption keys and real student data out of Git.
