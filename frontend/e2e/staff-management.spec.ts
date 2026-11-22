import { expect, test, type Page, type Route } from "@playwright/test";
import { schoolDateValue } from "../src/shared/date/schoolDate";

const staffUser = {
  principal_id: 101,
  principal_type: "staff",
  status: "active",
  school_id: 10,
};

async function signInAndOpenStaff(page: Page, staffRows: Array<Record<string, unknown>>, onStaffMutation?: (path: string, route: Route) => Promise<void>, permissions = ["users.read", "users.create", "users.update"]) {
  const staffRecords: Record<string, Array<Record<string, unknown>>> = { qualifications: [], certifications: [], experience: [] };
  const leaveTypes: Array<Record<string, unknown>> = [{ id: 1, name: "Casual Leave", is_paid: true, requires_approval: true, is_active: true }];
  const leaveRequests: Array<Record<string, unknown>> = [{
    id: 21, staff_principal_id: 501, employee_code: "EMP-LEAVE-01", staff_name: "Leave Teacher",
    leave_type_id: 1, leave_type_name: "Casual Leave", start_date: "2026-11-24", end_date: "2026-11-24",
    duration_type: "full_day", reason: "Family commitment", status: "pending", created_at: "2026-11-21T00:00:00.000Z",
  }];
  const schoolHolidays: Array<Record<string, unknown>> = [{ id: 31, holiday_date: "2026-11-26", name: "School Foundation Day", description: "Annual school holiday", is_active: true }];
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
      await route.fulfill({ json: { access_token: "staff-e2e-token", token_type: "Bearer", principal: staffUser } });
      return;
    }
    if (path === "/auth/me") {
      await route.fulfill({
        json: {
          ...staffUser,
          username: "staff-e2e-admin",
          permissions,
        },
      });
      return;
    }
    if (path === "/school/staff" && method === "GET") {
      await route.fulfill({ json: { staff: staffRows } });
      return;
    }
    if (path === "/school/staff-leave/types" && method === "GET") {
      await route.fulfill({ json: { data: leaveTypes } });
      return;
    }
    if (path === "/school/staff-leave/types" && method === "POST") {
      const payload = request.postDataJSON() as Record<string, unknown>;
      const saved = { ...payload, id: leaveTypes.length + 1 };
      leaveTypes.push(saved);
      await route.fulfill({ json: { data: saved } });
      return;
    }
    const leaveTypeUpdate = path.match(/^\/school\/staff-leave\/types\/(\d+)$/);
    if (leaveTypeUpdate && method === "PATCH") {
      const target = leaveTypes.find(item => item.id === Number(leaveTypeUpdate[1]));
      if (!target) { await route.fulfill({ status: 404, json: { message: "Leave type not found" } }); return; }
      Object.assign(target, request.postDataJSON() as Record<string, unknown>);
      await route.fulfill({ json: { data: target } });
      return;
    }
    if (path === "/school/staff-leave/requests" && method === "GET") {
      const status = url.searchParams.get("status") ?? "";
      const staffID = Number(url.searchParams.get("staff_principal_id") ?? 0);
      const data = leaveRequests.filter(item => (!status || item.status === status) && (!staffID || item.staff_principal_id === staffID));
      await route.fulfill({ json: { data } });
      return;
    }
    if (path === "/school/staff-leave/requests" && method === "POST") {
      const payload = request.postDataJSON() as Record<string, unknown>;
      const staff = staffRows.find(item => item.principal_id === payload.staff_principal_id) ?? {};
      const type = leaveTypes.find(item => item.id === payload.leave_type_id) ?? {};
      const saved = { ...payload, id: leaveRequests.length + 22, staff_name: staff.full_name, employee_code: staff.employee_code, leave_type_name: type.name, status: "pending", created_at: "2026-11-22T00:00:00.000Z" };
      leaveRequests.push(saved);
      await route.fulfill({ status: 201, json: { data: saved } });
      return;
    }
    const leaveDecision = path.match(/^\/school\/staff-leave\/requests\/(\d+)\/(approve|reject)$/);
    if (leaveDecision && method === "POST") {
      const target = leaveRequests.find(item => item.id === Number(leaveDecision[1]));
      if (!target) { await route.fulfill({ status: 404, json: { message: "Request not found" } }); return; }
      const payload = request.postDataJSON() as Record<string, unknown>;
      target.status = leaveDecision[2] === "approve" ? "approved" : "rejected";
      target.decision_comment = payload.comment ?? null;
      await route.fulfill({ json: { message: "Decision saved" } });
      return;
    }
    const leaveCancel = path.match(/^\/school\/staff-leave\/requests\/(\d+)\/cancel$/);
    if (leaveCancel && method === "POST") {
      const target = leaveRequests.find(item => item.id === Number(leaveCancel[1]));
      if (target) target.status = "cancelled";
      await route.fulfill({ json: { message: "Request cancelled" } });
      return;
    }
    if (path === "/school/holidays" && method === "GET") {
      const includeInactive = url.searchParams.get("include_inactive") === "true";
      await route.fulfill({ json: { data: includeInactive ? schoolHolidays : schoolHolidays.filter(item => item.is_active === true) } });
      return;
    }
    const holidayUpdate = path.match(/^\/school\/holidays\/(\d+)$/);
    if (holidayUpdate && method === "PATCH") {
      const target = schoolHolidays.find(item => item.id === Number(holidayUpdate[1]));
      if (!target) { await route.fulfill({ status: 404, json: { message: "Holiday not found" } }); return; }
      Object.assign(target, request.postDataJSON() as Record<string, unknown>);
      await route.fulfill({ json: { data: target } });
      return;
    }
    if (path === "/school/holidays" && method === "POST") {
      const payload = request.postDataJSON() as Record<string, unknown>;
      const saved = { ...payload, id: schoolHolidays.length + 32 };
      schoolHolidays.push(saved);
      await route.fulfill({ json: { data: saved } });
      return;
    }
    const staffRecordMatch = path.match(/^\/school\/staff\/(\d+)\/(qualifications|certifications|experience)(?:\/(\d+))?$/);
    if (staffRecordMatch) {
      const principalID = Number(staffRecordMatch[1]);
      const collection = staffRecordMatch[2];
      if (method === "GET") {
        await route.fulfill({ json: { [collection]: staffRecords[collection] } });
        return;
      }
      if (method === "POST") {
        const payload = request.postDataJSON() as Record<string, unknown>;
        const saved = { ...payload, id: staffRecords[collection].length + 1, staff_principal_id: principalID, verification_status: "pending", created_at: "2026-11-21T00:00:00.000Z", updated_at: "2026-11-21T00:00:00.000Z" };
        staffRecords[collection].push(saved);
        const singular = collection === "qualifications" ? "qualification" : collection === "certifications" ? "certification" : "experience";
        await route.fulfill({ status: 201, json: { [singular]: saved } });
        return;
      }
      if (method === "PATCH" && staffRecordMatch[3]) {
        const payload = request.postDataJSON() as Record<string, unknown>;
        const recordID = Number(staffRecordMatch[3]);
        const index = staffRecords[collection].findIndex(item => item.id === recordID);
        if (index < 0) { await route.fulfill({ status: 404, json: { message: "Record not found" } }); return; }
        const saved = { ...staffRecords[collection][index], ...payload, updated_at: "2026-11-21T01:00:00.000Z" };
        staffRecords[collection][index] = saved;
        const singular = collection === "qualifications" ? "qualification" : collection === "certifications" ? "certification" : "experience";
        await route.fulfill({ status: 200, json: { [singular]: saved } });
        return;
      }
    }
    if (onStaffMutation && (path === "/school/staff" && method === "POST" || /^\/school\/staff\/\d+$/.test(path) && method === "PATCH")) {
      await onStaffMutation(path, route);
      return;
    }
    await route.fulfill({ status: 404, json: { message: `Unmocked E2E API request: ${method} ${path}` } });
  });

  await page.goto("/");
  await page.getByLabel("Username").fill("staff-e2e-admin");
  await page.getByLabel("Password").fill("E2E-only-password");
  await page.getByRole("button", { name: /Continue/ }).click();
  await page.getByRole("button", { name: "Staff", exact: true }).click();
}

