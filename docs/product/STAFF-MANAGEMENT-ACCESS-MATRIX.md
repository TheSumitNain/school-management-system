# Staff Management — Access, Page Visibility & Workflow Blueprint

**Status:** Batch 1 audit baseline — in progress  
**Branch:** `work` only. Never update `main`.  
**Last audited:** 2026-11-28  
**Scope:** School Admin, Teacher and Non-teaching Staff access to Staff Management; Platform Admin delegation controls that govern school permissions.  
**Deferred:** Staff document upload/storage remains out of current scope and is excluded from completion percentage.

> This is an evidence-based baseline from the current `work` branch. “Confirmed” means the current code explicitly implements the behaviour. “Needs verification” means the rule is not yet fully proven across UI, backend, data and real authenticated role sessions. Do not treat an intended role design as an implemented guarantee.

## Quick answer — default access and where to change it

This is the current code/migration-derived default, assuming the full migration chain is applied and no extra custom roles or overrides exist.

| User | What appears by default | What does not appear by default |
|---|---|---|
| School Admin | **Staff** workspace: Overview, Staff Directory, Academic Assignments, Staff Attendance, Leave & Holidays, Substitutions, Workload Report, Reports and ID Cards. Employee profiles expose all profile tabs; Account & Access panels are permission-gated. | Individual DENY overrides and incomplete prerequisites can still restrict actions. |
| Teacher | **My Workload** only inside the Staff workspace. Other shared pages: Dashboard, Students, student Attendance, Timetable and Settings. | Staff Directory, employee account actions, role/permission administration, staff attendance, leave/holidays, substitutions, ID Cards and school-wide workload. |
| New Non-teaching Staff with no additional role | Dashboard and Settings only. | Teacher pages and all Staff workspace pages until an approved role is assigned. |
| Platform Admin | Dashboard, Platform Admin and Settings. | School-level Staff workspace and employee profiles; platform admin delegates school permission ceilings instead. |

**Important:** Teacher's shared **Attendance** page is student attendance, not **Staff Management → Attendance**. Teacher's default timetable-read permission produces the **My Workload** navigation label; it does not grant Staff Directory access.

### Where each access change happens

1. **Platform Admin → Delegate school permissions:** choose the school and a delegable permission; set whether School Admin may grant it to custom roles (can_assign_to_roles) and/or individual users (can_override_users).
2. **School Admin → Roles & permissions:** create a custom role and add/remove only permissions covered by an active delegation.
3. **Staff Directory → open employee profile → Account & Access:** assign/revoke a role, or add/remove an individual ALLOW/DENY override with a reason.
4. **Staff Directory → Edit employee profile → Employment:** change employment Active/Inactive.
5. **Staff Directory → Disable account / Enable account:** control login access separately from employment status.

The full route, permission-key and lifecycle details follow below. See [STAFF-MANAGEMENT-PERMISSION-DICTIONARY.md](STAFF-MANAGEMENT-PERMISSION-DICTIONARY.md) for each key's meaning, delegability and granting screen. The canonical continuation record is this file plus STAFF-MANAGEMENT-ROADMAP.md and docs/project/STATUS.md.

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
- **Teacher:** Dashboard, My Workload, Students, Attendance, Timetable, Settings. The Staff workspace sidebar entry is labelled **My Workload** because the default Teacher bundle has timetable.read but no other Staff-management capability; its only accessible Staff workspace tab is My Workload. The separate Timetable page is also visible. Teacher does not get Staff Directory or employee-account controls by default.
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

## 15. ACCESS-12 and staff-type workflow UX implementation checkpoint — 2026-11-28

