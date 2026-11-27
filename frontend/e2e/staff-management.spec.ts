import { readFile } from "node:fs/promises";
import { expect, test, type Locator, type Page, type Route } from "@playwright/test";
import { schoolDateValue } from "../src/shared/date/schoolDate";

function schoolDateOffset(days: number) {
  const [year, month, day] = schoolDateValue().split("-").map(Number);
  const date = new Date(year, month - 1, day, 12);
  date.setDate(date.getDate() + days);
  return date;
}

function schoolDateLabel(date: Date) {
  return date.toLocaleDateString(undefined, {
    weekday: "long", month: "long", day: "numeric", year: "numeric",
  });
}

async function chooseDatePickerValue(scope: Locator, ariaLabel: string, value: string) {
  const [year, month, day] = value.split("-").map(Number);
  await scope.getByRole("button", { name: ariaLabel }).click();
  const calendar = scope.getByRole("dialog", { name: `Choose ${ariaLabel.toLowerCase()}` });
  await calendar.getByRole("button", { name: "Choose year" }).click();
  while (true) {
    const range = await calendar.locator(".app-date-header strong").innerText();
    const [first, last] = range.split("–").map(Number);
    if (year < first) {
      await calendar.getByRole("button", { name: "Previous decade" }).click();
      await expect(calendar.locator(".app-date-year-grid button[aria-pressed='true']")).toBeFocused();
    } else if (year > last) {
      await calendar.getByRole("button", { name: "Next decade" }).click();
      await expect(calendar.locator(".app-date-year-grid button[aria-pressed='true']")).toBeFocused();
    } else break;
  }
  await calendar.locator(".app-date-year-grid button").filter({ hasText: new RegExp(`^${year}$`) }).click();
  await calendar.getByRole("button", { name: "Choose month" }).click();
  const monthDate = new Date(year, month - 1, 1, 12);
  await calendar.locator(".app-date-month-grid button").filter({ hasText: monthDate.toLocaleDateString(undefined, { month: "short" }) }).click();
  const selectedDate = new Date(year, month - 1, day, 12);
  await calendar.getByRole("button", { name: schoolDateLabel(selectedDate) }).click();
}

const staffUser = {
  principal_id: 101,
  principal_type: "staff",
  status: "active",
  school_id: 10,
};

async function signInAndOpenStaff(page: Page, staffRows: Array<Record<string, unknown>>, onStaffMutation?: (path: string, route: Route) => Promise<void>, permissions = ["users.read", "users.create", "users.update"], onStaffRecordMutation?: (collection: string, route: Route) => Promise<void>, leaveRequestRows?: Array<Record<string, unknown>>, staffRecordRows?: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>>) {
  const staffRecords: Record<string, Array<Record<string, unknown>>> = staffRecordRows ?? { qualifications: [], certifications: [], experience: [] };
  const leaveTypes: Array<Record<string, unknown>> = [{ id: 1, name: "Casual Leave", is_paid: true, requires_approval: true, is_active: true }];
  const leaveRequests: Array<Record<string, unknown>> = leaveRequestRows ?? [{
    id: 21, staff_principal_id: 501, employee_code: "EMP-LEAVE-01", staff_name: "Leave Teacher",
    leave_type_id: 1, leave_type_name: "Casual Leave", start_date: schoolDateValue(), end_date: schoolDateValue(),
    duration_type: "full_day", reason: "Family commitment", status: "pending", created_at: "2026-11-21T00:00:00.000Z",
  }];
  const schoolHolidays: Array<Record<string, unknown>> = [{ id: 31, holiday_date: schoolDateOffset(2).toISOString().slice(0, 10), name: "School Foundation Day", description: "Annual school holiday", is_active: true }];
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
    if (path === "/school/staff/summary" && method === "GET") {
      await route.fulfill({ json: { summary: {
        total: staffRows.length,
        active_employment: staffRows.filter(item => item.employment_status === "active").length,
        login_enabled: staffRows.filter(item => item.status === "active").length,
        profile_incomplete: staffRows.filter(item => item.profile_complete !== true).length,
        active_employment_disabled_login: staffRows.filter(item => item.employment_status === "active" && item.status !== "active").length,
        must_change_password: staffRows.filter(item => item.must_change_password === true).length,
        inactive_employment: staffRows.filter(item => item.employment_status === "inactive").length,
      } } });
      return;
    }
    if (path === "/school/staff" && method === "GET") {
      if (url.searchParams.has("limit")) {
        const limit = Number(url.searchParams.get("limit"));
        const offset = Number(url.searchParams.get("offset") ?? "0");
        const search = (url.searchParams.get("search") ?? "").toLowerCase();
        const staffType = url.searchParams.get("staff_type") ?? "";
        const employment = url.searchParams.get("employment_status") ?? "";
        const account = url.searchParams.get("account_status") ?? "";
        const profile = url.searchParams.get("profile_status") ?? "";
        const matching = staffRows.filter(item =>
          (!search || [item.username, item.full_name, item.employee_code, item.designation, item.mobile_number].some(value => String(value ?? "").toLowerCase().includes(search))) &&
          (!staffType || (staffType === "unclassified" ? (!item.staff_type || item.staff_type === "") : item.staff_type === staffType)) &&
          (!employment || item.employment_status === employment) &&
          (!account || (account === "enabled" ? item.status === "active" : item.status !== "active")) &&
          (!profile || (profile === "complete" ? item.profile_complete === true : item.profile_complete !== true))
        );
        await route.fulfill({ json: { staff: matching.slice(offset, offset + limit), total: matching.length, limit, offset } });
      } else {
        await route.fulfill({ json: { staff: staffRows } });
      }
      return;
    }
    if (path === "/school/staff-attendance" && method === "GET") {
      await route.fulfill({ json: { date: url.searchParams.get("date") ?? schoolDateValue(), data: [] } });
      return;
    }
    if (path === "/school/staff-attendance/month" && method === "GET") {
      const limit = Number(url.searchParams.get("limit") ?? 50);
      const offset = Number(url.searchParams.get("offset") ?? 0);
      const data = [{ staff_principal_id: 501, employee_code: "EMP-ACTIVE", full_name: "Active Teacher", designation: "Mathematics Teacher", present_days: 18, absent_days: 1, late_days: 2, half_days: 0, records: [] }];
      await route.fulfill({ json: { month: url.searchParams.get("month"), data: data.slice(offset, offset + limit), pagination: { limit, offset, total: data.length } } });
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
      const fromDate = url.searchParams.get("from_date") ?? "";
      const toDate = url.searchParams.get("to_date") ?? "";
      const matching = leaveRequests.filter(item => {
        const startsOn = String(item.start_date ?? "");
        const endsOn = String(item.end_date ?? "");
        const overlapsDateRange = (!fromDate || endsOn >= fromDate) && (!toDate || startsOn <= toDate);
        return (!status || item.status === status) && (!staffID || item.staff_principal_id === staffID) && overlapsDateRange;
      });
      const page = Number(url.searchParams.get("page") ?? 1);
      const pageSize = Number(url.searchParams.get("page_size") ?? 50);
      const data = matching.slice((page - 1) * pageSize, page * pageSize);
      await route.fulfill({ json: { data, pagination: { page, page_size: pageSize, total: matching.length } } });
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
      const matching = substitutions.filter(item => String(item.cover_date) >= from && String(item.cover_date) <= to &&
        (!staffID || item.original_teacher_principal_id === staffID || item.substitute_teacher_principal_id === staffID));
      const page = Number(url.searchParams.get("page") ?? 1);
      const pageSize = Number(url.searchParams.get("page_size") ?? 50);
      const data = matching.slice((page - 1) * pageSize, page * pageSize);
      await route.fulfill({ json: { substitutions: data, pagination: { page, page_size: pageSize, total: matching.length } } });
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
        if (onStaffRecordMutation) {
          await onStaffRecordMutation(collection, route);
          return;
        }
        const payload = request.postDataJSON() as Record<string, unknown>;
        const saved = { ...payload, id: staffRecords[collection].length + 1, staff_principal_id: principalID, verification_status: "pending", created_at: "2026-11-21T00:00:00.000Z", updated_at: "2026-11-21T00:00:00.000Z" };
        staffRecords[collection].push(saved);
        const singular = collection === "qualifications" ? "qualification" : collection === "certifications" ? "certification" : "experience";
        await route.fulfill({ status: 201, json: { [singular]: saved } });
        return;
      }
      if (method === "PATCH" && staffRecordMatch[3]) {
        if (onStaffRecordMutation) {
          await onStaffRecordMutation(collection, route);
          return;
        }
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
  const unboundedStaffListRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname === "/api/school/staff" && request.method() === "GET" && !url.searchParams.has("limit")) {
      unboundedStaffListRequests.push(request.url());
    }
  });
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
  expect(unboundedStaffListRequests).toEqual([]);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("1–2 of 2 staff members")).toBeVisible();
  expect(unboundedStaffListRequests).toEqual([]);
});


test("staff directory filters independently narrow staff type, employment, login account and profile completeness", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 611, school_id: 10, staff_type: "teaching", username: "active.enabled", employee_code: "EMP-F-01", full_name: "Active Enabled", designation: "Teacher", mobile_number: "9876543210", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 612, school_id: 10, staff_type: "teaching", username: "active.disabled", employee_code: "EMP-F-02", full_name: "Active Disabled", designation: "Teacher", mobile_number: "9876543211", employment_status: "active", profile_complete: true, status: "disabled", must_change_password: true },
    { id: 3, principal_id: 613, school_id: 10, staff_type: "non_teaching", username: "inactive.disabled", employee_code: "EMP-F-03", full_name: "Inactive Disabled", designation: "Office Assistant", mobile_number: "9876543212", employment_status: "inactive", profile_complete: true, status: "disabled", must_change_password: false },
    { id: 4, principal_id: 614, school_id: 10, username: "profile.pending", employee_code: "EMP-F-04", full_name: "Profile Pending", designation: "Staff", mobile_number: "9876543213", employment_status: "active", profile_complete: false, status: "disabled", must_change_password: false },
    { id: 5, principal_id: 615, school_id: 10, username: "profile.unknown", employee_code: "EMP-F-05", full_name: "Profile Unknown", designation: "Staff", mobile_number: "9876543214", employment_status: "active", status: "disabled", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("1–5 of 5 staff members")).toBeVisible();

  await page.getByRole("button", { name: "Filter by staff type" }).click();
  await page.getByRole("option", { name: "Teaching", exact: true }).click();
  await expect(page.getByText("1–2 of 2 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Enabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Disabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Inactive Disabled" })).toHaveCount(0);
  await page.getByRole("button", { name: "Filter by staff type" }).click();
  await page.getByRole("option", { name: "Non-teaching", exact: true }).click();
  await expect(page.getByText("1–1 of 1 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Inactive Disabled" })).toBeVisible();
  await page.getByRole("button", { name: "Filter by staff type" }).click();
  await page.getByRole("option", { name: "Unclassified", exact: true }).click();
  await expect(page.getByText("1–2 of 2 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Unknown" })).toBeVisible();
  await page.getByRole("button", { name: "Filter by staff type" }).click();
  await page.getByRole("option", { name: "All staff types", exact: true }).click();

  await page.getByRole("button", { name: "Filter by employment status" }).click();
  await page.getByRole("option", { name: "Active", exact: true }).click();
  await expect(page.getByText("1–4 of 4 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Enabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Disabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Unknown" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Inactive Disabled" })).toHaveCount(0);

  await page.getByRole("button", { name: "Filter by login account status" }).click();
  await page.getByRole("option", { name: "Disabled", exact: true }).click();
  await expect(page.getByText("1–3 of 3 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Disabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Unknown" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Enable profile.unknown" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Active Enabled" })).toHaveCount(0);
  await page.getByRole("button", { name: "Filter by login account status" }).click();
  await page.getByRole("option", { name: "All accounts", exact: true }).click();
  await page.getByRole("button", { name: "Filter by employment status" }).click();
  await page.getByRole("option", { name: "All employment", exact: true }).click();

  const profileCompletenessFilter = page.locator(".staff-directory-filters .app-select").nth(3).locator("button.app-select-trigger");
  await expect(profileCompletenessFilter).toBeVisible();
  await profileCompletenessFilter.click();
  await page.getByRole("option", { name: "Incomplete", exact: true }).click();
  await expect(page.getByText("1–2 of 2 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Unknown" })).toBeVisible();

  // Reuse the same trigger after the custom Select closes and restores focus.
  await expect(profileCompletenessFilter).toHaveAttribute("aria-expanded", "false");
  await profileCompletenessFilter.click();
  await page.getByRole("option", { name: "Complete", exact: true }).click();
  await expect(page.getByText("1–3 of 3 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Active Enabled" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Unknown" })).toHaveCount(0);

  // Search is tested independently of the profile filter selected above.
  await profileCompletenessFilter.click();
  await page.getByRole("option", { name: "All profiles", exact: true }).click();
  const search = page.getByRole("textbox", { name: "Search staff by name, code, designation, mobile or username" });
  await search.fill("9876543213");
  const searchContainer = page.locator(".staff-directory-toolbar .search");
  await search.focus();
  await expect(searchContainer).toHaveCSS("outline-style", "solid");
  await expect(searchContainer).toHaveCSS("outline-width", "2px");
  await expect(page.getByText("1–1 of 1 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Profile Pending" })).toBeVisible();
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(search).toHaveValue("");
  await expect(page.getByText("1–5 of 5 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Clear filters" })).toHaveCount(0);
});


test("staff directory uses server-side search and stable pagination", async ({ page }) => {
  await signInAndOpenStaff(page, []);
  await page.route(url => url.pathname === "/api/school/staff" && url.searchParams.has("limit"), async route => {
    const url = new URL(route.request().url());
    const limit = Number(url.searchParams.get("limit"));
    const offset = Number(url.searchParams.get("offset"));
    const search = url.searchParams.get("search") ?? "";
    const all = Array.from({ length: 120 }, (_, index) => ({
      id: index + 1,
      principal_id: index + 1001,
      school_id: 10,
      username: `directory.${String(index + 1).padStart(3, "0")}`,
      employee_code: `DIR-${String(index + 1).padStart(3, "0")}`,
      full_name: `Directory Employee ${String(index + 1).padStart(3, "0")}`,
      designation: "Office Staff",
      mobile_number: "9876543210",
      employment_status: "active",
      profile_complete: true,
      status: "active",
      must_change_password: false,
    }));
    const matching = search ? all.filter(item => item.full_name.toLowerCase().includes(search.toLowerCase())) : all;
    const staff = matching.slice(offset, offset + limit);
    await route.fulfill({ json: { staff, total: matching.length, limit, offset } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("1–50 of 120 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Directory Employee 001" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Directory Employee 051" })).toHaveCount(0);
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("51–100 of 120 staff members")).toBeVisible();
  await expect(page.getByText("Page 2 of 3")).toBeVisible();
  await expect(page.getByRole("button", { name: "Directory Employee 051" })).toBeVisible();

  const search = page.getByRole("textbox", { name: "Search staff by name, code, designation, mobile or username" });
  await search.fill("Directory Employee 075");
  await expect(page.getByText("1–1 of 1 staff members")).toBeVisible();
  await expect(page.getByRole("button", { name: "Directory Employee 075" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Previous" })).toHaveCount(0);
});

test("staff directory returns to the last valid page after disabling the final account", async ({ page }) => {
  const all=Array.from({length:51},(_,index)=>({
    id:index+1,principal_id:index+2001,school_id:10,username:`directory.disable.${index+1}`,
    employee_code:`DIR-DIS-${index+1}`,full_name:`Directory Disable ${index+1}`,
    designation:"Office Staff",mobile_number:"9876543210",employment_status:"active",
    profile_complete:true,status:"active",must_change_password:false,
  }));
  let disabled=false;
  await signInAndOpenStaff(page,[],undefined,["users.read","users.disable"]);
  await page.route(url=>url.pathname==="/api/school/staff"&&url.searchParams.has("limit"),async route=>{
    const url=new URL(route.request().url());const limit=Number(url.searchParams.get("limit"));const offset=Number(url.searchParams.get("offset"));
    const matching=disabled?all.slice(0,50):all;
    await route.fulfill({json:{staff:matching.slice(offset,offset+limit),total:matching.length,limit,offset}});
  });
  await page.route("**/api/school/staff/2051",async route=>{
    if(route.request().method()!=="DELETE"){await route.fallback();return;}
    disabled=true;await route.fulfill({json:{message:"Staff account disabled."}});
  });
  await page.getByRole("button",{name:/Staff Directory/}).click();
  await expect(page.getByText("Page 1 of 2")).toBeVisible();
  await page.getByRole("button",{name:"Next"}).click();
  const lastRow=page.locator("tr").filter({hasText:"directory.disable.51"});
  await expect(lastRow).toBeVisible();
  await lastRow.getByRole("button",{name:"Disable directory.disable.51"}).click();
  await page.getByRole("dialog",{name:"Disable staff account?"}).getByRole("button",{name:"Disable account"}).click();
  await expect(page.getByText("1–50 of 50 staff members")).toBeVisible();
  await expect(page.getByText("Page 1 of 1")).toHaveCount(0);
  await expect(page.getByRole("button",{name:"Directory Disable 1",exact:true})).toBeVisible();
  await expect(page.getByRole("button",{name:"Previous"})).toHaveCount(0);
});

test("staff directory filters stay within mobile, tablet and desktop layouts", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 621, school_id: 10, username: "responsive.active", employee_code: "EMP-R-01", full_name: "Responsive Active", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 622, school_id: 10, username: "responsive.inactive", employee_code: "EMP-R-02", full_name: "Responsive Inactive", designation: "Staff", employment_status: "inactive", profile_complete: false, status: "disabled", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();

  const controls = page.locator(".staff-directory-controls");
  const filters = controls.locator(".staff-directory-filters");
  const labels = filters.locator("label");
  await expect(labels).toHaveCount(4);

  for (const viewport of [
    { width: 390, height: 844 },
    { width: 768, height: 900 },
    { width: 1280, height: 900 },
  ]) {
    await page.setViewportSize(viewport);
    await expect(controls).toBeVisible();
    for (let index = 0; index < 4; index += 1) {
      await expect(labels.nth(index)).toBeVisible();
      await expect(labels.nth(index).locator("button.app-select-trigger")).toBeVisible();
    }
    const geometry = await filters.evaluate((element) => {
      const bounds = element.getBoundingClientRect();
      const labelBounds = Array.from(element.querySelectorAll("label")).map((label) => {
        const rect = label.getBoundingClientRect();
        return { left: rect.left, right: rect.right };
      });
      return {
        left: bounds.left,
        right: bounds.right,
        clientWidth: element.clientWidth,
        scrollWidth: element.scrollWidth,
        labelsInside: labelBounds.every((rect) => rect.left >= bounds.left - 1 && rect.right <= bounds.right + 1),
      };
    });
    expect(geometry.scrollWidth).toBeLessThanOrEqual(geometry.clientWidth + 1);
    expect(geometry.labelsInside).toBe(true);
    expect(geometry.left).toBeGreaterThanOrEqual(0);
    expect(geometry.right).toBeLessThanOrEqual(viewport.width + 1);
  }

  await page.setViewportSize({ width: 390, height: 844 });
  await filters.locator("label").nth(1).locator("button.app-select-trigger").click();
  await page.getByRole("option", { name: "Active", exact: true }).click();
  await expect(page.getByText("1–1 of 1 staff members")).toBeVisible();
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

test("staff leave decisions refresh the employee audit history when the profile is reopened", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let auditReads = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read", "staff.leave.decide", "school.audit.read"]);
  await page.route("**/api/school/staff/501/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Leave Teacher" }).click();
  const profile = page.getByRole("dialog", { name: "Leave Teacher" });
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect(profile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstAuditReadCount = auditReads;
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByText("Family commitment")).toBeVisible();
  await page.getByRole("button", { name: "Approve" }).click();
  const decision = page.getByRole("dialog", { name: "Approve leave request" });
  await decision.getByRole("button", { name: "Confirm approval" }).click();
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Leave Teacher" }).click();
  const reopenedProfile = page.getByRole("dialog", { name: "Leave Teacher" });
  await reopenedProfile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstAuditReadCount);
});

test("staff leave creation and cancellation refresh employee audit history", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.teacher",
    employee_code: "EMP-LEAVE-01", full_name: "Leave Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let auditReads = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read", "staff.leave.manage_requests", "school.audit.read"]);
  await page.route("**/api/school/staff/501/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Leave Teacher" }).click();
  let profile = page.getByRole("dialog", { name: "Leave Teacher" });
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect(profile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstAuditReadCount = auditReads;
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await page.getByRole("button", { name: "Staff member", exact: true }).click();
  await page.getByRole("option", { name: /Leave Teacher/ }).click();
  await page.getByLabel("Leave type").click();
  await page.getByRole("option", { name: /Casual Leave/ }).click();
  const todayLabel = schoolDateLabel(schoolDateOffset(0));
  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: todayLabel }).click();
  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: todayLabel }).click();
  await page.getByLabel("Reason").fill("Audit history create and cancel");
  await page.getByRole("button", { name: "Submit leave request" }).click();
  const createdRequest = page.locator("article").filter({ hasText: "Audit history create and cancel" });
  await expect(createdRequest).toBeVisible();

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Leave Teacher" }).click();
  profile = page.getByRole("dialog", { name: "Leave Teacher" });
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstAuditReadCount);
  const afterCreateAuditReadCount = auditReads;
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await createdRequest.getByRole("button", { name: "Cancel request" }).click();
  const confirmation = page.getByRole("dialog", { name: "Cancel leave request?" });
  await confirmation.getByRole("button", { name: "Cancel request", exact: true }).click();
  await expect(createdRequest).toHaveCount(0);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Leave Teacher" }).click();
  profile = page.getByRole("dialog", { name: "Leave Teacher" });
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(afterCreateAuditReadCount);
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
      mobile_number: "9876543216", employment_status: "inactive", profile_complete: false,
      status: "disabled", must_change_password: false,
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

test("staff leave request date filters return only overlapping requests and can be cleared together", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.dates",
    employee_code: "EMP-LEAVE-DATE-01", full_name: "Date Filter Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read"]);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  const today = schoolDateValue();
  const tomorrow = schoolDateOffset(1).toISOString().slice(0, 10);
  const dateRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return request.method() === "GET" && url.pathname.endsWith("/school/staff-leave/requests") &&
      url.searchParams.get("from_date") === today &&
      url.searchParams.get("to_date") === tomorrow;
  });
  await chooseDatePickerValue(page, "Leave requests from date", today);
  await chooseDatePickerValue(page, "Leave requests to date", tomorrow);
  const filteredURL = new URL((await dateRequest).url());
  expect(filteredURL.searchParams.get("from_date")).toBe(today);
  expect(filteredURL.searchParams.get("to_date")).toBe(tomorrow);
  await expect(page.getByText("Leave Teacher")).toBeVisible();

  const narrowedRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return request.method() === "GET" && url.pathname.endsWith("/school/staff-leave/requests") &&
      url.searchParams.get("from_date") === tomorrow &&
      url.searchParams.get("to_date") === tomorrow;
  });
  await chooseDatePickerValue(page, "Leave requests from date", tomorrow);
  const narrowedURL = new URL((await narrowedRequest).url());
  expect(narrowedURL.searchParams.get("from_date")).toBe(tomorrow);
  expect(narrowedURL.searchParams.get("to_date")).toBe(tomorrow);
  await expect(page.getByText("No leave requests match these filters.")).toBeVisible();
  await expect(page.getByText("Leave Teacher")).toHaveCount(0);

  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page.getByLabel("Leave requests from date")).toContainText("Any start date");
  await expect(page.getByLabel("Leave requests to date")).toContainText("Any end date");
  await expect(page.getByRole("button", { name: "Clear filters" })).toHaveCount(0);
  await expect(page.getByText("Leave Teacher")).toBeVisible();
});

test("staff leave date filter errors are field-specific and invalid ranges do not fetch", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.filter.validation",
    employee_code: "EMP-LEAVE-FILTER-01", full_name: "Filter Validation Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  let requestCount = 0;
  page.on("request", request => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname.endsWith("/school/staff-leave/requests")) requestCount += 1;
  });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read"]);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect.poll(() => requestCount).toBeGreaterThan(0);
  await page.waitForLoadState("networkidle");
  const initialRequestCount = requestCount;

  const today = schoolDateOffset(0);
  const tomorrow = schoolDateOffset(1);
  await page.getByLabel("Leave requests to date").click();
  await page.getByRole("dialog", { name: "Choose leave requests to date" }).getByRole("button", { name: schoolDateLabel(today) }).click();
  await expect(page.getByLabel("Leave requests from date")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Leave requests from date")).toHaveAttribute("aria-describedby", "leave-filter-from-date-error");
  await expect(page.getByText("Select both a From date and a To date to filter by date range.")).toBeVisible();

  await page.getByLabel("Leave requests from date").click();
  await page.getByRole("dialog", { name: "Choose leave requests from date" }).getByRole("button", { name: schoolDateLabel(tomorrow) }).click();
  await expect(page.getByLabel("Leave requests to date")).toHaveAttribute("aria-invalid", "true");
  await expect(page.getByLabel("Leave requests to date")).toHaveAttribute("aria-describedby", "leave-filter-to-date-error");
  await expect(page.getByText("The To date must be on or after the From date.")).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(requestCount).toBe(initialRequestCount);
  await expect(page.getByRole("button", { name: "Refresh", exact: true })).toBeDisabled();
  expect(requestCount).toBe(initialRequestCount);
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

test("read-only staff profile hides every employee edit action", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 431, school_id: 10, username: "teacher.readonly",
    employee_code: "EMP-RO-01", full_name: "Read Only Teacher", designation: "Science Teacher",
    mobile_number: "9876543218", email: "readonly.teacher@example.com",
    joining_date: "2025-04-01", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByRole("button", { name: /Edit employee profile for teacher.readonly/ })).toHaveCount(0);
  await page.getByRole("button", { name: "Read Only Teacher" }).click();

  const dialog = page.getByRole("dialog", { name: "Read Only Teacher" });
  await expect(dialog.getByRole("button", { name: "Edit employee details" })).toHaveCount(0);
  await dialog.getByRole("button", { name: "Personal Details" }).click();
  await expect(dialog.getByRole("button", { name: "Edit personal details" })).toHaveCount(0);
  await dialog.getByRole("button", { name: "Employment" }).click();
  await expect(dialog.getByRole("button", { name: "Edit employment details" })).toHaveCount(0);
  await dialog.getByRole("button", { name: "Academic Assignments" }).click();
  await expect(dialog.getByRole("button", { name: "Open assignments workspace" })).toHaveCount(0);
  await expect(dialog.getByText("You do not have permission to view academic assignments.")).toBeVisible();
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
  await expect(dialog.getByText("Choose Teaching or Non-teaching staff.")).toBeVisible();
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter a designation (up to 100 characters).")).toBeVisible();

  await dialog.getByLabel("Designation").fill("Class Teacher");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Use 3–32 lowercase letters, numbers, dots, underscores or hyphens; start with a letter or number.")).toBeVisible();
  await expect(dialog.getByText("Password must contain at least 12 characters.")).toBeVisible();

  await dialog.getByLabel("Login username").fill(".validation");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Use 3–32 lowercase letters, numbers, dots, underscores or hyphens; start with a letter or number.")).toBeVisible();
  await expect(dialog.getByRole("heading", { name: "Review & create" })).toHaveCount(0);

  await dialog.getByLabel("Login username").fill("validation.teacher");
  await dialog.getByLabel("Initial password").fill("short");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Password must contain at least 12 characters.")).toBeVisible();
  await expect(dialog.getByRole("heading", { name: "Review & create" })).toHaveCount(0);

  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByRole("heading", { name: "Review & create" })).toBeVisible();
});


