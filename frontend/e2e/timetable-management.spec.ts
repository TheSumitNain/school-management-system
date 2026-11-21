import { expect, test } from "@playwright/test";

test("timetable period form validates every required field inline without browser validation", async ({ page }) => {
  const year = {
    id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31",
    status: "active", is_current: true,
  };

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
      await route.fulfill({ json: {
        access_token: "e2e-timetable-token",
        token_type: "Bearer",
        principal: { principal_id: 112, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 112, principal_type: "staff", school_id: 10,
        username: "e2e-timetable-admin", permissions: ["timetable.read", "timetable.manage"],
      } });
      return;
    }
    if (path === "/school/timetable/current-year") {
      await route.fulfill({ json: { academic_year: year } });
      return;
    }
    if (path === "/school/timetable/periods") {
      await route.fulfill({ json: { periods: [] } });
      return;
    }
    if (path === "/school/timetable/entries") {
      await route.fulfill({ json: { entries: [] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `No timetable fixture for ${method} ${path}.` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-timetable-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("heading", { level: 1, name: "Timetable" })).toBeVisible();

  await page.getByRole("button", { name: "Add period" }).click();
  const form = page.locator("form.timetable-form");
  await expect(form).toHaveAttribute("novalidate", "");

  await page.getByRole("button", { name: "Save period" }).click();

  const fields = [
    { label: "Period number", error: "Enter a positive whole period number.", describedBy: "timetable-period-number-error" },
    { label: "Period name", error: "Enter a period name.", describedBy: "timetable-period-name-error" },
    { label: "Starts at", error: "Choose a start time.", describedBy: "timetable-period-start-error" },
    { label: "Ends at", error: "Choose an end time.", describedBy: "timetable-period-end-error" },
  ];

  for (const field of fields) {
    const input = page.getByLabel(field.label, { exact: true });
    await expect(input).toHaveAttribute("aria-invalid", "true");
    await expect(input).toHaveAttribute("aria-describedby", field.describedBy);
    await expect(page.locator(`#${field.describedBy}[role="alert"]`)).toHaveText(field.error);
  }
  await expect(page.locator("#timetable-period-number-error")).toBeVisible();
  await expect(page.locator("#timetable-period-name-error")).toBeVisible();
  await expect(page.locator("#timetable-period-start-error")).toBeVisible();
  await expect(page.locator("#timetable-period-end-error")).toBeVisible();
});