test("staff workspace opens on a real-data overview and hides permission-gated or later-phase tabs", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 111, school_id: 10, username: "teacher.overview",
    employee_code: "EMP-OV-01", full_name: "Overview Teacher", designation: "Teacher",
    mobile_number: "9876543210", employment_status: "active", profile_complete: true,
    status: "disabled", must_change_password: true, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "People at a glance" })).toBeVisible();
  await expect(page.getByText("Total staff")).toBeVisible();
  await expect(page.locator(".staff-overview-cards article").nth(0).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-overview-cards article").nth(1).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-overview-cards article").nth(2).locator("strong")).toHaveText("0");
  await expect(page.getByRole("button", { name: /Attendance/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Leave/ })).toBeDisabled();
  await expect(page.getByRole("button", { name: /Reports/ })).toBeDisabled();
});

test("staff leave workspace reviews applications and displays configured leave types and school holidays", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "users.create", "users.update", "staff.leave.read", "staff.leave.manage_types", "staff.leave.manage_requests", "staff.leave.decide", "staff.holidays.read", "staff.holidays.manage"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  await expect(page.getByText("Leave Teacher")).toBeVisible();
  await expect(page.getByText("Family commitment")).toBeVisible();
  await page.getByRole("button", { name: "Approve" }).click();
  const decision = page.getByRole("dialog", { name: "Approve leave request" });
  await expect(decision).toBeVisible();
  await decision.getByRole("button", { name: "Confirm approval" }).click();
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();
  await page.getByRole("button", { name: "Holiday calendar" }).click();
  await expect(page.getByText("School Foundation Day")).toBeVisible();
  await page.getByRole("button", { name: "Leave types" }).click();
  await expect(page.getByText("Casual Leave")).toBeVisible();
});

test("staff leave request filters send the selected status and staff scope", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
      employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "second.teacher",
      employee_code: "EMP-LEAVE-02", full_name: "Second Teacher", designation: "Math Teacher",
      mobile_number: "9876543216", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
  ];
  const permissions = ["users.read", "staff.leave.read"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  const approvedRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return request.method() === "GET" && url.pathname.endsWith("/school/staff-leave/requests") &&
      url.searchParams.get("status") === "approved";
  });
  await page.getByRole("button", { name: "Leave request status" }).click();
  await page.getByRole("option", { name: "Approved" }).click();
  const approvedURL = new URL((await approvedRequest).url());
  expect(approvedURL.searchParams.get("status")).toBe("approved");
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();

  const staffScopedRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return request.method() === "GET" && url.pathname.endsWith("/school/staff-leave/requests") &&
      url.searchParams.get("status") === "approved" &&
      url.searchParams.get("staff_principal_id") === "502";
  });
  await page.getByRole("button", { name: "Filter by staff" }).click();
  await page.getByRole("option", { name: "Second Teacher · EMP-LEAVE-02" }).click();
  const scopedURL = new URL((await staffScopedRequest).url());
  expect(scopedURL.searchParams.get("status")).toBe("approved");
  expect(scopedURL.searchParams.get("staff_principal_id")).toBe("502");
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();
});

test("staff leave types can be edited and safely deactivated", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.manage_types"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Leave types" }).click();

  const typeRow = page.locator(".staff-leave-type-row").filter({ hasText: "Casual Leave" });
  await typeRow.getByRole("button", { name: "Edit" }).click();
  await page.getByPlaceholder("e.g. Casual Leave").fill("Annual Leave");
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Annual Leave")).toBeVisible();

  const updatedRow = page.locator(".staff-leave-type-row").filter({ hasText: "Annual Leave" });
  await updatedRow.getByRole("button", { name: "Deactivate" }).click();
  const confirmation = page.getByRole("dialog", { name: "Deactivate leave type?" });
  await expect(confirmation).toBeVisible();
  await confirmation.getByRole("button", { name: "Deactivate leave type" }).click();
  await expect(updatedRow.getByText("Inactive")).toBeVisible();
});

test("staff directory opens the eight-tab employee profile and shows saved profile details", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 411, school_id: 10, username: "teacher.profile",
    employee_code: "EMP-PRO-01", full_name: "Profile Teacher", designation: "English Teacher",
    mobile_number: "9876543212", email: "profile.teacher@example.com",
    joining_date: "2026-04-01", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Profile Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Profile Teacher" });
  await expect(dialog.getByRole("navigation", { name: "Staff profile sections" }).getByRole("button")).toHaveCount(8);
  await expect(dialog.getByText("EMP-PRO-01").first()).toBeVisible();
  await dialog.getByRole("button", { name: "Personal Details" }).click();
  await expect(dialog.getByText("profile.teacher@example.com")).toBeVisible();
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await expect(dialog.getByText("No education records added yet.")).toBeVisible();
  await expect(dialog.getByText("No certifications added yet.")).toBeVisible();
  await dialog.getByRole("button", { name: "Documents" }).click();
  await expect(dialog.getByText(/private file storage/)).toBeVisible();
});

