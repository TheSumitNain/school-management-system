import { readFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";

test("monthly staff report downloads attendance, approved leave and holidays as separate CSV rows", async ({ page }) => {
  const permissions = ["users.read", "staff.attendance.read", "staff.leave.read", "staff.holidays.read"];

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: {
        access_token: "staff-report-token",
        token_type: "Bearer",
        principal: { principal_id: 702, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 702,
        principal_type: "staff",
        school_id: 10,
        username: "staff-report-admin",
        permissions,
      } });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      await route.fulfill({ json: { staff: [{
        id: 702, principal_id: 702, school_id: 10, username: "teacher.report",
        employee_code: "T-702", full_name: "=1+1 Report Teacher", designation: "Teacher",
        employment_status: "active", profile_complete: true, status: "active",
        must_change_password: false,
      }] } });
      return;
    }
    if (path === "/school/staff-attendance/month" && request.method() === "GET") {
      const month = url.searchParams.get("month") ?? "2026-11";
      await route.fulfill({ json: { month, data: [{
        staff_principal_id: 702, employee_code: "T-702", full_name: "=1+1 Report Teacher",
        designation: "Teacher", present_days: 2, absent_days: 1, late_days: 0, half_days: 1,
        records: [
          { id: 1, date: month + "-03", status: "present", remarks: "=1+1" },
          { id: 2, date: month + "-04", status: "absent", remarks: "Approved medical leave is separate" },
          { id: 3, date: month + "-05", status: "half_day", remarks: null },
        ],
      }] } });
      return;
    }
    if (path === "/school/staff-leave/requests" && request.method() === "GET") {
      await route.fulfill({ json: { data: [{
        id: 81, staff_principal_id: 702, employee_code: "T-702", staff_name: "Report Teacher",
        leave_type_id: 4, leave_type_name: "Medical Leave", start_date: "2026-11-04",
        end_date: "2026-11-04", duration_type: "full_day", reason: "Medical appointment",
        status: "approved", created_at: "2026-11-01T09:00:00Z",
      }] } });
      return;
    }
    if (path === "/school/holidays" && request.method() === "GET") {
      await route.fulfill({ json: { data: [{
        id: 91, holiday_date: "2026-11-14", name: "Children's Day", description: null, is_active: true,
      }] } });
      return;
    }

    await route.fulfill({ status: 404, json: { message: `No fixture for staff monthly report: ${request.method()} ${path}` } });
  });

  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");
  await page.getByLabel("Username").fill("staff-report-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await page.getByRole("button", { name: "Monthly summary", exact: true }).click();

  const month = await page.getByLabel("Attendance month").inputValue();
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download monthly report" }).click();
  const download = await downloadPromise;

  expect(download.suggestedFilename()).toBe(`staff-attendance-${month}.csv`);
  const filePath = await download.path();
  expect(filePath).toBeTruthy();
  const csv = await readFile(filePath!, "utf8");
  expect(csv).toContain("Attendance summary");
  expect(csv).toContain("Report Teacher");
  expect(csv).toContain("Attendance entry");
  expect(csv).toContain("'=1+1 Report Teacher");
  expect(csv).toContain("Approved leave");
  expect(csv).toContain("Medical Leave");
  expect(csv).toContain("School holiday");
  expect(csv).toContain("Children's Day");
});
