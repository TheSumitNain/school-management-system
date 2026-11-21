import { expect, test } from "@playwright/test";

test("subject teacher assignment validates fields, maps server errors and preserves revoke history", async ({ page }) => {
  let assignments: Array<Record<string, unknown>> = [];
  let createCalls = 0;
  let deleteCalls = 0;
  let failCreate = true;
  let failDelete = true;
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: { access_token: "e2e-teacher-assignments-token", token_type: "Bearer", principal: { principal_id: 121, principal_type: "staff", status: "active", school_id: 10 } } }); return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: { principal_id: 121, principal_type: "staff", school_id: 10, username: "e2e-teacher-assignments", permissions: ["academic_years.read","classes.read","subjects.read","teacher_assignments.read","teacher_assignments.manage","class_incharge_assignments.read","class_incharge_assignments.manage"] } }); return;
    }
    if (path === "/school/academic-years") { await route.fulfill({ json: { academic_years: [{ id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: false }] } }); return; }
    if (path === "/school/classes") { await route.fulfill({ json: { classes: [{ id: 8, name: "Class 4", is_active: true }] } }); return; }
    if (path === "/school/sections") { await route.fulfill({ json: { sections: [{ id: 19, academic_year_id: 7, class_id: 8, name: "A", is_active: true }] } }); return; }
    if (path === "/school/subjects") { await route.fulfill({ json: { subjects: [{ id: 21, name: "Mathematics", is_active: true }] } }); return; }
    if (path === "/school/eligible-teachers") { await route.fulfill({ json: { teachers: [{ principal_id: 31, username: "teacher.one" },{ principal_id: 32, username: "teacher.two" }] } }); return; }
    if (path === "/school/teacher-assignments" && method === "GET") { await route.fulfill({ json: { assignments } }); return; }
    if (path === "/school/class-incharge-assignments") { await route.fulfill({ json: { assignments: [] } }); return; }
    if (path === "/school/teacher-assignments" && method === "POST") {
      createCalls += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      if (failCreate) { failCreate = false; await route.fulfill({ status: 409, json: { status: "error", message: "This teacher already has this subject assignment.", field_errors: { teacher_principal_id: "Choose a different teacher for this subject and section." } } }); return; }
      assignments = [{ id: 91, academic_year_id: 7, section_id: 19, section_name: "A", class_name: "Class 4", subject_id: 21, subject_name: "Mathematics", teacher_principal_id: Number(body.teacher_principal_id), teacher_username: body.teacher_principal_id === 32 ? "teacher.two" : "teacher.one", revoked_at: null }];
      await route.fulfill({ status: 201, json: { assignment: assignments[0] } }); return;
    }
    if (path === "/school/teacher-assignments/91" && method === "DELETE") {
      deleteCalls += 1;
      if (failDelete) { failDelete = false; await route.fulfill({ status: 500, json: { status: "error", message: "Temporary revoke failure." } }); return; }
      assignments = assignments.map(item => ({ ...item, revoked_at: "2026-11-21T10:00:00Z" }));
      await route.fulfill({ json: { message: "Assignment revoked." } }); return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-teacher-assignments");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("tab", { name: "Teacher assignments" }).click();

  await page.getByRole("button", { name: "Assign subject teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Assign a subject teacher" });
  await dialog.getByRole("button", { name: "Assign subject" }).click();
  await expect(dialog.getByText("Select an academic year.")).toBeVisible();
  await expect(dialog.getByText("Select a section.")).toBeVisible();
  await expect(dialog.getByText("Select a subject.")).toBeVisible();
  await expect(dialog.getByText("Select a teacher.")).toBeVisible();
  expect(createCalls).toBe(0);

  const subjectYearSelect = dialog.locator('button[aria-label="Subject assignment academic year"]');
  await subjectYearSelect.click();
  await dialog.getByRole("option", { name: "2026–2027" }).click();
  if (await subjectYearSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  const subjectSectionSelect = dialog.locator('button[aria-label="Subject assignment section"]');
  await subjectSectionSelect.click();
  await dialog.getByRole("option", { name: "Class 4 · A" }).click();
  if (await subjectSectionSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  const subjectSelect = dialog.getByRole("button", { name: "Subject", exact: true });
  await subjectSelect.click();
  await dialog.getByRole("option", { name: "Mathematics" }).click();
  if (await subjectSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  const subjectTeacherSelect = dialog.locator('button[aria-label="Subject assignment teacher"]');
  await subjectTeacherSelect.click();
  await dialog.getByRole("option", { name: "teacher.one" }).click();
  if (await subjectTeacherSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Assign subject" }).click();

  await expect(dialog.getByText("Choose a different teacher for this subject and section.")).toBeVisible();
  await expect(page.locator(".app-toast-error").filter({ hasText: "This teacher already has this subject assignment." })).toBeVisible();
  expect(createCalls).toBe(1);

  await subjectTeacherSelect.click();
  await dialog.getByRole("option", { name: "teacher.two" }).click();
  if (await subjectTeacherSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Assign subject" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Teacher assigned to the subject and section." })).toBeVisible();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("cell", { name: "teacher.two" })).toBeVisible();
  expect(createCalls).toBe(2);

  await page.getByRole("button", { name: "Revoke" }).click();
  const revokeDialog = page.getByRole("dialog", { name: "Revoke subject assignment?" });
  await revokeDialog.getByRole("button", { name: "Cancel" }).click();
  await expect(revokeDialog).not.toBeVisible();
  expect(deleteCalls).toBe(0);
  await expect(page.getByRole("cell", { name: "Active", exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Revoke" }).click();
  const retryDialog = page.getByRole("dialog", { name: "Revoke subject assignment?" });
  await retryDialog.getByRole("button", { name: "Revoke assignment" }).click();
  await expect(retryDialog.getByText("Temporary revoke failure.")).toBeVisible();
  expect(deleteCalls).toBe(1);
  await retryDialog.getByRole("button", { name: "Revoke assignment" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Subject assignment revoked. Its history has been retained." })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Revoked", exact: true })).toBeVisible();
  expect(deleteCalls).toBe(2);
});

test("failed assignment refresh clears stale rows and disables new assignment actions", async ({ page }) => {
  let failAssignments = false;
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") { await route.fulfill({ json: { access_token: "e2e-teacher-load-token", token_type: "Bearer", principal: { principal_id: 122, principal_type: "staff", status: "active", school_id: 10 } } }); return; }
    if (path === "/auth/me") { await route.fulfill({ json: { principal_id: 122, principal_type: "staff", school_id: 10, username: "e2e-teacher-load", permissions: ["academic_years.read","classes.read","subjects.read","teacher_assignments.read","teacher_assignments.manage"] } }); return; }
    if (path === "/school/academic-years") { await route.fulfill({ json: { academic_years: [{ id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true }] } }); return; }
    if (path === "/school/classes") { await route.fulfill({ json: { classes: [{ id: 8, name: "Class 4", is_active: true }] } }); return; }
    if (path === "/school/sections") { await route.fulfill({ json: { sections: [{ id: 19, academic_year_id: 7, class_id: 8, name: "A", is_active: true }] } }); return; }
    if (path === "/school/subjects") { await route.fulfill({ json: { subjects: [{ id: 21, name: "Mathematics", is_active: true }] } }); return; }
    if (path === "/school/eligible-teachers") { await route.fulfill({ json: { teachers: [{ principal_id: 31, username: "teacher.one" }] } }); return; }
    if (path === "/school/teacher-assignments" && method === "GET") {
      if (failAssignments) { await route.fulfill({ status: 503, json: { message: "Assignment service temporarily unavailable." } }); return; }
      await route.fulfill({ json: { assignments: [{ id: 91, academic_year_id: 7, section_id: 19, section_name: "A", class_name: "Class 4", subject_id: 21, subject_name: "Mathematics", teacher_principal_id: 31, teacher_username: "teacher.one", revoked_at: null }] } }); return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-teacher-load");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("tab", { name: "Teacher assignments" }).click();
  await expect(page.getByRole("cell", { name: "teacher.one" })).toBeVisible();

  failAssignments = true;
  await page.locator(".teacher-assignments .assignment-heading > button").click();
  await expect(page.getByRole("alert").filter({ hasText: "Assignment service temporarily unavailable." })).toBeVisible();
  await expect(page.getByRole("cell", { name: "teacher.one" })).not.toBeVisible();
  await expect(page.getByRole("button", { name: "Assign subject teacher" })).toBeDisabled();
});


test("class in-charge assignment validates fields, recovers from server errors and revokes with history", async ({ page }) => {
  let incharges: Array<Record<string, unknown>> = [];
  let createCalls = 0;
  let deleteCalls = 0;
  let failCreate = true;
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (path === "/auth/refresh") { await route.fulfill({ status: 401, json: { message: "No active session" } }); return; }
    if (path === "/auth/login" && method === "POST") { await route.fulfill({ json: { access_token: "e2e-incharge-token", token_type: "Bearer", principal: { principal_id: 123, principal_type: "staff", status: "active", school_id: 10 } } }); return; }
    if (path === "/auth/me") { await route.fulfill({ json: { principal_id: 123, principal_type: "staff", school_id: 10, username: "e2e-incharge", permissions: ["academic_years.read","classes.read","class_incharge_assignments.read","class_incharge_assignments.manage"] } }); return; }
    if (path === "/school/academic-years") { await route.fulfill({ json: { academic_years: [{ id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: false }] } }); return; }
    if (path === "/school/classes") { await route.fulfill({ json: { classes: [{ id: 8, name: "Class 4", is_active: true }] } }); return; }
    if (path === "/school/sections") { await route.fulfill({ json: { sections: [{ id: 19, academic_year_id: 7, class_id: 8, name: "A", is_active: true }] } }); return; }
    if (path === "/school/eligible-teachers") { await route.fulfill({ json: { teachers: [{ principal_id: 31, username: "teacher.one" },{ principal_id: 32, username: "teacher.two" }] } }); return; }
    if (path === "/school/class-incharge-assignments" && method === "GET") { await route.fulfill({ json: { assignments: incharges } }); return; }
    if (path === "/school/class-incharge-assignments" && method === "POST") {
      createCalls += 1;
      const body = request.postDataJSON() as Record<string, unknown>;
      if (failCreate) { failCreate = false; await route.fulfill({ status: 409, json: { status: "error", message: "This section already has an active class in-charge.", field_errors: { teacher_principal_id: "Choose an eligible teacher for this section." } } }); return; }
      incharges = [{ id: 51, academic_year_id: 7, section_id: 19, section_name: "A", class_name: "Class 4", teacher_principal_id: Number(body.teacher_principal_id), teacher_username: body.teacher_principal_id === 32 ? "teacher.two" : "teacher.one", revoked_at: null }];
      await route.fulfill({ status: 201, json: { assignment: incharges[0] } }); return;
    }
    if (path === "/school/class-incharge-assignments/51" && method === "DELETE") {
      deleteCalls += 1;
      incharges = incharges.map(item => ({ ...item, revoked_at: "2026-11-21T10:00:00Z" }));
      await route.fulfill({ json: { message: "Class in-charge revoked." } }); return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-incharge");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("tab", { name: "Teacher assignments" }).click();
  await page.getByRole("button", { name: "Assign class in-charge" }).click();
  const dialog = page.getByRole("dialog", { name: "Assign class in-charge" });
  await dialog.getByRole("button", { name: "Assign in-charge" }).click();
  await expect(dialog.getByText("Select an academic year.")).toBeVisible();
  await expect(dialog.getByText("Select a section.")).toBeVisible();
  await expect(dialog.getByText("Select a teacher.")).toBeVisible();
  expect(createCalls).toBe(0);

  const inchargeYearSelect = dialog.locator('button[aria-label="Class in-charge academic year"]');
  await inchargeYearSelect.click();
  await dialog.getByRole("option", { name: "2026–2027" }).click();
  if (await inchargeYearSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  const inchargeSectionSelect = dialog.locator('button[aria-label="Class in-charge section"]');
  await inchargeSectionSelect.click();
  await dialog.getByRole("option", { name: "Class 4 · A" }).click();
  if (await inchargeSectionSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  const inchargeTeacherSelect = dialog.locator('button[aria-label="Class in-charge teacher"]');
  await inchargeTeacherSelect.click();
  await dialog.getByRole("option", { name: "teacher.one" }).click();
  if (await inchargeTeacherSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Assign in-charge" }).click();
  await expect(dialog.getByText("Choose an eligible teacher for this section.")).toBeVisible();
  expect(createCalls).toBe(1);

  await inchargeTeacherSelect.click();
  await page.getByRole("option", { name: "teacher.two" }).click();
  if (await inchargeTeacherSelect.getAttribute("aria-expanded") === "true") await page.keyboard.press("Escape");
  await dialog.getByRole("button", { name: "Assign in-charge" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Class in-charge assigned successfully." })).toBeVisible();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("cell", { name: "teacher.two" })).toBeVisible();
  expect(createCalls).toBe(2);

  await page.getByRole("button", { name: "Revoke" }).click();
  const revokeDialog = page.getByRole("dialog", { name: "Revoke class in-charge?" });
  await revokeDialog.getByRole("button", { name: "Revoke assignment" }).click();
  await expect(page.getByRole("status").filter({ hasText: "Class in-charge assignment revoked. Its history has been retained." })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Revoked", exact: true })).toBeVisible();
  expect(deleteCalls).toBe(1);
});
