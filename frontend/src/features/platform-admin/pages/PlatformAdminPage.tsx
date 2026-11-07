import { useEffect, useState, type FormEvent } from "react";
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
  const [selectedSchoolID, setSelectedSchoolID] = useState<number | null>(null);
  const [selectedPermission, setSelectedPermission] = useState("");
  const [canAssignToRoles, setCanAssignToRoles] = useState(true);
  const [canOverrideUsers, setCanOverrideUsers] = useState(false);

  const schoolsQuery = usePlatformSchoolsQuery(canDelegate);
  const permissionQuery = useDelegableSchoolPermissionsQuery(canDelegate);
  const delegationsQuery = usePlatformSchoolDelegationsQuery(selectedSchoolID, canDelegate);
  const grantDelegation = useGrantPlatformSchoolPermissionDelegationMutation();
  const revokeDelegation = useRevokePlatformSchoolPermissionDelegationMutation();
  useEffect(() => {
    if (selectedSchoolID === null && schoolsQuery.data?.schools.length) {
      setSelectedSchoolID(schoolsQuery.data.schools[0].id);
    }
  }, [selectedSchoolID, schoolsQuery.data?.schools]);
  const delegablePermissions = (permissionQuery.data?.permissions ?? []).filter(item => item.is_delegable);

  const adminsQuery = usePlatformAdminsQuery(canManageAdmins);
  const createAdmin = useCreatePlatformAdminMutation();
  const deactivateAdmin = useDeactivatePlatformAdminMutation();
  const onboardSchool = useOnboardSchoolMutation();

  async function submitAdmin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    setFeedback(null);
    try {
      const result = await createAdmin.mutateAsync({
        email: String(form.get("email") ?? "").trim(),
        password: String(form.get("password") ?? ""),
      });
      formElement.reset();
      setFeedback({
        kind: "success",
        message: `Platform Admin ${result.admin.email} created. MFA enrollment is required at first sign-in.`,
      });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to create Platform Admin." });
    }
  }

  async function submitSchool(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFeedback(null);
    try {
      const result = await onboardSchool.mutateAsync({
        name: String(form.get("name") ?? "").trim(),
        code: String(form.get("code") ?? "").trim(),
        school_admin_username: String(form.get("school_admin_username") ?? "").trim(),
        school_admin_initial_password: String(form.get("school_admin_initial_password") ?? ""),
      });
      event.currentTarget.reset();
      setFeedback({
        kind: "success",
        message: `School ${result.school.name} (${result.school.code}) created. School Admin username: ${result.school_admin.username}. They must change the initial password after sign-in.`,
      });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to onboard school." });
    }
  }


  async function grantPermission(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (selectedSchoolID === null || !selectedPermission || (!canAssignToRoles && !canOverrideUsers)) {
      setFeedback({ kind: "error", message: "Choose a school, permission and at least one delegation capability." });
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
      setFeedback({ kind: "success", message: "Permission delegated to the selected school." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to delegate permission." });
    }
  }

  async function revokePermission(delegationID: number, permissionKey: string) {
    if (selectedSchoolID === null || !window.confirm(`Revoke ${permissionKey} from this school? Existing role grants and user overrides tied to this delegation will stop authorizing access.`)) return;
    setFeedback(null);
    try {
      await revokeDelegation.mutateAsync({ delegationID, schoolID: selectedSchoolID });
      setFeedback({ kind: "success", message: `${permissionKey} delegation revoked.` });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to revoke permission." });
    }
  }

  async function deactivate(principalID: number, email: string) {
    if (!window.confirm(`Deactivate Platform Admin ${email}? Their active refresh sessions will be revoked.`)) return;
    setFeedback(null);
    try {
      await deactivateAdmin.mutateAsync(principalID);
      setFeedback({ kind: "success", message: `${email} was deactivated.` });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to deactivate Platform Admin." });
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

      {feedback && <div className={`platform-feedback ${feedback.kind}`} role="status">{feedback.message}</div>}

      {activeTab === "admins" && canManageAdmins && (
        <div className="platform-grid">
          <section className="table-panel">
            <div className="toolbar"><div><h2>Platform Admin accounts</h2><small>Global administrator access</small></div><span className="platform-count">{adminsQuery.data?.admins.length ?? 0} accounts</span></div>
            {adminsQuery.isPending ? <div className="table-message">Loading administrator accounts…</div>
              : adminsQuery.error ? <div className="table-message error-text">{adminsQuery.error instanceof Error ? adminsQuery.error.message : "Unable to load accounts."}<button className="platform-inline-button" onClick={() => void adminsQuery.refetch()}>Retry</button></div>
              : adminsQuery.data?.admins.length ? <div className="table-scroll"><table><thead><tr><th>Administrator</th><th>Status</th><th>MFA</th><th>Created</th><th>Action</th></tr></thead><tbody>{adminsQuery.data.admins.map(admin => <tr key={admin.principal_id}><td><b>{admin.email}</b><small className="platform-subtext">ID #{admin.principal_id}</small></td><td><span className={`pill ${admin.status === "active" ? "active" : "inactive"}`}>{admin.status}</span></td><td>{admin.mfa_enrolled ? "Enrolled" : "Pending"}</td><td>{new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(admin.created_at))}</td><td>{admin.status === "active" ? <button className="platform-danger-button" disabled={deactivateAdmin.isPending} onClick={() => void deactivate(admin.principal_id, admin.email)}>Deactivate</button> : "—"}</td></tr>)}</tbody></table></div>
              : <div className="table-message">No Platform Admin accounts found.</div>}
          </section>

          <section className="form-panel platform-form-panel">
            <div className="platform-form-title"><UserPlus size={19}/><div><h2>Create Platform Admin</h2><p>New accounts must enroll MFA on first sign-in.</p></div></div>
            <form onSubmit={submitAdmin}>
              <label>Email address<input name="email" type="email" autoComplete="email" maxLength={254} required /></label>
              <label>Initial password<div className="password-input-wrap"><input name="password" type={showAdminPassword ? "text" : "password"} autoComplete="new-password" minLength={12} required /><button className="password-visibility" type="button" aria-label={showAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showAdminPassword} onClick={() => setShowAdminPassword(value => !value)}>{showAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small>At least 12 characters. The server also checks password safety.</small></label>
              <button className="primary" type="submit" disabled={createAdmin.isPending}>{createAdmin.isPending ? "Creating account…" : "Create Platform Admin"}</button>
            </form>
          </section>
        </div>
      )}


      {activeTab === "delegations" && canDelegate && (
        <div className="platform-delegations">
          <section className="form-panel platform-form-panel">
            <div className="platform-form-title"><KeyRound size={19}/><div><h2>Delegate school permissions</h2><p>Only delegable permissions can be granted. Choose what the School Admin may assign to custom roles or individual staff.</p></div></div>
            {schoolsQuery.isPending ? <div className="table-message">Loading schools…</div>
              : schoolsQuery.error ? <div className="table-message error-text">{schoolsQuery.error instanceof Error ? schoolsQuery.error.message : "Unable to load schools."}<button className="platform-inline-button" onClick={() => void schoolsQuery.refetch()}>Retry</button></div>
              : !schoolsQuery.data?.schools.length ? <div className="table-message">No schools have been onboarded yet.</div>
              : <>
                <label className="platform-select-label">School
                  <select value={selectedSchoolID ?? ""} onChange={event => { setSelectedSchoolID(Number(event.target.value) || null); setFeedback(null); }}>
                    {schoolsQuery.data.schools.map(school => <option key={school.id} value={school.id}>{school.name} · {school.code} · {school.status}</option>)}
                  </select>
                </label>
                <form onSubmit={grantPermission}>
                  <label>Permission
                    <select value={selectedPermission} onChange={event => setSelectedPermission(event.target.value)} required>
                      <option value="">Select a delegable permission</option>
                      {delegablePermissions.map(permission => <option key={permission.permission_key} value={permission.permission_key}>{permission.permission_key}</option>)}
                    </select>
                  </label>
                  {selectedPermission && <p className="platform-permission-description">{delegablePermissions.find(item => item.permission_key === selectedPermission)?.description}</p>}
                  <div className="platform-delegation-options">
                    <label><input type="checkbox" checked={canAssignToRoles} onChange={event => setCanAssignToRoles(event.target.checked)}/> School Admin can assign it to custom roles</label>
                    <label><input type="checkbox" checked={canOverrideUsers} onChange={event => setCanOverrideUsers(event.target.checked)}/> School Admin can grant individual user overrides</label>
                  </div>
                  <button className="primary" type="submit" disabled={grantDelegation.isPending || !selectedPermission || (!canAssignToRoles && !canOverrideUsers)}>
                    {grantDelegation.isPending ? "Delegating…" : "Grant permission to school"}
                  </button>
                </form>
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

      {activeTab === "school" && canOnboardSchool && (
        <section className="form-panel platform-form-panel platform-school-form">
          <div className="platform-form-title"><Building2 size={19}/><div><h2>Onboard a new school</h2><p>Create the school and its first School Admin account together.</p></div></div>
          <form onSubmit={submitSchool}>
            <div className="platform-form-fields">
              <label>School name<input name="name" autoComplete="organization" maxLength={150} required /></label>
              <label>School code<input name="code" autoCapitalize="characters" minLength={2} maxLength={50} pattern="[A-Za-z0-9][A-Za-z0-9_-]{1,49}" title="Use 2–50 letters, numbers, underscores or hyphens; start with a letter or number." required /></label>
              <label>School Admin username<input name="school_admin_username" autoComplete="username" autoCapitalize="none" minLength={3} maxLength={32} pattern="[A-Za-z0-9][A-Za-z0-9._-]{2,31}" title="Use 3–32 letters, numbers, dots, underscores or hyphens; start with a letter or number." required /></label>
              <label>Initial password<div className="password-input-wrap"><input name="school_admin_initial_password" type={showSchoolAdminPassword ? "text" : "password"} autoComplete="new-password" minLength={12} required /><button className="password-visibility" type="button" aria-label={showSchoolAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showSchoolAdminPassword} onClick={() => setShowSchoolAdminPassword(value => !value)}>{showSchoolAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small>At least 12 characters. The School Admin must change it after signing in.</small></label>
            </div>
            <button className="primary" type="submit" disabled={onboardSchool.isPending}>{onboardSchool.isPending ? "Creating school…" : "Create school and School Admin"}</button>
          </form>
        </section>
      )}
    </div>
  );
}
