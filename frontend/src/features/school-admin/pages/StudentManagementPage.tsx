import { useDeferredValue, useMemo, useState, type FormEvent } from "react";
import { Archive, GraduationCap, Pencil, Plus, RefreshCw, Search, Users, X } from "lucide-react";
import {
  useArchiveSchoolStudentMutation, useCreateSchoolStudentEnrollmentMutation, useCreateSchoolStudentMutation,
  useSchoolStudentAcademicYearsQuery, useSchoolStudentEnrollmentsQuery, useSchoolStudentSectionsQuery,
  useSchoolStudentsQuery, useUpdateSchoolStudentMutation,
} from "../queries";
import type { StudentProfile } from "../api";
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
    setFeedback(null);
  }
  async function submitStudent(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFeedback(null);
    try {
      if (action === "create") {
        await createStudent.mutateAsync({
          admission_number: String(form.get("admission_number") ?? "").trim(),
          full_name: String(form.get("full_name") ?? "").trim(),
        });
        setFeedback({ kind: "success", message: "Student record created. Enroll the student into an academic year and section next." });
      } else if (action === "edit" && selected) {
        await updateStudent.mutateAsync({ studentID: selected.id, fullName: String(form.get("full_name") ?? "").trim() });
        setFeedback({ kind: "success", message: "Student details updated." });
      }
      setAction(null);
      setSelected(null);
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to save the student record." });
    }
  }
  async function archive(item: StudentProfile) {
    if (!window.confirm(`Archive student "${item.full_name}"? The student record will be retained and cannot be used for a new enrollment.`)) return;
    setFeedback(null);
    try {
      await archiveStudent.mutateAsync(item.id);
      setFeedback({ kind: "success", message: `${item.full_name} has been archived.` });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to archive this student." });
    }
  }
  async function submitEnrollment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
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
    {action && <section className="student-form-panel">
      <div className="student-form-heading"><div><h2>{action === "create" ? "Add student" : "Edit student"}</h2><p>Only the approved student identity fields are collected in this workflow.</p></div><button type="button" onClick={() => openStudent(null)} aria-label="Close form"><X size={18}/></button></div>
      <form onSubmit={submitStudent}>
        {action === "create" && <label>Admission number<input name="admission_number" maxLength={50} autoComplete="off" required/></label>}
        <label>Student full name<input name="full_name" defaultValue={selected?.full_name ?? ""} maxLength={200} autoComplete="name" required/></label>
        <div className="student-form-actions"><button type="button" className="student-secondary" onClick={() => openStudent(null)}>Cancel</button><button className="primary" disabled={busy}>{busy ? "Saving…" : "Save student"}</button></div>
      </form>
    </section>}
    {showEnrollment && <section className="student-form-panel">
      <div className="student-form-heading"><div><h2>Enroll student</h2><p>Choose an active academic year and a section belonging to that year.</p></div><button type="button" onClick={() => setShowEnrollment(false)} aria-label="Close form"><X size={18}/></button></div>
      {!canReadStudents ? <div className="student-feedback error">Student directory permission is required to select a student for enrollment.</div> :
      <form onSubmit={submitEnrollment}>
        <label>Find student<input type="search" value={studentSearch} onChange={event => { setStudentSearch(event.target.value); setSelectedStudentID(""); }} placeholder="Search name or admission number" autoComplete="off" aria-label="Search active students by name or admission number"/></label>
        {studentOptionsQuery.isLoading ? <div className="student-state">Searching students…</div> : studentOptionsQuery.isError ? <div className="student-feedback error" role="alert">{studentOptionsQuery.error.message}<button type="button" className="student-secondary" onClick={() => void studentOptionsQuery.refetch()}>Try again</button></div> : <label>Student<select name="student_id" value={selectedStudentID} onChange={event => setSelectedStudentID(event.target.value)} required disabled={(studentOptionsQuery.data?.students.length ?? 0) === 0}><option value="" disabled>{deferredStudentSearch ? "Select a matching student" : "Search to find a student"}</option>{(studentOptionsQuery.data?.students ?? []).map(item => <option key={item.id} value={item.id}>{item.full_name} · {item.admission_number}</option>)}</select><small>{studentOptionsQuery.data?.pagination.total ?? 0} matching active students. Refine your search if needed.</small></label>}
        <label>Academic year<select name="academic_year_id" value={academicYearID || ""} onChange={event => setAcademicYearID(Number(event.target.value))} required><option value="" disabled>Select active year</option>{activeYears.map(year => <option key={year.id} value={year.id}>{year.name}{year.is_current ? " · Current" : ""}</option>)}</select></label>
        <label>Section<select name="section_id" defaultValue="" required disabled={!academicYearID || sectionsQuery.isLoading}><option value="" disabled>Select section</option>{sections.map(section => <option key={section.id} value={section.id}>{section.name}</option>)}</select></label>
        <label>Roll number<input name="roll_number" maxLength={30}/></label>
        <label>Enrollment date<input name="enrolled_at" type="date" defaultValue={new Date().toISOString().slice(0, 10)} required/></label>
        {sectionsQuery.isError && <div className="student-feedback error" role="alert">{sectionsQuery.error.message}</div>}
        <div className="student-form-actions"><button type="button" className="student-secondary" onClick={() => setShowEnrollment(false)}>Cancel</button><button className="primary" disabled={busy || !selectedStudentID || studentOptionsQuery.isLoading || studentOptionsQuery.isError || activeYears.length === 0 || sections.length === 0}>{busy ? "Enrolling…" : "Create enrollment"}</button></div>
      </form>}
    </section>}
    {tab === "students" && canReadStudents && <>
      <div className="student-toolbar">
        <label className="student-search"><Search size={16}/><input aria-label="Search students" placeholder="Search name or admission number" value={search} onChange={event => { setSearch(event.target.value); setOffset(0); }}/></label>
        <select aria-label="Filter by student status" value={status} onChange={event => { setStatus(event.target.value); setOffset(0); }}><option value="">All statuses</option><option value="active">Active</option><option value="inactive">Inactive</option><option value="archived">Archived</option></select>
        <button type="button" className="student-secondary" onClick={() => void studentsQuery.refetch()} disabled={studentsQuery.isFetching}><RefreshCw size={15}/> Refresh</button>
      </div>
      {studentsQuery.isLoading ? <div className="student-state">Loading student records…</div> :
       studentsQuery.isError ? <div className="student-state error" role="alert"><p>{studentsQuery.error.message}</p><button className="student-secondary" onClick={() => void studentsQuery.refetch()}>Try again</button></div> :
       students.length === 0 ? <div className="student-state"><Users size={28}/><b>No students found</b><span>{search || status ? "Try changing the search or status filter." : "Add the first student record to get started."}</span></div> :
       <section className="student-table-wrap"><table className="student-table"><thead><tr><th>Student</th><th>Admission no.</th><th>Status</th><th>Actions</th></tr></thead><tbody>
        {students.map(item => <tr key={item.id}><td data-label="Student"><b>{item.full_name}</b><small>Student ID #{item.id}</small></td><td data-label="Admission no.">{item.admission_number}</td><td data-label="Status"><span className={"student-status " + item.status}>{item.status}</span></td><td data-label="Actions"><div className="student-row-actions">{canUpdate && item.status !== "archived" && <button type="button" title="Edit student" aria-label={`Edit ${item.full_name}`} onClick={() => openStudent("edit", item)}><Pencil size={15}/></button>}{canArchive && item.status !== "archived" && <button type="button" className="archive" title="Archive student" aria-label={`Archive ${item.full_name}`} onClick={() => void archive(item)}><Archive size={15}/></button>}</div></td></tr>)}
       </tbody></table></section>}
      <div className="student-pagination"><span>{total === 0 ? "0 students" : `Showing ${offset + 1}–${Math.min(offset + 20, total)} of ${total}`}</span><div><button type="button" className="student-secondary" disabled={offset === 0 || studentsQuery.isFetching} onClick={() => setOffset(Math.max(0, offset - 20))}>Previous</button><span>Page {Math.floor(offset / 20) + 1} of {pageCount}</span><button type="button" className="student-secondary" disabled={offset + 20 >= total || studentsQuery.isFetching} onClick={() => setOffset(offset + 20)}>Next</button></div></div>
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
