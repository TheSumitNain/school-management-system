import { useQuery } from "@tanstack/react-query";
import { getSchoolStaff } from "./api";

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
