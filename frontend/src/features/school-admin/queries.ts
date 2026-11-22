import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getSchoolClassCurricula, getSchoolStudentEnrollmentCurriculum, getSchoolStudentEnrollmentDirectory, previewSchoolClassCurriculumRemoval, saveSchoolClassCurriculum,
  saveSchoolStudentEnrollmentSubjects,
  createSchoolStaff, disableSchoolStaff, enableSchoolStaff, getSchoolStaff,
  resetSchoolStaffPassword, updateSchoolStaff,
  getStaffQualifications, createStaffQualification, updateStaffQualification, getStaffCertifications, createStaffCertification, updateStaffCertification, getStaffExperience, createStaffExperience, updateStaffExperience,
  getStaffLeaveTypes, saveStaffLeaveType, getStaffLeaveRequests, createStaffLeaveRequest, decideStaffLeaveRequest, cancelStaffLeaveRequest, getSchoolHolidays, saveSchoolHoliday,
  getStaffSubstitutions, createStaffSubstitution, cancelStaffSubstitution, getCurrentTimetableYear, getTimetableEntries,
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
  archiveSchoolStudent, createSchoolStudentAdmission, createSchoolStudentEnrollment, promoteSchoolStudentEnrollment,
  getSchoolStudentBasicDetails, saveSchoolStudentBasicDetails,
  createSchoolStudentLocation, deactivateSchoolStudentLocation, getSchoolAcademicYears,
  getSchoolSections, getSchoolStudentAddresses, getSchoolStudentContactDetails,
  getSchoolStudentMedicalDetails, getSchoolStudentPhysicalDetails,
  getSchoolStudentEnrollments, getSchoolStudentEnrollmentHistory, getSchoolStudentLocations, getSchoolStudentRelatedStudents,
  getSchoolStudentGuardians, getSchoolStudents, getSchoolGuardians, linkSchoolStudentGuardian,
  revokeSchoolStudentGuardianLink, updateSchoolStudentGuardianLink, createSchoolGuardian, updateSchoolGuardian, saveSchoolStudentAddress,
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
  list: (filters: { q: string; status: string; academicYearID?: number; limit: number; offset: number }) => [...schoolStudentQueryKeys.all, "list", filters] as const,
  enrollments: (academicYearID: number) => [...schoolStudentQueryKeys.all, "enrollments", academicYearID] as const,
  enrollmentDirectory: (academicYearID: number, limit: number, offset: number) => [...schoolStudentQueryKeys.all, "enrollment-directory", academicYearID, limit, offset] as const,
  enrollmentHistory: (studentID: number) => [...schoolStudentQueryKeys.all, "enrollment-history", studentID] as const,
  curriculum: (enrollmentID: number) => [...schoolStudentQueryKeys.all, "curriculum", enrollmentID] as const,
  curricula: () => [...schoolStudentQueryKeys.all, "curriculum"] as const,
  classCurricula: (yearID: number, classID: number, sectionID: number) => [...schoolStudentQueryKeys.all, "class-curricula", yearID, classID, sectionID] as const,
  years: () => [...schoolStudentQueryKeys.all, "academic-years"] as const,
  sections: (academicYearID: number) => [...schoolStudentQueryKeys.all, "sections", academicYearID] as const,
  addresses: (studentID: number) => [...schoolStudentQueryKeys.all, "addresses", studentID] as const,
  medicalDetails: (studentID: number) => [...schoolStudentQueryKeys.all, "medical-details", studentID] as const,
  physicalDetails: (studentID: number) => [...schoolStudentQueryKeys.all, "physical-details", studentID] as const,
  locations: (filters: { parentID?: number; level?: string; q?: string }) => [...schoolStudentQueryKeys.all, "locations", filters] as const,
};
export function useSchoolStudentsQuery(filters: { q: string; status: string; academicYearID?: number; limit: number; offset: number }, enabled = true) {
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
export function useSchoolStudentEnrollmentDirectoryQuery(academicYearID: number, limit = 20, offset = 0, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.enrollmentDirectory(academicYearID, limit, offset),
    queryFn: ({ signal }) => getSchoolStudentEnrollmentDirectory(academicYearID, limit, offset, signal),
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
    retry: false,
  });
}
export function useSchoolStudentBasicDetailsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: [...schoolStudentQueryKeys.all, "basic-details", studentID] as const,
    queryFn: ({ signal }) => getSchoolStudentBasicDetails(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
    retry: false,
  });
}
export function useSchoolStudentContactDetailsQuery(studentID: number, enabled = true) {
  return useQuery({
    queryKey: [...schoolStudentQueryKeys.all, "contact-details", studentID] as const,
    queryFn: ({ signal }) => getSchoolStudentContactDetails(studentID, signal),
    enabled: enabled && studentID > 0,
    staleTime: 15_000,
    retry: false,
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
    // The write is confirmed by the server before onSuccess runs. Start
    // background refetches without holding the mutation promise open; callers
    // can show success feedback immediately while queries refresh.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: schoolStudentQueryKeys.all });
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
export function usePromoteSchoolStudentEnrollmentMutation() {
  return useStudentMutation(({ sourceEnrollmentID, input }: {
    sourceEnrollmentID: number;
    input: { academic_year_id: number; section_id: number; roll_number?: string | null;
      admission_number: string; admission_date: string; enrolled_at: string };
  }) => promoteSchoolStudentEnrollment(sourceEnrollmentID, input));
}
export function useSaveSchoolStudentAddressMutation() {
  return useStudentMutation(({ studentID, addressType, input }: {
    studentID: number; addressType: "current" | "permanent";
    input: { address_line_1?: string | null; address_line_2?: string | null; area_mohalla?: string | null;
      landmark?: string | null; postal_code?: string | null; location_id?: number | null; same_as_current?: boolean };
  }) => saveSchoolStudentAddress(studentID, addressType, input));
}
export function useSaveSchoolStudentBasicDetailsMutation() {
  return useStudentMutation(({ studentID, input }: {
    studentID: number; input: { date_of_birth?: string | null; place_of_birth?: string | null;
      gender?: string | null; nationality?: string | null; religion?: string | null;
      social_category?: string | null; ews_status?: boolean | null; school_registration_number?: string | null };
  }) => saveSchoolStudentBasicDetails(studentID, input));
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
    studentID: number; input: { height_cm?: number | null; weight_kg?: number | null; measured_on: string;
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
export function useUpdateSchoolGuardianMutation() {
  return useGuardianMutation(({ guardianID, input }: {
    guardianID: number; input: {
      full_name: string; mobile_number?: string | null; email_address?: string | null;
      alternate_mobile_number?: string | null; occupation?: string | null; education?: string | null;
      workplace_name?: string | null; status: string;
    };
  }) => updateSchoolGuardian(guardianID, input));
}
export function useLinkSchoolStudentGuardianMutation() {
  return useGuardianMutation(({ studentID, input }: {
    studentID: number; input: { guardian_id: number; relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean };
  }) => linkSchoolStudentGuardian(studentID, input));
}
export function useUpdateSchoolStudentGuardianMutation() {
  return useGuardianMutation(({ linkID, input }: {
    linkID: number; input: { relationship_type: string; is_primary_contact: boolean; is_emergency_contact: boolean };
  }) => updateSchoolStudentGuardianLink(linkID, input));
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

export function useSchoolClassCurriculaQuery(academicYearID: number, classID: number, sectionID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.classCurricula(academicYearID, classID, sectionID),
    queryFn: ({ signal }) => getSchoolClassCurricula(academicYearID, classID, sectionID, signal),
    enabled: enabled && academicYearID > 0 && classID > 0 && sectionID > 0,
    staleTime: 15_000,
  });
}
export function useSchoolStudentEnrollmentCurriculumQuery(enrollmentID: number, enabled = true) {
  return useQuery({
    queryKey: schoolStudentQueryKeys.curriculum(enrollmentID),
    queryFn: ({ signal }) => getSchoolStudentEnrollmentCurriculum(enrollmentID, signal),
    enabled: enabled && enrollmentID > 0,
    staleTime: 15_000,
  });
}
export function usePreviewSchoolClassCurriculumRemovalMutation() {
  return useMutation({ mutationFn: previewSchoolClassCurriculumRemoval });
}
export function useSaveSchoolClassCurriculumMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: saveSchoolClassCurriculum,
    onSuccess: async (_data, input) => {
      await Promise.all([
        client.invalidateQueries({ queryKey: schoolStudentQueryKeys.classCurricula(input.academic_year_id, input.class_id, input.section_id) }),
        client.invalidateQueries({ queryKey: schoolStudentQueryKeys.curricula() }),
      ]);
    },
  });
}
export function useSaveSchoolStudentEnrollmentSubjectsMutation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ enrollmentID, ...input }: { enrollmentID: number; curriculum_id: number; selected_optional_subject_ids: number[] }) =>
      saveSchoolStudentEnrollmentSubjects(enrollmentID, input),
    onSuccess: (_data, input) => client.invalidateQueries({ queryKey: schoolStudentQueryKeys.curriculum(input.enrollmentID) }),
  });
}

