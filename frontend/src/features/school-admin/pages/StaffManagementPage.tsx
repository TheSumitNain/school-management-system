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
  const [disableError, setDisableError] = useState("");

  const staff = query.data?.staff ?? [];
  const filtered = useMemo(() => {
    const term = search.trim().toLowerCase();
    return staff.filter(item => !term || [item.username, item.full_name, item.employee_code, item.designation, item.mobile_number, item.status, item.employment_status].some(value => value?.toLowerCase().includes(term)));
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
    const fullName = String(form.get("full_name") ?? "").trim();
    const employeeCode = String(form.get("employee_code") ?? "").trim();
    const designation = String(form.get("designation") ?? "").trim();
    const mobileNumber = String(form.get("mobile_number") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const joiningDate = String(form.get("joining_date") ?? "").trim();
    const employmentStatus = String(form.get("employment_status") ?? "active");
    if ((action === "create" || action === "edit") && !/^[a-z0-9._-]{3,32}$/.test(username)) nextErrors.username = "Use 3–32 lowercase letters, numbers, dots, underscores or hyphens.";
    if (action === "create" || action === "edit") {
      if (!fullName || fullName.length > 160) nextErrors.full_name = "Enter the employee's full name (up to 160 characters).";
      if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$/.test(employeeCode)) nextErrors.employee_code = "Use 1–32 letters, numbers, dots, underscores or hyphens.";
      if (!designation || designation.length > 100) nextErrors.designation = "Enter a designation (up to 100 characters).";
      if (!/^\+?[0-9]{10,15}$/.test(mobileNumber)) nextErrors.mobile_number = "Enter one mobile number with 10–15 digits; an international + prefix is allowed.";
      if (email && (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254)) nextErrors.email = "Enter a valid email address.";
      if (employmentStatus !== "active" && employmentStatus !== "inactive") nextErrors.employment_status = "Choose Active or Inactive.";
    }
    if (action !== "edit" && password.length < 12) nextErrors.password = "Password must contain at least 12 characters.";
    setFieldErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;
    setFeedback(null);
    try {
      if (action === "create") {
        await create.mutateAsync({ username, initial_password: password, full_name: fullName, employee_code: employeeCode, designation, mobile_number: mobileNumber, email, joining_date: joiningDate || undefined });
        setFeedback({ kind: "success", message: "Employee profile and staff login created. The staff member must change the initial password after signing in." });
      } else if (action === "edit" && selected) {
        await update.mutateAsync({ principalID: selected.principal_id, username, full_name: fullName, employee_code: employeeCode, designation, mobile_number: mobileNumber, email, joining_date: joiningDate || undefined, employment_status: employmentStatus as "active" | "inactive" });
        setFeedback({ kind: "success", message: "Employee profile and login details updated." });
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
  function deactivate(item: StaffRecord) { setDisableError(""); setStaffToDisable(item); }
  async function confirmDeactivate() {
    if (!staffToDisable) return;
    const item = staffToDisable;
    setDisableError("");
    setFeedback(null);
    try {
      await disable.mutateAsync(item.principal_id);
      setStaffToDisable(null);
      setFeedback({ kind: "success", message: `${item.username} has been disabled and active sessions revoked.` });
    } catch (error) {
      setDisableError(error instanceof Error ? error.message : "Unable to disable this account. Please try again.");
    }
  }

  const pending = create.isPending || update.isPending || reset.isPending || enable.isPending;
  const formTitle = action === "create" ? "Add employee" : action === "edit" ? "Edit employee profile" : action === "reset" ? "Reset staff password" : "Enable staff account";
  const passwordLabel = action === "create" ? "Initial password" : action === "reset" ? "New temporary password" : "Temporary password";

  return (
    <div className="content staff-management-page">
      <div className="heading">
        <div><span className="eyebrow">PEOPLE</span><h1>Staff accounts</h1><p>Create and manage the staff accounts belonging to this school.</p></div>
        {canCreate && <button className="primary staff-create-button" type="button" onClick={() => openAction("create")}><Plus size={16}/> Add staff</button>}
      </div>

      {feedback && (action === null || feedback.kind === "success") && <div className={`staff-feedback ${feedback.kind}`} role="status">{feedback.message}</div>}

      <Modal
        open={action !== null}
        title={formTitle}
        description={selected ? `Employee: ${selected.full_name || selected.username}. Login usernames must be unique across the platform.` : "Create the employee profile and login account together."}
        onClose={closeAction}
        closeDisabled={pending}
      >
        {feedback?.kind === "error" && <p className="staff-feedback error" role="alert">{feedback.message}</p>}
        <form onSubmit={submit} noValidate>
          {(action === "create" || action === "edit") && <label>Username<input name="username" placeholder="e.g. rahul.sharma" defaultValue={selected?.username ?? ""} minLength={3} maxLength={32} pattern="[a-z0-9._-]{3,32}" autoComplete="off" required aria-invalid={Boolean(fieldErrors.username)} aria-describedby={fieldErrors.username ? "staff-username-error" : "staff-username-help"} onChange={()=>setFieldErrors(current=>({...current,username:""}))}/><small id="staff-username-help">3–32 lowercase letters, numbers, dots, underscores or hyphens.</small>{fieldErrors.username && <small className="app-field-error" id="staff-username-error">{fieldErrors.username}</small>}</label>}
          {(action === "create" || action === "edit") && <>
            <label>Full name<input name="full_name" placeholder="e.g. Rahul Sharma" defaultValue={selected?.full_name ?? ""} maxLength={160} required aria-invalid={Boolean(fieldErrors.full_name)} onChange={()=>setFieldErrors(current=>({...current,full_name:""}))}/>{fieldErrors.full_name && <small className="app-field-error">{fieldErrors.full_name}</small>}</label>
            <label>Employee code<input name="employee_code" placeholder="e.g. EMP-001" defaultValue={selected?.employee_code ?? ""} maxLength={32} required aria-invalid={Boolean(fieldErrors.employee_code)} onChange={()=>setFieldErrors(current=>({...current,employee_code:""}))}/><small>Must be unique within this school.</small>{fieldErrors.employee_code && <small className="app-field-error">{fieldErrors.employee_code}</small>}</label>
            <label>Designation<input name="designation" placeholder="e.g. Mathematics Teacher" defaultValue={selected?.designation ?? ""} maxLength={100} required aria-invalid={Boolean(fieldErrors.designation)} onChange={()=>setFieldErrors(current=>({...current,designation:""}))}/>{fieldErrors.designation && <small className="app-field-error">{fieldErrors.designation}</small>}</label>
            <label>Mobile number<input name="mobile_number" type="tel" placeholder="e.g. 9876543210" defaultValue={selected?.mobile_number ?? ""} maxLength={16} required aria-invalid={Boolean(fieldErrors.mobile_number)} onChange={()=>setFieldErrors(current=>({...current,mobile_number:""}))}/><small>Enter one contact number (10–15 digits).</small>{fieldErrors.mobile_number && <small className="app-field-error">{fieldErrors.mobile_number}</small>}</label>
            <label>Email address <small>(optional)</small><input name="email" type="email" placeholder="e.g. rahul@example.com" defaultValue={selected?.email ?? ""} maxLength={254} aria-invalid={Boolean(fieldErrors.email)} onChange={()=>setFieldErrors(current=>({...current,email:""}))}/>{fieldErrors.email && <small className="app-field-error">{fieldErrors.email}</small>}</label>
            <label>Joining date <small>(optional)</small><input name="joining_date" type="date" defaultValue={selected?.joining_date ? selected.joining_date.slice(0,10) : ""}/></label>
            {action === "edit" && <label>Employment status<select name="employment_status" defaultValue={selected?.employment_status || "active"}><option value="active">Active</option><option value="inactive">Inactive</option></select><small>Inactive employees cannot sign in. Reactivating employment does not automatically enable a disabled login.</small></label>}
          </>}
          {action !== "edit" && <label>{passwordLabel}<div className="staff-password-field"><input name="password" type={showPassword ? "text" : "password"} placeholder="Enter a strong password (12+ characters)" minLength={12} autoComplete="new-password" required aria-invalid={Boolean(fieldErrors.password)} aria-describedby={fieldErrors.password ? "staff-password-error" : "staff-password-help"} onChange={()=>setFieldErrors(current=>({...current,password:""}))}/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Hide password" : "Show password"}>{showPassword ? "Hide" : "Show"}</button></div><small id="staff-password-help">At least 12 characters. The server also checks password safety and compromised-password status.</small>{fieldErrors.password && <small className="app-field-error" id="staff-password-error">{fieldErrors.password}</small>}</label>}
          <div className="app-modal-actions"><button className="app-modal-cancel" type="button" onClick={closeAction} disabled={pending}>Cancel</button><button className="primary" type="submit" disabled={pending}>{pending ? "Saving…" : action === "reset" ? "Reset password" : action === "enable" ? "Enable account" : "Save changes"}</button></div>
        </form>
      </Modal>

      <section className="table-panel">
        <div className="toolbar"><div><h2>All staff</h2><small>{staff.length} accounts</small></div><div className="search"><Search size={16}/><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search by username or status" aria-label="Search staff"/></div></div>
        {query.isPending ? <div className="table-message" aria-live="polite">Loading staff accounts…</div>
          : query.isError ? <div className="table-message error-text">{query.error instanceof Error ? query.error.message : "Unable to load staff accounts."}<button className="staff-retry-button" type="button" onClick={() => void query.refetch()}>Try again</button></div>
          : filtered.length === 0 ? <div className="table-message">{search ? "No staff match this search." : "No staff accounts yet. Add the first staff account to get started."}</div>
          : <div className="table-scroll"><table><thead><tr><th>Employee</th><th>Employee code</th><th>Designation</th><th>Mobile</th><th>Login status</th><th>Employment</th><th>Password</th><th>Created</th><th>Actions</th></tr></thead><tbody>
            {filtered.map(item => <Fragment key={item.principal_id}><tr><td><b>{item.full_name || "Profile incomplete"}</b><small className="staff-id">{item.username} · Principal #{item.principal_id}</small></td><td>{item.employee_code || "—"}</td><td>{item.designation || "—"}</td><td>{item.mobile_number || "—"}</td><td><span className={`pill ${item.status === "active" ? "active" : "inactive"}`}>{item.status}</span></td><td>{item.employment_status ? <span className={`pill ${item.employment_status === "active" ? "active" : "inactive"}`}>{item.employment_status}</span> : "Profile pending"}</td><td>{item.must_change_password ? <span className="staff-password-required">Change required</span> : "—"}</td><td>{item.created_at ? new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(item.created_at)) : "—"}</td><td><div className="staff-row-actions">
              {canUpdate && <button type="button" onClick={() => openAction("edit", item)} title="Edit username" aria-label={`Edit ${item.username}`}><Pencil size={15}/></button>}
              {canReset && item.status === "active" && <button type="button" onClick={() => openAction("reset", item)} title="Reset password" aria-label={`Reset password for ${item.username}`}><KeyRound size={15}/></button>}
              {canReadRoles && item.status === "active" && <button type="button" className={rolesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setRolesForPrincipal(item.principal_id)} title="Manage roles" aria-label={`Manage roles for ${item.username}`} aria-expanded={rolesForPrincipal === item.principal_id}><Shield size={15}/></button>}
              {canManageOverrides && item.status === "active" && <button type="button" className={overridesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setOverridesForPrincipal(item.principal_id)} title="Manage permission overrides" aria-label={`Manage permission overrides for ${item.username}`} aria-expanded={overridesForPrincipal === item.principal_id}><LockKeyhole size={15}/></button>}
              {item.status === "active" && canDisable && <button type="button" className="danger" onClick={() => void deactivate(item)} disabled={disable.isPending} title="Disable account" aria-label={`Disable ${item.username}`}><UserMinus size={15}/></button>}
              {item.status !== "active" && item.employment_status !== "inactive" && canUpdate && <button type="button" onClick={() => openAction("enable", item)} title="Enable account" aria-label={`Enable ${item.username}`}><UserCheck size={15}/></button>}
            </div></td></tr>
            </Fragment>)}
          </tbody></table></div>}
      </section>
      {rolesForPrincipal !== null && canReadRoles && (() => { const target=staff.find(item=>item.principal_id===rolesForPrincipal); return target ? <Modal open title={`Access roles · ${target.username}`} description="Review the roles assigned to this staff account. Add or remove roles here; changes take effect on the next protected request." onClose={()=>setRolesForPrincipal(null)}><StaffRoleAssignments staff={target} canAssign={canAssignRoles}/></Modal> : null; })()}
      {overridesForPrincipal !== null && canManageOverrides && (() => { const target=staff.find(item=>item.principal_id===overridesForPrincipal); return target ? <Modal open title={`Permission overrides · ${target.username}`} description="Review individual access exceptions. Use an override only when the staff member's role permissions need a documented exception." onClose={()=>setOverridesForPrincipal(null)}><StaffPermissionOverrides staff={target}/></Modal> : null; })()}
      <p className="staff-security-note"><ShieldCheck size={15}/> Every action is checked again by the server using your current school permissions.</p>
      <ConfirmDialog open={Boolean(staffToDisable)} title="Disable staff account?" description={staffToDisable ? `Disable ${staffToDisable.username}? Their active sessions will be revoked and they will no longer be able to sign in.` : ""} confirmLabel="Disable account" pending={disable.isPending} error={disableError || undefined} onCancel={() => {setStaffToDisable(null);setDisableError("");}} onConfirm={() => void confirmDeactivate()}/>
    </div>
  );
}
