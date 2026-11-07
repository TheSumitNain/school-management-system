# SchoolDesk Frontend

React + TypeScript + Vite foundation for the Phase 1 school web application.

## Requirements

- Node.js 22.12 or newer
- npm 10 or newer
- Backend API running at http://127.0.0.1:8081

## Run locally (Windows CMD)

From the repository root:

```cmd
cd /d D:\school-management-system\frontend
copy .env.example .env
npm install
npm run dev
```

Open http://127.0.0.1:5173. Vite proxies `/api` requests to the local Go API.

## Build and verification

```cmd
npm run typecheck
npm run build
```

Run these after dependency or source changes. Do not treat a successful dev-server start as a production-build verification.

## Architecture

See [Frontend Architecture](../docs/frontend/ARCHITECTURE.md) for the feature-oriented target structure and state ownership rules.

Current foundation:

- Existing staff and Platform Admin sign-in, MFA verification/enrollment, session refresh/logout and `/auth/me` bootstrap are preserved.
- TanStack Query is mounted at the application root with conservative cache and retry defaults.
- Zustand stores the small global theme preference (Light, Dark or System).
- Semantic design tokens and theme overrides are centralized under `src/styles/`.
- Shared theme toggle, dashboard skeleton and application error boundary are available.
- Platform Admin management and school onboarding are wired only to endpoints currently registered by the backend.
- Platform Admin page code is split into a separate lazy-loaded chunk.
- School staff data is migrated to a feature API module and TanStack Query hook.

The existing dashboard and academic screens are being migrated incrementally. Do not bypass the existing authentication/session client or duplicate it in feature modules.

The browser permission list is only a navigation hint. The backend remains authoritative for every API operation.

Parent OTP/SMS, Parent dashboard and Driver interface are explicitly deferred.
