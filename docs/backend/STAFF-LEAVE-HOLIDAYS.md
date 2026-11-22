# Staff Leave & Holiday Management — Backend Contract

**Status:** Initial backend and frontend workflows implemented; phase exit gate remains open.  
**Reviewed source checkpoint:** `ddc99f0dd764b8ef9823b9c56866d74eaf968982` on `work`.  
**Product source:** [Staff Management Roadmap](../product/STAFF-MANAGEMENT-ROADMAP.md).

This contract is additive. It does not authorize applying a migration to a persistent database. It deliberately does not define statutory or school-specific leave entitlement rules.

## 1. Scope and explicit non-goals

Phase 4 provides school-scoped leave-type configuration, School Admin-managed staff leave applications, approval/rejection, and a school holiday calendar. It does not provide employee self-service, paid messaging, document uploads, leave balances, accrual, carry-forward, encashment, payroll deductions, statutory compliance calculations or automatic attendance marking.

A leave application is an independent record. Approval does not create, overwrite or delete a row in `staff_attendance_records`. Attendance may display approved leave as contextual information after an explicit read integration; it must not convert it into a manual attendance mark. Holidays are calendar entries, not attendance records.

## 2. Data model

All tables carry `school_id`; every staff/leave-type relationship uses a composite same-school foreign key. School ownership is derived from the authenticated actor, never from request JSON.

### 2.1 `staff_leave_types`

- `id`, `school_id`, `name`, `is_paid`, `requires_approval`, `is_active`, `created_by`, `created_at`, `updated_at`.
- Unique normalized name per school.
- Paid/unpaid and approval-required are explicit administrator-entered configuration.
- No quota, accrual, carry-forward or balance columns in this phase. Do not display a balance.

### 2.2 `staff_leave_requests`

- `id`, `school_id`, `staff_principal_id`, `leave_type_id`.
- Inclusive `start_date` and `end_date` (date-only, interpreted in Asia/Kolkata).
- `duration_type`: `full_day` or `half_day`; `half_day_session` is required only for a one-day half-day request and is otherwise null.
- Required trimmed `reason`; `status`: `pending`, `approved`, `rejected`, or `cancelled`.
- `decision_comment`, `decided_by`, `decided_at`, `created_by`, timestamps.
- Composite same-school foreign keys to the staff principal and leave type.
- CHECK constraints for date order, status and half-day shape; indexes for staff/date and school/status/date queries.
- Rejection requires a non-empty decision comment. Approval comment is optional. Decision actor/time are server-derived.
- No supporting-document field until private storage and authenticated download authorization exist.

### 2.3 `school_holidays`

- `id`, `school_id`, `holiday_date`, `name`, optional `description`, `is_active`, `created_by`, timestamps.
- Date-only in Asia/Kolkata. Multiple named observances may share a date; prevent duplicate normalized names for the same school/date.
- No automatic national/state holiday import or assumed recurring holiday rules.

## 3. Invariants and transactional behavior

1. Every read and write is school-scoped and uses live effective permissions; target staff must belong to the authenticated school.
2. Leave type must be active when a request is created. Staff must have active employment status. Login status is not a proxy for employment status.
3. Start date must be on/before end date. Half-day requests cover exactly one date and require a valid configured session; full-day requests have no half-day session.
4. Reject overlapping non-cancelled requests for the same staff member when either request is pending or approved. Perform the overlap check under a transaction/appropriate row or advisory lock so concurrent submissions cannot both succeed.
5. Approval/rejection is allowed only from pending state. Reject requires a comment. Decision actor is the authenticated School Admin. A second decision returns conflict; no silent rewrite.
6. If a leave type is configured with `requires_approval=false`, creation may become approved only if that behavior is explicitly implemented and audited in the same transaction. Otherwise keep all new requests pending until the workflow is finalized; do not silently bypass review.
7. Every create, decision, cancellation, leave-type change and holiday change writes an append-only audit event in the same transaction. Audit metadata contains IDs/status changes only, never free-text reason/comment unless the audit policy explicitly allows it.
8. Do not automatically modify staff attendance. Approved leave can be queried as a derived overlay; manual attendance conflicts must be surfaced for explicit resolution, not silently overwritten.
9. Holiday dates are calendar context only. They do not create attendance rows or auto-mark employees absent/present.
10. Do not calculate balances or payroll effects in this phase.

## 4. Permission keys

Introduce dedicated, non-delegable School Admin permissions; no teacher or parent grants:

| Permission | Purpose |
|---|---|
| `staff.leave.read` | Read leave types, requests and leave calendar |
| `staff.leave.manage_types` | Create/update/deactivate leave types |
| `staff.leave.manage_requests` | Create/cancel staff requests |
| `staff.leave.decide` | Approve/reject pending requests |
| `staff.holidays.read` | Read school holiday calendar |
| `staff.holidays.manage` | Create/update/deactivate holidays |

