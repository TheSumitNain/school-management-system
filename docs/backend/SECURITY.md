# Backend Security

## Implemented controls

- bcrypt password hashing and password policy checks, including common/breached-password checks.
- Principal-aware authentication and database-backed token-version/status validation.
- Short-lived JWT access tokens and opaque refresh tokens stored as hashes.
- Refresh-token rotation, replay/family revocation and transaction-lock concurrency protections.
- Protected-request validation of active refresh-session families.
- Mandatory TOTP MFA for Platform Admin and School Admin, recovery codes, failure lockout and audit events.
- Versioned AES-GCM TOTP secret encryption with previous-key support and lazy re-encryption after successful verification.
- Login lockout, IP rate limiting and fail-closed behavior when required security controls are unavailable.
- Live database-backed RBAC, scoped delegation, ALLOW/DENY overrides and route-level permission checks.
- Staff and Platform Admin password lifecycle, account status changes and session/token invalidation.
- Per-device 30-minute idle timeout for Platform Admin and active system-role School Admin sessions.

## Authorization requirements

Every protected operation must enforce:
1. authenticated principal and active session family;
2. current effective permission;
3. active school and school ownership where applicable;
4. resource ownership and exact teacher assignment scope;
5. approved field-level disclosure.

A route permission alone is not sufficient. Repositories must not trust client-supplied school IDs or resource ownership.

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
