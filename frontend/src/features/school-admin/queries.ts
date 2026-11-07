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


import {
  archiveSchoolStudent, createSchoolStudent, createSchoolStudentEnrollment,
  getSchoolAcademicYears, getSchoolSections, getSchoolStudentEnrollments, getSchoolStudents,
  updateSchoolStudent,
} from "./api";

export const schoolStudentQueryKeys = {
  all: [...schoolAdminQueryKeys.all, "students"] as const,
  list: (filters: { q: string; status: string; limit: number; offset: number }) => [...schoolStudentQueryKeys.all, "list", filters] as const,
  enrollments: (academicYearID: number) => [...schoolStudentQueryKeys.all, "enrollments", academicYearID] as const,
  years: () => [...schoolStudentQueryKeys.all, "academic-years"] as const,
  sections: (academicYearID: number) => [...schoolStudentQueryKeys.all, "sections", academicYearID] as const,
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
  return useStudentMutation(createSchoolStudent);
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
