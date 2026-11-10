# System Architecture

## Current architecture

The project is a Go/Gin modular-monolith backend using PostgreSQL through pgx/pgxpool. The initial tenancy model uses one shared database and schema. School isolation is enforced by application authorization and school-scoped data access; the whole future product surface is not yet complete.

## Intended client architecture

- Web client: React + TypeScript + Vite (planned; source not yet present).
- Backend: Go + Gin HTTP API.
- Database: PostgreSQL.
- Mobile clients: deferred Phase 2; APIs should remain reusable without prematurely building mobile-specific infrastructure.

## Backend boundaries

The backend is organized around HTTP handlers, services, repositories, models, middleware and versioned migrations. Handlers validate/translate HTTP requests; services coordinate business workflows; repositories own database access and transactional invariants; middleware authenticates principals and resolves current authorization context.

## Identity and authorization

- Principals represent the authenticated identity type.
- Staff principals are bound to one school.
- Platform Admin principals are global platform identities.
- Parent foundations are retained for Phase 2.
- JWTs carry identity/session claims, not an authoritative snapshot of permissions.
- Protected requests reload principal/session state and effective permissions from PostgreSQL.
- Route permission checks are necessary but not sufficient: repositories/queries must also enforce school, resource, assignment and field boundaries.

## Session security

The backend uses short-lived JWT access tokens and opaque refresh tokens stored as hashes. Refresh rotation and replay detection are transactionally protected. Protected requests validate the active refresh-session family. Refresh sessions have no server-side idle timeout or absolute expiry and remain valid until manual logout or security/account revocation. The persistent HttpOnly refresh cookie uses a 400-day Max-Age and is renewed on successful refresh, subject to browser cookie-retention policies. Platform Admin and School Admin must complete TOTP MFA on every fresh login; background refresh of an existing session does not repeat MFA. Recovery codes and versioned TOTP encryption keys remain in use.

## Database change policy

- Schema changes are versioned migrations.
- Prefer additive, reversible changes where practical.
- Run migration/integration verification against disposable PostgreSQL databases.
- Do not apply migrations to persistent user databases without explicit approval.
- Never infer a local database version from CI.

## Tenant and assignment boundaries

Every school-owned resource must be scoped by authenticated school identity. Teacher access must additionally be scoped to assigned subjects/classes/sections and approved fields. Parent access, when Phase 2 begins, must be scoped to verified active guardian-child links. Platform-global access must be explicitly permissioned and must not implicitly expose school records.

## Architecture maturity

This architecture is not yet a production deployment blueprint. Deployment topology, database least privilege, observability, backup/restore, secrets management, load testing and disaster recovery still require explicit design and verification.
