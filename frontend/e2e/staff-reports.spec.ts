import { expect, test } from "@playwright/test";

const schoolAdmin = {
  principal_id: 101,
  principal_type: "staff",
  status: "active",
  school_id: 10,
  username: "reports-admin",
  permissions: ["users.read"],
};

test("staff qualifications and experience reports filter, paginate and retain historical employees", async ({ page }) => {
  const qualificationRequests: URL[] = [];
  const experienceRequests: URL[] = [];
  let failQualificationExport = false;
  let duplicateQualificationExport = false;
  let invalidExperienceExportMetadata = false;
  let qualificationTotal = 51;
  let experienceTotal = 51;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-reports-e2e-token", token_type: "Bearer", principal: schoolAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: schoolAdmin });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      await route.fulfill({ json: { staff: [
        { id: 1, principal_id: 201, school_id: 10, username: "active.teacher", employee_code: "EMP-201", full_name: "Active Teacher", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active" },
        { id: 2, principal_id: 202, school_id: 10, username: "former.teacher", employee_code: "EMP-202", full_name: "Former Teacher", designation: "Teacher", employment_status: "inactive", profile_complete: true, status: "disabled" },
      ] } });
      return;
    }
    if (path === "/school/staff/qualifications-report" && request.method() === "GET") {
      qualificationRequests.push(url);
      if (failQualificationExport && url.searchParams.get("limit") === "200" && url.searchParams.get("offset") === "0") {
        await route.fulfill({ status: 503, json: { message: "Temporary report failure" } });
        return;
      }
      const recordType = url.searchParams.get("record_type") ?? "all";
      const status = url.searchParams.get("verification_status") ?? "";
      const offset = Number(url.searchParams.get("offset") ?? "0");
      const filtered = recordType === "certification" && status === "pending";
      const requestedLimit = Number(url.searchParams.get("limit") ?? "50");
      const allRows = Array.from({ length: 51 }, (_, index) => ({
        record_type: "qualification", record_id: index + 1, staff_principal_id: 201, employee_code: "EMP-201",
        staff_name: "Active Teacher", record_name: `Qualification ${index + 1}`, organization: "State University",
        record_date: "2021", expiry_date: "", verification_status: "pending",
      }));
      const rows = filtered
        ? [{ record_type: "certification", record_id: 900, staff_principal_id: 201, employee_code: "EMP-201", staff_name: "Active Teacher", record_name: "First Aid Certificate", organization: "Safety Institute", record_date: "2026-01-10", expiry_date: "2028-01-10", verification_status: "pending" }]
        : allRows.slice(offset, Math.min(offset + requestedLimit, qualificationTotal));
      if (duplicateQualificationExport && requestedLimit === 200 && rows.length > 1) rows[1] = { ...rows[1], record_id: rows[0].record_id };
      const counts = filtered
        ? [{ record_type: "certification", verification_status: "pending", count: 1 }]
        : [{ record_type: "qualification", verification_status: "pending", count: qualificationTotal }];
      await route.fulfill({ json: { report: { rows, counts, limit: requestedLimit, offset, total: filtered ? 1 : qualificationTotal } } });
      return;
    }
    if (path === "/school/staff/experience-report" && request.method() === "GET") {
      experienceRequests.push(url);
      const currentOnly = url.searchParams.get("current_only") === "true";
      const offset = Number(url.searchParams.get("offset") ?? "0");
      const requestedLimit = Number(url.searchParams.get("limit") ?? "50");
      const allRows = [
        { record_id: 701, staff_principal_id: 202, employee_code: "EMP-202", staff_name: "Former Teacher", designation: "Teacher", employment_status: "inactive", employer_name: "Previous School", previous_designation: "Teacher", employment_type: "contract", start_date: "2018-04-01", end_date: "2022-03-31", is_current: false },
        { record_id: 702, staff_principal_id: 202, employee_code: "EMP-202", staff_name: "Former Teacher", designation: "Teacher", employment_status: "inactive", employer_name: "Current Academy", previous_designation: "Senior Teacher", employment_type: "permanent", start_date: "2022-04-01", end_date: "", is_current: true },
        ...Array.from({ length: 48 }, (_, index) => ({
          record_id: 703 + index, staff_principal_id: 202, employee_code: "EMP-202", staff_name: "Former Teacher", designation: "Teacher", employment_status: "inactive",
          employer_name: `Historical School ${index + 1}`, previous_designation: "Teacher", employment_type: "contract", start_date: "2010-04-01", end_date: "2011-03-31", is_current: false,
        })),
        { record_id: 751, staff_principal_id: 202, employee_code: "EMP-202", staff_name: "Former Teacher", designation: "Teacher", employment_status: "inactive", employer_name: "Older Academy", previous_designation: "Teacher", employment_type: "contract", start_date: "2015-04-01", end_date: "2018-03-31", is_current: false },
      ];
      const rows = currentOnly ? [allRows[1]] : allRows.slice(offset, Math.min(offset + requestedLimit, experienceTotal));
      const counts = currentOnly
        ? [{ employment_type: "permanent", is_current: true, count: 1 }]
        : [{ employment_type: "contract", is_current: false, count: 50 }, { employment_type: "permanent", is_current: true, count: 1 }];
      await route.fulfill({ json: { report: { rows, counts, limit: requestedLimit, offset, total: invalidExperienceExportMetadata && requestedLimit === 200 ? experienceTotal + 1 : currentOnly ? 1 : experienceTotal } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("reports-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();
  await page.getByRole("button", { name: "Reports", exact: true }).click();

  await expect(page.getByRole("heading", { name: "Staff Reports" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Qualifications & certifications" })).toBeVisible();
  await expect(page.getByText("Qualification 1", { exact: true })).toBeVisible();
  const qualificationPagination = page.locator(".staff-operational-report-section").nth(0).locator(".staff-operational-report-pagination");
  await expect(qualificationPagination).toContainText("Showing 1–50 of 51");
  await qualificationPagination.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Qualification 51", { exact: true })).toBeVisible();
  await expect(qualificationPagination).toContainText("Showing 51–51 of 51");
  expect(qualificationRequests.some(url => url.searchParams.get("offset") === "50")).toBeTruthy();

  const qualificationSection = page.locator(".staff-operational-report-section").nth(0);
  const qualificationDownloadPromise = page.waitForEvent("download");
  await qualificationSection.getByRole("button", { name: "Export all matching CSV" }).click();
  const qualificationDownload = await qualificationDownloadPromise;
  expect(qualificationDownload.suggestedFilename()).toBe("staff-qualifications-certifications.csv");
  expect(qualificationRequests.some(url => url.searchParams.get("offset") === "0" && url.searchParams.get("limit") === "200")).toBeTruthy();
  qualificationTotal = 50;
  await qualificationSection.getByRole("button", { name: "Refresh" }).click();
  await expect(qualificationSection.getByText("Qualification 50", { exact: true })).toBeVisible();
  await expect(qualificationSection.getByText("Qualification 51", { exact: true })).toHaveCount(0);
  await expect(qualificationPagination).toHaveCount(0);
  expect(qualificationRequests.some(url => url.searchParams.get("offset") === "0" && url.searchParams.get("limit") === "50")).toBeTruthy();
  await qualificationSection.getByLabel("Record type").selectOption("certification");
  await qualificationSection.getByLabel("Verification status").selectOption("pending");
  await expect(page.getByText("First Aid Certificate", { exact: true })).toBeVisible();
  expect(qualificationRequests.some(url => url.searchParams.get("record_type") === "certification" && url.searchParams.get("verification_status") === "pending" && url.searchParams.get("offset") === "0")).toBeTruthy();

  await expect(page.getByRole("heading", { name: "Staff experience history" })).toBeVisible();
  const experienceSection = page.locator(".staff-operational-report-section").nth(1);
  await expect(experienceSection.getByText("Previous School", { exact: true })).toBeVisible();
  await expect(experienceSection.getByText("Former Teacher").first()).toBeVisible();
  const experiencePagination = experienceSection.locator(".staff-operational-report-pagination");
  await expect(experiencePagination).toContainText("Showing 1–50 of 51");
  await experiencePagination.getByRole("button", { name: "Next" }).click();
  await expect(experienceSection.getByText("Older Academy", { exact: true })).toBeVisible();
  await expect(experiencePagination).toContainText("Showing 51–51 of 51");
  expect(experienceRequests.some(url => url.searchParams.get("offset") === "50")).toBeTruthy();
  const experienceDownloadPromise = page.waitForEvent("download");
  await experienceSection.getByRole("button", { name: "Export all matching CSV" }).click();
  const experienceDownload = await experienceDownloadPromise;
  expect(experienceDownload.suggestedFilename()).toBe("staff-experience-report.csv");
  expect(experienceRequests.some(url => url.searchParams.get("offset") === "0" && url.searchParams.get("limit") === "200")).toBeTruthy();
  experienceTotal = 50;
  await experienceSection.getByRole("button", { name: "Refresh" }).click();
  await expect(experienceSection.getByText("Previous School", { exact: true })).toBeVisible();
  await expect(experienceSection.getByText("Older Academy", { exact: true })).toHaveCount(0);
  await expect(experiencePagination).toHaveCount(0);
  expect(experienceRequests.some(url => url.searchParams.get("offset") === "0" && url.searchParams.get("limit") === "50")).toBeTruthy();
  await experienceSection.getByLabel("Experience status").selectOption("current");
  await expect(experienceSection.getByText("Current Academy", { exact: true })).toBeVisible();
  expect(experienceRequests.some(url => url.searchParams.get("current_only") === "true")).toBeTruthy();
  await experienceSection.getByLabel("Experience status").selectOption("all");
  await expect(experienceSection.getByText("Previous School", { exact: true })).toBeVisible();
  invalidExperienceExportMetadata = true;
  const invalidExperienceDownload = page.waitForEvent("download", { timeout: 1000 }).catch(() => null);
  await experienceSection.getByRole("button", { name: "Export all matching CSV" }).click();
  await expect(experienceSection.getByRole("alert")).toContainText("No partial CSV was created");
  expect(await invalidExperienceDownload).toBeNull();
  invalidExperienceExportMetadata = false;

  // An export-page failure must not create a misleading partial CSV.
  failQualificationExport = true;
  await qualificationSection.getByLabel("Record type").selectOption("all");
  await qualificationSection.getByLabel("Verification status").selectOption("");
  const failedDownload = page.waitForEvent("download", { timeout: 1000 }).catch(() => null);
  await qualificationSection.getByRole("button", { name: "Export all matching CSV" }).click();
  await expect(qualificationSection.getByRole("alert")).toContainText("No partial CSV was created");
  expect(await failedDownload).toBeNull();

  // Duplicate record IDs must abort the whole export rather than silently duplicating data.
  failQualificationExport = false;
  duplicateQualificationExport = true;
  const duplicateDownload = page.waitForEvent("download", { timeout: 1000 }).catch(() => null);
  await qualificationSection.getByRole("button", { name: "Export all matching CSV" }).click();
  await expect(qualificationSection.getByRole("alert")).toContainText("No partial CSV was created");
  expect(await duplicateDownload).toBeNull();
});

test("staff operational reports are reachable with report-only permissions without loading the staff directory", async ({ page }) => {
  const attendanceOnlyAdmin = {
    principal_id: 102,
    principal_type: "staff",
    status: "active",
    school_id: 10,
    username: "attendance-reports-admin",
    permissions: ["staff.attendance.read"],
  };
  const staffDirectoryRequests: string[] = [];
  const attendanceRequests: string[] = [];

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-report-only-e2e-token", token_type: "Bearer", principal: attendanceOnlyAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: attendanceOnlyAdmin });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      staffDirectoryRequests.push(request.method() + " " + path);
      await route.fulfill({ json: { staff: [] } });
      return;
    }
    if (path === "/school/staff-attendance/month" && request.method() === "GET") {
      attendanceRequests.push(url.searchParams.get("month") ?? "");
      const limit = Number(url.searchParams.get("limit") ?? 50);
      const offset = Number(url.searchParams.get("offset") ?? 0);
      await route.fulfill({ json: { month: url.searchParams.get("month"), data: [], pagination: { limit, offset, total: 0 } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("attendance-reports-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Attendance & leave reports" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Monthly staff attendance" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Staff Reports" })).toHaveCount(0);
  await expect.poll(() => attendanceRequests.length).toBeGreaterThan(0);
  expect(staffDirectoryRequests).toHaveLength(0);
});

test("staff leave reports are reachable with leave-only permissions without loading staff directory or profile reports", async ({ page }) => {
  const leaveOnlyAdmin = {
    principal_id: 103,
    principal_type: "staff",
    status: "active",
    school_id: 10,
    username: "leave-reports-admin",
    permissions: ["staff.leave.read"],
  };
  const staffDirectoryRequests: string[] = [];
  const leaveReportRequests: string[] = [];

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-leave-report-only-e2e-token", token_type: "Bearer", principal: leaveOnlyAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: leaveOnlyAdmin });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      staffDirectoryRequests.push(request.method() + " " + path);
      await route.fulfill({ json: { staff: [] } });
      return;
    }
    if (path === "/school/staff-leave/requests" && request.method() === "GET") {
      const pageSize = Number(url.searchParams.get("page_size") ?? "50");
      if (pageSize === 100) leaveReportRequests.push(url.search);
      await route.fulfill({ json: { data: [], pagination: { page: 1, page_size: pageSize, total: 0 } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("leave-reports-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Attendance & leave reports" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Leave requests" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Staff Reports" })).toHaveCount(0);
  await expect.poll(() => leaveReportRequests.length).toBeGreaterThan(0);
  expect(staffDirectoryRequests).toHaveLength(0);
});

test("academic assignment report respects teacher-subject-only permission", async ({ page }) => {
  const subjectOnlyAdmin = {
    principal_id: 104,
    principal_type: "staff",
    status: "active",
    school_id: 10,
    username: "subject-report-admin",
    permissions: ["teacher_assignments.read"],
  };
  const subjectRequests: string[] = [];
  const classInchargeRequests: string[] = [];
  let currentYearRequests = 0;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-subject-report-only-e2e-token", token_type: "Bearer", principal: subjectOnlyAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: subjectOnlyAdmin });
      return;
    }
    if (path === "/school/timetable/current-year" && request.method() === "GET") {
      currentYearRequests += 1;
      await route.fulfill({ json: { academic_year: { id: 2026, name: "2026-27" } } });
      return;
    }
    if (path === "/school/teacher-assignments" && request.method() === "GET") {
      subjectRequests.push(url.search);
      await route.fulfill({ json: { assignments: [] } });
      return;
    }
    if (path === "/school/class-incharge-assignments" && request.method() === "GET") {
      classInchargeRequests.push(url.search);
      await route.fulfill({ json: { assignments: [] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("subject-report-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Academic assignments" })).toBeVisible();
  await expect(page.getByText("Subject assignments", { exact: true })).toBeVisible();
  await expect(page.getByText("Class-in-charge assignments", { exact: true })).toHaveCount(0);
  await expect.poll(() => subjectRequests.length).toBeGreaterThan(0);
  expect(currentYearRequests).toBeGreaterThan(0);
  expect(classInchargeRequests).toHaveLength(0);
});

test("academic assignment report respects class-in-charge-only permission", async ({ page }) => {
  const inchargeOnlyAdmin = {
    principal_id: 105,
    principal_type: "staff",
    status: "active",
    school_id: 10,
    username: "incharge-report-admin",
    permissions: ["class_incharge_assignments.read"],
  };
  const subjectRequests: string[] = [];
  const inchargeRequests: string[] = [];
  let currentYearRequests = 0;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-incharge-report-only-e2e-token", token_type: "Bearer", principal: inchargeOnlyAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: inchargeOnlyAdmin });
      return;
    }
    if (path === "/school/timetable/current-year" && request.method() === "GET") {
      currentYearRequests += 1;
      await route.fulfill({ json: { academic_year: { id: 2026, name: "2026-27" } } });
      return;
    }
    if (path === "/school/teacher-assignments" && request.method() === "GET") {
      subjectRequests.push(url.search);
      await route.fulfill({ json: { assignments: [] } });
      return;
    }
    if (path === "/school/class-incharge-assignments" && request.method() === "GET") {
      inchargeRequests.push(url.search);
      await route.fulfill({ json: { assignments: [] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("incharge-report-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Academic assignments" })).toBeVisible();
  await expect(page.getByText("Class-in-charge assignments", { exact: true })).toBeVisible();
  await expect(page.getByText("Subject assignments", { exact: true })).toHaveCount(0);
  await expect.poll(() => inchargeRequests.length).toBeGreaterThan(0);
  expect(currentYearRequests).toBeGreaterThan(0);
  expect(subjectRequests).toHaveLength(0);
});


test("workload-only permission opens the workload report without loading staff directory or operational reports", async ({ page }) => {
  const workloadOnlyAdmin = {
    principal_id: 106,
    principal_type: "staff",
    status: "active",
    school_id: 10,
    username: "workload-report-admin",
    permissions: ["timetable.read"],
  };
  const staffDirectoryRequests: string[] = [];
  const workloadRequests: string[] = [];
  let currentYearRequests = 0;

  await page.route("**/api/**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname.replace(/^\/api/, "");

    if (path === "/auth/refresh") {
      await route.fulfill({ status: 401, json: { message: "No active session" } });
      return;
    }
    if (path === "/auth/login" && request.method() === "POST") {
      await route.fulfill({ json: { access_token: "staff-workload-report-only-e2e-token", token_type: "Bearer", principal: workloadOnlyAdmin } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({ json: workloadOnlyAdmin });
      return;
    }
    if (path === "/school/staff" && request.method() === "GET") {
      staffDirectoryRequests.push(request.method() + " " + path);
      await route.fulfill({ json: { staff: [] } });
      return;
    }
    if (path === "/school/timetable/current-year" && request.method() === "GET") {
      currentYearRequests += 1;
      await route.fulfill({ json: { academic_year: { id: 2026, name: "2026-27" } } });
      return;
    }
    if (path === "/school/timetable/workload-report" && request.method() === "GET") {
      workloadRequests.push(url.search);
      await route.fulfill({ json: { workloads: [{
        teacher_principal_id: 501, teacher_name: "Workload Teacher", employee_code: "EMP-501", designation: "Teacher",
        employment_status: "active", login_status: "active", day_of_week: 1, scheduled_periods: 3,
        distinct_sections: 2, distinct_subjects: 2, first_period: 1, last_period: 3,
      }] } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${request.method()} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("workload-report-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();

  await expect(page.getByRole("button", { name: "Workload Report", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Reports", exact: true })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "School-wide Workload Report" })).toBeVisible();
  await expect(page.getByText("Workload Teacher", { exact: true })).toBeVisible();
  await expect.poll(() => workloadRequests.length).toBeGreaterThan(0);
  expect(currentYearRequests).toBeGreaterThan(0);
  expect(staffDirectoryRequests).toHaveLength(0);
});
