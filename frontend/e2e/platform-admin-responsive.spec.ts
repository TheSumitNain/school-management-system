import { expect, test } from "@playwright/test";

test("platform administration tabs and onboarding dialog remain responsive", async ({ page }) => {
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\\/api/, "");
    const method = request.method();

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/platform-admin/login" && method === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-platform-responsive-token", token_type: "Bearer",
        principal: { principal_id: 2, principal_type: "platform_admin", status: "active" },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 2, principal_type: "platform_admin", username: "e2e-platform-admin",
        permissions: [
          "platform.admins.manage",
          "platform.schools.create",
          "platform.school_admins.create",
          "platform.permissions.delegate",
          "platform.schools.read",
        ],
      } });
      return;
    }
    if (path === "/platform/admins" && method === "GET") {
      await route.fulfill({ json: { admins: [] } });
      return;
    }
    if (path === "/platform/schools" && method === "GET") {
      await route.fulfill({ json: { schools: [] } });
      return;
    }
    if (path === "/platform/permissions/delegable" && method === "GET") {
      await route.fulfill({ json: { permissions: [] } });
      return;
    }
    await route.fulfill({ status: 200, json: {} });
  });

  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/");
  await page.getByRole("button", { name: "Platform admin" }).click();
  await page.getByLabel("Email address").fill("platform@example.test");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Platform Admin" }).click();
  await expect(page.getByRole("heading", { level: 1, name: "Platform administration" })).toBeVisible();

  const tabNames = ["Platform Admins", "Onboard school", "School permissions"];
  for (const width of [320, 360, 390, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 844 });
    for (const name of tabNames) {
      const tab = page.getByRole("tab", { name });
      await tab.click();
      await expect(tab).toHaveAttribute("aria-selected", "true");
      await expect(tab).toBeInViewport();
      await expect.poll(
        () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
        { message: `Platform administration should not cause page-level horizontal overflow at ${width}px on the ${name} tab` },
      ).toBe(true);
    }
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("tab", { name: "Onboard school" }).click();
  await page.getByRole("button", { name: /Onboard a new school/ }).click();
  const dialog = page.getByRole("dialog", { name: "Onboard a new school" });
  await expect(dialog).toBeVisible();
  const bounds = await dialog.boundingBox();
  expect(bounds).not.toBeNull();
  expect(bounds!.x).toBeGreaterThanOrEqual(0);
  expect(bounds!.y).toBeGreaterThanOrEqual(0);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
  expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(844);
  await page.getByRole("button", { name: "Close dialog" }).click();
});
