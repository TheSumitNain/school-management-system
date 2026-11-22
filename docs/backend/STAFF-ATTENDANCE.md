# Staff Attendance — implementation contract

Status: V58 schema and permission catalogue added; API and UI implementation in progress. Do not treat the feature as available until backend integration tests, browser acceptance tests and CI all pass.

## Scope

Staff attendance is a separate school-admin workflow. It must not read from, write to, or alter the student `attendance_records` table or student attendance permissions.

- Daily register for active staff in the authenticated school.
- One attendance record per staff principal per school-local calendar date.
- Supported statuses: `present`, `absent`, `late`, `half_day`.
- Optional remarks, maximum 500 characters.
- Monthly register/summary is derived from the daily records; never store or guess totals.
- Marking is allowed only for the current Asia/Kolkata school date. Historical changes require the dedicated past-correction permission.
- Same-day changes require the same-day update permission.
- No biometric, GPS, geofencing, device tracking, automatic punch-in, payroll deduction or leave integration in this phase.

## Data and tenant isolation

V58 adds `staff_attendance_records` with:
- `school_id` derived only from authenticated context.
- Composite same-school FK to `staff_profiles(principal_id, school_id)`.
- Composite same-school FKs for marker and updater principals.
- Unique `(school_id, staff_principal_id, attendance_date)`.
- Immutable staff identity and append-only audit events for mark/update operations.

The migration is additive. Its down migration refuses to discard any staff attendance record. No persistent/local database may be migrated without explicit user approval.

## Permissions

V58 adds four school-scoped, non-delegable permissions, granted only to each school's School Admin system role:
- `staff.attendance.read`
- `staff.attendance.mark`
- `staff.attendance.update_same_day`
- `staff.attendance.correct_past`

The future-school trigger grants these to the newly provisioned School Admin role after the existing system-role provisioning trigger. Teacher role permissions and the student attendance permission bundle remain unchanged.

Every API operation must re-evaluate effective permissions on the server, including DENY overrides, active principal/user/school state and school scope. Frontend visibility is not authorization.

## API contract

- `GET /school/staff-attendance?date=YYYY-MM-DD`: active staff roster and that day's attendance state.
- `GET /school/staff-attendance/month?month=YYYY-MM`: month register with derived per-staff totals and daily entries.
- `POST /school/staff-attendance`: create a daily record; duplicate staff/date returns conflict, never silently overwrites.
- `PATCH /school/staff-attendance/:recordID`: update status/remarks under same-day or historical permission.

Every list is school-scoped and bounded to one date or one calendar month. Mutations and their audit events are one transaction. Do not return records for another school or disclose whether a cross-school staff principal exists.

## UI contract

Staff Management → Attendance:
- Date picker defaults to the school-local current date.
- Roster shows employee name/code/designation and clear Present/Absent/Late/Half-day/Not marked state.
- Marking and edits have field-level validation, pending/disabled states, success/error toasts and explicit correction confirmation.
- Month view shows derived counts and per-day status without mixing leave or holidays into attendance totals.
- Responsive desktop/tablet/mobile layouts; loading, empty, error/retry and no-permission states.
- No attendance data is fabricated. No change to Academic, Student Management or Settings pages.

## Verification

Required before declaring complete:
- Full migration chain on disposable PostgreSQL, including V58.
- PostgreSQL integration tests for same-school FK, uniqueness, permission grants, audit atomicity, date rules, DENY override and cross-school isolation.
- Backend `go test -count=1 -p 1 ./...` and `go vet ./...`.
- Frontend typecheck/build and Playwright coverage for daily marking, duplicate conflict, same-day update, historical permission gate, month summary, field validation and responsive behavior.
- Never apply V58 to a user database without explicit approval.
