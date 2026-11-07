import { api } from "../../lib/api";

export type StaffRecord = Record<string, unknown>;
export type StaffListResponse = { staff: StaffRecord[] };

export function getSchoolStaff(signal?: AbortSignal) {
  return api<StaffListResponse>("/school/staff", { signal });
}
