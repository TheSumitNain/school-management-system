import { useCallback, useEffect, useState } from "react";
import { getCurrentTimetableYear, getTimetableEntries, type TimetableEntry } from "../api";
import type { StaffRecord } from "../api";

type Props = { staff: StaffRecord };

const weekdays = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

export function StaffWorkloadPanel({ staff }: Props) {
  const [entries, setEntries] = useState<TimetableEntry[]>([]);
  const [yearName, setYearName] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");
    try {
      const current = await getCurrentTimetableYear(signal);
      const result = await getTimetableEntries(current.academic_year.id, 0, signal, staff.principal_id);
      setYearName(current.academic_year.name);
      setEntries(result.entries);
    } catch (cause) {
      if (signal?.aborted) return;
      setError(cause instanceof Error ? cause.message : "Unable to load this staff member's timetable.");
      setEntries([]);
    } finally {
      if (!signal?.aborted) setLoading(false);
    }
  }, [staff.principal_id]);

  useEffect(() => {
    const controller = new AbortController();
    void load(controller.signal);
    return () => controller.abort();
  }, [load]);

  const teachingDays = new Set(entries.map(entry => entry.day_of_week)).size;
  const sections = new Set(entries.map(entry => entry.section_id)).size;
  const subjects = new Set(entries.map(entry => entry.subject_id)).size;

  return (
    <section className="staff-workload-panel">
      <div className="staff-records-heading">
        <div><h3>Teaching workload</h3><p>{yearName ? `Scheduled timetable · ${yearName}` : "Scheduled timetable for the current academic year."}</p></div>
        <button className="staff-secondary-button" type="button" onClick={() => void load()} disabled={loading}>Refresh</button>
      </div>
      {loading ? <p className="staff-record-empty" role="status">Loading teacher-scoped timetable…</p>
        : error ? <div className="staff-record-error" role="alert"><span>{error}</span><button type="button" onClick={() => void load()}>Retry</button></div>
        : <>
          <div className="staff-workload-summary">
            <article><span>Scheduled periods</span><strong>{entries.length}</strong></article>
            <article><span>Teaching days</span><strong>{teachingDays}</strong></article>
            <article><span>Sections</span><strong>{sections}</strong></article>
            <article><span>Subjects</span><strong>{subjects}</strong></article>
          </div>
          {entries.length === 0 ? <p className="staff-record-empty">No timetable entries are scheduled for this staff member in the current academic year.</p>
            : <div className="staff-workload-table-wrap"><table className="staff-workload-table">
              <thead><tr><th>Day</th><th>Time</th><th>Class & section</th><th>Subject</th><th>Period</th><th>Room</th></tr></thead>
              <tbody>{entries.map(entry => <tr key={entry.id}>
                <td>{weekdays[entry.day_of_week - 1] ?? "—"}</td>
                <td>{entry.starts_at}–{entry.ends_at}</td>
                <td>{entry.class_name} · {entry.section_name}</td>
                <td>{entry.subject_name}</td>
                <td>{entry.period_name}</td>
                <td>{entry.room || "Not assigned"}</td>
              </tr>)}</tbody>
            </table></div>}
          <p className="staff-workload-note">This view shows only scheduled timetable entries returned for this staff member. An unscheduled period is not treated as confirmed availability.</p>
        </>}
    </section>
  );
}
