# Project Status

**Last reconciled:** 2026-11-14  
**Repository:** `TheSumitNain/school-management-system`  
**Reviewed branch:** `work`  
**Reviewed through:** V39 guardian-permission implementation commits on `work`; V32–V38 were locally verified in the user's disposable `school_management_test` database; V39 and subsequent browser acceptance are pending

## Administrator access correction — School Admin guardian permissions (V39; local verification pending)

- Root cause of the missing Parent/Sibling area: the guardian UI and APIs existed, but the School Admin system-role bundle and its runtime allowlist omitted `guardians.read` and `guardians.manage`.
- V39 adds both school-scoped permissions to every existing School Admin system role and updates the school-creation provisioning function so newly onboarded schools receive the same bundle. The immutable system-role protection trigger is removed only around the reviewed migration insert and is recreated immediately.
- Teacher permissions remain unchanged; Teacher system roles receive neither guardian permission. The School Admin runtime permission allowlist and provisioning integration expectations now include both keys.
- Added PostgreSQL integration coverage for existing-school grants, Teacher exclusion and restoration of the system-role immutability trigger. Newly created-school provisioning coverage also expects the updated 53-permission School Admin bundle.
- No guardian/student records are modified. No frontend permission check or backend authorization middleware is bypassed.
- **V39 has been committed to `work` but has not been run against any local database.** Verify it only on the disposable `school_management_test` database first. Do not apply it to the persistent `school_management` database without separate explicit approval and a verified backup.
- **Platform Admin cross-school business operations remain an explicit architecture gap:** Platform Admin is a platform-scoped principal, while current school business APIs require a school-scoped staff context. Do not impersonate a School Admin or bypass tenant checks. Implement a reviewed selected-school administrative context with explicit target-school authorization and audit before exposing Platform Admin to school business records.

## Student Management — approved yearly admission model (V31–V38 schema; application cutover in progress)

