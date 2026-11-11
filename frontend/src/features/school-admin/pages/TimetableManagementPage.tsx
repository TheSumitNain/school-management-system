import { useEffect, useMemo, useState, type FormEvent } from "react";
import { CalendarClock, Plus, RefreshCw, Trash2 } from "lucide-react";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import {
  getSchoolAcademicYears, getCurrentTimetableYear, getSchoolSections, getTeacherSubjectAssignments,
  getTimetablePeriods, getTimetableEntries, saveTimetablePeriod, createTimetableEntry, deleteTimetableEntry,
  type StudentAcademicYear, type StudentSection, type TeacherSubjectAssignment,
  type TimetablePeriod, type TimetableEntry,
} from "../api";
import "./TimetableManagementPage.css";

type Props = { permissions: string[] };
type Dialog = "period" | "entry" | null;
const weekdays = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
const errorText = (error: unknown) => error instanceof Error ? error.message : "The timetable request could not be completed.";

export function TimetableManagementPage({ permissions }: Props) {
  const canRead = permissions.includes("timetable.read");
  const canManage = permissions.includes("timetable.manage");
  const canReadYears = permissions.includes("academic_years.read");
  const canReadSections = permissions.includes("classes.read");
  const canReadAssignments = permissions.includes("teacher_assignments.read");

  const [years, setYears] = useState<StudentAcademicYear[]>([]);
  const [yearID, setYearID] = useState(0);
  const [sections, setSections] = useState<StudentSection[]>([]);
  const [sectionID, setSectionID] = useState(0);
  const [periods, setPeriods] = useState<TimetablePeriod[]>([]);
  const [entries, setEntries] = useState<TimetableEntry[]>([]);
  const [assignments, setAssignments] = useState<TeacherSubjectAssignment[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [dialog, setDialog] = useState<Dialog>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [periodDraft, setPeriodDraft] = useState({ period_number: "", name: "", starts_at: "", ends_at: "", is_break: false });
  const [entryDraft, setEntryDraft] = useState({ section_id: "", assignment_id: "", period_id: "", day_of_week: "1", room: "" });

  const activeYears = useMemo(() => years.filter(year => year.status !== "closed"), [years]);
  const selectedYear = activeYears.find(year => year.id === yearID);
  const visibleEntries = useMemo(
    () => sectionID ? entries.filter(entry => entry.section_id === sectionID) : entries,
    [entries, sectionID],
  );
  const orderedPeriods = useMemo(() => [...periods].sort((a, b) => a.period_number - b.period_number), [periods]);
  const filteredAssignments = useMemo(
    () => assignments.filter(item => !item.revoked_at && (!entryDraft.section_id || item.section_id === Number(entryDraft.section_id))),
    [assignments, entryDraft.section_id],
  );

  async function loadSchedule(nextYearID: number, nextSectionID = sectionID) {
    if (!nextYearID) return;
    setLoading(true);
    setError("");
    try {
      const [periodResult, entryResult] = await Promise.all([
        getTimetablePeriods(nextYearID),
        getTimetableEntries(nextYearID, nextSectionID),
      ]);
      setPeriods(periodResult.periods ?? []);
      setEntries(entryResult.entries ?? []);
    } catch (reason) {
      setError(errorText(reason));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let alive = true;
    async function loadYears() {
      if (!canRead) { setLoading(false); return; }
      try {
        const result = canReadYears ? await getSchoolAcademicYears() : null;
        const current = canReadYears ? null : await getCurrentTimetableYear();
        if (!alive) return;
        const available = (result?.academic_years ?? (current?.academic_year ? [current.academic_year] : [])).filter(year => year.status !== "closed");
        setYears(available);
        const initial = available.find(year => year.is_current)?.id ?? available[0]?.id ?? 0;
        setYearID(current => current || initial);
      } catch (reason) {
        if (alive) { setError(errorText(reason)); setLoading(false); }
      }
    }
    void loadYears();
    return () => { alive = false; };
  }, [canRead, canReadYears]);

  useEffect(() => {
    let alive = true;
    async function loadReferenceData() {
      if (!yearID) { setLoading(false); return; }
      setLoading(true);
      setError("");
      try {
        const jobs: Promise<unknown>[] = [getTimetablePeriods(yearID), getTimetableEntries(yearID, sectionID)];
        if (canReadSections && canManage) jobs.push(getSchoolSections(yearID));
        if (canReadAssignments && canManage) jobs.push(getTeacherSubjectAssignments({ academicYearID: yearID }));
        const results = await Promise.all(jobs);
        if (!alive) return;
        const periodResult = results[0] as { periods: TimetablePeriod[] };
        const entryResult = results[1] as { entries: TimetableEntry[] };
        setPeriods(periodResult.periods ?? []);
        setEntries(entryResult.entries ?? []);
        if (canReadSections && canManage) {
          const sectionResult = results[2] as { sections: StudentSection[] };
          setSections((sectionResult.sections ?? []).filter(section => section.is_active));
        }
        if (canReadAssignments && canManage) {
          const assignmentResult = results[canReadSections ? 3 : 2] as { assignments: TeacherSubjectAssignment[] };
          setAssignments(assignmentResult.assignments ?? []);
        }
      } catch (reason) {
        if (alive) setError(errorText(reason));
      } finally {
        if (alive) setLoading(false);
      }
    }
    void loadReferenceData();
    return () => { alive = false; };
  }, [yearID, sectionID, canReadSections, canReadAssignments, canManage]);

  function closeDialog() {
    if (saving) return;
    setDialog(null);
    setFieldErrors({});
  }
  function openPeriodDialog() {
    setPeriodDraft({ period_number: String(periods.length + 1), name: "", starts_at: "", ends_at: "", is_break: false });
    setFieldErrors({});
    setDialog("period");
  }
  function openEntryDialog() {
    setEntryDraft({ section_id: String(sectionID || sections[0]?.id || ""), assignment_id: "", period_id: String(periods.find(period => !period.is_break)?.id ?? ""), day_of_week: "1", room: "" });
    setFieldErrors({});
    setDialog("entry");
  }

  async function submitPeriod(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors: Record<string, string> = {};
    const number = Number(periodDraft.period_number);
    if (!Number.isInteger(number) || number <= 0) errors.period_number = "Enter a positive whole period number.";
    else if (periods.some(period => period.period_number === number)) errors.period_number = "This period number is already in use for the selected year.";
    if (!periodDraft.name.trim()) errors.name = "Enter a period name.";
    if (!periodDraft.starts_at) errors.starts_at = "Choose a start time.";
    if (!periodDraft.ends_at) errors.ends_at = "Choose an end time.";
    else if (periodDraft.starts_at && periodDraft.ends_at <= periodDraft.starts_at) errors.ends_at = "End time must be later than start time.";
    else if (periodDraft.starts_at && periods.some(period => period.starts_at < periodDraft.ends_at && period.ends_at > periodDraft.starts_at)) { errors.starts_at = "This period overlaps an existing period."; errors.ends_at = "Choose a time range that does not overlap another period."; }
    if (Object.keys(errors).length) { setFieldErrors(errors); return; }
    setSaving(true); setFeedback(null);
    try {
      await saveTimetablePeriod({ academic_year_id: yearID, period_number: number, name: periodDraft.name.trim(), starts_at: periodDraft.starts_at, ends_at: periodDraft.ends_at, is_break: periodDraft.is_break });
      setDialog(null);
      setFeedback({ kind: "success", message: "Timetable period saved." });
      await loadSchedule(yearID);
    } catch (reason) {
      setFeedback({ kind: "error", message: errorText(reason) });
    } finally { setSaving(false); }
  }

  async function submitEntry(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors: Record<string, string> = {};
    const section = sections.find(item => item.id === Number(entryDraft.section_id));
    const assignment = assignments.find(item => item.id === Number(entryDraft.assignment_id) && !item.revoked_at);
    const period = periods.find(item => item.id === Number(entryDraft.period_id));
    if (!section) errors.section_id = "Select a class section.";
    if (!assignment) errors.assignment_id = "Select an active teacher-subject assignment.";
    if (!period || period.is_break) errors.period_id = "Select a teaching period, not a break.";
    if (!Number(entryDraft.day_of_week)) errors.day_of_week = "Select a weekday.";
    if (section && assignment && period && Number(entryDraft.day_of_week)) {
      if (entries.some(item => item.section_id === section.id && item.day_of_week === Number(entryDraft.day_of_week) && item.period_id === period.id)) errors.period_id = "This section already has a class in that period and day.";
      if (entries.some(item => item.teacher_principal_id === assignment.teacher_principal_id && item.day_of_week === Number(entryDraft.day_of_week) && item.period_id === period.id)) errors.assignment_id = "This teacher is already scheduled in that period and day.";
    }
    if (Object.keys(errors).length) { setFieldErrors(errors); return; }
    if (!assignment || !section || !period) return;
    setSaving(true); setFeedback(null);
    try {
      await createTimetableEntry({
        academic_year_id: yearID, section_id: section.id, subject_id: assignment.subject_id,
        teacher_principal_id: assignment.teacher_principal_id, teacher_subject_assignment_id: assignment.id,
        period_id: period.id, day_of_week: Number(entryDraft.day_of_week), room: entryDraft.room.trim() || null,
      });
      setDialog(null);
      setFeedback({ kind: "success", message: "Class added to the weekly timetable." });
      await loadSchedule(yearID, sectionID);
    } catch (reason) {
      setFeedback({ kind: "error", message: errorText(reason) });
    } finally { setSaving(false); }
  }

  async function removeEntry(entry: TimetableEntry) {
    if (!window.confirm(`Remove ${entry.subject_name} from ${entry.class_name} · ${entry.section_name}?`)) return;
    setFeedback(null);
    try {
      await deleteTimetableEntry(entry.id, yearID);
      setFeedback({ kind: "success", message: "Timetable entry removed." });
      await loadSchedule(yearID, sectionID);
    } catch (reason) {
      setFeedback({ kind: "error", message: errorText(reason) });
    }
  }

  if (!canRead) return <section className="content timetable-page"><div className="timetable-state"><b>Timetable access unavailable</b><span>Your account does not have timetable.read permission.</span></div></section>;

  return (
    <section className="content timetable-page">
      <div className="heading">
        <div><span className="eyebrow">SCHOOL OPERATIONS</span><h1>Timetable</h1><p>Plan the weekly teaching schedule and keep class and teacher periods conflict-free.</p></div>
        {canManage && <div className="timetable-actions"><button type="button" className="timetable-secondary" onClick={openPeriodDialog}><Plus size={15}/> Add period</button><button type="button" className="timetable-primary" onClick={openEntryDialog} disabled={!periods.some(period => !period.is_break) || !sections.length || !assignments.length}><Plus size={15}/> Add class</button></div>}
      </div>

      <div className="timetable-filters">
        <label>Academic year
          <Select ariaLabel="Academic year" value={String(yearID || "")} placeholder="Select academic year" onChange={value => { setYearID(Number(value)); setSectionID(0); }} options={activeYears.map(year => ({ value: String(year.id), label: year.name + (year.is_current ? " · Current" : "") }))} />
        </label>
        {canManage && canReadSections && <label>Class / section
          <Select ariaLabel="Class / section" value={String(sectionID || "")} placeholder="All sections" onChange={value => setSectionID(Number(value))} options={[{ value: "0", label: "All sections" }, ...sections.map(section => ({ value: String(section.id), label: `${section.class_name} · Section ${section.name}` }))]} />
        </label>}
        <button type="button" className="timetable-refresh" onClick={() => void loadSchedule(yearID)} disabled={loading}><RefreshCw size={15}/> Refresh</button>
      </div>

      {feedback && <div className={"timetable-feedback " + feedback.kind} role="status">{feedback.message}</div>}
      {error ? <div className="timetable-state error"><b>Could not load timetable</b><span>{error}</span><button type="button" onClick={() => void loadSchedule(yearID)}>Try again</button></div>
        : loading ? <div className="timetable-state"><span className="timetable-spinner"/>Loading timetable…</div>
        : !selectedYear ? <div className="timetable-state"><b>No academic year available</b><span>Create an academic year before setting up a timetable.</span></div>
        : periods.length === 0 ? <div className="timetable-state"><CalendarClock size={24}/><b>No periods configured</b><span>{canManage ? "Add the school's teaching periods and breaks to begin." : "The school has not configured timetable periods for this academic year yet."}</span></div>
        : visibleEntries.length === 0 ? <div className="timetable-state"><CalendarClock size={24}/><b>No classes scheduled yet</b><span>{canManage ? "Add a class to a teaching period to start building the weekly timetable." : "No timetable entries are assigned to your account for this academic year."}</span></div>
        : <div className="timetable-table-wrap"><table className="timetable-grid">
          <thead><tr><th>Period</th>{weekdays.map(day => <th key={day}>{day}</th>)}</tr></thead>
          <tbody>{orderedPeriods.map(period => <tr key={period.id} className={period.is_break ? "break-row" : ""}>
            <th><b>{period.name}</b><span>{period.starts_at}–{period.ends_at}</span></th>
            {weekdays.map((day,index) => {
              const cell = visibleEntries.filter(entry => entry.period_id === period.id && entry.day_of_week === index + 1);
              return <td key={day}>{period.is_break ? <span className="break-label">Break</span> : cell.map(entry => <article className="timetable-entry" key={entry.id}>
                <b>{entry.subject_name}</b><span>{entry.class_name} · {entry.section_name}</span><small>{entry.teacher_username}{entry.room ? ` · ${entry.room}` : ""}</small>
                {canManage && <button type="button" aria-label={`Remove ${entry.subject_name} from ${day}`} onClick={() => void removeEntry(entry)}><Trash2 size={13}/></button>}
              </article>)}</td>;
            })}
          </tr>)}</tbody>
        </table></div>}

      {canManage && <div className="timetable-note">Changes are checked against active teacher-subject assignments. A section or teacher cannot be scheduled twice in the same period.</div>}

      <Modal open={dialog === "period"} title="Add timetable period" description="Define a school-wide teaching period or break for this academic year." onClose={closeDialog} closeDisabled={saving}>
        <form className="timetable-form" onSubmit={submitPeriod}>
          <label>Period number<input type="number" min="1" step="1" value={periodDraft.period_number} placeholder="e.g. 1" onChange={event => { setPeriodDraft(current => ({ ...current, period_number: event.target.value })); setFieldErrors(current => ({ ...current, period_number: "" })); }} aria-invalid={Boolean(fieldErrors.period_number)} required/>{fieldErrors.period_number && <small role="alert">{fieldErrors.period_number}</small>}</label>
          <label>Period name<input value={periodDraft.name} placeholder="e.g. Period 1 or Lunch Break" maxLength={80} onChange={event => { setPeriodDraft(current => ({ ...current, name: event.target.value })); setFieldErrors(current => ({ ...current, name: "" })); }} aria-invalid={Boolean(fieldErrors.name)} required/>{fieldErrors.name && <small role="alert">{fieldErrors.name}</small>}</label>
          <div className="timetable-time-fields">
            <label>Starts at<input type="time" value={periodDraft.starts_at} placeholder="e.g. 08:00" onChange={event => { setPeriodDraft(current => ({ ...current, starts_at: event.target.value })); setFieldErrors(current => ({ ...current, starts_at: "" })); }} aria-invalid={Boolean(fieldErrors.starts_at)} required/>{fieldErrors.starts_at && <small role="alert">{fieldErrors.starts_at}</small>}</label>
            <label>Ends at<input type="time" value={periodDraft.ends_at} placeholder="e.g. 08:45" onChange={event => { setPeriodDraft(current => ({ ...current, ends_at: event.target.value })); setFieldErrors(current => ({ ...current, ends_at: "" })); }} aria-invalid={Boolean(fieldErrors.ends_at)} required/>{fieldErrors.ends_at && <small role="alert">{fieldErrors.ends_at}</small>}</label>
          </div>
          <label className="timetable-checkbox"><input type="checkbox" checked={periodDraft.is_break} onChange={event => setPeriodDraft(current => ({ ...current, is_break: event.target.checked }))}/> This is a break period</label>
          <div className="timetable-form-actions"><button type="button" className="timetable-secondary" onClick={closeDialog} disabled={saving}>Cancel</button><button type="submit" className="timetable-primary" disabled={saving}>{saving ? "Saving…" : "Save period"}</button></div>
        </form>
      </Modal>

      <Modal open={dialog === "entry"} title="Add class to timetable" description="Choose an active teacher-subject assignment and a free weekly slot." onClose={closeDialog} closeDisabled={saving}>
        <form className="timetable-form" onSubmit={submitEntry}>
          <label>Class / section<Select ariaLabel="Class / section" value={entryDraft.section_id} onChange={value => { setEntryDraft(current => ({ ...current, section_id: value, assignment_id: "" })); setFieldErrors(current => ({ ...current, section_id: "", assignment_id: "" })); }} options={sections.map(section => ({ value: String(section.id), label: `${section.class_name} · Section ${section.name}` }))} placeholder="Select section" invalid={Boolean(fieldErrors.section_id)}/>{fieldErrors.section_id && <small role="alert">{fieldErrors.section_id}</small>}</label>
          <label>Teacher and subject assignment<Select ariaLabel="Teacher and subject assignment" value={entryDraft.assignment_id} onChange={value => { setEntryDraft(current => ({ ...current, assignment_id: value })); setFieldErrors(current => ({ ...current, assignment_id: "" })); }} options={filteredAssignments.map(item => ({ value: String(item.id), label: `${item.subject_name} · ${item.teacher_username}` }))} placeholder="Select assigned teacher and subject" disabled={!entryDraft.section_id} invalid={Boolean(fieldErrors.assignment_id)}/>{fieldErrors.assignment_id && <small role="alert">{fieldErrors.assignment_id}</small>}</label>
          <label>Period<Select ariaLabel="Teaching period" value={entryDraft.period_id} onChange={value => { setEntryDraft(current => ({ ...current, period_id: value })); setFieldErrors(current => ({ ...current, period_id: "" })); }} options={periods.filter(period => !period.is_break).map(period => ({ value: String(period.id), label: `${period.period_number}. ${period.name} · ${period.starts_at}–${period.ends_at}` }))} placeholder="Select period" invalid={Boolean(fieldErrors.period_id)}/>{fieldErrors.period_id && <small role="alert">{fieldErrors.period_id}</small>}</label>
          <label>Day of week<Select ariaLabel="Day of week" value={entryDraft.day_of_week} onChange={value => setEntryDraft(current => ({ ...current, day_of_week: value }))} options={weekdays.map((day,index) => ({ value: String(index+1), label: day }))}/>{fieldErrors.day_of_week && <small role="alert">{fieldErrors.day_of_week}</small>}</label>
          <label>Room (optional)<input value={entryDraft.room} maxLength={80} onChange={event => setEntryDraft(current => ({ ...current, room: event.target.value }))} placeholder="e.g. Room 12"/></label>
          <div className="timetable-form-actions"><button type="button" className="timetable-secondary" onClick={closeDialog} disabled={saving}>Cancel</button><button type="submit" className="timetable-primary" disabled={saving}>{saving ? "Saving…" : "Add to timetable"}</button></div>
        </form>
      </Modal>
    </section>
  );
}
