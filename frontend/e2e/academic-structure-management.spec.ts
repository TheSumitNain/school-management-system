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
  await expect(page.getByRole("status").filter({ hasText: 'Class "Class 1" created successfully.' })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Class 1", exact: true })).toBeVisible();
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
  await expect(page.locator(".app-toast-error").filter({ hasText: "A subject with this name already exists." })).toBeVisible();
  expect(writes).toHaveLength(2);
  await expect(subjectDialog.getByLabel("Name")).toHaveValue("Mathematics");
});


test("closing an academic year archives it without exposing a delete action", async ({ page }) => {
  let yearStatus: "active" | "closed" = "active";
  let updatePayload: Record<string, unknown> | null = null;
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
        access_token: "e2e-academic-year-token", token_type: "Bearer",
        principal: { principal_id: 115, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 115, principal_type: "staff", school_id: 10, username: "e2e-academic-year",
        permissions: ["academic_years.read", "academic_years.manage"],
      } });
      return;
    }
    if (path === "/school/academic-years" && method === "GET") {
      await route.fulfill({ json: { academic_years: [{
        id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31",
        status: yearStatus, is_current: yearStatus === "active",
      }] } });
      return;
    }
    if (path === "/school/academic-years/7" && method === "PATCH") {
      updatePayload = request.postDataJSON() as Record<string, unknown>;
      yearStatus = "closed";
      await route.fulfill({ json: { status: "success", academic_year: {
        id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31",
        status: "closed", is_current: false,
      } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-academic-year");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("button", { name: "Edit 2026–2027" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit academic year" });
  await expect(dialog.getByText("Closed academic years are archived and retained for historical records; they cannot be edited or deleted.")).toBeVisible();
  const status = dialog.getByLabel("Academic year status");
  await status.click();
  await page.getByRole("option", { name: "Closed" }).click();
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("status").filter({ hasText: 'Academic year "2026–2027" closed and archived successfully.' })).toBeVisible();
  await expect(page.getByRole("button", { name: "Edit 2026–2027" })).toBeDisabled();
  expect(updatePayload).toMatchObject({
    name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31",
    status: "closed", is_current: false,
  });
});


test("section deactivation shows exact dependency counts and active student list before saving", async ({ page }) => {
  let deactivated = false;
  let impactRequests = 0;
  let patchRequests = 0;
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-section-impact-token", token_type: "Bearer",
        principal: { principal_id: 116, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 116, principal_type: "staff", school_id: 10, username: "e2e-section-impact",
        permissions: ["academic_years.read", "classes.read", "classes.manage"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [
        { id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true },
      ] } });
      return;
    }
    if (path === "/school/classes") {
      await route.fulfill({ json: { classes: [{ id: 8, name: "Class 4", display_order: 4, is_active: true }] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [{
        id: 19, academic_year_id: 7, class_id: 8, name: "C", medium: "english",
        display_order: 3, is_active: !deactivated,
      }] } });
      return;
    }
    if (path === "/school/sections/19/deactivation-impact" && method === "GET") {
      impactRequests += 1;
      await route.fulfill({ json: { status: "success", impact: {
        section_id: 19, section_name: "C", class_name: "Class 4", academic_year_name: "2026–2027",
        active_enrollment_count: 2, historical_enrollment_count: 1, active_curriculum_count: 1,
        active_teacher_assignment_count: 2, active_class_incharge_count: 1,
        affected_students: [
          { student_id: 101, full_name: "Aarav Sharma", admission_number: "ADM-101", enrollment_status: "active" },
          { student_id: 102, full_name: "Diya Verma", admission_number: "ADM-102", enrollment_status: "active" },
        ],
      } } });
      return;
    }
    if (path === "/school/sections/19" && method === "PATCH") {
      patchRequests += 1;
      deactivated = true;
      await route.fulfill({ json: { status: "success", section: {
        id: 19, academic_year_id: 7, class_id: 8, name: "C", medium: "english",
        display_order: 3, is_active: false,
      } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-section-impact");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("tab", { name: "Sections" }).click();
  await page.getByRole("button", { name: "Edit C" }).click();
  const editDialog = page.getByRole("dialog", { name: "Edit section" });
  const status = editDialog.getByLabel("Section status");
  await status.click();
  await page.getByRole("option", { name: "Inactive" }).click();
  await editDialog.getByRole("button", { name: "Save changes" }).click();

  const impactDialog = page.getByRole("dialog", { name: "Review section deactivation" });
  await expect(impactDialog.getByText(/2 active student enrollments/)).toBeVisible();
  await expect(impactDialog.getByText("Aarav Sharma")).toBeVisible();
  await expect(impactDialog.getByText("Diya Verma")).toBeVisible();
  await expect(impactDialog.getByText(/does not transfer students or revoke assignments automatically/)).toBeVisible();
  expect(impactRequests).toBe(1);
  expect(patchRequests).toBe(0);

  await impactDialog.getByRole("button", { name: "Go back" }).click();
  await expect(editDialog).toBeVisible();
  expect(patchRequests).toBe(0);

  await editDialog.getByRole("button", { name: "Save changes" }).click();
  const confirmation = page.getByRole("dialog", { name: "Review section deactivation" });
  await confirmation.getByRole("button", { name: "Confirm deactivation" }).click();
  await expect(page.getByRole("status").filter({ hasText: 'Section "C" updated successfully.' })).toBeVisible();
  expect(impactRequests).toBe(2);
  expect(patchRequests).toBe(1);
});