test("staff modal traps focus from its dialog container and restores focus to its trigger", async ({ page }) => {
  await signInAndOpenStaff(page, [], undefined, ["users.read", "users.create"]);
  const trigger = page.getByRole("button", { name: "Add staff", exact: true });
  await trigger.focus();
  await trigger.click();

  const dialog = page.getByRole("dialog", { name: "Add employee" });
  const close = dialog.getByRole("button", { name: "Close dialog" });
  const continueButton = dialog.getByRole("button", { name: "Continue" });
  await expect(dialog).toBeFocused();

  // Initial focus is on the dialog container; reverse tab must wrap to the
  // last visible control rather than escaping the modal or landing on hidden fields.
  await page.keyboard.press("Shift+Tab");
  await expect(continueButton).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(close).toBeFocused();

  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(trigger).toBeFocused();
});


test("staff profile email validation rejects malformed address components", async ({ page }) => {
  await signInAndOpenStaff(page, [], async (_path, route) => {
    await route.fulfill({ status: 201, json: { staff: { id: 93, principal_id: 293, school_id: 10, username: "email.teacher", employee_code: "EMP-EMAIL", full_name: "Email Teacher", designation: "Class Teacher", mobile_number: "9876543210", employment_status: "active", profile_complete: true, status: "active", must_change_password: true } } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Add staff" }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Full name").fill("Email Teacher");
  await dialog.getByLabel("Employee code").fill("EMP-EMAIL");
  await dialog.getByLabel("Mobile number").fill("9876543210");
  const email = dialog.getByLabel("Email address");
  for (const invalid of ["teacher@domain..com", "teacher..name@domain.com", "teacher@domain.com.", "teacher@-domain.com"]) {
    await email.fill(invalid);
    await dialog.getByRole("button", { name: "Continue" }).click();
    await expect(dialog.getByText("Enter a valid email address.")).toBeVisible();
  }
  await email.fill("a@b.c");
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter a valid email address.")).toHaveCount(0);
  await expect(dialog.getByLabel("Designation")).toBeVisible();
  await dialog.getByRole("button", { name: "Back" }).click();
  await email.fill("é@例子.测试");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter a valid email address.")).toHaveCount(0);
  await expect(dialog.getByLabel("Designation")).toBeVisible();
  await dialog.getByRole("button", { name: "Back" }).click();
  const tooLongUnicodeEmail = "é".repeat(251) + "@x.y";
  await email.fill(tooLongUnicodeEmail);
  await expect(email).toHaveValue("é".repeat(251) + "@x.");
  const maxLengthUnicodeEmail = "é".repeat(250) + "@x.y";
  await email.fill(maxLengthUnicodeEmail);
  await expect(email).toHaveValue(maxLengthUnicodeEmail);
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(dialog.getByText("Enter a valid email address.")).toHaveCount(0);
  await expect(dialog.getByLabel("Designation")).toBeVisible();
});

test("staff profile creation accepts Unicode names and designations at database character limits", async ({ page }) => {
  const created: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, [], async (_path, route) => {
    created.push(route.request().postDataJSON() as Record<string, unknown>);
    await route.fulfill({ status: 201, json: { staff: {
      id: 91, principal_id: 291, school_id: 10, username: "unicode.teacher",
      employee_code: "EMP-UNICODE", full_name: created[0].full_name,
      designation: created[0].designation, mobile_number: "9876543210",
      employment_status: "active", profile_complete: true, status: "active",
      must_change_password: true,
    } } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Add staff" }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  const fullName = "🔤".repeat(160);
  const designation = "📚".repeat(100);
  const fullNameInput = dialog.getByLabel("Full name");
  await fullNameInput.fill("🔤".repeat(161));
  await expect(fullNameInput).toHaveValue(fullName);
  await fullNameInput.fill(fullName);
  await dialog.getByLabel("Employee code").fill("EMP-UNICODE");
  await dialog.getByLabel("Mobile number").fill("9876543210");
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  const designationInput = dialog.getByLabel("Designation");
  await designationInput.fill("📚".repeat(101));
  await expect(designationInput).toHaveValue(designation);
  await designationInput.fill(designation);
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByLabel("Login username").fill("unicode.teacher");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();

  await expect(dialog.getByRole("heading", { name: "Review & create" })).toBeVisible();
  await dialog.getByRole("button", { name: "Create employee" }).click();
  await expect.poll(() => created.length).toBe(1);
  expect(Array.from(String(created[0].full_name)).length).toBe(160);
  expect(Array.from(String(created[0].designation)).length).toBe(100);
  expect(created[0].staff_type).toBe("teaching");
  expect(created[0].full_name).toBe(fullName);
  expect(created[0].designation).toBe(designation);
  await expect(page.getByRole("dialog", { name: "Share staff login credentials" })).toBeVisible();
});

test("shared date picker preserves years below 100 and navigates across month boundaries", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 710, school_id: 10, username: "teacher.early-date",
    employee_code: "EMP-EARLY-DATE", full_name: "Early Date Teacher", designation: "History Teacher",
    mobile_number: "9876543210", joining_date: "0001-02-28", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Edit employee profile for teacher.early-date" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit employee profile" });
  await dialog.getByRole("button", { name: "Joining date" }).click();
  const calendar = dialog.getByRole("dialog", { name: "Choose joining date" });
  await expect(calendar.getByRole("button", { name: "Choose year" })).toHaveText("1");
  const activeDate = calendar.locator(".app-date-day[tabindex='0']");
  await expect(activeDate).toHaveText("28");
  await activeDate.press("ArrowRight");
  await expect(calendar.getByRole("button", { name: "Choose month" })).toHaveText("March");
  const nextDate = calendar.locator(".app-date-day[tabindex='0']");
  await expect(nextDate).toHaveText("1");
  await nextDate.click();
  await expect(dialog.locator('input[name="joining_date"]')).toHaveValue("0001-03-01");
});

test("staff creation uses the shared IST date picker and clears the credential handoff after copying", async ({ page }) => {
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
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
  await expect(page.getByRole("textbox", { name: "Search staff by name, code, designation, mobile or username" })).toBeVisible();
  await page.getByRole("button", { name: "Add staff" }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Full name").fill("Teacher One");
  await dialog.getByLabel("Employee code").fill("EMP-001");
  await dialog.getByLabel("Mobile number").fill("9876543210");
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();

  await dialog.getByLabel("Designation").fill("Class Teacher");
  await dialog.getByRole("button", { name: "Joining date" }).click();
  const calendar = dialog.getByRole("dialog", { name: "Choose joining date" });
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await calendar.getByRole("button", { name: "Choose year" }).click();
  await expect(calendar.locator(".app-date-year-grid button[aria-pressed='true']")).toBeFocused();
  await expect(calendar.locator(".app-date-year-grid button", { hasText: "2028" })).toBeDisabled();
  await calendar.locator(".app-date-year-grid button[aria-pressed='true']").click();
  await calendar.locator(".app-date-day[tabindex='0']").press("ArrowRight");
  await expect(calendar.locator(".app-date-day[tabindex='0']")).toBeFocused();
  await calendar.getByRole("button", { name: "Choose month" }).click();
  await expect(calendar.locator(".app-date-month-grid button[aria-pressed='true']")).toBeFocused();
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
  await expect(handoff.getByRole("button", { name: "Copy credentials" })).toBeVisible();
  await handoff.getByRole("button", { name: "Copy credentials" }).click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toContain("Temporary password: E2E-Strong-Password-2026");
  await expect(handoff).toHaveCount(0);

  // A second successful creation can explicitly defer sharing; the one-time
  // password handoff must be cleared just as it is after a successful copy.
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Add staff" }).click();
  const secondDialog = page.getByRole("dialog", { name: "Add employee" });
  await secondDialog.getByLabel("Full name").fill("Teacher Two");
  await secondDialog.getByLabel("Employee code").fill("EMP-002");
  await secondDialog.getByLabel("Mobile number").fill("9876543212");
  await secondDialog.getByRole("button", { name: "Staff type" }).click();
  await secondDialog.getByRole("option", { name: "Non-teaching", exact: true }).click();
  await secondDialog.getByRole("button", { name: "Continue" }).click();
  await secondDialog.getByLabel("Designation").fill("Science Teacher");
  await secondDialog.getByRole("button", { name: "Continue" }).click();
  await secondDialog.getByLabel("Login username").fill("teacher.two");
  await secondDialog.getByLabel("Initial password").fill("E2E-Strong-Password-2027");
  await secondDialog.getByRole("button", { name: "Continue" }).click();
  await secondDialog.getByRole("button", { name: "Create employee" }).click();
  await expect.poll(() => created.length).toBe(2);
  expect(created[1].staff_type).toBe("non_teaching");
  const deferredHandoff = page.getByRole("dialog", { name: "Share staff login credentials" });
  await expect(deferredHandoff).toBeVisible();
  await deferredHandoff.getByRole("button", { name: "Do this later" }).click();
  await expect(deferredHandoff).toHaveCount(0);

  // A clipboard denial must not falsely report success or discard the one-time handoff.
  await page.getByRole("button", { name: "Add staff" }).click();
  const thirdDialog = page.getByRole("dialog", { name: "Add employee" });
  await thirdDialog.getByLabel("Full name").fill("Teacher Three");
  await thirdDialog.getByLabel("Employee code").fill("EMP-003");
  await thirdDialog.getByLabel("Mobile number").fill("9876543213");
  await thirdDialog.getByRole("button", { name: "Staff type" }).click();
  await thirdDialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await thirdDialog.getByRole("button", { name: "Continue" }).click();
  await thirdDialog.getByLabel("Designation").fill("Science Teacher");
  await thirdDialog.getByRole("button", { name: "Continue" }).click();
  await thirdDialog.getByLabel("Login username").fill("teacher.three");
  await thirdDialog.getByLabel("Initial password").fill("E2E-Strong-Password-2028");
  await thirdDialog.getByRole("button", { name: "Continue" }).click();
  await thirdDialog.getByRole("button", { name: "Create employee" }).click();
  await expect.poll(() => created.length).toBe(3);
  const failedCopyHandoff = page.getByRole("dialog", { name: "Share staff login credentials" });
  await page.evaluate(() => Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: async () => { throw new Error("Clipboard permission denied"); } },
  }));
  await failedCopyHandoff.getByRole("button", { name: "Copy credentials" }).click();
  await expect(page.getByText("Clipboard access is unavailable. Use the password-reset flow if needed.")).toBeVisible();
  await expect(failedCopyHandoff).toBeVisible();
  await expect(failedCopyHandoff.getByText("teacher.three")).toBeVisible();
  await failedCopyHandoff.getByRole("button", { name: "Do this later" }).click();
  await expect(failedCopyHandoff).toHaveCount(0);
  await expect(page.locator('input[type="date"]')).toHaveCount(0);
});

test("staff creation keeps entered details after failure and blocks concurrent duplicate submits", async ({ page }) => {
  let attempts = 0;
  let releaseCreate: (() => void) | undefined;
  await signInAndOpenStaff(page, [], async (_path, route) => {
    attempts += 1;
    if (attempts === 1) {
      await route.fulfill({ status: 503, json: { message: "Staff account creation is temporarily unavailable." } });
      return;
    }
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    await new Promise<void>(resolve => { releaseCreate = resolve; });
    await route.fulfill({ status: 201, json: { staff: {
      id: 1, principal_id: 601, school_id: 10, username: payload.username,
      employee_code: payload.employee_code, full_name: payload.full_name,
      designation: payload.designation, mobile_number: payload.mobile_number,
      employment_status: "active", profile_complete: true, status: "active",
      must_change_password: true,
    } } });
  });

  await page.getByRole("button", { name: "Add staff", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Full name").fill("Retry Teacher");
  await dialog.getByLabel("Employee code").fill("EMP-RETRY-01");
  await dialog.getByLabel("Mobile number").fill("9876543211");
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByLabel("Designation").fill("English Teacher");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByLabel("Login username").fill("retry.teacher");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2028");
  await dialog.getByRole("button", { name: "Continue" }).click();

  await dialog.getByRole("button", { name: "Create employee" }).click();
  await expect(dialog.getByText("Staff account creation is temporarily unavailable.")).toBeVisible();
  await expect(dialog.getByLabel("Full name")).toHaveValue("Retry Teacher");
  await expect(dialog.getByLabel("Employee code")).toHaveValue("EMP-RETRY-01");
  await expect(dialog.getByLabel("Login username")).toHaveValue("retry.teacher");
  await expect(dialog.getByLabel("Initial password")).toHaveValue("E2E-Strong-Password-2028");

  await dialog.getByRole("button", { name: "Create employee" }).click();
  await expect.poll(() => attempts).toBe(2);
  const createButton = dialog.getByRole("button", { name: "Creating…" });
  await expect(createButton).toBeDisabled();
  await dialog.locator("form").evaluate(form => (form as HTMLFormElement).requestSubmit());
  expect(attempts).toBe(2);
  releaseCreate?.();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: "Share staff login credentials" })).toBeVisible();
});

test("staff creation returns server field conflicts to the correct wizard step", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [];
  const attempts: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    attempts.push(payload);
    if (attempts.length === 1) {
      await route.fulfill({ status: 409, json: {
        status: "error", code: "employee_code_taken",
        message: "This employee code is already used in this school",
      } });
      return;
    }
    if (attempts.length === 2) {
      await route.fulfill({ status: 409, json: { status: "error", message: "This username is already in use" } });
      return;
    }
    if (attempts.length === 3) {
      await route.fulfill({ status: 400, json: { status: "error", message: "This password has appeared in known data breaches. Choose a different password." } });
      return;
    }
    if (attempts.length === 4) {
      await route.fulfill({ status: 503, json: { status: "error", message: "Password safety verification is temporarily unavailable. Try again shortly." } });
      return;
    }
    const saved = {
      id: 1, principal_id: 611, school_id: 10, username: payload.username,
      employee_code: payload.employee_code, full_name: payload.full_name,
      designation: payload.designation, mobile_number: payload.mobile_number,
      staff_type: payload.staff_type, employment_status: "active",
      profile_complete: true, status: "active", must_change_password: true,
    };
    rows.push(saved);
    await route.fulfill({ status: 201, json: { staff: saved } });
  });

  await page.getByRole("button", { name: "Add staff", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Add employee" });
  await dialog.getByLabel("Full name").fill("Conflict Teacher");
  await dialog.getByLabel("Employee code").fill("EMP-DUP-01");
  await dialog.getByLabel("Mobile number").fill("9876543218");
  await dialog.getByRole("button", { name: "Staff type" }).click();
  await dialog.getByRole("option", { name: "Teaching", exact: true }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByLabel("Designation").fill("Class Teacher");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByLabel("Login username").fill("conflict.teacher");
  await dialog.getByLabel("Initial password").fill("E2E-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => attempts.length).toBe(1);
  await expect(dialog.getByLabel("Employee code")).toBeVisible();
  await expect(dialog.getByText("This employee code is already used in this school.")).toBeVisible();
  await expect(dialog.getByLabel("Full name")).toHaveValue("Conflict Teacher");
  await expect(dialog.getByLabel("Employee code")).toHaveValue("EMP-DUP-01");
  await expect(dialog.getByLabel("Mobile number")).toHaveValue("9876543218");
  await expect(dialog.getByLabel("Designation")).toHaveValue("Class Teacher");
  await expect(dialog.getByLabel("Login username")).toHaveValue("conflict.teacher");

  await dialog.getByLabel("Employee code").fill("EMP-UNIQUE-01");
  await expect(dialog.getByText("This employee code is already used in this school.")).toHaveCount(0);
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => attempts.length).toBe(2);
  await expect(dialog.getByLabel("Login username")).toBeVisible();
  await expect(dialog.getByText("This username is already in use.")).toBeVisible();
  await expect(dialog.getByLabel("Login username")).toHaveValue("conflict.teacher");
  await dialog.getByLabel("Login username").fill("unique.teacher");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => attempts.length).toBe(3);
  await expect(dialog.getByLabel("Initial password")).toBeVisible();
  await expect(dialog.getByText("This password has appeared in known data breaches. Choose a different password.")).toBeVisible();
  await dialog.getByLabel("Initial password").fill("E2E-Another-Strong-Password-2026");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => attempts.length).toBe(4);
  await expect(dialog.getByLabel("Initial password")).toBeVisible();
  await expect(dialog.getByText("Password safety verification is temporarily unavailable. Try again shortly.")).toBeVisible();
  await expect(dialog.getByLabel("Initial password")).toHaveValue("E2E-Another-Strong-Password-2026");
  // A server-side field error returns the wizard to Login & Access (step 3).
  // Revalidate that step before proceeding to the final review screen.
  await dialog.getByRole("button", { name: "Continue" }).click();
  await dialog.getByRole("button", { name: "Create employee" }).click();

  await expect.poll(() => attempts.length).toBe(5);
  expect(attempts[1].employee_code).toBe("EMP-UNIQUE-01");
  expect(attempts[2].username).toBe("unique.teacher");
  expect(attempts[3].initial_password).toBe("E2E-Another-Strong-Password-2026");
  expect(attempts[4].initial_password).toBe("E2E-Another-Strong-Password-2026");
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole("dialog", { name: "Share staff login credentials" })).toBeVisible();
});


