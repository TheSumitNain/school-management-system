# Backend Security

## Implemented controls

- bcrypt password hashing and password policy checks, including common/breached-password checks.
- Principal-aware authentication and database-backed token-version/status validation.
- Short-lived JWT access tokens held in browser memory and opaque refresh tokens stored as hashes in the database.
- Refresh tokens are delivered only through a persistent HttpOnly, SameSite=Lax cookie (Secure in production); they are not stored in browser JavaScript storage.
- Refresh-token rotation, replay/family revocation and transaction-lock concurrency protections.
- Cross-tab refresh serialization and logout/session-expiry notifications; every protected API request still validates the active server-side session family.
- Protected-request validation of active refresh-session families; revoked families and principal/token-version mismatches are rejected immediately.
- Mandatory TOTP MFA for Platform Admin and School Admin on every fresh login; background access-token refresh and restoration of an existing session do not repeat MFA. Recovery codes, failure lockout and audit events remain enabled.
- Versioned AES-GCM TOTP secret encryption with previous-key support and lazy re-encryption after successful verification.
- Login lockout, IP rate limiting and fail-closed behavior when required security controls are unavailable.
- Live database-backed RBAC, scoped delegation, ALLOW/DENY overrides and route-level permission checks.
- Staff and Platform Admin password lifecycle, account status changes and session/token invalidation.
- Refresh sessions have no server-side idle timeout or absolute lifetime. They remain valid until manual logout or a security/account event revokes them.
- The persistent HttpOnly refresh cookie uses a 400-day Max-Age and is renewed on every successful refresh. Browsers may enforce stricter cookie-retention limits; persistence across browser/device restarts is therefore subject to browser policy.
- The browser refresh cookie is scoped to `/api/auth`, uses `SameSite=Lax`, and is `Secure` in production. Reverse proxies must preserve the `/api/auth` browser-facing path.

## Authorization requirements

Every protected operation must enforce:
1. authenticated principal and active session family;
2. current effective permission;
3. active school and school ownership where applicable;
4. resource ownership and exact teacher assignment scope;
5. approved field-level disclosure.

A route permission alone is not sufficient. Repositories must not trust client-supplied school IDs or resource ownership.

### Administrator scope

- School Admin is the full administrator for its own school. Its system role receives the approved school-operation permission bundle, including guardian/parent/sibling relationship management. Repository checks still bind every operation to the authenticated staff principal's school.
- Teacher and other limited roles do not inherit School Admin permissions. Guardian access is not granted to the Teacher system role.
- Platform Admin is a global platform principal, not a school-scoped staff principal. Existing platform routes authorize platform-level operations, while school business APIs currently require an authenticated school context. Do not bypass tenant checks or treat a Platform Admin token as a School Admin token. A reviewed selected-school administrative context is required before Platform Admin can operate on school business records across tenants; this is an explicit remaining implementation gap.

## Known coverage gap

The complete tenant/resource/assignment/field-level authorization model has not yet been implemented and verified across every business module. Selected academic, attendance and guardian routes have targeted checks and tests. New routes must not be treated as secure until their negative cross-school, unassigned-resource and sensitive-field cases are covered.

## Operational requirements before production

- Secret manager and coordinated key rotation.
- TLS, proxy and deployment hardening.
- Least-privilege database roles.
- Centralized security monitoring, alerting and incident response.
- Backup/restore drills and key/backup retention policy.
- Penetration testing and load/capacity testing.

## TOTP key rotation

Use `TOTP_ENCRYPTION_KEY`, `TOTP_ENCRYPTION_KEY_VERSION` and optional `TOTP_ENCRYPTION_PREVIOUS_KEYS` from the environment. Generate random 32-byte keys, Base64-encode them, and store them in a secret manager. Never write real key material into source, docs or Git. Keep previous keys until no stored credentials or retained backups depend on them. Follow the deployment procedure documented in source-controlled operational guidance and test restore before retiring keys.

See [Project Status](../project/STATUS.md) for the current implementation boundary.
