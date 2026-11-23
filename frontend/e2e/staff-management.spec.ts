import { readFile } from "node:fs/promises";
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
  const substitutions: Array<Record<string, unknown>> = [];
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
    if (path === "/school/staff-attendance/month" && method === "GET") {
      await route.fulfill({ json: { month: url.searchParams.get("month"), data: [{ staff_principal_id: 501, employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher", present_days: 18, absent_days: 1, late_days: 2, half_days: 0, records: [] }] } });
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
    if (path === "/school/staff-substitutions" && method === "GET") {
      const from = url.searchParams.get("from_date") ?? "";
      const to = url.searchParams.get("to_date") ?? "";
      const staffID = Number(url.searchParams.get("staff_principal_id") ?? 0);
      const data = substitutions.filter(item => String(item.cover_date) >= from && String(item.cover_date) <= to &&
        (!staffID || item.original_teacher_principal_id === staffID || item.substitute_teacher_principal_id === staffID));
      await route.fulfill({ json: { substitutions: data } });
      return;
    }
    if (path === "/school/staff-substitutions" && method === "POST") {
      const payload = request.postDataJSON() as Record<string, unknown>;
      const original = staffRows.find(item => item.principal_id === payload.original_teacher_principal_id) ?? {};
      const substitute = staffRows.find(item => item.principal_id === payload.substitute_teacher_principal_id) ?? {};
      const saved = {
        ...payload, id: substitutions.length + 91, status: "active",
        original_teacher_name: original.full_name ?? "Original Teacher",
        substitute_teacher_name: substitute.full_name ?? "Substitute Teacher",
        section_name: "A", class_name: "Class 6", subject_name: "Mathematics",
        period_number: 2, period_name: "Period 2", starts_at: "09:40", ends_at: "10:20", room: "Room 12",
        created_by_principal_id: staffUser.principal_id, created_at: "2026-11-22T00:00:00.000Z",
      };
      substitutions.push(saved);
      await route.fulfill({ status: 201, json: { substitution: saved } });
      return;
    }
    const substitutionCancel = path.match(/^\/school\/staff-substitutions\/(\d+)\/cancel$/);
    if (substitutionCancel && method === "POST") {
      const target = substitutions.find(item => item.id === Number(substitutionCancel[1]));
      if (!target) { await route.fulfill({ status: 404, json: { message: "Substitution not found" } }); return; }
      target.status = "cancelled";
      target.cancelled_by_principal_id = staffUser.principal_id;
      target.cancelled_at = "2026-11-22T00:05:00.000Z";
      await route.fulfill({ json: { message: "Substitution cancelled" } });
      return;
    }
    if (path === "/school/timetable/workload-report" && method === "GET") {
      const workloads = staffRows.map((item) => ({
        teacher_principal_id: item.principal_id,
        teacher_name: item.full_name ?? item.username,
        employee_code: item.employee_code ?? "EMP",
        designation: item.principal_id === 501 ? '   =HYPERLINK("x")' : item.designation ?? "Teacher",
        employment_status: item.employment_status ?? "active",
        login_status: item.status ?? "active",
        day_of_week: item.principal_id === 501 ? 1 : 0,
        scheduled_periods: item.principal_id === 501 ? 4 : 0,
        distinct_sections: item.principal_id === 501 ? 2 : 0,
        distinct_subjects: item.principal_id === 501 ? 2 : 0,
        first_period: item.principal_id === 501 ? 1 : null,
        last_period: item.principal_id === 501 ? 4 : null,
      }));
      await route.fulfill({ json: { workloads } });
      return;
    }
    if (path === "/school/teacher-assignments" && method === "GET") {
      await route.fulfill({ json: { assignments: [
        { id: 71, academic_year_id: 2026, section_id: 301, section_name: "A", class_name: "Class 6", subject_id: 9, subject_name: "Mathematics", teacher_principal_id: 501, teacher_username: "teacher.active", assigned_at: "2026-04-01T00:00:00.000Z", revoked_at: null },
        { id: 72, academic_year_id: 2026, section_id: 302, section_name: "B", class_name: "Class 7", subject_id: 10, subject_name: "Science", teacher_principal_id: 501, teacher_username: "teacher.active", assigned_at: "2026-04-01T00:00:00.000Z", revoked_at: "2026-06-01T00:00:00.000Z" },
      ] } });
      return;
    }
    if (path === "/school/class-incharge-assignments" && method === "GET") {
      await route.fulfill({ json: { assignments: [
        { id: 73, academic_year_id: 2026, section_id: 301, section_name: "A", class_name: "Class 6", teacher_principal_id: 501, teacher_username: "teacher.active", assigned_at: "2026-04-01T00:00:00.000Z", revoked_at: null },
      ] } });
      return;
    }
    if (path === "/school/timetable/current-year" && method === "GET") {
      await route.fulfill({ json: { academic_year: { id: 2026, name: "2026–27", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true } } });
      return;
    }
    if (path === "/school/timetable/periods" && method === "GET") {
      await route.fulfill({ json: { periods: [{ id: 4, school_id: 10, academic_year_id: 2026, period_number: 2, name: "Period 2", starts_at: "09:40", ends_at: "10:20", is_break: false }] } });
      return;
    }
    if (path === "/school/timetable/entries" && method === "GET") {
      const teacherFilter = url.searchParams.get("teacher_principal_id");
      if (teacherFilter && teacherFilter !== "501") {
        await route.fulfill({ status: 400, json: { message: "A teacher-specific filter is required." } });
        return;
      }
      await route.fulfill({ json: { entries: [{
        id: 81, school_id: 10, academic_year_id: 2026, section_id: 301, section_name: "A", class_name: "Class 6",
        subject_id: 9, subject_name: "Mathematics", teacher_principal_id: 501, teacher_username: "teacher.workload",
        teacher_subject_assignment_id: 12, period_id: 4, period_number: 2, period_name: "Period 2",
        starts_at: "09:40", ends_at: "10:20", day_of_week: 1, room: "Room 12",
      }] } });
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
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 111, school_id: 10, username: "teacher.overview",
      employee_code: "EMP-OV-01", full_name: "Overview Teacher", designation: "Teacher",
      mobile_number: "9876543210", employment_status: "active", profile_complete: true,
      status: "disabled", must_change_password: true, created_at: "2026-04-01T00:00:00.000Z",
    },
    {
      id: 2, principal_id: 112, school_id: 10, username: "staff.incomplete",
      employee_code: "EMP-OV-02", full_name: "Incomplete Profile", designation: "Office Assistant",
      mobile_number: "9876543211", employment_status: "inactive", profile_complete: false,
      status: "disabled", must_change_password: false, created_at: "2026-04-02T00:00:00.000Z",
    },
  ];
  await signInAndOpenStaff(page, rows);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "People at a glance" })).toBeVisible();
  await expect(page.getByText("Total staff")).toBeVisible();
  await expect(page.locator(".staff-overview-cards article").nth(0).locator("strong")).toHaveText("2");
  await expect(page.locator(".staff-overview-cards article").nth(1).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-overview-cards article").nth(2).locator("strong")).toHaveText("0");
  await expect(page.locator(".staff-overview-cards article").nth(3).locator("strong")).toHaveText("1");
  const readiness = page.locator(".staff-readiness-list");
  await expect(readiness).toContainText("Active employees with disabled login");
  await expect(readiness.locator("li").nth(0).locator("strong")).toHaveText("1");
  await expect(readiness.locator("li").nth(1).locator("strong")).toHaveText("1");
  await expect(readiness.locator("li").nth(2).locator("strong")).toHaveText("1");
  await expect(page.getByRole("button", { name: /Attendance/ })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Leave/ })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Reports", exact: true })).toBeEnabled();
});


