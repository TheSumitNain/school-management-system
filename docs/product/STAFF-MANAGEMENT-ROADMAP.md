## 2026-11-26 — Staff Leave/Substitution pagination and complete report export verified

- Staff Leave Requests and Substitution History now use bounded server-side pages (default 50, maximum 100), exact filtered totals, stable ordering, and Previous/Next navigation. Query keys include pagination state and filters reset to page one.
- Staff Leave report CSV export fetches every matching page in bounded batches and fails visibly if pagination metadata is invalid or the server stops returning rows before the reported total. The export continues to use the privacy allowlist and does not expose free-text leave reasons.
- Empty leave/substitution API responses preserve arrays and include pagination metadata. The earlier 1,000-row SQL caps were also removed from Leave Requests, Substitution History and the school-scoped Holiday Calendar.
- Added backend handler coverage, frontend pagination/export acceptance, and fixture/response-contract checks. Latest Frontend CI [45737923385](https://github.com/TheSumitNain/school-management-system/actions/runs/45737923385) **PASS** on `b164d1b9ffe838e25df9034b39b27340847b522e`, including TypeScript, production build and **132 Playwright tests**. Latest Backend CI [45737923468](https://github.com/TheSumitNain/school-management-system/actions/runs/45737923468) **PASS** on the same commit, including migrations, Go tests, PostgreSQL integration and `go vet`.
- This closes the previously open Leave/Substitution bounded-pagination and complete Leave CSV-export items for the implemented workflows. It does not claim manual sign-off on a live authenticated school account or approval-gated product decisions.
- No schema migration or persistent school data change in this batch. Changes remain on `work`; `main` remains untouched.

---



## 2026-11-25 — Staff Directory aggregate summary and lazy full-list loading

- The overview now loads aggregate counts from `GET /school/staff/summary`; it no longer downloads all staff records. The Dashboard staff metric uses the same summary endpoint instead of the unbounded list API.
- The Directory uses bounded PostgreSQL pagination, search and server-side filters without requesting the legacy full staff list. Full-list retrieval is lazy-loaded only by Reports, Leave, Substitutions and ID Cards where existing workflows still need full row-level data.
- Staff mutations invalidate the aggregate summary query. Backend authorization and integration tests cover summary count consistency, School Admin/live `users.read` access and school scoping; Playwright verifies Dashboard, overview and Directory avoid unbounded list requests.
- Backend CI [45644907855](https://github.com/TheSumitNain/school-management-system/actions/runs/45644907855) **PASS** and Frontend CI [45644907743](https://github.com/TheSumitNain/school-management-system/actions/runs/45644907743) **PASS** on `5c3584666626115c76983cf933e52c9936edfb34`; all **130 Playwright tests** passed.
- No schema migration or persistent database change. This improves the large-school directory path without changing the legacy report/ID-card data contracts.

---

## 2026-11-25 — Staff Directory server-side pagination

- The directory now requests bounded PostgreSQL pages (default 50, maximum 100) instead of downloading every staff row for directory search and filtering. Search is case-insensitive across username, employee name, employee code, designation and mobile; employment, account and profile-completeness filters are applied server-side. Stable ordering uses normalized employee name/username plus ID.
- The UI provides Previous/Next navigation, accurate total counts, resets pagination when filters change, and exports all matching rows by fetching every filtered page. Existing unfiltered staff data remains available to overview/report/ID-card consumers.
- Added service/handler/PostgreSQL integration tests and a browser test for server-side filtering, page totals and navigation. Backend CI [45641156564](https://github.com/TheSumitNain/school-management-system/actions/runs/45641156564) **PASS** and Frontend CI [45641156148](https://github.com/TheSumitNain/school-management-system/actions/runs/45641156148) **PASS** on `d896f8b28b2de81c94ff4d243f91c045093b6a00`; the frontend suite passed **130 Playwright tests**.
- No schema migration or persistent database change. This closes the Staff Directory server-pagination requirement in Phase 1; profile-only onboarding and Bulk Import remain separately blocked on the unapproved identity-schema migration.

---

## 2026-11-25 — Staff attendance audit-history refresh verified

- Attendance mark, bulk-mark and update mutations invalidate staff audit-history queries as well as attendance queries. The History & Activity query's 30-second freshness window does not prevent explicit invalidation from refreshing the history.
- Existing Playwright coverage verifies the end-to-end path: open a staff member's audit history, perform individual and bulk attendance writes, reopen the profile, and confirm another audit-history read occurs.
- Frontend CI [45623944157](https://github.com/TheSumitNain/school-management-system/actions/runs/45623944157) **PASS** and Backend CI [45623944253](https://github.com/TheSumitNain/school-management-system/actions/runs/45623944253) **PASS** on `248287a36ea205aa4ef2bb1ed7205984b9e632e3`; full browser acceptance and backend migration/integration/vet checks passed.
- No schema migration, persistent database or application data changed. Manual authenticated-school sign-off and existing identity-schema, leave-policy and payroll approval gates remain separate; private document/photo storage remains deferred.

## 2026-11-25 — Staff ID card and audit-history privacy review

- ID card eligibility is restricted to active, complete staff profiles with employee code, full name and designation. Disabled login does not incorrectly exclude an active employee. The printable card allowlist excludes mobile/email, username, login status, passwords and permissions; optional validity date rejects past dates.
- Staff History & Activity requires `school.audit.read` in both UI and backend. Unauthorized viewers do not trigger an audit fetch; repository authorization rechecks live permission and same-school actor/target membership. Events are scoped by authenticated school and staff principal, metadata is excluded from API JSON, results are capped at 100, newest-first and paginated.
- Existing Playwright and PostgreSQL tests cover permission denial, event rendering, employee switching, pagination, profile reopen refresh, cross-school denial and metadata privacy.
- No new production defect found. Backend [45602855583](https://github.com/TheSumitNain/school-management-system/actions/runs/45602855583) and Frontend [45602855658](https://github.com/TheSumitNain/school-management-system/actions/runs/45602855658) **PASS** on `bb7e0af18c3eb2af3e0bb4129c1408f0a41f12f5`; this checkpoint changes documentation only.
- No production behavior, schema, persistent database or application data changed. Existing identity-schema, leave-policy and payroll approval gates remain unchanged; private document/photo storage remains deferred and excluded from completion estimates.

## 2026-11-25 — Staff report CSV formula-injection regression verification

- Added browser regression coverage for tab-prefixed user-controlled report values in CSV exports, ensuring spreadsheet formula-like input is escaped as text.
- Backend CI [45602855583](https://github.com/TheSumitNain/school-management-system/actions/runs/45602855583) **PASS** and Frontend CI [45602855658](https://github.com/TheSumitNain/school-management-system/actions/runs/45602855658) **PASS** on `bb7e0af18c3eb2af3e0bb4129c1408f0a41f12f5`; complete ephemeral PostgreSQL migration/integration suite, Go tests, `go vet`, frontend production build and Playwright suite all passed.
- Report audit reconfirmed tenant-scoped queries, bounded filters/pagination, active-only qualification/certification reporting, intentional retention of former employees in experience history, and privacy exclusions for credential references, verifier notes and free-text descriptions.
- Verification/test-only checkpoint; no production behavior, schema, persistent database or application data changed. Existing identity-schema, leave-policy and payroll approval gates remain unchanged; private document/photo storage remains deferred and excluded from completion estimates.


## 2026-11-25 — Staff Leave decision/cancellation concurrency regression

- Added PostgreSQL coverage for concurrent approval and cancellation of the same pending leave request. The test requires one winner, one not-found loser, final status matching the winner, and exactly one corresponding audit event.
- Backend CI [45600058755](https://github.com/TheSumitNain/school-management-system/actions/runs/45600058755) **PASS** on `decc195c592003394d0b01e72f9498c9162b3c0e`: full ephemeral migration chain, complete Go/PostgreSQL integration suite, and `go vet`.
- Leave and holiday workflow audit found no new production defect. Leave entitlement policy (quotas, accrual, carry-forward/balances) remains an explicit school-approval gate; `requires_approval=false` remains pending by design until automatic approval behavior is explicitly authorized. No balances or payroll effects are claimed.
- Test-only change; no production code, schema, persistent database or application data changed.


## 2026-11-25 — Staff Attendance bulk API and transaction verification

- Backend CI [45598938508](https://github.com/TheSumitNain/school-management-system/actions/runs/45598938508) **PASS**: bulk attendance HTTP handler contract tests cover authenticated school context, payload forwarding, invalid date rejection, and missing authentication.
- Backend CI [45599478011](https://github.com/TheSumitNain/school-management-system/actions/runs/45599478011) **PASS** on `41851e988b17bd90a32c9cf623913aa61556ae28`: full migration chain, Go unit/integration suite including PostgreSQL, and `go vet`.
- Bulk attendance service coverage now checks invalid actor/school, invalid status/remarks, empty/duplicate/non-positive IDs, and the 1,000-target batch boundary.
- PostgreSQL integration coverage verifies concurrent bulk marking serializes through the unique school/staff/date constraint: one complete transaction wins, one conflicts, and only the winning records/audit events persist.
- Automated backend verification is green. Existing frontend CI remains applicable because no frontend source or browser test files changed. Manual authenticated-school browser sign-off, approved leave-entitlement policy, identity-schema approval, and persistent-database migration remain separate gates. No production behavior/schema/data changed.

## 2026-11-25 — Latest Staff Management verification on b8e46ac

- Backend CI [45593176075](https://github.com/TheSumitNain/school-management-system/actions/runs/45593176075) **PASS**: complete Go tests, PostgreSQL integration suite and `go vet`.
- Frontend CI [45593176068](https://github.com/TheSumitNain/school-management-system/actions/runs/45593176068) **PASS**: TypeScript, Vite production build and full Playwright browser acceptance suite.
- Rechecked staff profile/query lifecycle: employee-specific qualification, certification, experience and audit queries are keyed by principal ID and removed when the profile closes; authentication/session changes clear the shared React Query cache, preventing previous-school records from remaining available after sign-out/account changes.
- Closed the older substitution test-gap note below: current `staff_substitution_v60_workflow_integration_test.go` contains forced audit-failure rollback checks for both assignment and cancellation, and `staff_substitution_v61_concurrency_integration_test.go` verifies concurrent substitute-slot serialization. These are included in the passing backend suite above.
- No new production defect found in this review. No document upload work was started; private storage and staff-scoped secure download remain approval-gated. This verification does not claim manual authenticated-school browser sign-off or persistent-database migration.

## 2026-11-25 — Staff Management consolidated automated-acceptance audit

- Re-audited the Staff Management browser suite across `staff-management.spec.ts`, `staff-reports.spec.ts`, `staff-monthly-report.spec.ts` and `staff-responsive.spec.ts`. Coverage includes directory/profile onboarding and account controls, qualification/experience records and reports, academic assignments, history pagination, attendance, leave/holidays, substitutions, workload, CSV privacy, ID cards, permission-limited views and responsive layouts.
- Frontend CI [45546968103](https://github.com/TheSumitNain/school-management-system/actions/runs/45546968103) **PASS** on `129a6e1e3383a783529940e5e99892d992169737` (TypeScript, Vite production build and full Playwright suite). A GitHub compare against current `work` HEAD confirms the 13 commits since that frontend checkpoint changed only backend report tests and documentation; no frontend source, E2E spec, package manifest or lockfile changed. Therefore that successful browser run remains applicable to the current frontend tree.
- Latest Backend CI [45550085168](https://github.com/TheSumitNain/school-management-system/actions/runs/45550085168) **PASS** on `f9291fe9e721188907d136f49cb1363a35b93922`, including ephemeral PostgreSQL migrations/integration tests and `go vet`.
- Automated acceptance is verified for the current code tree; this is not a claim of manual authenticated-school browser sign-off. Remaining explicit gates: school-approved leave quota/accrual/carry-forward policy; identity-schema approval for profile-only onboarding and Bulk Import; external private-storage selection/approval for document/photo upload; separate approval for payroll and sensitive personal fields. These are not counted as defects and were not implemented. Persistent databases were not migrated.

## 2026-11-25 — Staff report API filter-forwarding and qualification pagination verification

- Backend CI [45550085168](https://github.com/TheSumitNain/school-management-system/actions/runs/45550085168) **PASS** on `f9291fe9e721188907d136f49cb1363a35b93922`: all migrations applied to ephemeral PostgreSQL, the complete Go test suite including PostgreSQL integration passed, and `go vet` passed.
- Added qualification-report PostgreSQL assertions for both first and second page boundaries, total-count consistency and stable record ordering.
- Added handler tests proving valid qualification/certification and experience report filters plus limit/offset values reach the service unchanged. Existing malformed query tests continue to require HTTP 400.
- Regression coverage only; no production behavior, schema, persistent database or application data changed. Approval-gated work and document/photo-storage deferral remain unchanged.

## 2026-11-25 — Staff report pagination verification update

Backend CI [45547729975](https://github.com/TheSumitNain/school-management-system/actions/runs/45547729975) **PASS** on `64ee5559d50eb7ac58582ba3fd0c62fcf684506a`, including the full Go/PostgreSQL integration suite and `go vet`. New coverage verifies multi-page Experience History repository results/counts/filtering and HTTP 400 handling for malformed pagination/filter query parameters. Combined with Frontend CI [45546968103](https://github.com/TheSumitNain/school-management-system/actions/runs/45546968103), report pagination is verified across database, handler and browser layers. Regression tests only; no production schema or data changes. Approval-gated work and document/photo-storage deferral remain unchanged.

## 2026-11-25 verification — substitution concurrency and audit rollback

- The V60 database-guard fixture now uses a username within the enforced 3–32 lowercase ASCII character limit. This is test-fixture-only; no production behavior or schema changed.
- Backend CI [45546585508](https://github.com/TheSumitNain/school-management-system/actions/runs/45546585508) **PASS** on `c516b63716bbba555d6abdbb78328476b149791f`: ephemeral PostgreSQL migrations, full Go/PostgreSQL integration suite and `go vet` all passed.
- The successful run verifies the new V61 repository-level concurrent assignment race and transactional audit-failure rollback regressions alongside the V60 database guards. No persistent database or application data changed; existing leave-policy/identity-schema approval gates and document/photo storage deferral remain unchanged.

## 2026-11-25 — Staff Leave and Substitution consistency audit

- Re-verified Staff Leave repository workflows and PostgreSQL integration coverage: school-scoped authorization, active staff/leave-type eligibility, concurrent overlap serialization per school/staff, pending/approved overlap rejection, row-locked decisions, self-approval prevention, rejection-reason validation, and transactional audit rollback for decisions/cancellations.
- Re-verified School Holiday writes use school-scoped transactions and audit events; integration coverage checks duplicate/conflicting dates and rollback on audit failure. Holiday overlays remain informational for attendance; no attendance status is automatically mutated.
- Re-verified Staff Substitution validation and database guards: same-school timetable/period scope, matching original teacher and weekday, active original/substitute staff, enabled substitute login, matching active class/subject assignment, optional approved full-day leave reference for the original teacher/date, duplicate active cover constraints, teacher-period conflicts, and timetable write serialization. Current/future active covers block relevant timetable, assignment and staff/login lifecycle changes; cancellation retains immutable history and releases the recurring slot.
- Substitution repository create/cancel writes and their audit events share a transaction. Follow-up test gap retained: the inspected substitution integration suite does not contain a dedicated forced-audit-failure rollback scenario or a true concurrent repository-level assignment race test. Existing unique constraints, database triggers, V61 lock-order coverage and database-guard integration tests are present; this follow-up is not being represented as a known production defect.
- No application code, schema migration, production data, leave-policy entitlement rules, or attendance auto-marking behavior changed. No approval-gated policy was introduced.

## 2026-11-25 — School-wide staff report browser acceptance audit

- Re-inspected the complete Staff Management Playwright spec and confirmed dedicated scenarios exist for Qualifications & Certifications and Staff Experience History. The scenarios exercise the respective aggregate report endpoints, type/status filters, visible result rows and CSV export privacy allowlists.
- The qualification CSV regression asserts credential references and verification notes are absent; the experience CSV regression asserts free-text descriptions are absent. The complete frontend Playwright suite passed in CI [45543354522](https://github.com/TheSumitNain/school-management-system/actions/runs/45543354522) on `4eb297516f652de07d4900455fcab4a119b0da57`. Backend CI [45543354503](https://github.com/TheSumitNain/school-management-system/actions/runs/45543354503) also passed on the same code checkpoint.
- This closes the previously documented gap about not locating dedicated report-screen Playwright scenarios. It does not claim separate browser coverage for every pagination interaction. No schema migration, persistent database or application data changed.

## 2026-11-25 — Staff profile Unicode validation acceptance

- Staff profile form limits for full name (160), designation (100) and email (254) now enforce the same Unicode-code-point limits as backend validation. Employee code (32) and mobile number constraints were already aligned.
- Playwright regression coverage includes supplementary Unicode input. Frontend CI [45543354522](https://github.com/TheSumitNain/school-management-system/actions/runs/45543354522) **PASS** and Backend CI [45543354503](https://github.com/TheSumitNain/school-management-system/actions/runs/45543354503) **PASS** on `4eb297516f652de07d4900455fcab4a119b0da57`.
- No schema migration, persistent database or application data changed. This acceptance does not remove the existing approval gates for identity/profile schema, Bulk Import, leave policy, sensitive personal fields or external private document/photo storage.

# Staff Management — Product and Implementation Roadmap

**Status:** The implemented School Admin Staff Management workflows are automated-test and browser-verified. Phase 1 remains partially met because profile-only onboarding is blocked on identity-schema approval; Phase 4 remains partial pending school leave-policy approval; Phase 6 remains partial because Bulk Import shares the identity-schema gate. Payroll and document storage remain separately deferred.  
**Scope:** Phase 1 School Admin web workspace, with Teacher self-service as a separate future experience.  
**Last updated:** 2026-11-26

**Latest verification checkpoints (2026-11-26):** Backend CI [45744694365](https://github.com/TheSumitNain/school-management-system/actions/runs/45744694365) **PASS** and Frontend CI [45744694369](https://github.com/TheSumitNain/school-management-system/actions/runs/45744694369) **PASS** on `185f3e3a2e97f8801a4f7cfc313c293cd235743c`. Backend verification includes the full migration chain, Go/PostgreSQL integration suite and `go vet`; the attendance tests cover single-write audit rollback, update rollback preserving original values, daily/monthly read permission enforcement, cross-school denial and inactive-employee history retention. Frontend verification includes TypeScript, Vite production build and **134 Playwright tests passed**. New browser regressions prove users with only `staff.attendance.read` or only `staff.substitutions.read` can open their permitted workspace without triggering staff-directory/profile API reads. Leave/Substitution pagination has direct PostgreSQL coverage for multi-page ordering, filtered totals, date overlap and cross-school denial. Leave Requests and Substitution History use bounded server-side pagination (default 50, maximum 100), exact filtered totals and page navigation; the Leave report CSV fetches every matching page and retains its privacy allowlist. Automated acceptance is verified; manual authenticated-school browser sign-off remains distinct. Phase 4 and Phase 6 remain partial for their separately documented policy/schema approval gates.

This document defines what Staff Management will build, in what order, the screens/tabs and forms users will see, each field's required/optional status, the key workflows, security boundaries and acceptance criteria. It is a roadmap, not evidence that a feature is implemented. Check [Project Status](../project/STATUS.md) before each phase and verify existing code before changing it.

## 1. Product decisions and goals

**Schema implementation gate:** The current principal-keyed staff profile cannot support the approved profile-without-login workflow. The architecture proposal and dependent-table migration sequence are documented in [Staff Identity and Profile Schema Design](../backend/STAFF-IDENTITY-SCHEMA-DESIGN.md). This is not an approved migration: profile-only onboarding and Bulk Import remain blocked until the explicit decisions in that design are reviewed and approved. No database has been changed.


Build one school-scoped **Staff Management** module for both teaching and non-teaching employees. The first implementation and acceptance focus on teacher workflows, while the underlying profile model must support non-teaching employees without a redesign.

The module must let a School Admin maintain employee records; manage personal, employment, qualification and document information; assign teachers to academic years, classes, sections and subjects; manage login accounts and access separately from employment; preserve lifecycle history; and progressively manage attendance, leave, workload, reports, ID cards and payroll.

The Teacher's daily-work experience (own timetable, assigned classes, attendance, homework, marks, notices and leave) is a **separate Teacher Dashboard**, not another tab inside School Admin Staff Management.

### Explicit decisions

1. Teaching and non-teaching employees use the same staff foundation and directory, distinguished by Staff Type.
2. A staff profile does not require a login account. Login identity and effective access are managed separately. A teacher who needs system access receives an account through the guided workflow.
3. Staff accounts are school-specific. The same person working at two schools has separate school accounts and records. Staff usernames remain globally unique and case-insensitive under the existing RBAC contract.
4. Employment status and login-account status are separate. An inactive employee cannot sign in. Reactivating employment must not silently re-enable a disabled account.
5. Deactivate/relieve/archive instead of hard-deleting records that have history or dependencies. Preserve attendance, leave, assignments, timetable, payroll and audit history.
6. Reuse Academic Structure's academic years, classes, sections and subject catalogue. Do not create duplicate master-data screens or models.
7. No paid SMS/OTP dependency. Parent OTP and parent-facing workflows remain deferred under the project-wide Phase 1 boundary.
8. Staff document upload/download and profile photograph upload are explicitly deferred until the project is ready to go live and an external private-storage provider and access model are selected and approved. Until then, the Documents profile tab is an explanatory disabled state; do not build upload APIs, local-disk storage, file metadata writes, expiry reminders or mock uploads.
9. DOB, gender, home addresses and emergency contacts require a separate field-level access, minimization, retention and legal/privacy review before collection or storage. Do not expose these fields through directory/list/report/ID-card APIs.
10. This roadmap does not authorize speculative database migrations. Confirm the existing schema/API and define each required additive migration before implementation.

## 2. Navigation and tab plan

### 2.1 School Admin Staff Management — 9 permission-gated workspaces

| # | Workspace | Phase | Purpose |
|---:|---|---|---|
| 1 | Overview | P0 | Staff summary, setup gaps and quick actions |
| 2 | Staff Directory | P0 | Search, filter, create and manage employee profiles |
| 3 | Academic Assignments | P0 | Class-in-charge and subject/class/section assignments |
| 4 | Attendance | P1 | Daily staff attendance and monthly register |
| 5 | Leave & Holidays | P1 | Leave requests, approvals and school holiday calendar; balances remain deferred |
| 6 | Substitutions | P1 | Dated teacher cover, timetable validation and cancellation history |
| 7 | Workload Report | P1 | School-wide scheduled teacher periods and workload summary |
| 8 | Reports | P1, expanded P2 | Employment, attendance, leave, assignments, qualifications, certifications and experience |
| 9 | ID Cards | P2 | Permission-gated selection, preview and print of approved staff identity fields |

**Account & Access is not a separate top-level tab.** It belongs to the selected employee's profile because it is specific to that employee. This avoids splitting one employee workflow across distant pages.

Tab visibility and actions are permission-aware. A hidden tab is not authorization: every API operation must enforce live permissions and school/resource scope on the server.

### 2.2 Staff profile — 9 tabs

Selecting a staff row opens a full profile page (preferred for deep profiles and direct links; a responsive drawer may be used for a compact quick view). The profile has **9 tabs**; Workload & Timetable is shown only when the viewer has `timetable.read`:

| # | Profile tab | Phase | Contents |
|---:|---|---|---|
| 1 | Overview | P0 | Identity summary, employment/account status, current assignments and recent activity |
| 2 | Personal Details | P0 / gated fields | Supported name and contact fields; sensitive personal fields stay unavailable until schema, access and privacy approval |
| 3 | Employment | P0 | Employee code, staff type, department, designation, employment type, dates and status |
| 4 | Qualifications & Experience | P0 | Education, specialization, certifications, experience and previous employment |
| 5 | Documents | Deferred | Explanatory disabled state only; upload/download waits until go-live storage approval |
| 6 | Academic Assignments | P0 | Class-in-charge and subject teaching assignments with history |
| 7 | Workload & Timetable | P1 | Current-year teacher-specific scheduled periods and timetable rows; unscheduled slots are not confirmed availability |
| 8 | Account & Access | P0 | Login lifecycle, roles, delegated permissions and password reset |
| 9 | History & Activity | P0 | Employment, assignment and relevant audit/activity events |

Tabs with no records must show an explanatory empty state and an action when the viewer has permission. Sensitive fields/documents must be field-authorized; directory access does not imply access to every profile field. DOB, gender, addresses and emergency contacts are deferred pending dedicated schema, field-level permission, retention and privacy approval. Do not collect them through placeholder/local-only forms.

## 3. Forms and field specifications

**Field notation:** **Required** means required for the described operation; **Optional** may be left blank. Conditional fields are required only when their condition applies. Required/Optional status must be visible beside the label. Backend validation is authoritative.

### 3.1 Form A — Add Staff (guided 4-step form, P0)

Open from Overview → Add Staff or Staff Directory → Add Staff. Use the shared accessible modal/wizard for the short creation flow; if it grows beyond a comfortable modal size, use a dedicated full-page wizard without changing the four-step sequence. Keep the directory in place when using a modal.

#### Step 1 — Staff Type & Identity

| Field | Required? | Rules / behavior |
|---|---|---|
| Staff Type (Teaching / Non-teaching) | Required | Default Teaching only from a teacher-specific action; otherwise make choice explicit |
| Full Name | Required | Trim whitespace; reasonable length and human-name validation |
| Employee Code | Required | Unique within the school; normalize before duplicate check |
| Mobile Number | Required | Valid Indian mobile format for current India-focused release |
| Email | Optional | Valid format; normalize case; apply uniqueness only if data contract requires it |
| Photograph | Deferred | Profile-photo upload is deferred with document storage; do not accept or persist a local/mock upload |

#### Step 2 — Employment Details

| Field | Required? | Rules / behavior |
|---|---|---|
| Designation | Required | Select from school designation master; controlled setup if catalogue is absent |
| Department | Optional initially | Select from school department master; avoid duplicate free text |
| Joining Date | Optional | Shared IST-aware DatePicker; store date-only value |
| Employment Type | Optional initially | Permanent, Contract, Part-time, Temporary, Internship, Visiting Faculty; keep machine values separate from labels |
| Employment Status | Required | Active / Inactive; default Active for new employee |
| Reporting Manager | Optional | Active staff in same school; prevent self-reference/cycles |
| Work Location / Branch | Optional | Only if multiple locations are configured |

#### Step 3 — Login & Access

Show this step for Teaching staff by default, and for any non-teaching employee when Create login account is selected.

**Current implementation gap:** the existing Add Staff wizard still requires a username and initial password for every employee because `staff_profiles` is currently coupled to `users`. The optional profile-only path described below is a target product behavior, not yet implemented. Do not generate hidden credentials or create placeholder accounts; profile-only onboarding (single or bulk) remains blocked pending the approved identity-schema redesign in [Staff Bulk Import Architecture Gate](../backend/STAFF-BULK-IMPORT-DESIGN.md).

| Field | Required? | Rules / behavior |
|---|---|---|
| Create Login Account | Required decision | Yes / No; No is allowed for employees who do not need access |
| Username | Conditional | Required when login is enabled; preserve global case-insensitive uniqueness |
| Initial Password | Conditional | Required when current account-provisioning contract requires admin-issued temporary password; validate current password policy and never reveal after save |
| Role | Conditional | Required when login is enabled; Teacher role by default for teaching staff, otherwise an authorized school role |
| Require Password Change at First Login | Conditional | Required decision when temporary password is issued; default enabled |
| Credential Sharing Option | Conditional | After account creation succeeds, offer an explicit **Share Credentials** action; never auto-send |
| Share Channel / Recipient | Conditional | Show only supported, configured channels; confirm recipient/contact before sharing |
| Additional Permission Overrides | Optional | Not part of basic add; configure later with reason and delegation checks |

**Credential sharing workflow (required in the login provisioning phase):**
1. Admin creates staff login credentials and submits the form.
2. After the backend confirms account creation, show **Share Credentials** and **Do This Later** actions.
3. Before sharing, show the destination/contact and delivery channel; require explicit admin confirmation.
4. Do not assume paid SMS/OTP or an external messaging provider. Initially support only channels already available in the project (for example, a secure copy/share handoff). Email or messaging integrations require separate provider approval.
5. Never store plaintext passwords for later retrieval, log credentials, or put passwords in URLs. If the password cannot safely be re-displayed after creation, provide a one-time credential handoff at creation time or use a secure reset/invitation flow.
6. Audit who initiated sharing, when, channel, masked destination metadata and outcome—never the password or secret.
7. Sharing is optional; account creation must succeed if the admin chooses **Do This Later**. Require password change at first login when a temporary password is issued.

Selecting Teacher role does not grant class/subject access. Assignment-scoped access begins only after explicit academic assignments exist. A profile-only employee can be created without login.

#### Step 4 — Review & Create

- Show values grouped by Identity, Employment and Login.
- Identify omitted optional information without blocking creation.
- Allow Back/Edit without losing values.
- Explain that creating a login provisions school-scoped access.
- Submit once; prevent duplicate clicks.
- On success, show a global success toast and open the new staff profile.
- On recoverable failure, preserve non-secret values and show field errors beneath corresponding fields. Do not retain/display password after security-sensitive failure.

### 3.2 Form B — Edit Staff Profile (P0)

Edit from the relevant profile tab. Use a focused form for the selected section, not one enormous form. Existing values must load correctly and unsaved drafts must not be overwritten by background refetch.

**Personal Details form**

| Field | Required? |
|---|---|
| Full Name | Required |
| Mobile Number | Required |
| Email | Optional |
| Date of Birth | Deferred |
| Gender | Deferred |
| Photograph | Deferred |
| Current Address | Deferred |
| Permanent Address | Deferred |
| Emergency Contact Name | Deferred |
| Emergency Contact Relationship | Deferred |
| Emergency Contact Mobile | Deferred |

**Employment form**

| Field | Required? |
|---|---|
| Employee Code | Required |
| Staff Type | Required |
| Designation | Required |
| Department | Optional |
| Joining Date | Optional |
| Employment Type | Optional |
| Reporting Manager | Optional |
| Work Location / Branch | Optional |
| Employment Status | Required |

Employee code changes must be audited and uniqueness-checked. Changing Staff Type or employment status must warn about incompatible active assignments/account effects and require explicit resolution where needed.

### 3.3 Form C — Add / Edit Qualification (P0)

Repeatable form; one staff member may have multiple qualification records.

| Field | Required? |
|---|---|
| Qualification Level (12th, Diploma, Bachelor's, Master's, PhD) | Required |
| Qualification / Degree Name | Required |
| Specialization / Subject | Optional |
| Institution / University | Optional |
| Board | Optional |
| Year of Completion | Optional |
| Grade / Percentage / CGPA | Optional |
| Certificate Document | Optional |
| Verification Status | Required, default Pending |
| Verification Notes | Optional |

Add a separate repeatable **Certification / Training** record with Certificate Name (Required), Issuing Organization (Optional), Issue Date (Optional), Expiry Date (Optional), Credential ID (Optional), Document (Optional), Verification Status (Required). Qualification mismatch should warn by default, not block assignment, unless a school policy explicitly requires it.

### 3.4 Documents tab — deferred until go-live storage approval

The Documents profile tab must remain a clear, disabled explanatory state during the current development phase. The user has explicitly deferred document upload/download until the project is ready to go live. Profile photograph upload is deferred for the same private-storage reason.

Do not implement a local-disk upload workaround, public URL, mock upload, document metadata write path, expiry reminder, or file API in this phase. Before reopening this scope, select and approve an external private-storage provider, authenticated download/preview authorization, file validation and size limits, malware/content scanning expectations, retention/deletion policy, audit events and backup/recovery responsibilities. The existing [Staff Qualifications & Documents — Backend Contract](../backend/STAFF-QUALIFICATIONS-DOCUMENTS.md) remains a security reference, not permission to implement storage now.

Qualifications, certifications and experience records remain separate structured staff records and are not blocked by this document-storage deferral.

### 3.5 Form E — Academic Assignment (P0)

Provide an Assignment Type selector: **Class In-charge** or **Subject Teacher**. Use existing Academic Structure data.

**Common fields**

| Field | Required? |
|---|---|
| Teacher | Required |
| Academic Year | Required |
| Assignment Type | Required |
| Effective From | Required |
| Effective To | Optional |
| Notes / Reason | Optional |

**Class In-charge fields**

| Field | Required? |
|---|---|
| Class | Required |
| Section | Required |
| Primary / Additional responsibility | Required |
| Effective dates | Start required; end optional |

**Subject Teacher fields**

| Field | Required? |
|---|---|
| Class | Required |
| Section | Required |
| Subject | Required |
| Periods per week / workload target | Optional in P0; required only when timetable workload policy is enabled |
| Effective dates | Start required; end optional |

Allow multiple assignment rows. Prevent duplicate active assignments, cross-school references, inactive year/class/section/subject selection and invalid class-section-subject combinations. Teacher must be active and Teaching type. Qualification mismatch is a warning unless configured policy makes it blocking.

Changing or ending an assignment must preview affected timetable entries and dependencies before confirmation. Never silently delete timetable history. Class-in-charge and subject teaching are distinct relationships and permissions.

### 3.6 Form F — Account & Access (P0)

Actions: create account for existing profile; enable/disable login; reset password; assign/remove authorized role; add/remove individual permission override with reason; review effective permissions (read-only preview).

| Field | Required? |
|---|---|
| Username (create account) | Required |
| Temporary Password (create/reset where admin-issued) | Required by current password-reset contract |
| Role | Required when assigning |
| Permission Override (permission + ALLOW/DENY) | Required when adding |
| Reason for override | Required |
| Confirmation of disable/reset consequence | Required for consequential action |

Show employment status separately from account status. Explain that disabling an account revokes sessions. Reactivating employment does not automatically enable login. Backend resolves permissions live; JWT role/permission snapshots are not authorization truth. Preserve default-deny, DENY-over-ALLOW, delegation ceilings and exact school/resource/assignment checks. Never allow School Admin to assign permissions not delegated to that school.

### 3.7 Form G — Employment Lifecycle Change (P0 foundation; full workflow P2)

Actions: department/designation change, transfer, employment status change, resignation/relieving and rehire/reactivation.

| Field | Required? |
|---|---|
| Change Type | Required |
| Effective Date | Required |
| New Department | Conditional |
| New Designation | Conditional |
| New Reporting Manager | Optional |
| Last Working Date | Conditional for relieving |
| Reason / Notes | Required for consequential changes |
| Dependency resolution / reassignment | Conditional when active assignments or timetable entries exist |

Show impact before confirmation: active assignments, upcoming timetable entries, pending leave and later payroll dependencies. Write change and audit event transactionally. Preserve prior values in history.

### 3.8 Form H — Staff Attendance (P1)

Daily register is primary; avoid an individual modal for every employee.

| Field | Required? | Rules |
|---|---|---|
| Attendance Date | Required | Shared IST-aware date-only picker |
| Department | Optional | Filter |
| Staff Type | Optional | Filter |
| Staff Search | Optional | Filter |
| Attendance Status per staff row | Required on save | Present, Absent, Late, Half-day, On Leave |
| Check-in Time | Optional | Policy-dependent |
| Check-out Time | Optional | Policy-dependent |
| Remarks | Optional | Per-row note |

Holiday/Off should be policy/calendar-derived where applicable. Prevent duplicate attendance for the same staff/date/session. Audit corrections; manual correction requires a reason and permission. Do not infer On Leave without an approved leave record unless an authorized override is explicitly supported.

### 3.9 Form I — Leave Application & Approval (P1)

**Leave type/policy setup**

| Field | Required? |
|---|---|
| Leave Type Name | Required |
| Paid / Unpaid | Required |
| Annual / Accrual Quota | Optional until policy is finalized |
| Carry Forward Rule | Optional until policy is finalized |
| Requires Approval | Required |
| Active Status | Required |

**Leave application**

| Field | Required? |
|---|---|
| Staff Member | Required |
| Leave Type | Required |
| Start Date | Required |
| End Date | Required |
| Full Day / Half Day | Required |
| Half-day Session | Conditional |
| Reason | Required |
| Supporting Document | Optional |

**Approval / rejection**

| Field | Required? |
|---|---|
| Decision (Approve / Reject) | Required |
| Decision Comment | Required for rejection; optional for approval |
| Approver | Derived from authenticated actor |
| Decision Date | System-generated |

Detect overlapping requests, invalid date ranges, inactive staff and conflicting approved leave. Display balances only after school policy is explicitly defined. Do not invent statutory entitlement rules.

### 3.10 Form J — Timetable Workload & Substitute (P1)

Workload view is read-only and derived from timetable and teacher assignments. Filters: Academic Year (Required), Teacher (Optional), Class (Optional), Subject (Optional). Show periods/week, daily distribution, free periods and conflicts.

**Substitute assignment**

| Field | Required? |
|---|---|
| Original Teacher | Required |
| Substitute Teacher | Required |
| Date | Required |
| Period(s) | Required |
| Class / Section | Required |
| Subject | Required |
| Reason | Required |
| Notes | Optional |

Validate substitute is active, eligible and not already booked for selected periods. Preserve original timetable and record temporary substitution separately with audit trail.

### 3.11 Form K — Bulk Import (P1)

Guided 4-step flow: (1) download template/upload CSV or XLSX, (2) map columns, (3) preview and validate all rows, (4) confirm import.

Minimum fields: Staff Type (Required), Full Name (Required), Employee Code (Required), Designation (Required), Mobile (Required). Department, Email, Joining Date, Employment Type and other profile fields are Optional. Username/login provisioning is separate; never import plaintext passwords.

Show row number, field, error and correction guidance for every invalid row. Detect duplicates within file and against existing school records. No writes before explicit confirmation. Specify batch/partial-success policy before implementation; show exact created/skipped/failed counts and downloadable error report.

### 3.12 Form L — Staff ID Card (P2)

Fields: Template (Required), Staff Selection (Required; one or many), Card Validity/Expiry (Optional), Print Layout (Required). The current print workspace supports Classic/Compact/Modern templates and only active, complete profiles. It prints employee name, designation, employee code, joining date, employment status and optional validity. QR is intentionally omitted until a secure verification endpoint is designed. Never print contact details, login data or permission information.

### 3.13 Form M — Payroll Setup & Processing (P2)

Payroll requires separate approval of salary/tax/compliance policy. Do not guess statutory deductions, tax treatment or salary rules.

**Salary structure**

| Field | Required? |
|---|---|
| Staff Member | Required |
| Effective From | Required |
| Pay Frequency | Required |
| Basic Salary | Required |
| Allowances | Optional / repeatable |
| Deductions | Optional / repeatable |
| Bank/payment details | Optional and highly restricted |
| Notes | Optional |

**Payroll run**

| Field | Required? |
|---|---|
| Pay Period | Required |
| Staff selection | Required |
| Attendance/leave inputs | Derived; corrections require review |
| Adjustments | Optional with reason |
| Approval | Required before finalization |
| Payment reference/date | Optional until payment is recorded |

Payroll requires stricter field-level access, audit history and restricted exports. Payslip access belongs to the future employee self-service experience. Payroll is not a P0/P1 dependency.

## 4. Overview dashboard

Overview (P0) shows only metrics supported by real APIs:
- Total staff; teaching and non-teaching staff.
- Active and inactive employment.
- Profiles missing required information.
- Teachers without current academic assignments.
- Teachers without a login account where login is expected.
- Quick actions: Add Staff, View Directory, Assign Teacher.

P1 adds today's staff attendance, staff on approved leave and pending leave approvals. P2 may add document-expiry and payroll summaries. No fake counts or client-side metrics from incomplete data. Every metric respects viewer permissions and school scope.

## 5. Directory columns and filters

Columns: employee/name/avatar; employee code; staff type; designation/department; mobile; employment status; login status (only with account-visibility permission); joining date; permitted row actions.

Filters: Staff Type, Department, Designation, Employment Status, Account Status and Academic Assignment Status. Search by name, employee code, designation, mobile and username only when account visibility permits it. Use server pagination and stable sorting. Empty, loading, error and no-results states are distinct.

## 6. Phased build sequence

Phases are dependency ordered, not a claim that every listed item is absent. Inspect and reuse current Staff Management CRUD, account lifecycle, roles/overrides and shared controls before adding features. Do not rewrite working behavior.

### Phase 0 — Audit, contract and design freeze

1. Inspect current Staff Management frontend, API handlers/services/repositories, migrations, permission catalogue and Playwright coverage.
2. Map existing fields/endpoints to this roadmap: Keep / Extend / Missing / Needs correction.
3. Confirm canonical employee/profile/account relationship; identify legacy staff accounts without complete profiles.
4. Define additive schema/API changes only where needed; document uniqueness, lifecycle, assignment and audit invariants.
5. Freeze tabs, field rules, permission matrix and API contracts.
6. Establish seed/test data and migration-safe verification plan.

**Exit gate:** approved gap analysis, data/API contract and test plan. No feature coding before this gate.

### Phase 1 — Staff Foundation (P0)

1. Staff Management shell with Overview and Staff Directory tabs.
2. Add Staff 4-step form and employee profile creation.
3. Profile page with 9 tabs (Workload & Timetable is permission-conditional); implement Personal Details, Employment, Qualifications, Documents and History foundations.
4. Preserve existing username/password/account creation, reset, role assignment and permission override behavior.
5. Separate employment status from account status; implement safe deactivate/reactivate/relieve semantics.
6. Add audit events and profile completeness indicators.
7. Support non-teaching staff in same foundation without Teacher access by default.

**Exit gate: PARTIALLY MET.** The implemented login-backed staff profile, account lifecycle, role/override, qualification/experience and audit workflows are browser-verified. Profile-only employee creation (and any dependent bulk onboarding) remains blocked until the identity/profile schema proposal is explicitly approved and migrated. Confirm teaching/non-teaching staff-type support against the approved schema before declaring this phase complete.

### Phase 2 — Teacher Academic Assignments (P0)

1. Build Academic Assignments top-level tab and profile tab.
2. Assign Class In-charge and Subject Teacher relationships using existing Academic Structure data.
3. Support multiple classes, sections and subjects, effective dates and assignment history.
4. Validate duplicates, inactive records, qualification warnings and timetable dependency impact.
5. Enforce teacher resource scope on backend routes and repositories; update teacher views to use exact assignments.

**Exit gate:** School Admin manages assignments; teacher API/UI exposes only assigned classes/sections/subjects; invalid assignments rejected; history and audit retained.

### Phase 3 — Staff Attendance (P1)

1. Daily staff register with filters and bulk status entry. **Implemented: employee search, status filters and atomic bulk marking of visible unmarked staff.**
2. Attendance correction/reason workflow. **Implemented: individual same-day/historical correction, remarks, explicit historical confirmation and transactional audit.**
3. Monthly register and reports. **Implemented: monthly report, separate attendance/approved-leave/active-holiday context, and permission-aware CSV export with formula-injection protection.**
4. Connect approved leave and school calendar rules. **Implemented as informational context only; approved leave and holidays never create or overwrite attendance marks.**

**Exit gate: MET for the currently approved Phase 3 scope (2026-11-24).** Duplicate/date/permission safeguards, audited corrections, accurate monthly reports, approved-leave/holiday informational context, permission-scoped exports and responsive browser acceptance are implemented and verified by the current Backend CI [45320435534](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435534) and Frontend CI [45320435562](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435562) (95 Playwright tests). This does not authorize leave-balance policy or automatic attendance changes.

### Phase 4 — Leave & Holiday Management (P1)

1. Leave type foundation. **Implemented:** school-scoped leave type schema, non-delegable permissions, create/list/update APIs, and frontend create/edit/deactivate/reactivate lifecycle with confirmation and history preservation.
2. Staff leave application. **Implemented:** staff/leave-type selection, inclusive dates, full/half-day request, reason, request listing and filters. Backend rejects invalid/inactive targets and overlapping pending/approved requests; per-staff transaction advisory locking serializes overlap checks. Form errors are isolated by workflow pane and recoverable values remain available.
3. Approval/rejection. **Implemented:** pending-only decisions, authenticated decision actor/time, required rejection comment, applicant self-approval denial, cancellation and transactional audit. PostgreSQL integration covers concurrency, authorization, rollback, rejection reason, audit persistence and cross-school denial. **Balance view pending:** quotas/accrual/carry-forward remain undefined until a school policy is approved; no balance is fabricated.
4. Holiday calendar. **Implemented:** school-scoped create/list/update/deactivate APIs; active-only default listing; manager-only inactive listing; explicit deactivate/reactivate confirmations; edit preserves inactive state. PostgreSQL integration covers normalized duplicates, conflicting updates, cross-school denial and audit rollback.
5. Attendance context and reporting. **Implemented:** approved leave and active holidays appear separately in daily/monthly attendance context when the viewer has the relevant read permissions. Monthly CSV export separates attendance, approved leave and holidays and protects against spreadsheet formula injection. No attendance rows are automatically created or overwritten.
6. Permission boundaries and browser acceptance. **Implemented:** the workspace requires a read permission; write/decision-only grants cannot expose leave list views. Browser coverage includes the leave workflows, holiday reactivation, form-error isolation, monthly CSV export and responsive layouts. A read-only holiday-calendar acceptance test verifies that `staff.holidays.read` without `staff.holidays.manage` exposes the active calendar but no mutation controls or inactive records (frontend and backend CI pass at `eaaea2f2f0f4a3665221a3fe0d1807f404eac6a7`).

**Exit gate: PARTIALLY MET (2026-11-24).** The end-to-end leave/holiday workflow review and permission-specific browser acceptance are complete, including read-only leave access without staff-directory permission; current Backend CI [45320435534](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435534) and Frontend CI [45320435562](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435562) pass. School-specific leave policy approval remains open. Leave balances, quotas, accrual and carry-forward must not be implemented until the school explicitly approves the policy. Keep attendance context informational; manual attendance conflicts require explicit human resolution.

### Phase 5 — Workload, Timetable & Substitution (P1)

1. **Initial workload summary implemented (2026-11-22):** the existing timetable read API accepts an optional `teacher_principal_id` filter applied in PostgreSQL. It retains live `timetable.read` authorization, requires a matching `staff_profiles` row in the authenticated school, and preserves the teacher self-only boundary. The Staff profile's permission-gated Workload & Timetable tab shows current-year scheduled counts and a weekly grid using actual timetable entries and the period catalogue. Unscheduled timetable slots are shown for planning context only; they do not establish teacher availability, working days or assignability. Backend CI (Go tests, PostgreSQL integration suite, `go vet`) and frontend CI (TypeScript, production build, Playwright) passed on `30270fe0927a6eb3f284d462ddd6dc2b4ca2959e`.
2. **PostgreSQL tenant/actor scope verified:** `TestAcademicFoundationAndStudentLifecyclePostgresIntegration` proves School Admin target filtering for two same-school teachers, Teacher self-only enforcement when requesting a peer, and no results for a foreign-school teacher principal ID. Backend CI run `45009905703` passed the full PostgreSQL integration suite and `go vet`.
3. If the school later approves working-day and availability rules, implement a true teacher availability/free-period view. The current unscheduled-slot grid is informational only and must not be used as a substitute for those rules.
4. **Timetable conflict feedback implemented (2026-11-22):** existing database unique-slot guards remain authoritative; repository and HTTP layers now distinguish teacher double-booking from section double-booking with stable HTTP 409 codes and actionable guidance. Repository and handler tests added; backend CI run `45010784607` passed the full Go/PostgreSQL integration suite and `go vet`. **Cross-section conflict visibility implemented (2026-11-23):** the School Admin timetable view loads all current-year school entries regardless of the selected display filter; Add Class previews the selected teacher's and section's weekday schedule and highlights occupied slots. Teacher self-only access remains enforced server-side, empty slots do not imply availability, and database constraints remain authoritative. Frontend CI [45107027353](https://github.com/TheSumitNain/school-management-system/actions/runs/45107027353) **PASS** (76 Playwright tests); Backend CI [45107027221](https://github.com/TheSumitNain/school-management-system/actions/runs/45107027221) **PASS** (Go/PostgreSQL integration and go vet) on checkpoint `82596d2b5a4e7a76345e5fe5bb65e34b3032253b`.
5. **Weekly workload reporting implemented and browser-verified (2026-11-22):** the profile view reports actual-entry-based weekday breakdowns (scheduled periods, distinct sections/subjects, first/last period), busiest day and weekly totals. The report explicitly excludes unscheduled-slot availability assumptions, approved leave and substitute coverage. Frontend CI [45012140676](https://github.com/TheSumitNain/school-management-system/actions/runs/45012140676) passed TypeScript, production build and the full Playwright browser suite; Backend CI [45012140670](https://github.com/TheSumitNain/school-management-system/actions/runs/45012140670) passed Go tests, ephemeral PostgreSQL integration and `go vet` on checkpoint `dd126cd939e1bdad8e817a9ccaf458db4b632b38`.
6. **Substitute workflow backend implemented and PostgreSQL-verified (2026-11-22):** additive migration V60 creates dated coverage records linked to an existing timetable entry; recurring teacher assignments are never rewritten for one-day cover. The database snapshots original teacher/substitute names, class, section, subject, period and room; prevents duplicate class coverage and substitute double-booking; validates weekday, active employment/login and canonical `auth_principals` status, matching active class/subject assignment and optional approved full-day leave reference; V61 locks the timetable-entry row then the shared teacher/weekday/period advisory key before validation to serialize concurrent substitution and timetable writes without lock-order inversion; protects active future cover from staff inactivation, substitute login disable and assignment revocation; V63 blocks recurring schedule changes only for current/future active covers while completed history remains immutable and no longer blocks later schedule edits; preserves audit history. API routes are documented in [Backend API](../backend/API.md). Backend CI [45016169600](https://github.com/TheSumitNain/school-management-system/actions/runs/45016169600) **PASS** (Go tests, ephemeral PostgreSQL integration and go vet) on checkpoint 8c68684ef2566bc8b5285eed1cbe2ac0352b6d20. Follow-up hardening V61–V63 was verified on code checkpoint `08c136d11a641e082db1f1be8b1d8ea82782058b`: Backend CI [45104388491](https://github.com/TheSumitNain/school-management-system/actions/runs/45104388491) **PASS** (full PostgreSQL integration and go vet), covering row-lock-first concurrency serialization, canonical principal status and completed-cover lifecycle.
7. **Substitute coverage UI implemented and browser-verified (2026-11-22):** Staff Management has a Substitutions workspace for selecting the recurring lesson, matching weekday/date, active substitute, optional approved full-day leave reference and reason; it lists dated coverage and requires explicit cancellation confirmation. Field validation is inline. Frontend CI [45015865422](https://github.com/TheSumitNain/school-management-system/actions/runs/45015865422) **PASS** (TypeScript, production build and full Playwright suite including assign/weekday-validation/cancel acceptance) on checkpoint 1cdbc59dfef59f8d4a8e98d114bc4b3f80de2da0. Follow-up Frontend CI [45104388370](https://github.com/TheSumitNain/school-management-system/actions/runs/45104388370) **PASS** (75 Playwright tests), including completed-cover history visibility and no cancellation action for past covers.
8. **School-wide workload report implemented and verified (2026-11-22):** `GET /school/timetable/workload-report` returns scheduled periods and daily section/subject counts, first/last period, employment and login status for the authenticated school. School Admin sees same-school staff; Teacher scope is self-only. Staff Management provides search, employment filtering, weekly totals, busiest day and daily details. It explicitly does not infer working days, availability or workload limits. Backend CI [45017634124](https://github.com/TheSumitNain/school-management-system/actions/runs/45017634124) **PASS** (PostgreSQL report fixture, School Admin scope, Teacher self-only scope and go vet) on checkpoint b858f1ad782ea254b8589fb5b474bdec9d90fd1d. Frontend CI [45017345426](https://github.com/TheSumitNain/school-management-system/actions/runs/45017345426) **PASS** (TypeScript, production build and Playwright report acceptance) on checkpoint a022ceb5be1e8c1f41bb2c70d84a35195b477d12.
9. **Persistent migration gate remains:** V60–V63 have been applied only to ephemeral CI PostgreSQL. No persistent local/user database was migrated. Do not apply it to a persistent database without explicit approval and the required verified backup/restore process.

**Exit gate: MET for the implemented timetable/workload/substitution scope (2026-11-24).** Database guards and PostgreSQL integration tests cover teacher/section double-booking, dated-cover concurrency, recurring timetable history, school/actor scope and audit. Browser acceptance covers assignment, validation, cancellation, completed-cover history, read-only access and responsive workload/substitution views. Current Backend CI [45320435534](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435534) and Frontend CI [45320435562](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435562) pass. Teacher working-day/free-period availability and workload limits remain separate school-policy decisions; workload reports describe actual scheduled entries only.

### Phase 6 — Reports, Import/Export & ID Cards (P1/P2)

1. Staff directory/employment readiness summary, monthly attendance and leave-request summaries, current-academic-year assignment summary, Qualifications & Certifications report, and Experience History report are implemented. Aggregate reports use bounded school-scoped queries, live read permissions, pagination and allowlisted CSV fields. Qualifications and Experience reports do not fetch per employee (no N+1 requests). Qualifications and Experience History report backend checks and frontend browser acceptance are verified; Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) passed TypeScript, production build and the Playwright suite (83 tests).
2. Printable Staff ID Cards workspace is implemented in the Staff Management UI. It is permission-gated and includes only active records with `profile_complete === true` plus required employee identity fields. It supports multi-select, Classic/Compact/Modern preview, optional validity date and print-only A4 layout. Only employee name, employee code, designation, joining date, employment status and optional validity are printed. Contact details, login/permission information and QR codes are excluded. ID Cards Playwright coverage is present; the latest Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) passed the production build and Playwright suite after the preview locator and eligibility fixes.
3. Bulk Staff Import is intentionally blocked on the existing identity schema. `staff_profiles.principal_id` is a required foreign key to `users`, while the approved product contract requires employee profiles to exist independently from optional login accounts. Do not create usernames/passwords as an import side effect or implement a profile-only import against the current schema. See [Staff Bulk Import Architecture Gate](../backend/STAFF-BULK-IMPORT-DESIGN.md). The decoupled identity model, compatibility/backfill plan and migration must be reviewed and explicitly approved before import implementation.
4. Permission-controlled exports remain implemented for the existing reports and directory workspace. Exports omit restricted remarks/reasons, exclude revoked assignments, omit unauthorized assignment columns, and neutralize spreadsheet formula injection.
5. Staff document upload/download, expiry reminders and verification remain deferred until the user purchases/selects an external storage service and approves its integration. Until then, do not implement upload UI, local-disk storage or file APIs.

**Exit gate: PARTIALLY MET for Phase 6 (2026-11-24).** Reports, permission-controlled exports and ID-card print workflows are implemented and browser-verified; the latest consolidated Frontend CI [45323508729](https://github.com/TheSumitNain/school-management-system/actions/runs/45323508729) passes 98 Playwright tests, and Backend CI [45323508810](https://github.com/TheSumitNain/school-management-system/actions/runs/45323508810) passes Go/PostgreSQL integration and `go vet`. Bulk Staff Import is not implemented and remains blocked until the identity-schema/product proposal is explicitly approved; its preview/confirmation/atomicity criteria apply only after that approval. Document storage remains separately deferred.

### Phase 7 — Payroll (P2, separate approval gate)

1. Approve payroll policy, salary components, deductions, statutory handling and access matrix.
2. Define salary structures and effective-dated changes.
3. Build payroll period processing, review, approval and finalization.
4. Generate payslips and reports with restricted access.

**Exit gate:** policy sign-off, reproducible calculations, transactional finalization, audit trail and restricted field-level access. Do not begin until policy is approved.

### Phase 8 — Teacher Self-Service Dashboard (separate future workstream)

A separate Teacher Dashboard, not a Staff Management tab:
- My timetable and substitutions.
- My assigned classes/subjects and permitted class rosters.
- Attendance workflows permitted by role.
- Homework, marks and syllabus workflows when those modules are approved and available.
- Notices and school communication.
- My leave requests and balances.
- My profile and approved documents.
- Payslips only after Payroll exists.

Reuse the same identity, profile and assignment foundations. Parent OTP, Parent dashboard and Driver experiences remain deferred and are not part of this roadmap.

## 7. Roles, permissions and data boundaries

| Actor | Intended access |
|---|---|
| Platform Admin | Platform operations only; no implicit access to school staff records |
| School Admin | Staff administration within exactly one school, limited by current delegated permissions |
| Teacher | Own permitted profile/account functions and explicitly assigned class/subject workflows |
| Other school staff | Only explicitly assigned permissions and resource scope |
| Parent | Not in scope; Phase 2 |

Mandatory rules:
- Default deny; authorize every API request using live permissions.
- Derive school ownership from authenticated context, never trust client-supplied school IDs.
- Teacher access is restricted to exact active class-in-charge and subject assignments, academic year, resource and allowed fields.
- Roster permission does not imply full student profiles, guardian contacts, medical information or financial data.
- DENY overrides ALLOW; custom roles and overrides cannot exceed active Platform Admin delegation.
- Account disable/revocation takes effect immediately on protected requests and revokes sessions as specified by auth contract.
- Audit profile/employment/assignment/account/role changes, attendance corrections, leave decisions, imports, exports and payroll actions. Never log passwords, tokens or document contents.
- Enforce school-scoped employee-code uniqueness, global case-insensitive username uniqueness and assignment uniqueness in backend/database.
- Keep personal, emergency-contact, identity-document and payroll data behind explicit field-level permissions.

## 8. Shared UX and quality contract

Every screen, tab, form and asynchronous action must follow [Product UX and Usability Standards](UX-STANDARDS.md) and [Frontend Documentation](../frontend/README.md).

Non-negotiable acceptance:
- Visible labels and Required/Optional indicators; helper text for non-obvious formats.
- Validate all applicable fields on submit; show each error directly below its field and associate it with the input. Map structured server errors to fields.
- Preserve non-secret values after recoverable failures; do not overwrite unsaved drafts during background refetch.
- Use shared DatePicker for editable dates and shared Select for app-owned select menus. Use IST-aware date-only values in Asia/Kolkata.
- Use shared modal/dialog patterns for focused forms and destructive/security-sensitive confirmations; no page-specific modal implementations.
- Global success toast plus actionable inline validation/error feedback. Prevent duplicate submissions.
- Explicit impact preview and confirmation before deactivation, assignment removal, employment termination or other consequential operations.
- Distinct loading, skeleton, empty, no-results, permission-denied, network-error, retry, success and pending states.
- Responsive mobile/tablet/desktop and landscape layouts; no clipped controls or accidental horizontal scrolling.
- Light, Dark and System themes; shared design tokens and normalized controls.
- Keyboard accessibility, visible focus, semantic labels, screen-reader associations, contrast and reduced-motion support.
- Real API behavior only; no mock success or frontend-only authorization.

## 9. Verification and definition of done

Each phase must be verified against implementation, not inferred from a route or table existing.

**Frontend**
- npm run typecheck
- npm run build
- npm run test:e2e
- Browser coverage for create/edit/disable/assignment/approval/import actions, field errors, toast, cancel/no-write, retry and duplicate-click behavior.
- Responsive widths: narrow mobile, mobile landscape, tablet and desktop.
- Supported themes and keyboard-only operation.

**Backend**
- go test -count=1 -p 1 ./...
- go vet ./...
- PostgreSQL integration tests on a dedicated test database.
- Positive and negative tests for tenant isolation, live permission checks, assignment scope, field-level access, uniqueness, lifecycle, transaction rollback and audit events.

**Database and safety**
- Additive, reviewed migrations only.
- Verify migration up/down behavior where safe; document irreversible history migrations.
- Never apply migrations to persistent local databases without explicit user approval.
- Before approved persistent migration, identify and restore-test a backup. Never delete a backup without explicit approval.
- No real school/student/staff personal data or secrets in Git or test fixtures.

Update [Project Status](../project/STATUS.md) after each verified milestone with exact code checkpoint, CI evidence, local migration state (if applicable), and remaining acceptance work. Keep planned, implemented, automated-tested and manually accepted states distinct.

## 10. Out of scope / do not build in these phases

- Parent OTP, OTP/SMS delivery or Parent login/dashboard.
- Driver app or driver workflows.
- Native mobile apps during Phase 1.
- Fees/payment gateway, live bus GPS or advanced AI analytics.
- Unapproved statutory payroll/tax assumptions.
- Biometric/RFID integration without separate approval and hardware/privacy design.
- Automatic qualification-based blocking without explicit school policy.
- Hard deletion of staff with employment, assignment, attendance, leave, payroll or audit history.
- Duplicate Academic Structure, Subject Management or Timetable modules.

## 11. Research basis

The roadmap was informed by public product pages, manuals and walkthroughs for multiple school ERP/HRMS products, including CoreAxis, School Orbit, MyClassCampus, OpenEduCat, Fedena, Teachmint, Nipun Schools and MyClassboard. Common patterns included employee directories/profiles, qualifications/documents, login/role management, teacher assignments, attendance, leave, workload/timetable, reports, ID cards and payroll. Public materials vary in age and completeness; this roadmap selects features according to this project's Phase 1 priorities, existing architecture and budget constraints rather than copying any one vendor.


## 2026-11-23 implementation checkpoint — directory safety and permissions

- Directory completeness uses a conservative rule: only explicit complete=true qualifies as complete; missing/unknown profile state is incomplete and cannot be re-enabled. Employment must also be active before account re-enable is offered.
- Staff profile edit controls are permission-gated by live `users.update` across directory rows and Overview, Personal Details and Employment. Qualification/certification/experience mutations are also hidden for read-only users. Automated regression coverage is present.
- Directory filter layout coverage now checks mobile (390px), tablet (768px) and desktop (1280px), including overflow and a mobile filter interaction.
- The mocked-session test was corrected to reset filters without reloading the page. Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) **PASS** (83 Playwright tests plus production build); Backend CI [45134091085](https://github.com/TheSumitNain/school-management-system/actions/runs/45134091085) **PASS** (Go/PostgreSQL integration and `go vet`).
- Documents remain explicitly deferred pending storage-service purchase and approval. Bulk import/profile-only onboarding remain blocked pending identity-schema design approval; no migration is authorized by this checkpoint.


## 2026-11-23 implementation checkpoint — attendance export privacy

- The Attendance workspace monthly export now omits free-text attendance remarks. It continues to include aggregate attendance counts/status, employee identifiers, approved leave dates/type and school holiday context; it does not include leave reasons.
- Playwright fixture coverage includes unique private attendance-remark and leave-reason values and checks both are absent from the downloaded monthly CSV. Frontend CI [45129975410](https://github.com/TheSumitNain/school-management-system/actions/runs/45129975410) **PASS** (production build and Playwright suite); the subsequent Backend CI [45134091085](https://github.com/TheSumitNain/school-management-system/actions/runs/45134091085) **PASS** (PostgreSQL integration and `go vet`).
- Commits: `9800956`, `bfc813e`, `41e92b7`, `a6e9684`.


## 2026-11-23 implementation checkpoint — Staff permission audit

- School Admin's runtime permission allowlist now retains the dedicated substitution permissions already provisioned by migration V60. A PostgreSQL integration assertion verifies the effective permissions, and route-level tests verify default-deny behavior for leave, holidays, substitutions, staff qualifications, certifications, experience, reports and audit history.
- Backend CI [45137329371](https://github.com/TheSumitNain/school-management-system/actions/runs/45137329371) **PASS** on code checkpoint `20814d1f` (ephemeral PostgreSQL migrations, complete Go suite including integration tests, and `go vet`).
- This checkpoint does not authorize or apply persistent migrations. Staff document storage remains deferred pending external service selection/purchase and explicit approval; Bulk Import remains blocked pending identity-schema design approval.
- Staff Management is not yet declared complete. The remaining gate is the final consolidated end-to-end/browser acceptance and any outstanding permission/security review; do not treat CI for individual workstreams as a manual acceptance sign-off.


## 2026-11-23 implementation checkpoint — timetable permission verification

- Added positive and negative route coverage for timetable current-year, workload report, periods and entries. Reads require `timetable.read`; mutations require `timetable.manage`; missing permissions are denied before handler execution.
- Backend CI [45137945323](https://github.com/TheSumitNain/school-management-system/actions/runs/45137945323) **PASS** on `cc94c773`: ephemeral PostgreSQL migrations, full Go/PostgreSQL integration suite and `go vet`.
- No persistent database change.

## 2026-11-23 implementation checkpoint — timetable write authorization

- Timetable period and entry mutation authorization now runs in the same transaction context as the write and audit event, before input validation or mutation. This does not claim serializable isolation or complete serialization against every concurrent permission revocation.
- Backend CI [45138373886](https://github.com/TheSumitNain/school-management-system/actions/runs/45138373886) **PASS** on `04590b9d`: migrations, Go/PostgreSQL integration and `go vet`.
- No persistent database change.

## 2026-11-23 implementation checkpoint — permission override browser fixture

- The permission-override Playwright fixture now uses `students.read_basic`, a genuinely delegable permission, instead of the non-delegable `staff.attendance.read`. Application behavior is unchanged.
- Frontend CI [45138964793](https://github.com/TheSumitNain/school-management-system/actions/runs/45138964793) **PASS** with production build and 86 Playwright tests; Backend CI [45138964862](https://github.com/TheSumitNain/school-management-system/actions/runs/45138964862) **PASS** with migrations, Go/PostgreSQL integration and `go vet`, both on `4ddcf02c`.
- No persistent database change.

## 2026-11-23 implementation checkpoint — School Admin staff permission bundle

- PostgreSQL role integration now verifies School Admin effective permissions for attendance read/mark/update/correction, leave read/type/request/decision, holiday read/manage, substitution read/manage and timetable read/manage. This covers provisioned role grants and runtime allowlist filtering.
- Backend CI [45139881590](https://github.com/TheSumitNain/school-management-system/actions/runs/45139881590) **PASS** on `ecefdaf4`: ephemeral migrations, Go/PostgreSQL integration and `go vet`.
- No persistent database change.

## 2026-11-23 implementation checkpoint — attendance target authorization order

- Attendance marking now checks the live mark/correct-past permission inside the write transaction before resolving the target staff profile, preventing unauthorized target-existence disclosure.
- Added a PostgreSQL integration regression test for an unprivileged actor attempting to mark a non-existent staff principal.
- Backend CI [45140616380](https://github.com/TheSumitNain/school-management-system/actions/runs/45140616380) **PASS** on `cbb64f3f`: ephemeral migrations, Go/PostgreSQL integration suite and `go vet`.
- No persistent schema or data changes.


## 2026-11-23 implementation checkpoint — attendance update authorization

- Attendance update now evaluates both same-day-update and past-correction permissions before resolving the record. Unprivileged and partial-scope callers cannot distinguish missing record IDs from records they cannot update.
- Added PostgreSQL integration regression coverage for unauthorized attendance mark and update target enumeration.
- Backend CI [45141655488](https://github.com/TheSumitNain/school-management-system/actions/runs/45141655488) **PASS** on `f340fbe0` (migrations, Go/PostgreSQL integration and `go vet`).
- No persistent database change.


## 2026-11-23 implementation checkpoint — staff handler authentication ordering

- Staff attendance, records and leave/holiday handlers now authenticate before checking service availability, avoiding disclosure of internal dependency state to unauthenticated callers while preserving 503 for authenticated nil-service cases.
- Regression tests cover affected handler actions and unauthenticated requests. Backend CI [45142232011](https://github.com/TheSumitNain/school-management-system/actions/runs/45142232011) **PASS** on `36f278e5` (migrations, Go/PostgreSQL integration and `go vet`).
- No persistent database change.


## 58. Staff leave and holiday read-permission browser acceptance (2026-11-23)

- Added a negative-permission Playwright scenario for a staff account with `staff.leave.manage_requests` and `staff.holidays.read`, but without `staff.leave.read`. It verifies that the holiday calendar remains available while Leave Requests, Applications and New Leave Request UI are absent, and that the browser makes no leave-request API calls.
- The test fixture was aligned with the actual `/api/school/staff-leave/requests` route in commit `965e5d2`.
- Frontend CI [45143798048](https://github.com/TheSumitNain/school-management-system/actions/runs/45143798048) **PASS** (production build and Playwright suite); Backend CI [45143798043](https://github.com/TheSumitNain/school-management-system/actions/runs/45143798043) **PASS** (ephemeral PostgreSQL migrations, Go/PostgreSQL integration suite and `go vet`).
- No persistent database migration or data change. Staff Management remains subject to the consolidated final browser acceptance and remaining security review. Bulk Import and document upload remain deferred under their existing approval constraints.

## 2026-11-23 implementation checkpoint — staff attendance responsive acceptance

- Added a Playwright browser acceptance test for the Staff Attendance workspace at 390px mobile, 768px tablet and 1280px desktop. It verifies no page-level horizontal overflow, that search/status controls and attendance actions remain available, that the data table stays within its own horizontal-scroll container, and that employee search continues to work.
- Frontend CI [45145808505](https://github.com/TheSumitNain/school-management-system/actions/runs/45145808505) **PASS** (TypeScript, production build and full Playwright browser suite) and Backend CI [45145808565](https://github.com/TheSumitNain/school-management-system/actions/runs/45145808565) **PASS** (ephemeral PostgreSQL migrations, Go/PostgreSQL integration suite and go vet) on commit `e7ce7a33`.
- No persistent database migration or data change. Staff Management remains subject to the consolidated final browser acceptance and remaining security review. Bulk Import and staff document storage remain deferred under their existing approval constraints.

## 2026-11-23 implementation checkpoint — monthly attendance responsive acceptance

- Added a separate Playwright browser acceptance scenario for the Staff Attendance monthly summary at 390px mobile, 768px tablet and 1280px desktop. It verifies no page-level horizontal overflow, the month selector and CSV action remain visible, and the table uses contained horizontal scrolling only where the viewport requires it.
- Frontend CI [45146424378](https://github.com/TheSumitNain/school-management-system/actions/runs/45146424378) **PASS** (TypeScript, production build and full Playwright browser suite) and Backend CI [45146424383](https://github.com/TheSumitNain/school-management-system/actions/runs/45146424383) **PASS** (ephemeral PostgreSQL migrations, Go/PostgreSQL integration suite and go vet) on commit `91807957`.
- No persistent database migration or data change. Staff Management remains subject to the consolidated final browser acceptance and remaining security review. Bulk Import and staff document storage remain deferred under their existing approval constraints.

## 2026-11-23 implementation checkpoint — strict leave and holiday date validation

- Centralized staff leave/holiday date parsing: strict canonical `YYYY-MM-DD`, valid calendar dates only, and rejection of year zero (which Go can parse but PostgreSQL `date` does not represent). Applied it to leave request creation, leave request date filters, holiday range reads and holiday creation/updates.
- Moved live permission checks ahead of user-supplied date/field validation for leave request creation and holiday reads/writes, preserving school-scoped transaction behavior for mutations. Added regression coverage for year-zero and non-canonical dates.
- Backend CI [45147539725](https://github.com/TheSumitNain/school-management-system/actions/runs/45147539725) **PASS** (ephemeral PostgreSQL migrations, full Go/PostgreSQL integration suite and go vet) on commit `b8da3ede`.
- No persistent database migration or data change. Staff Management remains subject to the consolidated final browser acceptance and remaining security review. Bulk Import and staff document storage remain deferred under their existing approval constraints.

## 2026-11-23 implementation checkpoint — responsive staff profile navigation

- Added browser acceptance for the employee profile dialog and its eight permission-visible sections at 390px mobile, 768px tablet and 1280px desktop. It verifies the dialog remains inside the viewport, the page has no horizontal overflow, narrow layouts contain section navigation in its own horizontal scroller, and Personal Details remains reachable.
- Frontend CI [45148105892](https://github.com/TheSumitNain/school-management-system/actions/runs/45148105892) **PASS** (TypeScript, production build and full Playwright browser suite) and Backend CI [45148105810](https://github.com/TheSumitNain/school-management-system/actions/runs/45148105810) **PASS** (ephemeral PostgreSQL migrations, full Go/PostgreSQL integration suite and go vet) on commit `d3d613be`.
- No persistent database migration or data change. Staff Management remains subject to the consolidated final browser acceptance and remaining security review. Bulk Import and staff document storage remain deferred under their existing approval constraints.

## 2026-11-23 implementation checkpoint — leave and substitution validation and permission acceptance

- Leave-request and substitution date filters now validate strict calendar dates, paired range endpoints and chronological order before SQL date casts; substitution creation rejects non-canonical dates and year zero.
- Substitution authorization is evaluated before request input validation. PostgreSQL integration regressions protect this ordering for both list and create operations.
- Browser acceptance now covers substitution read-only access (view-only, no assign/cancel controls or timetable fetch) and Leave & Holidays responsive layouts at mobile, tablet and desktop widths.
- Frontend CI [45168421521](https://github.com/TheSumitNain/school-management-system/actions/runs/45168421521) **PASS** with 92 Playwright tests; Backend CI [45169195285](https://github.com/TheSumitNain/school-management-system/actions/runs/45169195285) **PASS** including PostgreSQL integration and `go vet`.
- No persistent database changes. Final Staff Management acceptance is still in progress; Bulk Import and staff document storage remain deferred pending the previously stated design/approval requirements.

## 2026-11-23 implementation checkpoint — attendance read-only browser acceptance

- Added browser regression coverage proving attendance-read-only users cannot mark new records, update existing records, or submit bulk attendance; the UI disables write controls and sends no write requests.
- Frontend CI [45169926822](https://github.com/TheSumitNain/school-management-system/actions/runs/45169926822) **PASS** with the production build and full Playwright suite. Backend validation work remains under separate CI verification.

## 2026-11-23 implementation checkpoint — strict staff record dates and responsive report acceptance

- Certifications and previous-employment date fields now use strict canonical calendar-date validation for both create and update. Year zero, malformed/nonexistent dates and reversed ranges are rejected before PostgreSQL casts; year-one dates are handled correctly, including optional certification issue/expiry comparisons.
- Attendance date and month inputs reject year zero and non-canonical formats. Handler, service and repository layers now agree that year-one calendar values are valid.
- Staff profile joining-date bounds are consistent across create and edit: year 1 or later, no more than one year ahead.
- Added mobile/tablet/desktop browser checks for Substitutions and the school-wide Workload Report, plus read-only attendance and substitution permission acceptance.
- Frontend CI [45171132153](https://github.com/TheSumitNain/school-management-system/actions/runs/45171132153) **PASS**; Backend CI [45172378554](https://github.com/TheSumitNain/school-management-system/actions/runs/45172378554) **PASS** (PostgreSQL migrations/integration and `go vet`) on `617c235e`.
- No persistent DB changes. Profile-only onboarding/Bulk Import remain gated on the identity-schema design and approval; document storage remains deferred pending an external storage selection/purchase and explicit approval.

## 2026-11-23 implementation checkpoint — teacher profile timetable responsive acceptance

- Added mobile/tablet/desktop browser coverage for the teacher-specific Workload & Timetable tab inside the employee profile. It verifies the profile stays within the viewport and timetable/report tables scroll within their own containers.
- Frontend CI [45172972979](https://github.com/TheSumitNain/school-management-system/actions/runs/45172972979) **PASS**; Backend CI [45172973009](https://github.com/TheSumitNain/school-management-system/actions/runs/45172973009) **PASS** on `93da54e6` (PostgreSQL integration and `go vet` included).
- No persistent DB changes. Schema-bound profile fields and the previously deferred identity/document-storage capabilities remain subject to their approval gates.


## 2026-11-24 implementation checkpoint — credential handoff and date-stable browser acceptance

- The one-time credential handoff now clears after both successful copy and **Do this later**; Playwright covers both creation paths.
- Staff substitution, leave and holiday browser scenarios derive calendar dates from the current IST school date. The substitution database-guard integration fixture selects the next Monday in IST to keep its future-cover assertion valid over time.
- Backend CI [45315575893](https://github.com/TheSumitNain/school-management-system/actions/runs/45315575893) **PASS** and Frontend CI [45315575878](https://github.com/TheSumitNain/school-management-system/actions/runs/45315575878) **PASS** (93 Playwright tests) on checkpoint `3c9b6d57`.
- These improvements do not close the identity-schema approval gate, document-storage deferral or consolidated final Staff Management acceptance. No persistent database was changed.


## 2026-11-24 design checkpoint — staff identity and import contract consistency

- Clarified the proposal-only staff identity model: operational Staff Type is teaching/non-teaching; `unclassified` is temporary migration reconciliation only and is never teaching-eligible. Employment Type values are aligned to the product list.
- Added school-scoped designation and department references with saved display-label snapshots. Master renames do not silently rewrite historical employee labels; explicit reassignment refreshes the snapshot. New selections use active school-managed master values.
- Hardened the future CSV bulk-import contract: read-only preview, actor/school/preview-bound expiring confirmation, revalidation and duplicate checks inside the all-or-nothing write transaction, idempotent batch keys, bounded parsing and safe treatment of imported cell content.
- These remain design-only requirements. Profile-only onboarding and Bulk Import are still blocked pending explicit schema/product approval. No migration, runtime code or persistent database was changed.


## 2026-11-24 implementation checkpoint — permission-scoped Staff workspace entry

- The Staff workspace navigation now appears for users who have a relevant staff-specific read capability (attendance, leave/holidays, substitutions, academic assignments or timetable workload), as well as staff directory readers and staff creators. The page opens directly on the first permitted workspace; directory, overview metrics, reports and ID cards remain hidden without `users.read`.
- The staff directory query is disabled when `users.read` is absent, preventing an unauthorized `/school/staff` request from a user who only has access to a separate staff workspace. Added Playwright acceptance for attendance-only access, including no directory request and no directory/overview controls.
- Frontend CI [45318837352](https://github.com/TheSumitNain/school-management-system/actions/runs/45318837352) **PASS**: TypeScript, design-token check, production build and **94 Playwright tests**. Backend CI [45318837314](https://github.com/TheSumitNain/school-management-system/actions/runs/45318837314) **PASS**: migrations, Go/PostgreSQL integration suite and `go vet`.
- No database migration or persistent database change. `main` remains untouched; identity-schema approval gates and the previously agreed document/photo deferral remain in force.


## 2026-11-24 follow-up — read-only leave workspace isolation

- Added a leave-only role acceptance case: a user with `staff.leave.read` can open Leave & Holidays and view applications without receiving Staff Directory or overview access and without triggering `/school/staff`.
- Finalized permission isolation for Staff workspace entry: substitution management is not exposed as a standalone entry capability unless the user can also read staff records; cached staff-directory data is suppressed whenever `users.read` is absent, and directory errors are not surfaced in create-only views.
- Frontend CI [45319733688](https://github.com/TheSumitNain/school-management-system/actions/runs/45319733688) **PASS**: production build and **95 Playwright tests**. Backend CI [45319733387](https://github.com/TheSumitNain/school-management-system/actions/runs/45319733387) **PASS**: PostgreSQL integration, full Go tests and `go vet`.
- No database migration or persistent data change; `main` remains untouched.


## 2026-11-24 implementation checkpoint — report date and CSV download reliability

- Staff attendance CSV object URLs now remain available through the browser download handoff before cleanup, matching the other staff report exporters. The operational reports default month now derives from the shared Asia/Kolkata school-date helper rather than a hard-coded fallback year.
- Frontend CI [45320435562](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435562) **PASS**: production build and 95 Playwright tests. Backend CI [45320435534](https://github.com/TheSumitNain/school-management-system/actions/runs/45320435534) **PASS**: full Go/PostgreSQL integration and `go vet`.
- No schema migration or persistent database change.


## 2026-11-24 follow-up — consolidated Staff Management CI and leave eligibility acceptance

- Frontend CI [45323508729](https://github.com/TheSumitNain/school-management-system/actions/runs/45323508729) **PASS** on `59b7a82f3a81ba1ca89db753dbc8794d9b5318f2`: TypeScript, Vite production build and **98 Playwright browser tests**. The three latest leave-request acceptance cases verify that managers without Staff Directory read access receive a safe selection gate, only active staff with `profile_complete === true` are selectable, and an empty eligible-staff set produces a clear empty state rather than a broken form.
- Backend CI [45323508810](https://github.com/TheSumitNain/school-management-system/actions/runs/45323508810) **PASS** on the same commit: full Go tests, ephemeral PostgreSQL integration suite and `go vet`.
- No persistent database migration or data change. `main` remains untouched. Profile-only onboarding/Bulk Import, external document storage and other previously recorded approval-gated work remain deferred.


## 2026-11-24 implementation checkpoint — employee profile record isolation

- Added delayed-response Playwright coverage for Qualifications & Experience proving that qualifications, certifications and previous-employment history from one employee never flash in the next employee's profile while their requests are pending.
- Added PostgreSQL integration coverage for same-school employee ownership boundaries: listing another employee's qualification, certification or experience returns no records, and updating another employee's record by ID is rejected.
- Frontend CI [45356454122](https://github.com/TheSumitNain/school-management-system/actions/runs/45356454122) **PASS** with TypeScript, Vite production build and **118 Playwright tests**. Backend CI [45356977023](https://github.com/TheSumitNain/school-management-system/actions/runs/45356977023) **PASS** with full Go tests, ephemeral PostgreSQL integration and `go vet`.
- No migration, seed, persistent database or application data changed. Phase 4 leave-policy approval and Phase 6 identity-schema approval gates remain open; document storage and sensitive personal-field collection remain deferred.


## 2026-11-24 implementation checkpoint — permission override audit history

- Fixed the staff audit-history query so `user_permission_override` events are included for the target employee using the target principal ID, while retaining the school-scoped filter.
- Added PostgreSQL integration fixtures and assertions for both permission-override set and remove events. The change contains no schema migration or persistent database change.
- Backend CI [45384692873](https://github.com/TheSumitNain/school-management-system/actions/runs/45384692873) **PASS** on `9d77be79745e99787b885c9736b3db778ec72813` (full Go tests, PostgreSQL integration and `go vet`). The previously completed frontend CI [45383871353](https://github.com/TheSumitNain/school-management-system/actions/runs/45383871353) **PASS** on `66af8c6adb08d6937f1abf13d50f1d7e9a7f3e14`; this backend-only change does not alter frontend code.
- Phase 4 leave-policy approval and Phase 6 identity-schema approval remain open. No persistent database or application data was changed.


## 2026-11-24 implementation checkpoint — staff record Unicode length parity

- Staff qualification, certification and employment-history text validation now counts Unicode code points consistently across Go service validation and frontend local validation. Browser input limits allow supplementary-plane characters up to the backend's equivalent character limit rather than truncating emoji/supplementary Unicode by UTF-16 code units.
- Backend regression tests cover 120/121 Unicode-character credential references and certification IDs on create/update, with invalid records rejected before repository writes. Frontend browser acceptance exercises 120-character Unicode credential values.
- Backend CI [45392293745](https://github.com/TheSumitNain/school-management-system/actions/runs/45392293745) **PASS**: migrations, full Go/PostgreSQL integration tests and `go vet`. Frontend CI [45392293732](https://github.com/TheSumitNain/school-management-system/actions/runs/45392293732) **PASS**: TypeScript/design-token checks, Vite production build and Playwright browser acceptance suite.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and the previously agreed external document-storage deferral remain open; `main` remains untouched.

## 2026-11-24 implementation checkpoint — staff profile email validation

- Staff profile create/update now share an application-level email validator: optional blank values remain valid; supplied values must be 5–254 Unicode code points, contain exactly one `@`, contain no whitespace, and have a non-empty dotted domain. This matches the frontend's email-format rule and 254-character cap while remaining stricter than the database's basic storage constraint.
- Added service regression coverage for valid 254-character Unicode email values, malformed/over-limit values, and create/update repository-call boundaries.
- Backend CI [45401808973](https://github.com/TheSumitNain/school-management-system/actions/runs/45401808973) **PASS** on `22977d39356d0df4c4e04b5cb44743c393e6306e`: migrations, full Go/PostgreSQL integration tests and `go vet`.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and external document-storage deferral remain open; `main` remains untouched.


## 2026-11-24 implementation checkpoint — staff profile email browser parity

- Tightened the frontend browser acceptance coverage for malformed email components (consecutive dots, trailing domain dot, and leading domain-label hyphen), accepted the shortest supported address, accepted a Unicode address, and verified a 254-code-point Unicode email is not truncated by the input limit.
- The frontend uses a text input with an email keyboard hint so browser-native ASCII email validation does not reject Unicode values that the shared backend/frontend validator accepts. The backend validator comment now accurately describes it as stricter than the database storage constraint.
- Frontend CI [45403612085](https://github.com/TheSumitNain/school-management-system/actions/runs/45403612085) **PASS** on `606ff51b5af809f0ceb1eadef13bf77b6ab6d995`: TypeScript, Vite production build and full Playwright browser acceptance suite. Backend CI [45403916189](https://github.com/TheSumitNain/school-management-system/actions/runs/45403916189) **PASS** on `5c5f91c75a6db42f1863dfad94a8396413580a3d`: ephemeral database migrations, full Go tests including PostgreSQL integration and `go vet`.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and external document-storage deferral remain open; `main` remains untouched.


## 2026-11-24 implementation checkpoint — optional staff email clearing

- Added a PostgreSQL integration regression to update an existing employee profile with a blank optional email and verify the database stores SQL `NULL` while the API model returns an empty string. This protects the profile editor's ability to remove a previously saved email without violating the staff profile constraint.
- Backend CI [45404634591](https://github.com/TheSumitNain/school-management-system/actions/runs/45404634591) **PASS** on `6454bb9779254cc1a2619cbb4870f3f3630e73c7`: all migrations, full Go tests including PostgreSQL integration and `go vet`.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and external document-storage deferral remain open; `main` remains untouched.

## 2026-11-24 implementation checkpoint — optional staff email and employment-status validation

- Added a PostgreSQL integration regression proving a new staff profile may omit optional email: storage uses SQL `NULL` while the returned API model exposes an empty string. Added a Playwright regression proving the profile editor can clear a previously saved email.
- Tightened profile updates to require an explicit `employment_status` (`active` or `inactive`) instead of silently defaulting an omitted value to `active`. This prevents incomplete update payloads from unintentionally changing an inactive employee's employment state. Added a service regression proving missing status is rejected before repository writes.
- Frontend CI [45405411349](https://github.com/TheSumitNain/school-management-system/actions/runs/45405411349) **PASS** on `ef58bb0f984baa9edd6a43037d7112bd2d1c4efe`. Backend CI [45405803945](https://github.com/TheSumitNain/school-management-system/actions/runs/45405803945) **PASS** on `b3d73d6d1f17b3ab0e6407289ca3ad4b54c65a91`; Backend CI [45406177450](https://github.com/TheSumitNain/school-management-system/actions/runs/45406177450) **PASS** on `c848ae65f8604fab545bf75b126fad67dbd40d06`, including all migrations, full Go/PostgreSQL integration tests and `go vet`.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and external document-storage deferral remain open; `main` remains untouched.

## 2026-11-24 implementation checkpoint — school-local joining-date validation

- Staff profile creation and update now validate the optional joining date as a calendar date in `Asia/Kolkata`, with a one-year-ahead limit and a leap-day-safe anniversary boundary. This removes dependence on the backend host's local timezone and avoids comparing a date-only value against a time-of-day timestamp.
- The staff form's date picker now disables dates beyond the same school-local one-year limit, and create/edit submission validation places an out-of-range date error under the Joining date field. The update API and mutation types now require an explicit `employment_status`, matching backend validation.
- Added deterministic backend boundary tests for the school-local date transition and leap-day clamp, plus Playwright coverage that dates beyond the supported year are disabled. Backend CI [45408217416](https://github.com/TheSumitNain/school-management-system/actions/runs/45408217416) **PASS**: all migrations, full Go/PostgreSQL integration tests and `go vet`. Frontend CI [45407819629](https://github.com/TheSumitNain/school-management-system/actions/runs/45407819629) **PASS**: TypeScript, Vite production build and full Playwright browser acceptance suite.
- No schema migration or persistent database/application-data change. Phase 4 leave-policy approval, Phase 6 identity-schema approval, and external document-storage deferral remain open; `main` remains untouched.


## 2026-11-25 follow-up — staff profile date parity and credential privacy

- Staff profile create and edit now share the same Asia/Kolkata calendar-date validator for the optional joining date, including the one-year-ahead limit and leap-day-safe boundary. The update path no longer relies on the backend host's local clock. No database migration or persistent data change.
- Added explicit handler regression assertions that seeded password hashes are absent from staff-create and staff-directory responses.
- Backend CI [45534647049](https://github.com/TheSumitNain/school-management-system/actions/runs/45534647049) **PASS** on eb863eb2bbf888a08f9ac53fc4f166c2ac13e3dc, including migrations, full Go/PostgreSQL integration tests and go vet.
- Existing Phase 4 leave-policy and Phase 6 identity-schema approvals remain outstanding; sensitive personal fields, Payroll and external document/photo storage remain deferred. main remains untouched.

## 2026-11-25 follow-up — Edit Staff HTTP acceptance

- Added direct backend handler regression coverage for Edit Staff profile updates: authenticated actor/school/target propagation, complete profile payload mapping, date-only parsing, safe response allowlist and malformed joining-date rejection without a service call.
- Backend CI [45535081221](https://github.com/TheSumitNain/school-management-system/actions/runs/45535081221) **PASS** on `9eb6150429718d559f88933c04c7487181af78d9`, including migrations, full Go/PostgreSQL integration tests and `go vet`.
- No schema or persistent data change. Leave-policy and identity-schema approvals remain open; document/photo storage remains deferred; `main` remains untouched.

## 2026-11-25 follow-up — Staff account lifecycle transaction and isolation regressions

- Added PostgreSQL integration regressions proving staff profile updates roll back when their audit insert fails, reject cross-school targets without mutation, and reject duplicate employee codes atomically without a success audit event.
- Added lifecycle rollback regressions for username changes, password resets, account disable and account re-enable. Each forces the corresponding audit insert to fail and verifies the account/principal state, token version, password state and active refresh-session state remain unchanged.
- Added an inactive-employment guard regression: a disabled login cannot be re-enabled while its employee profile is inactive. Added cross-school lifecycle coverage and nil-dependency safety checks for credential lookup/password operations; handler coverage verifies incomplete or inactive employee profiles map to HTTP 409.
- The initial run exposed a malformed dollar-quote delimiter in the staff-profile audit-trigger test. Replaced it with a named PL/pgSQL dollar-quote delimiter in commit `c4345e9f2d01ed2bc7115d2606c6c3f0792b4c20`.
- Backend CI [45537146943](https://github.com/TheSumitNain/school-management-system/actions/runs/45537146943) **PASS** on `c4345e9f2d01ed2bc7115d2606c6c3f0792b4c20`: all migrations, full Go tests including PostgreSQL integration, and `go vet`.
- These are regression-test and test-fixture changes; no schema migration or persistent school data change. Leave-policy and identity-schema approvals remain open; document/photo storage remains deferred and excluded from the Staff Management completion estimate. `main` remains untouched.


## 2026-11-25 follow-up — qualification/certification verification-state integrity

- Strengthened the PostgreSQL integration regression by explicitly submitting forged `verification_status: "verified"` values on both qualification and certification create and update operations. The repository must return newly created records as `pending` and preserve the existing `pending` state after edits.
- Backend CI [45540370935](https://github.com/TheSumitNain/school-management-system/actions/runs/45540370935) **PASS** on `6f21d439daa010cd997ee0dc60e90d380d87e66a`: full migration chain, Go tests including PostgreSQL integration and `go vet`.
- This is regression-test hardening; no production behavior change, schema migration or persistent data change. A verification/approval workflow and archive/delete policy remain undefined and are not being enabled without an explicit permission and retention decision. Document/photo storage remains deferred and excluded from the Staff Management completion estimate. `main` remains untouched.


## 2026-11-25 follow-up — staff leave validation and API error contract

- Added holiday-calendar service validation tests for the 366-day maximum, invalid/reversed/missing dates, year zero, and repository non-invocation on invalid input; valid school-scoped filters are forwarded unchanged.
- Added leave-handler tests for stable 400/403/404/409/500 responses and for suppressing internal database error details from client responses.
- Backend CI [45601046350](https://github.com/TheSumitNain/school-management-system/actions/runs/45601046350) **PASS** on `53a31c317a72af379b10d71fe6beb7672f842bf5`: full migrations, Go/PostgreSQL integration tests and `go vet`.
- Regression-test changes only; no schema or persistent database changes. Leave balances, accrual and carry-forward remain blocked on explicit school policy approval. Document/photo storage remains deferred and excluded from the Staff Management completion estimate; `main` remains untouched.


## 2026-11-25 follow-up — leave decision payload and holiday-manager access

- Fixed chunked/unknown-length leave decision request parsing so approval/rejection comments are not dropped; added a handler regression.
- Aligned the holiday calendar GET route with the existing frontend permission model: either `staff.holidays.read` or `staff.holidays.manage` may reach the handler. Repository checks continue to distinguish active-only reads from inactive-record management access. Added route-level regression coverage for manage-only access.
- Backend CI [45602173749](https://github.com/TheSumitNain/school-management-system/actions/runs/45602173749) **PASS** on `42ca6cf7709a742dd53029c284cae5c25d671a7c`: full migrations, Go/PostgreSQL integration tests and `go vet`.
- No persistent database change. Leave entitlement policy remains approval-gated; document/photo storage remains deferred and excluded from the Staff Management completion estimate. `main` remains untouched.


## 2026-11-25 follow-up — staff audit history lifecycle and immediate refresh

- Audit-history coverage now includes account disabled/enabled, username updated, password reset/changed, role assigned/revoked and individual permission override set/removed events. PostgreSQL integration fixtures mirror the production metadata needed for school-scoped target resolution; Playwright verifies the corresponding event labels.
- Fixed stale open audit-history panels after account/profile/credential changes, qualification/certification/experience mutations, role assignment/removal and permission-override save/removal. These mutations now invalidate the selected employee's audit-history query; browser regressions verify role and override changes trigger a fresh history read.
- Frontend CI [45617925032](https://github.com/TheSumitNain/school-management-system/actions/runs/45617925032) **PASS**: TypeScript, Vite build and full Playwright suite. Backend CI [45618359896](https://github.com/TheSumitNain/school-management-system/actions/runs/45618359896) **PASS**: all migrations, full Go/PostgreSQL integration suite and `go vet`.
- No schema or persistent data change. Existing leave-policy, employee identity-schema and qualification/certification approval/retention decisions remain explicit product gates. Document/photo storage remains deferred and excluded from the completion estimate; `main` remains untouched.


## 2026-11-25 follow-up — leave decisions refresh staff audit history

- Successful staff leave request creation, approval/rejection and cancellation now invalidate employee audit-history queries, so the History & Activity panel cannot keep recently cached leave events after a profile is reopened.
- Added Playwright coverage for opening a staff member's history, approving their pending leave request, reopening the profile and asserting that audit history is fetched again.
- Backend CI [45619016333](https://github.com/TheSumitNain/school-management-system/actions/runs/45619016333) **PASS**: complete migration chain, Go/PostgreSQL integration suite and `go vet`. Frontend CI [45619016246](https://github.com/TheSumitNain/school-management-system/actions/runs/45619016246) **PASS**: TypeScript, Vite build and complete Playwright suite.
- No schema or persistent data change. Leave entitlement policy remains approval-gated; document/photo storage remains deferred and excluded from Staff Management completion estimates; `main` remains untouched.


## 2026-11-25 follow-up — staff qualification, certification and experience transaction rollback

- Added direct PostgreSQL failure-injection coverage for all six qualification/certification/experience repository write paths (create and update). If the matching audit event cannot be inserted, the operation must fail atomically: create leaves no row behind, while update retains the original field values.
- Backend CI [45627964262](https://github.com/TheSumitNain/school-management-system/actions/runs/45627964262) **PASS** on `4c07ed148cd939add6dffca03635c67f8fb6a53e`, including full migrations, Go/PostgreSQL integration tests and `go vet`.
- This is regression-test hardening only; no production code or schema migration changed. Verification approval and archive/retention policy remain undefined product decisions and are not enabled. Document/photo storage remains deferred and excluded from the completion estimate; `main` remains untouched.


## 2026-11-25 follow-up — leave request creation transaction regression

- Added a PostgreSQL failure-injection test for staff leave request creation. When the required `staff.leave.requested` audit event cannot be inserted, the transaction must leave neither a request row nor an audit row. Existing integration coverage already verifies rollback for decisions, cancellation and holiday creation.
- Backend CI [45630342631](https://github.com/TheSumitNain/school-management-system/actions/runs/45630342631) **PASS** on `c9ee6e8671b0fcf3faa76fa2ec0aa02fb58a68af`: all migrations, full Go/PostgreSQL integration tests and `go vet`.
- Test-only hardening; no production code or schema migration changed. School leave quotas, accrual and carry-forward remain approval-gated; `main` remains untouched.


## 2026-11-25 follow-up — leave-type audit transaction regressions

- Added PostgreSQL failure-injection checks for leave-type creation and update. Both operations must roll back when their required audit event fails; the new type must not persist, the original type must remain unchanged, and no false audit event may be committed.
- Backend CI [45631108229](https://github.com/TheSumitNain/school-management-system/actions/runs/45631108229) **PASS** on `67267c6a7c2b5f9c4fc240d6df34fe0c0148b85c`: full migrations, Go/PostgreSQL integration suite and `go vet`.
- Test-only hardening; no production code or schema migration changed. School leave entitlement rules remain approval-gated; `main` remains untouched.


## 2026-11-25 follow-up — attendance correction audit rollback

- Added PostgreSQL regression coverage for an audit insert failure during individual attendance correction. The prior attendance status must remain unchanged and no additional `school.staff_attendance.updated` audit event may commit. This complements existing single-mark and bulk-mark rollback tests.
- Backend CI [45632375019](https://github.com/TheSumitNain/school-management-system/actions/runs/45632375019) **PASS** on `236eac8c11764f9e12a3057ab3c08e29c2d629f1`: all migrations, full Go/PostgreSQL integration suite and `go vet`.
- Test-only hardening; no production behavior or schema change. `main` remains untouched.


## 2026-11-25 follow-up — holiday update audit rollback

- Extended the holiday audit-failure regression to cover both create and update. A failed update audit must leave the existing holiday's date and name intact and must not append a false success event; failed creation must leave no partial holiday row or audit event.
- Backend CI [45633050577](https://github.com/TheSumitNain/school-management-system/actions/runs/45633050577) **PASS** on `27bea4e68dbdf6bb2e4c5997d9f602d3b3523dbc`: migrations, full Go/PostgreSQL integration tests and `go vet`.
- Test-only hardening; no production behavior or schema change. `main` remains untouched.


## 2026-11-25 follow-up — leave creation and cancellation history refresh

- Extended browser acceptance coverage beyond leave decisions: after creating a pending request and after cancelling it, reopening the employee profile must trigger a fresh History & Activity read. The test confirms both invalidation paths work through the UI.
- Backend CI [45634722560](https://github.com/TheSumitNain/school-management-system/actions/runs/45634722560) **PASS** and Frontend CI [45634722478](https://github.com/TheSumitNain/school-management-system/actions/runs/45634722478) **PASS** on `117b25a6ec0820d0371fdf7c7cafda79b305c5c0`.
- Test-only hardening; no production behavior or schema change. Leave entitlement policy remains approval-gated; `main` remains untouched.


## 2026-11-25 follow-up — completed substitution cancellation enforced by backend

- Consolidated audit found that the Substitutions UI correctly hides cancellation for past/completed cover dates, but the repository cancellation operation previously accepted any active row. A caller with the manage permission could bypass the UI and cancel historical coverage.
- The repository now locks and checks the target record using the Asia/Kolkata school date and rejects cancellation of completed covers with a stable conflict response. The update itself repeats the date predicate to keep the write condition authoritative; cancelled and unknown records retain the existing not-found behavior.
- Added PostgreSQL integration coverage proving a past substitution remains active and retained when cancellation is attempted. Existing browser acceptance already verifies that completed covers show no cancellation action.
- Backend and Frontend CI are pending for the new code checkpoint; do not mark this fix verified until both complete successfully.
- No schema migration or persistent database change. `main` remains untouched.

## 2026-11-26 — Leave and substitution result-set capacity checkpoint

- Removed the former 1,000-row SQL caps from the school-scoped Staff Leave Requests and Substitution History queries. The APIs no longer silently return an incomplete result set when a date/filter combination exceeds that threshold.
- Backend CI [45648554311](https://github.com/TheSumitNain/school-management-system/actions/runs/45648554311) **PASS** on `5304b86609762741b500046e730e91c874a6bb81`: migrations, Go tests including PostgreSQL integration tests, and `go vet` passed.
- This is a capacity/truncation correction, **not** server-side pagination. Bounded Leave/Substitution pages with total counts and complete all-pages CSV/report export remain open; do not describe these workflows as fully scalable until those are implemented and verified.
- `main` remains untouched. No schema migration or persistent database change was made.



## 2026-11-26 follow-up — read-only leave/holiday permissions and substitution tenant boundary

- Added browser acceptance for leave-read-only and holiday-read-only principals. Both can reach the permitted workspace without triggering Staff Directory requests; mutation actions are absent for these read-only principals.
- Frontend CI [45745435651](https://github.com/TheSumitNain/school-management-system/actions/runs/45745435651) **PASS** with TypeScript, production build and the full Playwright suite. Backend CI [45745435658](https://github.com/TheSumitNain/school-management-system/actions/runs/45745435658) **PASS**.
- Added PostgreSQL integration assertions using a School Admin with home-school substitution permissions: cross-school create, cancel and list attempts must all return access denied. A separate test covers foreign-school authorization before input/record handling.
- Backend CI [45746067897](https://github.com/TheSumitNain/school-management-system/actions/runs/45746067897) **PASS**: full migrations, Go/PostgreSQL integration suite and `go vet`.
- These are regression-test improvements, not a claim that Staff Management is complete. Identity-schema approval for onboarding/bulk import, school policy approval for leave entitlements, document-storage approval and authenticated School Admin browser sign-off remain outstanding. `main` remains untouched.



## 2026-11-26 follow-up — staff record and attendance write isolation

- Added cross-tenant regression assertions for Qualification, Certification and Experience reads/creates/updates, including database row-count checks after rejected writes. Backend CI [45746562607](https://github.com/TheSumitNain/school-management-system/actions/runs/45746562607) **PASS**.
- Added forged-school-ID tests for single attendance mark, bulk mark and attendance update/correction. The existing daily/monthly read isolation tests remain in place; rejected bulk writes are verified to leave no rows. Backend CI [45746893796](https://github.com/TheSumitNain/school-management-system/actions/runs/45746893796) **PASS**.
- Both CI runs execute the complete migration chain, Go/PostgreSQL integration suite and `go vet`. These are test-only tenant-isolation improvements, not production behavior changes.
- Staff Management remains in progress pending remaining workflow audit and authenticated School Admin browser sign-off. `main` remains untouched.
