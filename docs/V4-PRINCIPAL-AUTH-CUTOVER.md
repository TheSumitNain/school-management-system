# V4 principal authentication cutover — code patch

This patch adds principal-aware primitives without deleting the legacy user-based repository. It is intended to be applied only after migration `000004_add_principal_rbac_foundation` has been independently validated and applied to a disposable PostgreSQL 18 database.

## Included

- `models.AuthPrincipal`: canonical principal identity returned by principal refresh rotation.
- `services.TokenService`: JWT subject is `auth_principals.id`; claims contain `principal_type` and `token_version`. No school ID, roles, or permissions are trusted from the JWT.
- `services.SessionService`: refresh issue/rotate/revoke APIs accept principal IDs and use principal-aware repository methods.
- `repository.PrincipalRefreshSessionRepository`: principal-bound persistence using `refresh_sessions.principal_id`, principal-wide advisory locking, family replay revocation, and transactional replacement.
- Focused token-service tests for principal types and invalid identities.

The legacy `RefreshSessionRepository` is intentionally not replaced in this patch, so its existing repository tests and callers are not silently broken. New principal-aware handlers must use `NewPrincipalRefreshSessionRepository`.

## Required call-site cutover (not included because handler/middleware source was not part of the source supplied for this patch)

1. Staff authentication must load `users.principal_id` and the matching `auth_principals.token_version`; do not derive a principal ID from `users.id`.
2. Platform-admin authentication must resolve `platform_admin_accounts` and its principal; parent login must resolve a verified `parent_accounts` principal only after OTP verification.
3. On successful authentication, call `GenerateAccessToken(principal.ID, principal.TokenVersion, principal.Type)` and `SessionService.Issue(ctx, principal.ID)`.
4. On refresh, `SessionService.Rotate` returns `*models.AuthPrincipal`; generate the replacement access token from that principal.
5. Middleware must validate the JWT, parse the subject as principal ID, then load the current principal on every protected request and compare status, principal type, and `token_version`. For staff, also load active school membership/status. A syntactically valid token alone is not sufficient.
6. Resolve effective permissions from the live database on every protected request. Do not cache permissions in JWT claims. Enforce resource-level teacher assignments and parent-child links in the relevant service/repository queries.
7. Update handler, middleware, API, and tests together. Existing legacy callers of `GenerateAccessToken(userID, schoolID)` must be migrated; this is intentionally a breaking security-boundary change.

## Database assumptions

This code expects the V4 schema columns/tables: `auth_principals(id, principal_type, status, token_version)`, `users(principal_id, school_id, username)`, `schools(status)`, `platform_admin_accounts(principal_id)`, `parent_accounts(principal_id, mobile_verified_at)`, and `refresh_sessions(principal_id, user_id nullable, family_id, token_hash, expires_at, revoked_at)`. `family_id` must have the existing v3 default for newly issued sessions.

## Verification status

- `gofmt` applied to included Go files.
- Not compiled against the user's full repository: the exact current `go.mod`, handler, middleware, and test call sites are not in this patch source set.
- Not run against PostgreSQL 18. Run unit, repository integration, refresh concurrency, replay, logout/rotation race, and migration acceptance tests on a disposable database before considering adoption.
- Do not apply the migration or use this patch against production yet.
