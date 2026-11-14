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
  const basicDetails: Record<string, unknown> = {
    id: 1, school_id: 10, student_id: 1, date_of_birth: null, place_of_birth: null,
    gender: null, nationality: null, religion: null, social_category: null,
    ews_status: null, school_registration_number: null,
  };
  const physicalDetails: Record<string, unknown> = {
    id: 0, school_id: 10, student_id: 1, height_cm: null, weight_kg: null, measured_on: null,
    vision_notes: null, physical_notes: null, history: [],
  };
  const medicalDetails: Record<string, unknown> = {
    id: 0, school_id: 10, student_id: 1, version_number: 0, blood_group: null,
    allergies: null, known_conditions: null, ongoing_medications: null,
    emergency_medical_notes: null, history: [],
  };
  const addresses: Record<string, unknown>[] = [];
  const locationRows = [
    { id: 11, school_id: null, parent_id: null, country_code: "IN", level: "state", name: "Haryana", is_active: true },
    { id: 21, school_id: null, parent_id: 11, country_code: "IN", level: "district", name: "Ambala", is_active: true },
    { id: 31, school_id: null, parent_id: 21, country_code: "IN", level: "subdistrict", name: "Ambala", is_active: true },
    { id: 41, school_id: null, parent_id: 31, country_code: "IN", level: "locality", name: "Naraingarh", locality_type: "town", is_active: true },
  ];

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
    if (path === "/school/locations") {
      const level = url.searchParams.get("level");
      const parentID = Number(url.searchParams.get("parent_id") || 0);
      await route.fulfill({
        json: { locations: locationRows.filter(item => item.level === level && (parentID === 0 || item.parent_id === parentID)) },
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
        if (method === "PUT") Object.assign(basicDetails, request.postDataJSON() as Record<string, unknown>);
        await route.fulfill({ json: { basic_details: basicDetails } });
        return;
      }
      if (subpath === "guardians" || subpath === "related-students") {
        await route.fulfill({ json: subpath === "guardians" ? { student_guardians: [guardianLink] } : { related_students: [] } });
        return;
      }
      if (subpath === "addresses") {
        await route.fulfill({ json: { addresses } });
        return;
      }
      const addressMatch = subpath.match(/^addresses\/(current|permanent)$/);
      if (addressMatch && method === "PUT") {
        const addressType = addressMatch[1];
        const input = request.postDataJSON() as Record<string, unknown>;
        const existing = addresses.find(item => item.address_type === addressType);
        const location = locationRows.find(item => item.id === input.location_id);
        const saved = {
          id: existing?.id ?? (addressType === "current" ? 801 : 802),
          school_id: 10, student_id: studentID, address_type: addressType,
          address_line_1: input.address_line_1 ?? null, address_line_2: input.address_line_2 ?? null,
          area_mohalla: input.area_mohalla ?? null, landmark: input.landmark ?? null,
          postal_code: input.postal_code ?? null, location_id: input.location_id ?? null,
          location_name: location?.name ?? null, location_level: location?.level ?? null,
          locality_type: location?.locality_type ?? null,
          state_location_id: location?.level === "locality" ? 11 : null,
          district_location_id: location?.level === "locality" ? 21 : null,
          subdistrict_location_id: location?.level === "locality" ? 31 : null,
          state_location_name: location?.level === "locality" ? "Haryana" : null,
          district_location_name: location?.level === "locality" ? "Ambala" : null,
          subdistrict_location_name: location?.level === "locality" ? "Ambala" : null,
          same_as_current: input.same_as_current === true,
        };
        if (existing) Object.assign(existing, saved);
        else addresses.push(saved);
        await route.fulfill({ json: { address: saved } });
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
        if (method === "PUT") {
          const input = request.postDataJSON() as Record<string, unknown>;
          const previousVersion = { ...medicalDetails, changed_by_principal_id: 101, changed_at: "2026-11-14T00:00:00.000Z" };
          Object.assign(medicalDetails, input, { id: 901, school_id: 10, student_id: studentID, version_number: Number(medicalDetails.version_number) + 1 });
          if (Number(previousVersion.version_number) > 0) {
            (medicalDetails.history as Record<string, unknown>[]).unshift(previousVersion);
          }
        }
        await route.fulfill({ json: { medical_details: medicalDetails } });
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
    const guardianRevokeMatch = path.match(/^\/school\/student-guardian-links\/(\d+)\/revoke$/);
    if (guardianRevokeMatch && method === "POST") {
      const input = request.postDataJSON() as { reason: string };
      Object.assign(guardianLink, { revoked_at: "2026-11-14T12:00:00.000Z", revocation_reason: input.reason });
      await route.fulfill({ json: { message: "Guardian relationship revoked." } });
      return;
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
    const guardianProfileMatch = path.match(/^\/school\/guardians\/(\d+)$/);
    if (guardianProfileMatch && method === "PATCH") {
      Object.assign(guardianLink, request.postDataJSON() as Record<string, unknown>);
      await route.fulfill({ json: { guardian: guardianLink } });
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
  await page.getByPlaceholder("City, district or place").fill("Ambala, Haryana");
  await page.getByPlaceholder("e.g. Indian").fill("Indian");
  await page.getByRole("button", { name: "Gender", exact: true }).click();
  await page.getByRole("option", { name: "Male", exact: true }).click();
  const basicSaveButton = page.getByRole("button", { name: "Save basic details" });
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => basicDetails.place_of_birth).toBe("Ambala, Haryana");
  expect(basicDetails).toMatchObject({ nationality: "Indian", gender: "male" });
  await expect(page.getByText("Basic details saved.")).toBeVisible();
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
  await page.getByRole("button", { name: "Edit shared guardian details" }).click();
  await expect(page.getByRole("heading", { name: "Edit shared guardian profile" })).toBeVisible();
  await expect(page.getByText("These contact and profile details are shared across all students linked to this guardian.")).toBeVisible();
  await page.getByPlaceholder("Mobile number").fill("9999912345");
  await page.getByPlaceholder("guardian@example.com").fill("not-an-email");
  const saveGuardianDetails = page.getByRole("button", { name: "Save guardian details" });
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
  expect(guardianLink.mobile_number).toBe("9876543210");
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toHaveCount(0);
  await page.getByPlaceholder("guardian@example.com").fill("ramesh.updated@example.test");
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toBeVisible();
  await page.getByRole("button", { name: "Update all linked records" }).click();
  await expect.poll(() => guardianLink.mobile_number).toBe("9999912345");
  expect(guardianLink).toMatchObject({ relationship_type: "Legal Guardian", is_primary_contact: true, is_emergency_contact: true });
  await expect(page.getByText("Guardian profile updated. Changes apply to every student linked to this guardian.")).toBeVisible();

  await page.getByRole("button", { name: "Revoke link" }).click();
  const revokeDialog = page.getByRole("alertdialog", { name: "Revoke guardian relationship" });
  await expect(revokeDialog).toBeVisible();
  await revokeDialog.getByRole("button", { name: "Confirm revoke" }).click();
  await expect(revokeDialog.getByText("A reason is required to revoke this relationship.")).toBeVisible();
  await revokeDialog.getByPlaceholder("Explain why this link is being revoked").fill("Family requested removal of this contact link.");
  await revokeDialog.getByRole("button", { name: "Confirm revoke" }).click();
  await expect.poll(() => guardianLink.revoked_at).toBe("2026-11-14T12:00:00.000Z");
  await expect(page.getByText("Guardian relationship revoked. Its history is preserved.")).toBeVisible();
  await expect(page.getByText("Ramesh Sharma")).toHaveCount(0);
  await page.getByLabel("Show revoked history").check();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect(page.getByText(/Revoked/)).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Address & communication/ }).click();
  await expect(page.getByRole("dialog", { name: "Address & Communication" })).toBeVisible();
  await page.getByPlaceholder("House number, street or building").fill("House 12, Main Road");
  await page.getByLabel("Search States / UTs").fill("Haryana");
  await page.getByRole("button", { name: "State or Union Territory", exact: true }).click();
  await page.getByRole("option", { name: "Haryana" }).click();
  const addressSaveButton = page.getByRole("button", { name: "Save address" });
  await addressSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Address save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Complete State → District → Sub-District / Tehsil → Village / City, or clear the location selection.")).toBeVisible();
  await page.getByLabel("Search districts").fill("Ambala");
  await page.getByRole("button", { name: "District", exact: true }).click();
  await page.getByRole("option", { name: "Ambala" }).click();
  await page.getByLabel("Search sub-districts / tehsils").fill("Ambala");
  await page.getByRole("button", { name: "Sub-District or Tehsil", exact: true }).click();
  await page.getByRole("option", { name: "Ambala" }).click();
  await page.getByLabel("Search villages / cities").fill("Naraingarh");
  await page.getByRole("button", { name: "Village or City", exact: true }).click();
  await page.getByRole("option", { name: /Naraingarh/ }).click();
  await addressSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Address save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => addresses.length).toBe(1);
  expect(addresses[0]).toMatchObject({
    address_type: "current", address_line_1: "House 12, Main Road", location_id: 41,
    state_location_id: 11, district_location_id: 21, subdistrict_location_id: 31,
  });
  await expect(page.getByText("Current address saved.")).toBeVisible();

  await page.getByRole("button", { name: "Permanent address", exact: true }).click();
  await page.getByRole("checkbox", { name: /Permanent address is same as current address/ }).check();
  await addressSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Address save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => addresses.length).toBe(2);
  expect(addresses.find(item => item.address_type === "permanent")).toMatchObject({ same_as_current: true });
  await expect(page.getByText("Permanent address saved.")).toBeVisible();

  await page.getByRole("button", { name: "Contact details", exact: true }).click();
  await page.getByPlaceholder("Primary contact number").fill("9876501234");
  await page.getByPlaceholder("student@example.com").fill("not-an-email");
  const contactSaveButton = page.getByRole("button", { name: "Save contact details" });
  await contactSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Contact save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
  expect(contactDetails.primary_phone).toBeNull();
  await page.getByPlaceholder("student@example.com").fill("student@example.test");
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
  await page.getByRole("button", { name: "Blood group", exact: true }).click();
  await page.getByRole("option", { name: "O+" }).click();
  await page.getByPlaceholder("Record known allergies, if any").fill("Peanuts");
  await page.getByPlaceholder("Record school-relevant conditions, if any").fill("No other known conditions");
  const medicalSaveButton = page.getByRole("button", { name: "Save medical details" });
  await medicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Medical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => medicalDetails.blood_group).toBe("O+");
  await expect.poll(() => medicalDetails.allergies).toBe("Peanuts");
  await expect.poll(() => medicalDetails.known_conditions).toBe("No other known conditions");
  await expect.poll(() => medicalDetails.version_number).toBe(1);
  await expect(page.getByText("Medical details saved.")).toBeVisible();
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

