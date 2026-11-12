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
- Assigned-section teacher enrollment roster.
- Attendance listing, marking and correction.
- Timetable current-year lookup, period definitions and weekly section schedules.
- Guardian management.

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