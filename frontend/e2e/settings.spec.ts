import { expect, test } from "@playwright/test";

test("appearance mode, palette and automatic schedule save safely on this device", async ({ page }) => {
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "e2e-settings-token", token_type: "Bearer", principal: { principal_id: 141, principal_type: "staff", status: "active", school_id: 10 } } }); return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: { principal_id: 141, principal_type: "staff", school_id: 10, username: "e2e-settings", permissions: [] } }); return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-settings");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Settings" }).click();

  const darkMode = page.getByRole("button", { name: /^Dark\\b/ });
  await darkMode.click();
  await expect(darkMode).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("status").filter({ hasText: "Appearance mode updated." })).toBeVisible();

  const evergreen = page.getByRole("button", { name: "Evergreen" });
  await evergreen.click();
  await expect(evergreen).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("status").filter({ hasText: "Brand palette updated." })).toBeVisible();

  await page.getByRole("button", { name: /Automatic schedule/ }).click();
  const darkStart = page.getByLabel("Turn Dark Mode on");
  const lightStart = page.getByLabel("Turn Light Mode on");
  await darkStart.fill("06:00");
  await expect(darkStart).toHaveAttribute("aria-invalid", "true");
  await expect(lightStart).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("alert").filter({ hasText: "Choose a time different from the Light Mode start time." })).toBeVisible();
  await expect(page.getByRole("alert").filter({ hasText: "Choose a time different from the Dark Mode start time." })).toBeVisible();
  await expect(page.getByRole("status").filter({ hasText: "Choose different Dark and Light start times to enable the automatic schedule." })).toBeVisible();

  await lightStart.fill("07:00");
  await expect(darkStart).toHaveAttribute("aria-invalid", "false");
  await expect(lightStart).toHaveAttribute("aria-invalid", "false");
  await expect(page.getByRole("status").filter({ hasText: "Theme schedule updated." })).toBeVisible();

  const persisted = await page.evaluate(() => JSON.parse(localStorage.getItem("schooldesk.theme") ?? "{}"));
  expect(persisted.state).toMatchObject({ preference: "schedule", palette: "evergreen", darkStart: "06:00", lightStart: "07:00" });
});