- Product decisions are recorded in [Student Management — Domain and Database Design](../product/STUDENT-MANAGEMENT-DESIGN.md): one permanent school-scoped student identity, one enrollment/admission number per academic year, combined initial admission, and independently editable profile sections.
- V31 adds nullable `admission_number` and `admission_date` to `student_enrollments`, a paired-nullability check, a non-empty/length check, and a normalized unique index scoped to school + academic year. Historical enrollments may remain NULL/NULL when their original identity cannot be proven.
- V32 decouples permanent student identity from the legacy school-wide admission number: `students.admission_number` becomes nullable, its school-wide unique index is replaced with a non-unique legacy lookup index, and existing values remain untouched. Its down migration refuses rollback if null or duplicate legacy identities exist.
- The enrollment repository read model now exposes year-specific admission number/date. Attendance projections now read admission number from the enrollment and return an empty value for historical NULL rows. Student-profile reads safely tolerate a NULL legacy number.
- New enrollment and promotion repository writes now require an admission number and admission date, validate both dates against the selected academic year, and write those values with audit metadata. Promotion preserves the prior enrollment and creates the destination-year identity.
- V33 extends the existing enrollment-history trigger so admission number/date are immutable from the moment an enrollment is created, including while active. The existing active-to-terminal transition remains allowed without rewriting identity; a dedicated PostgreSQL integration test covers both rules.
- New student creation now has one transactional repository operation that creates the permanent student identity (legacy admission number NULL), its first year-specific enrollment, and both audit events. The `POST /school/students` route now uses this combined operation; an enrollment uniqueness conflict rolls back the student insert.
- Student directory/profile reads now display and search the current academic year's enrollment admission number. If a student has a current-year enrollment with unknown legacy admission identity, the directory deliberately shows no number rather than incorrectly carrying forward an unverified old-year value. Attendance and enrollment lists also expose the enrollment-level number.
- The Student Management page now uses a guided first-admission form (student name, admission number, academic year, section, optional roll number and admission date) and requires year-specific admission number/date when enrolling an existing student.
- The student directory now opens a profile detail view showing the permanent student identity and year-wise enrollment history, including each year's admission number/date, class/section, roll number and enrollment lifecycle dates. Address & Communication and Parents & Siblings are launched as separate profile-section actions; the directory row no longer carries redundant section shortcuts.
- V34 adds tenant-aware hierarchical locations and separate current/permanent student address records. Backend APIs support location search/add/deactivate and address read/update; the frontend now has an independently editable Address & Communication section with dependent State → District → Block/Tehsil → Village/City selectors and school-owned additions. No guessed national geography catalogue has been seeded.
- V35 adds optional primary phone, alternate phone and email contact details with live read/update permissions, audit metadata that excludes PII values, and a Contact Details tab in the same independently editable profile section. Phone/email values are intentionally non-unique.
- The Parents & Siblings section now reuses the existing V11 guardian model: staff can create guardian contacts, link multiple guardians with relationship/primary/emergency flags, and revoke links with reasons while retaining history. Existing V12 partial uniqueness allows a revoked relationship to be re-established as a new history row. A related-student list is derived from active shared guardian links and explicitly does not claim biological sibling status. Parent OTP/login remains deferred.
- V36 adds optional Medical and Physical profile sections. Medical data includes blood group, allergies, known conditions, ongoing medication notes and emergency medical notes; every edit preserves the previous snapshot in append-only version history. Physical data is a dated append-only measurement history for height/weight and optional vision/physical notes; new measurements never overwrite old ones.
- V37 adds an optional one-to-one Basic Details profile section for date/place of birth, gender, nationality, religion, social category, EWS status and school registration number. It is not required during admission; reads require `students.read_sensitive`, writes require `students.update`, and audit metadata records field-presence flags rather than demographic values. Aadhaar is intentionally excluded. The section is linked from Student Profile and supports view-only access.
- V38 extends school-owned guardian contacts with optional alternate mobile, occupation, education and workplace/organization fields. These details belong to the guardian identity and are shared across linked students; relationship type and primary/emergency flags remain per-student relationship data. Existing guardian rows remain valid with NULL optional fields. Student address remains managed in the separate Address & Communication section; no duplicate guardian address is stored. The user locally verified the V38 migration, full backend test suite, `go vet`, frontend typecheck and production build on 2026-11-14. The V38 down migration refuses rollback once optional guardian profile data exists, preventing silent data loss.
- Medical/physical reads use `students.read_sensitive` and are audited; updates use `students.update` and are audited with version/presence metadata only. Health values and notes are excluded from audit metadata, Teacher access is denied, and sensitive query caching is disabled on the frontend. Documents remain deferred until private storage and document-specific access controls are designed.
- **User-reported local verification (2026-11-14, after pull through `2a5da95`):** with `DB_NAME=school_management_test` and `RUN_POSTGRES_INTEGRATION_TESTS=1`, `go run ./cmd/migrate` applied the full migration chain successfully through V37; `go test -count=1 -p 1 ./...` passed across all backend packages; and the chained `go vet ./...` completed without an error. `GOTMPDIR=D:\\go-test-tmp` was used to avoid Windows temporary test-executable cleanup conflicts. This verifies the disposable test database and backend code only; it does not verify the frontend in a browser.
- **GitHub Actions verification (2026-11-14):** backend CI passed on commit `429985b`, including all migrations on an ephemeral PostgreSQL database, full backend integration tests, and `go vet`; frontend CI passed on commit `7599c3c`, including dependency installation, design-token checks, TypeScript and production build. The V34 backend migration/API passed GitHub CI on commit `13d1a4d` (PostgreSQL migrations/integration tests and `go vet`); the V34 Address & Communication UI passed frontend CI on commit `4900836` (design-token checks, TypeScript and production build). V35 contact-details backend passed GitHub CI on commit `e2e0cac` (migrations, PostgreSQL integration tests and `go vet`); its frontend tab passed on `3375631` (design-token checks, TypeScript and production build). Parents & Siblings backend, shared-guardian query, revocation/re-link history integration tests passed backend CI on `610a6ee`; the complete frontend section passed design-token, TypeScript and production-build CI on `1a86c88`. V36 revised schema and repository passed backend CI on `853b3f8`, including the full migration chain, PostgreSQL integration tests and `go vet`; coverage includes medical version snapshots, append-only physical measurements, audit metadata privacy, database constraints and Teacher access denial. The profile and health frontend passed design-token, TypeScript and production-build CI on `17508c4`, including the version/measurement history UI and duplicate-save protection. User-local V32–V37 migration and backend test verification passed on 2026-11-14; frontend/browser acceptance remains pending.
- V37 Basic Details schema, repository, API routes and UI passed GitHub CI: backend on `def28e6` (full migration chain on ephemeral PostgreSQL, all backend/integration tests and `go vet`), frontend on `f4d53ce` (design-token checks, TypeScript and production build). The V37 integration test verifies optional profile persistence, same-school authorization, normalized registration-number uniqueness, immutable creator/identity, deletion protection and audit metadata that excludes demographic values. User-local V32–V37 migration and backend test verification passed on 2026-11-14; frontend/browser acceptance remains pending.
- **Application cutover remains in verification:** V32–V38 migrations and backend integration tests have passed in the user's local disposable `school_management_test` database; `go vet`, frontend typecheck and production build also passed for commit `9c07309`. Existing guardian profile editing has since been added through the existing `PATCH /school/guardians/:guardianID` route; that latest edit workflow and its follow-up status-preservation changes await a fresh local verification. The end-to-end browser workflow, role-by-role acceptance and document profile sections remain pending. The combined-admission and profile APIs are implemented and covered by backend tests, but the end-to-end browser workflow, frontend production build after this pull, and role-by-role acceptance remain pending. Document profile sections remain future implementation work.
- The persistent `school_management` application database has not been migrated by this work. Do not apply V32 or any pending migration to it without explicit approval and a verified backup.

