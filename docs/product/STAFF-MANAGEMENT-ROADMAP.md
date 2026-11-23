[object Object]

## 2026-11-23 implementation checkpoint — attendance authorization ordering

- Moved attendance-mark permission evaluation ahead of staff target lookup, within the same transaction, and added a PostgreSQL regression test that ensures unauthorized callers cannot infer target existence.
- Backend CI [45140616380](https://github.com/TheSumitNain/school-management-system/actions/runs/45140616380) **PASS** on `cbb64f3f` (migrations, Go/PostgreSQL integration and `go vet`).
- No persistent database change.
