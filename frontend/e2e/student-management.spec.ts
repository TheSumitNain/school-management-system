import { expect, test, type Page } from "@playwright/test";
import { schoolDateValue, schoolDateValueYearsAgo } from "../src/shared/date/schoolDate";

// SQL DATE values are serialized as midnight UTC timestamps. A negative-offset
// browser timezone reproduces the previous-day display regression.
test.use({ timezoneId: "America/Los_Angeles" });

test("school calendar date helpers use Asia/Kolkata midnight", () => {
  expect(schoolDateValue(new Date("2026-11-17T18:29:59.000Z"))).toBe("2026-11-17");
  expect(schoolDateValue(new Date("2026-11-17T18:30:00.000Z"))).toBe("2026-11-18");
  // 2024-02-29 in IST minus 130 years clamps to the last valid day of February.
  expect(schoolDateValueYearsAgo(130, new Date("2024-02-28T18:30:00.000Z"))).toBe("1894-02-28");
});

async function invalidateProfileQueries(page: Page, family: "students" | "guardians") {
  await page.evaluate(async (queryFamily) => {
    const moduleUrl = new URL("/src/app/providers.tsx", window.location.origin).href;
    const { queryClient } = await import(moduleUrl);
    await queryClient.invalidateQueries({ queryKey: ["school-admin", queryFamily] });
  }, family);
}

type Student = {
  id: number;
  school_id: number;
  admission_number: string;
  full_name: string;
  status: "active";
  created_at: string;
  updated_at: string;
};

const academicYear = { id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true };
const previousAcademicYear = { id: 6, name: "2025–2026", starts_on: "2025-04-01", ends_on: "2026-03-31", status: "inactive", is_current: false };
const section = { id: 12, academic_year_id: 7, class_id: 3, class_name: "Class 1", name: "A", is_active: true };

