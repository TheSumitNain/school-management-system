import { useMemo, useState, type FormEvent } from "react";
import "./StaffRoleAssignments.css";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Shield, Trash2 } from "lucide-react";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import {
  assignStaffRole, getSchoolRoles, getStaffRoleAssignments, revokeStaffRole,
  type StaffRecord,
} from "../api";

type Props = { staff: StaffRecord; canAssign: boolean };

export function StaffRoleAssignments({ staff, canAssign }: Props) {
  const queryClient = useQueryClient();
  const [roleID, setRoleID] = useState("");
  const [feedback, setFeedback] = useState("");
  const [roleToRevoke, setRoleToRevoke] = useState<{id:number;name:string}|null>(null);
  const [roleError, setRoleError] = useState("");
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
  function submitAssign(event: FormEvent<HTMLFormElement>) { event.preventDefault(); if(!roleID){setRoleError("Choose a role to assign.");return;} setRoleError("");setFeedback("");assign.mutate(); }

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
            {canAssign && item.role_key !== "school_admin" && <button type="button" disabled={busy} aria-label={`Remove ${item.name} role`} title="Remove role" onClick={() => setRoleToRevoke({id:item.role_id,name:item.name})}><Trash2 size={15}/></button>}
          </li>)}</ul> : <p className="staff-role-message">No additional roles assigned.</p>}
          {canAssign && <form className="staff-role-assign" onSubmit={submitAssign}>
            <label htmlFor={`role-select-${staff.principal_id}`}>Assign a role</label>
            <div><select id={`role-select-${staff.principal_id}`} value={roleID} onChange={event => {setRoleID(event.target.value);setRoleError("");}} disabled={busy || availableRoles.length === 0} required aria-invalid={Boolean(roleError)} aria-describedby={roleError?`role-select-error-${staff.principal_id}`:undefined}>
              <option value="">Choose an available role</option>
              {availableRoles.map(role => <option key={role.id} value={role.id}>{role.name}{role.is_system ? " (system)" : " (custom)"}</option>)}
            </select>{roleError&&<small className="app-field-error" id={`role-select-error-${staff.principal_id}`} role="alert">{roleError}</small>}<button className="primary" type="submit" disabled={busy || availableRoles.length === 0}><Check size={15}/>{assign.isPending ? "Assigning…" : "Assign"}</button></div>
            {availableRoles.length === 0 && <small>There are no other assignable roles available.</small>}
          </form>}
        </>}
      {(feedback || assign.isError || revoke.isError) && <p className={`staff-role-feedback ${assign.isError || revoke.isError ? "error-text" : ""}`}>
        {assign.isError ? (assign.error instanceof Error ? assign.error.message : "Unable to assign role.")
          : revoke.isError ? (revoke.error instanceof Error ? revoke.error.message : "Unable to remove role.") : feedback}
      </p>}
      {!canAssign && <p className="staff-role-message">You can view assigned roles but cannot change them.</p>}
      <ConfirmDialog open={roleToRevoke !== null} title="Remove staff role?" description={roleToRevoke ? `Remove the ${roleToRevoke.name} role from ${staff.username}? Their effective permissions will update on the next protected request.` : ""} confirmLabel="Remove role" pending={revoke.isPending} onCancel={() => setRoleToRevoke(null)} onConfirm={() => {if(roleToRevoke){setFeedback("");revoke.mutate(roleToRevoke.id);setRoleToRevoke(null);}}}/>
    </section>
  );
}
