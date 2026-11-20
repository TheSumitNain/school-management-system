import { expect, test } from "@playwright/test";

test("section curriculum validates optional groups, edits in place, and confirms affected students before removal", async ({ page }) => {
  const browserErrors: string[] = [];
  let saveStartedResolve!: () => void;
  let releaseSaveResolve!: () => void;
  const saveStarted = new Promise<void>(resolve => { saveStartedResolve = resolve; });
  const saveGate = new Promise<void>(resolve => { releaseSaveResolve = resolve; });
  let createPayload: Record<string, unknown> | null = null;
  let removalPreviewPayload: Record<string, unknown> | null = null;
  let removalSavePayload: Record<string, unknown> | null = null;
  let curriculumSaved = false;
  let saveCount = 0;
  let currentCurriculum: Record<string, unknown> = {
    id: 701, school_id: 10, academic_year_id: 7, class_id: 3, section_id: 44,
    version_number: 1, status: "active", created_at: "2026-11-15T00:00:00.000Z",
    optional_groups: [{ id: 801, name: "Second language", min_selections: 1, max_selections: 2 }],
    subjects: [
      { id: 901, subject_id: 91, name: "Mathematics", is_compulsory: false, is_optional: true },
      { id: 902, subject_id: 92, name: "Hindi", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
      { id: 903, subject_id: 93, name: "Sanskrit", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
    ],
  };

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
      await route.fulfill({ json: { sections: [{ id: 44, academic_year_id: 7, class_id: 3, name: "A", medium: "english", is_active: true }] } });
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
      await route.fulfill({ json: { curricula: curriculumSaved ? [currentCurriculum] : [] } });
      return;
    }
    if (path === "/school/class-curricula/removal-impact" && method === "POST") {
      removalPreviewPayload = request.postDataJSON() as Record<string, unknown>;
      await route.fulfill({ json: { impacts: [{ subject_id: 92, subject_name: "Hindi", affected_students: 2 }] } });
      return;
    }
    if (path === "/school/class-curricula" && method === "POST") {
      const payload = request.postDataJSON() as Record<string, unknown>;
      if (saveCount === 0) {
        createPayload = payload;
        saveStartedResolve();
        await saveGate;
        curriculumSaved = true;
      } else {
        removalSavePayload = payload;
        currentCurriculum = {
          ...currentCurriculum,
          subjects: [
            { id: 901, subject_id: 91, name: "Mathematics", is_compulsory: false, is_optional: true },
            { id: 903, subject_id: 93, name: "Sanskrit", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
          ],
        };
      }
      saveCount += 1;
      await route.fulfill({ status: 200, json: { status: "success", curriculum: currentCurriculum } });
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
  const sectionSelect = page.getByLabel("Section");
  await yearSelect.selectOption("7");
  await classSelect.selectOption("3");
  await sectionSelect.selectOption("44");
  await page.getByRole("button", { name: "Add optional group" }).click();

  const groupName = page.getByLabel("Group name");
  const minChoices = page.getByLabel("Minimum choices");
  const maxChoices = page.getByLabel("Maximum choices");
  await groupName.fill("Second language");

  await page.getByRole("button", { name: "Save curriculum" }).click();
  await expect(page.getByText("Assign at least one subject to this optional group.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save curriculum" })).toBeEnabled();

  await minChoices.fill("2");
  await maxChoices.fill("1");
  await page.getByRole("checkbox", { name: /Mathematics/ }).check();
  await page.getByRole("checkbox", { name: /Hindi/ }).check();
  await page.getByRole("checkbox", { name: /Sanskrit/ }).check();
  await page.getByRole("button", { name: "Save curriculum" }).click();
  await expect(page.getByText("Minimum choices cannot exceed maximum choices.", { exact: true })).toBeVisible();
  await expect(minChoices).toHaveAttribute("aria-invalid", "true");

  await minChoices.fill("1");
  await maxChoices.fill("2");
  const subjectRows = page.locator(".curriculum-subject-row");
  await subjectRows.filter({ hasText: "Mathematics" }).getByLabel("Type").selectOption("__standalone_optional__");
  await subjectRows.filter({ hasText: "Hindi" }).getByLabel("Type").selectOption("Second language");
  await subjectRows.filter({ hasText: "Sanskrit" }).getByLabel("Type").selectOption("Second language");

  await page.getByRole("button", { name: "Save curriculum" }).click();
  await saveStarted;
  await expect(yearSelect).toBeDisabled();
  await expect(classSelect).toBeDisabled();
  await expect(sectionSelect).toBeDisabled();
  await expect(groupName).toBeDisabled();
  await expect(minChoices).toBeDisabled();
  await expect(maxChoices).toBeDisabled();
  await expect(page.getByRole("checkbox", { name: /Mathematics/ })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Add optional group" })).toBeDisabled();

  releaseSaveResolve();
  await expect(page.getByText("Curriculum saved successfully. Existing student selections are preserved except the subjects you confirmed for removal.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Edit curriculum" })).toBeVisible();
  await expect(page.getByText("3 subjects · 1 optional groups", { exact: true })).toBeVisible();
  await expect(page.locator(".curriculum-existing").getByText("Mathematics · Optional", { exact: true })).toBeVisible();
  await expect(page.locator(".curriculum-existing").getByText("Hindi · Second language", { exact: true })).toBeVisible();
  expect(createPayload).toMatchObject({
    academic_year_id: 7,
    class_id: 3,
    section_id: 44,
    optional_groups: [{ name: "Second language", min_selections: 1, max_selections: 2 }],
    subjects: [
      { subject_id: 91, is_optional: true },
      { subject_id: 92, is_optional: true, optional_group: "Second language" },
      { subject_id: 93, is_optional: true, optional_group: "Second language" },
    ],
  });

  await page.getByRole("checkbox", { name: /Hindi/ }).uncheck();
  await maxChoices.fill("1");
  await page.getByRole("button", { name: "Save curriculum changes" }).click();
  await expect(page.getByRole("heading", { name: "Confirm subject removal" })).toBeVisible();
  await expect(page.getByRole("dialog").getByText("Hindi", { exact: true })).toBeVisible();
  await expect(page.getByText("2 affected students", { exact: true })).toBeVisible();
  expect(removalPreviewPayload).toMatchObject({
    academic_year_id: 7, class_id: 3, section_id: 44, subject_ids: [92],
  });

  await page.getByRole("button", { name: "Confirm and save" }).click();
  await expect(page.getByText("Curriculum saved successfully. Existing student selections are preserved except the subjects you confirmed for removal.")).toBeVisible();
  await expect(page.locator(".curriculum-existing").getByText("Hindi", { exact: false })).toHaveCount(0);
  expect(removalSavePayload).toMatchObject({
    academic_year_id: 7, class_id: 3, section_id: 44,
    removal_confirmations: [{ subject_id: 92, affected_students: 2 }],
    subjects: [
      { subject_id: 91, is_optional: true },
      { subject_id: 93, is_optional: true, optional_group: "Second language" },
    ],
  });
  expect(browserErrors).toEqual([]);
});
