# Staff Identity and Profile Schema — design review gate

**Status:** Architecture proposal only; no migration or application write path is authorized by this document.  
**Date:** 2026-11-23  
**Branch:** `work`  
**Related:** [Staff Bulk Import Design](./STAFF-BULK-IMPORT-DESIGN.md), [Staff Management Roadmap](../product/STAFF-MANAGEMENT-ROADMAP.md)

## 1. Purpose and non-goals

The approved product workflow requires an employee record to exist independently of a login account. A school may record teaching and non-teaching employees without creating credentials. A login may be linked later, separately and explicitly.

The current V28 schema cannot represent this: `staff_profiles.principal_id` is the profile primary key and a required composite FK to `users(principal_id, school_id)`. The current Add Staff flow also requires a username and initial password. This design describes a safe path to a stable school-scoped employee identity.

This is a design artifact only. It does not authorize a migration, production data change, local DB change, credential provisioning, staff document storage, or a change to existing login/permission behavior.

Out of scope:
- Staff document upload/download and profile photograph upload. Both require an explicitly selected/purchased external storage service and separate approval.
- Payroll, leave balances and payroll calculations; policy is not approved.
- Parent OTP or parent authentication.
- Automatic login creation, generated usernames/passwords, or implicit account linking.
- Deleting, merging or rewriting historical staff records.

## 2. Current schema dependencies verified in the repository

| Area | Current staff identity reference | Other principal references that must remain actor identities |
|---|---|---|
| `staff_profiles` (V28) | `principal_id` PK + FK to `users(principal_id, school_id)` | — |
| Qualifications, certifications, experience (V57) | `staff_principal_id` → `staff_profiles(principal_id, school_id)` | Qualification/certification verifier → `users` |
| Attendance (V58) | `staff_principal_id` → `staff_profiles` | Marker/updater → `users` |
| Leave requests (V59) | `staff_principal_id` → `staff_profiles` | Creator/decider and leave-type/holiday actors → `users` |
| Teacher subject assignments (V17) | `teacher_principal_id` → `users` | `assigned_by_principal_id` → `users` |
| Class-in-charge assignments (V17) | `teacher_principal_id` → `users` | `assigned_by_principal_id` → `users` |
| Timetable entries (V26) | `teacher_principal_id` and composite assignment reference | Creator/updater → `users` |
| Substitutions (V60+) | Original/substitute teacher principal IDs → `staff_profiles` | Creator/canceller → `users` |
| Audit events | Resource subject may be a staff identity | Actor principal ID must continue to reference the authenticated principal |

Do not globally rename every `principal_id` column. A principal ID is correct for an actor, approver, verifier, creator, canceller or account. Only employee-subject references should migrate to the stable `staff_id`.

The exact current migration sequence is V28, V29, V30, V57–V63 for staff/profile, academic/timetable, attendance, leave, substitution and related protections. The implementation must re-check the latest branch migration tree and constraints before writing any migration.

## 3. Target identity model

### 3.1 `staff_profiles` becomes the canonical employee record

Proposed target columns:

| Column | Type / rule | Purpose |
|---|---|---|
| `staff_id` | BIGINT identity, primary key | Stable employee identity independent of authentication |
| `school_id` | BIGINT NOT NULL | Tenant boundary; never accepted from an untrusted client |
| `principal_id` | BIGINT NULL | Optional, unique linked staff login principal |
| `employee_code` | VARCHAR(32) NOT NULL | Case-insensitive unique within a school |
| `staff_type` | `teaching` / `non_teaching`, required | Determines teaching-specific workflow eligibility |
| `full_name` | VARCHAR(160) NOT NULL | Employee display name |
| `designation` | VARCHAR(100) NOT NULL | Role/designation |
| `department_id` | BIGINT NULL | School-controlled department master |
| `employment_type` | Controlled value | Permanent, contract, part-time, temporary, internship, other |
| `mobile_number` | VARCHAR(16) NOT NULL | Work contact |
| `email` | VARCHAR(254) NULL | Work email |
| `joining_date` | DATE NULL | Employment start date |
| `employment_status` | active/inactive | Employment lifecycle, separate from login status |
| timestamps | TIMESTAMPTZ | Record lifecycle |