test("disabling a staff account requires explicit confirmation and refreshes its status", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 516, school_id: 10, username: "disable.teacher",
    employee_code: "EMP-DISABLE-01", full_name: "Disable Teacher", designation: "Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let disableCalls = 0;
  let holdDisable = false;
  let releaseDisable!: () => void;
  const pendingDisable = new Promise<void>(resolve => { releaseDisable = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.disable"]);
  await page.route("**/api/school/staff/516", async route => {
    if (route.request().method() === "DELETE") {
      disableCalls += 1;
      if (holdDisable) await pendingDisable;
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
  holdDisable = true;
  const confirmButton = confirmation.locator("button.app-modal-confirm");
  await confirmButton.click();
  await expect(confirmButton).toBeDisabled();
  await expect(confirmation.getByRole("button", { name: "Close dialog" })).toBeDisabled();
  await expect(confirmation.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(confirmation).toBeVisible();
  await page.locator(".app-modal-backdrop").last().click({ position: { x: 4, y: 4 } });
  await expect(confirmation).toBeVisible();
  await confirmButton.evaluate((element: HTMLButtonElement) => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
  });
  await expect.poll(() => disableCalls).toBe(1);
  releaseDisable();
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

test("staff password reset ignores a second submit while the request is pending", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 520, school_id: 10, username: "reset.double",
    employee_code: "EMP-RESET-DOUBLE", full_name: "Reset Double", designation: "Teacher",
    mobile_number: "9876543219", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let resetCalls = 0;
  let releaseReset!: () => void;
  const pendingReset = new Promise<void>(resolve => { releaseReset = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.password_reset"]);
  await page.route("**/api/school/staff/520/password/reset", async route => {
    if (route.request().method() !== "POST") { await route.fallback(); return; }
    resetCalls += 1;
    await pendingReset;
    await route.fulfill({ json: { message: "Password reset." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Reset password for reset.double" }).click();
  const dialog = page.getByRole("dialog", { name: "Reset staff password" });
  await dialog.getByLabel("New temporary password").fill("StrongTempPass!2026");
  const submit = dialog.getByRole("button", { name: "Reset password", exact: true });
  await submit.click();
  await expect(dialog.getByRole("button", { name: "Saving…" })).toBeDisabled();
  await expect(dialog.getByRole("button", { name: "Close dialog" })).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
  await page.locator(".app-modal-backdrop").last().click({ position: { x: 4, y: 4 } });
  await expect(dialog).toBeVisible();
  await dialog.locator("form").evaluate((form: HTMLFormElement) => form.requestSubmit());
  await page.waitForTimeout(150);
  expect(resetCalls).toBe(1);
  releaseReset();
  await expect(page.getByText("Password reset. The staff member must change it at next sign-in.")).toBeVisible();
  expect(resetCalls).toBe(1);
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

test("enabling a staff account ignores a second submit while the request is pending", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 521, school_id: 10, username: "enable.double",
    employee_code: "EMP-ENABLE-DOUBLE", full_name: "Enable Double", designation: "Teacher",
    mobile_number: "9876543220", employment_status: "active", profile_complete: true,
    status: "disabled", must_change_password: false,
  }];
  let enableCalls = 0;
  let releaseEnable!: () => void;
  const pendingEnable = new Promise<void>(resolve => { releaseEnable = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.update"]);
  await page.route("**/api/school/staff/521/enable", async route => {
    if (route.request().method() !== "POST") { await route.fallback(); return; }
    enableCalls += 1;
    await pendingEnable;
    rows[0].status = "active";
    rows[0].must_change_password = true;
    await route.fulfill({ json: { message: "Staff account enabled." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  const staffRow = page.locator("tr").filter({ hasText: "enable.double" });
  await expect(staffRow).toBeVisible();
  await staffRow.getByRole("button", { name: "Enable enable.double" }).click();
  const dialog = page.getByRole("dialog", { name: "Enable staff account" });
  await dialog.getByLabel("Temporary password").fill("EnableTempPass!2026");
  const submit = dialog.getByRole("button", { name: "Enable account", exact: true });
  await submit.click();
  await expect(dialog.getByRole("button", { name: "Saving…" })).toBeDisabled();
  await expect(dialog.getByRole("button", { name: "Close dialog" })).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
  await page.locator(".app-modal-backdrop").last().click({ position: { x: 4, y: 4 } });
  await expect(dialog).toBeVisible();
  await dialog.locator("form").evaluate((form: HTMLFormElement) => form.requestSubmit());
  await page.waitForTimeout(150);
  expect(enableCalls).toBe(1);
  releaseEnable();
  await expect(page.getByText("Staff account enabled. A password change is required at next sign-in.")).toBeVisible();
  expect(enableCalls).toBe(1);
});

test("staff account re-enable keeps the dialog open and explains an inactive employee conflict", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 519, school_id: 10, username: "inactive.enable",
    employee_code: "EMP-INACTIVE-ENABLE", full_name: "Inactive Enable", designation: "Teacher",
    mobile_number: "9876543218", employment_status: "active", profile_complete: true,
    status: "disabled", must_change_password: false,
  }];
  let enableCalls = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.update"]);
  await page.route("**/api/school/staff/519/enable", async route => {
    if (route.request().method() !== "POST") { await route.fallback(); return; }
    enableCalls += 1;
    await route.fulfill({
      status: 409,
      contentType: "application/json",
      body: JSON.stringify({
        status: "error",
        message: "Reactivate the employee profile before enabling this login account",
      }),
    });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  const staffRow = page.locator("tr").filter({ hasText: "inactive.enable" });
  await expect(staffRow).toBeVisible();
  await staffRow.getByRole("button", { name: "Enable inactive.enable" }).click();
  const enableDialog = page.getByRole("dialog", { name: "Enable staff account" });
  await enableDialog.getByLabel("Temporary password").fill("EnableTempPass!2026");
  await enableDialog.getByRole("button", { name: "Enable account", exact: true }).click();

  await expect(enableDialog).toBeVisible();
  await expect(enableDialog.getByText("Reactivate the employee profile before enabling this login account")).toBeVisible();
  expect(enableCalls).toBe(1);
  await expect(staffRow).toContainText("disabled");
  await expect(page.getByText("Staff account enabled. A password change is required at next sign-in.")).toHaveCount(0);
});

test("staff profile edit uses the shared employment-status select and sends the selected value", async ({ page }) => {
  const updates: Array<Record<string, unknown>> = [];
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 301, school_id: 10, username: "teacher.two",
    employee_code: "EMP-002", full_name: "Teacher Two", designation: "Science Teacher",
    mobile_number: "9876543211", email: "teacher.two@example.com", joining_date: "2026-04-01",
    employment_status: "active", staff_type: "teaching", profile_complete: true, status: "active",
    must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  const browser = page.context().browser();
  if (!browser) throw new Error("Browser context is unavailable.");
  const timezoneContext = await browser.newContext({ baseURL: "http://127.0.0.1:4173", timezoneId: "America/Los_Angeles" });
  const timezonePage = await timezoneContext.newPage();
  await signInAndOpenStaff(timezonePage, rows, async (_path, route) => {
    updates.push(route.request().postDataJSON() as Record<string, unknown>);
    await route.fulfill({ json: { staff: { ...rows[0], ...(updates[0] ?? {}) }, message: "Updated" } });
  });

  await timezonePage.getByRole("button", { name: /Staff Directory/ }).click();
  const teacherRow = timezonePage.locator("tr").filter({ hasText: "teacher.two" });
  await expect(teacherRow).toContainText("1 Apr 2026");
  await timezonePage.getByRole("button", { name: "Teacher Two", exact: true }).click();
  const profileDialog = timezonePage.getByRole("dialog", { name: "Teacher Two" });
  await profileDialog.getByRole("button", { name: "Employment" }).click();
  await expect(profileDialog.getByText("1 Apr 2026")).toBeVisible();
  await profileDialog.getByRole("button", { name: "Close dialog" }).click();
  await timezonePage.getByRole("button", { name: "Edit employee profile for teacher.two" }).click();
  const dialog = timezonePage.getByRole("dialog", { name: "Edit employee profile" });
  await expect(dialog.getByRole("button", { name: "Joining date" })).toContainText("01 Apr 2026");
  await dialog.getByRole("button", { name: "Joining date" }).click();
  const joiningCalendar = dialog.getByRole("dialog", { name: "Choose joining date" });
  await joiningCalendar.getByRole("button", { name: "Today", exact: true }).click();
  await expect(dialog.locator('input[name="joining_date"]')).toHaveValue(schoolDateValue());
  await dialog.getByLabel(/Email address/).fill("");
  await dialog.getByRole("button", { name: "Employment status" }).click();
  await dialog.getByRole("listbox", { name: "Employment status" }).getByRole("option", { name: "Inactive" }).click();
  await dialog.getByRole("button", { name: "Save changes" }).click();

  await expect.poll(() => updates.length).toBe(1);
  expect(updates[0].employment_status).toBe("inactive");
  expect(updates[0].joining_date).toBe(schoolDateValue());
  expect(updates[0].email).toBe("");
  await expect(dialog).toHaveCount(0);
  await expect(timezonePage.locator("select")).toHaveCount(0);
  await timezoneContext.close();
});


test("staff profile edit preserves entered values after a server failure and succeeds on retry", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 302, school_id: 10, username: "teacher.retry",
    employee_code: "EMP-RETRY-01", full_name: "Retry Teacher", designation: "Science Teacher",
    mobile_number: "9876543212", email: "retry.teacher@example.com", joining_date: "2025-04-01",
    employment_status: "active", staff_type: "teaching", profile_complete: true, status: "active",
    must_change_password: false,
  }];
  let attempts = 0;
  const payloads: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    attempts += 1;
    payloads.push(route.request().postDataJSON() as Record<string, unknown>);
    if (attempts === 1) {
      await route.fulfill({ status: 503, json: { message: "Staff profile is temporarily unavailable. Please try again." } });
      return;
    }
    Object.assign(rows[0], payloads[1]);
    await route.fulfill({ json: { staff: rows[0], message: "Updated" } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Edit employee profile for teacher.retry" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit employee profile" });
  await dialog.getByLabel(/Full name/).fill("Retry Teacher Updated");
  await dialog.getByLabel(/Mobile number/).fill("9876543299");
  await dialog.getByRole("button", { name: "Save changes" }).click();

  await expect(dialog).toBeVisible();
  await expect(dialog.getByText("Staff profile is temporarily unavailable. Please try again.")).toBeVisible();
  await expect(dialog.getByLabel(/Full name/)).toHaveValue("Retry Teacher Updated");
  await expect(dialog.getByLabel(/Mobile number/)).toHaveValue("9876543299");
  expect(attempts).toBe(1);

  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Retry Teacher Updated" })).toBeVisible();
  expect(attempts).toBe(2);
  expect(payloads[0]).toMatchObject({ full_name: "Retry Teacher Updated", mobile_number: "9876543299" });
  expect(payloads[1]).toMatchObject({ full_name: "Retry Teacher Updated", mobile_number: "9876543299" });
});

test("staff profile edit skips the API and shows a no-changes toast when values are unchanged", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 303, school_id: 10, username: "teacher.unchanged",
    employee_code: "EMP-UNCHANGED", full_name: "Unchanged Teacher", designation: "English Teacher",
    mobile_number: "9876543213", email: "unchanged.teacher@example.com", joining_date: "2025-04-01",
    employment_status: "active", staff_type: "teaching", profile_complete: true, status: "active",
    must_change_password: false,
  }];
  let updateRequests = 0;
  await signInAndOpenStaff(page, rows, async (_path, route) => {
    updateRequests += 1;
    await route.fulfill({ json: { staff: rows[0], message: "Unexpected update" } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Edit employee profile for teacher.unchanged" }).click();
  const dialog = page.getByRole("dialog", { name: "Edit employee profile" });
  await dialog.getByRole("button", { name: "Save changes" }).click();

  await expect(dialog).toBeVisible();
  await expect(page.getByText("No changes to save.", { exact: true })).toBeVisible();
  expect(updateRequests).toBe(0);
});

test("unchanged qualification, certification and experience edits skip API requests", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 620, school_id: 10, username: "teacher.record-unchanged",
    employee_code: "EMP-RECORD-UNCHANGED", full_name: "Unchanged Records Teacher",
    designation: "Teacher", mobile_number: "9876543222", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  const savedRecords: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = {
    qualifications: [{
      id: 921, staff_principal_id: 620, qualification_type: "undergraduate",
      qualification_name: "B.Ed.", specialization: "", institution: "State University",
      board_university: "", completion_year: null, score: null, score_scale: null,
      credential_reference: "", verification_status: "pending",
    }],
    certifications: [{
      id: 922, staff_principal_id: 620, certificate_name: "First Aid",
      issuing_organization: "State Board", issue_date: null, expiry_date: null,
      credential_id: "", verification_status: "pending",
    }],
    experience: [{
      id: 923, staff_principal_id: 620, employer_name: "Previous School",
      designation: "Teacher", employment_type: "permanent", start_date: "2019-04-01",
      end_date: null, is_current: false, description: "", verification_status: "pending",
    }],
  };
  let mutationRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (_collection, route) => {
    mutationRequests += 1;
    await route.fulfill({ status: 500, json: { message: "An unchanged record must not be submitted." } });
  }, undefined, savedRecords);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Unchanged Records Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Unchanged Records Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();

  for (const [collection, recordLabel] of [
    ["qualifications", "B.Ed."],
    ["certifications", "First Aid"],
    ["experience", "Previous School"],
  ] as const) {
    const recordCard = dialog.locator(".staff-record-card").filter({ hasText: recordLabel });
    await recordCard.getByRole("button", { name: "Edit" }).click();
    await dialog.getByRole("button", { name: "Save changes" }).click();
    await expect(page.getByText("No changes to save.", { exact: true })).toBeVisible();
    expect(mutationRequests, collection).toBe(0);
    await expect(recordCard).toBeVisible();
  }
});

test("staff record save prevents duplicate submissions while the request is pending", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 621, school_id: 10, username: "teacher.record-pending",
    employee_code: "EMP-RECORD-PENDING", full_name: "Pending Record Teacher",
    designation: "Teacher", mobile_number: "9876543223", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  const recordRows: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = {
    qualifications: [], certifications: [], experience: [],
  };
  let mutationRequests = 0;
  let releaseRequest!: () => void;
  const requestGate = new Promise<void>(resolve => { releaseRequest = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (collection, route) => {
    mutationRequests += 1;
    expect(collection).toBe("qualifications");
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    await requestGate;
    const saved = {
      ...payload, id: 924, staff_principal_id: 621, verification_status: "pending",
      created_at: "2026-11-21T00:00:00.000Z", updated_at: "2026-11-21T00:00:00.000Z",
    };
    recordRows.qualifications.push(saved);
    await route.fulfill({ status: 201, json: { qualification: saved } });
  }, undefined, recordRows);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Pending Record Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Pending Record Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add qualification" }).click();
  await dialog.locator('input[name="qualification_name"]').fill("B.Ed.");
  await dialog.locator('input[name="institution"]').fill("State University");

  const saveButton = dialog.getByRole("button", { name: "Save record" });
  await saveButton.click();
  await expect(dialog.getByRole("button", { name: "Saving…" })).toBeDisabled();
  expect(mutationRequests).toBe(1);
  await expect(dialog.getByRole("button", { name: "Close dialog" })).toBeDisabled();
  await expect(dialog.locator(".staff-record-form-heading").getByRole("button", { name: "Cancel" })).toBeDisabled();
  await expect(dialog.locator(".staff-record-form-actions").getByRole("button", { name: "Cancel" })).toBeDisabled();
  await expect(dialog.locator("form.staff-record-form")).toHaveAttribute("inert", "");
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
  await page.locator(".app-modal-backdrop").last().click({ position: { x: 4, y: 4 } });
  await expect(dialog).toBeVisible();

  await dialog.locator("form.staff-record-form").evaluate(form => (form as HTMLFormElement).requestSubmit());
  expect(mutationRequests).toBe(1);

  releaseRequest();
  await expect(dialog.getByText("B.Ed.")).toBeVisible();
  await expect(dialog.getByRole("button", { name: "Close dialog" })).toBeEnabled();
  expect(mutationRequests).toBe(1);
});

test("staff record server field errors render under the matching field and preserve entered values", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 612, school_id: 10, username: "teacher.record-validation",
    employee_code: "EMP-REC-02", full_name: "Record Validation Teacher", designation: "Science Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.update"], async (collection, route) => {
    expect(collection).toBe("qualifications");
    await route.fulfill({
      status: 400,
      json: {
        status: "error",
        message: "Check the highlighted staff record fields",
        field_errors: { qualification_name: "The qualification name needs a valid value." },
      },
    });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("1–1 of 1 staff members")).toBeVisible();
  const staffRow = page.locator("tr").filter({ hasText: "teacher.record-validation" });
  await expect(staffRow).toBeVisible();
  await staffRow.locator("button.staff-profile-link").click();
  const dialog = page.getByRole("dialog", { name: "Record Validation Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add qualification" }).click();
  await dialog.locator('input[name="qualification_name"]').fill("M.Sc. Physics");
  await dialog.locator('input[name="institution"]').fill("State University");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("The qualification name needs a valid value.")).toBeVisible();
  await expect(dialog.locator('input[name="qualification_name"]')).toHaveValue("M.Sc. Physics");
  await expect(dialog.locator('input[name="qualification_name"]')).toHaveAttribute("aria-invalid", "true");
  await expect(dialog.locator(".staff-record-form").getByText("Check the highlighted staff record fields")).toHaveCount(0);
  await dialog.locator('input[name="qualification_name"]').fill("M.Sc. Corrected Physics");
  await expect(dialog.getByText("The qualification name needs a valid value.")).toHaveCount(0);
  await expect(dialog.locator('input[name="qualification_name"]')).not.toHaveAttribute("aria-invalid", "true");
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
  const qualificationNameInput = dialog.locator('input[name="qualification_name"]');
  await qualificationNameInput.fill("Q".repeat(161));
  await expect(qualificationNameInput).toHaveValue("Q".repeat(160));
  await qualificationNameInput.fill("M.Sc. Mathematics");
  const specializationInput = dialog.locator('input[name="specialization"]');
  await specializationInput.fill("🔐".repeat(161));
  await expect(specializationInput).toHaveValue("🔐".repeat(160));
  await dialog.locator('input[name="institution"]').fill("State University");
  const unicodeCredential = "🔐".repeat(120);
  await expect(dialog.locator('input[name="credential_reference"]')).not.toHaveAttribute("maxLength");
  await dialog.locator('input[name="credential_reference"]').fill("🔐".repeat(121));
  await expect(dialog.locator('input[name="credential_reference"]')).toHaveValue(unicodeCredential);
  await dialog.locator('input[name="completion_year"]').fill("1899");
  await dialog.locator('input[name="score"]').fill("8.5");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Enter a year between 1900 and 2200.")).toBeVisible();
  await expect(dialog.getByText("Choose a score scale when a score is entered.")).toBeVisible();
  await dialog.locator('input[name="completion_year"]').fill("2021");
  await dialog.locator('select[name="score_scale"]').selectOption("percentage");
  await dialog.locator('input[name="score"]').fill("100.001");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Percentage cannot exceed 100.")).toBeVisible();
  await dialog.locator('input[name="score"]').fill("8.5");
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
  const certificateNameInput = dialog.locator('input[name="certificate_name"]');
  await certificateNameInput.fill("C".repeat(181));
  await expect(certificateNameInput).toHaveValue("C".repeat(180));
  await certificateNameInput.fill("Teacher Training");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await expect(dialog.locator('input[name="credential_id"]')).not.toHaveAttribute("maxLength");
  await dialog.locator('input[name="credential_id"]').fill("🔐".repeat(121));
  await expect(dialog.locator('input[name="credential_id"]')).toHaveValue(unicodeCredential);
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Teacher Training")).toBeVisible();
  const certificationCard = dialog.locator(".staff-record-card").filter({ hasText: "Teacher Training" });
  await certificationCard.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Advanced Teacher Training");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Advanced Teacher Training")).toBeVisible();

  await dialog.getByRole("button", { name: "Add experience" }).click();
  const employerNameInput = dialog.locator('input[name="employer_name"]');
  await employerNameInput.fill("E".repeat(201));
  await expect(employerNameInput).toHaveValue("E".repeat(200));
  await employerNameInput.fill("Previous School");
  const recordDesignationInput = dialog.locator('input[name="record_designation"]');
  await recordDesignationInput.fill("D".repeat(121));
  await expect(recordDesignationInput).toHaveValue("D".repeat(120));
  await recordDesignationInput.fill("Science Teacher");
  const experienceDescription = dialog.locator('textarea[name="description"]');
  await experienceDescription.fill("X".repeat(2001));
  await expect(experienceDescription).toHaveValue("X".repeat(2000));
  await chooseDatePickerValue(dialog, "Previous employment start date", "2012-04-01");
  await dialog.getByRole("button", { name: "Previous employment end date" }).click();
  const experienceEndCalendar = dialog.getByRole("dialog", { name: "Choose previous employment end date" });
  await experienceEndCalendar.getByRole("button", { name: "Choose year" }).click();
  await expect(experienceEndCalendar.locator(".app-date-year-grid button")).toHaveCount(10);
  await page.keyboard.press("Escape");
  await chooseDatePickerValue(dialog, "Previous employment end date", "2022-03-31");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Previous School")).toBeVisible();
  const experienceCard = dialog.locator(".staff-record-card").filter({ hasText: "Previous School" });
  await experienceCard.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="record_designation"]').fill("Senior Science Teacher");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Senior Science Teacher")).toBeVisible();
});

test("staff qualification, certification and experience edits preserve values after failure and retry", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 619, school_id: 10, username: "teacher.record-edit-retry",
    employee_code: "EMP-RECORD-EDIT", full_name: "Record Edit Retry Teacher",
    designation: "Teacher", mobile_number: "9876543221", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  const savedRecords: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = {
    qualifications: [{ id: 911, staff_principal_id: 619, qualification_type: "undergraduate", qualification_name: "B.Ed.", institution: "Old University", verification_status: "pending" }],
    certifications: [{ id: 912, staff_principal_id: 619, certificate_name: "First Aid", issuing_organization: "Old Board", verification_status: "pending" }],
    experience: [{ id: 913, staff_principal_id: 619, employer_name: "Old School", designation: "Teacher", employment_type: "permanent", start_date: "2019-04-01", end_date: "2022-03-31", is_current: false, verification_status: "pending" }],
  };
  const attempts: Record<string, number> = { qualifications: 0, certifications: 0, experience: 0 };
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (collection, route) => {
    expect(route.request().method()).toBe("PATCH");
    attempts[collection] += 1;
    const payload = route.request().postDataJSON() as Record<string, unknown>;
    if (attempts[collection] === 1) {
      await route.fulfill({ status: 503, json: { message: "Record update is temporarily unavailable. Please retry." } });
      return;
    }
    const recordID = Number(new URL(route.request().url()).pathname.split("/").pop());
    const target = savedRecords[collection as keyof typeof savedRecords].find(item => item.id === recordID);
    expect(target).toBeTruthy();
    Object.assign(target!, payload);
    await route.fulfill({ status: 200, json: {
      [collection === "qualifications" ? "qualification" : collection === "certifications" ? "certification" : "experience"]: target,
    } });
  }, undefined, savedRecords);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Record Edit Retry Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Record Edit Retry Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();

  const qualification = dialog.locator(".staff-record-card").filter({ hasText: "B.Ed." });
  await qualification.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="institution"]').fill("Updated University");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Record update is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="institution"]')).toHaveValue("Updated University");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Updated University")).toBeVisible();

  const certification = dialog.locator(".staff-record-card").filter({ hasText: "First Aid" });
  await certification.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="issuing_organization"]').fill("Updated Board");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Record update is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="issuing_organization"]')).toHaveValue("Updated Board");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Updated Board")).toBeVisible();

  const experience = dialog.locator(".staff-record-card").filter({ hasText: "Old School" });
  await experience.getByRole("button", { name: "Edit" }).click();
  await dialog.locator('input[name="record_designation"]').fill("Senior Teacher");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Record update is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="record_designation"]')).toHaveValue("Senior Teacher");
  await dialog.getByRole("button", { name: "Save changes" }).click();
  await expect(dialog.getByText("Senior Teacher")).toBeVisible();

  expect(attempts).toEqual({ qualifications: 2, certifications: 2, experience: 2 });
});

test("staff qualification form preserves values after a failed save and succeeds on retry", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 616, school_id: 10, username: "teacher.qualification-retry",
    employee_code: "EMP-QUAL-RETRY", full_name: "Qualification Retry Teacher",
    designation: "Teacher", mobile_number: "9876543218", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  let attempts = 0;
  const payloads: Array<Record<string, unknown>> = [];
  const savedRecords: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = { qualifications: [], certifications: [], experience: [] };
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (collection, route) => {
    expect(collection).toBe("qualifications");
    attempts += 1;
    payloads.push(route.request().postDataJSON() as Record<string, unknown>);
    if (attempts === 1) {
      await route.fulfill({ status: 503, json: { message: "Qualification service is temporarily unavailable. Please retry." } });
      return;
    }
    const saved = { id: 901, staff_principal_id: rows[0].principal_id, ...payloads[1], verification_status: "pending", created_at: "2026-11-27T00:00:00.000Z", updated_at: "2026-11-27T00:00:00.000Z" };
    savedRecords.qualifications.push(saved);
    await route.fulfill({ status: 201, json: { qualification: saved } });
  }, undefined, savedRecords);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Qualification Retry Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Qualification Retry Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add qualification" }).click();
  await dialog.locator('input[name="qualification_name"]').fill("B.Ed. Retry");
  await dialog.locator('input[name="institution"]').fill("State University");
  await dialog.getByRole("button", { name: "Save record" }).click();

  await expect(dialog.getByText("Qualification service is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="qualification_name"]')).toHaveValue("B.Ed. Retry");
  await expect(dialog.locator('input[name="institution"]')).toHaveValue("State University");
  expect(attempts).toBe(1);

  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("B.Ed. Retry")).toBeVisible();
  expect(attempts).toBe(2);
  expect(payloads[0]).toMatchObject({ qualification_name: "B.Ed. Retry", institution: "State University" });
  expect(payloads[1]).toMatchObject({ qualification_name: "B.Ed. Retry", institution: "State University" });
});

test("staff certification form preserves values after a failed save and succeeds on retry", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 617, school_id: 10, username: "teacher.certification-retry",
    employee_code: "EMP-CERT-RETRY", full_name: "Certification Retry Teacher",
    designation: "Teacher", mobile_number: "9876543219", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  let attempts = 0;
  const payloads: Array<Record<string, unknown>> = [];
  const savedRecords: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = { qualifications: [], certifications: [], experience: [] };
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (collection, route) => {
    expect(collection).toBe("certifications");
    attempts += 1;
    payloads.push(route.request().postDataJSON() as Record<string, unknown>);
    if (attempts === 1) {
      await route.fulfill({ status: 503, json: { message: "Certification service is temporarily unavailable. Please retry." } });
      return;
    }
    const saved = { id: 902, staff_principal_id: rows[0].principal_id, ...payloads[1], verification_status: "pending", created_at: "2026-11-27T00:00:00.000Z", updated_at: "2026-11-27T00:00:00.000Z" };
    savedRecords.certifications.push(saved);
    await route.fulfill({ status: 201, json: { certification: saved } });
  }, undefined, savedRecords);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Certification Retry Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Certification Retry Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add certification" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Child Safety Training");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await dialog.getByRole("button", { name: "Save record" }).click();

  await expect(dialog.getByText("Certification service is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="certificate_name"]')).toHaveValue("Child Safety Training");
  await expect(dialog.locator('input[name="issuing_organization"]')).toHaveValue("State Board");
  expect(attempts).toBe(1);

  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Child Safety Training")).toBeVisible();
  expect(attempts).toBe(2);
  expect(payloads[0]).toMatchObject({ certificate_name: "Child Safety Training", issuing_organization: "State Board" });
  expect(payloads[1]).toMatchObject({ certificate_name: "Child Safety Training", issuing_organization: "State Board" });
});

test("staff experience form preserves values after a failed save and succeeds on retry", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 618, school_id: 10, username: "teacher.experience-retry",
    employee_code: "EMP-EXP-RETRY", full_name: "Experience Retry Teacher",
    designation: "Teacher", mobile_number: "9876543220", employment_status: "active",
    profile_complete: true, status: "active", must_change_password: false,
  }];
  let attempts = 0;
  const payloads: Array<Record<string, unknown>> = [];
  const savedRecords: Record<"qualifications" | "certifications" | "experience", Array<Record<string, unknown>>> = { qualifications: [], certifications: [], experience: [] };
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (collection, route) => {
    expect(collection).toBe("experience");
    attempts += 1;
    payloads.push(route.request().postDataJSON() as Record<string, unknown>);
    if (attempts === 1) {
      await route.fulfill({ status: 503, json: { message: "Experience service is temporarily unavailable. Please retry." } });
      return;
    }
    const saved = { id: 903, staff_principal_id: rows[0].principal_id, ...payloads[1], verification_status: "pending", created_at: "2026-11-27T00:00:00.000Z", updated_at: "2026-11-27T00:00:00.000Z" };
    savedRecords.experience.push(saved);
    await route.fulfill({ status: 201, json: { experience: saved } });
  }, undefined, savedRecords);

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Experience Retry Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Experience Retry Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add experience" }).click();
  await dialog.locator('input[name="employer_name"]').fill("Previous School");
  await dialog.locator('input[name="record_designation"]').fill("Science Teacher");
  await chooseDatePickerValue(dialog, "Previous employment start date", "2019-04-01");
  await dialog.getByRole("button", { name: "Save record" }).click();

  await expect(dialog.getByText("Experience service is temporarily unavailable. Please retry.")).toBeVisible();
  await expect(dialog.locator('input[name="employer_name"]')).toHaveValue("Previous School");
  await expect(dialog.locator('input[name="record_designation"]')).toHaveValue("Science Teacher");
  await expect(dialog.locator('input[name="start_date"]')).toHaveValue("2019-04-01");
  expect(attempts).toBe(1);

  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Previous School")).toBeVisible();
  expect(attempts).toBe(2);
  expect(payloads[0]).toMatchObject({ employer_name: "Previous School", designation: "Science Teacher", start_date: "2019-04-01" });
  expect(payloads[1]).toMatchObject({ employer_name: "Previous School", designation: "Science Teacher", start_date: "2019-04-01" });
});

test("staff certification rejects impossible calendar dates before saving", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 612, school_id: 10, username: "teacher.date-check",
    employee_code: "EMP-DATE-01", full_name: "Date Check Teacher", designation: "Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Date Check Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Date Check Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add certification" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Impossible Date Certificate");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await dialog.locator('input[name="issue_date"]').evaluate((element: HTMLInputElement) => {
    element.value = "2026-02-30";
  });
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Enter a real calendar issue date.")).toBeVisible();
  await expect(dialog.locator('input[name="certificate_name"]')).toHaveValue("Impossible Date Certificate");
});


test("staff certification saves valid issue and expiry dates selected from the date pickers", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 613, school_id: 10, username: "teacher.valid-date",
    employee_code: "EMP-DATE-02", full_name: "Valid Date Teacher", designation: "Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Valid Date Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Valid Date Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add certification" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Valid Calendar Certificate");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await chooseDatePickerValue(dialog, "Certification issue date", "2020-01-15");
  await chooseDatePickerValue(dialog, "Certification expiry date", "2025-01-15");
  await expect(dialog.locator('input[name="issue_date"]')).toHaveValue("2020-01-15");
  await expect(dialog.locator('input[name="expiry_date"]')).toHaveValue("2025-01-15");
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Valid Calendar Certificate")).toBeVisible();
  const card = dialog.locator(".staff-record-card").filter({ hasText: "Valid Calendar Certificate" });
  await expect(card).toContainText("2020-01-15");
  await expect(card).toContainText("2025-01-15");
});


