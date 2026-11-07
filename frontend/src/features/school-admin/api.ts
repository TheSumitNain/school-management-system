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