Required constraints and indexes:
- `PRIMARY KEY (staff_id)`.
- `UNIQUE (staff_id, school_id)` for composite tenant-safe references.
- `UNIQUE (principal_id, school_id)` allowing multiple NULL principal IDs but at most one linked staff profile per principal/school.
- Composite FK `(principal_id, school_id) → users(principal_id, school_id)`; NULL principal means no login is linked.
- `UNIQUE (school_id, lower(employee_code))`.
- Composite FK `(department_id, school_id) → staff_departments(id, school_id)`.
- Check `staff_type`, `employment_type` and `employment_status` against explicit controlled values.
- Do not cascade-delete staff, department, assignment or history rows.

The current `principal_id`-keyed uniqueness must be retained as a candidate key while legacy FKs are being transitioned. Do not drop it until all dependent FKs and application queries have been migrated and verified.

### 3.2 Department master

Proposed `staff_departments`:
- `id` identity PK, `school_id` NOT NULL, `name`, `is_active`, timestamps.
- `UNIQUE(id, school_id)` and case-insensitive `UNIQUE(school_id, lower(btrim(name)))`.
- Department deactivation is allowed only after impact is shown; it must not delete employee records or rewrite historical department snapshots.
- Do not seed arbitrary department names or use free-text department values. Each school administers its own list.
- A nullable department supports schools that have not configured the master yet.

### 3.3 Personal details and sensitive information

Keep sensitive personal fields in a separate `staff_personal_details` table keyed by `(staff_id, school_id)`, rather than returning them in every directory/list API.

Candidate fields after field-level access review:
- date of birth;
- gender (only with a clearly defined, inclusive and legally reviewed value policy; avoid an arbitrary enum);
- current and permanent address fields;
- emergency contact name, relationship and phone.

Security requirements before implementing this table:
- Separate permissions for viewing and updating sensitive personal details; `users.read` or directory visibility alone must not grant access.
- Minimize fields collected; mark each optional/required with a documented school/business purpose.
- Never include DOB, home address or emergency contact in directory search, CSV reports, ID cards, audit metadata, ordinary logs or broad staff list responses.
- Redact sensitive values from generic audit events; record field names/action and actor, not before/after PII values.
- Define retention, correction and archival behavior with applicable Indian privacy/legal review before collection.
- Decide encryption-at-rest/key-management requirements against the existing application/database security architecture before storing high-risk contact details.
- Validate and normalize phone/address input server-side; do not infer personal data.

Photograph is deliberately excluded from this database proposal until the external storage service and authorization model are approved.

### 3.4 Optional login association

A profile may have `principal_id = NULL`. Linking an account is a separate privileged, audited operation:
- The target principal must be an active staff principal and belong to the same school.
- One principal can link to at most one staff profile in that school.
- Linking must not change username, password, roles or permission overrides.
- Unlinking must be blocked if any policy or active workflow requires that account; it must never delete the auth principal or its audit/session history.
- Existing account-plus-profile creation remains supported during rollout.
- Enabling/disabling login remains a distinct operation from employment activation/deactivation.
- An inactive employee cannot sign in; employment reactivation does not silently re-enable a disabled login.
- Account linking requires a dedicated permission and explicit confirmation; never auto-link by matching name, email or phone.

## 4. Employee-subject references and compatibility plan

All employee-subject foreign keys must eventually use `staff_id`; actor foreign keys remain principal IDs.

### 4.1 Tables that require a staff_id migration

Add `staff_id` and composite same-school FK to:
- `staff_qualifications`
- `staff_certifications`
- `staff_experience`
- `staff_attendance_records`
- `staff_leave_requests`
- `teacher_subject_assignments`
- `class_incharge_assignments`
- `timetable_entries`
- `staff_substitutions` (original and substitute staff IDs)

The existing principal-based columns should remain temporarily as compatibility columns. For all existing rows, backfill `staff_id` by joining the current employee principal and school to `staff_profiles`. Verify every legacy employee reference maps to exactly one staff profile. Abort the migration if any orphan, cross-school mismatch, duplicate or missing mapping is found.

For newly created rows after the transition begins:
- Write canonical `staff_id`.
- Where compatibility code still needs the principal ID, populate it from the linked profile in the same transaction; for profile-only staff, principal columns may be NULL only after all constraints and consumers have been safely migrated.
- Never use actor principal columns as employee IDs.
- Keep school scope in every lookup, join, FK and unique constraint.

### 4.2 Teacher assignment and timetable behavior

