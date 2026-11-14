import { useDeferredValue, useEffect, useMemo, useState, type FormEvent } from "react";
import { Archive, Eye, GraduationCap, Pencil, Plus, RefreshCw, Search, Users } from "lucide-react";
import {
  useArchiveSchoolStudentMutation, useCreateSchoolStudentEnrollmentMutation, useCreateSchoolStudentMutation,
  useSchoolStudentAcademicYearsQuery, useSchoolStudentEnrollmentsQuery, useSchoolStudentSectionsQuery,
  useSchoolStudentsQuery,
} from "../queries";
import type { StudentEnrollment, StudentProfile } from "../api";
import { Select } from "../../../shared/components/Select";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { StudentFieldLabel } from "../components/StudentFieldLabel";
import { StudentAddressSection } from "../components/StudentAddressSection";
import { StudentBasicDetailsSection } from "../components/StudentBasicDetailsSection";
import { StudentFamilySection } from "../components/StudentFamilySection";
import { StudentProfileDetail } from "../components/StudentProfileDetail";
import { StudentHealthSection } from "../components/StudentHealthSection";
import { StudentPromotionDialog } from "../components/StudentPromotionDialog";
import "./StudentManagementPage.css";

type Props = { permissions: string[] };
type StudentAction = "create" | null;
type Tab = "students" | "enrollments";

function localDateValue() {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
}

