import { expect, test } from "@playwright/test";

test("class curriculum validates each optional-group field and locks the editor while saving", async ({ page }) => {
  const browserErrors: string[] = [];
  let saveStartedResolve!: () => void;
  let releaseSaveResolve!: () => void;
  const saveStarted = new Promise<void>(resolve => { saveStartedResolve = resolve; });
  const saveGate = new Promise<void>(resolve => { releaseSaveResolve = resolve; });
  let savedPayload: Record<string, unknown> | null = null;
  let curriculumSaved = false;

  page.on("pageerror", error => browserErrors.push(error.message));
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
        access_token: "e2e-academic-token", token_type: "Bearer",
        principal: { principal_id: 111, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 111, principal_type: "staff", school_id: 10, username: "e2e-academic-admin",
        permissions: ["academic_years.read", "classes.read", "subjects.read", "subjects.manage"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [{ id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true }] } });
      return;
    }
    if (path === "/school/classes") {
      await route.fulfill({ json: { classes: [{ id: 3, name: "Class 1", display_order: 1, is_active: true }] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [] } });
      return;
    }
    if (path === "/school/subjects") {
      await route.fulfill({ json: { subjects: [
        { id: 91, name: "Mathematics", subject_code: "MATH", is_active: true },
        { id: 92, name: "Hindi", subject_code: "HIN", is_active: true },
        { id: 93, name: "Sanskrit", subject_code: "SAN", is_active: true },
      ] } });
      return;
    }
    if (path === "/school/class-curricula" && method === "GET") {
      await route.fulfill({ json: { curricula: curriculumSaved ? [{
        id: 701, school_id: 10, academic_year_id: 7, class_id: 3, version_number: 1,
        status: "active", created_at: "2026-11-15T00:00:00.000Z",
        optional_groups: [{ id: 801, name: "Second language", min_selections: 1, max_selections: 2 }],
        subjects: [
          { id: 901, subject_id: 91, name: "Mathematics", is_compulsory: false, is_optional: true },
          { id: 902, subject_id: 92, name: "Hindi", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
          { id: 903, subject_id: 93, name: "Sanskrit", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
        ],
      }] : [] } });
      return;
    }
    if (path === "/school/class-curricula" && method === "POST") {
      savedPayload = request.postDataJSON() as Record<string, unknown>;
      saveStartedResolve();
      await saveGate;
      curriculumSaved = true;
      await route.fulfill({ status: 201, json: { status: "success", curriculum: { id: 701, version_number: 1, status: "active" } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-academic-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Academics" }).click();
  await page.getByRole("tab", { name: "Class curriculum" }).click();

  const yearSelect = page.getByLabel("Academic year");
  const classSelect = page.getByLabel("Class");
  await yearSelect.selectOption("7");
  await classSelect.selectOption("3");
  await page.getByRole("button", { name: "Add optional group" }).click();

  const groupName = page.getByLabel("Group name");
  const minChoices = page.getByLabel("Minimum choices");
  const maxChoices = page.getByLabel("Maximum choices");
  await groupName.fill("Second language");
  await page.getByRole("checkbox", { name: /Mathematics/ }).check();
  await page.getByRole("checkbox", { name: /Hindi/ }).check();
  await page.getByRole("checkbox", { name: /Sanskrit/ }).check();

  await page.getByRole("button", { name: "Save curriculum version" }).click();
  await expect(page.getByText("Assign at least one subject to this optional group.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save curriculum version" })).toBeEnabled();

  await minChoices.fill("2");
  await maxChoices.fill("1");
  await page.getByRole("button", { name: "Save curriculum version" }).click();
  await expect(page.getByText("Minimum choices cannot exceed maximum choices.", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Minimum choices")).toHaveAttribute("aria-invalid", "true");

  await minChoices.fill("1");
  await maxChoices.fill("2");
  const subjectRows = page.locator(".curriculum-subject-row");
  await subjectRows.filter({ hasText: "Mathematics" }).getByLabel("Type").selectOption("__standalone_optional__");
  await subjectRows.filter({ hasText: "Hindi" }).getByLabel("Type").selectOption("Second language");
  await subjectRows.filter({ hasText: "Sanskrit" }).getByLabel("Type").selectOption("Second language");

  await page.getByRole("button", { name: "Save curriculum version" }).click();
  await saveStarted;
  await expect(yearSelect).toBeDisabled();
  await expect(classSelect).toBeDisabled();
  await expect(groupName).toBeDisabled();
  await expect(minChoices).toBeDisabled();
  await expect(maxChoices).toBeDisabled();
  await expect(page.getByRole("checkbox", { name: /Mathematics/ })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Add optional group" })).toBeDisabled();

  releaseSaveResolve();
  await expect(page.getByText("Curriculum version saved. Existing versions and assigned student choices are preserved.")).toBeVisible();
  await expect(page.getByText("Version 1", { exact: true })).toBeVisible();
  await expect(page.getByText("3 subjects · 1 optional groups", { exact: true })).toBeVisible();
  await expect(page.getByText("Mathematics · Optional", { exact: true })).toBeVisible();
  await expect(page.getByText("Hindi · Second language", { exact: true })).toBeVisible();
  expect(savedPayload).toMatchObject({
    academic_year_id: 7,
    class_id: 3,
    optional_groups: [{ name: "Second language", min_selections: 1, max_selections: 2 }],
    subjects: [
      { subject_id: 91, is_optional: true },
      { subject_id: 92, is_optional: true, optional_group: "Second language" },
      { subject_id: 93, is_optional: true, optional_group: "Second language" },
    ],
  });
  await expect(page.getByLabel("Group name")).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});
