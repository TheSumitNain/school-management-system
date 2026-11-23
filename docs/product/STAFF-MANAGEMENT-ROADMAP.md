# Staff Management — Product and Implementation Roadmap

**Status:** Approved planning baseline; Phase 0 audit is complete and the P0 Staff Management workspace rebuild is in progress.  
**Scope:** Phase 1 School Admin web workspace, with Teacher self-service as a separate future experience.  
**Last updated:** 2026-11-23

This document defines what Staff Management will build, in what order, the screens/tabs and forms users will see, each field's required/optional status, the key workflows, security boundaries and acceptance criteria. It is a roadmap, not evidence that a feature is implemented. Check [Project Status](../project/STATUS.md) before each phase and verify existing code before changing it.

## 1. Product decisions and goals

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
8. This roadmap does not authorize speculative database migrations. Confirm the existing schema/API and define each required additive migration before implementation.

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
| 2 | Personal Details | P0 | Name, contact, DOB, gender, address and emergency contact |
| 3 | Employment | P0 | Employee code, staff type, department, designation, employment type, dates and status |
| 4 | Qualifications & Experience | P0 | Education, specialization, certifications, experience and previous employment |
| 5 | Documents | P0 | Staff documents, category, issue/expiry dates and verification status |
| 6 | Academic Assignments | P0 | Class-in-charge and subject teaching assignments with history |
| 7 | Workload & Timetable | P1 | Current-year teacher-specific scheduled periods and timetable rows; unscheduled slots are not confirmed availability |
| 8 | Account & Access | P0 | Login lifecycle, roles, delegated permissions and password reset |
| 9 | History & Activity | P0 | Employment, assignment and relevant audit/activity events |

Tabs with no records must show an explanatory empty state and an action when the viewer has permission. Sensitive fields/documents must be field-authorized; directory access does not imply access to every profile field.

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
| Photograph | Optional | Supported image types/size limits; preview before save |

#### Step 2 — Employment Details

| Field | Required? | Rules / behavior |
|---|---|---|
| Designation | Required | Select from school designation master; controlled setup if catalogue is absent |
| Department | Optional initially | Select from school department master; avoid duplicate free text |
| Joining Date | Optional | Shared IST-aware DatePicker; store date-only value |
| Employment Type | Optional initially | Full-time, Part-time, Contract, Temporary, Visiting |
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
| Date of Birth | Optional |
| Gender | Optional |
| Photograph | Optional |
| Current Address | Optional |
| Permanent Address | Optional |
| Emergency Contact Name | Optional |
| Emergency Contact Relationship | Optional |
| Emergency Contact Mobile | Optional |

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

### 3.4 Form D — Add / Edit Staff Document (P0)

**Backend security contract:** see [Staff Qualifications & Documents — Backend Contract](../backend/STAFF-QUALIFICATIONS-DOCUMENTS.md). Qualifications, certifications and experience require school-scoped CRUD, verification permissions and transactional audit. Document upload/download remains gated on private storage; do not implement public URLs or mock uploads.


| Field | Required? |
|---|---|
| Document Type / Category | Required |
| Display Name | Required |
| File | Required on create; optional on metadata-only edit |
| Document Number | Optional |
| Issue Date | Optional |
| Expiry Date | Optional |
| Verification Status | Required, default Pending |
| Notes | Optional |

Validate allowed MIME/type, file size and upload errors. Restrict access to authorized roles. Do not expose document URLs or sensitive metadata to unauthorized users. P0 records expiry dates; expiry reminders are P2.

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
3. Profile page with 8 tabs; implement Personal Details, Employment, Qualifications, Documents and History foundations.
4. Preserve existing username/password/account creation, reset, role assignment and permission override behavior.
5. Separate employment status from account status; implement safe deactivate/reactivate/relieve semantics.
6. Add audit events and profile completeness indicators.
7. Support non-teaching staff in same foundation without Teacher access by default.

**Exit gate:** create/edit/view teaching and non-teaching profiles; profile-only employee; teacher login provisioning; safe account/employment lifecycle; strict school isolation; field-level validation; automated tests and responsive browser coverage.

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

**Exit gate:** duplicate/date/permission safeguards, audited corrections, accurate reports and responsive browser coverage.

### Phase 4 — Leave & Holiday Management (P1)