test("staff certification rejects expiry dates earlier than issue dates without an API request", async ({ page }) => {
  let mutationCalls = 0;
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 614, school_id: 10, username: "teacher.date-order",
    employee_code: "EMP-DATE-03", full_name: "Date Order Teacher", designation: "Teacher",
    mobile_number: "9876543216", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (_collection, route) => {
    mutationCalls += 1;
    await route.fulfill({ status: 201, json: { certification: { id: 1 } } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Date Order Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Date Order Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add certification" }).click();
  await dialog.locator('input[name="certificate_name"]').fill("Chronology Check Certificate");
  await dialog.locator('input[name="issuing_organization"]').fill("State Board");
  await dialog.locator('input[name="issue_date"]').evaluate((element: HTMLInputElement) => { element.value = "2025-06-01"; });
  await dialog.locator('input[name="expiry_date"]').evaluate((element: HTMLInputElement) => { element.value = "2025-05-31"; });
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("Expiry date cannot be before the issue date.")).toBeVisible();
  expect(mutationCalls).toBe(0);
  await expect(dialog.locator('input[name="certificate_name"]')).toHaveValue("Chronology Check Certificate");
});

test("staff experience rejects end dates before start dates without an API request", async ({ page }) => {
  let mutationCalls = 0;
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 615, school_id: 10, username: "teacher.experience-order",
    employee_code: "EMP-EXP-01", full_name: "Experience Order Teacher", designation: "Teacher",
    mobile_number: "9876543217", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update"], async (_collection, route) => {
    mutationCalls += 1;
    await route.fulfill({ status: 201, json: { experience: { id: 1 } } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Experience Order Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Experience Order Teacher" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await dialog.getByRole("button", { name: "Add experience" }).click();
  await dialog.locator('input[name="employer_name"]').fill("Previous School");
  await dialog.locator('input[name="record_designation"]').fill("Teacher");
  await dialog.locator('input[name="start_date"]').evaluate((element: HTMLInputElement) => { element.value = "2024-08-01"; });
  await dialog.locator('input[name="end_date"]').evaluate((element: HTMLInputElement) => { element.value = "2024-07-31"; });
  await dialog.getByRole("button", { name: "Save record" }).click();
  await expect(dialog.getByText("End date must be a real calendar date on or after the start date.")).toBeVisible();
  expect(mutationCalls).toBe(0);
  await expect(dialog.locator('input[name="employer_name"]')).toHaveValue("Previous School");
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
  const expectedAssignmentDate = new Intl.DateTimeFormat("en-IN", { dateStyle: "medium", timeZone: "Asia/Kolkata" }).format(new Date("2026-04-01T00:00:00.000Z"));
  await expect(dialog.getByText(`Assigned ${expectedAssignmentDate}`).first()).toBeVisible();
  await expect(dialog.getByText(/Revoked ·/)).toBeVisible();
  await expect(dialog.getByText("2 active")).toBeVisible();
});

test("staff profile clears the previous employee's academic assignments while the next profile loads", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 411, school_id: 10, username: "teacher.first", employee_code: "EMP-FIRST", full_name: "First Teacher", designation: "Teacher", mobile_number: "9876543212", employment_status: "active", profile_complete: true, status: "active" },
    { id: 2, principal_id: 412, school_id: 10, username: "teacher.second", employee_code: "EMP-SECOND", full_name: "Second Teacher", designation: "Teacher", mobile_number: "9876543213", employment_status: "active", profile_complete: true, status: "active" },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "teacher_assignments.read"]);
  await page.route("**/api/school/teacher-assignments?*", async route => {
    const teacherID = new URL(route.request().url()).searchParams.get("teacher_principal_id");
    if (teacherID === "411") await new Promise(resolve => setTimeout(resolve, 500));
    if (teacherID === "412") await new Promise(resolve => setTimeout(resolve, 300));
    try {
      await route.fulfill({ json: { assignments: teacherID === "411" ? [
        { id: 1, academic_year_id: 3, class_name: "Class 8", section_name: "A", subject_name: "First Teacher Private Subject", assigned_at: "2026-04-01T00:00:00.000Z", teacher_principal_id: 411, revoked_at: null },
      ] : [
        { id: 2, academic_year_id: 3, class_name: "Class 9", section_name: "B", subject_name: "Second Teacher Subject", assigned_at: "2026-04-01T00:00:00.000Z", teacher_principal_id: 412, revoked_at: null },
      ] } });
    } catch {
      // The first employee's in-flight request is intentionally aborted on profile change.
    }
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "First Teacher" }).click();
  let dialog = page.getByRole("dialog", { name: "First Teacher" });
  const firstAssignmentRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return url.pathname.endsWith("/teacher-assignments") && url.searchParams.get("teacher_principal_id") === "411";
  });
  await dialog.getByRole("button", { name: "Academic Assignments" }).click();
  await firstAssignmentRequest;
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Second Teacher" }).click();
  dialog = page.getByRole("dialog", { name: "Second Teacher" });
  const secondAssignmentRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return url.pathname.endsWith("/teacher-assignments") && url.searchParams.get("teacher_principal_id") === "412";
  });
  await dialog.getByRole("button", { name: "Academic Assignments" }).click();
  await secondAssignmentRequest;
  await expect(dialog.getByRole("status")).toContainText("Loading academic assignments");
  await expect(dialog.getByText("First Teacher Private Subject")).toHaveCount(0);
  await expect(dialog.getByText("Second Teacher Subject")).toBeVisible();
  await page.waitForTimeout(600);
  await expect(dialog.getByText("Second Teacher Subject")).toBeVisible();
  await expect(dialog.getByText("First Teacher Private Subject")).toHaveCount(0);
});


test("staff profile does not retain the previous employee's timetable while the next workload loads", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 421, school_id: 10, username: "teacher.load.first", employee_code: "EMP-LOAD-FIRST", full_name: "Workload First", designation: "Teacher", mobile_number: "9876543214", employment_status: "active", profile_complete: true, status: "active" },
    { id: 2, principal_id: 422, school_id: 10, username: "teacher.load.second", employee_code: "EMP-LOAD-SECOND", full_name: "Workload Second", designation: "Teacher", mobile_number: "9876543215", employment_status: "active", profile_complete: true, status: "active" },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "timetable.read"]);
  await page.route("**/api/school/timetable/current-year", async route => {
    await route.fulfill({ json: { academic_year: { id: 3, name: "2026–27", starts_on: "2026-04-01", ends_on: "2027-03-31", status: "active", is_current: true } } });
  });
  await page.route("**/api/school/timetable/periods?*", async route => {
    await route.fulfill({ json: { periods: [{ id: 7, school_id: 10, academic_year_id: 3, period_number: 1, name: "Period 1", starts_at: "09:00", ends_at: "09:45", is_break: false }] } });
  });
  await page.route("**/api/school/timetable/entries?*", async route => {
    const teacherID = new URL(route.request().url()).searchParams.get("teacher_principal_id");
    if (teacherID === "421") await new Promise(resolve => setTimeout(resolve, 500));
    if (teacherID === "422") await new Promise(resolve => setTimeout(resolve, 300));
    try {
      await route.fulfill({ json: { entries: teacherID === "421" ? [
        { id: 1, school_id: 10, academic_year_id: 3, section_id: 8, section_name: "A", class_name: "Class 8", subject_id: 9, subject_name: "First Teacher Timetable Secret", teacher_principal_id: 421, teacher_username: "teacher.load.first", teacher_subject_assignment_id: 1, period_id: 7, period_number: 1, period_name: "Period 1", starts_at: "09:00", ends_at: "09:45", day_of_week: 1 },
      ] : [
        { id: 2, school_id: 10, academic_year_id: 3, section_id: 9, section_name: "B", class_name: "Class 9", subject_id: 10, subject_name: "Second Teacher Timetable", teacher_principal_id: 422, teacher_username: "teacher.load.second", teacher_subject_assignment_id: 2, period_id: 7, period_number: 1, period_name: "Period 1", starts_at: "09:00", ends_at: "09:45", day_of_week: 1 },
      ] } });
    } catch {
      // The first employee's in-flight request is intentionally aborted on profile change.
    }
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Workload First" }).click();
  let dialog = page.getByRole("dialog", { name: "Workload First" });
  const firstTimetableRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return url.pathname.endsWith("/timetable/entries") && url.searchParams.get("teacher_principal_id") === "421";
  });
  await dialog.getByRole("button", { name: "Workload & Timetable" }).click();
  await firstTimetableRequest;
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Workload Second" }).click();
  dialog = page.getByRole("dialog", { name: "Workload Second" });
  const secondTimetableRequest = page.waitForRequest(request => {
    const url = new URL(request.url());
    return url.pathname.endsWith("/timetable/entries") && url.searchParams.get("teacher_principal_id") === "422";
  });
  await dialog.getByRole("button", { name: "Workload & Timetable" }).click();
  await secondTimetableRequest;
  await expect(dialog.getByRole("status")).toContainText("Loading teacher-scoped timetable");
  await expect(dialog.getByText("First Teacher Timetable Secret")).toHaveCount(0);
  await expect(dialog.getByText("Second Teacher Timetable")).toBeVisible();
  await page.waitForTimeout(600);
  await expect(dialog.getByText("Second Teacher Timetable")).toBeVisible();
  await expect(dialog.getByText("First Teacher Timetable Secret")).toHaveCount(0);
});

