import { Fragment, useMemo, useRef, useState, type FormEvent } from "react";
import { notifyToast } from "../../../shared/notifications";
import { BarChart3, BookOpen, CalendarDays, Download, FileText, KeyRound, LayoutDashboard, LockKeyhole, Pencil, Plus, Search, Shield, ShieldCheck, UserCheck, UserMinus } from "lucide-react";
import { StaffRoleAssignments } from "../components/StaffRoleAssignments";
import { StaffPermissionOverrides } from "../components/StaffPermissionOverrides";
import { TeacherAssignmentsPage } from "./TeacherAssignmentsPage";
import { StaffAcademicAssignmentsPanel } from "../components/StaffAcademicAssignmentsPanel";
import { StaffHistoryActivityPanel } from "../components/StaffHistoryActivityPanel";
import { StaffWorkloadPanel } from "../components/StaffWorkloadPanel";
import { StaffAttendanceWorkspace } from "../components/StaffAttendanceWorkspace";
import { StaffLeaveWorkspace } from "../components/StaffLeaveWorkspace";
import { StaffSubstitutionWorkspace } from "../components/StaffSubstitutionWorkspace";
import { StaffWorkloadReportWorkspace } from "../components/StaffWorkloadReportWorkspace";
import { StaffOperationalReportsWorkspace } from "../components/StaffOperationalReportsWorkspace";
import { StaffIDCardsWorkspace } from "../components/StaffIDCardsWorkspace";
import {
  useCreateSchoolStaffMutation, useDisableSchoolStaffMutation, useEnableSchoolStaffMutation,
  useResetSchoolStaffPasswordMutation, useSchoolStaffQuery, useUpdateSchoolStaffMutation,
  useStaffQualificationsQuery, useStaffCertificationsQuery, useStaffExperienceQuery,
  useCreateStaffQualificationMutation, useCreateStaffCertificationMutation, useCreateStaffExperienceMutation,
  useUpdateStaffQualificationMutation, useUpdateStaffCertificationMutation, useUpdateStaffExperienceMutation,
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

function staffCSVCell(value:unknown){
  const text=String(value??"");
  const safe=/^\s*[=+\-@]/.test(text)?"'"+text:text;
  return '"'+safe.replace(/"/g,'""')+'"';
}
function exportStaffDirectoryCSV(rows:StaffRecord[]){
  const headers=["Employee Code","Name","Designation","Employment Status","Login Account","Profile Status"];
  const lines=[headers,...rows.map(item=>[
    item.employee_code,item.full_name||item.username,item.designation,item.employment_status,
    item.status==="active"?"Enabled":"Disabled",item.profile_complete===true?"Complete":"Incomplete",
  ])].map(row=>row.map(staffCSVCell).join(",")).join("\r\n");
  const blob=new Blob(["\uFEFF",lines],{type:"text/csv;charset=utf-8"});
  const url=URL.createObjectURL(blob);
  const link=document.createElement("a");
  link.href=url;
  link.download="staff-directory-"+new Intl.DateTimeFormat("en-CA",{timeZone:"Asia/Kolkata",year:"numeric",month:"2-digit",day:"2-digit"}).format(new Date())+".csv";
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(()=>URL.revokeObjectURL(url),1000);
}

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
  const [activeTab, setActiveTab] = useState<"overview" | "directory" | "reports" | "assignments" | "attendance" | "leave" | "substitutions" | "workloadReport" | "idCards">("overview");
  const [search, setSearch] = useState("");
  const [employmentFilter, setEmploymentFilter] = useState("all");
  const [accountFilter, setAccountFilter] = useState("all");
  const [profileFilter, setProfileFilter] = useState("all");
  const [action, setAction] = useState<Action>(null);
  const [selected, setSelected] = useState<StaffRecord | null>(null);
  const [profileFor, setProfileFor] = useState<StaffRecord | null>(null);
  const [profileTab, setProfileTab] = useState<"Overview" | "Personal Details" | "Employment" | "Qualifications & Experience" | "Documents" | "Academic Assignments" | "Account & Access" | "History & Activity" | "Workload & Timetable">("Overview");
  const staffRecordID = profileFor?.principal_id ?? 0;
  const staffRecordsEnabled = Boolean(profileFor) && profileTab === "Qualifications & Experience";
  const qualificationsQuery = useStaffQualificationsQuery(staffRecordID, staffRecordsEnabled);
  const certificationsQuery = useStaffCertificationsQuery(staffRecordID, staffRecordsEnabled);
  const experienceQuery = useStaffExperienceQuery(staffRecordID, staffRecordsEnabled);
  const createQualification = useCreateStaffQualificationMutation(staffRecordID);
  const createCertification = useCreateStaffCertificationMutation(staffRecordID);
  const createExperience = useCreateStaffExperienceMutation(staffRecordID);
  const updateQualification = useUpdateStaffQualificationMutation(staffRecordID);
  const updateCertification = useUpdateStaffCertificationMutation(staffRecordID);
  const updateExperience = useUpdateStaffExperienceMutation(staffRecordID);
  const [editingRecordID, setEditingRecordID] = useState<number | null>(null);
  const editingQualification = qualificationsQuery.data?.qualifications.find(item => item.id === editingRecordID);
  const editingCertification = certificationsQuery.data?.certifications.find(item => item.id === editingRecordID);
  const editingExperience = experienceQuery.data?.experience.find(item => item.id === editingRecordID);
  const [recordFormKind, setRecordFormKind] = useState<"qualification" | "certification" | "experience" | null>(null);
  const [recordError, setRecordError] = useState("");
  const [recordFieldErrors, setRecordFieldErrors] = useState<Record<string, string>>({});
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [createStep, setCreateStep] = useState(1);
  const wizardFormRef = useRef<HTMLFormElement>(null);
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
    return staff.filter(item => {
      const matchesSearch = !term || [item.username, item.full_name, item.employee_code, item.designation, item.mobile_number, item.status, item.employment_status].some(value => value?.toLowerCase().includes(term));
      const matchesEmployment = employmentFilter === "all" || item.employment_status === employmentFilter;
      const matchesAccount = accountFilter === "all" ||
        (accountFilter === "enabled" ? item.status === "active" : item.status !== "active");
      const matchesProfile = profileFilter === "all" ||
        (profileFilter === "complete" ? item.profile_complete === true : item.profile_complete !== true);
      return matchesSearch && matchesEmployment && matchesAccount && matchesProfile;
    });
  }, [staff, search, employmentFilter, accountFilter, profileFilter]);

  function openProfile(item: StaffRecord) { setProfileFor(item); setProfileTab("Overview"); setRecordFormKind(null); setEditingRecordID(null); setRecordError(""); }
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
  function advanceCreateStep(form: HTMLFormElement) {
    if (action !== "create" || createStep >= 4) return;
    const values = new FormData(form);
    const value = (key: string) => String(values.get(key) ?? "").trim();
    const username = value("username");
    const password = value("password");
    const fullName = value("full_name");
    const employeeCode = value("employee_code");
    const designation = value("designation");
    const mobileNumber = value("mobile_number");
    const email = value("email");
    const joiningDate = value("joining_date");
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
    setCreateStep(step => step + 1);
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    if (action === "create" && createStep < 4) { advanceCreateStep(event.currentTarget); return; }
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
  async function submitStaffRecord(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!profileFor || !recordFormKind) return;
    const form = new FormData(event.currentTarget);
    const value = (key: string) => String(form.get(key) ?? "").trim();
    setRecordError("");
    const errors: Record<string, string> = {};
    const yearValue = value("completion_year");
    const scoreValue = value("score");
    const issueDate = value("issue_date");
    const expiryDate = value("expiry_date");
    const startDate = value("start_date");
    const endDate = value("end_date");
    const isCurrent = Boolean(form.get("is_current"));
    const requiredText = (key: string, label: string, max: number) => {
      const textValue = value(key);
      if (!textValue) errors[key] = `Enter ${label.toLowerCase()}.`;
      else if (textValue.length > max) errors[key] = `${label} must be ${max} characters or fewer.`;
    };
    if (recordFormKind === "qualification") {
      requiredText("qualification_name", "Qualification name", 160);
      requiredText("institution", "Institution", 200);
      if (yearValue && (!/^\d{4}$/.test(yearValue) || Number(yearValue) < 1900 || Number(yearValue) > 2200)) errors.completion_year = "Enter a year between 1900 and 2200.";
      if (scoreValue && (!Number.isFinite(Number(scoreValue)) || Number(scoreValue) < 0)) errors.score = "Enter a valid non-negative score.";
      if (Boolean(scoreValue) !== Boolean(value("score_scale"))) errors.score_scale = "Choose a score scale when a score is entered.";
      if (value("specialization").length > 160) errors.specialization = "Specialization must be 160 characters or fewer.";
      if (value("board_university").length > 200) errors.board_university = "Board / University must be 200 characters or fewer.";
      if (value("credential_reference").length > 120) errors.credential_reference = "Credential reference must be 120 characters or fewer.";
    } else if (recordFormKind === "certification") {
      requiredText("certificate_name", "Certificate name", 180);
      requiredText("issuing_organization", "Issuing organization", 200);
      if (value("credential_id").length > 120) errors.credential_id = "Credential ID must be 120 characters or fewer.";
      if (issueDate && !/^\d{4}-\d{2}-\d{2}$/.test(issueDate)) errors.issue_date = "Enter a valid issue date.";
      if (expiryDate && !/^\d{4}-\d{2}-\d{2}$/.test(expiryDate)) errors.expiry_date = "Enter a valid expiry date.";
      if (issueDate && expiryDate && expiryDate < issueDate) errors.expiry_date = "Expiry date cannot be before the issue date.";
    } else {
      requiredText("employer_name", "Employer name", 200);
      requiredText("record_designation", "Designation", 120);
      if (!startDate || !/^\d{4}-\d{2}-\d{2}$/.test(startDate)) errors.start_date = "Enter a valid start date.";
      if (isCurrent && endDate) errors.end_date = "A current role cannot have an end date.";
      else if (endDate && (!/^\d{4}-\d{2}-\d{2}$/.test(endDate) || endDate < startDate)) errors.end_date = "End date must be on or after the start date.";
      if (value("description").length > 2000) errors.description = "Description must be 2,000 characters or fewer.";
    }
    setRecordFieldErrors(errors);
    if (Object.keys(errors).length) return;
    try {
      if (recordFormKind === "qualification") {
        const year = value("completion_year");
        const score = value("score");
        const data = {
          qualification_type: value("qualification_type") as "school" | "diploma" | "undergraduate" | "postgraduate" | "doctorate" | "other",
          qualification_name: value("qualification_name"), specialization: value("specialization"),
          institution: value("institution"), board_university: value("board_university"),
          completion_year: year ? Number(year) : null, score: score ? Number(score) : null,
          score_scale: score ? value("score_scale") as "percentage" | "cgpa" | "gpa" | "marks" | "grade" : undefined,
          credential_reference: value("credential_reference"),
        };
        if (editingRecordID !== null) await updateQualification.mutateAsync({ recordID: editingRecordID, data });
        else await createQualification.mutateAsync(data);
      } else if (recordFormKind === "certification") {
        const data = {
          certificate_name: value("certificate_name"), issuing_organization: value("issuing_organization"),
          issue_date: value("issue_date"), expiry_date: value("expiry_date"), credential_id: value("credential_id"),
        };
        if (editingRecordID !== null) await updateCertification.mutateAsync({ recordID: editingRecordID, data });
        else await createCertification.mutateAsync(data);
      } else {
        const data = {
          employer_name: value("employer_name"), designation: value("record_designation"),
          employment_type: value("employment_type") as "permanent" | "contract" | "part_time" | "temporary" | "internship" | "other",
          start_date: value("start_date"), end_date: form.get("is_current") ? "" : value("end_date"),
          is_current: Boolean(form.get("is_current")), description: value("description"),
        };
        if (editingRecordID !== null) await updateExperience.mutateAsync({ recordID: editingRecordID, data });
        else await createExperience.mutateAsync(data);
      }
      setRecordFormKind(null);
      setEditingRecordID(null);
      notifyToast("success", editingRecordID !== null ? "Staff record updated." : "Staff record saved.");
    } catch (error) {
      setRecordError(error instanceof Error ? error.message : "Unable to save this record. Your entered details are still available.");
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
      <div className="heading staff-workspace-heading">
        <div><span className="eyebrow">PEOPLE & LEARNING</span><h1>Staff Management</h1><p>Manage your school's employees, access and academic responsibilities.</p></div>
        {canCreate && <button className="primary staff-create-button" type="button" onClick={() => openAction("create")}><Plus size={16}/> Add staff</button>}
      </div>
      <nav className="staff-workspace-tabs" aria-label="Staff management sections">
        <button type="button" className={activeTab === "overview" ? "active" : ""} aria-current={activeTab === "overview" ? "page" : undefined} onClick={() => setActiveTab("overview")}><LayoutDashboard size={16}/> Overview</button>
        <button type="button" className={activeTab === "directory" ? "active" : ""} aria-current={activeTab === "directory" ? "page" : undefined} onClick={() => setActiveTab("directory")}><Search size={16}/> Staff Directory</button>
        {(permissions.includes("teacher_assignments.read") || permissions.includes("class_incharge_assignments.read")) && <button type="button" className={activeTab === "assignments" ? "active" : ""} aria-current={activeTab === "assignments" ? "page" : undefined} onClick={() => setActiveTab("assignments")}><BookOpen size={16}/> Academic Assignments</button>}
        {permissions.includes("staff.attendance.read") && <button type="button" className={activeTab === "attendance" ? "active" : ""} aria-current={activeTab === "attendance" ? "page" : undefined} onClick={() => setActiveTab("attendance")}><CalendarDays size={16}/> Attendance</button>}
        {(permissions.includes("staff.leave.read") || permissions.includes("staff.holidays.read") || permissions.includes("staff.holidays.manage")) && <button type="button" className={activeTab === "leave" ? "active" : ""} aria-current={activeTab === "leave" ? "page" : undefined} onClick={() => setActiveTab("leave")}><CalendarDays size={16}/> Leave & Holidays</button>}
        {(permissions.includes("staff.substitutions.read") || permissions.includes("staff.substitutions.manage")) && <button type="button" className={activeTab === "substitutions" ? "active" : ""} aria-current={activeTab === "substitutions" ? "page" : undefined} onClick={() => setActiveTab("substitutions")}><CalendarDays size={16}/> Substitutions</button>}
        {permissions.includes("timetable.read") && <button type="button" className={activeTab === "workloadReport" ? "active" : ""} aria-current={activeTab === "workloadReport" ? "page" : undefined} onClick={() => setActiveTab("workloadReport")}><BarChart3 size={16}/> Workload Report</button>}
        {!(permissions.includes("staff.leave.read") || permissions.includes("staff.holidays.read") || permissions.includes("staff.holidays.manage")) && <button type="button" className="upcoming" disabled title="Leave read permission required to open this workspace"><CalendarDays size={16}/> Leave <span>Restricted</span></button>}
        {permissions.includes("users.read") && <button type="button" className={activeTab === "reports" ? "active" : ""} aria-current={activeTab === "reports" ? "page" : undefined} onClick={() => setActiveTab("reports")}><FileText size={16}/> Reports</button>}
        {permissions.includes("users.read") && <button type="button" className={activeTab === "idCards" ? "active" : ""} aria-current={activeTab === "idCards" ? "page" : undefined} onClick={() => setActiveTab("idCards")}><FileText size={16}/> ID Cards</button>}
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
        <form ref={wizardFormRef} onSubmit={submit} noValidate>
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
            {action === "create" ? <><button className="app-modal-cancel" type="button" onClick={closeAction} disabled={pending}>Cancel</button>{createStep > 1 && <button className="app-modal-cancel" type="button" onClick={() => {setFieldErrors({});setCreateStep(step => Math.max(1, step - 1));}} disabled={pending}>Back</button>}<button className="primary" type={action === "create" && createStep < 4 ? "button" : "submit"} disabled={pending} onPointerDown={event => { if (action === "create" && createStep < 4) event.stopPropagation(); }} onClick={event => { if (action === "create" && createStep < 4) { event.preventDefault(); if (wizardFormRef.current) advanceCreateStep(wizardFormRef.current); } }}>{pending ? "Creating…" : createStep === 4 ? "Create employee" : "Continue"}</button></>
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
          <article><span>Profile completion needed</span><strong>{query.isPending ? "—" : staff.filter(item => item.profile_complete !== true).length}</strong><small>Existing records needing attention</small></article>
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
      {activeTab === "attendance" && <StaffAttendanceWorkspace permissions={permissions}/>}
      {activeTab === "leave" && <StaffLeaveWorkspace permissions={permissions} staff={staff}/>}
      {activeTab === "substitutions" && <StaffSubstitutionWorkspace permissions={permissions} staff={staff}/>}
      {activeTab === "workloadReport" && <StaffWorkloadReportWorkspace permissions={permissions}/>}
      {activeTab === "reports" && permissions.includes("users.read") && <section className="staff-reports-workspace">
        <div className="staff-reports-heading"><div><h2>Staff Reports</h2><p>School-scoped employment, account and profile readiness summary. Sensitive contact and login details are excluded.</p></div><button type="button" className="staff-directory-export" onClick={() => exportStaffDirectoryCSV(staff)} disabled={query.isPending || query.isError || staff.length === 0}><Download size={14}/> Export staff report</button></div>
        {query.isPending && <p role="status">Loading staff report…</p>}
        {query.isError && <div className="staff-overview-error" role="alert">Unable to load staff report. <button type="button" onClick={() => void query.refetch()}>Retry</button></div>}
        {query.isSuccess && <>
          <div className="staff-reports-summary">
            <article><span>Total staff</span><strong>{staff.length}</strong></article>
            <article><span>Active employment</span><strong>{staff.filter(item => item.employment_status === "active").length}</strong></article>
            <article><span>Inactive employment</span><strong>{staff.filter(item => item.employment_status === "inactive").length}</strong></article>
            <article><span>Profile incomplete</span><strong>{staff.filter(item => item.profile_complete !== true).length}</strong></article>
            <article><span>Login disabled</span><strong>{staff.filter(item => item.status !== "active").length}</strong></article>
          </div>
          {staff.length === 0 ? <p className="table-message">No staff records are available for this school.</p> : <div className="table-scroll"><table><thead><tr><th>Employee</th><th>Employee code</th><th>Designation</th><th>Employment</th><th>Login account</th><th>Profile</th></tr></thead><tbody>{staff.map(item => <tr key={item.principal_id}><td>{item.full_name || "Profile incomplete"}</td><td>{item.employee_code || "—"}</td><td>{item.designation || "—"}</td><td>{item.employment_status || "Not set"}</td><td>{item.status === "active" ? "Enabled" : "Disabled"}</td><td>{item.profile_complete === true ? "Complete" : "Incomplete"}</td></tr>)}</tbody></table></div>}
          <p className="staff-reports-note">This summary covers current staff records only. Attendance and leave reports appear below when their separate read permissions are available. Qualifications, leave balances and payroll calculations are not included.</p>
        </>}
      </section>}
      {activeTab === "reports" && permissions.includes("users.read") && <StaffOperationalReportsWorkspace permissions={permissions}/>}
      {activeTab === "idCards" && permissions.includes("users.read") && <StaffIDCardsWorkspace staff={staff}/>}
      {activeTab === "directory" && <section className="table-panel">
        <div className="toolbar staff-directory-toolbar"><div><h2>All staff</h2><small>{filtered.length} of {staff.length} staff members</small></div><div className="staff-directory-controls"><button type="button" className="staff-directory-export" onClick={() => exportStaffDirectoryCSV(filtered)} disabled={filtered.length===0 || !permissions.includes("users.read")}><Download size={14}/> Export CSV</button><div className="search"><Search size={16}/><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search by name, code, designation or username" aria-label="Search staff by name, code, designation or username"/></div><div className="staff-directory-filters"><label>Employment<Select value={employmentFilter} onChange={setEmploymentFilter} ariaLabel="Filter by employment status" options={[{value:"all",label:"All employment"},{value:"active",label:"Active"},{value:"inactive",label:"Inactive"}]}/></label><label>Login account<Select value={accountFilter} onChange={setAccountFilter} ariaLabel="Filter by login account status" options={[{value:"all",label:"All accounts"},{value:"enabled",label:"Enabled"},{value:"disabled",label:"Disabled"}]}/></label><label>Profile<Select value={profileFilter} onChange={setProfileFilter} ariaLabel="Filter by profile completeness" options={[{value:"all",label:"All profiles"},{value:"complete",label:"Complete"},{value:"incomplete",label:"Incomplete"}]}/></label></div></div></div>
        {query.isPending ? <div className="table-message" aria-live="polite">Loading staff accounts…</div>
          : query.isError ? <div className="table-message error-text">{query.error instanceof Error ? query.error.message : "Unable to load staff accounts."}<button className="staff-retry-button" type="button" onClick={() => void query.refetch()}>Try again</button></div>
          : filtered.length === 0 ? <div className="table-message">{staff.length === 0 ? "No staff accounts yet. Add the first staff account to get started." : "No staff match the current search and filters."}</div>
          : <div className="table-scroll"><table><thead><tr><th>Employee</th><th>Employee code</th><th>Designation</th><th>Mobile</th><th>Login status</th><th>Employment</th><th>Password</th><th>Created</th><th>Actions</th></tr></thead><tbody>
            {filtered.map(item => <Fragment key={item.principal_id}><tr><td><button type="button" className="staff-profile-link" onClick={() => openProfile(item)}>{item.full_name || "Profile incomplete"}</button><small className="staff-id">{item.username} · Principal #{item.principal_id}</small></td><td>{item.employee_code || "—"}</td><td>{item.designation || "—"}</td><td>{item.mobile_number || "—"}</td><td><span className={`pill ${item.status === "active" ? "active" : "inactive"}`}>{item.status}</span></td><td>{item.employment_status ? <span className={`pill ${item.employment_status === "active" ? "active" : "inactive"}`}>{item.employment_status}</span> : "Profile pending"}</td><td>{item.must_change_password ? <span className="staff-password-required">Change required</span> : "—"}</td><td>{item.created_at ? new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(new Date(item.created_at)) : "—"}</td><td><div className="staff-row-actions">
              {canUpdate && <button type="button" onClick={() => openAction("edit", item)} title="Edit employee profile" aria-label={`Edit employee profile for ${item.username}`}><Pencil size={15}/></button>}
              {canReset && item.status === "active" && <button type="button" onClick={() => openAction("reset", item)} title="Reset password" aria-label={`Reset password for ${item.username}`}><KeyRound size={15}/></button>}
              {canReadRoles && item.status === "active" && <button type="button" className={rolesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setRolesForPrincipal(item.principal_id)} title="Manage roles" aria-label={`Manage roles for ${item.username}`} aria-expanded={rolesForPrincipal === item.principal_id}><Shield size={15}/></button>}
              {canManageOverrides && item.status === "active" && <button type="button" className={overridesForPrincipal === item.principal_id ? "selected" : ""} onClick={() => setOverridesForPrincipal(item.principal_id)} title="Manage permission overrides" aria-label={`Manage permission overrides for ${item.username}`} aria-expanded={overridesForPrincipal === item.principal_id}><LockKeyhole size={15}/></button>}
              {item.status === "active" && canDisable && <button type="button" className="danger" onClick={() => void deactivate(item)} disabled={disable.isPending} title="Disable account" aria-label={`Disable ${item.username}`}><UserMinus size={15}/></button>}
              {item.status !== "active" && item.profile_complete === true && item.employment_status === "active" && canUpdate && <button type="button" onClick={() => openAction("enable", item)} title="Enable account" aria-label={`Enable ${item.username}`}><UserCheck size={15}/></button>}
            </div></td></tr>
            </Fragment>)}
          </tbody></table></div>}
      </section>}
      {profileFor && <Modal open title={profileFor.full_name || profileFor.username} description={`Staff profile · ${profileFor.employee_code || "Employee code not set"} · ${profileFor.username}`} onClose={() => setProfileFor(null)}>
        <nav className="staff-profile-tabs" aria-label="Staff profile sections">
          {(["Overview","Personal Details","Employment","Qualifications & Experience","Documents","Academic Assignments",...(permissions.includes("timetable.read") ? ["Workload & Timetable" as const] : []),"Account & Access","History & Activity"] as const).map(tab => <button type="button" key={tab} className={profileTab === tab ? "active" : ""} aria-current={profileTab === tab ? "page" : undefined} onClick={() => setProfileTab(tab)}>{tab}</button>)}
        </nav>
        {profileTab === "Overview" && <section className="staff-profile-overview"><div><span>Employee code</span><strong>{profileFor.employee_code || "Not set"}</strong></div><div><span>Designation</span><strong>{profileFor.designation || "Not set"}</strong></div><div><span>Employment</span><strong>{profileFor.employment_status || "Profile incomplete"}</strong></div><div><span>Login</span><strong>{profileFor.status === "active" ? "Enabled" : "Disabled"}</strong></div>{canUpdate && <button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit employee details</button>}</section>}
        {profileTab === "Personal Details" && <section className="staff-profile-section"><h3>Personal details</h3><dl><div><dt>Full name</dt><dd>{profileFor.full_name || "Not set"}</dd></div><div><dt>Employee code</dt><dd>{profileFor.employee_code || "Not set"}</dd></div><div><dt>Mobile number</dt><dd>{profileFor.mobile_number || "Not set"}</dd></div><div><dt>Email address</dt><dd>{profileFor.email || "Not provided"}</dd></div></dl>{canUpdate && <button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit personal details</button>}</section>}
        {profileTab === "Employment" && <section className="staff-profile-section"><h3>Employment details</h3><dl><div><dt>Designation</dt><dd>{profileFor.designation || "Not set"}</dd></div><div><dt>Joining date</dt><dd>{profileFor.joining_date ? new Intl.DateTimeFormat("en-IN",{dateStyle:"medium"}).format(new Date(profileFor.joining_date)) : "Not provided"}</dd></div><div><dt>Employment status</dt><dd>{profileFor.employment_status || "Profile incomplete"}</dd></div><div><dt>Login status</dt><dd>{profileFor.status === "active" ? "Enabled" : "Disabled"}</dd></div></dl>{canUpdate && <button className="primary" type="button" onClick={() => {const item=profileFor;setProfileFor(null);openAction("edit",item);}}>Edit employment details</button>}</section>}
        {profileTab === "Qualifications & Experience" && <section className="staff-records-panel">
          <div className="staff-records-heading"><div><h3>Qualifications & experience</h3><p>Education, professional certificates and previous employment records for this employee.</p></div></div>
          <div className="staff-record-group">
            <div className="staff-record-group-heading"><h4>Education</h4>{canUpdate && <button className="staff-secondary-button" type="button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(null);setRecordFormKind("qualification");}}>Add qualification</button>}</div>
            {qualificationsQuery.isLoading && <p role="status">Loading qualifications…</p>}
            {qualificationsQuery.isError && <div className="staff-record-error" role="alert"><span>Unable to load qualifications.</span><button type="button" onClick={() => void qualificationsQuery.refetch()}>Retry</button></div>}
            {!qualificationsQuery.isLoading && !qualificationsQuery.isError && (qualificationsQuery.data?.qualifications.length ?? 0) === 0 && <p className="staff-record-empty">No education records added yet.</p>}
            {qualificationsQuery.data?.qualifications.map(item => <article className="staff-record-card" key={item.id}><div><strong>{item.qualification_name}</strong><span>{item.institution}{item.completion_year ? ` · ${item.completion_year}` : ""}</span>{item.specialization && <small>{item.specialization}</small>}</div><span className="staff-record-status">{item.verification_status}</span>{canUpdate && <button type="button" className="staff-text-button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(item.id);setRecordFormKind("qualification");}}>Edit</button>}</article>)}
          </div>
          <div className="staff-record-group">
            <div className="staff-record-group-heading"><h4>Certifications</h4>{canUpdate && <button className="staff-secondary-button" type="button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(null);setRecordFormKind("certification");}}>Add certification</button>}</div>
            {certificationsQuery.isLoading && <p role="status">Loading certifications…</p>}
            {certificationsQuery.isError && <div className="staff-record-error" role="alert"><span>Unable to load certifications.</span><button type="button" onClick={() => void certificationsQuery.refetch()}>Retry</button></div>}
            {!certificationsQuery.isLoading && !certificationsQuery.isError && (certificationsQuery.data?.certifications.length ?? 0) === 0 && <p className="staff-record-empty">No certifications added yet.</p>}
            {certificationsQuery.data?.certifications.map(item => <article className="staff-record-card" key={item.id}><div><strong>{item.certificate_name}</strong><span>{item.issuing_organization}</span><small>{item.issue_date || "Issue date not provided"}{item.expiry_date ? ` · Expires ${item.expiry_date}` : ""}</small></div><span className="staff-record-status">{item.verification_status}</span>{canUpdate && <button type="button" className="staff-text-button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(item.id);setRecordFormKind("certification");}}>Edit</button>}</article>)}
          </div>
          <div className="staff-record-group">
            <div className="staff-record-group-heading"><h4>Employment history</h4>{canUpdate && <button className="staff-secondary-button" type="button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(null);setRecordFormKind("experience");}}>Add experience</button>}</div>
            {experienceQuery.isLoading && <p role="status">Loading experience…</p>}
            {experienceQuery.isError && <div className="staff-record-error" role="alert"><span>Unable to load experience.</span><button type="button" onClick={() => void experienceQuery.refetch()}>Retry</button></div>}
            {!experienceQuery.isLoading && !experienceQuery.isError && (experienceQuery.data?.experience.length ?? 0) === 0 && <p className="staff-record-empty">No previous employment records added yet.</p>}
            {experienceQuery.data?.experience.map(item => <article className="staff-record-card" key={item.id}><div><strong>{item.designation}</strong><span>{item.employer_name} · {item.employment_type.replace("_"," ")}</span><small>{item.start_date} – {item.is_current ? "Present" : item.end_date || "End date not provided"}</small>{item.description && <small>{item.description}</small>}</div>{canUpdate && <button type="button" className="staff-text-button" onClick={() => {setRecordError("");setRecordFieldErrors({});setEditingRecordID(item.id);setRecordFormKind("experience");}}>Edit</button>}</article>)}
          </div>
          {!canUpdate && <p className="staff-wizard-note">You have read-only access to these records.</p>}
          <p className="staff-wizard-note">New records are saved with Pending verification. Existing records are not silently overwritten or deleted.</p>
          {recordFormKind && <form className="staff-record-form" noValidate onSubmit={submitStaffRecord}>
            <div className="staff-record-form-heading"><h4>{editingRecordID !== null ? "Edit " : "Add "}{recordFormKind === "qualification" ? "education" : recordFormKind === "certification" ? "certification" : "employment history"}</h4><button type="button" className="staff-text-button" onClick={() => {setRecordFormKind(null);setEditingRecordID(null);setRecordError("");setRecordFieldErrors({});}}>Cancel</button></div>
            {recordFormKind === "qualification" && <>
              <label>Qualification type <select name="qualification_type" required defaultValue={editingQualification?.qualification_type ?? "undergraduate"}><option value="school">School</option><option value="diploma">Diploma</option><option value="undergraduate">Undergraduate</option><option value="postgraduate">Postgraduate</option><option value="doctorate">Doctorate</option><option value="other">Other</option></select></label>
              <label>Qualification name <input name="qualification_name" required maxLength={160} defaultValue={editingQualification?.qualification_name ?? ""} aria-invalid={Boolean(recordFieldErrors.qualification_name)} aria-describedby={recordFieldErrors.qualification_name ? "staff-record-qualification-name-error" : undefined}/>{recordFieldErrors.qualification_name && <small id="staff-record-qualification-name-error" className="app-field-error" role="alert">{recordFieldErrors.qualification_name}</small>}</label>
              <label>Specialization (optional) <input name="specialization" maxLength={160} defaultValue={editingQualification?.specialization ?? ""} aria-invalid={Boolean(recordFieldErrors.specialization)} aria-describedby={recordFieldErrors.specialization ? "staff-record-specialization-error" : undefined}/>{recordFieldErrors.specialization && <small id="staff-record-specialization-error" className="app-field-error" role="alert">{recordFieldErrors.specialization}</small>}</label>
              <label>Institution <input name="institution" required maxLength={200} defaultValue={editingQualification?.institution ?? ""} aria-invalid={Boolean(recordFieldErrors.institution)} aria-describedby={recordFieldErrors.institution ? "staff-record-institution-error" : undefined}/>{recordFieldErrors.institution && <small id="staff-record-institution-error" className="app-field-error" role="alert">{recordFieldErrors.institution}</small>}</label>
              <label>Board / University (optional) <input name="board_university" maxLength={200} defaultValue={editingQualification?.board_university ?? ""} aria-invalid={Boolean(recordFieldErrors.board_university)} aria-describedby={recordFieldErrors.board_university ? "staff-record-board-error" : undefined}/>{recordFieldErrors.board_university && <small id="staff-record-board-error" className="app-field-error" role="alert">{recordFieldErrors.board_university}</small>}</label>
              <label>Completion year (optional) <input name="completion_year" type="number" min="1900" max="2200" defaultValue={editingQualification?.completion_year ?? ""} aria-invalid={Boolean(recordFieldErrors.completion_year)} aria-describedby={recordFieldErrors.completion_year ? "staff-record-year-error" : undefined}/>{recordFieldErrors.completion_year && <small id="staff-record-year-error" className="app-field-error" role="alert">{recordFieldErrors.completion_year}</small>}</label>
              <label>Score (optional) <input name="score" type="number" min="0" step="0.001" defaultValue={editingQualification?.score ?? ""} aria-invalid={Boolean(recordFieldErrors.score)} aria-describedby={recordFieldErrors.score ? "staff-record-score-error" : undefined}/>{recordFieldErrors.score && <small id="staff-record-score-error" className="app-field-error" role="alert">{recordFieldErrors.score}</small>}</label>
              <label>Score scale (required with score) <select name="score_scale" defaultValue={editingQualification?.score_scale ?? ""} aria-invalid={Boolean(recordFieldErrors.score_scale)} aria-describedby={recordFieldErrors.score_scale ? "staff-record-score-scale-error" : undefined}><option value="">Select scale</option><option value="percentage">Percentage</option><option value="cgpa">CGPA</option><option value="gpa">GPA</option><option value="marks">Marks</option><option value="grade">Grade</option></select>{recordFieldErrors.score_scale && <small id="staff-record-score-scale-error" className="app-field-error" role="alert">{recordFieldErrors.score_scale}</small>}</label>
              <label>Credential reference (optional) <input name="credential_reference" maxLength={120} defaultValue={editingQualification?.credential_reference ?? ""} aria-invalid={Boolean(recordFieldErrors.credential_reference)} aria-describedby={recordFieldErrors.credential_reference ? "staff-record-credential-reference-error" : undefined}/>{recordFieldErrors.credential_reference && <small id="staff-record-credential-reference-error" className="app-field-error" role="alert">{recordFieldErrors.credential_reference}</small>}</label>
            </>}
            {recordFormKind === "certification" && <>
              <label>Certificate name <input name="certificate_name" required maxLength={180} defaultValue={editingCertification?.certificate_name ?? ""} aria-invalid={Boolean(recordFieldErrors.certificate_name)} aria-describedby={recordFieldErrors.certificate_name ? "staff-record-certificate-error" : undefined}/>{recordFieldErrors.certificate_name && <small id="staff-record-certificate-error" className="app-field-error" role="alert">{recordFieldErrors.certificate_name}</small>}</label>
              <label>Issuing organization <input name="issuing_organization" required maxLength={200} defaultValue={editingCertification?.issuing_organization ?? ""} aria-invalid={Boolean(recordFieldErrors.issuing_organization)} aria-describedby={recordFieldErrors.issuing_organization ? "staff-record-issuer-error" : undefined}/>{recordFieldErrors.issuing_organization && <small id="staff-record-issuer-error" className="app-field-error" role="alert">{recordFieldErrors.issuing_organization}</small>}</label>
              <label>Issue date (optional) <input name="issue_date" type="date" defaultValue={editingCertification?.issue_date ?? ""} aria-invalid={Boolean(recordFieldErrors.issue_date)} aria-describedby={recordFieldErrors.issue_date ? "staff-record-issue-error" : undefined}/>{recordFieldErrors.issue_date && <small id="staff-record-issue-error" className="app-field-error" role="alert">{recordFieldErrors.issue_date}</small>}</label>
              <label>Expiry date (optional) <input name="expiry_date" type="date" defaultValue={editingCertification?.expiry_date ?? ""} aria-invalid={Boolean(recordFieldErrors.expiry_date)} aria-describedby={recordFieldErrors.expiry_date ? "staff-record-expiry-error" : undefined}/>{recordFieldErrors.expiry_date && <small id="staff-record-expiry-error" className="app-field-error" role="alert">{recordFieldErrors.expiry_date}</small>}</label>
              <label>Credential ID (optional) <input name="credential_id" maxLength={120} defaultValue={editingCertification?.credential_id ?? ""} aria-invalid={Boolean(recordFieldErrors.credential_id)} aria-describedby={recordFieldErrors.credential_id ? "staff-record-credential-id-error" : undefined}/>{recordFieldErrors.credential_id && <small id="staff-record-credential-id-error" className="app-field-error" role="alert">{recordFieldErrors.credential_id}</small>}</label>
            </>}
            {recordFormKind === "experience" && <>
              <label>Employer name <input name="employer_name" required maxLength={200} defaultValue={editingExperience?.employer_name ?? ""} aria-invalid={Boolean(recordFieldErrors.employer_name)} aria-describedby={recordFieldErrors.employer_name ? "staff-record-employer-error" : undefined}/>{recordFieldErrors.employer_name && <small id="staff-record-employer-error" className="app-field-error" role="alert">{recordFieldErrors.employer_name}</small>}</label>
              <label>Designation <input name="record_designation" required maxLength={120} defaultValue={editingExperience?.designation ?? ""} aria-invalid={Boolean(recordFieldErrors.record_designation)} aria-describedby={recordFieldErrors.record_designation ? "staff-record-designation-error" : undefined}/>{recordFieldErrors.record_designation && <small id="staff-record-designation-error" className="app-field-error" role="alert">{recordFieldErrors.record_designation}</small>}</label>
              <label>Employment type <select name="employment_type" required defaultValue={editingExperience?.employment_type ?? "permanent"}><option value="permanent">Permanent</option><option value="contract">Contract</option><option value="part_time">Part time</option><option value="temporary">Temporary</option><option value="internship">Internship</option><option value="other">Other</option></select></label>
              <label>Start date <input name="start_date" type="date" required defaultValue={editingExperience?.start_date ?? ""} aria-invalid={Boolean(recordFieldErrors.start_date)} aria-describedby={recordFieldErrors.start_date ? "staff-record-start-error" : undefined}/>{recordFieldErrors.start_date && <small id="staff-record-start-error" className="app-field-error" role="alert">{recordFieldErrors.start_date}</small>}</label>
              <label>End date (optional) <input name="end_date" type="date" defaultValue={editingExperience?.end_date ?? ""} aria-invalid={Boolean(recordFieldErrors.end_date)} aria-describedby={recordFieldErrors.end_date ? "staff-record-end-error" : undefined}/>{recordFieldErrors.end_date && <small id="staff-record-end-error" className="app-field-error" role="alert">{recordFieldErrors.end_date}</small>}</label>
              <label className="staff-record-checkbox"><input name="is_current" type="checkbox" defaultChecked={editingExperience?.is_current ?? false}/> This is my current employment</label>
              <label>Description (optional) <textarea name="description" maxLength={2000} rows={3} defaultValue={editingExperience?.description ?? ""} aria-invalid={Boolean(recordFieldErrors.description)} aria-describedby={recordFieldErrors.description ? "staff-record-description-error" : undefined}/>{recordFieldErrors.description && <small id="staff-record-description-error" className="app-field-error" role="alert">{recordFieldErrors.description}</small>}</label>
            </>}
            {recordError && <p className="staff-feedback error" role="alert">{recordError}</p>}
            <div className="staff-record-form-actions"><button type="button" className="staff-secondary-button" onClick={() => {setRecordFormKind(null);setRecordError("");setRecordFieldErrors({});}}>Cancel</button><button className="primary" type="submit" disabled={createQualification.isPending || createCertification.isPending || createExperience.isPending || updateQualification.isPending || updateCertification.isPending || updateExperience.isPending}>{createQualification.isPending || createCertification.isPending || createExperience.isPending || updateQualification.isPending || updateCertification.isPending || updateExperience.isPending ? "Saving…" : editingRecordID !== null ? "Save changes" : "Save record"}</button></div>
          </form>}
        </section>}
        {profileTab === "Documents" && <section className="staff-profile-placeholder"><h3>Staff documents</h3><p>Document upload is not enabled yet. It will be added only with private file storage, staff-scoped access checks and secure download authorization.</p></section>}
        {profileTab === "Workload & Timetable" && permissions.includes("timetable.read") && <StaffWorkloadPanel staff={profileFor}/>} 
        {profileTab === "Academic Assignments" && <StaffAcademicAssignmentsPanel staff={profileFor} permissions={permissions} onOpenWorkspace={() => {setProfileFor(null);setActiveTab("assignments");}}/>}
        {profileTab === "Account & Access" && <section className="staff-profile-access">{canReadRoles ? <StaffRoleAssignments staff={profileFor} canAssign={canAssignRoles}/> : <p>You do not have permission to view role assignments.</p>}{canManageOverrides ? <StaffPermissionOverrides staff={profileFor}/> : <p>You do not have permission to manage individual permission overrides.</p>}<p className="staff-wizard-note">Login identity, role grants and individual overrides remain separate from employee profile details.</p></section>}
        {profileTab === "History & Activity" && <StaffHistoryActivityPanel staff={profileFor} permissions={permissions}/>}
      </Modal>}
      {rolesForPrincipal !== null && canReadRoles && (() => { const target=staff.find(item=>item.principal_id===rolesForPrincipal); return target ? <Modal open title={`Access roles · ${target.username}`} description="Review the roles assigned to this staff account. Add or remove roles here; changes take effect on the next protected request." onClose={()=>setRolesForPrincipal(null)}><StaffRoleAssignments staff={target} canAssign={canAssignRoles}/></Modal> : null; })()}
      {overridesForPrincipal !== null && canManageOverrides && (() => { const target=staff.find(item=>item.principal_id===overridesForPrincipal); return target ? <Modal open title={`Permission overrides · ${target.username}`} description="Review individual access exceptions. Use an override only when the staff member's role permissions need a documented exception." onClose={()=>setOverridesForPrincipal(null)}><StaffPermissionOverrides staff={target}/></Modal> : null; })()}
      <p className="staff-security-note"><ShieldCheck size={15}/> Every action is checked again by the server using your current school permissions.</p>
      <ConfirmDialog open={Boolean(staffToDisable)} title="Disable staff account?" description={staffToDisable ? `Disable ${staffToDisable.username}? Their active sessions will be revoked and they will no longer be able to sign in.` : ""} confirmLabel="Disable account" pending={disable.isPending} error={disableError || undefined} onCancel={() => {setStaffToDisable(null);setDisableError("");}} onConfirm={() => void confirmDeactivate()}/>
    </div>
  );
}
