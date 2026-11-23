[object Object]

## 2026-11-23 implementation checkpoint — full School Admin staff permission bundle

- The PostgreSQL role integration test now verifies the complete School Admin permission bundle for attendance, leave, holidays, substitutions and timetable/workload, including live allowlist filtering.
- Backend CI [45139881590](https://github.com/TheSumitNain/school-management-system/actions/runs/45139881590) **PASS** on `ecefdaf4` (Go tests, PostgreSQL integration and `go vet`).
- No persistent migration was applied.
