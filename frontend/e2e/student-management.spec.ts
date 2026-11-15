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
  const failStudentReads = new Set<string>();
  let failRegistrationConflict = false;
  let failAdmissionNumberConflict = false;
  let failRollNumberConflict = false;
  const failedStudentReads: string[] = [];
  const observedStudentDetailGets: string[] = [];
  const guardianHistoryRequests: (string | null)[] = [];
  let curriculumAssigned = false;
  let savedCurriculumSelections: number[] = [];
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

    if (method === "GET") {
      const detailRead = path.match(/^\/school\/students\/\d+\/(basic-details|addresses|contact-details)\/?$/)?.[1];
      if (detailRead) observedStudentDetailGets.push(detailRead);
      if (detailRead && failStudentReads.has(detailRead)) {
        failedStudentReads.push(detailRead);
        await route.fulfill({ status: 429, json: { message: `Temporary ${detailRead} read failure.` } });
        return;
      }
    }

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
            "students.archive", "academic_years.read", "classes.read", "subjects.read", "guardians.read", "guardians.manage",
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
      if (failAdmissionNumberConflict) {
        await route.fulfill({
          status: 409,
          json: {
            message: "Admission number is already used in this academic year",
            field_errors: { admission_number: "This admission number is already in use for the selected academic year." },
          },
        });
        return;
      }
      if (failRollNumberConflict) {
        await route.fulfill({
          status: 409,
          json: {
            message: "Roll number is already used in this section",
            field_errors: { roll_number: "This roll number is already assigned to an active student in the selected section." },
          },
        });
        return;
      }
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
    const curriculumEnrollmentMatch = path.match(/^\/school\/enrollments\/(\d+)\/(curriculum|subjects)$/);
    if (curriculumEnrollmentMatch) {
      const enrollmentID = Number(curriculumEnrollmentMatch[1]);
      const studentID = enrollmentID - 100;
      const curriculum = {
        id: 701, school_id: 10, academic_year_id: 7, class_id: 3, version_number: 1, status: "active",
        created_at: "2026-11-14T00:00:00.000Z",
        optional_groups: [{ id: 801, name: "Second language", min_selections: 1, max_selections: 1 }],
        subjects: [
          { id: 901, subject_id: 91, name: "Mathematics", is_compulsory: true },
          { id: 902, subject_id: 92, name: "Hindi", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false },
          { id: 903, subject_id: 93, name: "Sanskrit", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false },
        ],
      };
      if (curriculumEnrollmentMatch[2] === "curriculum" && method === "GET") {
        await route.fulfill({ json: { enrollment_curriculum: {
          enrollment_id: enrollmentID, student_id: studentID, student_status: "active", enrollment_status: "active",
          academic_year_id: 7, class_id: 3, assigned: curriculumAssigned, curriculum,
          selected_optional_subject_ids: savedCurriculumSelections,
          selection_history: savedCurriculumSelections.map(() => ({
            curriculum_version: 1, subject_name: "Hindi", selected_at: "2026-11-14T00:00:00.000Z",
          })),
        } } });
        return;
      }
      if (curriculumEnrollmentMatch[2] === "subjects" && method === "PUT") {
        const input = request.postDataJSON() as { selected_optional_subject_ids: number[] };
        curriculumAssigned = true;
        savedCurriculumSelections = input.selected_optional_subject_ids;
        await route.fulfill({ json: { enrollment_curriculum: {
          enrollment_id: enrollmentID, student_id: studentID, student_status: "active", enrollment_status: "active",
          academic_year_id: 7, class_id: 3, assigned: curriculumAssigned, curriculum,
          selected_optional_subject_ids: savedCurriculumSelections,
          selection_history: savedCurriculumSelections.map(() => ({
            curriculum_version: 1, subject_name: "Hindi", selected_at: "2026-11-14T00:00:00.000Z",
          })),
        } } });
        return;
      }
    }
    const studentMatch = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (studentMatch) {
      const studentID = Number(studentMatch[1]);
      const subpath = studentMatch[2] ?? "";
      if (subpath === "" && method === "PATCH") {
        const input = request.postDataJSON() as { full_name: string };
        const student = students.find(item => item.id === studentID);
        if (!student) {
          await route.fulfill({ status: 404, json: { message: "Student not found." } });
          return;
        }
        student.full_name = input.full_name;
        student.updated_at = new Date().toISOString();
        await route.fulfill({ json: { student } });
        return;
      }
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
        if (method === "PUT" && failRegistrationConflict) {
          await route.fulfill({ status: 409, json: { message: "Student basic details conflicts with an existing record" } });
          return;
        }
        if (method === "PUT") Object.assign(basicDetails, request.postDataJSON() as Record<string, unknown>);
        await route.fulfill({ json: { basic_details: basicDetails } });
        return;
      }
      if (subpath === "guardians" || subpath === "related-students") {
        if (subpath === "guardians") guardianHistoryRequests.push(url.searchParams.get("include_history"));
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
            id: 700, height_cm: 130, weight_kg: 30,
            measured_on: "2026-10-14", vision_notes: "Previous vision note",
            physical_notes: "Previous physical note", created_at: "2026-10-14T00:00:00.000Z",
          }, {
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
  const submitButton = page.getByRole("button", { name: "Admit student" });
  await page.getByPlaceholder("e.g. Aarav Sharma").fill("अ".repeat(201));
  await page.getByPlaceholder("e.g. ADM-2026-001").fill("अ".repeat(51));
  await page.getByPlaceholder("e.g. 01").fill("अ".repeat(31));
  await submitButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Admission submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Student name must be 200 characters or fewer.")).toBeVisible();
  await expect(page.getByText("Admission number must be 50 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Roll number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  expect(createdAdmissions).toHaveLength(0);
  await page.getByPlaceholder("e.g. Aarav Sharma").fill("Aarav Sharma");
  await page.getByPlaceholder("e.g. ADM-2026-001").fill("ADM-E2E-001");
  await page.getByPlaceholder("e.g. 01").fill("01");
  const admissionFormValid = await page.locator(".app-modal-content form").evaluate(
    form => (form as HTMLFormElement).checkValidity(),
  );
  expect(admissionFormValid).toBe(true);
  expect(await submitButton.evaluate(button => ({
    type: (button as HTMLButtonElement).type,
    belongsToForm: (button as HTMLButtonElement).form === button.closest("form"),
  }))).toEqual({ type: "submit", belongsToForm: true });
  failAdmissionNumberConflict = true;
  await submitButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Admission submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("This admission number is already in use for the selected academic year.")).toBeVisible();
  expect(createdAdmissions).toHaveLength(0);
  expect(students).toHaveLength(0);
  const admissionNumberInput = page.getByPlaceholder("e.g. ADM-2026-001");
  await admissionNumberInput.fill("ADM-E2E-RETRY");
  await expect(page.getByText("This admission number is already in use for the selected academic year.")).toHaveCount(0);
  await admissionNumberInput.fill("ADM-E2E-001");

  failAdmissionNumberConflict = false;
  failRollNumberConflict = true;
  await submitButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Admission submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("This roll number is already assigned to an active student in the selected section.")).toBeVisible();
  expect(createdAdmissions).toHaveLength(0);
  expect(students).toHaveLength(0);
  const rollNumberInput = page.getByPlaceholder("e.g. 01");
  await rollNumberInput.fill("02");
  await expect(page.getByText("This roll number is already assigned to an active student in the selected section.")).toHaveCount(0);
  await rollNumberInput.fill("01");

  failRollNumberConflict = false;
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
    roll_number: "01",
  });

  await expect(page.getByRole("button", { name: "Edit profile for Aarav Sharma" })).toBeVisible();
  await page.getByRole("button", { name: "View profile for Aarav Sharma" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Academic history", exact: true })).toBeVisible();
  const curriculumPanel = page.getByRole("region", { name: "Curriculum and subject choices" });
  await expect(curriculumPanel.getByText("Mathematics")).toBeVisible();
  await expect(curriculumPanel.getByText("This curriculum is available for the selected class and academic year. Saving subject choices will assign it to this student’s enrollment.")).toBeVisible();
  await curriculumPanel.getByRole("checkbox", { name: /Hindi/ }).check();
  await expect(curriculumPanel.getByText("1 selected · minimum 1, maximum 1")).toBeVisible();
  await curriculumPanel.getByRole("button", { name: "Save subject choices" }).click();
  await expect(curriculumPanel.getByText("Assigned", { exact: true })).toBeVisible();
  await expect(curriculumPanel.getByText("Previous subject choices (1)")).toBeVisible();
  await curriculumPanel.getByText("Previous subject choices (1)").click();
  await expect(curriculumPanel.getByText("Hindi · curriculum v1 · Current")).toBeVisible();
  await curriculumPanel.getByRole("checkbox", { name: /Hindi/ }).uncheck();
  await curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ }).check();

  failStudentReads.add("basic-details");
  await page.getByRole("button", { name: /Basic details/ }).click();
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toBeVisible();
  await expect.poll(() => observedStudentDetailGets).toContain("basic-details");
  await expect.poll(() => failedStudentReads).toContain("basic-details");
  const basicSave = page.getByRole("button", { name: "Save basic details" });
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(basicSave).toBeDisabled();
  failStudentReads.delete("basic-details");
  await page.getByRole("button", { name: "Retry loading details" }).click();
  await expect(basicSave).toBeEnabled();
  await page.getByPlaceholder("Enter student's full name").fill("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Save student name" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("Aarav Sharma Updated");
  const studentNameInput = page.getByPlaceholder("Enter student's full name");
  await studentNameInput.fill("अ".repeat(201));
  await page.getByRole("button", { name: "Save student name" }).click();
  await expect(page.getByText("Student name must be 200 characters or fewer.")).toBeVisible();
  expect(students[0]?.full_name).toBe("Aarav Sharma Updated");
  await studentNameInput.fill("अ".repeat(200));
  await page.getByRole("button", { name: "Save student name" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("अ".repeat(200));
  await studentNameInput.fill("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Save student name" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Date of birth", exact: true }).click();
  const dobLabels = await page.evaluate(() => {
    const label = (offset: number) => {
      const date = new Date();
      date.setDate(date.getDate() + offset);
      return date.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
    };
    return { yesterday: label(-1), tomorrow: label(1) };
  });
  const dobCalendar = page.getByRole("dialog", { name: "Choose date of birth" });
  await expect(dobCalendar.getByRole("button", { name: dobLabels.tomorrow })).toBeDisabled();
  await dobCalendar.getByRole("button", { name: dobLabels.yesterday }).click();
  await expect(dobCalendar).toBeHidden();
  await expect(page.getByRole("button", { name: "Date of birth", exact: true })).toContainText(/\d{2} [A-Za-z]{3} \d{4}/);
  const basicSaveButton = page.getByRole("button", { name: "Save basic details" });
  await page.getByPlaceholder("City, district or place").fill("अ".repeat(121));
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Place of birth must be 120 characters or fewer.", { exact: true })).toBeVisible();
  expect(basicDetails.place_of_birth).toBeNull();
  await page.getByPlaceholder("City, district or place").fill("Ambala, Haryana");
  await page.getByPlaceholder("e.g. Indian").fill("न".repeat(81));
  await page.getByPlaceholder("Optional", { exact: true }).fill("ध".repeat(81));
  await page.getByPlaceholder("Optional school SR number").fill("प".repeat(51));
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Nationality must be 80 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Religion must be 80 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Registration number must be 50 characters or fewer.", { exact: true })).toBeVisible();
  expect(basicDetails.place_of_birth).toBeNull();
  await page.getByPlaceholder("e.g. Indian").fill("Indian");
  await page.getByPlaceholder("Optional", { exact: true }).fill("");
  await page.getByPlaceholder("Optional school SR number").fill("SR-E2E-001");
  failRegistrationConflict = true;
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("This registration number is already in use for another student in this school.", { exact: true })).toBeVisible();
  expect(basicDetails.school_registration_number).toBeNull();
  failRegistrationConflict = false;
  await page.getByRole("button", { name: "Gender", exact: true }).click();
  await page.getByRole("option", { name: "Male", exact: true }).click();
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => basicDetails.place_of_birth).toBe("Ambala, Haryana");
  expect(basicDetails).toMatchObject({ nationality: "Indian", gender: "male" });
  expect(basicDetails.date_of_birth).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  await expect(page.getByText("Basic details saved.")).toBeVisible();
  await expect(curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ })).toBeChecked();
  await expect(curriculumPanel.getByRole("checkbox", { name: /Hindi/ })).not.toBeChecked();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await expect(page.getByText("Aarav Sharma Updated · Permanent student identity and year-wise academic history")).toBeVisible();

  await page.getByRole("button", { name: /Basic details/ }).click();
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toBeVisible();
  await expect(page.getByPlaceholder("City, district or place")).toHaveValue("Ambala, Haryana");
  await expect(page.getByPlaceholder("e.g. Indian")).toHaveValue("Indian");
  await expect(page.getByRole("button", { name: "Gender", exact: true })).toContainText("Male");
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toBeVisible();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect.poll(() => guardianHistoryRequests.length).toBeGreaterThan(0);
  expect(guardianHistoryRequests[0]).toBeNull();
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
  await page.getByPlaceholder("Mobile number").fill("अ".repeat(31));
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toHaveCount(0);
  await page.getByPlaceholder("Mobile number").fill("9999912345");
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
  await revokeDialog.getByPlaceholder("Explain why this link is being revoked").fill("अ".repeat(501));
  await revokeDialog.getByRole("button", { name: "Confirm revoke" }).click();
  await expect(revokeDialog.getByText("Reason must be 500 characters or fewer.", { exact: true })).toBeVisible();
  expect(guardianLink.revoked_at).toBeNull();
  await revokeDialog.getByPlaceholder("Explain why this link is being revoked").fill("Family requested removal of this contact link.");
  await revokeDialog.getByRole("button", { name: "Confirm revoke" }).click();
  await expect.poll(() => guardianLink.revoked_at).toBe("2026-11-14T12:00:00.000Z");
  await expect(page.getByText("Guardian relationship revoked. Its history is preserved.")).toBeVisible();
  await expect(page.getByText("Ramesh Sharma")).toHaveCount(0);
  await page.getByLabel("Show revoked history").check();
  await expect.poll(() => guardianHistoryRequests[guardianHistoryRequests.length - 1]).toBe("true");
  expect(guardianHistoryRequests[0]).toBeNull();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect(page.getByText(/Revoked/)).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toBeVisible();
  await page.getByLabel("Show revoked history").check();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect(page.getByText("Legal Guardian · Primary contact · Emergency contact")).toBeVisible();
  await expect(page.getByText(/9999912345/)).toBeVisible();
  await expect(page.getByText(/ramesh\.updated@example\.test/)).toBeVisible();
  await expect(page.getByText(/Revoked/)).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  failStudentReads.add("addresses");
  await page.getByRole("button", { name: /Address & communication/ }).click();
  await expect(page.getByRole("dialog", { name: "Address & Communication" })).toBeVisible();
  const addressSave = page.getByRole("button", { name: "Save address" });
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(addressSave).toBeDisabled();
  failStudentReads.delete("addresses");
  await page.getByRole("button", { name: "Retry loading addresses" }).click();
  await expect(addressSave).toBeEnabled();
  await page.getByPlaceholder("House number, street or building").fill("अ".repeat(201));
  await page.getByPlaceholder("Apartment, floor or additional details").fill("अ".repeat(201));
  await page.getByPlaceholder("Local area or mohalla").fill("अ".repeat(121));
  await page.getByPlaceholder("Nearby landmark").fill("अ".repeat(121));
  await page.getByPlaceholder("PIN or postal code").fill("अ".repeat(21));
  await addressSave.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Address save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("House / street must be 200 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Address line 2 must be 200 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Area / mohalla must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Landmark must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("PIN / postal code must be 20 characters or fewer.", { exact: true })).toBeVisible();
  expect(addresses).toHaveLength(0);
  await page.getByPlaceholder("House number, street or building").fill("House 12, Main Road");
  await page.getByPlaceholder("Apartment, floor or additional details").fill("");
  await page.getByPlaceholder("Local area or mohalla").fill("");
  await page.getByPlaceholder("Nearby landmark").fill("");
  await page.getByPlaceholder("PIN or postal code").fill("");
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

  failStudentReads.add("contact-details");
  await page.getByRole("button", { name: "Contact details", exact: true }).click();
  const contactSave = page.getByRole("button", { name: "Save contact details" });
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(contactSave).toBeDisabled();
  failStudentReads.delete("contact-details");
  await page.getByRole("button", { name: "Retry loading contact details" }).click();
  await expect(contactSave).toBeEnabled();
  const contactSaveButton = page.getByRole("button", { name: "Save contact details" });
  await page.getByPlaceholder("Primary contact number").fill("अ".repeat(31));
  await page.getByPlaceholder("Alternate contact number").fill("अ".repeat(31));
  await page.getByPlaceholder("student@example.com").fill("a".repeat(245) + "@example.test");
  await contactSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Contact save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Phone number must be 30 characters or fewer.")).toHaveCount(2);
  await expect(page.getByText("Email address must be 254 characters or fewer.", { exact: true })).toBeVisible();
  expect(contactDetails.primary_phone).toBeNull();
  await page.getByPlaceholder("Primary contact number").fill("9876501234");
  await page.getByPlaceholder("Alternate contact number").fill("");
  await page.getByPlaceholder("student@example.com").fill("not-an-email");
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

  await page.getByRole("button", { name: /Address & communication/ }).click();
  await expect(page.getByRole("dialog", { name: "Address & Communication" })).toBeVisible();
  await expect(page.getByPlaceholder("House number, street or building")).toHaveValue("House 12, Main Road");
  await page.getByRole("button", { name: "Contact details", exact: true }).click();
  await expect(page.getByPlaceholder("Primary contact number")).toHaveValue("9876501234");
  await expect(page.getByPlaceholder("student@example.com")).toHaveValue("student@example.test");
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Medical & physical/ }).click();
  await expect(page.getByRole("dialog", { name: "Medical & Physical Details" })).toBeVisible();
  await expect(page.getByText(/Teachers do not have access to these records/)).toBeVisible();
  await page.getByRole("button", { name: "Blood group", exact: true }).click();
  await page.getByRole("option", { name: "O+" }).click();
  await page.getByPlaceholder("Record known allergies, if any").fill("अ".repeat(2001));
  const medicalSaveButton = page.getByRole("button", { name: "Save medical details" });
  await medicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Medical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Use no more than 2,000 characters.")).toBeVisible();
  expect(medicalDetails.version_number).toBe(0);
  await page.getByPlaceholder("Record known allergies, if any").fill("अ".repeat(2000));
  await page.getByPlaceholder("Record school-relevant conditions, if any").fill("No other known conditions");
  await medicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Medical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => medicalDetails.blood_group).toBe("O+");
  await expect.poll(() => medicalDetails.allergies).toBe("अ".repeat(2000));
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
  await page.getByPlaceholder("Optional vision-related notes").fill("अ".repeat(1001));
  await physicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Use no more than 1,000 characters.")).toBeVisible();
  expect(physicalDetails.height_cm).toBeNull();
  await page.getByPlaceholder("Optional vision-related notes").fill("अ".repeat(1000));
  await physicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => physicalDetails.height_cm).toBe(132.5);
  await expect(page.getByText("Physical details saved.")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Medical & physical/ }).click();
  await expect(page.getByRole("dialog", { name: "Medical & Physical Details" })).toBeVisible();
  await expect(page.getByPlaceholder("Record known allergies, if any")).toHaveValue("अ".repeat(2000));
  await expect(page.getByPlaceholder("Record school-relevant conditions, if any")).toHaveValue("No other known conditions");
  await page.getByRole("button", { name: "Physical details" }).click();
  await expect(page.getByPlaceholder("e.g. 132.5")).toHaveValue("132.5");
  await page.getByText(/Previous physical measurements \(1\)/).click();
  await expect(page.getByText("Measured 14 Oct 2026")).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  expect(browserErrors).toEqual([]);
});


