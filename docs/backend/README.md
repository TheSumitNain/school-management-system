# Backend Documentation

The backend is the currently implemented application surface. It uses Go, Gin, PostgreSQL and pgx/pgxpool.

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
