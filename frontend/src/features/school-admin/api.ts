import { api } from "../../lib/api";

export type StaffRecord = {
  id: number;
  principal_id: number;
  school_id: number;
  username: string;
  employee_code?: string;
  full_name?: string;
  designation?: string;
  mobile_number?: string;
  email?: string;
  joining_date?: string | null;
  employment_status?: "active" | "inactive" | string;
  profile_complete?: boolean;
  status: string;
  must_change_password: boolean;
  created_at?: string;
  updated_at?: string;
};
export type StaffListResponse = { staff: StaffRecord[] };

export function getSchoolStaff(signal?: AbortSignal) {
  return api<StaffListResponse>("/school/staff", { signal });
}
export function createSchoolStaff(input: { username: string; initial_password: string } & { employee_code: string; full_name: string; designation: string; mobile_number: string; email?: string; joining_date?: string; employment_status?: "active" | "inactive" }) {
  return api<{ staff: StaffRecord }>("/school/staff", { method: "POST", body: JSON.stringify(input) });
}
export function updateSchoolStaff(principalID: number, input: { username: string } & { employee_code: string; full_name: string; designation: string; mobile_number: string; email?: string; joining_date?: string; employment_status?: "active" | "inactive" }) {
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


export type StudentBasicDetails = {
  id: number;
  school_id: number;
  student_id: number;
  date_of_birth?: string | null;
  place_of_birth?: string | null;
  gender?: "female" | "male" | "other" | "not_specified" | null;
  nationality?: string | null;
  religion?: string | null;
  social_category?: "general" | "obc" | "sc" | "st" | "other" | "not_specified" | null;
  ews_status?: boolean | null;
  school_registration_number?: string | null;
  created_at?: string;
  updated_at?: string;
};
export type StudentBasicDetailsResponse = { basic_details: StudentBasicDetails };

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
  student_status?: string;
  student_name?: string;
  academic_year_id?: number;
  class_id?: number;
  section_id?: number;
  roll_number?: string | null;
  class_name_snapshot: string;
  section_name_snapshot: string;
  admission_number?: string | null;
  admission_date?: string | null;
  status: string;
  enrolled_at: string;
  ended_at?: string | null;
};
export type StudentEnrollmentListResponse = { enrollments: StudentEnrollment[]; pagination?: { limit: number; offset: number; total: number } };
export type CurriculumOptionalGroup = { id: number; name: string; min_selections: number; max_selections: number };
export type CurriculumSubject = {
  id: number; subject_id: number; name: string; code?: string | null;
  optional_group_id?: number | null; optional_group_name?: string | null; master_active?: boolean; is_compulsory: boolean; is_optional: boolean;
};
export type ClassCurriculum = {
  id: number; school_id: number; academic_year_id: number; class_id: number; section_id?: number | null;
  version_number: number; status: "active" | "retired" | string; created_at: string;
  optional_groups: CurriculumOptionalGroup[]; subjects: CurriculumSubject[];
};
export type StudentEnrollmentCurriculum = {
  enrollment_id: number; student_id: number; student_status: string; enrollment_status: string;
  academic_year_id: number; class_id: number; section_id: number; curriculum?: ClassCurriculum | null; available_curriculum?: ClassCurriculum | null; assigned: boolean;
  selected_optional_subject_ids: number[];
};
export type StudentAcademicYear = { id: number; name: string; starts_on: string; ends_on: string; status: string; is_current: boolean };
export type StudentSection = { id: number; academic_year_id: number; class_id: number; class_name: string; name: string; is_active: boolean };
export type StudentClass = { id: number; name: string; is_active: boolean };

