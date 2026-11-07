import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createPlatformAdmin,
  deactivatePlatformAdmin,
  getPlatformAdmins,
  getPlatformSchools,
  getDelegableSchoolPermissions,
  getPlatformSchoolPermissionDelegations,
  grantPlatformSchoolPermissionDelegation,
  revokePlatformSchoolPermissionDelegation,
  onboardSchool,
  type CreatePlatformAdminInput,
  type CreateSchoolInput,
} from "./api";

export const platformAdminQueryKeys = {
  all: ["platform-admin"] as const,
  admins: () => [...platformAdminQueryKeys.all, "admins"] as const,
  schools: () => [...platformAdminQueryKeys.all, "schools"] as const,
  delegablePermissions: () => [...platformAdminQueryKeys.all, "delegable-permissions"] as const,
  delegations: (schoolID: number) => [...platformAdminQueryKeys.all, "schools", schoolID, "delegations"] as const,
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


export function usePlatformSchoolsQuery(enabled = true) {
  return useQuery({
    queryKey: platformAdminQueryKeys.schools(),
    queryFn: ({ signal }) => getPlatformSchools(signal),
    enabled,
    staleTime: 30_000,
  });
}

export function useDelegableSchoolPermissionsQuery(enabled = true) {
  return useQuery({
    queryKey: platformAdminQueryKeys.delegablePermissions(),
    queryFn: ({ signal }) => getDelegableSchoolPermissions(signal),
    enabled,
    staleTime: 5 * 60_000,
  });
}

export function usePlatformSchoolDelegationsQuery(schoolID: number | null, enabled = true) {
  return useQuery({
    queryKey: schoolID === null ? [...platformAdminQueryKeys.all, "schools", "none", "delegations"] : platformAdminQueryKeys.delegations(schoolID),
    queryFn: ({ signal }) => getPlatformSchoolPermissionDelegations(schoolID as number, signal),
    enabled: enabled && schoolID !== null && schoolID > 0,
    staleTime: 15_000,
  });
}

export function useGrantPlatformSchoolPermissionDelegationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { schoolID: number; permission_key: string; can_assign_to_roles: boolean; can_override_users: boolean }) => {
      const { schoolID, ...body } = input;
      return grantPlatformSchoolPermissionDelegation(schoolID, body);
    },
    onSuccess: (_result, variables) => queryClient.invalidateQueries({ queryKey: platformAdminQueryKeys.delegations(variables.schoolID) }),
  });
}

export function useRevokePlatformSchoolPermissionDelegationMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { delegationID: number; schoolID: number }) => revokePlatformSchoolPermissionDelegation(input.delegationID),
    onSuccess: (_result, variables) => queryClient.invalidateQueries({ queryKey: platformAdminQueryKeys.delegations(variables.schoolID) }),
  });
}