test("staff directory filters independently narrow employment, login account and profile completeness", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 611, school_id: 10, username: "active.enabled", employee_code: "EMP-F-01", full_name: "Active Enabled", designation: "Teacher", mobile_number: "9876543210", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 612, school_id: 10, username: "active.disabled", employee_code: "EMP-F-02", full_name: "Active Disabled", designation: "Teacher", mobile_number: "9876543211", employment_status: "active", profile_complete: true, status: "disabled", must_change_password: true },
    { id: 3, principal_id: 613, school_id: 10, username: "inactive.disabled", employee_code: "EMP-F-03", full_name: "Inactive Disabled", designation: "Office Assistant", mobile_number: "9876543212", employment_status: "inactive", profile_complete: true, status: "disabled", must_change_password: false },
    { id: 4, principal_id: 614, school_id: 10, username: "profile.pending", employee_code: "EMP-F-04", full_name: "Profile Pending", designation: "Staff", mobile_number: "9876543213", employment_status: "active", profile_complete: false, status: "disabled", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("4 of 4 staff members")).toBeVisible();

  await page.getByRole("button", { name: "Filter by employment status" }).click();
  await page.getByRole("option", { name: "Active", exact: true }).click();
  await expect(page.getByText("3 of 4 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Enabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Disabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Inactive Disabled" })).toHaveCount(0);

  await page.getByRole("button", { name: "Filter by login account status" }).click();
  await page.getByRole("option", { name: "Disabled", exact: true }).click();
  await expect(page.getByText("2 of 4 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Disabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Enabled" })).toHaveCount(0);

  await page.getByRole("button", { name: "Filter by profile completeness" }).click();
  await page.getByRole("option", { name: "Incomplete", exact: true }).click();
  await expect(page.getByText("1 of 4 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
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

test("disabling a staff account requires explicit confirmation and refreshes its status", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 516, school_id: 10, username: "disable.teacher",
    employee_code: "EMP-DISABLE-01", full_name: "Disable Teacher", designation: "Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let disableCalls = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.disable"]);
  await page.route("**/api/school/staff/516", async route => {
    if (route.request().method() === "DELETE") {
      disableCalls += 1;
      rows[0].status = "disabled";
      await route.fulfill({ json: { message: "Staff account disabled." } });
      return;
    }
    await route.fallback();
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  const staffRow = page.locator("tr").filter({ hasText: "disable.teacher" });
  await expect(staffRow).toBeVisible();
  await staffRow.getByRole("button", { name: "Disable disable.teacher" }).click();
  const cancelled = page.getByRole("dialog", { name: "Disable staff account?" });
  await expect(cancelled).toContainText("active sessions will be revoked");
  await cancelled.getByRole("button", { name: "Cancel", exact: true }).click();
  expect(disableCalls).toBe(0);
  await expect(staffRow).toContainText("active");

  await staffRow.getByRole("button", { name: "Disable disable.teacher" }).click();
  const confirmation = page.getByRole("dialog", { name: "Disable staff account?" });
  await confirmation.getByRole("button", { name: "Disable account" }).click();
  await expect(page.getByText("disable.teacher has been disabled and active sessions revoked.")).toBeVisible();
  expect(disableCalls).toBe(1);
  await expect(page.locator("tr").filter({ hasText: "disable.teacher" })).toContainText("disabled");
  await expect(page.getByRole("button", { name: "Disable disable.teacher" })).toHaveCount(0);
});

test("staff password reset validates a strong temporary password and requires next sign-in change", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 517, school_id: 10, username: "reset.teacher",
    employee_code: "EMP-RESET-01", full_name: "Reset Teacher", designation: "Teacher",
    mobile_number: "9876543216", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let resetCalls = 0;
  let submittedPayload: Record<string, unknown> | null = null;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.password_reset"]);
  await page.route("**/api/school/staff/517/password/reset", async route => {
    if (route.request().method() !== "POST") { await route.fallback(); return; }
    resetCalls += 1;
    submittedPayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({ json: { message: "Password reset." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Reset password for reset.teacher" }).click();
  const resetDialog = page.getByRole("dialog", { name: "Reset staff password" });
  const password = resetDialog.getByLabel("New temporary password");
  await password.fill("short");
  await resetDialog.getByRole("button", { name: "Reset password", exact: true }).click();
  await expect(resetDialog.getByText("Password must contain at least 12 characters.")).toBeVisible();
  expect(resetCalls).toBe(0);

  await password.fill("StrongTempPass!2026");
  await expect(resetDialog.getByText("Password must contain at least 12 characters.")).toHaveCount(0);
  await resetDialog.getByRole("button", { name: "Reset password", exact: true }).click();
  await expect(page.getByText("Password reset. The staff member must change it at next sign-in.")).toBeVisible();
  expect(resetCalls).toBe(1);
  expect(submittedPayload).toEqual({ new_password: "StrongTempPass!2026" });
});

test("enabling a disabled staff account validates a temporary password and enforces next sign-in change", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 518, school_id: 10, username: "enable.teacher",
    employee_code: "EMP-ENABLE-01", full_name: "Enable Teacher", designation: "Teacher",
    mobile_number: "9876543217", employment_status: "active", profile_complete: true,
    status: "disabled", must_change_password: false,
  }];
  let enableCalls = 0;
  let submittedPayload: Record<string, unknown> | null = null;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.update"]);
  await page.route("**/api/school/staff/518/enable", async route => {
    if (route.request().method() !== "POST") { await route.fallback(); return; }
    enableCalls += 1;
    submittedPayload = route.request().postDataJSON() as Record<string, unknown>;
    rows[0].status = "active";
    rows[0].must_change_password = true;
    await route.fulfill({ json: { message: "Staff account enabled." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  const staffRow = page.locator("tr").filter({ hasText: "enable.teacher" });
  await expect(staffRow).toBeVisible();
  await staffRow.getByRole("button", { name: "Enable enable.teacher" }).click();
  const enableDialog = page.getByRole("dialog", { name: "Enable staff account" });
  const password = enableDialog.getByLabel("Temporary password");
  await password.fill("short");
  await enableDialog.getByRole("button", { name: "Enable account", exact: true }).click();
  await expect(enableDialog.getByText("Password must contain at least 12 characters.")).toBeVisible();
  expect(enableCalls).toBe(0);

  await password.fill("EnableTempPass!2026");
  await enableDialog.getByRole("button", { name: "Enable account", exact: true }).click();
  await expect(page.getByText("Staff account enabled. A password change is required at next sign-in.")).toBeVisible();
  expect(enableCalls).toBe(1);
  expect(submittedPayload).toEqual({ temporary_password: "EnableTempPass!2026" });
  await expect(page.locator("tr").filter({ hasText: "enable.teacher" })).toContainText("active");
  await expect(page.getByText("Change required")).toBeVisible();
  await expect(page.getByRole("button", { name: "Enable enable.teacher" })).toHaveCount(0);
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


test("staff workload requests only the selected teacher timetable and explains unscheduled availability", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "teacher.workload",
    employee_code: "EMP-WORK-01", full_name: "Workload Teacher", designation: "Mathematics Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "timetable.read"]);
  await page.locator(".staff-workspace-tabs").getByRole("button", { name: "Staff Directory" }).click();
  await page.getByRole("button", { name: "Workload Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Workload Teacher" });
  await dialog.getByRole("button", { name: "Workload & Timetable" }).click();
  await expect(dialog.getByRole("heading", { name: "Teaching workload" })).toBeVisible();
  await expect(dialog.getByText("Scheduled periods", { exact: true })).toBeVisible();
  await expect(dialog.locator(".staff-workload-summary article").first().locator("strong")).toHaveText("1");
  await expect(dialog.getByText("Busiest day")).toBeVisible();
  await expect(dialog.getByText("Weekly workload report")).toBeVisible();
  await expect(dialog.locator(".staff-workload-table tbody tr").first()).toContainText("Monday");
  await expect(dialog.locator(".staff-workload-table tfoot")).toContainText("Weekly total");
  await expect(dialog.getByText("Mathematics")).toBeVisible();
  await expect(dialog.locator(".staff-workload-table")).toBeVisible();
  await expect(dialog.getByText("Unscheduled", { exact: true }).first()).toBeVisible();
  await expect(dialog.getByText(/do not confirm teacher availability/)).toBeVisible();
});


test("staff substitution assigns a dated cover, validates the timetable weekday and retains cancellation history", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "teacher.original",
      employee_code: "EMP-ORIGINAL", full_name: "Original Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.substitute",
      employee_code: "EMP-SUBSTITUTE", full_name: "Substitute Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543216", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
  ];
  const permissions = ["users.read", "staff.substitutions.read", "staff.substitutions.manage", "timetable.read", "staff.leave.read"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Substitutions" }).click();
  await expect(page.getByRole("heading", { name: "Substitute Teacher Coverage" })).toBeVisible();

  await page.getByRole("button", { name: "Recurring timetable lesson" }).click();
  await page.getByRole("option", { name: /Monday · Class 6 A · Mathematics/ }).click();
  await expect(page.getByText("Original teacher: Original Teacher")).toBeVisible();

  await page.getByLabel("Substitution cover date").click();
  const dateDialog = page.getByRole("dialog", { name: "Choose substitution cover date" });
  await dateDialog.getByRole("button", { name: "Tuesday, November 24, 2026" }).click();
  await page.getByRole("button", { name: "Substitute staff member" }).click();
  await page.getByRole("option", { name: /Substitute Teacher/ }).click();
  await page.getByLabel("Reason").fill("Original teacher unavailable");
  await page.getByRole("button", { name: "Assign substitute" }).click();
  await expect(page.getByText("This date is not a Monday; choose a matching date.")).toBeVisible();

  await page.getByLabel("Substitution cover date").click();
  await page.getByRole("dialog", { name: "Choose substitution cover date" }).getByRole("button", { name: "Monday, November 23, 2026" }).click();
  await expect(page.getByText("This date is not a Monday; choose a matching date.")).toHaveCount(0);
  await page.getByRole("button", { name: "Assign substitute" }).click();

  const record = page.locator(".staff-substitution-record");
  await expect(record).toContainText("Class 6");
  await expect(record).toContainText("Original Teacher");
  await expect(record).toContainText("Substitute Teacher");
  await expect(record).toContainText("Scheduled");
  await record.getByRole("button", { name: "Cancel cover" }).click();
  await page.getByRole("button", { name: "Cancel cover" }).last().click();
  await expect(record).toContainText("Cancelled");
  await expect(record.getByRole("button", { name: "Cancel cover" })).toHaveCount(0);
});


test("completed substitute coverage remains visible without a cancellation action", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "teacher.original",
      employee_code: "EMP-ORIGINAL", full_name: "Original Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.substitute",
      employee_code: "EMP-SUBSTITUTE", full_name: "Substitute Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543216", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.substitutions.read", "staff.substitutions.manage"]);
  await page.route("**/api/school/staff-substitutions*", async route => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }
    await route.fulfill({
      json: {
        substitutions: [{
          id: 77, timetable_entry_id: 700, cover_date: "2026-11-16",
          original_teacher_principal_id: 501, original_teacher_name: "Original Teacher",
          substitute_teacher_principal_id: 502, substitute_teacher_name: "Substitute Teacher",
          reason: "Completed cover", status: "active", section_name: "A", class_name: "Class 6",
          subject_name: "Mathematics", period_number: 2, period_name: "Period 2",
          starts_at: "09:40", ends_at: "10:20", room: "Room 12",
          created_by_principal_id: 101, created_at: "2026-11-15T10:00:00.000Z",
        }],
      },
    });
  });
  await page.getByRole("button", { name: "Substitutions" }).click();
  const record = page.locator(".staff-substitution-record");
  await expect(record).toContainText("Completed");
  await expect(record).toContainText("Original Teacher");
  await expect(record).toContainText("Substitute Teacher");
  await expect(record.getByRole("button", { name: "Cancel cover" })).toHaveCount(0);
});


test("school-wide workload report shows scheduled periods without inferring availability", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "teacher.original",
      employee_code: "EMP-ORIGINAL", full_name: "Original Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.inactive",
      employee_code: "EMP-INACTIVE", full_name: "Inactive Teacher", designation: "Science Teacher",
      mobile_number: "9876543217", employment_status: "inactive", profile_complete: true,
      status: "disabled", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "timetable.read"]);
  await page.getByRole("button", { name: "Workload Report" }).click();
  await expect(page.getByRole("heading", { name: "School-wide Workload Report" })).toBeVisible();
  const row = page.locator(".staff-workload-report-row").filter({ hasText: "Original Teacher" });
  await expect(row).toContainText("4");
  await expect(row).toContainText("Monday (4)");
  await row.locator("summary").click();
  await expect(row).toContainText("Sections");
  await expect(row).toContainText("Subjects");
  await expect(page.getByText(/does not infer working days, free-period availability or workload limits/)).toBeVisible();
  await expect(page.getByText("Inactive Teacher")).toBeVisible();

  await page.getByLabel("Filter workload report by employment status").selectOption("active");
  const exportButton = page.getByRole("button", { name: "Export filtered CSV" });
  await expect(exportButton).toBeEnabled();
  const downloadPromise = page.waitForEvent("download");
  await exportButton.click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("staff-workload-report-2026.csv");
  const downloadPath = await download.path();
  expect(downloadPath).not.toBeNull();
  const csv = await readFile(downloadPath!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("Original Teacher");
  expect(csv).not.toContain("Inactive Teacher");
  expect(csv).toContain(`'   =HYPERLINK(""x"")`);
});



test("staff reports show school-scoped employment readiness and export approved fields only", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "teacher.active",
      employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.inactive",
      employee_code: "EMP-INACTIVE", full_name: "Inactive Teacher", designation: "Science Teacher",
      mobile_number: "9876543216", employment_status: "inactive", profile_complete: false,
      status: "disabled", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff Reports" })).toBeVisible();
  await expect(page.getByText("Total staff").locator("..")).toContainText("2");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Active employment", { exact: true }) }).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Inactive employment", { exact: true }) }).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Profile incomplete", { exact: true }) }).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Login disabled", { exact: true }) }).locator("strong")).toHaveText("1");
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export staff report" }).click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("EMP-ACTIVE");
  expect(csv).toContain("EMP-INACTIVE");
  expect(csv).not.toContain("9876543215");
  expect(csv).not.toContain("9876543216");
});

