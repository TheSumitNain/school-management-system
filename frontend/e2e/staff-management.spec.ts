import { expect, test, type Page, type Route } from "@playwright/test";
import { schoolDateValue } from "../src/shared/date/schoolDate";

const staffUser = {
  principal_id: 101,
  principal_type: "staff",
  status: "active",
  school_id: 10,
};

async function signInAndOpenStaff(page: Page, staffRows: Array<Record<string, unknown>>, onStaffMutation?: (path: string, route: Route) => Promise<void>) {
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "staff-e2e-token", token_type: "Bearer", principal: staffUser } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({
        json: {
          ...staffUser,
          username: "staff-e2e-admin",
          permissions: ["users.read", "users.create", "users.update"],
        },
      });
      return;
    }
    if (path === "/school/staff" && method === "GET") {
      await route.fulfill({ json: { staff: staffRows } });
      return;
    }
    if (onStaffMutation && (path === "/school/staff" && method === "POST" || /^\/school\/staff\/\d+$/.test(path) && method === "PATCH")) {
      await onStaffMutation(path, route);
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("staff-e2e-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();
}

test("staff creation uses the shared IST date picker and submits its selected date", async ({ page }) => {
  const created: Array<Record<string, unknown>> = [];
  const rows: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    created.push(payload);
    rows.push({
      id: 1, principal_id: 201, school_id: 10, username: payload.username,
      employee_code: payload.employee_code, full_name: payload.full_name,
      designation: payload.designation, mobile_number: payload.mobile_number,
      joining_date: payload.joining_date, employment_status: "active",
      profile_complete: true, status: "active", must_change_password: true,
      created_at: "2026-11-21T00:00:00.000Z",
    });
    await route.fulfill({ json: { staff: rows[0] } });
  });

  await expect(page.getByRole("heading", { name: "Staff management" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Search staff by name, code, designation or username" })).toBeVisible();
  await page.getByRole("button", { name: "Add staff" }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Login username").fill("teacher.one");
  await dialog.getByLabel("Full name").fill("Teacher One");
  await dialog.getByLabel("Employee code").fill("EMP-001");
  await dialog.getByLabel("Designation").fill("Class Teacher");
  await dialog.getByLabel("Mobile number").fill("9876543210");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Joining date" }).click();
  await dialog.getByRole("button", { name: "Today", exact: true }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => created.length).toBe(1);
  expect(created[0].joining_date).toBe(schoolDateValue());
  await expect(dialog).toHaveCount(0);
  await expect(page.locator('input[type="date"]')).toHaveCount(0);
});

test("staff profile edit uses the shared employment-status select and sends the selected value", async ({ page }) => {
  const updates: Array<Record<string, unknown>> = [];
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 301, school_id: 10, username: "teacher.two",
    employee_code: "EMP-002", full_name: "Teacher Two", designation: "Science Teacher",
    mobile_number: "9876543211", joining_date: "2026-04-01",
    employment_status: "active", profile_complete: true, status: "active",
    must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    updates.push(route.request().postDataJSON() as Record<string, unknown>);
    await route.fulfill({ json: { staff: { ...rows[0], ...(updates[0] ?? {}) }, message: "Updated" } });
  });

  await page.getByRole("button", { name: "Edit employee profile for teacher.two" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit employee profile" });
  await expect(dialog.getByRole("button", { name: "Joining date" })).toContainText("01 Apr 2026");
  await dialog.getByRole("button", { name: "Employment status" }).click();
  await dialog.getByRole("listbox", { name: "Employment status" }).getByRole("option", { name: "Inactive" }).click();
  await dialog.getByRole("button", { name: "Save changes" }).click();

  await expect.poll(() => updates.length).toBe(1);
  expect(updates[0].employment_status).toBe("inactive");
  await expect(dialog).toHaveCount(0);
  await expect(page.locator("select")).toHaveCount(0);
});
