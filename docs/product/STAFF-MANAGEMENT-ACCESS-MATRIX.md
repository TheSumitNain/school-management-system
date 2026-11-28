# Staff Management — Access, Page Visibility & Workflow Blueprint

**Status:** Batch 1 audit baseline — in progress  
**Branch:** `work` only. Never update `main`.  
**Last audited:** 2026-11-28  
**Scope:** School Admin, Teacher and Non-teaching Staff access to Staff Management; Platform Admin delegation controls that govern school permissions.  
**Deferred:** Staff document upload/storage remains out of current scope and is excluded from completion percentage.

> This is an evidence-based baseline from the current `work` branch. “Confirmed” means the current code explicitly implements the behaviour. “Needs verification” means the rule is not yet fully proven across UI, backend, data and real authenticated role sessions. Do not treat an intended role design as an implemented guarantee.

## 1. The access model — four separate concepts

| Concept | Meaning | Where it is currently controlled |
|---|---|---|
| Principal/account status | Whether the login principal can authenticate. Current staff account states include active and disabled. | Staff Management → Staff Directory account actions (disable / enable); backend account lifecycle endpoints. |
| Employment status | Whether the employee is Active or Inactive as an employee record. | Staff Management → Add/Edit Staff → Edit employee profile → Employment status. |
| Role assignment | Which school-scoped role bundles are assigned to a staff principal (e.g. Teacher, custom role). | Staff Management → open employee profile → Account & Access → Assigned roles; also role assignment modal from directory where available. |
| Individual permission override | A documented per-user ALLOW or DENY exception, only for a permission delegated for individual overrides. | Staff Management → open employee profile → Account & Access → Individual permission overrides. |

These concepts must not be conflated. Employment Inactive is not the same as login Disabled. Reactivating employment does not automatically enable a disabled login. Disabling an account revokes active sessions. Role changes and overrides are evaluated on protected requests; the UI message says changes take effect on the next protected request.

## 2. Top-level application navigation — current code gates

The shared shell builds navigation from the authenticated user's current permission list in `frontend/src/App.tsx`.

| Main page | Current navigation condition | What this proves / limitation |
|---|---|---|
| Dashboard | Always included for authenticated users. | Dashboard content still needs its own data/action permission checks. |
| Platform Admin | Principal must be `platform_admin` and have one of: `platform.admins.manage`, `platform.audit.read`, both `platform.permissions.delegate` + `platform.schools.read`, or both `platform.schools.create` + `platform.school_admins.create`. | Platform-only page; its tabs have separate permission gates. |
| Staff | Any one of `users.read`, `users.create`, `staff.attendance.read`, `staff.leave.read`, `staff.holidays.read`, `staff.holidays.manage`, `staff.substitutions.read`, `teacher_assignments.read`, `class_incharge_assignments.read`, `timetable.read`. | This is a broad entry gate, not full access to every Staff tab. Each tab/action must be checked separately. |
| Roles & permissions | `roles.read`. | Role creation and permission grants have stricter action permissions on the page/API. |
| Academics | Any one of `classes.read`, `academic_years.read`, `subjects.read`, `teacher_assignments.read`, `class_incharge_assignments.read`. | Individual academic workspaces have separate gates. |
| Students | Any one of `students.read_sensitive` or `students.read_basic`. | Data visibility and sensitive fields require additional checks. |
| Attendance | `attendance.read`. | Separate from Staff Attendance permission family. |
| Timetable | `timetable.read`. | Teacher data must also be restricted to assigned resources server-side. |
| Settings | Included for authenticated users. | Settings actions require their own controls/API authorization. |

A visible sidebar item is only a navigation decision. It is never a substitute for backend authorization.

## 3. Staff Management workspace — current known page gates

The Staff workspace uses a broad entry gate from the shell, then chooses an initial tab based on available permissions.