test("add staff wizard validates each step before allowing progress", async ({ page }) => {
  await signInAndOpenStaff(page, []);
  await page.getByRole("button", { name: "Add staff", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter the employee full name (up to 160 characters).")).toBeVisible();
  await expect(dialog.getByText("Use 1–32 letters, numbers, dots, underscores or hyphens.")).toBeVisible();
  await expect(dialog.getByText("Enter one mobile number with 10–15 digits.")).toBeVisible();

  await dialog.getByLabel("Full name").fill("Validation Teacher");
  await dialog.getByLabel("Employee code").fill("EMP-VAL-01");
  await dialog.getByLabel("Mobile number").fill("9876543219");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter a designation (up to 100 characters).")).toBeVisible();
});

test("staff creation uses the shared IST date picker and submits its selected date", async ({ page }) => {
  const created: Array<Record<string, unknown>> = [];
  const rows: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    created.push(payload);
    rows.push({
      id: 1, principal_id: 201, school_id: 10, username: payload.username,
      employee_code: payload.employee_code, full_name: payload.full_name,
      designation: payload.designation, mobile_number: payload.mobile_number,
      joining_date: payload.joining_date, employment_status: "active",
      profile_complete: true, status: "active", must_change_password: true,
      created_at: "2026-11-21T00:00:00.000Z",
    });
    await route.fulfill({ json: { staff: rows[0] } });
  });

  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByRole("textbox", { name: "Search staff by name, code, designation or username" })).toBeVisible();
  await page.getByRole("button", { name: "Add staff" }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Full name").fill("Teacher One");
  await dialog.getByLabel("Employee code").fill("EMP-001");
  await dialog.getByLabel("Mobile number").fill("9876543210");
  await dialog.getByRole("button", { name: "Continue" }).click();

  await dialog.getByLabel("Designation").fill("Class Teacher");
  await dialog.getByRole("button", { name: "Joining date" }).click();
  const calendar = dialog.getByRole("dialog", { name: "Choose joining date" });
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await calendar.locator(".app-date-day[tabindex='0']").press("ArrowRight");
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await calendar.getByRole("button", { name: "Choose month" }).click();
  await calendar.locator(".app-date-month-grid button[aria-pressed='true']").click();
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await calendar.getByRole("button", { name: "Choose year" }).click();
  await calendar.locator(".app-date-year-grid button[aria-pressed='true']").click();
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await dialog.getByRole("button", { name: "Today", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();

  await dialog.getByLabel("Login username").fill("teacher.one");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByRole("heading", { name: "Review & create" })).toBeVisible();
  await expect(dialog.getByText("Teacher One", { exact: true })).toBeVisible();
  await expect(dialog.getByText("EMP-001", { exact: true })).toBeVisible();
  await expect(dialog.getByText("teacher.one", { exact: true })).toBeVisible();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => created.length).toBe(1);
  expect(created[0].joining_date).toBe(schoolDateValue());
  await expect(dialog).toHaveCount(0);
  const handoff = page.getByRole("dialog", { name: "Share staff login credentials" });
  await expect(handoff).toBeVisible();
  await expect(handoff.getByText("teacher.one")).toBeVisible();
  await expect(handoff.getByRole("button", { name: "Do this later" })).toBeVisible();
  await handoff.getByRole("button", { name: "Do this later" }).click();
  await expect(handoff).toHaveCount(0);
  await expect(page.locator('input[type="date"]')).toHaveCount(0);
});

test("staff profile edit uses the shared employment-status select and sends the selected value", async ({ page }) => {
  const updates: Array<Record<string, unknown>> = [];
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 301, school_id: 10, username: "teacher.two",
    employee_code: "EMP-002", full_name: "Teacher Two", designation: "Science Teacher",
    mobile_number: "9876543211", joining_date: "2026-04-01",
    employment_status: "active", profile_complete: true, status: "active",
    must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    updates.push(route.request().postDataJSON() as Record<string, unknown>);
    await route.fulfill({ json: { staff: { ...rows[0], ...(updates[0] ?? {}) }, message: "Updated" } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Edit employee profile for teacher.two" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit employee profile" });
  await expect(dialog.getByRole("button", { name: "Joining date" })).toContainText("01 Apr 2026");
  await dialog.getByRole("button", { name: "Employment status" }).click();
  await dialog.getByRole("listbox", { name: "Employment status" }).getByRole("option", { name: "Inactive" }).click();
  await dialog.getByRole("button", { name: "Save changes" }).click();

  await expect.poll(() => updates.length).toBe(1);
  expect(updates[0].employment_status).toBe("inactive");
  await expect(dialog).toHaveCount(0);
  await expect(page.locator("select")).toHaveCount(0);
});

