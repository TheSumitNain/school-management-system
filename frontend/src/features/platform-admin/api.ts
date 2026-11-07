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