test("staff reports show monthly attendance and leave summaries only with matching permissions", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 501, school_id: 10, username: "teacher.active", employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read", "staff.leave.read"]);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Attendance & leave reports" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Monthly staff attendance" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Leave requests" })).toBeVisible();
  await expect(page.getByRole("cell", { name: "Active Teacher" }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: "Leave Teacher" })).toBeVisible();

  const attendanceDownloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export attendance CSV" }).click();
  const attendanceDownload = await attendanceDownloadPromise;
  const attendancePath = await attendanceDownload.path();
  expect(attendancePath).toBeTruthy();
  const attendanceCSV = await readFile(attendancePath!, "utf8");
  expect(attendanceCSV.startsWith("\uFEFF")).toBe(true);
  expect(attendanceCSV).toContain("EMP-ACTIVE");
  expect(attendanceCSV).toContain("18");

  const leaveDownloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export leave CSV" }).click();
  const leaveDownload = await leaveDownloadPromise;
  const leavePath = await leaveDownload.path();
  expect(leavePath).toBeTruthy();
  const leaveCSV = await readFile(leavePath!, "utf8");
  expect(leaveCSV.startsWith("\uFEFF")).toBe(true);
  expect(leaveCSV).toContain("EMP-LEAVE-01");
  expect(leaveCSV).toContain("Casual Leave");
});

