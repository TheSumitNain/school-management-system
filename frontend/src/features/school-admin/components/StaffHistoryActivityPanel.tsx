import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getStaffAuditEvents, type StaffRecord } from "../api";

type Props = { staff: StaffRecord; permissions: string[] };

const labels: Record<string, string> = {
  "staff.account.created": "Account created",
  "staff.profile.updated": "Profile updated",
  "staff.account.updated": "Account credentials updated",
  "staff.account.disabled": "Account disabled",
  "staff.account.enabled": "Account enabled",
  "staff.password.reset": "Password reset",
  "staff.password.changed": "Password changed",
  "school.staff.role_assigned": "Role assigned",
  "school.staff.role_revoked": "Role removed",
  "school.staff.permission_override.set": "Individual permission override saved",
  "school.staff.permission_override.removed": "Individual permission override removed",
  "staff.qualification.created": "Qualification added",
  "staff.qualification.updated": "Qualification updated",
  "staff.certification.created": "Certification added",
  "staff.certification.updated": "Certification updated",
  "staff.experience.created": "Previous experience added",
  "staff.experience.updated": "Previous experience updated",
  "school.staff_attendance.marked": "Staff attendance marked",
  "school.staff_attendance.updated": "Staff attendance updated",
  "staff.leave.requested": "Leave requested",
  "staff.leave.approved": "Leave approved",
  "staff.leave.rejected": "Leave rejected",
  "staff.leave.cancelled": "Leave request cancelled",
  "staff.substitution.assigned": "Class substitution assigned",
  "staff.substitution.cancelled": "Class substitution cancelled",
  "school.teacher_subject_assignment.created": "Subject teaching assigned",
  "school.teacher_subject_assignment.revoked": "Subject teaching removed",
  "school.class_incharge_assignment.created": "Class in-charge assigned",
  "school.class_incharge_assignment.revoked": "Class in-charge removed",
};

function dateTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Date not available" : new Intl.DateTimeFormat("en-IN", { dateStyle: "medium", timeStyle: "short", timeZone: "Asia/Kolkata" }).format(date);
}

export function StaffHistoryActivityPanel({ staff, permissions }: Props) {
  const allowed = permissions.includes("school.audit.read");
  const pageSize = 50;
  const [offset, setOffset] = useState(0);
  useEffect(() => setOffset(0), [staff.principal_id]);
  const query = useQuery({
    queryKey: ["staff-audit-events", staff.principal_id, offset],
    queryFn: ({ signal }) => getStaffAuditEvents(staff.principal_id, { limit: pageSize, offset }, signal),
    enabled: allowed,
  });

  if (!allowed) return <section className="staff-records-panel"><h3>History & activity</h3><p className="staff-record-empty">You do not have permission to view audit history.</p></section>;

  return <section className="staff-records-panel">
    <div className="staff-records-heading">
      <div><h3>History & activity</h3><p>Audited account, profile, role, academic assignment, attendance, leave and substitution events for {staff.full_name || staff.username}. Showing 50 events per page.</p></div>
      <button type="button" className="staff-secondary-button" onClick={() => void query.refetch()} disabled={query.isFetching}>Refresh</button>
    </div>
    {query.isLoading && <p role="status">Loading staff activity…</p>}
    {query.isError && <div className="staff-record-error" role="alert"><span>Unable to load staff activity.</span><button type="button" onClick={() => void query.refetch()}>Try again</button></div>}
    {query.isSuccess && query.data.events.length === 0 && <p className="staff-history-empty">No audited activity is recorded for this employee yet.</p>}
    {query.isSuccess && query.data.events.length > 0 && <>
      <div className="staff-history-list">{query.data.events.map(event => <article className="staff-history-item" key={event.id}>
        <div><strong>{labels[event.action] || event.action.replaceAll(".", " ").replaceAll("_", " ")}</strong><p>By {event.actor_username}</p></div>
        <div><span className={event.outcome === "success" ? "staff-record-status" : "staff-record-status inactive"}>{event.outcome === "success" ? "Successful" : "Failed"}</span><time dateTime={event.created_at}>{dateTime(event.created_at)}</time></div>
      </article>)}</div>
      <div className="staff-records-heading" aria-label="Audit history pagination">
        <button type="button" className="staff-secondary-button" onClick={() => setOffset(current => Math.max(0, current - pageSize))} disabled={offset===0 || query.isFetching}>Newer events</button>
        <span aria-live="polite">Page {Math.floor(offset/pageSize)+1}</span>
        <button type="button" className="staff-secondary-button" onClick={() => setOffset(current => current + pageSize)} disabled={!query.data.has_more || query.isFetching}>Older events</button>
      </div>
    </>}
  </section>;
}
