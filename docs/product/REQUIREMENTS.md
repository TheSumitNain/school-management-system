# Product Requirements and Delivery Scope

**Status:** Approved product direction, reconciled 2026-11-13. This document describes intended product scope; it is not an implementation checklist.

## Product goal

Build a commercial multi-tenant School Management System for an initial planning target of 25–30 schools with approximately 400–500 students per school. These are planning assumptions, not proven capacity limits.

## Delivery phases and immediate priority

### Phase 1 — School web application and dashboards (current priority)

Complete the web application for:
- Platform Admin: global platform operations and school onboarding.
- School Admin: administration and day-to-day operation of exactly one school.
- Teachers and approved school staff: role- and assignment-scoped school workflows.

**The immediate implementation priority is a usable School Admin workspace backed by real school data.** Follow the exact dependency-driven sequence in [Phase 1 School Operations — Task-by-Task Build Order](PHASE1-WORKFLOW-ORDER.md). Existing frontend implementation is in progress; do not treat a page's presence as acceptance.

Do not divert the current Phase 1 workstream into Parent OTP, SMS delivery or parent-facing login.

### Phase 2 — Parent and Driver experiences

Parent and Driver mobile/web experiences are deferred until the Phase 1 web application and school dashboards are complete.

- Parent OTP login and OTP/SMS delivery are explicitly **not to be implemented now**.
- Parent-facing workflows and interfaces are Phase 2.
- Preserve already-approved parent/guardian relationship foundations where they already exist; do not expand the OTP, SMS, parent-authentication or parent-interface workstream without a new explicit approval.
- Parent access is intended to be read-only initially.

## Initial product boundaries

- Academic classes 1–12 are the initial default; schools may customize their academic structure.
- One shared database/schema is the initial deployment direction, with strict server-enforced school ownership.
- No free trial is included in the initial commercial scope.
- Initial exclusions: payment gateway, native parent app during Phase 1, live bus GPS and advanced AI analytics.
- Future-proofing must not create speculative tables or migrations.

## Core domain areas

- School onboarding and configuration.
- Staff accounts, roles and permissions.
- Academic years, classes, sections and subjects.
- Teacher-subject and class-in-charge assignments.
- Student admission, profiles, enrollment and promotion/history. A permanent school-scoped student identity is separate from academic-year enrollment; each academic year receives a new enrollment-specific admission number. Initial admission creates the student and first enrollment transactionally; returning students receive a new yearly enrollment without duplicate student identities. Student profile fields are organized into independently editable sections and may be completed incrementally.
- Student addresses use reusable hierarchical locations with separate current/permanent address support. Guardian relationships reuse the existing school-scoped foundation; medical, physical and document sections require dedicated data handling and field-level authorization.
- Class/year curriculum defines applicable compulsory and optional subjects; student subject choices are stored against the specific academic-year enrollment so historical selections remain stable.
- Attendance.
- Planned next: timetable, examinations, results and remaining school workflows.
- Guardian/parent relationship foundations are retained for Phase 2; parent OTP authentication and delivery are deferred.

## Academic structure lifecycle and safe changes

- Academic years are never hard-deleted. Closing an academic year is the archive operation; closed years remain available for historical enrollment, attendance, curriculum and reporting references and cannot be edited through the normal setup UI.
- Classes, sections and subjects use active/inactive lifecycle states rather than destructive deletion when they are referenced by school data. Inactivation must not silently delete, rewrite or cascade student records, enrollments, curriculum assignments, subject selections, teacher assignments or historical snapshots.
- Before a consequential inactivation/removal, the School Admin workflow must inspect dependencies and show the exact affected-record counts and useful affected-record details where available. The admin must explicitly confirm the displayed impact; cancellation must leave data unchanged.
- Section inactivation must call the section dependency-impact preview before any write. The preview includes the exact active and historical enrollment counts, active curriculum count, active teacher-subject assignment count, active class-in-charge count, and the active student list (name and admission number). The admin can return to editing without saving or explicitly confirm the impact.
- Class inactivation must call the class dependency-impact preview before any write. The preview includes active/inactive section counts, active/historical enrollment counts, active curriculum count, active teacher-subject and class-in-charge assignment counts, and the affected active-student list with admission number, section and academic year. Confirmation only inactivates the class; it never silently inactivates its sections or alters student/assignment history.
- Subject inactivation must call the subject dependency-impact preview before any write. The preview includes curriculum references, active teacher-subject assignments, active and historical student selections, and affected students with admission number, class, section and academic year. Confirmation only inactivates the catalogue subject; it must not remove curriculum subjects, revoke assignments or clear student selections.
- The backend is authoritative: re-check the dependency impact within the write transaction, reject stale confirmations, enforce school ownership and live permissions, and write the administrative audit event in the same transaction as the state change.
- Subject inactivation must preview all curriculum subject references, active teacher-subject assignments, active and historical student subject selections, and the affected student list (name, admission number, class, section and academic year). Explicit confirmation only inactivates the master subject; it must not silently remove curriculum rows, revoke assignments or alter student selections. Lock the subject and its dependent rows, re-check the impact fingerprint transactionally, reject stale confirmations and audit the inactivation in the same transaction.
- An inactive master subject cannot be newly added to a curriculum, newly selected by a student, or receive a new teacher-subject assignment. A new enrollment cannot be assigned a curriculum while that curriculum still references an inactive master subject; the School Admin must remove the subject through the separate curriculum impact-preview and confirmation flow first. Existing assigned enrollments and their snapshots remain intact, and an already-selected inactive optional subject can be retained or removed but cannot be newly selected.
- Never infer a successful deletion from an inactive status. UI wording and notifications must say “deactivated” or “archived” accurately and explain that historical records are retained.
- Class/section/subject changes must preserve enrollment-time snapshots and existing student curriculum choices. Curriculum subject removal follows its separately specified exact affected-student preview and explicit confirmation flow.

## Non-functional requirements

- Default-deny authorization with backend enforcement on every protected operation.
- School, resource, assignment and field-level boundaries.
- Auditable security-sensitive and administrative actions.
- Version-controlled additive migrations with disposable PostgreSQL integration testing.
- No secrets or real student data in source control.
- Test and load-test against realistic target scale before production capacity claims.

## Implementation sequence

Use [Phase 1 School Operations — Task-by-Task Build Order](PHASE1-WORKFLOW-ORDER.md) for the mandatory task-by-task creation order and dependency gates. Use [Project Status](../project/STATUS.md) for actual implementation and verification status; requirements and roadmap items are not evidence of completed code.

## Source of truth

Use [Project Status](../project/STATUS.md) for what is actually implemented. Use [Phase 1 RBAC](PHASE1-RBAC.md) for account and authorization rules. Product requirements must not be represented as completed code.
