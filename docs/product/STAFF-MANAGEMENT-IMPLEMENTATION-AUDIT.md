# Staff Management — Existing Implementation Audit

**Audit date:** 2026-11-21  
**Branch reviewed:** `work`  
**Reviewed commit:** `1ff17d6964f8300cb01fab800e4f4b2c2774ab05`  
**Purpose:** Phase 0 baseline before rebuilding the School Admin Staff Management experience.

This audit records what exists in the repository today. It is not an implementation-completion claim. The approved [Staff Management Roadmap](./STAFF-MANAGEMENT-ROADMAP.md) remains the product source of truth.

## 1. Current frontend baseline

The current `StaffManagementPage.tsx` is a single directory/table plus modal-driven account/profile actions. It is not the planned Staff Management workspace and will be replaced with a new page structure. The existing API client, query hooks, role/override components and shared UI primitives are reusable integration assets, not a layout constraint.

| Capability | Current state | Decision |
|---|---|---|
| Staff directory/list | Exists; search is client-side over loaded rows | Keep API contract; rebuild directory UI and add filters/pagination only with supported backend contracts |
| Create employee | Exists as one modal; creates employee profile and login together | Extend to approved 4-step wizard; backend must be extended before profile-only/no-login creation is offered |
| Employee profile fields | Full name, employee code, designation, mobile, optional email/joining date, employment status | Keep supported fields; remaining personal/employment fields are missing |
| Edit employee profile | Exists in same modal | Rebuild as focused profile-tab forms; preserve server validation mapping |
| Login username | Set at creation; shown read-only during profile edit | Keep separate from employee profile editing |
| Login enable/disable | Exists; enable requires temporary password; disable revokes sessions | Keep behavior and consequence confirmations |
| Password reset | Exists; requires temporary password and forces change at next sign-in | Keep; never expose or persist plaintext credentials after submission |
| Role assignment | Existing `StaffRoleAssignments` component and role APIs | Keep security behavior; relocate into Account & Access profile tab |
| Permission overrides | Existing `StaffPermissionOverrides` component with delegation lookup and mandatory reason | Keep security behavior; relocate into Account & Access profile tab |
| Effective-permission preview | Not present in current Staff page | Missing; implement only against a server-authoritative effective-permission API |
| Credential sharing | Not present | Missing; secure one-time handoff must be designed before adding a Share action |
| Overview dashboard | Not present | Missing |
| Staff profile with 8 tabs | Not present | Missing |
| Qualifications / certifications | No staff-specific frontend or API contract found in this audit | Missing; define schema, upload and verification contracts |
| Staff documents | No staff-specific frontend or API contract found in this audit | Missing; define secure file storage/access contract |
| Academic assignments inside Staff Management | Not present in Staff page | Existing academic assignment domain is in Academic APIs; audit exact read/write contracts and reuse it rather than duplicating models |
| Staff attendance / leave / workload | Not present in Staff page | P1; audit existing school attendance/timetable capabilities before extending |
| Reports / import / ID cards / payroll | Not present | Later phases; payroll remains gated on explicit policy approval |
| Employment history / lifecycle workflow | Not present in Staff page | Missing; design append-only history and transactional audit behavior |

## 2. Existing backend and security contracts to preserve

Repository inspection confirms these current staff operations and constraints:

- `GET /school/staff` lists same-school staff and includes employee profile fields plus `profile_complete`.
- `POST /school/staff` provisions a staff login and employee profile in one backend transaction. Current required profile fields are full name, employee code, designation and one mobile number; email and joining date are optional. Username and initial password are required.
- `PATCH /school/staff/:principalID` updates employee profile fields and employment status. Login identity is not changed through this profile update.
- `DELETE /school/staff/:principalID` disables the account; active sessions are revoked.
- `POST /school/staff/:principalID/enable` re-enables a disabled login with a new temporary password, subject to a complete Active employee profile.
- `POST /school/staff/:principalID/password/reset` resets the password and requires a change at next sign-in.
- Role assignment and individual permission overrides are separate same-school operations protected by live permission/delegation checks.
- Staff usernames are globally unique and case-insensitive; employee codes are unique within a school.
- Employment status and login status are distinct. Making an employee Active does not silently re-enable a disabled login. Inactive employees cannot sign in.
- Backend authorization remains authoritative; frontend permission checks are navigation/action hints only.

The current creation API does **not** support a profile without a login account. Do not show a “Create login: No” path until a reviewed backend/API change supports it transactionally.

## 3. Roadmap mapping: Keep / Extend / Missing / Needs verification

| Roadmap area | Audit result | Next action |
|---|---|---|
| Overview tab | Missing | Build after staff summary data contract is agreed; do not invent dashboard counts |
| Staff Directory tab | Extend | Rebuild page; preserve list API, loading/error/empty handling and permission checks |
| Add Staff four-step flow | Extend | Build wizard around supported fields; extend backend before exposing unsupported fields or no-login mode |
| Profile Overview / Personal / Employment | Extend | Build profile route and section forms; mark unsupported fields as unavailable until API/storage exists |
| Qualifications & Experience | Missing | Phase 1 schema/API/security design |
| Documents | Missing | Phase 1 secure upload/download contract and authorization |
| Academic Assignments | Existing domain; Staff UI missing | Inspect current academic assignment APIs, filters, history and constraints; reuse existing Academic Structure |
| Account & Access | Partially implemented | Preserve roles, overrides, reset, enable/disable; add safe effective-access preview if supported |
| History & Activity | Missing in staff UI | Define event source and read permissions; never fabricate activity |
| Attendance | Staff workflow missing | P1 contract audit against existing attendance foundations |
| Leave | Missing | P1 schema, policy and approval contract |
| Workload & Timetable | Staff workflow missing | P1 reuse timetable and assignment data |
| Reports / Import / ID cards | Missing | Later phase |
| Payroll | Missing | P2 only after policy sign-off |

## 4. Rebuild sequence

1. **Phase 0 — Audit and contract freeze:** this baseline; next inspect exact Academic Assignment APIs, shared routes, permission keys and current E2E fixtures.
2. **P0 UI foundation:** replace the current single-page layout with the approved seven-tab School Admin workspace and a proper Staff Directory; retain existing server-backed create/edit/account/role/override behavior.
3. **P0 staff profile:** implement the eight-tab profile shell and complete each section only when its backend contract is available. Do not use fake or local-only saves.
4. **P0 add-staff wizard:** four steps, including optional login only after backend support; add credential sharing only as a safe one-time handoff with explicit user action.
5. **P0 academic assignments:** connect to the existing Academic assignment domain after its exact API and permission contract has been verified.
6. **P1 workflows:** attendance, leave, workload/timetable, reports in roadmap order.
7. **P2 workflows:** import/export, ID cards and payroll only under their roadmap gates.

## 5. Non-negotiable rebuild safeguards

