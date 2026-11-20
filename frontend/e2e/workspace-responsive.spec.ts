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
    { path: "/staff", heading: /staff management/i },
    { path: "/roles", heading: /roles/i },
    { path: "/academics", heading: /academic structure/i },
    { path: "/students", heading: /students/i },
    { path: "/attendance", heading: /attendance/i },
    { path: "/timetable", heading: /timetable/i },
    { path: "/settings", heading: /settings/i },
  ];
  const widths = [320, 360, 390, 768, 1024, 1440];

  for (const width of widths) {
    await page.setViewportSize({ width, height: 900 });
    for (const item of pages) {
      await page.evaluate(path => { window.history.pushState({}, "", path); window.dispatchEvent(new PopStateEvent("popstate")); }, item.path);
      await expect(page.getByRole("heading", { level: 1, name: item.heading })).toBeVisible();
      await expect.poll(
        () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
        { message: `${item.path} should not cause page-level horizontal overflow at ${width}px` },
      ).toBe(true);
      if (item.path === "/students") {
        await expect(page.getByRole("group", { name: "Student management views" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Student directory" })).toBeVisible();
        await expect(page.getByRole("button", { name: "Enrollments" })).toBeVisible();
        await expect(page.getByRole("textbox", { name: "Search students" })).toBeVisible();
        const studentControlsFit = await page.evaluate(() => {
          return [".student-tabs", ".student-toolbar", ".student-search"].every(selector => {
            const element = document.querySelector(selector);
            if (!element) return false;
            const rect = element.getBoundingClientRect();
            return rect.left >= 0 && rect.right <= window.innerWidth + 1;
          });
        });
        expect(studentControlsFit, `student navigation and filters should fit at ${width}px`).toBe(true);
      }

      if (item.path === "/settings") {
        await expect(page.getByRole("heading", { name: "Appearance mode" })).toBeVisible();
        await expect(page.locator(".appearance-option")).toHaveCount(4);
        await expect(page.locator(".palette-option")).toHaveCount(15);
        const settingsControlsFit = await page.evaluate(() => {
          return [...document.querySelectorAll(".appearance-option, .palette-option")].every(element => {
            const rect = element.getBoundingClientRect();
            return rect.width > 0 && rect.left >= 0 && rect.right <= window.innerWidth + 1;
          });
        });
        expect(settingsControlsFit, `appearance and palette options should fit at ${width}px`).toBe(true);
      }
    }
  }

  await page.setViewportSize({ width: 844, height: 390 });
  for (const item of pages) {
      await page.evaluate(path => { window.history.pushState({}, "", path); window.dispatchEvent(new PopStateEvent("popstate")); }, item.path);
    await expect(page.getByRole("heading", { level: 1, name: item.heading })).toBeVisible();
    await expect.poll(
      () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      { message: `${item.path} should not cause page-level horizontal overflow in landscape` },
    ).toBe(true);
  }
});