## Staff Management — employee profile workflow implemented

- V28 adds the separate `staff_profiles` table. Full name, employee code, designation and one mobile number are required; email and joining date are optional. Employee codes are case-insensitively unique within each school.
- V29 makes employee inactivation disable the canonical staff principal, increment its token version and revoke active refresh sessions. V30 prevents any staff-login reactivation unless a complete employee profile exists with Active employment status.
- New staff creation requires employee details and creates the login account plus profile in one database transaction. Employee-code conflicts and profile failures cannot leave orphan login accounts.
- Staff listing includes employee details and a `profile_complete` indicator. Existing staff accounts remain intact and are shown as incomplete without fabricated profile data.
- Employee profiles can be edited independently of login credentials. Inactive employment disables login; returning employment to Active does not automatically enable the account. Explicit enablement with a temporary password requires a complete Active profile.
- The Staff Management UI provides the employee form, field-level validation, employee/profile listing, separate employment/login statuses, and hides login enablement for incomplete or inactive profiles.
- Profile changes do not delete academic assignments, attendance, timetable records or other history. No staff profile deletion operation is provided.
- Backend CI (fresh PostgreSQL migrations, full Go tests including PostgreSQL integration, and `go vet`) and frontend CI (TypeScript and Vite production build) are verified on the latest implementation commit. CI uses an ephemeral database; no persistent local database has been migrated by this work.

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

The frontend already restores a session at app startup by calling the refresh endpoint with the same-origin HttpOnly cookie; this was verified by source inspection, not by a fresh browser close/reopen test. Do not mark the task fully accepted until the user has reviewed and explicitly approved applying the pending migrations (V25–V30 at the time of this reconciliation) to the intended local database after a verified backup, and manually verified browser restart/session restoration, fresh-login MFA, background refresh, cross-tab account/logout behavior and security revocation. Disposable CI migrations do not change the user's local database.

