import { useEffect, useMemo, useState } from "react";
import { CalendarDays, Check, RefreshCw } from "lucide-react";
import {
  useSchoolAttendanceQuery, useMarkSchoolAttendanceMutation, useUpdateSchoolAttendanceMutation,
  useSchoolStudentAcademicYearsQuery, useSchoolStudentEnrollmentsQuery, useSchoolStudentSectionsQuery,
} from "../queries";
import type { AttendanceRecord } from "../api";
import "./AttendanceManagementPage.css";

type Props = { permissions: string[] };
type Draft = { status: string; remarks: string };
const statuses = [
  { value: "present", label: "Present" },
  { value: "absent", label: "Absent" },
  { value: "late", label: "Late" },
  { value: "excused", label: "Excused" },
];
function localDateValue() {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
}
function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Attendance could not be saved. Please try again.";
}

export function AttendanceManagementPage({ permissions }: Props) {
  const canRead = permissions.includes("attendance.read");
  const canReadRoster = permissions.includes("students.read_basic");
  const canReadYears = permissions.includes("academic_years.read");
  const canReadSections = permissions.includes("classes.read");
  const canMark = permissions.includes("attendance.mark");
  const canUpdateSameDay = permissions.includes("attendance.update_same_day");
  const canCorrectPast = permissions.includes("attendance.correct_past");

  const [academicYearID, setAcademicYearID] = useState(0);
  const [sectionID, setSectionID] = useState(0);
  const [date, setDate] = useState(localDateValue);
  const [drafts, setDrafts] = useState<Record<number, Draft>>({});
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const yearsQuery = useSchoolStudentAcademicYearsQuery(canReadYears && canRead);
  const years = yearsQuery.data?.academic_years ?? [];
  const activeYears = useMemo(() => years.filter(year => year.status === "active"), [years]);
  const effectiveYearID = academicYearID || activeYears.find(year => year.is_current)?.id || activeYears[0]?.id || 0;
  const sectionsQuery = useSchoolStudentSectionsQuery(effectiveYearID, canReadSections && canRead);
  const sections = (sectionsQuery.data?.sections ?? []).filter(section => section.is_active);
  const enrollmentsQuery = useSchoolStudentEnrollmentsQuery(effectiveYearID, canReadRoster && canRead && sectionID > 0);
  const attendanceQuery = useSchoolAttendanceQuery(
    { academicYearID: effectiveYearID, sectionID, date },
    canRead && effectiveYearID > 0 && sectionID > 0 && date.length === 10,
  );
  const markMutation = useMarkSchoolAttendanceMutation();
  const updateMutation = useUpdateSchoolAttendanceMutation();

  useEffect(() => {
    if (sectionID > 0 && !sections.some(section => section.id === sectionID)) setSectionID(0);
  }, [sectionID, sections]);

  const enrollments = (enrollmentsQuery.data?.enrollments ?? [])
    .filter(item => item.section_id === sectionID && item.status === "active")
    .sort((a, b) => (a.roll_number ?? "").localeCompare(b.roll_number ?? "", undefined, { numeric: true }) || (a.student_name ?? "").localeCompare(b.student_name ?? ""));
  const attendance = attendanceQuery.data?.data ?? [];
  const attendanceByEnrollment = useMemo(() => new Map<number, AttendanceRecord>(attendance.map(item => [item.enrollment_id, item])), [attendance]);
  const today = localDateValue();
  const isPast = date < today;
  const isFuture = date > today;
  const busy = markMutation.isPending || updateMutation.isPending;

  function draftFor(enrollmentID: number): Draft {
    const draft = drafts[enrollmentID];
    if (draft) return draft;
    const existing = attendanceByEnrollment.get(enrollmentID);
    return { status: existing?.status ?? "", remarks: existing?.remarks ?? "" };
  }
  function changeDraft(enrollmentID: number, patch: Partial<Draft>) {
    setDrafts(current => ({ ...current, [enrollmentID]: { ...draftFor(enrollmentID), ...patch } }));
    setFeedback(null);
  }
  async function save(enrollmentID: number) {
    const draft = draftFor(enrollmentID);
    if (!draft.status) return;
    const existing = attendanceByEnrollment.get(enrollmentID);
    setFeedback(null);
    try {
      if (existing) {
        await updateMutation.mutateAsync({ attendanceID: existing.id, status: draft.status, remarks: draft.remarks });
      } else {
        await markMutation.mutateAsync({ enrollment_id: enrollmentID, attendance_date: date, status: draft.status, remarks: draft.remarks || null });
      }
      setDrafts(current => { const next = { ...current }; delete next[enrollmentID]; return next; });
      setFeedback({ kind: "success", message: "Attendance saved successfully." });
    } catch (error) {
      setFeedback({ kind: "error", message: errorMessage(error) });
    }
  }

  if (!canRead) {
    return <section className="content attendance-page"><div className="attendance-state"><b>Attendance access unavailable</b><span>Your account does not have attendance.read permission.</span></div></section>;
  }

  return (
    <section className="attendance-page">
      <div className="heading">
        <div><span className="eyebrow">SCHOOL OPERATIONS</span><h1>Attendance</h1><p>Review the class roster and record daily attendance.</p></div>
      </div>

      <div className="attendance-filters">
        <label>Academic year
          <select value={effectiveYearID || ""} onChange={event => { setAcademicYearID(Number(event.target.value)); setSectionID(0); setDrafts({}); }} disabled={!canReadYears || yearsQuery.isLoading}>
            <option value="">Select academic year</option>
            {activeYears.map(year => <option key={year.id} value={year.id}>{year.name}{year.is_current ? " · Current" : ""}</option>)}
          </select>
        </label>
        <label>Class / section
          <select value={sectionID || ""} onChange={event => { setSectionID(Number(event.target.value)); setDrafts({}); }} disabled={!canReadSections || !effectiveYearID || sectionsQuery.isLoading}>
            <option value="">Select section</option>
            {sections.map(section => <option key={section.id} value={section.id}>Section {section.name} · Class #{section.class_id}</option>)}
          </select>
        </label>
        <label>Attendance date
          <input type="date" value={date} max={today} onChange={event => { setDate(event.target.value); setDrafts({}); }} />
        </label>
        <button type="button" className="attendance-refresh" onClick={() => { void enrollmentsQuery.refetch(); void attendanceQuery.refetch(); }} disabled={!sectionID || busy}><RefreshCw size={15}/> Refresh</button>
      </div>

      {feedback && <div className={"attendance-feedback " + feedback.kind} role="status">{feedback.message}</div>}
      {!canReadYears || !canReadSections || !canReadRoster ? (
        <div className="attendance-state"><b>Additional permissions required</b><span>To load the roster, your account needs academic_years.read, classes.read and students.read_basic.</span></div>
      ) : !effectiveYearID ? (
        <div className="attendance-state"><b>No active academic year</b><span>Create or activate an academic year before recording attendance.</span></div>
      ) : !sectionID ? (
        <div className="attendance-state"><b>Select a section</b><span>Choose an academic year and section to load its enrolled students.</span></div>
      ) : enrollmentsQuery.isLoading || attendanceQuery.isLoading ? (
        <div className="attendance-state"><span className="attendance-spinner"/>Loading attendance roster…</div>
      ) : enrollmentsQuery.isError || attendanceQuery.isError ? (
        <div className="attendance-state error"><b>Could not load attendance</b><span>{errorMessage(enrollmentsQuery.error ?? attendanceQuery.error)}</span><button type="button" onClick={() => { void enrollmentsQuery.refetch(); void attendanceQuery.refetch(); }}>Try again</button></div>
      ) : (
        <>
          <div className="attendance-summary">
            <div><b>{enrollments.length}</b><span>Enrolled students</span></div>
            <div><b>{attendance.length}</b><span>Attendance entries</span></div>
            <div><b>{enrollments.length - enrollments.filter(item => attendanceByEnrollment.has(item.id)).length}</b><span>Not marked</span></div>
          </div>
          {isFuture && <div className="attendance-feedback error">Future attendance cannot be recorded from this screen.</div>}
          {isPast && !canCorrectPast && <div className="attendance-feedback error">This is a past date. Your account needs attendance.correct_past to enter or correct it.</div>}
          <div className="attendance-table-wrap">
            <table className="attendance-table">
              <thead><tr><th>Student</th><th>Roll no.</th><th>Status</th><th>Remarks</th><th>Action</th></tr></thead>
              <tbody>
                {enrollments.map(enrollment => {
                  const existing = attendanceByEnrollment.get(enrollment.id);
                  const draft = draftFor(enrollment.id);
                  const canSave = !isFuture && (isPast ? canCorrectPast : existing ? canUpdateSameDay : canMark);
                  return <tr key={enrollment.id}>
                    <td><b>{enrollment.student_name || "Student"}</b><small>{enrollment.class_name_snapshot} · {enrollment.section_name_snapshot}</small></td>
                    <td>{enrollment.roll_number || "—"}</td>
                    <td><select aria-label={"Attendance status for " + (enrollment.student_name || "student")} value={draft.status} onChange={event => changeDraft(enrollment.id, { status: event.target.value })} disabled={!canSave || busy}>
                      <option value="">Select status</option>{statuses.map(status => <option key={status.value} value={status.value}>{status.label}</option>)}
                    </select></td>
                    <td><input aria-label={"Remarks for " + (enrollment.student_name || "student")} value={draft.remarks} maxLength={500} onChange={event => changeDraft(enrollment.id, { remarks: event.target.value })} placeholder="Optional" disabled={!canSave || busy}/></td>
                    <td><button type="button" className="attendance-save" onClick={() => void save(enrollment.id)} disabled={!canSave || !draft.status || busy}><Check size={14}/>{existing ? "Update" : "Save"}</button></td>
                  </tr>;
                })}
                {enrollments.length === 0 && <tr><td colSpan={5}><div className="attendance-empty">No active enrollments found for this section.</div></td></tr>}
              </tbody>
            </table>
          </div>
          <p className="attendance-note">Attendance changes are saved per student. The server enforces school, class-in-charge and date-specific permissions.</p>
        </>
      )}
    </section>
  );
}
