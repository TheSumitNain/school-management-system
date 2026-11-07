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
