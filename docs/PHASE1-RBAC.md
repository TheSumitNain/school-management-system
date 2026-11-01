# Phase 1 RBAC — Roles, Permissions and Account Rules

**Decision status:** Product and authorization requirements approved for Phase 1. Database schema and implementation are pending.

This document is the detailed reference for Phase 1 role scope, permission identifiers, account boundaries, teacher assignments, and parent access. It describes approved requirements, not implemented functionality.

## 1. Authorization principles

1. **Default deny.** An operation requires both an effective permission and a valid resource scope.
2. Permissions define **what** an account may do. School ownership, teaching assignments, class-in-charge assignments, and verified parent-child links define **which records** it may access.
3. The backend enforces authorization on every protected request. Frontend hiding/showing controls are not security controls.
4. Permission and role changes must apply to the next protected request, including requests using an existing session.
5. A permission override never bypasses school boundaries, class/subject assignment, parent-child links, field-level restrictions, account status, or school status.
6. Sensitive administrative changes and authorization failures must be auditable without logging passwords, OTPs, raw tokens, or other secrets.

## 2. Phase 1 roles and account boundaries

| Role | Scope | Provisioning | Boundary |
|---|---|---|---|
| Platform Admin | Global platform | Authorized platform provisioning | Manages platform operations and provisions School Admins. Global role does not automatically expose school academic/student records. |
| School Admin | Exactly one school | Platform Admin | Manages only that school. Cannot create/promote Platform Admins or another School Admin. |
| Teacher | Exactly one school | School Admin | Access is limited by permissions and exact teaching/class-in-charge assignments. |
| Parent | Officially linked children; links may span schools | First successful registered-mobile OTP verification after guardian matching | Read-only access to currently verified linked children only. |

### 2.1 Staff account rules

- Every School Admin and staff account belongs to exactly one school. The same account must never access multiple schools.
- If one person works at multiple schools, each school creates a separate account with independent username, password, and sessions.
- Staff usernames are globally unique and case-insensitive across the platform. Login must resolve exactly one account; school selection is not used to disambiguate duplicate usernames.
- School Admin creates staff accounts and provides an initial temporary password. Staff must change it on first login.
- Staff self-service password change requires the current password.
- School Admin can reset a staff password without knowing the old password. Reset revokes the target account's sessions and outstanding access, requires a password change at next login, and creates an audit event.
- School Admin accounts are provisioned by Platform Admin. A School Admin cannot create another School Admin or a Platform Admin.

### 2.2 Parent and student account rules

- Parent signs in using a registered mobile number and OTP; there is no separate parent registration form in the normal flow.
- A parent account is created or linked only after successful OTP verification and a matching officially recorded guardian relationship.
- One verified parent account can access multiple officially linked children, including children in different schools.
- Father and mother may each have separate verified mobile accounts linked to the same child.
- Parent dashboard lists all currently linked children. Parent selects one child to view that child's permitted data.
- Selecting a child is not authorization by itself: every API request verifies the authenticated parent-child link server-side.
- Removing one child link must not remove access to the parent's other valid child links.
- **No separate student login accounts or student passwords are included in Phase 1.**

## 3. Permission catalogue

Permission keys are stable identifiers in a platform-defined catalogue. School Admins may create custom roles from authorized catalogue entries; they cannot invent platform permissions or exceed their delegation authority.

### 3.1 Platform and school administration

| Permission key | Meaning |
|---|---|
| `platform.schools.read` | View platform school records |
| `platform.schools.create` | Create a school |
| `platform.schools.update` | Update platform-level school metadata |
| `platform.schools.suspend` | Suspend or reactivate a school |
| `platform.school_admins.create` | Provision/invite a School Admin |
| `platform.school_admins.disable` | Disable a School Admin account |
| `platform.audit.read` | Read platform-level audit events |
| `school.settings.read` | View school settings |
| `school.settings.update` | Update school settings |
| `school.audit.read` | Read authorized audit events for the school |

### 3.2 Academic structure and assignments

| Permission key | Meaning |
|---|---|
| `academic_years.read` | Read configured academic years |
| `academic_years.manage` | Create/update/close academic years |
| `classes.read` | Read classes and sections within school scope |
| `classes.manage` | Create/update/archive classes and sections |
| `subjects.read` | Read school subjects |
| `subjects.manage` | Create/update school subjects |
| `teacher_assignments.read` | Read teacher class/subject assignments |
| `teacher_assignments.manage` | Create/update/revoke teaching assignments |
| `class_incharge_assignments.read` | Read class-in-charge assignments |
| `class_incharge_assignments.manage` | Assign/change/revoke class-in-charge responsibility |
| `guardians.read` | Read guardian contact/relationship records within school authority |
| `guardians.manage` | Create/update/verify/revoke guardian relationship records |

### 3.3 Students

| Permission key | Meaning |
|---|---|
| `students.read_basic` | Read approved basic student fields |
| `students.read_sensitive` | Read restricted student profile fields |
| `students.create` | Create/admit a student |
| `students.update` | Update student records |
| `students.archive` | Archive a student record |