- The refresh cookie is persistent with a 400-day Max-Age and is renewed on each successful refresh. Browser close/reopen and laptop restart restore the session while the browser retains the cookie; browser-enforced retention limits still apply.
- Platform Admin and School Admin must complete mandatory MFA on every fresh login. MFA must not be repeated during background access-token refresh or restoration of an existing valid session.
- The 30-minute administrative idle timeout and server-side absolute refresh-session expiry have been removed. Protected requests and refresh continue to validate the active session family.
- A session remains valid until the user manually logs out or a security/account event requires revocation (for example, password/security reset, account disablement or confirmed compromise).
- Manual logout must revoke the session family. Preserve opaque refresh-token rotation, replay detection/family revocation, transaction/concurrency protections, HttpOnly + Secure cookie policy, cross-tab logout synchronization, principal token-version checks and live database-backed RBAC authorization.
- Backend session lifecycle, persistent-cookie policy, regression/integration tests and backend/frontend security documentation have been updated. Automated browser close/reopen and interactive MFA acceptance remain pending locally.
- Continue development on the agreed `work` branch; keep `main` stable and merge only a completed, verified feature.

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
- Student profile summary is read-only; permanent student name editing is placed in the Basic Details section alongside date of birth, gender and other optional personal details. The former standalone “Edit student name” profile action has been removed; this UI adjustment awaits fresh local verification.

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
- Timetable implementation exists; real-data browser and role acceptance remains pending. Examinations and results have no delivered schema, API routes or frontend workflow yet; see [Examinations and Results — Domain Design Proposal](../product/EXAMINATIONS-RESULTS-DESIGN.md).
- Teacher/staff-specific operational screens and assignment-limited browser workflows beyond the new School Admin assignment-management screen.
- Exhaustive HTTP coverage with real JWT signature validation for every protected route.
- Production deployment hardening, least-privilege database grants, monitoring/alerting, backup/restore drills, penetration testing and load/capacity testing.
- Parent OTP/SMS delivery and Parent/Driver Phase 2 interfaces remain deferred and are not current work.

## Safe next sequence

Follow the mandatory [Phase 1 School Operations task-by-task build order](../product/PHASE1-WORKFLOW-ORDER.md). The immediate operational sequence is: verify school/academic foundations → complete/verify subject catalogue → create teacher/staff accounts → assign teachers to subjects/classes/sections → create students and enroll them → finish timetable acceptance using real data → complete attendance → design and implement examinations/results → notices and dashboard integration.

This sequence does not imply that existing modules must be rebuilt. Inspect current code and close missing implementation or acceptance gaps in place. Subject management already belongs to Academic Structure; do not create a duplicate subject module. Timetable implementation already exists and must be preserved while its real-data browser/role acceptance is completed. Student and teacher profile creation may be prepared in parallel after academic foundations, but downstream workflows must respect the dependency gates in the roadmap.

Before production acceptance, complete the pending persistent-session/MFA browser verification and local database adoption only after explicit approval. For each new business module, reconcile authorization against actual routes/repositories and record browser, responsive, accessibility, theme and session acceptance. Parent OTP/SMS and Parent/Driver Phase 2 work remain deferred until Phase 1 school web completion and explicit reprioritization.

## Git and database safety

- Develop on the single ongoing `work` branch. Keep `main` stable; merge from `work` only after feature completion and verification, then synchronize `work` from `main`.
- Do not stage unrelated helper files or commit secrets.
- Do not run migrations against persistent databases without explicit approval.
- Keep `.env`, passwords, JWT secrets, TOTP encryption keys and real student data out of Git.


## Student profile screen corrections — local verification pending

- The directory Edit action now opens the section-based Student Profile workflow; the legacy name-only edit modal has been removed from the page.
- Address reads now include the saved State, District and Block/Tehsil names in addition to hierarchy IDs. The UI can keep previously selected inactive locations visible instead of rendering an empty lower-level selector.
- Added PostgreSQL integration coverage for full address hierarchy read-back after locality deactivation.
- The design document now records the acceptance checklist and confirms Aadhaar remains intentionally excluded from the default profile; document upload/download remains deferred pending secure private storage and dedicated access controls.
- These changes are committed on `work`; run backend PostgreSQL integration tests, `go vet`, frontend typecheck and production build locally before browser acceptance. Do not apply migrations to the persistent application database for this work.


## Student Basic Details sensitive-read audit — 2026-11-14
- Follow-up audit found Address and Contact Detail GET methods were permission-protected but did not create read-audit events. Both now perform permission check, tenant-scoped read and section-only audit insert in one transaction. User ran both targeted PostgreSQL integration tests, the full backend test suite and go vet ./... against school_management_test; all passed. Windows emitted a post-test temporary executable unlink warning, but the targeted and full test commands returned success.