| Staff workspace tab | Current visibility/entry rule in `StaffManagementPage.tsx` | Notes |
|---|---|---|
| Overview | `users.read` OR `users.create`. | Create-only users may enter without directory read access; verify overview does not leak directory data. |
| Staff Directory | `users.read`. | Main employee list and profile entry point. |
| Academic Assignments | `teacher_assignments.read` OR `class_incharge_assignments.read`. | Separate subject and class-in-charge permissions are used within the assignment panel/API. |
| Attendance | `staff.attendance.read` (verify exact JSX and mutation gates in the next audit pass). | Do not infer write access from read access. |
| Leave | Leave/holiday read/manage permission family; exact per-tab predicate to be captured in the next audit pass. | Requests, approvals, leave types and holidays have separate action scopes. |
| Substitutions | `staff.substitutions.read` OR (`users.read` AND `staff.substitutions.manage`). | Exact actions still require backend authorization. |
| Reports | `users.read` for the staff report; additional operational reports can be exposed by their own read permissions. | Report fields/export must remain privacy-allowlisted. |
| Workload Report | Initial-tab fallback can use `timetable.read`; exact tab gate needs explicit verification. | School-wide vs own workload depends on role and scope. |
| ID Cards | Exact tab predicate and API gate need verification. | Do not assume `users.read` alone is sufficient until handler authorization is checked. |

**Audit requirement:** before declaring this matrix final, enumerate every JSX tab predicate and every corresponding backend route permission. Do not rely on this initial summary for unlisted conditions.

## 4. Staff profile tabs — current known behaviour

The profile is opened from Staff Directory and maintains a selected employee plus a selected profile tab.

| Profile tab | Current implementation evidence | Access/edit rule to verify |
|---|---|---|
| Overview | Shows employee code, staff type, designation, employment and login state. | Edit shortcut requires `users.update` and excludes protected School Admin accounts. |
| Personal Details | Shows name, employee code, mobile and email. | Edit requires `users.update`; protected School Admin excluded. |
| Employment | Shows staff type, designation, joining date, employment and login status. | Edit requires `users.update`; employment Active/Inactive is changed in Edit employee profile. |
| Qualifications & Experience | Loads qualifications, certifications and experience for selected employee. | Mutations require `users.update`; backend audit/tenant enforcement must be checked. |
| Documents | Placeholder only; secure upload is not enabled. | Deferred, excluded from current completion scope. |
| Academic Assignments | Uses selected employee's academic assignment panel. | Separate assignment read/manage permissions; teaching eligibility and same-school restrictions are backend enforced. |
| Account & Access | Renders role assignments and individual overrides. | Role list: `roles.read`; role assign/revoke: `roles.assign`; individual overrides: `permissions.override`. |
| History & Activity | Uses staff history panel. | Exact audit-read permission and field redaction need verification. |
| Workload & Timetable | Renders workload panel when `timetable.read` is present. | Backend restricts teacher's own/assigned data; verify role-specific output. |

Important current UI behaviour: when the user lacks `roles.read`, the Account & Access content displays “You do not have permission to view role assignments.” When the user lacks `permissions.override`, it displays “You do not have permission to manage individual permission overrides.” The profile tab itself is not proven to be hidden by these permissions.

## 5. Exactly where permissions are changed today

### A. Platform Admin — delegates permission capabilities to a school

**Screen:** Platform Admin → Delegate school permissions.

**Required page permissions:** `platform.permissions.delegate` and `platform.schools.read`.

Platform Admin chooses a school and a delegable permission, then chooses whether the School Admin may:
- assign that permission to custom school roles (`can_assign_to_roles`); and/or
- create individual staff overrides (`can_override_users`).

Platform Admin can revoke a delegation. This is the ceiling of what the school can grant; School Admin cannot self-create a delegation. Backend routes live under `/platform/schools/:schoolID/permission-delegations` and require `platform.permissions.delegate`.

### B. School Admin — builds reusable custom roles

**Screen:** Roles & permissions.

- View roles: `roles.read`.
- Create custom role: `roles.create_custom`.
- View permission catalogue: `permissions.read`.
- Grant/revoke a delegated permission to a custom role: both `permissions.assign` and `roles.update_custom`.
- Only active Platform Admin delegations with `can_assign_to_roles=true` can be used.
- Backend routes are under `/school/roles` and `/school/roles/:roleID/permissions`.

### C. School Admin — assigns roles to an individual employee

**Screen:** Staff Management → Staff Directory → open employee profile → Account & Access → Assign a role.

- View assigned roles: `roles.read`.
- Assign/revoke a role: `roles.assign`.
- Protected School Admin target accounts cannot be changed through this panel.
- Self role assignment/revocation is rejected in repository logic.
- Assignment is school-scoped and audited; backend checks actor authority and tenant membership.
- Default Teacher role provisioning for newly created teaching accounts is a separate onboarding rule; existing accounts are not automatically backfilled.

### D. School Admin — adds individual ALLOW/DENY exception

