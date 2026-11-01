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

## Verification status — Windows Go 1.27.1 (2026-11-01)

The user ran the following commands from `D:\\school-management-system\\backend` and supplied their terminal output:

```cmd
gofmt -w cmd\\api\\main.go cmd\\api\\main_test.go internal\\handlers\\auth_handler.go internal\\handlers\\auth_handler_test.go internal\\middleware\\auth.go internal\\middleware\\auth_test.go internal\\models\\auth_principal.go internal\\repository\\user_repository.go internal\\repository\\principal_authorization_repository.go internal\\services\\auth_service.go internal\\services\\session_service_test.go
go test -count=1 -p 1 ./...
go vet ./...
```

**Observed result:**
- `gofmt` completed without reporting errors.
- `go test -count=1 -p 1 ./...` passed for `cmd/api`, `internal/config`, `internal/handlers`, `internal/middleware`, `internal/repository`, and `internal/services`.
- `cmd/devseed`, `cmd/migrate`, `internal/database`, and `internal/models` reported no test files.
- `go vet ./...` completed without diagnostics.

This verifies the Go package test suite and static analysis on the user's Windows environment. It does **not** establish that opt-in PostgreSQL integration tests actually ran against PostgreSQL, nor does it verify Migration V4. No PostgreSQL migration or integration-test execution was reported in this checkpoint.

## Git review status

The user reported local status:
- Modified: `APPLY-CHANGES.txt`, `backend/internal/repository/principal_refresh_session_repository.go`, `backend/internal/repository/refresh_session_repository_integration_test.go`.
- Untracked: `APPLY-CURRENT-BUILD-FIX.txt`, `APPLY-DOCS.txt`, `GO-FILES-LIST.txt`, `GO-RBAC-AUDIT-BUNDLE.txt`, `MIGRATIONS-FULL-AUDIT.txt`, `NEXT-CHAT-PROMPT.txt`, `fix_v4_delegation.py`, and `v4-corrective-patch/`.

The reviewed diff for the two Go files contains only gofmt import ordering and removal of extra blank lines; no functional code changes were observed. `APPLY-CHANGES.txt` contains stale verification instructions and says tests could not run, so it must not be committed unchanged. Preserve all untracked artifacts. Do not stage or commit the entire working tree.

## Required remaining delivery checks

1. Reconcile this handoff and project progress/changelog with the observed Windows test and vet results.
2. Pull documentation updates from `origin/main` without disturbing local modifications.
3. Review `git diff --check` and the exact staged diff.
4. Commit only the intended Go formatting changes and any explicitly reconciled documentation. Do not include unrelated untracked artifacts.
5. Confirm the push and remote synchronization.

Do not run Migration V4. Do not claim PostgreSQL integration tests or migration tests passed unless their actual execution output is supplied.

## Next-chat instruction

Continue from this verified checkpoint. The user's Windows Go package tests and `go vet` both passed. First finish the safe Git/documentation delivery review; preserve all modified and untracked local artifacts. Then continue the V4 source audit from C17, inspecting evidence before defining a finding. Migration V4 remains unapplied and must not be executed without the user's explicit approval.
