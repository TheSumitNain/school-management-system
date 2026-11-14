import { expect, test } from "@playwright/test";

type Student = {
  id: number;
  school_id: number;
  admission_number: string;
  full_name: string;
  status: "active";
  created_at: string;
  updated_at: string;
};

const academicYear = { id: 7, name: "2026–2027", status: "active", is_current: true };
const section = { id: 12, academic_year_id: 7, class_id: 3, class_name: "Class 1", name: "A", is_active: true };

test("student admission, profile navigation and parent details render without browser errors", async ({ page }) => {
  const browserErrors: string[] = [];
  page.on("pageerror", error => browserErrors.push(error.message));

  let nextStudentID = 1;
  const students: Student[] = [];
  const createdAdmissions: Record<string, unknown>[] = [];

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
      await route.fulfill({
        json: {
          access_token: "e2e-access-token",
          token_type: "Bearer",
          principal: { principal_id: 101, principal_type: "staff", status: "active", school_id: 10 },
        },
      });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({
        json: {
          principal_id: 101,
          principal_type: "staff",
          school_id: 10,
          username: "e2e-school-admin",
          permissions: [
            "students.read_sensitive", "students.read_basic", "students.create", "students.update",
            "students.archive", "academic_years.read", "classes.read", "guardians.read", "guardians.manage",
          ],
        },
      });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [academicYear] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [section] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students, pagination: { limit: 20, offset: 0, total: students.length } } });
      return;
    }
    if (path === "/school/students" && method === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      createdAdmissions.push(input);
      const now = new Date().toISOString();
      const student: Student = {
        id: nextStudentID++,
        school_id: 10,
        admission_number: String(input.admission_number),
        full_name: String(input.full_name),
        status: "active",
        created_at: now,
        updated_at: now,
      };
      students.push(student);
      await route.fulfill({
        json: {
          student,
          enrollment: {
            id: 100 + student.id,
            school_id: 10,
            student_id: student.id,
            academic_year_id: academicYear.id,
            class_name_snapshot: section.class_name,
            section_name_snapshot: section.name,
            admission_number: student.admission_number,
            admission_date: input.admission_date,
            status: "active",
            enrolled_at: input.admission_date,
          },
        },
      });
      return;
    }
    const studentMatch = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (studentMatch) {
      const studentID = Number(studentMatch[1]);
      const subpath = studentMatch[2] ?? "";
      if (subpath === "enrollments") {
        await route.fulfill({
          json: {
            enrollments: [{
              id: 100 + studentID, student_id: studentID, academic_year_id: academicYear.id,
              class_name_snapshot: section.class_name, section_name_snapshot: section.name,
              admission_number: students.find(item => item.id === studentID)?.admission_number ?? "ADM-E2E-001",
              admission_date: "2026-11-14", status: "active", enrolled_at: "2026-11-14",
            }],
          },
        });
        return;
      }
      if (subpath === "basic-details") {
        await route.fulfill({
          json: {
            basic_details: {
              id: studentID, school_id: 10, student_id: studentID, date_of_birth: null,
              place_of_birth: null, gender: null, nationality: null, religion: null,
              social_category: null, ews_status: null, school_registration_number: null,
            },
          },
        });
        return;
      }
      if (subpath === "guardians" || subpath === "related-students") {
        await route.fulfill({ json: subpath === "guardians" ? { student_guardians: [] } : { related_students: [] } });
        return;
      }
      if (subpath === "addresses") {
        await route.fulfill({ json: { addresses: [] } });
        return;
      }
      if (subpath === "contact-details") {
        await route.fulfill({ json: { contact_details: { id: 0, school_id: 10, student_id: studentID } } });
        return;
      }
      if (subpath === "medical-details") {
        await route.fulfill({ json: { medical_details: { id: 0, school_id: 10, student_id: studentID, version_number: 0, history: [] } } });
        return;
      }
      if (subpath === "physical-details") {
        await route.fulfill({ json: { physical_details: { id: 0, school_id: 10, student_id: studentID, history: [] } } });
        return;
      }
    }
    if (path === "/school/guardians") {
      await route.fulfill({ json: { guardians: [], total: 0 } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-school-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();

  await page.getByRole("button", { name: /New admission/ }).click();
  await page.getByPlaceholder("e.g. Aarav Sharma").fill("Aarav Sharma");
  await page.getByPlaceholder("e.g. ADM-2026-001").fill("ADM-E2E-001");
  await page.getByRole("button", { name: "Admission academic year" }).click();
  await page.getByRole("option", { name: "2026–2027" }).click();
  await page.getByRole("button", { name: "Admission section" }).click();
  await page.getByRole("option", { name: "Class 1 · A" }).click();
  await page.getByRole("button", { name: "Admission date", exact: true }).click();
  await page.getByRole("button", { name: "Today" }).click();
  const admissionFormState = await page.locator(".app-modal-content form").evaluate(form => ({
    valid: (form as HTMLFormElement).checkValidity(),
    fields: Array.from((form as HTMLFormElement).elements).map(element => {
      const field = element as HTMLInputElement;
      return { name: field.name, value: field.value, required: field.required, valid: field.validity?.valid };
    }),
  }));
  console.log("Admission form validity:", JSON.stringify(admissionFormState));
  expect(admissionFormState.valid).toBe(true);
  await page.locator(".app-modal-content form").evaluate(form => {
    form.addEventListener("submit", () => document.body.setAttribute("data-e2e-form-submitted", "true"), { once: true });
  });
  const submitButton = page.getByRole("button", { name: "Admit student" });
  expect(await submitButton.evaluate(button => ({
    type: (button as HTMLButtonElement).type,
    belongsToForm: (button as HTMLButtonElement).form === button.closest("form"),
  }))).toEqual({ type: "submit", belongsToForm: true });
  await submitButton.click();
  await expect(page.locator("body")).toHaveAttribute("data-e2e-form-submitted", "true");
  await expect.poll(() => createdAdmissions.length).toBe(1);

  await expect(page.getByText("Student admitted and enrolled for the selected academic year.")).toBeVisible();
  await expect(page.getByText("Aarav Sharma")).toBeVisible();
  expect(createdAdmissions).toHaveLength(1);
  expect(createdAdmissions[0]).toMatchObject({
    full_name: "Aarav Sharma",
    admission_number: "ADM-E2E-001",
    academic_year_id: 7,
    section_id: 12,
    roll_number: null,
  });

  await page.getByRole("button", { name: "View profile for Aarav Sharma" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await expect(page.getByText("Academic history")).toBeVisible();

  await page.getByRole("button", { name: /Basic details/ }).click();
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save basic details" })).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await expect(page.getByRole("dialog", { name: /Parents & siblings/ })).toBeVisible();
  await expect(page.getByText("Other students sharing a guardian")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  expect(browserErrors).toEqual([]);
});