All checks must be enforced by route middleware and repeated in repository/service boundaries consistent with staff attendance. Do not reuse `users.update` as leave-decision authority. Existing and future schools must receive grants through the protected system-role provisioning pattern used by V58; preserve delegation ceilings and DENY precedence.

## 5. HTTP API contract

All endpoints are under the authenticated School Admin API:

| Method | Path | Permission | Purpose |
|---|---|---|---|
| GET | `/school/staff-leave/types` | `staff.leave.read` | List leave types |
| POST | `/school/staff-leave/types` | `staff.leave.manage_types` | Create leave type |
| PATCH | `/school/staff-leave/types/:typeID` | `staff.leave.manage_types` | Update/deactivate type |
| GET | `/school/staff-leave/requests` | `staff.leave.read` | List/filter requests |
| POST | `/school/staff-leave/requests` | `staff.leave.manage_requests` | Create request |
| POST | `/school/staff-leave/requests/:requestID/approve` | `staff.leave.decide` | Approve pending request |
| POST | `/school/staff-leave/requests/:requestID/reject` | `staff.leave.decide` | Reject pending request with comment |
| POST | `/school/staff-leave/requests/:requestID/cancel` | `staff.leave.manage_requests` | Cancel a pending request |
| GET | `/school/holidays?from=YYYY-MM-DD&to=YYYY-MM-DD` | `staff.holidays.read` | List calendar range |
| POST | `/school/holidays` | `staff.holidays.manage` | Create holiday |
| PATCH | `/school/holidays/:holidayID` | `staff.holidays.manage` | Update/deactivate holiday |

List endpoints should support bounded date/status/staff filters and stable ordering. Reject invalid date formats and excessive ranges. Do not accept `school_id`, `created_by`, `decided_by` or `decided_at` from clients.

## 6. Attendance integration boundary

Current staff attendance statuses are `present`, `absent`, `late` and `half_day`. There is no `on_leave` status or holiday model in the reviewed source. Initial integration should expose approved leave/holiday context alongside the daily roster without changing the attendance status enum or creating attendance records. If a manual attendance mark exists on an approved leave/holiday date, surface a conflict indicator for an authorized human to resolve. Any future `on_leave` status requires a separately reviewed migration, permission and correction/audit contract.

## 7. Required verification before implementation is considered complete

- Migration up/down on disposable PostgreSQL; down migration must refuse destructive rollback if leave/holiday history exists, consistent with attendance history protection.
- Full Go unit/integration tests and `go vet`.
- Same-school positive and cross-school negative tests for every resource type.
- Live permission, DENY override, non-delegable permission and future-school provisioning tests.
- Date boundaries, half-day shape, inactive staff/type, overlap, concurrent overlap, decision state conflict, rejection comment, cancellation, holiday duplicate and transaction rollback tests.
- Audit event written in the same transaction for every mutation; failed operations leave no partial data/audit.
- Frontend browser acceptance for field-level errors, no-op prevention, duplicate-click protection, confirm/reject/cancel flows, loading/empty/error/retry states and responsive layouts.
- CI must pass at the exact code checkpoint before updating Project Status to verified.

**Database safety:** No migration is included or applied by this design document. Persistent local databases remain untouched.

## 8. Implementation checkpoint (2026-11-22)

Migration V59 and the first end-to-end Leave & Holiday workflow are implemented on branch `work`.

### Backend

- V59 creates `staff_leave_types`, `staff_leave_requests` and `school_holidays`, with school-scoped foreign keys, date/duration/status checks and indexes.
- Six dedicated non-delegable permissions are added and provisioned to existing and future School Admin system roles. The runtime School Admin permission allowlist is updated.
- Leave type create/list/update; request create/list/approve/reject/cancel; holiday create/list/update APIs are wired.
- Request creation validates active same-school staff and leave type, date range, duration and reason. An advisory transaction lock serializes overlap checks per school/staff; pending or approved overlapping requests conflict.
- Decisions require a pending request, use the authenticated actor and server timestamp, require a comment on rejection, and prevent the applicant from approving their own request.
- Leave and holiday writes record audit events in the same transaction.
- Requests with `requires_approval=false` still remain pending in this initial workflow; the flag does not bypass review. This avoids silently inventing an auto-approval policy.
- Holiday updates can deactivate an entry. Active holiday listing is date-range bounded by the service.

### Frontend

