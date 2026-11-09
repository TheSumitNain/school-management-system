import { useMemo, useState, type FormEvent } from "react";
import "./StaffPermissionOverrides.css";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LockKeyhole, Trash2 } from "lucide-react";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import {
  deleteStaffPermissionOverride, getSchoolPermissionDelegations, getSchoolPermissions,
  getStaffPermissionOverrides, setStaffPermissionOverride, type StaffRecord,
} from "../api";

type Props = { staff: StaffRecord };

export function StaffPermissionOverrides({ staff }: Props) {
  const queryClient = useQueryClient();
  const [permissionKey, setPermissionKey] = useState("");
  const [effect, setEffect] = useState<"allow" | "deny">("deny");
  const [reason, setReason] = useState("");
  const [feedback, setFeedback] = useState("");
  const [overrideToRemove, setOverrideToRemove] = useState<{key:string;effect:string}|null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string,string>>({});
  const permissionsQuery = useQuery({
    queryKey: ["school-admin", "permissions"],
    queryFn: ({ signal }) => getSchoolPermissions(signal),
    staleTime: 60_000,
  });
  const delegationsQuery = useQuery({
    queryKey: ["school-admin", "permission-delegations"],
    queryFn: ({ signal }) => getSchoolPermissionDelegations(signal),
    staleTime: 30_000,
  });
  const overridesQuery = useQuery({
    queryKey: ["school-admin", "staff-overrides", staff.principal_id],
    queryFn: ({ signal }) => getStaffPermissionOverrides(staff.principal_id, signal),
    staleTime: 15_000,
  });
  const activeDelegations = delegationsQuery.data?.delegations ?? [];
  const availablePermissions = useMemo(() => {
    const delegated = new Set(activeDelegations.filter(item => item.can_override_users).map(item => item.permission_key));
    return (permissionsQuery.data?.permissions ?? []).filter(item => item.is_delegable && delegated.has(item.permission_key));
  }, [permissionsQuery.data?.permissions, activeDelegations]);
  const overrides = overridesQuery.data?.overrides ?? [];
  const selectedDelegation = activeDelegations.find(item => item.permission_key === permissionKey && item.can_override_users);
  const save = useMutation({
    mutationFn: () => setStaffPermissionOverride(staff.principal_id, {
      permission_key: permissionKey, effect, reason: reason.trim(), delegation_id: selectedDelegation!.id,
    }),
    onSuccess: async () => {
      setReason("");
      setFeedback("Permission override saved. It takes effect on the next protected request.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "staff-overrides", staff.principal_id] });
    },
  });
  const remove = useMutation({
    mutationFn: (key: string) => deleteStaffPermissionOverride(staff.principal_id, key),
    onSuccess: async () => {
      setOverrideToRemove(null);
      setFeedback("Permission override removed. Effective permissions update on the next protected request.");
      await queryClient.invalidateQueries({ queryKey: ["school-admin", "staff-overrides", staff.principal_id] });
    },
  });
  const busy = save.isPending || remove.isPending;
  const loading = permissionsQuery.isPending || delegationsQuery.isPending || overridesQuery.isPending;
  const error = permissionsQuery.error ?? delegationsQuery.error ?? overridesQuery.error;
  function submitOverride(event: FormEvent<HTMLFormElement>) { event.preventDefault(); const errors:Record<string,string>={}; if(!permissionKey||!selectedDelegation)errors.permissionKey="Choose a permission delegated for individual overrides."; if(!reason.trim())errors.reason="Explain why this permission exception is needed."; setFieldErrors(errors); if(Object.keys(errors).length)return;setFeedback("");save.mutate(); }

  return (
    <section className="staff-override-panel" aria-label={`Permission overrides for ${staff.username}`}>
      <div className="staff-override-title"><LockKeyhole size={16}/><div><b>Individual permission overrides</b><small>Exceptions to this staff member's assigned roles</small></div></div>
      {loading ? <p className="staff-override-message">Loading permissions…</p>
        : error ? <div className="staff-override-message error-text">{error instanceof Error ? error.message : "Unable to load permission settings."}
          <button type="button" onClick={() => { void permissionsQuery.refetch(); void delegationsQuery.refetch(); void overridesQuery.refetch(); }}>Retry</button>
        </div>
        : <>
          {overrides.length ? <ul className="staff-override-list">{overrides.map(item => <li key={item.id}>
            <span><b>{item.permission_key}</b><small className={item.effect === "allow" ? "override-allow" : "override-deny"}>{item.effect.toUpperCase()} · {item.reason}</small></span>
            <button type="button" disabled={busy} title="Remove override" aria-label={`Remove ${item.permission_key} override`} onClick={() => {remove.reset();setOverrideToRemove({key:item.permission_key,effect:item.effect});}}><Trash2 size={15}/></button>
          </li>)}</ul> : <p className="staff-override-message">No individual permission overrides are configured.</p>}
          <form className="staff-override-form" onSubmit={submitOverride} noValidate>
            <label htmlFor={`override-permission-${staff.principal_id}`}>Delegated permission</label>
            <select id={`override-permission-${staff.principal_id}`} value={permissionKey} onChange={event => {setPermissionKey(event.target.value);setFieldErrors(current=>({...current,permissionKey:""}));}} disabled={busy || availablePermissions.length === 0} required aria-invalid={Boolean(fieldErrors.permissionKey)} aria-describedby={fieldErrors.permissionKey?`override-permission-error-${staff.principal_id}`:undefined}>
              <option value="">Choose a delegated permission</option>
              {availablePermissions.map(item => <option key={item.permission_key} value={item.permission_key}>{item.permission_key} — {item.description}</option>)}
            </select>
            {fieldErrors.permissionKey&&<small className="app-field-error" id={`override-permission-error-${staff.principal_id}`} role="alert">{fieldErrors.permissionKey}</small>}
            <label htmlFor={`override-effect-${staff.principal_id}`}>Effect</label>
            <select id={`override-effect-${staff.principal_id}`} value={effect} onChange={event => setEffect(event.target.value as "allow" | "deny")} disabled={busy}>
              <option value="deny">Deny — remove effective access</option>
              <option value="allow">Allow — grant delegated access</option>
            </select>
            <label htmlFor={`override-reason-${staff.principal_id}`}>Reason (required)</label>
            <textarea id={`override-reason-${staff.principal_id}`} value={reason} onChange={event => {setReason(event.target.value);setFieldErrors(current=>({...current,reason:""}));}} maxLength={1000} rows={2} placeholder="Explain why this exception is needed" required disabled={busy} aria-invalid={Boolean(fieldErrors.reason)} aria-describedby={fieldErrors.reason?`override-reason-error-${staff.principal_id}`:undefined}/>
            {fieldErrors.reason&&<small className="app-field-error" id={`override-reason-error-${staff.principal_id}`} role="alert">{fieldErrors.reason}</small>}
            <button className="primary" type="submit" disabled={busy || availablePermissions.length === 0}>{save.isPending ? "Saving…" : "Save override"}</button>
            {availablePermissions.length === 0 && <small>No permissions have been delegated by the Platform Admin for individual overrides.</small>}
          </form>
        </>}
      {(feedback || save.isError || remove.isError) && <p className={`staff-override-feedback ${save.isError || remove.isError ? "error-text" : ""}`}>
        {save.isError ? (save.error instanceof Error ? save.error.message : "Unable to save permission override.")
          : remove.isError ? (remove.error instanceof Error ? remove.error.message : "Unable to remove permission override.") : feedback}
      </p>}
      <ConfirmDialog open={overrideToRemove !== null} title="Remove permission override?" description={overrideToRemove ? `Remove the ${overrideToRemove.effect.toUpperCase()} override for ${overrideToRemove.key} from ${staff.username}? Effective access will be recalculated on the next protected request.` : ""} confirmLabel="Remove override" pending={remove.isPending} error={remove.isError?(remove.error instanceof Error?remove.error.message:"Unable to remove this override. Please try again."):undefined} onCancel={() => setOverrideToRemove(null)} onConfirm={() => {if(overrideToRemove){setFeedback("");remove.mutate(overrideToRemove.key);}}}/>
    </section>
  );
}
