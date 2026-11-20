import { expect, test } from "@playwright/test";

test("school workspace pages stay within narrow, tablet, and desktop viewports", async ({ page }) => {
  const permissions = [
    "users.read", "roles.read", "academic_years.read", "classes.read", "subjects.read",
    "teacher_assignments.read", "students.read_basic", "students.read_sensitive",
    "attendance.read", "timetable.read",
  ];

  await page.route("**/api/**", async route => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api/, "");
    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && route.request().method() === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-responsive-token",
        token_type: "Bearer",
        principal: { principal_id: 112, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 112, principal_type: "staff", school_id: 10,
        username: "e2e-responsive-admin", permissions,
      } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: "No fixture for responsive smoke test." } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-responsive-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();

  const pages = [
    { label: "Staff", heading: /staff/i },
    { label: "Roles & permissions", heading: /roles/i },
    { label: "Academics", heading: /academic structure/i },
    { label: "Students", heading: /students/i },
    { label: "Attendance", heading: /attendance/i },
    { label: "Timetable", heading: /timetable/i },
    { label: "Settings", heading: /settings/i },
  ];
  const widths = [320, 360, 390, 768, 1024, 1440];

  for (const width of widths) {
    await page.setViewportSize({ width, height: 900 });
    for (const item of pages) {
      if (width <= 740) await page.locator(".mobile-menu").click();
      await page.getByRole("button", { name: item.label, exact: true }).click();
      await expect(page.getByRole("heading", { level: 1, name: item.heading })).toBeVisible();
      await expect.poll(
        () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
        { message: `${item.label} should not cause page-level horizontal overflow at ${width}px` },
      ).toBe(true);
    }
  }

  await page.setViewportSize({ width: 844, height: 390 });
  for (const item of pages) {
    await page.getByRole("button", { name: item.label, exact: true }).click();
    await expect(page.getByRole("heading", { level: 1, name: item.heading })).toBeVisible();
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  }
});