**Screen:** Staff Management → Staff Directory → open employee profile → Account & Access → Individual permission overrides.

- Requires `permissions.override`.
- Only active Platform Admin delegations with `can_override_users=true` and a delegable permission are offered.
- A reason is mandatory (maximum 1,000 characters).
- Effect is ALLOW or DENY; removal is separately confirmed.
- Protected School Admin target accounts and self-overrides are rejected.
- Changes are audited; backend enforces same-school target and delegation capability.

**Important:** individual override is an exception, not a replacement for role design. Prefer a custom role when the same permission set is needed by multiple staff members.

## 6. Active / Inactive / Disabled — do not mix the statuses

| Status/control | Where changed | Current known effect | Still needs proof |
|---|---|---|---|
| Employment Active / Inactive | Staff Directory → Edit employee profile → Employment status. | Inactive employees cannot sign in; UI explicitly says reactivating employment does not automatically enable a disabled login. Teaching-only workflows reject inactive employment. | Confirm every Staff and non-Staff endpoint consistently rejects inactive employment where required; verify reactivation workflow. |
| Login Enabled / Disabled | Staff Directory account actions: Disable account / Enable staff account. | Disable revokes active sessions. Enable requires a new temporary password and forces password change at next sign-in. | Confirm disabled account cannot use password login, refresh, old access token, MFA recovery and all APIs; confirm audit and UI filters. |
| Role assigned / revoked | Account & Access. | Changes affect effective access on next protected request. | Confirm stale frontend navigation/session permissions refresh after role change and logout/login behaviour. |
| Permission override added / removed | Account & Access. | Effective permissions recalculate on protected requests. | Confirm effective-permission cache/token behaviour and UI refresh after change. |
| Staff Type Teaching / Non-teaching / Unclassified | Add/Edit Staff profile. | Teaching classification is required for teacher-specific assignments; it does not itself grant Teacher role. | Confirm all teacher-only routes, existing-account onboarding and conversion impact. |

Current code distinguishes **employment status** (`staff_profiles.employment_status`) from **login account status** (`users.status` and `auth_principals.status`). Avoid calling both simply “Active/Inactive” in the UI or documentation.

## 7. Initial role-to-page design target — not yet an implementation claim

The product target is:
- **School Admin:** school-wide Staff Management according to explicitly delegated permissions.
- **Teacher:** teacher workspace and only assigned classes, subjects, students and own employment/self-service data. No school-wide Staff Directory, employee account management or permission administration by default.
- **Non-teaching Staff:** no Teacher role by default. A login and additional role should be granted only when the employee actually needs an approved system workflow. No broad teacher/academic permissions by default.
- **Platform Admin:** platform setup and school permission delegation, not routine school employee management.

The actual effective role bundles and landing pages for Teacher and Non-teaching staff must be derived from the seeded role-permission rows, auth response and live navigation. They are not yet certified by this document. Do not invent a default permission list.

## 8. Batch 1 audit gaps and next work

| ID | Gap / question | Priority | Evidence required |
|---|---|---:|---|
| ACCESS-01 | Capture every Staff workspace tab's exact JSX visibility condition and API route permissions. | P0 | Full source audit + page/route matrix. |
| ACCESS-02 | Confirm School Admin, Teacher and Non-teaching seeded role permission bundles from migrations/database fixtures. | P0 | Seed migration review + auth integration tests. |
| ACCESS-03 | Confirm where Teacher and Non-teaching land after login and which pages they see. | P0 | Auth/nav code + authenticated E2E for each role. |
| ACCESS-04 | Verify whether Account & Access tab is visible to users with only one of `roles.read` / `permissions.override`; avoid confusing partial access. | P0 | UX review + E2E. |
| ACCESS-05 | Audit disable/enable vs employment inactivation across authentication, refresh, API authorization and teacher assignments. | P0 | Repository + handler + PostgreSQL lifecycle tests. |
| ACCESS-06 | Verify role changes/overrides refresh effective permissions and navigation without stale access. | P0 | Auth/session/query cache audit + E2E. |
| ACCESS-07 | Audit protected School Admin handling, self-escalation, delegation ceiling, cross-school denial and audit rollback. | P0 | Existing tests + missing negative-path tests. |
| ACCESS-08 | Reconcile ID Cards, reports, History and Workload UI visibility with actual backend authorization. | P1 | Route-by-route permission matrix + tests. |
| ACCESS-09 | Verify non-teaching profile onboarding, optional login, and no implicit Teacher role for old accounts. | P1 | Onboarding tests + explicit migration/backfill decision. |
| ACCESS-10 | Produce screenshot evidence for the three roles at desktop and mobile sizes. | P1 | Authenticated browser acceptance. |

