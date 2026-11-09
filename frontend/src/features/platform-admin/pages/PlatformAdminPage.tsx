import { useState, type FormEvent } from "react";
import { Building2, Eye, EyeOff, ShieldCheck, UserPlus, Users } from "lucide-react";
import {
  useCreatePlatformAdminMutation,
  useDeactivatePlatformAdminMutation,
  useOnboardSchoolMutation,
  usePlatformAdminsQuery,
} from "../queries";
import { Modal } from "../../../shared/components/Modal";
import "./PlatformAdminPage.css";

type Props = { permissions: string[] };

export function PlatformAdminPage({ permissions }: Props) {
  const canManageAdmins = permissions.includes("platform.admins.manage");
  const canOnboardSchool = permissions.includes("platform.schools.create") &&
    permissions.includes("platform.school_admins.create");
  const [activeTab, setActiveTab] = useState<"admins" | "school">(canManageAdmins ? "admins" : "school");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [showAdminPassword, setShowAdminPassword] = useState(false);
  const [showSchoolAdminPassword, setShowSchoolAdminPassword] = useState(false);
  const [adminDialogOpen, setAdminDialogOpen] = useState(false);
  const [schoolDialogOpen, setSchoolDialogOpen] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

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
    if (!/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(email)) errors.email = "Enter a valid email address.";
    if (password.length < 12) errors.password = "Password must contain at least 12 characters.";
    setFieldErrors(errors);
    if (Object.keys(errors).length) return;
    setFeedback(null);
    try {
      const result = await createAdmin.mutateAsync({ email, password,
      });
      formElement.reset();
      setAdminDialogOpen(false);
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
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const name = String(form.get("name") ?? "").trim();
    const code = String(form.get("code") ?? "").trim();
    const username = String(form.get("school_admin_username") ?? "").trim();
    const password = String(form.get("school_admin_initial_password") ?? "");
    const errors: Record<string, string> = {};
    if (!name) errors.name = "Enter the school name.";
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
      setFeedback({
        kind: "success",
        message: `School ${result.school.name} (${result.school.code}) created. School Admin username: ${result.school_admin.username}. They must change the initial password after sign-in.`,
      });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Unable to onboard school." });
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

          <div className="platform-create-action"><p>Create an additional global administrator account. New accounts must enroll MFA on first sign-in.</p><button type="button" className="primary" onClick={() => {setFieldErrors({});setAdminDialogOpen(true);}}><UserPlus size={16}/> Create Platform Admin</button></div>
        </div>
      )}

      {activeTab === "school" && canOnboardSchool && <section className="platform-create-action"><p>Create a school and its first School Admin account together.</p><button type="button" className="primary" onClick={() => {setFieldErrors({});setSchoolDialogOpen(true);}}><Building2 size={16}/> Onboard a new school</button></section>}
      <Modal open={adminDialogOpen} title="Create Platform Admin" description="The new administrator must enroll multi-factor authentication on first sign-in." onClose={() => setAdminDialogOpen(false)} closeDisabled={createAdmin.isPending}>
        <form onSubmit={submitAdmin} noValidate>
          <label>Email address<input name="email" type="email" autoComplete="email" maxLength={254} required aria-invalid={Boolean(fieldErrors.email)}/>{fieldErrors.email && <small className="app-field-error">{fieldErrors.email}</small>}</label>
          <label>Initial password<div className="password-input-wrap"><input name="password" type={showAdminPassword ? "text" : "password"} autoComplete="new-password" minLength={12} required aria-invalid={Boolean(fieldErrors.password)}/><button className="password-visibility" type="button" aria-label={showAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showAdminPassword} onClick={() => setShowAdminPassword(value => !value)}>{showAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small>At least 12 characters. The server also checks password safety.</small>{fieldErrors.password && <small className="app-field-error">{fieldErrors.password}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="platform-inline-button" onClick={() => setAdminDialogOpen(false)} disabled={createAdmin.isPending}>Cancel</button><button className="primary" type="submit" disabled={createAdmin.isPending}>{createAdmin.isPending ? "Creating account…" : "Create Platform Admin"}</button></div>
        </form>
      </Modal>
      <Modal open={schoolDialogOpen} title="Onboard a new school" description="Create the school and its first School Admin account together. The admin must change the initial password after signing in." onClose={() => setSchoolDialogOpen(false)} closeDisabled={onboardSchool.isPending}>
        <form onSubmit={submitSchool} noValidate>
          <label>School name<input name="name" autoComplete="organization" maxLength={150} required aria-invalid={Boolean(fieldErrors.name)}/>{fieldErrors.name && <small className="app-field-error">{fieldErrors.name}</small>}</label>
          <label>School code<input name="code" autoCapitalize="characters" minLength={2} maxLength={50} pattern="[A-Za-z0-9][A-Za-z0-9_-]{1,49}" required aria-invalid={Boolean(fieldErrors.code)}/><small>Use 2–50 letters, numbers, underscores or hyphens; start with a letter or number.</small>{fieldErrors.code && <small className="app-field-error">{fieldErrors.code}</small>}</label>
          <label>School Admin username<input name="school_admin_username" autoComplete="username" autoCapitalize="none" minLength={3} maxLength={32} pattern="[A-Za-z0-9][A-Za-z0-9._-]{2,31}" required aria-invalid={Boolean(fieldErrors.school_admin_username)}/><small>Use 3–32 letters, numbers, dots, underscores or hyphens.</small>{fieldErrors.school_admin_username && <small className="app-field-error">{fieldErrors.school_admin_username}</small>}</label>
          <label>Initial password<div className="password-input-wrap"><input name="school_admin_initial_password" type={showSchoolAdminPassword ? "text" : "password"} autoComplete="new-password" minLength={12} required aria-invalid={Boolean(fieldErrors.school_admin_initial_password)}/><button className="password-visibility" type="button" aria-label={showSchoolAdminPassword ? "Hide initial password" : "Show initial password"} aria-pressed={showSchoolAdminPassword} onClick={() => setShowSchoolAdminPassword(value => !value)}>{showSchoolAdminPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div><small>At least 12 characters. The School Admin must change it after signing in.</small>{fieldErrors.school_admin_initial_password && <small className="app-field-error">{fieldErrors.school_admin_initial_password}</small>}</label>
          <div className="app-modal-actions"><button type="button" className="platform-inline-button" onClick={() => setSchoolDialogOpen(false)} disabled={onboardSchool.isPending}>Cancel</button><button className="primary" type="submit" disabled={onboardSchool.isPending}>{onboardSchool.isPending ? "Creating school…" : "Create school and School Admin"}</button></div>
        </form>
      </Modal>
    </div>
  );
}