test("staff profile creates education, certification and experience records", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 611, school_id: 10, username: "teacher.records",
    employee_code: "EMP-REC-01", full_name: "Records Teacher", designation: "Science Teacher",
    mobile_number: "9876543213", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Records Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Records Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();

  await dialog.getByRole("button", { name: "Add qualification" }).click();
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Enter qualification name.")).toBeVisible();
  await expect(dialog.getByText("Enter institution.")).toBeVisible();
  await dialog.locator('input[name="qualification_name"]').fill("M.Sc. Mathematics");
  await dialog.locator('input[name="institution"]').fill("State University");
  await dialog.locator('input[name="completion_year"]').fill("1899");
  await dialog.locator('input[name="score"]').fill("8.5");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Enter a year between 1900 and 2200.")).toBeVisible();
  await expect(dialog.getByText("Choose a score scale when a score is entered.")).toBeVisible();
  await dialog.locator('input[name="completion_year"]').fill("2021");
  await dialog.locator('select[name="score_scale"]').selectOption("cgpa");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("M.Sc. Mathematics")).toBeVisible();
  const qualificationCard = dialog.locator(".staff-record-card").filter({ hasText: "M.Sc. Mathematics" });
  await qualificationCard.getByRole("button", { name: "Edit" }).click();
  await expect(dialog.locator('input[name="qualification_name"]')).toHaveValue("M.Sc. Mathematics");
  await dialog.locator('input[name="qualification_name"]').fill("M.Sc. Applied Mathematics");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("M.Sc. Applied Mathematics")).toBeVisible();

  await dialog.getByRole("button", { name: "Add certification" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Teacher Training");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await dialog.locator('input[name="issue_date"]').fill("2022-05-01");
  await dialog.locator('input[name="expiry_date"]').fill("2021-05-01");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Expiry date cannot be before the issue date.")).toBeVisible();
  await dialog.locator('input[name="expiry_date"]').fill("2027-05-01");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Teacher Training")).toBeVisible();
  const certificationCard = dialog.locator(".staff-record-card").filter({ hasText: "Teacher Training" });
  await certificationCard.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Advanced Teacher Training");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Advanced Teacher Training")).toBeVisible();

  await dialog.getByRole("button", { name: "Add experience" }).click();
  await dialog.locator('input[name="employer_name"]').fill("Previous School");
  await dialog.locator('input[name="record_designation"]').fill("Science Teacher");
  await dialog.locator('input[name="start_date"]').fill("2018-04-01");
  await dialog.locator('input[name="end_date"]').fill("2017-03-31");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("End date must be on or after the start date.")).toBeVisible();
  await dialog.locator('input[name="end_date"]').fill("2021-03-31");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Previous School")).toBeVisible();
  const experienceCard = dialog.locator(".staff-record-card").filter({ hasText: "Previous School" });
  await experienceCard.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="record_designation"]').fill("Senior Science Teacher");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Senior Science Teacher")).toBeVisible();
});