## 9. Working agreement and continuation protocol

Every batch must update this file and `docs/product/STAFF-MANAGEMENT-ROADMAP.md` (and `docs/project/STATUS.md` when project-wide status changes).

For each batch record:
1. Date, branch and commit SHA.
2. Scope completed and files changed.
3. Decisions made and why.
4. Frontend tests, backend tests, migrations, typecheck/build and CI evidence (never claim a test ran unless output/CI proves it).
5. Manual acceptance still pending.
6. Remaining gaps with stable IDs from this document.
7. Exact next batch.

Work only on `work`; never merge or push to `main` as part of this workflow. Do not ask the user to repeat decisions already documented. Continue in batches without pausing for routine approvals; pause only for a genuinely product/security-sensitive decision that cannot be safely inferred.

**Definition of 100%:** approved scope implemented; all role/page visibility rules mapped; UI and backend authorization match; relevant automated tests pass; tenant isolation and lifecycle paths are covered; authenticated manual acceptance evidence exists; documentation is current. Document upload is excluded until separately approved.


## 10. Verified default role bundles and role-specific navigation

### School Admin system role

The migration chain seeds an exact 51-permission school-scoped School Admin base bundle. Later staff-module migrations add another 12 permissions to the School Admin system role for existing and newly created schools:

- V58 staff attendance: `staff.attendance.read`, `staff.attendance.mark`, `staff.attendance.update_same_day`, `staff.attendance.correct_past` (4).
- V59 leave/holidays: `staff.leave.read`, `staff.leave.manage_types`, `staff.leave.manage_requests`, `staff.leave.decide`, `staff.holidays.read`, `staff.holidays.manage` (6).
- V60 substitutions: `staff.substitutions.read`, `staff.substitutions.manage` (2).

Therefore the currently approved School Admin system bundle is 63 permission keys (51 + 4 + 6 + 2), assuming the complete migration chain has been applied. V58/V59/V60 explicitly backfill existing schools through temporary-table trigger invocations; this is not limited to future schools.

### Teacher system role

The Teacher system role is asserted by the migration to have exactly these seven default permissions:

- `students.read_basic`
- `attendance.read`
- `attendance.mark`
- `attendance.update_same_day`
- `timetable.read`
- `homework.read`
- `notices.read`

It does **not** receive `users.read`, `users.create`, `roles.read`, `roles.assign`, staff attendance/leave/substitution permissions, or class/subject assignment-management permissions by default. Teacher resource endpoints still enforce assigned class/subject scope and active Teaching employment; role permissions alone do not grant school-wide data.

New Teaching staff onboarding assigns this Teacher role transactionally. New Non-teaching staff are not assigned the Teacher role. Existing accounts are not automatically backfilled.

### What each role currently sees in the shared navigation

- **School Admin:** Dashboard, Staff, Roles & permissions, Academics, Students, Attendance, Timetable, Settings (assuming the standard system role and no DENY override).
- **Teacher:** Dashboard, Staff, Students, Attendance, Timetable, Settings. The Staff entry appears because the Teacher bundle includes `timetable.read`; within Staff the initial tab becomes Workload Report. The separate Timetable page is also visible. Teacher does not get Staff Directory or employee-account controls by default.
- **Non-teaching staff with no additional role:** after the new onboarding behaviour, no default role is assigned. With no effective permissions, the shared shell shows Dashboard and Settings only. A School Admin must assign an approved custom/system role before other pages appear.

These are current code-derived navigation outcomes, not the final approved product UX. In particular, the Teacher seeing a top-level “Staff” page whose only available tab is “Workload Report” is confusing and should be reviewed in the role-aware navigation batch.

## 11. Additional verified gaps and constraints