- The shared navigation now labels the timetable-only Staff entry as **My Workload** (with a timetable icon) when the user has no other Staff workspace permission. School Admin and operational staff retain **Staff**.
- The Staff workspace page title, description and tab label switch to My Workload for timetable-only users. The workload report uses the non-delegable users.read capability to distinguish School Admin school-wide access from a teacher's self-only scope. Teacher sees “My Workload”, “My weekly periods” and “Days with lessons”; staff search and employment filters are hidden. CSV export is labelled and named as personal workload.
- Backend resource scope remains authoritative; this is a UI clarity change and does not widen the Teacher workload API scope.
- Staff Type create/edit helper text now matches onboarding: new Teaching employees automatically receive the standard Teacher role and require roles.assign; new Non-teaching employees receive no default role; editing staff type does not auto-add/remove roles.
- A staff-type conversion blocked by an active Teacher role is mapped to HTTP 409 with a clear instruction to revoke the Teacher role at Staff Directory → Employee Profile → Account & Access first.
- Added Playwright assertions for Teacher-only My Workload navigation, personal summary/export and the new Staff Type guidance/error path.
- Backend integration fixtures that create Teacher role assignments were updated to create/mark Teaching profiles before assigning the role. Legacy unclassified Teacher access assertions now expect explicit authorization denial because the effective permission resolver suppresses the whole Teacher bundle for non-Teaching profiles.
- Verification is pending on latest code commits: Backend CI and Frontend CI must both pass; local Windows test commands still required from the user. Do not mark ACCESS-11/12 as accepted until results are green.

## 16. Teacher and Non-teaching experience — pages that are still missing

### Teacher — actual current experience

With the default seven-key Teacher role and no additional custom role, the shared shell exposes:

- Dashboard — currently generic; it does not yet provide a complete Teacher Today experience and may show no overview metrics.
- My Workload — personal recurring timetable workload only.
- Students — basic student access, limited by server-side assigned class/subject scope.
- Attendance — read/mark/update within the Teacher's authorized attendance scope.
- Timetable — own assigned timetable entries.
- Settings — shared account/settings surface.

Teacher does **not** see Staff Directory, staff-wide reports, ID Cards, Roles & permissions, Academic Assignments management, Staff Attendance, Leave & Holidays, Substitutions or school-wide Workload Report by default. The Teacher system role does include homework.read and notices.read, but App.tsx currently has no Homework or Notices navigation/page entry, so those permissions are not surfaced in the current UI.

There is also no separate My Profile page for a Teacher to view/update their own employment profile, and no Teacher self-service leave request workflow in the current permission catalogue/UI. The Staff Leave workspace is an admin operations workspace gated by staff.leave.read / holiday permissions; Teacher has none of those. These are product gaps, not permissions that should be granted blindly.

### Non-teaching — actual current experience

New Non-teaching employees receive no default system role. After login, with no effective permissions, they see only shared Dashboard and Settings. They do not get Teacher pages, Staff Directory, My Profile, leave request, or any operational page unless an approved custom role is assigned. Current code does not include a dedicated Non-teaching self-service role or dashboard.

### Follow-up product gaps

| ID | Gap | Why it matters | Next action |
|---|---|---|---|
| ACCESS-18 | Teacher dashboard is generic; Homework/Notices permission keys have no matching app pages; no self-profile or self-service leave workflow. | The Teacher role has permissions that the UI does not surface, while important teacher tasks are absent. | Define Teacher home, homework/notices, own profile and leave request scope before adding more permissions. Build role-specific UX and tests after School Admin Account & Access is accepted. |
| ACCESS-19 | Non-teaching accounts have no default role or self-service experience. | A new Non-teaching login has no useful pages beyond shared Dashboard/Settings. | Decide whether non-teaching users need a limited base role (e.g. own profile and leave request) or remain login-disabled/role-less until assigned. Do not grant Teacher role. |


## 17. Directory status visibility detail

- Staff Directory's main table currently displays the employee's **Employment** status (Active/Inactive). The toolbar separately offers a **Login account** filter (Enabled/Disabled), and the employee profile Overview shows both Employment and Login status. The directory table itself does not currently have a dedicated Login status column.
- Disable account row action is shown for active logins when the viewer has users.disable. Enable account row action is shown for disabled logins only when the employee profile is complete, employment is Active, and the viewer has users.update. Password reset is separate and requires users.password_reset.
- Employment status is changed from Staff Directory → Edit employee profile → Employment status (users.update). Moving to Inactive disables the login and revokes sessions; moving back to Active does not automatically enable a disabled login.

New gap ACCESS-20: Directory table makes Employment status visible but not Login status; the separate filter and profile overview expose it. Consider a dedicated Login column so School Admin can distinguish an employed employee whose login is disabled from an employee whose employment is inactive without opening each profile.

