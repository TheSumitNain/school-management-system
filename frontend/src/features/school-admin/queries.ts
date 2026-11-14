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
  return useStaffMutation(({ principalID, ...input }: { principalID: number; username: string; employee_code: string; full_name: string; designation: string; mobile_number: string; email?: string; joining_date?: string; employment_status?: "active" | "inactive" }) => updateSchoolStaff(principalID, input));
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


import {
  archiveSchoolStudent, createSchoolStudentAdmission, createSchoolStudentEnrollment,
  createSchoolStudentLocation, deactivateSchoolStudentLocation, getSchoolAcademicYears,
  getSchoolSections, getSchoolStudentAddresses, getSchoolStudentContactDetails,
  getSchoolStudentEnrollments, getSchoolStudentLocations, getSchoolStudents,
  saveSchoolStudentAddress, saveSchoolStudentContactDetails, updateSchoolStudent,
} from "./api";

export const schoolStudentQueryKeys = {
  all: [...schoolAdminQueryKeys.all, "students"] as const,
  list: (filters: { q: string; status: string; limit: number; offset: number }) => [...schoolStudentQueryKeys.all, "list", filters] as const,
  enrollments: (academicYearID: number) => [...schoolStudentQueryKeys.all, "enrollments", academicYearID] as const,
  years: () => [...schoolStudentQueryKeys.all, "academic-years"] as const,
  sections: (academicYearID: number) => [...schoolStudentQueryKeys.all, "sections", academicYearID] as const,
  addresses: (studentID: number) => [...schoolStudentQueryKeys.all, "addresses", studentID] as const,
  locations: (filters: { parentID?: number; level?: string; q?: string }) => [...schoolStudentQueryKeys.all, "locations", filters] as const,
};
export function useSchoolStudentsQuery(filters: { q: string; status: string; limit: number; offset: number }, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.list(filters),
    queryFn: ({ signal }) => getSchoolStudents(filters, signal),
    enabled,
    staleTime: 15_000,
  });
}
export function useSchoolStudentEnrollmentsQuery(academicYearID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.enrollments(academicYearID),
    queryFn: ({ signal }) => getSchoolStudentEnrollments(academicYearID, signal),
    enabled,
    staleTime: 15_000,
  });
}
export function useSchoolStudentAddressesQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.addresses(studentID),
    queryFn: ({ signal }) => getSchoolStudentAddresses(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
  });
}
export function useSchoolStudentContactDetailsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: [...schoolStudentQueryKeys.all, "contact-details", studentID] as const,
    queryFn: ({ signal }) => getSchoolStudentContactDetails(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
  });
}
export function useSchoolStudentLocationsQuery(filters: { parentID?: number; level?: string; q?: string } = {}, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.locations(filters),
    queryFn: ({ signal }) => getSchoolStudentLocations(filters, signal),
    enabled,
    staleTime: 60_000,
  });
}
export function useSchoolStudentAcademicYearsQuery(enabled = true) {
  return useQuery({ queryKey: schoolStudentQueryKeys.years(), queryFn: ({ signal }) => getSchoolAcademicYears(signal), enabled, staleTime: 60_000 });
}
export function useSchoolStudentSectionsQuery(academicYearID: number, enabled = true) {
  return useQuery({ queryKey: schoolStudentQueryKeys.sections(academicYearID), queryFn: ({ signal }) => getSchoolSections(academicYearID, signal), enabled: enabled && academicYearID > 0, staleTime: 60_000 });
}
function useStudentMutation<TInput>(mutationFn: (input: TInput) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: schoolStudentQueryKeys.all });
    },
  });
}
export function useCreateSchoolStudentMutation() {
  return useStudentMutation(createSchoolStudentAdmission);
}
export function useUpdateSchoolStudentMutation() {
  return useStudentMutation(({ studentID, fullName }: { studentID: number; fullName: string }) => updateSchoolStudent(studentID, { full_name: fullName }));
}
export function useArchiveSchoolStudentMutation() {
  return useStudentMutation(archiveSchoolStudent);
}
export function useCreateSchoolStudentEnrollmentMutation() {
  return useStudentMutation(createSchoolStudentEnrollment);
}
export function useSaveSchoolStudentAddressMutation() {
  return useStudentMutation(({ studentID, addressType, input }: {
    studentID: number; addressType: "current" | "permanent";
    input: { address_line_1?: string | null; address_line_2?: string | null; area_mohalla?: string | null;
      landmark?: string | null; postal_code?: string | null; location_id?: number | null; same_as_current?: boolean };
  }) => saveSchoolStudentAddress(studentID, addressType, input));
}
export function useSaveSchoolStudentContactDetailsMutation() {
  return useStudentMutation(({ studentID, input }: {
    studentID: number; input: { primary_phone?: string | null; alternate_phone?: string | null; email_address?: string | null };
  }) => saveSchoolStudentContactDetails(studentID, input));
}
function useStudentLocationMutation<TInput>(mutationFn: (input: TInput) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => client.invalidateQueries({ queryKey: [...schoolStudentQueryKeys.all, "locations"] }),
  });
}
export function useCreateSchoolStudentLocationMutation() {
  return useStudentLocationMutation(createSchoolStudentLocation);
}
export function useDeactivateSchoolStudentLocationMutation() {
  return useStudentLocationMutation(deactivateSchoolStudentLocation);
}


import {
  getSchoolAttendance, markSchoolAttendance, updateSchoolAttendance,
} from "./api";

export const schoolAttendanceQueryKeys = {
  all: [...schoolAdminQueryKeys.all, "attendance"] as const,
  list: (filters: { academicYearID: number; sectionID: number; date: string }) => [...schoolAttendanceQueryKeys.all, filters] as const,
};
export function useSchoolAttendanceQuery(filters: { academicYearID: number; sectionID: number; date: string }, enabled = true) {
  return useQuery({
    queryKey: schoolAttendanceQueryKeys.list(filters),
    queryFn: ({ signal }) => getSchoolAttendance(filters, signal),
    enabled,
    staleTime: 10_000,
  });
}
function useAttendanceMutation<TInput>(mutationFn: (input: TInput) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => client.invalidateQueries({ queryKey: schoolAttendanceQueryKeys.all }),
  });
}
export function useMarkSchoolAttendanceMutation() {
  return useAttendanceMutation(markSchoolAttendance);
}
export function useUpdateSchoolAttendanceMutation() {
  return useAttendanceMutation(({ attendanceID, status, remarks }: { attendanceID: number; status: string; remarks?: string | null }) =>
    updateSchoolAttendance(attendanceID, { status, remarks }));
}
