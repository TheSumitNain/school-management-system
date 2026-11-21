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


test("dashboard explains partial loading failures and lets the user retry", async ({ page }) => {
  let staffRequests = 0;
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
        access_token: "e2e-dashboard-retry-token",
        token_type: "Bearer",
        principal: { principal_id: 152, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 152, principal_type: "staff", school_id: 10,
        username: "dashboard-retry-admin", permissions: ["users.read", "classes.read"],
      } });
      return;
    }
    if (path === "/school/staff") {
      staffRequests += 1;
      if (staffRequests === 1) {
        await route.fulfill({ status: 503, json: { message: "Staff service is temporarily unavailable while the school workspace is loading. Please retry in a moment or contact your administrator if the problem continues." } });
      } else {
        await route.fulfill({ json: { staff: [{ id: 1 }, { id: 2 }] } });
      }
      return;
    }
    if (path === "/school/classes") {
      await route.fulfill({ json: { classes: [{ id: 1 }, { id: 2 }, { id: 3 }] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked dashboard request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("dashboard-retry-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();

  const alert = page.getByRole("alert");
  await expect(alert).toContainText("Some dashboard information could not be loaded.");
  await expect(alert).toContainText("Staff accounts:");
  await page.setViewportSize({ width: 320, height: 720 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  const alertFits = await alert.evaluate(element => { const rect = element.getBoundingClientRect(); return rect.left >= 0 && rect.right <= window.innerWidth + 1; });
  expect(alertFits, "dashboard error feedback should wrap within a 320px viewport").toBe(true);
  await expect(page.locator(".stat").filter({ hasText: "Classes" }).locator("strong")).toHaveText("3");
  await page.getByRole("button", { name: "Retry" }).click();

  await expect(page.locator(".stat").filter({ hasText: "Staff accounts" }).locator("strong")).toHaveText("2");
  await expect(page.locator(".stat").filter({ hasText: "Classes" }).locator("strong")).toHaveText("3");
  await expect(page.getByRole("alert")).toHaveCount(0);
  expect(staffRequests).toBe(2);
});

test("dashboard metrics and feedback remain readable without viewport overflow", async ({ page }) => {
  const permissions = ["users.read", "classes.read", "academic_years.read", "students.read_basic"];
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
        access_token: "e2e-dashboard-responsive-token",
        token_type: "Bearer",
        principal: { principal_id: 153, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 153, principal_type: "staff", school_id: 10,
        username: "dashboard-responsive-admin", permissions,
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
      await route.fulfill({ json: { academic_years: [{ id: 7, is_current: true, status: "active" }] } });
      return;
    }
    if (path === "/school/enrollments") {
      await route.fulfill({ json: { enrollments: Array.from({ length: 18 }, (_, index) => ({ id: index + 1 })) } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked dashboard request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("dashboard-responsive-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await expect(page.locator(".stat")).toHaveCount(4);

  for (const width of [320, 360, 390, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    const cardsFit = await page.locator(".stat").evaluateAll(elements => elements.every(element => {
      const rect = element.getBoundingClientRect();
      return rect.width > 0 && rect.left >= 0 && rect.right <= window.innerWidth + 1;
    }));
    expect(cardsFit, `dashboard metric cards should fit within ${width}px`).toBe(true);
  }

  await page.setViewportSize({ width: 844, height: 390 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect(page.locator(".dashboard-page > .heading p")).toBeVisible();
});


test("dashboard shows a helpful empty state when the account has no overview permissions", async ({ page }) => {
  const unexpectedRequests: string[] = [];
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
        access_token: "e2e-dashboard-empty-token",
        token_type: "Bearer",
        principal: { principal_id: 154, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 154, principal_type: "staff", school_id: 10,
        username: "limited-dashboard-user", permissions: [],
      } });
      return;
    }
    unexpectedRequests.push(path);
    await route.fulfill({ status: 404, json: { message: "No dashboard data should be requested." } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("limited-dashboard-user");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();

  await expect(page.getByRole("status")).toContainText("No overview metrics available");
  await expect(page.locator(".stat")).toHaveCount(0);
  expect(unexpectedRequests).toEqual([]);
});
