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
  getSchoolStudentMedicalDetails, getSchoolStudentPhysicalDetails,
  getSchoolStudentEnrollments, getSchoolStudentEnrollmentHistory, getSchoolStudentLocations, getSchoolStudentRelatedStudents,
  getSchoolStudentGuardians, getSchoolStudents, getSchoolGuardians, linkSchoolStudentGuardian,
  revokeSchoolStudentGuardianLink, createSchoolGuardian, saveSchoolStudentAddress,
  saveSchoolStudentContactDetails, saveSchoolStudentMedicalDetails, saveSchoolStudentPhysicalDetails, updateSchoolStudent,
} from "./api";

export const guardianQueryKeys = {
  all: ["school-admin", "guardians"] as const,
  list: (filters: { q?: string; status?: string }) => ["school-admin", "guardians", "list", filters] as const,
  studentLinks: (studentID: number, includeHistory: boolean) => ["school-admin", "guardians", "student-links", studentID, includeHistory] as const,
  relatedStudents: (studentID: number) => ["school-admin", "guardians", "related-students", studentID] as const,
};

export const schoolStudentQueryKeys = {
  all: [...schoolAdminQueryKeys.all, "students"] as const,
  list: (filters: { q: string; status: string; limit: number; offset: number }) => [...schoolStudentQueryKeys.all, "list", filters] as const,
  enrollments: (academicYearID: number) => [...schoolStudentQueryKeys.all, "enrollments", academicYearID] as const,
  enrollmentHistory: (studentID: number) => [...schoolStudentQueryKeys.all, "enrollment-history", studentID] as const,
  years: () => [...schoolStudentQueryKeys.all, "academic-years"] as const,
  sections: (academicYearID: number) => [...schoolStudentQueryKeys.all, "sections", academicYearID] as const,
  addresses: (studentID: number) => [...schoolStudentQueryKeys.all, "addresses", studentID] as const,
  medicalDetails: (studentID: number) => [...schoolStudentQueryKeys.all, "medical-details", studentID] as const,
  physicalDetails: (studentID: number) => [...schoolStudentQueryKeys.all, "physical-details", studentID] as const,
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
export function useSchoolStudentEnrollmentHistoryQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.enrollmentHistory(studentID),
    queryFn: ({ signal }) => getSchoolStudentEnrollmentHistory(studentID, signal),
    enabled: enabled && studentID > 0,
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
export function useSchoolStudentMedicalDetailsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.medicalDetails(studentID),
    queryFn: ({ signal }) => getSchoolStudentMedicalDetails(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 0,
    gcTime: 0,
    refetchOnMount: "always",
  });
}
export function useSchoolStudentPhysicalDetailsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.physicalDetails(studentID),
    queryFn: ({ signal }) => getSchoolStudentPhysicalDetails(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 0,
    gcTime: 0,
    refetchOnMount: "always",
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
export function useSchoolGuardiansQuery(filters: { q?: string; status?: string } = {}, enabled = true) {
  return useQuery({
    queryKey: guardianQueryKeys.list(filters),
    queryFn: ({ signal }) => getSchoolGuardians(filters, signal),
    enabled,
    staleTime: 15_000,
  });
}
export function useSchoolStudentGuardiansQuery(studentID: number, includeHistory = false, enabled = true) {
  return useQuery({
    queryKey: guardianQueryKeys.studentLinks(studentID, includeHistory),
    queryFn: ({ signal }) => getSchoolStudentGuardians(studentID, includeHistory, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
  });
}
export function useSchoolStudentRelatedStudentsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: guardianQueryKeys.relatedStudents(studentID),
    queryFn: ({ signal }) => getSchoolStudentRelatedStudents(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
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
export function useSaveSchoolStudentMedicalDetailsMutation() {
  return useStudentMutation(({ studentID, input }: {
    studentID: number; input: { blood_group?: string | null; allergies?: string | null; known_conditions?: string | null;
      ongoing_medications?: string | null; emergency_medical_notes?: string | null };
  }) => saveSchoolStudentMedicalDetails(studentID, input));
}
export function useSaveSchoolStudentPhysicalDetailsMutation() {
  return useStudentMutation(({ studentID, input }: {
    studentID: number; input: { height_cm?: number | null; weight_kg?: number | null; measured_on?: string | null;
      vision_notes?: string | null; physical_notes?: string | null };
  }) => saveSchoolStudentPhysicalDetails(studentID, input));
}
function useGuardianMutation<TInput>(mutationFn: (input: TInput) => Promise<unknown>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      client.invalidateQueries({ queryKey: guardianQueryKeys.all });
      client.invalidateQueries({ queryKey: schoolStudentQueryKeys.all });
    },
  });
}
export function useCreateSchoolGuardianMutation() {
  return useGuardianMutation(createSchoolGuardian);
}
export function useLinkSchoolStudentGuardianMutation() {
  return useGuardianMutation(({ studentID, input }: {
    studentID: number; input: { guardian_id: number; relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean };
  }) => linkSchoolStudentGuardian(studentID, input));
}
export function useRevokeSchoolStudentGuardianMutation() {
  return useGuardianMutation(({ linkID, reason }: { linkID: number; reason: string }) => revokeSchoolStudentGuardianLink(linkID, reason));
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
