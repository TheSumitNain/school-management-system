import { api } from "../../lib/api";

export type StaffRecord = {
  id: number;
  principal_id: number;
  school_id: number;
  username: string;
  status: string;
  must_change_password: boolean;
  created_at?: string;
  updated_at?: string;
};
export type StaffListResponse = { staff: StaffRecord[] };

export function getSchoolStaff(signal?: AbortSignal) {
  return api<StaffListResponse>("/school/staff", { signal });
}
export function createSchoolStaff(input: { username: string; initial_password: string }) {
  return api<{ staff: StaffRecord }>("/school/staff", { method: "POST", body: JSON.stringify(input) });
}
export function updateSchoolStaff(principalID: number, input: { username: string }) {
  return api<{ staff: StaffRecord; message: string }>(`/school/staff/${principalID}`, { method: "PATCH", body: JSON.stringify(input) });
}
export function disableSchoolStaff(principalID: number) {
  return api<{ message: string }>(`/school/staff/${principalID}`, { method: "DELETE" });
}
export function enableSchoolStaff(principalID: number, temporary_password: string) {
  return api<{ message: string }>(`/school/staff/${principalID}/enable`, { method: "POST", body: JSON.stringify({ temporary_password }) });
}
export function resetSchoolStaffPassword(principalID: number, new_password: string) {
  return api<{ message: string }>(`/school/staff/${principalID}/password/reset`, { method: "POST", body: JSON.stringify({ new_password }) });
}

export type SchoolRole = {
  id: number;
  role_key: string;
  name: string;
  is_system: boolean;
  created_at: string;
};
export type StaffRoleAssignment = {
  role_id: number;
  role_key: string;
  name: string;
  is_system: boolean;
  assigned_at: string;
};
export type SchoolRolesResponse = { roles: SchoolRole[] };
export type StaffRoleAssignmentsResponse = { roles: StaffRoleAssignment[] };

export function getSchoolRoles(signal?: AbortSignal) {
  return api<SchoolRolesResponse>("/school/roles", { signal });
}
export function getStaffRoleAssignments(principalID: number, signal?: AbortSignal) {
  return api<StaffRoleAssignmentsResponse>(`/school/staff/${principalID}/roles`, { signal });
}
export function assignStaffRole(principalID: number, roleID: number) {
  return api<{ message: string }>(`/school/staff/${principalID}/roles/${roleID}`, { method: "POST" });
}
export function revokeStaffRole(principalID: number, roleID: number) {
  return api<{ message: string }>(`/school/staff/${principalID}/roles/${roleID}`, { method: "DELETE" });
}

export type SchoolPermission = {
  permission_key: string;
  description: string;
  is_delegable: boolean;
};
export type SchoolPermissionDelegation = {
  id: number;
  school_id: number;
  permission_key: string;
  can_assign_to_roles: boolean;
  can_override_users: boolean;
  granted_by: number;
  created_at: string;
  revoked_at?: string;
};
export type StaffPermissionOverride = {
  id: number;
  principal_id: number;
  permission_key: string;
  effect: "allow" | "deny";
  reason: string;
  granted_by: number;
  delegation_id: number;
  created_at: string;
};
export type CustomRolePermission = {
  permission_key: string;
  description: string;
  delegation_id: number;
  delegation_active: boolean;
  permission_active: boolean;
  created_at: string;
};
export type CustomRolePermissionsResponse = { permissions: CustomRolePermission[] };

export type SchoolPermissionsResponse = { permissions: SchoolPermission[] };
export type SchoolPermissionDelegationsResponse = { delegations: SchoolPermissionDelegation[] };
export type StaffPermissionOverridesResponse = { overrides: StaffPermissionOverride[] };

export function createSchoolRole(name: string) {
  return api<{ role: SchoolRole }>("/school/roles", { method: "POST", body: JSON.stringify({ name }) });
}
export function getCustomRolePermissions(roleID: number, signal?: AbortSignal) {
  return api<CustomRolePermissionsResponse>(`/school/roles/${roleID}/permissions`, { signal });
}
export function grantCustomRolePermission(roleID: number, input: { permission_key: string; delegation_id: number }) {
  return api<{ message: string }>(`/school/roles/${roleID}/permissions`, { method: "POST", body: JSON.stringify(input) });
}
export function revokeCustomRolePermission(roleID: number, permissionKey: string) {
  return api<{ message: string }>(`/school/roles/${roleID}/permissions/${encodeURIComponent(permissionKey)}`, { method: "DELETE" });
}