export function useStaffQualificationsQuery(principalID: number, enabled: boolean) {
  return useQuery({ queryKey: [...schoolAdminQueryKeys.staff(), principalID, "qualifications"], queryFn: ({signal}) => getStaffQualifications(principalID, signal), enabled: enabled && principalID > 0, staleTime: 15_000 });
}
export function useStaffCertificationsQuery(principalID: number, enabled: boolean) {
  return useQuery({ queryKey: [...schoolAdminQueryKeys.staff(), principalID, "certifications"], queryFn: ({signal}) => getStaffCertifications(principalID, signal), enabled: enabled && principalID > 0, staleTime: 15_000 });
}
export function useStaffExperienceQuery(principalID: number, enabled: boolean) {
  return useQuery({ queryKey: [...schoolAdminQueryKeys.staff(), principalID, "experience"], queryFn: ({signal}) => getStaffExperience(principalID, signal), enabled: enabled && principalID > 0, staleTime: 15_000 });
}
function useStaffRecordMutation<TInput>(principalID: number, mutationFn: (input:TInput)=>Promise<unknown>, key: string) {
  const client = useQueryClient();
  return useMutation({ mutationFn, onSuccess: () => client.invalidateQueries({queryKey:[...schoolAdminQueryKeys.staff(),principalID,key]}) });
}
export function useCreateStaffQualificationMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:Parameters<typeof createStaffQualification>[1])=>createStaffQualification(principalID,input),"qualifications");
}
export function useCreateStaffCertificationMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:Parameters<typeof createStaffCertification>[1])=>createStaffCertification(principalID,input),"certifications");
}
export function useCreateStaffExperienceMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:Parameters<typeof createStaffExperience>[1])=>createStaffExperience(principalID,input),"experience");
}

export function useUpdateStaffQualificationMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:{recordID:number;data:Parameters<typeof updateStaffQualification>[2]})=>updateStaffQualification(principalID,input.recordID,input.data),"qualifications");
}
export function useUpdateStaffCertificationMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:{recordID:number;data:Parameters<typeof updateStaffCertification>[2]})=>updateStaffCertification(principalID,input.recordID,input.data),"certifications");
}
export function useUpdateStaffExperienceMutation(principalID:number) {
  return useStaffRecordMutation(principalID,(input:{recordID:number;data:Parameters<typeof updateStaffExperience>[2]})=>updateStaffExperience(principalID,input.recordID,input.data),"experience");
}

export const staffLeaveQueryKeys = {
  all: [...schoolAdminQueryKeys.staff(), "leave"] as const,
  types: () => [...staffLeaveQueryKeys.all, "types"] as const,
  requests: (filters:{status?:string;staffPrincipalID?:number;onDate?:string}) => [...staffLeaveQueryKeys.all,"requests",filters] as const,
  holidays: (from:string,to:string,includeInactive:boolean) => [...staffLeaveQueryKeys.all,"holidays",from,to,includeInactive] as const,
};
export function useStaffLeaveTypesQuery(enabled=true){return useQuery({queryKey:staffLeaveQueryKeys.types(),queryFn:({signal})=>getStaffLeaveTypes(signal),enabled,staleTime:15000});}
export function useStaffLeaveRequestsQuery(filters:{status?:string;staffPrincipalID?:number;onDate?:string;fromDate?:string;toDate?:string},enabled=true){return useQuery({queryKey:staffLeaveQueryKeys.requests(filters),queryFn:({signal})=>getStaffLeaveRequests(filters,signal),enabled,staleTime:10000});}
export function useSchoolHolidaysQuery(from:string,to:string,enabled=true,includeInactive=false){return useQuery({queryKey:staffLeaveQueryKeys.holidays(from,to,includeInactive),queryFn:({signal})=>getSchoolHolidays(from,to,signal,includeInactive),enabled:enabled&&Boolean(from)&&Boolean(to),staleTime:30000});}
function useStaffLeaveMutation<T>(fn:(input:T)=>Promise<unknown>){const client=useQueryClient();return useMutation({mutationFn:fn,onSuccess:()=>client.invalidateQueries({queryKey:staffLeaveQueryKeys.all})});}
export function useSaveStaffLeaveTypeMutation(){return useStaffLeaveMutation(saveStaffLeaveType);}
export function useCreateStaffLeaveRequestMutation(){return useStaffLeaveMutation(createStaffLeaveRequest);}
export function useDecideStaffLeaveRequestMutation(){return useStaffLeaveMutation((input:{requestID:number;decision:"approve"|"reject";comment?:string})=>decideStaffLeaveRequest(input.requestID,input.decision,input.comment));}
export function useCancelStaffLeaveRequestMutation(){return useStaffLeaveMutation(cancelStaffLeaveRequest);}
export function useSaveSchoolHolidayMutation(){return useStaffLeaveMutation(saveSchoolHoliday);}