| ID | Finding | Severity | Current evidence / consequence | Next action |
|---|---|---:|---|---|
| ACCESS-11 | Manual Teacher role assignment does not verify target staff type/employment. | P0 | The StaffRoleAssignments UI offers every role except School Admin; `AssignStaffRole` checks same-school active principal and role scope but does not check target `staff_profiles.staff_type='teaching'` and `employment_status='active'`. New onboarding correctly defaults Teacher only for Teaching staff, but manual assignment can bypass that rule. | Add UI filtering plus authoritative repository/database guard; preserve existing assignments and audit them rather than silently rewriting historical data. Add PostgreSQL and Playwright negative tests for Non-teaching, Unclassified and inactive targets. |
| ACCESS-12 | Teacher's Staff workspace entry is driven by `timetable.read`, and the tab is labelled Workload Report. | P1 | Teacher default bundle includes `timetable.read`; Staff nav therefore appears, initial tab becomes Workload Report, while the separate Timetable page also appears. Workload component heading currently says “School-wide Workload Report”; backend report scope must remain teacher-own for Teacher. | Pass explicit principal/role context or use server response scope to render an accurate title and navigation label; verify Teacher cannot see school-wide workload. |
| ACCESS-13 | Staff attendance, leave/holiday and substitution permissions are marked non-delegable in the permission catalogue. | P1 / product policy | V58, V59 and V60 create these permission keys with `is_delegable=FALSE`. They are added to the School Admin system role, but cannot be granted through custom-role delegation or individual overrides. | Keep School Admin-only unless the product explicitly approves additional operational roles; do not flip delegability without a permission-scope review. |
| ACCESS-14 | Account & Access profile tab is shown to every user who can open Staff profiles. | P1 | Profile tabs are always rendered (except Workload & Timetable, which requires `timetable.read`). Inside Account & Access, missing `roles.read` or `permissions.override` produces explanatory text rather than hiding each unavailable panel. | Decide whether a single visible tab with partial read-only messaging is desired, or hide inaccessible panels/tab; keep API checks authoritative. |
| ACCESS-15 | Workload Report UI copy may overstate the viewer's data scope. | P1 | Component heading and summaries say “School-wide”; the backend integration contract previously established School Admin school-wide vs Teacher own workload. | Add a role-scope-aware title, export labels and tests. |


### Additional exact UI rules captured during the same audit

- Staff workspace visibility: Overview = users.read OR users.create; Directory and ID Cards = users.read; Academic Assignments = teacher-assignment read OR class-in-charge read; Attendance = staff.attendance.read; Leave & Holidays = leave read OR holiday read OR holiday manage; Substitutions = substitution read OR (users.read AND substitution manage); Workload Report = timetable.read; Reports = users.read OR staff attendance read OR staff leave read OR teacher assignment read OR class-in-charge read.
- When the viewer has no leave/holiday read/manage permission, the workspace still renders a disabled “Leave — Restricted” tab with a tooltip; it is not an accessible Leave page.
- Profile navigation shows Overview, Personal Details, Employment, Qualifications & Experience, Documents, Academic Assignments, Account & Access and History & Activity to every Staff Directory reader. Workload & Timetable is additionally shown when timetable.read exists. Panels and actions inside these tabs have separate permission checks.
- Disable account action requires users.disable. Edit employment status requires users.update. The Enable account action is currently gated by users.update (not a distinct enable permission), plus a complete profile and Active employment. Reset password requires users.password_reset.
- Staff attendance, leave/holiday and substitution permissions are not delegable, so current custom-role and per-user override screens cannot grant those operations to a non-School-Admin employee.

## 12. Status update after source audit

- ACCESS-01: **UI tab predicates confirmed** for all nine workspace tabs; corresponding route-level permission and resource checks still need one-to-one reconciliation.
- ACCESS-02: **Default role bundles confirmed** from V4/V13/V58/V59/V60 migrations: School Admin 63 keys; Teacher exactly 7.
- ACCESS-03: **Shared navigation outcomes confirmed** for default School Admin, Teacher and role-less Non-teaching account; actual authenticated browser acceptance remains pending.
- ACCESS-04: **Confirmed** — Account & Access tab is visible to Staff Directory readers; role/override panels individually show permission guidance.
- ACCESS-05: **Database lifecycle confirmed in code** — setting employment inactive disables the principal, increments token version and revokes refresh sessions; active login cannot be restored until employment is Active. Live login/refresh/API E2E remains pending.
- ACCESS-06: **Partially verified** — role/override mutations invalidate relevant query caches and server checks are live; authenticated navigation refresh after role changes still needs E2E.
- ACCESS-11 through ACCESS-15: newly recorded findings above; none is marked fixed.

## 13. ACCESS-11 implementation checkpoint — 2026-11-28

**Implemented on work; automated CI is pending. Do not mark fixed/accepted until CI and local verification pass.**

