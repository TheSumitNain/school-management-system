import { useMemo, useState, type FormEvent } from "react";
import { KeyRound, Pencil, Plus, Search, Shield, ShieldCheck, UserCheck, UserMinus, X } from "lucide-react";
import { StaffRoleAssignments } from "../components/StaffRoleAssignments";
import {
  useCreateSchoolStaffMutation, useDisableSchoolStaffMutation, useEnableSchoolStaffMutation,
  useResetSchoolStaffPasswordMutation, useSchoolStaffQuery, useUpdateSchoolStaffMutation,
} from "../queries";
import type { StaffRecord } from "../api";
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
  const [rolesForPrincipal, setRolesForPrincipal] = useState<number | null>(null);

  const staff = query.data?.staff ?? [];
  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return staff.filter(item => !term || item.username.toLowerCase().includes(term) || item.status.toLowerCase().includes(term));
  }, [staff, search]);

  function openAction(next: Action, item: StaffRecord | null = null) {
    setSelected(item);
    setAction(next);
    setShowPassword(false);
    setFeedback(null);
  }
  function closeAction() {
    setAction(null);
    setSelected(null);
    setShowPassword(false);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFeedback(null);
    try {
      if (action === "create") {
        await create.mutateAsync({ username: String(form.get("username") ?? "").trim(), initial_password: String(form.get("password") ?? "") });
        setFeedback({ kind: "success", message: "Staff account created. The staff member must change the initial password after signing in." });
      } else if (action === "edit" && selected) {
        await update.mutateAsync({ principalID: selected.principal_id, username: String(form.get("username") ?? "").trim() });
        setFeedback({ kind: "success", message: "Staff username updated. Their active sessions have been revoked." });
      } else if (action === "reset" && selected) {
        await reset.mutateAsync({ principalID: selected.principal_id, newPassword: String(form.get("password") ?? "") });
        setFeedback({ kind: "success", message: "Password reset. The staff member must change it at next sign-in." });
      } else if (action === "enable" && selected) {
        await enable.mutateAsync({ principalID: selected.principal_id, temporaryPassword: String(form.get("password") ?? "") });
        setFeedback({ kind: "success", message: "Staff account enabled. A password change is required at next sign-in." });
      }
      closeAction();
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "The request could not be completed." });
    }
  }
  async function deactivate(item: StaffRecord) {
    if (!window.confirm(`Disable staff account "${item.username}"? Their active sessions will be revoked.`)) return;
    setFeedback(null);
    try {
      await disable.mutateAsync(item.principal_id);
      setFeedback({ kind: "success", message: `${item.username} has been disabled and active sessions revoked.` });
    } catch (error) {
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

      {action && (
        <section className="staff-action-panel" aria-labelledby="staff-action-title">
          <div className="staff-action-heading"><div><h2 id="staff-action-title">{formTitle}</h2><p>{selected ? `Account: ${selected.username}` : "Staff usernames must be unique across the platform."}</p></div><button type="button" className="staff-icon-button" onClick={closeAction} aria-label="Close form"><X size={18}/></button></div>
          <form onSubmit={submit}>
            {(action === "create" || action === "edit") && <label>Username<input name="username" defaultValue={selected?.username ?? ""} minLength={3} maxLength={32} pattern="[a-z0-9._-]{3,32}" title="Use 3–32 lowercase letters, numbers, dots, underscores or hyphens." autoComplete="off" required/><small>3–32 lowercase letters, numbers, dots, underscores or hyphens.</small></label>}
            {action !== "edit" && <label>{passwordLabel}<div className="staff-password-field"><input name="password" type={showPassword ? "text" : "password"} minLength={12} autoComplete="new-password" required/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Hide password" : "Show password"}>{showPassword ? "Hide" : "Show"}</button></div><small>At least 12 characters. The server also checks password safety and compromised-password status.</small></label>}
            <div className="staff-form-actions"><button className="primary" type="submit" disabled={pending}>{pending ? "Saving…" : "Save changes"}</button><button className="staff-secondary-button" type="button" onClick={closeAction} disabled={pending}>Cancel</button></div>
          </form>
        </section>
      )}

      <section className="table-panel">
        <div className="toolbar"><div><h2>All staff</h2><small>{staff.length} accounts</small></div><div className="search"><Search size={16}/><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search username or status" aria-label="Search staff"/></div></div>
        {query.isPending ? <div className="table-message" aria-live="polite">Loading staff accounts…</div>
          : query.isError ? <div className="table-message error-text">{query.error instanceof Error ? query.error.message : "Unable to load staff accounts."}<button className="staff-retry-button" type="button" onClick={() => void query.refetch()}>Try again</button></div>
          : filtered.length === 0 ? <div className="table-message">{search ? "No staff match this search." : "No staff accounts yet. Add the first staff account to get started."}</div>
          : <div className="table-scroll"><table><thead><tr><th>Username</th><th>Status</th><th>Password</th><th>Created</th><th>Actions</th></tr></thead><tbody>
            {filtered.map(item => <tr key={item.principal_id}><td><b>{item.username}</b><small className="staff-id">Principal #{item.principal_id}</small></td><td><span className={`pill ${item.status === "active" ? "active" : "inactive"}`}>{item.status}</span></td><td>{item.must_change_password ? <span className="staff-password-required">Change required</span> : "—"}</td><td>{item.created_at ? new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(item.created_at)) : "—"}</td><td><div className="staff-row-actions">
              {canUpdate && <button type="button" onClick={() => openAction("edit", item)} title="Edit username" aria-label={`Edit ${item.username}`}><Pencil size={15}/></button>}
              {canReset && item.status === "active" && <button type="button" onClick={() => openAction("reset", item)} title="Reset password" aria-label={`Reset password for ${item.username}`}><KeyRound size={15}/></button>}
              {canReadRoles && item.status === "active" && <button type="button" className={rolesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setRolesForPrincipal(current => current === item.principal_id ? null : item.principal_id)} title="Manage roles" aria-label={`Manage roles for ${item.username}`} aria-expanded={rolesForPrincipal === item.principal_id}><Shield size={15}/></button>}
              {item.status === "active" && canDisable && <button type="button" className="danger" onClick={() => void deactivate(item)} disabled={disable.isPending} title="Disable account" aria-label={`Disable ${item.username}`}><UserMinus size={15}/></button>}
              {item.status !== "active" && canUpdate && <button type="button" onClick={() => openAction("enable", item)} title="Enable account" aria-label={`Enable ${item.username}`}><UserCheck size={15}/></button>}
            </div></td></tr>
            {rolesForPrincipal === item.principal_id && <tr className="staff-role-expanded-row"><td colSpan={5}><StaffRoleAssignments staff={item} canAssign={canAssignRoles}/></td></tr>})}
          </tbody></table></div>}
      </section>
      <p className="staff-security-note"><ShieldCheck size={15}/> Every action is checked again by the server using your current school permissions.</p>
    </div>
  );
}
