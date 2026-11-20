# Backend API Reference

**Current status:** Selected backend APIs are implemented. This is not a complete product API catalogue; check the route registrations and handler tests in `backend/` before relying on an endpoint.

## Authentication and identity

| Method | Path | Purpose |
|---|---|---|
| POST | `/auth/login` | Staff username/password login; administrator MFA flow is enforced. |
| POST | `/auth/refresh` | Refresh-token rotation and replay/family protection. |
| POST | `/auth/logout` | Revoke the supplied refresh-session family. |
| GET | `/auth/me` | Current principal identity; Staff and Platform Admin also receive sorted effective permission keys for UI navigation. |
| POST | `/auth/password/change` | Staff self-service password change. |
| POST | `/auth/platform-admin/password/change` | Platform Admin self-service password change. |
| POST | `/auth/mfa/*` | Administrator TOTP enrollment, verification and recovery workflows; consult source route registrations for exact subpaths. |

## Administrative and school routes

Implemented route groups include:
- Platform Admin account lifecycle and school onboarding.
- School staff account listing, creation, update, enable/disable and password reset.
- Staff account creation requires an employee profile in the same transaction: `full_name`, school-unique `employee_code`, `designation`, and exactly one required `mobile_number`, plus the login `username` and `initial_password`. `email` and `joining_date` are optional. The profile is stored separately from login credentials. `GET /school/staff` includes profile fields and `profile_complete`; legacy accounts without profile rows remain visible as incomplete.
- `PATCH /school/staff/:principalID` accepts employee profile fields and `employment_status` (`active` or `inactive`). Inactive employment disables the login and revokes sessions; returning employment to Active does not automatically re-enable the login. Explicit account enablement with a temporary password is required after the profile is complete and Active; the database rejects login activation for inactive or profile-incomplete employees.
- Role, permission, delegation and user-override administration.
- Academic year/class/section/subject and teacher assignment foundations. Section create/update accepts optional `medium` (`hindi` or `english`); section names remain school-defined and independent of medium.
- Student admission/profile and enrollment/promotion/history foundations.
- Student profile sections: optional basic details, address and contact details, guardian relationships, and restricted medical/physical details.
- Assigned-section teacher enrollment roster.

### Student enrollment directory and history

- `GET /school/enrollment-directory` lists the School Admin Student Management enrollment directory. Optional query parameters: `academic_year_id` (positive ID; omitted means all years), `limit` (default 20, allowed 1–100), and `offset` (default 0, non-negative). The response contains `enrollments` plus `pagination: { limit, offset, total }`; rows are ordered by `enrolled_at DESC, id DESC`.
- Access requires `students.read_basic` and the authenticated principal's own school. School Admins receive the enrollment view; Teachers see only enrollments in their active class-incharge or subject-assigned sections. Subject-only Teachers receive the minimal roster projection (student name, roll number, class name and section name), not admission identifiers or student IDs.
- `GET /school/students/:studentID/enrollments` remains the complete per-student enrollment history. It is intentionally not paginated so the profile can show the full academic history. The separate `/school/enrollments` roster endpoint remains available to existing section-roster consumers; do not substitute it for the bounded directory endpoint in Student Management.

- Attendance listing, marking and correction.
- Timetable current-year lookup, period definitions and weekly section schedules.
- Guardian management: mobile number and email are optional but unique within the authenticated school. Duplicate values return HTTP 409 with a field-specific `field_errors.mobile_number` or `field_errors.email_address` message. The message instructs the School Admin to use another contact address or leave that field blank. Guardian names may repeat; blank mobile/email values are allowed. The same checks apply to create and shared-profile update requests. V51 database indexes remain authoritative for concurrent requests.

Class curriculum is section-scoped. Both curriculum read and save require `academic_year_id`, `class_id` and `section_id`; student enrollment curriculum lookup uses the student's exact assigned section and never falls back to another section or a legacy class-level curriculum. Existing pre-section legacy assignments remain readable as historical records. Each section has one current editable curriculum: `GET /school/class-curricula` returns the current active curriculum, and `POST /school/class-curricula` creates it once or updates that same curriculum record in place. Saving an edit does not create a new curriculum version or reassign students. Each subject sends `is_optional` to distinguish compulsory from standalone optional subjects; an optional subject may additionally send `optional_group` to participate in a named group with minimum/maximum student selections. Student enrollment subject-selection endpoints accept only active optional subjects: standalone optional choices are independent, while grouped choices must satisfy that group's bounds. Before removing subjects, the frontend calls `POST /school/class-curricula/removal-impact` with the section scope and `subject_ids`; the response returns the exact affected-student count per subject (optional subjects count active students who selected them; compulsory subjects count active students in the section who are unassigned or assigned to this curriculum). The subsequent save sends `removal_confirmations` containing each subject ID and the reviewed count. The backend recalculates the counts under the curriculum transaction and rejects missing or stale confirmations. Confirmed removal permanently deletes the curriculum subject and all linked student optional-choice rows in the same transaction. No curriculum edit or student subject-selection history is retained; the normal security audit event records the admin action without storing removed subject details.

Student basic-profile endpoints are `GET/PUT /school/students/:studentID/basic-details`. They expose optional date/place of birth, gender, nationality, religion, social category, EWS status and school registration number. Reads require `students.read_sensitive`; writes require `students.update`. Audit metadata records only field-presence flags, not demographic values. Aadhaar is not collected by this profile.
Student health endpoints are `GET/PUT /school/students/:studentID/medical-details` and `GET/PUT /school/students/:studentID/physical-details`. Reads require `students.read_sensitive`; writes require `students.update`. The medical response includes the current profile plus prior versions; the physical response includes the latest measurement plus the dated measurement history. Physical writes append a new dated record; medical writes preserve the previous version before updating the current profile. Successful reads and writes create audit events; medical/physical values and notes are never copied into audit metadata. These endpoints are available only for the authenticated staff member's own school and do not grant Teacher roles access.

Timetable routes use `academic_year_id` on list requests. Period writes accept `period_number`, `name`, `starts_at` and `ends_at` (`HH:MM`), plus `is_break`. Entry writes accept `academic_year_id`, `section_id`, `subject_id`, `teacher_principal_id`, `teacher_subject_assignment_id`, `period_id`, `day_of_week` (1 = Monday through 7 = Sunday) and optional `room`. Break periods are defined in the period catalogue and cannot receive class entries. Period definitions cannot overlap within the same school and academic year. Entry writes require an active teacher-subject assignment matching the same school, year, section, subject and teacher. Teacher reads are restricted to the teacher's own scheduled entries.\n\nThe exact route paths, request/response DTOs, permission keys and status codes are defined by the current router and handlers under `backend/internal/`. Do not treat this summary as a substitute for source-level contract verification.

## Authorization contract

- Protected requests authenticate the principal and validate the active refresh-session family.
- Effective permissions are loaded live from PostgreSQL.
- Route-level permission middleware is combined with repository/resource checks.
- School ownership, assignment scope and field-level restrictions must be enforced server-side.
- `GET /auth/me.permissions` is a UI hint only; it is not an authorization credential.

## Not yet delivered

The API does not yet represent the complete school product. Examinations, results, the frontend and Parent/Driver Phase 2 workflows remain pending. Parent OTP/SMS delivery is paused.

For implementation status and verification evidence, see [Project Status](../project/STATUS.md). For authorization requirements, see [Phase 1 RBAC](../product/PHASE1-RBAC.md).

The `GET /school/timetable/current-year` endpoint returns the active current academic year to any timetable reader. This allows Teachers, whose standard role does not include `academic_years.read`, to view their current timetable without granting broader academic-year listing access.