1. Leave type foundation. **Implemented:** school-scoped leave type schema, non-delegable permissions, create/list/update APIs, and frontend create/edit/deactivate/reactivate lifecycle with confirmation and history preservation.
2. Staff leave application. **Implemented:** staff/leave-type selection, inclusive dates, full/half-day request, reason, request listing and filters. Backend rejects invalid/inactive targets and overlapping pending/approved requests; per-staff transaction advisory locking serializes overlap checks. Form errors are isolated by workflow pane and recoverable values remain available.
3. Approval/rejection. **Implemented:** pending-only decisions, authenticated decision actor/time, required rejection comment, applicant self-approval denial, cancellation and transactional audit. PostgreSQL integration covers concurrency, authorization, rollback, rejection reason, audit persistence and cross-school denial. **Balance view pending:** quotas/accrual/carry-forward remain undefined until a school policy is approved; no balance is fabricated.
4. Holiday calendar. **Implemented:** school-scoped create/list/update/deactivate APIs; active-only default listing; manager-only inactive listing; explicit deactivate/reactivate confirmations; edit preserves inactive state. PostgreSQL integration covers normalized duplicates, conflicting updates, cross-school denial and audit rollback.
5. Attendance context and reporting. **Implemented:** approved leave and active holidays appear separately in daily/monthly attendance context when the viewer has the relevant read permissions. Monthly CSV export separates attendance, approved leave and holidays and protects against spreadsheet formula injection. No attendance rows are automatically created or overwritten.
6. Permission boundaries and browser acceptance. **Implemented:** the workspace requires a read permission; write/decision-only grants cannot expose leave list views. Browser coverage includes the leave workflows, holiday reactivation, form-error isolation, monthly CSV export and responsive layouts. A read-only holiday-calendar acceptance test verifies that `staff.holidays.read` without `staff.holidays.manage` exposes the active calendar but no mutation controls or inactive records (frontend and backend CI pass at `eaaea2f2f0f4a3665221a3fe0d1807f404eac6a7`).

**Exit gate remains open:** complete any remaining end-to-end workflow review and school-specific policy approval. Leave balances, quotas, accrual and carry-forward must not be implemented until the school explicitly approves the policy. Keep attendance context informational; manual attendance conflicts require explicit human resolution.

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

**Exit gate:** no double-booking, recurring timetable history preserved, dated substitutions audited and assignment limits enforced. Teacher availability remains a separate policy decision; workload reports describe actual scheduled entries only.

### Phase 6 — Reports, Import/Export & ID Cards (P1/P2)

1. Staff directory/employment readiness summary, monthly attendance and leave-request summaries, current-academic-year assignment summary, Qualifications & Certifications report, and Experience History report are implemented. Aggregate reports use bounded school-scoped queries, live read permissions, pagination and allowlisted CSV fields. Qualifications and Experience reports do not fetch per employee (no N+1 requests). Qualifications and Experience History report backend checks and frontend browser acceptance are verified; Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) passed TypeScript, production build and the Playwright suite (83 tests).
2. Printable Staff ID Cards workspace is implemented in the Staff Management UI. It is permission-gated and includes only active records with `profile_complete === true` plus required employee identity fields. It supports multi-select, Classic/Compact/Modern preview, optional validity date and print-only A4 layout. Only employee name, employee code, designation, joining date, employment status and optional validity are printed. Contact details, login/permission information and QR codes are excluded. ID Cards Playwright coverage is present; the latest Frontend CI [45133733286](https://github.com/TheSumitNain/school-management-system/actions/runs/45133733286) passed the production build and Playwright suite after the preview locator and eligibility fixes.
3. Bulk Staff Import is intentionally blocked on the existing identity schema. `staff_profiles.principal_id` is a required foreign key to `users`, while the approved product contract requires employee profiles to exist independently from optional login accounts. Do not create usernames/passwords as an import side effect or implement a profile-only import against the current schema. See [Staff Bulk Import Architecture Gate](../backend/STAFF-BULK-IMPORT-DESIGN.md). The decoupled identity model, compatibility/backfill plan and migration must be reviewed and explicitly approved before import implementation.
4. Permission-controlled exports remain implemented for the existing reports and directory workspace. Exports omit restricted remarks/reasons, exclude revoked assignments, omit unauthorized assignment columns, and neutralize spreadsheet formula injection.
5. Staff document upload/download, expiry reminders and verification remain deferred until the user purchases/selects an external storage service and approves its integration. Until then, do not implement upload UI, local-disk storage or file APIs.

**Exit gate:** import preview must not write; explicit confirmation and accurate row counts; atomic confirmed batch writes; sensitive exports restricted; ID cards use approved fields; all report and print workflows pass browser acceptance.

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