export function getSchoolPermissions(signal?: AbortSignal) {
  return api<SchoolPermissionsResponse>("/school/permissions", { signal });
}
export function getSchoolPermissionDelegations(signal?: AbortSignal) {
  return api<SchoolPermissionDelegationsResponse>("/school/permission-delegations", { signal });
}
export function getStaffPermissionOverrides(principalID: number, signal?: AbortSignal) {
  return api<StaffPermissionOverridesResponse>(`/school/staff/${principalID}/permission-overrides`, { signal });
}
export function setStaffPermissionOverride(
  principalID: number,
  input: { permission_key: string; effect: "allow" | "deny"; reason: string; delegation_id: number },
) {
  return api<{ message: string }>(`/school/staff/${principalID}/permission-overrides`, { method: "POST", body: JSON.stringify(input) });
}
export function deleteStaffPermissionOverride(principalID: number, permissionKey: string) {
  return api<{ message: string }>(`/school/staff/${principalID}/permission-overrides/${encodeURIComponent(permissionKey)}`, { method: "DELETE" });
}


export type StudentProfile = {
  id: number;
  school_id: number;
  admission_number: string;
  full_name: string;
  status: "active" | "inactive" | "archived" | string;
  created_at: string;
  updated_at: string;
};
export type StudentListResponse = {
  students: StudentProfile[];
  pagination: { limit: number; offset: number; total: number };
};
export type StudentEnrollment = {
  id: number;
  school_id?: number;
  student_id?: number;
  student_name?: string;
  academic_year_id?: number;
  class_id?: number;
  section_id?: number;
  roll_number?: string | null;
  class_name_snapshot: string;
  section_name_snapshot: string;
  status: string;
  enrolled_at: string;
  ended_at?: string | null;
};
export type StudentEnrollmentListResponse = { enrollments: StudentEnrollment[] };
export type StudentAcademicYear = { id: number; name: string; status: string; is_current: boolean };
export type StudentSection = { id: number; academic_year_id: number; class_id: number; name: string; is_active: boolean };
export type StudentClass = { id: number; name: string; is_active: boolean };

export function getSchoolStudents(filters: { q?: string; status?: string; limit?: number; offset?: number }, signal?: AbortSignal) {
  const params = new URLSearchParams();
  if (filters.q) params.set("q", filters.q);
  if (filters.status) params.set("status", filters.status);
  params.set("limit", String(filters.limit ?? 20));
  params.set("offset", String(filters.offset ?? 0));
  return api<StudentListResponse>("/school/students?" + params.toString(), { signal });
}
export function createSchoolStudent(input: { admission_number: string; full_name: string }) {
  return api<{ student: StudentProfile }>("/school/students", { method: "POST", body: JSON.stringify(input) });
}
export function updateSchoolStudent(studentID: number, input: { full_name: string }) {
  return api<{ student: StudentProfile }>(`/school/students/${studentID}`, { method: "PATCH", body: JSON.stringify(input) });
}
export function archiveSchoolStudent(studentID: number) {
  return api<{ student: StudentProfile }>(`/school/students/${studentID}/archive`, { method: "POST" });
}
export function getSchoolStudentEnrollments(academicYearID = 0, signal?: AbortSignal) {
  const query = academicYearID > 0 ? `?academic_year_id=${academicYearID}` : "";
  return api<StudentEnrollmentListResponse>("/school/enrollments" + query, { signal });
}
export function getSchoolAcademicYears(signal?: AbortSignal) {
  return api<{ academic_years: StudentAcademicYear[] }>("/school/academic-years", { signal });
}
export function getSchoolSections(academicYearID: number, signal?: AbortSignal) {
  const query = academicYearID > 0 ? `?academic_year_id=${academicYearID}` : "";
  return api<{ sections: StudentSection[] }>("/school/sections" + query, { signal });
}
export function createSchoolStudentEnrollment(input: {
  student_id: number; academic_year_id: number; section_id: number; roll_number?: string | null; enrolled_at: string;
}) {
  return api<{ enrollment: StudentEnrollment }>("/school/enrollments", { method: "POST", body: JSON.stringify(input) });
}


export type AttendanceRecord = {
  id: number;
  school_id: number;
  enrollment_id: number;
  student_id: number;
  admission_number: string;
  student_name: string;
  academic_year_id: number;
  class_name: string;
  section_name: string;
  roll_number?: string | null;
  attendance_date: string;
  status: "present" | "absent" | "late" | "excused" | string;
  remarks?: string | null;
  marked_by_principal_id: number;
  updated_by_principal_id: number;
  created_at: string;
  updated_at: string;
};
export type AttendanceListResponse = { status: string; data: AttendanceRecord[] };

export function getSchoolAttendance(filters: { academicYearID: number; sectionID: number; date: string }, signal?: AbortSignal) {
  const params = new URLSearchParams({
    academic_year_id: String(filters.academicYearID),
    section_id: String(filters.sectionID),
    date: filters.date,
  });
  return api<AttendanceListResponse>("/school/attendance?" + params.toString(), { signal });
}
export type AttendanceMutationResponse = { status: string; data: AttendanceRecord };
export function markSchoolAttendance(input: { enrollment_id: number; attendance_date: string; status: string; remarks?: string | null }) {
  return api<AttendanceMutationResponse>("/school/attendance", { method: "POST", body: JSON.stringify(input) });
}
export function updateSchoolAttendance(attendanceID: number, input: { status: string; remarks?: string | null }) {
  return api<AttendanceMutationResponse>(`/school/attendance/${attendanceID}`, { method: "PATCH", body: JSON.stringify(input) });
}


