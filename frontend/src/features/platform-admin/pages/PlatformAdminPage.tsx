import { useState, type FormEvent } from "react";
import { Building2, ShieldCheck, UserPlus, Users } from "lucide-react";
import {
  useCreatePlatformAdminMutation,
  useDeactivatePlatformAdminMutation,
  useOnboardSchoolMutation,
  usePlatformAdminsQuery,
} from "../queries";
import "./PlatformAdminPage.css";

type Props = { permissions: string[] };

export function PlatformAdminPage({ permissions }: Props) {
  const canManageAdmins = permissions.includes("platform.admins.manage");
  const canOnboardSchool = permissions.includes("platform.schools.create") &&
    permissions.includes("platform.school_admins.create");
  const [activeTab, setActiveTab] = useState<"admins" | "school">(canManageAdmins ? "admins" : "school");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const adminsQuery = usePlatformAdminsQuery(canManageAdmins);
  const createAdmin = useCreatePlatformAdminMutation();
  const deactivateAdmin = useDeactivatePlatformAdminMutation();
  const onboardSchool = useOnboardSchoolMutation();

  async function submitAdmin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    setFeedback(null);
    try {
      const result = await createAdmin.mutateAsync({
        email: String(form.get("email") ?? "").trim(),
        password: String(form.get("password") ?? ""),
      });
      event.currentTarget.reset();
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

          <section className="form-panel platform-form-panel">
            <div className="platform-form-title"><UserPlus size={19}/><div><h2>Create Platform Admin</h2><p>New accounts must enroll MFA on first sign-in.</p></div></div>
            <form onSubmit={submitAdmin}>
              <label>Email address<input name="email" type="email" autoComplete="email" maxLength={254} required /></label>
              <label>Initial password<input name="password" type="password" autoComplete="new-password" minLength={12} required /><small>At least 12 characters. The server also checks password safety.</small></label>
              <button className="primary" type="submit" disabled={createAdmin.isPending}>{createAdmin.isPending ? "Creating account…" : "Create Platform Admin"}</button>
            </form>
          </section>
        </div>
      )}

      {activeTab === "school" && canOnboardSchool && (
        <section className="form-panel platform-form-panel platform-school-form">
          <div className="platform-form-title"><Building2 size={19}/><div><h2>Onboard a new school</h2><p>Create the school and its first School Admin account together.</p></div></div>
          <form onSubmit={submitSchool}>
            <div className="platform-form-fields">
              <label>School name<input name="name" autoComplete="organization" maxLength={160} required /></label>
              <label>School code<input name="code" autoCapitalize="characters" maxLength={32} required /></label>
              <label>School Admin username<input name="school_admin_username" autoComplete="username" maxLength={80} required /></label>
              <label>Initial password<input name="school_admin_initial_password" type="password" autoComplete="new-password" minLength={12} required /><small>At least 12 characters. The School Admin must change it after signing in.</small></label>
            </div>
            <button className="primary" type="submit" disabled={onboardSchool.isPending}>{onboardSchool.isPending ? "Creating school…" : "Create school and School Admin"}</button>
          </form>
        </section>
      )}
    </div>
  );
}