Teacher basic-field allowlist: **student name, roll number, class, and section**, plus only technical identifiers strictly needed by the API. This does not include home address, guardian contact details, fees/financial information, medical details, or other sensitive profile fields.

### 3.4 Attendance

| Permission key | Meaning |
|---|---|
| `attendance.read` | Read attendance within an independently verified scope |
| `attendance.mark` | Mark attendance for an authorized class/section and date |
| `attendance.update_same_day` | Correct same-day attendance within the authorized class and configured cutoff |
| `attendance.correct_past` | Correct past attendance; School Admin only in Phase 1 |

### 3.5 Timetable and homework

| Permission key | Meaning |
|---|---|
| `timetable.read` | Read timetable within assigned/linked scope |
| `timetable.manage` | Create or update school timetable |
| `homework.read` | Read homework within assigned/linked scope |
| `homework.create` | Create homework for an assigned subject/class or authorized class-in-charge scope |
| `homework.update` | Update homework within the same authorized scope |
| `homework.delete` | Delete homework where explicitly authorized |

### 3.6 Exams and results

| Permission key | Meaning |
|---|---|
| `exams.read` | Read exam information within scope |
| `exams.manage` | Create/update school exam configuration |
| `results.read` | Read results within scope |
| `results.enter` | Enter marks for an assigned class/subject when explicitly granted |
| `results.update` | Update marks within assigned scope before publication/lock when explicitly granted |
| `results.publish` | Publish results; School Admin only in the Phase 1 default matrix |

### 3.7 Notices

| Permission key | Meaning |
|---|---|
| `notices.read` | Read notices within school/class/linked-child scope |
| `notices.create` | Create notices |
| `notices.update` | Update notices |
| `notices.publish` | Publish notices |
| `notices.delete` | Delete notices where explicitly authorized |

### 3.8 Staff accounts, roles, permissions and audit

| Permission key | Meaning |
|---|---|
| `users.read` | Read staff account information within the school |
| `users.create` | Create staff accounts within the school |
| `users.update` | Update permitted staff account details |
| `users.disable` | Disable staff accounts within the school |
| `users.password_reset` | Reset same-school staff password without old password |
| `roles.read` | Read roles visible within the school |
| `roles.create_custom` | Create a school-scoped custom role |
| `roles.update_custom` | Update a school-scoped custom role |
| `roles.assign` | Assign permitted roles to staff in the same school |
| `permissions.read` | Read the platform-defined permission catalogue |
| `permissions.assign` | Assign catalogue permissions within the School Admin's explicit permission delegation ceiling |
| `permissions.override` | Set per-user allow/deny overrides within the same delegation ceiling |

A School Admin cannot create, edit, or assign Platform Admin or School Admin privileges, platform permissions, or permissions outside the school's authorized delegation ceiling.

### 3.9 Parent-only permissions

| Permission key | Meaning |
|---|---|
| `children.list` | List all currently verified linked children for the authenticated parent |
| `children.profile.read` | Read approved basic profile of a linked child |
| `children.attendance.read` | Read attendance of a linked child |
| `children.timetable.read` | Read timetable of a linked child |
| `children.homework.read` | Read homework of a linked child |
| `children.results.read` | Read published results/report cards of a linked child |
| `children.notices.read` | Read notices applicable to a linked child |

Parent permissions are read-only in Phase 1. No parent permission grants access to an unlinked child or staff-management functionality.

## 4. Default role permission matrix

The table describes default capabilities; every entry remains subject to resource scope and field filtering.

| Capability | Platform Admin | School Admin | Teacher | Parent |
|---|---|---|---|---|
| Manage schools / lifecycle | Yes | No | No | No |
| Provision School Admin | Yes | No | No | No |
| Manage own-school settings | Platform governance only | Yes | No | No |
| Manage student admission/profile | No by default | Yes | No by default | No |
| Read basic student identity | No by default | Yes | Assigned teaching classes and class-in-charge class only | Linked children only |
| Read sensitive student profile | No by default | Yes, authorized scope | No by default | No by default |
| Read/mark attendance | No by default | Yes | Class-in-charge class only | Linked children: read only |
| Same-day attendance correction | No by default | Yes | Class-in-charge class only, within cutoff | No |
| Past attendance correction | No by default | Yes | No | No |
| Read timetable | No by default | Yes | Assigned teaching classes/subjects and class-in-charge class | Linked children |
| Manage timetable | No by default | Yes | No by default | No |
| Read homework | No by default | Yes | Assigned subject/class and class-in-charge class | Linked children |
| Create/update homework | No by default | Yes | Assigned subject/class where explicitly permitted | No |
| Manage exam setup | No by default | Yes | No by default | No |
| Enter/update marks | No by default | Yes | No by default; may be explicitly delegated for assigned subject/class | No |
| Publish results | No by default | Yes | No by default | No |
| Read published results | No by default | Yes | Only if separately authorized | Linked children |
| Manage notices | No by default | Yes | No by default | No |
| Read applicable notices | No by default | Yes | Assigned/class-in-charge class scope | Linked children |
| Manage staff accounts / reset passwords | No by default | Yes, same-school staff only | No | No |
| Create/assign custom roles | Platform role governance | Yes, school-scoped and bounded | No | No |
| Read school audit events | Platform-level authorized audit only | Yes, own school | No | No |