test("staff profile shows only the selected employee's academic assignments and retained history", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 411, school_id: 10, username: "teacher.profile",
    employee_code: "EMP-PRO-01", full_name: "Profile Teacher", designation: "English Teacher",
    mobile_number: "9876543212", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update", "teacher_assignments.read", "class_incharge_assignments.read"]);
  await page.route("**/api/school/teacher-assignments?*", async route => {
    const url = new URL(route.request().url());
    expect(url.searchParams.get("teacher_principal_id")).toBe("411");
    await route.fulfill({ json: { assignments: [
      { id: 1, academic_year_id: 3, class_name: "Class 8", section_name: "A", subject_name: "English", assigned_at: "2026-04-01T00:00:00.000Z", teacher_principal_id: 411, revoked_at: null },
      { id: 2, academic_year_id: 2, class_name: "Class 7", section_name: "B", subject_name: "English", assigned_at: "2025-04-01T00:00:00.000Z", teacher_principal_id: 411, revoked_at: "2026-03-31T00:00:00.000Z" },
    ] } });
  });
  await page.route("**/api/school/class-incharge-assignments?*", async route => {
    const url = new URL(route.request().url());
    expect(url.searchParams.get("teacher_principal_id")).toBe("411");
    await route.fulfill({ json: { assignments: [
      { id: 3, academic_year_id: 3, class_name: "Class 8", section_name: "A", assigned_at: "2026-04-01T00:00:00.000Z", teacher_principal_id: 411, revoked_at: null },
    ] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Profile Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Profile Teacher" });
  await dialog.getByRole("button", { name: "Academic Assignments" }).click();
  await expect(dialog.getByRole("heading", { name: "Subject teaching" })).toBeVisible();
  await expect(dialog.getByText("Class 8 · A", { exact: true }).first()).toBeVisible();
  await expect(dialog.getByText(/Revoked ·/)).toBeVisible();
  await expect(dialog.getByText("2 active")).toBeVisible();
});

test("staff account access keeps role assignments read-only without delegation permissions", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 513, school_id: 10, username: "teacher.role-reader",
    employee_code: "EMP-ROLE-READ", full_name: "Role Reader", designation: "Teacher",
    mobile_number: "9876543213", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "roles.read"];
  const requests: Array<{ method: string; path: string }> = [];
  page.on("request", request => {
    const url = new URL(request.url());
    requests.push({ method: request.method(), path: url.pathname.replace(/^\/api/, "") });
  });
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.route("**/api/school/roles", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { roles: [{ id: 4, role_key: "teacher", name: "Teacher", is_system: true, created_at: "2026-01-01T00:00:00Z" }] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Role changes are not permitted." } });
  });
  await page.route("**/api/school/staff/513/roles", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { roles: [{ role_id: 4, role_key: "teacher", name: "Teacher", is_system: true, assigned_at: "2026-01-01T00:00:00Z" }] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Role changes are not permitted." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Role Reader" }).click();
  const profile = page.getByRole("dialog", { name: "Role Reader" });
  await profile.getByRole("button", { name: "Account & Access" }).click();
  await expect(profile.getByText("Access roles")).toBeVisible();
  await expect(profile.getByText("Teacher", { exact: true })).toBeVisible();
  await expect(profile.getByText("You can view assigned roles but cannot change them.")).toBeVisible();
  await expect(profile.getByRole("button", { name: "Assign", exact: true })).toHaveCount(0);
  await expect(profile.getByRole("button", { name: "Remove Teacher role" })).toHaveCount(0);
  await expect(profile.getByText("You do not have permission to manage individual permission overrides.")).toBeVisible();
  expect(requests.some(item => item.path.includes("permission-delegations") || item.path.includes("permission-overrides") || item.path.endsWith("/school/permissions"))).toBe(false);
  expect(requests.some(item => (item.method === "POST" || item.method === "DELETE") && item.path.includes("/roles"))).toBe(false);
});

test("staff role assignment and removal require the delegated permission and explicit confirmation", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 514, school_id: 10, username: "teacher.role-admin",
    employee_code: "EMP-ROLE-ADMIN", full_name: "Role Admin", designation: "School Admin",
    mobile_number: "9876543212", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const assignedRoles: Array<Record<string, unknown>> = [{
    role_id: 4, role_key: "teacher", name: "Teacher", is_system: true, assigned_at: "2026-01-01T00:00:00Z",
  }];
  let assignCalls = 0;
  let revokeCalls = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "roles.read", "roles.assign"]);
  await page.route("**/api/school/roles", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { roles: [
        { id: 4, role_key: "teacher", name: "Teacher", is_system: true, created_at: "2026-01-01T00:00:00Z" },
        { id: 5, role_key: "librarian", name: "Librarian", is_system: false, created_at: "2026-01-01T00:00:00Z" },
      ] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Unexpected role collection mutation." } });
  });
  await page.route("**/api/school/staff/514/roles**", async route => {
    const url = new URL(route.request().url());
    const path = url.pathname.replace(/^\/api/, "");
    if (path === "/school/staff/514/roles" && route.request().method() === "GET") {
      await route.fulfill({ json: { roles: assignedRoles } });
      return;
    }
    if (path === "/school/staff/514/roles/5" && route.request().method() === "POST") {
      assignCalls += 1;
      assignedRoles.push({ role_id: 5, role_key: "librarian", name: "Librarian", is_system: false, assigned_at: "2026-11-22T00:00:00Z" });
      await route.fulfill({ json: { message: "Role assigned." } });
      return;
    }
    if (path === "/school/staff/514/roles/5" && route.request().method() === "DELETE") {
      revokeCalls += 1;
      const index = assignedRoles.findIndex(item => item.role_id === 5);
      if (index >= 0) assignedRoles.splice(index, 1);
      await route.fulfill({ json: { message: "Role removed." } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: "Unexpected staff role request." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Role Admin" }).click();
  const profile = page.getByRole("dialog", { name: "Role Admin" });
  await profile.getByRole("button", { name: "Account & Access" }).click();
  const roleSelect = profile.locator('button[aria-label="Role to assign to teacher.role-admin"]');
  await roleSelect.click();
  await profile.getByRole("option", { name: "Librarian (custom)" }).click();
  await profile.getByRole("button", { name: "Assign", exact: true }).click();
  await expect(profile.getByText("Role assigned. New permissions apply on the next protected request.")).toBeVisible();
  await expect(profile.getByText("Librarian", { exact: true })).toBeVisible();
  expect(assignCalls).toBe(1);

  await profile.getByRole("button", { name: "Remove Librarian role" }).click();
  const cancelRemoval = page.getByRole("dialog", { name: "Remove staff role?" });
  await cancelRemoval.getByRole("button", { name: "Cancel" }).click();
  expect(revokeCalls).toBe(0);
  await profile.getByRole("button", { name: "Remove Librarian role" }).click();
  const confirmRemoval = page.getByRole("dialog", { name: "Remove staff role?" });
  await confirmRemoval.getByRole("button", { name: "Remove role" }).click();
  await expect(profile.getByText("Role removed. Updated permissions apply on the next protected request.")).toBeVisible();
  await expect(profile.getByText("Librarian", { exact: true })).toHaveCount(0);
  expect(revokeCalls).toBe(1);
});

test("staff permission overrides require a delegated permission, reason and removal confirmation", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 515, school_id: 10, username: "teacher.override-admin",
    employee_code: "EMP-OVERRIDE-ADMIN", full_name: "Override Admin", designation: "School Admin",
    mobile_number: "9876543213", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const overrides: Array<Record<string, unknown>> = [];
  let saveCalls = 0;
  let deleteCalls = 0;
  let submittedPayload: Record<string, unknown> | null = null;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "permissions.override"]);
  await page.route("**/api/school/roles", async route => {
    await route.fulfill({ json: { roles: [
      { id: 4, role_key: "teacher", name: "Teacher", is_system: true, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/515/roles", async route => {
    await route.fulfill({ json: { roles: [
      { role_id: 4, role_key: "teacher", name: "Teacher", is_system: true, assigned_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/permissions", async route => {
    await route.fulfill({ json: { permissions: [
      { permission_key: "staff.attendance.read", description: "View staff attendance", is_delegable: true },
    ] } });
  });
  await page.route("**/api/school/permission-delegations", async route => {
    await route.fulfill({ json: { delegations: [
      { id: 31, school_id: 10, permission_key: "staff.attendance.read", can_assign_to_roles: false, can_override_users: true, granted_by: 2, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/515/permission-overrides**", async route => {
    const method = route.request().method();
    if (method === "GET") {
      await route.fulfill({ json: { overrides } });
      return;
    }
    if (method === "POST") {
      saveCalls += 1;
      submittedPayload = route.request().postDataJSON() as Record<string, unknown>;
      overrides.push({
        id: 71, principal_id: 515, permission_key: "staff.attendance.read",
        effect: "deny", reason: "Temporary attendance access restriction",
        granted_by: 2, delegation_id: 31, created_at: "2026-11-22T00:00:00Z",
      });
      await route.fulfill({ json: { message: "Permission override saved." } });
      return;
    }
    if (method === "DELETE") {
      deleteCalls += 1;
      overrides.splice(0, overrides.length);
      await route.fulfill({ json: { message: "Permission override removed." } });
      return;
    }
    await route.fulfill({ status: 405, json: { message: "Unexpected override method." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Override Admin" }).click();
  const profile = page.getByRole("dialog", { name: "Override Admin" });
  await profile.getByRole("button", { name: "Account & Access" }).click();
  await expect(profile.getByText("Individual permission overrides", { exact: true })).toBeVisible();

  const permissionSelect = profile.locator('button[aria-label="Delegated permission override for teacher.override-admin"]');
  await permissionSelect.click();
  await profile.getByRole("option", { name: /staff.attendance.read/ }).click();
  await profile.getByRole("button", { name: "Save override" }).click();
  await expect(profile.getByText("Choose a permission delegated for individual overrides.")).toHaveCount(0);
  await expect(profile.getByText("Explain why this permission exception is needed.")).toBeVisible();
  await profile.getByLabel("Reason (required)").fill("Temporary attendance access restriction");
  await profile.getByRole("button", { name: "Save override" }).click();
  await expect(profile.getByText("Permission override saved. It takes effect on the next protected request.")).toBeVisible();
  await expect(profile.getByText("staff.attendance.read", { exact: true })).toBeVisible();
  expect(saveCalls).toBe(1);
  expect(submittedPayload).toMatchObject({
    permission_key: "staff.attendance.read",
    effect: "deny",
    reason: "Temporary attendance access restriction",
    delegation_id: 31,
  });

  await profile.getByRole("button", { name: "Remove staff.attendance.read override" }).click();
  const cancelRemoval = page.getByRole("dialog", { name: "Remove permission override?" });
  await cancelRemoval.getByRole("button", { name: "Cancel" }).click();
  expect(deleteCalls).toBe(0);
  await profile.getByRole("button", { name: "Remove staff.attendance.read override" }).click();
  const confirmRemoval = page.getByRole("dialog", { name: "Remove permission override?" });
  await confirmRemoval.getByRole("button", { name: "Remove override" }).click();
  await expect(profile.getByText("Permission override removed. Effective permissions update on the next protected request.")).toBeVisible();
  await expect(profile.getByText("staff.attendance.read", { exact: true })).toHaveCount(0);
  expect(deleteCalls).toBe(1);
});

test("staff profile history is permission-gated and renders staff audit events", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 512, school_id: 10, username: "teacher.history",
    employee_code: "EMP-HIS-01", full_name: "History Teacher", designation: "Maths Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update", "school.audit.read"]);
  await page.route("**/api/school/staff/512/audit-events", async route => {
    await route.fulfill({ json: { events: [
      { id: 9, action: "staff.profile.updated", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { employment_status: "active" }, created_at: "2026-11-20T10:00:00.000Z" },
      { id: 8, action: "staff.account.created", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { employee_code: "EMP-HIS-01" }, created_at: "2026-04-01T09:00:00.000Z" },
    ] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "History Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "History Teacher" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  await expect(dialog.getByText("Account created")).toBeVisible();
  await expect(dialog.getByText("school.admin").first()).toBeVisible();
});

test("staff attendance supports search, individual and atomic bulk marking, and monthly summaries", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 512, school_id: 10, username: "teacher.attendance",
    employee_code: "EMP-ATT-01", full_name: "Attendance Teacher", designation: "Science Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, [
    "users.read", "staff.attendance.read", "staff.attendance.mark",
    "staff.attendance.update_same_day", "staff.attendance.correct_past",
    "staff.leave.read", "staff.holidays.read",
  ]);
  await page.route("**/api/school/staff-leave/requests?*", async route => {
    const url = new URL(route.request().url());
    if (!url.searchParams.has("from_date")) { await route.fallback(); return; }
    expect(url.searchParams.get("status")).toBe("approved");
    expect(url.searchParams.get("from_date")).toMatch(/^\d{4}-\d{2}-01$/);
    expect(url.searchParams.get("to_date")).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    await route.fulfill({ json: { data: [{
      id: 71, staff_principal_id: 512, staff_name: "Attendance Teacher", employee_code: "EMP-ATT-01",
      leave_type_id: 1, leave_type_name: "Casual Leave", start_date: url.searchParams.get("from_date"),
      end_date: url.searchParams.get("from_date"), duration_type: "full_day", reason: "Approved leave",
      status: "approved", created_at: "2026-11-01T00:00:00.000Z",
    }] } });
  });
  let savedStatus = "";
  let bulkCreated = 0;
  const dailyRecords = [
    { id: null as number | null, school_id: 10, staff_principal_id: 512, employee_code: "EMP-ATT-01", full_name: "Attendance Teacher", designation: "Science Teacher", attendance_date: "", status: null as string | null },
    { id: null as number | null, school_id: 10, staff_principal_id: 513, employee_code: "EMP-ATT-02", full_name: "Bulk Attendance Teacher", designation: "English Teacher", attendance_date: "", status: null as string | null },
  ];
  await page.route("**/api/school/staff-attendance**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname.endsWith("/month")) {
      await route.fulfill({ json: { month: url.searchParams.get("month"), data: [{
        staff_principal_id: 512, employee_code: "EMP-ATT-01", full_name: "Attendance Teacher",
        designation: "Science Teacher", present_days: 1, absent_days: 1, late_days: 1, half_days: 0,
        records: [{ id: 7, date: "2026-11-03", status: "present" }, { id: 8, date: "2026-11-04", status: "late" }, { id: 9, date: "2026-11-05", status: "absent" }],
      }] } });
      return;
    }
    if (request.method() === "GET") {
      for (const item of dailyRecords) item.attendance_date = url.searchParams.get("date") ?? "";
      await route.fulfill({ json: { date: url.searchParams.get("date"), data: dailyRecords } });
      return;
    }
    if (request.method() === "POST" && url.pathname.endsWith("/bulk")) {
      const input = request.postDataJSON() as { staff_principal_ids: number[]; status: string };
      bulkCreated = input.staff_principal_ids.length;
      for (const item of dailyRecords) if (input.staff_principal_ids.includes(item.staff_principal_id)) {
        item.id = 23;
        item.status = input.status;
      }
      await route.fulfill({ status: 201, json: { created: bulkCreated } });
      return;
    }
    if (request.method() === "POST") {
      const input = request.postDataJSON() as { staff_principal_id: number; status: string; attendance_date: string };
      savedStatus = input.status;
      const item = dailyRecords.find(record => record.staff_principal_id === input.staff_principal_id);
      if (item) { item.id = 22; item.status = input.status; item.attendance_date = input.attendance_date; }
      await route.fulfill({ status: 201, json: { data: { ...item, id: 22 } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: "Unmocked staff attendance request" } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff attendance" })).toBeVisible();
  await page.getByRole("button", { name: "Mark", exact: true }).first().click();
  await expect(page.getByText("Select an attendance status.")).toBeVisible();
  const firstRow = page.getByRole("row").filter({ hasText: "Attendance Teacher" }).first();
  await firstRow.getByLabel("Attendance status for Attendance Teacher").selectOption("present");
  await firstRow.getByRole("button", { name: "Mark", exact: true }).click();
  await expect.poll(() => savedStatus).toBe("present");
  await expect(page.getByText("Staff attendance marked.")).toBeVisible();
  await page.getByLabel("Search staff attendance").fill("Bulk Attendance");
  await expect(page.getByText("Attendance Teacher", { exact: true })).toHaveCount(0);
  await page.getByLabel("Bulk attendance status").selectOption("absent");
  await page.getByRole("button", { name: "Mark 1 unmarked staff", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Confirm bulk staff attendance" })).toBeVisible();
  await page.getByRole("button", { name: "Mark 1 staff", exact: true }).click();
  await expect.poll(() => bulkCreated).toBe(1);
  await expect(page.getByText("Attendance marked for 1 staff members.")).toBeVisible();
  await page.getByLabel("Search staff attendance").fill("");
  await page.getByLabel("Filter staff attendance status").selectOption("absent");
  await expect(page.getByText("Bulk Attendance Teacher")).toBeVisible();
  await expect(page.getByText("Attendance Teacher", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Monthly summary" }).click();
  await expect(page.getByText("Attendance Teacher").first()).toBeVisible();
  await expect(page.getByText("View daily entries (3)")).toBeVisible();
  await expect(page.getByText("Calendar context · not included in attendance totals")).toBeVisible();
  await expect(page.getByText("Approved leave", { exact: true })).toBeVisible();
  await expect(page.getByText(/Attendance Teacher · Casual Leave/)).toBeVisible();
  await expect(page.getByText("School holidays", { exact: true })).toBeVisible();
  await expect(page.getByText(/School Foundation Day/)).toBeVisible();
});

test("staff leave request field errors clear as each required field is corrected", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.validation",
    employee_code: "EMP-VALID-01", full_name: "Validation Teacher", designation: "Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.manage_requests"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  await page.getByRole("button", { name: "Submit leave request" }).click();
  await expect(page.getByText("Select one employee.")).toBeVisible();
  await expect(page.getByText("Select a leave type.")).toBeVisible();
  await expect(page.getByText("Select a start date.")).toBeVisible();
  await expect(page.getByText("Select an end date.")).toBeVisible();
  await expect(page.getByText("Enter the reason for leave.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Staff member", exact: true })).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("button", { name: "Staff member", exact: true })).toHaveAttribute("aria-describedby", "leave-staff-error");
  await expect(page.getByLabel("Leave type")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Leave start date")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Leave end date")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Reason")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Reason")).toHaveAttribute("aria-describedby", "leave-reason-error");

  await page.getByRole("button", { name: "Staff member", exact: true }).click();
  await page.getByRole("option", { name: /Validation Teacher/ }).click();
  await expect(page.getByText("Select one employee.")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Staff member", exact: true })).not.toHaveAttribute("aria-invalid", "true");

  await page.getByLabel("Leave type").click();
  await page.getByRole("option", { name: /Casual Leave/ }).click();
  await expect(page.getByText("Select a leave type.")).toHaveCount(0);

  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: /25/ }).click();
  await expect(page.getByText("Select a start date.")).toHaveCount(0);
  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: "Wednesday, November 25, 2026" }).click();
  await expect(page.getByText("Select an end date.")).toHaveCount(0);

  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: "Thursday, November 26, 2026" }).click();
  await page.getByRole("button", { name: "Submit leave request" }).click();
  await expect(page.getByText("End date must be on or after the start date.")).toBeVisible();
  await expect(page.getByLabel("Leave end date")).toHaveAttribute("aria-describedby", "leave-end-date-error");

  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: "Tuesday, November 24, 2026" }).click();
  await expect(page.getByText("End date must be on or after the start date.")).toHaveCount(0);

  await page.getByLabel("Leave duration").click();
  await page.getByRole("option", { name: "Half day" }).click();
  await page.getByRole("button", { name: "Submit leave request" }).click();
  await expect(page.getByText("Half-day leave must cover one date only.")).toBeVisible();
  await expect(page.getByLabel("Leave duration")).toHaveAttribute("aria-describedby", "leave-duration-error");

  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: "Tuesday, November 24, 2026" }).click();
  await expect(page.getByText("Half-day leave must cover one date only.")).toHaveCount(0);

  await page.getByLabel("Reason").fill("Personal appointment");
  await expect(page.getByText("Enter the reason for leave.")).toHaveCount(0);
});

test("staff leave request submits a valid half-day application with its selected session", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.submit",
    employee_code: "EMP-SUBMIT-01", full_name: "Submission Teacher", designation: "Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.manage_requests"];
  let submitted: Record<string, unknown> | undefined;
  page.on("request", request => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/school/staff-leave/requests") {
      submitted = request.postDataJSON() as Record<string, unknown>;
    }
  });
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  await page.getByRole("button", { name: "Staff member", exact: true }).click();
  await page.getByRole("option", { name: /Submission Teacher/ }).click();
  await page.getByLabel("Leave type").click();
  await page.getByRole("option", { name: /Casual Leave/ }).click();
  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: "Tuesday, November 24, 2026" }).click();
  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: "Tuesday, November 24, 2026" }).click();
  await page.getByLabel("Leave duration").click();
  await page.getByRole("option", { name: "Half day" }).click();
  const leaveDuration = page.getByRole("button", { name: "Leave duration" });
  await leaveDuration.press("Escape");
  await expect(leaveDuration).toHaveAttribute("aria-expanded", "false");
  const halfDaySession = page.getByRole("button", { name: "Half-day session" });
  await halfDaySession.click();
  await page.getByRole("listbox", { name: "Half-day session" }).getByRole("option", { name: "Second half" }).click();
  await expect(halfDaySession).toContainText("Second half");
  await page.getByLabel("Reason").fill("Medical appointment");

  await page.getByRole("button", { name: "Submit leave request" }).click();
  const submittedApplication = page.locator("article").filter({ hasText: "Medical appointment" });
  await expect(submittedApplication).toBeVisible();
  await expect(submittedApplication).toContainText("pending");
  expect(submitted).toMatchObject({
    staff_principal_id: 501,
    leave_type_id: 1,
    start_date: "2026-11-24",
    end_date: "2026-11-24",
    duration_type: "half_day",
    half_day_session: "second_half",
    reason: "Medical appointment",
  });
});

test("staff leave rejection requires a comment before the pending request can be rejected", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.decide"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByText("Family commitment")).toBeVisible();

  await page.getByRole("button", { name: "Reject" }).click();
  const decision = page.getByRole("dialog", { name: "Reject leave request" });
  await decision.getByRole("button", { name: "Confirm rejection" }).click();
  await expect(decision.getByRole("alert")).toHaveText("A comment is required when rejecting a request.");
  await expect(decision.getByLabel(/Decision comment/)).toHaveAttribute("aria-invalid", "true");
  await expect(decision.getByLabel(/Decision comment/)).toHaveAttribute("aria-describedby", "leave-decision-comment-error");

  await decision.getByLabel(/Decision comment/).fill("Insufficient leave information.");
  await expect(decision.getByRole("alert")).toHaveCount(0);
  await expect(decision.getByLabel(/Decision comment/)).not.toHaveAttribute("aria-invalid", "true");
  await decision.getByRole("button", { name: "Confirm rejection" }).click();
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();
});

test("staff leave cancellation requires explicit confirmation", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.manage_requests"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByText("Family commitment")).toBeVisible();

  await page.getByRole("button", { name: "Cancel request" }).click();
  const confirmation = page.getByRole("dialog", { name: "Cancel leave request?" });
  await expect(confirmation).toBeVisible();
  await confirmation.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByText("Family commitment")).toBeVisible();

  await page.getByRole("button", { name: "Cancel request" }).click();
  const secondConfirmation = page.getByRole("dialog", { name: "Cancel leave request?" });
  await secondConfirmation.getByRole("button", { name: "Cancel request", exact: true }).click();
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();
});


test("read-only holiday permission shows the calendar without edit or inactive controls", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "holiday.reader",
    employee_code: "EMP-HOL-READ", full_name: "Holiday Reader", designation: "Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.holidays.read"];
  let holidayListURL: URL | undefined;
  page.on("request", request => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname.endsWith("/school/holidays")) holidayListURL = url;
  });
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Holiday calendar" }).click();
  await expect(page.getByText("School Foundation Day")).toBeVisible();
  expect(holidayListURL).toBeDefined();
  expect(holidayListURL?.searchParams.get("include_inactive")).toBeNull();
  await expect(page.getByRole("button", { name: "Edit", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Deactivate", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Reactivate", exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Save holiday" })).toHaveCount(0);
});

test("school holiday deactivation and reactivation require explicit confirmation", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "holiday.admin",
    employee_code: "EMP-HOL-01", full_name: "Holiday Admin", designation: "School Admin",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.holidays.manage"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Holiday calendar" }).click();
  await expect(page.getByText("School Foundation Day")).toBeVisible();

  await page.getByRole("button", { name: "Deactivate" }).click();
  const deactivate = page.getByRole("dialog", { name: "Deactivate school holiday?" });
  await deactivate.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByText("School Foundation Day")).toBeVisible();

  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByRole("button", { name: "Deactivate" }).click();
  const confirmDeactivate = page.getByRole("dialog", { name: "Deactivate school holiday?" });
  await confirmDeactivate.getByRole("button", { name: "Deactivate holiday" }).click();
  await expect(page.getByText("Inactive", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Inactive", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await page.getByRole("button", { name: "Reactivate" }).click();
  const reactivate = page.getByRole("dialog", { name: "Reactivate school holiday?" });
  await reactivate.getByRole("button", { name: "Reactivate holiday" }).click();
  await expect(page.getByText("Holiday reactivated.")).toBeVisible();
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Inactive", { exact: true })).toHaveCount(0);
});