- Replace only the Staff Management page and its dedicated styles/tests. Do not modify Academic, Student Management or Settings pages as part of this work.
- Preserve shared API client, session handling, route shell, design tokens, theme store, Modal, ConfirmDialog, DatePicker, Select and toast system.
- Keep school isolation, globally unique usernames, employee-code uniqueness, live permission checks, role delegation ceilings and audit behavior.
- All field errors appear beneath their fields; preserve non-secret form values after recoverable failures.
- Every page must handle loading, refresh, empty, permission denied, API/network failure, success and pending states.
- Verify responsive desktop/tablet/mobile, Light/Dark/System themes, keyboard/screen-reader behavior, production build and Playwright flows.
- No fake staff, assignment, attendance, leave or report data may be presented as real.
- No database migration is authorized by this audit. Any migration must be additive, reviewed, tested against a disposable database, and separately approved before applying to a persistent local database.

## 6. Audit limitations

This is a source-level repository audit on the reviewed branch/commit. It does not claim local database state, production behavior or authenticated browser acceptance. Exact Academic Assignment endpoint payloads and permissions, staff-specific upload support, and attendance/timetable reuse contracts still require focused inspection before implementation.


## 7. Follow-up implementation checkpoint (2026-11-21)

The initial workspace shell has since been extended on branch `work`:

- The Add Staff modal now follows the four-step roadmap wizard and validates each step before allowing progress. It uses only the fields accepted by the current transactional create endpoint.
- The final review displays the values entered in the previous steps.
- Successful creation opens a one-time optional credential handoff with Copy credentials / Do this later. No automatic message provider is assumed; clipboard copy is the only currently implemented handoff.
- The staff profile now exposes all eight roadmap tabs. Overview, Personal Details and Employment read current API-backed profile values; Account & Access reuses the existing role/override components.
- Qualifications & Experience now supports API-backed create/list/edit for education, certifications and previous experience. Documents remain blocked until private storage and authenticated download authorization are available.
- Academic Assignments now shows the selected staff member's subject-teaching and class-in-charge assignments, including revoked history, using existing school-scoped assignment APIs and exact read permissions.
- History & Activity now reads the latest 100 staff-related audit events through a staff-scoped API protected by `school.audit.read`. It includes account/profile, role, academic assignment and qualification/certification/experience events; it does not display secrets or raw audit metadata.
- New Playwright tests cover the wizard, credential handoff and profile tab shell. Execution is still pending.



## 8. Qualification records API checkpoint (2026-11-21)

- Qualifications, certifications and prior experience now have school-scoped create/list API operations, validation, audit events and PostgreSQL integration coverage. Backend CI passed on `0175725f`, including the full migration chain on a disposable PostgreSQL database, the complete Go test suite and `go vet`.
- The frontend profile section currently supports create/list only. Edit, verification and archive/delete remain pending. Documents remain unavailable until private storage and authenticated download authorization are implemented.
- The Add Staff browser suite currently has one unresolved date-picker/wizard progression failure (45 other browser tests passed). A more explicit wizard Continue action is under CI verification on `a9227c8`.
- These checks do not authorize or imply applying V57 to a user's persistent database. Academic, Student Management and Settings pages remain untouched.


## 9. Staff record edit workflow checkpoint (2026-11-21)

