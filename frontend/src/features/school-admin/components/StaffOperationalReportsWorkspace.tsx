import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Download, RefreshCw } from "lucide-react";
import { getMonthlyStaffAttendance, getStaffLeaveRequests } from "../api";
import type { StaffAttendanceMonthlySummary, StaffLeaveRequest } from "../api";
import "./StaffOperationalReportsWorkspace.css";

type Props = { permissions: string[] };

function currentMonthIST() {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Kolkata", year: "numeric", month: "2-digit" }).formatToParts(new Date());
  const year = parts.find(part => part.type === "year")?.value ?? "2026";
  const month = parts.find(part => part.type === "month")?.value ?? "01";
  return year + "-" + month;
}

function csvCell(value: string | number | null | undefined) {
  const raw = String(value ?? "");
  const safe = /^\\s*[=+\\-@]/.test(raw) ? "'" + raw : raw;
  return '"' + safe.replace(/"/g, '""') + '"';
}

function exportCSV(filename: string, rows: Array<Array<string | number | null | undefined>>) {
  const content = rows.map(row => row.map(csvCell).join(",")).join("\\r\\n");
  const blob = new Blob(["\\uFEFF", content], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}

function monthBounds(month: string) {
  const [year, value] = month.split("-").map(Number);
  const lastDay = new Date(Date.UTC(year, value, 0)).getUTCDate();
  return { from: month + "-01", to: month + "-" + String(lastDay).padStart(2, "0") };
}

function AttendanceReport({ rows, month }: { rows: StaffAttendanceMonthlySummary[]; month: string }) {
  const totals = useMemo(() => rows.reduce((result, row) => ({
    present: result.present + row.present_days,
    absent: result.absent + row.absent_days,
    late: result.late + row.late_days,
    half: result.half + row.half_days,
  }), { present: 0, absent: 0, late: 0, half: 0 }), [rows]);

  return <section className="staff-operational-report-section">
    <div className="staff-operational-report-title">
      <div><h3>Monthly staff attendance</h3><p>Recorded attendance by employee; unmarked days are not treated as absence.</p></div>
      <button type="button" className="staff-operational-report-export" onClick={() => exportCSV("staff-attendance-" + month + ".csv", [
        ["Employee code", "Staff name", "Designation", "Present days", "Absent days", "Late days", "Half days"],
        ...rows.map(row => [row.employee_code, row.full_name, row.designation, row.present_days, row.absent_days, row.late_days, row.half_days]),
      ])} disabled={rows.length === 0}><Download size={14}/> Export attendance CSV</button>
    </div>
    <div className="staff-operational-report-metrics">
      <article><span>Present</span><strong>{totals.present}</strong></article>
      <article><span>Absent</span><strong>{totals.absent}</strong></article>
      <article><span>Late</span><strong>{totals.late}</strong></article>
      <article><span>Half days</span><strong>{totals.half}</strong></article>
    </div>
    {rows.length === 0 ? <p className="staff-operational-report-empty">No attendance records were returned for this month.</p> : <div className="staff-operational-report-table-wrap"><table><thead><tr><th>Employee</th><th>Code</th><th>Present</th><th>Absent</th><th>Late</th><th>Half day</th></tr></thead><tbody>
      {rows.map(row => <tr key={row.staff_principal_id}><td>{row.full_name}</td><td>{row.employee_code}</td><td>{row.present_days}</td><td>{row.absent_days}</td><td>{row.late_days}</td><td>{row.half_days}</td></tr>)}
    </tbody></table></div>}
  </section>;
}

function LeaveReport({ rows, month }: { rows: StaffLeaveRequest[]; month: string }) {
  const approved = rows.filter(row => row.status === "approved").length;
  const pending = rows.filter(row => row.status === "pending").length;
  const rejected = rows.filter(row => row.status === "rejected").length;
  return <section className="staff-operational-report-section">
    <div className="staff-operational-report-title">
      <div><h3>Leave requests</h3><p>Requests overlapping the selected month; balances and entitlement are not calculated.</p></div>
      <button type="button" className="staff-operational-report-export" onClick={() => exportCSV("staff-leave-requests-" + month + ".csv", [
        ["Employee code", "Staff name", "Leave type", "Start date", "End date", "Duration", "Status"],
        ...rows.map(row => [row.employee_code, row.staff_name, row.leave_type_name, row.start_date, row.end_date, row.duration_type, row.status]),
      ])} disabled={rows.length === 0}><Download size={14}/> Export leave CSV</button>
    </div>
    <div className="staff-operational-report-metrics">
      <article><span>Approved</span><strong>{approved}</strong></article>
      <article><span>Pending</span><strong>{pending}</strong></article>
      <article><span>Rejected</span><strong>{rejected}</strong></article>
      <article><span>Total requests</span><strong>{rows.length}</strong></article>
    </div>
    {rows.length === 0 ? <p className="staff-operational-report-empty">No leave requests overlap this month.</p> : <div className="staff-operational-report-table-wrap"><table><thead><tr><th>Employee</th><th>Leave type</th><th>From</th><th>To</th><th>Duration</th><th>Status</th></tr></thead><tbody>
      {rows.map(row => <tr key={row.id}><td>{row.staff_name || row.employee_code || "Staff member"}</td><td>{row.leave_type_name || "—"}</td><td>{row.start_date}</td><td>{row.end_date}</td><td>{row.duration_type === "half_day" ? "Half day" : "Full day"}</td><td><span className={"staff-operational-report-status " + row.status}>{row.status}</span></td></tr>)}
    </tbody></table></div>}
  </section>;
}

export function StaffOperationalReportsWorkspace({ permissions }: Props) {
  const canReadAttendance = permissions.includes("staff.attendance.read");
  const canReadLeave = permissions.includes("staff.leave.read");
  const [month, setMonth] = useState(currentMonthIST);
  const bounds = useMemo(() => monthBounds(month), [month]);
  const attendance = useQuery({
    queryKey: ["staff-operational-report", "attendance", month],
    queryFn: ({ signal }) => getMonthlyStaffAttendance(month, signal),
    enabled: canReadAttendance,
  });
  const leave = useQuery({
    queryKey: ["staff-operational-report", "leave", month],
    queryFn: ({ signal }) => getStaffLeaveRequests({ fromDate: bounds.from, toDate: bounds.to }, signal),
    enabled: canReadLeave,
  });

  return <section className="staff-operational-reports">
    <div className="staff-operational-reports-toolbar">
      <div><h2>Attendance & leave reports</h2><p>Monthly operational records, loaded only when the viewer has the matching read permission.</p></div>
      <label>Report month<input type="month" value={month} onChange={event => setMonth(event.target.value)} aria-label="Select attendance and leave report month"/></label>
    </div>
    {!canReadAttendance && !canReadLeave && <p className="staff-operational-report-denied" role="status">Attendance and leave reports are hidden because you do not have their read permissions.</p>}
    {canReadAttendance && <div className="staff-operational-report-panel">
      <div className="staff-operational-report-panel-heading"><strong>Attendance</strong><button type="button" onClick={() => void attendance.refetch()} disabled={attendance.isFetching}><RefreshCw size={14}/> Refresh</button></div>
      {attendance.isLoading && <p role="status">Loading attendance report…</p>}
      {attendance.isError && <p className="staff-operational-report-error" role="alert">Attendance report could not be loaded. <button type="button" onClick={() => void attendance.refetch()}>Retry</button></p>}
      {attendance.isSuccess && <AttendanceReport rows={attendance.data.data} month={month}/>}
    </div>}
    {canReadLeave && <div className="staff-operational-report-panel">
      <div className="staff-operational-report-panel-heading"><strong>Leave</strong><button type="button" onClick={() => void leave.refetch()} disabled={leave.isFetching}><RefreshCw size={14}/> Refresh</button></div>
      {leave.isLoading && <p role="status">Loading leave report…</p>}
      {leave.isError && <p className="staff-operational-report-error" role="alert">Leave report could not be loaded. <button type="button" onClick={() => void leave.refetch()}>Retry</button></p>}
      {leave.isSuccess && <LeaveReport rows={leave.data.data} month={month}/>}
    </div>}
  </section>;
}