test("leave type save errors stay inside their own form when switching workspace panes", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.types.admin",
    employee_code: "EMP-TYPE-01", full_name: "Leave Types Admin", designation: "School Admin",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.read", "staff.leave.manage_types", "staff.holidays.read", "staff.holidays.manage"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.route("**/api/school/staff-leave/types", async route => {
    if (route.request().method() === "POST") {
      await route.fulfill({ status: 409, json: { message: "A leave type with this name already exists." } });
      return;
    }
    await route.fallback();
  });

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Leave types" }).click();
  await page.getByLabel(/Leave type name/).fill("Casual Leave");
  await page.getByRole("button", { name: "Add leave type" }).click();
  await expect(page.getByRole("alert")).toHaveText("A leave type with this name already exists.");

  await page.getByRole("button", { name: "Holiday calendar" }).click();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await page.getByRole("button", { name: "Leave types" }).click();
  await expect(page.getByRole("alert")).toHaveText("A leave type with this name already exists.");
  await page.getByLabel(/Leave type name/).fill("Updated Casual Leave");
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("holiday save errors clear when the holiday details are corrected", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "holiday.validation",
    employee_code: "EMP-HOL-VALID", full_name: "Holiday Validation Admin", designation: "School Admin",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.holidays.read", "staff.holidays.manage"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.route("**/api/school/holidays", async route => {
    if (route.request().method() === "POST") {
      await route.fulfill({ status: 409, json: { message: "A holiday already exists on this date." } });
      return;
    }
    await route.fallback();
  });

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Holiday calendar" }).click();
  await page.getByRole("button", { name: "Save holiday" }).click();
  await expect(page.getByText("Select the holiday date.")).toBeVisible();
  await expect(page.getByText("Enter a holiday name (at least 2 characters).")).toBeVisible();
  await expect(page.getByLabel("Holiday date")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Holiday date")).toHaveAttribute("aria-describedby", "holiday-date-error");
  await expect(page.getByLabel(/Holiday name/)).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel(/Holiday name/)).toHaveAttribute("aria-describedby", "holiday-name-error");

  await page.getByLabel("Holiday date").click();
  await page.getByRole("dialog", { name: "Choose holiday date" }).getByRole("button", { name: "Wednesday, November 25, 2026" }).click();
  await expect(page.getByText("Select the holiday date.")).toHaveCount(0);
  await expect(page.getByLabel("Holiday date")).not.toHaveAttribute("aria-invalid", "true");
  await page.getByLabel(/Holiday name/).fill("School Event");
  await expect(page.getByText("Enter a holiday name (at least 2 characters).")).toHaveCount(0);
  await expect(page.getByLabel(/Holiday name/)).not.toHaveAttribute("aria-invalid", "true");
  await expect(page.getByRole("dialog", { name: "Choose holiday date" })).toHaveCount(0);
  await page.getByLabel(/Holiday name/).fill("School Event");
  await page.getByRole("button", { name: "Save holiday" }).click();
  await expect(page.getByRole("alert")).toHaveText("A holiday already exists on this date.");

  await page.getByLabel(/Holiday name/).fill("Updated School Event");
  await expect(page.getByRole("alert")).toHaveCount(0);
});

test("write-only leave permissions do not expose list views without leave-read access", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.writeonly",
    employee_code: "EMP-WRITE-01", full_name: "Leave Write Only", designation: "Staff Manager",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const permissions = ["users.read", "staff.leave.manage_types", "staff.leave.manage_requests", "staff.leave.decide"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  const restricted = page.getByRole("button", { name: /Leave Restricted/ });
  await expect(restricted).toBeDisabled();
  await expect(restricted).toHaveAttribute("title", "Leave read permission required to open this workspace");
  await expect(page.getByRole("button", { name: "Leave & Holidays" })).toHaveCount(0);
});