test("student directory remains view-only when update and archive permissions are absent", async ({ page }) => {
  const writeRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname.startsWith("/api/school/") && !["GET", "HEAD", "OPTIONS"].includes(request.method())) {
      writeRequests.push(`${request.method()} ${url.pathname}`);
    }
  });

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({
        json: {
          access_token: "e2e-view-only-token",
          token_type: "Bearer",
          principal: { principal_id: 102, principal_type: "staff", status: "active", school_id: 10 },
        },
      });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({
        json: {
          principal_id: 102,
          principal_type: "staff",
          school_id: 10,
          username: "e2e-view-only-staff",
          permissions: ["students.read_sensitive", "students.read_basic", "guardians.read", "academic_years.read", "classes.read"],
        },
      });
      return;
    }
    if (path === "/school/students/91/guardians") {
      await route.fulfill({ json: { student_guardians: [{
        id: 951, student_id: 91, guardian_id: 44, guardian_name: "Guardian E2E",
        guardian_status: "active", relationship_type: "Mother",
        is_primary_contact: true, is_emergency_contact: false, revoked_at: null,
      }] } });
      return;
    }
    if (path === "/school/students/91/related-students") {
      await route.fulfill({ json: { related_students: [] } });
      return;
    }
    if (path === "/school/students/91/enrollments") {
      await route.fulfill({ json: { enrollments: [] } });
      return;
    }
    if (path === "/school/students" && request.method() === "GET") {
      await route.fulfill({
        json: {
          students: [{
            id: 91, school_id: 10, admission_number: "ADM-VIEW-091",
            full_name: "View Only Student", status: "active",
            created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
          }],
          pagination: { limit: 20, offset: 0, total: 1 },
        },
      });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-view-only-staff");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();

  await expect(page.getByText("View Only Student")).toBeVisible();
  await expect(page.getByRole("button", { name: "View profile for View Only Student" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Edit profile for View Only Student" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Archive View Only Student" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /New admission/ })).toHaveCount(0);
  await page.getByRole("button", { name: "View profile for View Only Student" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  await expect(page.getByRole("dialog", { name: "Parents & Siblings" })).toBeVisible();
  await expect(page.getByText("Guardian E2E")).toBeVisible();
  await expect(page.getByText(/View-only access/)).toBeVisible();
  await expect(page.getByRole("button", { name: "Edit relationship" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Edit shared guardian details" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Revoke link" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Link guardian" })).toHaveCount(0);
  expect(writeRequests).toEqual([]);
});

test("student promotion validates required fields and preserves prior enrollment history", async ({ page }) => {
  const promotionRequests: Record<string, unknown>[] = [];
  const browserErrors: string[] = [];
  page.on("pageerror", error => browserErrors.push(error.message));

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({
        json: {
          access_token: "e2e-promotion-token",
          token_type: "Bearer",
          principal: { principal_id: 103, principal_type: "staff", status: "active", school_id: 10 },
        },
      });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({
        json: {
          principal_id: 103, principal_type: "staff", school_id: 10, username: "e2e-promotion-admin",
          permissions: ["students.read_sensitive", "students.read_basic", "students.update", "academic_years.read", "classes.read"],
        },
      });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [
        { id: 6, name: "2025–2026", status: "active", is_current: false },
        { id: 7, name: "2026–2027", status: "active", is_current: true },
      ] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [{
        id: 72, academic_year_id: 7, class_id: 4, class_name: "Class 2", name: "B", is_active: true,
      }] } });
      return;
    }
    if (path === "/school/students" && request.method() === "GET") {
      await route.fulfill({ json: { students: [], pagination: { limit: 20, offset: 0, total: 0 } } });
      return;
    }
    if (path === "/school/enrollments" && request.method() === "GET") {
      await route.fulfill({
        json: {
          enrollments: [{
            id: 100, school_id: 10, student_id: 91, student_name: "Promotion Test Student",
            academic_year_id: 6, section_id: 61, class_name_snapshot: "Class 1", section_name_snapshot: "A",
            admission_number: "ADM-OLD-091", roll_number: "09", admission_date: "2025-04-01",
            status: "active", enrolled_at: "2025-04-01", ended_at: null,
          }],
          pagination: { limit: 20, offset: 0, total: 1 },
        },
      });
      return;
    }
    if (path === "/school/enrollments/100/promote" && request.method() === "POST") {
      promotionRequests.push(request.postDataJSON() as Record<string, unknown>);
      await route.fulfill({ json: { enrollment: {
        id: 101, school_id: 10, student_id: 91, academic_year_id: 7, section_id: 72,
        admission_number: "ADM-2026-091", class_name_snapshot: "Class 2", section_name_snapshot: "B",
        status: "active", admission_date: "2026-11-14", enrolled_at: "2026-11-14",
      } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-promotion-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "Enrollments" }).click();
  await expect(page.getByText("Promotion Test Student")).toBeVisible();
  await page.getByRole("button", { name: "Promote" }).click();
  const dialog = page.getByRole("dialog", { name: "Promote student" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("button", { name: "Destination academic year", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Destination class and section", exact: true }).click();
  await page.getByRole("option", { name: "Class 2 · B" }).click();

  const promoteButton = dialog.getByRole("button", { name: "Promote student" });
  await promoteButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Promotion submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(dialog.getByText("Enter the admission number for the destination year.")).toBeVisible();

  await dialog.getByPlaceholder("e.g. ADM-2027-001").fill("ADM-2026-091");
  await promoteButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Promotion submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => promotionRequests.length).toBe(1);
  expect(promotionRequests[0]).toMatchObject({
    academic_year_id: 7, section_id: 72, admission_number: "ADM-2026-091", roll_number: null,
  });
  expect(promotionRequests[0].admission_date).toBeTruthy();
  expect(promotionRequests[0].enrolled_at).toBe(promotionRequests[0].admission_date);
  await expect(page.getByText("Student promoted successfully. Previous academic history has been preserved.")).toBeVisible();
  await expect(dialog).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});
