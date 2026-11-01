# V4 Handler and Middleware Integration

This patch replaces the legacy user/school-ID JWT contract with the V4 principal-ID contract.

## Handler contract changes

- Staff login accepts `username` and `password`; `school_id` is intentionally removed because staff usernames are globally unique and case-insensitive.
- `AuthService.AuthenticateStaff(ctx, username, password)` must resolve the staff principal and verify the password against the linked staff account.
- Access-token subject is `auth_principals.id`; claims include principal type and token version only.
- Refresh rotation returns `*models.AuthPrincipal`, not `*models.User`.
- Responses return a `principal` DTO, not a legacy user DTO.
- Parent OTP and platform-admin MFA login are separate authentication flows and are not implemented by this staff password handler.

## Middleware dependencies

`AuthMiddleware` now requires three dependencies:

1. `AccessTokenValidator`: validates signature, issuer, time claims and principal claims.
2. `PrincipalLoader`: loads current principal state on every protected request. Its SQL implementation must ensure the principal is active, staff's school is active, parent mobile is verified, and subtype data is consistent.
3. `PermissionLoader`: resolves effective permissions from current RBAC data on every protected request, applying scope, overrides and delegation ceilings.

The principal loader should return `middleware.ErrPrincipalNotAuthenticated` for a missing/inactive/ineligible principal (401). Unexpected database or infrastructure errors return 500. Permission resolution failure also returns 500 and fails closed.

## Route usage

```go
protected := router.Group("/api")
protected.Use(middleware.AuthMiddleware(tokenService, principalRepository, permissionService))
protected.GET("/students", middleware.RequirePermission("students.read_basic"), studentHandler.List)
```

Do not use `ContextSchoolIDKey` as a substitute for checking resource ownership or teacher/parent scope in the service/repository query. Platform admins and parents may have no school ID in the principal context.

## Required follow-up before this compiles in the complete application

- Implement `AuthenticateStaff` in the authentication service using globally unique username lookup and V4 principal linkage.
- Implement `LoadPrincipalForAuthentication` with current principal, subtype, and school status checks.
- Implement `LoadEffectivePermissions` from the approved V4 RBAC schema.
- Update router construction and all handler/middleware fakes and tests to the new interfaces.
- Update `/auth/me` to return principal identity and update API docs/client contracts.
- Keep parent OTP and mandatory Platform/School Admin MFA as separate login workflows; do not bypass MFA by calling this staff login endpoint.
