[object Object]

## 2026-11-23 implementation checkpoint — School Admin timetable permission grant

- Added a PostgreSQL integration assertion for School Admin effective `timetable.read` and `timetable.manage` permissions, alongside substitution permissions, verifying real grants through the runtime allowlist.
- Backend CI [45139554701](https://github.com/TheSumitNain/school-management-system/actions/runs/45139554701) **PASS** on `2ad43a9c` (Go tests, PostgreSQL integration and `go vet`).
- No persistent migration was applied.