test("student admission, profile navigation and parent details render without browser errors", async ({ page }) => {
  test.setTimeout(60_000);
  const browserErrors: string[] = [];
  page.on("pageerror", error => browserErrors.push(error.message));

  let nextStudentID = 1;
  const students: Student[] = [];
  const studentDirectoryQueries: string[] = [];
  const createdAdmissions: Record<string, unknown>[] = [];
  const failStudentReads = new Set<string>();
  let failRegistrationConflict = false;
  let failDateOfBirthValidation = false;
  let failMeasurementDateValidation = false;
  let failAdmissionNumberConflict = false;
  let failRollNumberConflict = false;
  const failedStudentReads: string[] = [];
  const observedStudentDetailGets: string[] = [];
  const guardianHistoryRequests: (string | null)[] = [];
  const guardianLinkUpdateRequests: Record<string, unknown>[] = [];
  let failGuardianRelationshipUpdateConflict = false;
  const guardianProfileUpdateRequests: Record<string, unknown>[] = [];
  let rejectSharedGuardianEmailOnce = true;
  let rejectStudentContactEmailOnce = true;
  let curriculumAssigned = false;
  let curriculumReads = 0;
  let medicalDetailReads = 0;
  let physicalDetailReads = 0;
  let savedCurriculumSelections: number[] = [];
  const guardianLink = {
    id: 501, student_id: 1, guardian_id: 22, guardian_name: "Ramesh Sharma", guardian_status: "active" as const,
    mobile_number: "9876543210", email_address: "ramesh@example.test", alternate_mobile_number: null,
    occupation: "Teacher", education: "Graduate", workplace_name: "Local School",
    relationship_type: " father ", is_primary_contact: true, is_emergency_contact: true,
    created_at: "2026-11-14T00:00:00.000Z", revoked_at: null,
  };
  const legacyDuplicateGuardianLink = {
    id: 502, student_id: 1, guardian_id: 23, guardian_name: "Meena Sharma", guardian_status: "active" as const,
    mobile_number: "9876500023", email_address: "meena@example.test", alternate_mobile_number: null,
    occupation: "Homemaker", education: null, workplace_name: null,
    relationship_type: "FATHER ", is_primary_contact: false, is_emergency_contact: false,
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
      await route.fulfill({ json: { academic_years: [academicYear, previousAcademicYear] } });
      return;
    }
    if (path === "/school/sections") {
      await route.fulfill({ json: { sections: [section] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      const requestedYearID = Number(url.searchParams.get("academic_year_id") || academicYear.id);
      studentDirectoryQueries.push(url.searchParams.get("academic_year_id") ?? "");
      const directoryStudents = students.map(student => ({
        ...student,
        admission_number: requestedYearID === previousAcademicYear.id ? "" : student.admission_number,
      }));
      await route.fulfill({ json: { students: directoryStudents, pagination: { limit: 20, offset: 0, total: directoryStudents.length } } });
      return;
    }
    if (path === "/school/enrollment-directory" && method === "GET") {
      await route.fulfill({ json: { enrollments: students.map(student => ({
        id: 100 + student.id, school_id: 10, student_id: student.id,
        student_name: student.full_name, student_status: student.status,
        academic_year_id: academicYear.id, section_id: section.id,
        class_name_snapshot: section.class_name, section_name_snapshot: section.name,
        admission_number: student.admission_number, admission_date: "2026-11-14T00:00:00Z",
        status: "active", enrolled_at: "2026-11-14T00:00:00Z", ended_at: null,
      })), pagination: { limit: 20, offset: Number(url.searchParams.get("offset") || 0), total: students.length } } });
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
          { id: 901, subject_id: 91, name: "Mathematics", is_compulsory: true, is_optional: false },
          { id: 902, subject_id: 92, name: "Hindi", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
          { id: 903, subject_id: 93, name: "Sanskrit", optional_group_id: 801, optional_group_name: "Second language", is_compulsory: false, is_optional: true },
          { id: 904, subject_id: 94, name: "Computer Applications", is_compulsory: false, is_optional: true },
        ],
      };
      if (curriculumEnrollmentMatch[2] === "curriculum" && method === "GET") {
        curriculumReads++;
        await route.fulfill({ json: { enrollment_curriculum: {
          enrollment_id: enrollmentID, student_id: studentID, student_status: "active", enrollment_status: "active",
          academic_year_id: 7, class_id: 3, assigned: curriculumAssigned, curriculum,
          selected_optional_subject_ids: savedCurriculumSelections,
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
              admission_date: "2026-11-14T00:00:00Z", status: "active", enrolled_at: "2026-11-14T00:00:00Z",
            }],
          },
        });
        return;
      }
      if (subpath === "basic-details") {
        if (method === "PUT" && failDateOfBirthValidation) {
          await route.fulfill({ status: 400, json: { message: "Enter a valid date of birth", field_errors: { date_of_birth: "Date of birth must be within the past 130 years and cannot be in the future." } } });
          return;
        }
        if (method === "PUT" && failRegistrationConflict) {
          await route.fulfill({ status: 409, json: { message: "School registration number is already used by another student", field_errors: { school_registration_number: "This registration number is already in use for another student in this school." } } });
          return;
        }
        if (method === "PUT") Object.assign(basicDetails, request.postDataJSON() as Record<string, unknown>);
        await route.fulfill({ json: { basic_details: basicDetails } });
        return;
      }
      if (subpath === "guardians" || subpath === "related-students") {
        if (subpath === "guardians") guardianHistoryRequests.push(url.searchParams.get("include_history"));
        await route.fulfill({ json: subpath === "guardians" ? { student_guardians: [guardianLink, legacyDuplicateGuardianLink] } : { related_students: [] } });
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
          const input = request.postDataJSON() as Record<string, unknown>;
          if (rejectStudentContactEmailOnce && input.email_address === "server-invalid@example.test") {
            rejectStudentContactEmailOnce = false;
            await route.fulfill({ status: 400, json: {
              message: "Enter a valid student email address",
              field_errors: { email_address: "Enter a valid email address using one address only." },
            } });
            return;
          }
          Object.assign(contactDetails, input);
        }
        await route.fulfill({ json: { contact_details: contactDetails } });
        return;
      }
      if (subpath === "medical-details") {
        if (method === "GET") medicalDetailReads++;
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
        if (method === "GET") physicalDetailReads++;
        if (method === "PUT" && failMeasurementDateValidation) {
          await route.fulfill({ status: 400, json: { message: "Enter a valid measurement date", field_errors: { measured_on: "Measurement date is required and cannot be in the future." } } });
          return;
        }
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
      guardianLinkUpdateRequests.push(input);
      if (failGuardianRelationshipUpdateConflict && input.relationship_type === "Mother") {
        await route.fulfill({ status: 409, json: { message: "This relationship type is already linked to this student" } });
        return;
      }
      Object.assign(guardianLink, input);
      await route.fulfill({ json: { student_guardian: guardianLink } });
      return;
    }
    const guardianProfileMatch = path.match(/^\/school\/guardians\/(\d+)$/);
    if (guardianProfileMatch && method === "PATCH") {
      const input = request.postDataJSON() as {
        full_name: string; mobile_number: string | null; email_address: string | null;
        alternate_mobile_number: string | null; occupation: string | null;
        education: string | null; workplace_name: string | null;
      };
      if (rejectSharedGuardianEmailOnce && input.email_address === "server-invalid@example.test") {
        rejectSharedGuardianEmailOnce = false;
        await route.fulfill({
          status: 400,
          json: {
            message: "Enter a valid guardian email address",
            field_errors: { email_address: "Enter a valid email address using one address only." },
          },
        });
        return;
      }
      guardianProfileUpdateRequests.push(input);
      Object.assign(guardianLink, {
        guardian_name: input.full_name, mobile_number: input.mobile_number,
        email_address: input.email_address, alternate_mobile_number: input.alternate_mobile_number,
        occupation: input.occupation, education: input.education, workplace_name: input.workplace_name,
      });
      await route.fulfill({ json: { guardian: {
        id: 22, school_id: 10, full_name: input.full_name,
        mobile_number: input.mobile_number, email_address: input.email_address,
        alternate_mobile_number: input.alternate_mobile_number, occupation: input.occupation,
        education: input.education, workplace_name: input.workplace_name, status: "active",
      } } });
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
  await expect(page.getByRole("heading", { name: "Students", exact: true })).toBeVisible({ timeout: 15_000 });
  const studentDirectoryTab = page.getByRole("button", { name: "Student directory" });
  await expect(studentDirectoryTab).toBeVisible({ timeout: 15_000 });
  await expect(studentDirectoryTab).toHaveAttribute("aria-pressed", "true");
  await expect(page.getByRole("button", { name: "Filter students by academic year" })).toBeVisible({ timeout: 15_000 });
  await expect.poll(() => studentDirectoryQueries.at(-1), { timeout: 15_000 }).toBe(String(academicYear.id));
  const directoryYearFilter = page.getByRole("button", { name: "Filter students by academic year" });
  await expect(directoryYearFilter).toBeVisible();
  const directoryYearOptions = page.getByRole("listbox", { name: "Filter students by academic year" });
  await directoryYearFilter.click();
  await expect(directoryYearOptions).toBeVisible();
  await directoryYearOptions.getByRole("option").nth(1).click();
  await expect.poll(() => studentDirectoryQueries.at(-1)).toBe(String(previousAcademicYear.id));

  await page.getByRole("button", { name: /New admission/ }).click();
  const admissionDialog = page.getByRole("dialog", { name: "New student admission" });
  await expect(admissionDialog.locator(".student-field-hint__tooltip")).toHaveCount(0);
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

  await expect(page.getByRole("region", { name: "Notifications" }).getByText("Student admitted and enrolled for the selected academic year.")).toBeVisible();
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
  await expect(page.getByText("Not enrolled", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Enrollments", exact: true }).click();
  await expect(page.getByText("14 Nov 2026", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Student directory", exact: true }).click();
  await page.getByRole("button", { name: "Edit profile for Aarav Sharma" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Academic history", exact: true })).toBeVisible();
  const profileDialog = page.getByRole("dialog", { name: /Student profile/ });
  await expect(profileDialog.getByText("14 Nov 2026", { exact: true })).toHaveCount(2);
  const curriculumPanel = page.getByRole("region", { name: "Curriculum and subject choices" });
  await expect(curriculumPanel.getByText("Mathematics")).toBeVisible();
  const individualOptional = curriculumPanel.getByRole("checkbox", { name: /Computer Applications/ });
  await expect(individualOptional).toBeVisible();
  await individualOptional.check();
  await expect(individualOptional).toBeChecked();
  await expect(curriculumPanel.getByText("This curriculum is available for the selected class and academic year. Saving subject choices will assign it to this student’s enrollment.")).toBeVisible();
  const saveSubjectChoices = curriculumPanel.getByRole("button", { name: "Save subject choices" });
  // A required optional group cannot be saved empty, and its maximum must be enforced in the UI.
  await expect(saveSubjectChoices).toBeDisabled();
  await curriculumPanel.getByRole("checkbox", { name: /Hindi/ }).check();
  await expect(curriculumPanel.getByText("1 selected · minimum 1, maximum 1")).toBeVisible();
  await expect(saveSubjectChoices).toBeEnabled();
  await curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ }).check();
  await expect(curriculumPanel.getByText("2 selected · minimum 1, maximum 1")).toBeVisible();
  await expect(curriculumPanel.getByText("Select exactly 1 subject in Second language.")).toBeVisible();
  await expect(saveSubjectChoices).toBeDisabled();
  await curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ }).uncheck();
  await expect(saveSubjectChoices).toBeEnabled();
  await saveSubjectChoices.click();
  await expect(curriculumPanel.getByText("Assigned", { exact: true })).toBeVisible();
  // Standalone and grouped optional choices are persisted as current rows;
  // no previous-choice history is exposed in Student Management.
  expect(savedCurriculumSelections).toEqual([904, 902]);
  await expect(curriculumPanel.getByText(/Previous subject choices/)).toHaveCount(0);
  await curriculumPanel.getByRole("checkbox", { name: /Hindi/ }).uncheck();
  await curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ }).check();
  const readsBeforeRefresh = curriculumReads;
  await curriculumPanel.getByRole("button", { name: "Refresh curriculum and subject choices" }).click();
  await expect.poll(() => curriculumReads).toBeGreaterThan(readsBeforeRefresh);
  await expect(curriculumPanel.getByRole("checkbox", { name: /Sanskrit/ })).toBeChecked();
  await expect(curriculumPanel.getByRole("checkbox", { name: /Hindi/ })).not.toBeChecked();

  failStudentReads.add("basic-details");
  await page.getByRole("button", { name: /Basic details/ }).click();
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Basic Details" }).locator(".student-field-hint__tooltip")).toHaveCount(0);
  await expect.poll(() => observedStudentDetailGets).toContain("basic-details");
  await expect.poll(() => failedStudentReads).toContain("basic-details");
  await expect(page.getByRole("button", { name: "Save student name" })).toHaveCount(0);
  const basicSave = page.getByRole("button", { name: "Save basic details" });
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(basicSave).toBeDisabled();
  failStudentReads.delete("basic-details");
  await page.getByRole("button", { name: "Retry loading details" }).click();
  await expect(basicSave).toBeEnabled();

  // Field help must be available to pointer/touch users and keyboard users,
  // not only as a hover tooltip.
  await expect(page.locator(".student-field-hint__tooltip")).toHaveCount(0);
  const dobHelp = page.getByRole("button", { name: "Help for Date of birth" });
  await dobHelp.click();
  await expect(dobHelp).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("tooltip")).toContainText("future dates are not allowed");
  await dobHelp.click();
  await expect(dobHelp).toHaveAttribute("aria-expanded", "false");
  await dobHelp.focus();
  await dobHelp.press("Enter");
  await expect(dobHelp).toHaveAttribute("aria-expanded", "true");
  await dobHelp.press("Escape");
  await expect(dobHelp).toHaveAttribute("aria-expanded", "false");

  await page.getByPlaceholder("Enter student's full name").fill("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Save basic details" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("Aarav Sharma Updated");
  const studentNameInput = page.getByPlaceholder("Enter student's full name");
  await studentNameInput.fill("अ".repeat(201));
  await page.getByRole("button", { name: "Save basic details" }).click();
  await expect(page.getByText("Student name must be 200 characters or fewer.")).toBeVisible();
  expect(students[0]?.full_name).toBe("Aarav Sharma Updated");
  await studentNameInput.fill("अ".repeat(200));
  await page.getByRole("button", { name: "Save basic details" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("अ".repeat(200));
  await studentNameInput.fill("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Save basic details" }).click();
  await expect(page.getByText("Student name saved.")).toBeVisible();
  await expect.poll(() => students[0]?.full_name).toBe("Aarav Sharma Updated");
  await page.getByRole("button", { name: "Date of birth", exact: true }).click();
  const dobLabels = await page.evaluate(() => {
    const parts = new Intl.DateTimeFormat("en", {
      timeZone: "Asia/Kolkata", year: "numeric", month: "2-digit", day: "2-digit",
    }).formatToParts(new Date());
    const part = (type: string) => Number(parts.find(item => item.type === type)?.value);
    const today = new Date(part("year"), part("month") - 1, part("day"));
    const label = (offset: number) => {
      const date = new Date(today);
      date.setDate(date.getDate() + offset);
      return date.toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long", year: "numeric" });
    };
    return { yesterday: label(-1), tomorrow: label(1) };
  });
  const dobCalendar = page.getByRole("dialog", { name: "Choose date of birth" });
  await dobCalendar.getByRole("button", { name: "Choose year" }).click();
  await dobCalendar.getByRole("button", { name: "Previous decade" }).click();
  await dobCalendar.getByRole("button", { name: "Previous decade" }).click();
  await dobCalendar.getByRole("button", { name: "Previous decade" }).click();
  await dobCalendar.getByRole("button", { name: "1999", exact: true }).click();
  await expect(dobCalendar.getByRole("button", { name: "Choose year" })).toHaveText("1999");
  await page.keyboard.press("Escape");
  await expect(dobCalendar).toBeHidden();
  await page.getByRole("button", { name: "Date of birth", exact: true }).click();
  const activeDOBDay = dobCalendar.locator('button[tabindex="0"]');
  const initialDOBDay = await activeDOBDay.getAttribute("aria-label");
  await expect(activeDOBDay).toBeFocused();
  await activeDOBDay.press("ArrowLeft");
  const nextDOBDay = dobCalendar.locator('button[tabindex="0"]');
  await expect(nextDOBDay).not.toHaveAttribute("aria-label", initialDOBDay ?? "");
  await expect(nextDOBDay).toBeFocused();
  await nextDOBDay.press("Escape");
  await expect(dobCalendar).toBeHidden();
  const dobTrigger = page.getByRole("button", { name: "Date of birth", exact: true });
  await expect(dobTrigger).toBeFocused();
  await dobTrigger.click();
  await expect(dobCalendar.getByRole("button", { name: dobLabels.tomorrow })).toBeDisabled();
  await dobCalendar.getByRole("button", { name: dobLabels.yesterday }).click();
  await expect(dobCalendar).toBeHidden();
  await expect(page.getByRole("button", { name: "Date of birth", exact: true })).toContainText(/\d{2} [A-Za-z]{3} \d{4}/);
  const basicSaveButton = page.getByRole("button", { name: "Save basic details" });
  failDateOfBirthValidation = true;
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Date of birth must be within the past 130 years and cannot be in the future.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Date of birth", exact: true })).toHaveAttribute("aria-invalid", "true");
  failDateOfBirthValidation = false;
  await page.getByPlaceholder("City, district or place").fill("अ".repeat(121));
  await basicSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Basic details save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Place of birth must be 120 characters or fewer.", { exact: true })).toBeVisible();
  expect(basicDetails.place_of_birth).toBeNull();
  await page.getByPlaceholder("City, district or place").fill("Ambala, Haryana");
  const basicReadsBeforeFocus = observedStudentDetailGets.filter(item => item === "basic-details").length;
  await invalidateProfileQueries(page, "students");
  await expect.poll(() => observedStudentDetailGets.filter(item => item === "basic-details").length).toBeGreaterThan(basicReadsBeforeFocus);
  await expect(page.getByPlaceholder("City, district or place")).toHaveValue("Ambala, Haryana");
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
  await expect(page.getByText("Existing duplicate active relationships found: Father. Review the guardian details and revoke only the incorrect link. No records have been removed automatically.")).toBeVisible();
  await expect(page.getByText("Meena Sharma", { exact: true })).toBeVisible();
  const guardianCard = page.locator(".student-family-card").filter({ hasText: "Ramesh Sharma" });

  // A legacy value with mixed case/whitespace must still block a duplicate
  // Father link, while the existing relationship remains selectable for edit.
  const addRelationshipSelect = page.getByRole("button", { name: "Relationship to student", exact: true });
  await addRelationshipSelect.click();
  const addRelationshipList = page.getByRole("listbox", { name: "Relationship to student" });
  await expect(addRelationshipList.getByRole("option")).toHaveCount(8);
  await expect(addRelationshipList.getByRole("option", { name: "Father (already linked)" })).toBeDisabled();
  await page.keyboard.press("Escape");

  await guardianCard.getByRole("button", { name: "Edit relationship" }).click();
  const relationshipForm = page.locator(".student-family-create-form").filter({ has: page.getByRole("heading", { name: "Edit relationship for this student" }) });
  const relationshipSelect = relationshipForm.getByRole("button", { name: "Relationship to student", exact: true });
  await relationshipSelect.click();
  const editRelationshipList = relationshipForm.getByRole("listbox", { name: "Relationship to student" });
  await expect(editRelationshipList).toBeVisible();
  await expect(editRelationshipList.getByRole("option", { name: "Father (already linked)", exact: true })).toBeDisabled();
  await expect(editRelationshipList.getByRole("option", { name: "Father (already linked)", exact: true })).toHaveAttribute("aria-selected", "true");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toBeVisible();
  await expect(relationshipSelect).toHaveAttribute("aria-expanded", "false");
  await relationshipSelect.click();
  const editRelationshipOptions = relationshipForm.getByRole("listbox", { name: "Relationship to student" }).getByRole("option");
  await expect(editRelationshipOptions).toHaveCount(8);
  await expect(relationshipForm.getByRole("option", { name: "Other", exact: true })).toHaveCount(0);
  await relationshipForm.getByRole("option", { name: "Mother", exact: true }).click();
  const saveRelationship = relationshipForm.getByRole("button", { name: "Save relationship" });
  failGuardianRelationshipUpdateConflict = true;
  await saveRelationship.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Relationship submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(relationshipForm.getByText("Another active guardian already uses this relationship. Choose another available relationship.", { exact: true })).toBeVisible();
  expect(guardianLink.relationship_type).toBe(" father ");
  failGuardianRelationshipUpdateConflict = false;
  await relationshipSelect.click();
  await relationshipForm.getByRole("option", { name: "Grandfather", exact: true }).click();
  const guardianReadsBeforeFocus = guardianHistoryRequests.length;
  await invalidateProfileQueries(page, "guardians");
  await expect.poll(() => guardianHistoryRequests.length).toBeGreaterThan(guardianReadsBeforeFocus);
  await expect(relationshipSelect).toContainText("Grandfather");
  const editPrimaryRole = relationshipForm.getByRole("checkbox", { name: /Primary contact/ });
  const editEmergencyRole = relationshipForm.getByRole("checkbox", { name: /Emergency contact/ });
  await editPrimaryRole.check();
  await expect(editPrimaryRole).toBeChecked();
  await editPrimaryRole.uncheck();
  await expect(editPrimaryRole).not.toBeChecked();
  await editEmergencyRole.check();
  await expect(editEmergencyRole).toBeChecked();
  await editEmergencyRole.uncheck();
  await expect(editEmergencyRole).not.toBeChecked();
  await saveRelationship.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Relationship submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => guardianLink.relationship_type).toBe("Grandfather");
  expect(guardianLinkUpdateRequests).toHaveLength(2);
  expect(guardianLinkUpdateRequests[0]).toMatchObject({ relationship_type: "Mother" });
  expect(guardianLinkUpdateRequests[1]).toMatchObject({
    relationship_type: "Grandfather", is_primary_contact: false, is_emergency_contact: false,
  });
  await expect(page.locator(".student-family-card > div > span").filter({ hasText: /^Grandfather$/ })).toBeVisible();
  await expect(page.getByText(/Existing duplicate active relationships found/)).toHaveCount(0);
  await guardianCard.getByRole("button", { name: "Edit shared guardian details" }).click();
  await expect(page.getByRole("heading", { name: "Edit shared guardian profile" })).toBeVisible();
  await expect(page.getByText("These contact and profile details are shared across all students linked to this guardian.")).toBeVisible();
  const guardianNameInput = page.getByPlaceholder("Guardian full name");
  await expect(guardianNameInput).toHaveValue("Ramesh Sharma");
  await guardianNameInput.fill("अ".repeat(201));
  const saveGuardianDetails = page.getByRole("button", { name: "Save guardian details" });
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Guardian name must be 200 characters or fewer.", { exact: true })).toBeVisible();
  expect(guardianLink.guardian_name).toBe("Ramesh Sharma");
  await guardianNameInput.fill("Ramesh Sharma Updated");
  await page.getByPlaceholder("Mobile number").fill("9999912345");
  await page.getByPlaceholder("guardian@example.com").fill("not-an-email");
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
  expect(guardianLink.mobile_number).toBe("9876543210");
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toHaveCount(0);
  for (const invalidEmail of ["Guardian <guardian@example.test>", "guardian@example.test,other@example.test"]) {
    await page.getByPlaceholder("guardian@example.com").fill(invalidEmail);
    await saveGuardianDetails.evaluate(button => {
      const form = (button as HTMLButtonElement).form;
      if (!form) throw new Error("Guardian details submit button is not associated with its form.");
      form.requestSubmit(button as HTMLButtonElement);
    });
    await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
    expect(guardianLink.mobile_number).toBe("9876543210");
    expect(guardianLink.email_address).toBe("ramesh@example.test");
  }
  await page.getByPlaceholder("guardian@example.com").fill("server-invalid@example.test");
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toBeVisible();
  await page.getByRole("button", { name: "Update all linked records" }).click();
  await expect(page.getByText("Enter a valid email address using one address only.", { exact: true })).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toHaveCount(0);
  expect(guardianProfileUpdateRequests).toHaveLength(0);
  expect(guardianLink.email_address).toBe("ramesh@example.test");

  await page.getByPlaceholder("guardian@example.com").fill("ramesh.updated@example.test");
  await page.getByPlaceholder("Mobile number").fill("अ".repeat(31));
  await page.getByPlaceholder("Optional alternate number").fill("अ".repeat(31));
  await page.getByPlaceholder("e.g. Teacher, Business owner").fill("अ".repeat(121));
  await page.getByPlaceholder("Highest education (optional)").fill("अ".repeat(121));
  await page.getByPlaceholder("Organization name (optional)").fill("अ".repeat(161));
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Alternate mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Occupation must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Education must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Workplace must be 160 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toHaveCount(0);
  await page.getByPlaceholder("Mobile number").fill("9999912345");
  await page.getByPlaceholder("Optional alternate number").fill("9999987654");
  await page.getByPlaceholder("e.g. Teacher, Business owner").fill("Engineer");
  await page.getByPlaceholder("Highest education (optional)").fill("Postgraduate");
  await page.getByPlaceholder("Organization name (optional)").fill("City Hospital");
  await saveGuardianDetails.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Guardian details submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toBeVisible();
  await page.getByRole("button", { name: "Update all linked records" }).click();
  await expect.poll(() => guardianLink.mobile_number).toBe("9999912345");
  await expect.poll(() => guardianLink.guardian_name).toBe("Ramesh Sharma Updated");
  expect(guardianLink).toMatchObject({
    alternate_mobile_number: "9999987654", occupation: "Engineer",
    education: "Postgraduate", workplace_name: "City Hospital",
    relationship_type: "Grandfather", is_primary_contact: false, is_emergency_contact: false,
  });
  await expect(page.getByText("Guardian profile updated. Changes apply to every student linked to this guardian.")).toBeVisible();

  await guardianCard.getByRole("button", { name: "Revoke link" }).click();
  const revokeDialog = page.getByRole("dialog", { name: "Revoke guardian relationship" });
  await expect(revokeDialog).toBeVisible();
  await expect(revokeDialog).toContainText("Ramesh Sharma Updated");
  await expect(revokeDialog.getByPlaceholder("Explain why this link is being revoked")).toBeFocused();
  const confirmRevokeButton = revokeDialog.getByRole("button", { name: "Confirm revoke" });
  const revokeButtonBackground = await confirmRevokeButton.evaluate(element => getComputedStyle(element).backgroundColor);
  const revokeButtonText = await confirmRevokeButton.evaluate(element => getComputedStyle(element).color);
  expect(revokeButtonBackground).not.toBe("rgba(0, 0, 0, 0)");
  expect(revokeButtonBackground).not.toBe(await revokeDialog.evaluate(element => getComputedStyle(element).backgroundColor));
  expect(revokeButtonText).not.toBe(revokeButtonBackground);
  await revokeDialog.getByRole("button", { name: "Cancel" }).click();
  await expect(revokeDialog).toBeHidden();
  expect(guardianLink.revoked_at).toBeNull();

  await guardianCard.getByRole("button", { name: "Revoke link" }).click();
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
  await expect(page.getByText("Ramesh Sharma Updated")).toHaveCount(0);
  const historyToggle = page.getByLabel("Show revoked history");
  await historyToggle.check();
  await expect(historyToggle).toBeChecked();
  await expect.poll(() => historyToggle.evaluate(element => getComputedStyle(element, "::before").content)).not.toBe("none");
  await expect.poll(() => guardianHistoryRequests[guardianHistoryRequests.length - 1]).toBe("true");
  expect(guardianHistoryRequests[0]).toBeNull();
  await expect(page.getByText("Ramesh Sharma Updated")).toBeVisible();
  await expect(page.getByText(/Revoked/)).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toBeVisible();
  await page.getByLabel("Show revoked history").check();
  await expect(page.getByText("Ramesh Sharma")).toBeVisible();
  await expect(page.getByRole("article").getByText("Grandfather", { exact: true })).toBeVisible();
  await expect(page.getByText("Ramesh Sharma Updated")).toBeVisible();
  await expect(page.getByText(/9999912345/)).toBeVisible();
  await expect(page.getByText(/9999987654/)).toBeVisible();
  await expect(page.getByText(/ramesh\.updated@example\.test/)).toBeVisible();
  await expect(page.getByText(/Engineer/)).toBeVisible();
  await expect(page.getByText(/Postgraduate/)).toBeVisible();
  await expect(page.getByText(/City Hospital/)).toBeVisible();
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
  const addressReadsBeforeFocus = observedStudentDetailGets.filter(item => item === "addresses").length;
  await invalidateProfileQueries(page, "students");
  await expect.poll(() => observedStudentDetailGets.filter(item => item === "addresses").length).toBeGreaterThan(addressReadsBeforeFocus);
  await expect(page.getByPlaceholder("House number, street or building")).toHaveValue("House 12, Main Road");
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
  await page.getByPlaceholder("student@example.com").fill("server-invalid@example.test");
  await contactSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Contact save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.locator("#student-contact-email-error")).toHaveText("Enter a valid email address using one address only.");
  expect(contactDetails.primary_phone).toBeNull();
  await page.getByPlaceholder("student@example.com").fill("student@example.test");
  const contactReadsBeforeFocus = observedStudentDetailGets.filter(item => item === "contact-details").length;
  await invalidateProfileQueries(page, "students");
  await expect.poll(() => observedStudentDetailGets.filter(item => item === "contact-details").length).toBeGreaterThan(contactReadsBeforeFocus);
  await expect(page.getByPlaceholder("Primary contact number")).toHaveValue("9876501234");
  await expect(page.getByPlaceholder("student@example.com")).toHaveValue("student@example.test");
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
  const medicalDraft = "Draft must survive a background medical refetch";
  await page.getByPlaceholder("Record known allergies, if any").fill(medicalDraft);
  const medicalReadsBeforeFocus = medicalDetailReads;
  await page.bringToFront();
  await invalidateProfileQueries(page, "students");
  await expect.poll(() => medicalDetailReads).toBeGreaterThan(medicalReadsBeforeFocus);
  await expect(page.getByPlaceholder("Record known allergies, if any")).toHaveValue(medicalDraft);
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
  await expect(page.locator(".app-toast").filter({ hasText: "Medical details saved." })).toBeVisible({ timeout: 15_000 });
  await page.getByRole("button", { name: "Physical details" }).click();
  const physicalDraft = "Draft physical note survives background refetch";
  await page.getByPlaceholder("Optional vision-related notes").fill(physicalDraft);
  const physicalReadsBeforeFocus = physicalDetailReads;
  await page.bringToFront();
  await invalidateProfileQueries(page, "students");
  await expect.poll(() => physicalDetailReads).toBeGreaterThan(physicalReadsBeforeFocus);
  await expect(page.getByPlaceholder("Optional vision-related notes")).toHaveValue(physicalDraft);
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
  failMeasurementDateValidation = true;
  await physicalSaveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(page.getByText("Measurement date is required and cannot be in the future.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Measurement date", exact: true })).toHaveAttribute("aria-invalid", "true");
  failMeasurementDateValidation = false;
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
    if (path === "/school/enrollment-directory" && method === "GET") {
      await route.fulfill({ json: { enrollments: [], pagination: { limit: 20, offset: 0, total: 0 } } });
      return;
    }
    if (path === "/school/enrollments" && method === "GET") {
      await route.fulfill({ json: { enrollments: [] } });
      return;
    }
    if (path === "/school/enrollments" && method === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      requests.push(input);
      if (requests.length === 1) {
        await route.fulfill({
          status: 400,
          json: {
            status: "error",
            message: "Admission date must not be later than enrollment start date.",
            field_errors: { admission_date: "Admission date cannot be later than the enrollment start date." },
          },
        });
        return;
      }
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
  await expect(page.getByText("Admission date cannot be later than the enrollment start date.", { exact: true })).toBeVisible();
  await expect(page.getByText("Unable to create the enrollment.")).toHaveCount(0);

  await saveButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Enrollment submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => requests.length).toBe(2);
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
  await expect(page.getByRole("button", { name: "Edit profile for View Only Student" })).toHaveCount(1);
  await expect(page.getByRole("button", { name: "Archive View Only Student" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /New admission/ })).toHaveCount(0);
  await page.getByRole("button", { name: "Edit profile for View Only Student" }).click();
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
  let rejectPromotionOnce = true;
  let promotionStartedResolve!: () => void;
  let releasePromotionResolve!: () => void;
  const promotionStarted = new Promise<void>(resolve => { promotionStartedResolve = resolve; });
  const promotionGate = new Promise<void>(resolve => { releasePromotionResolve = resolve; });
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
        { id: 6, name: "2025–2026", starts_on: "2025-04-01", ends_on: "2026-03-31", status: "active", is_current: false },
        { id: 7, name: "2026–2027", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true },
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
    if (path === "/school/enrollment-directory" && request.method() === "GET") {
      await route.fulfill({ json: { enrollments, pagination: { limit: 20, offset: 0, total: enrollments.length } } });
      return;
    }
    if (path === "/school/enrollments/100/promote" && request.method() === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      promotionRequests.push(input);
      if (rejectPromotionOnce) {
        rejectPromotionOnce = false;
        await route.fulfill({
          status: 400,
          json: {
            status: "error",
            message: "Promotion date cannot be earlier than attendance already recorded for this enrollment",
            field_errors: { admission_date: "Choose a promotion date on or after the latest attendance recorded for this enrollment." },
          },
        });
        return;
      }
      promotionStartedResolve();
      await promotionGate;
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
  await expect(dialog.getByText("Choose a promotion date on or after the latest attendance recorded for this enrollment.")).toBeVisible();
  await expect.poll(() => promotionRequests.length).toBe(1);
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Admission and promotion date" }).click();
  await dialog.getByRole("button", { name: /November 17, 2026/ }).click();
  await expect(dialog.getByText("Choose a promotion date on or after the latest attendance recorded for this enrollment.")).toHaveCount(0);
  await promoteButton.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Promotion submit button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect.poll(() => promotionRequests.length).toBe(2);
  await promotionStarted;
  await expect(dialog.getByRole("button", { name: "Destination academic year", exact: true })).toBeDisabled();
  await expect(dialog.getByRole("button", { name: "Destination class and section", exact: true })).toBeDisabled();
  await expect(dialog.getByPlaceholder("e.g. ADM-2027-001")).toBeDisabled();
  await expect(dialog.getByPlaceholder("Leave blank for automatic assignment")).toBeDisabled();
  await expect(dialog.getByRole("button", { name: "Admission and promotion date" })).toBeDisabled();
  releasePromotionResolve();
  expect(promotionRequests[0]).toMatchObject({
    academic_year_id: 7, section_id: 72, admission_number: "ADM-2026-091", roll_number: null,
  });
  expect(promotionRequests[0].admission_date).toBeTruthy();
  expect(promotionRequests[1].admission_date).toBe("2026-11-17");
  expect(promotionRequests[1].enrolled_at).toBe(promotionRequests[1].admission_date);
  await expect(page.getByText("Student promoted successfully. Previous academic history has been preserved.")).toBeVisible();
  await expect(page.getByText("ADM-OLD-091")).toBeVisible();
  await expect(page.getByText("ADM-2026-091")).toBeVisible();
  await expect(page.getByText("promoted", { exact: true })).toBeVisible();
  await expect(dialog).toHaveCount(0);
  expect(browserErrors).toEqual([]);
});

test("student archive confirmation recovers when the archive committed but its response was lost", async ({ page }) => {
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
        // Simulate the server committing the archive while the response is lost.
        // Retrying must be safe and return the already-archived student.
        Object.assign(student, { status: "archived", updated_at: "2026-11-14T12:00:00.000Z" });
        await route.abort();
        return;
      }
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
  await expect(dialog.getByRole("alert")).toBeVisible();
  await expect(dialog).toBeVisible();
  await expect(page.getByText("Archive Retry Student", { exact: true })).toBeVisible();
  expect(archiveAttempts).toBe(1);

  await dialog.getByRole("button", { name: "Archive student", exact: true }).click();
  await expect.poll(() => archiveAttempts).toBe(2);
  await expect(page.getByText("Archive Retry Student has been archived.")).toBeVisible();
  await expect(page.getByText("archived", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Edit profile for Archive Retry Student" })).toBeVisible();
  expect(archiveRequests).toHaveLength(2);

  await page.getByRole("button", { name: "Edit profile for Archive Retry Student" }).click();
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
  test.setTimeout(60_000);
  const browserErrors: string[] = [];
  const guardianCreates: Record<string, unknown>[] = [];
  const guardianUpdates: Record<string, unknown>[] = [];
  const guardianLinks: Record<string, unknown>[] = [];
  let failGuardianLinkConflict = false;
  let failGuardianRelationshipConflict = false;
  let rejectGuardianEmailOnce = true;
  let rejectGuardianMobileDuplicateOnce = true;
  let rejectGuardianEmailDuplicateOnce = true;
  let rejectSharedGuardianMobileDuplicateOnce = true;
  const guardian = {
    id: 45, school_id: 10, full_name: "Nisha Verma", mobile_number: "9876501234",
    email_address: "nisha.verma@example.test", alternate_mobile_number: null,
    occupation: "Teacher", education: null, workplace_name: null, status: "active",
  };
  const otherStudent: Student = {
    id: 94, school_id: 10, admission_number: "ADM-GUARD-094", full_name: "Other Guardian Student",
    status: "active", created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
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
      }, otherStudent], pagination: { limit: 20, offset: 0, total: 2 } } });
      return;
    }
    const studentRoute = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (studentRoute) {
      const studentID = Number(studentRoute[1]);
      const subpath = studentRoute[2] ?? "";
      if (subpath === "enrollments" && request.method() === "GET") {
        await route.fulfill({ json: { enrollments: [] } });
        return;
      }
      if (subpath === "guardians" && request.method() === "GET") {
        await route.fulfill({ json: { student_guardians: guardianLinks.filter(link => Number(link.student_id) === studentID) } });
        return;
      }
      if (subpath === "related-students" && request.method() === "GET") {
        const currentLinks = guardianLinks.filter(link => Number(link.student_id) === studentID && !link.revoked_at);
        const guardianIDs = new Set(currentLinks.map(link => Number(link.guardian_id)));
        const relatedStudentIDs = new Set(guardianLinks
          .filter(link => guardianIDs.has(Number(link.guardian_id)) && Number(link.student_id) !== studentID && !link.revoked_at)
          .map(link => Number(link.student_id)));
        const relatedStudents = [...relatedStudentIDs].map(id => ({
          student_id: id,
          full_name: id === 94 ? otherStudent.full_name : "Guardian Workflow Student",
          shared_guardian_names: [guardian.full_name],
        }));
        await route.fulfill({ json: { related_students: relatedStudents } });
        return;
      }
    }
    if (path === "/school/guardians" && request.method() === "GET") {
      const linkedToCurrentStudent = guardianLinks.some(link => Number(link.student_id) === 93 && !link.revoked_at);
      await route.fulfill({ json: { guardians: linkedToCurrentStudent ? [] : [guardian], total: linkedToCurrentStudent ? 0 : 1 } });
      return;
    }
    if (path === "/school/guardians" && request.method() === "POST") {
      const input = request.postDataJSON() as Record<string, unknown>;
      if (rejectGuardianMobileDuplicateOnce && input.mobile_number === "9876500000") {
        rejectGuardianMobileDuplicateOnce = false;
        await route.fulfill({ status: 409, json: {
          message: "This mobile number is already registered with another guardian. Use another contact number or leave this field blank.",
          field_errors: { mobile_number: "This mobile number is already used by another guardian. Use another contact number or leave this field blank." },
        } });
        return;
      }
      if (rejectGuardianEmailDuplicateOnce && input.email_address === "already-used@example.test") {
        rejectGuardianEmailDuplicateOnce = false;
        await route.fulfill({ status: 409, json: {
          message: "This email address is already registered with another guardian. Use another email address or leave this field blank.",
          field_errors: { email_address: "This email address is already used by another guardian. Use another email address or leave this field blank." },
        } });
        return;
      }
      if (rejectGuardianEmailOnce && input.email_address === "server-invalid@example.test") {
        rejectGuardianEmailOnce = false;
        await route.fulfill({
          status: 400,
          json: {
            message: "Enter a valid guardian email address",
            field_errors: { email_address: "Enter a valid email address using one address only." },
          },
        });
        return;
      }
      guardianCreates.push(input);
      await route.fulfill({ json: { guardian } });
      return;
    }
    if (path === "/school/guardians/45" && request.method() === "PATCH") {
      const input = request.postDataJSON() as Record<string, unknown>;
      if (rejectSharedGuardianMobileDuplicateOnce && input.mobile_number === "9876500000") {
        rejectSharedGuardianMobileDuplicateOnce = false;
        await route.fulfill({ status: 409, json: {
          message: "This mobile number is already registered with another guardian. Use another contact number or leave this field blank.",
          field_errors: { mobile_number: "This mobile number is already used by another guardian. Use another contact number or leave this field blank." },
        } });
        return;
      }
      guardianUpdates.push(input);
      Object.assign(guardian, input);
      for (const link of guardianLinks) {
        Object.assign(link, {
          guardian_name: guardian.full_name,
          mobile_number: guardian.mobile_number,
          email_address: guardian.email_address,
          alternate_mobile_number: guardian.alternate_mobile_number,
          occupation: guardian.occupation,
          education: guardian.education,
          workplace_name: guardian.workplace_name,
          guardian_status: guardian.status,
        });
      }
      await route.fulfill({ json: { guardian } });
      return;
    }
    if (path === "/school/student-guardian-links/953" && request.method() === "PATCH") {
      const input = request.postDataJSON() as Record<string, unknown>;
      const link = guardianLinks.find(item => Number(item.id) === 953);
      if (!link) throw new Error("Student 93 guardian relationship fixture is missing.");
      Object.assign(link, input);
      await route.fulfill({ json: { student_guardian: link } });
      return;
    }
    if (path === "/school/students/93/guardians" && request.method() === "POST") {
      if (failGuardianRelationshipConflict) {
        await route.fulfill({ status: 409, json: { message: "This relationship type is already linked to this student" } });
        return;
      }
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
      // Simulate the same guardian already being linked to another student.
      // Shared contact fields must propagate, while this student's relationship flags stay independent.
      guardianLinks.push({
        id: 954, student_id: 94, guardian_id: 45, guardian_name: guardian.full_name,
        guardian_status: "active", mobile_number: guardian.mobile_number, email_address: guardian.email_address,
        alternate_mobile_number: guardian.alternate_mobile_number, occupation: guardian.occupation,
        education: guardian.education, workplace_name: guardian.workplace_name,
        relationship_type: "Father", is_primary_contact: true, is_emergency_contact: false, revoked_at: null,
      });
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
  await page.getByRole("button", { name: "Edit profile for Guardian Workflow Student" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  const familyDialog = page.getByRole("dialog", { name: "Parents & Siblings" });
  await expect(familyDialog).toBeVisible();
  await expect(page.getByText(/Staff-managed guardian records only/)).toBeVisible();
  const guardianPanel = familyDialog.locator(".student-family-section--guardians");
  const relatedPanel = familyDialog.locator(".student-family-section--related");
  const linkPanel = familyDialog.locator(".student-family-link-form--accent");
  await expect(guardianPanel).toBeVisible();
  await expect(relatedPanel).toBeVisible();
  await expect(linkPanel).toBeVisible();
  const sectionGap = await relatedPanel.evaluate(element => Number.parseFloat(getComputedStyle(element).marginTop));
  expect(sectionGap).toBeGreaterThanOrEqual(16);
  const lightPanelStyle = await guardianPanel.evaluate(element => getComputedStyle(element).backgroundColor);
  await page.locator("html").evaluate(element => element.setAttribute("data-theme", "dark"));
  const darkPanelStyle = await guardianPanel.evaluate(element => getComputedStyle(element).backgroundColor);
  expect(darkPanelStyle).not.toBe("rgba(0, 0, 0, 0)");
  expect(darkPanelStyle).not.toBe(lightPanelStyle);
  await page.locator("html").evaluate(element => element.setAttribute("data-theme", "light"));
  await expect(guardianPanel).toBeVisible();

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
  for (const invalidEmail of ["Guardian <guardian@example.test>", "guardian@example.test,other@example.test"]) {
    await page.getByPlaceholder("guardian@example.com").fill(invalidEmail);
    await page.getByRole("button", { name: "Save guardian contact" }).click();
    await expect(page.getByText("Enter a valid email address.", { exact: true })).toBeVisible();
    expect(guardianCreates).toHaveLength(0);
  }

  await page.getByPlaceholder("guardian@example.com").fill("server-invalid@example.test");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Enter a valid email address using one address only.", { exact: true })).toBeVisible();
  expect(guardianCreates).toHaveLength(0);

  await page.getByPlaceholder("guardian@example.com").fill("");
  await page.getByPlaceholder("Mobile number").fill("9876500000");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("This mobile number is already used by another guardian. Use another contact number or leave this field blank.", { exact: true })).toBeVisible();
  expect(guardianCreates).toHaveLength(0);

  await page.getByPlaceholder("Mobile number").fill("");
  await page.getByPlaceholder("guardian@example.com").fill("already-used@example.test");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("This email address is already used by another guardian. Use another email address or leave this field blank.", { exact: true })).toBeVisible();
  expect(guardianCreates).toHaveLength(0);

  await page.getByPlaceholder("guardian@example.com").fill("nisha.verma@example.test");
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Guardian contact created. Select the relationship details and link it to this student.")).toBeVisible();
  expect(guardianCreates).toHaveLength(1);
  expect(guardianCreates[0]).toMatchObject({
    full_name: "Nisha Verma", mobile_number: null,
    email_address: "nisha.verma@example.test", status: "active",
  });

  const linkRelationshipSelect = familyDialog.locator("form.student-family-link-form").getByRole("button", { name: "Relationship to student", exact: true });
  await linkRelationshipSelect.click();
  const linkRelationshipOptions = familyDialog.getByRole("listbox", { name: "Relationship to student" }).getByRole("option");
  await expect(linkRelationshipOptions).toHaveCount(8);
  await expect(familyDialog.getByRole("option", { name: "Other", exact: true })).toHaveCount(0);
  await familyDialog.getByRole("option", { name: "Mother", exact: true }).click();
  await expect(page.getByText("First person the school should contact for routine communication.", { exact: false })).toBeVisible();
  await expect(page.getByText("Person the school may contact if the student is ill, injured, or there is an urgent situation.", { exact: false })).toBeVisible();
  const linkPrimaryRole = familyDialog.getByRole("checkbox", { name: /Primary contact/ });
  const linkEmergencyRole = familyDialog.getByRole("checkbox", { name: /Emergency contact/ });
  await linkPrimaryRole.check();
  await expect(linkPrimaryRole).toBeChecked();
  await linkEmergencyRole.check();
  await expect(linkEmergencyRole).toBeChecked();
  failGuardianLinkConflict = true;
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect(page.getByText("This guardian is already linked to this student. Review the existing relationships before adding another link.", { exact: true })).toBeVisible();
  expect(guardianLinks).toHaveLength(0);
  failGuardianLinkConflict = false;
  failGuardianRelationshipConflict = true;
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect(page.getByText("This relationship is already linked to this student. Choose another available relationship.", { exact: true })).toBeVisible();
  expect(guardianLinks).toHaveLength(0);
  failGuardianRelationshipConflict = false;
  await page.getByRole("button", { name: "Link guardian", exact: true }).click();
  await expect.poll(() => guardianLinks.length).toBe(2);
  await linkRelationshipSelect.click();
  await expect(familyDialog.getByRole("option", { name: "Mother (already linked)", exact: true })).toBeDisabled();
  await page.keyboard.press("Escape");
  expect(guardianLinks.find(link => Number(link.student_id) === 93)).toMatchObject({
    guardian_id: 45, relationship_type: "Mother",
    is_primary_contact: true, is_emergency_contact: true,
  });
  expect(guardianLinks.find(link => Number(link.student_id) === 94)).toMatchObject({
    guardian_id: 45, relationship_type: "Father",
    is_primary_contact: true, is_emergency_contact: false,
  });
  await expect(page.getByText("Guardian linked to this student.")).toBeVisible();
  await expect(page.getByText("Nisha Verma", { exact: true })).toBeVisible();
  await expect(page.getByText("Mother · Primary contact · Emergency contact")).toBeVisible();

  await page.getByRole("button", { name: "Edit shared guardian details" }).click();
  await expect(page.getByPlaceholder("Mobile number")).toHaveValue("9876501234");
  const editGuardianName = page.getByPlaceholder("Guardian full name");
  const editGuardianMobile = page.getByPlaceholder("Mobile number");
  const editGuardianEmail = page.getByPlaceholder("guardian@example.com");
  const editGuardianAlternate = page.getByPlaceholder("Optional alternate number");
  const editGuardianOccupation = page.getByPlaceholder("e.g. Teacher, Business owner");
  const editGuardianEducation = page.getByPlaceholder("Highest education (optional)");
  const editGuardianWorkplace = page.getByPlaceholder("Organization name (optional)");
  await expect(editGuardianName).toHaveValue("Nisha Verma");
  await expect(editGuardianMobile).toHaveValue("9876501234");
  await expect(editGuardianEmail).toHaveValue("nisha.verma@example.test");

  await editGuardianMobile.fill("अ".repeat(31));
  await editGuardianAlternate.fill("अ".repeat(31));
  await editGuardianOccupation.fill("अ".repeat(121));
  await editGuardianEducation.fill("अ".repeat(121));
  await editGuardianWorkplace.fill("अ".repeat(161));
  await page.getByRole("button", { name: "Save guardian details" }).click();
  await expect(page.getByText("Mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Alternate mobile number must be 30 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Occupation must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Education must be 120 characters or fewer.", { exact: true })).toBeVisible();
  await expect(page.getByText("Workplace must be 160 characters or fewer.", { exact: true })).toBeVisible();
  expect(guardianUpdates).toHaveLength(0);

  // Reset every field that was intentionally made invalid above so this
  // submission reaches the server-side duplicate-contact validation.
  await editGuardianAlternate.fill("");
  await editGuardianOccupation.fill("Teacher");
  await editGuardianEducation.fill("");
  await editGuardianWorkplace.fill("");
  await editGuardianMobile.fill("9876500000");
  await page.getByRole("button", { name: "Save guardian details" }).click();
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toBeVisible();
  await page.getByRole("button", { name: "Update all linked records" }).click();
  await expect(page.getByText("This mobile number is already used by another guardian. Use another contact number or leave this field blank.", { exact: true })).toBeVisible();
  expect(guardianUpdates).toHaveLength(0);

  await editGuardianName.fill("Nisha Verma Updated");
  await editGuardianMobile.fill("9876501235");
  await editGuardianAlternate.fill("9876501236");
  await editGuardianOccupation.fill("Principal");
  await editGuardianEducation.fill("M.Ed.");
  await editGuardianWorkplace.fill("Sunrise Public School");
  await page.getByRole("button", { name: "Save guardian details" }).click();
  await expect(page.getByRole("dialog", { name: "Update shared guardian profile?" })).toBeVisible();
  await page.getByRole("button", { name: "Update all linked records" }).click();
  await expect(page.getByText("Guardian profile updated. Changes apply to every student linked to this guardian.")).toBeVisible();
  await expect.poll(() => guardianUpdates.length).toBe(1);
  expect(guardianUpdates[0]).toMatchObject({
    full_name: "Nisha Verma Updated",
    mobile_number: "9876501235",
    email_address: "nisha.verma@example.test",
    alternate_mobile_number: "9876501236",
    occupation: "Principal",
    education: "M.Ed.",
    workplace_name: "Sunrise Public School",
    status: "active",
  });
  await expect(page.getByText("Nisha Verma Updated", { exact: true })).toBeVisible();
  await expect(page.getByText("9876501235 · Alt: 9876501236 · nisha.verma@example.test")).toBeVisible();
  await expect(page.getByText("Principal · M.Ed. · Sunrise Public School")).toBeVisible();

  await page.getByRole("button", { name: "Edit relationship" }).click();
  const editRelationshipForm = page.locator("form.student-family-create-form").filter({ hasText: "Edit relationship for this student" });
  await editRelationshipForm.getByRole("button", { name: "Relationship to student", exact: true }).click();
  await editRelationshipForm.getByRole("option", { name: "Aunt", exact: true }).click();
  await page.getByRole("checkbox", { name: /Primary contact/ }).first().uncheck();
  await page.getByRole("button", { name: "Save relationship" }).click();
  await expect(page.getByText("Aunt · Emergency contact")).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  await expect(page.getByRole("dialog", { name: "Parents & Siblings" })).toBeVisible();
  await expect(page.getByText("Nisha Verma Updated", { exact: true })).toBeVisible();
  await expect(page.getByText("9876501235 · Alt: 9876501236 · nisha.verma@example.test")).toBeVisible();
  await expect(page.getByText("Principal · M.Ed. · Sunrise Public School")).toBeVisible();
  await expect(page.getByText("Aunt · Emergency contact")).toBeVisible();

  // A full browser reload must restore the persisted guardian and relationship
  // from the API, not merely retain the previous React component's local state.
  await page.reload();
  await expect(page.getByLabel("Username")).toBeVisible();
  await page.getByLabel("Username").fill("e2e-guardian-manager");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "Edit profile for Guardian Workflow Student" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  await expect(page.getByRole("dialog", { name: "Parents & Siblings" })).toBeVisible();
  await expect(page.getByText("Nisha Verma Updated", { exact: true })).toBeVisible();
  await expect(page.getByText("9876501235 · Alt: 9876501236 · nisha.verma@example.test")).toBeVisible();
  await expect(page.getByText("Principal · M.Ed. · Sunrise Public School")).toBeVisible();
  await expect(page.getByText("Aunt · Emergency contact")).toBeVisible();
  await expect(familyDialog.getByText("Other Guardian Student", { exact: true })).toBeVisible();

  // Legacy duplicate active links remain visible and are flagged for explicit
  // review; opening history must not silently revoke or delete either record.
  guardianLinks.push({
    id: 955, student_id: 93, guardian_id: 46, guardian_name: "Amit Verma",
    guardian_status: "active", mobile_number: "9876500046", email_address: null,
    alternate_mobile_number: null, occupation: null, education: null, workplace_name: null,
    relationship_type: " aunt ", is_primary_contact: false, is_emergency_contact: false,
    revoked_at: null,
  });
  await familyDialog.getByLabel("Show revoked history").check();
  await expect(familyDialog.getByRole("status").filter({ hasText: "Existing duplicate active relationships found: Aunt" })).toBeVisible();
  await expect(familyDialog.getByText("Aunt", { exact: true })).toBeVisible();
  await expect(familyDialog.getByText("Amit Verma", { exact: true })).toBeVisible();
  expect(guardianLinks.filter(link => Number(link.student_id) === 93 && !link.revoked_at)).toHaveLength(2);

  // The same guardian's updated contact profile is shared with the second
  // student, but that student's own relationship type and flags remain intact.
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Edit profile for Other Guardian Student" }).click();
  await page.getByRole("button", { name: /Parents & siblings/i }).click();
  await expect(page.getByRole("dialog", { name: "Parents & Siblings" })).toBeVisible();
  await expect(page.getByText("Nisha Verma Updated", { exact: true })).toBeVisible();
  await expect(page.getByText("9876501235 · Alt: 9876501236 · nisha.verma@example.test")).toBeVisible();
  await expect(page.getByText("Principal · M.Ed. · Sunrise Public School")).toBeVisible();
  await expect(page.getByText("Father · Primary contact", { exact: true })).toBeVisible();
  await expect(page.getByText("Other students sharing a guardian")).toBeVisible();
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
    if (path === "/school/enrollment-directory" && request.method() === "GET") {
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
  await expect(page.getByRole("button", { name: "Edit profile for Inactive Family Student" })).toBeVisible();
  await page.getByRole("button", { name: "Edit profile for Inactive Family Student" }).click();
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


test("switching student profiles clears unsaved basic-details drafts and validation errors", async ({ page }) => {
  const students: Student[] = [
    { id: 11, school_id: 10, admission_number: "ADM-A", full_name: "Student A", status: "active", created_at: "2026-01-01T00:00:00.000Z", updated_at: "2026-01-01T00:00:00.000Z" },
    { id: 12, school_id: 10, admission_number: "ADM-B", full_name: "Student B", status: "active", created_at: "2026-01-02T00:00:00.000Z", updated_at: "2026-01-02T00:00:00.000Z" },
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
      await route.fulfill({ json: { access_token: "e2e-access-token", token_type: "Bearer", principal: { principal_id: 101, principal_type: "staff", status: "active", school_id: 10 } } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 101, principal_type: "staff", school_id: 10, username: "e2e-school-admin",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "academic_years.read", "subjects.read"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students, pagination: { limit: 20, offset: 0, total: students.length } } });
      return;
    }
    const match = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (match) {
      const studentID = Number(match[1]);
      const subpath = match[2] ?? "";

      if (subpath === "enrollments" && method === "GET") {
        await route.fulfill({ json: { enrollments: [] } });
        return;
      }
      if (subpath === "basic-details" && method === "GET") {
        await route.fulfill({ json: { basic_details: {
          id: studentID, school_id: 10, student_id: studentID, date_of_birth: null,
          place_of_birth: studentID === 12 ? "Ambala, Haryana" : null, gender: null,
          nationality: studentID === 12 ? "Indian" : null, religion: null,
          social_category: null, ews_status: null, school_registration_number: null,
        } } });
        return;
      }
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-school-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();

  await page.getByRole("button", { name: "Edit profile for Student A" }).click();
  await page.getByRole("button", { name: "Basic details" }).click();
  const placeOfBirth = page.getByPlaceholder("City, district or place");
  await expect(placeOfBirth).toBeVisible();
  await placeOfBirth.fill("Unsaved draft belonging only to Student A");
  await page.getByPlaceholder("e.g. Indian").fill("अ".repeat(81));
  await page.getByRole("button", { name: "Save basic details" }).click();
  await expect(page.getByText("Nationality must be 80 characters or fewer.", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.getByRole("dialog", { name: /Student profile/ })).toBeVisible();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Edit profile for Student B" }).click();
  await page.getByRole("button", { name: "Basic details" }).click();

  await expect(page.getByPlaceholder("City, district or place")).toHaveValue("Ambala, Haryana");
  await expect(page.getByPlaceholder("e.g. Indian")).toHaveValue("Indian");
  await expect(page.getByText("Nationality must be 80 characters or fewer.", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Unsaved draft belonging only to Student A")).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: "Basic Details" })).toContainText("Student B");
});


test("switching student Parent Details clears unsaved guardian drafts and field errors", async ({ page }) => {
  const students: Student[] = [
    { id: 21, school_id: 10, admission_number: "ADM-FAM-A", full_name: "Family Student A", status: "active", created_at: "2026-01-01T00:00:00.000Z", updated_at: "2026-01-01T00:00:00.000Z" },
    { id: 22, school_id: 10, admission_number: "ADM-FAM-B", full_name: "Family Student B", status: "active", created_at: "2026-01-02T00:00:00.000Z", updated_at: "2026-01-02T00:00:00.000Z" },
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
      await route.fulfill({ json: { access_token: "e2e-access-token", token_type: "Bearer", principal: { principal_id: 101, principal_type: "staff", status: "active", school_id: 10 } } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 101, principal_type: "staff", school_id: 10, username: "e2e-school-admin",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "guardians.read", "guardians.manage"],
      } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students, pagination: { limit: 20, offset: 0, total: students.length } } });
      return;
    }
    if (path === "/school/guardians" && method === "GET") {
      await route.fulfill({ json: { guardians: [], total: 0 } });
      return;
    }
    const match = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (match) {
      const studentID = Number(match[1]);
      const subpath = match[2] ?? "";
      if (subpath === "enrollments" && method === "GET") {
        await route.fulfill({ json: { enrollments: [] } });
        return;
      }
      if (subpath === "guardians" && method === "GET") {
        await route.fulfill({ json: { student_guardians: [] } });
        return;
      }
      if (subpath === "related-students" && method === "GET") {
        await route.fulfill({ json: { related_students: [] } });
        return;
      }
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-school-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();

  await page.getByRole("button", { name: "Edit profile for Family Student A" }).click();
  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await page.getByRole("button", { name: "Create a new guardian contact" }).click();
  await page.getByPlaceholder("Guardian full name").fill("अ".repeat(201));
  await page.getByRole("button", { name: "Save guardian contact" }).click();
  await expect(page.getByText("Guardian name must be 200 characters or fewer.", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Close", exact: true }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Edit profile for Family Student B" }).click();
  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  await page.getByRole("button", { name: "Create a new guardian contact" }).click();

  await expect(page.getByPlaceholder("Guardian full name")).toHaveValue("");
  await expect(page.locator("form.student-family-link-form").getByRole("button", { name: "Relationship to student", exact: true })).toContainText("Select relationship");
  await expect(page.getByText("Guardian name must be 200 characters or fewer.", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: /Parents & Siblings/i })).toContainText("Family Student B");
});

test("switching student profiles clears address, contact, medical and physical drafts and errors", async ({ page }) => {
  test.setTimeout(60_000);
  const browserErrors: string[] = [];
  page.on("pageerror", error => browserErrors.push(error.message));
  const writeRequests: string[] = [];
  let failMedicalReads = false;
  let failPhysicalReads = false;
  const students: Student[] = [
    { id: 31, school_id: 10, admission_number: "ADM-STATE-A", full_name: "State Student A", status: "active", created_at: "2026-01-01T00:00:00.000Z", updated_at: "2026-01-01T00:00:00.000Z" },
    { id: 32, school_id: 10, admission_number: "ADM-STATE-B", full_name: "State Student B", status: "active", created_at: "2026-01-02T00:00:00.000Z", updated_at: "2026-01-02T00:00:00.000Z" },
  ];
  const addressFor = (studentID: number) => ({
    id: studentID * 10, school_id: 10, student_id: studentID, address_type: "current",
    address_line_1: studentID === 32 ? "Student B saved address" : "Student A saved address",
    address_line_2: null, area_mohalla: null, landmark: null, postal_code: null,
    location_id: null, state_location_id: null, district_location_id: null,
    subdistrict_location_id: null, same_as_current: false,
  });
  const contactFor = (studentID: number) => ({
    id: studentID * 20, school_id: 10, student_id: studentID,
    primary_phone: studentID === 32 ? "9999900032" : null,
    alternate_phone: null, email_address: null,
  });
  const medicalFor = (studentID: number) => ({
    id: studentID * 30, school_id: 10, student_id: studentID, version_number: 0,
    blood_group: null, allergies: studentID === 32 ? "Student B saved allergy record" : null,
    known_conditions: null, ongoing_medications: null, emergency_medical_notes: null, history: [],
  });
  const physicalFor = (studentID: number) => ({
    id: studentID * 40, school_id: 10, student_id: studentID,
    height_cm: studentID === 32 ? 140 : null, weight_kg: null,
    measured_on: studentID === 32 ? "2026-11-10T00:00:00Z" : null,
    vision_notes: studentID === 32 ? "Student B saved vision record" : null,
    physical_notes: null,
    history: studentID === 32 ? [{
      id: studentID * 40 - 1, height_cm: 138, weight_kg: 30,
      measured_on: "2026-11-10T00:00:00Z", vision_notes: "Previous vision record",
      physical_notes: null, recorded_by_principal_id: 131, created_at: "2026-11-10T10:00:00Z",
    }] : [],
  });

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");
    const method = request.method();
    if (method === "PUT" && /^\/school\/students\/\d+\/(addresses\/|contact-details$|medical-details$|physical-details$)/.test(path)) {
      writeRequests.push(path);
    }

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-profile-state-token", token_type: "Bearer",
        principal: { principal_id: 131, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 131, principal_type: "staff", school_id: 10, username: "e2e-profile-state",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "academic_years.read", "subjects.read"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students, pagination: { limit: 20, offset: 0, total: students.length } } });
      return;
    }
    if (path === "/school/locations") {
      await route.fulfill({ json: { locations: [] } });
      return;
    }

    const match = path.match(/^\/school\/students\/(\d+)(?:\/(.*))?$/);
    if (match) {
      const studentID = Number(match[1]);
      const subpath = match[2] ?? "";
      if (subpath === "enrollments" && method === "GET") {
        await route.fulfill({ json: { enrollments: [] } });
        return;
      }
      if (subpath === "basic-details" && method === "GET") {
        await route.fulfill({ json: { basic_details: {
          id: studentID, school_id: 10, student_id: studentID, date_of_birth: null,
          place_of_birth: null, gender: null, nationality: null, religion: null,
          social_category: null, ews_status: null, school_registration_number: null,
        } } });
        return;
      }
      if (subpath === "addresses" && method === "GET") {
        await route.fulfill({ json: { addresses: [addressFor(studentID)] } });
        return;
      }
      if (subpath === "contact-details" && method === "GET") {
        await route.fulfill({ json: { contact_details: contactFor(studentID) } });
        return;
      }
      if (subpath === "medical-details" && method === "GET") {
        if (failMedicalReads) {
          await route.fulfill({ status: 503, json: { message: "Temporary medical read failure." } });
          return;
        }
        await route.fulfill({ json: { medical_details: medicalFor(studentID) } });
        return;
      }
      if (subpath === "physical-details" && method === "GET") {
        if (failPhysicalReads) {
          await route.fulfill({ status: 503, json: { message: "Temporary physical read failure." } });
          return;
        }
        await route.fulfill({ json: { physical_details: physicalFor(studentID) } });
        return;
      }
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-profile-state");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await expect(page.getByRole("button", { name: "Students" })).toBeVisible();
  await page.getByRole("button", { name: "Students" }).click();

  await page.getByRole("button", { name: "Edit profile for State Student A" }).click();
  await page.getByRole("button", { name: /Address & communication/ }).click();
  const addressDialog = page.getByRole("dialog", { name: "Address & Communication" });
  await expect(addressDialog).toBeVisible();
  const addressLine = page.getByPlaceholder("House number, street or building");
  await expect(addressLine).toHaveValue("Student A saved address");
  await addressLine.fill("अ".repeat(201));
  const saveAddress = addressDialog.getByRole("button", { name: "Save address" });
  await saveAddress.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Address save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(addressDialog.getByText("House / street must be 200 characters or fewer.", { exact: true })).toBeVisible();

  await addressDialog.getByRole("button", { name: "Contact details", exact: true }).click();
  const primaryPhone = page.getByPlaceholder("Primary contact number");
  await primaryPhone.fill("अ".repeat(31));
  const saveContact = addressDialog.getByRole("button", { name: "Save contact details" });
  await saveContact.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Contact save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(addressDialog.getByText("Phone number must be 30 characters or fewer.", { exact: true })).toBeVisible();

  await addressDialog.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Edit profile for State Student B" }).click();
  await page.getByRole("button", { name: /Address & communication/ }).click();
  await expect(page.getByRole("dialog", { name: "Address & Communication" })).toBeVisible();
  await expect(page.getByPlaceholder("House number, street or building")).toHaveValue("Student B saved address");
  await expect(page.getByText("House / street must be 200 characters or fewer.", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Contact details", exact: true }).click();
  await expect(page.getByPlaceholder("Primary contact number")).toHaveValue("9999900032");
  await expect(page.getByText("Phone number must be 30 characters or fewer.", { exact: true })).toHaveCount(0);
  await expect(page.getByPlaceholder("Primary contact number")).not.toHaveValue("अ".repeat(31));
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();

  await page.getByRole("button", { name: "Edit profile for State Student A" }).click();
  failMedicalReads = true;
  await page.getByRole("button", { name: /Medical & physical/ }).click();
  const healthDialog = page.getByRole("dialog", { name: "Medical & Physical Details" });
  await expect(healthDialog).toBeVisible();
  await expect(healthDialog.getByRole("alert")).toContainText("Temporary medical read failure.");
  failMedicalReads = false;
  await healthDialog.getByRole("button", { name: "Retry loading medical details" }).click();
  const allergies = page.getByPlaceholder("Record known allergies, if any");
  await expect(allergies).toBeVisible();
  await allergies.fill("अ".repeat(2001));
  const saveMedical = healthDialog.getByRole("button", { name: "Save medical details" });
  await saveMedical.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Medical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(healthDialog.getByText("Use no more than 2,000 characters.", { exact: true })).toBeVisible();

  failPhysicalReads = true;
  await healthDialog.getByRole("button", { name: "Physical details" }).click();
  await expect(healthDialog.getByRole("alert")).toContainText("Temporary physical read failure.");
  failPhysicalReads = false;
  await healthDialog.getByRole("button", { name: "Retry loading physical details" }).click();
  const height = page.getByPlaceholder("e.g. 132.5");
  await height.fill("500");
  const savePhysical = healthDialog.getByRole("button", { name: "Save physical details" });
  await savePhysical.evaluate(button => {
    const form = (button as HTMLButtonElement).form;
    if (!form) throw new Error("Physical save button is not associated with its form.");
    form.requestSubmit(button as HTMLButtonElement);
  });
  await expect(healthDialog.getByText("Height must be between 20 and 250 cm.", { exact: true })).toBeVisible();
  await expect(healthDialog.getByText("Measurement date is required.", { exact: true })).toBeVisible();

  await healthDialog.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await page.getByRole("button", { name: "Edit profile for State Student B" }).click();
  await page.getByRole("button", { name: /Medical & physical/ }).click();
  await expect(page.getByPlaceholder("Record known allergies, if any")).toHaveValue("Student B saved allergy record");
  await expect(page.getByText("Use no more than 2,000 characters.", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Physical details" }).click();
  await expect(page.getByPlaceholder("e.g. 132.5")).toHaveValue("140");
  await expect(page.getByPlaceholder("Optional vision-related notes")).toHaveValue("Student B saved vision record");
  await healthDialog.getByText("Previous physical measurements (1)", { exact: true }).click();
  await expect(healthDialog.getByText("Measured 10 Nov 2026", { exact: true })).toBeVisible();
  await expect(page.getByText("Height must be between 20 and 250 cm.", { exact: true })).toHaveCount(0);
  expect(writeRequests).toEqual([]);
  expect(browserErrors).toEqual([]);
});


test("archived student profile sections remain readable but cannot be changed", async ({ page }) => {
  const archivedStudent = {
    id: 41, school_id: 10, admission_number: "ADM-ARCHIVED-41", full_name: "Archived Student",
    status: "archived", created_at: "2026-01-01T00:00:00.000Z", updated_at: "2026-01-02T00:00:00.000Z",
  };
  const attemptedWrites: string[] = [];
  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.startsWith("/api") ? url.pathname.slice(4) : url.pathname;
    const method = request.method();
    if (!path.startsWith("/auth/") && ["POST", "PUT", "PATCH", "DELETE"].includes(method)) attemptedWrites.push(method + " " + path);

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && method === "POST") {
      await route.fulfill({ json: {
        access_token: "e2e-archived-student-token", token_type: "Bearer",
        principal: { principal_id: 141, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 141, principal_type: "staff", school_id: 10, username: "e2e-archived-student",
        permissions: ["students.read_sensitive", "students.read_basic", "students.update", "guardians.read", "guardians.manage"],
      } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students: [archivedStudent], pagination: { limit: 20, offset: 0, total: 1 } } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [] } });
      return;
    }
    if (path === "/school/guardians" && method === "GET") {
      await route.fulfill({ json: { guardians: [], total: 0 } });
      return;
    }
    if (path === "/school/locations") {
      await route.fulfill({ json: { locations: [] } });
      return;
    }
    if (path.startsWith("/school/students/41/")) {
      const subpath = path.slice("/school/students/41/".length);
      if (subpath === "enrollments" && method === "GET") {
        await route.fulfill({ json: { enrollments: [{
          id: 99, student_id: 41, academic_year_id: 6,
          class_name_snapshot: "Nursery", section_name_snapshot: "B",
          admission_number: "ADM-2025-041", admission_date: "2025-04-01T00:00:00Z",
          status: "promoted", enrolled_at: "2025-04-01T00:00:00Z",
          ended_at: "2026-03-31T00:00:00Z",
        }] } });
        return;
      }
      if (subpath === "basic-details" && method === "GET") {
        await route.fulfill({ json: { basic_details: {
          id: 41, school_id: 10, student_id: 41, date_of_birth: null, place_of_birth: null,
          gender: null, nationality: null, religion: null, social_category: null,
          ews_status: null, school_registration_number: null,
        } } });
        return;
      }
      if (subpath === "addresses" && method === "GET") {
        await route.fulfill({ json: { addresses: [] } });
        return;
      }
      if (subpath === "contact-details" && method === "GET") {
        await route.fulfill({ json: { contact_details: {
          id: 0, school_id: 10, student_id: 41, primary_phone: null, alternate_phone: null, email_address: null,
        } } });
        return;
      }
      if (subpath === "medical-details" && method === "GET") {
        await route.fulfill({ json: { medical_details: {
          id: 0, school_id: 10, student_id: 41, version_number: 0, blood_group: null,
          allergies: null, known_conditions: null, ongoing_medications: null,
          emergency_medical_notes: null, history: [],
        } } });
        return;
      }
      if (subpath === "physical-details" && method === "GET") {
        await route.fulfill({ json: { physical_details: {
          id: 0, school_id: 10, student_id: 41, height_cm: null, weight_kg: null,
          measured_on: null, vision_notes: null, physical_notes: null, history: [],
        } } });
        return;
      }
      if (subpath === "guardians" && method === "GET") {
        await route.fulfill({ json: { student_guardians: [] } });
        return;
      }
      if (subpath === "related-students" && method === "GET") {
        await route.fulfill({ json: { related_students: [] } });
        return;
      }
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-archived-student");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "Edit profile for Archived Student" }).click();
  const profileDialog = page.getByRole("dialog", { name: "Student profile" });
  await expect(profileDialog.getByText("archived", { exact: true })).toBeVisible();
  await expect(profileDialog.getByText("31 Mar 2026", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Basic details" }).click();
  const basicDialog = page.getByRole("dialog", { name: "Basic Details" });
  await expect(basicDialog).toBeVisible();
  await expect(basicDialog.getByRole("button", { name: "Save basic details" })).toHaveCount(0);
  await expect(basicDialog.getByText(/View-only access/)).toBeVisible();
  await basicDialog.getByRole("button", { name: "Close dialog" }).click();

  await page.getByRole("button", { name: /Address & communication/ }).click();
  const addressDialog = page.getByRole("dialog", { name: "Address & Communication" });
  await expect(addressDialog).toBeVisible();
  await expect(addressDialog.getByRole("button", { name: "Save address" })).toHaveCount(0);
  await expect(addressDialog.getByRole("button", { name: "Save contact details" })).toHaveCount(0);
  await expect(addressDialog.getByText(/View-only access/)).toBeVisible();
  await addressDialog.getByRole("button", { name: "Close dialog" }).click();

  await page.getByRole("button", { name: /Medical & physical/ }).click();
  const healthDialog = page.getByRole("dialog", { name: "Medical & Physical Details" });
  await expect(healthDialog).toBeVisible();
  await expect(healthDialog.getByRole("button", { name: "Save medical details" })).toHaveCount(0);
  await expect(healthDialog.getByText(/View-only access/)).toBeVisible();
  await healthDialog.getByRole("button", { name: "Physical details" }).click();
  await expect(healthDialog.getByRole("button", { name: "Save physical details" })).toHaveCount(0);
  await healthDialog.getByRole("button", { name: "Close dialog" }).click();

  await page.getByRole("button", { name: /Parents & Siblings/i }).click();
  const familyDialog = page.getByRole("dialog", { name: /Parents & Siblings/ });
  await expect(familyDialog).toBeVisible();
  await expect(familyDialog.getByRole("button", { name: "Create a new guardian contact" })).toHaveCount(0);
  await expect(familyDialog.getByText(/view-only|read-only/i)).toBeVisible();
  expect(attemptedWrites).toEqual([]);
});

test("student directory pagination resets when search, status or academic-year filters change", async ({ page }) => {
  const requestedDirectoryFilters: Array<{ offset: string; status: string; academicYearID: string }> = [];
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
        access_token: "e2e-directory-pagination-token", token_type: "Bearer",
        principal: { principal_id: 151, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 151, principal_type: "staff", school_id: 10, username: "e2e-directory-pagination",
        permissions: ["students.read_sensitive", "academic_years.read"],
      } });
      return;
    }
    if (path === "/school/academic-years" && method === "GET") {
      await route.fulfill({ json: { academic_years: [academicYear, previousAcademicYear] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      const offset = Number(url.searchParams.get("offset") ?? "0");
      const query = url.searchParams.get("q") ?? "";
      const status = url.searchParams.get("status") ?? "";
      const academicYearID = url.searchParams.get("academic_year_id") ?? "";
      requestedDirectoryFilters.push({ offset: String(offset), status, academicYearID });
      const filtered = Boolean(query || status || (academicYearID && academicYearID !== String(academicYear.id)));
      const rows = query
        ? [{ id: 99, school_id: 10, admission_number: "ADM-SEARCH-99", full_name: "Search Match Student", status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }]
        : status
          ? [{ id: 98, school_id: 10, admission_number: "ADM-INACTIVE-98", full_name: "Inactive Filter Student", status: "inactive", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }]
          : academicYearID && academicYearID !== String(academicYear.id)
            ? [{ id: 97, school_id: 10, admission_number: "ADM-PREVIOUS-97", full_name: "Previous Year Student", status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" }]
            : Array.from({ length: 20 }, (_, index) => {
                const id = offset + index + 1;
                return { id, school_id: 10, admission_number: `ADM-${id}`, full_name: `Directory Student ${id}`, status: "active", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" };
              });
      await route.fulfill({ json: { students: rows, pagination: { limit: 20, offset, total: filtered ? 1 : 45 } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-directory-pagination");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await expect(page.getByText("Directory Student 1", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–20 of 45")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Directory Student 21", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 21–40 of 45")).toBeVisible();
  await expect.poll(() => requestedDirectoryFilters.at(-1)?.offset).toBe("20");

  await page.getByRole("textbox", { name: "Search students" }).fill("Search Match");
  await expect(page.getByText("Search Match Student", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();
  await expect.poll(() => requestedDirectoryFilters.at(-1)?.offset).toBe("0");
  await expect(page.getByRole("button", { name: "Previous" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Next" })).toHaveCount(0);

  await page.getByRole("textbox", { name: "Search students" }).fill("");
  await expect(page.getByText("Showing 1–20 of 45")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Showing 21–40 of 45")).toBeVisible();

  const statusFilter = page.getByRole("button", { name: "Filter by student status" });
  await statusFilter.click();
  const statusOptions = page.getByRole("listbox", { name: "Filter by student status" });
  await expect(statusOptions).toBeVisible();
  await statusOptions.getByRole("option", { name: "Inactive" }).click();
  await expect(page.getByText("Inactive Filter Student", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();
  await expect.poll(() => requestedDirectoryFilters.at(-1)).toEqual({ offset: "0", status: "inactive", academicYearID: String(academicYear.id) });

  await statusFilter.click();
  await page.getByRole("listbox", { name: "Filter by student status" }).getByRole("option", { name: "All statuses" }).click();
  await expect(page.getByText("Showing 1–20 of 45")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Showing 21–40 of 45")).toBeVisible();

  const yearFilter = page.getByRole("button", { name: "Filter students by academic year" });
  await yearFilter.click();
  await page.getByRole("listbox", { name: "Filter students by academic year" }).getByRole("option", { name: "2025–2026" }).click();
  await expect(page.getByText("Previous Year Student", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();
  await expect.poll(() => requestedDirectoryFilters.at(-1)).toEqual({ offset: "0", status: "", academicYearID: String(previousAcademicYear.id) });
});

test("student directory returns to the last valid page when archiving empties the current page", async ({ page }) => {
  let directoryTotal = 41;
  const requestedOffsets: number[] = [];
  const student = {
    id: 41, school_id: 10, admission_number: "ADM-041",
    full_name: "Directory Student 41", status: "active",
    created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
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
      await route.fulfill({ json: {
        access_token: "e2e-directory-archive-token", token_type: "Bearer",
        principal: { principal_id: 152, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 152, principal_type: "staff", school_id: 10, username: "e2e-directory-archive",
        permissions: ["students.read_sensitive", "students.archive", "academic_years.read"],
      } });
      return;
    }
    if (path === "/school/academic-years" && method === "GET") {
      await route.fulfill({ json: { academic_years: [academicYear] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      const offset = Number(url.searchParams.get("offset") ?? "0");
      requestedOffsets.push(offset);
      const count = Math.max(0, Math.min(20, directoryTotal - offset));
      const rows = Array.from({ length: count }, (_, index) => {
        const id = offset + index + 1;
        return {
          id, school_id: 10, admission_number: `ADM-${String(id).padStart(3, "0")}`,
          full_name: `Directory Student ${id}`, status: "active",
          created_at: "2026-11-14T00:00:00.000Z", updated_at: "2026-11-14T00:00:00.000Z",
        };
      });
      await route.fulfill({ json: { students: rows, pagination: { limit: 20, offset, total: directoryTotal } } });
      return;
    }
    if (path === "/school/students/41/archive" && method === "POST") {
      directoryTotal = 40;
      Object.assign(student, { status: "archived", updated_at: "2026-11-17T12:00:00.000Z" });
      await route.fulfill({ json: { student } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-directory-archive");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await expect(page.getByText("Showing 1–20 of 41")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Showing 21–40 of 41")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Directory Student 41", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 41–41 of 41")).toBeVisible();
  await expect.poll(() => requestedOffsets.at(-1)).toBe(40);

  await page.getByRole("button", { name: "Archive Directory Student 41" }).click();
  const dialog = page.getByRole("dialog", { name: "Archive student?" });
  await dialog.getByRole("button", { name: "Archive student", exact: true }).click();
  await expect(page.getByText("Directory Student 41 has been archived.")).toBeVisible();
  await expect(page.getByText("Directory Student 21", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 21–40 of 40")).toBeVisible();
  await expect(page.getByRole("button", { name: "Next" })).toBeDisabled();
  await expect.poll(() => requestedOffsets.slice(-2)).toEqual([40, 20]);
});

test("student directory sends wildcard characters as literal search text", async ({ page }) => {
  const receivedQueries: string[] = [];
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
        access_token: "e2e-literal-search-token", token_type: "Bearer",
        principal: { principal_id: 161, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 161, principal_type: "staff", school_id: 10, username: "e2e-literal-search",
        permissions: ["students.read_sensitive", "academic_years.read"],
      } });
      return;
    }
    if (path === "/school/academic-years" && method === "GET") {
      await route.fulfill({ json: { academic_years: [academicYear] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      receivedQueries.push(url.searchParams.get("q") ?? "");
      await route.fulfill({ json: { students: [], pagination: { limit: 20, offset: 0, total: 0 } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-literal-search");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  const literalSearch = "Updated%_\\";
  await page.getByRole("textbox", { name: "Search students" }).fill(literalSearch);
  await expect.poll(() => receivedQueries.at(-1)).toBe(literalSearch);
});


test("enrollment directory pagination requests only the selected page", async ({ page }) => {
  const requestedPages: Array<{ offset: number; yearID: number }> = [];
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
        access_token: "e2e-enrollment-pagination-token", token_type: "Bearer",
        principal: { principal_id: 171, principal_type: "staff", status: "active", school_id: 10 },
      } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: {
        principal_id: 171, principal_type: "staff", school_id: 10, username: "e2e-enrollment-pagination",
        permissions: ["students.read_sensitive", "students.read_basic", "academic_years.read"],
      } });
      return;
    }
    if (path === "/school/academic-years") {
      await route.fulfill({ json: { academic_years: [academicYear, previousAcademicYear] } });
      return;
    }
    if (path === "/school/students" && method === "GET") {
      await route.fulfill({ json: { students: [], pagination: { limit: 20, offset: 0, total: 0 } } });
      return;
    }
    if (path === "/school/enrollment-directory" && method === "GET") {
      const offset = Number(url.searchParams.get("offset") ?? "0");
      const yearID = Number(url.searchParams.get("academic_year_id") ?? "0");
      requestedPages.push({ offset, yearID });
      const total = yearID === previousAcademicYear.id ? 1 : 45;
      const count = Math.max(0, Math.min(20, total - offset));
      const enrollments = Array.from({ length: count }, (_, index) => {
        const id = offset + index + 1;
        return {
          id: 5000 + id, school_id: 10, student_id: id,
          student_name: yearID === previousAcademicYear.id ? "Previous Year Enrollment" : `Enrollment Student ${id}`,
          student_status: "active", academic_year_id: yearID || academicYear.id,
          class_id: 3, section_id: 12, roll_number: String(id),
          class_name_snapshot: "Class 1", section_name_snapshot: "A",
          admission_number: `ADM-${id}`, admission_date: "2026-04-01",
          status: "active", enrolled_at: "2026-04-01T00:00:00.000Z", ended_at: null,
        };
      });
      await route.fulfill({ json: { enrollments, pagination: { limit: 20, offset, total } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("e2e-enrollment-pagination");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Students" }).click();
  await page.getByRole("button", { name: "Enrollments" }).click();
  await expect(page.getByText("Enrollment Student 1", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–20 of 45")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Enrollment Student 21", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 21–40 of 45")).toBeVisible();
  await expect.poll(() => requestedPages.at(-1)).toEqual({ offset: 20, yearID: 0 });

  await page.getByRole("button", { name: "Filter enrollments by academic year" }).click();
  await page.getByRole("listbox", { name: "Filter enrollments by academic year" }).getByRole("option", { name: "2025–2026" }).click();
  await expect(page.getByText("Previous Year Enrollment", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1")).toBeVisible();
  await expect.poll(() => requestedPages.at(-1)).toEqual({ offset: 0, yearID: previousAcademicYear.id });
});

test("global notifications remain visible within a narrow mobile viewport", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("/");
  await page.evaluate(() => {
    window.dispatchEvent(new CustomEvent("sms:toast", {
      detail: { kind: "success", message: "Responsive notification check" },
    }));
  });
  const notification = page.getByRole("status").filter({ hasText: "Responsive notification check" });
  await expect(notification).toBeVisible();
  const bounds = await notification.boundingBox();
  expect(bounds).not.toBeNull();
  expect(bounds!.x).toBeGreaterThanOrEqual(0);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(375);
  expect(bounds!.y).toBeLessThan(100);

  await page.evaluate(() => {
    ["Queued notification two", "Queued notification three", "Queued notification four"].forEach(message => {
      window.dispatchEvent(new CustomEvent("sms:toast", {
        detail: { kind: "info", message },
      }));
    });
  });
  const viewport = page.locator(".app-toast-viewport");
  await expect(viewport.locator(".app-toast")).toHaveCount(3);
  await expect(viewport.getByText("Queued notification four")).toHaveCount(0);
  await viewport.locator(".app-toast-dismiss").first().click();
  await expect(viewport.getByText("Queued notification four")).toBeVisible();
});