Teacher assignment is an employee relationship, not proof that the employee has a login. After the staff_id transition:
- Only `staff_type = teaching` and `employment_status = active` employees may receive teaching/class-in-charge assignments.
- Teaching assignments and timetable rows reference `staff_id`, not an auth principal.
- A teacher without a linked login can be assigned by a permitted School Admin, but cannot access Teacher Dashboard/self-service until an account is explicitly linked and enabled.
- The existing timetable uniqueness constraints must be rebuilt against `staff_id` so an unlinked teacher cannot be double-booked.
- Timetable write validation must verify same-school staff identity, teaching type, active employment, active year/section/subject/period and the matching assignment.
- Substitution history stores original/substitute `staff_id`; creator/canceller remain principal IDs. Existing snapshot fields remain immutable.
- Future substitution and timetable conflict checks must use staff IDs and remain transactionally serialized. Do not weaken the current V60–V63 lifecycle protections.

### 4.3 Leave, attendance and records behavior

- Attendance, leave requests, qualifications, certifications and experience use `staff_id` as subject.
- Marker, approver, verifier, creator, updater and decision-maker remain authenticated principal IDs.
- Profile-only employees can be included in school-wide attendance and admin-managed leave records. A self-service leave submission is available only after an account is linked and the relevant self-service permission/product flow is implemented.
- Existing leave balances remain deferred; do not infer or calculate them from attendance.
- Existing attendance, leave, qualification, certification, experience and substitution history must remain readable and immutable through the transition.

## 5. Migration and deployment sequence (proposal only)

Use additive, expand/contract migrations. Do not combine the full conversion into one destructive migration.

1. **Preflight inventory:** verify current migration version/dirty state on every environment, count staff profiles, find duplicate employee codes, orphaned child references, cross-school references, current teacher assignments/timetable/substitutions, and accounts with/without profile rows. Capture counts and checksums without exporting PII.
2. **Verified backup gate:** take a consistent database backup and prove restore to an isolated database. Never delete existing backups to make room without explicit approval.
3. **Expand identity:** add `staff_id` to `staff_profiles`, backfill one ID per existing profile, add unique/candidate constraints and nullable optional `principal_id` only after dependent FKs are understood. Preserve existing principal IDs and user relationships.
4. **Master/profile fields:** add staff type, employment type, department master and department FK. Existing rows need a reviewed deterministic classification; do not guess teaching/non-teaching from designation text. If classification cannot be established safely, migration must assign an explicit `unclassified` state or pause and require a reviewed data-mapping exercise. The final product must not silently treat unknown as teaching.
5. **Dependent table expansion:** add staff_id columns, backfill from principal IDs and add NOT VALID composite FKs/checks where supported. Validate constraints after backfill; abort on any mismatch. Do not remove legacy columns yet.
6. **Application dual-read/write release:** deploy code that reads canonical staff_id, writes staff_id plus legacy principal ID when one exists, and accepts staff_id in routes/API. Preserve old API response fields temporarily for compatibility; introduce explicit staff_id fields. No route may accept a school ID that overrides authenticated tenant scope.
7. **Profile-only workflow:** add create/update/list by staff_id, with no username/password requirement. Account linking/unlinking is separate and permission-gated. Update directory, profile, reports, attendance and leave flows to represent missing login as a normal state rather than a broken profile.
8. **Academic conversion:** update assignments, class-in-charge, timetable and substitutions to staff_id. Rebuild unique indexes/triggers and run concurrency/integration tests for teacher-slot and section-slot conflicts.
9. **Compatibility observation:** monitor legacy principal-key usage and verify all app versions, exports and tests use staff_id. Keep compatibility fields through at least one stable release cycle.
10. **Contract only after evidence:** in a later approved migration, remove obsolete principal-based employee FKs/columns and rename API fields only after usage audit and rollback window. Actor principal references are never removed.
11. **Rollback strategy:** before the contract phase, rollback is application-first while both keys remain. Once profile-only staff or staff_id-only history exists, do not attempt a down migration that deletes those employees/history; restore from verified backup or use a forward repair migration. Never write a destructive down migration.

## 6. API and frontend contract changes

- Introduce immutable `staff_id` in all staff profile/list/detail payloads and use it for profile URLs and staff-subject operations.
- Keep `principal_id` nullable and expose it only where account/access UI needs to show whether a login is linked; never treat it as the employee primary key.
- Add explicit profile-only create and update endpoints. Do not overload the existing account-plus-profile create endpoint in a way that changes its required-password contract unexpectedly.
- Add a dedicated link/unlink login endpoint with explicit confirmation, same-school validation, live permission checks and audit events.
- Directory filters include staff type, department, employment type, employment status, login linked/unlinked and profile completeness.
- Show missing login as a clear state, not an error. Account & Access tab offers a separately permission-gated “Link existing account” workflow; no automatic username/password generation.
- Personal Details fields load only when the viewer has sensitive-profile read permission. API response and frontend state must not cache these values in broad directory queries.
- All field errors appear below their fields; duplicate employee code and department conflicts must be actionable; preview/confirmation is required for bulk imports and high-impact changes.

