import { Fragment, useMemo, useState, type FormEvent } from "react";
import { KeyRound, LockKeyhole, Pencil, Plus, Search, Shield, ShieldCheck, UserCheck, UserMinus } from "lucide-react";
import { StaffRoleAssignments } from "../components/StaffRoleAssignments";
import { StaffPermissionOverrides } from "../components/StaffPermissionOverrides";
import {
  useCreateSchoolStaffMutation, useDisableSchoolStaffMutation, useEnableSchoolStaffMutation,
  useResetSchoolStaffPasswordMutation, useSchoolStaffQuery, useUpdateSchoolStaffMutation,
} from "../queries";
import type { StaffRecord } from "../api";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import "./StaffManagementPage.css";

type Action = "create" | "edit" | "reset" | "enable" | null;
type Props = { permissions: string[] };

export function StaffManagementPage({ permissions }: Props) {
  const canCreate = permissions.includes("users.create");
  const canUpdate = permissions.includes("users.update");
  const canDisable = permissions.includes("users.disable");
  const canReset = permissions.includes("users.password_reset");
  const canReadRoles = permissions.includes("roles.read");
  const canAssignRoles = permissions.includes("roles.assign");
  const canManageOverrides = permissions.includes("permissions.override");
  const query = useSchoolStaffQuery();
  const create = useCreateSchoolStaffMutation();
  const update = useUpdateSchoolStaffMutation();
  const disable = useDisableSchoolStaffMutation();
  const enable = useEnableSchoolStaffMutation();
  const reset = useResetSchoolStaffPasswordMutation();
  const [search, setSearch] = useState("");
  const [action, setAction] = useState<Action>(null);
  const [selected, setSelected] = useState<StaffRecord | null>(null);
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [rolesForPrincipal, setRolesForPrincipal] = useState<number | null>(null);
  const [overridesForPrincipal, setOverridesForPrincipal] = useState<number | null>(null);
  const [staffToDisable, setStaffToDisable] = useState<StaffRecord | null>(null);

  const staff = query.data?.staff ?? [];
  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return staff.filter(item => !term || item.username.toLowerCase().includes(term) || item.status.toLowerCase().includes(term));
  }, [staff, search]);

  function openAction(next: Action, item: StaffRecord | null = null) {
    setSelected(item);
    setAction(next);
    setShowPassword(false);
    setFieldErrors({});
    setFeedback(null);
  }
  function closeAction() {
    setAction(null);
    setSelected(null);
    setShowPassword(false);
    setFieldErrors({});
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const nextErrors: Record<string, string> = {};
    const username = String(form.get("username") ?? "").trim();
    const password = String(form.get("password") ?? "");
    if ((action === "create" || action === "edit") && !/^[a-z0-9._-]{3,32}$/.test(username)) nextErrors.username = "Use 3–32 lowercase letters, numbers, dots, underscores or hyphens.";
    if (action !== "edit" && password.length < 12) nextErrors.password = "Password must contain at least 12 characters.";
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;
    setFeedback(null);
    try {
      if (action === "create") {
        await create.mutateAsync({ username, initial_password: password });
        setFeedback({ kind: "success", message: "Staff account created. The staff member must change the initial password after signing in." });
      } else if (action === "edit" && selected) {
        await update.mutateAsync({ principalID: selected.principal_id, username });
        setFeedback({ kind: "success", message: "Staff username updated. Their active sessions have been revoked." });
      } else if (action === "reset" && selected) {
        await reset.mutateAsync({ principalID: selected.principal_id, newPassword: password });
        setFeedback({ kind: "success", message: "Password reset. The staff member must change it at next sign-in." });
      } else if (action === "enable" && selected) {
        await enable.mutateAsync({ principalID: selected.principal_id, temporaryPassword: password });
        setFeedback({ kind: "success", message: "Staff account enabled. A password change is required at next sign-in." });
      }
      closeAction();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "The request could not be completed." });
    }
  }
  function deactivate(item: StaffRecord) { setStaffToDisable(item); }
  async function confirmDeactivate() {
    if (!staffToDisable) return;
    const item = staffToDisable;
    setFeedback(null);
    try {
      await disable.mutateAsync(item.principal_id);
      setStaffToDisable(null);
      setFeedback({ kind: "success", message: `${item.username} has been disabled and active sessions revoked.` });
    } catch (error) {
      setStaffToDisable(null);
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to disable this account." });
    }
  }

  const pending = create.isPending || update.isPending || reset.isPending || enable.isPending;
  const formTitle = action === "create" ? "Create staff account" : action === "edit" ? "Update username" : action === "reset" ? "Reset staff password" : "Enable staff account";
  const passwordLabel = action === "create" ? "Initial password" : action === "reset" ? "New temporary password" : "Temporary password";

  return (
    <div className="content staff-management-page">
      <div className="heading">
        <div><span className="eyebrow">PEOPLE</span><h1>Staff accounts</h1><p>Create and manage the staff accounts belonging to this school.</p></div>
        {canCreate && <button className="primary staff-create-button" type="button" onClick={() => openAction("create")}><Plus size={16}/> Add staff</button>}
      </div>

      {feedback && <div className={`staff-feedback ${feedback.kind}`} role="status">{feedback.message}</div>}

      <Modal
        open={action !== null}
        title={formTitle}
        description={selected ? `Account: ${selected.username}. Check the account carefully before making this change.` : "Staff usernames must be unique across the platform."}
        onClose={closeAction}
        closeDisabled={pending}
      >
        {feedback?.kind === "error" && <p className="staff-feedback error" role="alert">{feedback.message}</p>}
        <form onSubmit={submit} noValidate>
          {(action === "create" || action === "edit") && <label>Username<input name="username" defaultValue={selected?.username ?? ""} minLength={3} maxLength={32} pattern="[a-z0-9._-]{3,32}" autoComplete="off" required aria-invalid={Boolean(fieldErrors.username)} aria-describedby={fieldErrors.username ? "staff-username-error" : "staff-username-help"}/><small id="staff-username-help">3–32 lowercase letters, numbers, dots, underscores or hyphens.</small>{fieldErrors.username && <small className="app-field-error" id="staff-username-error">{fieldErrors.username}</small>}</label>}
          {action !== "edit" && <label>{passwordLabel}<div className="staff-password-field"><input name="password" type={showPassword ? "text" : "password"} minLength={12} autoComplete="new-password" required aria-invalid={Boolean(fieldErrors.password)} aria-describedby={fieldErrors.password ? "staff-password-error" : "staff-password-help"}/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Hide password" : "Show password"}>{showPassword ? "Hide" : "Show"}</button></div><small id="staff-password-help">At least 12 characters. The server also checks password safety and compromised-password status.</small>{fieldErrors.password && <small className="app-field-error" id="staff-password-error">{fieldErrors.password}</small>}</label>}
          <div className="staff-form-actions"><button className="primary" type="submit" disabled={pending}>{pending ? "Saving…" : action === "reset" ? "Reset password" : action === "enable" ? "Enable account" : "Save changes"}</button><button className="staff-secondary-button" type="button" onClick={closeAction} disabled={pending}>Cancel</button></div>
        </form>
      </Modal>

      <section className="table-panel">
        <div className="toolbar"><div><h2>All staff</h2><small>{staff.length} accounts</small></div><div className="search"><Search size={16}/><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search username or status" aria-label="Search staff"/></div></div>
        {query.isPending ? <div className="table-message" aria-live="polite">Loading staff accounts…</div>
          : query.isError ? <div className="table-message error-text">{query.error instanceof Error ? query.error.message : "Unable to load staff accounts."}<button className="staff-retry-button" type="button" onClick={() => void query.refetch()}>Try again</button></div>
          : filtered.length === 0 ? <div className="table-message">{search ? "No staff match this search." : "No staff accounts yet. Add the first staff account to get started."}</div>
          : <div className="table-scroll"><table><thead><tr><th>Username</th><th>Status</th><th>Password</th><th>Created</th><th>Actions</th></tr></thead><tbody>
            {filtered.map(item => <Fragment key={item.principal_id}><tr><td><b>{item.username}</b><small className="staff-id">Principal #{item.principal_id}</small></td><td><span className={`pill ${item.status === "active" ? "active" : "inactive"}`}>{item.status}</span></td><td>{item.must_change_password ? <span className="staff-password-required">Change required</span> : "—"}</td><td>{item.created_at ? new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(item.created_at)) : "—"}</td><td><div className="staff-row-actions">
              {canUpdate && <button type="button" onClick={() => openAction("edit", item)} title="Edit username" aria-label={`Edit ${item.username}`}><Pencil size={15}/></button>}
              {canReset && item.status === "active" && <button type="button" onClick={() => openAction("reset", item)} title="Reset password" aria-label={`Reset password for ${item.username}`}><KeyRound size={15}/></button>}
              {canReadRoles && item.status === "active" && <button type="button" className={rolesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setRolesForPrincipal(current => current === item.principal_id ? null : item.principal_id)} title="Manage roles" aria-label={`Manage roles for ${item.username}`} aria-expanded={rolesForPrincipal === item.principal_id}><Shield size={15}/></button>}
              {canManageOverrides && item.status === "active" && <button type="button" className={overridesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setOverridesForPrincipal(current => current === item.principal_id ? null : item.principal_id)} title="Manage permission overrides" aria-label={`Manage permission overrides for ${item.username}`} aria-expanded={overridesForPrincipal === item.principal_id}><LockKeyhole size={15}/></button>}
              {item.status === "active" && canDisable && <button type="button" className="danger" onClick={() => void deactivate(item)} disabled={disable.isPending} title="Disable account" aria-label={`Disable ${item.username}`}><UserMinus size={15}/></button>}
              {item.status !== "active" && canUpdate && <button type="button" onClick={() => openAction("enable", item)} title="Enable account" aria-label={`Enable ${item.username}`}><UserCheck size={15}/></button>}
            </div></td></tr>
            {(rolesForPrincipal === item.principal_id || overridesForPrincipal === item.principal_id) && <tr className="staff-role-expanded-row"><td colSpan={5}>{rolesForPrincipal === item.principal_id && canReadRoles && <StaffRoleAssignments staff={item} canAssign={canAssignRoles}/>}{overridesForPrincipal === item.principal_id && canManageOverrides && <StaffPermissionOverrides staff={item}/>}</td></tr>}</Fragment>)}
          </tbody></table></div>}
      </section>
      <p className="staff-security-note"><ShieldCheck size={15}/> Every action is checked again by the server using your current school permissions.</p>
      <ConfirmDialog open={Boolean(staffToDisable)} title="Disable staff account?" description={staffToDisable ? `Disable ${staffToDisable.username}? Their active sessions will be revoked and they will no longer be able to sign in.` : ""} confirmLabel="Disable account" pending={disable.isPending} onCancel={() => setStaffToDisable(null)} onConfirm={() => void confirmDeactivate()}/>
    </div>
  );
}