export type EligibleSchoolTeacher = { principal_id: number; username: string };
export type TeacherSubjectAssignment = { id: number; academic_year_id: number; section_id: number; section_name: string; class_name: string; subject_id: number; subject_name: string; teacher_principal_id: number; teacher_username: string; revoked_at?: string | null };
export type ClassInchargeAssignment = { id: number; academic_year_id: number; section_id: number; section_name: string; class_name: string; teacher_principal_id: number; teacher_username: string; revoked_at?: string | null };
export function getEligibleSchoolTeachers(signal?: AbortSignal) { return api<{ teachers: EligibleSchoolTeacher[] }>("/school/eligible-teachers", { signal }); }
export function getTeacherSubjectAssignments(filters: { academicYearID?: number; sectionID?: number } = {}, signal?: AbortSignal) { const params = new URLSearchParams(); if (filters.academicYearID) params.set("academic_year_id", String(filters.academicYearID)); if (filters.sectionID) params.set("section_id", String(filters.sectionID)); const query = params.toString(); return api<{ assignments: TeacherSubjectAssignment[] }>("/school/teacher-assignments" + (query ? "?" + query : ""), { signal }); }
export function createTeacherSubjectAssignment(input: { academic_year_id: number; section_id: number; subject_id: number; teacher_principal_id: number }) { return api<{ assignment: TeacherSubjectAssignment }>("/school/teacher-assignments", { method: "POST", body: JSON.stringify(input) }); }
export function revokeTeacherSubjectAssignment(assignmentID: number) { return api<{ message: string }>("/school/teacher-assignments/" + assignmentID, { method: "DELETE" }); }
export function getClassInchargeAssignments(academicYearID?: number, signal?: AbortSignal) { const query = academicYearID ? "?academic_year_id=" + academicYearID : ""; return api<{ assignments: ClassInchargeAssignment[] }>("/school/class-incharge-assignments" + query, { signal }); }
export function createClassInchargeAssignment(input: { academic_year_id: number; section_id: number; teacher_principal_id: number }) { return api<{ assignment: ClassInchargeAssignment }>("/school/class-incharge-assignments", { method: "POST", body: JSON.stringify(input) }); }
export function revokeClassInchargeAssignment(assignmentID: number) { return api<{ message: string }>("/school/class-incharge-assignments/" + assignmentID, { method: "DELETE" }); }


export type TimetablePeriod = {
  id: number; school_id: number; academic_year_id: number; period_number: number;
  name: string; starts_at: string; ends_at: string; is_break: boolean;
};
export type TimetableEntry = {
  id: number; school_id: number; academic_year_id: number; section_id: number;
  section_name: string; class_name: string; subject_id: number; subject_name: string;
  teacher_principal_id: number; teacher_username: string; teacher_subject_assignment_id: number;
  period_id: number; period_number: number; period_name: string; starts_at: string; ends_at: string;
  day_of_week: number; room?: string | null;
};
export function getTimetablePeriods(academicYearID: number, signal?: AbortSignal) {
  return api<{ periods: TimetablePeriod[] }>(`/school/timetable/periods?academic_year_id=${academicYearID}`, { signal });
}
export function saveTimetablePeriod(input: {
  academic_year_id: number; period_number: number; name: string; starts_at: string; ends_at: string; is_break: boolean;
}) {
  return api<{ period: TimetablePeriod }>("/school/timetable/periods", { method: "POST", body: JSON.stringify(input) });
}
export function getTimetableEntries(academicYearID: number, sectionID = 0, signal?: AbortSignal) {
  const params = new URLSearchParams({ academic_year_id: String(academicYearID) });
  if (sectionID > 0) params.set("section_id", String(sectionID));
  return api<{ entries: TimetableEntry[] }>(`/school/timetable/entries?${params.toString()}`, { signal });
}
export function createTimetableEntry(input: {
  academic_year_id: number; section_id: number; subject_id: number; teacher_principal_id: number;
  teacher_subject_assignment_id: number; period_id: number; day_of_week: number; room?: string | null;
}) {
  return api<{ entry: TimetableEntry }>("/school/timetable/entries", { method: "POST", body: JSON.stringify(input) });
}
export function deleteTimetableEntry(entryID: number, academicYearID: number) {
  return api<{ message: string }>(`/school/timetable/entries/${entryID}?academic_year_id=${academicYearID}`, { method: "DELETE" });
}
