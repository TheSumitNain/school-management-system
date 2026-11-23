import { expect, test } from "@playwright/test";

test("timetable conflict preview checks the teacher across all sections despite the section filter", async ({ page }) => {
  const year = {
    id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31",
    status: "active", is_current: true,
  };
  const entryQueries: string[] = [];
  let createRequests = 0;

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
        access_token: "e2e-timetable-conflict-token",
        token_type: "Bearer",
        principal: { principal_id: 112, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 112, principal_type: "staff", school_id: 10,
        username: "e2e-timetable-admin",
        permissions: ["timetable.read", "timetable.manage", "classes.read", "teacher_assignments.read"],
      } });
      return;
    }
    if (path === "/school/timetable/current-year") {
      await route.fulfill({ json: { academic_year: year } });
      return;
    }
    if (path === "/school/timetable/periods") {
      await route.fulfill({ json: { periods: [
        { id: 1, school_id: 10, academic_year_id: 7, period_number: 1, name: "Period 1", starts_at: "09:00", ends_at: "09:40", is_break: false },
        { id: 2, school_id: 10, academic_year_id: 7, period_number: 2, name: "Period 2", starts_at: "09:40", ends_at: "10:20", is_break: false },
      ] } });
      return;
    }
    if (path === "/school/timetable/entries" && method === "GET") {
      entryQueries.push(url.search);
      await route.fulfill({ json: { entries: [
        {
          id: 900, school_id: 10, academic_year_id: 7, section_id: 20,
          section_name: "A", class_name: "Class 6", subject_id: 11, subject_name: "Mathematics",
          teacher_principal_id: 501, teacher_username: "teacher.one", teacher_subject_assignment_id: 601,
          period_id: 1, period_number: 1, period_name: "Period 1", starts_at: "09:00", ends_at: "09:40",
          day_of_week: 1, room: "Room 1",
        },
        {
          id: 901, school_id: 10, academic_year_id: 7, section_id: 10,
          section_name: "B", class_name: "Class 6", subject_id: 14, subject_name: "History",
          teacher_principal_id: 503, teacher_username: "teacher.three", teacher_subject_assignment_id: 604,
          period_id: 2, period_number: 2, period_name: "Period 2", starts_at: "09:40", ends_at: "10:20",
          day_of_week: 1, room: "Room 2",
        },
      ] } });
      return;
    }
    if (path === "/school/sections" && method === "GET") {
      await route.fulfill({ json: { sections: [
        { id: 10, school_id: 10, academic_year_id: 7, class_id: 6, class_name: "Class 6", name: "B", is_active: true },
        { id: 20, school_id: 10, academic_year_id: 7, class_id: 6, class_name: "Class 6", name: "A", is_active: true },
      ] } });
      return;
    }
    if (path === "/school/teacher-assignments" && method === "GET") {
      await route.fulfill({ json: { assignments: [
        { id: 602, academic_year_id: 7, section_id: 10, section_name: "B", class_name: "Class 6", subject_id: 12, subject_name: "Science", teacher_principal_id: 501, teacher_username: "teacher.one", revoked_at: null },
        { id: 603, academic_year_id: 7, section_id: 10, section_name: "B", class_name: "Class 6", subject_id: 13, subject_name: "English", teacher_principal_id: 502, teacher_username: "teacher.two", revoked_at: null },
      ] } });
      return;
    }
    if (path === "/school/timetable/entries" && method === "POST") {
      createRequests += 1;
      await route.fulfill({ status: 201, json: { entry: {} } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `No timetable conflict fixture for ${method} ${path}.` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-timetable-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.evaluate(() => { window.history.pushState({}, "", "/timetable"); window.dispatchEvent(new PopStateEvent("popstate")); });
  await expect(page.getByRole("heading", { level: 1, name: "Timetable" })).toBeVisible();

  await page.getByRole("button", { name: "Class / section" }).click();
  await page.getByRole("option", { name: "Class 6 · Section B" }).click();
  expect(entryQueries.length).toBeGreaterThan(0);
  expect(entryQueries.every(query => !new URLSearchParams(query).has("section_id"))).toBe(true);

  await page.getByRole("button", { name: "Add class" }).click();
  const form = page.locator("form.timetable-form");
  await form.getByRole("button", { name: "Teacher and subject assignment" }).click();
  await page.getByRole("option", { name: "Science · teacher.one" }).click();

  const review = page.locator(".timetable-slot-review");
  await expect(review).toContainText("Slot review · Monday · Period 1");
  await expect(review).toContainText("Already scheduled: Mathematics · Class 6 A");
  await expect(review).toContainText("Teacher's other classes on Monday");
  await expect(review).toContainText("P1 · Mathematics · Class 6 A");
  await expect(review).toContainText("No existing class for this section in the selected slot.");
  await expect(review).toContainText("An empty slot does not establish staff availability.");

  await form.getByRole("button", { name: "Add to timetable" }).click();
  await expect(page.getByText("This teacher is already scheduled in that period and day.")).toBeVisible();

  await form.getByRole("button", { name: "Teacher and subject assignment" }).click();
  await page.getByRole("option", { name: "English · teacher.two" }).click();
  await form.locator(".app-select-trigger").nth(2).click();
  await page.getByRole("option", { name: /2\. Period 2/ }).click();
  await expect(review).toContainText("No existing recurring class for this teacher in the selected slot.");
  await expect(review).toContainText("Already scheduled: History");
  await form.getByRole("button", { name: "Add to timetable" }).click();
  await expect(page.getByText("This section already has a class in that period and day.")).toBeVisible();
  expect(createRequests).toBe(0);
});
