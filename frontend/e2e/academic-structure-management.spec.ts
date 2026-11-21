import { expect, test } from "@playwright/test";

test("academic structure create feedback is contextual and failed writes remain recoverable", async ({ page }) => {
  const writes: Array<{ path: string; method: string; body: Record<string, unknown> }> = [];
  let classes: Array<Record<string, unknown>> = [];
  let subjects: Array<Record<string, unknown>> = [];

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
        access_token: "e2e-academic-management-token", token_type: "Bearer",
        principal: { principal_id: 114, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 114, principal_type: "staff", school_id: 10, username: "e2e-academic-management",
        permissions: [
          "academic_years.read", "academic_years.manage", "classes.read", "classes.manage",
          "subjects.read", "subjects.manage",
        ],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [
        { id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true },
      ] } });
      return;
    }
    if (path === "/school/classes" && method === "GET") {
      await route.fulfill({ json: { classes } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [] } });
      return;
    }
    if (path === "/school/subjects" && method === "GET") {
      await route.fulfill({ json: { subjects } });
      return;
    }
    if (path === "/school/classes" && method === "POST") {
      writes.push({ path, method, body: request.postDataJSON() as Record<string, unknown> });
      classes = [{ id: 31, name: "Class 1", display_order: 1, is_active: true }];
      await route.fulfill({ status: 201, json: { status: "success", class: classes[0] } });
      return;
    }
    if (path === "/school/subjects" && method === "POST") {
      writes.push({ path, method, body: request.postDataJSON() as Record<string, unknown> });
      await route.fulfill({ status: 409, json: {
        status: "error", message: "A subject with this name already exists.",
        field_errors: { name: "This subject already exists in your school." },
      } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-academic-management");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();

  await page.getByRole("tab", { name: "Classes" }).click();
  await page.getByRole("button", { name: "Add class" }).click();
  const classDialog = page.getByRole("dialog", { name: "Add class" });
  await classDialog.getByLabel("Name").fill("Class 1");
  await classDialog.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Class created successfully." })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Class 1" })).toBeVisible();
  expect(writes[0]).toMatchObject({
    path: "/school/classes", method: "POST",
    body: { name: "Class 1", display_order: 0, is_active: true },
  });

  await page.getByRole("tab", { name: "Subjects" }).click();
  await page.getByRole("button", { name: "Add subject" }).click();
  const subjectDialog = page.getByRole("dialog", { name: "Add subject" });
  await subjectDialog.getByLabel("Name").fill("Mathematics");
  await subjectDialog.getByRole("button", { name: "Save changes" }).click();
  await expect(subjectDialog.getByText("This subject already exists in your school.")).toBeVisible();
  await expect(subjectDialog.getByLabel("Name")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("alert").filter({ hasText: "A subject with this name already exists." })).toBeVisible();
  expect(writes).toHaveLength(2);
  await expect(subjectDialog.getByLabel("Name")).toHaveValue("Mathematics");
});
