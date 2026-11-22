# Staff Management — Product and Implementation Roadmap

**Status:** Approved planning baseline; Phase 0 audit is complete and the P0 Staff Management workspace rebuild is in progress.  
**Scope:** Phase 1 School Admin web workspace, with Teacher self-service as a separate future experience.  
**Last updated:** 2026-11-21

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

### 2.1 School Admin Staff Management — 7 top-level tabs

| # | Tab | Phase | Purpose |
|---:|---|---|---|
| 1 | Overview | P0 | Staff summary, setup gaps and quick actions |
| 2 | Staff Directory | P0 | Search, filter, create and manage employee profiles |
| 3 | Academic Assignments | P0 | Class-in-charge and subject/class/section assignments |
| 4 | Attendance | P1 | Daily staff attendance and monthly register |
| 5 | Leave | P1 | Leave requests, balances, approvals and holiday calendar |
| 6 | Workload & Timetable | P1 | Teacher periods, conflicts and substitute coverage |
| 7 | Reports | P1, expanded P2 | Staff, assignment, attendance, leave and later payroll reports |

**Account & Access is not a separate top-level tab.** It belongs to the selected employee's profile because it is specific to that employee. This avoids splitting one employee workflow across distant pages.

Tab visibility and actions are permission-aware. A hidden tab is not authorization: every API operation must enforce live permissions and school/resource scope on the server.

### 2.2 Staff profile — 8 tabs

Selecting a staff row opens a full profile page (preferred for deep profiles and direct links; a responsive drawer may be used for a compact quick view). The profile has **8 tabs**:

| # | Profile tab | Phase | Contents |
|---:|---|---|---|
| 1 | Overview | P0 | Identity summary, employment/account status, current assignments and recent activity |
| 2 | Personal Details | P0 | Name, contact, DOB, gender, address and emergency contact |
| 3 | Employment | P0 | Employee code, staff type, department, designation, employment type, dates and status |
| 4 | Qualifications & Experience | P0 | Education, specialization, certifications, experience and previous employment |
| 5 | Documents | P0 | Staff documents, category, issue/expiry dates and verification status |
| 6 | Academic Assignments | P0 | Class-in-charge and subject teaching assignments with history |
| 7 | Account & Access | P0 | Login lifecycle, roles, delegated permissions and password reset |
| 8 | History & Activity | P0 | Employment, assignment and relevant audit/activity events |

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

Fields: Template (Required), Staff Selection (Required; one or many), Card Validity/Expiry (Optional), Show QR (Optional; only after secure verification endpoint design), Print Layout (Required). Include only approved public-facing identity fields. Never encode sensitive personal, login or permission data in a QR code.

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
3. Monthly register and reports. **Derived monthly register implemented; downloadable/report views pending.**
4. Connect approved leave and school calendar rules. **Pending Phase 4 leave/holiday contract.**

**Exit gate:** duplicate/date/permission safeguards, audited corrections, accurate reports and responsive browser coverage.

### Phase 4 — Leave & Holiday Management (P1)

1. Leave type foundation. **Implemented:** school-scoped leave type schema, permissions and create/list/update API; frontend create/list is available. Leave type edit/deactivate UI and explicit school policy configuration remain pending.
2. Staff leave application. **Implemented:** staff/leave-type selection, inclusive dates, full/half-day request, reason, request listing and filters. Backend rejects invalid/inactive targets and overlapping pending/approved requests; per-staff transaction advisory locking serializes overlap checks.
3. Approval/rejection. **Implemented:** pending-only decisions, authenticated decision actor/time, required rejection comment, applicant self-approval denial, cancellation and transactional audit. **Balance view pending:** quotas/accrual/carry-forward remain undefined until a school policy is approved; no balance is fabricated.
4. Holiday calendar. **Implemented:** school-scoped create/list/update/deactivate APIs and frontend calendar list with add/edit/deactivate actions. **Attendance integration pending:** approved leave and holidays are not yet displayed as attendance context and never auto-write attendance records.

**Exit gate remains open:** add dedicated PostgreSQL behavior tests for overlap/concurrent submissions, decision authorization, rollback and audit; finish leave-type policy/edit lifecycle; integrate approved leave/holiday context into attendance without silently changing manual marks; implement balances only after explicit policy approval.

### Phase 5 — Workload, Timetable & Substitution (P1)

1. Workload summary derived from existing timetable.
2. Teacher availability/free-period view.
3. Conflict warnings and workload reports.
4. Substitute assignment workflow.

**Exit gate:** no double-booking, timetable history preserved, substitutions audited and assignment limits enforced.

### Phase 6 — Reports, Import/Export & ID Cards (P1/P2)

1. Staff, employment, qualification, assignment, attendance and leave reports.
2. Validated bulk import with preview and row-level error report.
3. Permission-controlled exports.
4. ID card templates and bulk generation.
5. Document expiry reminders and verification workflow.

**Exit gate:** import preview has no writes; explicit confirmation; accurate row counts; sensitive exports restricted; ID cards use approved fields.

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
