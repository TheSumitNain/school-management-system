import { useEffect, useState } from "react";
import { getClassInchargeAssignments, getTeacherSubjectAssignments, type ClassInchargeAssignment, type TeacherSubjectAssignment, type StaffRecord } from "../api";

type Props = { staff: StaffRecord; permissions: string[]; onOpenWorkspace: () => void };

function formatDate(value?: string | null) {
  if (!value) return "Date not recorded";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Date not recorded" : new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(date);
}

export function StaffAcademicAssignmentsPanel({ staff, permissions, onOpenWorkspace }: Props) {
  const canReadSubjects = permissions.includes("teacher_assignments.read");
  const canReadIncharges = permissions.includes("class_incharge_assignments.read");
  const [subjects, setSubjects] = useState<TeacherSubjectAssignment[]>([]);
  const [incharges, setIncharges] = useState<ClassInchargeAssignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    const jobs: Promise<unknown>[] = [];
    if (canReadSubjects) jobs.push(getTeacherSubjectAssignments({ teacherPrincipalID: staff.principal_id }, controller.signal).then(result => setSubjects(result.assignments)));
    else setSubjects([]);
    if (canReadIncharges) jobs.push(getClassInchargeAssignments(undefined, controller.signal, staff.principal_id).then(result => setIncharges(result.assignments)));
    else setIncharges([]);
    if (jobs.length === 0) {
      setLoading(false);
      return () => controller.abort();
    }
    Promise.all(jobs).catch(reason => {
      if (!controller.signal.aborted) setError(reason instanceof Error ? reason.message : "Unable to load this employee's academic assignments.");
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false);
    });
    return () => controller.abort();
  }, [staff.principal_id, canReadSubjects, canReadIncharges, reloadKey]);

  const activeCount = subjects.filter(item => !item.revoked_at).length + incharges.filter(item => !item.revoked_at).length;
  const historyCount = subjects.length + incharges.length - activeCount;
  return <section className="staff-records-panel staff-academic-profile">
    <div className="staff-records-heading"><div><h3>Academic assignments</h3><p>Assignments linked to {staff.full_name || staff.username}. Revoked records remain visible as history.</p></div><button type="button" className="staff-secondary-button" onClick={onOpenWorkspace}>Manage assignments</button></div>
    {!canReadSubjects && !canReadIncharges ? <p className="staff-record-empty">You do not have permission to view academic assignments.</p> : loading ? <p role="status">Loading academic assignments…</p> : error ? <div className="staff-record-error" role="alert"><span>{error}</span><button type="button" onClick={() => setReloadKey(value => value + 1)}>Try again</button></div> : <>
      <div className="staff-assignment-summary"><span><strong>{activeCount}</strong> active</span><span><strong>{historyCount}</strong> revoked / historical</span></div>
      {canReadSubjects && <div className="staff-record-group"><div className="staff-record-group-heading"><h4>Subject teaching</h4><span>{subjects.length} records</span></div>{subjects.length === 0 ? <p className="staff-record-empty">No subject teaching assignments recorded.</p> : subjects.map(item => <article className="staff-record-card" key={item.id}><div><strong>{item.subject_name}</strong><span>{item.class_name} · {item.section_name} · Academic year #{item.academic_year_id}</span><small>Assigned {formatDate(item.assigned_at)}</small></div><span className={item.revoked_at ? "staff-record-status inactive" : "staff-record-status"}>{item.revoked_at ? `Revoked · ${formatDate(item.revoked_at)}` : "Active"}</span></article>)}</div>}
      {canReadIncharges && <div className="staff-record-group"><div className="staff-record-group-heading"><h4>Class in-charge</h4><span>{incharges.length} records</span></div>{incharges.length === 0 ? <p className="staff-record-empty">No class in-charge assignments recorded.</p> : incharges.map(item => <article className="staff-record-card" key={item.id}><div><strong>{item.class_name} · {item.section_name}</strong><span>Academic year #{item.academic_year_id}</span><small>Assigned {formatDate(item.assigned_at)}</small></div><span className={item.revoked_at ? "staff-record-status inactive" : "staff-record-status"}>{item.revoked_at ? `Revoked · ${formatDate(item.revoked_at)}` : "Active"}</span></article>)}</div>}
    </>}
  </section>;
}
