# Product Requirements and Delivery Scope

**Status:** Approved product direction, reconciled 2026-11-06. This document describes intended product scope; it is not an implementation checklist.

## Product goal

Build a commercial multi-tenant School Management System for an initial planning target of 25–30 schools with approximately 400–500 students per school. These are planning assumptions, not proven capacity limits.

## Delivery phases and immediate priority

### Phase 1 — School web application and dashboards (current priority)

Complete the web application for:
- Platform Admin: global platform operations and school onboarding.
- School Admin: administration and day-to-day operation of exactly one school.
- Teachers and approved school staff: role- and assignment-scoped school workflows.

**The immediate implementation priority is the school-facing dashboard and its usable end-to-end workflows.** Build the dashboard on top of the existing backend contracts, completing the required school administration and staff workflows in a coherent, usable web product. Frontend implementation has not yet started.

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
- Student admission, profiles, enrollment and promotion/history.
- Attendance.
- Planned next: timetable, examinations, results and remaining school workflows.
- Guardian/parent relationship foundations are retained for Phase 2; parent OTP authentication and delivery are deferred.

## Non-functional requirements

- Default-deny authorization with backend enforcement on every protected operation.
- School, resource, assignment and field-level boundaries.
- Auditable security-sensitive and administrative actions.
- Version-controlled additive migrations with disposable PostgreSQL integration testing.
- No secrets or real student data in source control.
- Test and load-test against realistic target scale before production capacity claims.

## Source of truth

Use [Project Status](../project/STATUS.md) for what is actually implemented. Use [Phase 1 RBAC](PHASE1-RBAC.md) for account and authorization rules. Product requirements must not be represented as completed code.
