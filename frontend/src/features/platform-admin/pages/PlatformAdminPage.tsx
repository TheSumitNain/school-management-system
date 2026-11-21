import { useEffect, useState, type FormEvent } from "react";
import { notifyToast } from "../../../shared/notifications";
import { Building2, Eye, EyeOff, KeyRound, ShieldCheck, UserPlus, Users } from "lucide-react";
import {
  useCreatePlatformAdminMutation,
  useDeactivatePlatformAdminMutation,
  useOnboardSchoolMutation,
  usePlatformAdminsQuery,
  usePlatformSchoolsQuery,
  useDelegableSchoolPermissionsQuery,
  usePlatformSchoolDelegationsQuery,
  useGrantPlatformSchoolPermissionDelegationMutation,
  useRevokePlatformSchoolPermissionDelegationMutation,
} from "../queries";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { ApiError } from "../../../lib/api";
import "./PlatformAdminPage.css";

type Props = { permissions: string[] };

export function PlatformAdminPage({ permissions }: Props) {
  const canManageAdmins = permissions.includes("platform.admins.manage");
  const canOnboardSchool = permissions.includes("platform.schools.create") &&
    permissions.includes("platform.school_admins.create");
  const canDelegate = permissions.includes("platform.permissions.delegate") && permissions.includes("platform.schools.read");
  const firstTab = canManageAdmins ? "admins" : canOnboardSchool ? "school" : "delegations";
  const [activeTab, setActiveTab] = useState<"admins" | "school" | "delegations">(firstTab);
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [showAdminPassword, setShowAdminPassword] = useState(false);
  const [showSchoolAdminPassword, setShowSchoolAdminPassword] = useState(false);
  const [adminDialogOpen, setAdminDialogOpen] = useState(false);
  const [schoolDialogOpen, setSchoolDialogOpen] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [adminToDeactivate, setAdminToDeactivate] = useState<{ principalID: number; email: string } | null>(null);
  const [deactivateError, setDeactivateError] = useState("");
  const [delegationToRevoke, setDelegationToRevoke] = useState<{ id: number; permissionKey: string } | null>(null);
  const [revokeError, setRevokeError] = useState("");
  const [selectedSchoolID, setSelectedSchoolID] = useState<number | null>(null);
  const [selectedPermission, setSelectedPermission] = useState("");
  const [canAssignToRoles, setCanAssignToRoles] = useState(true);
  const [canOverrideUsers, setCanOverrideUsers] = useState(false);

  const schoolsQuery = usePlatformSchoolsQuery(canDelegate);
  const permissionQuery = useDelegableSchoolPermissionsQuery(canDelegate);
  const delegationsQuery = usePlatformSchoolDelegationsQuery(selectedSchoolID, canDelegate);
  const grantDelegation = useGrantPlatformSchoolPermissionDelegationMutation();
  const revokeDelegation = useRevokePlatformSchoolPermissionDelegationMutation();
  const delegablePermissions = (permissionQuery.data?.permissions ?? []).filter(item => item.is_delegable);
  const activeSchools = (schoolsQuery.data?.schools ?? []).filter(school => school.status === "active");

  useEffect(() => {
    if (selectedSchoolID === null && activeSchools.length) setSelectedSchoolID(activeSchools[0].id);
  }, [selectedSchoolID, activeSchools]);


  const adminsQuery = usePlatformAdminsQuery(canManageAdmins);
  const createAdmin = useCreatePlatformAdminMutation();
  const deactivateAdmin = useDeactivatePlatformAdminMutation();
  const onboardSchool = useOnboardSchoolMutation();

  async function submitAdmin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const email = String(form.get("email") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const errors: Record<string, string> = {};
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.email = "Enter a valid email address.";
    if (password.length < 12) errors.password = "Password must contain at least 12 characters.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      const result = await createAdmin.mutateAsync({ email, password,
      });
      formElement.reset();
      setAdminDialogOpen(false);
      notifyToast("success", `Platform Admin ${result.admin.email} created. MFA enrollment is required at first sign-in.`);
      setFeedback(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to create Platform Admin.";
      if (error instanceof ApiError) {
        const mapped: Record<string, string> = {};
        for (const key of ["email", "password"]) {
          if (error.fieldErrors[key]) mapped[key] = error.fieldErrors[key];
        }
        if (Object.keys(mapped).length) setFieldErrors(current => ({ ...current, ...mapped }));
      }
      notifyToast("error", message);
      setFeedback({ kind: "error", message });
    }
  }

  async function submitSchool(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const name = String(form.get("name") ?? "").trim();
    const code = String(form.get("code") ?? "").trim();
    const username = String(form.get("school_admin_username") ?? "").trim();
    const password = String(form.get("school_admin_initial_password") ?? "");
    const errors: Record<string, string> = {};
    if (Array.from(name).length < 2 || Array.from(name).length > 150) errors.name = "School name must contain 2–150 characters.";
    if (!/^[A-Za-z0-9][A-Za-z0-9_-]{1,49}$/.test(code)) errors.code = "Use 2–50 letters, numbers, underscores or hyphens; start with a letter or number.";
    if (!/^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$/.test(username)) errors.school_admin_username = "Use 3–32 letters, numbers, dots, underscores or hyphens.";
    if (password.length < 12) errors.school_admin_initial_password = "Password must contain at least 12 characters.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      const result = await onboardSchool.mutateAsync({
        name, code, school_admin_username: username,
        school_admin_initial_password: password,
      });
      formElement.reset();
      setSchoolDialogOpen(false);
      notifyToast("success", `School ${result.school.name} (${result.school.code}) created. School Admin username: ${result.school_admin.username}. They must change the initial password after sign-in.`);
      setFeedback(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to onboard school.";
      if (error instanceof ApiError) {
        const mapped: Record<string, string> = {};
        for (const key of ["name", "code", "school_admin_username", "school_admin_initial_password"]) {
          if (error.fieldErrors[key]) mapped[key] = error.fieldErrors[key];
        }
        if (Object.keys(mapped).length) setFieldErrors(current => ({ ...current, ...mapped }));
      }
      notifyToast("error", message);
      setFeedback({ kind: "error", message });
    }
  }

  async function grantPermission(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const errors: Record<string, string> = {};
    if (selectedSchoolID === null) errors.school = "Choose a school.";
    if (!selectedPermission) errors.permission_key = "Choose a permission to delegate.";
    if (!canAssignToRoles && !canOverrideUsers) errors.capabilities = "Select at least one delegation capability.";
    setFieldErrors(current => ({ ...current, ...errors, permission_key: errors.permission_key ?? "", capabilities: errors.capabilities ?? "" }));
    if (Object.keys(errors).length) {
      setFeedback({ kind: "error", message: "Review the delegation details that need attention." });
      return;
    }
    setFeedback(null);
    try {
      await grantDelegation.mutateAsync({
        schoolID: selectedSchoolID,
        permission_key: selectedPermission,
        can_assign_to_roles: canAssignToRoles,
        can_override_users: canOverrideUsers,
      });
      setSelectedPermission("");
      setFieldErrors(current => ({ ...current, permission_key: "", capabilities: "" }));
      notifyToast("success", "Permission delegated to the selected school.");
      setFeedback(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to delegate permission.";
      if (error instanceof ApiError) {
        const mapped: Record<string, string> = {};
        if (error.fieldErrors.permission_key) mapped.permission_key = error.fieldErrors.permission_key;
        if (error.fieldErrors.can_assign_to_roles || error.fieldErrors.can_override_users) {
          mapped.capabilities = error.fieldErrors.can_assign_to_roles || error.fieldErrors.can_override_users;
        }
        if (Object.keys(mapped).length) setFieldErrors(current => ({ ...current, ...mapped }));
      }
      notifyToast("error", message);
      setFeedback({ kind: "error", message });
    }
  }

  function revokePermission(delegationID: number, permissionKey: string) {
    setRevokeError("");
    setDelegationToRevoke({ id: delegationID, permissionKey });
  }
  async function confirmRevokePermission() {
    if (!delegationToRevoke || selectedSchoolID === null) return;
    const target = delegationToRevoke;
    setRevokeError("");
    setFeedback(null);
    try {
      await revokeDelegation.mutateAsync({ delegationID: target.id, schoolID: selectedSchoolID });
      setDelegationToRevoke(null);
      notifyToast("success", `${target.permissionKey} delegation revoked.`);
      setFeedback(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to revoke permission.";
      notifyToast("error", message);
      setRevokeError(message);
      setFeedback({ kind: "error", message });
    }
  }

  function deactivate(principalID: number, email: string) { setDeactivateError(""); setAdminToDeactivate({ principalID, email }); }
  async function confirmDeactivate() {
    if (!adminToDeactivate) return;
    const target = adminToDeactivate;
    setDeactivateError("");
    setFeedback(null);
    try {
      await deactivateAdmin.mutateAsync(target.principalID);
      setAdminToDeactivate(null);
      notifyToast("success", `${target.email} was deactivated and their active sessions were revoked.`);
      setFeedback(null);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to deactivate Platform Admin. Please try again.";
      notifyToast("error", message);
      setDeactivateError(message);
    }
  }

  return (
    <div className="content platform-admin-page">
      <div className="heading">
        <div>
          <span className="eyebrow">GLOBAL OPERATIONS</span>
          <h1>Platform administration</h1>
          <p>Manage platform administrator access and onboard schools.</p>
        </div>
        <span className="platform-secure"><ShieldCheck size={16} /> MFA-protected operations</span>
      </div>

      <div className="platform-tabs" role="tablist" aria-label="Platform administration sections">
        {canManageAdmins && <button type="button" role="tab" aria-selected={activeTab === "admins"} className={activeTab === "admins" ? "selected" : ""} onClick={() => { setActiveTab("admins"); setFeedback(null); }}><Users size={16}/> Platform Admins</button>}
        {canOnboardSchool && <button type="button" role="tab" aria-selected={activeTab === "school"} className={activeTab === "school" ? "selected" : ""} onClick={() => { setActiveTab("school"); setFeedback(null); }}><Building2 size={16}/> Onboard school</button>}
        {canDelegate && <button type="button" role="tab" aria-selected={activeTab === "delegations"} className={activeTab === "delegations" ? "selected" : ""} onClick={() => { setActiveTab("delegations"); setFeedback(null); }}><KeyRound size={16}/> School permissions</button>}

      </div>

      {feedback && ((!adminDialogOpen && !schoolDialogOpen) || feedback.kind === "success") && <div className={`platform-feedback ${feedback.kind}`} role="status">{feedback.message}</div>}

      {activeTab === "admins" && canManageAdmins && (
        <div className="platform-grid">
          <section className="table-panel">
            <div className="toolbar"><div><h2>Platform Admin accounts</h2><small>Global administrator access</small></div><span className="platform-count">{adminsQuery.data?.admins.length ?? 0} accounts</span></div>
            {adminsQuery.isPending ? <div className="table-message">Loading administrator accounts…</div>
              : adminsQuery.error ? <div className="table-message error-text">{adminsQuery.error instanceof Error ? adminsQuery.error.message : "Unable to load accounts."}<button className="platform-inline-button" onClick={() => void adminsQuery.refetch()}>Retry</button></div>
              : adminsQuery.data?.admins.length ? <div className="table-scroll"><table><thead><tr><th>Administrator</th><th>Status</th><th>MFA</th><th>Created</th><th>Action</th></tr></thead><tbody>{adminsQuery.data.admins.map(admin => <tr key={admin.principal_id}><td><b>{admin.email}</b><small className="platform-subtext">ID #{admin.principal_id}</small></td><td><span className={`pill ${admin.status === "active" ? "active" : "inactive"}`}>{admin.status}</span></td><td>{admin.mfa_enrolled ? "Enrolled" : "Pending"}</td><td>{new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(admin.created_at))}</td><td>{admin.status === "active" ? <button className="platform-danger-button" disabled={deactivateAdmin.isPending} onClick={() => void deactivate(admin.principal_id, admin.email)}>Deactivate</button> : "—"}</td></tr>)}</tbody></table></div>
              : <div className="table-message">No Platform Admin accounts found.</div>}
          </section>

          <div className="platform-create-action"><p>Create an additional global administrator account. New accounts must enroll MFA on first sign-in.</p><button type="button" className="primary" onClick={() => {setFeedback(null);setFieldErrors({});setShowAdminPassword(false);setAdminDialogOpen(true);}}><UserPlus size={16}/> Create Platform Admin</button></div>
        </div>
      )}

      {activeTab === "delegations" && canDelegate && (
        <div className="platform-delegations">
          <section className="form-panel platform-form-panel">
            <div className="platform-form-title"><KeyRound size={19}/><div><h2>Delegate school permissions</h2><p>Only delegable permissions can be granted. Choose what the School Admin may assign to custom roles or individual staff.</p></div></div>
            {schoolsQuery.isPending ? <div className="table-message">Loading schools…</div>
              : schoolsQuery.error ? <div className="table-message error-text">{schoolsQuery.error instanceof Error ? schoolsQuery.error.message : "Unable to load schools."}<button className="platform-inline-button" onClick={() => void schoolsQuery.refetch()}>Retry</button></div>
              : !activeSchools.length ? <div className="table-message">No active schools are available for permission delegation.</div>
              : <>
                <label className="platform-select-label">School
                  <select value={selectedSchoolID ?? ""} aria-invalid={Boolean(fieldErrors.school)} aria-describedby={fieldErrors.school?"delegation-school-error":undefined} onChange={event => { setSelectedSchoolID(Number(event.target.value) || null); setFieldErrors(current=>({...current,school:""})); setFeedback(null); }}>
                    {activeSchools.map(school => <option key={school.id} value={school.id}>{school.name} · {school.code} · {school.status}</option>)}
                  </select>
                  {fieldErrors.school&&<small className="app-field-error" id="delegation-school-error" role="alert">{fieldErrors.school}</small>}
                </label>
                {permissionQuery.isPending ? <div className="table-message">Loading delegable permissions…</div>
                  : permissionQuery.error ? <div className="table-message error-text">{permissionQuery.error instanceof Error ? permissionQuery.error.message : "Unable to load permission catalogue."}<button className="platform-inline-button" onClick={() => void permissionQuery.refetch()}>Retry</button></div>
                  : delegablePermissions.length === 0 ? <div className="table-message">No active school permissions are currently marked as delegable.</div> : <form onSubmit={grantPermission} noValidate>
                  <label>Permission
                    <select value={selectedPermission} aria-invalid={Boolean(fieldErrors.permission_key)} aria-describedby={fieldErrors.permission_key?"delegation-permission-error":undefined} onChange={event => {setSelectedPermission(event.target.value);setFieldErrors(current=>({...current,permission_key:""}));setFeedback(null);}} required>
                      <option value="">Select a delegable permission</option>
                      {delegablePermissions.map(permission => <option key={permission.permission_key} value={permission.permission_key}>{permission.permission_key}</option>)}
                    </select>
                    {fieldErrors.permission_key && <small className="app-field-error" id="delegation-permission-error" role="alert">{fieldErrors.permission_key}</small>}
                  </label>
                  {selectedPermission && <p className="platform-permission-description">{delegablePermissions.find(item => item.permission_key === selectedPermission)?.description}</p>}
                  <div className="platform-delegation-options">
                    <label><input type="checkbox" checked={canAssignToRoles} aria-invalid={Boolean(fieldErrors.capabilities)} aria-describedby={fieldErrors.capabilities?"delegation-capabilities-error":undefined} onChange={event => {setCanAssignToRoles(event.target.checked);setFieldErrors(current=>({...current,capabilities:""}));setFeedback(null);}}/> School Admin can assign it to custom roles</label>
                    <label><input type="checkbox" checked={canOverrideUsers} aria-invalid={Boolean(fieldErrors.capabilities)} aria-describedby={fieldErrors.capabilities?"delegation-capabilities-error":undefined} onChange={event => {setCanOverrideUsers(event.target.checked);setFieldErrors(current=>({...current,capabilities:""}));setFeedback(null);}}/> School Admin can grant individual user overrides</label>
                  </div>
                  {fieldErrors.capabilities && <small className="app-field-error" id="delegation-capabilities-error" role="alert">{fieldErrors.capabilities}</small>}
                  <button className="primary" type="submit" disabled={grantDelegation.isPending || !selectedPermission || (!canAssignToRoles && !canOverrideUsers)}>
                    {grantDelegation.isPending ? "Delegating…" : "Grant permission to school"}
                  </button>
                </form>}
              </>}
          </section>

          <section className="table-panel">
            <div className="toolbar"><div><h2>Active delegations</h2><small>{selectedSchoolID === null ? "Select a school to view its permissions" : "Permissions currently delegated to this school"}</small></div><span className="platform-count">{delegationsQuery.data?.delegations.length ?? 0} active</span></div>
            {selectedSchoolID === null ? <div className="table-message">Select a school to continue.</div>
              : delegationsQuery.isPending ? <div className="table-message">Loading delegated permissions…</div>
              : delegationsQuery.error ? <div className="table-message error-text">{delegationsQuery.error instanceof Error ? delegationsQuery.error.message : "Unable to load delegations."}<button className="platform-inline-button" onClick={() => void delegationsQuery.refetch()}>Retry</button></div>
              : delegationsQuery.data?.delegations.length ? <div className="platform-delegation-list">
                {delegationsQuery.data.delegations.map(item => <article className="platform-delegation-item" key={item.id}>
                  <div><b>{item.permission_key}</b><small>{item.can_assign_to_roles ? "Custom roles" : ""}{item.can_assign_to_roles && item.can_override_users ? " · " : ""}{item.can_override_users ? "Individual overrides" : ""}</small></div>
                  <button type="button" className="platform-danger-button" disabled={revokeDelegation.isPending} onClick={() => void revokePermission(item.id, item.permission_key)}>Revoke</button>
                </article>)}
              </div> : <div className="table-message">No active permission delegations for this school.</div>}
          </section>
        </div>
      )}

      {activeTab === "school" && canOnboardSchool && <section className="platform-create-action"><p>Create a school and its first School Admin account together.</p><button type="button" className="primary" onClick={() => {setFeedback(null);setFieldErrors({});setShowSchoolAdminPassword(false);setSchoolDialogOpen(true);}}><Building2 size={16}/> Onboard a new school</button></section>}
      <Modal open={adminDialogOpen} title="Create Platform Admin" description="The new administrator must enroll multi-factor authentication on first sign-in." onClose={() => setAdminDialogOpen(false)} closeDisabled={createAdmin.isPending}>
        {feedback?.kind === "error" && <p className="platform-feedback error" role="alert">{feedback.message}</p>}
        <form onSubmit={submitAdmin} noValidate>
          <label>Email address<input name="email" type="email" autoComplete="email" placeholder="e.g. admin@school.com" maxLength={254} required aria-invalid={Boolean(fieldErrors.email)} aria-describedby={fieldErrors.email?"platform-email-error":undefined} onChange={()=>setFieldErrors(current=>({...current,email:""}))}/>{fieldErrors.email && <small className="app-field-error" id="platform-email-error" role="alert">{fieldErrors.email}</small>}</label>
          <label>Initial password<div className="password-input-wrap"><input name="password" type={showAdminPassword ? "text" : "password"} placeholder="Enter a strong password (12+ characters)" autoComplete="new-password" minLength={12} required aria-invalid={Boolean(fieldErrors.password)} aria-describedby={fieldErrors.password?"platform-password-error":"platform-password-help"} onChange={()=>setFieldErrors(current=>({...current,password:""}))}/><button className="password-visibility" type="button" aria-label={showAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showAdminPassword} onClick={() => setShowAdminPassword(value => !value)}>{showAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small id="platform-password-help">At least 12 characters. The server also checks password safety.</small>{fieldErrors.password && <small className="app-field-error" id="platform-password-error" role="alert">{fieldErrors.password}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => setAdminDialogOpen(false)} disabled={createAdmin.isPending}>Cancel</button><button className="primary" type="submit" disabled={createAdmin.isPending}>{createAdmin.isPending ? "Creating account…" : "Create Platform Admin"}</button></div>
        </form>
      </Modal>
      <Modal open={schoolDialogOpen} title="Onboard a new school" description="Create the school and its first School Admin account together. The admin must change the initial password after signing in." onClose={() => setSchoolDialogOpen(false)} closeDisabled={onboardSchool.isPending}>
        {feedback?.kind === "error" && <p className="platform-feedback error" role="alert">{feedback.message}</p>}
        <form onSubmit={submitSchool} noValidate>
          <label>School name<input name="name" autoComplete="organization" placeholder="e.g. Sunrise Public School" minLength={2} maxLength={150} required aria-invalid={Boolean(fieldErrors.name)} aria-describedby={fieldErrors.name?"school-name-error":undefined} onChange={()=>setFieldErrors(current=>({...current,name:""}))}/>{fieldErrors.name && <small className="app-field-error" id="school-name-error" role="alert">{fieldErrors.name}</small>}</label>
          <label>School code<input name="code" autoCapitalize="characters" placeholder="e.g. SUNRISE01" minLength={2} maxLength={50} pattern="[A-Za-z0-9][A-Za-z0-9_-]{1,49}" required aria-invalid={Boolean(fieldErrors.code)} aria-describedby={fieldErrors.code?"school-code-error":undefined} onChange={()=>setFieldErrors(current=>({...current,code:""}))}/><small>Use 2–50 letters, numbers, underscores or hyphens; start with a letter or number.</small>{fieldErrors.code && <small className="app-field-error" id="school-code-error" role="alert">{fieldErrors.code}</small>}</label>
          <label>School Admin username<input name="school_admin_username" autoComplete="username" placeholder="e.g. schooladmin" autoCapitalize="none" minLength={3} maxLength={32} pattern="[A-Za-z0-9][A-Za-z0-9._-]{2,31}" required aria-invalid={Boolean(fieldErrors.school_admin_username)} aria-describedby={fieldErrors.school_admin_username?"school-admin-username-error":undefined} onChange={()=>setFieldErrors(current=>({...current,school_admin_username:""}))}/><small>Use 3–32 letters, numbers, dots, underscores or hyphens.</small>{fieldErrors.school_admin_username && <small className="app-field-error" id="school-admin-username-error" role="alert">{fieldErrors.school_admin_username}</small>}</label>
          <label>Initial password<div className="password-input-wrap"><input name="school_admin_initial_password" type={showSchoolAdminPassword ? "text" : "password"} placeholder="Set an initial password (12+ characters)" autoComplete="new-password" minLength={12} required aria-invalid={Boolean(fieldErrors.school_admin_initial_password)} aria-describedby={fieldErrors.school_admin_initial_password?"school-admin-password-error":"school-admin-password-help"} onChange={()=>setFieldErrors(current=>({...current,school_admin_initial_password:""}))}/><button className="password-visibility" type="button" aria-label={showSchoolAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showSchoolAdminPassword} onClick={() => setShowSchoolAdminPassword(value => !value)}>{showSchoolAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small id="school-admin-password-help">At least 12 characters. The School Admin must change it after signing in.</small>{fieldErrors.school_admin_initial_password && <small className="app-field-error" id="school-admin-password-error" role="alert">{fieldErrors.school_admin_initial_password}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => setSchoolDialogOpen(false)} disabled={onboardSchool.isPending}>Cancel</button><button className="primary" type="submit" disabled={onboardSchool.isPending}>{onboardSchool.isPending ? "Creating school…" : "Create school and School Admin"}</button></div>
        </form>
      </Modal>
      <ConfirmDialog open={Boolean(adminToDeactivate)} title="Deactivate Platform Admin?" description={adminToDeactivate ? `Deactivate ${adminToDeactivate.email}? Their active sessions will be revoked and they will no longer be able to access platform administration.` : ""} confirmLabel="Deactivate account" pending={deactivateAdmin.isPending} error={deactivateError || undefined} onCancel={() => {setAdminToDeactivate(null);setDeactivateError("");}} onConfirm={() => void confirmDeactivate()}/>
      <ConfirmDialog open={delegationToRevoke !== null} title="Revoke school permission?" description={delegationToRevoke ? `Revoke ${delegationToRevoke.permissionKey} from this school? Existing role grants and user overrides tied to this delegation will stop authorizing access.` : ""} confirmLabel="Revoke permission" pending={revokeDelegation.isPending} error={revokeError || undefined} onCancel={() => { setDelegationToRevoke(null); setRevokeError(""); }} onConfirm={() => void confirmRevokePermission()}/>
    </div>
  );
}
