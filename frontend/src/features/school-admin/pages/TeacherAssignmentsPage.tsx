import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { Plus, RefreshCw, UserRoundX } from "lucide-react";
import { api } from "../../../lib/api";
import {
  createClassInchargeAssignment, createTeacherSubjectAssignment,
  getClassInchargeAssignments, getEligibleSchoolTeachers, getTeacherSubjectAssignments,
  revokeClassInchargeAssignment, revokeTeacherSubjectAssignment,
  type ClassInchargeAssignment, type EligibleSchoolTeacher, type TeacherSubjectAssignment,
} from "../api";
import { Select } from "../../../shared/components/Select";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import "./TeacherAssignmentsPage.css";

type Year = { id: number; name: string; status: string };
type Section = { id: number; academic_year_id: number; class_id: number; name: string; is_active: boolean };
type SchoolClass = { id: number; name: string; is_active: boolean };
type Subject = { id: number; name: string; is_active: boolean };

type Props = { permissions: string[] };

export function TeacherAssignmentsPage({ permissions }: Props) {
  const canReadSubjects = permissions.includes("subjects.read");
  const canReadYears = permissions.includes("academic_years.read");
  const canReadClasses = permissions.includes("classes.read");
  const canReadTeacherAssignments = permissions.includes("teacher_assignments.read");
  const canManageTeacherAssignments = permissions.includes("teacher_assignments.manage");
  const canReadIncharges = permissions.includes("class_incharge_assignments.read");
  const canManageIncharges = permissions.includes("class_incharge_assignments.manage");
  const [years, setYears] = useState<Year[]>([]);
  const [sections, setSections] = useState<Section[]>([]);
  const [classes, setClasses] = useState<SchoolClass[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [teachers, setTeachers] = useState<EligibleSchoolTeacher[]>([]);
  const [assignments, setAssignments] = useState<TeacherSubjectAssignment[]>([]);
  const [incharges, setIncharges] = useState<ClassInchargeAssignment[]>([]);
  const [yearID, setYearID] = useState("");
  const [sectionID, setSectionID] = useState("");
  const [subjectID, setSubjectID] = useState("");
  const [teacherID, setTeacherID] = useState("");
  const [inchargeYearID, setInchargeYearID] = useState("");
  const [inchargeSectionID, setInchargeSectionID] = useState("");
  const [inchargeTeacherID, setInchargeTeacherID] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [subjectDialogOpen,setSubjectDialogOpen]=useState(false);
  const [inchargeDialogOpen,setInchargeDialogOpen]=useState(false);
  const [fieldErrors,setFieldErrors]=useState<Record<string,string>>({});
  const [subjectToRevoke,setSubjectToRevoke]=useState<TeacherSubjectAssignment|null>(null);
  const [subjectRevokeError,setSubjectRevokeError]=useState("");
  const [inchargeToRevoke,setInchargeToRevoke]=useState<ClassInchargeAssignment|null>(null);
  const [inchargeRevokeError,setInchargeRevokeError]=useState("");

  const load = useCallback(async () => {
    setLoading(true);
    const jobs: Promise<unknown>[] = [];
    const keys: string[] = [];
    if (canReadYears) { keys.push("years"); jobs.push(api<{ academic_years: Year[] }>("/school/academic-years")); }
    if (canReadClasses) {
      keys.push("classes"); jobs.push(api<{ classes: SchoolClass[] }>("/school/classes"));
      keys.push("sections"); jobs.push(api<{ sections: Section[] }>("/school/sections"));
    }
    if (canReadSubjects) { keys.push("subjects"); jobs.push(api<{ subjects: Subject[] }>("/school/subjects")); }
    if (canReadTeacherAssignments || canReadIncharges) {
      keys.push("teachers"); jobs.push(getEligibleSchoolTeachers());
    }
    if (canReadTeacherAssignments) { keys.push("assignments"); jobs.push(getTeacherSubjectAssignments()); }
    if (canReadIncharges) { keys.push("incharges"); jobs.push(getClassInchargeAssignments()); }
    const results = await Promise.allSettled(jobs);
    const errors: string[] = [];
    results.forEach((result, index) => {
      if (result.status === "rejected") {
        errors.push(result.reason instanceof Error ? result.reason.message : "Unable to load assignment data.");
        return;
      }
      const data = result.value as Record<string, unknown>;
      switch (keys[index]) {
        case "years": setYears((data.academic_years as Year[]) ?? []); break;
        case "classes": setClasses((data.classes as SchoolClass[]) ?? []); break;
        case "sections": setSections((data.sections as Section[]) ?? []); break;
        case "subjects": setSubjects((data.subjects as Subject[]) ?? []); break;
        case "teachers": setTeachers((data.teachers as EligibleSchoolTeacher[]) ?? []); break;
        case "assignments": setAssignments((data.assignments as TeacherSubjectAssignment[]) ?? []); break;
        case "incharges": setIncharges((data.assignments as ClassInchargeAssignment[]) ?? []); break;
      }
    });
    if (errors.length) setFeedback({ kind: "error", message: errors.join(" · ") });
    setLoading(false);
  }, [canReadYears, canReadClasses, canReadSubjects, canReadTeacherAssignments, canReadIncharges]);

  useEffect(() => { void load(); }, [load]);

  const activeYears = useMemo(() => years.filter(item => item.status !== "closed"), [years]);
  const activeSections = useMemo(() => sections.filter(item => item.is_active), [sections]);
  const subjectSections = activeSections.filter(item => !yearID || String(item.academic_year_id) === yearID);
  const inchargeSections = activeSections.filter(item => !inchargeYearID || String(item.academic_year_id) === inchargeYearID);

  async function submitSubject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors:Record<string,string>={};
    if(!yearID)errors.yearID="Select an academic year.";
    if(!sectionID)errors.sectionID="Select a section.";
    if(!subjectID)errors.subjectID="Select a subject.";
    if(!teacherID)errors.teacherID="Select a teacher.";
    setFieldErrors(errors);
    if(Object.keys(errors).length)return;
    setSaving(true); setFeedback(null);
    try {
      await createTeacherSubjectAssignment({
        academic_year_id: Number(yearID), section_id: Number(sectionID),
        subject_id: Number(subjectID), teacher_principal_id: Number(teacherID),
      });
      setFeedback({ kind: "success", message: "Teacher assigned to the subject and section." });
      setSubjectDialogOpen(false);
      setSubjectID(""); setTeacherID(""); setFieldErrors({});
      await load();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to create teacher assignment." });
    } finally { setSaving(false); }
  }

  async function submitIncharge(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors:Record<string,string>={};
    if(!inchargeYearID)errors.inchargeYearID="Select an academic year.";
    if(!inchargeSectionID)errors.inchargeSectionID="Select a section.";
    if(!inchargeTeacherID)errors.inchargeTeacherID="Select a teacher.";
    setFieldErrors(errors);
    if(Object.keys(errors).length)return;
    setSaving(true); setFeedback(null);
    try {
      await createClassInchargeAssignment({
        academic_year_id: Number(inchargeYearID), section_id: Number(inchargeSectionID),
        teacher_principal_id: Number(inchargeTeacherID),
      });
      setFeedback({ kind: "success", message: "Class in-charge assigned successfully." });
      setInchargeDialogOpen(false);
      setInchargeTeacherID(""); setFieldErrors({});
      await load();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to assign class in-charge." });
    } finally { setSaving(false); }
  }

  async function confirmRevokeSubject() {
    if (!subjectToRevoke) return;
    const target = subjectToRevoke;
    setSaving(true); setFeedback(null);
    try {
      await revokeTeacherSubjectAssignment(target.id);
      setSubjectToRevoke(null);
      setSubjectRevokeError("");
      setFeedback({ kind: "success", message: "Subject assignment revoked. Its history has been retained." });
      await load();
    } catch (error) {
      setSubjectRevokeError(error instanceof Error ? error.message : "Unable to revoke assignment. Please try again.");
    } finally { setSaving(false); }
  }

  async function confirmRevokeIncharge() {
    if (!inchargeToRevoke) return;
    const target = inchargeToRevoke;
    setSaving(true); setFeedback(null);
    try {
      await revokeClassInchargeAssignment(target.id);
      setInchargeToRevoke(null);
      setInchargeRevokeError("");
      setFeedback({ kind: "success", message: "Class in-charge assignment revoked. Its history has been retained." });
      await load();
    } catch (error) {
      setInchargeRevokeError(error instanceof Error ? error.message : "Unable to revoke assignment. Please try again.");
    } finally { setSaving(false); }
  }

  return <div className="teacher-assignments">
    <div className="assignment-heading"><div><h2>Teacher assignments</h2><p>Assign active Teachers to subjects, sections and class in-charge responsibilities.</p></div><button type="button" onClick={() => void load()} disabled={loading}><RefreshCw size={15}/> Refresh</button></div>
    {feedback && <div className={"academic-feedback " + feedback.kind} role="status">{feedback.message}</div>}
    {loading ? <div className="table-message">Loading teacher assignments…</div> : <>
      {canReadTeacherAssignments && <section className="table-panel assignment-panel">
        <div className="toolbar"><div><h3>Subject teachers</h3><small>{assignments.filter(item => !item.revoked_at).length} active assignments</small></div></div>
        {canManageTeacherAssignments && <button type="button" className="primary assignment-open-dialog" onClick={()=>{setFieldErrors({});setSubjectDialogOpen(true);}}><Plus size={15}/> Assign subject teacher</button>}
        {assignments.length === 0 ? <div className="table-message">No subject teacher assignments yet.</div> : <div className="table-scroll"><table><thead><tr><th>Class / section</th><th>Subject</th><th>Teacher</th><th>Status</th>{canManageTeacherAssignments && <th>Action</th>}</tr></thead><tbody>{assignments.map(item => <tr key={item.id}><td>{item.class_name} · {item.section_name}</td><td>{item.subject_name}</td><td>{item.teacher_username}</td><td>{item.revoked_at ? "Revoked" : "Active"}</td>{canManageTeacherAssignments && <td>{!item.revoked_at && <button className="assignment-revoke" type="button" onClick={() => setSubjectToRevoke(item)} disabled={saving}><UserRoundX size={14}/> Revoke</button>}</td>}</tr>)}</tbody></table></div>}
      </section>}
      {canReadIncharges && <section className="table-panel assignment-panel">
        <div className="toolbar"><div><h3>Class in-charge</h3><small>{incharges.filter(item => !item.revoked_at).length} active assignments</small></div></div>
        {canManageIncharges && <button type="button" className="primary assignment-open-dialog" onClick={()=>{setFieldErrors({});setInchargeDialogOpen(true);}}><Plus size={15}/> Assign class in-charge</button>}
        {incharges.length === 0 ? <div className="table-message">No class in-charge assignments yet.</div> : <div className="table-scroll"><table><thead><tr><th>Class / section</th><th>Teacher</th><th>Status</th>{canManageIncharges && <th>Action</th>}</tr></thead><tbody>{incharges.map(item => <tr key={item.id}><td>{item.class_name} · {item.section_name}</td><td>{item.teacher_username}</td><td>{item.revoked_at ? "Revoked" : "Active"}</td>{canManageIncharges && <td>{!item.revoked_at && <button className="assignment-revoke" type="button" onClick={() => setInchargeToRevoke(item)} disabled={saving}><UserRoundX size={14}/> Revoke</button>}</td>}</tr>)}</tbody></table></div>}
      </section>}
    </>}
      <Modal open={subjectDialogOpen} title="Assign a subject teacher" description="Choose the academic year, section, subject and active teacher for this assignment." onClose={()=>setSubjectDialogOpen(false)} closeDisabled={saving}>
        {feedback?.kind==="error"&&<p className="academic-feedback error" role="alert">{feedback.message}</p>}
        <form className="assignment-form" onSubmit={submitSubject} noValidate>
          <label>Academic year<Select ariaLabel="Subject assignment academic year" invalid={Boolean(fieldErrors.yearID)} describedBy={fieldErrors.yearID?"year-error":undefined} value={yearID} onChange={value=>{setYearID(value);setSectionID("");setFieldErrors(current=>({...current,yearID:"",sectionID:""}));}} placeholder="Select year" options={activeYears.map(item=>({value:String(item.id),label:item.name}))}/>{activeYears.length===0&&<small>No active academic year is available. Create or activate an academic year first.</small>}{fieldErrors.yearID&&<small className="app-field-error" id="year-error" role="alert">{fieldErrors.yearID}</small>}</label>
          <label>Section<Select ariaLabel="Subject assignment section" invalid={Boolean(fieldErrors.sectionID)} describedBy={fieldErrors.sectionID?"section-error":undefined} value={sectionID} onChange={value=>{setSectionID(value);setFieldErrors(current=>({...current,sectionID:""}));}} placeholder="Select section" disabled={!yearID} options={subjectSections.filter(item=>String(item.academic_year_id)===yearID).map(item=>({value:String(item.id),label:`${classes.find(c=>c.id===item.class_id)?.name??"Class"} · ${item.name}`}))}/>{yearID&&subjectSections.filter(item=>String(item.academic_year_id)===yearID).length===0&&<small>No active sections exist for this academic year. Create or activate a section first.</small>}{fieldErrors.sectionID&&<small className="app-field-error" id="section-error" role="alert">{fieldErrors.sectionID}</small>}</label>
          <label>Subject<Select ariaLabel="Subject" invalid={Boolean(fieldErrors.subjectID)} describedBy={fieldErrors.subjectID?"subject-error":undefined} value={subjectID} onChange={value=>{setSubjectID(value);setFieldErrors(current=>({...current,subjectID:""}));}} placeholder="Select subject" disabled={subjects.filter(item=>item.is_active).length===0} options={subjects.filter(item=>item.is_active).map(item=>({value:String(item.id),label:item.name}))}/>{subjects.filter(item=>item.is_active).length===0&&<small>No active subjects are available. Add or activate a subject first.</small>}{fieldErrors.subjectID&&<small className="app-field-error" id="subject-error" role="alert">{fieldErrors.subjectID}</small>}</label>
          <label>Teacher<Select ariaLabel="Subject assignment teacher" invalid={Boolean(fieldErrors.teacherID)} describedBy={fieldErrors.teacherID?"teacher-error":undefined} value={teacherID} onChange={value=>{setTeacherID(value);setFieldErrors(current=>({...current,teacherID:""}));}} placeholder="Select teacher" disabled={teachers.length===0} options={teachers.map(item=>({value:String(item.principal_id),label:item.username}))}/>{teachers.length===0&&<small>No eligible active Teachers are available. Create or enable a Teacher account first.</small>}{fieldErrors.teacherID&&<small className="app-field-error" id="teacher-error" role="alert">{fieldErrors.teacherID}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={()=>setSubjectDialogOpen(false)} disabled={saving}>Cancel</button><button className="primary" type="submit" disabled={saving||!teachers.length||!activeYears.length}>{saving?"Assigning…":"Assign subject"}</button></div>
        </form>
      </Modal>
      <Modal open={inchargeDialogOpen} title="Assign class in-charge" description="Choose an academic year, section and active teacher responsible for the class." onClose={()=>setInchargeDialogOpen(false)} closeDisabled={saving}>
        {feedback?.kind==="error"&&<p className="academic-feedback error" role="alert">{feedback.message}</p>}
        <form className="assignment-form" onSubmit={submitIncharge} noValidate>
          <label>Academic year<Select ariaLabel="Class in-charge academic year" invalid={Boolean(fieldErrors.inchargeYearID)} describedBy={fieldErrors.inchargeYearID?"incharge-year-error":undefined} value={inchargeYearID} onChange={value=>{setInchargeYearID(value);setInchargeSectionID("");setFieldErrors(current=>({...current,inchargeYearID:"",inchargeSectionID:""}));}} placeholder="Select year" options={activeYears.map(item=>({value:String(item.id),label:item.name}))}/>{activeYears.length===0&&<small>No active academic year is available. Create or activate an academic year first.</small>}{fieldErrors.inchargeYearID&&<small className="app-field-error" id="incharge-year-error" role="alert">{fieldErrors.inchargeYearID}</small>}</label>
          <label>Section<Select ariaLabel="Class in-charge section" invalid={Boolean(fieldErrors.inchargeSectionID)} describedBy={fieldErrors.inchargeSectionID?"incharge-section-error":undefined} value={inchargeSectionID} onChange={value=>{setInchargeSectionID(value);setFieldErrors(current=>({...current,inchargeSectionID:""}));}} placeholder="Select section" disabled={!inchargeYearID} options={inchargeSections.filter(item=>String(item.academic_year_id)===inchargeYearID).map(item=>({value:String(item.id),label:`${classes.find(c=>c.id===item.class_id)?.name??"Class"} · ${item.name}`}))}/>{inchargeYearID&&inchargeSections.filter(item=>String(item.academic_year_id)===inchargeYearID).length===0&&<small>No active sections exist for this academic year. Create or activate a section first.</small>}{fieldErrors.inchargeSectionID&&<small className="app-field-error" id="incharge-section-error" role="alert">{fieldErrors.inchargeSectionID}</small>}</label>
          <label>Teacher<Select ariaLabel="Class in-charge teacher" invalid={Boolean(fieldErrors.inchargeTeacherID)} describedBy={fieldErrors.inchargeTeacherID?"incharge-teacher-error":undefined} value={inchargeTeacherID} onChange={value=>{setInchargeTeacherID(value);setFieldErrors(current=>({...current,inchargeTeacherID:""}));}} placeholder="Select teacher" disabled={teachers.length===0} options={teachers.map(item=>({value:String(item.principal_id),label:item.username}))}/>{teachers.length===0&&<small>No eligible active Teachers are available. Create or enable a Teacher account first.</small>}{fieldErrors.inchargeTeacherID&&<small className="app-field-error" id="incharge-teacher-error" role="alert">{fieldErrors.inchargeTeacherID}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={()=>setInchargeDialogOpen(false)} disabled={saving}>Cancel</button><button className="primary" type="submit" disabled={saving||!teachers.length||!activeYears.length}>{saving?"Assigning…":"Assign in-charge"}</button></div>
        </form>
      </Modal>
      <ConfirmDialog open={subjectToRevoke !== null} title="Revoke subject assignment?" description={subjectToRevoke ? `Revoke ${subjectToRevoke.teacher_username}'s ${subjectToRevoke.subject_name} assignment for ${subjectToRevoke.class_name} · ${subjectToRevoke.section_name}? The assignment history will be retained.` : ""} confirmLabel="Revoke assignment" pending={saving} error={subjectRevokeError || undefined} onCancel={() => {setSubjectToRevoke(null);setSubjectRevokeError("");}} onConfirm={() => void confirmRevokeSubject()}/>
      <ConfirmDialog open={inchargeToRevoke !== null} title="Revoke class in-charge?" description={inchargeToRevoke ? `Remove ${inchargeToRevoke.teacher_username} as class in-charge for ${inchargeToRevoke.class_name} · ${inchargeToRevoke.section_name}? The assignment history will be retained.` : ""} confirmLabel="Revoke assignment" pending={saving} error={inchargeRevokeError || undefined} onCancel={() => {setInchargeToRevoke(null);setInchargeRevokeError("");}} onConfirm={() => void confirmRevokeIncharge()}/>
  </div>;
}