- Basic Details GET now runs its authorization, student lookup, sensitive-profile read and audit insert in one transaction. A successful read records `school.student.basic_details.read` with the student reference and section name only; profile values are never copied into read-audit metadata.
- Added PostgreSQL integration assertions for the read audit event and for absence of personal values in its metadata. User ran `go test -count=1 -p 1 ./internal/repository -run TestStudentBasicDetailsV37PostgresIntegration`, the full `go test -count=1 -p 1 ./...`, and `go vet ./...` against `school_management_test`; all passed.
- Contact Details and Address now have verified transactional read-audit coverage alongside Basic Details. Medical and Physical detail GET methods already use transactional section-only read audits, with PostgreSQL integration assertions for audit counts and absence of health values. Parents & Siblings guardian/list/link/related-student read-audit changes have now passed the user's targeted PostgreSQL integration test, full backend test suite and go vet ./... against school_management_test.
- Aadhaar remains excluded from the UI/API until its dedicated encryption/key-rotation implementation is complete.

## Student Management reference inventory update — 2026-11-14

- Recorded the complete November 13 MySmartSchool reference field inventory: combined admission/enrollment fields; Basic/Personal (including optional Aadhaar number and SR number); Address & Communication; Parents & Siblings; Medical; Physical; Class & Subject; Transport; App Permissions; Documents.
- Aadhaar card image/PDF and other document uploads remain deferred. Aadhaar number itself is intended as an optional field only; implementation is blocked until dedicated application-level encryption/key rotation, masked responses, least-privilege access and audit behavior are implemented. Never store or return the full number in plaintext; Aadhaar must not be mandatory for admission.
- Fixed the stale `updateStudent` reference in StudentManagementPage's busy state after removal of the legacy name-only edit modal. Local frontend typecheck/build must be rerun after pulling this fix.


## Student Parents & Siblings read-audit follow-up — 2026-11-14

- Review confirmed the Parents & Siblings repository read methods were permission-protected and school-scoped but did not create read-audit events.
- Added transactional read audits for guardian directory lists, individual guardian reads, a student's guardian links/history, and students sharing a guardian. Audit metadata contains only section and result count; it excludes guardian names, phone numbers, email addresses and relationship/contact values.
- Added PostgreSQL integration assertions to verify the four read-action families and that guardian contact values do not appear in audit metadata.
- User ran TestGuardianRepositoryPostgresIntegration, the full backend test suite and go vet ./... against school_management_test; all passed. Windows emitted a temporary executable unlink warning after the targeted test, but the test command itself returned success.


## Student directory/profile read-audit follow-up — 2026-11-14

- Review found ListStudents and GetStudent were school-scoped and permission-protected but did not emit read-audit events.
- Added transactional audit events for student directory reads (section and result count only) and individual student profile reads (section only). The list count query, result query, authorization and audit insert share one transaction.
- Added PostgreSQL integration assertions that both read actions are recorded and that student names/admission numbers do not appear in read-audit metadata.
- User verified `TestStudentProfileRepositoryPostgresIntegration`, the full backend test suite (`go test -count=1 -p 1 ./...`) and `go vet ./...` against `school_management_test`; all passed. A Windows temporary executable unlink warning appeared after the targeted test, but the test itself returned success.

## Student create-event audit data minimization — targeted verification passed

- Review found that the student-created audit event stored the raw admission number in JSON metadata. Changed it to record only the boolean `has_admission_number`; the student ID remains the audit resource reference.
- Added a PostgreSQL integration assertion that the create event exists, records the presence flag, and contains neither the student's name nor admission number.
- User pulled commit `69bf4d7` and ran `go test -count=1 -p 1 ./internal/repository -run "^TestStudentProfileRepositoryPostgresIntegration$"` against `school_management_test`; the test passed. Windows emitted a post-test temporary executable unlink warning, but the test command itself returned success.
- **Follow-up full verification (user-reported, 2026-11-14):** after pulling through the create-audit argument fix and status update, the user ran `go test -count=1 -p 1 ./...` with `DB_NAME=school_management_test` and `RUN_POSTGRES_INTEGRATION_TESTS=1`; every backend package passed, including repository and services. `go vet ./...` also completed without errors. Windows printed a post-test `services.test.exe` unlink warning, but the full test command returned success. This verifies the disposable test database and backend code only; it does not verify browser workflows or migrate the persistent `school_management` database.
