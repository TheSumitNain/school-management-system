import { expect, test } from "@playwright/test";

test("dashboard shows permission-scoped staff, class and current-year enrollment totals", async ({ page }) => {
  const requested: string[] = [];
  const permissions = ["users.read", "classes.read", "academic_years.read", "students.read_basic"];

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    requested.push(path + url.search);
    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-dashboard-token",
        token_type: "Bearer",
        principal: { principal_id: 151, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 151, principal_type: "staff", school_id: 10,
        username: "e2e-dashboard-admin", permissions,
      } });
      return;
    }
    if (path === "/school/staff") {
      await route.fulfill({ json: { staff: Array.from({ length: 7 }, (_, index) => ({ id: index + 1 })) } });
      return;
    }
    if (path === "/school/classes") {
      await route.fulfill({ json: { classes: Array.from({ length: 4 }, (_, index) => ({ id: index + 1 })) } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [
        { id: 6, name: "2025–26", is_current: false, status: "active" },
        { id: 7, name: "2026–27", is_current: true, status: "active" },
      ] } });
      return;
    }
    if (path === "/school/enrollments") {
      expect(url.searchParams.get("academic_year_id")).toBe("7");
      await route.fulfill({ json: { enrollments: Array.from({ length: 18 }, (_, index) => ({ id: index + 1 })) } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked dashboard request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-dashboard-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();

  const metrics = [
    ["Staff accounts", "7"],
    ["Classes", "4"],
    ["Academic years", "2"],
    ["Current-year enrollments", "18"],
  ] as const;
  for (const [label, value] of metrics) {
    const metric = page.locator(".stat").filter({ hasText: label });
    await expect(metric).toBeVisible();
    await expect(metric.locator("strong")).toHaveText(value);
  }
  expect(requested).toContain("/school/enrollments?academic_year_id=7");
});
