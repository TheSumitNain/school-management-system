[object Object]

## 2026-11-23 implementation checkpoint — permission override test data

- Aligned the permission-override browser fixture with the actual permission catalogue by using delegable `students.read_basic` rather than non-delegable `staff.attendance.read`. No application behavior changed.
- Frontend CI [45138964793](https://github.com/TheSumitNain/school-management-system/actions/runs/45138964793) **PASS** (production build and 86 Playwright tests); Backend CI [45138964862](https://github.com/TheSumitNain/school-management-system/actions/runs/45138964862) **PASS** (full Go/PostgreSQL integration and `go vet`), both on `4ddcf02c`.
