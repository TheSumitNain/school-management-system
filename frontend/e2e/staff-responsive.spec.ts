import { expect, test } from "@playwright/test";

test("Staff Management tabs remain usable without page overflow on mobile, tablet, desktop and landscape", async ({ page }) => {
  const permissions = [
    "users.read",
    "staff.attendance.read",
    "staff.leave.read",
    "staff.holidays.read",
  ];

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
        access_token: "staff-responsive-token",
        token_type: "Bearer",
        principal: { principal_id: 701, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 701,
        principal_type: "staff",
        school_id: 10,
        username: "staff-responsive-admin",
        permissions,
      } });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      await route.fulfill({ json: { staff: [] } });
      return;
    }
    if (path === "/school/staff-attendance" && request.method() === "GET") {
      await route.fulfill({ json: { date: url.searchParams.get("date"), data: [] } });
      return;
    }
    if (path === "/school/staff-leave/requests" && request.method() === "GET") {
      await route.fulfill({ json: { data: [] } });
      return;
    }
    if (path === "/school/staff-leave/types" && request.method() === "GET") {
      await route.fulfill({ json: { data: [] } });
      return;
    }
    if (path === "/school/holidays" && request.method() === "GET") {
      await route.fulfill({ json: { data: [] } });
      return;
    }

    await route.fulfill({ status: 404, json: { message: `No fixture for Staff responsive test: ${request.method()} ${path}` } });
  });

  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto("/");
  await page.getByLabel("Username").fill("staff-responsive-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  const widths = [320, 360, 390, 768, 1024, 1440];
  for (const width of widths) {
    await page.setViewportSize({ width, height: 900 });
    await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
    await expect.poll(
      () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      { message: `Staff Management should not cause page-level horizontal overflow at ${width}px` },
    ).toBe(true);

    const tabsFit = await page.locator(".staff-workspace-tabs").evaluate(element => {
      const rect = element.getBoundingClientRect();
      const overflowX = getComputedStyle(element).overflowX;
      return rect.left >= 0 && rect.right <= window.innerWidth + 1 &&
        (element.scrollWidth <= element.clientWidth + 1 || overflowX === "auto" || overflowX === "scroll");
    });
    expect(tabsFit, `Staff Management tabs should fit or scroll within ${width}px`).toBe(true);
  }

  await page.getByRole("button", { name: "Staff Directory" }).click();
  await expect(page.getByRole("heading", { name: "Staff directory" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: "Search staff" })).toBeVisible();

  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff attendance" })).toBeVisible();

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();

  await page.setViewportSize({ width: 844, height: 390 });
  for (const section of [
    { button: "Overview", heading: "People at a glance" },
    { button: "Staff Directory", heading: "Staff directory" },
    { button: "Attendance", heading: "Staff attendance" },
    { button: "Leave & Holidays", heading: "Leave & Holidays" },
  ]) {
    await page.getByRole("button", { name: section.button, exact: true }).click();
    await expect(page.getByRole("heading", { name: section.heading })).toBeVisible();
    await expect.poll(
      () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      { message: `Staff Management ${section.button} should not overflow in landscape` },
    ).toBe(true);
  }
});