test("standalone enrollment validates admission and roll number before saving", async ({ page }) => {
  const requests: Record<string, unknown>[] = [];
  const student = {
    id: 44, school_id: 10, admission_number: "ADM-EXISTING-044",
    full_name: "Existing Student", status: "active", created_at: "2026-01-01T00:00:00.000Z",
    updated_at: "2026-01-01T00:00:00.000Z",
  };
  const years = [{ id: 7, name: "2026–2027", status: "active", is_current: true }];
  const sections = [
    { id: 12, academic_year_id: 7, class_id: 3, class_name: "Class 1", name: "A", is_active: true },
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
      await route.fulfill({ json: {
        access_token: "e2e-access-token", token_type: "Bearer",
        principal: { principal_id: 101, principal_type: "staff", status: "active", school_id: 10 },
      }});
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 101, principal_type: "staff", school_id: 10, username: "e2e-enrollment-admin",
        permissions: ["students.read_sensitive", "students.read_basic", "students.create", "academic_years.read", "classes.read"],
      }});
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: years } });
      return;
    }
    if (path === "/school/sections") {
      const yearID = Number(url.searchParams.get("academic_year_id") || 0);
      await route.fulfill({ json: { sections: sections.filter(item => item.academic_year_id === yearID) } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students: [student], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/enrollments" && method === "GET") {
      await route.fulfill({ json: { enrollments: [] } });
      return;
    }
    if (path === "/school/enrollments" && method === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      requests.push(input);
      await route.fulfill({ json: { enrollment: { id: 501, school_id: 10, student_id: student.id, ...input, status: "active" } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-enrollment-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "Enrollments" }).click();
  await page.getByRole("button", { name: "Enroll student" }).click();
  await page.getByPlaceholder("Search name or admission number").fill("Existing Student");
  await page.getByRole("button", { name: "Student", exact: true }).click();
  await page.getByRole("option", { name: "Existing Student · ADM-EXISTING-044" }).click();
  await page.getByRole("button", { name: "Enrollment section" }).click();
  await page.getByRole("option", { name: "A" }).click();
  await page.getByRole("button", { name: "Admission date", exact: true }).click();
  await page.getByRole("button", { name: "Today" }).click();

  const saveButton = page.getByRole("button", { name: "Create enrollment" });
  await page.getByPlaceholder("e.g. ADM-2026-001").fill("अ".repeat(51));
  await page.getByPlaceholder("e.g. 01").fill("अ".repeat(31));
  await saveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Enrollment submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Admission number must be 50 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Roll number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  expect(requests).toHaveLength(0);

  await page.getByPlaceholder("e.g. ADM-2026-001").fill("ADM-2027-044");
  await page.getByPlaceholder("e.g. 01").fill("");
  await saveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Enrollment submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => requests.length).toBe(1);
  expect(requests[0]).toMatchObject({
    student_id: 44, academic_year_id: 7, section_id: 12,
    admission_number: "ADM-2027-044", roll_number: null,
  });
  await expect(page.getByText("Student enrolled successfully.")).toBeVisible();
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
  const enrollments: Record<string, unknown>[] = [{
    id: 100, school_id: 10, student_id: 91, student_name: "Promotion Test Student", student_status: "active",
    academic_year_id: 6, section_id: 61, class_name_snapshot: "Class 1", section_name_snapshot: "A",
    admission_number: "ADM-OLD-091", roll_number: "09", admission_date: "2025-04-01",
    status: "active", enrolled_at: "2025-04-01", ended_at: null,
  }];
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
      await route.fulfill({ json: { enrollments, pagination: { limit: 20, offset: 0, total: enrollments.length } } });
      return;
    }
    if (path === "/school/enrollments/100/promote" && request.method() === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      promotionRequests.push(input);
      Object.assign(enrollments[0], { status: "promoted", ended_at: input.admission_date });
      enrollments.push({
        id: 101, school_id: 10, student_id: 91, student_status: "active", academic_year_id: 7, section_id: 72,
        admission_number: String(input.admission_number), class_name_snapshot: "Class 2", section_name_snapshot: "B",
        status: "active", admission_date: input.admission_date, enrolled_at: input.enrolled_at, ended_at: null,
      });
      await route.fulfill({ json: { enrollment: enrollments[1] } });
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

  await dialog.getByPlaceholder("e.g. ADM-2027-001").fill("अ".repeat(51));
  await promoteButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Promotion submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(dialog.getByText("Admission number must be 50 characters or fewer.", { exact: true })).toBeVisible();
  await dialog.getByPlaceholder("e.g. ADM-2027-001").fill("ADM-2026-091");
  await dialog.getByPlaceholder("Leave blank for automatic assignment").fill("अ".repeat(31));
  await promoteButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Promotion submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(dialog.getByText("Roll number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await dialog.getByPlaceholder("Leave blank for automatic assignment").fill("");
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
  await expect(page.getByText("ADM-OLD-091")).toBeVisible();
  await expect(page.getByText("ADM-2026-091")).toBeVisible();
  await expect(page.getByText("promoted", { exact: true })).toBeVisible();
  await expect(dialog).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});

test("student archive confirmation keeps the record on a retryable error and refreshes status after retry", async ({ page }) => {
  const browserErrors: string[] = [];
  const archiveRequests: string[] = [];
  let archiveAttempts = 0;
  const student = {
    id: 92, school_id: 10, admission_number: "ADM-ARCH-092",
    full_name: "Archive Retry Student", status: "active",
    created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
  };
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
      await route.fulfill({ json: {
        access_token: "e2e-archive-token", token_type: "Bearer",
        principal: { principal_id: 104, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 104, principal_type: "staff", school_id: 10, username: "e2e-archive-admin",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "students.archive", "academic_years.read", "classes.read", "guardians.read"],
      } });
      return;
    }
    if (path === "/school/students" && request.method() === "GET") {
      await route.fulfill({ json: { students: [student], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/students/92/basic-details" && request.method() === "GET") {
      await route.fulfill({ json: { basic_details: {
        id: 920, school_id: 10, student_id: 92, date_of_birth: "2017-04-12",
        place_of_birth: "Ambala", gender: "female", nationality: "Indian",
        religion: null, social_category: null, ews_status: null,
        school_registration_number: "SR-ARCH-092",
      } } });
      return;
    }
    if (path === "/school/students/92/guardians" && request.method() === "GET") {
      await route.fulfill({ json: { student_guardians: [{
        id: 9201, student_id: 92, guardian_id: 920, guardian_name: "Archived Student Guardian",
        guardian_status: "active", mobile_number: "9876500920", email_address: "guardian@example.test",
        alternate_mobile_number: null, occupation: "Teacher", education: "Graduate",
        workplace_name: "Local School", relationship_type: "Mother",
        is_primary_contact: true, is_emergency_contact: true, revoked_at: null,
      }] } });
      return;
    }
    if (path === "/school/students/92/related-students" && request.method() === "GET") {
      await route.fulfill({ json: { related_students: [] } });
      return;
    }
    if (path === "/school/students/92/archive" && request.method() === "POST") {
      archiveAttempts += 1;
      archiveRequests.push(request.method() + " " + path);
      if (archiveAttempts === 1) {
        await route.fulfill({ status: 503, json: { message: "Temporary archive failure. Please try again." } });
        return;
      }
      Object.assign(student, { status: "archived", updated_at: "2026-11-14T12:00:00.000Z" });
      await route.fulfill({ json: { student } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-archive-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await expect(page.getByText("Archive Retry Student", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Archive Archive Retry Student" }).click();
  let dialog = page.getByRole("dialog", { name: "Archive student?" });
  await expect(dialog).toBeVisible();
  await expect(dialog.getByText(/active enrollment will be closed as withdrawn/i)).toBeVisible();
  await expect(dialog.getByText(/existing attendance records will be retained/i)).toBeVisible();
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toHaveCount(0);
  expect(archiveAttempts).toBe(0);
  await expect(page.getByText("Archive Retry Student", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Archive Archive Retry Student" }).click();
  dialog = page.getByRole("dialog", { name: "Archive student?" });
  await dialog.getByRole("button", { name: "Archive student", exact: true }).click();
  await expect(dialog.getByRole("alert")).toHaveText("Temporary archive failure. Please try again.");
  await expect(dialog).toBeVisible();
  await expect(page.getByText("Archive Retry Student", { exact: true })).toBeVisible();
  expect(archiveAttempts).toBe(1);

  await dialog.getByRole("button", { name: "Archive student", exact: true }).click();
  await expect.poll(() => archiveAttempts).toBe(2);
  await expect(page.getByText("Archive Retry Student has been archived.")).toBeVisible();
  await expect(page.getByText("archived", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Edit profile for Archive Retry Student" })).toHaveCount(0);
  expect(archiveRequests).toHaveLength(2);

  await page.getByRole("button", { name: "View profile for Archive Retry Student" }).click();
  await page.getByRole("button", { name: "Basic details" }).click();
  await expect(page.getByText("View-only access. Student update permission is required to edit these details.")).toBeVisible();
  await expect(page.getByPlaceholder("City, district or place")).toHaveValue("Ambala");
  await expect(page.getByRole("button", { name: "Save basic details" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save student name" })).toHaveCount(0);
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  const familyDialog = page.getByRole("dialog", { name: "Parents & Siblings" });
  await expect(familyDialog).toBeVisible();
  await expect(page.getByText("Archived Student Guardian", { exact: true })).toBeVisible();
  await expect(page.getByText("Mother · Primary contact · Emergency contact")).toBeVisible();
  await expect(page.getByText("9876500920 · guardian@example.test", { exact: true })).toBeVisible();
  await expect(page.getByText("Teacher · Graduate · Local School", { exact: true })).toBeVisible();
  await expect(page.getByText("View-only access. Guardian records can be changed only for active students and require guardian management permission.")).toBeVisible();
  await expect(familyDialog.getByRole("button", { name: "Edit relationship" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Edit shared guardian details" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Revoke link" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Link guardian" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Create a new guardian contact" })).toHaveCount(0);

  expect(browserErrors).toEqual([]);
});

test("staff can create a guardian contact and link it to a student with relationship flags", async ({ page }) => {
  const browserErrors: string[] = [];
  const guardianCreates: Record<string, unknown>[] = [];
  const guardianLinks: Record<string, unknown>[] = [];
  let failGuardianLinkConflict = false;
  const guardian = {
    id: 45, school_id: 10, full_name: "Nisha Verma", mobile_number: "9876501234",
    email_address: "nisha.verma@example.test", alternate_mobile_number: null,
    occupation: "Teacher", education: null, workplace_name: null, status: "active",
  };
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
      await route.fulfill({ json: {
        access_token: "e2e-guardian-create-token", token_type: "Bearer",
        principal: { principal_id: 105, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 105, principal_type: "staff", school_id: 10, username: "e2e-guardian-manager",
        permissions: ["students.read_sensitive", "students.read_basic", "guardians.read", "guardians.manage", "academic_years.read"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/students" && request.method() === "GET") {
      await route.fulfill({ json: { students: [{
        id: 93, school_id: 10, admission_number: "ADM-GUARD-093",
        full_name: "Guardian Workflow Student", status: "active",
        created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
      }], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/students/93/enrollments") {
      await route.fulfill({ json: { enrollments: [] } });
      return;
    }
    if (path === "/school/students/93/guardians" && request.method() === "GET") {
      await route.fulfill({ json: { student_guardians: guardianLinks } });
      return;
    }
    if (path === "/school/students/93/related-students") {
      await route.fulfill({ json: { related_students: [] } });
      return;
    }
    if (path === "/school/guardians" && request.method() === "GET") {
      await route.fulfill({ json: { guardians: guardianLinks.length ? [] : [guardian], total: guardianLinks.length ? 0 : 1 } });
      return;
    }
    if (path === "/school/guardians" && request.method() === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      guardianCreates.push(input);
      await route.fulfill({ json: { guardian } });
      return;
    }
    if (path === "/school/students/93/guardians" && request.method() === "POST") {
      if (failGuardianLinkConflict) {
        await route.fulfill({ status: 409, json: { message: "Student guardian relationship conflicts with an existing record" } });
        return;
      }
      const input = request.postDataJSON() as Record<string, unknown>;
      const link = {
        id: 953, student_id: 93, guardian_id: 45, guardian_name: guardian.full_name,
        guardian_status: "active", mobile_number: guardian.mobile_number, email_address: guardian.email_address,
        alternate_mobile_number: null, occupation: guardian.occupation, education: null, workplace_name: null,
        relationship_type: input.relationship_type, is_primary_contact: input.is_primary_contact,
        is_emergency_contact: input.is_emergency_contact, revoked_at: null,
      };
      guardianLinks.push(link);
      await route.fulfill({ json: { student_guardian: link } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-guardian-manager");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "View profile for Guardian Workflow Student" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  const familyDialog = page.getByRole("dialog", { name: "Parents & Siblings" });
  await expect(familyDialog).toBeVisible();
  await expect(page.getByText(/Staff-managed guardian records only/)).toBeVisible();

  await page.getByRole("button", { name: "Create a new guardian contact" }).click();
  await page.getByPlaceholder("Guardian full name").fill("अ".repeat(201));
  await page.getByPlaceholder("Mobile number").fill("अ".repeat(31));
  await page.getByPlaceholder("guardian@example.com").fill("a".repeat(245) + "@example.test");
  await page.getByPlaceholder("Optional alternate number").fill("अ".repeat(31));
  await page.getByPlaceholder("e.g. Teacher, Business owner").fill("अ".repeat(121));
  await page.getByPlaceholder("Highest education (optional)").fill("अ".repeat(121));
  await page.getByPlaceholder("Organization name (optional)").fill("अ".repeat(161));
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Guardian name must be 200 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Email address must be 254 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Alternate mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Occupation must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Education must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Workplace must be 160 characters or fewer.", { exact: true })).toBeVisible();
  expect(guardianCreates).toHaveLength(0);

  await page.getByPlaceholder("Guardian full name").fill("Nisha Verma");
  await page.getByPlaceholder("Mobile number").fill("");
  await page.getByPlaceholder("guardian@example.com").fill("not-an-email");
  await page.getByPlaceholder("Optional alternate number").fill("");
  await page.getByPlaceholder("e.g. Teacher, Business owner").fill("");
  await page.getByPlaceholder("Highest education (optional)").fill("");
  await page.getByPlaceholder("Organization name (optional)").fill("");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
  expect(guardianCreates).toHaveLength(0);

  await page.getByPlaceholder("guardian@example.com").fill("nisha.verma@example.test");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Guardian contact created. Select the relationship details and link it to this student.")).toBeVisible();
  expect(guardianCreates).toHaveLength(1);
  expect(guardianCreates[0]).toMatchObject({
    full_name: "Nisha Verma", mobile_number: null,
    email_address: "nisha.verma@example.test", status: "active",
  });

  await page.getByPlaceholder("Father, Mother, Guardian…").fill("अ".repeat(41));
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect(page.getByText("Relationship must be 40 characters or fewer.", { exact: true })).toBeVisible();
  expect(guardianLinks).toHaveLength(0);
  await page.getByPlaceholder("Father, Mother, Guardian…").fill("Mother");
  await page.getByRole("checkbox", { name: /Primary contact/ }).check();
  await page.getByRole("checkbox", { name: /Emergency contact/ }).check();
  failGuardianLinkConflict = true;
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect(page.getByText("This guardian is already linked to this student. Review the existing relationships before adding another link.", { exact: true })).toBeVisible();
  expect(guardianLinks).toHaveLength(0);
  failGuardianLinkConflict = false;
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect.poll(() => guardianLinks.length).toBe(1);
  expect(guardianLinks[0]).toMatchObject({
    guardian_id: 45, relationship_type: "Mother",
    is_primary_contact: true, is_emergency_contact: true,
  });
  await expect(page.getByText("Guardian linked to this student.")).toBeVisible();
  await expect(page.getByText("Nisha Verma", { exact: true })).toBeVisible();
  await expect(page.getByText("Mother · Primary contact · Emergency contact")).toBeVisible();
  expect(browserErrors).toEqual([]);
});


test("staff with guardian management permission cannot change an inactive student's family records", async ({ page }) => {
  const browserErrors: string[] = [];
  const student = {
    id: 94, school_id: 10, admission_number: "ADM-INACTIVE-094",
    full_name: "Inactive Family Student", status: "inactive",
    created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
  };
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
      await route.fulfill({ json: {
        access_token: "e2e-inactive-family-token", token_type: "Bearer",
        principal: { principal_id: 106, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 106, principal_type: "staff", school_id: 10, username: "e2e-inactive-family",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "guardians.read", "guardians.manage", "academic_years.read", "classes.read"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [] } });
      return;
    }
    if (path === "/school/enrollments" && request.method() === "GET") {
      await route.fulfill({ json: { enrollments: [{
        id: 9400, school_id: 10, student_id: 94, student_name: "Inactive Family Student",
        student_status: "inactive", academic_year_id: 6, class_id: 3, section_id: 61,
        class_name_snapshot: "Class 1", section_name_snapshot: "A", roll_number: "04",
        admission_number: "ADM-INACTIVE-094", admission_date: "2025-04-01",
        status: "active", enrolled_at: "2025-04-01", ended_at: null,
      }], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/students" && request.method() === "GET") {
      await route.fulfill({ json: { students: [student], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/students/94/enrollments") {
      await route.fulfill({ json: { enrollments: [] } });
      return;
    }
    if (path === "/school/students/94/basic-details") {
      await route.fulfill({ json: { basic_details: {
        id: 940, school_id: 10, student_id: 94, date_of_birth: "2016-08-12",
        place_of_birth: "Ambala", gender: "female", nationality: "Indian",
        religion: null, social_category: null, ews_status: null,
        school_registration_number: "SR-INACTIVE-094",
      } } });
      return;
    }
    if (path === "/school/students/94/addresses") {
      await route.fulfill({ json: { addresses: [] } });
      return;
    }
    if (path === "/school/locations") {
      await route.fulfill({ json: { locations: [] } });
      return;
    }
    if (path === "/school/students/94/guardians" && request.method() === "GET") {
      await route.fulfill({ json: { student_guardians: [{
        id: 9401, student_id: 94, guardian_id: 940, guardian_name: "Inactive Student Guardian",
        guardian_status: "active", mobile_number: "9876500940", email_address: "inactive.guardian@example.test",
        alternate_mobile_number: null, occupation: "Farmer", education: "Graduate",
        workplace_name: "Family Farm", relationship_type: "Father",
        is_primary_contact: true, is_emergency_contact: true, revoked_at: null,
      }] } });
      return;
    }
    if (path === "/school/students/94/related-students") {
      await route.fulfill({ json: { related_students: [] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-inactive-family");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await expect(page.getByRole("button", { name: "Edit profile for Inactive Family Student" })).toHaveCount(0);
  await page.getByRole("button", { name: "View profile for Inactive Family Student" }).click();
  await page.getByRole("button", { name: "Basic details" }).click();
  await expect(page.getByText("View-only access. Student update permission is required to edit these details.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Save basic details" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save student name" })).toHaveCount(0);
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Address & communication" }).click();
  await expect(page.getByText("View-only access. Student update permission is required to edit addresses or add locations.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Save address" })).toHaveCount(0);
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  const familyDialog = page.getByRole("dialog", { name: "Parents & Siblings" });
  await expect(familyDialog).toBeVisible();
  await expect(page.getByText("Inactive Student Guardian", { exact: true })).toBeVisible();
  await expect(page.getByText("Father · Primary contact · Emergency contact")).toBeVisible();
  await expect(page.getByText("9876500940 · inactive.guardian@example.test", { exact: true })).toBeVisible();
  await expect(page.getByText("Farmer · Graduate · Family Farm", { exact: true })).toBeVisible();
  await expect(page.getByText("View-only access. Guardian records can be changed only for active students and require guardian management permission.")).toBeVisible();
  await expect(familyDialog.getByRole("button", { name: "Edit relationship" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Edit shared guardian details" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Revoke link" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Link guardian" })).toHaveCount(0);
  await expect(familyDialog.getByRole("button", { name: "Create a new guardian contact" })).toHaveCount(0);
  await familyDialog.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Enrollments" }).click();
  await expect(page.getByText("Inactive Family Student", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Promote", exact: true })).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});
