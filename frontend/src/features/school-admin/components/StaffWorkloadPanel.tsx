import { useCallback, useEffect, useState } from "react";
import { getCurrentTimetableYear, getTimetableEntries, getTimetablePeriods, type TimetableEntry, type TimetablePeriod } from "../api";
import type { StaffRecord } from "../api";

type Props = { staff: StaffRecord };

const weekdays = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];

export function StaffWorkloadPanel({ staff }: Props) {
  const [entries, setEntries] = useState<TimetableEntry[]>([]);
  const [periods, setPeriods] = useState<TimetablePeriod[]>([]);
  const [yearName, setYearName] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError("");
    try {
      const current = await getCurrentTimetableYear(signal);
      const [result, periodResult] = await Promise.all([
        getTimetableEntries(current.academic_year.id, 0, signal, staff.principal_id),
        getTimetablePeriods(current.academic_year.id, signal),
      ]);
      setYearName(current.academic_year.name);
      setEntries(result.entries);
      setPeriods(periodResult.periods);
    } catch (cause) {
      if (signal?.aborted) return;
      setError(cause instanceof Error ? cause.message : "Unable to load this staff member's timetable.");
      setEntries([]);
      setPeriods([]);
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
  const teachingPeriods = periods.filter(period => !period.is_break);
  const unscheduledSlots = Math.max(0, teachingPeriods.length * weekdays.length - entries.length);

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
            <article><span>Unscheduled slots</span><strong>{unscheduledSlots}</strong></article>
          </div>
          {periods.length === 0 ? <p className="staff-record-empty">No timetable periods are configured for this academic year.</p>
            : <div className="staff-workload-grid-wrap"><table className="staff-workload-grid">
              <thead><tr><th>Period</th>{weekdays.map(day => <th key={day}>{day}</th>)}</tr></thead>
              <tbody>{periods.map(period => <tr key={period.id} className={period.is_break ? "break" : ""}>
                <th scope="row"><strong>{period.period_number}. {period.name}</strong><small>{period.starts_at}–{period.ends_at}</small></th>
                {weekdays.map((day, index) => {
                  const entry = entries.find(item => item.period_id === period.id && item.day_of_week === index + 1);
                  return <td key={day} className={period.is_break ? "break" : entry ? "scheduled" : "unscheduled"}>
                    {period.is_break ? <span>Break</span> : entry ? <span><strong>{entry.subject_name}</strong><small>{entry.class_name} · {entry.section_name}</small></span> : <span>Unscheduled</span>}
                  </td>;
                })}
              </tr>)}</tbody>
            </table></div>}
          {entries.length === 0 && periods.length > 0 && <p className="staff-record-empty">No timetable entries are scheduled for this staff member in the current academic year.</p>}
          <p className="staff-workload-note">Unscheduled timetable slots are shown for planning context only. They do not confirm teacher availability, working days, or permission to assign another class.</p>
        </>}
    </section>
  );
}
