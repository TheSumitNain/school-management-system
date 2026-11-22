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