- Added migration V73. A Teacher system role can only be assigned when the target has an active login and a same-school staff profile classified as Teaching with Active employment.
- Added a database guard that rejects changing a profile away from Teaching while an active Teacher role remains assigned. Required operator workflow: revoke Teacher role in Staff Directory → employee profile → Account & Access, then edit Staff Type. This prevents a dangling Teacher role during conversion to Non-teaching/Unclassified.
- Added repository pre-validation and a specific API 422 response for ineligible Teacher role assignment.
- Added a second effective-permission resolver guard: even if a legacy inconsistent Teacher role assignment exists, Teacher permissions are suppressed unless the profile is Teaching and Active. Existing role rows are not silently rewritten.
- Updated StaffRoleAssignments UI to omit Teacher from available roles for disabled accounts, non-Teaching profiles and inactive employment; it explains the eligibility requirement. Other custom roles remain available under existing delegation rules.
- Added PostgreSQL integration coverage for rejection on Non-teaching profile, successful Teaching assignment, permission suppression for a simulated legacy mismatch, blocking staff-type conversion until Teacher role revocation, and successful conversion after revocation.
- Added Playwright coverage confirming Teacher is not offered to an active Non-teaching employee while an eligible custom role remains assignable.
- Files changed: V73 up/down migration; role repository and integration test; effective-permission resolver; role handler; StaffRoleAssignments component; Staff Management Playwright spec.
- CI at checkpoint commit d38e4e23996a99ab98a86462e8cb2a8a7967bd29: Backend CI and Frontend CI started; results pending. No local test result is claimed.
- Next: inspect CI results; fix any failures; then run the focused PostgreSQL integration test and focused Staff Account & Access Playwright suite locally. After that audit the remaining permission management screen and lifecycle.

## 14. Exact permission-control screen gates

| Screen / control | UI permission requirements | Result |
|---|---|---|
| Sidebar: Roles & permissions | roles.read | Opens the school role-management page. |
| Role list | roles.read (page entry) | Lists system and custom roles. System roles are protected and cannot be edited here. |
| Create custom role | roles.create_custom | Opens the custom role creation dialog. |
| Read permission catalogue | permissions.read | Loads the permission descriptions for the role-management page. |
| Add/remove permission on custom role | permissions.assign AND roles.update_custom | Requires an active Platform Admin delegation with can_assign_to_roles=true for that exact permission. |
| Platform Admin → Delegate school permissions | platform.permissions.delegate AND platform.schools.read | Select active school and a delegable permission; independently choose custom-role assignment and/or individual override capability. |
| Staff profile → Assigned roles panel | roles.read | Panel is not rendered without roles.read. If roles.assign is absent, it is read-only. |
| Assign/remove staff role | roles.read (panel access) AND roles.assign (mutation) | Same-school active staff account; protected School Admin target and self-mutation are blocked. Teacher role also requires active Teaching profile under V73. |
| Staff profile → Individual permission overrides panel | permissions.override | This single permission currently controls both viewing and changing overrides; there is no separate read-only override permission. |

System roles are immutable in the School Admin Roles & permissions screen. To add extra access to one Teacher or Non-teaching employee, the School Admin creates/uses a custom role with delegated permissions and assigns it through Staff Profile → Account & Access. Individual ALLOW/DENY is for one-off exceptions and requires its own exact active delegation with can_override_users=true plus a reason.

New gap ACCESS-16: roles.assign alone does not expose the Staff Role Assignments panel because that panel is rendered only when roles.read is present; backend assignment authority is still separately checked. New gap ACCESS-17: permissions.override combines read and write, so a read-only auditor cannot inspect override history without also receiving override mutation capability. Review these as permission-model product decisions, not automatic code changes.

### Who can manage roles and permissions?

The V4 permission catalogue marks the following capabilities as non-delegable: roles.create_custom, roles.update_custom, roles.assign, permissions.assign and permissions.override. They are part of the School Admin system role, but Platform Admin cannot delegate them to custom roles or individual overrides. Therefore role creation, custom-role permission grants and individual overrides remain School Admin-only operations in the current model. The read capabilities roles.read and permissions.read are delegable, so a delegated staff user may be allowed to inspect roles/catalogue without changing them.

This is an intentional central-control boundary in the current code, not an implementation claim that every School Admin can delegate role administration to another staff member. If the school needs a separate HR/IT access manager, that requires a new narrowly scoped permission design and explicit Platform Admin policy decision.
