import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { notifyToast } from "../../../shared/notifications";
import { DatePicker } from "../../../shared/components/DatePicker";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { schoolDateValue } from "../../../shared/date/schoolDate";
import {
  getDailyStaffAttendance, getMonthlyStaffAttendance, markStaffAttendance, markStaffAttendanceBulk, updateStaffAttendance,
  type StaffAttendanceRecord,
} from "../api";

type Props = { permissions: string[] };
type Status = "present" | "absent" | "late" | "half_day";
type Draft = { status: Status | ""; remarks: string; attempted: boolean };
type PendingCorrection = { item: StaffAttendanceRecord; draft: Draft };
const statuses: Array<{ value: Status; label: string }> = [
  { value: "present", label: "Present" },
  { value: "absent", label: "Absent" },
  { value: "late", label: "Late" },
  { value: "half_day", label: "Half day" },
];

export function StaffAttendanceWorkspace({ permissions }: Props) {
  const queryClient = useQueryClient();
  const today = schoolDateValue();
  const [view, setView] = useState<"daily" | "monthly">("daily");
  const [date, setDate] = useState(today);
  const [month, setMonth] = useState(today.slice(0, 7));
  const [drafts, setDrafts] = useState<Record<number, Draft>>({});
  const [pendingCorrection, setPendingCorrection] = useState<PendingCorrection | null>(null);
  const [search, setSearch] = useState("");
  const [bulkStatus, setBulkStatus] = useState<Status | "">("");
  const [bulkRemarks, setBulkRemarks] = useState("");
  const [bulkAttempted, setBulkAttempted] = useState(false);
  const [bulkConfirmOpen, setBulkConfirmOpen] = useState(false);
  const canRead = permissions.includes("staff.attendance.read");
  const daily = useQuery({
    queryKey: ["staff-attendance", "daily", date],
    queryFn: ({ signal }) => getDailyStaffAttendance(date, signal),
    enabled: canRead && view === "daily",
  });
  const monthly = useQuery({
    queryKey: ["staff-attendance", "monthly", month],
    queryFn: ({ signal }) => getMonthlyStaffAttendance(month, signal),
    enabled: canRead && view === "monthly",
  });
  const invalidate = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["staff-attendance", "daily"] }),
      queryClient.invalidateQueries({ queryKey: ["staff-attendance", "monthly"] }),
    ]);
  };
  const mark = useMutation({
    mutationFn: markStaffAttendance,
    onSuccess: async () => { setPendingCorrection(null); notifyToast("success", "Staff attendance marked."); await invalidate(); },
    onError: () => notifyToast("error", "Attendance could not be marked. Check permissions or whether it was already recorded."),
  });
  const update = useMutation({
    mutationFn: (input: { id: number; status: Status; remarks?: string }) => updateStaffAttendance(input.id, { status: input.status, remarks: input.remarks }),
    onSuccess: async () => { setPendingCorrection(null); notifyToast("success", "Staff attendance updated."); await invalidate(); },
    onError: () => notifyToast("error", "Attendance could not be updated. Check the correction permission."),
  });
  const bulkMark = useMutation({
    mutationFn: markStaffAttendanceBulk,
    onSuccess: async result => {
      setBulkConfirmOpen(false);
      setBulkStatus("");
      setBulkRemarks("");
      setBulkAttempted(false);
      notifyToast("success", `Attendance marked for ${result.created} staff members.`);
      await invalidate();
    },
    onError: () => notifyToast("error", "Bulk attendance was not saved. No partial records were committed; check for duplicate marks or permission changes."),
  });
  const past = date < today;
  const canMark = past ? permissions.includes("staff.attendance.correct_past") : permissions.includes("staff.attendance.mark");
  const canUpdate = past ? permissions.includes("staff.attendance.correct_past") : permissions.includes("staff.attendance.update_same_day");
  const records = daily.data?.data ?? [];
  const visibleRecords = useMemo(() => {
    const term = search.trim().toLocaleLowerCase();
    if (!term) return records;
    return records.filter(item => [item.full_name, item.employee_code, item.designation].some(value => value.toLocaleLowerCase().includes(term)));
  }, [records, search]);
  const bulkTargets = visibleRecords.filter(item => !item.id);
  const totals = useMemo(() => records.reduce((acc, item) => {
    if (item.status) acc[item.status] += 1;
    else acc.unmarked += 1;
    return acc;
  }, { present: 0, absent: 0, late: 0, half_day: 0, unmarked: 0 }), [records]);

  const draftFor = (item: StaffAttendanceRecord): Draft => drafts[item.staff_principal_id] ?? {
    status: item.status ?? "",
    remarks: item.remarks ?? "",
    attempted: false,
  };
  const changeDraft = (id: number, patch: Partial<Draft>, item: StaffAttendanceRecord) => {
    setDrafts(current => ({ ...current, [id]: { ...draftFor(item), ...patch } }));
  };
  const save = (item: StaffAttendanceRecord) => {
    const draft = draftFor(item);
    if (!draft.status) {
      changeDraft(item.staff_principal_id, { attempted: true }, item);
      return;
    }
    if (past) {
      setPendingCorrection({ item, draft });
      return;
    }
    submit(item, draft);
  };
  const submit = (item: StaffAttendanceRecord, draft: Draft) => {
    if (!draft.status) return;
    const remarks = draft.remarks.trim() || undefined;
    if (item.id) update.mutate({ id: item.id, status: draft.status, remarks });
    else mark.mutate({ staff_principal_id: item.staff_principal_id, attendance_date: date, status: draft.status, remarks });
  };

  if (!canRead) return <section className="staff-attendance-panel"><h2>Staff attendance</h2><p role="status">You do not have permission to view staff attendance.</p></section>;

  return <section className="staff-attendance-panel">
    <div className="staff-attendance-heading">
      <div><h2>Staff attendance</h2><p>Daily register and monthly summary for this school. Staff attendance is separate from student attendance.</p></div>
      <div className="staff-attendance-view-toggle">
        <button type="button" aria-pressed={view === "daily"} className={view === "daily" ? "active" : ""} onClick={() => setView("daily")}>Daily register</button>
        <button type="button" aria-pressed={view === "monthly"} className={view === "monthly" ? "active" : ""} onClick={() => setView("monthly")}>Monthly summary</button>
      </div>
    </div>
    {view === "daily" && <>
      <div className="staff-attendance-toolbar">
        <label><span>Attendance date</span><DatePicker value={date} onChange={setDate} ariaLabel="Staff attendance date" max={today}/></label>
        <label><span>Search employees</span><input aria-label="Search staff attendance" value={search} onChange={event => setSearch(event.target.value)} placeholder="Name, employee code or designation"/></label>
        <button type="button" className="staff-secondary-button" onClick={() => void daily.refetch()} disabled={daily.isFetching}><RefreshCw size={15}/> Refresh</button>
      </div>
      <div className="staff-attendance-totals" aria-live="polite">
        <span>Present <strong>{totals.present}</strong></span><span>Absent <strong>{totals.absent}</strong></span><span>Late <strong>{totals.late}</strong></span><span>Half day <strong>{totals.half_day}</strong></span><span>Not marked <strong>{totals.unmarked}</strong></span>
      </div>
      <div className="staff-attendance-bulk">
        <label><span>Bulk status for unmarked visible staff</span><select aria-label="Bulk attendance status" value={bulkStatus} disabled={!canMark || bulkMutation.isPending} aria-invalid={bulkAttempted && !bulkStatus} aria-describedby={bulkAttempted && !bulkStatus ? "staff-attendance-bulk-status-error" : undefined} onChange={event => setBulkStatus(event.target.value as Status | "")}>
          <option value="">Select status</option>{statuses.map(status => <option key={status.value} value={status.value}>{status.label}</option>)}
        </select>{bulkAttempted && !bulkStatus && <small id="staff-attendance-bulk-status-error" className="staff-attendance-field-error">Select a bulk attendance status.</small>}</label>
        <label><span>Bulk remarks (optional)</span><input aria-label="Bulk attendance remarks" maxLength={500} value={bulkRemarks} disabled={!canMark || bulkMutation.isPending} onChange={event => setBulkRemarks(event.target.value)} placeholder="Optional remarks for selected staff"/></label>
        <div><button type="button" className="staff-attendance-save" disabled={!canMark || bulkMutation.isPending || bulkTargets.length === 0} onClick={() => { if (!bulkStatus) { setBulkAttempted(true); return; } setBulkConfirmOpen(true); }}>Mark {bulkTargets.length} unmarked staff</button><small>Only visible, unmarked employees are included. All records save together or none do.</small></div>
      </div>
      {past && !permissions.includes("staff.attendance.correct_past") && <p className="staff-attendance-note">Historical attendance is read-only. Past corrections require separate permission.</p>}
      {daily.isPending && <p role="status" className="staff-attendance-message">Loading staff register…</p>}
      {daily.isError && <div className="staff-attendance-message error" role="alert">Unable to load staff attendance. <button type="button" onClick={() => void daily.refetch()}>Try again</button></div>}
      {daily.isSuccess && records.length === 0 && <p className="staff-attendance-message">No active staff profiles are available for this school.</p>}
      {daily.isSuccess && records.length > 0 && visibleRecords.length === 0 && <p className="staff-attendance-message">No staff match this search.</p>}
      {daily.isSuccess && visibleRecords.length > 0 && <div className="staff-attendance-table-wrap"><table className="staff-attendance-table">
        <thead><tr><th>Employee</th><th>Attendance</th><th>Remarks</th><th>Action</th></tr></thead>
        <tbody>{visibleRecords.map(item => {
          const draft = draftFor(item);
          const busy = mark.isPending || update.isPending;
          const allowed = item.id ? canUpdate : canMark;
          const unchanged = Boolean(item.id && draft.status === item.status && draft.remarks.trim() === (item.remarks ?? ""));
          return <tr key={item.staff_principal_id}>
            <td><strong>{item.full_name}</strong><small>{item.employee_code} · {item.designation}</small></td>
            <td><select aria-label={`Attendance status for ${item.full_name}`} value={draft.status} disabled={!allowed || busy} aria-invalid={draft.attempted && !draft.status} aria-describedby={draft.attempted && !draft.status ? `staff-attendance-status-error-${item.staff_principal_id}` : undefined} onChange={event => changeDraft(item.staff_principal_id, { status: event.target.value as Status | "" }, item)}>
              <option value="">Select status</option>{statuses.map(status => <option key={status.value} value={status.value}>{status.label}</option>)}
            </select>{draft.attempted && !draft.status && <small id={`staff-attendance-status-error-${item.staff_principal_id}`} className="staff-attendance-field-error">Select an attendance status.</small>}
              {item.status && <small className="staff-attendance-current">Currently: {statuses.find(status => status.value === item.status)?.label}</small>}
            </td>
            <td><input aria-label={`Remarks for ${item.full_name}`} maxLength={500} value={draft.remarks} disabled={!allowed || busy} onChange={event => changeDraft(item.staff_principal_id, { remarks: event.target.value }, item)} placeholder="Optional remarks"/></td>
            <td><button type="button" className="staff-attendance-save" disabled={!allowed || busy || unchanged} onClick={() => save(item)}>{item.id ? "Update" : "Mark"}</button>{!allowed && <small className="staff-attendance-field-error">No permission</small>}</td>
          </tr>;
        })}</tbody>
      </table></div>}
    </>}
    {view === "monthly" && <>
      <div className="staff-attendance-toolbar"><label><span>Month</span><input aria-label="Attendance month" type="month" value={month} max={today.slice(0, 7)} onChange={event => setMonth(event.target.value)}/></label><button type="button" className="staff-secondary-button" onClick={() => void monthly.refetch()} disabled={monthly.isFetching}><RefreshCw size={15}/> Refresh</button></div>
      {monthly.isPending && <p role="status" className="staff-attendance-message">Loading monthly register…</p>}
      {monthly.isError && <div className="staff-attendance-message error" role="alert">Unable to load monthly register. <button type="button" onClick={() => void monthly.refetch()}>Try again</button></div>}
      {monthly.isSuccess && monthly.data.data.length === 0 && <p className="staff-attendance-message">No active staff profiles are available for this school.</p>}
      {monthly.isSuccess && monthly.data.data.length > 0 && <div className="staff-attendance-table-wrap"><table className="staff-attendance-table"><thead><tr><th>Employee</th><th>Present</th><th>Absent</th><th>Late</th><th>Half day</th><th>Recorded days</th></tr></thead><tbody>
        {monthly.data.data.map(item => <tr key={item.staff_principal_id}><td><strong>{item.full_name}</strong><small>{item.employee_code} · {item.designation}</small><details><summary>View daily entries ({item.records.length})</summary><ul>{item.records.map(record => <li key={record.id}>{record.date}: {statuses.find(status => status.value === record.status)?.label ?? record.status}{record.remarks ? ` — ${record.remarks}` : ""}</li>)}</ul></details></td><td>{item.present_days}</td><td>{item.absent_days}</td><td>{item.late_days}</td><td>{item.half_days}</td><td>{item.present_days + item.absent_days + item.late_days + item.half_days}</td></tr>)}
      </tbody></table></div>}
    </>}
    <ConfirmDialog
      open={bulkConfirmOpen}
      title="Confirm bulk staff attendance"
      description={`This will mark ${bulkTargets.length} unmarked employees on ${date} as ${statuses.find(status => status.value === bulkStatus)?.label ?? "the selected status"}. The batch is atomic: if any employee already has a mark or fails validation, no records will be saved.`}
      confirmLabel={`Mark ${bulkTargets.length} staff`}
      pending={bulkMutation.isPending}
      onConfirm={() => { if (bulkStatus && bulkTargets.length > 0) bulkMutation.mutate({ staff_principal_ids: bulkTargets.map(item => item.staff_principal_id), attendance_date: date, status: bulkStatus, remarks: bulkRemarks.trim() || undefined }); }}
      onCancel={() => setBulkConfirmOpen(false)}
    />
    <ConfirmDialog
      open={pendingCorrection !== null}
      title="Confirm historical attendance correction"
      description={pendingCorrection ? `This changes attendance for ${pendingCorrection.item.full_name} on ${date} to ${statuses.find(status => status.value === pendingCorrection.draft.status)?.label ?? "the selected status"}. This action will be recorded in the audit history.` : ""}
      confirmLabel="Confirm correction"
      pending={mark.isPending || update.isPending}
      onConfirm={() => { if (pendingCorrection) submit(pendingCorrection.item, pendingCorrection.draft); }}
      onCancel={() => setPendingCorrection(null)}
    />
  </section>;
}
