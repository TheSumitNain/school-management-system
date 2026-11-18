# Backend Documentation

The backend is the currently implemented application surface. It uses Go, Gin, PostgreSQL and pgx/pgxpool.

## School calendar timezone

The current India-focused release uses `Asia/Kolkata` (IST) as the system-wide school calendar timezone. Date-only business rules such as date-of-birth and physical-measurement future-date validation, and attendance past-date authorization, must use the shared `internal/schooltime` package rather than the host timezone or PostgreSQL `CURRENT_DATE`. Do not add a per-school or user-local timezone assumption without an explicit product decision. Before onboarding schools outside India, introduce a persisted per-school IANA timezone and migrate these rules consistently.

## Reference documents

- [API contract](API.md)
- [Database and migrations](DATABASE.md)
- [Security model and operational requirements](SECURITY.md)
- [Testing and verification](TESTING.md)

## Run locally

From Windows CMD:

```cmd
cd /d D:\school-management-system\backend
go test -count=1 -p 1 ./...
go vet ./...
```

To start the API, configure a local `backend\.env` using `backend\.env.example`, then follow the commands in the backend module. Never commit `.env` or real key material.

PostgreSQL integration tests are opt-in and must target a dedicated disposable/test database. Read [Testing](TESTING.md) before enabling them. Never run test migrations against production data.

## Scope warning

The backend has implemented authentication/RBAC and selected academic/attendance/guardian APIs, but it is not the complete school product. See [Project Status](../project/STATUS.md) for current completed and pending areas.