- Staff Management shows Leave & Holidays only when the current user has an appropriate leave/holiday permission. Users without those permissions retain a disabled restricted navigation item.
- Workspace includes request listing/filtering, leave type creation, holiday listing and create/edit/deactivate controls, approval/rejection with a required rejection comment, and cancellation confirmation.
- Request form includes staff, leave type, inclusive dates, full/half-day selection and reason. No leave balance is displayed.
- Browser acceptance covers loading the leave application, approving it through the confirmation modal, and reading configured leave types and the holiday calendar.

### Verification and remaining work

- Backend CI [44982323577](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323577) passed the complete migration chain on disposable PostgreSQL, full Go tests including PostgreSQL integration, and `go vet`.
- Frontend CI [44982323582](https://github.com/TheSumitNain/school-management-system/actions/runs/44982323582) passed the TypeScript/Vite production build and all 50 Playwright browser acceptance tests.
- The CI-tested implementation checkpoint is `816ee99511d4a7ed0e393dda9a04dc758f954202`. A subsequent repository update did not change the tested file content.
- The current integration test verifies V59 schema, non-delegable permission catalogue and School Admin grants. Dedicated automated PostgreSQL behavior tests for overlapping/concurrent submissions, decision authorization, transaction rollback and audit events remain to be added.
- Approved leave and holidays are not yet shown as contextual overlays in the staff attendance register. They do not create, modify or delete attendance records.
- Leave balances, accrual, carry-forward, statutory entitlements, payroll effects, supporting-document uploads and automated holiday feeds remain out of scope.
- V59 has only been applied in disposable CI PostgreSQL. No migration was applied to a persistent local database.

## 8. Verified implementation update (2026-11-22)

- Latest backend checkpoint: `8c163b5b35a644cf08a13e280c0c44f238a80037`.
- Backend CI [44984389499](https://github.com/TheSumitNain/school-management-system/actions/runs/44984389499) **PASS** at this exact SHA. The complete migration chain including V59 ran only on disposable PostgreSQL; Go tests (including new service-level validation tests) and `go vet` passed.
- Service validation rejects invalid/reversed leave dates, invalid full/half-day combinations, missing/oversized reasons, invalid holiday dates/names/descriptions and invalid date ranges before repository operations.
- Repository overlap serialization uses a hash-derived transaction advisory lock from the complete school/staff identifiers. Repository cancellation and holiday-range reads also reject invalid repository inputs.
- Frontend CI [44982936799](https://github.com/TheSumitNain/school-management-system/actions/runs/44982936799) **PASS** on UI checkpoint `36ab57484de809c30849c030187f0190dab43a60`. The daily staff attendance register displays approved leave and active holiday context when the viewer has the respective read permissions; this is informational only and does not mutate attendance rows.
- Remaining verification: dedicated PostgreSQL behavior tests for concurrent overlap, decision authorization, transaction rollback and audit events. V59 remains unapplied to persistent/local databases.

## 9. Leave type management and final verification (2026-11-22)

- Leave type UI supports edit and active/inactive status. Deactivation requires confirmation; existing requests/history are preserved; inactive types are not offered for new requests and can be reactivated through the edit form.
- Added Playwright coverage for leave type edit/deactivation. Frontend CI [44984794710](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794710) **PASS** at `f1b71d4e2999f868f62910ee878f78eee1460a8b`, with TypeScript/Vite build and all 51 browser tests.
- Backend CI [44984794698](https://github.com/TheSumitNain/school-management-system/actions/runs/44984794698) **PASS** at the same checkpoint, including complete migration chain on disposable PostgreSQL, Go tests and `go vet`.
- Remaining: dedicated PostgreSQL behavior tests for concurrent overlap, decision authorization, transaction rollback and audit events. School policy/balance calculations remain unconfigured and out of scope until approved. V59 remains unapplied to persistent/local databases.

## 10. PostgreSQL workflow verification (2026-11-22)

- Added integration tests for concurrent overlapping leave requests, unauthorized decisions, self-approval prevention, transactional approval/rejection audit, rollback on audit failure, required rejection comments and cross-school cancellation denial.
- The overlap test exposed a pgx/PostgreSQL parameter typing issue in the advisory lock. Fixed by explicitly casting both lock identifiers to BIGINT before converting them to text for `hashtextextended`.
- Backend CI [44985769667](https://github.com/TheSumitNain/school-management-system/actions/runs/44985769667) **PASS** at code checkpoint `e028baffd2485afeee792f98b068f89e2ce1ae07`: complete migration chain on disposable PostgreSQL, full Go test suite including the new V59 workflow integration tests, and `go vet ./...`.
- The forced audit insert exception in the rollback test is intentional and confirms the decision remains pending when its audit write fails.
- Persistent/local DBs remain untouched; V59 has only been applied to disposable CI PostgreSQL. Leave balances, accrual, carry-forward, encashment and attendance/payroll effects remain unconfigured until school policy is approved.