export function getSchoolStudents(filters: { q?: string; status?: string; academicYearID?: number; limit?: number; offset?: number }, signal?: AbortSignal) {
  const params = new URLSearchParams();
  if (filters.q) params.set("q", filters.q);
  if (filters.status) params.set("status", filters.status);
  if (filters.academicYearID) params.set("academic_year_id", String(filters.academicYearID));
  params.set("limit", String(filters.limit ?? 20));
  params.set("offset", String(filters.offset ?? 0));
  return api<StudentListResponse>("/school/students?" + params.toString(), { signal });
}
export function createSchoolStudentAdmission(input: {
  full_name: string; admission_number: string; academic_year_id: number; section_id: number;
  roll_number?: string | null; admission_date: string;
}) {
  return api<{ student: StudentProfile; enrollment: StudentEnrollment }>("/school/students", {
    method: "POST", body: JSON.stringify(input),
  });
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
export function getSchoolStudentEnrollmentDirectory(academicYearID = 0, limit = 20, offset = 0, signal?: AbortSignal) {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  if (academicYearID > 0) params.set("academic_year_id", String(academicYearID));
  return api<StudentEnrollmentListResponse>(`/school/enrollment-directory?${params.toString()}`, { signal });
}
export function getSchoolStudentEnrollmentHistory(studentID: number, signal?: AbortSignal) {
  return api<StudentEnrollmentListResponse>(`/school/students/${studentID}/enrollments`, { signal });
}
export function getSchoolClassCurricula(academicYearID: number, classID: number, sectionID: number, signal?: AbortSignal) {
  const params = new URLSearchParams({ academic_year_id: String(academicYearID), class_id: String(classID), section_id: String(sectionID) });
  return api<{ curricula: ClassCurriculum[] }>(`/school/class-curricula?${params.toString()}`, { signal });
}
export type CurriculumRemovalImpact = { subject_id: number; subject_name: string; affected_students: number; students: Array<{ student_id: number; admission_number: string; full_name: string }> };
export type CurriculumSaveInput = {
  academic_year_id: number; class_id: number; section_id: number;
  optional_groups: Array<{ name: string; min_selections: number; max_selections: number }>;
  subjects: Array<{ subject_id: number; is_optional?: boolean; optional_group?: string }>;
  removal_confirmations?: Array<{ subject_id: number; affected_students: number; student_ids: number[] }>;
};
export function previewSchoolClassCurriculumRemoval(input: {
  academic_year_id: number; class_id: number; section_id: number; subject_ids: number[];
}) {
  return api<{ impacts: CurriculumRemovalImpact[] }>("/school/class-curricula/removal-impact", { method: "POST", body: JSON.stringify(input) });
}
export function saveSchoolClassCurriculum(input: CurriculumSaveInput) {
  return api<{ curriculum: ClassCurriculum }>("/school/class-curricula", { method: "POST", body: JSON.stringify(input) });
}
export function getSchoolStudentEnrollmentCurriculum(enrollmentID: number, signal?: AbortSignal) {
  return api<{ enrollment_curriculum: StudentEnrollmentCurriculum }>(`/school/enrollments/${enrollmentID}/curriculum`, { signal });
}
export function saveSchoolStudentEnrollmentSubjects(enrollmentID: number, input: {
  curriculum_id: number; selected_optional_subject_ids: number[];
}) {
  return api<{ enrollment_curriculum: StudentEnrollmentCurriculum }>(`/school/enrollments/${enrollmentID}/subjects`, { method: "PUT", body: JSON.stringify(input) });
}
export function getSchoolAcademicYears(signal?: AbortSignal) {
  return api<{ academic_years: StudentAcademicYear[] }>("/school/academic-years", { signal });
}
export function getSchoolSections(academicYearID: number, signal?: AbortSignal) {
  const query = academicYearID > 0 ? `?academic_year_id=${academicYearID}` : "";
  return api<{ sections: StudentSection[] }>("/school/sections" + query, { signal });
}
export function createSchoolStudentEnrollment(input: {
  student_id: number; academic_year_id: number; section_id: number; roll_number?: string | null;
  admission_number: string; admission_date: string; enrolled_at: string;
}) {
  return api<{ enrollment: StudentEnrollment }>("/school/enrollments", { method: "POST", body: JSON.stringify(input) });
}
export function promoteSchoolStudentEnrollment(sourceEnrollmentID: number, input: {
  academic_year_id: number; section_id: number; roll_number?: string | null;
  admission_number: string; admission_date: string; enrolled_at: string;
}) {
  return api<{ enrollment: StudentEnrollment }>(`/school/enrollments/${sourceEnrollmentID}/promote`, {
    method: "POST", body: JSON.stringify(input),
  });
}

export type StudentLocation = {
  id: number; school_id?: number | null; parent_id?: number | null; country_code: string;
  level: "state" | "district" | "subdistrict" | "locality" | string; name: string;
  locality_type?: "village" | "city" | "town" | "other" | null; is_active: boolean;
};
export type StudentAddress = {
  id: number; school_id: number; student_id: number; address_type: "current" | "permanent";
  address_line_1?: string | null; address_line_2?: string | null; area_mohalla?: string | null;
  landmark?: string | null; postal_code?: string | null; location_id?: number | null;
  location_name?: string | null; location_level?: string | null; locality_type?: string | null;
  state_location_id?: number | null; district_location_id?: number | null; subdistrict_location_id?: number | null;
  state_location_name?: string | null; district_location_name?: string | null; subdistrict_location_name?: string | null;
  same_as_current: boolean;
};
export function getSchoolStudentLocations(filters: { parentID?: number; level?: string; q?: string } = {}, signal?: AbortSignal) {
  const params = new URLSearchParams();
  if (filters.parentID) params.set("parent_id", String(filters.parentID));
  if (filters.level) params.set("level", filters.level);
  if (filters.q) params.set("q", filters.q);
  const query = params.toString();
  return api<{ locations: StudentLocation[] }>("/school/locations" + (query ? "?" + query : ""), { signal });
}
export function createSchoolStudentLocation(input: {
  parent_id?: number | null; country_code?: string; level: string; name: string; locality_type?: string | null;
}) {
  return api<{ location: StudentLocation }>("/school/locations", { method: "POST", body: JSON.stringify(input) });
}
export function deactivateSchoolStudentLocation(locationID: number) {
  return api<{ location: StudentLocation }>(`/school/locations/${locationID}/deactivate`, { method: "POST" });
}
export function getSchoolStudentAddresses(studentID: number, signal?: AbortSignal) {
  return api<{ addresses: StudentAddress[] }>(`/school/students/${studentID}/addresses`, { signal });
}
export function saveSchoolStudentAddress(studentID: number, addressType: "current" | "permanent", input: {
  address_line_1?: string | null; address_line_2?: string | null; area_mohalla?: string | null;
  landmark?: string | null; postal_code?: string | null; location_id?: number | null; same_as_current?: boolean;
}) {
  return api<{ address: StudentAddress }>(`/school/students/${studentID}/addresses/${addressType}`, {
    method: "PUT", body: JSON.stringify(input),
  });
}

export type StudentContactDetails = {
  id: number; school_id: number; student_id: number; primary_phone?: string | null;
  alternate_phone?: string | null; email_address?: string | null;
};
export function getSchoolStudentBasicDetails(studentID: number, signal?: AbortSignal) {
  return api<StudentBasicDetailsResponse>(`/school/students/${studentID}/basic-details`, { signal });
}
export function saveSchoolStudentBasicDetails(studentID: number, input: {
  date_of_birth?: string | null; place_of_birth?: string | null; gender?: string | null;
  nationality?: string | null; religion?: string | null; social_category?: string | null;
  ews_status?: boolean | null; school_registration_number?: string | null;
}) {
  return api<StudentBasicDetailsResponse>(`/school/students/${studentID}/basic-details`, { method: "PUT", body: JSON.stringify(input) });
}
export function getSchoolStudentContactDetails(studentID: number, signal?: AbortSignal) {
  return api<{ contact_details: StudentContactDetails }>(`/school/students/${studentID}/contact-details`, { signal });
}
export function saveSchoolStudentContactDetails(studentID: number, input: {
  primary_phone?: string | null; alternate_phone?: string | null; email_address?: string | null;
}) {
  return api<{ contact_details: StudentContactDetails }>(`/school/students/${studentID}/contact-details`, {
    method: "PUT", body: JSON.stringify(input),
  });
}

export type StudentMedicalDetailsVersion = {
  id: number; version_number: number; blood_group?: string | null; allergies?: string | null;
  known_conditions?: string | null; ongoing_medications?: string | null;
  emergency_medical_notes?: string | null; changed_by_principal_id: number; changed_at: string;
};
export type StudentMedicalDetails = {
  id: number; school_id: number; student_id: number; version_number: number;
  blood_group?: string | null; allergies?: string | null; known_conditions?: string | null;
  ongoing_medications?: string | null; emergency_medical_notes?: string | null;
  history: StudentMedicalDetailsVersion[];
};
export type StudentPhysicalMeasurement = {
  id: number; height_cm?: number | null; weight_kg?: number | null; measured_on: string;
  vision_notes?: string | null; physical_notes?: string | null;
  recorded_by_principal_id: number; created_at: string;
};
export type StudentPhysicalDetails = {
  school_id: number; student_id: number; id: number; height_cm?: number | null;
  weight_kg?: number | null; measured_on?: string | null; vision_notes?: string | null;
  physical_notes?: string | null; recorded_by_principal_id?: number; created_at?: string;
  history: StudentPhysicalMeasurement[];
};
export function getSchoolStudentMedicalDetails(studentID: number, signal?: AbortSignal) {
  return api<{ medical_details: StudentMedicalDetails }>(`/school/students/${studentID}/medical-details`, { signal });
}
export function saveSchoolStudentMedicalDetails(studentID: number, input: {
  blood_group?: string | null; allergies?: string | null; known_conditions?: string | null;
  ongoing_medications?: string | null; emergency_medical_notes?: string | null;
}) {
  return api<{ medical_details: StudentMedicalDetails }>(`/school/students/${studentID}/medical-details`, {
    method: "PUT", body: JSON.stringify(input),
  });
}
export function getSchoolStudentPhysicalDetails(studentID: number, signal?: AbortSignal) {
  return api<{ physical_details: StudentPhysicalDetails }>(`/school/students/${studentID}/physical-details`, { signal });
}
export function saveSchoolStudentPhysicalDetails(studentID: number, input: {
  height_cm?: number | null; weight_kg?: number | null; measured_on: string;
  vision_notes?: string | null; physical_notes?: string | null;
}) {
  return api<{ physical_details: StudentPhysicalDetails }>(`/school/students/${studentID}/physical-details`, {
    method: "PUT", body: JSON.stringify(input),
  });
}

export type SchoolGuardian = {
  id: number; school_id: number; full_name: string; mobile_number?: string | null;
  email_address?: string | null; alternate_mobile_number?: string | null; occupation?: string | null;
  education?: string | null; workplace_name?: string | null; status: "active" | "inactive" | "archived";
};
export type StudentGuardian = {
  id: number; student_id: number; guardian_id: number; guardian_name: string; guardian_status: "active" | "inactive" | "archived";
  mobile_number?: string | null; email_address?: string | null; alternate_mobile_number?: string | null;
  occupation?: string | null; education?: string | null; workplace_name?: string | null; relationship_type: string;
  is_primary_contact: boolean; is_emergency_contact: boolean; created_at: string; revoked_at?: string | null; revocation_reason?: string | null;
};
export type StudentRelatedStudent = {
  student_id: number; full_name: string; admission_number?: string | null;
  class_name?: string | null; section_name?: string | null; academic_year_name?: string | null;
  shared_guardian_names: string[];
};
export function getSchoolGuardians(filters: { q?: string; status?: string } = {}, signal?: AbortSignal) {
  const params = new URLSearchParams({ limit: "50", offset: "0" });
  if (filters.q) params.set("q", filters.q);
  if (filters.status) params.set("status", filters.status);
  return api<{ guardians: SchoolGuardian[]; total: number }>("/school/guardians?" + params.toString(), { signal });
}
export function getSchoolStudentGuardians(studentID: number, includeHistory = false, signal?: AbortSignal) {
  const params = includeHistory ? "?include_history=true" : "";
  return api<{ student_guardians: StudentGuardian[] }>(`/school/students/${studentID}/guardians${params}`, { signal });
}
export function getSchoolStudentRelatedStudents(studentID: number, signal?: AbortSignal) {
  return api<{ related_students: StudentRelatedStudent[] }>(`/school/students/${studentID}/related-students`, { signal });
}
export function createSchoolGuardian(input: {
  full_name: string; mobile_number?: string | null; email_address?: string | null; alternate_mobile_number?: string | null;
  occupation?: string | null; education?: string | null; workplace_name?: string | null; status?: string;
}) {
  return api<{ guardian: SchoolGuardian }>("/school/guardians", { method: "POST", body: JSON.stringify(input) });
}
export function updateSchoolGuardian(guardianID: number, input: {
  full_name: string; mobile_number?: string | null; email_address?: string | null; alternate_mobile_number?: string | null;
  occupation?: string | null; education?: string | null; workplace_name?: string | null; status: string;
}) {
  return api<{ guardian: SchoolGuardian }>(`/school/guardians/${guardianID}`, { method: "PATCH", body: JSON.stringify(input) });
}
export function linkSchoolStudentGuardian(studentID: number, input: {
  guardian_id: number; relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean;
}) {
  return api<{ student_guardian: StudentGuardian }>(`/school/students/${studentID}/guardians`, {
    method: "POST", body: JSON.stringify(input),
  });
}
export function updateSchoolStudentGuardianLink(linkID: number, input: {
  relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean;
}) {
  return api<{ student_guardian: StudentGuardian }>(`/school/student-guardian-links/${linkID}`, {
    method: "PATCH", body: JSON.stringify(input),
  });
}
export function revokeSchoolStudentGuardianLink(linkID: number, reason: string) {
  return api<{ message: string }>(`/school/student-guardian-links/${linkID}/revoke`, {
    method: "POST", body: JSON.stringify({ reason }),
  });
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
export type TeacherSubjectAssignment = { id: number; academic_year_id: number; section_id: number; section_name: string; class_name: string; subject_id: number; subject_name: string; teacher_principal_id: number; teacher_username: string; assigned_at?: string; revoked_at?: string | null };
export type ClassInchargeAssignment = { id: number; academic_year_id: number; section_id: number; section_name: string; class_name: string; teacher_principal_id: number; teacher_username: string; assigned_at?: string; revoked_at?: string | null };
export function getEligibleSchoolTeachers(signal?: AbortSignal) { return api<{ teachers: EligibleSchoolTeacher[] }>("/school/eligible-teachers", { signal }); }
export function getTeacherSubjectAssignments(filters: { academicYearID?: number; sectionID?: number; teacherPrincipalID?: number } = {}, signal?: AbortSignal) { const params = new URLSearchParams(); if (filters.academicYearID) params.set("academic_year_id", String(filters.academicYearID)); if (filters.sectionID) params.set("section_id", String(filters.sectionID)); if (filters.teacherPrincipalID) params.set("teacher_principal_id", String(filters.teacherPrincipalID)); const query = params.toString(); return api<{ assignments: TeacherSubjectAssignment[] }>("/school/teacher-assignments" + (query ? "?" + query : ""), { signal }); }
export function createTeacherSubjectAssignment(input: { academic_year_id: number; section_id: number; subject_id: number; teacher_principal_id: number }) { return api<{ assignment: TeacherSubjectAssignment }>("/school/teacher-assignments", { method: "POST", body: JSON.stringify(input) }); }
export function revokeTeacherSubjectAssignment(assignmentID: number) { return api<{ message: string }>("/school/teacher-assignments/" + assignmentID, { method: "DELETE" }); }
export function getClassInchargeAssignments(academicYearID?: number, signal?: AbortSignal, teacherPrincipalID?: number) { const params = new URLSearchParams(); if (academicYearID) params.set("academic_year_id", String(academicYearID)); if (teacherPrincipalID) params.set("teacher_principal_id", String(teacherPrincipalID)); const query = params.toString(); return api<{ assignments: ClassInchargeAssignment[] }>("/school/class-incharge-assignments" + (query ? "?" + query : ""), { signal }); }
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
export function getTimetableEntries(academicYearID: number, sectionID = 0, signal?: AbortSignal, teacherPrincipalID?: number) {
  const params = new URLSearchParams({ academic_year_id: String(academicYearID) });
  if (sectionID > 0) params.set("section_id", String(sectionID));
  if (teacherPrincipalID && teacherPrincipalID > 0) params.set("teacher_principal_id", String(teacherPrincipalID));
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

export function getCurrentTimetableYear(signal?: AbortSignal) {
  return api<{ academic_year: StudentAcademicYear }>("/school/timetable/current-year", { signal });
}

export type StaffQualification = {
  id: number; staff_principal_id: number; qualification_type: "school" | "diploma" | "undergraduate" | "postgraduate" | "doctorate" | "other";
  qualification_name: string; specialization?: string; institution: string; board_university?: string;
  completion_year?: number | null; score?: number | null; score_scale?: "percentage" | "cgpa" | "gpa" | "marks" | "grade";
  credential_reference?: string; verification_status: "pending" | "verified" | "rejected"; created_at: string; updated_at: string;
};
export type StaffCertification = {
  id: number; staff_principal_id: number; certificate_name: string; issuing_organization: string;
  issue_date?: string; expiry_date?: string; credential_id?: string; verification_status: "pending" | "verified" | "rejected";
  created_at: string; updated_at: string;
};
export type StaffExperience = {
  id: number; staff_principal_id: number; employer_name: string; designation: string;
  employment_type: "permanent" | "contract" | "part_time" | "temporary" | "internship" | "other";
  start_date: string; end_date?: string; is_current: boolean; description?: string; created_at: string; updated_at: string;
};
export function getStaffQualifications(principalID: number, signal?: AbortSignal) {
  return api<{ qualifications: StaffQualification[] }>(`/school/staff/${principalID}/qualifications`, { signal });
}
export function createStaffQualification(principalID: number, input: Omit<StaffQualification,"id"|"staff_principal_id"|"verification_status"|"created_at"|"updated_at">) {
  return api<{ qualification: StaffQualification }>(`/school/staff/${principalID}/qualifications`, { method:"POST", body:JSON.stringify(input) });
}
export function getStaffCertifications(principalID: number, signal?: AbortSignal) {
  return api<{ certifications: StaffCertification[] }>(`/school/staff/${principalID}/certifications`, { signal });
}
export function createStaffCertification(principalID: number, input: Omit<StaffCertification,"id"|"staff_principal_id"|"verification_status"|"created_at"|"updated_at">) {
  return api<{ certification: StaffCertification }>(`/school/staff/${principalID}/certifications`, { method:"POST", body:JSON.stringify(input) });
}
export function getStaffExperience(principalID: number, signal?: AbortSignal) {
  return api<{ experience: StaffExperience[] }>(`/school/staff/${principalID}/experience`, { signal });
}
export function createStaffExperience(principalID: number, input: Omit<StaffExperience,"id"|"staff_principal_id"|"created_at"|"updated_at">) {
  return api<{ experience: StaffExperience }>(`/school/staff/${principalID}/experience`, { method:"POST", body:JSON.stringify(input) });
}

export function updateStaffQualification(principalID: number, recordID: number, input: Omit<StaffQualification,"id"|"staff_principal_id"|"verification_status"|"created_at"|"updated_at">) {
  return api<{ qualification: StaffQualification }>(`/school/staff/${principalID}/qualifications/${recordID}`, { method:"PATCH", body:JSON.stringify(input) });
}
export function updateStaffCertification(principalID: number, recordID: number, input: Omit<StaffCertification,"id"|"staff_principal_id"|"verification_status"|"created_at"|"updated_at">) {
  return api<{ certification: StaffCertification }>(`/school/staff/${principalID}/certifications/${recordID}`, { method:"PATCH", body:JSON.stringify(input) });
}
export function updateStaffExperience(principalID: number, recordID: number, input: Omit<StaffExperience,"id"|"staff_principal_id"|"created_at"|"updated_at">) {
  return api<{ experience: StaffExperience }>(`/school/staff/${principalID}/experience/${recordID}`, { method:"PATCH", body:JSON.stringify(input) });
}

export type StaffAuditEvent = { id: number; action: string; resource_type: string; outcome: "success" | "failure"; actor_username: string; metadata: Record<string, unknown>; created_at: string };
export function getStaffAuditEvents(principalID: number, signal?: AbortSignal) {
  return api<{ events: StaffAuditEvent[] }>(`/school/staff/${principalID}/audit-events`, { signal });
}

export type StaffAttendanceRecord = {
  id: number | null;
  school_id: number;
  staff_principal_id: number;
  employee_code: string;
  full_name: string;
  designation: string;
  attendance_date: string;
  status: "present" | "absent" | "late" | "half_day" | null;
  remarks?: string | null;
  marked_by_username?: string | null;
  updated_at?: string | null;
};
export type StaffAttendanceMonthlySummary = {
  staff_principal_id: number;
  employee_code: string;
  full_name: string;
  designation: string;
  present_days: number;
  absent_days: number;
  late_days: number;
  half_days: number;
  records: Array<{ id: number; date: string; status: "present" | "absent" | "late" | "half_day"; remarks?: string | null }>;
};
export function getDailyStaffAttendance(date: string, signal?: AbortSignal) {
  return api<{ date: string; data: StaffAttendanceRecord[] }>(`/school/staff-attendance?date=${encodeURIComponent(date)}`, { signal });
}
export function getMonthlyStaffAttendance(month: string, signal?: AbortSignal) {
  return api<{ month: string; data: StaffAttendanceMonthlySummary[] }>(`/school/staff-attendance/month?month=${encodeURIComponent(month)}`, { signal });
}
export function markStaffAttendance(input: { staff_principal_id: number; attendance_date: string; status: "present" | "absent" | "late" | "half_day"; remarks?: string }) {
  return api<{ data: StaffAttendanceRecord }>("/school/staff-attendance", { method: "POST", body: JSON.stringify(input) });
}
export function updateStaffAttendance(recordID: number, input: { status: "present" | "absent" | "late" | "half_day"; remarks?: string }) {
  return api<{ data: StaffAttendanceRecord }>(`/school/staff-attendance/${recordID}`, { method: "PATCH", body: JSON.stringify(input) });
}

export function markStaffAttendanceBulk(input: { staff_principal_ids: number[]; attendance_date: string; status: "present" | "absent" | "late" | "half_day"; remarks?: string }) {
  return api<{ created: number }>("/school/staff-attendance/bulk", { method: "POST", body: JSON.stringify(input) });
}

export type StaffLeaveType = { id:number; name:string; is_paid:boolean; requires_approval:boolean; is_active:boolean };
export type StaffLeaveRequest = {
  id:number; staff_principal_id:number; employee_code?:string; staff_name?:string; leave_type_id:number; leave_type_name?:string;
  start_date:string; end_date:string; duration_type:"full_day"|"half_day"; half_day_session?:"first_half"|"second_half";
  reason:string; status:"pending"|"approved"|"rejected"|"cancelled"; decision_comment?:string|null;
  decided_by_username?:string|null; decided_at?:string|null; created_at:string;
};
export type SchoolHoliday = { id:number; holiday_date:string; name:string; description?:string|null; is_active:boolean };
export function getStaffLeaveTypes(signal?:AbortSignal){return api<{data:StaffLeaveType[]}>("/school/staff-leave/types",{signal});}
export function saveStaffLeaveType(input:Omit<StaffLeaveType,"id"> & {id?:number}) {
 const {id,...body}=input;return api<{data:StaffLeaveType}>(id?("/school/staff-leave/types/"+id):"/school/staff-leave/types",{method:id?"PATCH":"POST",body:JSON.stringify(body)});
}
export function getStaffLeaveRequests(filters:{status?:string;staffPrincipalID?:number;onDate?:string;fromDate?:string;toDate?:string},signal?:AbortSignal){
 const params=new URLSearchParams();if(filters.status)params.set("status",filters.status);if(filters.staffPrincipalID)params.set("staff_principal_id",String(filters.staffPrincipalID));if(filters.onDate)params.set("on_date",filters.onDate);if(filters.fromDate)params.set("from_date",filters.fromDate);if(filters.toDate)params.set("to_date",filters.toDate);
 return api<{data:StaffLeaveRequest[]}>("/school/staff-leave/requests"+(params.size?"?"+params.toString():""),{signal});
}
export function createStaffLeaveRequest(input:{staff_principal_id:number;leave_type_id:number;start_date:string;end_date:string;duration_type:"full_day"|"half_day";half_day_session?:"first_half"|"second_half";reason:string}){
 return api<{data:StaffLeaveRequest}>("/school/staff-leave/requests",{method:"POST",body:JSON.stringify(input)});
}
export function decideStaffLeaveRequest(requestID:number,decision:"approve"|"reject",comment?:string){
 return api<{message:string}>("/school/staff-leave/requests/"+requestID+"/"+decision,{method:"POST",body:JSON.stringify(comment?{comment}:{})});
}
export function cancelStaffLeaveRequest(requestID:number){return api<{message:string}>("/school/staff-leave/requests/"+requestID+"/cancel",{method:"POST"});}
export function getSchoolHolidays(from:string,to:string,signal?:AbortSignal,includeInactive=false){const params=new URLSearchParams({from,to});if(includeInactive)params.set("include_inactive","true");return api<{data:SchoolHoliday[]}>("/school/holidays?"+params.toString(),{signal});}
export function saveSchoolHoliday(input:Omit<SchoolHoliday,"id"> & {id?:number}){
 const {id,...body}=input;return api<{data:SchoolHoliday}>(id?("/school/holidays/"+id):"/school/holidays",{method:id?"PATCH":"POST",body:JSON.stringify(body)});
}

export type StaffSubstitution = {
  id:number; timetable_entry_id:number; cover_date:string;
  original_teacher_principal_id:number; original_teacher_name:string;
  substitute_teacher_principal_id:number; substitute_teacher_name:string;
  approved_leave_request_id?:number|null; reason:string; status:"active"|"cancelled";
  section_name:string; class_name:string; subject_name:string; period_number:number; period_name:string;
  starts_at:string; ends_at:string; room?:string|null; created_by_principal_id:number; created_at:string;
  cancelled_by_principal_id?:number|null; cancelled_at?:string|null;
};
export function getStaffSubstitutions(filters:{fromDate:string;toDate:string;staffPrincipalID?:number},signal?:AbortSignal){
  const params=new URLSearchParams({from_date:filters.fromDate,to_date:filters.toDate});
  if(filters.staffPrincipalID)params.set("staff_principal_id",String(filters.staffPrincipalID));
  return api<{substitutions:StaffSubstitution[]}>(`/school/staff-substitutions?${params.toString()}`,{signal});
}
export function createStaffSubstitution(input:{
  timetable_entry_id:number; original_teacher_principal_id:number; substitute_teacher_principal_id:number;
  cover_date:string; approved_leave_request_id?:number|null; reason:string;
}){
  return api<{substitution:StaffSubstitution}>("/school/staff-substitutions",{method:"POST",body:JSON.stringify(input)});
}
export function cancelStaffSubstitution(substitutionID:number){
  return api<{message:string}>(`/school/staff-substitutions/${substitutionID}/cancel`,{method:"POST"});
}

export type TeacherWorkloadReportRow = {
  teacher_principal_id:number; teacher_name:string; employee_code:string; designation:string;
  employment_status:"active"|"inactive"; login_status:"active"|"disabled"; day_of_week:number;
  scheduled_periods:number; distinct_sections:number; distinct_subjects:number;
  first_period?:number|null; last_period?:number|null;
};
export function getTeacherWorkloadReport(academicYearID:number,signal?:AbortSignal){
  return api<{workloads:TeacherWorkloadReportRow[]}>(`/school/timetable/workload-report?academic_year_id=${academicYearID}`,{signal});
}


export type StaffQualificationReportRow = {
  record_type: "qualification" | "certification";
  record_id: number;
  staff_principal_id: number;
  employee_code: string;
  staff_name: string;
  record_name: string;
  organization: string;
  record_date?: string;
  expiry_date?: string;
  verification_status: "pending" | "verified" | "rejected";
};
export type StaffQualificationReportPage = {
  rows: StaffQualificationReportRow[];
  counts: Array<{ record_type: "qualification" | "certification"; verification_status: "pending" | "verified" | "rejected"; count: number }>;
  limit: number;
  offset: number;
  total: number;
};
export function getStaffQualificationsReport(params: { recordType?: "all" | "qualification" | "certification"; verificationStatus?: "pending" | "verified" | "rejected"; limit?: number; offset?: number } = {}, signal?: AbortSignal) {
  const query = new URLSearchParams();
  if (params.recordType) query.set("record_type", params.recordType);
  if (params.verificationStatus) query.set("verification_status", params.verificationStatus);
  if (params.limit !== undefined) query.set("limit", String(params.limit));
  if (params.offset !== undefined) query.set("offset", String(params.offset));
  const suffix = query.toString();
  return api<{ report: StaffQualificationReportPage }>(`/school/staff/qualifications-report${suffix ? `?${suffix}` : ""}`, { signal });
}


export type StaffExperienceReportRow = {
  record_id:number;
  staff_principal_id:number;
  employee_code:string;
  staff_name:string;
  designation:string;
  employment_status:"active"|"inactive"|string;
  employer_name:string;
  previous_designation:string;
  employment_type:"permanent"|"contract"|"part_time"|"temporary"|"internship"|"other";
  start_date:string;
  end_date?:string;
  is_current:boolean;
};
export type StaffExperienceReportPage = {
  rows:StaffExperienceReportRow[];
  counts:Array<{employment_type:string;is_current:boolean;count:number}>;
  limit:number;
  offset:number;
  total:number;
};
export function getStaffExperienceReport(params:{employmentType?:StaffExperienceReportRow["employment_type"];currentOnly?:boolean;limit?:number;offset?:number}={},signal?:AbortSignal){
  const query=new URLSearchParams();
  if(params.employmentType)query.set("employment_type",params.employmentType);
  if(params.currentOnly)query.set("current_only","true");
  if(params.limit!==undefined)query.set("limit",String(params.limit));
  if(params.offset!==undefined)query.set("offset",String(params.offset));
  const suffix=query.toString();
  return api<{report:StaffExperienceReportPage}>(`/school/staff/experience-report${suffix?"?"+suffix:""}`,{signal});
}
