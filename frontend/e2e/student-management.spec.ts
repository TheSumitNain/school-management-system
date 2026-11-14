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
  const guardianLink = {
    id: 501, student_id: 1, guardian_id: 22, guardian_name: "Ramesh Sharma", guardian_status: "active" as const,
    mobile_number: "9876543210", email_address: "ramesh@example.test", alternate_mobile_number: null,
    occupation: "Teacher", education: "Graduate", workplace_name: "Local School",
    relationship_type: "Father", is_primary_contact: true, is_emergency_contact: true,
    created_at: "2026-11-14T00:00:00.000Z", revoked_at: null,
  };
  const contactDetails: Record<string, unknown> = {
    id: 601, school_id: 10, student_id: 1, primary_phone: null, alternate_phone: null, email_address: null,
  };
  const physicalDetails: Record<string, unknown> = {
    id: 0, school_id: 10, student_id: 1, height_cm: null, weight_kg: null, measured_on: null,
    vision_notes: null, physical_notes: null, history: [],
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
        await route.fulfill({ json: subpath === "guardians" ? { student_guardians: [guardianLink] } : { related_students: [] } });
        return;
      }
      if (subpath === "addresses") {
        await route.fulfill({ json: { addresses: [] } });
        return;
      }
      if (subpath === "contact-details") {
        if (method === "PUT") {
          Object.assign(contactDetails, request.postDataJSON() as Record<string, unknown>);
        }
        await route.fulfill({ json: { contact_details: contactDetails } });
        return;
      }
      if (subpath === "medical-details") {
        await route.fulfill({
          json: { medical_details: {
            id: 0, school_id: 10, student_id: studentID, version_number: 0,
            blood_group: null, allergies: null, known_conditions: null,
            ongoing_medications: null, emergency_medical_notes: null, history: [],
          } },
        });
        return;
      }
      if (subpath === "physical-details") {
        if (method === "PUT") {
          const input = request.postDataJSON() as Record<string, unknown>;
          Object.assign(physicalDetails, input, { id: 701, school_id: 10, student_id: studentID });
          physicalDetails.history = [{
            id: 701, height_cm: physicalDetails.height_cm, weight_kg: physicalDetails.weight_kg,
            measured_on: physicalDetails.measured_on, vision_notes: physicalDetails.vision_notes,
            physical_notes: physicalDetails.physical_notes, created_at: "2026-11-14T00:00:00.000Z",
          }];
        }
        await route.fulfill({ json: { physical_details: physicalDetails } });
        return;
      }
    }
    const guardianLinkMatch = path.match(/^\/school\/student-guardian-links\/(\d+)$/);
    if (guardianLinkMatch && method === "PATCH") {
      const input = request.postDataJSON() as {
        relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean;
      };
      Object.assign(guardianLink, input);
      await route.fulfill({ json: { student_guardian: guardianLink } });
      return;
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
  const admissionFormValid = await page.locator(".app-modal-content form").evaluate(
    form => (form as HTMLFormElement).checkValidity(),
  );
  expect(admissionFormValid).toBe(true);
  const submitButton = page.getByRole("button", { name: "Admit student" });
  expect(await submitButton.evaluate(button => ({
    type: (button as HTMLButtonElement).type,
    belongsToForm: (button as HTMLButtonElement).form === button.closest("form"),
  }))).toEqual({ type: "submit", belongsToForm: true });
  await submitButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Admission submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
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
  await expect(page.getByRole("heading", { name: "Academic history", exact: true })).toBeVisible();

  await page.getByRole("button", { name: /Basic details/ }).click();
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Save basic details" })).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toBeVisible();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect(page.getByText("Father · Primary contact · Emergency contact")).toBeVisible();
  await expect(page.getByText("Other students sharing a guardian")).toBeVisible();

  await page.getByRole("button", { name: "Edit relationship" }).click();
  await page.getByPlaceholder("Father, Mother, Legal Guardian…").fill("Legal Guardian");
  const relationshipForm = page.locator(".student-family-create-form").filter({ has: page.getByRole("heading", { name: "Edit relationship for this student" }) });
  const saveRelationship = relationshipForm.getByRole("button", { name: "Save relationship" });
  await saveRelationship.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Relationship submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => guardianLink.relationship_type).toBe("Legal Guardian");
  await expect(page.getByText("Legal Guardian · Primary contact · Emergency contact")).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Address & communication/ }).click();
  await expect(page.getByRole("dialog", { name: "Address & Communication" })).toBeVisible();
  await page.getByRole("button", { name: "Contact details", exact: true }).click();
  await page.getByPlaceholder("Primary contact number").fill("9876501234");
  const contactSaveButton = page.getByRole("button", { name: "Save contact details" });
  await contactSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Contact save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => contactDetails.primary_phone).toBe("9876501234");
  await expect(page.getByText("Student contact details saved.")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Medical & physical/ }).click();
  await expect(page.getByRole("dialog", { name: "Medical & Physical Details" })).toBeVisible();
  await expect(page.getByText(/Teachers do not have access to these records/)).toBeVisible();
  await page.getByRole("button", { name: "Physical details" }).click();
  await page.getByPlaceholder("e.g. 132.5").fill("500");
  const physicalSaveButton = page.getByRole("button", { name: "Save physical details" });
  await physicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Height must be between 20 and 250 cm.")).toBeVisible();
  await expect(page.getByText("Measurement date is required.")).toBeVisible();
  await page.getByPlaceholder("e.g. 132.5").fill("132.5");
  await page.getByRole("button", { name: "Measurement date", exact: true }).click();
  await page.getByRole("button", { name: "Today" }).click();
  await physicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => physicalDetails.height_cm).toBe(132.5);
  await expect(page.getByText("Physical details saved.")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  expect(browserErrors).toEqual([]);
});