test("staff reports include active academic assignments and export only permitted assignment fields", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 501, school_id: 10, username: "teacher.active", employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "teacher_assignments.read", "class_incharge_assignments.read"]);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Academic assignments" })).toBeVisible();
  await expect(page.getByText("2026–27 · Active assignments only")).toBeVisible();
  await expect(page.getByText("teacher.active")).toBeVisible();
  await expect(page.getByText("Subject assignments").last()).toBeVisible();
  await expect(page.getByText("Class-in-charge assignments").last()).toBeVisible();

  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export assignments CSV" }).click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("teacher.active");
  expect(csv).toContain("2026–27");
  expect(csv).not.toContain("Science");
});

test("staff reports do not request attendance or leave data without their read permissions", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff Reports" })).toBeVisible();
  await expect(page.getByText("Attendance and leave reports are hidden because you do not have their read permissions.")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Monthly staff attendance" })).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Leave requests" })).toHaveCount(0);
});

test("staff directory CSV export is filtered, UTF-8 compatible and safe from spreadsheet formulas", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 501, school_id: 10, username: "teacher.original",
      employee_code: "EMP-ORIGINAL", full_name: '\t=HYPERLINK("https://example.invalid","Click")', designation: "Mathematics Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.filtered",
      employee_code: "EMP-FILTERED", full_name: "Filtered Out Teacher", designation: "Science Teacher",
      mobile_number: "9876543216", employment_status: "inactive", profile_complete: false,
      status: "disabled", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Staff Directory", exact: true }).click();
  await page.getByLabel("Search staff by name, code, designation or username").fill("HYPERLINK");
  await expect(page.getByText("Filtered Out Teacher")).toHaveCount(0);

  const exportButton = page.getByRole("button", { name: /Export CSV/ });
  await expect(exportButton).toBeEnabled();
  const downloadPromise = page.waitForEvent("download");
  await exportButton.click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^staff-directory-\d{4}-\d{2}-\d{2}\.csv$/);
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.charCodeAt(0)).toBe(0xfeff);
  expect(csv).toContain("'\t=HYPERLINK");
  expect(csv).toContain("EMP-ORIGINAL");
  expect(csv).not.toContain("EMP-FILTERED");
  expect(csv).not.toContain("Filtered Out Teacher");
});


