import { Fragment, useMemo, useState, type FormEvent } from "react";
import { notifyToast } from "../../../shared/notifications";
import { BookOpen, CalendarDays, Clock3, FileText, KeyRound, LayoutDashboard, LockKeyhole, Pencil, Plus, Search, Shield, ShieldCheck, UserCheck, UserMinus } from "lucide-react";
import { StaffRoleAssignments } from "../components/StaffRoleAssignments";
import { StaffPermissionOverrides } from "../components/StaffPermissionOverrides";
import { TeacherAssignmentsPage } from "./TeacherAssignmentsPage";
import {
  useCreateSchoolStaffMutation, useDisableSchoolStaffMutation, useEnableSchoolStaffMutation,
  useResetSchoolStaffPasswordMutation, useSchoolStaffQuery, useUpdateSchoolStaffMutation,
} from "../queries";
import type { StaffRecord } from "../api";
import { ApiError } from "../../../lib/api";
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Select } from "../../../shared/components/Select";
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
  const [activeTab, setActiveTab] = useState<"overview" | "directory" | "assignments">("overview");
  const [search, setSearch] = useState("");
  const [action, setAction] = useState<Action>(null);
  const [selected, setSelected] = useState<StaffRecord | null>(null);
  const [profileFor, setProfileFor] = useState<StaffRecord | null>(null);
  const [profileTab, setProfileTab] = useState<"Overview" | "Personal Details" | "Employment" | "Qualifications & Experience" | "Documents" | "Academic Assignments" | "Account & Access" | "History & Activity">("Overview");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [createStep, setCreateStep] = useState(1);
  const [createReview, setCreateReview] = useState<{fullName:string;employeeCode:string;mobileNumber:string;email:string;designation:string;joiningDate:string;username:string} | null>(null);
  const [credentialHandoff, setCredentialHandoff] = useState<{ fullName: string; username: string; password: string } | null>(null);
  const [joiningDate, setJoiningDate] = useState("");
  const [employmentStatus, setEmploymentStatus] = useState<"active" | "inactive">("active");
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

  function openProfile(item: StaffRecord) { setProfileFor(item); setProfileTab("Overview"); }
  function openAction(next: Action, item: StaffRecord | null = null) {
    setSelected(item);
    setAction(next);
    setJoiningDate(item?.joining_date ? item.joining_date.slice(0, 10) : "");
    setEmploymentStatus(item?.employment_status === "inactive" ? "inactive" : "active");
    setShowPassword(false);
    setCreateStep(1);
    setCreateReview(null);
    setFieldErrors({});
    setFeedback(null);
  }
  function closeAction() {
    setAction(null);
    setSelected(null);
    setJoiningDate("");
    setEmploymentStatus("active");
    setShowPassword(false);
    setFieldErrors({});
    setCreateStep(1);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const nextErrors: Record<string, string> = {};
    const username = action === "edit" ? (selected?.username ?? "") : String(form.get("username") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const fullName = String(form.get("full_name") ?? "").trim();
    const employeeCode = String(form.get("employee_code") ?? "").trim();
    const designation = String(form.get("designation") ?? "").trim();
    const mobileNumber = String(form.get("mobile_number") ?? "").trim();
    const email = String(form.get("email") ?? "").trim();
    const joiningDate = String(form.get("joining_date") ?? "").trim();
    const employmentStatus = String(form.get("employment_status") ?? "active");
    if (action === "create" && createStep < 4) {
      const stepErrors: Record<string, string> = {};
      if (createStep === 1) {
        if (!fullName || fullName.length > 160) stepErrors.full_name = "Enter the employee full name (up to 160 characters).";
        if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$/.test(employeeCode)) stepErrors.employee_code = "Use 1–32 letters, numbers, dots, underscores or hyphens.";
        if (!/^\+?[0-9]{10,15}$/.test(mobileNumber)) stepErrors.mobile_number = "Enter one mobile number with 10–15 digits.";
        if (email && (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254)) stepErrors.email = "Enter a valid email address.";
      }
      if (createStep === 2 && (!designation || designation.length > 100)) stepErrors.designation = "Enter a designation (up to 100 characters).";
      if (createStep === 3) {
        if (!/^[a-z0-9._-]{3,32}$/.test(username)) stepErrors.username = "Use 3–32 lowercase letters, numbers, dots, underscores or hyphens.";
        if (password.length < 12) stepErrors.password = "Password must contain at least 12 characters.";
      }
      setFieldErrors(stepErrors);
      if (Object.keys(stepErrors).length) return;
      if (createStep === 3) setCreateReview({fullName,employeeCode,mobileNumber,email,designation,joiningDate,username});
      setCreateStep(step => Math.min(4, step + 1));
      return;
    }
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
        setCredentialHandoff({ fullName, username, password });
        notifyToast("success", "Employee profile and staff login created. Review the one-time credential handoff.");
        setFeedback(null);
      } else if (action === "edit" && selected) {
        await update.mutateAsync({ principalID: selected.principal_id, username, full_name: fullName, employee_code: employeeCode, designation, mobile_number: mobileNumber, email, joining_date: joiningDate || undefined, employment_status: employmentStatus as "active" | "inactive" });
        notifyToast("success", "Employee profile and login details updated.");
        setFeedback(null);
      } else if (action === "reset" && selected) {
        await reset.mutateAsync({ principalID: selected.principal_id, newPassword: password });
        notifyToast("success", "Password reset. The staff member must change it at next sign-in.");
        setFeedback(null);
      } else if (action === "enable" && selected) {
        await enable.mutateAsync({ principalID: selected.principal_id, temporaryPassword: password });
        notifyToast("success", "Staff account enabled. A password change is required at next sign-in.");
        setFeedback(null);
      }
      closeAction();
    } catch (error) {
      const message = error instanceof Error ? error.message : "The request could not be completed.";
      const mapped: Record<string, string> = {};
      if (error instanceof ApiError) {
        for (const key of ["username", "initial_password", "full_name", "employee_code", "designation", "mobile_number", "email", "joining_date", "employment_status"]) {
          if (error.fieldErrors[key]) mapped[key === "initial_password" ? "password" : key] = error.fieldErrors[key];
        }
      }
      if (message.toLowerCase().includes("employee code")) {
        mapped.employee_code = "This employee code is already used in this school.";
      }
      if (Object.keys(mapped).length) {
        setFieldErrors(current => ({ ...current, ...mapped }));
        setFeedback(null);
      } else {
        setFeedback({ kind: "error", message });
      }
      notifyToast("error", message);
    }
  }
  async function copyCredentialHandoff() {
    if (!credentialHandoff) return;
    try {
      if (!navigator.clipboard?.writeText) throw new Error("Clipboard access is unavailable.");
      await navigator.clipboard.writeText("School staff login\nName: " + credentialHandoff.fullName + "\nUsername: " + credentialHandoff.username + "\nTemporary password: " + credentialHandoff.password + "\nChange this password after your first sign-in.");
      notifyToast("success", "Credentials copied. Share them only through a verified channel.");
    } catch {
      notifyToast("error", "Clipboard access is unavailable. Use the password-reset flow if needed.");
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
      notifyToast("success", `${item.username} has been disabled and active sessions revoked.`);
      setFeedback(null);
    } catch (error) {
      setDisableError(error instanceof Error ? error.message : "Unable to disable this account. Please try again.");
    }
  }

  const pending = create.isPending || update.isPending || reset.isPending || enable.isPending;
  const formTitle = action === "create" ? "Add employee" : action === "edit" ? "Edit employee profile" : action === "reset" ? "Reset staff password" : "Enable staff account";
  const passwordLabel = action === "create" ? "Initial password" : action === "reset" ? "New temporary password" : "Temporary password";

  return (
    <div className="content staff-management-page">
      <div className="staff-workspace-heading">
        <div><h1>Staff Management</h1><p>Manage your school's employees, access and academic responsibilities.</p></div>
        {canCreate && <button className="primary staff-create-button" type="button" onClick={() => openAction("create")}><Plus size={16}/> Add staff</button>}
      </div>
      <nav className="staff-workspace-tabs" aria-label="Staff management sections">
        <button type="button" className={activeTab === "overview" ? "active" : ""} aria-current={activeTab === "overview" ? "page" : undefined} onClick={() => setActiveTab("overview")}><LayoutDashboard size={16}/> Overview</button>
        <button type="button" className={activeTab === "directory" ? "active" : ""} aria-current={activeTab === "directory" ? "page" : undefined} onClick={() => setActiveTab("directory")}><Search size={16}/> Staff Directory</button>
        {(permissions.includes("teacher_assignments.read") || permissions.includes("class_incharge_assignments.read")) && <button type="button" className={activeTab === "assignments" ? "active" : ""} aria-current={activeTab === "assignments" ? "page" : undefined} onClick={() => setActiveTab("assignments")}><BookOpen size={16}/> Academic Assignments</button>}
        <button type="button" className="upcoming" disabled title="Planned for the next phase"><CalendarDays size={16}/> Attendance <span>Phase 1</span></button>
        <button type="button" className="upcoming" disabled title="Planned for the next phase"><Clock3 size={16}/> Leave <span>Phase 1</span></button>
        <button type="button" className="upcoming" disabled title="Planned for the next phase"><BookOpen size={16}/> Workload & Timetable <span>Phase 1</span></button>
        <button type="button" className="upcoming" disabled title="Planned for the next phase"><FileText size={16}/> Reports <span>Phase 1</span></button>
      </nav>

      {feedback && (action === null || feedback.kind === "success") && <div className={`staff-feedback ${feedback.kind}`} role="status">{feedback.message}</div>}

      <Modal
        open={action !== null}
        title={formTitle}
        description={selected ? `Employee: ${selected.full_name || selected.username}. Login usernames must be unique across the platform.` : "Create the employee profile and login account together."}
        onClose={closeAction}
        closeDisabled={pending}
      >
        {feedback?.kind === "error" && <p className="staff-feedback error" role="alert">{feedback.message}</p>}
        {action === "create" && <div className="staff-wizard-progress" aria-label="Add staff progress">
          {["Identity", "Employment", "Login & Access", "Review"].map((step, index) => <div key={step} className={createStep === index + 1 ? "current" : createStep > index + 1 ? "complete" : ""} aria-current={createStep === index + 1 ? "step" : undefined}><span>{createStep > index + 1 ? "✓" : index + 1}</span><small>{step}</small></div>)}
        </div>}
        {action === "create" && createStep === 4 && <section className="staff-wizard-review"><h3>Review & create</h3><p>Check the details before creating this employee and login account.</p>
          <dl>{createReview && <><div><dt>Full name</dt><dd>{createReview.fullName}</dd></div><div><dt>Employee code</dt><dd>{createReview.employeeCode}</dd></div><div><dt>Mobile</dt><dd>{createReview.mobileNumber}</dd></div><div><dt>Email</dt><dd>{createReview.email || "Not provided"}</dd></div><div><dt>Designation</dt><dd>{createReview.designation}</dd></div><div><dt>Joining date</dt><dd>{createReview.joiningDate || "Not set"}</dd></div><div><dt>Login username</dt><dd>{createReview.username}</dd></div></>}</dl>
          <p>Temporary password change is required at first sign-in. After creation, you can explicitly copy credentials for a trusted handoff. Nothing is sent automatically.</p></section>}
        <form onSubmit={submit} noValidate>
          {action === "create" && <label hidden={createStep !== 3}>Login username<input name="username" placeholder="e.g. rahul.sharma" minLength={3} maxLength={32} pattern="[a-z0-9._-]{3,32}" autoComplete="off" required aria-invalid={Boolean(fieldErrors.username)} aria-describedby={fieldErrors.username ? "staff-username-error" : "staff-username-help"} onChange={()=>setFieldErrors(current=>({...current,username:""}))}/><small id="staff-username-help">3–32 lowercase letters, numbers, dots, underscores or hyphens. Unique across the platform.</small>{fieldErrors.username && <small className="app-field-error" id="staff-username-error" role="alert">{fieldErrors.username}</small>}</label>}
          {action === "edit" && <label>Login username<input value={selected?.username ?? ""} disabled/><small>Login identity is managed separately from employee details.</small></label>}
          {(action === "create" || action === "edit") && <>
            <label hidden={action === "create" && createStep !== 1}>Full name<input name="full_name" placeholder="e.g. Rahul Sharma" defaultValue={selected?.full_name ?? ""} maxLength={160} required aria-invalid={Boolean(fieldErrors.full_name)} aria-describedby={fieldErrors.full_name ? "staff-full-name-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,full_name:""}))}/>{fieldErrors.full_name && <small className="app-field-error" id="staff-full-name-error" role="alert">{fieldErrors.full_name}</small>}</label>
            <label hidden={action === "create" && createStep !== 1}>Employee code<input name="employee_code" placeholder="e.g. EMP-001" defaultValue={selected?.employee_code ?? ""} maxLength={32} required aria-invalid={Boolean(fieldErrors.employee_code)} aria-describedby={fieldErrors.employee_code ? "staff-employee-code-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,employee_code:""}))}/><small>Must be unique within this school.</small>{fieldErrors.employee_code && <small className="app-field-error" id="staff-employee-code-error" role="alert">{fieldErrors.employee_code}</small>}</label>
            <label hidden={action === "create" && createStep !== 2}>Designation<input name="designation" placeholder="e.g. Mathematics Teacher" defaultValue={selected?.designation ?? ""} maxLength={100} required aria-invalid={Boolean(fieldErrors.designation)} aria-describedby={fieldErrors.designation ? "staff-designation-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,designation:""}))}/>{fieldErrors.designation && <small className="app-field-error" id="staff-designation-error" role="alert">{fieldErrors.designation}</small>}</label>
            <label hidden={action === "create" && createStep !== 1}>Mobile number<input name="mobile_number" type="tel" placeholder="e.g. 9876543210" defaultValue={selected?.mobile_number ?? ""} maxLength={16} required aria-invalid={Boolean(fieldErrors.mobile_number)} aria-describedby={fieldErrors.mobile_number ? "staff-mobile-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,mobile_number:""}))}/><small>Enter one contact number (10–15 digits).</small>{fieldErrors.mobile_number && <small className="app-field-error" id="staff-mobile-error" role="alert">{fieldErrors.mobile_number}</small>}</label>
            <label hidden={action === "create" && createStep !== 1}>Email address <small>(optional)</small><input name="email" type="email" placeholder="e.g. rahul@example.com" defaultValue={selected?.email ?? ""} maxLength={254} aria-invalid={Boolean(fieldErrors.email)} aria-describedby={fieldErrors.email ? "staff-email-error" : undefined} onChange={()=>setFieldErrors(current=>({...current,email:""}))}/>{fieldErrors.email && <small className="app-field-error" id="staff-email-error" role="alert">{fieldErrors.email}</small>}</label>
            <label hidden={action === "create" && createStep !== 2}>Joining date <small>(optional)</small><input type="hidden" name="joining_date" value={joiningDate}/><DatePicker value={joiningDate} onChange={setJoiningDate} ariaLabel="Joining date" placeholder="Select joining date"/></label>
            {action === "edit" && <label>Employment status<input type="hidden" name="employment_status" value={employmentStatus}/><Select value={employmentStatus} onChange={value => { setEmploymentStatus(value as "active" | "inactive"); setFieldErrors(current => ({ ...current, employment_status: "" })); }} ariaLabel="Employment status" options={[{ value: "active", label: "Active" }, { value: "inactive", label: "Inactive" }]} invalid={Boolean(fieldErrors.employment_status)} describedBy={fieldErrors.employment_status ? "staff-employment-status-error" : undefined}/><small>Inactive employees cannot sign in. Reactivating employment does not automatically enable a disabled login.</small>{fieldErrors.employment_status && <small className="app-field-error" id="staff-employment-status-error" role="alert">{fieldErrors.employment_status}</small>}</label>}
          </>}
          {action !== "edit" && <label hidden={action === "create" && createStep !== 3}>{passwordLabel}<div className="staff-password-field"><input name="password" type={showPassword ? "text" : "password"} placeholder="Enter a strong password (12+ characters)" minLength={12} autoComplete="new-password" required aria-invalid={Boolean(fieldErrors.password)} aria-describedby={fieldErrors.password ? "staff-password-error" : "staff-password-help"} onChange={()=>setFieldErrors(current=>({...current,password:""}))}/><button type="button" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? "Hide password" : "Show password"}>{showPassword ? "Hide" : "Show"}</button></div><small id="staff-password-help">At least 12 characters. The server also checks password safety and compromised-password status.</small>{fieldErrors.password && <small className="app-field-error" id="staff-password-error" role="alert">{fieldErrors.password}</small>}</label>}
          <div className="app-modal-actions">
            {action === "create" ? <><button className="app-modal-cancel" type="button" onClick={closeAction} disabled={pending}>Cancel</button>{createStep > 1 && <button className="app-modal-cancel" type="button" onClick={() => {setFieldErrors({});setCreateStep(step => Math.max(1, step - 1));}} disabled={pending}>Back</button>}<button className="primary" type="submit" disabled={pending}>{pending ? "Creating…" : createStep === 4 ? "Create employee" : "Continue"}</button></>
              : <><button className="app-modal-cancel" type="button" onClick={closeAction} disabled={pending}>Cancel</button><button className="primary" type="submit" disabled={pending}>{pending ? "Saving…" : action === "reset" ? "Reset password" : action === "enable" ? "Enable account" : "Save changes"}</button></>}
          </div>
        </form>
      </Modal>

      <Modal open={credentialHandoff !== null} title="Share staff login credentials" description="This one-time handoff is shown only after account creation succeeds. The password is not stored for later retrieval." onClose={() => setCredentialHandoff(null)}>
        {credentialHandoff && <section className="staff-credential-handoff"><p>Confirm the staff member's identity and use a trusted channel you have verified. Nothing is sent automatically.</p>
          <dl><div><dt>Staff member</dt><dd>{credentialHandoff.fullName}</dd></div><div><dt>Login username</dt><dd>{credentialHandoff.username}</dd></div><div><dt>Temporary password</dt><dd>{credentialHandoff.password}</dd></div></dl>
          <p className="staff-wizard-note">Do not share credentials in a public or group channel. The staff member must change this password at first sign-in.</p>
          <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={() => setCredentialHandoff(null)}>Do this later</button><button type="button" className="primary" onClick={() => void copyCredentialHandoff()}>Copy credentials</button></div>
        </section>}
      </Modal>
      {activeTab === "overview" && <section className="staff-overview">
        <div className="staff-overview-intro"><div><h2>People at a glance</h2><p>A quick view of employee records and account readiness.</p></div><button type="button" className="staff-overview-link" onClick={() => setActiveTab("directory")}>Open staff directory <span aria-hidden="true">→</span></button></div>
        <div className="staff-overview-cards">
          <article><span>Total staff</span><strong>{query.isPending ? "—" : staff.length}</strong><small>Employee records in this school</small></article>
          <article><span>Active employment</span><strong>{query.isPending ? "—" : staff.filter(item => item.employment_status === "active").length}</strong><small>Currently marked as active</small></article>
          <article><span>Login enabled</span><strong>{query.isPending ? "—" : staff.filter(item => item.status === "active").length}</strong><small>Accounts currently able to sign in</small></article>
          <article><span>Profile completion needed</span><strong>{query.isPending ? "—" : staff.filter(item => !item.profile_complete).length}</strong><small>Existing records needing attention</small></article>
        </div>
        <div className="staff-overview-lower">
          <section className="staff-overview-panel"><h3>Quick actions</h3><p>Start with the most common staff administration tasks.</p><div className="staff-quick-actions">
            {canCreate && <button type="button" onClick={() => openAction("create")}><Plus size={16}/> Add staff member</button>}
            <button type="button" onClick={() => setActiveTab("directory")}><Search size={16}/> Find an employee</button>
            {(permissions.includes("teacher_assignments.read") || permissions.includes("class_incharge_assignments.read")) && <button type="button" onClick={() => setActiveTab("assignments")}><BookOpen size={16}/> Manage academic assignments</button>}
          </div></section>
          <section className="staff-overview-panel"><h3>Account readiness</h3><p>Keep employment and login status aligned without mixing the two.</p><ul className="staff-readiness-list">
            <li><span>Active employees with disabled login</span><strong>{staff.filter(item => item.employment_status === "active" && item.status !== "active").length}</strong></li>
            <li><span>Temporary password change pending</span><strong>{staff.filter(item => item.must_change_password).length}</strong></li>
            <li><span>Inactive employees</span><strong>{staff.filter(item => item.employment_status === "inactive").length}</strong></li>
          </ul></section>
        </div>
        {query.isError && <p className="staff-overview-error" role="alert">Staff summary could not be loaded. Open Staff Directory and retry the request.</p>}
      </section>}
      {activeTab === "assignments" && <TeacherAssignmentsPage permissions={permissions}/>}
      {activeTab === "directory" && <section className="table-panel">
        <div className="toolbar"><div><h2>All staff</h2><small>{staff.length} accounts</small></div><div className="search"><Search size={16}/><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search by name, code, designation or username" aria-label="Search staff by name, code, designation or username"/></div></div>
        {query.isPending ? <div className="table-message" aria-live="polite">Loading staff accounts…</div>
          : query.isError ? <div className="table-message error-text">{query.error instanceof Error ? query.error.message : "Unable to load staff accounts."}<button className="staff-retry-button" type="button" onClick={() => void query.refetch()}>Try again</button></div>
          : filtered.length === 0 ? <div className="table-message">{search ? "No staff match this search." : "No staff accounts yet. Add the first staff account to get started."}</div>
          : <div className="table-scroll"><table><thead><tr><th>Employee</th><th>Employee code</th><th>Designation</th><th>Mobile</th><th>Login status</th><th>Employment</th><th>Password</th><th>Created</th><th>Actions</th></tr></thead><tbody>
            {filtered.map(item => <Fragment key={item.principal_id}><tr><td><button type="button" className="staff-profile-link" onClick={() => openProfile(item)}>{item.full_name || "Profile incomplete"}</button><small className="staff-id">{item.username} · Principal #{item.principal_id}</small></td><td>{item.employee_code || "—"}</td><td>{item.designation || "—"}</td><td>{item.mobile_number || "—"}</td><td><span className={`pill ${item.status === "active" ? "active" : "inactive"}`}>{item.status}</span></td><td>{item.employment_status ? <span className={`pill ${item.employment_status === "active" ? "active" : "inactive"}`}>{item.employment_status}</span> : "Profile pending"}</td><td>{item.must_change_password ? <span className="staff-password-required">Change required</span> : "—"}</td><td>{item.created_at ? new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(item.created_at)) : "—"}</td><td><div className="staff-row-actions">
              {canUpdate && <button type="button" onClick={() => openAction("edit", item)} title="Edit employee profile" aria-label={`Edit employee profile for ${item.username}`}><Pencil size={15}/></button>}
              {canReset && item.status === "active" && <button type="button" onClick={() => openAction("reset", item)} title="Reset password" aria-label={`Reset password for ${item.username}`}><KeyRound size={15}/></button>}
              {canReadRoles && item.status === "active" && <button type="button" className={rolesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setRolesForPrincipal(item.principal_id)} title="Manage roles" aria-label={`Manage roles for ${item.username}`} aria-expanded={rolesForPrincipal === item.principal_id}><Shield size={15}/></button>}
              {canManageOverrides && item.status === "active" && <button type="button" className={overridesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setOverridesForPrincipal(item.principal_id)} title="Manage permission overrides" aria-label={`Manage permission overrides for ${item.username}`} aria-expanded={overridesForPrincipal === item.principal_id}><LockKeyhole size={15}/></button>}
              {item.status === "active" && canDisable && <button type="button" className="danger" onClick={() => void deactivate(item)} disabled={disable.isPending} title="Disable account" aria-label={`Disable ${item.username}`}><UserMinus size={15}/></button>}
              {item.status !== "active" && item.profile_complete && item.employment_status === "active" && canUpdate && <button type="button" onClick={() => openAction("enable", item)} title="Enable account" aria-label={`Enable ${item.username}`}><UserCheck size={15}/></button>}
            </div></td></tr>
            </Fragment>)}
          </tbody></table></div>}
      </section>}
      {profileFor && <Modal open title={profileFor.full_name || profileFor.username} description={`Staff profile · ${profileFor.employee_code || "Employee code not set"} · ${profileFor.username}`} onClose={() => setProfileFor(null)}>
        <nav className="staff-profile-tabs" aria-label="Staff profile sections">
          {(["Overview","Personal Details","Employment","Qualifications & Experience","Documents","Academic Assignments","Account & Access","History & Activity"] as const).map(tab => <button type="button" key={tab} className={profileTab === tab ? "active" : ""} aria-current={profileTab === tab ? "page" : undefined} onClick={() => setProfileTab(tab)}>{tab}</button>)}
        </nav>
        {profileTab === "Overview" && <section className="staff-profile-overview"><div><span>Employee code</span><strong>{profileFor.employee_code || "Not set"}</strong></div><div><span>Designation</span><strong>{profileFor.designation || "Not set"}</strong></div><div><span>Employment</span><strong>{profileFor.employment_status || "Profile incomplete"}</strong></div><div><span>Login</span><strong>{profileFor.status === "active" ? "Enabled" : "Disabled"}</strong></div><button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit employee details</button></section>}
        {profileTab === "Personal Details" && <section className="staff-profile-section"><h3>Personal details</h3><dl><div><dt>Full name</dt><dd>{profileFor.full_name || "Not set"}</dd></div><div><dt>Employee code</dt><dd>{profileFor.employee_code || "Not set"}</dd></div><div><dt>Mobile number</dt><dd>{profileFor.mobile_number || "Not set"}</dd></div><div><dt>Email address</dt><dd>{profileFor.email || "Not provided"}</dd></div></dl><button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit personal details</button></section>}
        {profileTab === "Employment" && <section className="staff-profile-section"><h3>Employment details</h3><dl><div><dt>Designation</dt><dd>{profileFor.designation || "Not set"}</dd></div><div><dt>Joining date</dt><dd>{profileFor.joining_date ? new Intl.DateTimeFormat("en-IN",{dateStyle:"medium"}).format(new Date(profileFor.joining_date)) : "Not provided"}</dd></div><div><dt>Employment status</dt><dd>{profileFor.employment_status || "Profile incomplete"}</dd></div><div><dt>Login status</dt><dd>{profileFor.status === "active" ? "Enabled" : "Disabled"}</dd></div></dl><button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit employment details</button></section>}
        {profileTab === "Qualifications & Experience" && <section className="staff-profile-placeholder"><h3>Qualifications & experience</h3><p>This section is part of the approved profile. Qualification, certification and experience records need their staff-specific API and database contract before they can be saved.</p></section>}
        {profileTab === "Documents" && <section className="staff-profile-placeholder"><h3>Staff documents</h3><p>Document upload is not enabled yet. It will be added only with private file storage, staff-scoped access checks and secure download authorization.</p></section>}
        {profileTab === "Academic Assignments" && <section className="staff-profile-placeholder"><h3>Academic assignments</h3><p>School-wide teacher and class-in-charge assignment management is available in the workspace tab. A staff-specific assignment view is pending an exact scoped API contract.</p><button type="button" className="staff-secondary-button" onClick={() => {setProfileFor(null);setActiveTab("assignments");}}>Open Academic Assignments workspace</button></section>}
        {profileTab === "Account & Access" && <section className="staff-profile-access">{canReadRoles ? <StaffRoleAssignments staff={profileFor} canAssign={canAssignRoles}/> : <p>You do not have permission to view role assignments.</p>}{canManageOverrides ? <StaffPermissionOverrides staff={profileFor}/> : <p>You do not have permission to manage individual permission overrides.</p>}<p className="staff-wizard-note">Login identity, role grants and individual overrides remain separate from employee profile details.</p></section>}
        {profileTab === "History & Activity" && <section className="staff-profile-placeholder"><h3>Employment history & activity</h3><p>A staff-specific history endpoint is not available yet. This tab will show audited lifecycle and profile events once the read API and permission contract are implemented.</p></section>}
      </Modal>}
      {rolesForPrincipal !== null && canReadRoles && (() => { const target=staff.find(item=>item.principal_id===rolesForPrincipal); return target ? <Modal open title={`Access roles · ${target.username}`} description="Review the roles assigned to this staff account. Add or remove roles here; changes take effect on the next protected request." onClose={()=>setRolesForPrincipal(null)}><StaffRoleAssignments staff={target} canAssign={canAssignRoles}/></Modal> : null; })()}
      {overridesForPrincipal !== null && canManageOverrides && (() => { const target=staff.find(item=>item.principal_id===overridesForPrincipal); return target ? <Modal open title={`Permission overrides · ${target.username}`} description="Review individual access exceptions. Use an override only when the staff member's role permissions need a documented exception." onClose={()=>setOverridesForPrincipal(null)}><StaffPermissionOverrides staff={target}/></Modal> : null; })()}
      <p className="staff-security-note"><ShieldCheck size={15}/> Every action is checked again by the server using your current school permissions.</p>
      <ConfirmDialog open={Boolean(staffToDisable)} title="Disable staff account?" description={staffToDisable ? `Disable ${staffToDisable.username}? Their active sessions will be revoked and they will no longer be able to sign in.` : ""} confirmLabel="Disable account" pending={disable.isPending} error={disableError || undefined} onCancel={() => {setStaffToDisable(null);setDisableError("");}} onConfirm={() => void confirmDeactivate()}/>
    </div>
  );
}
