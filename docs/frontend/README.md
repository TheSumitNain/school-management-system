# Frontend Documentation

<badge>Planned — not implemented</badge>

The intended web stack is React + TypeScript + Vite. At the 2026-11-06 status reconciliation, this repository has no frontend source tree, package manifest or runnable frontend application.

## Phase 1 users

- Platform Admin
- School Admin
- Teachers and approved school staff

Parent and Driver mobile interfaces are Phase 2 and deferred.

## Integration contract

- The backend is the source of truth for identity, permissions and resource access.
- `GET /auth/me` returns principal identity and, for Staff and Platform Admin, the current sorted effective permission keys.
- Permission keys may guide navigation visibility only. Every API operation must still be authorized server-side.
- The client must handle authentication expiry, logout, MFA challenges and temporary-password restrictions.
- Do not trust school IDs, role labels, permission lists or selected student IDs supplied by the browser as authorization proof.

## Work required before and during frontend implementation

1. Agree on frontend folder/package conventions and supported Node version.
2. Establish typed API client, environment configuration and secure session handling.
3. Build role-aware application shells for Platform Admin, School Admin and Teacher.
4. Implement each workflow against documented API contracts with loading, empty, validation and error states.
5. Verify browser workflows and ensure backend authorization remains effective when UI controls are bypassed.

No frontend commands are currently available because the frontend has not been created.
