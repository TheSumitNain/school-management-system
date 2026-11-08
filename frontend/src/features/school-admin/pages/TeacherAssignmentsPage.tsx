import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { Plus, RefreshCw, UserRoundX } from "lucide-react";
import { api } from "../../../lib/api";
import {
  createClassInchargeAssignment, createTeacherSubjectAssignment,
  getClassInchargeAssignments, getEligibleSchoolTeachers, getTeacherSubjectAssignments,
  revokeClassInchargeAssignment, revokeTeacherSubjectAssignment,
  type ClassInchargeAssignment, type EligibleSchoolTeacher, type TeacherSubjectAssignment,
} from "../api";
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
    if (!yearID || !sectionID || !subjectID || !teacherID) {
      setFeedback({ kind: "error", message: "Select an academic year, section, subject and teacher." });
      return;
    }
    setSaving(true); setFeedback(null);
    try {
      await createTeacherSubjectAssignment({
        academic_year_id: Number(yearID), section_id: Number(sectionID),
        subject_id: Number(subjectID), teacher_principal_id: Number(teacherID),
      });
      setFeedback({ kind: "success", message: "Teacher assigned to the subject and section." });
      setSubjectID(""); setTeacherID("");
      await load();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to create teacher assignment." });
    } finally { setSaving(false); }
  }

  async function submitIncharge(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!inchargeYearID || !inchargeSectionID || !inchargeTeacherID) {
      setFeedback({ kind: "error", message: "Select an academic year, section and teacher." });
      return;
    }
    setSaving(true); setFeedback(null);
    try {
      await createClassInchargeAssignment({
        academic_year_id: Number(inchargeYearID), section_id: Number(inchargeSectionID),
        teacher_principal_id: Number(inchargeTeacherID),
      });
      setFeedback({ kind: "success", message: "Class in-charge assigned successfully." });
      setInchargeTeacherID("");
      await load();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to assign class in-charge." });
    } finally { setSaving(false); }
  }

  async function revokeSubject(item: TeacherSubjectAssignment) {
    if (!window.confirm("Revoke this teacher's subject assignment? The assignment history will be retained.")) return;
    setSaving(true); setFeedback(null);
    try { await revokeTeacherSubjectAssignment(item.id); setFeedback({ kind: "success", message: "Subject assignment revoked." }); await load(); }
    catch (error) { setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to revoke assignment." }); }
    finally { setSaving(false); }
  }

  async function revokeIncharge(item: ClassInchargeAssignment) {
    if (!window.confirm("Revoke this class in-charge assignment? The assignment history will be retained.")) return;
    setSaving(true); setFeedback(null);
    try { await revokeClassInchargeAssignment(item.id); setFeedback({ kind: "success", message: "Class in-charge assignment revoked." }); await load(); }
    catch (error) { setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to revoke assignment." }); }
    finally { setSaving(false); }
  }

  return <div className="teacher-assignments">
    <div className="assignment-heading"><div><h2>Teacher assignments</h2><p>Assign active Teachers to subjects, sections and class in-charge responsibilities.</p></div><button type="button" onClick={() => void load()} disabled={loading}><RefreshCw size={15}/> Refresh</button></div>
    {feedback && <div className={"academic-feedback " + feedback.kind} role="status">{feedback.message}</div>}
    {loading ? <div className="table-message">Loading teacher assignments…</div> : <>
      {canReadTeacherAssignments && <section className="table-panel assignment-panel">
        <div className="toolbar"><div><h3>Subject teachers</h3><small>{assignments.filter(item => !item.revoked_at).length} active assignments</small></div></div>
        {canManageTeacherAssignments && <form className="assignment-form" onSubmit={submitSubject}>
          <label>Academic year<select value={yearID} onChange={event => { setYearID(event.target.value); setSectionID(""); }} required><option value="">Select year</option>{activeYears.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>Section<select value={sectionID} onChange={event => setSectionID(event.target.value)} required disabled={!yearID}><option value="">Select section</option>{subjectSections.filter(item => String(item.academic_year_id) === yearID).map(item => <option key={item.id} value={item.id}>{classes.find(c => c.id === item.class_id)?.name ?? "Class"} · {item.name}</option>)}</select></label>
          <label>Subject<select value={subjectID} onChange={event => setSubjectID(event.target.value)} required><option value="">Select subject</option>{subjects.filter(item => item.is_active).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>Teacher<select value={teacherID} onChange={event => setTeacherID(event.target.value)} required><option value="">Select Teacher</option>{teachers.map(item => <option key={item.principal_id} value={item.principal_id}>{item.username}</option>)}</select></label>
          <button className="primary" type="submit" disabled={saving || !teachers.length || !activeYears.length}><Plus size={15}/> Assign subject</button>
        </form>}
        {assignments.length === 0 ? <div className="table-message">No subject teacher assignments yet.</div> : <div className="table-scroll"><table><thead><tr><th>Class / section</th><th>Subject</th><th>Teacher</th><th>Status</th>{canManageTeacherAssignments && <th>Action</th>}</tr></thead><tbody>{assignments.map(item => <tr key={item.id}><td>{item.class_name} · {item.section_name}</td><td>{item.subject_name}</td><td>{item.teacher_username}</td><td>{item.revoked_at ? "Revoked" : "Active"}</td>{canManageTeacherAssignments && <td>{!item.revoked_at && <button className="assignment-revoke" type="button" onClick={() => void revokeSubject(item)} disabled={saving}><UserRoundX size={14}/> Revoke</button>}</td>}</tr>)}</tbody></table></div>}
      </section>}
      {canReadIncharges && <section className="table-panel assignment-panel">
        <div className="toolbar"><div><h3>Class in-charge</h3><small>{incharges.filter(item => !item.revoked_at).length} active assignments</small></div></div>
        {canManageIncharges && <form className="assignment-form" onSubmit={submitIncharge}>
          <label>Academic year<select value={inchargeYearID} onChange={event => { setInchargeYearID(event.target.value); setInchargeSectionID(""); }} required><option value="">Select year</option>{activeYears.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>Section<select value={inchargeSectionID} onChange={event => setInchargeSectionID(event.target.value)} required disabled={!inchargeYearID}><option value="">Select section</option>{inchargeSections.filter(item => String(item.academic_year_id) === inchargeYearID).map(item => <option key={item.id} value={item.id}>{classes.find(c => c.id === item.class_id)?.name ?? "Class"} · {item.name}</option>)}</select></label>
          <label>Teacher<select value={inchargeTeacherID} onChange={event => setInchargeTeacherID(event.target.value)} required><option value="">Select Teacher</option>{teachers.map(item => <option key={item.principal_id} value={item.principal_id}>{item.username}</option>)}</select></label>
          <button className="primary" type="submit" disabled={saving || !teachers.length || !activeYears.length}><Plus size={15}/> Assign in-charge</button>
        </form>}
        {incharges.length === 0 ? <div className="table-message">No class in-charge assignments yet.</div> : <div className="table-scroll"><table><thead><tr><th>Class / section</th><th>Teacher</th><th>Status</th>{canManageIncharges && <th>Action</th>}</tr></thead><tbody>{incharges.map(item => <tr key={item.id}><td>{item.class_name} · {item.section_name}</td><td>{item.teacher_username}</td><td>{item.revoked_at ? "Revoked" : "Active"}</td>{canManageIncharges && <td>{!item.revoked_at && <button className="assignment-revoke" type="button" onClick={() => void revokeIncharge(item)} disabled={saving}><UserRoundX size={14}/> Revoke</button>}</td>}</tr>)}</tbody></table></div>}
      </section>}
    </>}
  </div>;
}
