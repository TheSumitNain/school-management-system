import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createPlatformAdmin,
  deactivatePlatformAdmin,
  getPlatformAdmins,
  onboardSchool,
  type CreatePlatformAdminInput,
  type CreateSchoolInput,
} from "./api";

export const platformAdminQueryKeys = {
  all: ["platform-admin"] as const,
  admins: () => [...platformAdminQueryKeys.all, "admins"] as const,
};

export function usePlatformAdminsQuery(enabled = true) {
  return useQuery({
    queryKey: platformAdminQueryKeys.admins(),
    queryFn: ({ signal }) => getPlatformAdmins(signal),
    enabled,
    staleTime: 30_000,
  });
}

export function useCreatePlatformAdminMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreatePlatformAdminInput) => createPlatformAdmin(input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: platformAdminQueryKeys.admins() }),
  });
}

export function useDeactivatePlatformAdminMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (principalID: number) => deactivatePlatformAdmin(principalID),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: platformAdminQueryKeys.admins() }),
  });
}

export function useOnboardSchoolMutation() {
  return useMutation({
    mutationFn: (input: CreateSchoolInput) => onboardSchool(input),
  });
}