test("staff qualifications report uses one paginated aggregate endpoint and exports approved fields only", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 501, school_id: 10, username: "teacher.active", employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, staffRows, undefined, ["users.read"]);
  const requests: string[] = [];
  await page.route("**/api/school/staff/qualifications-report**", async route => {
    const url = new URL(route.request().url());
    requests.push(url.pathname + url.search);
    const certificationOnly = url.searchParams.get("record_type") === "certification";
    const rows = certificationOnly ? [{
      record_type: "certification", record_id: 22, staff_principal_id: 501, employee_code: "EMP-ACTIVE",
      staff_name: "Active Teacher", record_name: "Child Safety Training", organization: "State Board",
      record_date: "2025-04-01", expiry_date: "2027-04-01", verification_status: "pending",
    }] : [{
      record_type: "qualification", record_id: 11, staff_principal_id: 501, employee_code: "EMP-ACTIVE",
      staff_name: "Active Teacher", record_name: "B.Ed.", organization: "State University",
      record_date: "2022", expiry_date: "", verification_status: "verified",
    }, {
      record_type: "certification", record_id: 22, staff_principal_id: 501, employee_code: "EMP-ACTIVE",
      staff_name: "Active Teacher", record_name: "Child Safety Training", organization: "State Board",
      record_date: "2025-04-01", expiry_date: "2027-04-01", verification_status: "pending",
    }];
    await route.fulfill({ json: { report: {
      rows, limit: 50, offset: Number(url.searchParams.get("offset") ?? 0), total: certificationOnly ? 1 : 2,
      counts: certificationOnly ? [{ record_type: "certification", verification_status: "pending", count: 1 }] : [
        { record_type: "qualification", verification_status: "verified", count: 1 },
        { record_type: "certification", verification_status: "pending", count: 1 },
      ],
    } } });
  });
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Qualifications & certifications" })).toBeVisible();
  await expect(page.getByText("B.Ed.")).toBeVisible();
  await expect(page.getByText("Child Safety Training")).toBeVisible();
  const qualificationSection=page.locator(".staff-operational-report-section").filter({has:page.getByRole("heading",{name:"Qualifications & certifications"})});
  await expect(qualificationSection.getByText("Matching records").locator("..")).toContainText("2");
  expect(requests.length).toBeGreaterThanOrEqual(1);
  expect(requests.every(request => request.includes("record_type=all"))).toBe(true);

  await page.getByLabel("Record type").selectOption("certification");
  await expect(page.getByText("B.Ed.")).toHaveCount(0);
  await expect(page.getByText("Child Safety Training")).toBeVisible();
  expect(requests.some(request => request.includes("record_type=certification"))).toBe(true);
  expect(requests.every(request => request.startsWith("/api/school/staff/qualifications-report?"))).toBe(true);

  const downloadPromise = page.waitForEvent("download");
  await qualificationSection.getByRole("button", { name: "Export current page CSV" }).click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("Child Safety Training");
  expect(csv).not.toContain("credential");
  expect(csv).not.toContain("verification_notes");
});