## 18. Batch 1 continuation checkpoint — 2026-11-28

**Audited branch head:** `work` at `7c697f205bb68c9cdab07f9afcc5109b9e986268`. `main` was not changed.

### Confirmed UI entry points

- Shared sidebar: `App.tsx` exposes the Staff entry when the user has at least one of `users.read`, `users.create`, `staff.attendance.read`, `staff.leave.read`, `staff.holidays.read`, `staff.holidays.manage`, `staff.substitutions.read`, `teacher_assignments.read`, `class_incharge_assignments.read`, or `timetable.read`. It labels the entry **Staff** when any non-workload staff capability is present; otherwise **My Workload**.
- Staff workspace tabs and their exact predicates are recorded in sections 3 and 11. A restricted Leave tab is rendered disabled as an explanatory affordance; it is not an accessible workspace.
- Employee profile tabs are shown to Staff Directory readers, with Workload & Timetable additionally gated by `timetable.read`. Account & Access itself remains visible; each internal panel has its own gate and explanatory message.
- Role assignments can be opened from the employee profile's Account & Access tab and from the directory's standalone Access roles action where rendered. Individual overrides can be opened from Account & Access and the directory's Permission overrides action where rendered.
- Inside Account & Access: role list requires `roles.read`; role mutations additionally require `roles.assign`. Individual override panel requires `permissions.override` for both listing and mutation. This confirms gaps ACCESS-16 and ACCESS-17 remain permission-model limitations, not hidden UI bugs.
- Employment status is changed in Edit employee profile; login disable/enable are separate account actions. The current UI gates disable with `users.disable`, enable with `users.update`, and reset password with `users.password_reset`.

### Verified default navigation outcomes (code + migration-derived; live login acceptance pending)

- School Admin system role: 63 school permissions after V4 base 51 plus V58/V59/V60 staff-module additions, assuming all migrations are applied.
- Teacher system role: exactly seven keys listed in section 10. It sees Dashboard, My Workload, Students, Attendance, Timetable and Settings in the current shared shell. Resource APIs still enforce assignment scope.
- New Non-teaching staff: no default role; with no additional role, only Dashboard and Settings are available.
- These are code/migration-derived outcomes. Do not claim live authenticated role acceptance until separately tested.

### Batch 1 disposition

- ACCESS-01: Staff workspace JSX visibility predicates captured; route-by-route backend authorization reconciliation remains open.
- ACCESS-02: default role bundles captured from migrations; verify actual target DB migration state during local acceptance.
- ACCESS-03: shared navigation outcomes captured; authenticated School Admin, Teacher and Non-teaching browser acceptance remains open.
- ACCESS-04: Account & Access tab and internal panel visibility confirmed.
- ACCESS-05: employment inactivation and login disable distinction documented; end-to-end password, refresh, stale token and protected API lifecycle acceptance remains open.
- ACCESS-06: mutation query invalidation is present; live session/navigation permission refresh after role or override changes remains open.
- ACCESS-07: protected School Admin/self-mutation/delegation/tenant controls have repository coverage in places; full negative-path inventory and test reconciliation remain open.
- ACCESS-08: ID Cards, reports, History and Workload API route-to-UI authorization reconciliation remains open.
- ACCESS-11/12: code changes are present in this branch; check current CI and local Windows test results before treating them as accepted.
- ACCESS-13 through ACCESS-20: product constraints and UX gaps remain tracked; no implicit permission broadening is approved.

### Next batch — Account & Access and permission-control reconciliation

1. Inspect Roles & permissions UI, Platform Admin delegation UI, role/override APIs and repository authorization as one end-to-end workflow.
2. Build a permission dictionary: key, plain-language meaning, page/tab, read/action level, delegable flag, granting screen, revoking screen and target resource scope.
3. Verify School Admin authority ceiling, custom role grants, individual ALLOW/DENY, protected School Admin targets, self-escalation and cross-school denial.
4. Add or repair authenticated Playwright + PostgreSQL regression tests for the gaps found; do not change policy or widen default roles silently.
5. Update this matrix, the roadmap and project status with exact commit and verified test evidence before moving to the next page.