## 7. Backfill and classification rules

- Existing staff_profiles rows map one-to-one to staff_id; existing principal IDs remain linked.
- Existing staff accounts with no staff_profiles row are not silently promoted into complete employee records. They remain legacy accounts and must be shown in a reviewed reconciliation report for an administrator to complete.
- Do not derive staff type from designation strings, usernames, role names or timetable presence without a reviewed deterministic rule.
- A migration must not invent department values, DOB, gender, address, emergency contact or employment type.
- Existing experience rows keep their historical `employment_type` values; this history field is not automatically the employee's current employment type.
- Do not overwrite current profile values with inferred values. Any manually reconciled mapping must be auditable and approved.
- Employee codes remain school-scoped and case-insensitive. Resolve collisions before adding/enforcing the new key; do not auto-rename codes.

## 8. Test and acceptance gates before any migration is proposed for execution

### Database / integration
- Fresh database migration and upgrade from every supported pre-migration schema version.
- Backfill maps every legacy employee row and all dependent history exactly once; orphan/cross-school data causes a clean abort.
- Two schools can use the same employee code; one school cannot use case-variant duplicates.
- Profile-only staff can be created, edited, listed and deactivated without creating auth_principals/users/refresh_sessions.
- Optional principal association is unique, same-school and cannot cross-link.
- Unlinking never deletes auth principal, role, override, refresh/audit history.
- Child history and attendance/leave remain attached to the same staff_id after linking/unlinking.
- Teaching-only assignment rules and teacher/section timetable conflict constraints remain race-safe.
- Deactivation cannot erase records or bypass future substitution protection.
- Actor principal FKs still reject cross-school and non-existent actors.

### API / authorization
- Every staff_id route checks authenticated school scope and live permission in the same transaction as writes.
- A caller cannot enumerate another school's staff IDs or learn whether an ID exists without permission.
- Directory read does not reveal DOB, addresses or emergency contact.
- Sensitive personal details require dedicated read/write permission and are excluded from exports and generic audit metadata.
- Link/unlink login is separately authorized, audited and does not modify credentials or roles.
- No-login staff do not accidentally receive JWTs, refresh sessions, roles or permissions.

### Frontend / browser
- Add Staff offers a clear no-login path without asking for username/password.
- Link existing login is a separate action with confirmation; linked and unlinked states are clear.
- Teaching/non-teaching fields and assignment actions are type-aware.
- Department list handles empty, inactive, duplicate and network-error states.
- Personal details remain hidden for users without the field-specific permission, including in network responses.
- Responsive acceptance at 390px, 768px and 1280px; no page-level horizontal overflow.
- CSV, reports, ID cards and logs include only approved fields.

## 9. Decisions requiring explicit product approval

This proposal intentionally leaves the following decisions open; no implementation should guess them:

1. Confirm whether Staff Type values are exactly Teaching / Non-teaching and whether any third category is needed.
2. Confirm the Employment Type values and whether employee type is distinct from employment contract type.
3. Confirm whether Department is mandatory or optional and who can create/deactivate department master values.
4. Confirm whether Branch/Work Location is required for the first release; if yes, define its school master-data contract rather than a free-text field.
5. Approve the minimum personal-details fields and the dedicated permission model, retention and legal review.
6. Confirm whether profile-only teaching staff may be assigned to timetable and substitutions before they have a login (this proposal says yes; self-service remains unavailable until linked).
7. Approve the stable staff_id API migration and the expand/contract deployment window.
8. Approve a backup/restore rehearsal and the eventual migration implementation plan before any persistent database is touched.

## 10. Current implementation boundary

Until the design decisions and migration plan are approved:
- Keep the existing account-plus-profile create flow intact.
- Do not add a fake profile-only UI or generate credentials.
- Do not create or run schema migrations.
- Do not alter local, test, staging or production databases.
- Do not implement staff document or photo upload/storage.
- Continue improving existing Staff Management behavior only where it fits the current schema and approved permissions.
