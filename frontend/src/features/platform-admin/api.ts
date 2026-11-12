import { api } from "../../lib/api";

export type PlatformAdminAccount = {
  principal_id: number;
  email: string;
  status: string;
  mfa_enrolled: boolean;
  created_at: string;
};

export type PlatformAdminListResponse = { admins: PlatformAdminAccount[] };
export type CreatePlatformAdminInput = { email: string; password: string };
export type CreatePlatformAdminResponse = {
  admin: PlatformAdminAccount;
  mfa_enrollment_required: boolean;
};

export type CreateSchoolInput = {
  name: string;
  code: string;
  school_admin_username: string;
  school_admin_initial_password: string;
};

export type CreateSchoolResponse = {
  school: { id: number; name: string; code: string; status: string };
  school_admin: { principal_id: number; username: string };
};

export function getPlatformAdmins(signal?: AbortSignal) {
  return api<PlatformAdminListResponse>("/platform/admins", { signal });
}

export function createPlatformAdmin(input: CreatePlatformAdminInput) {
  return api<CreatePlatformAdminResponse>("/platform/admins", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function deactivatePlatformAdmin(principalID: number) {
  return api<{ message: string }>(`/platform/admins/${principalID}/deactivate`, {
    method: "POST",
  });
}

export function onboardSchool(input: CreateSchoolInput) {
  return api<CreateSchoolResponse>("/platform/schools", {
    method: "POST",
    body: JSON.stringify(input),
  });
}


export type PlatformSchool = {
  id: number;
  name: string;
  code: string;
  status: "pending_setup" | "active" | "suspended" | string;
  created_at: string;
};
export type PlatformSchoolsResponse = { schools: PlatformSchool[] };
export type DelegableSchoolPermission = {
  permission_key: string;
  description: string;
  is_delegable: boolean;
};
export type DelegableSchoolPermissionsResponse = { permissions: DelegableSchoolPermission[] };
export type PlatformSchoolPermissionDelegation = {
  id: number;
  school_id: number;
  permission_key: string;
  can_assign_to_roles: boolean;
  can_override_users: boolean;
  granted_by: number;
  created_at: string;
  revoked_at?: string;
};
export type PlatformSchoolPermissionDelegationsResponse = { delegations: PlatformSchoolPermissionDelegation[] };

export function getPlatformSchools(signal?: AbortSignal) {
  return api<PlatformSchoolsResponse>("/platform/schools", { signal });
}
export function getDelegableSchoolPermissions(signal?: AbortSignal) {
  return api<DelegableSchoolPermissionsResponse>("/platform/permissions/delegable", { signal });
}
export function getPlatformSchoolPermissionDelegations(schoolID: number, signal?: AbortSignal) {
  return api<PlatformSchoolPermissionDelegationsResponse>(`/platform/schools/${schoolID}/permission-delegations`, { signal });
}
export function grantPlatformSchoolPermissionDelegation(schoolID: number, input: {
  permission_key: string;
  can_assign_to_roles: boolean;
  can_override_users: boolean;
}) {
  return api<{ delegation: PlatformSchoolPermissionDelegation; message: string }>(
    `/platform/schools/${schoolID}/permission-delegations`,
    { method: "POST", body: JSON.stringify(input) },
  );
}
export function revokePlatformSchoolPermissionDelegation(delegationID: number) {
  return api<{ message: string }>(`/platform/permission-delegations/${delegationID}/revoke`, { method: "POST" });
}
