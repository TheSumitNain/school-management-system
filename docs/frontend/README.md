# Frontend Documentation

<badge>Planned — not implemented</badge>

The intended web stack is React + TypeScript + Vite. At the 2026-11-06 status reconciliation, this repository has no frontend source tree, package manifest or runnable frontend application.

## Phase 1 users and delivery priority

Phase 1 web users:
- School Admin
- Teachers and approved school staff
- Platform Admin

**Build the school-facing dashboard first.** Prioritize a usable School Admin dashboard and the role/assignment-aware staff and teacher workflows needed to operate the school. Add the Platform Admin web workflows as part of the Phase 1 web product.

Parent OTP login, OTP/SMS delivery, parent dashboards and Parent/Driver mobile interfaces are not part of the current frontend work. They are Phase 2 and must wait until the Phase 1 school web application is complete.

## Integration contract

- The backend is the source of truth for identity, permissions and resource access.
- `GET /auth/me` returns principal identity and, for Staff and Platform Admin, the current sorted effective permission keys.
- Permission keys may guide navigation visibility only. Every API operation must still be authorized server-side.
- The client must handle authentication expiry, logout, MFA challenges and temporary-password restrictions.
- Do not trust school IDs, role labels, permission lists or selected student IDs supplied by the browser as authorization proof.

## Work required before and during frontend implementation

1. Agree on frontend folder/package conventions and supported Node version.
2. Establish typed API client, environment configuration and secure session handling.
3. Build the School Admin dashboard shell and its core school-operating workflows first.
4. Implement teacher/staff views with the backend's assignment and field-level limits.
5. Implement Platform Admin workflows for platform operations and school onboarding.
6. Implement each workflow against documented API contracts with loading, empty, validation and error states.
7. Verify browser workflows and ensure backend authorization remains effective when UI controls are bypassed.

## Explicitly deferred

- Parent OTP login and OTP/SMS delivery.
- Parent dashboard and parent-facing web/mobile workflows.
- Driver mobile experience.

Do not create frontend routes, forms, mock authentication or placeholder flows for these deferred features as part of the current Phase 1 scope.

No frontend commands are currently available because the frontend has not been created.