**Continuation rule:** In a new chat, read this matrix and the latest roadmap/status checkpoint first. Continue on `work` only, in batches, without asking the user to repeat these decisions. Do not mark any page 100% without automated and authenticated manual acceptance. Document upload remains deferred and excluded.

## 19. Batch 2 — permission control to backend route reconciliation

### Permission-control screens and backend endpoints

| User goal | Screen | UI capability | Backend route / guard |
|---|---|---|---|
| Open role list | School Admin → Roles & permissions | `roles.read` | `GET /school/roles` — `roles.read` |
| Create reusable custom role | Roles & permissions → Create role | `roles.create_custom` | `POST /school/roles` — `roles.create_custom`; repository requires School Admin authority |
| Read permission catalogue | Roles & permissions | `permissions.read` | `GET /school/permissions` — `permissions.read` OR `permissions.override` |
| Add/remove permission on a custom role | Roles & permissions → select custom role → Manage permissions | `permissions.assign` AND `roles.update_custom`; exact active delegation must allow `can_assign_to_roles` | `POST/DELETE /school/roles/:roleID/permissions...` — both permissions; repository validates exact delegation and school |
| Delegate permission capability to a school | Platform Admin → Delegate school permissions | Platform Admin: `platform.permissions.delegate` (and school list access) | `GET/POST /platform/schools/:schoolID/permission-delegations`, `GET /platform/permissions/delegable`, `POST /platform/permission-delegations/:delegationID/revoke` |
| View one employee's assigned roles | Staff Directory → employee → Account & Access (or Manage roles dialog) | `roles.read` | `GET /school/staff/:principalID/roles` — `roles.read` |
| Assign/revoke one employee role | Same Account & Access panel/dialog | `roles.read` to load panel + `roles.assign` to mutate | `POST /school/staff/:principalID/roles/:roleID` and matching DELETE route — `roles.assign`; repository enforces School Admin authority, same-school target, no self-mutation, protected School Admin and Teacher eligibility |
| View/set/remove one employee ALLOW/DENY override | Staff Directory → employee → Account & Access (or Permission overrides dialog) | `permissions.override` controls both viewing and mutation | `GET/POST /school/staff/:principalID/permission-overrides` and DELETE route — `permissions.override`; exact active delegation with `can_override_users`, reason and same-school target required |

**Important distinction:** Platform Admin delegates *which permission keys and which grant mechanisms* a school may use. School Admin then either adds those keys to a reusable custom role or grants a per-person exception. Assigning a role to an employee is a separate action from editing the permissions inside that role.

### Current automated coverage located in source

- Frontend Playwright contains focused cases for role read-only access, protected peer School Admin accounts, role assign/revoke with confirmation and pending-request suppression, Teacher role exclusion for active Non-teaching staff, override delegation/reason/removal confirmation, and standalone access dialogs refreshing data on reopen.
- Backend PostgreSQL repository integration covers custom role and permission grants/revokes, Platform Admin delegation grant/revoke, staff role assignment/revocation, ALLOW/DENY overrides, self-escalation rejection, cross-school target denial, and audit-write rollback.
- Real HTTP/PostgreSQL RBAC integration exercises permission override changes taking effect on the next protected request and cross-school denial paths.
- These tests are present in the branch. This source audit does not itself execute them; local Windows results and current Frontend CI evidence are still required.

### Permission model constraints confirmed

- `roles.assign`, `roles.create_custom`, `roles.update_custom`, `permissions.assign` and `permissions.override` are non-delegable in the current catalogue. Role administration and individual overrides therefore remain School Admin-only.
- `permissions.read` and `roles.read` are separate from mutation permissions.
- ACCESS-16 remains: `roles.assign` without `roles.read` cannot use the UI panel because the role list must be read first. Current product-safe rule is to require both for the visible assign/revoke workflow; do not silently add a broad role-read grant.
- ACCESS-17 remains: `permissions.override` combines read and write; there is no separate read-only permission to inspect overrides.
- Staff operational permissions V58–V60 are non-delegable. Do not make them delegable as a shortcut to giving operational staff access.

### Batch 2 status

