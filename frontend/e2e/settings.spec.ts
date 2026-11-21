import { expect, test } from "@playwright/test";

test.use({ timezoneId: "Asia/Kolkata" });

test("appearance mode, palette and automatic schedule save safely on this device", async ({ page }) => {
  test.setTimeout(120_000);
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
  await expect(page.getByRole("heading", { name: "Settings", exact: true })).toBeVisible();

  // Verify the complete Settings surface at compact phone and landscape sizes,
  // not only the page shell's horizontal-overflow smoke test.
  for (const viewport of [{ width: 320, height: 568 }, { width: 844, height: 390 }]) {
    await page.setViewportSize(viewport);
    const layout = await page.evaluate(() => {
      const controls = Array.from(document.querySelectorAll<HTMLElement>(
        ".appearance-option, .palette-option, .theme-schedule-field input",
      ));
      return {
        pageFits: document.documentElement.scrollWidth <= window.innerWidth,
        controlsFit: controls.every(control => {
          const rect = control.getBoundingClientRect();
          return rect.left >= 0 && rect.right <= window.innerWidth + 1 &&
            control.scrollWidth <= control.clientWidth + 1;
        }),
        optionCounts: {
          modes: document.querySelectorAll(".appearance-option").length,
          palettes: document.querySelectorAll(".palette-option").length,
        },
      };
    });
    expect(layout.pageFits, `Settings should not overflow at ${viewport.width}×${viewport.height}`).toBe(true);
    expect(layout.controlsFit, `Settings controls should fit at ${viewport.width}×${viewport.height}`).toBe(true);
    expect(layout.optionCounts).toEqual({ modes: 4, palettes: 15 });
  }
  await page.setViewportSize({ width: 1440, height: 900 });

  const darkMode = page.getByRole("button", { name: /^Dark\b/ });
  await darkMode.click();
  await expect(darkMode).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("status").filter({ hasText: "Appearance mode updated." })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  const evergreen = page.getByRole("button", { name: "Evergreen" });
  await evergreen.click();
  await expect(evergreen).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("status").filter({ hasText: "Brand palette updated." })).toBeVisible();

  // Every documented palette must be selectable, with exactly one active
  // selection; restore Evergreen for the persistence assertion below.
  const paletteNames = ["Cobalt", "Alpine", "Evergreen", "Lagoon", "Iris", "Amethyst", "Blush", "Terracotta", "Saffron", "Dune", "Glacier", "Obsidian", "Graphite", "Ruby", "Copper"];
  await expect(page.locator(".palette-option")).toHaveCount(paletteNames.length);
  for (const name of paletteNames) {
    const option = page.getByRole("button", { name, exact: true });
    await option.click();
    await expect(option).toHaveAttribute("aria-pressed", "true");
    await expect(page.locator(".palette-option[aria-pressed='true']")).toHaveCount(1);
  }
  await evergreen.click();
  await expect(evergreen).toHaveAttribute("aria-pressed", "true");

  await page.getByRole("button", { name: /Automatic schedule/ }).click();
  const darkStart = page.getByLabel("Turn Dark Mode on");
  const lightStart = page.getByLabel("Turn Light Mode on");
  await darkStart.fill("06:00");
  await expect(darkStart).toHaveAttribute("aria-invalid", "true");
  await expect(lightStart).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("alert").filter({ hasText: "Choose a time different from the Light Mode start time." })).toBeVisible();
  await expect(page.getByRole("alert").filter({ hasText: "Choose a time different from the Dark Mode start time." })).toBeVisible();
  await expect(page.getByRole("status").filter({ hasText: "Choose different Dark and Light start times to enable the automatic schedule." })).toBeVisible();

  const schedule = await page.evaluate(() => {
    const now = new Date();
    const nextMinute = new Date(now.getTime() + 60_000);
    const format = (date: Date) => `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`;
    return {
      darkStart: format(new Date(now.getTime() - 60_000)),
      lightStart: format(nextMinute),
      waitMs: 60_000 - (now.getSeconds() * 1_000 + now.getMilliseconds()) + 31_000,
    };
  });
  await darkStart.fill(schedule.darkStart);
  await lightStart.fill(schedule.lightStart);
  await expect(darkStart).toHaveAttribute("aria-invalid", "false");
  await expect(lightStart).toHaveAttribute("aria-invalid", "false");
  await expect(page.getByRole("status").filter({ hasText: "Theme schedule updated." })).toBeVisible();

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  const persisted = await page.evaluate(() => JSON.parse(localStorage.getItem("schooldesk.theme") ?? "{}"));
  expect(persisted.state).toMatchObject({ preference: "schedule", palette: "evergreen", darkStart: schedule.darkStart, lightStart: schedule.lightStart });

  // The schedule must continue updating after leaving Settings.
  await page.getByRole("button", { name: "Dashboard" }).click();
  await page.waitForTimeout(schedule.waitMs);
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  // System mode follows device appearance changes without reloading the app.
  await page.getByRole("button", { name: "Settings" }).click();
  await page.getByRole("button", { name: /^System\b/ }).click();
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});