export function StudentManagementPage({ permissions }: Props) {
  const canReadStudents = permissions.includes("students.read_sensitive");
  const canReadEnrollments = permissions.includes("students.read_basic");
  const canReadGuardians = permissions.includes("guardians.read");
  const canManageGuardians = permissions.includes("guardians.manage");
  const canCreate = permissions.includes("students.create");
  const canUpdate = permissions.includes("students.update");
  const canArchive = permissions.includes("students.archive");
  const canReadYears = permissions.includes("academic_years.read");
  const canReadSections = permissions.includes("classes.read");
  const [tab, setTab] = useState<Tab>(canReadStudents ? "students" : "enrollments");
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search.trim());
  const [status, setStatus] = useState("");
  const [offset, setOffset] = useState(0);
  const [action, setAction] = useState<StudentAction>(null);
  const [showEnrollment, setShowEnrollment] = useState(false);
  const [addressStudent, setAddressStudent] = useState<StudentProfile | null>(null);
  const [basicDetailsStudent, setBasicDetailsStudent] = useState<StudentProfile | null>(null);
  const [familyStudent, setFamilyStudent] = useState<StudentProfile | null>(null);
  const [profileStudent, setProfileStudent] = useState<StudentProfile | null>(null);
  const [promotionEnrollment, setPromotionEnrollment] = useState<StudentEnrollment | null>(null);
  const [healthStudent, setHealthStudent] = useState<StudentProfile | null>(null);
  const [studentSearch, setStudentSearch] = useState("");
  const deferredStudentSearch = useDeferredValue(studentSearch.trim());
  const [selectedStudentID, setSelectedStudentID] = useState("");
  const [academicYearID, setAcademicYearID] = useState(0);
  const [selectedSectionID, setSelectedSectionID] = useState("");
  const [enrolledAt, setEnrolledAt] = useState(localDateValue);
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [studentToArchive, setStudentToArchive] = useState<StudentProfile | null>(null);
  const [archiveError, setArchiveError] = useState("");

  const filters = useMemo(() => ({ q: deferredSearch, status, limit: 20, offset }), [deferredSearch, status, offset]);
  const studentsQuery = useSchoolStudentsQuery(filters, canReadStudents && tab === "students");
  const studentOptionsQuery = useSchoolStudentsQuery({ q: deferredStudentSearch, status: "active", limit: 20, offset: 0 }, canReadStudents && showEnrollment);
  const enrollmentQuery = useSchoolStudentEnrollmentsQuery(academicYearID, canReadEnrollments && tab === "enrollments");
  const yearsQuery = useSchoolStudentAcademicYearsQuery(canReadYears && (canReadEnrollments || showEnrollment || action === "create"));
  const sectionsQuery = useSchoolStudentSectionsQuery(academicYearID, (showEnrollment || action === "create") && canReadSections);
  const createStudent = useCreateSchoolStudentMutation();
  const archiveStudent = useArchiveSchoolStudentMutation();
  const createEnrollment = useCreateSchoolStudentEnrollmentMutation();

  const students = studentsQuery.data?.students ?? [];
  const total = studentsQuery.data?.pagination.total ?? 0;
  const years = yearsQuery.data?.academic_years ?? [];
  const sections = (sectionsQuery.data?.sections ?? []).filter(section => section.is_active);
  const activeYears = years.filter(year => year.status === "active");
  const enrollments = enrollmentQuery.data?.enrollments ?? [];

  useEffect(() => {
    if ((action === "create" || showEnrollment) && academicYearID === 0 && activeYears.length > 0) {
      setAcademicYearID(activeYears.find(year => year.is_current)?.id ?? activeYears[0].id);
    }
  }, [action, showEnrollment, academicYearID, activeYears]);
  const busy = createStudent.isPending || archiveStudent.isPending || createEnrollment.isPending;

  function openStudent(next: StudentAction) {
    setAction(next);
    setFieldErrors({});
    setFeedback(null);
    if (next === "create") {
      setAcademicYearID(activeYears.find(year => year.is_current)?.id ?? activeYears[0]?.id ?? 0);
      setSelectedSectionID("");
      setEnrolledAt(localDateValue());
    }
  }
  async function submitStudent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const admissionNumber = String(form.get("admission_number") ?? "").trim();
    const fullName = String(form.get("full_name") ?? "").trim();
    const errors: Record<string, string> = {};
    if (action === "create" && !admissionNumber) errors.admission_number = "Enter the admission number for this academic year.";
    if (action === "create" && !academicYearID) errors.academic_year_id = "Select an active academic year.";
    if (action === "create" && !selectedSectionID) errors.section_id = "Select a section for this academic year.";
    if (action === "create" && !enrolledAt) errors.admission_date = "Choose the admission date.";
    if (!fullName) errors.full_name = "Enter the student full name.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      if (action === "create") {
        await createStudent.mutateAsync({
          admission_number: admissionNumber,
          full_name: fullName,
          academic_year_id: academicYearID,
          section_id: Number(selectedSectionID),
          roll_number: String(form.get("roll_number") ?? "").trim() || null,
          admission_date: enrolledAt,
        });
        setFeedback({ kind: "success", message: "Student admitted and enrolled for the selected academic year." });
      }
      setAction(null);
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to save the student record." });
    }
  }
  function archive(item: StudentProfile) { setArchiveError(""); setStudentToArchive(item); }
  async function confirmArchive() {
    if (!studentToArchive) return;
    const item = studentToArchive;
    setArchiveError("");
    setFeedback(null);
    try {
      await archiveStudent.mutateAsync(item.id);
      setStudentToArchive(null);
      setFeedback({ kind: "success", message: `${item.full_name} has been archived.` });
    } catch (error) {
      setArchiveError(error instanceof Error ? error.message : "Unable to archive this student. Please try again.");
    }
  }
  async function submitEnrollment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const errors: Record<string, string> = {};
    const admissionNumber = String(form.get("admission_number") ?? "").trim();
    if (!selectedStudentID) errors.student_id = "Search for and select a student.";
    if (!academicYearID) errors.academic_year_id = "Select an active academic year.";
    if (!selectedSectionID) errors.section_id = "Select a section for this academic year.";
    if (!admissionNumber) errors.admission_number = "Enter the admission number for this academic year.";
    if (!enrolledAt) errors.enrolled_at = "Choose the admission date.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      await createEnrollment.mutateAsync({
        student_id: Number(selectedStudentID),
        academic_year_id: academicYearID,
        section_id: Number(selectedSectionID),
        roll_number: String(form.get("roll_number") ?? "").trim() || null,
        admission_number: admissionNumber,
        admission_date: enrolledAt,
        enrolled_at: enrolledAt,
      });
      setShowEnrollment(false);
      setStudentSearch("");
      setSelectedStudentID("");
      setSelectedSectionID("");
      setEnrolledAt(localDateValue());
      setFeedback({ kind: "success", message: "Student enrolled successfully." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to create the enrollment." });
    }
  }
  const pageCount = Math.max(1, Math.ceil(total / 20));

  return <div className="content student-page">
    <div className="heading">
      <div><span className="eyebrow">PEOPLE & LEARNING</span><h1>Students</h1><p>Manage student records and school-year enrollments.</p></div>
      <div className="student-heading-actions">
        {canCreate && canReadStudents && canReadYears && canReadSections && tab === "students" && <button className="primary" type="button" onClick={() => openStudent("create")}><Plus size={16}/> New admission</button>}
        {canCreate && canReadStudents && canReadEnrollments && canReadYears && canReadSections && tab === "enrollments" && <button className="primary" type="button" onClick={() => {setFieldErrors({});setFeedback(null);setStudentSearch("");setSelectedStudentID("");setAcademicYearID(activeYears.find(year=>year.is_current)?.id??activeYears[0]?.id??0);setSelectedSectionID("");setEnrolledAt(localDateValue());setShowEnrollment(true);}}><Plus size={16}/> Enroll student</button>}
      </div>
    </div>
    <div className="student-tabs" role="tablist" aria-label="Student management">
      {canReadStudents && <button type="button" role="tab" aria-selected={tab === "students"} className={tab === "students" ? "selected" : ""} onClick={() => { setTab("students"); setShowEnrollment(false); setFeedback(null); }}>Student directory</button>}
      {canReadEnrollments && <button type="button" role="tab" aria-selected={tab === "enrollments"} className={tab === "enrollments" ? "selected" : ""} onClick={() => { setTab("enrollments"); setAction(null); setFeedback(null); }}>Enrollments</button>}
    </div>
    {feedback && !((action!==null||showEnrollment)&&feedback.kind==="error") && <div className={"student-feedback " + feedback.kind} role="status">{feedback.message}</div>}
    <Modal open={action !== null} title="New student admission" description="Create the permanent student record and first academic-year enrollment together." onClose={() => openStudent(null)} closeDisabled={busy}>
      {feedback?.kind === "error" && <p className="student-feedback error" role="alert">{feedback.message}</p>}
      <form onSubmit={submitStudent} noValidate>
        <label><StudentFieldLabel label="Student full name" required hint="Enter the student’s full name as it should appear in school records. Maximum 200 characters." /><input name="full_name" placeholder="e.g. Aarav Sharma" defaultValue="" maxLength={200} autoComplete="name" required aria-invalid={Boolean(fieldErrors.full_name)} aria-describedby={fieldErrors.full_name ? "student-name-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,full_name:""}))}/>{fieldErrors.full_name && <small className="app-field-error" id="student-name-error">{fieldErrors.full_name}</small>}</label>
        {action === "create" && <>
          <label><StudentFieldLabel label="Admission number" required hint="Enter the admission number for this academic year. Maximum 50 characters; unique within this school and academic year." /><input name="admission_number" placeholder="e.g. ADM-2026-001" maxLength={50} autoComplete="off" required aria-invalid={Boolean(fieldErrors.admission_number)} aria-describedby={fieldErrors.admission_number ? "student-admission-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,admission_number:""}))}/>{fieldErrors.admission_number && <small className="app-field-error" id="student-admission-error">{fieldErrors.admission_number}</small>}</label>
          <label><StudentFieldLabel label="Academic year" required hint="Choose an active academic year. The selected section must belong to this year." /><Select ariaLabel="Admission academic year" invalid={Boolean(fieldErrors.academic_year_id)} describedBy={fieldErrors.academic_year_id?"new-admission-year-error":undefined} value={String(academicYearID||"")} onChange={value=>{setAcademicYearID(Number(value));setSelectedSectionID("");setFieldErrors(current=>({...current,academic_year_id:"",section_id:""}));}} placeholder="Select active year" options={activeYears.map(year=>({value:String(year.id),label:year.name+(year.is_current?" · Current":"")}))}/>{fieldErrors.academic_year_id&&<small className="app-field-error" id="new-admission-year-error">{fieldErrors.academic_year_id}</small>}{activeYears.length===0&&<small>No active academic year is available. Activate an academic year before admitting students.</small>}</label>
          <label><StudentFieldLabel label="Class / section" required hint="Choose an active section within the selected academic year." /><Select ariaLabel="Admission section" invalid={Boolean(fieldErrors.section_id)} describedBy={fieldErrors.section_id?"new-admission-section-error":undefined} value={selectedSectionID} onChange={value=>{setSelectedSectionID(value);setFieldErrors(current=>({...current,section_id:""}));}} placeholder="Select section" disabled={!academicYearID||sectionsQuery.isLoading||sections.length===0} options={sections.map(section=>({value:String(section.id),label:section.class_name+" · "+section.name}))}/>{fieldErrors.section_id&&<small className="app-field-error" id="new-admission-section-error">{fieldErrors.section_id}</small>}{academicYearID>0&&!sectionsQuery.isLoading&&!sectionsQuery.isError&&sections.length===0&&<small>No active sections are available for this academic year. Create a class section first.</small>}</label>
          <label><StudentFieldLabel label="Roll number" hint="Optional. Leave blank to assign the next available roll number automatically. You may enter a manual number if authorized." /><input name="roll_number" placeholder="e.g. 01" maxLength={30}/><small>Leave blank to assign the next available roll number automatically. School Admins can enter a manual number.</small></label>
          <label><StudentFieldLabel label="Admission date" required hint="Choose the date the student is admitted/enrolled. It must satisfy the selected academic-year rules." /><DatePicker ariaLabel="Admission date" placeholder="Choose the admission date" invalid={Boolean(fieldErrors.admission_date)} describedBy={fieldErrors.admission_date?"new-admission-date-error":undefined} value={enrolledAt} onChange={value=>{setEnrolledAt(value);setFieldErrors(current=>({...current,admission_date:""}));}}/>{fieldErrors.admission_date&&<small className="app-field-error" id="new-admission-date-error">{fieldErrors.admission_date}</small>}</label>
          {sectionsQuery.isError && <div className="student-feedback error" role="alert">{sectionsQuery.error.message}</div>}
        </>}
        <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => openStudent(null)} disabled={busy}>Cancel</button><button className="primary" disabled={busy || (action === "create" && (yearsQuery.isLoading || sectionsQuery.isLoading || sectionsQuery.isError || activeYears.length === 0 || sections.length === 0))}>{busy ? "Saving…" : "Admit student"}</button></div>
      </form>
    </Modal>
    <Modal open={showEnrollment} title="Enroll student" description="Select an active student, academic year and section. The section must belong to the selected academic year." onClose={() => setShowEnrollment(false)} closeDisabled={busy}>
      {feedback?.kind === "error" && <p className="student-feedback error" role="alert">{feedback.message}</p>}
      {!canReadStudents ? <div className="student-feedback error">Student directory permission is required to select a student for enrollment.</div> :
      <form onSubmit={submitEnrollment} noValidate>
        <label><StudentFieldLabel label="Find student" hint="Search by student name or admission number, then select the matching active student below." /><input type="search" value={studentSearch} onChange={event => { setStudentSearch(event.target.value); setSelectedStudentID(""); setFieldErrors(current => ({...current, student_id: ""})); }} placeholder="Search name or admission number" autoComplete="off" aria-label="Search active students by name or admission number"/></label>
        {studentOptionsQuery.isLoading ? <div className="student-state">Searching students…</div> : studentOptionsQuery.isError ? <div className="student-feedback error" role="alert">{studentOptionsQuery.error.message}<button type="button" className="student-secondary" onClick={() => void studentOptionsQuery.refetch()}>Try again</button></div> : <label><StudentFieldLabel label="Student" required hint="Select the existing student record. Do not create a duplicate record for a returning student." /><Select ariaLabel="Student" invalid={Boolean(fieldErrors.student_id)} describedBy={fieldErrors.student_id?"enrollment-student-error":undefined} value={selectedStudentID} onChange={value => {setSelectedStudentID(value);setFieldErrors(current=>({...current,student_id:""}));}} placeholder={deferredStudentSearch ? "Select a matching student" : "Search to find a student"} disabled={(studentOptionsQuery.data?.students.length ?? 0) === 0} options={(studentOptionsQuery.data?.students ?? []).map(item=>({value:String(item.id),label:`${item.full_name} · ${item.admission_number}`}))}/><small>{(studentOptionsQuery.data?.pagination.total ?? 0) === 0 ? "No active students match this search. Create a student record first, then return here to enroll them." : `${studentOptionsQuery.data?.pagination.total ?? 0} matching active students. Refine your search if needed.`}</small>{fieldErrors.student_id && <small className="app-field-error" id="enrollment-student-error" role="alert">{fieldErrors.student_id}</small>}</label>}
        <label><StudentFieldLabel label="Academic year" required hint="Choose an active academic year. The selected section must belong to this year." /><Select ariaLabel="Enrollment academic year" invalid={Boolean(fieldErrors.academic_year_id)} describedBy={fieldErrors.academic_year_id?"enrollment-year-error":undefined} value={String(academicYearID||"")} onChange={value=>{setAcademicYearID(Number(value));setSelectedSectionID("");setFieldErrors(current=>({...current,academic_year_id:"",section_id:""}));}} placeholder="Select active year" options={activeYears.map(year=>({value:String(year.id),label:year.name+(year.is_current?" · Current":"")}))}/>{fieldErrors.academic_year_id&&<small className="app-field-error" id="enrollment-year-error" role="alert">{fieldErrors.academic_year_id}</small>}{activeYears.length===0&&<small>No active academic year is available. Ask a School Admin to activate an academic year before enrolling students.</small>}</label>
        <label><StudentFieldLabel label="Section" required hint="Choose an active section within the selected academic year." /><Select ariaLabel="Enrollment section" invalid={Boolean(fieldErrors.section_id)} describedBy={fieldErrors.section_id?"enrollment-section-error":undefined} value={selectedSectionID} onChange={value=>{setSelectedSectionID(value);setFieldErrors(current=>({...current,section_id:""}));}} placeholder="Select section" disabled={!academicYearID||sectionsQuery.isLoading||sections.length===0} options={sections.map(section=>({value:String(section.id),label:section.name}))}/>{fieldErrors.section_id&&<small className="app-field-error" id="enrollment-section-error" role="alert">{fieldErrors.section_id}</small>}{academicYearID>0&&!sectionsQuery.isLoading&&!sectionsQuery.isError&&sections.length===0&&<small>No active sections are available for this academic year. Create or activate a section first.</small>}</label>
        <label><StudentFieldLabel label="Admission number" required hint="Enter the admission number for this academic year. Maximum 50 characters; unique within this school and academic year." /><input name="admission_number" placeholder="e.g. ADM-2026-001" maxLength={50} required aria-invalid={Boolean(fieldErrors.admission_number)} aria-describedby={fieldErrors.admission_number?"enrollment-admission-error":undefined} onChange={()=>setFieldErrors(current=>({...current,admission_number:""}))}/>{fieldErrors.admission_number&&<small className="app-field-error" id="enrollment-admission-error">{fieldErrors.admission_number}</small>}</label>
        <label><StudentFieldLabel label="Roll number" hint="Optional. Leave blank to assign the next available roll number automatically. You may enter a manual number if authorized." /><input name="roll_number" placeholder="e.g. 01" maxLength={30}/><small>Leave blank to assign the next available roll number automatically. School Admins can enter a manual number.</small></label>
        <label><StudentFieldLabel label="Admission date" required hint="Choose the date the student is admitted/enrolled. It must satisfy the selected academic-year rules." /><DatePicker ariaLabel="Admission date" placeholder="Choose the admission date" invalid={Boolean(fieldErrors.enrolled_at)} describedBy={fieldErrors.enrolled_at?"enrollment-date-error":undefined} value={enrolledAt} onChange={value=>{setEnrolledAt(value);setFieldErrors(current=>({...current,enrolled_at:""}));}}/>{fieldErrors.enrolled_at&&<small className="app-field-error" id="enrollment-date-error">{fieldErrors.enrolled_at}</small>}</label>
        {sectionsQuery.isError && <div className="student-feedback error" role="alert">{sectionsQuery.error.message}</div>}
        <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => setShowEnrollment(false)} disabled={busy}>Cancel</button><button className="primary" disabled={busy || studentOptionsQuery.isLoading || studentOptionsQuery.isError || activeYears.length === 0 || sectionsQuery.isLoading || sectionsQuery.isError}>{busy ? "Enrolling…" : "Create enrollment"}</button></div>
      </form>}
    </Modal>
    <ConfirmDialog open={Boolean(studentToArchive)} title="Archive student?" description={studentToArchive ? `Archive ${studentToArchive.full_name}? The record will be retained, but it cannot be used for a new enrollment.` : ""} confirmLabel="Archive student" pending={archiveStudent.isPending} error={archiveError || undefined} onCancel={() => {setStudentToArchive(null);setArchiveError("");}} onConfirm={() => void confirmArchive()}/>
    {tab === "students" && canReadStudents && <>
      <div className="student-toolbar">
        <label className="student-search"><Search size={16}/><input aria-label="Search students" placeholder="Search name or admission number" value={search} onChange={event => { setSearch(event.target.value); setOffset(0); }}/></label>
        <Select ariaLabel="Filter by student status" value={status} onChange={value => { setStatus(value); setOffset(0); }} placeholder="All statuses" options={[{value:"",label:"All statuses"},{value:"active",label:"Active"},{value:"inactive",label:"Inactive"},{value:"archived",label:"Archived"}]} />
        <button type="button" className="student-secondary" onClick={() => void studentsQuery.refetch()} disabled={studentsQuery.isFetching}><RefreshCw size={15}/> Refresh</button>
      </div>
      {studentsQuery.isLoading ? <div className="student-state">Loading student records…</div> :
       studentsQuery.isError ? <div className="student-state error" role="alert"><p>{studentsQuery.error.message}</p><button className="student-secondary" onClick={() => void studentsQuery.refetch()}>Try again</button></div> :
       students.length === 0 ? <div className="student-state"><Users size={28}/><b>No students found</b><span>{search || status ? "Try changing the search or status filter." : "Add the first student record to get started."}</span></div> :
       <section className="student-table-wrap"><table className="student-table"><thead><tr><th>Student</th><th>Admission no.</th><th>Status</th><th>Actions</th></tr></thead><tbody>
        {students.map(item => <tr key={item.id}><td data-label="Student"><b>{item.full_name}</b><small>Student ID #{item.id}</small></td><td data-label="Admission no.">{item.admission_number}</td><td data-label="Status"><span className={"student-status " + item.status}>{item.status}</span></td><td data-label="Actions"><div className="student-row-actions">{canReadStudents && <button type="button" title="View student profile" aria-label={`View profile for ${item.full_name}`} onClick={() => setProfileStudent(item)}><Eye size={15}/></button>}{canUpdate && item.status !== "archived" && <button type="button" title="Edit student profile sections" aria-label={`Edit profile for ${item.full_name}`} onClick={() => setProfileStudent(item)}><Pencil size={15}/></button>}{canArchive && item.status !== "archived" && <button type="button" className="archive" title="Archive student" aria-label={`Archive ${item.full_name}`} onClick={() => void archive(item)}><Archive size={15}/></button>}</div></td></tr>)}
       </tbody></table></section>}
      {total > 0 && <div className="student-pagination"><span>{`Showing ${offset + 1}–${Math.min(offset + 20, total)} of ${total}`}</span>{pageCount > 1 && <div><button type="button" className="student-secondary" disabled={offset === 0 || studentsQuery.isFetching} onClick={() => setOffset(Math.max(0, offset - 20))}>Previous</button><span>Page {Math.floor(offset / 20) + 1} of {pageCount}</span><button type="button" className="student-secondary" disabled={offset + 20 >= total || studentsQuery.isFetching} onClick={() => setOffset(offset + 20)}>Next</button></div>}</div>}
    </>}
    {tab === "enrollments" && canReadEnrollments && <>
      <div className="student-toolbar"><label><StudentFieldLabel label="Academic year" hint="Optional filter. Choose an academic year to narrow the directory, or leave “All academic years” selected." /><Select ariaLabel="Filter enrollments by academic year" value={String(academicYearID)} onChange={value=>setAcademicYearID(Number(value))} options={[{value:"0",label:"All academic years"},...years.map(year=>({value:String(year.id),label:year.name}))]}/></label><button type="button" className="student-secondary" onClick={() => void enrollmentQuery.refetch()} disabled={enrollmentQuery.isFetching}><RefreshCw size={15}/> Refresh</button></div>
      {enrollmentQuery.isLoading ? <div className="student-state">Loading enrollments…</div> :
       enrollmentQuery.isError ? <div className="student-state error" role="alert"><p>{enrollmentQuery.error.message}</p><button className="student-secondary" onClick={() => void enrollmentQuery.refetch()}>Try again</button></div> :
       enrollments.length === 0 ? <div className="student-state"><GraduationCap size={28}/><b>No enrollments found</b><span>Enrollments appear here after students are assigned to a class section.</span></div> :
       <section className="student-table-wrap"><table className="student-table"><thead><tr><th>Student</th><th>Admission no.</th><th>Class / section</th><th>Roll number</th><th>Status</th><th>Enrolled on</th><th>Actions</th></tr></thead><tbody>
        {enrollments.map(item => <tr key={item.id}><td data-label="Student"><b>{item.student_name || (item.student_id ? `Student #${item.student_id}` : "Assigned student")}</b></td><td data-label="Admission no.">{item.admission_number || "—"}</td><td data-label="Class / section">{item.class_name_snapshot} · {item.section_name_snapshot}</td><td data-label="Roll number">{item.roll_number || "—"}</td><td data-label="Status"><span className={"student-status " + item.status}>{item.status}</span></td><td data-label="Enrolled on">{new Intl.DateTimeFormat("en-IN",{dateStyle:"medium"}).format(new Date(item.enrolled_at))}</td><td data-label="Actions">{canUpdate && canReadYears && canReadSections && item.status === "active" && item.student_id && item.academic_year_id ? <button type="button" className="student-secondary" onClick={() => setPromotionEnrollment(item)}>Promote</button> : "—"}</td></tr>)}
       </tbody></table></section>}
    </>}
    {profileStudent && <StudentProfileDetail
      student={profileStudent}
      canReadHistory={canReadEnrollments}
      canReadFamily={canReadGuardians}
      academicYears={yearsQuery.data?.academic_years ?? []}
      onOpenBasic={() => { const item = profileStudent; setProfileStudent(null); setBasicDetailsStudent(item); }}
      onOpenAddress={() => { const item = profileStudent; setProfileStudent(null); setAddressStudent(item); }}
      onOpenFamily={() => { const item = profileStudent; setProfileStudent(null); setFamilyStudent(item); }}
      onOpenHealth={() => { const item = profileStudent; setProfileStudent(null); setHealthStudent(item); }}
      onClose={() => setProfileStudent(null)}
    />}
    {basicDetailsStudent && <StudentBasicDetailsSection student={basicDetailsStudent} canUpdate={canUpdate && basicDetailsStudent.status !== "archived"} onClose={() => setBasicDetailsStudent(null)} />}
    {addressStudent && <StudentAddressSection student={addressStudent} canUpdate={canUpdate && addressStudent.status !== "archived"} onClose={() => setAddressStudent(null)} />}
    {familyStudent && <StudentFamilySection student={familyStudent} canManage={canManageGuardians && familyStudent.status !== "archived"} canReadRelatedStudents={canReadEnrollments} onClose={() => setFamilyStudent(null)} />}
    {healthStudent && <StudentHealthSection student={healthStudent} canUpdate={canUpdate && healthStudent.status === "active"} onClose={() => setHealthStudent(null)} />}
    {promotionEnrollment && <StudentPromotionDialog
      enrollment={promotionEnrollment}
      academicYears={years}
      onClose={() => setPromotionEnrollment(null)}
      onPromoted={() => { setPromotionEnrollment(null); setFeedback({ kind: "success", message: "Student promoted successfully. Previous academic history has been preserved." }); }}
    />}
  </div>;
}