- UI-to-route mapping for role list, custom-role creation/grants, Platform Admin delegation, staff role assignment and per-user overrides is now documented.
- Existing automated test coverage has been located; execution status is not inferred from test presence.
- Remaining checks: exact actor-authority SQL for each mutation; test assertions for exact delegation revocation, self-mutation and audit rollback; frontend CI and local focused suite on the current branch; stale authenticated navigation after effective permission changes.
- Next: complete the permission-key dictionary and inspect employment/login lifecycle end-to-end, then run the focused local Account & Access and lifecycle test batches.

## 20. Batch 3 — employment and login lifecycle audit

### Current states and exact transitions

| State | Where changed | Server-side effect | Re-enable path |
|---|---|---|---|
| Employment Active → Inactive | Staff Directory → Edit employee profile → Employment status (`users.update`) | V28/V29 database trigger disables the canonical staff principal, increments `token_version`, and revokes active refresh sessions in the same transaction. User status is synchronized from principal status. | Edit employment back to Active first; this does **not** enable login. |
| Login Enabled → Disabled | Staff Directory → Disable account action (`users.disable`) | Repository disables principal (user status syncs), increments `token_version`, revokes refresh sessions and writes an audit event in the same transaction. Self-disable and protected School Admin targets are rejected. Active substitution coverage can block the operation. | Use Enable staff account only after employment is Active. |
| Login Disabled → Enabled | Staff Directory → Enable account action (`users.update`) | Requires disabled account, complete same-school profile, Active employment and a new temporary password. Principal is reactivated, `token_version` increments, password is replaced, `must_change_password=TRUE`, stale sessions are revoked and an audit event is written atomically. | Staff member must change the temporary password at next sign-in. |
| Password reset | Staff Directory → Reset password (`users.password_reset`) | Active same-school staff account only; new temporary password, `must_change_password=TRUE`, token-version increment, refresh-session revocation and audit are transactional. | Staff member changes password at next sign-in. |

The database trigger also disables a profile inserted directly as Inactive. A separate database guard rejects attempts to activate a principal while the employment profile is still Inactive. These protections apply beyond the Staff Management UI.

### What “Active / Inactive” does not mean

- Current employment status supports only `active` and `inactive`.
- Current login status supports `active` and `disabled`.
- There is no distinct persisted `Suspended`, `Relieved`, `Archived`, `Resigned` or `Terminated` employment state in this workflow.
- Do not hard-delete staff to simulate exit; preserve historical attendance, leave, academic assignments, substitutions and audit events.
- Product roadmap language about relieving/archiving is a future lifecycle target, not a currently implemented status.

**New gap ACCESS-21 (P1):** define a formal employee exit workflow (relieving date, reason, final working day, archive/read-only treatment and handling of active assignments/leave/substitutions) before adding a new employment status or database migration. Keep this separate from the immediate Active/Inactive + Login Enable/Disable behaviour.

### Lifecycle test evidence located

- Frontend Playwright cases exist for disable confirmation, enable with temporary password, duplicate-submit suppression, inactive-employment re-enable conflict, employment-status edit, and protected School Admin lifecycle actions.
- PostgreSQL lifecycle integration tests cover disable/enable status, token version, refresh-session revocation, inactive-employment re-enable rejection, audit-write rollback and retry behaviour.
- V28/V29 migration triggers enforce the employment/login relationship at the database boundary.
- Test definitions and migration code were inspected; this audit did not execute them locally.

### CI checkpoint discovered during audit

