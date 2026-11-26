## 2026-11-26 — Staff directory and attendance read snapshots hardened

- Staff Directory page counts and page rows now run in one read-only repeatable-read transaction for pool-backed requests. The directory summary's authorization and aggregate query also share one snapshot. Caller-owned transactions retain their existing isolation.
- Staff Attendance Daily and Monthly totals and rows now use the same read-only repeatable-read pattern for pool-backed requests, preventing page totals from drifting away from returned rows while attendance is being marked or corrected.
- Monthly Attendance CSV export now requires valid pagination metadata on the first and every subsequent page, verifies page size/offset/total/row count and unique staff identities, and refuses to create a CSV if metadata is missing or any page is incomplete. Playwright covers 101 staff across two export pages and verifies missing metadata produces no download; frontend CI [45825300711](https://github.com/TheSumitNain/school-management-system/actions/runs/45825300711) **PASS** with 154 tests.
- Backend CI [45827227688](https://github.com/TheSumitNain/school-management-system/actions/runs/45827227688) **PASS** on `ec0825d386889bbc84bd4a243f0d58fe2ced6881`: full migrations, Go/PostgreSQL integration tests and `go vet`.
- Frontend CI [45825300711](https://github.com/TheSumitNain/school-management-system/actions/runs/45825300711) **PASS** on the preceding frontend checkpoint: TypeScript, Vite production build and 154 Playwright tests.
- No schema migration or persistent school-data change. Changes remain on `work`; `main` remains untouched.

---

## 2026-11-26 — Staff Attendance global filters and consistent pagination verified

- Daily Attendance now applies employee search (name, employee code, designation) and status filters on the server before pagination. Supported status filters are all, unmarked, present, absent, late and half-day; search is trimmed and bounded to 100 Unicode code points.
- Daily and Monthly count/page queries are read in one read-only repeatable-read transaction for pool-backed requests, so the total cannot disagree with returned rows because an attendance mark changed between separate queries. Caller-owned transactions retain their existing isolation.
- Daily and Monthly pagination remain bounded (default 50, maximum 100). Bulk marking is still limited to visible, unmarked employees on the current page and remains atomic.
- PostgreSQL integration and service/handler tests cover search/status filtering, filter validation and exact filtered totals. Playwright now verifies that searching for a staff member beyond the first 50-row page returns that employee, and that status filters update the filtered total before pagination.
- Backend CI [45825300901](https://github.com/TheSumitNain/school-management-system/actions/runs/45825300901) **PASS** on `c78032b27df34672e7e7cc0d258cc8ce90b78917`: full migrations, Go/PostgreSQL integration tests and `go vet`. Frontend CI [45825300711](https://github.com/TheSumitNain/school-management-system/actions/runs/45825300711) **PASS** on the same checkpoint: TypeScript, Vite production build and **154 Playwright tests passed**.
- No schema migration or persistent school-data change. Changes remain on `work`; `main` remains untouched.

---


## 2026-11-25 — bulk attendance HTTP, service validation and concurrency verification

- Backend CI [45598938508](https://github.com/TheSumitNain/school-management-system/actions/runs/45598938508) **PASS**: bulk HTTP handler tests verify authenticated principal/school context, payload forwarding, created-count response, invalid-date rejection before service invocation, and unauthenticated rejection.
- Backend CI [45599478011](https://github.com/TheSumitNain/school-management-system/actions/runs/45599478011) **PASS** on `41851e988b17bd90a32c9cf623913aa61556ae28`: ephemeral PostgreSQL migrations, full Go suite including integration tests, and `go vet`.
- New service tests verify invalid actor/school, status, remarks, empty/duplicate/non-positive IDs, the 1,000-record maximum, and acceptance at the maximum batch size.
- New PostgreSQL concurrency regression starts two simultaneous bulk requests for the same staff/date set: exactly one transaction succeeds, the competing transaction returns the attendance conflict, and the final state contains exactly one complete set of attendance records and audit events (no partial losing transaction).
- Regression coverage only; no production behavior, schema, persistent database, or school data changed.

# Staff Attendance — implementation contract

Status: Daily/monthly register, employee search/status filtering, atomic bulk marking, monthly CSV export, and permission-gated approved-leave/school-holiday calendar context are implemented. Automated backend and browser acceptance coverage exists; manual authenticated-school sign-off remains separate.

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

- `GET /school/staff-attendance?date=YYYY-MM-DD&limit=50&offset=0`: active staff roster and that day's attendance state. Optional `search` matches name, employee code or designation across the school roster; optional `status` accepts `all`, `unmarked`, `present`, `absent`, `late` or `half_day`. Daily limit is 1–100 (default 50), offset is 0–1,000,000, and totals reflect the active filters.
- `GET /school/staff-attendance/month?month=YYYY-MM&limit=50&offset=0`: month register with derived per-staff totals and daily entries; pagination is bounded to 1–100 (default 50).
- `POST /school/staff-attendance`: create one daily record; duplicate staff/date returns conflict, never silently overwrites.
- `POST /school/staff-attendance/bulk`: atomically create records for 1–1000 unique staff principals; if any target is invalid or any duplicate exists, the entire batch rolls back.
- `PATCH /school/staff-attendance/:recordID`: update status/remarks under same-day or historical permission.

Every list is school-scoped and bounded to one date or one calendar month. Mutations and their audit events are one transaction. Do not return records for another school or disclose whether a cross-school staff principal exists.

## UI contract

Staff Management → Attendance:
- Date picker defaults to the school-local current date.
- Roster supports school-wide employee name/code/designation search and status filtering before pagination; filtered totals and pages stay aligned. Shows clear Present/Absent/Late/Half-day/Not marked state.
- Bulk status entry applies only to visible, unmarked staff and requires explicit count confirmation; all-or-nothing transaction semantics prevent partial batches.
- Marking and edits have field-level validation, pending/disabled states, success/error toasts and explicit correction confirmation.
- Month view shows derived counts and per-day status without mixing leave or holidays into attendance totals.
- Responsive desktop/tablet/mobile layouts; loading, empty, error/retry and no-permission states.
- No attendance data is fabricated. No change to Academic, Student Management or Settings pages.

## Verification

Required before declaring complete:
- Full migration chain on disposable PostgreSQL, including V58.
- PostgreSQL integration tests for same-school FK, uniqueness, permission grants, single and bulk audit atomicity, bulk all-or-nothing rollback, date rules, DENY override and cross-school isolation.
- Backend `go test -count=1 -p 1 ./...` and `go vet ./...`.
- Frontend typecheck/build and Playwright coverage for daily marking, duplicate conflict, same-day update, historical permission gate, month summary, field validation and responsive behavior.
- Never apply V58 to a user database without explicit approval.

## 7. Monthly calendar context (2026-11-22)

- Monthly attendance view now has a separate calendar-context section for approved leave requests and active school holidays.
- Context is loaded only when the signed-in principal has the corresponding `staff.leave.read` / `staff.holidays.read` permission.
- Calendar context is explicitly excluded from attendance totals. No leave request or holiday creates, removes or changes an attendance record.
- Leave requests are filtered by date-range overlap with the selected month; holiday rows are bounded to the same month.
- Backend PostgreSQL CI [44986574096](https://github.com/TheSumitNain/school-management-system/actions/runs/44986574096) **PASS** at code checkpoint `a01514119b0d395b106accd990f59b2f7a5e992f`, including the new date-range repository integration test and `go vet`. Frontend production build passed; final Playwright run is pending.



## 2026-11-25 follow-up — bulk transaction rollback coverage

- Added PostgreSQL integration regressions for successful multi-employee marking with one audit event per record, full rollback when an audit insert fails, duplicate-target rejection before writes, and full rollback when a later target is invalid after an earlier target has been inserted in the transaction.
- The initial test run exposed an overlong generated fixture username; shortened it to comply with the database's 3–32 character username constraint. No production code or schema change was needed.
- Backend CI [45598044386](https://github.com/TheSumitNain/school-management-system/actions/runs/45598044386) **PASS** on `3bad57eb5ced6d3686824ef0e2c1e3575c5438cc`: full migration chain, Go tests including PostgreSQL integration and `go vet`.
- Backend CI [45598469332](https://github.com/TheSumitNain/school-management-system/actions/runs/45598469332) **PASS** on `e84c46d7e56d3cdb64d39ebeac281c3f8ab0e464`: full migration chain, Go tests including the added invalid-target rollback case, and `go vet`.
- Existing Playwright acceptance covers individual marking, explicit bulk-count confirmation, atomic bulk request, monthly summary/export, permission-denied controls, and calendar context. Frontend CI [45593176068](https://github.com/TheSumitNain/school-management-system/actions/runs/45593176068) **PASS** on the unchanged frontend code checkpoint `b8e46ac4bbe936ac64517524f3af328e7ac6d5d4`.
- No production behavior change, schema migration or persistent school data change. Manual authenticated-school sign-off and the separately approved leave-entitlement policy remain outside this automated verification.