test("staff profile does not retain the previous employee's qualifications while the next profile loads", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 431, school_id: 10, username: "teacher.qualification.first", employee_code: "EMP-QUAL-FIRST", full_name: "Qualification First", designation: "Teacher", mobile_number: "9876543216", employment_status: "active", profile_complete: true, status: "active" },
    { id: 2, principal_id: 432, school_id: 10, username: "teacher.qualification.second", employee_code: "EMP-QUAL-SECOND", full_name: "Qualification Second", designation: "Teacher", mobile_number: "9876543217", employment_status: "active", profile_complete: true, status: "active" },
  ];
  await signInAndOpenStaff(page, rows);
  await page.route("**/api/school/staff/*/qualifications", async route => {
    const principalID = new URL(route.request().url()).pathname.split("/")[4];

    if (principalID === "432") await new Promise(resolve => setTimeout(resolve, 400));
    const qualification = principalID === "431"
      ? { id: 1, staff_principal_id: 431, qualification_name: "First Employee Private Qualification", institution: "First Teacher College", verification_status: "verified" }
      : { id: 2, staff_principal_id: 432, qualification_name: "Second Employee Qualification", institution: "Second Teacher College", verification_status: "verified" };
    await route.fulfill({ json: { qualifications: [qualification] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Qualification First" }).click();
  let dialog = page.getByRole("dialog", { name: "Qualification First" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await expect(dialog.getByText("First Employee Private Qualification")).toBeVisible();

  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Qualification Second" }).click();
  dialog = page.getByRole("dialog", { name: "Qualification Second" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await expect(dialog.getByRole("status").filter({ hasText: "Loading qualifications" })).toBeVisible();
  await expect(dialog.getByText("First Employee Private Qualification")).toHaveCount(0);
  await expect(dialog.getByText("Second Employee Qualification")).toBeVisible();
});

test("staff profile does not retain the previous employee's certifications or experience while the next profile loads", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 433, school_id: 10, username: "teacher.records.first", employee_code: "EMP-RECORDS-FIRST", full_name: "Records First", designation: "Teacher", mobile_number: "9876543218", employment_status: "active", profile_complete: true, status: "active" },
    { id: 2, principal_id: 434, school_id: 10, username: "teacher.records.second", employee_code: "EMP-RECORDS-SECOND", full_name: "Records Second", designation: "Teacher", mobile_number: "9876543219", employment_status: "active", profile_complete: true, status: "active" },
  ];
  await signInAndOpenStaff(page, rows);
  await page.route("**/api/school/staff/*/certifications", async route => {
    const principalID = new URL(route.request().url()).pathname.split("/")[4];
    if (principalID === "434") await new Promise(resolve => setTimeout(resolve, 400));
    const certification = principalID === "433"
      ? { id: 1, staff_principal_id: 433, certificate_name: "First Employee Private Certificate", issuing_organization: "First Certification Board", verification_status: "verified" }
      : { id: 2, staff_principal_id: 434, certificate_name: "Second Employee Certificate", issuing_organization: "Second Certification Board", verification_status: "verified" };
    await route.fulfill({ json: { certifications: [certification] } });
  });
  await page.route("**/api/school/staff/*/experience", async route => {
    const principalID = new URL(route.request().url()).pathname.split("/")[4];
    if (principalID === "434") await new Promise(resolve => setTimeout(resolve, 400));
    const experience = principalID === "433"
      ? { id: 1, staff_principal_id: 433, employer_name: "First Employee Private School", designation: "Senior Teacher First", employment_type: "permanent", start_date: "2018-04-01", end_date: "2021-03-31", is_current: false }
      : { id: 2, staff_principal_id: 434, employer_name: "Second Employee School", designation: "Teacher Second", employment_type: "contract", start_date: "2022-04-01", is_current: true };
    await route.fulfill({ json: { experience: [experience] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Records First" }).click();
  let dialog = page.getByRole("dialog", { name: "Records First" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await expect(dialog.getByText("First Employee Private Certificate")).toBeVisible();
  await expect(dialog.getByText("First Employee Private School")).toBeVisible();

  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Records Second" }).click();
  dialog = page.getByRole("dialog", { name: "Records Second" });
  await dialog.getByRole("button", { name: "Qualifications & Experience" }).click();
  await expect(dialog.getByRole("status").filter({ hasText: "Loading certifications" })).toBeVisible();
  await expect(dialog.getByRole("status").filter({ hasText: "Loading experience" })).toBeVisible();
  await expect(dialog.getByText("First Employee Private Certificate")).toHaveCount(0);
  await expect(dialog.getByText("First Employee Private School")).toHaveCount(0);
  await expect(dialog.getByText("Second Employee Certificate")).toBeVisible();
  await expect(dialog.getByText("Second Employee School")).toBeVisible();
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

test("protected peer School Admin account is read-only for role and permission changes", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 517, school_id: 10, username: "teacher.protected-admin",
    employee_code: "EMP-PROTECTED-ADMIN", full_name: "Protected Admin", designation: "School Admin",
    mobile_number: "9876543211", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const requests: Array<{ method: string; path: string }> = [];
  page.on("request", request => {
    const url = new URL(request.url());
    requests.push({ method: request.method(), path: url.pathname.replace(/^\/api/, "") });
  });
  await signInAndOpenStaff(page, rows, undefined, [
    "users.read", "roles.read", "roles.assign", "permissions.override",
  ]);
  await page.route("**/api/school/roles", async route => {
    await route.fulfill({ json: { roles: [
      { id: 4, role_key: "school_admin", name: "School Admin", is_system: true, created_at: "2026-01-01T00:00:00Z" },
      { id: 5, role_key: "teacher", name: "Teacher", is_system: true, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/517/roles**", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { roles: [
        { role_id: 4, role_key: "school_admin", name: "School Admin", is_system: true, assigned_at: "2026-01-01T00:00:00Z" },
      ] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Protected School Admin account." } });
  });
  await page.route("**/api/school/permissions", async route => {
    await route.fulfill({ json: { permissions: [
      { permission_key: "students.read_basic", description: "Read basic student information", is_delegable: true },
    ] } });
  });
  await page.route("**/api/school/permission-delegations", async route => {
    await route.fulfill({ json: { delegations: [
      { id: 31, school_id: 10, permission_key: "students.read_basic", can_assign_to_roles: false, can_override_users: true, granted_by: 2, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/517/permission-overrides**", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { overrides: [
        { id: 71, principal_id: 517, permission_key: "students.read_basic", effect: "deny", reason: "Existing policy", granted_by: 2, delegation_id: 31, created_at: "2026-01-01T00:00:00Z" },
      ] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Protected School Admin account." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Protected Admin" }).click();
  const profile = page.getByRole("dialog", { name: "Protected Admin" });
  await profile.getByRole("button", { name: "Account & Access" }).click();
  await expect(profile.getByText("Protected School Admin account: role changes are disabled.")).toBeVisible();
  await expect(profile.getByText("Protected School Admin account: individual permission overrides are disabled.")).toBeVisible();
  await expect(profile.getByRole("button", { name: "Assign", exact: true })).toHaveCount(0);
  await expect(profile.getByRole("button", { name: /Remove .* role/ })).toHaveCount(0);
  await expect(profile.getByRole("button", { name: "Save override" })).toHaveCount(0);
  await expect(profile.getByRole("button", { name: /Remove .* override/ })).toHaveCount(0);
  expect(requests.some(item =>
    (item.method === "POST" || item.method === "DELETE") &&
    (item.path.includes("/roles") || item.path.includes("permission-overrides")),
  )).toBe(false);
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
  let roleAssignmentReads = 0;
  let auditReads = 0;
  let releaseAssignment: (() => void) | null = null;
  let notifyAssignmentStarted: (() => void) | null = null;
  const assignmentStarted = new Promise<void>(resolve => { notifyAssignmentStarted = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "roles.read", "roles.assign", "school.audit.read"]);
  await page.route("**/api/school/staff/514/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });
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
      roleAssignmentReads += 1;
      await route.fulfill({ json: { roles: assignedRoles } });
      return;
    }
    if (path === "/school/staff/514/roles/5" && route.request().method() === "POST") {
      assignCalls += 1;
      await new Promise<void>(resolve => {
        releaseAssignment = resolve;
        notifyAssignmentStarted?.();
      });
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
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect(profile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstAuditReadCount = auditReads;
  await profile.getByRole("button", { name: "Account & Access" }).click();
  await expect(profile.getByText("Teacher", { exact: true })).toBeVisible();
  const firstRoleReadCount = roleAssignmentReads;
  expect(firstRoleReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Role Admin" }).click();
  const reopenedProfile = page.getByRole("dialog", { name: "Role Admin" });
  await reopenedProfile.getByRole("button", { name: "Account & Access" }).click();
  await expect.poll(() => roleAssignmentReads).toBeGreaterThan(firstRoleReadCount);
  const roleSelect = reopenedProfile.locator('button[aria-label="Role to assign to teacher.role-admin"]');
  await roleSelect.click();
  await profile.getByRole("option", { name: "Librarian (custom)" }).click();
  await reopenedProfile.getByRole("button", { name: "Assign", exact: true }).click();
  await assignmentStarted;
  await expect(reopenedProfile.getByRole("button", { name: "Assigning…" })).toBeDisabled();
  await reopenedProfile.locator("form.staff-role-assign").evaluate((form: HTMLFormElement) => form.requestSubmit());
  await page.waitForTimeout(150);
  expect(assignCalls).toBe(1);
  releaseAssignment?.();
  await expect(reopenedProfile.getByText("Role assigned. New permissions apply on the next protected request.")).toBeVisible();
  await expect(reopenedProfile.getByText("Librarian", { exact: true })).toBeVisible();

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
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstAuditReadCount);
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
  let overrideReads = 0;
  let auditReads = 0;
  let releaseOverrideSave: (() => void) | null = null;
  let notifyOverrideSaveStarted: (() => void) | null = null;
  const overrideSaveStarted = new Promise<void>(resolve => { notifyOverrideSaveStarted = resolve; });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "permissions.override", "school.audit.read"]);
  await page.route("**/api/school/staff/515/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });
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
      { permission_key: "students.read_basic", description: "Read basic student information", is_delegable: true },
    ] } });
  });
  await page.route("**/api/school/permission-delegations", async route => {
    await route.fulfill({ json: { delegations: [
      { id: 31, school_id: 10, permission_key: "students.read_basic", can_assign_to_roles: false, can_override_users: true, granted_by: 2, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/515/permission-overrides**", async route => {
    const method = route.request().method();
    if (method === "GET") {
      overrideReads += 1;
      await route.fulfill({ json: { overrides } });
      return;
    }
    if (method === "POST") {
      saveCalls += 1;
      submittedPayload = route.request().postDataJSON() as Record<string, unknown>;
      await new Promise<void>(resolve => {
        releaseOverrideSave = resolve;
        notifyOverrideSaveStarted?.();
      });
      overrides.push({
        id: 71, principal_id: 515, permission_key: "students.read_basic",
        effect: "deny", reason: "Temporary roster access restriction",
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
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect(profile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstAuditReadCount = auditReads;
  await profile.getByRole("button", { name: "Account & Access" }).click();
  await expect(profile.getByText("Individual permission overrides", { exact: true })).toBeVisible();
  const firstOverrideReadCount = overrideReads;
  expect(firstOverrideReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Override Admin" }).click();
  const reopenedProfile = page.getByRole("dialog", { name: "Override Admin" });
  await reopenedProfile.getByRole("button", { name: "Account & Access" }).click();
  await expect(reopenedProfile.getByText("Individual permission overrides", { exact: true })).toBeVisible();
  await expect.poll(() => overrideReads).toBeGreaterThan(firstOverrideReadCount);

  const permissionSelect = reopenedProfile.locator('button[aria-label="Delegated permission override for teacher.override-admin"]');
  await permissionSelect.click();
  await profile.getByRole("option", { name: /students.read_basic/ }).click();
  await profile.getByRole("button", { name: "Save override" }).click();
  await expect(profile.getByText("Choose a permission delegated for individual overrides.")).toHaveCount(0);
  await expect(profile.getByText("Explain why this permission exception is needed.")).toBeVisible();
  await reopenedProfile.getByLabel("Reason (required)").fill("Temporary roster access restriction");
  await reopenedProfile.getByRole("button", { name: "Save override" }).click();
  await overrideSaveStarted;
  await expect(reopenedProfile.getByRole("button", { name: "Saving…" })).toBeDisabled();
  await reopenedProfile.locator("form.staff-override-form").evaluate((form: HTMLFormElement) => form.requestSubmit());
  await page.waitForTimeout(150);
  expect(saveCalls).toBe(1);
  releaseOverrideSave?.();
  await expect(reopenedProfile.getByText("Permission override saved. It takes effect on the next protected request.")).toBeVisible();
  await expect(reopenedProfile.getByText("students.read_basic", { exact: true })).toBeVisible();
  expect(submittedPayload).toMatchObject({
    permission_key: "students.read_basic",
    effect: "deny",
    reason: "Temporary roster access restriction",
    delegation_id: 31,
  });

  await profile.getByRole("button", { name: "Remove students.read_basic override" }).click();
  const cancelRemoval = page.getByRole("dialog", { name: "Remove permission override?" });
  await cancelRemoval.getByRole("button", { name: "Cancel" }).click();
  expect(deleteCalls).toBe(0);
  await profile.getByRole("button", { name: "Remove students.read_basic override" }).click();
  const confirmRemoval = page.getByRole("dialog", { name: "Remove permission override?" });
  await confirmRemoval.getByRole("button", { name: "Remove override" }).click();
  await expect(profile.getByText("Permission override removed. Effective permissions update on the next protected request.")).toBeVisible();
  await expect(profile.getByText("students.read_basic", { exact: true })).toHaveCount(0);
  expect(deleteCalls).toBe(1);
  await profile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstAuditReadCount);
});

test("standalone staff role and permission override dialogs reload employee access data after closing", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 516, school_id: 10, username: "teacher.standalone-access",
    employee_code: "EMP-STANDALONE", full_name: "Standalone Access Teacher", designation: "Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let roleReads = 0;
  let overrideReads = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "roles.read", "roles.assign", "permissions.override"]);
  await page.route("**/api/school/roles", async route => {
    await route.fulfill({ json: { roles: [
      { id: 4, role_key: "teacher", name: "Teacher", is_system: true, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/516/roles", async route => {
    if (route.request().method() === "GET") {
      roleReads += 1;
      await route.fulfill({ json: { roles: [
        { role_id: 4, role_key: "teacher", name: "Teacher", is_system: true, assigned_at: "2026-01-01T00:00:00Z" },
      ] } });
      return;
    }
    await route.fulfill({ status: 405, json: { message: "Unexpected role mutation." } });
  });
  await page.route("**/api/school/permissions", async route => {
    await route.fulfill({ json: { permissions: [
      { permission_key: "students.read_basic", description: "Read basic student information", is_delegable: true },
    ] } });
  });
  await page.route("**/api/school/permission-delegations", async route => {
    await route.fulfill({ json: { delegations: [
      { id: 31, school_id: 10, permission_key: "students.read_basic", can_assign_to_roles: false, can_override_users: true, granted_by: 2, created_at: "2026-01-01T00:00:00Z" },
    ] } });
  });
  await page.route("**/api/school/staff/516/permission-overrides**", async route => {
    if (route.request().method() === "GET") {
      overrideReads += 1;
      await route.fulfill({ json: { overrides: [] } });
      return;
    }
    await route.fulfill({ status: 405, json: { message: "Unexpected permission override mutation." } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  const manageRoles = page.getByRole("button", { name: "Manage roles for teacher.standalone-access" });
  await manageRoles.click();
  const roleDialog = page.getByRole("dialog", { name: "Access roles · teacher.standalone-access" });
  await expect(roleDialog.getByText("Teacher", { exact: true })).toBeVisible();
  const firstRoleReadCount = roleReads;
  expect(firstRoleReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");
  await expect(roleDialog).toHaveCount(0);
  await manageRoles.click();
  const reopenedRoleDialog = page.getByRole("dialog", { name: "Access roles · teacher.standalone-access" });
  await expect(reopenedRoleDialog.getByText("Teacher", { exact: true })).toBeVisible();
  await expect.poll(() => roleReads).toBeGreaterThan(firstRoleReadCount);
  await page.keyboard.press("Escape");

  const manageOverrides = page.getByRole("button", { name: "Manage permission overrides for teacher.standalone-access" });
  await manageOverrides.click();
  const overrideDialog = page.getByRole("dialog", { name: "Permission overrides · teacher.standalone-access" });
  await expect(overrideDialog.getByText("Individual permission overrides", { exact: true })).toBeVisible();
  const firstOverrideReadCount = overrideReads;
  expect(firstOverrideReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");
  await expect(overrideDialog).toHaveCount(0);
  await manageOverrides.click();
  const reopenedOverrideDialog = page.getByRole("dialog", { name: "Permission overrides · teacher.standalone-access" });
  await expect(reopenedOverrideDialog.getByText("Individual permission overrides", { exact: true })).toBeVisible();
  await expect.poll(() => overrideReads).toBeGreaterThan(firstOverrideReadCount);
});

test("staff profile history does not request audit events without school.audit.read", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 513, school_id: 10, username: "teacher.history.readonly",
    employee_code: "EMP-HIS-02", full_name: "Read Only History Teacher", designation: "Science Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  let auditRequests = 0;
  await page.route("**/api/school/staff/513/audit-events**", async route => {
    auditRequests += 1;
    await route.fulfill({ json: { events: [] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Read Only History Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Read Only History Teacher" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("You do not have permission to view audit history.")).toBeVisible();
  expect(auditRequests).toBe(0);
});

test("staff profile history is permission-gated and renders staff audit events", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 512, school_id: 10, username: "teacher.history",
    employee_code: "EMP-HIS-01", full_name: "History Teacher", designation: "Maths Teacher",
    mobile_number: "9876543214", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false, created_at: "2026-04-01T00:00:00.000Z",
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "users.create", "users.update", "school.audit.read"]);
  await page.route("**/api/school/staff/512/audit-events**", async route => {
    await route.fulfill({ json: { events: [
      { id: 17, action: "staff.password.changed", resource_type: "staff_user", outcome: "success", actor_username: "history.teacher", metadata: { password_change_required_cleared: true }, created_at: "2026-11-22T16:00:00.000Z" },
      { id: 16, action: "staff.account.updated", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { old_username: "old-history-teacher", username: "history.teacher", sessions_revoked: true }, created_at: "2026-11-22T15:00:00.000Z" },
      { id: 15, action: "staff.account.disabled", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { target_principal_id: 512 }, created_at: "2026-11-22T14:00:00.000Z" },
      { id: 14, action: "staff.account.enabled", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { target_principal_id: 512 }, created_at: "2026-11-22T13:00:00.000Z" },
      { id: 13, action: "staff.password.reset", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { must_change_password: true }, created_at: "2026-11-22T12:00:00.000Z" },
      { id: 12, action: "staff.substitution.cancelled", resource_type: "staff_substitution", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T11:00:00.000Z" },
      { id: 11, action: "staff.substitution.assigned", resource_type: "staff_substitution", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T10:00:00.000Z" },
      { id: 10, action: "staff.leave.approved", resource_type: "staff_leave_request", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-21T10:00:00.000Z" },
      { id: 9, action: "staff.profile.updated", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { employment_status: "active" }, created_at: "2026-11-20T10:00:00.000Z" },
      { id: 8, action: "staff.account.created", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: { employee_code: "EMP-HIS-01" }, created_at: "2026-04-01T09:00:00.000Z" },
      { id: 9, action: "school.staff.role_assigned", resource_type: "staff_principal", outcome: "success", actor_username: "school.admin", metadata: { role_id: 4 }, created_at: "2026-03-31T09:00:00.000Z" },
      { id: 8, action: "school.staff.role_revoked", resource_type: "staff_principal", outcome: "success", actor_username: "school.admin", metadata: { role_id: 4 }, created_at: "2026-03-30T12:00:00.000Z" },
      { id: 7, action: "school.staff.permission_override.set", resource_type: "user_permission_override", outcome: "success", actor_username: "school.admin", metadata: { permission_key: "attendance.update", effect: "deny" }, created_at: "2026-03-30T09:00:00.000Z" },
      { id: 6, action: "school.staff.permission_override.removed", resource_type: "user_permission_override", outcome: "success", actor_username: "school.admin", metadata: { permission_key: "attendance.update" }, created_at: "2026-03-29T09:00:00.000Z" },
    ] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "History Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "History Teacher" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  await expect(dialog.getByText("Account disabled")).toBeVisible();
  await expect(dialog.getByText("Account enabled")).toBeVisible();
  await expect(dialog.getByText("Account credentials updated")).toBeVisible();
  await expect(dialog.getByText("Password changed")).toBeVisible();
  await expect(dialog.getByText("Password reset")).toBeVisible();
  await expect(dialog.getByText("Account created")).toBeVisible();
  await expect(dialog.getByText("Role assigned")).toBeVisible();
  await expect(dialog.getByText("Role removed")).toBeVisible();
  await expect(dialog.getByText("Individual permission override saved")).toBeVisible();
  await expect(dialog.getByText("Individual permission override removed")).toBeVisible();
  await expect(dialog.getByText("Leave approved")).toBeVisible();
  await expect(dialog.getByText("Class substitution assigned")).toBeVisible();
  await expect(dialog.getByText("Class substitution cancelled")).toBeVisible();
  await expect(dialog.getByText(/attendance, leave and substitution events/)).toBeVisible();
  await expect(dialog.getByText("school.admin").first()).toBeVisible();
  const expectedIST = new Intl.DateTimeFormat("en-IN", { dateStyle: "medium", timeStyle: "short", timeZone: "Asia/Kolkata" }).format(new Date("2026-11-22T11:00:00.000Z"));
  await expect(dialog.getByText(expectedIST)).toBeVisible();
});

test("leave-read-only permission opens applications without staff directory or mutation controls", async ({ page }) => {
  const staffDirectoryRequests: string[] = [];
  const leaveReads: string[] = [];
  page.on("request", request => {
    const pathname = new URL(request.url()).pathname;
    if (/^\/api\/school\/staff(?:\/|$)/.test(pathname)) staffDirectoryRequests.push(request.url());
    if (pathname === "/api/school/staff-leave/requests") leaveReads.push(request.url());
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.leave.read"]);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Applications" })).toBeVisible();
  await expect(page.getByText("Leave Teacher")).toBeVisible();
  await expect(page.getByRole("heading", { name: "New leave request" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Approve" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Reject" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Staff Directory/ })).toHaveCount(0);
  expect(leaveReads.length).toBeGreaterThan(0);
  expect(staffDirectoryRequests).toEqual([]);
});

test("holiday-read-only permission opens the calendar without staff directory or mutation controls", async ({ page }) => {
  const staffDirectoryRequests: string[] = [];
  const holidayReads: string[] = [];
  page.on("request", request => {
    const pathname = new URL(request.url()).pathname;
    if (/^\/api\/school\/staff(?:\/|$)/.test(pathname)) staffDirectoryRequests.push(request.url());
    if (pathname === "/api/school/holidays") holidayReads.push(request.url());
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.holidays.read"]);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  await expect(page.getByText(/School holiday calendar/)).toBeVisible();
  await expect(page.getByText("School Foundation Day")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Add school holiday" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Edit" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Deactivate" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Reactivate" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Staff Directory/ })).toHaveCount(0);
  expect(holidayReads.length).toBeGreaterThan(0);
  expect(staffDirectoryRequests).toEqual([]);
});

test("attendance-only permission opens the attendance workspace without loading the staff directory", async ({ page }) => {
  const staffDirectoryRequests: string[] = [];
  page.on("request", request => {
    const pathname = new URL(request.url()).pathname;
    if (/^\/api\/school\/staff(?:\/|$)/.test(pathname)) staffDirectoryRequests.push(request.url());
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.attendance.read"]);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Staff attendance" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Attendance", exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: /Staff Directory/ })).toHaveCount(0);
  expect(staffDirectoryRequests).toEqual([]);
});

test("staff substitution history exports every matching page", async ({ page }) => {
  const rows = Array.from({ length: 51 }, (_, index) => ({
    id: index + 1, timetable_entry_id: 700 + index, cover_date: "2026-11-16",
    original_teacher_principal_id: 501, original_teacher_name: "Original Teacher",
    substitute_teacher_principal_id: 502, substitute_teacher_name: "Substitute Teacher",
    approved_leave_request_id: null, reason: "Private reason must not be exported", status: "active",
    section_name: "A", class_name: "Class 6", subject_name: "Mathematics",
    period_number: 2, period_name: "Period 2", starts_at: "09:40", ends_at: "10:20",
    room: "Room 12", created_by_principal_id: 101, created_at: "2026-11-15T10:00:00.000Z",
  }));
  const requestedPages: number[] = [];
  await signInAndOpenStaff(page, [], undefined, ["staff.substitutions.read"]);
  await page.route("**/api/school/staff-substitutions?*", async route => {
    const url = new URL(route.request().url());
    const pageNumber = Number(url.searchParams.get("page") ?? 1);
    const pageSize = Number(url.searchParams.get("page_size") ?? 50);
    requestedPages.push(pageNumber);
    await route.fulfill({ json: {
      substitutions: rows.slice((pageNumber - 1) * pageSize, pageNumber * pageSize),
      pagination: { page: pageNumber, page_size: pageSize, total: 51 },
    } });
  });
  await expect(page.getByRole("heading", { name: "Substitute Teacher Coverage" })).toBeVisible();
  await expect(page.getByText("Showing 1–50 of 51")).toBeVisible();
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export all CSV" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("staff-substitution-history.csv");
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv.split(/\r?\n/)).toHaveLength(52);
  expect(csv).toContain("Substitute Teacher");
  expect(csv).not.toContain("Private reason");
  expect(requestedPages).toContain(2);
});

test("staff substitution export rejects a failed later page without downloading partial CSV", async ({ page }) => {
  const rows=Array.from({length:50},(_,index)=>({
    id:index+1,timetable_entry_id:700+index,cover_date:"2026-11-16",
    original_teacher_principal_id:501,original_teacher_name:"Original Teacher",
    substitute_teacher_principal_id:502,substitute_teacher_name:"Substitute Teacher",
    approved_leave_request_id:null,reason:"Operational reason",status:"active",
    section_name:"A",class_name:"Class 6",subject_name:"Mathematics",
    period_number:2,period_name:"Period 2",starts_at:"09:40",ends_at:"10:20",
    room:"Room 12",created_by_principal_id:101,created_at:"2026-11-15T10:00:00.000Z",
  }));
  await signInAndOpenStaff(page,[],undefined,["staff.substitutions.read"]);
  await page.route("**/api/school/staff-substitutions?*",async route=>{
    const url=new URL(route.request().url());
    const pageNumber=Number(url.searchParams.get("page")??1);
    if(pageNumber===2){await route.fulfill({status:503,json:{message:"Temporary report failure"}});return;}
    await route.fulfill({json:{substitutions:rows,pagination:{page:1,page_size:50,total:51}}});
  });
  await expect(page.getByRole("heading",{name:"Substitute Teacher Coverage"})).toBeVisible();
  const noDownload=page.waitForEvent("download",{timeout:1000}).catch(()=>null);
  await page.getByRole("button",{name:"Export all CSV"}).click();
  await expect(page.getByRole("alert").filter({hasText:"No partial CSV was created"})).toBeVisible();
  expect(await noDownload).toBeNull();
});

test("substitution-history-only permission opens read-only history without loading the staff directory", async ({ page }) => {
  const staffDirectoryRequests: string[] = [];
  page.on("request", request => {
    const pathname = new URL(request.url()).pathname;
    if (/^\/api\/school\/staff(?:\/|$)/.test(pathname)) staffDirectoryRequests.push(request.url());
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.substitutions.read"]);
  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Substitute Teacher Coverage" })).toBeVisible();
  await expect(page.getByText("Dated coverage records")).toBeVisible();
  await expect(page.getByRole("button", { name: /Staff Directory/ })).toHaveCount(0);
  expect(staffDirectoryRequests).toEqual([]);
});


test("staff attendance ignores duplicate individual and bulk submissions while requests are pending", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 612, school_id: 10, username: "attendance.guard.one", employee_code: "EMP-GUARD-01", full_name: "Guard One", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 613, school_id: 10, username: "attendance.guard.two", employee_code: "EMP-GUARD-02", full_name: "Guard Two", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  const records = [
    { id: null as number | null, school_id: 10, staff_principal_id: 612, employee_code: "EMP-GUARD-01", full_name: "Guard One", designation: "Teacher", attendance_date: "", status: null as string | null, remarks: "" },
    { id: null as number | null, school_id: 10, staff_principal_id: 613, employee_code: "EMP-GUARD-02", full_name: "Guard Two", designation: "Teacher", attendance_date: "", status: null as string | null, remarks: "" },
  ];
  let individualPosts = 0;
  let bulkPosts = 0;
  let releaseIndividual: (() => void) | null = null;
  let releaseBulk: (() => void) | null = null;
  await signInAndOpenStaff(page, staffRows, undefined, [
    "users.read", "staff.attendance.read", "staff.attendance.mark",
  ]);
  await page.route("**/api/school/staff-attendance**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "GET") {
      for (const item of records) item.attendance_date = url.searchParams.get("date") ?? "";
      await route.fulfill({ json: {
        date: url.searchParams.get("date"),
        data: records,
        pagination: { limit: 50, offset: 0, total: records.length },
      } });
      return;
    }
    if (request.method() === "POST" && url.pathname.endsWith("/bulk")) {
      bulkPosts += 1;
      const input = request.postDataJSON() as { staff_principal_ids: number[]; status: string };
      await new Promise<void>(resolve => { releaseBulk = resolve; });
      for (const item of records) if (input.staff_principal_ids.includes(item.staff_principal_id)) {
        item.id = 702;
        item.status = input.status;
      }
      await route.fulfill({ status: 201, json: { created: input.staff_principal_ids.length } });
      return;
    }
    if (request.method() === "POST") {
      individualPosts += 1;
      const input = request.postDataJSON() as { staff_principal_id: number; status: string; attendance_date: string };
      await new Promise<void>(resolve => { releaseIndividual = resolve; });
      const item = records.find(record => record.staff_principal_id === input.staff_principal_id);
      if (item) { item.id = 701; item.status = input.status; item.attendance_date = input.attendance_date; }
      await route.fulfill({ status: 201, json: { data: item } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: "Unmocked staff attendance request" } });
  });

  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const firstRow = page.getByRole("row").filter({ hasText: "Guard One" });
  await firstRow.getByLabel("Attendance status for Guard One").selectOption("present");
  const individualButton = firstRow.getByRole("button", { name: "Mark", exact: true });
  await individualButton.click();
  await expect.poll(() => individualPosts).toBe(1);
  await individualButton.evaluate(element => element.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  await expect.poll(() => individualPosts).toBe(1);
  expect(bulkPosts).toBe(0);
  releaseIndividual?.();
  await expect(page.getByText("Staff attendance marked.")).toBeVisible();

  await page.getByLabel("Bulk attendance status").selectOption("absent");
  await page.getByRole("button", { name: "Mark 1 unmarked staff", exact: true }).click();
  const confirmation = page.getByRole("dialog", { name: "Confirm bulk staff attendance" });
  const confirmButton = confirmation.locator("button.app-modal-confirm");
  await confirmButton.click();
  await expect.poll(() => bulkPosts).toBe(1);
  await confirmButton.evaluate(element => element.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  await expect.poll(() => bulkPosts).toBe(1);
  expect(individualPosts).toBe(1);
  releaseBulk?.();
  await expect(page.getByText("Attendance marked for 1 staff members.")).toBeVisible();
});

test("staff attendance updates existing records, retries failed writes, and confirms historical corrections once", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 622, school_id: 10, username: "attendance.lifecycle", employee_code: "EMP-LIFE-01", full_name: "Lifecycle Teacher", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  const record = {
    id: 881, school_id: 10, staff_principal_id: 622, employee_code: "EMP-LIFE-01",
    full_name: "Lifecycle Teacher", designation: "Teacher", attendance_date: "",
    status: "present" as string | null, remarks: "",
  };
  let patchCalls = 0;
  let releaseSameDay: (() => void) | null = null;
  let releaseHistorical: (() => void) | null = null;
  await signInAndOpenStaff(page, staffRows, undefined, [
    "users.read", "staff.attendance.read", "staff.attendance.mark",
    "staff.attendance.update_same_day", "staff.attendance.correct_past",
  ]);
  await page.route("**/api/school/staff-attendance**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "GET") {
      record.attendance_date = url.searchParams.get("date") ?? "";
      await route.fulfill({ json: {
        date: record.attendance_date,
        data: [{ ...record }],
        pagination: { limit: 50, offset: 0, total: 1 },
      } });
      return;
    }
    if (request.method() === "PATCH") {
      patchCalls += 1;
      const input = request.postDataJSON() as { status: string; remarks?: string };
      if (patchCalls === 1) {
        await route.fulfill({ status: 500, json: { message: "Temporary write failure" } });
        return;
      }
      if (patchCalls === 2) {
        await new Promise<void>(resolve => { releaseSameDay = resolve; });
      } else {
        await new Promise<void>(resolve => { releaseHistorical = resolve; });
      }
      record.status = input.status;
      record.remarks = input.remarks ?? "";
      await route.fulfill({ status: 200, json: { data: { ...record } } });
      return;
    }
    await route.fulfill({ status: 404, json: { message: "Unmocked staff attendance request" } });
  });

  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const row = page.getByRole("row").filter({ hasText: "Lifecycle Teacher" });
  await expect(row).toBeVisible();
  await row.getByLabel("Attendance status for Lifecycle Teacher").selectOption("absent");
  const updateButton = row.getByRole("button", { name: "Update", exact: true });

  // A failed request must release the synchronous lock so a user can retry.
  await updateButton.click();
  await expect(page.getByText("Attendance could not be updated. Check the correction permission.")).toBeVisible();
  await expect(updateButton).toBeEnabled();
  await updateButton.click();
  await expect.poll(() => patchCalls).toBe(2);
  await updateButton.evaluate(element => element.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  await expect.poll(() => patchCalls).toBe(2);
  releaseSameDay?.();
  await expect(page.getByText("Staff attendance updated.")).toBeVisible();

  // Move to a past date, then verify the explicit correction confirmation and duplicate guard.
  await page.getByLabel("Attendance date").click();
  await page.getByRole("button", { name: "Previous month" }).click();
  await page.locator(".app-date-day").first().click();
  const historicalRow = page.getByRole("row").filter({ hasText: "Lifecycle Teacher" });
  await expect(historicalRow).toBeVisible();
  await historicalRow.getByLabel("Attendance status for Lifecycle Teacher").selectOption("late");
  await historicalRow.getByRole("button", { name: "Update", exact: true }).click();
  const correctionDialog = page.getByRole("dialog", { name: "Confirm historical attendance correction" });
  const confirmCorrection = correctionDialog.locator("button.app-modal-confirm");
  await expect(confirmCorrection).toBeVisible();
  await confirmCorrection.click();
  await expect.poll(() => patchCalls).toBe(3);
  await confirmCorrection.evaluate(element => element.dispatchEvent(new MouseEvent("click", { bubbles: true })));
  await expect.poll(() => patchCalls).toBe(3);
  releaseHistorical?.();
  await expect(page.getByText("Staff attendance updated.")).toBeVisible();
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
    "staff.leave.read", "staff.holidays.read", "school.audit.read",
  ]);
  let auditReads = 0;
  await page.route("**/api/school/staff/512/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Attendance Teacher" }).click();
  const attendanceProfile = page.getByRole("dialog", { name: "Attendance Teacher" });
  await attendanceProfile.getByRole("button", { name: "History & Activity" }).click();
  await expect(attendanceProfile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstAttendanceAuditRead = auditReads;
  await page.keyboard.press("Escape");
  await page.route("**/api/school/staff-leave/requests?*", async route => {
    const url = new URL(route.request().url());
    if (!url.searchParams.has("from_date")) { await route.fallback(); return; }
    expect(url.searchParams.get("status")).toBe("approved");
    expect(url.searchParams.get("from_date")).toMatch(/^\d{4}-\d{2}-01$/);
    expect(url.searchParams.get("to_date")).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    const page = Number(url.searchParams.get("page") ?? 1);
    const pageSize = Number(url.searchParams.get("page_size") ?? 100);
    const allApproved = Array.from({length:101},(_,index)=>({
      id:71+index,staff_principal_id:512,staff_name:"Attendance Teacher",employee_code:"EMP-ATT-01",
      leave_type_id:1,leave_type_name:index===100?"Leave 101":"Casual Leave",
      start_date:url.searchParams.get("from_date"),end_date:url.searchParams.get("from_date"),
      duration_type:"full_day",reason:"PRIVATE_LEAVE_REASON",status:"approved",
      created_at:"2026-11-01T00:00:00.000Z",
    }));
    await route.fulfill({json:{data:allApproved.slice((page-1)*pageSize,page*pageSize),pagination:{page,page_size:pageSize,total:allApproved.length}}});
  });
  let savedStatus = "";
  let bulkCreated = 0;
  let monthlyTotal = 101;
  const dailyRecords = [
    { id: null as number | null, school_id: 10, staff_principal_id: 512, employee_code: "EMP-ATT-01", full_name: "Attendance Teacher", designation: "Science Teacher", attendance_date: "", status: null as string | null },
    { id: null as number | null, school_id: 10, staff_principal_id: 513, employee_code: "EMP-ATT-02", full_name: "Bulk Attendance Teacher", designation: "English Teacher", attendance_date: "", status: null as string | null },
  ];
  await page.route("**/api/school/staff-attendance**", async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname.endsWith("/month")) {
      const limit = Number(url.searchParams.get("limit") ?? 50);
      const offset = Number(url.searchParams.get("offset") ?? 0);
      const allMonthly = Array.from({ length: 101 }, (_, index) => ({
        staff_principal_id: 512 + index,
        employee_code: `EMP-ATT-${String(index + 1).padStart(2, "0")}`,
        full_name: index === 0 ? "Attendance Teacher" : `Monthly Teacher ${index + 1}`,
        designation: "Science Teacher", present_days: index === 0 ? 1 : 0,
        absent_days: index === 0 ? 1 : 0, late_days: index === 0 ? 1 : 0, half_days: 0,
        records: index === 0 ? [{ id: 7, date: "2026-11-03", status: "present", remarks: "PRIVATE_ATTENDANCE_REMARK" }, { id: 8, date: "2026-11-04", status: "late" }, { id: 9, date: "2026-11-05", status: "absent" }] : [],
      }));
      await route.fulfill({ json: {
        month: url.searchParams.get("month"),
        data: allMonthly.slice(offset, Math.min(offset + limit, monthlyTotal)),
        pagination: { limit, offset, total: monthlyTotal },
      } });
      return;
    }
    if (request.method() === "GET") {
      for (const item of dailyRecords) item.attendance_date = url.searchParams.get("date") ?? "";
      const search = (url.searchParams.get("search") ?? "").toLocaleLowerCase();
      const status = url.searchParams.get("status") ?? "all";
      const filtered = dailyRecords.filter(item => {
        const matchesSearch = !search || [item.full_name, item.employee_code, item.designation].some(value => String(value ?? "").toLocaleLowerCase().includes(search));
        const matchesStatus = status === "all" || (status === "unmarked" ? !item.status : item.status === status);
        return matchesSearch && matchesStatus;
      });
      const limit = Number(url.searchParams.get("limit") ?? 50);
      const offset = Number(url.searchParams.get("offset") ?? 0);
      await route.fulfill({ json: { date: url.searchParams.get("date"), data: filtered.slice(offset, offset + limit), pagination: { limit, offset, total: filtered.length } } });
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
  await expect(page.getByText("Showing 1–50 of 101 staff")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).last().click();
  await expect(page.getByText("Monthly Teacher 51")).toBeVisible();
  await page.getByRole("button", { name: "Previous" }).last().click();
  await expect(page.getByText("Attendance Teacher").first()).toBeVisible();
  await expect(page.getByText("Calendar context · not included in attendance totals")).toBeVisible();
  await expect(page.getByText("Approved leave", { exact: true })).toBeVisible();
  await expect(page.getByText(/Attendance Teacher · Casual Leave/).first()).toBeVisible();
  await expect(page.getByText("School holidays", { exact: true })).toBeVisible();
  await expect(page.getByText(/School Foundation Day/)).toBeVisible();
  const monthlyDownloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download complete monthly report" }).click();
  const monthlyDownload = await monthlyDownloadPromise;
  const monthlyPath = await monthlyDownload.path();
  expect(monthlyPath).toBeTruthy();
  const monthlyCSV = await readFile(monthlyPath!, "utf8");
  expect(monthlyCSV.startsWith("\uFEFF")).toBe(true);
  expect(monthlyCSV).toContain("EMP-ATT-01");
  expect(monthlyCSV).toContain("EMP-ATT-101");
  expect(monthlyCSV).not.toContain("PRIVATE_ATTENDANCE_REMARK");
  await page.getByRole("button", { name: "Next" }).last().click();
  await expect(page.getByText("Monthly Teacher 51")).toBeVisible();
  monthlyTotal = 50;
  await page.getByRole("button", { name: "Refresh" }).click();
  await expect(page.getByText("Showing 1–50 of 50 staff")).toBeVisible();
  await expect(page.getByText("Monthly Teacher 51")).toHaveCount(0);
  expect(monthlyCSV).not.toContain("PRIVATE_LEAVE_REASON");
  expect(monthlyCSV).toContain("Leave 101");

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Attendance Teacher" }).click();
  const refreshedAttendanceProfile = page.getByRole("dialog", { name: "Attendance Teacher" });
  await refreshedAttendanceProfile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstAttendanceAuditRead);
});

test("leave decisions refresh approved-leave context when attendance is reopened", async ({ page }) => {
  const staffRows:Array<Record<string,unknown>>=[{
    id:1,principal_id:530,school_id:10,username:"teacher.leave-context",
    employee_code:"EMP-LEAVE-CONTEXT",full_name:"Leave Context Teacher",designation:"Teacher",
    employment_status:"active",profile_complete:true,status:"active",must_change_password:false,
  }];
  let approvedContextReads=0;
  page.on("request",request=>{
    const url=new URL(request.url());
    if(request.method()==="GET"&&url.pathname==="/api/school/staff-leave/requests"&&url.searchParams.has("on_date"))approvedContextReads++;
  });
  await signInAndOpenStaff(page,staffRows,undefined,[
    "users.read","staff.attendance.read","staff.leave.read","staff.leave.decide",
  ]);
  await page.getByRole("button",{name:"Attendance",exact:true}).click();
  await expect.poll(()=>approvedContextReads).toBeGreaterThan(0);
  const initialReads=approvedContextReads;
  await page.getByRole("button",{name:"Leave & Holidays"}).click();
  const requestRow=page.locator("article.staff-leave-request").filter({hasText:"Family commitment"});
  await requestRow.getByRole("button",{name:"Approve"}).click();
  const decision=page.getByRole("dialog",{name:"Approve leave request"});
  await decision.getByRole("button",{name:"Confirm approval"}).click();
  await expect(page.getByText("Leave request approved.")).toBeVisible();
  await expect(requestRow).toHaveCount(0);
  await page.getByRole("button",{name:"Attendance",exact:true}).click();
  await expect.poll(()=>approvedContextReads).toBeGreaterThan(initialReads);
});

test("attendance drafts stay scoped to their selected date", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 520, school_id: 10, username: "teacher.date-draft",
    employee_code: "EMP-DATE-01", full_name: "Date Draft Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read", "staff.attendance.mark", "staff.attendance.correct_past"]);
  await page.route("**/api/school/staff-attendance**", async route => {
    if (route.request().method() !== "GET") { await route.fulfill({ status: 201, json: { data: {} } }); return; }
    const requestedDate = new URL(route.request().url()).searchParams.get("date") ?? schoolDateValue();
    await route.fulfill({ json: { date: requestedDate, data: [{
      id: null, school_id: 10, staff_principal_id: 520, employee_code: "EMP-DATE-01",
      full_name: "Date Draft Teacher", designation: "Teacher", attendance_date: requestedDate, status: null, remarks: "",
    }] } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const status = page.getByLabel("Attendance status for Date Draft Teacher");
  await status.selectOption("late");
  const selectedDate = schoolDateValue();
  const [year, month, day] = selectedDate.split("-").map(Number);
  const previousDate = new Date(year, month - 1, day - 1, 12);
  const previousLabel = previousDate.toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric", year: "numeric" });
  await page.getByLabel("Staff attendance date").click();
  const calendar = page.getByRole("dialog", { name: "Choose staff attendance date" });
  if (previousDate.getMonth() !== month - 1 || previousDate.getFullYear() !== year) {
    await calendar.getByRole("button", { name: "Previous month" }).click();
  }
  await calendar.getByRole("button", { name: previousLabel }).click();
  await expect(page.getByLabel("Attendance status for Date Draft Teacher")).toHaveValue("");
  await page.getByLabel("Staff attendance date").click();
  const returnCalendar = page.getByRole("dialog", { name: "Choose staff attendance date" });
  if (previousDate.getMonth() !== month - 1 || previousDate.getFullYear() !== year) {
    await returnCalendar.getByRole("button", { name: "Next month" }).click();
  }
  await returnCalendar.getByRole("button", { name: new Date(year, month - 1, day, 12).toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric", year: "numeric" }) }).click();
  await expect(page.getByLabel("Attendance status for Date Draft Teacher")).toHaveValue("late");
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

  const today = schoolDateOffset(0);
  const tomorrow = schoolDateOffset(1);
  const dayAfterTomorrow = schoolDateOffset(2);
  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: schoolDateLabel(tomorrow) }).click();
  await expect(page.getByText("Select a start date.")).toHaveCount(0);
  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: schoolDateLabel(tomorrow) }).click();
  await expect(page.getByText("Select an end date.")).toHaveCount(0);

  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: schoolDateLabel(dayAfterTomorrow) }).click();
  await page.getByRole("button", { name: "Submit leave request" }).click();
  await expect(page.getByText("End date must be on or after the start date.")).toBeVisible();
  await expect(page.getByLabel("Leave end date")).toHaveAttribute("aria-describedby", "leave-end-date-error");

  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: schoolDateLabel(today) }).click();
  await expect(page.getByText("End date must be on or after the start date.")).toHaveCount(0);

  await page.getByLabel("Leave duration").click();
  await page.getByRole("option", { name: "Half day" }).click();
  await page.getByRole("button", { name: "Submit leave request" }).click();
  await expect(page.getByText("Half-day leave must cover one date only.")).toBeVisible();
  await expect(page.getByLabel("Leave duration")).toHaveAttribute("aria-describedby", "leave-duration-error");

  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: schoolDateLabel(today) }).click();
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
  const todayLabel = schoolDateLabel(schoolDateOffset(0));
  await page.getByLabel("Leave start date").click();
  await page.getByRole("dialog", { name: "Choose leave start date" }).getByRole("button", { name: todayLabel }).click();
  await page.getByLabel("Leave end date").click();
  await page.getByRole("dialog", { name: "Choose leave end date" }).getByRole("button", { name: todayLabel }).click();
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
    start_date: schoolDateValue(),
    end_date: schoolDateValue(),
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
  await page.getByRole("dialog", { name: "Choose holiday date" }).getByRole("button", { name: schoolDateLabel(schoolDateOffset(1)) }).click();
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

  // Verify the employee workload dialog at each viewport while it is still open.
  for (const viewport of [{ width: 390, height: 844 }, { width: 768, height: 900 }, { width: 1280, height: 900 }]) {
    await page.setViewportSize(viewport);
    await expect(dialog.getByRole("heading", { name: "Teaching workload" })).toBeVisible();
    const dimensions = await page.evaluate(() => ({
      viewport: window.innerWidth,
      document: document.documentElement.scrollWidth,
    }));
    expect(dimensions.document).toBeLessThanOrEqual(dimensions.viewport);
    for (const selector of [".staff-workload-table-wrap", ".staff-workload-grid-wrap"]) {
      const wrapper = dialog.locator(selector);
      await expect(wrapper).toBeVisible();
      const overflowX = await wrapper.evaluate(element => getComputedStyle(element).overflowX);
      expect(["auto", "scroll"]).toContain(overflowX);
    }
  }

  // The school-wide report is a separate permission-gated workspace, not part
  // of the individual employee profile dialog.
  await page.keyboard.press("Escape");
  await page.locator(".staff-workspace-tabs").getByRole("button", { name: "Workload Report" }).click();
  await expect(page.getByRole("heading", { name: "School-wide Workload Report" })).toBeVisible();
  await page.getByLabel("Search workload report staff").fill("Workload");
  await expect(page.getByText("Workload Teacher")).toBeVisible();
  await expect(page.getByRole("button", { name: "Clear filters" })).toBeVisible();
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page.getByLabel("Search workload report staff")).toHaveValue("");
  await expect(page.getByLabel("Filter workload report by employment status")).toHaveValue("all");
  await expect(page.getByRole("button", { name: "Clear filters" })).toHaveCount(0);
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
  const permissions = ["users.read", "staff.substitutions.read", "staff.substitutions.manage", "timetable.read", "staff.leave.read", "school.audit.read"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  let auditReads = 0;
  await page.route("**/api/school/staff/501/audit-events**", async route => {
    auditReads += 1;
    await route.fulfill({ json: { events: [] } });
  });
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Original Teacher" }).click();
  const originalProfile = page.getByRole("dialog", { name: "Original Teacher" });
  await originalProfile.getByRole("button", { name: "History & Activity" }).click();
  await expect(originalProfile.getByText("No audited activity is recorded for this employee yet.")).toBeVisible();
  const firstSubstitutionAuditRead = auditReads;
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Substitutions" }).click();
  await expect(page.getByRole("heading", { name: "Substitute Teacher Coverage" })).toBeVisible();

  await page.getByRole("button", { name: "Recurring timetable lesson" }).click();
  await page.getByRole("option", { name: /Monday · Class 6 A · Mathematics/ }).click();
  await expect(page.getByText("Original teacher: Original Teacher")).toBeVisible();

  const [year, month, day] = schoolDateValue().split("-").map(Number);
  const today = new Date(year, month - 1, day, 12);
  const daysUntil = (weekday: number) => {
    const delta = (weekday - today.getDay() + 7) % 7;
    return delta === 0 ? 7 : delta;
  };
  const nextTuesday = new Date(today);
  nextTuesday.setDate(today.getDate() + daysUntil(2));
  const nextMonday = new Date(today);
  nextMonday.setDate(today.getDate() + daysUntil(1));
  const dateLabel = (date: Date) => date.toLocaleDateString(undefined, {
    weekday: "long", month: "long", day: "numeric", year: "numeric",
  });

  await page.getByLabel("Substitution cover date").click();
  const dateDialog = page.getByRole("dialog", { name: "Choose substitution cover date" });
  if (nextTuesday.getMonth() !== today.getMonth() || nextTuesday.getFullYear() !== today.getFullYear()) {
    await dateDialog.getByRole("button", { name: "Next month" }).click();
  }
  await dateDialog.getByRole("button", { name: dateLabel(nextTuesday) }).click();
  await page.getByRole("button", { name: "Substitute staff member" }).click();
  await page.getByRole("option", { name: /Substitute Teacher/ }).click();
  await page.getByLabel("Reason").fill("Original teacher unavailable");
  await page.getByRole("button", { name: "Assign substitute" }).click();
  await expect(page.getByText("This date is not a Monday; choose a matching date.")).toBeVisible();

  await page.getByLabel("Substitution cover date").click();
  const mondayDialog = page.getByRole("dialog", { name: "Choose substitution cover date" });
  if (nextMonday.getMonth() !== nextTuesday.getMonth() || nextMonday.getFullYear() !== nextTuesday.getFullYear()) {
    await mondayDialog.getByRole("button", { name: "Previous month" }).click();
  }
  await mondayDialog.getByRole("button", { name: dateLabel(nextMonday) }).click();
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

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Original Teacher" }).click();
  const refreshedOriginalProfile = page.getByRole("dialog", { name: "Original Teacher" });
  await refreshedOriginalProfile.getByRole("button", { name: "History & Activity" }).click();
  await expect.poll(() => auditReads).toBeGreaterThan(firstSubstitutionAuditRead);
});



test("substitution history clamps to the last valid page when the server-side total shrinks", async ({ page }) => {
  const rows=Array.from({length:51},(_,index)=>({
    id:index+1,timetable_entry_id:700+index,cover_date:"2026-11-16",
    original_teacher_principal_id:501,original_teacher_name:"Original Teacher",
    substitute_teacher_principal_id:502,substitute_teacher_name:"Substitute Teacher",
    approved_leave_request_id:null,reason:"Coverage record "+(index+1),status:"cancelled",
    section_name:"A",class_name:"Class 6",subject_name:"Mathematics",
    period_number:2,period_name:"Period 2",starts_at:"09:40",ends_at:"10:20",
    room:"Room 12",created_by_principal_id:101,created_at:"2026-11-15T10:00:00.000Z",
  }));
  let total=51;
  await signInAndOpenStaff(page,[],undefined,["users.read","staff.substitutions.read"]);
  await page.route("**/api/school/staff-substitutions?*",async route=>{
    const url=new URL(route.request().url());
    const pageNumber=Number(url.searchParams.get("page")??1);
    const pageSize=Number(url.searchParams.get("page_size")??50);
    await route.fulfill({json:{
      substitutions:rows.slice((pageNumber-1)*pageSize,Math.min(pageNumber*pageSize,total)),
      pagination:{page:pageNumber,page_size:pageSize,total},
    }});
  });
  await page.getByRole("button",{name:"Substitutions",exact:true}).click();
  await expect(page.getByText("Showing 1–50 of 51")).toBeVisible();
  await page.getByRole("button",{name:"Next"}).click();
  await expect(page.getByText("Showing 51–51 of 51")).toBeVisible();
  await expect(page.getByText("Coverage record 51")).toBeVisible();
  total=50;
  await page.locator(".staff-substitution-list-heading").getByRole("button",{name:"Refresh",exact:true}).click();
  await expect(page.getByText("Showing 1–50 of 50")).toBeVisible();
  await expect(page.getByText("Coverage record 50")).toBeVisible();
  await expect(page.getByText("Coverage record 51")).toHaveCount(0);
  await expect(page.getByRole("button",{name:"Previous"})).toBeDisabled();
  await expect(page.getByRole("button",{name:"Next"})).toBeDisabled();
});

test("substitution list staff filter can be reset without changing the default date window", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 501, school_id: 10, username: "teacher.original", employee_code: "EMP-ORIGINAL", full_name: "Original Teacher", designation: "Mathematics Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 502, school_id: 10, username: "teacher.substitute", employee_code: "EMP-SUBSTITUTE", full_name: "Substitute Teacher", designation: "Mathematics Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  const requests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname.endsWith("/school/staff-substitutions") && request.method() === "GET") requests.push(url.search);
  });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.substitutions.read"]);
  await page.getByRole("button", { name: "Substitutions" }).click();
  await page.getByRole("button", { name: "Filter substitutions by staff" }).click();
  await page.getByRole("option", { name: /Original Teacher/ }).click();
  await expect.poll(() => requests.some(query => query.includes("staff_principal_id=501"))).toBe(true);
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page.getByRole("button", { name: "Clear filters" })).toHaveCount(0);
  await expect.poll(() => requests.some(query => !query.includes("staff_principal_id"))).toBe(true);
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
      mobile_number: "9876543217", staff_type: "teaching", employment_status: "inactive", profile_complete: true,
      status: "disabled", must_change_password: false,
    },
    {
      id: 3, principal_id: 503, school_id: 10, username: "staff.non-teaching",
      employee_code: "EMP-NONTEACH", full_name: "Non-teaching Staff", designation: "Office Assistant",
      mobile_number: "9876543218", staff_type: "non_teaching", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 4, principal_id: 504, school_id: 10, username: "staff.unclassified",
      employee_code: "EMP-UNCLASSIFIED", full_name: "Unclassified Employee", designation: "Legacy Staff",
      mobile_number: "9876543219", employment_status: "inactive", profile_complete: true,
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
  await expect(page.getByText("Non-teaching Staff")).toBeVisible();
  await expect(page.getByText("Unclassified Employee")).toBeVisible();
  await expect(page.getByText("Staff profiles").locator("..")).toContainText("4");

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
  expect(csv).toContain("Non-teaching Staff");
  expect(csv).not.toContain("Inactive Teacher");
  expect(csv).not.toContain("Unclassified Employee");
  expect(csv).toContain(`'   =HYPERLINK(""x"")`);
  for (const viewport of [{ width: 390, height: 844 }, { width: 768, height: 900 }, { width: 1280, height: 900 }]) {
    await page.setViewportSize(viewport);
    await expect(page.getByLabel("Search workload report staff")).toBeVisible();
    await expect(page.getByLabel("Filter workload report by employment status")).toBeVisible();
    const dimensions = await page.evaluate(() => ({
      viewport: window.innerWidth,
      document: document.documentElement.scrollWidth,
    }));
    expect(dimensions.document).toBeLessThanOrEqual(dimensions.viewport);
  }
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
    {
      id: 3, principal_id: 503, school_id: 10, username: "teacher.legacy",
      employee_code: "EMP-LEGACY", full_name: "Legacy Profile", designation: "Office Staff",
      mobile_number: "9876543217", employment_status: "active",
      status: "disabled", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff Reports" })).toBeVisible();
  await expect(page.getByText("Total staff").locator("..")).toContainText("3");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Active employment", { exact: true }) }).locator("strong")).toHaveText("2");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Inactive employment", { exact: true }) }).locator("strong")).toHaveText("1");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Profile incomplete", { exact: true }) }).locator("strong")).toHaveText("2");
  await expect(page.locator(".staff-reports-summary article").filter({ has: page.getByText("Login disabled", { exact: true }) }).locator("strong")).toHaveText("2");
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export staff report" }).click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("EMP-ACTIVE");
  expect(csv).toContain("EMP-INACTIVE");
  expect(csv).toContain("EMP-LEGACY");
  expect(csv).not.toContain("9876543215");
  expect(csv).not.toContain("9876543216");
  expect(csv).not.toContain("9876543217");
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

test("staff operational reports calculate month boundaries correctly for early years", async ({ page }) => {
  await signInAndOpenStaff(page, [], undefined, ["users.read", "staff.attendance.read", "staff.leave.read"]);
  const requested: string[] = [];
  await page.route("**/api/school/staff-attendance/month?*", async route => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("month") === "0001-02") requested.push("attendance:" + url.searchParams.get("month"));
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    await route.fulfill({ json: { month: url.searchParams.get("month"), data: [], pagination: { limit, offset, total: 0 } } });
  });
  await page.route("**/api/school/staff-leave/requests?*", async route => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("from_date") === "0001-02-01") {
      requested.push("leave:" + url.searchParams.get("from_date") + ":" + url.searchParams.get("to_date"));
    }
    const page = Number(url.searchParams.get("page") ?? 1);
    const pageSize = Number(url.searchParams.get("page_size") ?? 100);
    await route.fulfill({ json: { data: [], pagination: { page, page_size: pageSize, total: 0 } } });
  });
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  const month = page.getByLabel("Select attendance and leave report month");
  await month.fill("0001-02");
  await expect.poll(() => requested.length).toBe(2);
  expect(requested).toContain("attendance:0001-02");
  expect(requested).toContain("leave:0001-02-01:0001-02-28");
  // Native input[type="month"] rejects year 0000 before the application can validate it.
  await expect.poll(() => requested.length).toBe(2);
});

test("staff monthly reports do not request data when the report month is cleared", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [];
  let attendanceRequests = 0;
  let leaveRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read", "staff.leave.read"]);
  page.on("request", request => {
    const url = new URL(request.url());
    if (request.method() !== "GET") return;
    if (url.pathname === "/api/school/staff-attendance/month") attendanceRequests += 1;
    if (url.pathname === "/api/school/staff-leave/requests") leaveRequests += 1;
  });
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  const monthInput = page.locator('input[type="month"][aria-label="Select attendance and leave report month"]');
  await expect(monthInput).toHaveValue(/^\d{4}-\d{2}$/);
  await expect.poll(() => attendanceRequests).toBeGreaterThan(0);
  await expect.poll(() => leaveRequests).toBeGreaterThan(0);
  const initialAttendanceRequests = attendanceRequests;
  const initialLeaveRequests = leaveRequests;
  await monthInput.fill("");
  await expect(page.getByText("Select a valid report month to load attendance and leave records.")).toBeVisible();
  await expect.poll(() => attendanceRequests).toBe(initialAttendanceRequests);
  await expect.poll(() => leaveRequests).toBe(initialLeaveRequests);
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

test("staff academic assignment report explains missing current academic year", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "teacher_assignments.read", "class_incharge_assignments.read"]);
  await page.route("**/api/school/timetable/current-year", route =>
    route.fulfill({ json: { academic_year: null } }),
  );
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Academic assignments" })).toBeVisible();
  await expect(page.getByText("No current academic year is configured. Set a current academic year before viewing academic assignments.")).toBeVisible();
  await expect(page.getByText("Assigned teachers")).toHaveCount(0);
  await expect(page.getByText("Subject assignments")).toHaveCount(0);
  await expect(page.getByText("Class-in-charge assignments")).toHaveCount(0);
  await expect(page.getByText("No active academic assignments were found for this academic year.")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Export assignments CSV" })).toBeDisabled();
  await expect(page.locator(".staff-operational-report-table-wrap")).toHaveCount(0);
});

test("staff assignment report only aggregates data allowed by each assignment permission", async ({ page }) => {
  await signInAndOpenStaff(page, [], undefined, ["users.read", "class_incharge_assignments.read"]);
  let subjectAssignmentRequests = 0;
  await page.route("**/api/school/timetable/current-year", route =>
    route.fulfill({ json: { academic_year: { id: 3, name: "2026–27" } } }),
  );
  await page.route("**/api/school/teacher-assignments**", route => {
    subjectAssignmentRequests += 1;
    return route.fulfill({ json: { assignments: [{
      id: 21, academic_year_id: 3, section_id: 80, section_name: "Class 8 A",
      subject_id: 9, subject_name: "Science", teacher_principal_id: 801,
      teacher_username: "teacher.subject-only", revoked_at: null,
    }] } });
  });
  await page.route("**/api/school/class-incharge-assignments**", route =>
    route.fulfill({ json: { assignments: [{
      id: 31, academic_year_id: 3, section_id: 81, section_name: "Class 7 B",
      class_name: "Class 7", teacher_principal_id: 802,
      teacher_username: "teacher.incharge-only", revoked_at: null,
    }] } }),
  );
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  const report = page.locator(".staff-operational-report-section").filter({
    has: page.getByRole("heading", { name: "Academic assignments" }),
  });
  await expect(report.getByText("teacher.incharge-only")).toBeVisible();
  await expect(report.getByText("teacher.subject-only")).toHaveCount(0);
  await expect(report.getByText("1", { exact: true }).first()).toBeVisible();
  await expect.poll(() => subjectAssignmentRequests).toBe(0);
  const downloadPromise = page.waitForEvent("download");
  await report.getByRole("button", { name: "Export assignments CSV" }).click();
  const download = await downloadPromise;
  const csvPath = await download.path();
  expect(csvPath).toBeTruthy();
  const csv = await readFile(csvPath!, "utf8");
  expect(csv).toContain("teacher.incharge-only");
  expect(csv).not.toContain("teacher.subject-only");
  expect(csv).not.toContain("Science");
});

test("staff assignment report excludes class-incharge data without its permission", async ({ page }) => {
  await signInAndOpenStaff(page, [], undefined, ["users.read", "teacher_assignments.read"]);
  let inchargeAssignmentRequests = 0;
  await page.route("**/api/school/timetable/current-year", route =>
    route.fulfill({ json: { academic_year: { id: 3, name: "2026–27" } } }),
  );
  await page.route("**/api/school/teacher-assignments**", route =>
    route.fulfill({ json: { assignments: [{
      id: 41, academic_year_id: 3, section_id: 90, section_name: "Class 9 A",
      subject_id: 11, subject_name: "Mathematics", teacher_principal_id: 901,
      teacher_username: "teacher.subject-only", revoked_at: null,
    }] } }),
  );
  await page.route("**/api/school/class-incharge-assignments**", route => {
    inchargeAssignmentRequests += 1;
    return route.fulfill({ json: { assignments: [{
      id: 42, academic_year_id: 3, section_id: 91, section_name: "Class 10 B",
      class_name: "Class 10", teacher_principal_id: 902,
      teacher_username: "teacher.incharge-secret", revoked_at: null,
    }] } });
  });
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  const report = page.locator(".staff-operational-report-section").filter({
    has: page.getByRole("heading", { name: "Academic assignments" }),
  });
  await expect(report.getByText("teacher.subject-only")).toBeVisible();
  await expect(report.getByText("teacher.incharge-secret")).toHaveCount(0);
  await expect(report.getByRole("columnheader", { name: "Class-in-charge" })).toHaveCount(0);
  await expect.poll(() => inchargeAssignmentRequests).toBe(0);

  const downloadPromise = page.waitForEvent("download");
  await report.getByRole("button", { name: "Export assignments CSV" }).click();
  const download = await downloadPromise;
  const csvPath = await download.path();
  expect(csvPath).toBeTruthy();
  const csv = await readFile(csvPath!, "utf8");
  expect(csv).toContain("teacher.subject-only");
  expect(csv).toContain("Subject assignments");
  expect(csv).not.toContain("Mathematics");
  expect(csv).not.toContain("teacher.incharge-secret");
  expect(csv).not.toContain("Class-in-charge assignments");
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
      id: 1, principal_id: 501, school_id: 10, username: "teacher.original", staff_type: "teaching",
      employee_code: "EMP-ORIGINAL", full_name: '\t=HYPERLINK("https://example.invalid","Click")', designation: "Teacher",
      mobile_number: "9876543215", employment_status: "active", profile_complete: true,
      status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 502, school_id: 10, username: "teacher.filtered", staff_type: "non_teaching",
      employee_code: "EMP-FILTERED", full_name: "Filtered Out Teacher", designation: "Teacher",
      mobile_number: "9876543216", employment_status: "inactive", profile_complete: false,
      status: "disabled", must_change_password: false,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Staff Directory", exact: true }).click();
  await page.getByRole("button", { name: "Filter by staff type" }).click();
  await page.getByRole("option", { name: "Teaching", exact: true }).click();
  await page.getByLabel("Search staff by name, code, designation, mobile or username").fill("Teacher");
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


test("staff directory export refuses a changed total instead of downloading partial rows", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = Array.from({ length: 51 }, (_, index) => ({
    id: index + 1, principal_id: 501 + index, school_id: 10, username: "directory." + (index + 1),
    employee_code: "EMP-DIR-" + String(index + 1).padStart(3, "0"), full_name: "Directory Teacher " + (index + 1),
    designation: "Teacher", employment_status: "active", profile_complete: true, status: "active",
    must_change_password: false,
  }));
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: "Staff Directory", exact: true }).click();
  await expect(page.getByText("Directory Teacher 1", { exact: true })).toBeVisible();
  await page.route("**/api/school/staff?**", async route => {
    const url = new URL(route.request().url());
    const limit = Number(url.searchParams.get("limit") ?? "50");
    const offset = Number(url.searchParams.get("offset") ?? "0");
    await route.fulfill({
      json: {
        staff: rows.slice(offset, offset + limit),
        total: offset === 0 ? 51 : 52,
        limit,
        offset,
      },
    });
  });
  const noDownload = page.waitForEvent("download", { timeout: 1000 }).catch(() => null);
  await page.getByRole("button", { name: "Export CSV" }).click();
  await expect(page.getByRole("alert").filter({ hasText: "No partial CSV was created" })).toBeVisible();
  expect(await noDownload).toBeNull();
});

test("staff directory filters stay locked while the complete CSV is being prepared", async ({ page }) => {
  const rows:Array<Record<string,unknown>>=Array.from({length:51},(_,index)=>({
    id:index+1,principal_id:801+index,school_id:10,username:"directory.lock."+index,
    employee_code:"EMP-LOCK-"+index,full_name:"Directory Lock "+index,designation:"Teacher",
    employment_status:"active",profile_complete:true,status:"active",must_change_password:false,
  }));
  await signInAndOpenStaff(page,rows,undefined,["users.read"]);
  await page.getByRole("button",{name:"Staff Directory",exact:true}).click();
  await page.getByLabel("Filter by employment status").click();
  await page.getByRole("option",{name:"Active",exact:true}).click();
  await page.route("**/api/school/staff?**",async route=>{
    const url=new URL(route.request().url());
    const limit=Number(url.searchParams.get("limit")??50);
    const offset=Number(url.searchParams.get("offset")??0);
    if(offset===50)await new Promise(resolve=>setTimeout(resolve,250));
    await route.fulfill({json:{staff:rows.slice(offset,offset+limit),total:51,limit,offset}});
  });
  const secondPage=page.waitForRequest(request=>new URL(request.url()).searchParams.get("offset")==="50");
  const downloadPromise=page.waitForEvent("download");
  await page.getByRole("button",{name:"Export CSV"}).click();
  await secondPage;
  await expect(page.getByLabel("Search staff by name, code, designation, mobile or username")).toBeDisabled();
  await expect(page.getByLabel("Filter by employment status")).toBeDisabled();
  await expect(page.getByRole("button",{name:"Clear filters"})).toBeDisabled();
  const download=await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/^staff-directory-\d{4}-\d{2}-\d{2}\.csv$/);
  await expect(page.getByLabel("Search staff by name, code, designation, mobile or username")).toBeEnabled();
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
      staff_name: "Active Teacher", record_name: "Child Safety Training", organization: "\t=HYPERLINK(\"x\")",
      record_date: "2025-04-01", expiry_date: "2027-04-01", verification_status: "pending",
    }] : [{
      record_type: "qualification", record_id: 11, staff_principal_id: 501, employee_code: "EMP-ACTIVE",
      staff_name: "Active Teacher", record_name: "B.Ed.", organization: "State University",
      record_date: "2022", expiry_date: "", verification_status: "verified",
    }, {
      record_type: "certification", record_id: 22, staff_principal_id: 501, employee_code: "EMP-ACTIVE",
      staff_name: "Active Teacher", record_name: "Child Safety Training", organization: "\t=HYPERLINK(\"x\")",
      record_date: "2025-04-01", expiry_date: "2027-04-01", verification_status: "pending",
    }];
    await route.fulfill({ json: { report: {
      rows, limit: Number(url.searchParams.get("limit") ?? 50), offset: Number(url.searchParams.get("offset") ?? 0), total: certificationOnly ? 1 : 2,
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
  await qualificationSection.getByRole("button", { name: "Export all matching CSV" }).click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("Child Safety Training");
  expect(csv).toContain("'\t=HYPERLINK(");
  expect(csv).not.toContain("credential");
  expect(csv).not.toContain("verification_notes");
  expect(requests.some(request => request.includes("limit=200") && request.includes("offset=0"))).toBe(true);
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
    await route.fulfill({json:{report:{rows,limit:Number(url.searchParams.get("limit")??50),offset:Number(url.searchParams.get("offset")??0),total:current?1:2,counts:current?[{employment_type:"permanent",is_current:true,count:1}]:[
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
  await page.getByRole("button",{name:"Export all matching CSV"}).last().click();
  const download=await downloadPromise;const path=await download.path();expect(path).toBeTruthy();
  const csv=await readFile(path!,"utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  expect(csv).toContain("Current School");
  expect(csv).not.toContain("description");
  expect(requests.some(request => request.includes("limit=200") && request.includes("offset=0"))).toBe(true);
});


test("staff qualification and experience exports reject duplicate records without partial downloads", async ({ page }) => {
  const staffRows:Array<Record<string,unknown>>=[
    {id:1,principal_id:501,school_id:10,username:"teacher.active",employee_code:"EMP-ACTIVE",full_name:"Active Teacher",designation:"Mathematics Teacher",employment_status:"active",profile_complete:true,status:"active",must_change_password:false},
  ];
  await signInAndOpenStaff(page,staffRows,undefined,["users.read"]);
  await page.route("**/api/school/staff/qualifications-report**",async route=>{
    const url=new URL(route.request().url());
    const row={record_type:"qualification",record_id:71,staff_principal_id:501,employee_code:"EMP-ACTIVE",staff_name:"Active Teacher",record_name:"B.Ed.",organization:"University",record_date:"2022",expiry_date:"",verification_status:"verified"};
    await route.fulfill({json:{report:{rows:[row,{...row}],limit:Number(url.searchParams.get("limit")??50),offset:Number(url.searchParams.get("offset")??0),total:2,counts:[]}}});
  });
  await page.route("**/api/school/staff/experience-report**",async route=>{
    const url=new URL(route.request().url());
    const row={record_id:81,staff_principal_id:501,employee_code:"EMP-ACTIVE",staff_name:"Active Teacher",designation:"Teacher",employment_status:"active",employer_name:"School",previous_designation:"Teacher",employment_type:"permanent",start_date:"2020-04-01",end_date:"",is_current:true};
    await route.fulfill({json:{report:{rows:[row,{...row}],limit:Number(url.searchParams.get("limit")??50),offset:Number(url.searchParams.get("offset")??0),total:2,counts:[]}}});
  });
  await page.getByRole("button",{name:"Reports",exact:true}).click();
  const qualificationSection=page.locator(".staff-operational-report-section").filter({has:page.getByRole("heading",{name:"Qualifications & certifications"})});
  const noQualificationDownload=page.waitForEvent("download",{timeout:1000}).catch(()=>null);
  await qualificationSection.getByRole("button",{name:"Export all matching CSV"}).click();
  await expect(qualificationSection.getByRole("alert").filter({hasText:"No partial CSV was created"})).toBeVisible();
  expect(await noQualificationDownload).toBeNull();

  const experienceSection=page.locator(".staff-operational-report-section").filter({has:page.getByRole("heading",{name:"Staff experience history"})});
  const noExperienceDownload=page.waitForEvent("download",{timeout:1000}).catch(()=>null);
  await experienceSection.getByRole("button",{name:"Export all matching CSV"}).click();
  await expect(experienceSection.getByRole("alert").filter({hasText:"No partial CSV was created"})).toBeVisible();
  expect(await noExperienceDownload).toBeNull();
});

test("staff ID cards select complete active employees regardless of login status and expose approved print fields", async ({ page }) => {
  const rows:Array<Record<string,unknown>>=[
    {id:1,principal_id:501,school_id:10,username:"active.teacher",employee_code:"EMP-501",full_name:"Active Teacher",designation:"Mathematics Teacher",mobile_number:"9876543210",email:"private@example.test",joining_date:"2024-04-01",employment_status:"active",profile_complete:true,status:"active",must_change_password:false},
    {id:2,principal_id:502,school_id:10,username:"inactive.teacher",employee_code:"EMP-502",full_name:"Inactive Teacher",designation:"Former Teacher",employment_status:"inactive",profile_complete:true,status:"disabled",must_change_password:false},
    {id:3,principal_id:503,school_id:10,username:"incomplete.teacher",employee_code:"",full_name:"Incomplete Teacher",designation:"",employment_status:"active",profile_complete:false,status:"active",must_change_password:false},
    {id:4,principal_id:504,school_id:10,username:"unknown.profile",employee_code:"EMP-504",full_name:"Unknown Profile",designation:"Teacher",employment_status:"active",status:"active",must_change_password:false},
    {id:5,principal_id:505,school_id:10,username:"disabled.login.active.employee",employee_code:"EMP-505",full_name:"Disabled Login Teacher",designation:"English Teacher",joining_date:"2025-06-01",employment_status:"active",profile_complete:true,status:"disabled",must_change_password:false},
  ];
  await signInAndOpenStaff(page,rows,undefined,["users.read"]);
  await page.getByRole("button",{name:"ID Cards",exact:true}).click();
  await expect(page.getByRole("heading",{name:"Staff ID cards"})).toBeVisible();
  await expect(page.getByLabel(/Inactive Teacher/)).toHaveCount(0);
  await expect(page.getByLabel(/Incomplete Teacher/)).toHaveCount(0);
  await expect(page.getByLabel(/Unknown Profile/)).toHaveCount(0);
  await expect(page.getByLabel(/Disabled Login Teacher/)).toBeVisible();
  await page.getByRole("button",{name:"Select all eligible"}).click();
  const preview = page.locator(".staff-id-print-area");
  await expect(preview.getByText("Active Teacher",{exact:true})).toBeVisible();
  await expect(preview.getByText("EMP-501",{exact:true})).toBeVisible();
  await expect(preview.getByText("Disabled Login Teacher",{exact:true})).toBeVisible();
  await expect(preview.getByText("EMP-505",{exact:true})).toBeVisible();
  await expect(preview.locator(".staff-id-card")).toHaveCount(2);
  await expect(preview.getByText("1 Apr 2024",{exact:true})).toBeVisible();
  await expect(preview.getByText("Active",{exact:true}).first()).toBeVisible();
  await expect(preview.getByText("Mathematics Teacher",{exact:true})).toBeVisible();
  await expect(preview.locator(".staff-id-avatar")).toHaveText(["AT", "DL"]);
  const validUntilPicker = page.getByRole("button", { name: "ID card valid until" });
  await expect(validUntilPicker).toContainText("No expiry date");
  await expect(page.locator('.staff-id-controls input[type="date"]')).toHaveCount(0);
  await validUntilPicker.click();
  const expiryCalendar = page.getByRole("dialog", { name: "Choose id card valid until" });
  await expect(expiryCalendar.getByRole("button", { name: schoolDateLabel(schoolDateOffset(-1)) })).toBeDisabled();
  await expiryCalendar.getByRole("button", { name: "Today", exact: true }).click();
  await expect(validUntilPicker).toContainText(new Date(schoolDateValue() + "T12:00:00").toLocaleDateString("en-IN", { day: "2-digit", month: "short", year: "numeric" }));
  await expect(page.getByText("private@example.test")).toHaveCount(0);
  await expect(page.getByText("9876543210")).toHaveCount(0);
  await expect(preview.getByText("active.teacher")).toHaveCount(0);
  await expect(preview.locator("canvas, svg, img, [data-qr]")).toHaveCount(0);
  await expect(preview).not.toContainText(/password|credential|login status/i);
  await page.getByLabel("Card template").selectOption("modern");
  await expect(page.locator(".staff-id-card--modern")).toHaveCount(2);
  await page.setViewportSize({width:390,height:844});
  await expect.poll(() => page.locator(".staff-id-selection").evaluate(element => getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length)).toBe(1);
  await expect(page.locator(".staff-id-card")).toHaveCount(2);
  await expect(page.locator(".staff-id-card").nth(0)).toBeVisible();
  await expect(page.locator(".staff-id-card").nth(1)).toBeVisible();
  await page.setViewportSize({width:768,height:900});
  await expect.poll(() => page.locator(".staff-id-selection").evaluate(element => getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length)).toBe(2);
  await page.setViewportSize({width:1280,height:900});
  await expect.poll(() => page.locator(".staff-id-selection").evaluate(element => getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length)).toBe(3);
  await page.evaluate(() => {
    Object.defineProperty(window,"__staffPrintCalled",{value:false,writable:true,configurable:true});
    window.print = () => { (window as Window & {__staffPrintCalled?:boolean}).__staffPrintCalled = true; };
  });
  await page.getByRole("button",{name:"Print 2 cards"}).click();
  await expect.poll(() => page.evaluate(() => Boolean((window as Window & {__staffPrintCalled?:boolean}).__staffPrintCalled))).toBe(true);
  await page.emulateMedia({media:"print"});
  await expect(page.locator(".staff-workspace-tabs")).toHaveCSS("visibility","hidden");
  await expect(preview).toHaveCSS("visibility","visible");
  await expect(preview).toHaveCSS("display","grid");
  await expect(page.locator(".staff-id-card")).toHaveCount(2);
  await expect(page.locator(".staff-id-card").nth(0)).toHaveCSS("box-sizing","border-box");
  await expect(page.locator(".staff-id-card").nth(1)).toHaveCSS("box-sizing","border-box");
  await expect.poll(() => page.locator(".staff-id-card").nth(0).evaluate(element => Number.parseFloat(getComputedStyle(element).width))).toBeGreaterThan(321);
  await expect.poll(() => page.locator(".staff-id-card").nth(1).evaluate(element => Number.parseFloat(getComputedStyle(element).width))).toBeGreaterThan(321);
  await page.emulateMedia({media:"screen"});
  await page.getByRole("button",{name:"Clear selection"}).click();
  await expect(page.getByText("Select one or more employees to preview their cards.")).toBeVisible();
  await expect(page.getByRole("button",{name:/Print/})).toBeDisabled();
});


test("holiday-only leave access does not expose write-only leave request data or actions", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.holidayonly",
    employee_code: "EMP-HOLIDAY-01", full_name: "Holiday Only", designation: "Staff Manager",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  const leaveRequestCalls: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/api/school/staff-leave/requests")) leaveRequestCalls.push(request.url());
  });
  const permissions = ["users.read", "staff.leave.manage_requests", "staff.holidays.read"];
  await signInAndOpenStaff(page, rows, undefined, permissions);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  const leaveWorkspace = page.locator(".staff-leave-workspace");
  await expect(leaveWorkspace.getByRole("button", { name: "Holiday calendar" })).toBeVisible();
  await expect(leaveWorkspace.getByRole("button", { name: "Leave requests" })).toHaveCount(0);
  await expect(leaveWorkspace.getByRole("heading", { name: /School holiday calendar/ })).toBeVisible();
  await expect(leaveWorkspace.getByRole("heading", { name: "Applications" })).toHaveCount(0);
  await expect(leaveWorkspace.getByRole("heading", { name: "New leave request" })).toHaveCount(0);
  expect(leaveRequestCalls).toHaveLength(0);
});


test("staff leave and holiday workspace remains usable without horizontal overflow on mobile, tablet and desktop", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "responsive.staff",
    employee_code: "EMP-RESP-01", full_name: "Responsive Staff", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, [
    "users.read", "staff.leave.read", "staff.leave.manage_requests",
    "staff.leave.decide", "staff.holidays.read",
  ]);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  const workspace = page.locator(".staff-leave-workspace");
  await expect(workspace.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  const grid = workspace.locator(".staff-leave-grid");
  await expect(grid).toBeVisible();

  for (const viewport of [
    { width: 390, height: 844, columns: 1 },
    { width: 768, height: 900, columns: 1 },
    { width: 1280, height: 900, columns: 2 },
  ]) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await expect.poll(async () => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect.poll(async () => grid.evaluate(element =>
      getComputedStyle(element).gridTemplateColumns.trim().split(/\s+/).length,
    )).toBe(viewport.columns);
    await expect(workspace.getByRole("button", { name: "Leave requests", exact: true })).toBeVisible();
    await expect(workspace.getByRole("button", { name: "Holiday calendar" })).toBeVisible();
  }
});

test("attendance-only staff access does not request or expose the staff directory", async ({ page }) => {
  const staffListRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname === "/api/school/staff" && request.method() === "GET") {
      staffListRequests.push(request.method() + " " + url.pathname);
    }
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.attendance.read"]);
  await page.route("**/api/school/staff-attendance?*", async route => {
    const date = new URL(route.request().url()).searchParams.get("date");
    await route.fulfill({ json: { date, data: [] } });
  });

  await expect(page.getByRole("heading", { name: "Staff attendance" })).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Staff management sections" }).getByRole("button", { name: "Overview" })).toHaveCount(0);
  await expect(page.getByRole("navigation", { name: "Staff management sections" }).getByRole("button", { name: "Staff Directory" })).toHaveCount(0);
  expect(staffListRequests).toEqual([]);
});

test("leave-only staff access opens leave applications without requesting the staff directory", async ({ page }) => {
  const staffListRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname === "/api/school/staff" && request.method() === "GET") {
      staffListRequests.push(request.method() + " " + url.pathname);
    }
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.leave.read"]);

  const workspace = page.locator(".staff-leave-workspace");
  await expect(workspace.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  await expect(workspace.getByRole("heading", { name: "Applications" })).toBeVisible();
  await expect(workspace.getByText("Leave Teacher")).toBeVisible();
  await expect(page.getByRole("navigation", { name: "Staff management sections" }).getByRole("button", { name: "Overview" })).toHaveCount(0);
  await expect(page.getByRole("navigation", { name: "Staff management sections" }).getByRole("button", { name: "Staff Directory" })).toHaveCount(0);
  expect(staffListRequests).toEqual([]);
});

test("leave request managers without staff-directory access get a clear employee-selection gate", async ({ page }) => {
  const staffListRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname === "/api/school/staff" && request.method() === "GET") {
      staffListRequests.push(request.method() + " " + url.pathname);
    }
  });
  await signInAndOpenStaff(page, [], undefined, ["staff.leave.read", "staff.leave.manage_requests"]);

  const workspace = page.locator(".staff-leave-workspace");
  await expect(workspace.getByRole("heading", { name: "Leave & Holidays" })).toBeVisible();
  await expect(workspace.getByRole("heading", { name: "Applications" })).toBeVisible();
  await expect(workspace.getByRole("status").filter({ hasText: "Creating a leave request on behalf of an employee also requires Staff Directory read access" })).toBeVisible();
  await expect(workspace.getByRole("heading", { name: "New leave request" })).toHaveCount(0);
  expect(staffListRequests).toEqual([]);
});

test("leave request employee selector includes only active staff with explicitly complete profiles", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 581, school_id: 10, username: "leave.complete", employee_code: "EMP-LEAVE-OK-01", full_name: "Complete Profile Staff", designation: "Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 582, school_id: 10, username: "leave.incomplete", employee_code: "EMP-LEAVE-NO-01", full_name: "Incomplete Profile Staff", designation: "Teacher", employment_status: "active", profile_complete: false, status: "active", must_change_password: false },
    { id: 3, principal_id: 583, school_id: 10, username: "leave.unknown", employee_code: "EMP-LEAVE-UNKNOWN-01", full_name: "Unknown Profile Staff", designation: "Teacher", employment_status: "active", status: "active", must_change_password: false },
    { id: 4, principal_id: 584, school_id: 10, username: "leave.inactive", employee_code: "EMP-LEAVE-INACTIVE-01", full_name: "Inactive Complete Staff", designation: "Teacher", employment_status: "inactive", profile_complete: true, status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read", "staff.leave.manage_requests"]);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();

  const workspace = page.locator(".staff-leave-workspace");
  await expect(workspace.getByRole("heading", { name: "New leave request" })).toBeVisible();
  const employeeSelect = workspace.getByRole("button", { name: "Staff member" });
  await employeeSelect.click();
  const employeeOptions = workspace.getByRole("listbox", { name: "Staff member" }).getByRole("option");
  await expect(employeeOptions).toHaveCount(1);
  await expect(employeeOptions.first()).toHaveText(/Complete Profile Staff/);
  await expect(employeeOptions.filter({ hasText: "Incomplete Profile Staff" })).toHaveCount(0);
  await expect(employeeOptions.filter({ hasText: "Unknown Profile Staff" })).toHaveCount(0);
  await expect(employeeOptions.filter({ hasText: "Inactive Complete Staff" })).toHaveCount(0);
});

test("leave request managers see an empty-state gate when no eligible staff exist", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 585, school_id: 10, username: "leave.incompleteonly", employee_code: "EMP-LEAVE-NO-02", full_name: "Incomplete Only Staff", designation: "Teacher", employment_status: "active", profile_complete: false, status: "active", must_change_password: false },
    { id: 2, principal_id: 586, school_id: 10, username: "leave.unknownonly", employee_code: "EMP-LEAVE-UNKNOWN-02", full_name: "Unknown Only Staff", designation: "Teacher", employment_status: "active", status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.leave.read", "staff.leave.manage_requests"]);
  await page.getByRole("navigation", { name: "Staff management sections" }).getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.locator(".staff-leave-workspace")).toBeVisible();

  const workspace = page.locator(".staff-leave-workspace");
  await expect(workspace.getByRole("status").filter({ hasText: "No eligible active staff members are available for a new leave request" })).toBeVisible();
  await expect(workspace.getByRole("heading", { name: "New leave request" })).toHaveCount(0);
});

test("staff daily attendance search and status filters apply across the full roster before pagination", async ({ page }) => {
  const directoryRows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 590, school_id: 10, username: "attendance.search.admin",
    employee_code: "EMP-ATT-SEARCH-ADMIN", full_name: "Attendance Search Admin", designation: "Administrator",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, directoryRows, undefined, ["users.read", "staff.attendance.read"]);
  const all = Array.from({ length: 121 }, (_, index) => ({
    id: index === 0 ? 900 : null,
    school_id: 10,
    staff_principal_id: 600 + index,
    employee_code: `EMP-ATT-${String(index + 1).padStart(3, "0")}`,
    full_name: `Attendance Staff ${String(index + 1).padStart(3, "0")}`,
    designation: "Teacher",
    attendance_date: "2026-11-26",
    status: index === 0 ? "present" : null,
  }));
  await page.route("**/api/school/staff-attendance?*", async route => {
    const url = new URL(route.request().url());
    const search = (url.searchParams.get("search") ?? "").toLocaleLowerCase();
    const status = url.searchParams.get("status") ?? "all";
    const filtered = all.filter(item => {
      const matchesSearch = !search || [item.full_name, item.employee_code, item.designation].some(value => value.toLocaleLowerCase().includes(search));
      const matchesStatus = status === "all" || (status === "unmarked" ? !item.status : item.status === status);
      return matchesSearch && matchesStatus;
    });
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    await route.fulfill({ json: { date: url.searchParams.get("date"), data: filtered.slice(offset, offset + limit), pagination: { limit, offset, total: filtered.length } } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await expect(page.getByText("Showing 1–50 of 121 staff matching filters")).toBeVisible();
  await page.getByLabel("Search staff attendance").fill("Attendance Staff 120");
  await expect(page.getByRole("row").filter({ hasText: "Attendance Staff 120" })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1 staff matching filters")).toBeVisible();
  await expect(page.getByRole("row").filter({ hasText: "Attendance Staff 001" })).toHaveCount(0);
  await page.getByLabel("Filter staff attendance status").selectOption("present");
  await expect(page.getByText("No staff match these filters.")).toBeVisible();
  await page.getByLabel("Filter staff attendance status").selectOption("unmarked");
  await expect(page.getByRole("row").filter({ hasText: "Attendance Staff 120" })).toBeVisible();
  await expect(page.getByText("Showing 1–1 of 1 staff matching filters")).toBeVisible();
});

test("staff attendance workspace remains usable on mobile, tablet and desktop", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 521, school_id: 10, username: "attendance.responsive", employee_code: "EMP-RESP-ATT-01", full_name: "Responsive Attendance Teacher", designation: "Science Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 522, school_id: 10, username: "attendance.responsive2", employee_code: "EMP-RESP-ATT-02", full_name: "Second Attendance Teacher", designation: "English Teacher", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  await signInAndOpenStaff(page, rows, undefined, [
    "users.read", "staff.attendance.read", "staff.attendance.mark",
  ]);
  await page.route("**/api/school/staff-attendance?*", async route => {
    const url = new URL(route.request().url());
    const all = rows.map(row => ({
      id: null, school_id: 10, staff_principal_id: row.principal_id,
      employee_code: row.employee_code, full_name: row.full_name, designation: row.designation,
      attendance_date: url.searchParams.get("date"), status: null,
    }));
    const search = (url.searchParams.get("search") ?? "").toLocaleLowerCase();
    const filtered = all.filter(item => !search || [item.full_name, item.employee_code, item.designation].some(value => value.toLocaleLowerCase().includes(search)));
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    await route.fulfill({ json: { date: url.searchParams.get("date"), data: filtered.slice(offset, offset + limit), pagination: { limit, offset, total: filtered.length } } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const workspace = page.locator(".staff-attendance-panel");
  await expect(workspace.getByRole("heading", { name: "Staff attendance" })).toBeVisible();
  const tableWrap = workspace.locator(".staff-attendance-table-wrap");
  await expect(tableWrap).toBeVisible();

  for (const viewport of [
    { width: 390, height: 844 },
    { width: 768, height: 900 },
    { width: 1280, height: 900 },
  ]) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect(workspace.getByLabel("Search staff attendance")).toBeVisible();
    await expect(workspace.getByLabel("Filter staff attendance status")).toBeVisible();
    await expect(workspace.getByLabel("Attendance status for Responsive Attendance Teacher")).toBeVisible();
    await expect.poll(() => tableWrap.evaluate(element => element.scrollWidth >= element.clientWidth)).toBe(true);
  }
  await workspace.getByLabel("Search staff attendance").fill("Second Attendance");
  await expect(workspace.getByRole("row").filter({ hasText: "Second Attendance Teacher" })).toBeVisible();
  await expect(workspace.getByRole("row").filter({ hasText: "Responsive Attendance Teacher" })).toHaveCount(0);
});

test("staff monthly attendance summary remains contained and readable across viewports", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 531, school_id: 10, username: "attendance.monthlyresponsive",
    employee_code: "EMP-MONTH-RESP-01", full_name: "Monthly Responsive Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read"]);
  await page.route("**/api/school/staff-attendance/month?*", async route => {
    const url = new URL(route.request().url());
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    const data = [{
      staff_principal_id: 531, employee_code: "EMP-MONTH-RESP-01", full_name: "Monthly Responsive Teacher",
      designation: "Teacher", present_days: 18, absent_days: 1, late_days: 2, half_days: 0, records: [],
    }];
    await route.fulfill({ json: { month: url.searchParams.get("month"), data: data.slice(offset, offset + limit), pagination: { limit, offset, total: data.length } } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const workspace = page.locator(".staff-attendance-panel");
  await page.getByRole("button", { name: "Monthly summary" }).click();
  await expect(workspace.getByRole("cell", { name: "Monthly Responsive Teacher" })).toBeVisible();
  const tableWrap = workspace.locator(".staff-attendance-table-wrap");
  for (const viewport of [
    { width: 390, height: 844 },
    { width: 768, height: 900 },
    { width: 1280, height: 900 },
  ]) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect(workspace.getByLabel("Attendance month")).toBeVisible();
    await expect(workspace.getByRole("button", { name: "Download complete monthly report" })).toBeVisible();
    await expect(tableWrap).toBeVisible();
    if (viewport.width <= 768) {
      await expect.poll(() => tableWrap.evaluate(element => element.scrollWidth > element.clientWidth)).toBe(true);
    } else {
      await expect.poll(() => tableWrap.evaluate(element => element.scrollWidth === element.clientWidth)).toBe(true);
    }
  }
});

test("staff attendance monthly summary does not request data for an invalid month", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 532, school_id: 10, username: "attendance.invalidmonth",
    employee_code: "EMP-MONTH-INVALID", full_name: "Invalid Month Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  let monthlyRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read"]);
  await page.route("**/api/school/staff-attendance/month?*", async route => {
    monthlyRequests += 1;
    const url = new URL(route.request().url());
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    const data = [{
      staff_principal_id: 532, employee_code: "EMP-MONTH-INVALID", full_name: "Invalid Month Teacher",
      designation: "Teacher", present_days: 18, absent_days: 0, late_days: 0, half_days: 0, records: [],
    }];
    await route.fulfill({ json: { month: url.searchParams.get("month"), data: data.slice(offset, offset + limit), pagination: { limit, offset, total: data.length } } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  const workspace = page.locator(".staff-attendance-panel");
  await workspace.getByRole("button", { name: "Monthly summary" }).click();
  await expect(workspace.getByRole("cell", { name: "Invalid Month Teacher" })).toBeVisible();
  await expect.poll(() => monthlyRequests).toBe(1);

  await workspace.getByLabel("Attendance month").fill("");
  await expect(workspace.getByRole("alert")).toContainText("Select a valid attendance month");
  await expect(workspace.getByRole("button", { name: "Download complete monthly report" })).toBeDisabled();
  await expect(workspace.locator(".staff-attendance-table-wrap")).toHaveCount(0);
  await expect(workspace.getByText("No active staff profiles are available for this school.")).toHaveCount(0);
  expect(monthlyRequests).toBe(1);
});

test("staff attendance monthly context uses correct dates for years below 100", async ({ page }) => {
  await signInAndOpenStaff(page, [], undefined, [
    "staff.attendance.read", "staff.leave.read", "staff.holidays.read",
  ]);
  const requested: string[] = [];
  await page.route("**/api/school/staff-attendance/month?*", async route => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("month") === "0001-02") requested.push("attendance:" + url.searchParams.get("month"));
    const limit = Number(url.searchParams.get("limit") ?? 50);
    const offset = Number(url.searchParams.get("offset") ?? 0);
    await route.fulfill({ json: { month: url.searchParams.get("month"), data: [], pagination: { limit, offset, total: 0 } } });
  });
  await page.route("**/api/school/staff-leave/requests?*", async route => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("from_date") === "0001-02-01") {
      requested.push("leave:" + url.searchParams.get("from_date") + ":" + url.searchParams.get("to_date"));
    }
    const page = Number(url.searchParams.get("page") ?? 1);
    const pageSize = Number(url.searchParams.get("page_size") ?? 100);
    await route.fulfill({ json: { data: [], pagination: { page, page_size: pageSize, total: 0 } } });
  });
  await page.route("**/api/school/holidays?*", async route => {
    const url = new URL(route.request().url());
    if (url.searchParams.get("from") === "0001-02-01") {
      requested.push("holiday:" + url.searchParams.get("from") + ":" + url.searchParams.get("to"));
    }
    await route.fulfill({ json: { data: [] } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await page.getByRole("button", { name: "Monthly summary" }).click();
  const month = page.getByLabel("Attendance month");
  await month.fill("0001-02");
  await expect.poll(() => requested.length).toBe(3);
  expect(requested).toContain("attendance:0001-02");
  expect(requested).toContain("leave:0001-02-01:0001-02-28");
  expect(requested).toContain("holiday:0001-02-01:0001-02-28");
});

test("staff profile dialog and section navigation remain responsive on mobile, tablet and desktop", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 541, school_id: 10, username: "profile.responsive",
    employee_code: "EMP-PRO-RESP-01", full_name: "Responsive Profile Teacher", designation: "Teacher",
    mobile_number: "9876543210", email: "responsive.profile@example.test", joining_date: "2025-04-01",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  await signInAndOpenStaff(page, rows, undefined, ["users.read"]);
  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Responsive Profile Teacher" }).click();
  const dialog = page.getByRole("dialog", { name: "Responsive Profile Teacher" });
  const navigation = dialog.getByRole("navigation", { name: "Staff profile sections" });
  await expect(navigation.getByRole("button")).toHaveCount(8);

  for (const viewport of [
    { width: 320, height: 720 },
    { width: 360, height: 800 },
    { width: 390, height: 844 },
    { width: 768, height: 900 },
    { width: 1280, height: 900 },
  ]) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
    await expect.poll(() => dialog.evaluate(element => {
      const rect = element.getBoundingClientRect();
      return rect.left >= 0 && rect.right <= window.innerWidth;
    })).toBe(true);
    await expect(navigation).toBeVisible();
    await expect(dialog.getByText("EMP-PRO-RESP-01").first()).toBeVisible();
    if (viewport.width <= 768) {
      await expect.poll(() => navigation.evaluate(element => element.scrollWidth > element.clientWidth)).toBe(true);
    }
  }
  await page.setViewportSize({ width: 844, height: 390 });
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await expect.poll(() => dialog.evaluate(element => {
    const rect = element.getBoundingClientRect();
    return rect.left >= 0 && rect.right <= window.innerWidth;
  })).toBe(true);
  await expect(navigation).toBeVisible();
  await navigation.getByRole("button", { name: "Personal Details" }).click();
  await expect(dialog.getByText("responsive.profile@example.test")).toBeVisible();
});


test("read-only substitution access lists coverage without exposing management actions or timetable data", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 551, school_id: 10, username: "substitution.reader",
    employee_code: "EMP-SUB-READ-01", full_name: "Substitution Reader", designation: "School Admin",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  const requests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname.includes("/staff-substitutions") || url.pathname.includes("/timetable/")) {
      requests.push(request.method() + " " + url.pathname);
    }
  });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.substitutions.read"]);
  await page.route("**/api/school/staff-substitutions*", async route => {
    if (route.request().method() === "GET") {
      await route.fulfill({ json: { substitutions: [{
        id: 91, timetable_entry_id: 701, cover_date: schoolDateValue(),
        original_teacher_principal_id: 501, original_teacher_name: "Original Teacher",
        substitute_teacher_principal_id: 502, substitute_teacher_name: "Cover Teacher",
        reason: "Approved cover", status: "active", section_name: "A", class_name: "Class 6",
        subject_name: "Mathematics", period_number: 2, period_name: "Period 2",
        starts_at: "09:40", ends_at: "10:20", room: "Room 12",
        created_by_principal_id: 101, created_at: "2026-11-22T10:00:00.000Z",
      }] } });
      return;
    }
    await route.fulfill({ status: 403, json: { message: "Forbidden" } });
  });
  await page.getByRole("button", { name: "Substitutions" }).click();

  const workspace = page.locator(".staff-substitution-workspace");
  await expect(workspace.getByRole("heading", { name: "Substitute Teacher Coverage" })).toBeVisible();
  await expect(workspace.getByText("Original Teacher")).toBeVisible();
  await expect(workspace.getByText("Cover Teacher")).toBeVisible();
  await expect(workspace.getByRole("button", { name: "Cancel cover" })).toHaveCount(0);
  await expect(workspace.getByRole("button", { name: "Assign substitute" })).toHaveCount(0);
  for (const viewport of [{ width: 390, height: 844 }, { width: 768, height: 900 }, { width: 1280, height: 900 }]) {
    await page.setViewportSize(viewport);
    await expect(workspace.getByLabel("Substitution list from date")).toBeVisible();
    await expect(workspace.getByLabel("Substitution list to date")).toBeVisible();
    await expect(workspace.getByLabel("Filter substitutions by staff")).toBeVisible();
    const dimensions = await page.evaluate(() => ({
      viewport: window.innerWidth,
      document: document.documentElement.scrollWidth,
    }));
    expect(dimensions.document).toBeLessThanOrEqual(dimensions.viewport);
  }
  expect(requests.some(item => item.startsWith("POST "))).toBe(false);
  expect(requests.some(item => item.includes("/timetable/"))).toBe(false);
});


test("attendance read-only permission cannot mark, update or bulk-save records", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 514, school_id: 10, username: "attendance.reader",
    employee_code: "EMP-ATT-READ-01", full_name: "Read Only Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  const writes: string[] = [];
  page.on("request", request => {
    if (request.url().includes("/api/school/staff-attendance") && request.method() !== "GET") {
      writes.push(request.method() + " " + new URL(request.url()).pathname);
    }
  });
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "staff.attendance.read"]);
  await page.route("**/api/school/staff-attendance**", async route => {
    if (route.request().method() !== "GET") {
      await route.fulfill({ status: 403, json: { message: "Forbidden" } });
      return;
    }
    const date = new URL(route.request().url()).searchParams.get("date") ?? schoolDateValue();
    await route.fulfill({ json: { date, data: [
      { id: null, school_id: 10, staff_principal_id: 514, employee_code: "EMP-ATT-READ-01", full_name: "Read Only Teacher", designation: "Teacher", attendance_date: date, status: null },
      { id: 52, school_id: 10, staff_principal_id: 515, employee_code: "EMP-ATT-READ-02", full_name: "Already Marked Teacher", designation: "Teacher", attendance_date: date, status: "present" },
    ] } });
  });
  await page.getByRole("button", { name: "Attendance", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Staff attendance" })).toBeVisible();
  await expect(page.getByLabel("Attendance status for Read Only Teacher")).toBeDisabled();
  await expect(page.getByLabel("Remarks for Read Only Teacher")).toBeDisabled();
  await expect(page.getByRole("button", { name: "Mark", exact: true }).last()).toBeDisabled();
  await expect(page.getByLabel("Attendance status for Already Marked Teacher")).toBeDisabled();
  await expect(page.getByLabel("Remarks for Already Marked Teacher")).toBeDisabled();
  await expect(page.getByRole("button", { name: "Update", exact: true })).toBeDisabled();
  await expect(page.getByLabel("Bulk attendance status")).toBeDisabled();
  await expect(page.getByLabel("Bulk attendance remarks")).toBeDisabled();
  await expect(page.getByRole("button", { name: /Mark .* unmarked staff/ })).toBeDisabled();
  expect(writes).toEqual([]);
});

test("staff-create-only access can start onboarding without loading or revealing staff directory metrics", async ({ page }) => {
  const staffListRequests: string[] = [];
  page.on("request", request => {
    const url = new URL(request.url());
    if (url.pathname === "/api/school/staff" && request.method() === "GET") {
      staffListRequests.push(request.method() + " " + url.pathname);
    }
  });

  await signInAndOpenStaff(page, [], undefined, ["users.create"]);

  await expect(page.getByRole("heading", { name: "Staff Management" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "People at a glance" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Add staff", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add staff", exact: true }).click();
  const createDialog = page.getByRole("dialog", { name: "Add employee" });
  await expect(createDialog).toBeVisible();
  await expect(createDialog.getByText("Staff Type & Identity", { exact: true })).toBeVisible();
  await expect(createDialog.getByRole("button", { name: "Continue", exact: true })).toBeEnabled();
  await expect(page.getByText("Total staff")).toHaveCount(0);
  await expect(page.locator(".staff-overview-cards")).toHaveCount(0);
  await expect(page.locator(".staff-readiness-list")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Staff Directory", exact: true })).toHaveCount(0);
  expect(staffListRequests).toEqual([]);
});

test("workload report explains missing current academic year without requesting a zero-year report", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "teacher.workload",
    employee_code: "EMP-WORK-01", full_name: "Workload Teacher", designation: "Mathematics Teacher",
    mobile_number: "9876543215", employment_status: "active", profile_complete: true,
    status: "active", must_change_password: false,
  }];
  let workloadRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "timetable.read"]);
  await page.route("**/api/school/timetable/current-year", route =>
    route.fulfill({ json: { academic_year: null } }),
  );
  await page.route("**/api/school/timetable/workload-report**", route => {
    workloadRequests += 1;
    return route.fulfill({ json: { workloads: [] } });
  });
  await page.locator(".staff-workspace-tabs").getByRole("button", { name: "Workload Report" }).click();
  await expect(page.getByRole("heading", { name: "School-wide Workload Report" })).toBeVisible();
  await expect(page.locator(".staff-workload-report-empty")).toContainText("No current academic year is configured");
  await expect(page.locator(".staff-workload-report-summary")).toHaveCount(0);
  await expect(page.locator(".staff-workload-report-filters")).toHaveCount(0);
  await expect.poll(() => workloadRequests).toBe(0);
  await page.getByRole("button", { name: "Refresh" }).click();
  await expect.poll(() => workloadRequests).toBe(0);
});


test.describe("staff ID card school-date formatting in non-school browser timezones", () => {
  test.use({ timezoneId: "America/Los_Angeles" });

  test("joining dates retain the school calendar date", async ({ page }) => {
    const rows:Array<Record<string,unknown>>=[
      {id:1,principal_id:501,school_id:10,username:"active.teacher",employee_code:"EMP-501",full_name:"Active Teacher",designation:"Mathematics Teacher",joining_date:"2024-04-01",employment_status:"active",profile_complete:true,status:"active",must_change_password:false},
    ];
    await signInAndOpenStaff(page,rows,undefined,["users.read"]);
    await page.getByRole("button",{name:"ID Cards",exact:true}).click();
    await page.getByRole("button",{name:"Select all eligible"}).click();
    const preview=page.locator(".staff-id-print-area");
    await expect(preview.getByText("1 Apr 2024",{exact:true})).toBeVisible();
  });
});

test("staff profile history never flashes the previous employee's audit events while the next employee loads", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 521, school_id: 10, username: "teacher.audit.first", employee_code: "EMP-AUD-01", full_name: "Audit First", designation: "Teacher", mobile_number: "9876543221", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
    { id: 2, principal_id: 522, school_id: 10, username: "teacher.audit.second", employee_code: "EMP-AUD-02", full_name: "Audit Second", designation: "Teacher", mobile_number: "9876543222", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  let auditRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "school.audit.read"]);
  await page.route("**/api/school/staff/521/audit-events**", async route => {
    auditRequests += 1;
    await route.fulfill({ json: { events: [
      { id: 1, action: "staff.profile.updated", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T10:00:00.000Z" },
      { id: 2, action: "staff.account.created", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T09:00:00.000Z" },
    ] } });
  });
  await page.route("**/api/school/staff/522/audit-events**", async route => {
    auditRequests += 1;
    await new Promise(resolve => setTimeout(resolve, 400));
    await route.fulfill({ json: { events: [
      { id: 3, action: "staff.leave.approved", resource_type: "staff_leave_request", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T11:00:00.000Z" },
    ] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Audit First" }).click();
  let dialog = page.getByRole("dialog", { name: "Audit First" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  const firstAuditReadCount = auditRequests;
  expect(firstAuditReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Audit Second" }).click();
  dialog = page.getByRole("dialog", { name: "Audit Second" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByRole("status")).toContainText("Loading staff activity");
  await expect(dialog.getByText("Profile updated")).toHaveCount(0);
  await expect(dialog.getByText("Account created")).toHaveCount(0);
  await expect(dialog.getByText("Leave approved")).toBeVisible();
});


test("staff history paginates older audit events without mixing employees", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 541, school_id: 10, username: "teacher.audit.pages", employee_code: "EMP-AUD-PAGES", full_name: "Audit Pages", designation: "Teacher", mobile_number: "9876543241", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  const requestedOffsets: string[] = [];
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "school.audit.read"]);
  await page.route("**/api/school/staff/541/audit-events**", async route => {
    const url = new URL(route.request().url());
    const offset = url.searchParams.get("offset") || "0";
    requestedOffsets.push(offset);
    const events = offset === "0"
      ? Array.from({ length: 50 }, (_, index) => ({
          id: index + 1, action: index === 0 ? "staff.profile.updated" : "staff.account.created",
          resource_type: "staff_user", outcome: "success", actor_username: "school.admin",
          created_at: "2026-11-22T10:00:00.000Z",
        }))
      : [{
          id: 51, action: "staff.leave.approved", resource_type: "staff_leave_request",
          outcome: "success", actor_username: "school.admin", created_at: "2026-11-22T11:00:00.000Z",
        }];
    await route.fulfill({ json: { events, limit: 50, offset: Number(offset), has_more: offset === "0" } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Audit Pages" }).click();
  const dialog = page.getByRole("dialog", { name: "Audit Pages" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  await expect(dialog.getByText("Page 1")).toBeVisible();
  await dialog.getByRole("button", { name: "Older events" }).click();
  await expect(dialog.getByText("Leave approved")).toBeVisible();
  await expect(dialog.getByText("Page 2")).toBeVisible();
  await expect.poll(() => requestedOffsets).toContain("50");
  await dialog.getByRole("button", { name: "Newer events" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  await expect(dialog.getByText("Page 1")).toBeVisible();
  await expect.poll(() => requestedOffsets).toContain("0");
});

test("closing and reopening the same staff profile reloads audit history instead of reusing cached events", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    { id: 1, principal_id: 531, school_id: 10, username: "teacher.audit.reopen", employee_code: "EMP-AUD-REOPEN", full_name: "Audit Reopen", designation: "Teacher", mobile_number: "9876543231", employment_status: "active", profile_complete: true, status: "active", must_change_password: false },
  ];
  let auditRequests = 0;
  await signInAndOpenStaff(page, rows, undefined, ["users.read", "school.audit.read"]);
  await page.route("**/api/school/staff/531/audit-events**", async route => {
    auditRequests += 1;
    await route.fulfill({ json: { events: [
      { id: auditRequests, action: "staff.profile.updated", resource_type: "staff_user", outcome: "success", actor_username: "school.admin", metadata: {}, created_at: "2026-11-22T10:00:00.000Z" },
    ] } });
  });

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await page.getByRole("button", { name: "Audit Reopen" }).click();
  let dialog = page.getByRole("dialog", { name: "Audit Reopen" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  const firstAuditReadCount = auditRequests;
  expect(firstAuditReadCount).toBeGreaterThan(0);
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Audit Reopen" }).click();
  dialog = page.getByRole("dialog", { name: "Audit Reopen" });
  await dialog.getByRole("button", { name: "History & Activity" }).click();
  await expect(dialog.getByText("Profile updated")).toBeVisible();
  await expect.poll(() => auditRequests).toBeGreaterThan(firstAuditReadCount);
});



test("staff leave report CSV exports every matching page", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.export.teacher",
    employee_code: "EMP-EXPORT-01", full_name: "Export Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  const requests = Array.from({ length: 205 }, (_, index) => ({
    id: index + 1, staff_principal_id: 501, employee_code: "EMP-EXPORT-01", staff_name: "Export Teacher",
    leave_type_id: 1, leave_type_name: "Casual Leave", start_date: schoolDateValue(), end_date: schoolDateValue(),
    duration_type: "full_day", reason: "Export request " + (index + 1), status: "approved",
    created_at: "2026-11-21T00:00:00.000Z",
  }));
  await signInAndOpenStaff(page, staffRows, undefined,
    ["users.read", "staff.leave.read"], undefined, requests);
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Leave requests" })).toBeVisible();
  const exportButton = page.getByRole("button", { name: "Export leave CSV" });
  await expect(exportButton).toBeEnabled();
  const downloadPromise = page.waitForEvent("download");
  await exportButton.click();
  const download = await downloadPromise;
  const path = await download.path();
  expect(path).toBeTruthy();
  const csv = await readFile(path!, "utf8");
  expect(csv.startsWith("\uFEFF")).toBe(true);
  const csvLines = csv.split(/\r?\n/);
  expect(csvLines).toHaveLength(206);
  expect(csvLines[205]).toContain("EMP-EXPORT-01");
  expect(csv).not.toContain("Export request 205");
});

test("staff leave report rejects changing totals instead of showing a partial export", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.integrity.teacher",
    employee_code: "EMP-INTEGRITY-01", full_name: "Integrity Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  const requests = Array.from({ length: 101 }, (_, index) => ({
    id: index + 1, staff_principal_id: 501, employee_code: "EMP-INTEGRITY-01", staff_name: "Integrity Teacher",
    leave_type_id: 1, leave_type_name: "Casual Leave", start_date: schoolDateValue(), end_date: schoolDateValue(),
    duration_type: "full_day", reason: "Integrity check " + (index + 1), status: "approved",
    created_at: "2026-11-21T00:00:00.000Z",
  }));
  await signInAndOpenStaff(page, staffRows, undefined, ["users.read", "staff.leave.read"], undefined, requests);
  await page.route("**/api/school/staff-leave/requests?**", async route => {
    const url = new URL(route.request().url());
    const pageNumber = Number(url.searchParams.get("page") ?? "1");
    const pageSize = Number(url.searchParams.get("page_size") ?? "100");
    const data = pageNumber === 1 ? requests.slice(0, pageSize) : requests.slice(pageSize);
    await route.fulfill({
      json: {
        data,
        pagination: { page: pageNumber, page_size: pageSize, total: pageNumber === 1 ? 101 : 102 },
      },
    });
  });
  await page.getByRole("button", { name: "Reports", exact: true }).click();
  await expect(page.getByRole("alert").filter({ hasText: "Leave report could not be loaded." })).toBeVisible();
  await expect(page.getByRole("button", { name: "Export leave CSV" })).toHaveCount(0);
});

test("staff leave requests use bounded pages and show the matching total", async ({ page }) => {
  const staffRows: Array<Record<string, unknown>> = [{
    id: 1, principal_id: 501, school_id: 10, username: "leave.page.teacher",
    employee_code: "EMP-PAGE-01", full_name: "Pagination Teacher", designation: "Teacher",
    employment_status: "active", profile_complete: true, status: "active", must_change_password: false,
  }];
  const requests = Array.from({ length: 55 }, (_, index) => ({
    id: index + 1, staff_principal_id: 501, employee_code: "EMP-PAGE-01", staff_name: "Pagination Teacher",
    leave_type_id: 1, leave_type_name: "Casual Leave", start_date: schoolDateValue(), end_date: schoolDateValue(),
    duration_type: "full_day", reason: "Leave request " + (index + 1), status: "pending",
    created_at: "2026-11-21T00:00:00.000Z",
  }));
  await signInAndOpenStaff(page, staffRows, undefined,
    ["users.read", "staff.leave.read", "staff.leave.manage_requests"], undefined, requests);
  await page.getByRole("button", { name: "Leave & Holidays" }).click();
  await expect(page.getByText("Leave request 1", { exact: true })).toBeVisible();
  await expect(page.getByText("Leave request 51", { exact: true })).toHaveCount(0);
  await expect(page.getByText("Showing 1–50 of 55")).toBeVisible();
  await page.getByRole("button", { name: "Next" }).click();
  await expect(page.getByText("Leave request 51", { exact: true })).toBeVisible();
  await expect(page.getByText("Showing 51–55 of 55")).toBeVisible();
  for(let index=51;index<=55;index++){
    const requestRow=page.locator("article.staff-leave-request").filter({hasText:"Leave request "+index});
    await requestRow.getByRole("button",{name:"Cancel request"}).click();
    const confirmation=page.getByRole("dialog",{name:"Cancel leave request?"});
    await confirmation.getByRole("button",{name:"Cancel request",exact:true}).click();
    await expect(requestRow).toHaveCount(0);
  }
  await expect(page.getByText("Showing 1–50 of 50")).toBeVisible();
  await expect(page.getByText("Leave request 1",{exact:true})).toBeVisible();
});


test("protected School Admin staff records expose no ordinary account lifecycle actions", async ({ page }) => {
  const rows: Array<Record<string, unknown>> = [
    {
      id: 1, principal_id: 941, school_id: 10, username: "protected.admin",
      employee_code: "ADM-941", full_name: "Protected Admin", designation: "School Admin",
      mobile_number: "9876543210", employment_status: "active", profile_complete: true,
      protected_school_admin: true, status: "active", must_change_password: false,
    },
    {
      id: 2, principal_id: 942, school_id: 10, username: "protected.disabled",
      employee_code: "ADM-942", full_name: "Disabled Protected Admin", designation: "School Admin",
      mobile_number: "9876543212", employment_status: "active", profile_complete: true,
      protected_school_admin: true, status: "disabled", must_change_password: true,
    },
  ];
  await signInAndOpenStaff(page, rows, undefined, [
    "users.read", "users.update", "users.disable", "users.password_reset",
    "roles.read", "roles.assign", "permissions.override",
  ]);
  await page.getByRole("button", { name: /Staff Directory/ }).click();

  const activeRow = page.locator("tr").filter({ hasText: "protected.admin" });
  const disabledRow = page.locator("tr").filter({ hasText: "protected.disabled" });
  await expect(activeRow).toBeVisible();
  await expect(disabledRow).toBeVisible();
  for (const [row, username] of [[activeRow, "protected.admin"], [disabledRow, "protected.disabled"]] as const) {
    await expect(row.getByRole("button", { name: "Edit employee profile for " + username })).toHaveCount(0);
    await expect(row.getByRole("button", { name: "Reset password for " + username })).toHaveCount(0);
    await expect(row.getByRole("button", { name: "Disable " + username })).toHaveCount(0);
    await expect(row.getByRole("button", { name: "Enable " + username })).toHaveCount(0);
  }

  await activeRow.getByRole("button", { name: "Protected Admin", exact: true }).click();
  const profile = page.getByRole("dialog", { name: "Protected Admin" });
  await expect(profile.getByRole("button", { name: "Edit employee details" })).toHaveCount(0);
  await expect(profile.getByRole("button", { name: "Edit personal details" })).toHaveCount(0);
});

test("staff directory retries after a transient server error and accepts a subsequent search", async ({ page }) => {
  let attempts = 0;
  await signInAndOpenStaff(page, []);
  await page.route(
    url => url.pathname === "/api/school/staff" && url.searchParams.has("limit"),
    async route => {
      attempts += 1;
      if (attempts <= 2) {
        await route.fulfill({ status: 503, json: { message: "Staff directory is temporarily unavailable." } });
        return;
      }
      const url = new URL(route.request().url());
      const search = url.searchParams.get("search") ?? "";
      const staff = search === "recovered"
        ? [{ id: 1, principal_id: 901, school_id: 10, username: "recovered.staff", employee_code: "REC-901", full_name: "Recovered Staff", designation: "Office Staff", mobile_number: "9876543210", employment_status: "active", profile_complete: true, status: "active", must_change_password: false }]
        : [];
      await route.fulfill({ json: { staff, total: staff.length, limit: Number(url.searchParams.get("limit")), offset: Number(url.searchParams.get("offset") ?? 0) } });
    },
  );

  await page.getByRole("button", { name: /Staff Directory/ }).click();
  await expect(page.getByText("Staff directory is temporarily unavailable.")).toBeVisible();
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByText("No staff accounts yet. Add the first staff account to get started.")).toBeVisible();
  expect(attempts).toBe(3);

  await page.getByRole("textbox", { name: "Search staff by name, code, designation, mobile or username" }).fill("recovered");
  await expect(page.getByRole("button", { name: "Recovered Staff" })).toBeVisible();
  expect(attempts).toBe(4);
});