- Backend CI run [46113123536](https://github.com/TheSumitNain/school-management-system/actions/runs/46113123536) **PASS** on code commit `a73f16c2c750fae9d18f2601e0f6766c553f8fd1`.
- Frontend CI run [46113046959](https://github.com/TheSumitNain/school-management-system/actions/runs/46113046959) **FAIL** on `25d39647187bdf66aa7724ccf4985e1ffe7d48c7`: Vite production build passed, but Playwright reported 19 failures, 173 passes and 1 flaky test. Failures included Staff disable/enable and Account & Access tests timing out before the expected employee row/profile was available; unrelated Platform Admin, Student and Teacher Assignment tests also failed. A separate Roles & permissions test had an ambiguous button locator. This is a real red Frontend CI checkpoint, not a passing result.
- The diff from `25d3964` to `a73f16c` changes only `backend/internal/repository/user_repository.go`; therefore the frontend test source at the failing checkpoint is unchanged by that later backend commit. No green Frontend CI run was found for `a73f16c`.
- Local Windows evidence currently available in this conversation: frontend typecheck passed; Academic Assignments Playwright 6/6 passed; full backend tests and `go vet` passed. The focused Account & Access/lifecycle Playwright suite has not yet been confirmed locally.

### Next batch

1. Reproduce the focused Staff Account & Access and disable/enable tests locally on the current `work` head.
2. Inspect the first failed network request and test fixture/route handling; do not increase timeouts or weaken assertions to hide the issue.
3. Fix the root cause, then rerun the focused group and the full Frontend CI suite.
4. Record the exact passing commit and test output. Keep ACCESS-21 as a product-design gap; do not add a new status field/migration without a complete exit workflow.

## 21. Batch 4 — Staff Account & Access and lifecycle test reconciliation

**Code checkpoint:** fbcfd8c76895cb193c92f5d6aa6dd567c89cdf44 on work; main untouched.

### Changes made

- Corrected Staff Account & Access Playwright locators to use the actual employee-profile accessible button name (View profile for username) and the visible employee code in directory rows. The directory table shows employee name, employee code and employment status; it does not show the login username as row text or a separate login-status column.
- Corrected lifecycle assertions to test the visible employment label and the Enable/Disable action state rather than expecting login status text in the Employment column. The Enable test fixture includes users.disable so it can verify the post-enable Disable action.
- Updated the timetable-only Teacher test to use the current My Workload sidebar/tab and heading, rather than the old Staff / Workload Report / School-wide labels.
- Fixed the Academic Assignments workspace so a failed subject-assignment list load disables the Assign subject teacher action, and a failed class-in-charge list load disables the Assign class in-charge action. This prevents writes when the current assignment state cannot be verified.

### CI state at this checkpoint

- Backend CI [46114885133](https://github.com/TheSumitNain/school-management-system/actions/runs/46114885133): **PASS** — complete migration chain, Go tests, PostgreSQL integration and vet.
- Frontend CI [46114885123](https://github.com/TheSumitNain/school-management-system/actions/runs/46114885123): **IN PROGRESS** when this checkpoint was written. TypeScript/Vite production build and Chromium installation had passed; full Playwright suite was still running. Do not infer its final result.
- The prior Frontend CI run [46114254986](https://github.com/TheSumitNain/school-management-system/actions/runs/46114254986) had 177 passing, 15 failing and 1 flaky test. Account & Access role/override profile-navigation locator failures were resolved in the current checkpoint; the current run is the acceptance gate for the lifecycle/workload/assignment fixes.
- Local Windows focused Account & Access/lifecycle tests and authenticated School Admin acceptance are still required.

### Next batch

1. Read the final result of Frontend CI [46114885123] and inspect any remaining Staff-related failures before changing anything else.
2. Run the focused local Windows command from the latest roadmap/status checkpoint.
3. Complete authenticated School Admin acceptance for role assign/revoke, ALLOW/DENY override, protected School Admin, Teacher eligibility, employment Active/Inactive and login Enable/Disable.
4. Only after those pass, mark Account & Access accepted and move to the next unfinished page. Do not declare the whole Staff Management page 100%.

### Page-level reporting rule

Keep previously accepted scope and pending manual acceptance separate. Staff Dashboard and Staff Directory were previously accepted for their current planned scope (document upload excluded). Other modules with green automated suites remain pending their recorded authenticated manual sign-off. Do not publish a single overall Staff Management percentage until the role/page matrix, all page workflows and manual acceptance evidence are reconciled.

## 22. Open product decisions — delegated HR access and Teacher self-service

| ID | Gap | Why the current user cannot see/use it | Safe future design; no implementation approved yet |
|---|---|---|---|
| ACCESS-22 (P1) | No delegated HR / Staff Manager role exists. | users.read and all staff attendance, leave/holiday and substitution permissions are non-delegable. A Non-teaching employee cannot be given Staff Directory or staff operations through a custom role. | If the school needs an HR/Office Manager, add dedicated staff-directory read/create/update permissions and carefully scoped operational permissions. Define whether the manager can see all employees or only assigned groups; add same-school and record-scope enforcement before making keys delegable. |
| ACCESS-23 (P1) | Teacher self-service staff attendance and leave are not implemented. | staff.attendance.* and staff.leave.* are non-delegable admin permissions; there are no separate self-only keys for viewing one's own staff attendance or requesting/cancelling one's own leave. | If required, introduce distinct self-service permissions and endpoints (own record only), separate from School Admin manage/decide permissions. Test that Teachers cannot view or change another employee's records. |

These are product/security decisions, not quick permission toggles. Keep the current School Admin-only policy until the user explicitly approves a scoped HR Manager or Teacher self-service design. Do not change V4/V58/V59/V60 delegability flags as a shortcut.

## 23. Current code checkpoint — 2026-11-28

**Code commit:** 9506a04ef00a63848d04d2781ee9f1841cbe94e8 on branch work. Latest documentation-only checkpoint is cd9013544951610f0fda8e41caf5b58bba360810. main remains unchanged.

### Latest changes after Batch 4

- Staff Account & Access and login lifecycle E2E tests now use the actual profile accessible labels, visible employee codes and the actual Enable/Disable action state.
- Employment Active/Inactive and login Enable/Disable are tested as separate states; inactive employment reactivation alone does not enable login.
- DatePicker decade navigation now re-runs the selected-year focus effect when the displayed year changes.
- Teacher-only navigation is My Workload; School Admin navigation remains Staff. The workload-only test now distinguishes the page H1 from the repeated section H2.
- Role Management and Platform Admin E2E navigation now scopes clicks to the sidebar nav item rather than also matching the dashboard shortcut button.
- Academic Assignment create actions remain disabled if the assignment list needed to validate the current state fails to load.

### CI evidence

- Backend CI [46115928661](https://github.com/TheSumitNain/school-management-system/actions/runs/46115928661): **PASS**.
- Frontend CI [46115928660](https://github.com/TheSumitNain/school-management-system/actions/runs/46115928660): **IN PROGRESS** at the time of this note. Build and Chromium setup passed; Playwright suite was running. Read its final result before accepting the batch.
- Previous Frontend CI [46115486523](https://github.com/TheSumitNain/school-management-system/actions/runs/46115486523): 182 passed, 10 failed, 1 flaky. Staff lifecycle, Staff Profile qualification/certification/experience and Academic Assignment load-error tests passed after fixes; remaining failures included Platform Admin navigation, workload heading, and Student Management. The current 9506 run contains fixes for the Platform Admin and workload-heading failures. Student Management failures remain outside this Staff Management batch.

### Next action

1. Verify Frontend CI [46115928660] and identify any remaining Staff/Role/Platform permission test failures.
2. Run the focused Windows command in docs/project/STATUS.md on the user's local database/browser.
3. Complete authenticated School Admin acceptance for Account & Access and employment/login lifecycle.
4. Only then close this page and proceed to the next Staff Management page. Do not report overall percentage or 100% without the acceptance evidence.


## 24. Responsive Account & Access UI batch — 2026-11-28

### Implemented on `work` (verification pending)

- Shared shell sidebar breakpoint is 900px: fixed sidebar above 900px; hamburger-controlled drawer at 900px and below. The independent Dashboard content layout breakpoint remains 740px.
- Staff Directory row actions switch to a three-dot menu at 520px and below. The menu contains only actions allowed by the current principal's permissions and the target account's state; protected School Admin targets do not receive edit/reset/disable actions. Desktop action icons remain unchanged.
- The menu is portalled to `document.body` so it is not clipped by the horizontally scrollable table. It closes on outside pointer interaction, Escape, resize and scroll.
- Individual permission Effect control is disabled when `availablePermissions.length === 0`, matching the delegated-permission picker and Save button. Backend delegation checks remain authoritative.

### Regression coverage added

- Mobile directory action menu exposes the expected permitted actions, closes on Escape, and opens the staff profile from the menu.
- With no Platform Admin individual-override delegation, delegated permission, Effect and Save controls are all disabled.
- Workspace navigation drawer is exercised at 768px and 900px; fixed sidebar is exercised at 901px.

### Acceptance status

- Code checkpoint before the docs update: `613016782d95b32516511963f774fac2efcb11f8`. Follow-up CSS explicitly hides the mobile trigger above 520px; Playwright asserts desktop-hidden/mobile-visible behaviour.
- Frontend CI [46146670940](https://github.com/TheSumitNain/school-management-system/actions/runs/46146670940) and Backend CI [46146671047](https://github.com/TheSumitNain/school-management-system/actions/runs/46146671047) were pending when recorded. Local Windows verification is pending.
- User has manually confirmed mobile bottom-sheet modal and centred tablet modal behaviour. New mobile action menu and 768px/900px sidebar changes are awaiting local acceptance.
- Account & Access remains in progress; no 100% completion claim. Continue on this page until role-wise, responsive, automated and local acceptance gates are met.

## 20. History & Activity — current UI contract and open backend audit

- Profile tab is visible in the Staff profile tab list; the panel itself requires school.audit.read. Without that permission, it displays an access-denied message and must not issue the staff audit-events request.
- The panel requests 50 events per page using limit=50 and offset pagination. has_more controls Older events; Newer events returns to the previous offset. Refresh and retry are explicit actions.
- The UI currently renders only an action label, actor username, outcome and IST-formatted timestamp. It does not intentionally render event metadata, free-text reasons, credentials or other payload fields. Unknown action keys use a humanized fallback label.
- Existing browser coverage verifies that leave decisions/creation/cancellation and role assignment/removal cause the employee audit history to be fetched again when the profile is reopened or the History & Activity tab is revisited.
- New Playwright coverage added on 2026-11-28 verifies the no-permission/no-request boundary, 50-row pagination with safe-summary-only rendering, and retry after a failed audit request. These tests are pending local Windows and CI execution at commit 20a704be5d5fbd8afb4e3e20817f053f7429db69.
- **Backend verification still required:** inspect the exact audit-events route, handler, service and repository to prove same-school target isolation, server-side school.audit.read enforcement, metadata exclusion from the response, bounded limit/offset validation, deterministic newest-first ordering with a unique tie-breaker, and correct has_more calculation. Frontend type definitions alone do not prove the server response is safe.
- Do not mark History & Activity complete until backend integration/authorization tests, Playwright, responsive acceptance and authenticated School Admin role checks pass. Continue this tab only; do not jump to another Staff page.

### Backend source audit follow-up (2026-11-28)

- Route registration in backend/cmd/api/main.go applies authentication middleware and RequirePermission("school.audit.read") before StaffRecordsHandler.ListAuditEvents.
- The handler independently validates authenticated actor/school context, positive target principal ID, limit 1–100 (default 50), and offset 0–1,000,000 (default 0); it returns an empty event array for a nil page.
- The repository repeats the school.audit.read authorization check, confirms the target is a staff principal belonging to the actor's school (including legacy accounts without a staff_profiles row), scopes events by audit_events.school_id, and selects only id/action/resource_type/outcome/actor username/created_at. Event metadata is not selected into the response.
- Event ordering is deterministic: created_at DESC, id DESC. The query fetches limit+1 rows and derives has_more from the extra row.
- This closes the source-code review items for permission, tenant scoping, response allowlist, pagination bounds, ordering and has_more. PostgreSQL integration coverage for same-school/cross-school target access and exact pagination behaviour must still be confirmed from test output; source inspection is not a substitute for test execution.

### Retry regression correction (2026-11-28)

- The shared QueryClient retry policy is one automatic retry for network/5xx failures. The earlier E2E fixture/assertion changes still failed locally because the error state did not become visible reliably in that scenario; no production defect has been established.
- The manual retry test is now deliberately isolated from automatic retry behavior: the first request returns a non-retryable 400, which must expose the error state; clicking **Try again** then returns a successful audit page. The test asserts exactly one initial request and one explicit retry.
- This verifies the user-visible manual recovery control deterministically without duplicating the global React Query retry-policy test. Local focused/full-suite verification is pending.

History & Activity retry regression note (2026-11-28): the Playwright retry test now simulates a network failure by aborting the initial audit request and the single configured automatic network retry, then verifies that the visible manual Try again action recovers successfully. This avoids relying on mocked HTTP 400 behavior while exercising the actual React Query network retry policy. The profile remains pending final local focused/full-suite and backend verification.