export const staffSubstitutionQueryKeys = {
  all: [...schoolAdminQueryKeys.staff(), "substitutions"] as const,
  list: (filters:{fromDate:string;toDate:string;staffPrincipalID?:number}) => [...staffSubstitutionQueryKeys.all,filters] as const,
  timetableYear: () => [...staffSubstitutionQueryKeys.all,"timetable-year"] as const,
  timetableEntries: (academicYearID:number) => [...staffSubstitutionQueryKeys.all,"timetable-entries",academicYearID] as const,
};
export function useStaffSubstitutionsQuery(filters:{fromDate:string;toDate:string;staffPrincipalID?:number},enabled=true){
  return useQuery({queryKey:staffSubstitutionQueryKeys.list(filters),queryFn:({signal})=>getStaffSubstitutions(filters,signal),enabled:enabled&&Boolean(filters.fromDate)&&Boolean(filters.toDate),staleTime:10000});
}
export function useStaffSubstitutionTimetableYearQuery(enabled=true){
  return useQuery({queryKey:staffSubstitutionQueryKeys.timetableYear(),queryFn:({signal})=>getCurrentTimetableYear(signal),enabled,staleTime:60000});
}
export function useStaffSubstitutionTimetableEntriesQuery(academicYearID:number,enabled=true){
  return useQuery({queryKey:staffSubstitutionQueryKeys.timetableEntries(academicYearID),queryFn:({signal})=>getTimetableEntries(academicYearID,0,signal),enabled:enabled&&academicYearID>0,staleTime:15000});
}
function useStaffSubstitutionMutation<T>(fn:(input:T)=>Promise<unknown>){
  const client=useQueryClient();
  return useMutation({mutationFn:fn,onSuccess:()=>client.invalidateQueries({queryKey:staffSubstitutionQueryKeys.all})});
}
export function useCreateStaffSubstitutionMutation(){
  return useStaffSubstitutionMutation(createStaffSubstitution);
}
export function useCancelStaffSubstitutionMutation(){
  return useStaffSubstitutionMutation(cancelStaffSubstitution);
}