- Qualifications, certifications and employment experience now support school-scoped PATCH updates in addition to create/list. The repository scopes every update to the record ID, target staff principal and authenticated school, performs live School Admin authorization, preserves verification fields, and writes the corresponding audit event in the same transaction.
- The profile UI now exposes Edit actions and prefilled forms for all three record types. It reuses the existing validation rules, keeps the form open with entered values after API errors, and refreshes the affected record query after successful saves.
- Playwright coverage verifies editing each record type. Frontend CI **PASS** — [run 44918115850](https://github.com/TheSumitNain/school-management-system/actions/runs/44918115850), tested frontend checkpoint `a8c6b3e0`: production build and all 46 browser acceptance tests passed.
- Backend PostgreSQL integration coverage verifies updates, audit events and cross-school update denial. Backend CI **PASS** — [run 44918463838](https://github.com/TheSumitNain/school-management-system/actions/runs/44918463838), checkpoint `66339b2b`: full migration chain on disposable PostgreSQL, full Go tests and `go vet` passed.
- Verification is intentionally not exposed yet: the current permission catalogue does not include the proposed dedicated `staff.qualifications.verify` key. General `users.update` is not treated as verification authority. Document upload remains blocked pending private storage and authenticated download authorization.
- No migration was applied to the user's persistent database. Academic, Student Management and Settings pages remain untouched.


## 10. Staff academic assignments and audit history (2026-11-22)

- Staff profile → Academic Assignments now loads only the selected employee's subject-teaching and class-in-charge records. The existing school-wide assignment endpoints accept an optional `teacher_principal_id` filter; existing callers retain their previous list behavior. Active and revoked assignments are shown separately, and management remains in the existing Academic Assignments workspace.
- Staff profile → History & Activity now calls `GET /school/staff/:principalID/audit-events`. Route middleware and repository authorization both require `school.audit.read`, recheck the authenticated School Admin, active school, target staff membership and DENY override, and return at most the newest 100 relevant events.
- The timeline maps staff account/profile lifecycle, role assignment, subject/class-in-charge assignment and qualification/certification/experience create/update audit events. Existing staff-record audit events are associated through their same-school record IDs; new record audit events also persist the target staff principal ID in metadata. The UI displays only event labels, actor username, outcome and timestamp, not raw metadata.
- Added PostgreSQL integration assertions for history retrieval and cross-school denial, plus Playwright coverage for the staff-specific academic assignment and history panels. CI verification is in progress; do not mark this checkpoint complete until both workflows pass.
- No database migration or persistent database change was made. Academic, Student Management and Settings pages remain untouched.

## 11. Staff Attendance implementation checkpoint (2026-11-22)

Staff Attendance core register is implemented and CI-verified. Phase 3 remains open for search/filtering, bulk status entry and leave/calendar integration.

- Dedicated backend data model and permissions are in V58. Student attendance remains a separate module and data table.
- Daily roster and monthly derived register APIs are implemented with school scope and live effective-permission checks.
- Mark/update writes include append-only audit events in the same transaction. Past dates require the dedicated correction permission; future dates are blocked in both service and database trigger; duplicates return conflict.
- Staff profile History & Activity now recognizes staff attendance mark/update events without exposing raw audit metadata.
- Staff Management has Daily Register and Monthly Summary views, a school-local date picker, permission-aware marking/correction, and a confirmation step for historical corrections.
- Added PostgreSQL integration and Playwright coverage. Frontend CI [44979930421](https://github.com/TheSumitNain/school-management-system/actions/runs/44979930421) passed all 49 browser tests, including search/status filtering and atomic bulk marking; Backend CI [44979930424](https://github.com/TheSumitNain/school-management-system/actions/runs/44979930424) passed the full migration chain, Go/PostgreSQL integration suite (including bulk rollback/date guard) and `go vet` at code checkpoint `32df5ead`.
- Remaining Phase 3 work: downloadable/report views and approved leave/school-calendar integration. Do not call the full Staff Attendance phase complete until its roadmap exit gate is met.
- V58 remains unapplied to persistent/local databases. No changes were made to Academic, Student Management or Settings pages.

The detailed schema, permission, API, UI and verification contract is in [STAFF-ATTENDANCE.md](../backend/STAFF-ATTENDANCE.md).

## 12. Leave & Holiday Phase 4 source audit (2026-11-22)

**Reviewed branch:** `work`  
**Reviewed checkpoint:** `ddc99f0dd764b8ef9823b9c56866d74eaf968982`

The current source tree contains no staff leave or holiday implementation: no leave/holiday migration, repository, service, handler, route, frontend workspace, API client or leave/holiday permission keys were found. The school calendar is not currently a reusable holiday-management domain.

The existing staff attendance implementation is a separate foundation:

- `staff_attendance_records` stores one daily record per staff member and supports only `present`, `absent`, `late` and `half_day`.
- Daily and monthly APIs are school-scoped and protected by dedicated live attendance permissions.
- Attendance writes and audit events are transactional; duplicate marks conflict, future dates are rejected, and historical corrections require the dedicated permission.
- There is currently no `on_leave` attendance status, holiday calendar lookup, leave overlap check or leave-to-attendance integration. Do not infer leave from an absent/unmarked attendance row or mutate attendance merely because a leave request is approved.

### Phase 4 implementation boundary

1. Build school-scoped leave-type configuration with explicit paid/unpaid and approval-required attributes. Quota/accrual and carry-forward remain unset/disabled until the school has explicitly configured policy; do not show invented balances.
2. Build staff applications with leave type, inclusive start/end dates, full/half-day selection, conditional half-day session and required reason. Supporting documents remain unavailable until the private document-storage contract is implemented.
3. Build approve/reject decisions with authenticated actor and server timestamp. Rejection requires a comment. Only authorized School Admin actors may decide; applicants cannot approve their own requests unless a later explicit policy says otherwise.
4. Reject overlapping pending or approved applications for the same staff member. Reject invalid ranges, inactive staff, invalid half-day spans and requests against inactive leave types. Recheck all constraints inside the write transaction.
5. Keep approved leave as its own authoritative record. Attendance must expose approved leave as a derived calendar context; it must not silently create, overwrite or delete manual attendance rows. A future explicit attendance override/derivation workflow needs its own audited contract.
6. Build school holiday calendar entries with date, name, optional description and active status. No national/state holiday feed or automatic holiday import is assumed. Holiday/Off is calendar-derived and must not be written as a staff attendance mark.
7. Do not implement statutory entitlement, accrual, carry-forward, encashment, payroll deduction or automatic attendance deduction in this phase. These require separately approved school policy.
8. Add dedicated non-delegable leave/holiday permissions, tenant-isolation tests, transaction/audit tests, overlap and date-boundary tests, and browser acceptance for validation, approval, filters and responsive states before calling the phase verified.

**No persistent database was changed.** No migration has been created or applied at this audit checkpoint. Academic, Student Management and Settings pages were not touched.

## 13. Leave & Holiday implementation checkpoint (2026-11-22)

- Implemented V59 school-scoped leave types, leave requests and holiday calendar, with six dedicated non-delegable permissions granted to School Admin system roles for existing and future schools.
- Backend APIs now support leave type create/list/update, request create/list/approve/reject/cancel and holiday create/list/update. Repository operations enforce same-school membership, active staff/type checks, inclusive date validation, half-day shape, overlap detection under a per-school/staff transaction advisory lock, pending-only decisions, required rejection comment, applicant self-approval denial and transactional audit events.
- Staff Management now has a permission-gated Leave & Holidays workspace with request listing/filtering, leave type creation, holiday calendar create/edit/deactivate, approve/reject modal and cancellation confirmation. Rejection requires a comment; no balances are shown.
- The request UI always sends new requests to Pending, even when a leave type's `requires_approval` flag is false. Auto-approval is not enabled until its policy is explicitly designed.
- Frontend browser acceptance now covers the staff leave workspace, approval confirmation, configured leave types and holiday calendar. Frontend CI [44982323582](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323582) passed production build and all 50 Playwright tests. Backend CI [44982323577](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323577) passed the full migration chain on disposable PostgreSQL, Go/PostgreSQL tests and `go vet`.
- CI-tested code checkpoint: `816ee99511d4a7ed0e393dda9a04dc758f954202`. V59 remains unapplied to persistent/local databases.
- Remaining Phase 4 work: dedicated PostgreSQL behavior tests for overlap/concurrency, authorization, rollback and audit; leave-type edit/deactivation UI and approved school policy; approved leave/holiday context in staff attendance. No leave balance, accrual, carry-forward, payroll deduction, statutory entitlement or supporting-document upload is implemented.
- Academic, Student Management and Settings pages were not modified.

## 14. Leave & Holiday verification update (2026-11-22)

This section supersedes the earlier Phase 4 notes where they differ.

- The latest `work` code checkpoint is `8c163b5b35a644cf08a13e280c0c44f238a80037`.
- Backend CI [44984389499](https://github.com/TheSumitNain/school-management-system/actions/runs/44984389499) **PASS** at that exact checkpoint: V59 and the complete migration chain applied to disposable PostgreSQL, Go tests including the new leave/holiday service validation tests, and `go vet`.
- Frontend code remains at the previously verified UI checkpoint `36ab57484de809c30849c030187f0190dab43a60`; Frontend CI [44982936799](https://github.com/TheSumitNain/school-management-system/actions/runs/44982936799) **PASS**, including the leave workspace browser acceptance.
- Approved leave and active holiday context is now displayed alongside the staff daily attendance roster when the viewer has the corresponding read permissions. These are informational overlays only; attendance rows/statuses are not created or changed.
- Service validation now rejects reversed leave dates, invalid duration/half-day combinations and invalid holiday date/name/description before repository writes. Repository overlap locking uses a hashed school/staff advisory key; cancellation and holiday range reads validate nil/invalid inputs.
- Remaining Phase 4 work: deeper PostgreSQL behavior coverage for concurrent overlap, decision authorization, rollback and audit; leave-type edit/deactivation controls; policy approval and any future balance rules. Current V59 integration coverage verifies schema and permission provisioning; do not overstate it as full workflow integration coverage.
- V59 remains unapplied to persistent/local databases. No Academic, Student Management or Settings pages were modified.

## 15. Leave type management and final CI checkpoint (2026-11-22)

- Added Staff Management UI to edit leave type name, paid/unpaid, approval flag and active status. Deactivation requires explicit confirmation and preserves all existing leave requests/history. Inactive types are excluded from new request options; they can be reactivated through Edit.
- Added Playwright coverage for editing and deactivating a leave type. Frontend CI [44984794710](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794710) **PASS** at code checkpoint `f1b71d4e2999f868f62910ee878f78eee1460a8b`: TypeScript/Vite production build and all 51 browser tests passed.
- Backend CI [44984794698](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794698) **PASS** at the same checkpoint: full migration chain on disposable PostgreSQL, Go tests including new leave/holiday service validation tests, and `go vet`.
- Phase 4 remains partially implemented pending dedicated PostgreSQL behavior tests for overlap/concurrency, decision authorization, rollback and audit. School-specific policy and balance rules remain unconfigured. V59 remains unapplied to persistent/local databases.

## 16. Leave workflow PostgreSQL behavior tests (2026-11-22)

- Added real PostgreSQL integration coverage for concurrent overlapping submissions (exactly one succeeds), unauthorized decisions, applicant self-approval prevention, approval/rejection audit persistence, rollback when the audit insert fails, rejection-comment requirement and cross-school cancellation denial.
- The new concurrency test exposed a pgx parameter-type issue in the overlap advisory-lock query. Fixed the query to type both identifiers as BIGINT before text conversion.
- Backend CI [44985769667](https://github.com/TheSumitNain/school-management-system/actions/runs/44985769667) **PASS** at exact code checkpoint `e028baffd2485afeee792f98b068f89e2ce1ae07`: full migration chain on disposable PostgreSQL, all Go tests including the new V59 workflow tests, and `go vet ./...`.
- The audit-failure test intentionally raises a database exception to prove the leave decision transaction rolls back; the resulting PostgreSQL error in CI logs is expected by the test.
- V59 remains unapplied to persistent/local databases. Remaining Phase 4 work includes further negative school-isolation/authorization cases, frontend review of full leave workflows, attendance reporting integration, and school-approved leave policy/balance rules.

## 17. Attendance monthly calendar context (2026-11-22)

- Fixed daily attendance status filtering so it works even when the employee search field is empty; E2E coverage now clears search before asserting status-only filtering.
- Monthly attendance view adds a separate approved-leave and active-holiday context section. This context is permission-gated and never included in present/absent/late/half-day attendance totals.
- Added bounded month-overlap filtering to the leave request list API, service validation and PostgreSQL integration coverage.
- Backend CI [44986574096](https://github.com/TheSumitNain/school-management-system/actions/runs/44986574096) **PASS** at `a01514119b0d395b106accd990f59b2f7a5e992f`; frontend production build passes. Playwright is pending after correcting an E2E route mock that initially intercepted the daily context request instead of falling back to the existing mock.
- No persistent/local database migration. Leave balance policy and payroll effects remain unconfigured.


## 18. Staff Management permission and workflow acceptance update (2026-11-22)

This section supersedes earlier Phase 4 remaining-work notes where later implementation checkpoints have completed the item.

- Leave type edit/deactivation and reactivation, leave request filters, valid half-day application, rejection-comment validation, cancellation confirmation, holiday deactivation/reactivation confirmation, holiday form error recovery and pane-specific error isolation now have browser acceptance coverage.
- Attendance daily/monthly context keeps approved leave and active holidays informational and separate from manual attendance; monthly CSV reporting keeps the categories separate and protects spreadsheet formula injection.
- Staff Management page heading and section tabs now follow the shared Student Management dashboard visual language, including underline active tabs, visible keyboard focus and horizontal mobile scrolling.
- Added a read-only holiday calendar acceptance case for staff.holidays.read without staff.holidays.manage: active calendar remains visible, inactive entries are not requested, and mutation controls are absent.
- Exact checkpoint eaaea2f2f0f4a3665221a3fe0d1807f404eac6a7: Frontend CI [45001889888](https://github.com/TheSumitNain/school-management-system/actions/runs/45001889888) **PASS** with the full Playwright browser suite; Backend CI [45001889885](https://github.com/TheSumitNain/school-management-system/actions/runs/45001889885) **PASS** with PostgreSQL integration tests and go vet.
- Leave balances, quotas, accrual, carry-forward, statutory entitlement and payroll remain out of scope until the school approves policy. No new persistent/local database change was made by these acceptance and styling updates.
- A further Staff Account & Access test now checks that a user with roles.read but without roles.assign can inspect assigned roles but receives no assign/remove controls, and does not trigger permission-override data requests. Its CI verification is pending at code checkpoint 15d74d324de85425a94039c45c715478d153cae9; do not treat that test as verified until both workflows finish.
## 19. Staff Account & Access read-only role acceptance (2026-11-22)

- Added a Staff profile Account & Access browser test for an account with roles.read but without roles.assign or permissions.override.
- The test verifies the assigned role remains visible, role assignment/removal controls are absent, and the profile does not request permission catalogues, delegation data or individual override data when the viewer lacks the required permission.
- Exact code checkpoint 15d74d324de85425a94039c45c715478d153cae9: Frontend CI [45002264200](https://github.com/TheSumitNain/school-management-system/actions/runs/45002264200) **PASS**, including TypeScript/design-token checks, Vite build and complete Playwright browser suite; Backend CI [45002264188](https://github.com/TheSumitNain/school-management-system/actions/runs/45002264188) **PASS**, including PostgreSQL integration tests and go vet.
- This closes the reviewed read-only Staff Account & Access acceptance gap; no backend behavior or database schema changed.

## 20. Staff Account & Access role assignment/removal confirmation (2026-11-22)

- Added a positive-permission Playwright workflow for a staff profile viewer with `users.read`, `roles.read` and `roles.assign`: the user assigns a custom role, verifies the success state, opens role removal and cancels to prove no DELETE request occurs, then explicitly confirms removal and verifies the updated role list.
- The test asserts exactly one assignment request and exactly one confirmed removal request; cancellation leaves the assignment unchanged.
- Exact code checkpoint `6ca0fb6e8ffe3b80c86446d2dfc9d8cc6521bd7d`: Frontend CI [45002546475](https://github.com/TheSumitNain/school-management-system/actions/runs/45002546475) **PASS**, including TypeScript/design-token checks, Vite production build and the complete Playwright browser suite. Backend CI [45002546411](https://github.com/TheSumitNain/school-management-system/actions/runs/45002546411) **PASS**, including disposable PostgreSQL integration tests and `go vet`.
- This closes the reviewed positive role-assignment/removal confirmation acceptance gap. No backend runtime change or database migration was introduced by this test.


## 21. Staff individual permission override acceptance (2026-11-22)

- Added a positive-permission Playwright workflow for `permissions.override`: the staff administrator loads only permissions delegated for individual overrides, selects a permission, is blocked from saving without a reason, then saves a documented DENY override and verifies the submitted permission, effect, reason and delegation ID.
- The test verifies removal is confirmation-gated: cancelling produces no DELETE request; explicit confirmation removes the override and updates the visible list. This complements the role assignment/removal workflow and read-only access coverage.
- Exact checkpoint `d3d00fdd578e42dd9bcc89d0826afb9b9a471368`: Frontend CI [45003372900](https://github.com/TheSumitNain/school-management-system/actions/runs/45003372900) **PASS** — TypeScript/design-token checks, Vite production build and all 66 Playwright tests. Backend CI [45003372918](https://github.com/TheSumitNain/school-management-system/actions/runs/45003372918) **PASS** — Go tests, PostgreSQL integration and `go vet`.
- No database migration or backend runtime change was introduced by this acceptance-test update.


## 22. Permission-override-only delegation metadata access (2026-11-22)

- Review found that the Staff Permission Overrides panel is authorized by `permissions.override`, but its supporting permission catalogue and active-delegation reads previously required the separate `permissions.read` and `permissions.assign` grants. That made the UI fail for a School Admin whose access was intentionally narrowed to `permissions.override`.
- Updated the two metadata GET routes to accept either the existing read/assignment permission or `permissions.override`. Updated the repository's delegation-list authorization to accept either `permissions.assign` or `permissions.override`; the query still enforces an active same-school School Admin and honors explicit DENY overrides. Mutation routes and delegation ceilings are unchanged.
- Added/adjusted the browser acceptance fixture to grant `users.read` and `permissions.override` only (no `roles.read`, `permissions.read` or `permissions.assign`), then load the delegated permission, require a reason, save the override and confirm removal.
- Frontend acceptance commit `4599533c0d59bf9e3a5f7da6a0ce22a14fea64a4`: Frontend CI [45003723942](https://github.com/TheSumitNain/school-management-system/actions/runs/45003723942) **PASS**, including production build and complete Playwright suite. Backend code checkpoint `f8f1b8764635762089414334028855f92f510403`: Backend CI [45003754631](https://github.com/TheSumitNain/school-management-system/actions/runs/45003754631) **PASS**, including Go tests, PostgreSQL integration and `go vet`.
- No schema or migration change was required.


## 23. Staff account disable confirmation acceptance (2026-11-22)

- Added a Staff Directory browser workflow for an account with `users.read` and `users.disable`: the test opens the disable dialog, verifies the session-revocation warning, cancels and asserts no DELETE request, then confirms disable and verifies exactly one request, the success notification, refreshed disabled status and removal of the disable action.
- Exact code checkpoint `7c77bef8f752557a049fd4a05471b2881a2f2b79`: Frontend CI [45004040475](https://github.com/TheSumitNain/school-management-system/actions/runs/45004040475) **PASS** with all 67 Playwright tests; Backend CI [45004040409](https://github.com/TheSumitNain/school-management-system/actions/runs/45004040409) **PASS** with PostgreSQL integration and `go vet`.
- This is acceptance-test coverage only; no runtime behavior or database schema changed.


## 24. Staff password reset acceptance (2026-11-22)

- Added a Staff Directory browser workflow for `users.password_reset`: a short temporary password is rejected inline with no network mutation; a valid 12+ character temporary password is submitted once, and the success message explicitly tells the staff member to change it at next sign-in.
- The test asserts the exact request payload (`new_password`) and that the validation error clears when the password is corrected.
- Exact checkpoint `90d7969babd87e60db19cebb57f702ef93f75ed0`: Frontend CI [45004357985](https://github.com/TheSumitNain/school-management-system/actions/runs/45004357985) **PASS** with all 68 Playwright tests; Backend CI [45004358053](https://github.com/TheSumitNain/school-management-system/actions/runs/45004358053) **PASS** with PostgreSQL integration and `go vet`.
- Acceptance-test-only change; no backend runtime behavior or database schema changed.


## 25. Staff account enable acceptance (2026-11-22)

- Added a browser workflow for re-enabling a disabled but employed staff account with `users.update`: a temporary password shorter than 12 characters is rejected inline without a request; a valid password enables the account, refreshes the directory status, and displays the required next-sign-in password-change state.
- The test verifies exactly one enable request with the `temporary_password` payload and confirms the Enable action is no longer shown after refresh.
- Exact checkpoint `247686b29c6e30d0b6de2b542b74be9faa42fc7d`: Frontend CI [45004695381](https://github.com/TheSumitNain/school-management-system/actions/runs/45004695381) **PASS** with all 69 Playwright tests; Backend CI [45004695364](https://github.com/TheSumitNain/school-management-system/actions/runs/45004695364) **PASS** with PostgreSQL integration and `go vet`.
- Acceptance-test-only change; no runtime or database schema change.


## 26. Staff leave request PostgreSQL validation and eligibility coverage (2026-11-22)

- Expanded real-PostgreSQL integration coverage for malformed/reversed dates, invalid calendar dates, blank reasons, unsupported durations, missing/invalid half-day sessions and multi-date half-day requests.
- Added eligibility and lifecycle assertions: approved leave blocks overlapping requests; cancellation releases the date for a new request; inactive leave types, inactive staff profiles and cross-school staff targets are rejected.
- Exact checkpoint `062d9a2d9acba4d939985c1d6a0cdb2907cd247b`: Backend CI [45005336749](https://github.com/TheSumitNain/school-management-system/actions/runs/45005336749) **PASS**, including all Go tests, PostgreSQL integration and `go vet`. No migration or runtime change was needed.


## 27. Staff overview metrics and account-readiness acceptance (2026-11-22)

- Expanded the Staff Overview browser fixture to include two deliberately different staff states: an active employee with an incomplete login-readiness state, and an inactive employee with an incomplete profile. The acceptance assertions independently verify total staff, active employment, enabled logins, incomplete profiles, and readiness counts rather than conflating employment, profile and account status.
- Exact checkpoint `176ce3e85fa8cb4d621066e88207c63b35c9c740`: Frontend CI [45005957248](https://github.com/TheSumitNain/school-management-system/actions/runs/45005957248) **PASS**, including TypeScript, production build and the complete Playwright browser acceptance suite. Backend CI [45005957357](https://github.com/TheSumitNain/school-management-system/actions/runs/45005957357) **PASS**, including Go tests, ephemeral PostgreSQL migrations/integration and `go vet`.
- Acceptance-test-only change; no backend runtime change or persistent database migration.


## 28. Staff directory independent filtering (2026-11-22)

- Added combined text search and independent filters for employment status (active/inactive), login-account status (enabled/disabled) and profile completeness (complete/incomplete). Result counts update with the combined filters; shared accessible Select controls are responsive on narrow screens.
- Corrected an important semantic distinction during review: profile completeness is derived only from the profile-completeness field, never inferred from employment status. Browser coverage deliberately includes an active employee with an incomplete profile to prove these filters remain independent.
- Exact checkpoint `674133cf169619a236bf88184bb971b53471f86d`: Frontend CI [45007749959](https://github.com/TheSumitNain/school-management-system/actions/runs/45007749959) **PASS**, including TypeScript, production build and complete Playwright suite. Backend CI [45007750034](https://github.com/TheSumitNain/school-management-system/actions/runs/45007750034) **PASS**, including Go tests, ephemeral PostgreSQL migrations/integration and `go vet`.
- Frontend-only behavior; no backend runtime change or persistent database migration.

## 29. Workload summary integration contract review (2026-11-22)

- The existing Timetable domain already stores the fields needed for a teacher workload summary: academic year, teacher principal ID, section/class, subject, weekday and period start/end times. The Staff profile currently has no workload panel.
- The existing timetable entries read endpoint is protected by `timetable.read` and accepts academic-year and optional section filters only. It does not expose a server-side teacher-principal filter. Fetching the entire school timetable into a staff profile and filtering in the browser would unnecessarily disclose unrelated staff schedules to that client and is not an acceptable implementation shortcut.
- Before adding the Phase 5 workload summary, extend the existing read contract with a server-side teacher filter (or a dedicated staff-scoped read route), validate target staff membership in the authenticated school, enforce the live timetable read permission, and add PostgreSQL tests for cross-school denial and target scoping. The profile should show period counts and weekly schedule derived from real timetable entries only; no guessed workload quota or synthetic availability.
- The next implementation checkpoint is therefore a narrow, additive API/query-contract change plus integration and browser acceptance tests. No migration is currently justified by this review; do not apply any migration to a persistent database.


## 30. Teacher-scoped workload view implementation checkpoint (2026-11-22)

The initial staff workload view is now implemented without a database migration:

- The existing `GET /school/timetable/entries` read API accepts an optional positive `teacher_principal_id` filter. The filter is applied in the PostgreSQL query, not in the browser.
- A requested teacher must have a `staff_profiles` row in the authenticated school. The existing teacher-only actor boundary remains in force, so a teacher cannot use the filter to read another teacher's schedule. The route and repository continue to enforce live `timetable.read` authorization.
- The frontend API helper exposes the optional filter. A selected employee's profile now includes a permission-gated **Workload & Timetable** tab that loads the current academic year and that employee's scheduled entries only.
- The panel shows scheduled-period, teaching-day, section and subject counts, actual timetable rows in a weekly grid, and unscheduled timetable slots derived from the real period catalogue. It explicitly states that unscheduled slots do not confirm teacher availability, working days or permission to assign another class.
- Handler tests cover forwarding a valid teacher scope and rejecting a non-positive filter. A Playwright test verifies that the profile requests a teacher-specific timetable and renders the schedule, unscheduled slots and availability caveat.

Verification on branch `work`, commit `30270fe0927a6eb3f284d462ddd6dc2b4ca2959e`:
- Backend CI run `45009158991`: PASS — all migrations on the ephemeral PostgreSQL database, full Go test suite including PostgreSQL integration, and `go vet`.
- Frontend CI run `45009158987`: PASS — TypeScript/Vite production build and complete Playwright browser acceptance suite (71 tests).
- A dedicated PostgreSQL integration assertion was added to `TestAcademicFoundationAndStudentLifecyclePostgresIntegration`: School Admin can read one selected same-school teacher's schedule; selecting a second same-school teacher returns only that teacher's schedule; a Teacher cannot read a peer's schedule even when supplying the peer principal ID; a foreign-school principal ID returns no entries.
- Backend CI run `45009905703` on commit `1a40546d25a66ab8714789b5835be1afb0661b34`: PASS — migration application, full Go/PostgreSQL integration suite including the new tenant/actor-scope assertions, and `go vet`.
- The weekly grid and unscheduled-slot display passed frontend CI run `45010205988` (TypeScript/Vite production build and all 71 Playwright tests); backend CI run `45010206045` also passed on the same source checkpoint.
- No persistent database was changed and no migration was added.

**Remaining Phase 5 work:** server-authoritative workload/conflict reporting and the separately authorized, audited substitute workflow. The initial teacher-specific profile workload view, its browser acceptance, and its PostgreSQL tenant/actor-scope checks are verified; this does not mark all of Phase 5 complete.


## 31. Timetable conflict feedback hardening (2026-11-22)

- The V26 database already prevents double-booking with unique weekly-slot indexes for teacher and section. The write path previously collapsed both constraint violations into one generic HTTP 409 message, which made concurrent edits and stale browser data difficult to resolve.
- Repository error mapping now distinguishes `ux_timetable_entries_teacher_slot` from `ux_timetable_entries_section_slot`. The timetable handler returns stable conflict codes and actionable messages: `timetable_teacher_conflict` (choose another teacher or period), `timetable_section_conflict` (choose another period), and a safe generic `timetable_slot_conflict` fallback for unknown uniqueness/exclusion conflicts.
- Added repository unit coverage for PostgreSQL constraint-to-domain-error mapping and handler coverage for the HTTP 409 code/message contract. Existing database uniqueness remains authoritative; no migration or persistent database change was made.
- API contract documentation updated in `docs/backend/API.md`.

**Verification:** Backend CI run `45010784607` passed — ephemeral PostgreSQL migration application, full Go/PostgreSQL integration suite, and `go vet`.


## 32. Staff directory CSV formula-injection regression coverage (2026-11-23)

- The directory export already neutralizes formula-leading values after leading whitespace and writes UTF-8 BOM/CRLF CSV. The browser acceptance fixture previously covered a value beginning directly with `=`; it now begins with a tab followed by `=HYPERLINK(...)` and asserts the exported cell receives a protective apostrophe before the tab.
- This is a test-only change; no runtime code, API contract, migration or persistent database change.
- Exact checkpoint: `774a74c12805a71134ae9827b80f78c9ff37ef7f` ([commit](https://github.com/TheSumitNain/school-management-system/commit/774a74c12805a71134ae9827b80f78c9ff37ef7f)). Frontend CI is pending; the new assertion is not yet marked verified. A follow-up workload-report fixture also covers a formula preceded by ordinary spaces (commit `d165907d30941bf14171b8696aca4d7baa837c4d`).


## 33. Staff summary reports workspace (2026-11-23)

- Replaced the disabled Reports navigation item with a permission-gated Staff Reports workspace (`users.read`). It uses the existing school-scoped staff query; no new endpoint or duplicate reporting table was introduced.
- Summary metrics: total staff, active/inactive employment, incomplete profiles and disabled login accounts.
- Export is limited to approved operational fields (employee code, name, designation, employment status, login status, profile completeness); contact details and secrets are intentionally omitted. CSV retains BOM, quoting and formula neutralization through the existing shared directory export helper.
- Added responsive summary layout and a Playwright acceptance test for counts and sensitive-field exclusion.
- Code: `c0d5301acb474d16aad3cda9ca1659681f55738b`; CSS: `b7caf141000cef1af3cedc6bd7613980f778c622`; test: `f07589d40f867ceaffbabb5df73e82c1e6b5a87e`.
- Verification remains pending CI; do not mark the browser test as passed yet. Qualifications, assignments, attendance and leave reporting remain separate Phase 6 work.



## 34. Monthly attendance and leave reporting (2026-11-23)

- Expanded the permission-gated Reports tab with monthly attendance summaries and date-overlap leave request summaries. Attendance is fetched only with `staff.attendance.read`; leave requests are fetched only with `staff.leave.read`. The frontend does not issue either query without its matching permission.
- Reused existing school-scoped APIs: `GET /school/staff-attendance/month` and `GET /school/staff-leave/requests?from_date=...&to_date=...`. No backend route, migration, database or student/academic module was changed.
- Added separate CSV exports for attendance and leave. Exported fields omit attendance remarks and leave reasons; cells are quoted and formula-leading values are neutralized. Leave balances and entitlement calculations remain out of scope.
- Added Playwright acceptance coverage for both report panels, exports, and absence of attendance/leave API requests when their read permissions are missing.
- Checkpoints: workspace `443735c`; styles `bde8a11`; integration `0a420ad`; E2E `5320ed6`; BOM assertion correction `62e6687`; CSV sanitizer correction `a282d12`. The first frontend CI attempt exposed a TypeScript regex parse error, which was corrected in `a282d12`; CI for the corrected checkpoint is running, so browser verification remains pending.


## 35. Academic assignment reporting (2026-11-23)

- Added a current-academic-year report aggregating active teacher-subject and class-in-charge assignments using existing APIs. Revoked assignments are excluded. Data requests are independently permission-gated by `teacher_assignments.read` and `class_incharge_assignments.read`.
- CSV exports contain usernames and aggregate counts only, and omit columns for assignment types the viewer cannot read. No new backend route, migration, schema or database change was introduced.
- Playwright covers the assignment summary and export, including revoked assignment exclusion. Latest browser CI is pending; do not treat it as browser-verified until CI completes successfully.
- Checkpoints: report `53ae4e4`; encoding fix `7e5b704`; active/permission export fix `0ff6cb3`; E2E `7b2cfe8`; BOM test fix `8de8bb7`.


## 36. Qualification, certification and experience reporting (2026-11-23)

- Implemented a school-wide Qualifications & Certifications report with one bounded aggregate endpoint, record-type and verification-status filters, count metadata, pagination and allowlisted CSV fields. The backend repository enforces active principal/user/school, School Admin role, live `users.read` permission and explicit DENY override; queries constrain staff to the authenticated school. PostgreSQL integration coverage exercises mixed record types, filters, pagination and cross-school denial. Frontend UI uses a single aggregate endpoint rather than per-staff N+1 calls.
- Implemented a school-wide Experience History report using one bounded aggregate endpoint and the existing experience schema. It supports employment-type/current-role filters, retains historical records for inactive employees, uses school-scoped authorization and exposes only report-safe fields. Frontend includes pagination, loading/error/empty states and CSV export. Backend and frontend tests are included; final browser CI evidence is tracked separately below.
- The Documents tab remains disabled until the user purchases/selects an external storage service and explicitly approves integration. No local-disk storage, upload UI, file APIs or document migration may be introduced before that approval.
- No persistent database migration or data change was made in these report implementations.

## 38. Directory profile-completeness semantics and onboarding schema gap (2026-11-23)

- Corrected Staff Directory profile-completeness behavior: only an explicit `profile_complete === true` is classified as Complete. Explicit false and missing/unknown values are both surfaced as Incomplete / needing attention, and the Complete filter no longer treats unknown values as complete.
- Updated the directory export, overview metric, report summary and directory status label to use the same conservative rule. Added Playwright fixture coverage for a legacy/unknown profile-completeness value and assertions for both Complete and Incomplete filters.
- Frontend build passed on checkpoint `f9b6024`; Backend CI [45120632271](https://github.com/TheSumitNain/school-management-system/actions/runs/45120632271) **PASS**. The first Playwright run [45120632286](https://github.com/TheSumitNain/school-management-system/actions/runs/45120632286) found a test setup issue (the Disabled account filter was still active when checking an active unknown-profile fixture). The test now resets that filter to All accounts; frontend rerun [45121009614](https://github.com/TheSumitNain/school-management-system/actions/runs/45121009614) is queued. Do not mark the new filter assertions verified until rerun completes.
- Confirmed the Add Staff wizard currently requires username and initial password for every employee. The roadmap's optional login Yes/No and profile-only onboarding are target behavior, not currently implemented. Updated the bulk-import architecture gate and roadmap to state that profile-only single-create and bulk import are both blocked by the current `staff_profiles.principal_id -> users` coupling. No credentials, placeholder accounts or migration were introduced.

## 37. Staff ID cards and bulk import architecture gate (2026-11-23)

- Added a permission-gated ID Cards workspace with multi-select, Classic/Compact/Modern layouts, responsive preview, optional validity date and print-only output. Eligibility requires `employment_status === "active"`, `profile_complete === true`, employee code, full name and designation. Cards include only employee name, designation, employee code, joining date, employment status and optional validity; no phone/email, login details, permissions or QR code are included.
- Added Playwright coverage for active/inactive/incomplete/unknown profile eligibility, safe field allowlist, layout selection, initials and empty-selection behavior. Initial CI exposed a strict-mode locator collision because the employee name appeared in both selector and preview. The test was corrected to scope assertions to the print preview; the latest frontend CI rerun is pending.
- Bulk Import is blocked on the current identity schema: `staff_profiles.principal_id` is a required FK to `users`, but the approved workflow requires employee profiles without mandatory login accounts. The existing create-with-profile path creates login credentials. Do not loop over it, generate usernames/passwords or create a migration without explicit review/approval. [Staff Bulk Import Architecture Gate](../backend/STAFF-BULK-IMPORT-DESIGN.md) defines the target identity separation, compatibility/backfill review and all-or-nothing import acceptance contract.
- Backend CI for the ID card checkpoint [45119045035](https://github.com/TheSumitNain/school-management-system/actions/runs/45119045035) passed Go tests, PostgreSQL integration and `go vet`. Frontend CI [45119045034](https://github.com/TheSumitNain/school-management-system/actions/runs/45119045034) production build passed, but Playwright failed on the duplicate-name locator; corrected assertions and stricter eligibility/initials handling are now committed for rerun.
- No database migration was added/applied. Document upload remains deferred pending storage-service purchase and explicit approval.


## 39. Directory filters, conservative completeness and read-only profile permissions (2026-11-23)

- Directory completeness semantics are conservative: `profile_complete === true` alone is Complete. Explicit false and missing/unknown values are Incomplete. The Complete/Incomplete filters, summary metrics, report counts and CSV classification follow this rule.
- Re-enable eligibility requires active employment and an explicitly complete profile. The backend already rejects incomplete-profile re-enable attempts; the frontend now mirrors that restriction and does not show an Enable action for legacy/unknown profiles.
- Staff profile edit controls are gated by `users.update` in directory rows and the Overview, Personal Details and Employment profile sections. Qualification, certification and experience create/edit controls are also permission-gated. Added a read-only regression test for the three profile sections.
- Added responsive Staff Directory filter coverage for 390px mobile, 768px tablet and 1280px desktop, asserting all three controls remain visible, labels stay within the filter container, no horizontal overflow occurs, and mobile filter selection works.
- Checkpoint commits: `f686857` (profile edit permission gate), `6049464` (read-only regression test). The prior Frontend CI on `7fa70e1` failed because the filter test called `page.reload()`, which discarded the in-memory mocked login and prevented navigation back to Staff Directory. The current test resets account and employment filters in-place without reloading. Fresh frontend and backend CI for the combined checkpoint remains required; do not mark this section as CI-verified yet.
- No persistent database migration or data change was made. Documents remain deferred pending external storage-service purchase and explicit approval. Bulk import/profile-only onboarding remain blocked by the identity-schema gate and require separate approval before any migration.


## 40. Academic assignment profile navigation permission gate (2026-11-23)

- The employee Academic Assignments profile panel previously displayed a navigation button even when the viewer lacked both `teacher_assignments.read` and `class_incharge_assignments.read`. The panel correctly withheld the data, but the navigation affordance was misleading and allowed entry into an inaccessible workspace.
- The navigation action is now rendered only when at least one corresponding read permission exists. Its label is “Open assignments workspace” to describe the read/navigation action without implying write access.
- Extended the read-only profile Playwright test to open Academic Assignments, assert the denied explanation, and assert the workspace navigation action is absent.
- Commits: `b2c4655` (permission-gated navigation), `76b9880` (regression coverage). Do not mark CI verified until fresh frontend and backend checks for the latest branch head complete.


## 41. Attendance monthly CSV privacy (2026-11-23)

- A secondary export path in the Attendance workspace (`Download monthly report`) was found to include free-text attendance remarks, even though the operational attendance/leave report export already excludes remarks and leave reasons.
- Removed the Remarks column and its source values from the monthly export. The export remains a bounded monthly view of attendance summaries/entries plus approved leave and holiday context; leave reasons are not exported.
- Extended the existing attendance Playwright scenario with unique sentinel values for a private attendance remark and a private leave reason. It downloads the CSV and asserts neither sentinel is present while the employee code remains present.
- Commits: `9800956` (privacy removal), `bfc813e` and `41e92b7` (CSV row alignment), `a6e9684` (regression test). Fresh CI for the latest combined branch head is still required.


## 42. Staff substitution authorization and attendance CSV test corrections (2026-11-23)

- Staff substitution create and cancel operations now pass the transaction handle to the live RBAC evaluator before performing the mutation. This places authorization evaluation in the same transaction context as the write. The change does not claim serializable isolation or eliminate every possible concurrent permission-revocation race.
- The first fresh Frontend CI after the monthly attendance CSV privacy change built successfully; 83 Playwright tests passed and two test assertions failed. The product export remained correct: the BOM test was checking a literal escaped sequence rather than the BOM character, and formula-injection coverage was still attached to the removed remarks field.
- Corrected the BOM assertion to use the U+FEFF character and moved the formula-injection fixture to an exported staff-name value. Private attendance remarks and leave reasons remain excluded from the monthly CSV.
- Commits: `b06ce37` (transaction-context authorization), `f64296f` (BOM assertion), `78eac8f` (formula-injection test fixture). Latest CI for `78eac8f` must complete successfully before acceptance is marked green.
- No persistent database changes or migrations were made. Documents and Bulk Import remain subject to their existing approval gates.


## 43. Staff audit metadata response minimization (2026-11-23)

- Staff History & Activity needs only event action, resource type, outcome, actor username and timestamp. Raw audit metadata is no longer selected or scanned as result data; it may still be referenced in SQL predicates to identify relevant events. Arbitrary audit payloads are not sent to the browser.
- Added a regression test proving sensitive/raw metadata values are absent from serialized StaffAuditEvent JSON while safe activity fields remain present.
- Commits: `59430b1` and `80847f6` (remove metadata from response/result projection), `0b967c5`, `70ff326` and `6f5cc96` (serialization regression test), and `f1be661` (remove unused import). Backend CI [45131300374](https://github.com/TheSumitNain/school-management-system/actions/runs/45131300374) **PASS**: all migrations on ephemeral PostgreSQL, Go tests including PostgreSQL integration, and `go vet`.
- No database migration or persistent data change was made. Documents and Bulk Import approval gates remain unchanged.

## 44. Staff record read authorization boundary (2026-11-23)

- Staff qualifications, certifications, experience and Staff History & Activity reads require the active same-school system School Admin role with `users.read`. This is intentional: migration `000004_add_principal_rbac_foundation.up.sql` marks `users.read` as non-delegable, so custom roles and user ALLOW overrides cannot validly grant this sensitive staff-wide read capability.
- Preserve the backend School Admin role check and DENY override precedence. Do not broaden these endpoints to arbitrary custom roles or treat a UI permission label as authority; the backend remains the final enforcement point.
- The initial custom-role integration-test experiment was invalid because it attempted to delegate a permission explicitly marked non-delegable. That test was removed; no product permission policy or persistent database schema was changed.

## 45. Staff records handler availability guards (2026-11-23)

- Qualifications, certifications, experience, staff audit history and both aggregate report handlers now fail closed with a consistent HTTP 503 response when the handler receiver or service dependency is unavailable. This avoids nil-pointer panics and does not expose partial or fabricated records.
- Added table-driven regression coverage for nil receivers and nil service dependencies across all 12 handler methods.
- Backend CI [45133291269](https://github.com/TheSumitNain/school-management-system/actions/runs/45133291269) **PASS** on checkpoint `b2fa865e`: ephemeral PostgreSQL initialization and migrations, Go tests including PostgreSQL integration, and `go vet`.
- No migration, persistent data change, or document/bulk-import implementation was introduced. Browser acceptance still needs final full-suite confirmation on the current combined branch head.


## 46. Staff account and leave handler availability guards (2026-11-23)

- Staff account create/list/disable/update/re-enable handlers now have regression coverage for nil handler receivers and unavailable services. Staff leave and holiday endpoints share a guarded actor/dependency check; tests cover all leave-type, request, decision, cancellation and holiday handler actions for nil receivers and nil service dependencies.
- This is a fail-closed reliability safeguard: unavailable dependencies return HTTP 503 rather than panic. It does not bypass authentication, authorization, or service validation when dependencies are available.
- Commits: `b124c23` (leave service guard and tests), `566ce3d` (staff account handler tests). Backend CI [45134091085](https://github.com/TheSumitNain/school-management-system/actions/runs/45134091085) **PASS** on `566ce3df`: ephemeral PostgreSQL migrations, Go tests including PostgreSQL integration, and `go vet`. Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) **PASS** on `bb659f1c`: Vite production build and Playwright browser acceptance suite.
- No migration or persistent database change was made. Staff document upload remains deferred until external storage is selected/purchased and explicitly approved. Bulk Import remains blocked on identity-schema review and approval.


## 47. Attendance and substitution handler availability regression coverage (2026-11-23)

- Added table-driven tests for staff attendance daily/monthly reads, single/bulk marking and updates, plus substitution list/create/cancel handlers. Each action is exercised with both a nil handler receiver and an unavailable service dependency.
- The handlers fail closed with HTTP 503 rather than panicking when their service is unavailable; normal authentication, live authorization and business validation remain in force when dependencies are present.
- Backend CI [45134503749](https://github.com/TheSumitNain/school-management-system/actions/runs/45134503749) **PASS** on `afcd762e`: ephemeral PostgreSQL initialization and all migrations, Go tests including PostgreSQL integration, and `go vet`.
- No migration or persistent database change was made. Staff document upload remains deferred pending external storage selection/purchase and explicit approval. Bulk Import remains blocked on identity-schema review and approval.


## 48. Staff leave handler response reliability (2026-11-23)

- Consolidated the leave/holiday handler's authentication-context failure path so it emits one HTTP 401 JSON response rather than writing a second response after the shared actor helper has already handled the failure.
- Strengthened nil-receiver and nil-service regression tests to parse the response as JSON and verify the expected unavailable message for all nine leave/holiday actions.
- Backend CI [45135360996](https://github.com/TheSumitNain/school-management-system/actions/runs/45135360996) **PASS** on `5f3935bb`: ephemeral PostgreSQL initialization and all migrations, Go tests including PostgreSQL integration, and `go vet`.
- No schema migration or persistent database change.


## 49. Staff route permissions and School Admin substitution access (2026-11-23)

- Extended the live School Admin permission allowlist to include the two dedicated staff substitution permissions already provisioned by the substitution migration. Without this runtime allowlist entry, database grants were filtered out and a School Admin could be incorrectly denied substitution access.
- Added a PostgreSQL integration assertion that a School Admin's effective permission set includes `staff.substitutions.read` and `staff.substitutions.manage`.
- Added route-level negative/positive permission tests for leave types and requests, holiday calendar, substitution list/create/cancel, staff qualifications, certifications, experience records, reports and staff audit history. Denied requests must stop at HTTP 403; requests with the dedicated permission reach the handler.
- Backend CI [45137329371](https://github.com/TheSumitNain/school-management-system/actions/runs/45137329371) **PASS** on `20814d1f`: ephemeral PostgreSQL migrations, all Go tests including PostgreSQL integration, and `go vet`.
- Hardened the RBAC HTTP integration test's generated fixture suffix to remain collision-resistant while keeping every generated username within the database's 32-character limit.
- No persistent database change or migration was made.
