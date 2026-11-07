import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createSchoolStaff, disableSchoolStaff, enableSchoolStaff, getSchoolStaff,
  resetSchoolStaffPassword, updateSchoolStaff,
} from "./api";

export const schoolAdminQueryKeys = {
  all: ["school-admin"] as const,
  staff: () => [...schoolAdminQueryKeys.all, "staff"] as const,
};

export function useSchoolStaffQuery() {
  return useQuery({
    queryKey: schoolAdminQueryKeys.staff(),
    queryFn: ({ signal }) => getSchoolStaff(signal),
    staleTime: 30_000,
  });
}

function useStaffMutation<TInput>(mutationFn: (input: TInput) => Promise<unknown>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: schoolAdminQueryKeys.staff() }),
  });
}
export function useCreateSchoolStaffMutation() {
  return useStaffMutation(createSchoolStaff);
}
export function useUpdateSchoolStaffMutation() {
  return useStaffMutation(({ principalID, username }: { principalID: number; username: string }) => updateSchoolStaff(principalID, { username }));
}
export function useDisableSchoolStaffMutation() {
  return useStaffMutation(disableSchoolStaff);
}
export function useEnableSchoolStaffMutation() {
  return useStaffMutation(({ principalID, temporaryPassword }: { principalID: number; temporaryPassword: string }) => enableSchoolStaff(principalID, temporaryPassword));
}
export function useResetSchoolStaffPasswordMutation() {
  return useStaffMutation(({ principalID, newPassword }: { principalID: number; newPassword: string }) => resetSchoolStaffPassword(principalID, newPassword));
}