Platform Admin's global role does not automatically provide routine access to school student, guardian, attendance, or financial records. Exceptional support access, if introduced, must be separately authorized, narrowly scoped, time-bound where appropriate, and audited.

## 5. Teacher assignment rules — strict separation

Teaching assignment and class-in-charge assignment are separate relationships. One never implies the other.

### 5.1 Subject/class teaching assignment

A teacher assigned to teach a particular class/section/subject may:

- Read only the basic student fields: name, roll number, class, and section for the assigned class/section.
- Read timetable relevant to the assigned class/subject.
- Read and manage homework only for the assigned subject/class when the corresponding permission is granted.
- Access exam/marks operations only if explicitly granted, and only for the assigned subject/class.

A teaching assignment alone does **not** grant attendance marking, attendance correction, full student profiles, guardian contacts, financial data, or access to other classes/subjects.

### 5.2 Class-in-charge assignment

A teacher explicitly assigned as class-in-charge for a specific class/section may:

- Read basic student information for that class.
- Read and mark attendance for that class.
- Make same-day attendance updates within the school's configured cutoff.
- Read that class's timetable, homework, and applicable class notices.
- Access only other class-management actions explicitly granted.

Being class-in-charge of Class 8-A does not grant access to Class 8-B, even in the same school. Past attendance corrections remain School Admin-only.

### 5.3 Backend enforcement

Every request must validate authenticated user status, active school, effective permission, exact assignment, resource school/class/section/subject/date, and field-level response allowlist. A client-supplied `school_id`, `class_id`, `section_id`, `subject_id`, or `student_id` is never proof of authorization.

## 6. Custom roles and overrides

- School Admin may create school-scoped custom roles using the platform-defined permission catalogue.
- A custom role cannot be global or cross school boundaries.
- A School Admin may delegate only permissions within an explicit authorized delegation ceiling.
- A School Admin cannot create/assign Platform Admin or School Admin privileges or platform permissions.
- One staff account may have multiple roles, all within its one school.
- Role permissions are combined before per-user overrides are evaluated.
- Explicit user `deny` overrides any role-derived `allow`.
- Explicit user `allow` may grant a permission absent from assigned roles only when the assigning administrator is authorized to delegate it.
- Overrides never bypass tenant, assignment, parent-child, field-level, account-status, or school-status checks.
- Default is deny.

## 7. Parent access rules

- Parent signs in using a registered mobile number and OTP.
- Parent identity/account is created or linked only after successful OTP verification and a matching officially recorded guardian relationship.
- One parent account can access all officially linked children, including children in different schools.
- Father and mother may each have separate verified mobile accounts linked to the same child.
- Parent dashboard lists linked children; selecting a child shows that child's profile, attendance, timetable, homework, published results/report cards, and applicable notices.
- Every child-data API request verifies the parent-child link on the backend; selecting a child in the UI is not authorization.
- Removing one child link must not remove other valid child links.
- Parent access is read-only in Phase 1; there are no student login accounts.

## 8. Password and account lifecycle rules

- Staff account is created by the School Admin with a temporary initial password; `must_change_password` behavior is required at first login.
- Staff can change their own password only after verifying the current password.
- School Admin can reset a same-school staff password without knowing the old password.
- Admin reset must revoke all refresh sessions and invalidate already-issued access tokens, require a password change at next login, and write an audit event with actor, target, timestamp, and outcome/context. Never store or log the password itself.
- Disabling a user or suspending a school must block subsequent protected requests and revoke sessions/tokens.
- Role/permission changes must affect the next protected request; permissions must not be trusted from stale JWT claims.

## 9. Future roles

Clerk, Manager, Librarian, Gate Keeper, Helper, Bus Conductor, and other operational roles are not implemented in Phase 1. The RBAC design must support adding them later without redesigning authentication.

## 10. Implementation status and next work

The above decisions are approved requirements. They are **not yet implemented** in the current backend application. The current source still has school-scoped username uniqueness and school-ID + username + password login; the current JWT subject is `users.id`, and middleware does not yet enforce principal status/token version, RBAC, tenant/resource scope, teaching/class-in-charge assignments, or parent-child links.

A draft additive migration now exists at `backend/migrations/000004_add_principal_rbac_foundation.up.sql`. It introduces the shared principal model, account subtype tables, global username uniqueness, permission catalogue, system roles, role permissions, principal-role assignments, overrides, delegation ceilings, principal-bound refresh sessions, MFA/OTP storage, and audit events. The existing `users.id` and refresh-token family design are preserved. See `docs/V4-IMPLEMENTATION-NOTES.md` for exact contents and blockers.

**Migration v4 is not applied and has not been executed against PostgreSQL. Do not run it yet.** Complete the principal-aware Go authentication/session refactor, database-backed token-version checks, permission evaluation, migration tests on a disposable PostgreSQL 18 database, and explicit confirmation of the existing staff account's role first. Preserve refresh-token family locks, cross-family principal serialization, replay detection, and family revocation. Do not infer the existing account's role from database counts.
