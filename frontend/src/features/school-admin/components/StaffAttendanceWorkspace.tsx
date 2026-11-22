import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw } from "lucide-react";
import { notifyToast } from "../../../shared/notifications";
import { DatePicker } from "../../../shared/components/DatePicker";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { schoolDateValue } from "../../../shared/date/schoolDate";
import {
  getDailyStaffAttendance, getMonthlyStaffAttendance, markStaffAttendance, markStaffAttendanceBulk, updateStaffAttendance,
  getStaffLeaveRequests, getSchoolHolidays,
  type StaffAttendanceRecord, type StaffLeaveRequest,
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

function csvCell(value: string | number | null | undefined) {
  let text = String(value ?? "");
  if (/^[\\t\\r ]*[=+\\-@]/.test(text)) text = "'" + text;
  return '"' + text.replace(/"/g, '""') + '"';
}

function downloadCSV(filename: string, rows: Array<Array<string | number | null | undefined>>) {
  const csv = rows.map(row => row.map(csvCell).join(",")).join("\\r\\n");
  const blob = new Blob(["\\uFEFF", csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export function StaffAttendanceWorkspace({ permissions }: Props) {
  const queryClient = useQueryClient();
  const today = schoolDateValue();
  const [view, setView] = useState<"daily" | "monthly">("daily");
  const [date, setDate] = useState(today);
  const [month, setMonth] = useState(today.slice(0, 7));
  const [drafts, setDrafts] = useState<Record<number, Draft>>({});
  const [pendingCorrection, setPendingCorrection] = useState<PendingCorrection | null>(null);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "unmarked" | Status>("all");
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
  const canReadLeaveContext = permissions.includes("staff.leave.read");
  const canReadHolidayContext = permissions.includes("staff.holidays.read");
  const approvedLeaveContext = useQuery({
    queryKey: ["staff-leave-context", date],
    queryFn: ({ signal }) => getStaffLeaveRequests({ status: "approved", onDate: date }, signal),
    enabled: canReadLeaveContext && view === "daily",
    staleTime: 10_000,
  });
  const holidayContext = useQuery({
    queryKey: ["school-holiday-context", date],
    queryFn: ({ signal }) => getSchoolHolidays(date, date, signal),
    enabled: canReadHolidayContext && view === "daily",
    staleTime: 30_000,
  });
  const monthly = useQuery({
    queryKey: ["staff-attendance", "monthly", month],
    queryFn: ({ signal }) => getMonthlyStaffAttendance(month, signal),
    enabled: canRead && view === "monthly",
  });
  const monthStart = month ? `${month}-01` : "";
  const monthEnd = month ? new Date(Date.UTC(Number(month.slice(0, 4)), Number(month.slice(5, 7)), 0)).toISOString().slice(0, 10) : "";
  const approvedMonthLeave = useQuery({
    queryKey: ["staff-leave-report", "approved", month],
    queryFn: ({ signal }) => getStaffLeaveRequests({ status: "approved", fromDate: monthStart, toDate: monthEnd }, signal),
    enabled: canReadLeaveContext && view === "monthly" && Boolean(monthStart && monthEnd),
    staleTime: 10_000,
  });
  const monthHolidays = useQuery({
    queryKey: ["school-holiday-report", month],
    queryFn: ({ signal }) => getSchoolHolidays(monthStart, monthEnd, signal),
    enabled: canReadHolidayContext && view === "monthly" && Boolean(monthStart && monthEnd),
    staleTime: 30_000,
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
  const approvedLeaveByStaff = useMemo(() => {
    const grouped = new Map<number, StaffLeaveRequest[]>();
    for (const item of approvedLeaveContext.data?.data ?? []) {
      const current = grouped.get(item.staff_principal_id) ?? [];
      current.push(item);
      grouped.set(item.staff_principal_id, current);
    }
    return grouped;
  }, [approvedLeaveContext.data]);
  const schoolHolidayNames = (holidayContext.data?.data ?? []).map(item => item.name);

  const monthlyReportReady = monthly.isSuccess &&
    (!canReadLeaveContext || approvedMonthLeave.isSuccess) &&
    (!canReadHolidayContext || monthHolidays.isSuccess);

  const exportMonthlyReport = () => {
    if (!monthlyReportReady || !monthly.data) return;
    const rows: Array<Array<string | number | null | undefined>> = [[
      "Record type", "Staff name", "Employee code", "Designation",
      "Present days", "Absent days", "Late days", "Half days", "Recorded days",
      "Date", "Attendance status", "Remarks", "Leave type", "Leave start date",
      "Leave end date", "Holiday date", "Holiday name",
    ]];

    for (const item of monthly.data.data) {
      rows.push([
        "Attendance summary", item.full_name, item.employee_code, item.designation,
        item.present_days, item.absent_days, item.late_days, item.half_days,
        item.present_days + item.absent_days + item.late_days + item.half_days,
        "", "", "", "", "", "", "", "",
      ]);
      for (const record of item.records) {
        rows.push([
          "Attendance entry", item.full_name, item.employee_code, item.designation,
          "", "", "", "", "", record.date,
          statuses.find(status => status.value === record.status)?.label ?? record.status,
          record.remarks ?? "", "", "", "", "", "",
        ]);
      }
    }

    for (const leave of approvedMonthLeave.data?.data ?? []) {
      rows.push([
        "Approved leave", leave.staff_name ?? "Staff", leave.employee_code, "",
        "", "", "", "", "", "", "", "", leave.leave_type_name ?? "Leave",
        leave.start_date, leave.end_date, "", "",
      ]);
    }
    for (const holiday of monthHolidays.data?.data ?? []) {
      rows.push([
        "School holiday", "", "", "", "", "", "", "", "", "", "", "", "",
        "", "", holiday.holiday_date, holiday.name,
      ]);
    }

    downloadCSV("staff-attendance-" + month + ".csv", rows);
  };
  const records = daily.data?.data ?? [];
  const visibleRecords = useMemo(() => {
    const term = search.trim().toLocaleLowerCase();
    return records.filter(item => {
      const matchesSearch = !term || [item.full_name, item.employee_code, item.designation].some(value => (value ?? "").toLocaleLowerCase().includes(term));
      const matchesStatus = statusFilter === "all" || (statusFilter === "unmarked" ? !item.status : item.status === statusFilter);
      return matchesSearch && matchesStatus;
    });
  }, [records, search, statusFilter]);
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
        <label><span>Filter by status</span><select aria-label="Filter staff attendance status" value={statusFilter} onChange={event => setStatusFilter(event.target.value as "all" | "unmarked" | Status)}>
          <option value="all">All statuses</option><option value="unmarked">Not marked</option>{statuses.map(status => <option key={status.value} value={status.value}>{status.label}</option>)}
        </select></label>
        <button type="button" className="staff-secondary-button" onClick={() => void daily.refetch()} disabled={daily.isFetching}><RefreshCw size={15}/> Refresh</button>
      </div>
      <div className="staff-attendance-totals" aria-live="polite">
        <span>Present <strong>{totals.present}</strong></span><span>Absent <strong>{totals.absent}</strong></span><span>Late <strong>{totals.late}</strong></span><span>Half day <strong>{totals.half_day}</strong></span><span>Not marked <strong>{totals.unmarked}</strong></span>
      </div>
      <div className="staff-attendance-bulk">
        <label><span>Bulk status for unmarked visible staff</span><select aria-label="Bulk attendance status" value={bulkStatus} disabled={!canMark || bulkMark.isPending} aria-invalid={bulkAttempted && !bulkStatus} aria-describedby={bulkAttempted && !bulkStatus ? "staff-attendance-bulk-status-error" : undefined} onChange={event => setBulkStatus(event.target.value as Status | "")}>
          <option value="">Select status</option>{statuses.map(status => <option key={status.value} value={status.value}>{status.label}</option>)}
        </select>{bulkAttempted && !bulkStatus && <small id="staff-attendance-bulk-status-error" className="staff-attendance-field-error">Select a bulk attendance status.</small>}</label>
        <label><span>Bulk remarks (optional)</span><input aria-label="Bulk attendance remarks" maxLength={500} value={bulkRemarks} disabled={!canMark || bulkMark.isPending} onChange={event => setBulkRemarks(event.target.value)} placeholder="Optional remarks for selected staff"/></label>
        <div><button type="button" className="staff-attendance-save" disabled={!canMark || bulkMark.isPending || bulkTargets.length === 0} onClick={() => { if (!bulkStatus) { setBulkAttempted(true); return; } setBulkConfirmOpen(true); }}>Mark {bulkTargets.length} unmarked staff</button><small>Only visible, unmarked employees are included. All records save together or none do.</small></div>
      </div>
      {past && !permissions.includes("staff.attendance.correct_past") && <p className="staff-attendance-note">Historical attendance is read-only. Past corrections require separate permission.</p>}
      {holidayContext.data?.data.length ? <div className="staff-attendance-calendar-note"><strong>School holiday:</strong> {schoolHolidayNames.join(", ")}. Attendance records are not changed automatically.</div> : null}
      {holidayContext.isError && <p className="staff-attendance-context-warning" role="status">School holiday context could not be loaded. Attendance marking remains available according to your permissions.</p>}
      {approvedLeaveContext.isError && <p className="staff-attendance-context-warning" role="status">Approved leave context could not be loaded. Check the leave read permission or retry the register.</p>}
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
            <td><strong>{item.full_name}</strong><small>{item.employee_code} · {item.designation}</small>
              {(approvedLeaveByStaff.get(item.staff_principal_id) ?? []).map(leave => <small className="staff-attendance-leave-context" key={leave.id}>Approved leave: {leave.leave_type_name ?? "Leave"}</small>)}
              {item.id && (approvedLeaveByStaff.has(item.staff_principal_id) || schoolHolidayNames.length > 0) && <small className="staff-attendance-context-warning">Existing attendance mark conflicts with calendar context — review manually.</small>}
            </td>
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
      <div className="staff-attendance-toolbar"><label><span>Month</span><input aria-label="Attendance month" type="month" value={month} max={today.slice(0, 7)} onChange={event => setMonth(event.target.value)}/></label><button type="button" className="staff-secondary-button" onClick={() => void Promise.all([monthly.refetch(), ...(canReadLeaveContext ? [approvedMonthLeave.refetch()] : []), ...(canReadHolidayContext ? [monthHolidays.refetch()] : [])])} disabled={monthly.isFetching || approvedMonthLeave.isFetching || monthHolidays.isFetching}><RefreshCw size={15}/> Refresh</button><button type="button" className="staff-attendance-save" onClick={exportMonthlyReport} disabled={!monthlyReportReady || monthly.data.data.length === 0}>Download monthly report</button></div>
      {monthly.isPending && <p role="status" className="staff-attendance-message">Loading monthly register…</p>}
      {monthly.isError && <div className="staff-attendance-message error" role="alert">Unable to load monthly register. <button type="button" onClick={() => void monthly.refetch()}>Try again</button></div>}
      {monthly.isSuccess && monthly.data.data.length === 0 && <p className="staff-attendance-message">No active staff profiles are available for this school.</p>}
      {monthly.isSuccess && monthly.data.data.length > 0 && <div className="staff-attendance-table-wrap"><table className="staff-attendance-table"><thead><tr><th>Employee</th><th>Present</th><th>Absent</th><th>Late</th><th>Half day</th><th>Recorded days</th></tr></thead><tbody>
        {monthly.data.data.map(item => <tr key={item.staff_principal_id}><td><strong>{item.full_name}</strong><small>{item.employee_code} · {item.designation}</small><details><summary>View daily entries ({item.records.length})</summary><ul>{item.records.map(record => <li key={record.id}>{record.date}: {statuses.find(status => status.value === record.status)?.label ?? record.status}{record.remarks ? ` — ${record.remarks}` : ""}</li>)}</ul></details></td><td>{item.present_days}</td><td>{item.absent_days}</td><td>{item.late_days}</td><td>{item.half_days}</td><td>{item.present_days + item.absent_days + item.late_days + item.half_days}</td></tr>)}
      </tbody></table></div>}
      {(canReadLeaveContext || canReadHolidayContext) && <section className="staff-attendance-calendar-note" aria-label="Monthly leave and holiday context">
        <strong>Calendar context · not included in attendance totals</strong>
        {canReadLeaveContext && <div>
          <h3>Approved leave</h3>
          {approvedMonthLeave.isPending && <p role="status">Loading approved leave…</p>}
          {approvedMonthLeave.isError && <p role="status">Approved leave context could not be loaded. Attendance totals remain unchanged.</p>}
          {approvedMonthLeave.isSuccess && approvedMonthLeave.data.data.length === 0 && <p>No approved leave overlaps this month.</p>}
          {approvedMonthLeave.data?.data.map(item => <p key={item.id}>{item.staff_name || "Staff"} · {item.leave_type_name || "Leave"} · {item.start_date}{item.end_date !== item.start_date ? ` → ${item.end_date}` : ""}</p>)}
        </div>}
        {canReadHolidayContext && <div>
          <h3>School holidays</h3>
          {monthHolidays.isPending && <p role="status">Loading school holidays…</p>}
          {monthHolidays.isError && <p role="status">School holiday context could not be loaded.</p>}
          {monthHolidays.isSuccess && monthHolidays.data.data.length === 0 && <p>No active school holidays in this month.</p>}
          {monthHolidays.data?.data.map(item => <p key={item.id}>{item.holiday_date} · {item.name}</p>)}
        </div>}
      </section>}
    </>}
    <ConfirmDialog
      open={bulkConfirmOpen}
      title="Confirm bulk staff attendance"
      description={`This will mark ${bulkTargets.length} unmarked employees on ${date} as ${statuses.find(status => status.value === bulkStatus)?.label ?? "the selected status"}. The batch is atomic: if any employee already has a mark or fails validation, no records will be saved.`}
      confirmLabel={`Mark ${bulkTargets.length} staff`}
      pending={bulkMark.isPending}
      onConfirm={() => { if (bulkStatus && bulkTargets.length > 0) bulkMark.mutate({ staff_principal_ids: bulkTargets.map(item => item.staff_principal_id), attendance_date: date, status: bulkStatus, remarks: bulkRemarks.trim() || undefined }); }}
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
