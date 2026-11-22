## Staff Attendance — core register verified (2026-11-22)

- Audited the existing student attendance module and kept it separate; staff attendance uses its own table, routes, API client and Staff Management workspace tab.
- Added V58 migration files for `staff_attendance_records`, same-school composite foreign keys, one-record-per-staff/date uniqueness, Asia/Kolkata future-date trigger, and four dedicated non-delegable `staff.attendance.*` permissions. School Admin grants are applied to existing schools through the protected nested-trigger path and to future schools after V13 role provisioning. Teacher/student attendance grants are unchanged.
- Added daily roster, monthly derived summary, mark and update repository/service/handler APIs. Writes and audit events are transactional; past dates require the dedicated correction permission; duplicate marks conflict rather than overwrite; disabled login does not erase active employment attendance eligibility.
- Added Staff Management daily/monthly UI with shared date picker, permission-aware actions, field-level status requirement, no-op update prevention, correction confirmation, loading/error/empty states and responsive table layout.
- Added migration/permission/date-trigger integration coverage, route permission tests, service validation tests, daily/monthly/audit/cross-school repository integration coverage and Playwright daily-mark/month-summary coverage.
- Frontend CI **PASS** — [run 44979930421](https://github.com/TheSumitNain/school-management-system/actions/runs/44979930421), checkpoint `32df5ead`: TypeScript/Vite production build, centralized design-token check and all **49 Playwright browser tests** passed, including staff search/status filtering, inline validation, individual marking, atomic bulk marking and monthly summary.
- Backend CI **PASS** — [run 44979930424](https://github.com/TheSumitNain/school-management-system/actions/runs/44979930424), same checkpoint `32df5ead`: complete migration chain including V58 applied to ephemeral PostgreSQL, full Go test suite with PostgreSQL integration tests (including bulk rollback and date guard), and `go vet ./...` passed.
- Phase 3 remains open only for downloadable/report views and approved leave/school-calendar integration. The verified scope includes employee search/status filtering, transactional bulk marking, individual marking/correction, monthly derived summary and audit trail.
- V58 has **not** been applied to any local or persistent database. No local migration was run. Academic, Student Management and Settings pages remain untouched.

---

## Staff Management — record edit workflow checkpoint (2026-11-21)

- Added audited PATCH operations for staff qualifications, certifications and employment experience. Each write re-checks live `users.update` authorization, derives school scope from the authenticated context, restricts the target by record + staff + school IDs, preserves verification state, and commits the update with its audit event in one transaction.
- Added matching prefilled Edit forms in the staff profile. Existing validation is applied to edits; API failures leave entered values available for correction. Playwright coverage edits all three record types and checks the updated values.
- Frontend CI **PASS** — [run 44918115850](https://github.com/TheSumitNain/school-management-system/actions/runs/44918115850): TypeScript/Vite build and all **46 browser acceptance tests** passed, including qualification, certification and experience edits. The tested frontend checkpoint is `a8c6b3e0`; later branch changes in this checkpoint were backend-only.
- Backend CI **PASS** — [run 44918463838](https://github.com/TheSumitNain/school-management-system/actions/runs/44918463838), code checkpoint `66339b2b`: full migration chain applied to disposable PostgreSQL, full Go test suite including create/edit/audit/cross-school integration tests, and `go vet ./...` passed.
- Verification actions remain pending: the permission catalogue does not currently contain the proposed dedicated `staff.qualifications.verify` permission. Do not reuse general `users.update` as verification authority. Staff Documents remain blocked pending private storage and authenticated download authorization.
- V57 was applied only to the disposable CI database; no persistent/local database migration was run. Academic, Student Management and Settings pages remain untouched.

---

## Staff Management — qualification records and wizard verification checkpoint (2026-11-21)

- Backend implementation now includes school-scoped create/list APIs for qualifications, certifications and prior experience, with request validation, same-school staff checks and audit events. PostgreSQL integration coverage verifies create/list, audit records and cross-school denial.
- Backend CI on `0175725f` passed: the complete migration chain (including V57) applied to the disposable PostgreSQL 18 CI database, `go test -count=1 -p 1 ./...` passed, and `go vet ./...` passed. This does not apply V57 to any user/local/persistent database.
- Frontend TypeScript, Vite production build and design-token checks passed on `0175725f`. Browser acceptance had 45 passing tests and one failing staff wizard date-picker scenario: after selecting Today, the wizard remained on Employment when the test attempted to continue. The latest change `a9227c8` makes Continue an explicit button action rather than a form submit; its CI run is pending.
- Qualifications & Experience UI currently supports creating/listing records; edit, verification and archive/delete workflows are not complete. Staff Documents remain blocked pending an approved private storage adapter. Staff-scoped academic assignments and History & Activity also remain pending integration.
- The one-time credential handoff is manual copy / do later only; no credentials are auto-sent and plaintext passwords are not persisted.
- Academic, Student Management and Settings pages have not been changed. No migration has been run against the user's local or persistent databases.

## Staff Management — first records API CI checkpoint (2026-11-21)

- Backend CI on `aa0fe50` passed: V57 applied successfully to the disposable CI PostgreSQL database, full `go test -count=1 -p 1 ./...` passed, and `go vet ./...` passed.
- Frontend TypeScript and Vite production build on `aa0fe50` passed, including the design-token check.
- The full browser suite on that commit had 41 passing tests and 4 failing Staff Management tests. Failures exposed stale profile-tab assertions, ambiguous Add Staff selectors, a missing directory navigation in one test, and an Add Staff wizard progression issue. These are being corrected; browser acceptance is NOT green yet.
- Added a dedicated PostgreSQL integration test source for qualification/certification/experience create-list, transactional audit and cross-school denial. It still needs to run in CI.
- The current implementation remains create/list only. Edit, verification, archive/delete and document storage are not complete. V57 has not been applied to the user's local or persistent databases.
- Academic, Student Management and Settings product pages remain untouched.

## Staff Management — Add Staff wizard and employee profile shell (2026-11-21)

- Replaced the single-step Add Employee modal flow with the roadmap's four steps: Staff Identity, Employment Details, Login & Access, and Review & Create. Step transitions validate the visible step; server-side validation remains mapped to the relevant fields on final submission.
- The wizard currently exposes only fields supported by the existing create API: full name, employee code, mobile number, optional email, designation, optional joining date, globally unique username and initial password. New records retain the backend's current Active employment default; no unsupported staff-type, department, profile-only or role-selection fields are fabricated.
- Added a final review summary using the entered values. The employee and staff login are still created together through the existing transactional endpoint.
- After successful account creation, an optional one-time credential handoff appears with an explicit Copy credentials / Do this later choice. Nothing is auto-sent; the password is held only in transient component state for this handoff and cleared when the dialog closes. No password is persisted or logged by this UI.
- Added the eight-tab employee profile shell: Overview, Personal Details, Employment, Qualifications & Experience, Documents, Academic Assignments, Account & Access, and History & Activity. Current profile and employment fields are read from the existing staff record; edit actions continue through the existing validated update endpoint. Role assignment and permission overrides reuse the existing server-authorized components.
- Qualifications, Documents, staff-scoped Academic Assignments and History & Activity are intentionally marked as pending integration. No fake saves, document uploads or fabricated event history are shown. Document upload remains blocked until private storage and resource-specific access checks exist.
- Added Playwright coverage for the four-step create flow, review values, one-time handoff, and profile tabs. Tests are committed but not executed in this environment. Latest commit has no reported GitHub status checks; frontend typecheck, production build and browser run remain pending.
- No backend changes, database migrations, local database changes or edits to Academic, Student Management or Settings were made.

## Staff Management — workspace rebuild started (2026-11-21)

- Completed a source-level Phase 0 audit and recorded the Keep / Extend / Missing / Needs verification map in [Staff Management — Existing Implementation Audit](../product/STAFF-MANAGEMENT-IMPLEMENTATION-AUDIT.md).
- Replaced the old single-table page header with a new Staff Management workspace navigation and a data-backed Overview tab. Overview counts come from the existing school staff API; while the query is loading, counts display as a neutral placeholder rather than false zeroes.
- Added the Staff Directory navigation while retaining the existing employee list, profile edit, account lifecycle, password reset, role assignment and permission-override behavior.
- Connected the Academic Assignments tab to the existing Teacher Assignments page and its existing school-scoped APIs. The tab is shown only when the viewer has teacher-assignment or class-in-charge read permission. The existing Academic Structure page and its data model were not modified.
- Attendance, Leave, Workload & Timetable and Reports remain visible as disabled Phase 1 entries; no fake screens or sample operational data were added.
- Added Playwright coverage for the overview metrics, phase-gated navigation and directory navigation. These tests are committed but have not yet been executed in CI or in the user's local browser. The latest commit has no reported status checks yet.
- This is only the first workspace-shell milestone. The four-step add-staff wizard, eight-tab staff profile, profile-only creation support, credential handoff, qualifications, documents and lifecycle history remain incomplete. No backend code or database migration was changed.

## Staff Management — shared control consistency and browser regression coverage (2026-11-21)

- Replaced the Staff Management joining-date native browser control with the shared IST-aware DatePicker, preserving the optional date-only API value.
- Replaced the employment-status native select with the shared accessible Select and retained its value in the existing form submission contract; field-level server validation remains attached to the control.
- Clarified the create action label, edit action accessible name, and staff search description so they match the actual operations and searchable fields.
- Added Playwright coverage for creating an employee with the shared date picker and editing employment status through the shared Select. This test coverage is committed but has not yet been executed in the user's local browser.
- Frontend typecheck, production build and Playwright execution are pending verification on the updated branch. No backend code, database migration or local school data was changed.

## Local database migration checkpoint — V56 applied (2026-11-21)

- Applied V56 successfully to the local test database `school_management_test` and application database `school_management`; both were independently checked at schema version `56`, `dirty=false`.
- Before applying the migration, created fresh V55 backups: `school_management_v55_20261121_133949.backup` and `school_management_test_v55_20261121_133949.backup`.
- Both backups were restore-tested in separate temporary databases; each restored to V55 with `dirty=false`. Temporary verification databases were dropped after successful checks. Original databases and backup files were not altered.
- V56 restores append-only curriculum/student subject-selection history. Keep revoked selections and soft-deactivated curriculum subjects; do not downgrade once either kind of history exists.

## Platform Admin — read-only audit history (2026-11-21)

- Added `GET /platform/audit-events?limit=50&offset=0`, protected by the live `platform.audit.read` permission and a second repository-level active Platform Admin/system-role permission check.
- The endpoint returns only `platform.*` audit actions initiated by Platform Admin principals or the system bootstrap actor. It includes actor email and school name/code where applicable, but deliberately omits raw audit metadata to avoid exposing unrelated or sensitive payload values.
- Pagination is bounded to 1–100 events per request. The Platform Admin workspace now includes an Audit history tab with loading, empty, retry, outcome, school context and previous/next states. Audit-only Platform Admin accounts can reach the workspace through permission-aware navigation.
- Backend CI **PASS** — [run 44882670480](https://github.com/TheSumitNain/school-management-system/actions/runs/44882670480): complete migration chain, Go tests including PostgreSQL integration, and `go vet`; this run includes the route-level live `platform.audit.read` permission regression test.
- Frontend CI **PASS** — [run 44882457119](https://github.com/TheSumitNain/school-management-system/actions/runs/44882457119): TypeScript/Vite production build and the complete Playwright browser suite, including Platform Admin account lifecycle, school onboarding/delegation and audit-history pagination. Frontend code was unchanged in the later backend-only test commit.
- Verified code checkpoint: `ca7feddf2f1fb04dd3dbb9b493d457c2a00e5b23`. No database migration or local school-data change was made. Authenticated local browser acceptance remains pending.

## Roles & Permissions — page verification checkpoint (2026-11-21)

- School-specific custom roles can be created with field-level validation, reviewed, and assigned only permissions covered by active Platform Admin delegations that permit role assignment.
- Permission grants show whether the grant and underlying permission remain active. Removing a grant requires explicit confirmation; failed create/grant/revoke operations remain recoverable and preserve the form context.
- Built-in system roles are protected from custom-role editing. Read-only users can review role grants but cannot create roles, grant/revoke permissions, or trigger delegation-management requests.
- Frontend CI **PASS** — [run 44871907536](https://github.com/TheSumitNain/school-management-system/actions/runs/44871907536), checkpoint `3985db10d8b7eab2d35a49e323ddf5c3fd7b4b39`: TypeScript/Vite build and all **34 Playwright browser acceptance tests** passed, including custom-role lifecycle, delegated grant/revoke recovery, delegation-load retry, and read-only access.
- Backend CI **PASS** — [run 44871612871](https://github.com/TheSumitNain/school-management-system/actions/runs/44871612871), backend code checkpoint `31bf1bc7ddf33d5c4a886393aaf48722613ac2f4`: full Go tests, PostgreSQL integration tests, and `go vet ./...` passed. No backend code changed after that checkpoint.
- Local authenticated browser acceptance remains for the user; no local database migration or school-data change was made for this page.

## Dashboard — page verification checkpoint (2026-11-21)

- Dashboard metrics are permission-scoped: staff accounts, classes, academic years, and current-year enrollments are requested only when the signed-in principal has the corresponding permissions.
- Current-year enrollment totals are loaded against the active current academic year; when no current year is configured, the dashboard explains why the metric is unavailable. Partial API failures are surfaced without hiding successful metrics.
- Dedicated Playwright coverage verifies all four metric totals and confirms that the enrollment request uses the selected current academic-year ID. Existing responsive coverage checks dashboard and workspace navigation across mobile, tablet, and desktop widths.
- Frontend CI **PASS** — [run 44871612842](https://github.com/TheSumitNain/school-management-system/actions/runs/44871612842), checkpoint `31bf1bc7ddf33d5c4a886393aaf48722613ac2f4`: TypeScript/Vite build and all **33 Playwright browser acceptance tests** passed.
- Backend CI **PASS** — [run 44871612871](https://github.com/TheSumitNain/school-management-system/actions/runs/44871612871), same code checkpoint: full Go tests, PostgreSQL integration tests, and `go vet ./...` passed.
- Local authenticated browser acceptance remains for the user; no local database migration or school-data change was made for this page.

## Settings / Appearance — page verification checkpoint (2026-11-21)

- Appearance preferences include Light, Dark, System, and Automatic schedule modes, with 15 coordinated brand palettes. Preferences are stored locally on this device; no school database setting is changed.
- Automatic scheduling validates distinct Dark/Light start times, uses the device's local timezone, and continues updating the application theme while the Settings page is closed. System mode responds to operating-system appearance changes without reloading.
- Responsive acceptance covers narrow mobile, tablet, desktop, and landscape widths; mode and palette controls remain within the viewport. The schedule form exposes inline validation and accessible error associations.
- Frontend CI **PASS** — [run 44871312360](https://github.com/TheSumitNain/school-management-system/actions/runs/44871312360), checkpoint `0f77d005b5badca9bc177d8a0832335758d8a268`: TypeScript/Vite build and all **32 Playwright browser acceptance tests** passed, including live system-mode changes and automatic schedule transition after leaving Settings.
- Backend CI **PASS** — [run 44871312315](https://github.com/TheSumitNain/school-management-system/actions/runs/44871312315), same code checkpoint: full Go tests, PostgreSQL integration tests, and `go vet ./...` passed.
- Local manual browser acceptance remains for the user. Settings changes are device-local and require no database migration.

## Student Management — page verification checkpoint (2026-11-21)

- Student directory supports search, academic-year and status filters, pagination, new admission, profile editing and safe archive/restore-after-lost-response behavior.
- Student profile workflows cover Basic Details, Parent Details / guardian relationships and shared guardian profiles, address and contact details, medical and physical details, enrollment history, and curriculum/optional-subject selections with permission-aware access.
- The Enrollments view supports year filtering, paginated enrollment records, new enrollment, and promotion while preserving prior enrollment history and recorded attendance boundaries.
- Frontend CI **PASS** — [run 44870000404](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000404), checkpoint `ad383869f1af8cbfca0e069a752c8f79fa79cf07`: TypeScript/Vite build and all **32 Playwright browser acceptance tests** passed, including Parent Details draft isolation, enrollment, promotion, archive recovery, and mobile layout coverage.
- Backend CI **PASS** — [run 44870000355](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000355), same code checkpoint: full Go tests, PostgreSQL integration tests, and `go vet ./...` passed.
- Local authenticated browser/API/database acceptance remains for the user to perform. No local database migration or school-data change was made in this checkpoint; verify a restorable backup before applying any pending migrations.

## Class Curriculum removal review — verified checkpoint (2026-11-21)

- Before a subject is removed from a section curriculum, the preview shows the exact affected-student count and the affected students' names and admission numbers.
- The confirmation request includes the reviewed student IDs. The backend recomputes the affected set in the save transaction and rejects a stale confirmation even when the total count is unchanged but the student membership differs.
- Explicit confirmation is required before the curriculum subject and affected students' current optional choices are removed. Cancel closes the preview without issuing a save request. The confirmed write and its audit event remain transactional; historical curriculum and selection behavior is documented separately from current selections.
- PostgreSQL integration tests cover affected-student details for optional and compulsory subjects and rejection of a same-count/different-student confirmation. Playwright coverage verifies the impact count, names, admission numbers, cancel-without-write behavior, and confirmation payload.
- Frontend CI **PASS** — [run 44870000404](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000404), checkpoint `ad383869f1af8cbfca0e069a752c8f79fa79cf07`: TypeScript/Vite production build and all browser acceptance tests passed.
- Backend CI **PASS** — [run 44870000355](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000355), checkpoint `ad383869f1af8cbfca0e069a752c8f79fa79cf07`: full Go tests including PostgreSQL integration and `go vet ./...` passed.
- No local database migration or live school data change was performed as part of this checkpoint. Verify the local backup and apply any pending repository migrations before manual testing.

## Academic Structure lifecycle safety — verified checkpoint (2026-11-21)

- Academic years close/archive; they are not hard-deleted. Closed years remain readable for historical records, and the current-year flag is cleared when a year is closed.
- Class, section and subject deactivation now each has a school-scoped dependency-impact preview, explicit confirmation UI, a fingerprinted confirmation request, and a transactional backend write that locks the relevant parent/dependent records, rechecks the fingerprint, rejects stale confirmations, changes only the master record's active state, and writes the audit event in the same transaction.
- Section preview includes active/historical enrollments, active section curricula, active teacher-subject and class-in-charge assignments, and affected students. Class preview includes active/inactive sections, active/historical enrollments, active curricula, active assignments and affected students with section/year context. Subject preview includes curriculum references, active teacher assignments, active/historical student selections and affected students with class/section/year context.
- Direct API attempts to deactivate active classes/sections/subjects without a reviewed fingerprint are rejected. Confirmed deactivation does not silently delete or cascade student enrollments, curriculum references, subject choices, sections or assignments.
- Subject deactivation is coordinated with curriculum and student-selection writes: inactive catalogue subjects are not offered for new curriculum/teacher-assignment use, while already assigned inactive subject choices remain visible and are preserved rather than silently cleared. The Student Profile shows the inactive-catalogue status without mobile-width overflow.
- Added PostgreSQL integration coverage for section/class/subject stale-fingerprint rejection and refreshed confirmation, retention of linked records, and transactional audit events. Added browser acceptance coverage for all three deactivation previews, cancellation without writes, explicit confirmation and the academic-year archive flow.
- Frontend CI **PASS** — [run 44863350595](https://github.com/TheSumitNain/school-management-system/actions/runs/44863350595), checkpoint `f42afc2ddbcd51aae7595c03b0f54efbc99dd1d7`: TypeScript/Vite build and all 26 Playwright browser acceptance tests passed.
- Backend CI **PASS** — [run 44863685108](https://github.com/TheSumitNain/school-management-system/actions/runs/44863685108), checkpoint `8543983c132b78818531537471bec6bea06c507e`: full Go tests including PostgreSQL integration and `go vet` passed.
- No migration or live/local database data change was made for these UI flows. The requested Class 4 A/B-only configuration has not been applied to a live school database; any existing extra section must be reviewed and deactivated through this preview/confirmation flow, not removed by an unreviewed data change.
- Timetable, Staff and Attendance pages were not modified as part of this work.

## Subject deactivation dependency preview checkpoint (2026-11-21)

- Added a school-scoped `GET /school/subjects/:subjectID/deactivation-impact` endpoint protected by the live `subjects.manage` permission.
- The preview reports curriculum reference count, active teacher-subject assignments, active and historical student subject selections, and affected students with admission number, class, section and academic year.
- The Subject editor now fetches the impact before any active-to-inactive write, shows the exact counts and affected student list, supports returning to edit without saving, and submits the reviewed fingerprint only after explicit confirmation.
- The backend blocks direct unconfirmed subject inactivation. The confirmed transaction locks the subject and its curriculum references, teacher assignments and student selections, recomputes a fingerprint, rejects stale confirmation, changes only the master subject's active state, and writes the audit event in the same transaction. No curriculum rows, assignments or student choices are silently deleted or revoked.
- Inactive master subjects cannot be added to new curriculum edits, receive new teacher-subject assignments, or be newly selected by students. New enrollment curriculum assignment is blocked while an active curriculum still contains an inactive master subject. Existing assigned students retain their current curriculum snapshots and already-selected inactive optional choices; the Student panel marks these records and allows removal, while the Curriculum editor keeps inactive referenced subjects visible for the separate student-impact confirmation flow. Curriculum edits, teacher assignment creation and student enrollment choice writes take shared locks on master subject rows to serialize them against deactivation.
- Added PostgreSQL integration coverage for preview, stale-fingerprint rejection after an assignment change, successful refreshed confirmation, curriculum-reference retention and audit persistence. Added a handler conflict test and a Playwright subject confirmation/cancel flow.
- Frontend CI **PASS** — [run 44863350595](https://github.com/TheSumitNain/school-management-system/actions/runs/44863350595), code checkpoint `f42afc2ddbcd51aae7595c03b0f54efbc99dd1d7`: TypeScript/Vite production build, design-token check and all **26 Playwright browser acceptance tests** passed.
- Backend CI **PASS** — [run 44863685108](https://github.com/TheSumitNain/school-management-system/actions/runs/44863685108), code checkpoint `8543983c132b78818531537471bec6bea06c507e`: full Go test suite, PostgreSQL 18 integration tests and `go vet ./...` passed.
- No migration or persistent local database change was made.

## Section deactivation dependency preview checkpoint (2026-11-21)

- Added a school-scoped, permission-checked API endpoint `GET /school/sections/:sectionID/deactivation-impact`.
- The preview reports active and historical enrollment totals, active section curricula, active teacher-subject assignments, active class-in-charge assignments, and the list of active students with admission numbers.
- The Academic Structure UI now fetches this preview before deactivation, shows the counts and student list in a confirmation dialog, and offers a safe return-to-edit path. It does not transfer students or revoke assignments automatically.
- Added Playwright coverage for preview-before-write, visible impact details, cancel-without-write, and explicit confirmation.
- Added focused styling for the impact review and documented the preview requirements.
- The final deactivation write now locks the section and all dependent enrollment, curriculum, teacher-assignment and class-in-charge rows, recomputes a dependency fingerprint, rejects stale confirmations, and writes the section state change plus audit event in one transaction. Direct attempts to deactivate an active section without a confirmed preview are rejected.
- PostgreSQL integration coverage verifies that a dependency added after preview invalidates the old confirmation, a refreshed preview succeeds, the enrollment is retained, and the audit event is recorded. Backend CI passed on checkpoint `71f78e0fdb3efcd215dc7185f33784b2d4e36502`; frontend build and all browser acceptance tests passed on checkpoint `58a8df4637960379d89fe2e61260ea9bf92eca64`.
- Class and subject deactivation are now also covered by their own impact previews and transaction guards; see the combined verified lifecycle checkpoint at the top of this document.

## Academic Structure management UX checkpoint (2026-11-21)

- Updated `AcademicStructurePage` so backend `field_errors` are mapped to the matching inline form fields while preserving entered values; the form-level server message and error toast remain available for recovery.
- Success notifications now identify the actual academic record and action (create/update/close/archive) rather than showing a generic “saved” message.
- Academic-year closure is presented as archive/retention, not deletion. Selecting Closed automatically clears the current-year flag; closed years remain visible for historical references and their normal edit action is disabled.
- Class, section and subject inactive-state helper text explains that the records and existing student/curriculum history are retained; no hard-delete operation was added.
- Added Playwright coverage for class creation success, subject server-field-error mapping and preserved input, and academic-year close/archive behavior.
- Added the safe academic-structure lifecycle rules to `docs/product/REQUIREMENTS.md`: no hard-delete for referenced records, dependency impact review, explicit confirmation, transactional stale-impact rechecks and audit logging are required before consequential inactivation/removal.
- Frontend CI **PASS** — [run 44859591969](https://github.com/TheSumitNain/school-management-system/actions/runs/44859591969), checkpoint `5cebd996c01986d91cfe41d5ed2e0f34d90ca182`: TypeScript/Vite production build and all 23 Playwright browser acceptance tests passed.
- Backend CI **PASS** — [run 44859592026](https://github.com/TheSumitNain/school-management-system/actions/runs/44859592026), same checkpoint: complete migration chain, Go tests including PostgreSQL integration, and `go vet` passed.
- No migration or local database changes were made in this checkpoint.
- The remaining items in this historical checkpoint were completed in the combined verified lifecycle checkpoint at the top of this document. Do not hard-delete academic years, classes, sections or subjects.

## Editable section-wise Class Curriculum (2026-11-20)

- Approved product decision: one current curriculum per Academic Year + Class + Section. The School Admin edits that same record; normal curriculum changes do not create V2/V3 versions or reassign student enrollments.
- Class 11 Art, Commerce and Science remain independent section curricula. Compulsory, standalone Optional and Optional Group subject types remain supported.
- V54 introduced safe in-place editing and soft-deactivation with retained selection history. V55 temporarily changed this to hard deletion/current-state replacement; V56 restores the safer V54 policy. Confirmed subject removals now revoke active choices and soft-deactivate curriculum subjects, retaining their linked historical rows.
- Before removal, the UI requests an impact preview. The backend counts active students who selected an optional subject, or all active students in the section who are unassigned or assigned to this curriculum for a compulsory subject. The save request must include the reviewed count; the backend recalculates it under the curriculum lock and rejects stale/missing confirmation.
- Confirmed removals revoke linked active optional selections and soft-deactivate the curriculum subject in the same transaction; historical rows and snapshots remain retained. The student remains assigned to the same curriculum record; choices for retained subjects are preserved. Removing an optional group converts its subjects to standalone optional choices and preserves their current selections; the empty group row is deleted.
- Group-limit edits are rejected if they would invalidate existing active student choices. Changing a subject's optional/group classification is rejected while students have active selections for that subject.
- Backend CI **PASS** — run [44875478815](https://github.com/TheSumitNain/school-management-system/actions/runs/44875478815). The full migration chain through V56, Go tests with PostgreSQL integration, and `go vet` passed, including append-only student-selection history, confirmed soft-deactivation, and safe optional-group cleanup.
- Frontend CI **PASS** — run [44875478781](https://github.com/TheSumitNain/school-management-system/actions/runs/44875478781). TypeScript/Vite production build and all Playwright browser acceptance tests passed, including the Class Curriculum confirmation flow and platform-school delegation onboarding/grant/revoke flow.
- Remaining: pull the verified `work` branch locally, inspect/preserve existing uncommitted changes, create and verify fresh restorable backups of both local databases, apply the pending V52–V56 migrations to test and app databases only after explicit review, restart the API, then perform authenticated local browser acceptance. Local databases remain untouched until the user runs the reviewed migration steps.

## Class curriculum subject types and editor styling (2026-11-19)

- In progress: Class Curriculum now has an explicit three-way subject type in the editor: Compulsory, standalone Optional, and Optional Group. A grouped subject is also optional; only grouped subjects participate in the group's min/max choice limits. Standalone optional subjects can be selected independently per student.
- Added additive/reversible V52 migration. Existing grouped subjects are backfilled as optional, existing ungrouped subjects remain compulsory, and the database selection guard now permits standalone optional choices while still rejecting compulsory choices. A compatibility trigger marks legacy/direct inserts with a group as optional.
- Backend curriculum read/write and enrollment-selection validation now use the explicit optional flag. Added a PostgreSQL integration scenario for standalone optional selection and adjusted Parent registration fixtures to respect V51 school-scoped guardian contact uniqueness (same mobile remains valid in different schools).
- Updated the Class Curriculum UI selector and responsive panel styling. Frontend E2E now exercises compulsory, standalone optional and grouped optional subjects.
- Backend CI **PASS** — latest checkpoint `fd5e9c85b6135fcf2e6c6ed328fe145cc3be7601`: complete migration chain through V52, Go tests including PostgreSQL integration, and `go vet`.
- Frontend TypeScript/Vite production build **PASS** at checkpoint `fd5e9c85b6135fcf2e6c6ed328fe145cc3be7601`. Playwright browser acceptance did not start because Ubuntu package mirrors returned HTTP 503 while installing Chromium system dependencies; this was an infrastructure/package-download failure, not a browser-test assertion failure.
- Updated frontend CI to configure APT retries before Playwright dependency installation. New CI run is executing on `4ef24885244afa6e59e1fa440c1d4aff4cf3a7d3`; browser tests must pass before treating the frontend work as verified.
- Follow-up UI work: Student Enrollment now displays standalone Optional subjects as individual checkboxes (previously only grouped options were rendered), with clear compulsory/optional/group sections and responsive styles. Class Curriculum now summarizes compulsory, individual optional and grouped counts and shows subject classifications in saved-version cards.
- Added/updated Playwright coverage for standalone optional selection, saving individual optional and grouped choices together, and saved curriculum subject classifications. The frontend API payload type explicitly includes `is_optional`. Latest code checkpoint `1d85e746`: Backend CI passed; Frontend TypeScript/Vite build passed, but Chromium setup failed before E2E because Ubuntu package mirrors returned HTTP 503 even after retries. Updated frontend CI to install the pinned Playwright Chromium browser without invoking apt dependency downloads; browser E2E will run on the next checkpoint.
- Next: confirm Frontend CI browser tests, then pull `work`, inspect local changes before merging, run local typecheck/build and apply V52 to the already-restored local app/test databases. Finally verify curriculum save and student-level compulsory/individual/group selection in the authenticated School Admin browser flow before entering dummy curriculum data.

## Guardian contact identity and uniqueness (2026-11-19)

- Approved rule: guardian full names may repeat; mobile number and email address are independently optional but must each be unique within the same school when supplied.
- Duplicate mobile/email saves are rejected on both create and shared-profile edit. The matching field receives an inline error: use another contact address or leave that field blank.
- Mobile comparison removes common display formatting and treats Indian 10-digit, leading-zero and +91-prefixed representations as the same number, while preserving the entered display value. Email comparison is trimmed and case-insensitive.
- PostgreSQL V51 adds school-scoped partial unique indexes and a preflight. It does not merge/delete records; if existing duplicates are found, migration stops and requires manual review. The same contact may exist in separate schools.
- Alternate mobile is not a login identifier and is excluded from the uniqueness rule.
- Future Parent Login reuses the existing guardian mobile/email; no separate contact details are collected. Contact ownership must be verified, and a future School Admin setting controls whether parent login is allowed. Parent OTP/SMS remains deferred.
- V51 migration, backend uniqueness checks, HTTP field errors, frontend create/edit inline feedback, regression tests and product/backend/frontend documentation have been committed to `work`. Do not apply V51 to a persistent local database until duplicate preflight is reviewed and a verified restorable backup exists. Backend/frontend tests and CI verification are still pending.

## Guardian relationship uniqueness and legacy duplicate handling (2026-11-19)

- Approved exactly eight relationship types: Father, Mother, Grandfather, Grandmother, Brother, Sister, Uncle and Aunt.
- Each student may have at most one active guardian link for a given relationship type. The same relationship type remains valid for different students.
- Frontend disables relationship types already active for the selected student. While editing a link, the current link is excluded from duplicate detection so its existing relationship remains selectable when it is not duplicated by another active link.
- Legacy relationship labels are compared and displayed case-insensitively after trimming whitespace (for example, `" father "` displays and behaves as `Father`). New backend writes are normalized to canonical title case.
- Backend validates the allowlist and maps database duplicate-relationship rejection to HTTP 409. The V50 trigger serializes writes on the student row and rejects a second active link of the same relationship type, including writes that bypass the repository.
- Existing duplicate active rows are deliberately retained for human review; no guardian link is automatically deleted or revoked by the application. The previously identified local dummy-data duplicate was manually reviewed: the correct Father link was retained and the duplicate link was soft-revoked by the user with an active same-school staff actor and reason, preserving revocation history. This resolves that identified pair only; any other legacy duplicates still require individual review.
- Added PostgreSQL integration coverage for duplicate inserts and edits, handler coverage for the HTTP 409 response, and Playwright coverage for legacy whitespace/case normalization, all eight choices, disabled duplicate options, and keeping the current relationship selectable in Edit.
- Backend CI **PASS** — [run 44448918406](https://github.com/TheSumitNain/school-management-system/actions/runs/44448918406), checkpoint `14972b8de6c4a96537016c5eac8592ab9d682d42`.
- Frontend CI **PASS** — [run 44448918434](https://github.com/TheSumitNain/school-management-system/actions/runs/44448918434), checkpoint `14972b8de6c4a96537016c5eac8592ab9d682d42`. The edit-mode browser assertion now checks the rendered `Father (already linked)` label while retaining the disabled and selected-state checks.

- Local app and test databases were migrated through V50 and the user verified full Go tests, `go vet`, frontend typecheck and production build. Authenticated browser acceptance against the local app remains a separate final acceptance step.

## Student Management — approved guardian relationship vocabulary (2026-11-18)

- Standardized the guardian relationship selector to exactly eight options: Father, Mother, Grandfather, Grandmother, Brother, Sister, Uncle and Aunt.
- Removed Step-father, Step-mother, Legal Guardian, Foster Parent and free-text Other from new links and relationship edits.
- Enforced the same case-insensitive allowlist in the backend service and repository; accepted values are stored in canonical title case. Unsupported API values are rejected.
- Existing guardian links and history are preserved; this change does not revoke, delete or rewrite existing records.
- Added backend unit coverage for all eight accepted values and rejected legacy/free-text values. Updated Student Management browser coverage to assert the selector contains exactly eight options.

## Student Management — system-wide school calendar timezone (2026-11-18)

- Product decision confirmed: use `Asia/Kolkata` (Indian Standard Time) as the single school calendar timezone for the current India-focused release. Haryana, Delhi, Mumbai and Kolkata share this timezone.
- Added a shared backend `schooltime` package. It embeds Go timezone data so Windows/Linux hosts do not depend on an installed OS timezone database, and includes a fixed IST fallback because India has no daylight-saving transitions.
- Date of birth (including the 130-year lower bound), physical measurement future-date validation, attendance past-date permission classification, frontend date bounds/defaults (including Attendance), and the DatePicker Today action now use the same IST calendar date. Forward migration V49 updates the attendance trigger and both DOB/physical-measurement CHECK constraints to use the explicit `Asia/Kolkata` date rather than PostgreSQL session `CURRENT_DATE`.
- Added deterministic backend and frontend timezone-boundary tests for the instant immediately before and at Indian midnight, updated date-validation integration fixtures to use IST, and asserted that the applied PostgreSQL attendance trigger plus DOB/physical-measurement constraints name `Asia/Kolkata`.
- This is a system-wide default, not a per-school setting. If schools outside India are onboarded, add an explicit per-school IANA timezone configuration before relying on local date boundaries for those schools.
- Backend CI **PASS** — [run 44266629951](https://github.com/TheSumitNain/school-management-system/actions/runs/44266629951): migration chain including V49, all Go tests including PostgreSQL integration, and `go vet ./...`.
- Frontend CI **PASS** — [run 44266398402](https://github.com/TheSumitNain/school-management-system/actions/runs/44266398402): TypeScript/Vite production build and all Student Management browser tests.
- Verified backend checkpoint: `ad322818a129d40d0fc285ab7a119a742e80fac6`; frontend checkpoint: `8fe05148e0a0bd712274737813cb502b448be7d2`.
- Migration V49 has not yet been applied to the user's local databases. Authenticated local API/PostgreSQL/browser acceptance remains outstanding; CI uses disposable databases and mocked browser APIs.

#### Student Management — guardian, health, promotion and profile follow-up audit (2026-11-17)

- Rechecked Parent Details create/edit/link/revoke validation, shared-guardian confirmation, relationship-specific primary/emergency flags, inactive/archived mutation gates, history visibility and related-student permission messaging. The form clears student-specific drafts when switching students; no confirmed defect found.
- Rechecked Medical & Physical details loading/retry, field-level limits, decimal precision, measurement-date requirements, future-date rejection, unchanged-value no-op behavior and student-switch/background-refetch draft preservation. The IST conversion exposed a date-only idempotency comparison that could append a duplicate physical measurement; the repository now compares calendar-date strings without converting either side through UTC, and the PostgreSQL integration test covers retry idempotency.
- Rechecked promotion destination-year filtering, section ownership/loading/retry, admission-number and roll-number validation, promotion-date bounds, API field-error mapping and transaction-backed history preservation. No confirmed defect found.
- Rechecked curriculum subject selection against enrollment identity, saved-choice synchronization, group minimum/maximum constraints, history display, and inactive-student/enrollment read-only behavior. No confirmed defect found.
- Rechecked profile summary and year-wise academic history for permission gating, loading/error retry, current-year enrollment status, historical admission number/date, class/section snapshots and lifecycle dates.
- This was a source-level review; it does not replace authenticated browser acceptance against the local API/database. No speculative behavior changes were made.

## Student Management — academic-year-scoped student directory (2026-11-17)

- Added an Academic Year filter to the student directory for users with `academic_years.read`. The selected year controls which enrollment admission number is displayed and searched; name and student-status filtering continue to work across the school-scoped student identity list.
- Students without an enrollment in the selected year remain visible and are clearly shown as **Not enrolled**. The directory does not borrow an admission number from another academic year or from the legacy student-level field when a specific year is selected.
- The API accepts optional `academic_year_id`, requires `academic_years.read` when a specific year is requested, rejects malformed/non-positive IDs at the handler, validates that the year belongs to the current school, and applies the same year-scoped enrollment projection to both count and page queries. Omitting the parameter preserves the existing current/preferred-enrollment behavior for clients without the year filter.
- Added PostgreSQL integration coverage for selected-year admission search, exclusion of another year's admission number, blank admission number for students without a selected-year enrollment, invalid year input, and rejection of an academic-year ID owned by another school. Added route tests for malformed/non-positive query IDs and Playwright coverage for changing the directory year and displaying **Not enrolled** for a student admitted in a different year. The filter toolbar wraps instead of overlapping other controls at narrower content widths.
- Frontend CI **PASS** — [run 44037028496](https://github.com/TheSumitNain/school-management-system/actions/runs/44037028496): TypeScript/Vite production build, design-token check, and all 12 Student Management browser tests.
- Backend CI **PASS** — [run 44037028302](https://github.com/TheSumitNain/school-management-system/actions/runs/44037028302): complete migration chain, all Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `9461627067cb4b67cda85e8b871dd87a223e4766`. Authenticated acceptance against the user's local API/database/browser remains outstanding.

### Student Management — archived attendance history retention (2026-11-17)

- Strengthened the existing PostgreSQL archive lifecycle integration test to record a real attendance mark before archiving the student, then assert that the attendance row remains present after the student's active enrollment is withdrawn.
- The same test still verifies the archive retry is idempotent, the historical enrollment row remains, and any new attendance mark against the archived student is rejected.
- Backend CI **PASS** — [run 44032472405](https://github.com/TheSumitNain/school-management-system/actions/runs/44032472405): complete migration chain, all Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `b00b52464376a41453de02be774ae6e1a2801989`. This uses the disposable CI PostgreSQL database; authenticated acceptance against the user's local API/database remains outstanding.

### Student Management — archive/attendance concurrency regression (2026-11-17)

- Added a real PostgreSQL two-operation race test. It holds the permanent student row lock, queues archive first, then queues an attendance mark behind it, and releases the lock. The test verifies archive commits, the waiting attendance mark rechecks the now-archived student and returns `ErrAttendanceNotFound`, the enrollment is withdrawn, and no attendance row is inserted.
- This turns the archive-versus-new-attendance lock-order guarantee from source-level reasoning into an automated concurrency regression check. Existing attendance history is not deleted by archive.
- Backend CI **PASS** — [run 44032106922](https://github.com/TheSumitNain/school-management-system/actions/runs/44032106922): complete migration chain, all Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `7d8e66cc32a5f90908e331766c56db2d433c5264`. Authenticated acceptance against the user's local API/database/browser remains outstanding; CI uses an ephemeral PostgreSQL database.

### Student Management — date-picker keyboard and modal focus safety (2026-11-17)

- Extended browser acceptance for the Date of Birth picker to verify keyboard day navigation, Escape dismissal and focus restoration to the date trigger.
- The test exposed a shared DatePicker defect: its document-level Escape listener closed the calendar but allowed the same Escape event to reach the parent Modal, which closed the entire student form before focus could return to the date trigger.
- DatePicker now handles Escape in the document capture phase and stops the event before the parent modal's keyboard handler. The calendar closes, focus returns to its trigger, and the surrounding form remains open.
- Frontend CI **PASS** — [run 44031469376](https://github.com/TheSumitNain/school-management-system/actions/runs/44031469376): TypeScript/Vite production build and all 12 Playwright tests passed, including the new DatePicker keyboard and focus assertions.
- Backend CI **PASS** — [run 44031469470](https://github.com/TheSumitNain/school-management-system/actions/runs/44031469470): complete migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `8b777ae1d9656d2c6a6d082ec5244326d4e371c1`. CI browser tests use mocked API responses; authenticated acceptance against the user's local API/database/browser remains outstanding.

### Student Management — field-help keyboard accessibility (2026-11-17)

- Extended the Student Management browser acceptance journey to verify that contextual field help opens by click/tap, can be opened by keyboard activation, and closes with Escape.
- The new browser test exposed a real dialog interaction defect: Escape from an open field-help tooltip bubbled to the parent modal and closed the entire Basic Details dialog. The field-help handler now prevents default Escape handling and stops propagation, so Escape dismisses only the tooltip while preserving the form and its unsaved state.
- Frontend CI **PASS** — [run 44030621918](https://github.com/TheSumitNain/school-management-system/actions/runs/44030621918): TypeScript/Vite production build and all Student Management Playwright acceptance tests.
- Backend CI **PASS** — [run 44030621916](https://github.com/TheSumitNain/school-management-system/actions/runs/44030621916): complete migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `68c7b3ac7d89897b5b0738b3aef189752efc3b10`. CI browser tests use mocked API responses; authenticated acceptance against the user's local API/database/browser remains outstanding.

### Student Management — combined admission and enrollment audit (2026-11-17)

- Rechecked the combined admission handler, service contract and PostgreSQL repository transaction. A new student identity and first academic-year enrollment are committed together; authorization, active academic-year validation, school/year-scoped active section validation, roll-number allocation, both audit events and the enrollment insert all occur in the same transaction. A failure before commit rolls back the combined operation.
- Confirmed the permanent student row deliberately keeps its legacy admission-number column NULL; the admission number is stored on the first year-specific enrollment and projected into the returned student/profile representation. This preserves the approved one-student-identity / one-enrollment-per-academic-year model.
- Verified server-side Unicode character limits for student name (200), year-specific admission number (50) and optional roll number (30), including maximum-length Hindi test values. The first enrollment date is parsed as strict YYYY-MM-DD and checked against the selected active academic year's date range.
- Confirmed duplicate admission-number and active section roll-number conflicts map to their corresponding field-level API errors. No confirmed defect was found in this pass; no speculative code change was made.
- This source/test audit does not replace authenticated local API/database/browser acceptance, which remains outstanding. The `Asia/Kolkata` policy is now implemented for date-boundary validation; authenticated local acceptance remains outstanding.

### Student Management — attendance authorization and date-boundary audit (2026-11-17)

- Reviewed attendance repository and route authorization: list requires `attendance.read`; marking accepts `attendance.mark` or `attendance.correct_past` according to the attendance date; updates accept `attendance.update_same_day` or `attendance.correct_past`. Repository checks additionally restrict records to the caller's school and authorized School Admin / assigned Teacher section scope.
- Attendance writes lock the student row before checking the enrollment and inserting a mark, and audit events are written in the same transaction. Updates lock the attendance row and audit the correction transactionally. Attendance status and remarks are validated server-side; remarks are limited to 500 Unicode characters.
- Resolution (2026-11-18): the system-wide `Asia/Kolkata` policy is implemented. Attendance mark/correction permission checks use the shared school-local date, and migration V49 updates the database attendance trigger to reject future dates against the same IST calendar day. Backend CI and PostgreSQL integration tests pass; local authenticated acceptance remains pending.
- No new confirmed tenant-isolation or permission defect found in this pass. Authenticated local API/database/browser acceptance remains pending.

### Student Management — archived-profile and directory behavior audit (2026-11-17)

- Reviewed the student directory's status/search/pagination contract against the repository query: status is restricted to active/inactive/archived, search is bounded to 100 Unicode characters and matches student name plus the selected/current enrollment admission number, results are school-scoped and deterministically ordered, and page size is bounded to 1–100. The frontend sends a 20-row page and resets the offset when search/status filters change.
- Reviewed archived-profile navigation and mutation gates: archived students remain viewable; Basic Details, Address, Medical/Physical and Parent Details management controls are disabled by the active-status checks, while guardian history remains readable when the caller has guardian-read permission. Guardian relationship/profile mutation controls are gated separately by active student status and guardian-manage permission.
- Archive mutation invalidates the student query namespace, refreshing directory and enrollment data. Existing archive transaction/idempotency and history-retention integration coverage remains the authoritative backend check.
- No new defect demonstrated in this pass; no speculative code change made. Local authenticated browser acceptance remains pending.

# Project Status

## Staff Management — qualifications/documents contract checkpoint (2026-11-21)
- Migration-order correction: project database documentation already describes curriculum migrations through V56. An initially drafted V31 migration pair was removed immediately after detecting the conflict; neither migration was applied to any database. Before adding schema files, verify the exact migration directory head and use the next available version.

- Source review confirms the existing `staff_profiles` foundation and transactional/audited staff account/profile patterns. No staff qualification/certification/experience schema or API was found.
- Added `docs/backend/STAFF-QUALIFICATIONS-DOCUMENTS.md` defining school-scoped education, certification and experience records; proposed API and permission contracts; tenant-isolation, verification and audit invariants.
- Document uploads remain blocked pending a private storage adapter and authorized streaming-download design. No public URLs, web-root uploads or fake upload flows.
- This is a design/documentation checkpoint only: no migration, backend/frontend runtime code, or local/persistent database was changed. Academic, Student Management and Settings remain untouched.



### Student Management — location parent lifecycle locking (2026-11-17)

- Audit found that the V34 hierarchy trigger used `FOR KEY SHARE` when validating a parent location. That lock is compatible with the `NO KEY UPDATE` lock taken when a parent is deactivated by changing only `is_active`, leaving a concurrency window between child creation and parent deactivation.
- Migration V47 changes the parent validation lock to `FOR SHARE`. This serializes child creation against parent deactivation while preserving the existing hierarchy, school-ownership and historical-address rules. The down migration restores the V34 lock mode.
- Added PostgreSQL integration coverage that verifies the applied hierarchy trigger uses `FOR SHARE` and no longer uses `FOR KEY SHARE`.
- Added a two-transaction concurrency regression test: child creation holds the parent `FOR SHARE` lock while parent deactivation attempts its update; after the child commits, deactivation must resume, observe the active child and fail without deactivating the parent. Test fixtures are deactivated during cleanup.
- Backend CI **PASS** — [run 44024624805](https://github.com/TheSumitNain/school-management-system/actions/runs/44024624805): all migrations, Go tests including the concurrent PostgreSQL integration test, and `go vet ./...`.
- Verified code checkpoint: `08158bd53c03a73d1090d759f1efdee3447bf4ae`. Authenticated local API/database/browser acceptance remains outstanding.


### Guardian-link concurrency hardening (2026-11-17)

- The guardian-link transaction now rechecks the guardian's active status under a FOR SHARE row lock after it has serialized on the owning student row. This closes a time-of-check/time-of-use race where a guardian could be deactivated after the initial active check but before a link was inserted.
- Added PostgreSQL integration coverage proving an inactive guardian cannot be linked. The lock also keeps the active status stable through link commit; guardian profile updates wait rather than racing the link.
- The first backend CI run [44011948819](https://github.com/TheSumitNain/school-management-system/actions/runs/44011948819) exposed a stale aggregate guardian-contact audit expectation after the new deactivate/reactivate fixture added two profile audit events. Corrected the expected total to seven in commit `a7c424b1ad06477dba7aa1b87e62013427000030`; backend CI **PASS** — [run 44012966481](https://github.com/TheSumitNain/school-management-system/actions/runs/44012966481). Authenticated live acceptance remains outstanding.

## Active handover — Student Management (2026-11-15)

**Current focus:** finish Student Management only. Parent OTP/SMS/login remains deferred until after the school dashboard; Parent Details in the student record remain required.

### New cross-student profile acceptance coverage — verified (2026-11-15)

- Added a two-student Playwright journey covering unsaved drafts and field-validation errors in Address, Contact, Medical and Physical sections. The test switches from Student A to Student B and verifies that Student B's persisted values load without Student A's drafts or errors.
- Final test commit: `c215c1f52eddc947033bef630f4ecbbbaaff563f` (includes the initial test addition, navigation correction, and assertions that no unintended profile writes or browser errors occur).
- Frontend CI **PASS** — [run 43717701638](https://github.com/TheSumitNain/school-management-system/actions/runs/43717701638): Vite production build succeeded and all 11 Playwright tests passed. The new four-section cross-student test passed in 5.4 seconds; it verified student-specific saved values, cleared drafts/errors, no unintended profile PUTs, and no browser page errors.
- Backend CI **PASS** — [run 43717701635](https://github.com/TheSumitNain/school-management-system/actions/runs/43717701635): complete migration chain, Go tests including PostgreSQL integration, and `go vet ./...` all succeeded.
- This remains mocked browser coverage. Authenticated live acceptance against the local API/database is still separate and pending.

### Parent Details full-browser-reload acceptance — verified (2026-11-15)

- Extended the guardian create/link/edit journey to perform a complete browser reload, sign in again, reopen the student profile, and verify that the updated shared guardian name/contact/profile fields and student-specific relationship flags are restored from the mocked API state.
- Test commit: `ae1e9a3007eddfcb5c2f10f10ddcb6814cc478ca`.
- Frontend CI **PASS** — [run 43718345623](https://github.com/TheSumitNain/school-management-system/actions/runs/43718345623): production build and all 11 Playwright tests passed. The full browser-reload guardian workflow passed in 5.3 seconds; the complete browser suite passed in 48.5 seconds.
- Backend CI **PASS** — [run 43718345636](https://github.com/TheSumitNain/school-management-system/actions/runs/43718345636): all migrations applied to ephemeral PostgreSQL, all Go tests including PostgreSQL integration passed, and `go vet ./...` passed.
- This is mocked browser persistence plus ephemeral-database CI, not authenticated live acceptance against the user's local API/database.

### Optional curriculum subject-selection limits — verified (2026-11-15)

- Extended the student academic-history browser journey to assert that an optional subject group cannot be saved below its minimum or above its maximum, and that correcting the selection re-enables save.
- Test commit: `743970d82dc655e7c63d5a72c1097531ba2cb2dc`.
- Frontend CI [run 43719477558](https://github.com/TheSumitNain/school-management-system/actions/runs/43719477558) reached the new boundary assertions but failed because the expected validation text used `Language` while the fixture group is named `Second language`; the actual UI correctly rendered `Second language`. Corrected the assertion in `9c7dce3feaef57ead8771c7a8c2773f3a3d7ed20`. Backend CI for the original test commit passed in [run 43719477590](https://github.com/TheSumitNain/school-management-system/actions/runs/43719477590). The final component/test commit `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26` improves fixed-size group guidance to “Select exactly 1 subject” (instead of the grammatically incorrect “between 1 and 1 subjects”). Frontend CI [run 43719842826](https://github.com/TheSumitNain/school-management-system/actions/runs/43719842826) **PASS**: production build and all 11 Playwright tests passed, including optional subject minimum/maximum boundaries and exact-one-subject guidance. Backend CI [run 43719842838](https://github.com/TheSumitNain/school-management-system/actions/runs/43719842838) **PASS**: complete migration chain, PostgreSQL integration suite and `go vet ./...` passed. The final full-stack verified Student Management test checkpoint is now `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26`.

### Shared guardian profile across linked students — verified (2026-11-15)

- Extended the Parent Details browser journey with a second active student linked to the same guardian. The test verifies that editing the shared guardian profile updates the contact/profile shown under both students, while the second student's own relationship type and primary/emergency flags remain unchanged.
- Final test commit: `cdfb455ec2aa4451d8a94f421ea022ed23cece09`. The initial test addition was `033cd9ee1d7ee45b2009114aaa26c76e76811adf`; CI caught and the follow-up commits corrected the route-matcher regex and dialog-scoped locator.
- Frontend CI **PASS** — [run 43718944849](https://github.com/TheSumitNain/school-management-system/actions/runs/43718944849): production build and all 11 Playwright tests passed in 51.8 seconds. The shared-guardian journey passed in 6.1 seconds, including a full browser reload and verification across both linked students.
- Backend CI **PASS** — [run 43718944872](https://github.com/TheSumitNain/school-management-system/actions/runs/43718944872): full migration chain, PostgreSQL integration tests and `go vet ./...` all passed.
- Browser API responses are mocked; CI PostgreSQL is ephemeral. Authenticated live acceptance against the user's local API/database remains outstanding.

### Latest verified code and CI

- Active branch: `work`; latest frontend/browser and backend verification checkpoint is `77aec36840f4bf4282e516b97679c84609c2704e` (Frontend CI run 44016765201 PASS; Backend CI run 44016765225 PASS).
- Latest Student Management browser/test checkpoint: `77aec36840f4bf4282e516b97679c84609c2704e`. Frontend CI **PASS** — [run 44016765201](https://github.com/TheSumitNain/school-management-system/actions/runs/44016765201): production build and all 11 Playwright tests passed, including SQL `DATE` midnight-UTC rendering in `America/Los_Angeles` across enrollment directory, profile admission/enrollment history and physical measurement history; archived-student Basic Details/Address/Medical/Physical/Parent Details read-only behavior; cross-student profile isolation, retry/preservation and curriculum subject-selection journeys.
- Backend CI **PASS** — [run 44016765225](https://github.com/TheSumitNain/school-management-system/actions/runs/44016765225) on `77aec36840f4bf4282e516b97679c84609c2704e`: complete migration chain, PostgreSQL integration suite and `go vet ./...`. Coverage includes cross-school read/write denial for student identity, Basic Details, Contact, Address, Medical and Physical data; enrollment history/create/promotion; promotion rejection before recorded attendance with field-specific `admission_date` response; guardian email and student-contact email field-error contracts; location directory tenant boundaries; sensitive profile/enrollment/location route permissions; guardian ownership and relationship isolation; cross-school guardian-link revoke denial; archived-student attendance rejection; independent `guardians.read` plus `students.read_basic` enforcement on related-students routes; and attendance route permission alternatives for read, mark, same-day update and past-date correction.
- Earlier frontend implementation fixes include guardian form validation, student-specific state reset, enrollment curriculum identity, Basic Details empty-state cleanup, Address/Health transient-state cleanup and shared profile feedback handling. The latest green checkpoint includes these commits.
- Background-refetch draft preservation is covered for Basic Details, Parent Details, Address, Contact, Medical, Physical and enrollment subject choices. Cross-student browser journeys for Basic Details, Parent Details, Address, Contact, Medical and Physical are all Frontend-CI-verified at `2b7e6c38f34f1ba28ca38dd3f3e527392b77fd26`.
- Authenticated live acceptance remains pending. CI uses mocked browser APIs and an ephemeral PostgreSQL database; it does not verify the user's local API/database.

### Student profile state isolation correction (2026-11-15)

- Added the current `student.id` to the synchronization effect dependencies for Basic Details, Address, Contact, Medical and Physical forms.
- This ensures that switching the active student rehydrates each section from that student's persisted query data even when two students happen to have identical field-value snapshots; unsaved values from the previously selected student must not carry over.
- Code commits: `295faa89cca2ca33cb673a43c1d95b0db2c655f8`, `19c7f5d763823996d2557a9672efc9b55d5cc130`, `f2bee9b6161f2c3d725d8753abd319c54e77d422`.
- These frontend source changes are included in the passing Frontend CI checkpoint `eaed7705bcf14f156a423a157f5f2adebfc5181c` ([run 43712030788](https://github.com/TheSumitNain/school-management-system/actions/runs/43712030788)); authenticated live acceptance remains pending.

### Student form state isolation — active student switch (2026-11-15)

- Basic Details, Address, Contact, Medical and Physical synchronization now includes `student.id`; equal field snapshots across two students no longer prevent the new student's values from loading.
- Parent Details / Guardian management now clears relationship, shared-guardian edit, create, revoke, history-filter and validation drafts when the active student changes. Guardian records themselves remain shared entities; only the student-specific form state is reset.
- Frontend commits: `295faa89cca2ca33cb673a43c1d95b0db2c655f8`, `19c7f5d763823996d2557a9672efc9b55d5cc130`, `f2bee9b6161f2c3d725d8753abd319c54e77d422`, `ba6f03548ac91ccee979d6775d9cc7d583ae2295`.
- The changes are included in the passing Frontend CI checkpoint `eaed7705bcf14f156a423a157f5f2adebfc5181c` ([run 43712030788](https://github.com/TheSumitNain/school-management-system/actions/runs/43712030788)). Background-refetch draft preservation is separately covered by the same suite; authenticated live acceptance remains pending.

### Enrollment curriculum form identity guard (2026-11-15)

- Added `enrollmentID` to the subject-choice synchronization effect in `StudentEnrollmentSubjectsPanel`. If the panel is reused for a different enrollment whose saved curriculum snapshot happens to match, local optional-subject selections and curriculum target are now rehydrated for the new enrollment instead of carrying over the prior enrollment's draft.
- Frontend code commit: `8f0ded6f42e9c0c9ea9595b5fe57e33463b03b41`.
- Verified by Frontend CI on checkpoint `433461bec48f502765b97464ba6e188a9324d687` ([run 43715113384](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113384)); the latest 10-test checkpoint `b6d14c6ed4030724857838ef8cebc131041219ae` also includes it.

### Basic-details empty-state isolation correction (2026-11-15)

- Fixed a student-switch edge case in `StudentBasicDetailsSection`: when the newly selected student has no saved basic-details row, the form now resets to the empty state and clears field errors instead of retaining the previous student's values.
- Frontend code commit: `1df5e2d0b8c1ee67d21b34e88c2572f281889661`.
- Included in the passing Frontend CI checkpoints `433461bec48f502765b97464ba6e188a9324d687` ([run 43715113384](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113384)) and `b6d14c6ed4030724857838ef8cebc131041219ae` ([run 43716057611]); authenticated live browser acceptance remains pending.

### Student-switch feedback cleanup (2026-11-15)

- Basic Details now also clears stale name-save and section feedback when the selected student changes, so validation/success messages from one student's edit are not shown against another student.
- Frontend code commit: `860bf218dc73b5926079a55b318776afbc24c689`.
- Included in the passing Frontend CI checkpoint `b6d14c6ed4030724857838ef8cebc131041219ae` ([run 43716057611]); authenticated browser acceptance against the real API/database remains pending.

### Student-switch stale feedback and transient-state cleanup (2026-11-15)

- Address & Communication now clears stale contact/address field errors, save feedback, location-creator drafts, pending deactivation confirmation and location-search drafts when the active student changes. The new student's persisted address/contact values continue to be loaded by the existing value-snapshot synchronization effects.
- Medical & Physical now clears stale save feedback and validation errors when the active student changes; its existing snapshot synchronization continues to load the selected student's persisted medical/measurement values.
- Frontend commits: `b302d3761b8ddd4f8e65724234eced9eae9dea6f` (Address & Communication), `6b3355f7d7d5b520fd2ca3bf06028c9b09485e11` (Medical & Physical).
- Included in the passing Frontend CI checkpoint `b6d14c6ed4030724857838ef8cebc131041219ae` ([run 43716057611]); authenticated browser acceptance against the real API/database remains pending.

### Editable profile background-refetch draft regression coverage (2026-11-15)

- Extended the main Student Management Playwright journey to trigger a real TanStack Query window-focus refetch and assert that in-progress Basic Details, Address, Contact, Medical and Physical drafts remain unchanged when the server returns the same persisted snapshot.
- Basic Details, Address, Contact and Parent Details wait beyond their 15-second stale window before the focus refetch; Medical and Physical use their configured immediate-stale behavior.
- Test commits: `2f22369bf41901e7efea80227fc36d483ae77754` (Medical and Physical) and `70f24c208f6705021c21f1e4e03ea6df9aa04044` plus `d81947170aeea9fc2f3082bd6fc8bef335ab2277` (Basic Details, Address and Contact, including explicit stale-window waits and timeout protection); Parent Details relationship draft coverage is added in `ce21411d7ea1a52d5c4633a3c57e317f7f268965` with a 120-second timeout for the full journey.
- The initial frontend run on `2f22369bf41901e7efea80227fc36d483ae77754` exposed the name-save feedback race documented below; the run failed for that assertion, and its concurrent backend run was cancelled by the next push. The expanded test and fix are now committed; verify the workflow on the latest branch head. These are mocked browser tests and do not replace authenticated live acceptance.

### Basic Details name-save feedback regression (2026-11-15)

- The first browser run exposed a real UI race: saving a student's name updates the parent student object, which changed the synchronization effect dependency and immediately cleared the success message before it could be observed.
- Restricted the name-form reset effect to the active student identity. Saving a new name for the same student no longer triggers the student-switch reset; switching to a different student still clears stale form feedback.
- Fix commit: `2419d61299a878669392f69f44549a8e45f7ff9a`. Verify the new frontend CI run before treating it as resolved.

### Student profile query refresh and draft-preservation behavior (2026-11-15)

- The shared QueryClient intentionally disables window-focus refetch globally. Synthetic browser focus events did not reliably trigger React Query in CI, so the regression uses the app's real QueryClient to invalidate the appropriate active query family. This exercises the background-refetch path used by successful student/guardian mutations without changing the app-wide focus policy.
- A per-query `refetchOnWindowFocus` override was attempted and then reverted in `eaed7705bcf14f156a423a157f5f2adebfc5181c`. No new network-on-focus behavior is enabled by that experiment.
- The final mocked Playwright suite verifies preservation of in-progress drafts when the relevant active queries are invalidated/refetched; the final cross-student journey also verifies that changing student identity loads that student's saved values and clears prior drafts/errors. Both frontend and backend CI passed on `c215c1f52eddc947033bef630f4ecbbbaaff563f`.

### Root causes fixed

1. Guardian form length errors were not appearing because HTML `maxLength` blocked over-limit input before app validation. Removed those truncating attributes; explicit field-level validation remains. Frontend CI now passes.
2. Guardian revocation was audited twice: V12's database trigger already inserted the audit event, and Go repository code inserted another. Migration V45 now adds only a boolean `has_revocation_reason` to trigger metadata; repository relies on the trigger as the single audit writer. Free-text revocation reason is never copied into audit metadata. Backend integration tests and `go vet` now pass.

### Implemented Student Management scope

Student profile/basic details; address and contact; Parent/Guardian create/edit/link/revoke/history; medical and physical details/history; academic-year admission/enrollment and promotion; class/year curriculum and optional-subject assignment/history; guardian permission and audit boundaries. Implementation does not mean every live workflow has been accepted.

### Remaining before completion

- Complete authenticated browser/live acceptance against the local API for admission, profile edit/reload, Parent Details load/edit/save/reload, address/contact, medical/physical history, enrollment/promotion/curriculum and permissions. Mocked Playwright CI now covers inactive/archived read-only boundaries, including archived Basic Details, Address, Medical/Physical and Parent Details; repeat these checks against the live local API before sign-off.
- Basic Details, Parent Details, Address, Contact, Medical and Physical cross-student draft/error isolation are now explicitly covered by passing Playwright tests.
- Record exact live acceptance results and any remaining defect; only then mark Student Management complete.
- Background-refetch draft preservation is already covered by the mocked Playwright journey for basic details, Parent Details, address, contact, medical, physical, and enrollment curriculum/subject choices; it is not a substitute for authenticated live acceptance.
- Current Playwright acceptance tests use mocked APIs; passing CI does not replace live authenticated acceptance.
- Location master-data follow-up remains. - The official [LGD Download Directory](https://lgdirectory.gov.in/downloadDirectory.do) exposes a state-wise export workflow with options for all districts, all sub-districts, all villages, all Development Blocks and Development Blocks with covered villages, plus date-scoped modification reports. It offers PDF/HTML/XLS/ODT output and requires a human CAPTCHA; the report-generation form is verified, but no actual export has yet been generated/downloaded or its headers inspected. The official [OGD India LGD catalog](https://www.data.gov.in/catalog/local-government-directory-lgd) showed an Updated On date of 2026-11-13 and exposes a ZIP Download action plus a Catalog API entry; the browser lookup did not resolve the ZIP action to an actual archive, so its contents and schema remain unverified. The portal landing-page summary observed on 2026-11-15 reports 790 districts, 7,112 sub-districts, 7,328 Development Blocks and 677,660 villages. The current `backend/cmd/lgdvalidate` normalized CSV schema covers State → District → Sub-District → locality only; it does **not** model the independent Development Block dimension or village-to-block mapping, so it must not be used to approve a full LGD import yet. Next: obtain the official state-wise exports through the portal UI without bypassing CAPTCHA, preserve untouched bytes/checksums and download metadata, inspect actual XLS/other report schemas, then extend the read-only validator to represent District → Development Block and Village ↔ Development Block mappings separately from Sub-District. Review counts, duplicate codes, cross-district mappings and unmapped entities before designing a separately approved import. Do not use third-party derived datasets as authoritative or write location data to the persistent DB during preparation.
- Persistent database migration/acceptance writes are approval-gated. Do not touch the user's persistent `school_management` database without explicit approval and a verified restorable backup.

### Work-session rule

At every new session, reconcile this file, Student Management design docs, backend/frontend testing docs, current branch tip and latest Actions before editing. Continue from the recorded next task; do not restart completed work. After meaningful code changes, record exact commit, tests, CI run and result. Never claim PASS before the exact check finishes. Keep the repository free of temporary clutter.

**Source of truth:** this section is the first document to reconcile at the start of every new work session. Read this file, the Student Management design, relevant backend/frontend testing docs, and the latest GitHub Actions runs before changing code. Do not restart an already completed audit or treat an earlier green run as proof for a newer commit.

**Last reconciled:** 2026-11-17  
**Repository:** `TheSumitNain/school-management-system`  
**Reviewed branch:** `work`  
**Branch:** `work`.  
**Latest frontend/browser checkpoint:** `77aec36840f4bf4282e516b97679c84609c2704e`. Frontend CI run `44016765201` PASS (production build and all 11 Playwright tests, including negative-offset SQL DATE rendering for enrollment and physical measurement history, plus archived-student read-only coverage across profile sections).  
**Latest backend security-test checkpoint:** `77aec36840f4bf4282e516b97679c84609c2704e`. Backend CI run `44016765225` PASS (full migration chain, PostgreSQL integration tests and `go vet ./...`), including cross-school student identity/basic/contact/address/medical/physical data, enrollment history/create/promotion, location directory operations, guardian ownership/relationship isolation, cross-school guardian-revoke denial, archived-student attendance rejection, independent guardian/student-basic permission checks for related-student routes, and attendance route permission alternatives. 
**Latest Student Management acceptance:** 11 mocked Playwright tests pass, covering cross-student draft/error isolation for Basic Details, Parent Details, Address, Contact, Medical and Physical; background-refetch draft preservation; guardian lifecycle and relationship history; academic-year enrollment, promotion and versioned curriculum choices; date-only calendar display under `Asia/Kolkata`; and archived-student read-only behavior across profile sections. The Playwright API is mocked and CI PostgreSQL is ephemeral, so authenticated live acceptance against the user's local API/database remains outstanding.  
**Location master data:** V40/V42 provide the State/UT baseline and source-reference history. The all-India LGD hierarchy is not imported; `backend/cmd/lgdvalidate` remains a read-only normalized-CSV validator. No persistent database writes or migrations are authorized by this status update.

## Student Management — profile form refetch resilience (2026-11-15)

- Address & Communication now synchronizes its address and contact drafts using stable persisted-field snapshots rather than response-object identity. A background refetch returning unchanged values therefore does not reset unsaved user input.
- Basic Details uses the same value-based synchronization for its editable profile fields.
- Medical and Physical sections likewise compare the persisted field values, preventing automatic remount/refetch behavior from erasing an in-progress medical or measurement draft.
- Frontend CI run `43695705443` passed the production build and all 8 Playwright tests; backend CI run `43695705432` passed the complete ephemeral PostgreSQL migration chain, backend tests including PostgreSQL integration, and `go vet`.
- This CI does not replace authenticated browser acceptance against the user's local API/database. Persistent database migration remains approval-gated.

## Student Management — curriculum edit-state resilience (2026-11-15)

- The enrollment subject-choice panel compares saved optional subject IDs by value rather than array reference when synchronizing query data into form state.
- An explicit Refresh action refetches the current assignment and available curriculum without discarding an in-progress checkbox draft when the server's saved selections have not changed.
- Class curriculum editor controls are disabled while a curriculum save is pending, preventing a year/class switch or a new draft from being silently cleared by the previous save's success handler.
- Historical V44 wording; current Playwright coverage verifies current subject-choice save and preservation of an unsaved choice during explicit refresh, with no selection-history UI. Frontend CI run `43694724225` passed the production build and all 8 browser tests, including the Student Management curriculum and Academic Structure editor scenarios.

## Student Management — V44 curriculum repository verification added (2026-11-15)

- Added a PostgreSQL integration journey through the production repository for the active student enrollment: create version 1 with compulsory and optional subjects, read the available curriculum before assignment, save an optional choice, and verify read-after-write state.
- Historical V44 test scenario; superseded by the current one-record-per-section curriculum model and V55 current-choice-only behavior. No V2/V3 curriculum versions or revoked subject-choice history are part of the current workflow.
- This is committed test coverage, not a test-pass claim. Run the full backend integration suite against the disposable `school_management_test` database and verify the exact commit's CI before marking this workflow complete.
- Browser acceptance must verify curriculum setup/editing, first student assignment, selection-limit errors, confirmed subject/group removal, inactive-student read-only behavior, and Parent Details load/edit/save/reload. Student subject-choice history is not displayed or retained. Parent OTP remains deferred.

## Administrator access correction — School Admin guardian permissions (V39; local verification pending)

- Root cause of the missing Parent/Sibling area: the guardian UI and APIs existed, but the School Admin system-role bundle and its runtime allowlist omitted `guardians.read` and `guardians.manage`.
- V39 adds both school-scoped permissions to every existing School Admin system role and updates the school-creation provisioning function so newly onboarded schools receive the same bundle. The immutable system-role protection trigger is removed only around the reviewed migration insert and is recreated immediately.
- Teacher permissions remain unchanged; Teacher system roles receive neither guardian permission. The School Admin runtime permission allowlist and provisioning integration expectations now include both keys.
- Added PostgreSQL integration coverage for existing-school grants, Teacher exclusion and restoration of the system-role immutability trigger. Newly created-school provisioning coverage also expects the updated 53-permission School Admin bundle.
- No guardian/student records are modified. No frontend permission check or backend authorization middleware is bypassed.
- **V39 has been committed to `work` but has not been run against any local database.** Verify it only on the disposable `school_management_test` database first. Do not apply it to the persistent `school_management` database without separate explicit approval and a verified backup.
- **Platform Admin cross-school business operations remain an explicit architecture gap:** Platform Admin is a platform-scoped principal, while current school business APIs require a school-scoped staff context. Do not impersonate a School Admin or bypass tenant checks. Implement a reviewed selected-school administrative context with explicit target-school authorization and audit before exposing Platform Admin to school business records.

## Student profile UI and guardian relationship verification (2026-11-14)

- The guardian relationship edit regression test was corrected to assert that the previously selected primary contact is cleared, while the newly selected primary contact remains selected. The backend implementation clears the previous primary only within the same student's active guardian links.
- The full backend CI run on commit `3fd23690` passed: complete migrations on ephemeral PostgreSQL, backend tests including PostgreSQL integration, and `go vet`.
- The database engineering standards in [DATABASE.md](../backend/DATABASE.md) now cover immutable versioned migrations, integrity and tenant constraints, transaction/concurrency rules, query/index performance, fresh-install and upgrade verification, and backup/restore safety.
- Address & Communication and Medical & Physical reuse the `.student-address-tabs` class. Its missing styles have now been added with active, hover, keyboard-focus and narrow-screen states. The frontend production build passed on commit `d4fcf7e0`; backend CI for that exact commit is still pending.
- Source review confirms the Basic Details and Address/Contact save actions intentionally remain unavailable while their initial read request is loading or has failed. This prevents saving blank/default form state over data that could not be loaded. The UI exposes the read error and retry action. The precise cause of the previously reported disabled Save buttons has not been established without a live authenticated browser session; do not remove these safeguards or claim the issue resolved until browser acceptance verifies the read endpoints and permissions.
- Browser acceptance remains pending: verify view/edit permissions, load and retry behavior, successful save and reload persistence for Basic Details, current/permanent address, contact details, and Parent Details. Parent OTP/authentication remains deferred.

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
- Admission/enrollment conflict handling now identifies the exact V31 year-scoped admission-number unique index and V17 active-section roll-number unique index instead of treating every PostgreSQL unique violation as the same field error. The API returns structured `field_errors.admission_number` / `field_errors.roll_number` responses; the frontend preserves entered values and displays each conflict directly beneath its field for first admission, subsequent enrollment and promotion. Focused backend tests ensure unrelated unique constraints remain generic, and the mocked browser journey verifies both conflicts → inline feedback → successful retry without creating a partial student.
- Admission conflict coverage now also exercises the HTTP handler contract directly: duplicate admission and roll constraints return HTTP 409 with exactly one matching `field_errors` key, while unrelated academic conflicts remain generic and do not claim a field. The browser journey additionally verifies that correcting either field clears its inline server error. Backend CI passed the full ephemeral PostgreSQL migration chain, all backend tests and `go vet` at `6ceb467`; frontend CI passed typecheck/build and all 6 Student Management Playwright tests at `5f70401`.
- The student directory now opens a profile detail view showing the permanent student identity and year-wise enrollment history, including each year's admission number/date, class/section, roll number and enrollment lifecycle dates. Address & Communication and Parents & Siblings are launched as separate profile-section actions; the directory row no longer carries redundant section shortcuts.
- V34 adds tenant-aware hierarchical locations and separate current/permanent student address records. Backend APIs support location search/add/deactivate and address read/update; the frontend now has an independently editable Address & Communication section with dependent State → District → Block/Tehsil → Village/City selectors and school-owned additions. At V34, no guessed national geography catalogue was seeded; V40 later adds only the verified national State/UT reference list.
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


## Student profile UI review — 2026-11-14

- Confirmed backend automatic roll-number allocation is implemented for combined admission, returning-student enrollment and promotion. Blank values are allocated transactionally after locking the destination section; manual roll-number overrides remain supported. The academic foundation PostgreSQL integration test includes assertions for automatic assignment and manual override, and the user-reported targeted test passed on 2026-11-14.
- Confirmed the promotion API and backend route exist, but the Student Management frontend does not yet expose a promotion action/workflow. This remains a frontend gap; do not treat promotion as end-to-end complete.
- Moved student address/medical profile tab styling into the shared modal stylesheet with explicit selected, hover, focus-visible and mobile states. Removed the duplicate page-level tab rules so styles are owned by the shared modal shell. This is a source-level styling correction only; browser rendering and crash-free interaction still require local browser acceptance.
- User's latest local frontend production build passed before this CSS-only follow-up. Re-run the production build after pulling the new commits; do not claim browser acceptance until manually exercised.


## Student promotion UI and modal tab styling follow-up — 2026-11-14

- Added a promotion action to active enrollment rows for users who have student-update, academic-year-read and class-read permissions. The dialog requires a destination active academic year, destination section, new admission number and date; roll number may be left blank for transactional automatic assignment or entered manually.
- Promotion calls the existing school-scoped backend route. On success, the student enrollment list/history query family is invalidated and the UI reports that the previous academic history is preserved.
- Student address and medical/physical profile tab styles now live in the shared modal stylesheet, with explicit active/pressed, hover, keyboard-focus and mobile layouts. Duplicate feature-page tab rules were removed. This keeps styling with the modal shell used by the profile editors.
- Review also found promotion audit metadata was storing the raw destination admission number and date. It now stores presence flags only, and the PostgreSQL integration test asserts that the raw values are absent.
- These frontend/backend follow-ups have been committed to the work branch. The user's earlier build/test output predates these changes. Run the targeted academic integration test, full backend tests, go vet and frontend production build after pulling; then manually test promotion, profile tabs, edit/view modals, permissions, responsive layout and refresh persistence. No browser acceptance or crash-free claim is made yet.

## Student Management end-to-end audit — 2026-11-14

- User-reported issues: Basic Details and Contact Details Save controls remain inactive after entry; required fields are not consistently identified; the State → District → Block/Tehsil → Village/City location selectors do not load expected data; profile tabs need consistent styling and the full Student Management workflow has not received browser acceptance.
- Source review confirms Basic Details and Contact Details Save controls are disabled while their prerequisite GET query is loading or has failed. Their input controls are not consistently disabled on query failure, so a user can enter values while Save remains disabled. Add explicit loading/error/retry states and prevent editing until existing values are safely loaded; do not bypass the query guard.
- The current status record says V32–V38 were verified in disposable school_management_test, while the persistent school_management application database was not migrated. Basic, contact, medical and address APIs depend on tables introduced by V34–V37. A stale application schema is therefore a strong candidate cause of the failed prerequisite reads and inactive Save controls; confirm the actual application migration version before making a definitive runtime diagnosis. Do not migrate the persistent database until a verified backup and explicit approval are in place.
- V40 now adds source_system/source_code metadata and seeds all 36 India State/UT entries from the Government of India's IGOD State/UT directory (checked 2026-11-14); the migration and its PostgreSQL integration test passed Backend CI. The frontend reads this data from the database rather than hard-coding it. District, Block/Tehsil and Village/City rows remain unseeded; the UI now explains when a reference level is empty and allows authorized school-owned additions. A versioned bulk import from the Ministry of Panchayati Raj LGD is still needed for the full national hierarchy. Do not invent district/locality data or add a runtime dependency on an unverified mirror.
- New UX acceptance rule: every Student Management field visibly says Required or Optional; conditional requirements state their condition; non-obvious fields may show accessible info icons explaining purpose, format, allowed values and restrictions. Help must work with hover, focus and touch. Field errors remain directly beneath their controls. The rule is recorded in the Student Management design and Frontend Architecture documents.
- Scope boundary: this audit/rebuild is limited to Student Management. Preserve Auth, RBAC, academic years, classes, sections and subjects; do not rewrite their migrations or APIs. Preserve student/enrollment/attendance/guardian history. Schema corrections must use additive forward migrations, not edits to applied migration history.
- Code changes are now committed on work: reusable Required/Optional/Required-when-used field labels, contextual help icons, loading/error/retry states for Basic, Contact, Address, Medical, Guardian, academic-year/section and history reads, location empty-state guidance, and V40 State/UT reference data. Frontend CI passed through commit 1ee4eecb. Backend CI passed V40 migration/integration at 0fa6fe5d and the gofmt-only test commit 58dba9fa; the latest frontend-only commit's Backend CI is still running. Browser acceptance and the persistent application-database migration are still pending. Do not apply V40 to school_management until a verified backup and explicit approval are in place.


## Student Management — latest verification (2026-11-14; commit aa8dc03b)

- V40 adds the 36 India State/UT entries from the Government of India IGOD directory as shared canonical location rows, with stable source-system/source-code identifiers and uniqueness constraints. The migration deliberately does not fabricate district, block/tehsil or village/city data; schools can add missing entries through the existing school-owned location workflow.
- V40 has a PostgreSQL integration test that verifies all 36 source-code/name pairs. GitHub Backend CI on `aa8dc03b` completed successfully, including the migration chain, PostgreSQL integration tests and `go vet`; Frontend CI on the same commit completed successfully, including TypeScript and the Vite production build.
- The latest student-profile safety fixes prevent Basic Details and Contact Details values from being edited or saved when their initial profile read fails. Retry controls remain available so the user can recover without accidentally overwriting existing values with empty defaults.
- Guardian management is school-scoped and uses live `guardians.read` / `guardians.manage` permission evaluation in the repository, in addition to route authentication. Guardian profile edits are shared identity updates and intentionally affect every linked student; the UI explicitly warns staff before/after this operation. Relationship type, primary-contact and emergency-contact flags are stored per student relationship. Revocation requires a reason and preserves history.
- Current branch head: `aa8dc03bfb14854184f0bf7a3efaf824fca04ee0`. Auth, RBAC, Class, Section and Subject modules were not changed by this Student Management UI safety pass.
- **Local V39–V40 migration application and browser-level acceptance remain pending.** CI success does not establish that the user's local persistent `school_management` database has these migrations. Do not migrate that database without explicit approval and a verified backup; use only `school_management_test` for local integration verification.

## Shared guardian update confirmation — 2026-11-14

- Added an explicit confirmation step before saving edits to a shared guardian profile. The confirmation explains that the guardian's contact/profile values are shared by every linked student, and the action label makes the multi-record scope clear.
- Validation still runs before confirmation; the update mutation remains protected by the existing live guardian-management permission and invalidates guardian/student query families after success. Relationship-specific values (relationship type, primary contact and emergency contact) remain on each student-guardian link, not on the shared guardian profile.
- Historical verification note: the shared-profile confirmation workflow is now covered by frontend CI run `43692374206`, which passed all 7 Student Management Playwright tests and the production build.

## Student Basic Details date-range alignment — 2026-11-14

- The Date of birth picker now enforces both ends of the same range as the backend: no future date and no date earlier than 130 years before today. This prevents users from selecting values that the save API will reject and makes the field guidance actionable at entry time.
- Historical verification note: the date-range alignment is included in the frontend code checkpoint whose TypeScript/Vite build and 7 Student Management Playwright tests passed in CI run `43692374206`. Live acceptance against the user's persistent database remains pending.

## Student Parents & Siblings editability audit — 2026-11-14

- The student profile exposes Parents & Siblings as a dedicated section in the same student-record workflow. It loads guardian relationships and related students, supports creating/linking guardians, editing shared guardian profile fields with an all-linked-students confirmation, and revoking links with a required reason while preserving history. Parent OTP/login remains out of scope.
- At the time of this audit, the remaining gap was student-specific relationship editing. It was subsequently closed by the authorized `PATCH /school/student-guardian-links/:linkID` operation and the UI below; the active Parent Details workflow now edits shared guardian fields and per-student relationship/contact flags separately. Do not work around relationship edits by changing the shared profile or revoking/recreating links.
- This was a historical implementation note; relationship editing was delivered in the following section and the complete mocked Parent Details workflow later passed all 7 Student Management Playwright tests in CI run `43692374206`.

## Student-specific guardian relationship editing implemented — 2026-11-14

- Added `PATCH /school/student-guardian-links/:linkID`, protected by the existing live `guardians.manage` permission and school-scoped repository authorization.
- Staff can now edit Relationship Type, Primary Contact and Emergency Contact on an active student–guardian link without changing the shared guardian profile or revoking/recreating the relationship.
- Primary-contact selection is serialized by locking the owning student row; choosing a new primary clears the previous active primary only for that same student. Other students linked to the same guardian remain unchanged.
- The update is transactional, restricted to active students and active links, and writes an append-only audit event with before/after relationship values and contact flags. Revoked links cannot be edited; relationship history remains intact.
- Guardian relationship edit forms and action controls wrap responsively inside each guardian card on narrow screens.
- Frontend includes field-level relationship validation, per-student labels/help text, loading/error/success feedback and query invalidation. Backend handler/route tests and PostgreSQL integration coverage include permission enforcement, audit evidence, primary-contact exclusivity and isolation from another student's link.
- The original backend/frontend implementation was subsequently verified by backend CI run `43692087083` and frontend CI run `43692374206`. The browser suite uses mocked APIs; live acceptance against the user's persistent database remains pending.

## Student guardian relationship database editability correction — 2026-11-14

- Backend CI exposed a schema/application mismatch: V12's relationship-history trigger treated `relationship_type` as immutable, so the new authorized relationship-edit repository operation was rejected by PostgreSQL even though its application validation allowed the change.
- Added forward migration V41 to permit edits to the relationship label while keeping student ID, guardian ID, school ownership, row identity, creation time and revoked-history fields immutable. The existing 1–40 character database constraint remains in force.
- V41 retains the same actor/school checks for revocation. The down migration restores the original V12 trigger behavior.
- Backend CI run `43692087083` subsequently passed the full migration chain, PostgreSQL integration suite and `go vet`, including relationship-type changes, primary-contact exclusivity, cross-student isolation, revoked-link immutability, inactive-student restrictions and audit logging.
- This is an additive migration; do not edit V12 or apply V41 to the persistent `school_management` database without the user's explicit approval and a verified restorable backup.


## Student Management verification update — 2026-11-14

- Backend CI for commit 98c10e62 completed successfully: the full migration chain applied to an ephemeral PostgreSQL database, all backend tests including PostgreSQL integration tests passed, and go vet passed. The earlier V41 status entry above is therefore verified; its CI-pending wording is historical.
- The guardian relationship integration test now asserts primary-contact exclusivity against active links only. The include-history query verifies history visibility/count; it is not a historical snapshot of previous field values. A redundant second active-link read was removed.
- The frontend source audit confirms Basic Details and Contact Details Save controls intentionally remain disabled when the prerequisite sensitive-profile GET fails. Do not remove this guard or permit editing against empty fallback values. Retry must recover the read first.
- GET access for Basic Details, Contact Details and Address requires the existing students.read_sensitive permission; corresponding writes require students.update. Keep these existing live permission boundaries unchanged. If the UI still reports a failed read after the local database is current, capture the API error and diagnose the specific endpoint rather than weakening authorization.
- The known local adoption gap remains: the user's persistent school_management database was last reported at migration V24, while student address, contact, medical/physical and basic-details tables are introduced in V34–V37, guardian profile extensions in V38, State/UT reference data in V40 and relationship editability in V41. This is a likely explanation for failed profile reads on that database, not proof of its current version. Do not migrate it without explicit approval and a verified restorable backup.
- Current Student Management gaps: authenticated live browser/API acceptance against the user's persistent database and persistent migration adoption remain pending; the national District → Block/Tehsil → Village/City reference hierarchy still needs an authoritative LGD import. State/UT rows are seeded by V40, but lower-level rows are deliberately not fabricated. Mocked browser create/view/edit/save acceptance is covered by frontend CI; ephemeral PostgreSQL migration/integration acceptance is covered by backend CI.
- Current work remains scoped to Student Management. Auth, RBAC, Class, Section and Subject are unchanged.


## Read-only database adoption preflight — 2026-11-14

- Added `backend/cmd/dbstatus`, a read-only command that reports the configured database name, golang-migrate version and dirty flag. It performs no schema or data writes and exits with an error for dirty or malformed migration state.
- Documented the command in the backend database guide. Use it to establish the actual persistent database version before discussing an upgrade; it does not replace a backup/restore rehearsal or the user's explicit approval.
- The new command is awaiting Backend CI verification. Do not run `go run ./cmd/migrate` against the persistent `school_management` database as part of this preflight.


## Address hierarchy terminology correction — 2026-11-14

- Corrected Student Address labels and help text: the database's `subdistrict` level is now presented as **Sub-District / Tehsil**, not Block / Tehsil. This avoids conflating a revenue sub-district (Tehsil/Taluk/Mandal) with a Development Block, which is a distinct LGD administrative entity.
- The current address path remains State/UT → District → Sub-District/Tehsil → Village/City. A future Development Block field, if required by school workflows, must be a separate optional reference relationship and must not be inserted into the revenue hierarchy.
- Documented official Ministry of Panchayati Raj / OGD LGD entry points for District, Sub-District and Village datasets and the requirement to verify actual official export schema before writing an importer. Community mirrors remain untrusted for production ingestion.
- The read-only `backend/cmd/dbstatus` command and its documentation passed Backend CI on commits a7ea3c3c / 22733318 (migration chain, PostgreSQL integration suite and go vet).
- The terminology change passed Frontend CI on commit 46fe22fe. The follow-up label-only correction and subsequent documentation commits are awaiting their own CI runs. Browser acceptance and the persistent database upgrade remain pending.


## LGD snapshot pre-import validation — 2026-11-14

- Added `backend/cmd/lgdvalidate`, a read-only standard-library CSV validator for a normalized all-India State → District → Sub-District → Village/City snapshot.
- The validator checks exact headers, required values, source-code/name limits, duplicate and conflicting hierarchy mappings, parent relationships, locality types, one-to-one two-character IGOD State/UT crosswalks, and minimum coverage counts. Leading-zero source codes are preserved as strings.
- It does not connect to PostgreSQL and cannot mutate data. The raw official OGD/LGD export schema has not yet been verified; the exact downloaded source file must be preserved and transformed to the documented normalized contract before validation. No community mirror is used as a production import dependency.
- Added unit tests for valid hierarchy parsing, malformed headers/crosswalks, duplicate or conflicting codes/parents, missing hierarchy consistency, and unsupported locality types. Backend CI is pending for the new command and tests.
- No LGD location rows have been imported. A database importer remains gated on a verified official snapshot, reviewed IGOD↔LGD State/UT crosswalk, validation report, and explicit apply mode.

- During source inspection on 2026-11-14, the OGD resource pages were returning sandbox/maintenance or no-result content in the page view, so the live raw export schema could not be independently inspected there. Dataset metadata pages still identify Ministry of Panchayati Raj as owner and describe monthly refreshes, but this is not sufficient to claim the current downloadable file schema is verified. Keep the normalized validator contract separate from raw-export parsing until an actual official file is obtained and inspected.


## Backend CI efficiency and validator test corrections — 2026-11-14

- Backend CI now skips documentation-only changes and uses a branch-scoped concurrency group with cancel-in-progress, preventing stale runs from consuming PostgreSQL runner capacity after new code is pushed.
- The first LGD validator CI run exposed test-fixture mistakes (short header, impossible missing-parent expectations in a denormalized ancestor-row format). The tests were corrected to check the actual contract: exact ten-column header, conflicting source-code/parent mappings, duplicate IGOD crosswalks, and BOM/leading-zero preservation.
- The corrected validator commits are undergoing Backend CI. Do not mark the tool verified until the latest branch-head run completes.

## LGD validator verification — 2026-11-14

- The corrected read-only validator and its test suite passed Backend CI on code commit `98c6acfb72` (workflow run `43593036081`). The ephemeral PostgreSQL migration chain, complete backend tests including PostgreSQL integration, and `go vet` all passed.
- The branch documentation-only HEAD is `ede9692470`; the CI workflow intentionally skips documentation-only changes, so the successful code commit remains the relevant verification target.
- This verifies the normalized CSV validator only. It does **not** verify a parser for the raw OGD/LGD export, import any locations, migrate the persistent local database, or replace browser acceptance.
- Next data gate: obtain and preserve an actual official LGD snapshot, verify its raw schema and source checksum, review the IGOD↔LGD crosswalk and row-count drift, then implement a staged importer with dry-run/report mode before any explicit database apply.

## Guardian directory least-privilege query — 2026-11-14

- Source review found that opening a read-only Parents & Siblings profile also fetched the full active guardian directory, even though the guardian picker is only used by users with `guardians.manage`.
- The directory query is now enabled only when the current user has `guardians.manage`. Read-only staff continue to load the student's linked guardian details through the existing `guardians.read` permission, without requesting unrelated guardian-directory records.
- This is a frontend least-privilege/data-minimization correction; it does not change backend permissions or allow/deny rules. Frontend CI passed on commit `bd9cec3b0f` (TypeScript/design-token checks and Vite production build). Backend CI for the same commit is running; the frontend build is verified, while browser-level role acceptance remains pending. Browser-level role acceptance remains pending.

## Guardian relationship concurrency hardening — 2026-11-14

- Guardian relationship revocation now resolves the owning student and locks that student row inside the same transaction before revoking the active link.
- This uses the same per-student serialization point as link creation and relationship edits, preventing revocation from racing with primary-contact selection.
- The lock intentionally does not require the student to remain active, preserving the existing ability to revoke a relationship for an inactive student.
- Added a PostgreSQL integration regression case that links a guardian, deactivates the student, revokes the relationship, and verifies the revoked relationship remains visible in history.
- Backend CI for commit `f902940b68` is running; the final result will determine whether this concurrency hardening is verified.


## Guardian relationship database invariants — 2026-11-14

- Backend CI for revocation serialization commit `f902940b68` passed: full migration chain, PostgreSQL integration tests (including inactive-student revocation/history), and `go vet`.
- Audit found that the application serialized primary-contact changes but PostgreSQL did not independently enforce the one-active-primary-per-student rule.
- Added V43 partial unique index `ux_student_guardians_active_primary_per_student`; migration preflights existing active primary duplicates and fails rather than silently changing records.
- Added PostgreSQL integration coverage proving a direct duplicate-primary insert is rejected, while existing repository behavior can safely transfer primary status.
- Multiple emergency contacts remain allowed intentionally. The V43 down migration drops only the index and preserves all relationship data.
- V43 migration and new constraint test are awaiting Backend CI. Persistent databases have not been migrated.


## Shared guardian profile propagation regression — 2026-11-14

- Added PostgreSQL integration coverage that edits a guardian's shared name, mobile and occupation after that guardian is linked to two students.
- The test verifies both students immediately read the updated shared profile while their independent relationship type, primary-contact and emergency-contact values remain unchanged.
- This complements existing tests for cross-school isolation, relationship edits, primary-contact uniqueness, revocation and history.
- Backend CI for commit `0efba97f51` is pending. No persistent database has been migrated.

## Student profile repository regression coverage — 2026-11-14

- Backend CI for commit `3f51290f2906262e85f4513525ad76c76dafbc46` (workflow run `43597056423`) passed: full migration chain on ephemeral PostgreSQL, all backend tests including PostgreSQL integration, and `go vet`.
- Added address save round-trip coverage that verifies updated current-address fields, preserves the independent permanent-address `same_as_current` setting, and checks the address-update audit event.
- Added contact save round-trip coverage that verifies primary phone, alternate phone and email persistence and checks the contact-update audit event.
- Existing V36 integration coverage verifies medical profile version snapshots, append-only physical measurement history, invalid measurement rejection, cross-school read denial, teacher health-data denial, and audit metadata that excludes private health values.
- Existing V37 integration coverage verifies basic-profile save/read, registration-number normalization/uniqueness, immutable identity/creator fields, cross-school read denial, and value-free audit metadata.
- Source review confirms address/contact/basic/medical/physical saves are separate transactional operations; all writes require live `students.update`, sensitive reads require `students.read_sensitive`, and each write locks the owning student row before mutation. Existing tenant and sensitive-data boundaries remain unchanged.
- No persistent database was migrated. Browser-level create/view/edit/save acceptance remains pending; do not claim full UI acceptance based on repository tests alone.

## Student Management view-switcher accessibility — 2026-11-14

- Student Directory / Enrollments switcher now uses a labelled button group with `aria-pressed`, matching its actual interaction model. It no longer advertises an incomplete ARIA tablist without tabpanel and keyboard-roving-focus behavior.
- Existing visual styling, permission-based visibility, and view-switching callbacks are unchanged.
- Frontend CI for commit `233fa27595dd2336edc23f21b7765a701152dbf2` (workflow run `43597474039`) passed: design-token/style checks, TypeScript project build, and Vite production build.
- Backend CI for the same commit (workflow run `43597474026`) was still running when this status entry was written.
- Browser-level Student Management acceptance remains pending; no claim of end-to-end browser verification is made.

## Student profile edit navigation — 2026-11-14

- Closing Basic Details, Address & Communication, Parents & Siblings, or Medical & Physical section now returns to the same student's profile instead of dropping the user back to the directory.
- This avoids repeated student searches when completing multiple profile sections in one session. Existing modal close/pending guards and permission checks are unchanged.
- Frontend CI for commit `478af7360832d5210d5a387e573f4839cde7d642` (workflow run `43597540743`) passed: `npm ci`, design-token/style checks, TypeScript project build, and Vite production build.
- Backend CI for this commit is queued/pending; no backend source changes were made.

## Student Management browser acceptance baseline — 2026-11-14

- Added a Playwright Chromium browser test and CI execution on Ubuntu 24.04 LTS. The runner is pinned to Playwright `1.56.1`; it is installed ephemerally in CI so the production dependency lockfile remains unchanged.
- Browser acceptance workflow run `43599136283` passed for commit `9a1d4f8b545a`: frontend style-token checks, TypeScript/Vite production build, Chromium installation, and the Student Management browser test.
- The browser test signs in with a test-only mocked staff session, admits a student with a blank roll number (automatic assignment path), verifies directory refresh and academic history, opens/closes Basic Details, opens Parents & Siblings, and checks that the same student profile remains available after closing each section. It also fails on uncaught browser exceptions.
- All `/api` calls in this browser test are intercepted with in-memory fixtures. It does not connect to or mutate the local/persistent school database; backend PostgreSQL integration tests remain the source of truth for persistence, tenant isolation and audit behavior.
- Failure screenshots and traces are retained as a seven-day GitHub Actions artifact. No generated test output is committed to the repository.
- Browser coverage is now in CI; manual acceptance against the user's configured local API/database remains a separate final verification step.

### Parent relationship browser coverage follow-up

- Expanded the Playwright test to return an existing linked guardian and verify the guardian name, relationship, primary-contact and emergency-contact labels render.
- The test edits the relationship from `Father` to `Legal Guardian`, verifies the PATCH payload updates only that student's relationship record, and confirms the updated label renders after query refresh.
- Frontend CI run `43599397403` passed for commit `43ea1aef0f5663053b806a393c2f3aec811cfcba`, including the expanded parent relationship browser flow.
- Guardian profile shared-field editing (which affects every linked student) remains separately protected by its confirmation flow; this smoke test deliberately does not mutate shared guardian details.

### Address, contact and physical-details browser coverage follow-up

- Expanded the same Chromium acceptance test to open Address & Communication, save a primary contact phone number, and verify the saved value in the mocked API state.
- Expanded Medical & Physical Details coverage: the test checks the restricted-health privacy notice, switches to Physical Details, confirms invalid height and missing measurement date errors appear together, then saves a valid height with a measurement date.
- Frontend CI run `43599830202` passed for commit `6732d1c8a5cfe42b44a0fd37b7b3cbc473d65d85`, including all expanded browser flows.
- These remain frontend-only mocked API checks. PostgreSQL integration tests separately verify persistence, tenant boundaries, permissions, history and audit behavior.


## Student Management address and medical browser coverage — 2026-11-14

- Expanded the Playwright Chromium workflow to exercise the Current Address location chain using test fixtures: Haryana → Ambala district → Ambala Sub-District/Tehsil → Naraingarh locality. The test verifies the saved locality and resolved State, District and Sub-District IDs in the mocked API state.
- Added Permanent Address coverage for the explicit “same as current” option and verifies the saved same_as_current flag.
- Added Medical Details save coverage for Blood Group, Allergies and Known Conditions, including the returned profile version increment. Existing physical-measurement range/date validation and save coverage remains in the same browser journey.
- The test still intercepts every API request and uses in-memory fixtures. It verifies frontend interaction and request/response handling only; it does not prove live database persistence or replace backend PostgreSQL integration tests.
- Frontend CI passed for commit f0267654b18c904359c12b2cdaf0ff45ba8f57d7 (workflow run 43600789669), including design-token checks, TypeScript, production build and Playwright Chromium.
- Backend CI for the same commit was still running at the time of this entry. Persistent database migration/adoption and manual browser acceptance against the configured local API remain pending.

## Student Management guardian permissions and promotion browser coverage — 2026-11-14

- Expanded guardian browser coverage for shared-profile editing: invalid email is rejected without saving; valid edits require explicit confirmation because contact/profile data affect all linked students; student-specific relationship, primary-contact and emergency-contact flags remain unchanged.
- Added guardian revocation browser coverage: a reason is mandatory, the active relationship disappears after revocation, and the revoked relationship remains available in history.
- Added view-only permission coverage: staff with student read and guardian read permissions but without update/archive/guardian-management permissions can view the student and linked guardian, but cannot see edit, archive, link or revoke actions. The test verifies no write request is sent.
- Added promotion browser coverage for required admission number validation, destination academic year/section selection, blank roll-number auto-assignment path, and preservation of the prior enrollment with promoted status alongside the new enrollment.
- Frontend CI passed for commit `e8b907abbb20` (run `43602032049`), including shared guardian update and revocation/history coverage.
- Frontend and Backend CI passed for commit `288a70a25440` (runs `43602633101` and `43602632967`), including student and guardian view-only permission coverage.
- Frontend and Backend CI passed for commit `1d739fcb0169` (runs `43603172771` and `43603172772`), including promotion validation and preserved enrollment history.
- Browser tests intercept all API requests and use in-memory fixtures; they do not write to the local or persistent school database. Backend PostgreSQL integration tests run separately against the ephemeral CI database. No persistent database migration was run.

## Student Management archive and guardian create/link browser coverage — 2026-11-14

- Added archive browser acceptance coverage: cancelling confirmation sends no request; a server conflict keeps the confirmation open and the student visible; retry success refreshes the directory to archived status.
- Added guardian create/link browser acceptance coverage: invalid email is rejected before create; a valid guardian contact is created, then linked to the student with a relationship and primary/emergency contact flags.
- Frontend CI passed for commit `5c555459a6f8` (run `43603791131`), including archive cancel/conflict/retry and archived directory refresh.
- Frontend CI passed for commit `196f3e41af15` (run `43603948048`), including guardian create/link validation and relationship flags.
- Backend CI passed for commit `5c555459a6f8` (run `43603791165`); the guardian create/link commit changes only frontend browser tests.
- Browser API requests are mocked and use in-memory fixtures; they do not mutate a real school database. Parent OTP/login remains deferred. No persistent database migration was run.

## Student name edit/profile state consistency — 2026-11-14

- Fixed a stale profile-state issue: after saving a student's name in Basic Details, the parent Student Management state now receives the updated name, so closing Basic Details returns to a profile showing the new name rather than the previous value.
- Preserved the successful name-save feedback while synchronizing the updated student prop.
- Added a browser assertion for the name update, success feedback, and profile description after returning from Basic Details.
- Frontend CI passed for commit `f5b9a2ff7396` (run `43604393646`).
- Browser test uses mocked API fixtures; this is not yet a live local-PostgreSQL/browser acceptance test.


## Latest branch reconciliation — 2026-11-14

- Reconciled this status document with the current `work` branch tip `32a8af14777ecabcfd5daaf3d5d2dce002778411`; the previous header stopped at `d4fcf7e0` and was stale.
- Branch history since the previous status checkpoint includes `3dcbb84e` (staff, timetable, attendance and teacher-assignment workflows plus backend foundations), followed by Student Management and Academic Structure UI refinements through `32a8af14`.
- Source review of the current Student Management page confirms the combined admission payload includes the academic year, section, admission number/date and optional roll number; profile sections continue to keep Basic Details, Parent Details, Address/Contact and Health separate. Parent OTP remains deferred.
- Source review of Academic Structure confirms section medium is optional, section year/class are immutable during section edit, and status values are submitted through controlled hidden inputs. No confirmed regression was found in these paths during this read-only review.
- The current browser suite remains explicitly mocked at the API boundary. It must not be described as live local-API or persistent-PostgreSQL browser acceptance.
- CI status for the latest branch tip was not available from the checks inspected during this reconciliation; do not infer a pass from earlier commits. Run/inspect the exact tip's frontend and backend workflows before marking it verified.
- No source code, database migration, local database, or backup was modified as part of this reconciliation. The persistent `school_management` database must remain untouched unless separately approved with a verified backup.
- Remaining acceptance work: verify Staff, Teacher Assignments, Timetable and Attendance flows end-to-end; expand browser tests beyond Student Management; run fresh-database and upgrade-path integration checks for migrations V25 onward in a disposable PostgreSQL database; then perform live local API/browser acceptance.


## Student Management integration-test audit and CI reconciliation — 2026-11-15

- Backend CI for exact branch tip `ab8993413af53eb67a2b40ca955e37bc8e2de78b` passed (workflow run `43611122792`): ephemeral PostgreSQL containers initialized, the complete migration chain applied, all backend tests including PostgreSQL integration passed, and `go vet` passed.
- The backend CI failure on earlier commit `0d76f272` was traced to expected PostgreSQL constraint/trigger failures aborting the surrounding test transaction. Savepoint recovery was added to the V42 location-source and guardian-schema integration tests in `e9fb7761` and `ab899341`; the latest exact-tip CI now passes.
- Source audit confirmed rollback-scoped fixture transactions in the V12 parent/guardian authorization, V31–V33 admission identity, V34 address/location, V35 contact, V36 medical/physical, V37 basic-details, guardian repository, and student-profile integration tests. Read-only schema/reference queries outside those transactions are intentional. The shared Platform Admin lifecycle actor helper in the V36 test remains an intentional reusable fixture, not an accidental per-test data leak.
- Expected database rejections in the reviewed V12, V31, V33–V37 and V42 tests are isolated with savepoints where later assertions reuse the transaction. No additional persistent-fixture leak was identified in this targeted review.
- Frontend CI run `43610711016` passed for `dfeb4bd7`, including the expanded mocked Playwright Student Management journey. Subsequent `e9fb7761` and `ab899341` commits changed backend integration tests only; this remains the latest relevant frontend verification.
- Live acceptance is still pending: Playwright API calls remain mocked, no local persistent database was migrated, and authenticated browser verification of actual load/save/reload behavior and role permissions remains outstanding. Keep `school_management` untouched until explicit approval and a verified backup.


## Guardian contact validation and cross-layer review — 2026-11-15

- Guardian email is now normalized and validated on the backend repository write path using Go's `net/mail.ParseAddress`; blank optional values become NULL, values over 254 bytes are rejected, and display-name/multiple-address input is not accepted as a single email field.
- Unit coverage includes valid and whitespace-trimmed addresses, nil/blank optional values, missing local/domain portions, display-name and multiple-address input, and over-length values.
- Latest backend CI for exact commit `05b9caa8ef9236417817c50353bac9325faba856` passed: complete ephemeral PostgreSQL migration chain, backend unit and PostgreSQL integration tests, and `go vet` (run `43669081600`).
- Cross-layer review confirmed the Parents & Siblings UI already validates email fields inline before submit, preserves shared guardian profile semantics, keeps per-student relationship flags separate, and refreshes related query data after writes. Existing Playwright coverage exercises invalid email rejection and guardian create/link/edit/revoke flows using mocked APIs.
- Database-triggered guardian revocation audit is already implemented in V12 and asserted by repository integration coverage; no duplicate application-level audit insert was added.
- Remaining acceptance is unchanged: run authenticated browser checks against a migrated disposable/local test environment to prove real backend read/write/reload and role-based behavior. Do not apply outstanding migrations to persistent `school_management` without explicit approval and a verified restorable backup.


## Guardian history privacy correction — 2026-11-15

- The Parents & Siblings UI previously requested `include_history=true` on every open, even while the “Show revoked history” checkbox was off. The screen hid revoked rows, but the backend still returned their guardian contact fields.
- The query now keys and fetches by the checkbox state: default reads omit `include_history` and return active links only; enabling history issues a separate `include_history=true` read. This reduces unnecessary access to revoked relationship/contact data and preserves history-on-demand behavior.
- Playwright coverage now checks the initial active-only request and verifies that enabling the checkbox requests history. It tolerates legitimate React Query refetches after mutations rather than assuming exactly one request.
- Latest exact-tip verification on `06d2325a5e7700bf493f1d8bf21f07d19c9d5fe6`: Frontend CI run `43669467828` passed production build and all 5 Playwright tests; Backend CI run `43669467831` passed all migrations on ephemeral PostgreSQL, backend unit/integration tests, and `go vet`.
- Playwright uses mocked API responses; it is not proof of persistence against the user's local database. Live authenticated browser acceptance and local V39+ migration remain outstanding. Do not modify persistent `school_management` without explicit approval and a verified restorable backup.


## LGD reference-data source review — 2026-11-15

- Rechecked the official Ministry of Panchayati Raj download page on 2026-11-15: <https://lgdirectory.gov.in/downloadDirectory.do>. Its Download Directory workflow offers “one entity at a time”, “all entities of State” and “only modifications”; entity choices include all districts, all sub-districts, all villages, separate development blocks and urban local bodies. Report formats shown are PDF, HTML, XLS and ODT; report generation requires the portal's CAPTCHA. Do not automate around the CAPTCHA.
- The OGD LGD catalog is published by the Ministry of Panchayati Raj and showed catalog metadata updated on 2026-11-13: <https://www.data.gov.in/catalog/local-government-directory-lgd>. Resource-level dates are not a single common snapshot (the district resource showed 2026-10-27 while the indexed village resource showed 2025-11-17); the Catalog API link returned HTTP 503 during this review. Treat OGD metadata as discovery only until a complete, date-consistent official export set is actually obtained and verified.
- For any human-procured download, record the exact download date, selected entity/state, report format and source URL for each original file. Preserve each original file unchanged and calculate its SHA-256 before any deterministic XLS-to-normalized-CSV conversion.
- The existing `lgdvalidate` command is a read-only validator for a normalized hierarchy CSV. It is not an XLS parser or database importer. Any XLS-to-CSV normalization must be documented and deterministic, preserve all LGD codes as strings (including leading zeroes), and retain the original downloaded files and checksums for review before import.
- The validator currently expects one flattened state → district → sub-district → locality row per locality and a reviewed one-to-one IGOD State/UT crosswalk. It restricts IGOD codes to the 36-code V40 crosswalk, detects conflicting source-code/name/parent/locality-type mappings within each entity namespace, duplicate locality rows, verifies parent consistency and supported locality types, preserves leading-zero source codes, applies PostgreSQL-compatible Unicode character limits, and enforces configurable minimum coverage. Identical code strings across different entity classes are allowed because V42 source_system namespaces are entity-specific (LGD_STATE, LGD_DISTRICT, LGD_SUBDISTRICT, LGD_VILLAGE). It remains read-only and must run before any import.
- Important hierarchy distinction: LGD exports villages, development blocks and urban local bodies as separate entity types. Do not silently map development blocks to revenue sub-districts or urban local bodies to villages. The current student address hierarchy is state → district → subdistrict → locality; a reviewed mapping policy is still needed for urban cities/towns and any locality that does not have a direct revenue sub-district parent.
- **No LGD district/sub-district/village snapshot has been imported.** The official portal download is manual/CAPTCHA-gated, and the city/town crosswalk is unresolved. The import remains blocked until source files, checksums, date, field mapping and hierarchy reconciliation are reviewed. No database was changed during this source audit.

## Student Management readiness checkpoint — 2026-11-15

**Code and CI state:** Student Management workflows on `work` include admission and yearly enrollment/promotion, identity/basic details, address/location, contact, medical/physical details, guardian contact profiles, student-specific guardian relationships, related-student lookup, history/revocation, live permissions and audit behavior. Class Curriculum is section-scoped and has one editable current record per academic year/class/section. V54+V55 preserve the same curriculum and student assignment identities while supporting confirmed permanent subject removal; removed subjects and linked current student choices are deleted, and saving changed choices replaces the current choice rows. Curriculum edits do not create V2/V3 versions or edit history, and Student Management no longer exposes student subject-selection history. Compulsory subjects, standalone optional subjects and configurable optional groups remain supported. Parent Details remains staff-managed; Parent OTP/SMS/login is deferred. Inactive/archived student records retain read-only family access while writes are blocked at both UI and repository layers.

**Latest automated verification:** Frontend CI run `43694724225` passed the design-token check, TypeScript/Vite production build and all 8 Playwright tests (7 Student Management journeys plus the Academic Structure curriculum editor journey). Backend CI run `43693986179` passed the complete migration chain on ephemeral PostgreSQL, backend unit/integration tests and `go vet` on backend code commit `938573ca866a73f71e8645f7ec7076f2ccea4b2a`; the subsequent backend change only clarified a code comment. These are automated CI results, not a live test of the user's local installation.

**Still required before calling this section fully accepted:**

1. Verify the user's configured API against a disposable local PostgreSQL database with migrations V25–V44. The persistent `school_management` database must not be migrated without explicit approval and a verified restorable backup. Its current version must be checked rather than assumed; the last reported local application-database version was V24.
2. Run authenticated browser acceptance against that local API/database: admission, profile load, Basic Details, current/permanent address, contact, Parent Details, medical/physical details, promotion, role restrictions, save/reopen/reload persistence and error/retry behavior. The 8 Playwright tests use mocked APIs and do not prove persistence against the user's database.
3. Complete the authoritative LGD District → Sub-District/Tehsil → Village/City reference-data import. V40 seeds State/UT values; lower levels are intentionally not fabricated. The official LGD portal requires a human CAPTCHA download; no verified common snapshot or reviewed urban-locality crosswalk is available yet.

Parent OTP/SMS and Parent login remain deferred. Aadhaar and document upload remain excluded until their dedicated security/storage controls are designed. No persistent database was modified during this review.


## Student Management Unicode validation hardening — 2026-11-15

- Student full-name, address-line and location-name limits now count Unicode code points, matching PostgreSQL VARCHAR(n) character semantics instead of Go UTF-8 byte length.
- The same rule is applied to combined admission, standalone enrollment and promotion paths, including year-scoped admission numbers and optional roll numbers; standalone legacy student create/update paths use the same student-name and legacy admission-number limits.
- Rollback-scoped PostgreSQL integration coverage now exercises maximum-length Hindi-script student names, admission numbers, roll numbers, location names and address lines, plus rejection immediately above each limit.
- The Basic Details frontend now uses Unicode code-point counting for student name and optional text fields; the student-name input no longer relies on the browser's UTF-16 `maxLength` behavior. Mocked Playwright coverage checks 200 Hindi characters are accepted and 201 are rejected without sending a save request.
- Guardian mobile and email limits also use Unicode code-point counts; rollback-scoped PostgreSQL integration coverage rejects over-limit Hindi-script contact values.
- Backend CI run `43693986179` passed the full migration chain, Go integration/unit tests and `go vet`, including the Unicode validation integration coverage. Frontend CI run `43694724225` passed all 8 Playwright tests, including the Hindi-script boundary checks. The persistent school database was not migrated or modified.

## Student Management Unicode input consistency — 2026-11-15

- Removed native HTML `maxLength` caps from admission, enrollment and guardian/relationship inputs whose server-side limits are measured in Unicode code points. Browser `maxLength` counts UTF-16 code units, so it could reject valid supplementary-plane characters before the API's PostgreSQL-compatible character validation ran.
- Backend repository validation remains authoritative for the documented limits; this change does not weaken server validation or change database schemas.
- Frontend CI run `43694724225` passed the TypeScript/Vite production build and all 8 Playwright tests on the current frontend checkpoint. Mocked Playwright coverage and live authenticated browser acceptance remain separate checks. No database migration or persistent data change was performed.

## Student address/contact Unicode input consistency — 2026-11-15

- Student contact phone length validation now counts Unicode code points, matching the backend's PostgreSQL character-limit semantics.
- Removed browser-native UTF-16 `maxLength` caps from student phone/email, address text, postal code and school-added location-name inputs. These controls no longer reject otherwise valid Unicode input based on UTF-16 code-unit counts; server-side validation remains authoritative.
- Backend CI run `43693986179` passed the migration/integration suite and `go vet`; frontend CI run `43694724225` passed the production build and all 8 Playwright tests. These are CI checks, not authenticated acceptance against the user's persistent database; no database schema or persistent data was changed by this input-behavior correction.


## Student health text validation consistency — 2026-11-15

- Removed browser-native UTF-16 maxLength enforcement from medical and physical note fields. The backend already enforces the corresponding 2,000-character and 1,000-character limits using the shared Unicode-aware repository text normalizer.
- Added frontend code-point validation for all medical notes, vision notes and physical notes. Every over-limit field now receives its own inline error and accessible description; errors clear as the user edits the affected field.
- Extended the mocked Student Management browser journey to verify that 2,001 Hindi characters are rejected and 2,000 accepted for a medical note, and that 1,001 Hindi characters are rejected and 1,000 accepted for vision notes. These are frontend interaction checks, not live database persistence tests.
- Updated the Student Management design standard so the Unicode code-point rule and prohibition on UTF-16-based native caps apply to health fields as well as identity, address, contact and guardian fields.
- No schema, migration, local database or persistent school data was changed. Exact-tip CI and authenticated browser acceptance against a real API/database remain unverified.


## Student text-limit audit across profile workflows — 2026-11-15

- Extended the Unicode input audit beyond the health section. Removed UTF-16-based native maxLength caps from Basic Details and Promotion fields; the existing Basic Details code-point checks remain in place, and promotion now validates destination admission numbers (50 characters) and optional roll numbers (30 characters) before sending the request.
- Added field-level validation for guardian relationship text (40 characters) on create/edit and guardian revocation reasons (500 characters). Removed the revocation reason's browser-native UTF-16 cap; backend repository validation already enforces the same Unicode character limit.
- Expanded the mocked browser journey to reject 121 Hindi characters in Place of Birth, 51 in a promotion admission number, 31 in a manual roll number and 501 in a guardian revocation reason before any write is sent. Existing valid workflows continue after correcting the values.
- Audited the Student Management page and Basic Details, Address, Family, Health and Promotion components for native maxLength attributes. The only remaining native cap is the two-character ISO country code, whose allowed input is ASCII uppercase by design.
- These changes are frontend validation/input behavior only. No schema or database migration was made and no persistent database was modified. CI for the exact current branch tip and live authenticated API/database browser acceptance remain unverified.


## Student address field-level validation — 2026-11-15

- Added Unicode code-point length checks to the Current/Permanent Address form for House / Street (200), Address line 2 (200), Area / Mohalla (120), Landmark (120) and PIN / Postal Code (20). All applicable errors are shown together directly beneath their fields with accessible invalid-state descriptions; editing a field clears its own error.
- Added the matching 120-character code-point check and inline error to school-created location names. Existing backend repository checks remain authoritative, and the location hierarchy validation remains a separate form-level error because it concerns the combined State → District → Sub-District → Locality path.
- Associated primary and alternate phone errors with their controls using aria-invalid and aria-describedby, matching the existing contact email accessibility behavior.
- Extended the mocked browser journey to submit all five over-limit Hindi address values together, verify every field error appears and confirm no address write occurs before correcting the values.
- No database or schema change was made. These browser checks remain mocked; exact-tip CI and live authenticated API/database acceptance are still outstanding.


## Student contact field-level validation — 2026-11-15

- Student contact email now receives a frontend 254-character Unicode code-point check before syntax validation; the error is shown beneath the email field and the backend's normalized email parser/length validation remains authoritative.
- Expanded the mocked browser journey to submit over-limit primary phone, alternate phone and email values together, verify all three field errors, and confirm no write occurs until corrected. The existing invalid-email and valid-save checks remain in the same flow.
- No database or schema changes were made. CI for the exact current branch tip and live authenticated API/database acceptance remain unverified.


## Guardian profile field-level validation — 2026-11-15

- Guardian create and shared-profile edit now validate all repository text limits before opening the shared-update confirmation or sending a write: full name 200, mobile/alternate mobile 30, email 254, occupation/education 120 and workplace 160 Unicode characters.
- Added field-specific inline errors and accessible invalid-state descriptions for every guardian profile field. Email validation checks the character limit before email syntax; the backend's normalized parser and repository character limits remain authoritative.
- Extended mocked browser coverage to submit all over-limit Hindi guardian-create values together, verify each error and no create request, then continue through invalid-email and successful create/link. Shared guardian edit now also verifies that an over-limit phone is rejected before the cross-student confirmation.
- No database or schema changes were made. Exact-tip CI and live authenticated API/database acceptance remain unverified.


## Student admission and enrollment field validation — 2026-11-15

- Combined first admission now validates student full name (200), year-scoped admission number (50) and optional manual roll number (30) using Unicode code-point counts before the create request. Each error is shown beneath its field.
- Standalone academic-year enrollment applies the same 50-character admission-number and 30-character optional roll-number checks, with field-level accessible errors. Blank roll numbers remain valid and continue to request backend auto-assignment.
- Extended the mocked admission browser journey to submit over-limit Hindi student name, admission number and roll number values together, verify all three inline errors and confirm no admission request is sent before correction.
- Backend repository validation remains authoritative. No database or schema changes were made; exact-tip CI and live authenticated API/database acceptance remain unverified.


## Guardian profile and admission input validation follow-up — 2026-11-15

- Guardian create/edit forms now validate every shared guardian profile field against the repository's Unicode character limits (name, mobile, email, alternate mobile, occupation, education and workplace) before saving. Each field has its own accessible inline error; closing/reopening forms clears stale field errors.
- Student first admission and standalone enrollment now validate the student name, admission number and optional roll number in code points before requests. The admission Playwright flow now verifies simultaneous over-limit name/admission/roll errors and no write before correction. The standalone enrollment validation is implemented but does not yet have a dedicated mocked browser scenario.
- Guardian Playwright coverage now verifies all over-limit fields together on create and an over-limit mobile number on shared-profile edit before the confirmation step.
- All these browser scenarios use mocked APIs. No local/persistent database was changed; exact-tip CI and authenticated browser acceptance against a real API/database remain outstanding.


## Student location country-code validation — 2026-11-15

- Tightened school-created State/UT location validation: country codes must now be exactly two ASCII uppercase letters after normalization. A two-byte non-ASCII character or arbitrary punctuation/digit pair can no longer pass the previous byte-length-only check.
- Added rollback-scoped PostgreSQL integration assertions for invalid non-ASCII and non-letter country codes alongside the existing valid `IN` location hierarchy and Unicode location-name/address tests.
- This is repository validation/test coverage only; no migration or persistent database change. Exact-tip backend CI remains unverified.


## Verified Student Management validation checkpoint — 2026-11-15

- Exact code checkpoint `6e3085218c6615b154d67edc6518c598b9d7ddf9` passed Frontend CI (`43673900145`): centralized design-token check, `tsc -b`, Vite production build and 5 Playwright Student Management scenarios.
- The same code checkpoint passed Backend CI (`43673900042`): all migrations applied to ephemeral PostgreSQL, complete backend tests including PostgreSQL integration, and `go vet`.
- The Playwright scenarios are API-mocked browser tests. They verify UI validation and interactions but do not prove persistence against the user's local database. Real authenticated admission/profile/Parent Details/address/contact/health/promotion save-reopen-reload acceptance remains outstanding.
- No persistent `school_management` database was accessed or migrated during this work.


## Official LGD acquisition path documented — 2026-11-15

- Verified the Ministry of Panchayati Raj's official LGD portal and its Download Directory workflow. It exposes state-wide/entity-level downloads for districts, sub-districts, villages and development blocks, plus date-scoped modifications; report generation is CAPTCHA-gated and offers PDF/HTML/XLS/ODT output.
- Documented a human-downloaded, immutable source snapshot workflow with retrieval metadata/checksum, reviewed field crosswalk, parent/code reconciliation, dry-run validation and versioned delta imports. CAPTCHA must not be automated or bypassed.
- Clarified that LGD Development Blocks and Sub-Districts/Tehsils are distinct levels; a village's block coverage must not be silently flattened into the sub-district parent chain used by the address selector.
- Official LGD download page: https://lgdirectory.gov.in/downloadDirectory.do. OGD catalog discovery: https://data.gov.in/catalog/local-government-directory-lgd. No LGD data file has been downloaded or imported; actual bulk import remains blocked until a human-procured official snapshot is available and its headers/hierarchy are reviewed.


## Standalone enrollment browser coverage and verified checkpoint — 2026-11-15

- Added a dedicated mocked browser journey for enrolling an existing student into an academic year. It verifies that 51-character admission numbers and 31-character roll numbers are rejected together with field-level errors and no POST, then confirms a valid enrollment is submitted with a blank roll number preserved as `null` for backend auto-assignment.
- Exact code checkpoint `02f5b1f5ea1bc9000a55e93da399a6abb9d06f72` passed Frontend CI (`43674731005`): centralized design-token check, TypeScript/Vite production build, and all 6 Student Management Playwright scenarios.
- The same checkpoint passed Backend CI (`43674731037`): full migrations on ephemeral PostgreSQL, backend unit/integration tests, and `go vet`.
- The browser scenarios still mock APIs; they do not prove data persistence against the user's local database. Live authenticated API/database acceptance remains pending. No persistent `school_management` database was accessed or migrated.


## Health-note Unicode boundary integration checkpoint — 2026-11-15

- Extended rollback-scoped V36 PostgreSQL integration coverage for all four medical note fields at exactly 2,000 Hindi Unicode characters and rejection at 2,001; also covered physical vision notes at 1,000/1,001 and physical notes at 2,000/2,001.
- Exact code checkpoint `f5b41c6de228997eaddbe93271718f5c5183a980` passed Backend CI (`43675249107`): full migration chain on ephemeral PostgreSQL, backend unit/integration tests, and `go vet`.
- Existing frontend browser coverage already verifies the matching health-field Unicode boundaries, inline errors, and reload of saved health details. No frontend code changed in this checkpoint.
- No persistent `school_management` database was accessed or migrated. Real local API/database browser acceptance remains outstanding.


## LGD pre-import validator and location terminology hardening — 2026-11-15

- Corrected the V34 location integration fixture and current design documentation to name the database `subdistrict` level **Sub-District / Tehsil**, not Block/Tehsil. The current address path is State/UT → District → Sub-District/Tehsil → Village/City. Development Blocks are distinct and are not currently representable in `student_locations`; any future Block data needs a separate reviewed reference relationship/crosswalk, not substitution into the revenue hierarchy.
- Audited the read-only `backend/cmd/lgdvalidate` contract against V42. Removed an incorrect global cross-level source-code collision check: V42 deliberately uses entity-specific `source_system` namespaces (`LGD_STATE`, `LGD_DISTRICT`, `LGD_SUBDISTRICT`, `LGD_VILLAGE`), so identical code strings in different entity classes are valid. Duplicate/conflicting codes remain rejected within each entity's namespace.
- Hardened normalized snapshot validation to reject invalid UTF-8 and NUL bytes that PostgreSQL text columns cannot store. Added optional `-source-file` checksum-only input so the validator reports both the normalized CSV SHA-256 and the untouched official export SHA-256 without parsing or modifying the original file.
- Added unit tests for entity-namespace code reuse, invalid UTF-8, NUL bytes and raw-file checksum calculation. Exact code checkpoint `0598d160a9e69749813f4298bb321c00897f5ccb` passed Backend CI (`43676041876`): complete migration chain on ephemeral PostgreSQL, backend unit/integration tests, and `go vet`. The earlier terminology fixture correction passed Backend CI (`43675472024`).
- The validator remains read-only. No official LGD snapshot has been obtained, normalized, or imported; raw-export schema, urban-locality mapping and any Block relationship still require review against a human-downloaded official source file. No persistent school database was accessed or migrated.
- Refreshed OGD source discovery on 2026-11-15: the catalog page showed an update date of 2026-10-28 and identifies Ministry of Panchayati Raj; the District resource showed 2026-10-27, while the indexed Village resource showed 2025-11-17 and the Sub-District resource did not expose a reliable current update date. These are portal metadata only, not a verified common export snapshot. Obtain the exact official exports together and reconcile their embedded dates, headers, codes, parent fields and counts before writing an importer; do not fill stale/incomplete data from community mirrors.


## Basic Details Unicode boundary verification — 2026-11-15

- Added rollback-scoped PostgreSQL integration coverage for the Basic Details text fields: Place of birth accepts exactly 120 Hindi Unicode characters and rejects 121; Nationality and Religion accept 80 and reject 81; School registration number accepts 50 and rejects 51. Repository validation and PostgreSQL storage are checked together.
- Extended the Student Management Playwright workflow to assert all three 80/50-character field errors appear together under their own fields and that no invalid form is submitted. The earlier place-of-birth over-limit check remains in the same workflow.
- Frontend CI run `43676902824` passed all 6 Student Management Playwright tests and the TypeScript/Vite production build on `0bb83f06ed1b7923a8b9bc3eb92ea94471110515`. Backend CI run `43676936136` passed all migrations, backend unit/integration tests (including the new V37 Unicode boundaries), and `go vet` on `fbbe1a6107623de1788458c697e747899d1e9b40`.
- This verifies the code and mocked browser workflow plus ephemeral PostgreSQL integration; it does not replace authenticated browser acceptance against the user's local API/database. No persistent school database was accessed or migrated.


## Basic Details registration conflict feedback — 2026-11-15

- School registration-number uniqueness conflicts returned by the API are now mapped to an inline field error beneath School registration number, rather than a generic form-level error. The field's `aria-describedby` and `aria-invalid` state carry the error to assistive technology; changing the field clears the error.
- The Playwright workflow now simulates an API 409 conflict, verifies the field-level message, verifies the existing profile value is not overwritten, then retries successfully after the conflict is cleared.
- Backend CI `43677196532` and Frontend CI `43677196535` both passed on `8a2c32e15deb8464c58b40d0d89fe0ae5187f1a5`. Frontend includes all 6 Student Management Playwright tests; backend includes the full ephemeral PostgreSQL migration chain, integration tests and `go vet`.
- These checks do not replace authenticated browser acceptance against the local API/database. No persistent school database was accessed or migrated.


## Guardian-link conflict feedback — 2026-11-15

- A duplicate active guardian-link conflict (HTTP 409) is now mapped to an inline error beside the Guardian selector in Parents & Siblings. The selected guardian and relationship form values are retained so staff can review existing links and retry; other unexpected errors remain in form-level feedback.
- Extended the guardian workflow Playwright test to simulate the conflict, assert that no link is created, verify the inline message, and then retry successfully. Existing relationship type validation and primary/emergency flags remain covered.
- Frontend CI run `43677649489` and Backend CI run `43677649516` passed code checkpoint `58b255d2c16a755e0cc39ad86c96675267f1482b`. This includes all 6 Student Management Playwright tests, TypeScript/Vite build, full migrations on ephemeral PostgreSQL, backend unit/integration tests, and `go vet`.
- No persistent school database was accessed or migrated. Authenticated browser acceptance against the local API/database remains pending.


## Archived student Parent Details view-only browser coverage — 2026-11-15

- Extended the archive/retry Student Management browser journey to reopen an archived student's Parents & Siblings section when the operator has `guardians.read` but the student is archived.
- The test verifies that existing guardian name, relationship, primary/emergency flags, mobile/email, occupation, education and workplace details remain visible after archival, while relationship editing, shared guardian editing, revocation, linking and new guardian creation controls are absent.
- This protects the distinction between retaining historical student/guardian information for read access and preventing further edits to an archived student. Parent OTP/login remains deferred.
- Extended assertions: `76bbd02b742579bf603c444c649d80fdf16d4125` also checks that mobile/email and occupation/education/workplace values are preserved in the archived read-only view. Frontend CI run `43692374206` passed all 7 Student Management Playwright tests and the production build, including this archived Parent Details read-only scenario. No backend, migration or database changes were made for this browser assertion.


## Inactive / archived student guardian-revocation boundary — 2026-11-15

- Closed a backend authorization/business-rule gap: guardian relationship revocation now locks the owning student only when its status is `active`, matching the existing link-creation and relationship-edit rules. Direct API calls can no longer revoke a relationship belonging to an inactive or archived student merely because the frontend hides the control.
- Added PostgreSQL integration coverage for both inactive and archived student states: relationship editing/revocation return the not-found boundary, and the guardian link remains unrevoked in history.
- Code commits on `work`: `3ac8d275d39de264560086a67c42df62012689c8` and `7c56040a64f9ee39c04384662f3a30bd59379ac8` and archived-state follow-up test `a8ae00df0a19b22c935ff1a2f93715f36fe0bd13`. Backend CI run `43692087083` passed the full migration chain, PostgreSQL integration tests and `go vet`, including the inactive/archived guardian-history boundary. No persistent database changes were made.


## Inactive student Parent Details read-only enforcement — 2026-11-15

- Student Management now enables guardian create/link/edit/revoke controls only when the selected student is `active` and the operator has `guardians.manage`. Inactive and archived students continue to expose their existing Parent Details to users with `guardians.read`, but the entire family workflow remains read-only.
- Aligned Basic Details and Address & Communication edit controls with the backend's existing active-student write boundary. Previously, inactive students could see edit controls because the frontend excluded only archived records, while the repository rejected writes for every non-active status. Medical & Physical was already restricted to active students.
- Removed the misleading directory-level Edit profile shortcut for inactive students. They retain the separate View profile action, while active students retain the edit shortcut. This keeps the directory affordance consistent with the read-only profile sections.
- Closed the matching backend gap in `UpdateStudent`: the repository previously rejected archived names but still accepted direct name updates for inactive students. The write now requires `status='active'`, and the PostgreSQL integration test marks a student inactive, verifies the direct update returns `ErrAcademicNotFound`, then restores the fixture before continuing archive assertions.
- Updated the read-only explanation so it states both requirements: active student status and guardian management permission.
- Added a Playwright browser acceptance scenario for an inactive student where the operator deliberately has `guardians.manage`; the test asserts that existing guardian identity, relationship, mobile/email, occupation, education and workplace remain visible while all mutation controls are absent. Updated the archived-student scenario to assert the same guidance. The same inactive-student scenario now also verifies Basic Details and Address & Communication are view-only even when the operator has `students.update`.
- Commits on `work`: `91811d836a26d8163cd55eaa5e4f048fbd7e39d2`, `27da307e470bc5db1222903281fdb1004dde9528`, `8b8d2198a94e16b26bb66bf17fd0d4f530adb831`, `20ad91248cef8ae4a108315f790f141f3d31dc67`, `67658bec214c984523423786ca8e6d3ea13be6ed`, `d60bcc0b545069aed4c77772ac471a9d01ecb24c`, assertion correction `048dbb4c9945e1a17d0222d35b1d3796e8bc886d`, least-privilege fixture correction `20023ccd7896f0cf675589361d96302f2272cf90`, update-permission coverage `25be81dda97dd4ef9d27fb425e930116bf358413`, directory edit-action fix `061b44d8c06429b75c04cb8316a7bd06100da371`, its browser assertion `d23e001ad51111e5156b7fa05c915fd99d1c2ecb`, backend active-only name update `347bcdf6f0dade4aa7fddc9367d91df935995a0d`, and PostgreSQL integration coverage `eb24c2523c1bfd017d816374863e2475c244fbf6`. Active-student edit-shortcut regression assertion `1c1ad7b71dcc3da4bf4e237d5892fbb79689b1ba`. Frontend CI run `43692374206` passed all 7 Student Management Playwright tests, including the inactive-student Parent Details view-only scenario. No local Playwright execution occurred in this session.


## Student archive closes active enrollment and blocks new attendance — 2026-11-15

- Closed a lifecycle gap: archiving a student now transitions any active academic-year enrollment to `withdrawn` in the same transaction, with an end date no earlier than the enrollment start date. Enrollment rows and existing attendance history are retained; no historical record is deleted or rewritten.
- Attendance marking now requires and locks the owning student record as active. This blocks direct API attendance writes for inactive/archived students and serializes a concurrent mark against archive; historical attendance reads remain preserved.
- Promotion now locks the permanent student row before the enrollment row, matching the archive/create lock order; inactive students cannot be promoted through a direct API request. A PostgreSQL integration regression test confirms the source enrollment remains active after a rejected promotion.
- Enrollment-list responses now include the owning student's status for authorized full roster views; the Student Management UI shows Promote only when both the enrollment and student are active. Subject-only teachers continue to receive only the approved four-field minimal roster.
- Added PostgreSQL integration coverage for archive → withdrawn enrollment, retained enrollment history, and rejection of new attendance marks for archived students.
- Backend CI run `43692087083` and frontend CI run `43692374206` passed the archive/enrollment/attendance lifecycle regression coverage. The checks used an ephemeral PostgreSQL database and mocked browser APIs; no local database was migrated or modified.


## Curriculum and enrollment subject-selection foundation — 2026-11-15

- Added V44 schema for versioned class/year curricula, compulsory subjects, named optional-subject groups with min/max selection rules, per-enrollment curriculum assignments, and append-only optional-subject selection history.
- Added database guards for immutable curriculum identity, no curriculum-child edits after assignment, active-student/active-enrollment assignment, optional-only subject selection, and history-preserving replacement/revocation.
- Added repository workflows to list/save class curriculum versions, read enrollment curriculum and subject history, and validate/save optional choices transactionally with school scoping, live permission checks and audit events.
- Added service methods, HTTP handlers and routes: `GET/POST /school/class-curricula`, `GET /school/enrollments/:enrollmentID/curriculum`, and `PUT /school/enrollments/:enrollmentID/subjects`.
- Added student-profile enrollment UI to display compulsory subjects, optional groups, selection limits, saved choices and previous choice history; active students/enrollments with both `students.update` and `subjects.read` can save choices. Inactive records remain view-only.
- Added route permission-gate tests for curriculum setup/read and enrollment subject-choice endpoints.
- Completed: class/year curriculum setup UI, versioned enrollment assignments, compulsory/optional choice validation, append-only history, repository and database guards, PostgreSQL integration coverage, and browser acceptance for first assignment/save/refresh/history. Backend CI run `43693986179` passed migrations, integration tests and `go vet`; frontend CI run `43693753551` passed the production build and all 7 Student Management Playwright tests.

## Remaining approved Student Management scope — curriculum and subject selection

- The Academic Year + Class + Section curriculum and enrollment-specific subject-choice workflow is implemented. Each section has one editable current curriculum record; edits do not create new versions or reassign students. Compulsory subjects are inherited from the curriculum; standalone optional subjects and optional groups carry configured student-choice rules. Student choices are current state only. Confirmed subject removal permanently deletes the subject and linked current choices after exact impact confirmation; removing a group makes its subjects standalone optional and preserves their current choices.
- Repository integration tests verify first assignment, read-after-write, current-choice replacement, confirmed subject removal, optional-group removal, invalid selection rejection and tenant isolation. Legacy V44 version/history behavior is superseded.
- Student transport, app-permission and document-upload submodules remain outside this immediate completion target; secure document upload is deferred. Parent OTP/SMS/login remains deferred.

## First curriculum assignment guidance — 2026-11-15

- Historical V44 selector behavior; current UI identifies the student's current assignment and the current section curriculum without offering curriculum versions.
- Current guidance distinguishes a first assignment from switching to the current section curriculum; changed subject choices replace the prior current rows and no history is retained.
- Frontend fix: `cc1340616fa1e73913a6280354a4b00cfa50327d`. Frontend CI run `43692374206` passed the Student Management browser suite and production build, including first-assignment guidance.

## First student curriculum assignment messaging — 2026-11-15

- Student Enrollment Subjects explains first assignment and switching to the current section curriculum. The panel does not refer to newer versions or preserved subject-choice history.
- Code commit: `9b4ce26355f76a212a3abe5af87c6ebc268977af`. Frontend CI run `43692374206` passed all 7 Student Management Playwright tests, including the first-assignment guidance.

## Student Management — curriculum cache coherence after class revisions (2026-11-15)

- Fixed a stale-profile edge case: saving a class/year curriculum previously refreshed the class-curriculum list but did not invalidate already-open student enrollment curriculum queries. A profile could therefore continue showing the previous available/assigned curriculum until its query became stale or the profile was reopened.
- Class curriculum saves now invalidate both the matching class/year curriculum list and the enrollment-curriculum query family, so open student profiles refetch and expose the latest version state. Student-specific subject saves continue to invalidate only the affected enrollment curriculum.
- Code commit: `b2d3d225fa50b0800c8ed76e66869c561897cb5c`. Frontend CI run `43692374206` passed the production build and Student Management browser suite; the curriculum cache invalidation behavior is included in the reviewed frontend code.

## Student Management — enrollment curriculum browser journey (2026-11-15)

- Historical V44 test description; the current test asserts compulsory/optional choice behavior, current selection persistence and no previous-choice history.
- The test uses stateful mocked API responses so the post-save refetch returns the saved assignment and choices instead of resetting to the initial unassigned fixture. It also verifies that an explicit curriculum refresh does not overwrite a different unsaved optional-subject choice.
- Repository integration coverage rejects missing required optional choices, compulsory-subject IDs and duplicate selected IDs without creating an assignment; rejects cross-school curriculum reads/writes; and prevents reverting an enrollment to a retired curriculum version after replacement while preserving the active version and history.
- Test commits include `152e0bc044137c1ec07310cf284fcd76d6f4c554`, refresh regression `fec014334c1cf91f8042b4f0e670dbf8de514e30`, invalid-choice checks `b3ea8f92e2c2d55b6038d1516b983b71ca3f2c25`, tenant-isolation checks `1f17c7f6e5baa82fe08ae4029126c90c708bddf1`, retired-version protection `39c9ce1f3cd58e394fa9223009e0c90aa83c1f00`, and final browser locator correction `64adf774ed1f2e94618b0ce8d0ae37ee247e6554`. Frontend CI run `43694724225` passed all 8 Playwright tests, including the new academic-structure curriculum editor scenario; backend CI run `43693986179` passed PostgreSQL integration and `go vet` on backend commit `938573ca866a73f71e8645f7ec7076f2ccea4b2a`. Browser coverage is mocked, not live database acceptance.

## Student Management — CI regression follow-up (2026-11-15)

- The first CI run for the new curriculum browser coverage exposed a test-only assertion issue: subject-choice history is intentionally inside a collapsed disclosure. The Playwright test now opens the history disclosure before asserting its saved entry.
- The backend integration run exposed a stale guardian-repository assertion that still expected inactive-student guardian revocation to succeed. This conflicts with the current Student Management rule that inactive and archived student family records are view-only. The integration test now expects `ErrGuardianNotFound` and verifies the existing guardian history remains unrevoked and unchanged.
- Fix commits: `5bc2c0033277511cd344259a2f0c1dbf52fc5738` (browser history disclosure) and `3fc629db3fa8045ecae1619b6a0df9e3a5a6bc73` plus `76a67a9e2b4446b6f68ee6264651bac44357be04` (inactive/archived guardian history and active-fixture restoration). Backend CI run `43692087083` passed PostgreSQL integration and `go vet`; frontend CI run `43692374206` passed all 7 Playwright tests.

## Student Management — Parent Details edit, validation and reload acceptance (2026-11-15)

- Expanded the active-student family workflow browser test to verify that existing guardian mobile/email values load into the edit form, all contact/profile fields validate together at their documented limits, and a confirmed shared guardian profile update saves name, mobile, email, alternate mobile, occupation, education and workplace.
- The same browser journey edits the student-specific relationship and contact flags, then closes and reopens Parents & Siblings to verify the saved guardian details and relationship are loaded again.
- This is staff-managed Parent Details only. Parent OTP, parent authentication and paid SMS remain deferred; no parent login is created by this workflow.
- Frontend CI run `43692374206` passed the expanded browser journey as part of all 7 Student Management Playwright tests. The test uses mocked API responses; live persistent-database/browser acceptance remains a separate pending verification.

## Student Management — V44 curriculum validation and tenant-isolation hardening (2026-11-15)

- Repository integration tests now prove invalid subject selections (missing required optional choice, selecting a compulsory subject as optional, and duplicate IDs) return `ErrInvalidAcademicInput` without creating an enrollment curriculum assignment.
- Cross-school reads and writes for enrollment curriculum, enrollment subject choices and class curriculum management are rejected with `ErrAcademicAccessDenied`.
- Once a student enrollment has moved from curriculum v1 to v2, attempting to assign the retired v1 again returns `ErrAcademicNotFound`; the active v2 selection and append-only history remain unchanged.
- Backend CI run `43693986179` passed migrations, Go unit/integration tests and `go vet` for commit `938573ca866a73f71e8645f7ec7076f2ccea4b2a`. The integration database was ephemeral; the persistent school database was not changed.

## Student Management — curriculum authorization and save/readback consistency (2026-11-15)

- The enrollment curriculum GET route now requires both `subjects.read` and `students.read_basic`, matching the repository's live authorization checks and enrollment visibility rules.
- Saving subject choices requires `students.update`, `subjects.read` and `students.read_basic`. The repository checks all three before beginning the write so a caller cannot commit a choice and then receive a 403 when the response reloads the enrollment curriculum.
- The Student Management profile only renders the curriculum panel when the operator has both student basic-read and subject-read permissions; the save controls additionally require student update permission.
- Route tests verify that each required permission is independently enforced. Backend CI run `43693986179` passed the full migration chain, Go unit/integration tests and `go vet`.

## Student Management — curriculum editor pending-save protection (2026-11-15)

- Academic year/class selectors and curriculum group/subject controls are disabled while a curriculum save is in flight. This prevents switching the editor context or changing its draft while the success handler is about to clear the completed form.
- Optional-group validation now places name, minimum, maximum and subject-assignment errors beside the corresponding control/group. Dependent errors clear when the group name, selection limits or subject membership changes. Group-name normalization uses locale-independent lowercase to match the Go backend's comparison.
- Added `frontend/e2e/academic-structure.spec.ts` to exercise group-level subject assignment validation, minimum/maximum field errors, a successful version save, and disabled editor controls while the save is pending.
- Frontend CI run `43694724225` passed design-token checks, TypeScript/Vite production build and all 8 Playwright tests.

## Student Management — class curriculum editor validation and pending-save acceptance (2026-11-15)

- Optional-group name errors are displayed below the group-name field; minimum and maximum bounds are reported below their own number inputs; missing group membership is reported at the affected optional group. Editing a dependent field clears its stale validation message.
- Optional group name matching is locale-independent and consistent with backend normalization, avoiding client/server duplicate-name disagreement on devices using a non-English locale.
- The new Academic Structure Playwright test verifies required group membership, minimum-versus-maximum validation, saving a version, and that year/class/group/subject controls remain disabled until the pending save completes.
- Frontend CI run `43694724225` passed the production build and all 8 Playwright tests. No backend schema or persistent database was changed by this frontend validation improvement.


## Student Management — sensitive profile write authorization and physical measurement dates (2026-11-15)

- Student address, basic details, contact details, medical details and physical details write routes now require both `students.update` and `students.read_sensitive`. The repository methods repeat both live-permission checks inside their transactions so direct repository calls cannot bypass the route policy.
- Student route regression tests cover each of the five write endpoints and verify that every required permission is independently necessary; a caller missing either permission is denied.
- Physical measurement dates are limited to today or earlier in the editor and repository validation. PostgreSQL integration coverage verifies that a future measurement date is rejected; measurement history remains append-only.
- Backend CI run `43696751599` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet`. Frontend CI run `43696751608` passed the TypeScript/Vite production build and all Student Management Playwright tests.
- These runs use ephemeral CI databases and mocked browser APIs. Local authenticated acceptance remains pending; persistent database migration remains approval-gated.


## Student Management — promotion date bounds (2026-11-15)

- The frontend academic-year type now exposes the existing backend `starts_on` and `ends_on` values to Student Management forms.
- Student promotion date selection is constrained to the destination academic year and cannot precede the source enrollment date. The initial date is adjusted into the valid range when the destination year is selected; an impossible date window is reported and submission is blocked.
- The promotion browser fixture now includes real academic-year date bounds. Frontend CI run `43697071621` passed the production build and all 8 Playwright tests; backend CI run `43697071585` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet` for code checkpoint `094e80b37b48481567a3ccde943534c461b58141`.
- CI uses mocked browser APIs and an ephemeral PostgreSQL database; local authenticated browser/database acceptance remains pending.


## Student Management — admission and enrollment date bounds (2026-11-15)

- The Student Management academic-year API type now requires the existing `starts_on` and `ends_on` values returned by the backend.
- New student admission and standalone enrollment forms constrain their admission date pickers to the selected academic-year range, validate the range before submit, and adjust the initial date into the selected year when necessary.
- Promotion uses the same academic-year boundaries plus the source enrollment date as its minimum. Its date picker and submit validation prevent dates outside the destination year or earlier than the source enrollment.
- Shared admission E2E fixtures now include academic-year date bounds. Frontend CI run `43697440509` passed the production build and all 8 Playwright tests; backend CI run `43697440503` passed the full migration chain, PostgreSQL integration tests and `go vet` for code checkpoint `1d4b998fe05bae4162b03edabef9201df818cea6`.
- Local authenticated browser/database acceptance remains pending; persistent database migration remains approval-gated.


## Student Management — future-year admission identity and Unicode directory search (2026-11-15)

- Student directory and profile reads now select the admission number from the preferred active enrollment: current academic year first, then the most recent active enrollment when a student is pre-admitted into a future year. This avoids blank admission numbers before the destination year becomes current while preserving the current-year number when one exists.
- Student directory search limits now count Unicode code points rather than UTF-8 bytes, so valid Indian-language admission-number searches within the 100-character limit are accepted.
- PostgreSQL integration coverage verifies directory search and profile loading for a student admitted only into a non-current active academic year, and rejects searches exceeding 100 Unicode characters. It also verifies read-audit metadata remains free of student names and admission numbers.
- Backend CI run `43698230677` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet` for code checkpoint `7b3268a2beb14afe63638964c70c06032f4f303d`. Persistent school database remains unchanged.


## Student Management — Unicode location search validation (2026-11-15)

- The hierarchical student-location search now enforces its 100-character limit by Unicode code points, matching student directory search and the PostgreSQL-compatible field validation rules. Valid Hindi-script searches are no longer rejected merely because their UTF-8 representation uses multiple bytes per character.
- Integration coverage searches a 50-character Hindi location name successfully and rejects 101 Unicode characters. Backend CI run `43698497479` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet` for code checkpoint `8485e909a9de75b9f775e8ff9e5b0268fcbe0740`.


## Student Management — enrollment history read auditing (2026-11-15)

- Student academic enrollment history reads now write an append-only audit event in the same database transaction as the authorized query. If the audit write fails, the read does not return as a successful unaudited operation.
- The event records the section, result count and whether an academic-year filter was applied. It does not copy student names, admission numbers or other personal details into audit metadata; a single-student history read uses the student resource reference.
- PostgreSQL integration coverage verifies that the history read produces exactly one event with the expected metadata and no student-name/admission-number leakage.
- Backend CI run `43701689198` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet` for code checkpoint `82f9f7a3cca25841ea0377053745020e06079fab`. The CI database was ephemeral; the persistent school database was not changed.


## Student Management — enrollment and curriculum read auditing (2026-11-15)

- Student enrollment history collection and single-student reads write append-only audit events in the same transaction as the authorized query. Metadata contains result counts and filter context, not student names or admission numbers.
- Student enrollment curriculum reads emit a dedicated `school.student.enrollment_curriculum.read` event after validating both `students.read_basic` and `subjects.read`. Audit metadata records the current selected-option count and curriculum presence, without subject names or student choice history. The read is not returned if the audit write fails.
- PostgreSQL integration coverage verifies single-student enrollment history auditing, teacher-scoped collection auditing, and current curriculum-selection audit metadata.
- Backend CI run `43702682493` passed the complete migration chain, Go tests including PostgreSQL integration, and `go vet` for code checkpoint `f549674007ff64c79dbb0bcdcc3d27708b1f0baa`. The CI database was ephemeral; the persistent school database was not changed.


## Student Management — final CI checkpoint (2026-11-15)

- Enrollment history and enrollment curriculum read-audit coverage is included in the verified backend checkpoint.
- Backend CI run `43703198391` passed the complete migration chain, Go tests including PostgreSQL integration, and `go vet` for `1391a23594c75eed58e281acef6515aa3aa418b2`.
- Frontend CI run `43703198426` passed design-token checks, TypeScript/Vite production build, and the Student Management Playwright acceptance suite for the same checkpoint.
- CI databases and browser APIs are ephemeral/mocked. Local authenticated browser acceptance against the user's database remains outstanding; do not migrate the persistent database without explicit approval and a verified restorable backup.


## Student Management — search input bounds (2026-11-15)

- Student directory and active-student enrollment search fields now cap input at 100 characters, matching backend search validation.
- State/UT, district, sub-district/tehsil and village/city location search inputs also cap at 100 characters, avoiding requests that the location repository would reject.
- Backend CI run `43703502100` passed the full migration chain, Go tests including PostgreSQL integration, and `go vet`; frontend CI run `43703502092` passed design-token checks, TypeScript/Vite production build, and the Student Management Playwright acceptance suite for checkpoint `d4680a19b8a5e899ee535fb92ad950b1d84c0fd7`.


## Student Management — shared Unicode text normalization unit coverage (2026-11-15)

- Added focused backend unit tests for the shared text normalizer used by student Basic Details, Address, Contact and Health persistence: nil values, whitespace-only values, trimming before counting, exact Hindi Unicode-code-point boundaries, and one-code-point-over rejection.
- Test commits: `7644776e2392d7bd28468d80b4bed2460332755e` (shared Unicode normalizer unit boundaries) and `5a031c694f839338d4e766f5f6a72fcaf525dbea` (PostgreSQL contact repository rejects a 31-code-point phone value and a display-name email address).
- Backend CI **PASS** for exact test commit `5a031c694f839338d4e766f5f6a72fcaf525dbea` — [run 43712579620](https://github.com/TheSumitNain/school-management-system/actions/runs/43712579620), including full migrations, PostgreSQL integration tests and `go vet ./...`. The contact validation assertions run before any database write; no persistent school database or school data was changed.


## Student Management — guardian full-name edit regression coverage (2026-11-15)

- Extended the active Parent Details browser journey to validate the guardian full-name 200-Unicode-character limit, save a changed guardian name through the shared guardian profile endpoint, and verify the new name after closing and reopening Parent Details.
- Corrected the mocked PATCH response to return the guardian profile shape and update the linked guardian's displayed name, so the reload assertion checks the actual shared-profile read path rather than an unrelated stale fixture field.
- Test commits: `bc869769b2a22e43548266f4ea6bcfabea449c4a` (full-name edit/validation) and `fc43c199b48a26b08c15a33e86ff28115c8b244c` (renamed name is retained in revoked-history assertions). Frontend CI **PASS** on exact test checkpoint `fc43c199b48a26b08c15a33e86ff28115c8b244c` — [run 43712924033](https://github.com/TheSumitNain/school-management-system/actions/runs/43712924033): production build and all 8 Playwright tests passed, including guardian full-name validation, save, revoke-history display and reload. This is mocked browser coverage, not live local-database acceptance.


## Student Management — complete shared guardian profile edit coverage (2026-11-15)

- Extended the same Parent Details browser journey to validate mobile, alternate mobile, occupation, education and workplace limits together, then save changed values for all of those fields plus guardian full name, email and alternate contact.
- The test now checks the persisted shared-profile response and verifies all edited values after closing and reopening Parent Details. The fixture models the returned guardian profile separately from the student-guardian relationship and keeps both views coherent.
- Test commit: `7e78fd01bc88277d420b276b3ecb82f42eba1753`. Frontend CI **PASS** on exact test checkpoint `7e78fd01bc88277d420b276b3ecb82f42eba1753` — [run 43713194822](https://github.com/TheSumitNain/school-management-system/actions/runs/43713194822): production build and all 8 Playwright tests passed, including field-level validation and save/reload of the full shared guardian profile. Authenticated live acceptance against the local API/database remains outstanding.


## Student Management — guardian relationship field-level regression (2026-11-15)

- Extended the Parent Details Playwright journey to reject a 41-code-point Hindi relationship label at the 40-character boundary without sending a PATCH request.
- The same journey now edits the student-specific relationship label and clears both Primary contact and Emergency contact flags, verifies the exact PATCH payload, and confirms the changed flags remain cleared after the shared guardian profile update and Parent Details reload.
- This protects the distinction between shared guardian profile fields and per-student relationship fields. Existing backend PostgreSQL integration coverage continues to verify relationship authorization, school/student scope, primary-contact behavior, and append-only update audit metadata.
- Test commit: `c4a221ad24048333bafc2dda2f52d9c9cc8542eb`. Frontend CI **PASS** — [run 43713791921](https://github.com/TheSumitNain/school-management-system/actions/runs/43713791921): production build and all 8 Playwright tests passed, including the new 41-code-point rejection and per-student contact-flag updates. Backend CI **PASS** — [run 43713791860](https://github.com/TheSumitNain/school-management-system/actions/runs/43713791860): full migration chain, PostgreSQL integration suite and `go vet ./...` passed on the same test checkpoint. Authenticated local API/database acceptance remains outstanding.


## Student Management — cross-student draft isolation browser coverage (2026-11-15)

- Added a two-student browser journey that opens Student A's Basic Details, enters an unsaved draft and a field-level validation error, closes that section/profile, then opens Student B.
- Verified Student B receives its own saved basic details and neither Student A's unsaved value nor Student A's validation error appears.
- Test commit: `433461bec48f502765b97464ba6e188a9324d687`. Frontend CI **PASS** — [run 43715113384](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113384), production build and all 9 Playwright tests passed. Backend CI **PASS** — [run 43715113361](https://github.com/TheSumitNain/school-management-system/actions/runs/43715113361): full migrations, PostgreSQL integration tests and `go vet ./...` passed on the same checkpoint. This is mocked browser coverage; authenticated local API/database acceptance remains outstanding.


## Student Management — cross-student Parent Details draft isolation (2026-11-15)

- Added a second two-student browser journey for Parent Details. Student A's new-guardian form is opened, populated with an over-limit Unicode name and a relationship draft, and submitted to create a field-level validation error.
- After closing Student A's Parent Details and profile, the test opens Student B's Parent Details and verifies the new-guardian form starts empty and Student A's validation error is absent.
- Latest test checkpoint: `b6d14c6ed4030724857838ef8cebc131041219ae`. Frontend CI **PASS** — [run 43716057611](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057611), production build and all 10 Playwright tests passed. Backend CI **PASS** — [run 43716057614](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057614), full migration chain, PostgreSQL integration tests and `go vet ./...` passed. These are mocked browser and ephemeral-DB CI checks; authenticated local API/database acceptance remains outstanding.


## Student Management — Parent Details cross-student regression test selector correction (2026-11-15)

- The first CI attempt exposed a case-sensitive test locator and the follow-up edit exposed a duplicate regex flag in pre-existing Parent Details selectors. Corrected the selectors; no application source change was required.
- Final test checkpoint: `b6d14c6ed4030724857838ef8cebc131041219ae`. Frontend CI **PASS** — [run 43716057611](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057611), production build and all 10 Playwright tests passed, including both Basic Details and Parent Details cross-student draft-isolation journeys. Backend CI **PASS** — [run 43716057614](https://github.com/TheSumitNain/school-management-system/actions/runs/43716057614), full migrations, PostgreSQL integration tests and `go vet ./...` passed. Authenticated local API/database acceptance remains outstanding.

## Student Management — guardian email validation regression coverage (2026-11-15)

- Backend PostgreSQL integration coverage now rejects malformed guardian email values, including a display-name address and a comma-separated multiple-address value, in addition to the existing over-limit Unicode email and malformed single-address cases. The repository rejects these inputs before issuing a database write.
- Parent Details browser coverage exercises the same invalid email categories in both shared guardian profile editing and new guardian creation, verifies field-level errors, and confirms no profile/create request changes persisted data for invalid input.
- Test checkpoint: `da85edc9f020c7db2d0b474c4855e3be7249bf80`. Frontend CI **PASS** — [run 43724412256](https://github.com/TheSumitNain/school-management-system/actions/runs/43724412256): production build and all 11 Playwright tests passed. Backend CI **PASS** — [run 43724412200](https://github.com/TheSumitNain/school-management-system/actions/runs/43724412200): full migration chain, PostgreSQL integration tests and `go vet ./...` passed. These remain CI/mocked-browser checks; authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — promotion preserves recorded attendance boundaries (2026-11-16)

- Promotion now checks the source enrollment's existing attendance before shortening its end date. A promotion date earlier than any already-recorded attendance is rejected, preventing historical attendance from becoming inconsistent with the enrollment period.
- PostgreSQL integration coverage creates an attendance record, attempts an otherwise date-valid promotion before that attendance, and verifies the source enrollment remains active with no destination enrollment created.
- Backend CI **PASS** on checkpoint `60674df6b419a5504d4d11813f7354ecbe3058ae` — [run 43794753873](https://github.com/TheSumitNain/school-management-system/actions/runs/43794753873): full migration chain, PostgreSQL integration tests and `go vet ./...` passed. Authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — promotion attendance validation and field-level recovery (2026-11-16)

- The backend rejects a promotion date earlier than attendance already recorded for the source enrollment. The check runs while the permanent student row is locked, using the same lock order as attendance marking, so a concurrent attendance write cannot bypass the boundary.
- The API returns a specific `admission_date` field error. The promotion dialog displays it directly beneath the date picker, preserves the rest of the form, clears the error when the date changes, and allows a retry.
- PostgreSQL integration coverage verifies that a rejected promotion leaves the source enrollment active and creates no destination enrollment. Playwright coverage verifies the field-level server error, correction and successful retry.
- Final checkpoint `ec6a9d4a9d4204674feb3c24d03c95862a453e46`: Backend CI **PASS** — [run 43795365553](https://github.com/TheSumitNain/school-management-system/actions/runs/43795365553), full migrations, PostgreSQL integration and `go vet ./...`; Frontend CI **PASS** — [run 43795365580](https://github.com/TheSumitNain/school-management-system/actions/runs/43795365580), production build and all 11 Playwright tests. Authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — guardian email field-level server validation (2026-11-16)

- Guardian email normalization now returns a dedicated `ErrInvalidGuardianEmail` sentinel for malformed, multi-address, display-name, and over-254-character email values. Other guardian validation failures retain their existing generic validation classification.
- The guardian API returns an `email_address` field error. Both create and shared-profile edit forms display that server error beneath the email input, preserve the other entered guardian details, and allow correction/retry.
- Repository integration tests cover malformed email rejection for both create and update. Handler tests assert the field-specific response; Playwright verifies server-side rejection, inline display, correction and successful create.
- Checkpoint `947f6f2c37ab8c5d2edfe3ac50530010cbaf446e`: Backend CI **PASS** — [run 43796041256](https://github.com/TheSumitNain/school-management-system/actions/runs/43796041256), full migrations, PostgreSQL integration and `go vet ./...`; Frontend CI **PASS** — [run 43796041255](https://github.com/TheSumitNain/school-management-system/actions/runs/43796041255), production build and all 11 Playwright tests. Authenticated local API/database acceptance remains outstanding.


## Student Management — guardian email server-error regression verified (2026-11-16)

- Guardian create and shared-profile update now expose malformed email as the specific `email_address` field error; the frontend maps the API field error beneath the email input and preserves other entered values.
- Corrected the Playwright assertion to track the actual shared-profile update request collection. The earlier frontend failure was a test variable-name error; no product-code rollback was needed.
- Verified checkpoint `42e25772be3aa3f9f75690b99fd7459f8203e5a2`: Frontend CI **PASS** — [run 43796919939](https://github.com/TheSumitNain/school-management-system/actions/runs/43796919939), production build and all 11 Playwright tests; Backend CI **PASS** — [run 43796919839](https://github.com/TheSumitNain/school-management-system/actions/runs/43796919839), full migration chain, PostgreSQL integration tests and `go vet ./...`. These are CI checks; authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — guardian email server errors in create and shared edit (2026-11-16)

- Guardian email validation uses a dedicated repository sentinel and the API returns `field_errors.email_address`, separately from other guardian validation failures.
- The create form and shared guardian profile edit form both render server email errors directly beneath the email input. The forms preserve other typed values; failed updates do not mutate the shared guardian profile. Correcting the email clears the field error and permits retry.
- Repository integration tests verify malformed and over-limit email rejection on create and update. Handler tests verify field-level API responses for both POST and PATCH. Playwright verifies the create and shared-edit flows, including no persisted mutation on rejection and successful retry.
- Verified checkpoint `42e25772be3aa3f9f75690b99fd7459f8203e5a2`: Backend CI **PASS** — [run 43796919839](https://github.com/TheSumitNain/school-management-system/actions/runs/43796919839), full migrations, PostgreSQL integration and `go vet ./...`; Frontend CI **PASS** — [run 43796919939](https://github.com/TheSumitNain/school-management-system/actions/runs/43796919939), production build and all 11 Playwright tests. Authenticated local API/database acceptance remains outstanding.


## Student Management — student contact email field-level validation (2026-11-16)

- Student contact email validation now has a dedicated `ErrInvalidStudentContactEmail` classification. The API maps it to `field_errors.email_address`; the Address & Communication form displays the server error directly below Email and preserves the entered values for correction and retry.
- Backend integration and handler tests cover the specific validation contract. The Playwright acceptance journey simulates a server-side email rejection, verifies inline feedback and confirms no contact data was persisted before a corrected retry.
- Verified checkpoint `52d945d9ea067430aea094464e54580d48f0da33`: Backend CI **PASS** — [run 43797770850](https://github.com/TheSumitNain/school-management-system/actions/runs/43797770850), full migrations, PostgreSQL integration tests and `go vet ./...`. Frontend CI **PASS** — [run 43797513589](https://github.com/TheSumitNain/school-management-system/actions/runs/43797513589), production build and all 11 Playwright tests on frontend checkpoint `4634742e7fb3c231a0e6ef3b5a1b0697a8ac3e23`. Authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — basic-details and physical-measurement boundary tests (2026-11-16)

- Extended PostgreSQL integration coverage for Basic Details to verify that a future date of birth is rejected by repository validation before persistence.
- Extended Physical Details integration coverage for invalid weight, missing measurement date and future measurement date, in addition to the existing height/precision and empty-measurement checks.
- Backend CI **PASS** on exact checkpoint `c5f0f4107e653fcd76d7bc5b2afcd2107cbe0f99` — [run 43798404680](https://github.com/TheSumitNain/school-management-system/actions/runs/43798404680): full migration chain, PostgreSQL integration tests and `go vet ./...` passed.
- These are backend CI checks. Authenticated acceptance against the user's local API/database remains outstanding.


## Student Management — school registration number server-side field validation (2026-11-16)

- Duplicate school registration numbers now have a dedicated `ErrStudentRegistrationNumberConflict` classification based on the named PostgreSQL unique index. The API returns a specific `field_errors.school_registration_number` response instead of a generic academic conflict.
- Basic Details maps the API field error directly beneath School registration number using the shared `ApiError` contract. The form preserves other entered values and supports correction/retry.
- PostgreSQL integration coverage verifies duplicate registration rejection through the repository API; handler tests verify the field-error response; Playwright exercises the actual server field-error payload and confirms no duplicate value is persisted.
- Verified checkpoint `63c8d98fd70615a62d139028a5ad0060171496c5`: Backend CI **PASS** — [run 43799163105](https://github.com/TheSumitNain/school-management-system/actions/runs/43799163105), full migrations, PostgreSQL integration tests and `go vet ./...`; Frontend CI **PASS** — [run 43799163073](https://github.com/TheSumitNain/school-management-system/actions/runs/43799163073), production build and all Student Management Playwright tests. Authenticated local API/database acceptance remains outstanding.

## Student Management — date-only timezone acceptance risk (2026-11-16)

- Audit identified a calendar-date consistency risk in Date of Birth and Physical Details measurement dates: the browser date picker uses the user's local calendar date, while backend future-date checks compare UTC calendar dates. In timezones ahead of UTC, a valid local “today” can be interpreted as a future UTC date during the local early-morning window.
- Resolution (2026-11-18): the product selected `Asia/Kolkata` as the system-wide school calendar timezone. The implementation and regression coverage are recorded in the current timezone section at the top of this document.
- Authenticated local API/database and browser acceptance remains outstanding; passing CI does not replace that verification.

## Student Management — admission and enrollment date ordering (2026-11-16)

- Repository validation now rejects an admission date later than the enrollment start date, while allowing the same date or an earlier admission date within the selected academic year.
- Migration V46 adds a NOT VALID database CHECK constraint so new and updated enrollment rows enforce the same rule without blocking deployment on legacy rows that may need separate review.
- Added unit coverage for same-day, earlier-admission, later-admission and out-of-year cases, plus PostgreSQL integration coverage for the named database constraint.
- The repository returns a dedicated admission-date ordering error and the API maps it to `field_errors.admission_date`, so callers receive a precise field-level correction instead of a generic academic validation response.
- Backend CI **PASS** on code commit `1f1147d6e4914d0e8fd02f8eea7a8b7f882c3676` — [run 43804202314](https://github.com/TheSumitNain/school-management-system/actions/runs/43804202314): complete migration chain including V46, all Go tests including PostgreSQL integration, and `go vet ./...` passed. Latest branch HEAD at this documentation update is `3bcd5bbe15873141522cd714d92ce6ee2a6c097e` (docs-only after the tested code commit). Authenticated local API/database/browser acceptance remains outstanding.

## Student Management — enrollment date field-error mapping (2026-11-16)

- The enrollment form exposes one date control (enrolled_at) while the backend's admission/enrollment ordering validation identifies the same field as admission_date. The frontend now maps that API field error to the visible enrollment date control, preserving inline field-level feedback.
- Added Playwright coverage that returns the API error shape (field_errors.admission_date), verifies the message appears at the enrollment date field, and retries successfully without losing the form values.
- Backend CI **PASS** — [run 43807314486](https://github.com/TheSumitNain/school-management-system/actions/runs/43807314486): full migration chain, PostgreSQL integration tests, Go tests and go vet ./...
- Frontend CI **PASS** — [run 43807314407](https://github.com/TheSumitNain/school-management-system/actions/runs/43807314407): production build and Student Management Playwright browser acceptance tests.
- Both workflows tested code commit 8b968c6c95da9d5d9f1e4ccb0b10efa857c5b9ab. Authenticated local API/database/browser acceptance on the developer's local environment remains outstanding.

## Student Management — unchanged medical profile save (2026-11-16)

- Audit found that submitting an unchanged Medical Details form created another version of the current medical profile and copied the same prior values into immutable history, even though no medical data had changed.
- The repository now compares normalized values while holding the student/current-profile locks. An unchanged save returns the existing current profile and history without creating a new history row, incrementing the version, or writing a medical-update audit event. Actual changes retain the existing append-only history behavior.
- Added PostgreSQL integration coverage that saves the exact same medical values after a real update and verifies the version remains 2 with exactly one prior history entry. Existing audit-count assertions also verify the no-op did not create another update event.
- Backend CI **PASS** on test commit `2b1d9935ceb532133d7ff54f51ce6b6ad606a2d6` — [run 43808235755](https://github.com/TheSumitNain/school-management-system/actions/runs/43808235755): full migration chain, Go tests including PostgreSQL integration, and `go vet ./...` passed.
- Authenticated local API/database/browser acceptance remains outstanding; CI used an ephemeral database.

## Student Management — physical measurement retry safety (2026-11-16)

- The physical-details API now treats a repeated save of the latest measurement with the same date and values as an idempotent retry. It returns the existing measurement without appending a duplicate history row or creating another `school.student.physical_details.recorded` audit event. A changed measurement remains a new immutable history entry.
- Added PostgreSQL integration coverage for repeating both the initial measurement and the latest updated measurement; assertions verify that the measurement ID and history length remain unchanged. The existing audit-count assertion confirms only actual measurements are recorded.
- Backend CI **PASS** on code commit `79136223528e4a472ee101c2dac7f87cabfc4fac` — [run 43808861512](https://github.com/TheSumitNain/school-management-system/actions/runs/43808861512): full migration chain, Go tests including PostgreSQL integration, and `go vet ./...` passed.
- Latest branch HEAD before this documentation update: `79136223528e4a472ee101c2dac7f87cabfc4fac`. Authenticated local API/database/browser acceptance remains outstanding.

## Student Management — cross-school directory and family-read isolation (2026-11-16)

- Added PostgreSQL integration assertions that a School Admin cannot list the student directory using another school's scope; the repository must return `ErrAcademicAccessDenied` with no rows and a zero total.
- Added family/guardian isolation assertions for listing a foreign school's student-guardian links and related-student records, plus attempting to list the guardian directory under another school's scope. Foreign student reads must return not-found within the authorized school, while a foreign school scope must be denied.
- Verified checkpoint `98205a858a12bd5eb3b508167bbfa3b2d61bd67f`: Backend CI **PASS** — [run 43810025268](https://github.com/TheSumitNain/school-management-system/actions/runs/43810025268), full migration chain, Go tests including PostgreSQL integration, and `go vet ./...` passed.
- Existing integration tests were also reviewed for cross-school read/write isolation in Basic Details, Contact Details, Address/Locations and Medical/Physical Details; no additional defect was confirmed in this pass.
- Authenticated local API/database/browser acceptance remains outstanding. CI uses an ephemeral database and does not replace local acceptance.

## Student Management — safe archive retry after a lost response (2026-11-16)

- Audit of the archive confirmation flow identified a retry edge case: the database could commit an archive while the HTTP response was lost. Retrying then returned not-found because the student was already archived, leaving the UI unable to confirm the completed operation.
- `ArchiveStudent` is now idempotent. When the authorized school-scoped student is already archived, the repository returns the existing archived profile without changing enrollment history or writing a duplicate archive audit event. First-time archive behavior remains transactional: archive the student, withdraw active enrollment(s), preserve history, and write one audit event.
- PostgreSQL integration coverage archives a student, retries the same operation, verifies the archived profile is returned, checks that exactly one archive audit event exists, and confirms enrollment history remains preserved.
- Playwright now simulates the server committing the archive while the response is lost, then retries and verifies the UI completes successfully and refreshes the archived status.
- Backend CI **PASS** — [run 43810799396](https://github.com/TheSumitNain/school-management-system/actions/runs/43810799396): full migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Frontend CI **PASS** — [run 43810799453](https://github.com/TheSumitNain/school-management-system/actions/runs/43810799453): TypeScript/Vite production build and Student Management Playwright browser acceptance tests.
- Tested code checkpoint: `b6d06783d848078f11db2101dc40b9cecd934e1d`. Authenticated acceptance against the user's local API/database remains outstanding.

## Student Management — enrollment history school isolation (2026-11-16)

- Added PostgreSQL integration coverage with a real foreign-school student and enrollment record. A caller scoped to its own school cannot retrieve that foreign student's enrollment history; the repository returns not-found with no rows. Supplying the other school's scope is rejected with access-denied and no data.
- Backend CI **PASS** — [run 43811188398](https://github.com/TheSumitNain/school-management-system/actions/runs/43811188398): full migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `c8436ea635a4727ea41023908f4778cae0dbb369`. Authenticated local API/database/browser acceptance remains outstanding.

## Student Management — curriculum school isolation and scoped writes (2026-11-16)

- Added explicit `school_id` predicates to selected-subject reads, selection-history revocations, and curriculum-assignment replacement writes. These operations now consistently constrain both the enrollment-assignment identifier and tenant scope.
- Added PostgreSQL integration assertions that cross-school requests cannot read a student's selected curriculum, update subject selections, list class curricula, or save a class curriculum using another school's scope.
- Backend CI **PASS** — [run 43811590860](https://github.com/TheSumitNain/school-management-system/actions/runs/43811590860): full migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `3aa23a8a8a2110094bfc4a6de935c0650057e4b8`. Authenticated local API/database/browser acceptance remains outstanding.

 
## Student Management — basic-details save retry safety (2026-11-17)

- Updating a student's Basic Details with the exact same normalized name now returns the existing school-scoped active profile without changing `updated_at` or inserting another `school.student.updated` audit event. Real changes continue to update the record and create one audit event.
- Added PostgreSQL integration coverage that performs a real name update, retries the identical payload, and verifies the student ID/name remain stable, the timestamp does not advance, and the audit count remains exactly one.
- Backend CI **PASS** — [run 44007420006](https://github.com/TheSumitNain/school-management-system/actions/runs/44007420006): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `522f2c6a378a1ff1f052fbd8436200e98aadd2e4`. Authenticated acceptance against the user's local API/database/browser remains outstanding.

 
## Student Management — guardian relationship save retry safety (2026-11-17)

- Re-saving an unchanged active student–guardian relationship now returns the existing school-scoped link without advancing its timestamp or inserting a duplicate relationship-update audit event. When a relationship genuinely changes, primary-contact serialization and the existing audit behavior are preserved.
- PostgreSQL integration coverage repeats an identical relationship update and verifies the link, timestamp and audit count remain unchanged.
- Backend CI **PASS** — [run 44007696968](https://github.com/TheSumitNain/school-management-system/actions/runs/44007696968): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `709dd194e16d7c5e2215bdcca6c2b1473a2318f6`. Authenticated local API/database/browser acceptance remains outstanding.

 
## Student Management — basic profile-details retry safety (2026-11-17)

- An unchanged Basic Details save now skips the upsert update, timestamp mutation and duplicate sensitive-profile audit event. Actual field changes continue to update the profile and record one audit event.
- PostgreSQL integration coverage retries the exact normalized payload and checks the original profile timestamp and single audit event are preserved.
- Backend CI **PASS** — [run 44008388337](https://github.com/TheSumitNain/school-management-system/actions/runs/44008388337): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `b7ccc9106f4e8183faf5c65985429dfc7eac5607`. Local authenticated API/database/browser acceptance remains outstanding.

 
## Student Management — guardian-link creation retry safety (2026-11-17)

- Repeating the exact same active student–guardian link request now returns the existing link instead of surfacing a duplicate-pair conflict. It does not clear/rewrite the primary contact or emit another creation audit event. A conflicting payload for an already-active pair remains a conflict.
- PostgreSQL integration coverage retries the exact same link and verifies stable identity, timestamp and single creation audit event.
- Backend CI **PASS** — [run 44008653061](https://github.com/TheSumitNain/school-management-system/actions/runs/44008653061): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `844c53d13650f30fb9abab22dad4ff2523246f67`. Local authenticated API/database/browser acceptance remains outstanding.

 
## Student Management — date-of-birth field validation (2026-11-17)

- Invalid or out-of-range dates of birth now return a dedicated API field error, and the Basic Details form displays that error directly beneath the date picker with invalid-state accessibility attributes. Other generic academic validation errors remain separate.
- Added repository and handler regression coverage plus a Playwright case for the server-returned date-of-birth error.
- Backend CI **PASS** — [run 44009150470](https://github.com/TheSumitNain/school-management-system/actions/runs/44009150470): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Frontend CI **PASS** — [run 44009150439](https://github.com/TheSumitNain/school-management-system/actions/runs/44009150439): TypeScript/Vite production build and Student Management Playwright browser acceptance.
- Tested code checkpoint: `16b2b3fd6daf057a38d524a954f4dae5098d7050`. Local authenticated API/database/browser acceptance remains outstanding.

 
## Student Management — physical measurement date field validation (2026-11-17)

- Invalid or missing physical measurement dates now return a dedicated API field error. The Physical Details form displays the message directly beneath the measurement date picker and marks the control invalid.
- Added repository and handler regression coverage and a Playwright case for a server-returned measurement-date validation error.
- Backend CI **PASS** — [run 44009887924](https://github.com/TheSumitNain/school-management-system/actions/runs/44009887924): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Frontend CI **PASS** — [run 44009644248](https://github.com/TheSumitNain/school-management-system/actions/runs/44009644248): TypeScript/Vite production build and Student Management Playwright browser acceptance.
- Tested code checkpoint: `7774dcd77fa1917923d3198f9b5b0b17e6e613b9`. Local authenticated API/database/browser acceptance remains outstanding.

## Student Management — date-only API calendar rendering (2026-11-17)

- PostgreSQL DATE fields represented by Go time.Time can reach the frontend as midnight timestamps. Formatting those values as instants can shift the displayed calendar day in browsers west of UTC.
- Student academic history, the enrollment directory, and physical-measurement history now explicitly format date-only fields as calendar dates. Actual timestamps such as record creation and medical-history change times continue to use timezone-aware instant formatting.
- Playwright now runs the Student Management suite in America/Los_Angeles and uses midnight-UTC timestamp fixtures matching the API shape. Coverage verifies the enrollment directory, student academic history, and historical physical-measurement date.
- Frontend CI **PASS** — [run 44016765201](https://github.com/TheSumitNain/school-management-system/actions/runs/44016765201): TypeScript/Vite production build and Student Management Playwright browser acceptance.
- Backend CI **PASS** — [run 44016765225](https://github.com/TheSumitNain/school-management-system/actions/runs/44016765225): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- The strict enrollment-date parser now has handler regression coverage for valid dates, leap days, whitespace, impossible dates, and rejection of timestamps/locale-formatted values.
- Archived student profile browser coverage now verifies the historical enrollment `ended_at` calendar date in the America/Los_Angeles timezone, alongside the existing enrollment and physical-history date checks.
- Frontend CI **PASS** — [run 44018618470](https://github.com/TheSumitNain/school-management-system/actions/runs/44018618470): TypeScript/Vite production build and Student Management Playwright browser acceptance.
- Backend CI **PASS** — [run 44018618455](https://github.com/TheSumitNain/school-management-system/actions/runs/44018618455): full migrations, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: 90f314c6dce921be40c555103dc362c3a3416c83. Authenticated acceptance against the user's local API/database/browser remains outstanding. The separate school-timezone policy was subsequently resolved as `Asia/Kolkata` and implemented; this change addresses display only.

## Student Management — Basic Details API/UI contract audit (2026-11-17)

- Reviewed the Basic Details repository, handler and edit form together. The backend normalizes and enforces the documented Unicode-character limits (place of birth 120, nationality 80, religion 80, school registration number 50), restricts gender/social-category values, validates date of birth, and enforces school-scoped registration-number uniqueness.
- The form validates text lengths before submission and maps the server's date-of-birth and duplicate-registration-number field errors directly beneath their respective controls. Failed profile reads disable the profile save flow rather than submitting against an unknown baseline; successful unchanged saves remain idempotent and do not create duplicate audit events.
- Sensitive reads and writes retain the existing permission checks and privacy-safe audit metadata. No confirmed regression was found in this pass; no speculative code change was made.
- Follow-up hardening: Basic Details now maps every supported server-returned field error from the API error payload to its matching field control, rather than maintaining a two-field special case. The current backend emits date-of-birth and school-registration-number errors; both remain covered by the existing Playwright inline-error/recovery assertions. This keeps the form aligned if another supported field-level validation is added later.
- Frontend CI **PASS** — [run 44027107826](https://github.com/TheSumitNain/school-management-system/actions/runs/44027107826): TypeScript/Vite production build and Student Management Playwright browser acceptance.
- Backend CI **PASS** — [run 44027107816](https://github.com/TheSumitNain/school-management-system/actions/runs/44027107816): complete migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Tested code checkpoint: `cb72cd96d5dc477225d1ad59c0fb80336eb50428`. These are CI/mocked-browser results; authenticated acceptance against the user's local API/database/browser remains outstanding.

- This is a code-level contract review, not authenticated local API/database/browser acceptance. The school-wide `Asia/Kolkata` policy now governs date-of-birth future-date validation.

## Student Management — medical and physical read-retry coverage (2026-11-17)

- Extended the Student Management Playwright profile-state scenario to simulate temporary server failures when loading restricted Medical Details and Physical Details.
- The test holds each failure through the query client's configured automatic retry, verifies the visible error/retry state, then explicitly retries and confirms the form becomes available. This checks the user-facing recovery action rather than allowing an automatic retry to mask it.
- Code checkpoint: `c4f7ff5523709be0c68ed44039d3b05156ca6ef9`. Frontend CI passed (TypeScript/Vite production build and Student Management Playwright browser acceptance): https://github.com/TheSumitNain/school-management-system/actions/runs/44051159786. Backend CI passed (full PostgreSQL migration chain, Go tests including PostgreSQL integration, and `go vet ./...`): https://github.com/TheSumitNain/school-management-system/actions/runs/44051159743.
- Authenticated acceptance against the user's local API/database/browser remains outstanding.

## Student Management — promotion form save-state locking (2026-11-17)

- Audit found the destination class/section selector remained interactive while a promotion request was pending, even though the other promotion fields and modal close action were locked. This could let the visible destination differ from the submitted destination while the request was in flight.
- Updated the selector to disable during the mutation. Extended the promotion Playwright scenario to hold a successful promotion response and assert that the academic-year selector, section selector, admission/roll fields, and promotion date control all remain disabled until the request completes.
- Code checkpoint: `b1ea5d9e3de2551650e42b52bbb84e19e6f922ef`. Frontend CI **PASS** — [run 44052042190](https://github.com/TheSumitNain/school-management-system/actions/runs/44052042190): TypeScript/Vite production build and Student Management Playwright browser acceptance. Backend CI **PASS** — [run 44052042241](https://github.com/TheSumitNain/school-management-system/actions/runs/44052042241): complete PostgreSQL migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- Authenticated acceptance against the user's local API/database/browser remains outstanding.

## Student Management — directory pagination and filter reset coverage (2026-11-17)

- Added focused Playwright coverage for the student directory's 20-row pagination, search/status/academic-year filter resets, and recovery when archiving the only student on the final page reduces the total. The directory now clamps an out-of-range offset to the last valid page after refreshed results arrive, avoiding an invalid range such as `41–40 of 40`. Added PostgreSQL integration assertions that filtered totals remain stable beyond the last result and invalid status, limit, and offset values are rejected.
- Code checkpoint: `a5a1d0ab002660042b6fe81839fcaafd087d8aac`. Frontend CI **PASS** — [run 44055551562](https://github.com/TheSumitNain/school-management-system/actions/runs/44055551562): TypeScript/Vite production build and all Student Management Playwright browser acceptance tests. Backend CI **PASS** — [run 44055551417](https://github.com/TheSumitNain/school-management-system/actions/runs/44055551417): complete PostgreSQL migration chain, Go tests including PostgreSQL integration, and `go vet ./...`.
- This is mocked-browser regression coverage. Authenticated acceptance against the user's local API/database/browser remains outstanding.

## Student Management — literal directory search matching (2026-11-17)

- Hardened student directory search so user-entered `%`, `_` and backslash characters are treated as literal characters instead of `ILIKE` wildcard syntax. This prevents a typed search from unexpectedly broadening to unrelated student/admission records. The same escaped search pattern is used for both the filtered total and the returned page.
- Added PostgreSQL integration coverage confirming that literal percent, underscore and backslash characters do not act as wildcards, and that 100 literal wildcard characters remain within the documented input limit.
- Code checkpoint: `57d97ea2390ee02964594706bd57c13ed7d1c07e`. The search-length limit is checked against the original user input before wildcard escaping; integration coverage includes 100 literal wildcard characters at the limit. Backend CI **PASS** — [run 44057479897](https://github.com/TheSumitNain/school-management-system/actions/runs/44057479897): complete PostgreSQL migration chain, Go tests including the expanded literal percent/underscore/backslash integration matrix, and `go vet ./...`.

## Student Management — enrollment directory scalability review (2026-11-17)

- Source review found that the School Admin enrollment tab currently loads every visible enrollment in one response when no academic-year filter is selected. The query is tenant- and teacher-scope constrained, but it has no page-size/offset bound; historical enrollment growth can therefore increase response size and browser rendering work over time.
- This is a scalability follow-up, not a demonstrated correctness or authorization defect. Student profile enrollment history remains a separate per-student view and should retain its complete history semantics.
- Follow-up implementation should add bounded pagination and a total count to the enrollment directory endpoint/UI, preserve the existing year filter and teacher roster privacy projection, and leave the per-student academic history endpoint unchanged. No behavior was changed in this review.

## Student Management — bounded enrollment directory pagination (2026-11-17)

- Added a separate paginated enrollment-directory repository/service path with a default page size of 20, a maximum page size of 100, total-count metadata, stable newest-enrollment-first ordering, and offset support. The existing complete per-student enrollment-history method and endpoint are unchanged.
- Reused the existing school-admin and teacher class-incharge/subject-assignment visibility predicates and the existing minimal roster JSON projection. Directory reads have a distinct audit action and record page size, offset, returned count, total count, and academic-year filter state.
- Added PostgreSQL integration assertions for teacher visibility, count stability on an out-of-range page, and invalid page bounds. Frontend query keys include year, page size and offset; the directory year filter has independent state from admission/enrollment forms and changing it resets to the first page.
- Backend CI **PASS** — full PostgreSQL migration chain, Go tests including the teacher-scope/pagination integration assertions, and `go vet ./...` ([run 44066066228](https://github.com/TheSumitNain/school-management-system/actions/runs/44066066228)). Frontend CI **PASS** — TypeScript/Vite production build and Student Management Playwright browser acceptance, including enrollment-directory page navigation and academic-year filter reset ([run 44066066361](https://github.com/TheSumitNain/school-management-system/actions/runs/44066066361)). Local authenticated API/database/browser acceptance remains outstanding.

## Student Management — enrollment directory ordering indexes (2026-11-17)

- Added two reversible PostgreSQL indexes for the paginated enrollment directory's stable `enrolled_at DESC, id DESC` ordering: one scoped by school for the all-years view, and one scoped by school plus academic year for the filtered view. This avoids relying on the section-oriented enrollment index or sorting the entire school history for these access patterns.
- Added a PostgreSQL integration assertion that both indexes are present after the full migration chain. No API response, authorization predicate, or student-history behavior changed.
- Backend CI **PASS** — full migration chain (including V48), PostgreSQL integration tests including the new index-presence assertion, and `go vet ./...` ([run 44066803600](https://github.com/TheSumitNain/school-management-system/actions/runs/44066803600)).
## Future Student Profile print/export — planned, not implemented

- Add a **Print Complete Profile** action from the existing Student Profile; do not create a duplicate student data-entry page.
- The printable report should assemble the student's already-recorded information into clear sections: identity/basic details, current and historical academic enrollment, parents/guardians and sibling relationships, contact/address, and medical/physical information only when the viewer is explicitly authorized.
- Include school identity and student identifiers where available, clear missing/not-recorded states, readable page breaks, print preview, page numbering, and browser Print / Save as PDF.
- Allow a complete report or selected sections only if the permission model authorizes every selected section. Never let print/export bypass backend tenant, assignment, role, field-level or sensitive-data checks.
- School Admin access remains school-scoped. Class In-charge and other teachers must receive only records and fields permitted for their exact class/section/subject assignments; medical, guardian-contact and other sensitive fields remain separately permission-controlled.
- This is a future enhancement only. Do not build a new route, report endpoint, print UI or schema/migration until Student Management is accepted and the print work is explicitly started. Parent OTP remains deferred.

## School Admin role and staff permission controls — current capability and gaps

- The School Admin workspace includes a Roles & Permissions page for creating school-specific custom roles and managing permission grants that the Platform Admin has actively delegated to that school. Built-in system roles, including Teacher, are protected and cannot be edited directly from this page.
- Staff Management includes role assignment and individual permission override workflows. ALLOW/DENY overrides require the relevant permission and an active exact delegation; DENY precedence and backend authorization remain authoritative.
- Teacher and Class In-charge visibility is not determined by role labels alone. Effective access is the combination of live permissions, school ownership, active class/section/subject assignments, and field-level restrictions. A class-in-charge assignment does not grant access outside its assigned class/section.
- The current UI is not yet a complete, easy-to-understand access matrix for comparing Teacher vs Class In-charge or configuring every module's view/create/edit/delete rights in one place. Treat a consolidated permission matrix and role preview as a future usability improvement, not an already-delivered feature.
- Do not grant broad student-profile, guardian-contact, medical, document or other sensitive permissions merely to make a roster or class-in-charge workflow function. Verify each module's backend enforcement and positive/negative role tests before claiming a role can access or edit it.


## Student Management — curriculum edit history simplification (2026-11-20)

- Curriculum remains one editable record per academic year/class/section; no V2/V3 versions or curriculum edit history are created.
- Confirmed subject removal now permanently deletes the current curriculum subject and all linked student optional-choice rows in the same transaction. Saving a student's changed optional choices also replaces prior choice rows instead of retaining revoked history. Removing an optional group converts its subjects to standalone optional choices, preserves current student choices and deletes the empty group row. The existing impact preview and exact affected-student confirmation remain mandatory; stale counts abort the save.
- Student enrollment curriculum responses and the frontend API contract no longer include selection history; the Student Management panel no longer renders previous-choice history. Saving choices replaces current rows, while the standard security audit event records only curriculum scope and removed-subject count (not removed subject IDs or student choice details).
- Added reversible migration V55 to allow only this confirmed current-state deletion while retaining selection insert validation and one-way assignment-replacement revocation protections.
- Backend integration assertions and frontend fixtures were updated. Backend CI **PASS** — [run 44870000355](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000355); Frontend CI **PASS** — [run 44870000404](https://github.com/TheSumitNain/school-management-system/actions/runs/44870000404). V54/V55 have not been applied to the user's local database; verify a restorable backup before applying them for local manual acceptance.


## Curriculum history policy correction (2026-11-21)

- The earlier V55 current-state-only policy is superseded. V56 restores append-only student subject-selection history and the V54 soft-deactivation behavior for removed curriculum subjects.
- Student choice edits now set `revoked_at` on prior active selection rows and insert the replacement current choices. Confirmed curriculum removals revoke affected active choices and set `class_curriculum_subjects.is_active=FALSE`; neither historical row is deleted.
- The School Admin confirmation continues to show the exact affected-student list and count before any write. The backend recomputes the exact student set under transaction locks, rejects stale confirmations, and writes the audit event in the same transaction.
- The new PostgreSQL integration assertions verify selection history remains after replacement and removed curriculum subject rows remain stored but inactive. V56 was subsequently applied successfully to both local databases on 2026-11-21. Both `school_management` and `school_management_test` were verified at `56|false`. Before migration, fresh V55 backups were created and independently restored into temporary databases; both restores reported `55|false`. The original databases and backup files were left intact.

## Student Management — address location validation accessibility (2026-11-21)

- The address form now associates an incomplete State → District → Sub-District / Tehsil → Village / City path error with the first missing location selector, marks that selector invalid, and renders the error directly below it. Choosing the missing level clears the error and invalid state.
- Added browser acceptance assertions for `aria-invalid`, `aria-describedby`, the inline alert, and clearing the error after the missing district is selected.
- Frontend CI **PASS** — [run 44865807238](https://github.com/TheSumitNain/school-management-system/actions/runs/44865807238): TypeScript/Vite production build and all **31 Student Management browser acceptance tests** passed. No backend or database changes were made.

## Staff Leave & Holiday Management — source audit and backend contract (2026-11-22)

- Audited the latest `work` source checkpoint. No leave/holiday schema, API, permissions, frontend workspace or reusable school holiday calendar exists yet.
- Confirmed current staff attendance stores only present/absent/late/half-day. Approved leave and holidays must remain separate calendar context; they must not silently create, overwrite or delete attendance rows.
- Added [Staff Leave & Holiday Backend Contract](../backend/STAFF-LEAVE-HOLIDAYS.md) covering school-scoped data model, leave request lifecycle, overlap/concurrency safeguards, holiday calendar, dedicated non-delegable permissions, API routes, audit requirements and verification gates.
- No leave quotas, accrual, carry-forward, encashment, payroll deductions or statutory entitlements are assumed. Balances remain unavailable until a school policy is explicitly configured and separately implemented.
- This is a design/audit checkpoint only. No migration or application code has been added for leave/holidays, and no persistent database was changed. Next implementation work is the additive backend migration, permission provisioning, repository/service/handler/routes and PostgreSQL integration tests, followed by the frontend workspace.
- Attendance Phase 3 remains open for downloadable/report views and leave/calendar integration. Academic, Student Management and Settings pages remain untouched.

## Staff Leave & Holiday — initial backend/frontend workflow implemented (2026-11-22)

- Added V59 school-scoped leave type, leave request and school holiday tables. Six dedicated non-delegable permissions are granted to existing and future School Admin system roles and included in the live permission allowlist.
- Backend endpoints support leave type create/list/update; leave request create/list/approve/reject/cancel; and holiday create/list/update. Request creation validates active same-school staff and leave type, dates, duration and reason; pending/approved overlaps conflict under a per-school/staff transaction advisory lock. Decisions are pending-only, use authenticated actor/time, require a rejection comment and deny applicant self-approval. Mutations write transactional audit events.
- Staff Management now exposes a permission-gated Leave & Holidays workspace with leave request list/filters, create request form, leave type create/list, holiday calendar add/edit/deactivate and approve/reject/cancel flows. The request UI always starts at Pending; a leave type with approval flag off does not silently bypass review.
- Backend CI [44982323577](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323577) **PASS** — complete migration chain on disposable PostgreSQL, full Go/PostgreSQL integration suite and `go vet`.
- Frontend CI [44982323582](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323582) **PASS** — TypeScript/Vite production build and all 50 Playwright browser acceptance tests, including the new Leave & Holidays workspace approval/calendar flow.
- CI-tested code checkpoint: `816ee99511d4a7ed0e393dda9a04dc758f954202`. V59 was applied only to disposable CI PostgreSQL. No persistent/local database migration was run.
- Phase 4 remains **partially implemented**, not complete. Remaining: dedicated PostgreSQL behavior tests for overlap/concurrent submissions, authorization, transaction rollback and audit; leave-type edit/deactivation UI and approved school policy; approved leave/holiday contextual integration in staff attendance. No leave balances, accrual, carry-forward, statutory entitlement, payroll effects, document upload or automatic attendance mutation.
- Attendance records remain separate and are not silently created/overwritten/deleted by leave or holiday operations. Academic, Student Management and Settings pages remain untouched.

## Staff Leave & Holiday — backend validation hardening verified (2026-11-22)

- Backend code checkpoint: `8c163b5b35a644cf08a13e280c0c44f238a80037`.
- Backend CI [44984389499](https://github.com/TheSumitNain/school-management-system/actions/runs/44984389499) **PASS** at this exact SHA: full migration chain on disposable PostgreSQL, Go tests (including new service validation tests) and `go vet`.
- Added service validation for ordered leave dates, full/half-day shape, required reason, and holiday date/name/description. Added repository-level input guards for cancellation and holiday date-range reads; strengthened the per-school/staff overlap advisory lock.
- Frontend UI checkpoint `36ab57484de809c30849c030187f0190dab43a60`; Frontend CI [44982936799](https://github.com/TheSumitNain/school-management-system/actions/runs/44982936799) **PASS**. Staff Attendance now shows approved leave and active holiday as informational context only; attendance records are never silently changed.
- Phase 4 remains partially implemented. Dedicated PostgreSQL behavior tests for concurrency/overlap, decision authorization, rollback and audit are still required; leave-type edit/deactivation UI and approved school policy are pending.
- V59 has only run in disposable CI PostgreSQL. No persistent/local database migration was applied. Academic, Student Management and Settings remain untouched.

## Staff Leave & Holiday — leave type controls and browser acceptance (2026-11-22)

- Staff Management leave type configuration now supports editing, reactivation and confirmed deactivation. Existing requests/history remain intact; inactive leave types cannot be selected for new requests.
- Added Playwright coverage for leave type edit/deactivation.
- Frontend CI [44984794710](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794710) **PASS** at checkpoint `f1b71d4e2999f868f62910ee878f78eee1460a8b`: production build and all 51 browser tests.
- Backend CI [44984794698](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794698) **PASS** at the same checkpoint: full migration chain on disposable PostgreSQL, Go tests and `go vet`.
- Phase 4 is still partially implemented pending dedicated PostgreSQL behavior tests for concurrency/overlap, authorization, rollback and audit, plus school policy decisions for balances. V59 has only been applied in disposable CI PostgreSQL. Persistent/local DBs remain untouched.

## Staff Leave & Holiday — PostgreSQL workflow verification (2026-11-22)

- Added real PostgreSQL tests for concurrent overlap serialization, unauthorized decision denial, self-approval prevention, approval/rejection audit persistence, audit-failure transaction rollback, rejection reason validation and cross-school cancellation denial.
- Fixed advisory lock parameter typing after the new concurrency test caught an integer-to-text pgx encoding issue.
- Backend CI [44985769667](https://github.com/TheSumitNain/school-management-system/actions/runs/44985769667) **PASS** at code checkpoint `e028baffd2485afeee792f98b068f89e2ce1ae07`: full migration chain on disposable PostgreSQL, Go test suite and `go vet ./...`.
- No persistent/local database migration applied. Phase 4 still needs further frontend workflow acceptance, attendance reporting integration and school-approved leave policy/balance decisions.

## Staff Attendance / Leave reporting follow-up (2026-11-22)

- Daily attendance status filter now applies even without a search term.
- Monthly attendance report shows approved leave and active holidays in a distinct, permission-gated calendar context; neither is counted as attendance.
- Leave list API supports validated inclusive month-overlap date ranges. Added PostgreSQL integration coverage for in-range vs out-of-range requests.
- Backend CI [44986574096](https://github.com/TheSumitNain/school-management-system/actions/runs/44986574096) **PASS** at `a01514119b0d395b106accd990f59b2f7a5e992f` including migrations, Go tests and vet. Frontend build passes; Playwright run is pending after an E2E mock routing correction.
- No persistent/local DB migrations. No changes to Academic, Student Management or Settings pages.

## Staff Leave & Holiday — holiday reactivation and local acceptance (2026-11-22)

- School holiday listing defaults to active entries. Users with `staff.holidays.manage` can request inactive entries; that expanded listing requires the manage permission, including for manager-only accounts without a separate holiday-read grant.
- Inactive holidays have an explicit Reactivate action and confirmation. Editing an inactive holiday preserves its inactive state; only the separate reactivation action activates it. Deactivation also remains explicitly confirmed.
- Added PostgreSQL integration coverage for active-only default listing, manager-only inactive listing and denial when a non-manager requests inactive entries.
- Local verification at `48c4aef32ff2e92d2cd4dde8b8a0e7bf5645ee63`: `npm run typecheck` **PASS**; `npm run test:e2e -- staff-management.spec.ts` **PASS (14/14)**; `npm run build` **PASS**.
- Backend CI [44990112117](https://github.com/TheSumitNain/school-management-system/actions/runs/44990112117) **PASS** at the same commit: full migration chain on disposable PostgreSQL, backend tests including PostgreSQL integration, and `go vet`. Frontend CI [44990112138](https://github.com/TheSumitNain/school-management-system/actions/runs/44990112138) **PASS**: TypeScript/Vite production build and the full Playwright browser acceptance suite (the workflow runs all specs in `frontend/e2e`, including Staff and Student Management). Staff-specific 14-test browser acceptance was separately verified locally.
- No persistent/local database migration was applied. Phase 4 remains partially implemented; approved school leave policy/balance rules and remaining end-to-end workflow review are still outstanding. Academic, Student Management and Settings implementation was not changed by this Staff follow-up.

## Staff Leave & Holiday — holiday write-path integrity tests (2026-11-22)

- Added PostgreSQL integration tests for normalized duplicate holiday creation, conflicting holiday updates, unchanged state/audit on conflict, cross-school holiday write denial, and transactional rollback when the holiday audit insert fails.
- Backend CI [44990622671](https://github.com/TheSumitNain/school-management-system/actions/runs/44990622671) **PASS** at exact checkpoint 92a307cb62abbab902d8f5441deb55219e4ea374: complete migration chain on disposable PostgreSQL, full Go/PostgreSQL integration suite and go vet.
- No persistent/local DB migration was applied. Staff Leave & Holiday Phase 4 remains partially implemented pending remaining browser workflow review and explicit school leave-policy decisions. Academic, Student Management and Settings remain out of scope for this work.

## Staff Leave & Holiday — form error isolation (2026-11-22)

- Split request, leave-type and holiday server-error state so failures remain attached to their own form across pane navigation. Added an E2E regression test for leave-type conflict errors persisting only in the leave-type pane.
- Code checkpoint `2e6c73678798189a689d8b3223cb797872e6de22`: Frontend CI [44991137384](https://github.com/TheSumitNain/school-management-system/actions/runs/44991137384) **PASS** (TypeScript/design-token checks, Vite build, full Playwright suite: **57 passed**); Backend CI [44991137364](https://github.com/TheSumitNain/school-management-system/actions/runs/44991137364) **PASS** (complete disposable-PostgreSQL migration chain, Go/PostgreSQL integration suite and `go vet`).
- No persistent/local DB migration. Staff Leave & Holiday Phase 4 remains partially implemented pending remaining workflow acceptance and approved school leave-policy decisions. Academic, Student Management and Settings were not changed by this follow-up.


## Staff Leave & Holiday — read-permission navigation acceptance (2026-11-22)

- Added a Playwright regression test for a School Admin with leave write/decision permissions but without `staff.leave.read`. The Leave & Holidays workspace remains unavailable and the restricted navigation item explains that read permission is required; write permissions alone do not expose list views.
- Latest verified code checkpoint: `5b1162b4a53207f73f5f422a12f5cb84ddc244b6`.
- Backend CI [44991408484](https://github.com/TheSumitNain/school-management-system/actions/runs/44991408484) **PASS**: complete migration chain on ephemeral PostgreSQL, Go tests including PostgreSQL integration, and `go vet`.
- Frontend CI [44991408480](https://github.com/TheSumitNain/school-management-system/actions/runs/44991408480) **PASS**: TypeScript/Vite production build and the complete Playwright browser acceptance suite.
- No persistent/local database migration was applied. Leave balances/accrual remain undefined pending explicit school policy approval; Staff Leave & Holiday Phase 4 remains open for remaining workflow review.

 
## Staff Leave & Holiday — accessible field validation acceptance (2026-11-22)

- Fixed the shared date picker label-activation regression: selecting a calendar day no longer bubbles into the surrounding form label and reopens the calendar. The browser test explicitly verifies the dialog closes after selection.
- Leave request fields (staff member, leave type, start/end dates, duration and reason), leave-type name, and holiday name/date now expose field-level validation through aria-invalid and associated aria-describedby messages; errors are announced with alert semantics where applicable.
- Added browser acceptance checks for accessible leave request validation and holiday date/name validation, including immediate error clearing after valid corrections.
- Exact code checkpoint: f631fd21e5f0b6bb11c5ded315f47d58108d068c.
- Frontend CI [44993812197](https://github.com/TheSumitNain/school-management-system/actions/runs/44993812197) **PASS**: TypeScript/Vite production build and complete Playwright browser acceptance suite.
- Backend CI [44993812207](https://github.com/TheSumitNain/school-management-system/actions/runs/44993812207) **PASS**: full migration chain on disposable PostgreSQL, Go tests including PostgreSQL integration, and go vet.
- No persistent/local database migration was applied. Leave & Holidays Phase 4 remains open; balances/accrual/carry-forward remain undefined pending explicit school policy approval. Academic, Student Management and Settings remain out of scope.
