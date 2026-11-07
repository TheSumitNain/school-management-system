import { useMemo, useState } from "react";
import "./StaffRoleAssignments.css";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Shield, Trash2 } from "lucide-react";
import {
  assignStaffRole, getSchoolRoles, getStaffRoleAssignments, revokeStaffRole,
  type StaffRecord,
} from "../api";

type Props = { staff: StaffRecord; canAssign: boolean };

export function StaffRoleAssignments({ staff, canAssign }: Props) {
  const queryClient = useQueryClient();
  const [roleID, setRoleID] = useState("");
  const [feedback, setFeedback] = useState("");
  const rolesQuery = useQuery({
    queryKey: ["school-admin", "roles"],
    queryFn: ({ signal }) => getSchoolRoles(signal),
    staleTime: 30_000,
  });
  const assignmentsQuery = useQuery({
    queryKey: ["school-admin", "staff-roles", staff.principal_id],
    queryFn: ({ signal }) => getStaffRoleAssignments(staff.principal_id, signal),
    staleTime: 15_000,
  });
  const assign = useMutation({
    mutationFn: () => assignStaffRole(staff.principal_id, Number(roleID)),
    onSuccess: async () => {
      setRoleID("");
      setFeedback("Role assigned. New permissions apply on the next protected request.");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["school-admin", "staff-roles", staff.principal_id] }),
        queryClient.invalidateQueries({ queryKey: ["school-admin", "staff"] }),
      ]);
    },
  });
  const revoke = useMutation({
    mutationFn: (id: number) => revokeStaffRole(staff.principal_id, id),
    onSuccess: async () => {
      setFeedback("Role removed. Updated permissions apply on the next protected request.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "staff-roles", staff.principal_id] });
    },
  });
  const assignments = assignmentsQuery.data?.roles ?? [];
  const availableRoles = useMemo(
    () => (rolesQuery.data?.roles ?? []).filter(role =>
      role.role_key !== "school_admin" && !assignments.some(assigned => assigned.role_id === role.id)),
    [rolesQuery.data?.roles, assignments],
  );
  const busy = assign.isPending || revoke.isPending;

  return (
    <section className="staff-role-panel" aria-label={`Roles for ${staff.username}`}>
      <div className="staff-role-panel-title"><Shield size={16}/><div><b>Access roles</b><small>Roles currently assigned to {staff.username}</small></div></div>
      {rolesQuery.isPending || assignmentsQuery.isPending ? <p className="staff-role-message">Loading role access…</p>
        : rolesQuery.isError || assignmentsQuery.isError ? <div className="staff-role-message error-text">
          {rolesQuery.error instanceof Error ? rolesQuery.error.message : assignmentsQuery.error instanceof Error ? assignmentsQuery.error.message : "Unable to load role access."}
          <button type="button" onClick={() => { void rolesQuery.refetch(); void assignmentsQuery.refetch(); }}>Retry</button>
        </div>
        : <>
          {assignments.length ? <ul className="staff-role-list">{assignments.map(item => <li key={item.role_id}>
            <span><b>{item.name}</b><small>{item.is_system ? "System role" : "Custom role"} · {item.role_key}</small></span>
            {canAssign && item.role_key !== "school_admin" && <button type="button" disabled={busy} aria-label={`Remove ${item.name} role`} title="Remove role" onClick={() => {
              if (window.confirm(`Remove the ${item.name} role from ${staff.username}?`)) {
                setFeedback("");
                revoke.mutate(item.role_id);
              }
            }}><Trash2 size={15}/></button>}
          </li>)}</ul> : <p className="staff-role-message">No additional roles assigned.</p>}
          {canAssign && <form className="staff-role-assign" onSubmit={event => { event.preventDefault(); setFeedback(""); assign.mutate(); }}>
            <label htmlFor={`role-select-${staff.principal_id}`}>Assign a role</label>
            <div><select id={`role-select-${staff.principal_id}`} value={roleID} onChange={event => setRoleID(event.target.value)} disabled={busy || availableRoles.length === 0}>
              <option value="">Choose an available role</option>
              {availableRoles.map(role => <option key={role.id} value={role.id}>{role.name}{role.is_system ? " (system)" : " (custom)"}</option>)}
            </select><button className="primary" type="submit" disabled={busy || !roleID}><Check size={15}/>{assign.isPending ? "Assigning…" : "Assign"}</button></div>
            {availableRoles.length === 0 && <small>There are no other assignable roles available.</small>}
          </form>}
        </>}
      {(feedback || assign.isError || revoke.isError) && <p className={`staff-role-feedback ${assign.isError || revoke.isError ? "error-text" : ""}`}>
        {assign.isError ? (assign.error instanceof Error ? assign.error.message : "Unable to assign role.")
          : revoke.isError ? (revoke.error instanceof Error ? revoke.error.message : "Unable to remove role.") : feedback}
      </p>}
      {!canAssign && <p className="staff-role-message">You can view assigned roles but cannot change them.</p>}
    </section>
  );
}
