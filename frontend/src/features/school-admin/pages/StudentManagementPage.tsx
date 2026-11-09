import { useDeferredValue, useMemo, useState, type FormEvent } from "react";
import { Archive, GraduationCap, Pencil, Plus, RefreshCw, Search, Users } from "lucide-react";
import {
  useArchiveSchoolStudentMutation, useCreateSchoolStudentEnrollmentMutation, useCreateSchoolStudentMutation,
  useSchoolStudentAcademicYearsQuery, useSchoolStudentEnrollmentsQuery, useSchoolStudentSectionsQuery,
  useSchoolStudentsQuery, useUpdateSchoolStudentMutation,
} from "../queries";
import type { StudentProfile } from "../api";
import { Select } from "../../../shared/components/Select";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import "./StudentManagementPage.css";

type Props = { permissions: string[] };
type StudentAction = "create" | "edit" | null;
type Tab = "students" | "enrollments";

export function StudentManagementPage({ permissions }: Props) {
  const canReadStudents = permissions.includes("students.read_sensitive");
  const canReadEnrollments = permissions.includes("students.read_basic");
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
  const [selected, setSelected] = useState<StudentProfile | null>(null);
  const [showEnrollment, setShowEnrollment] = useState(false);
  const [studentSearch, setStudentSearch] = useState("");
  const deferredStudentSearch = useDeferredValue(studentSearch.trim());
  const [selectedStudentID, setSelectedStudentID] = useState("");
  const [academicYearID, setAcademicYearID] = useState(0);
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [studentToArchive, setStudentToArchive] = useState<StudentProfile | null>(null);

  const filters = useMemo(() => ({ q: deferredSearch, status, limit: 20, offset }), [deferredSearch, status, offset]);
  const studentsQuery = useSchoolStudentsQuery(filters, canReadStudents && tab === "students");
  const studentOptionsQuery = useSchoolStudentsQuery({ q: deferredStudentSearch, status: "active", limit: 20, offset: 0 }, canReadStudents && showEnrollment);
  const enrollmentQuery = useSchoolStudentEnrollmentsQuery(academicYearID, canReadEnrollments && tab === "enrollments");
  const yearsQuery = useSchoolStudentAcademicYearsQuery(canReadYears && (canReadEnrollments || showEnrollment));
  const sectionsQuery = useSchoolStudentSectionsQuery(academicYearID, showEnrollment && canReadSections);
  const createStudent = useCreateSchoolStudentMutation();
  const updateStudent = useUpdateSchoolStudentMutation();
  const archiveStudent = useArchiveSchoolStudentMutation();
  const createEnrollment = useCreateSchoolStudentEnrollmentMutation();

  const students = studentsQuery.data?.students ?? [];
  const total = studentsQuery.data?.pagination.total ?? 0;
  const years = yearsQuery.data?.academic_years ?? [];
  const sections = (sectionsQuery.data?.sections ?? []).filter(section => section.is_active);
  const activeYears = years.filter(year => year.status === "active");
  const enrollments = enrollmentQuery.data?.enrollments ?? [];
  const busy = createStudent.isPending || updateStudent.isPending || archiveStudent.isPending || createEnrollment.isPending;

  function openStudent(next: StudentAction, student: StudentProfile | null = null) {
    setSelected(student);
    setAction(next);
    setFieldErrors({});
    setFeedback(null);
  }
  async function submitStudent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const admissionNumber = String(form.get("admission_number") ?? "").trim();
    const fullName = String(form.get("full_name") ?? "").trim();
    const errors: Record<string, string> = {};
    if (action === "create" && !admissionNumber) errors.admission_number = "Enter the student admission number.";
    if (!fullName) errors.full_name = "Enter the student full name.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      if (action === "create") {
        await createStudent.mutateAsync({
          admission_number: admissionNumber,
          full_name: fullName,
        });
        setFeedback({ kind: "success", message: "Student record created. Enroll the student into an academic year and section next." });
      } else if (action === "edit" && selected) {
        await updateStudent.mutateAsync({ studentID: selected.id, fullName });
        setFeedback({ kind: "success", message: "Student details updated." });
      }
      setAction(null);
      setSelected(null);
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to save the student record." });
    }
  }
  function archive(item: StudentProfile) { setStudentToArchive(item); }
  async function confirmArchive() {
    if (!studentToArchive) return;
    const item = studentToArchive;
    setFeedback(null);
    try {
      await archiveStudent.mutateAsync(item.id);
      setStudentToArchive(null);
      setFeedback({ kind: "success", message: `${item.full_name} has been archived.` });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to archive this student." });
    }
  }
  async function submitEnrollment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const errors: Record<string, string> = {};
    if (!selectedStudentID) errors.student_id = "Search for and select a student.";
    if (!academicYearID) errors.academic_year_id = "Select an active academic year.";
    if (!String(form.get("section_id") ?? "")) errors.section_id = "Select a section for this academic year.";
    if (!String(form.get("enrolled_at") ?? "")) errors.enrolled_at = "Choose the enrollment date.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      await createEnrollment.mutateAsync({
        student_id: Number(form.get("student_id")),
        academic_year_id: Number(form.get("academic_year_id")),
        section_id: Number(form.get("section_id")),
        roll_number: String(form.get("roll_number") ?? "").trim() || null,
        enrolled_at: String(form.get("enrolled_at") ?? ""),
      });
      setShowEnrollment(false);
      setStudentSearch("");
      setSelectedStudentID("");
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
        {canCreate && canReadStudents && tab === "students" && <button className="primary" type="button" onClick={() => openStudent("create")}><Plus size={16}/> Add student</button>}
        {canCreate && canReadStudents && canReadEnrollments && canReadYears && canReadSections && tab === "enrollments" && <button className="primary" type="button" onClick={() => setShowEnrollment(true)}><Plus size={16}/> Enroll student</button>}
      </div>
    </div>
    <div className="student-tabs" role="tablist" aria-label="Student management">
      {canReadStudents && <button type="button" role="tab" aria-selected={tab === "students"} className={tab === "students" ? "selected" : ""} onClick={() => { setTab("students"); setShowEnrollment(false); setFeedback(null); }}>Student directory</button>}
      {canReadEnrollments && <button type="button" role="tab" aria-selected={tab === "enrollments"} className={tab === "enrollments" ? "selected" : ""} onClick={() => { setTab("enrollments"); setAction(null); setFeedback(null); }}>Enrollments</button>}
    </div>
    {feedback && <div className={"student-feedback " + feedback.kind} role="status">{feedback.message}</div>}
    <Modal open={action !== null} title={action === "create" ? "Add student" : "Edit student"} description="Enter the student's official details. You can enroll the student into a class section after creating the record." onClose={() => openStudent(null)} closeDisabled={busy}>
      {feedback?.kind === "error" && <p className="student-feedback error" role="alert">{feedback.message}</p>}
      <form onSubmit={submitStudent} noValidate>
        {action === "create" && <label>Admission number<input name="admission_number" maxLength={50} autoComplete="off" required aria-invalid={Boolean(fieldErrors.admission_number)} aria-describedby={fieldErrors.admission_number ? "student-admission-error" : undefined}/>{fieldErrors.admission_number && <small className="app-field-error" id="student-admission-error">{fieldErrors.admission_number}</small>}</label>}
        <label>Student full name<input name="full_name" defaultValue={selected?.full_name ?? ""} maxLength={200} autoComplete="name" required aria-invalid={Boolean(fieldErrors.full_name)} aria-describedby={fieldErrors.full_name ? "student-name-error" : undefined}/>{fieldErrors.full_name && <small className="app-field-error" id="student-name-error">{fieldErrors.full_name}</small>}</label>
        <div className="app-modal-actions"><button type="button" className="student-secondary" onClick={() => openStudent(null)} disabled={busy}>Cancel</button><button className="primary" disabled={busy}>{busy ? "Saving…" : "Save student"}</button></div>
      </form>
    </Modal>
    <Modal open={showEnrollment} title="Enroll student" description="Select an active student, academic year and section. The section must belong to the selected academic year." onClose={() => setShowEnrollment(false)} closeDisabled={busy}>
      {feedback?.kind === "error" && <p className="student-feedback error" role="alert">{feedback.message}</p>}
      {!canReadStudents ? <div className="student-feedback error">Student directory permission is required to select a student for enrollment.</div> :
      <form onSubmit={submitEnrollment} noValidate>
        <label>Find student<input type="search" value={studentSearch} onChange={event => { setStudentSearch(event.target.value); setSelectedStudentID(""); setFieldErrors(current => ({...current, student_id: ""})); }} placeholder="Search name or admission number" autoComplete="off" aria-label="Search active students by name or admission number"/></label>
        {studentOptionsQuery.isLoading ? <div className="student-state">Searching students…</div> : studentOptionsQuery.isError ? <div className="student-feedback error" role="alert">{studentOptionsQuery.error.message}<button type="button" className="student-secondary" onClick={() => void studentOptionsQuery.refetch()}>Try again</button></div> : <label>Student<select name="student_id" value={selectedStudentID} onChange={event => {setSelectedStudentID(event.target.value);setFieldErrors(current=>({...current,student_id:""}));}} required disabled={(studentOptionsQuery.data?.students.length ?? 0) === 0} aria-invalid={Boolean(fieldErrors.student_id)}><option value="" disabled>{deferredStudentSearch ? "Select a matching student" : "Search to find a student"}</option>{(studentOptionsQuery.data?.students ?? []).map(item => <option key={item.id} value={item.id}>{item.full_name} · {item.admission_number}</option>)}</select><small>{studentOptionsQuery.data?.pagination.total ?? 0} matching active students. Refine your search if needed.</small>{fieldErrors.student_id && <small className="app-field-error">{fieldErrors.student_id}</small>}</label>}
        <label>Academic year<select name="academic_year_id" value={academicYearID || ""} onChange={event => {setAcademicYearID(Number(event.target.value));setFieldErrors(current=>({...current,academic_year_id:"",section_id:""}));}} required aria-invalid={Boolean(fieldErrors.academic_year_id)}><option value="" disabled>Select active year</option>{activeYears.map(year => <option key={year.id} value={year.id}>{year.name}{year.is_current ? " · Current" : ""}</option>)}</select>{fieldErrors.academic_year_id && <small className="app-field-error">{fieldErrors.academic_year_id}</small>}</label>
        <label>Section<select name="section_id" defaultValue="" required disabled={!academicYearID || sectionsQuery.isLoading} aria-invalid={Boolean(fieldErrors.section_id)} onChange={() => setFieldErrors(current=>({...current,section_id:""}))}><option value="" disabled>Select section</option>{sections.map(section => <option key={section.id} value={section.id}>{section.name}</option>)}</select>{fieldErrors.section_id && <small className="app-field-error">{fieldErrors.section_id}</small>}</label>
        <label>Roll number<input name="roll_number" maxLength={30}/><small>Optional. Leave blank if roll numbers have not been assigned yet.</small></label>
        <label>Enrollment date<input name="enrolled_at" type="date" defaultValue={new Date().toISOString().slice(0, 10)} required aria-invalid={Boolean(fieldErrors.enrolled_at)}/>{fieldErrors.enrolled_at && <small className="app-field-error">{fieldErrors.enrolled_at}</small>}</label>
        {sectionsQuery.isError && <div className="student-feedback error" role="alert">{sectionsQuery.error.message}</div>}
        <div className="app-modal-actions"><button type="button" className="student-secondary" onClick={() => setShowEnrollment(false)} disabled={busy}>Cancel</button><button className="primary" disabled={busy || !selectedStudentID || studentOptionsQuery.isLoading || studentOptionsQuery.isError || activeYears.length === 0 || sections.length === 0}>{busy ? "Enrolling…" : "Create enrollment"}</button></div>
      </form>}
    </Modal>
    <ConfirmDialog open={Boolean(studentToArchive)} title="Archive student?" description={studentToArchive ? `Archive ${studentToArchive.full_name}? The record will be retained, but it cannot be used for a new enrollment.` : ""} confirmLabel="Archive student" pending={archiveStudent.isPending} onCancel={() => setStudentToArchive(null)} onConfirm={() => void confirmArchive()}/>
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
        {students.map(item => <tr key={item.id}><td data-label="Student"><b>{item.full_name}</b><small>Student ID #{item.id}</small></td><td data-label="Admission no.">{item.admission_number}</td><td data-label="Status"><span className={"student-status " + item.status}>{item.status}</span></td><td data-label="Actions"><div className="student-row-actions">{canUpdate && item.status !== "archived" && <button type="button" title="Edit student" aria-label={`Edit ${item.full_name}`} onClick={() => openStudent("edit", item)}><Pencil size={15}/></button>}{canArchive && item.status !== "archived" && <button type="button" className="archive" title="Archive student" aria-label={`Archive ${item.full_name}`} onClick={() => void archive(item)}><Archive size={15}/></button>}</div></td></tr>)}
       </tbody></table></section>}
      {total > 0 && <div className="student-pagination"><span>{`Showing ${offset + 1}–${Math.min(offset + 20, total)} of ${total}`}</span>{pageCount > 1 && <div><button type="button" className="student-secondary" disabled={offset === 0 || studentsQuery.isFetching} onClick={() => setOffset(Math.max(0, offset - 20))}>Previous</button><span>Page {Math.floor(offset / 20) + 1} of {pageCount}</span><button type="button" className="student-secondary" disabled={offset + 20 >= total || studentsQuery.isFetching} onClick={() => setOffset(offset + 20)}>Next</button></div>}</div>}
    </>}
    {tab === "enrollments" && canReadEnrollments && <>
      <div className="student-toolbar"><label>Academic year<select value={academicYearID} onChange={event => setAcademicYearID(Number(event.target.value))}><option value={0}>All academic years</option>{years.map(year => <option key={year.id} value={year.id}>{year.name}</option>)}</select></label><button type="button" className="student-secondary" onClick={() => void enrollmentQuery.refetch()} disabled={enrollmentQuery.isFetching}><RefreshCw size={15}/> Refresh</button></div>
      {enrollmentQuery.isLoading ? <div className="student-state">Loading enrollments…</div> :
       enrollmentQuery.isError ? <div className="student-state error" role="alert"><p>{enrollmentQuery.error.message}</p><button className="student-secondary" onClick={() => void enrollmentQuery.refetch()}>Try again</button></div> :
       enrollments.length === 0 ? <div className="student-state"><GraduationCap size={28}/><b>No enrollments found</b><span>Enrollments appear here after students are assigned to a class section.</span></div> :
       <section className="student-table-wrap"><table className="student-table"><thead><tr><th>Student</th><th>Class / section</th><th>Roll number</th><th>Status</th><th>Enrolled on</th></tr></thead><tbody>
        {enrollments.map(item => <tr key={item.id}><td data-label="Student"><b>{item.student_name || (item.student_id ? `Student #${item.student_id}` : "Assigned student")}</b></td><td data-label="Class / section">{item.class_name_snapshot} · {item.section_name_snapshot}</td><td data-label="Roll number">{item.roll_number || "—"}</td><td data-label="Status"><span className={"student-status " + item.status}>{item.status}</span></td><td data-label="Enrolled on">{new Intl.DateTimeFormat("en-IN",{dateStyle:"medium"}).format(new Date(item.enrolled_at))}</td></tr>)}
       </tbody></table></section>}
    </>}
  </div>;
}