test("staff experience report uses one paginated school aggregate endpoint and exports safe history fields", async ({ page }) => {
  const staffRows:Array<Record<string,unknown>>=[
    {id:1,principal_id:501,school_id:10,username:"teacher.active",employee_code:"EMP-ACTIVE",full_name:"Active Teacher",designation:"Mathematics Teacher",employment_status:"active",profile_complete:true,status:"active",must_change_password:false},
  ];
  await signInAndOpenStaff(page,staffRows,undefined,["users.read"]);
  const requests:string[]=[];
  await page.route("**/api/school/staff/experience-report**",async route=>{
    const url=new URL(route.request().url());requests.push(url.pathname+url.search);
    const current=url.searchParams.get("current_only")==="true";
    const rows=current?[{
      record_id:33,staff_principal_id:501,employee_code:"EMP-ACTIVE",staff_name:"Active Teacher",designation:"Mathematics Teacher",employment_status:"active",
      employer_name:"Current School",previous_designation:"Senior Teacher",employment_type:"permanent",start_date:"2024-04-01",end_date:"",is_current:true,
    }]:[{
      record_id:32,staff_principal_id:501,employee_code:"EMP-ACTIVE",staff_name:"Active Teacher",designation:"Mathematics Teacher",employment_status:"active",
      employer_name:"Previous School",previous_designation:"Teacher",employment_type:"contract",start_date:"2020-04-01",end_date:"2023-03-31",is_current:false,
    },{
      record_id:33,staff_principal_id:501,employee_code:"EMP-ACTIVE",staff_name:"Active Teacher",designation:"Mathematics Teacher",employment_status:"active",
      employer_name:"Current School",previous_designation:"Senior Teacher",employment_type:"permanent",start_date:"2024-04-01",end_date:"",is_current:true,
    }];
    await route.fulfill({json:{report:{rows,limit:50,offset:Number(url.searchParams.get("offset")??0),total:current?1:2,counts:current?[{employment_type:"permanent",is_current:true,count:1}]:[
      {employment_type:"contract",is_current:false,count:1},{employment_type:"permanent",is_current:true,count:1},
    ]}}});
  });
  await page.getByRole("button",{name:"Reports",exact:true}).click();
  await expect(page.getByRole("heading",{name:"Staff experience history"})).toBeVisible();
  const experienceSection=page.locator(".staff-operational-report-section").filter({has:page.getByRole("heading",{name:"Staff experience history"})});
  await expect(experienceSection.getByRole("cell",{name:"Previous School"})).toBeVisible();
  await expect(experienceSection.getByRole("cell",{name:"Current School"})).toBeVisible();
  expect(requests.length).toBeGreaterThanOrEqual(1);
  expect(requests.every(request=>request.startsWith("/api/school/staff/experience-report?"))).toBe(true);
  await page.getByLabel("Experience status").selectOption("current");
  await expect(experienceSection.getByRole("cell",{name:"Previous School"})).toHaveCount(0);
  await expect(experienceSection.getByRole("cell",{name:"Current School"})).toBeVisible();
  expect(requests.some(request=>request.includes("current_only=true"))).toBe(true);
  const downloadPromise=page.waitForEvent("download");
  await page.getByRole("button",{name:"Export current page CSV"}).last().click();
  const download=await downloadPromise;const path=await download.path();expect(path).toBeTruthy();
  const csv=await readFile(path!,"utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("Current School");
  expect(csv).not.toContain("description");
});


test("staff ID cards select only complete active profiles and expose approved print fields", async ({ page }) => {
  const rows:Array<Record<string,unknown>>=[
    {id:1,principal_id:501,school_id:10,username:"active.teacher",employee_code:"EMP-501",full_name:"Active Teacher",designation:"Mathematics Teacher",mobile_number:"9876543210",email:"private@example.test",joining_date:"2024-04-01",employment_status:"active",profile_complete:true,status:"active",must_change_password:false},
    {id:2,principal_id:502,school_id:10,username:"inactive.teacher",employee_code:"EMP-502",full_name:"Inactive Teacher",designation:"Former Teacher",employment_status:"inactive",profile_complete:true,status:"disabled",must_change_password:false},
    {id:3,principal_id:503,school_id:10,username:"incomplete.teacher",employee_code:"",full_name:"Incomplete Teacher",designation:"",employment_status:"active",profile_complete:false,status:"active",must_change_password:false},
    {id:4,principal_id:504,school_id:10,username:"unknown.profile",employee_code:"EMP-504",full_name:"Unknown Profile",designation:"Teacher",employment_status:"active",status:"active",must_change_password:false},
  ];
  await signInAndOpenStaff(page,rows,undefined,["users.read"]);
  await page.getByRole("button",{name:"ID Cards",exact:true}).click();
  await expect(page.getByRole("heading",{name:"Staff ID cards"})).toBeVisible();
  await expect(page.getByLabel(/Inactive Teacher/)).toHaveCount(0);
  await expect(page.getByLabel(/Incomplete Teacher/)).toHaveCount(0);
  await expect(page.getByLabel(/Unknown Profile/)).toHaveCount(0);
  await page.getByRole("button",{name:"Select all eligible"}).click();
  const preview = page.locator(".staff-id-print-area");
  await expect(preview.getByText("Active Teacher",{exact:true})).toBeVisible();
  await expect(preview.getByText("EMP-501",{exact:true})).toBeVisible();
  await expect(preview.getByText("Mathematics Teacher",{exact:true})).toBeVisible();
  await expect(preview.locator(".staff-id-avatar")).toHaveText("AT");
  await expect(page.getByText("private@example.test")).toHaveCount(0);
  await expect(page.getByText("9876543210")).toHaveCount(0);
  await page.getByLabel("Card template").selectOption("modern");
  await expect(page.locator(".staff-id-card--modern")).toHaveCount(1);
  await page.getByRole("button",{name:"Clear selection"}).click();
  await expect(page.getByText("Select one or more employees to preview their cards.")).toBeVisible();
  await expect(page.getByRole("button",{name:/Print/})).toBeDisabled();
});
