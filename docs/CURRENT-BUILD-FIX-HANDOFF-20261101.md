# Current Build-Fix Handoff — 2026-11-01

## Purpose of this checkpoint
This checkpoint addresses the V4 Go application cutover compile failures reported by the user. It is deliberately limited to wiring principal-based authentication, sessions, middleware, repository loaders, and their test doubles. It does not complete the overall RBAC feature set or correct/execute migration v4.

## Source and backup
- Source: user-uploaded `school-management-system-current.zip`.
- A byte-for-byte backup of that uploaded ZIP was created separately as `school-management-system-current-BACKUP-20261101.zip`.
- The original uploaded snapshot and backup were not modified.
- The corrected project snapshot is a separate working copy.
- `.git`, `.env`, `node_modules`, `dist`, and log files are not included in the corrected archive.

## Go cutover changes in this checkpoint
1. `cmd/api/main.go`
   - Uses `PrincipalRefreshSessionRepository` rather than the legacy user-ID refresh repository.
   - Constructs `PrincipalAuthorizationRepository` and wires it as both live-principal and effective-permission loader.
   - `/auth/me` uses principal context keys and returns principal identity fields.
2. `internal/services/auth_service.go`
   - Adds `AuthenticateStaff`, authenticating globally unique staff usernames against the canonical principal and existing password hash.
   - Maps inactive staff/school states to service-level errors.
3. `internal/repository/user_repository.go`
   - Adds `GetStaffPrincipalByUsername`, joining users, auth_principals and schools.
4. `internal/repository/principal_authorization_repository.go`
   - Adds current principal/subtype/status loading.
   - Adds a fail-closed permission resolver: built-in system-role grants only; custom-role grants and ALLOW overrides are deliberately ignored because current V4 schema does not bind them to the exact delegation that authorized them. Explicit DENY overrides are applied.
   - Enforces the approved School Admin permission allowlist at runtime, preventing Guardian permissions and future school-scope catalogue rows from being inherited from the draft migration's broad seed query.
5. `internal/models/auth_principal.go` and `internal/middleware/auth.go`
   - Share the principal-authentication sentinel without introducing a package cycle.
6. Handler, middleware, API and session-service tests/fakes
   - Updated to the principal-based token/session contracts and new middleware dependencies.

## Important security and completeness limitations
- Migration `000004_add_principal_rbac_foundation.up.sql` remains a draft with previously identified delegation/audit/permission-catalogue defects. DO NOT RUN IT.
- Principal authorization resolver is intentionally conservative, not the final RBAC implementation. Custom role grants and user ALLOW overrides remain disabled until exact delegation linkage is represented and validated in the schema and service layer.
- `RequirePermission` exists but there are no Phase 1 business-resource routes wired to it yet. This checkpoint does not implement student/attendance/academic authorization endpoints.
- Platform Admin/School Admin MFA, parent OTP login, failed-login lockout, rate limiting, and full audit event writing are not implemented by this checkpoint.
- Staff login queries V4 `auth_principals` and therefore requires the V4 identity migration to exist in a database. The migration is not applied here.

## Verification status
- All edited Go files were run through `gofmt`; no Go formatting/parser errors were reported.
- A full `go test ./...` and `go vet ./...` could NOT be run in the review environment. The only installed compiler was Go 1.23.2, while this project requires Go 1.27.1; module dependencies were also not cached and network module lookup was unavailable.
- Therefore this checkpoint is a compile-error correction candidate, NOT a claim that the full suite has passed. The user must run the commands below on the Windows development machine with Go 1.27.1.

## Required Windows verification
Run from CMD:

```cmd
cd /d D:\school-management-system\backend
gofmt -w cmd\api\main.go cmd\api\main_test.go internal\handlers\auth_handler.go internal\handlers\auth_handler_test.go internal\middleware\auth.go internal\middleware\auth_test.go internal\models\auth_principal.go internal\repository\user_repository.go internal\repository\principal_authorization_repository.go internal\services\auth_service.go internal\services\session_service_test.go
go test -count=1 ./...
go vet ./...
```

Do not run the migration. Do not commit or push until the test output and full Git diff are reviewed.

## Next-chat instruction
Continue from this checkpoint. Do not ask the user to upload individual files or recreate the ZIP. First inspect this handoff and the current build-fix package. The immediate next action is to review the user's `go test -count=1 ./...` output, correct any remaining compile/test errors, and only then prepare a separate final RBAC/migration correction pass.
