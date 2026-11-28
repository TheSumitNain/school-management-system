# Staff Management Permission Dictionary

**Source:** V4, V58, V59 and V60 permission catalogue migrations, current App.tsx navigation, StaffManagementPage.tsx, RoleManagementPage.tsx, PlatformAdminPage.tsx and backend route/repository authorization.

This document answers four questions for every permission: what it means, which UI it controls, whether it can be delegated, and where an administrator changes it. “Delegable” means Platform Admin may allow School Admin to grant that permission to a custom role or to one user through an individual override. It does not mean the permission is automatically granted.

## 1. Staff account and profile permissions

| Key | Meaning / UI effect | Delegable? | Where granted or changed |
|---|---|---:|---|
| users.read | Open Staff workspace as Staff, view Staff Directory, employee profiles, ID Cards and profile-based reports. Also distinguishes School Admin school-wide workload scope. | No | School Admin system role only in current policy. |
| users.create | Show Overview and Add staff onboarding. | No | School Admin system role only. |
| users.update | Edit employee profile, change employment Active/Inactive, and enable a disabled login after profile/employment checks. | No | School Admin system role only. |
| users.disable | Disable an employee login. Does not mean employment is Inactive. | No | School Admin system role only. |
| users.password_reset | Reset an active same-school staff account password. | No | School Admin system role only. |

**Important:** users.read is non-delegable. A custom role cannot be used to give a Non-teaching employee access to the Staff Directory in the current permission model.

## 2. Roles, custom roles and individual overrides

| Key | Meaning / UI effect | Delegable? | Where granted or changed |
|---|---|---:|---|
| roles.read | Read school roles and permission assignments. | Yes | Platform Admin may delegate it; School Admin may include it in a custom role. Staff profile access still separately requires users.read. |
| roles.create_custom | Create a reusable school-scoped custom role. | No | School Admin → Roles & permissions. |
| roles.update_custom | Change a custom role's name/description and permission set. | No | School Admin → Roles & permissions. |
| roles.assign | Assign or revoke a permitted role for an employee. | No | Staff Directory → employee profile → Account & Access. |
| permissions.read | Read the permission catalogue and descriptions. | Yes | Platform Admin delegation → School Admin custom role. |
| permissions.assign | Add/remove a delegated permission on a custom role. | No | School Admin → Roles & permissions. The permission being added must separately be delegable with can_assign_to_roles=true. |
| permissions.override | Use the per-employee ALLOW/DENY override control. This capability itself is not delegated. | No | Staff Directory → employee profile → Account & Access. The target permission must separately be delegable with can_override_users=true; a reason is required. |
| platform.permissions.delegate | Set/revoke the permission delegation ceiling for a school. | No (platform scope) | Platform Admin → Delegate school permissions. |

**Do not confuse the permission being managed with the capability used to manage it.** Platform Admin delegates a target permission key and its grant mechanisms. School Admin uses the non-delegable permissions.assign or permissions.override capability to apply that delegation. roles.assign is a separate non-delegable action for assigning a role to an employee.

## 3. Staff academic assignments and workload

| Key | Meaning / UI effect | Delegable? | Where granted or changed |
|---|---|---:|---|
| teacher_assignments.read | Open Academic Assignments and read subject-teacher assignment history. | No | School Admin system role in current policy. |
| teacher_assignments.manage | Create/revoke subject-teacher assignments. | No | School Admin system role in current policy. |
| class_incharge_assignments.read | Read class-in-charge assignments and history. | No | School Admin system role in current policy. |
| class_incharge_assignments.manage | Create/revoke class-in-charge assignments. | No | School Admin system role in current policy. |
| timetable.read | Open Workload Report. A timetable-only Teacher sees My Workload and own scope; users.read plus timetable.read enables School Admin school-wide scope. | Yes | Platform Admin delegation → School Admin custom role or individual override. Backend scope remains authoritative. |
| school.audit.read | Open employee History & Activity and read authorized school audit events. | No | School Admin system role in current policy. |

Teacher role eligibility is separately enforced: only an active Teaching employee with an active login may receive the system Teacher role. Non-teaching staff must not receive Teacher by default.

## 4. Staff attendance

All four keys below are non-delegable in V58. The Staff Attendance workspace is therefore School Admin-only under the current policy.

| Key | Meaning / UI effect | Delegable? |
|---|---|---:|
| staff.attendance.read | Open daily register and monthly staff attendance reports. | No |
| staff.attendance.mark | Mark staff attendance for the current school day. | No |
| staff.attendance.update_same_day | Update same-day staff attendance. | No |
| staff.attendance.correct_past | Correct historical staff attendance. | No |

## 5. Staff leave and holidays

All six keys below are non-delegable in V59. The current Leave & Holidays workspace is an admin operations workspace; it does not create Teacher self-service leave access.

| Key | Meaning / UI effect | Delegable? |
|---|---|---:|
| staff.leave.read | Read staff leave types and requests. | No |
| staff.leave.manage_types | Create/update leave types. | No |
| staff.leave.manage_requests | Create/cancel staff leave requests. | No |
| staff.leave.decide | Approve/reject staff leave requests. | No |
| staff.holidays.read | Read school holiday calendar. | No |
| staff.holidays.manage | Create/update school holidays. | No |

## 6. Staff substitutions

Both keys below are non-delegable in V60.

| Key | Meaning / UI effect | Delegable? |
|---|---|---:|
| staff.substitutions.read | Read dated substitute-teacher coverage and history. | No |
| staff.substitutions.manage | Assign/cancel dated substitute-teacher coverage. | No |

## 7. Current policy consequences

- A new Non-teaching employee receives no default system role. Without an approved custom role, the employee sees only shared Dashboard and Settings.
- School Admin currently owns Staff Directory, employee account actions, role assignment, individual overrides, staff attendance, leave/holidays, substitutions and academic assignment management.
- Teacher's default role exposes My Workload, Students, student Attendance, Timetable, Dashboard and Settings. It does not expose Staff Directory or Staff Attendance.
- Although roles.read, permissions.read and timetable.read are delegable, this does not bypass the non-delegable users.read requirement for Staff Directory.
- Staff Attendance, Leave/Holidays, Substitutions and Academic Assignment permissions are non-delegable. Do not flip their delegability or grant broader roles as a shortcut; that requires a product/security decision and resource-scope tests.
- Employment Active/Inactive is not a permission. It is a profile state changed under Edit employee profile → Employment. Login Enable/Disable is a separate account action.

## 8. Continuation rule

Before changing a permission, verify its catalogue description and delegability in the current migration, its UI gate, its API route guard, its repository authorization and its tenant/resource scope. Update this dictionary and STAFF-MANAGEMENT-ACCESS-MATRIX.md in the same batch whenever a permission policy changes.

### Additional design constraints

- There is currently no delegated HR/Staff Manager permission bundle: users.read and staff operations are non-delegable. A Non-teaching custom role cannot be given Staff Directory access in the current policy.
- There are no Teacher self-service staff attendance/leave permissions. The current staff attendance and leave keys are School Admin operational permissions, not own-record permissions.
- Do not make these keys delegable without dedicated row-scoped/self-only API authorization and cross-employee negative tests.
