# SchoolDesk Frontend

React + TypeScript + Vite foundation for the Phase 1 school web application.

## Requirements
- Node.js 22.12 or newer
- npm 10 or newer
- Backend API running at http://127.0.0.1:8081

## Run locally (Windows CMD)
From the repository root:
1. cd /d D:\school-management-system\frontend
2. copy .env.example .env
3. npm install
4. npm run dev

Open http://127.0.0.1:5173. Vite proxies /api requests to the local Go API.

## Implemented foundation
- Staff and Platform Admin sign-in.
- MFA verification and enrollment challenge.
- Session-scoped token storage, refresh rotation and logout.
- /auth/me identity and live permission bootstrap.
- Temporary-password change gate.
- Permission-aware navigation.
- Overview metrics assembled from permitted existing APIs.
- Read-only staff and academic structure views.

The browser permission list is only a navigation hint. The backend remains authoritative for every API operation.

Parent OTP/SMS, Parent dashboard and Driver interface are explicitly deferred.
