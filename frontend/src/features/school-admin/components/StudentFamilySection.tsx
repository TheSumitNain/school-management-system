import { useDeferredValue, useEffect, useState, type FormEvent } from "react";

function guardianEmailFieldError(error: unknown): string | undefined {
  if (!error || typeof error !== "object" || !("fieldErrors" in error)) return undefined;
  const fieldErrors = (error as { fieldErrors?: unknown }).fieldErrors;
  if (!fieldErrors || typeof fieldErrors !== "object") return undefined;
  const message = (fieldErrors as Record<string, unknown>).email_address;
  return typeof message === "string" ? message : undefined;
}
import { Modal } from "../../../shared/components/Modal";
import { ConfirmDialog } from "../../../shared/components/ConfirmDialog";
import { Select } from "../../../shared/components/Select";
import { StudentFieldLabel } from "./StudentFieldLabel";
import type { StudentProfile } from "../api";
import {
  useCreateSchoolGuardianMutation,
  useUpdateSchoolGuardianMutation,
  useLinkSchoolStudentGuardianMutation,
  useRevokeSchoolStudentGuardianMutation,
  useUpdateSchoolStudentGuardianMutation,
  useSchoolGuardiansQuery,
  useSchoolStudentGuardiansQuery,
  useSchoolStudentRelatedStudentsQuery,
} from "../queries";

type Props = {
  student: StudentProfile;
  canManage: boolean;
  canReadRelatedStudents: boolean;
  onClose: () => void;
};

export function StudentFamilySection({ student, canManage, canReadRelatedStudents, onClose }: Props) {
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search.trim());
  const [selectedGuardianID, setSelectedGuardianID] = useState(0);
  const [relationshipType, setRelationshipType] = useState("");
  const [customRelationship, setCustomRelationship] = useState("");
  const [editRelationshipCustom, setEditRelationshipCustom] = useState("");
  const [primaryContact, setPrimaryContact] = useState(false);
  const [emergencyContact, setEmergencyContact] = useState(false);
  const [includeHistory, setIncludeHistory] = useState(false);
  const [showCreateGuardian, setShowCreateGuardian] = useState(false);
  const [newGuardianName, setNewGuardianName] = useState("");
  const [newGuardianMobile, setNewGuardianMobile] = useState("");
  const [newGuardianEmail, setNewGuardianEmail] = useState("");
  const [newGuardianAlternateMobile, setNewGuardianAlternateMobile] = useState("");
  const [newGuardianOccupation, setNewGuardianOccupation] = useState("");
  const [newGuardianEducation, setNewGuardianEducation] = useState("");
  const [newGuardianWorkplace, setNewGuardianWorkplace] = useState("");
  const [editingGuardianID, setEditingGuardianID] = useState(0);
  const [editingLinkID, setEditingLinkID] = useState(0);
  const [editRelationshipForm, setEditRelationshipForm] = useState({ relationship_type: "", is_primary_contact: false, is_emergency_contact: false });
  const [editRelationshipError, setEditRelationshipError] = useState("");
  const [editRelationshipFieldError, setEditRelationshipFieldError] = useState("");
  const [editGuardianForm, setEditGuardianForm] = useState({
    full_name: "", mobile_number: "", email_address: "", alternate_mobile_number: "",
    occupation: "", education: "", workplace_name: "", status: "active" as "active" | "inactive" | "archived",
  });
  const [editGuardianError, setEditGuardianError] = useState("");
  const [editGuardianNameError, setEditGuardianNameError] = useState("");
  const [editGuardianEmailError, setEditGuardianEmailError] = useState("");
  const [confirmSharedGuardianSave, setConfirmSharedGuardianSave] = useState(false);
  const [pendingRevokeID, setPendingRevokeID] = useState(0);
  const [revokeReason, setRevokeReason] = useState("");
  const [guardianSelectError, setGuardianSelectError] = useState("");
  const [relationshipError, setRelationshipError] = useState("");
  const [newGuardianNameError, setNewGuardianNameError] = useState("");
  const [newGuardianEmailError, setNewGuardianEmailError] = useState("");
  const [newGuardianFieldErrors, setNewGuardianFieldErrors] = useState<Record<string, string>>({});
  const [editGuardianFieldErrors, setEditGuardianFieldErrors] = useState<Record<string, string>>({});
  const [revokeReasonError, setRevokeReasonError] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const linksQuery = useSchoolStudentGuardiansQuery(student.id, includeHistory);
  const guardiansQuery = useSchoolGuardiansQuery({ q: deferredSearch, status: "active" }, canManage);
  const relatedQuery = useSchoolStudentRelatedStudentsQuery(student.id, canReadRelatedStudents);
  const createGuardian = useCreateSchoolGuardianMutation();
  const updateGuardian = useUpdateSchoolGuardianMutation();
  const linkGuardian = useLinkSchoolStudentGuardianMutation();
  const revokeLink = useRevokeSchoolStudentGuardianMutation();
  const updateLink = useUpdateSchoolStudentGuardianMutation();

  const allLinks = linksQuery.data?.student_guardians ?? [];
  const visibleLinks = includeHistory ? allLinks : allLinks.filter(link => !link.revoked_at);
  // A revoked relationship is history, not a permanent ban on re-linking.
  // V12 keeps history rows while its partial unique index protects active pairs.
  const linkedGuardianIDs = new Set(allLinks.filter(link => !link.revoked_at).map(link => link.guardian_id));
  const guardianOptions = (guardiansQuery.data?.guardians ?? [])
    .filter(guardian => !linkedGuardianIDs.has(guardian.id))
    .map(guardian => ({
      value: String(guardian.id),
      label: guardian.full_name + (guardian.mobile_number ? " · " + guardian.mobile_number : guardian.email_address ? " · " + guardian.email_address : ""),
    }));
  const relatedStudents = relatedQuery.data?.related_students ?? [];
  const busy = createGuardian.isPending || updateGuardian.isPending || linkGuardian.isPending || revokeLink.isPending || updateLink.isPending;

  // Never carry a guardian form draft or history filter from one student to another.
  useEffect(() => {
    setSearch("");
    setSelectedGuardianID(0);
    setRelationshipType("");
    setCustomRelationship("");
    setPrimaryContact(false);
    setEmergencyContact(false);
    setIncludeHistory(false);
    setShowCreateGuardian(false);
    setNewGuardianName("");
    setNewGuardianMobile("");
    setNewGuardianEmail("");
    setNewGuardianAlternateMobile("");
    setNewGuardianOccupation("");
    setNewGuardianEducation("");
    setNewGuardianWorkplace("");
    setEditingGuardianID(0);
    setEditingLinkID(0);
    setEditRelationshipForm({ relationship_type: "", is_primary_contact: false, is_emergency_contact: false });
    setEditRelationshipCustom("");
    setEditRelationshipError("");
    setEditRelationshipFieldError("");
    setEditGuardianForm({
      full_name: "", mobile_number: "", email_address: "", alternate_mobile_number: "",
      occupation: "", education: "", workplace_name: "", status: "active",
    });
    setEditGuardianError("");
    setEditGuardianNameError("");
    setEditGuardianEmailError("");
    setConfirmSharedGuardianSave(false);
    setPendingRevokeID(0);
    setRevokeReason("");
    setGuardianSelectError("");
    setRelationshipError("");
    setNewGuardianNameError("");
    setNewGuardianEmailError("");
    setNewGuardianFieldErrors({});
    setEditGuardianFieldErrors({});
    setRevokeReasonError("");
    setFeedback(null);
  }, [student.id]);


  async function saveGuardian(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNewGuardianNameError("");
    setNewGuardianEmailError("");
    if (!canManage) return;
    const nameLength = Array.from(newGuardianName.trim()).length;
    const nameError = !newGuardianName.trim() ? "Guardian name is required." : nameLength > 200 ? "Guardian name must be 200 characters or fewer." : "";
    const emailInput = event.currentTarget.elements.namedItem("guardian_email_address") as HTMLInputElement | null;
    const emailLength = Array.from(newGuardianEmail.trim()).length;
    const emailError = emailLength > 254 ? "Email address must be 254 characters or fewer." :
      newGuardianEmail.trim() && emailInput && !emailInput.checkValidity() ? "Enter a valid email address." : "";
    const fieldErrors: Record<string, string> = {};
    if (Array.from(newGuardianMobile.trim()).length > 30) fieldErrors.mobile_number = "Mobile number must be 30 characters or fewer.";
    if (Array.from(newGuardianAlternateMobile.trim()).length > 30) fieldErrors.alternate_mobile_number = "Alternate mobile number must be 30 characters or fewer.";
    if (Array.from(newGuardianOccupation.trim()).length > 120) fieldErrors.occupation = "Occupation must be 120 characters or fewer.";
    if (Array.from(newGuardianEducation.trim()).length > 120) fieldErrors.education = "Education must be 120 characters or fewer.";
    if (Array.from(newGuardianWorkplace.trim()).length > 160) fieldErrors.workplace_name = "Workplace must be 160 characters or fewer.";
    setNewGuardianNameError(nameError);
    setNewGuardianEmailError(emailError);
    setNewGuardianFieldErrors(fieldErrors);
    if (nameError || emailError || Object.keys(fieldErrors).length > 0) return;
    try {
      const result = await createGuardian.mutateAsync({
        full_name: newGuardianName.trim(),
        mobile_number: newGuardianMobile.trim() || null,
        email_address: newGuardianEmail.trim() || null,
        alternate_mobile_number: newGuardianAlternateMobile.trim() || null,
        occupation: newGuardianOccupation.trim() || null,
        education: newGuardianEducation.trim() || null,
        workplace_name: newGuardianWorkplace.trim() || null,
        status: "active",
      }) as { guardian?: { id: number } };
      const guardianID = result.guardian?.id;
      if (!guardianID) throw new Error("The guardian record was not returned by the server.");
      setSelectedGuardianID(guardianID);
      setSearch(newGuardianName.trim());
      setNewGuardianName("");
      setNewGuardianMobile("");
      setNewGuardianEmail("");
      setNewGuardianAlternateMobile("");
      setNewGuardianOccupation("");
      setNewGuardianEducation("");
      setNewGuardianWorkplace("");
      setNewGuardianFieldErrors({});
      setShowCreateGuardian(false);
      setFeedback({ kind: "success", message: "Guardian contact created. Select the relationship details and link it to this student." });
    } catch (error) {
      const emailFieldError = guardianEmailFieldError(error);
      if (emailFieldError) {
        setNewGuardianEmailError(emailFieldError);
        setFeedback(null);
      } else {
        setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Guardian contact could not be saved." });
      }
    }
  }

  function beginEditGuardian(link: (typeof allLinks)[number]) {
    setEditingGuardianID(link.guardian_id);
    setEditGuardianForm({
      full_name: link.guardian_name,
      mobile_number: link.mobile_number ?? "",
      email_address: link.email_address ?? "",
      alternate_mobile_number: link.alternate_mobile_number ?? "",
      occupation: link.occupation ?? "",
      education: link.education ?? "",
      workplace_name: link.workplace_name ?? "",
      status: link.guardian_status,
    });
    setEditGuardianError("");
    setEditGuardianNameError("");
    setEditGuardianEmailError("");
    setEditGuardianFieldErrors({});
  }

  async function saveEditedGuardian(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setEditGuardianError("");
    setEditGuardianNameError("");
    setEditGuardianEmailError("");
    if (!canManage || editingGuardianID <= 0) return;
    const name = editGuardianForm.full_name.trim();
    const nameLength = Array.from(name).length;
    const nameError = !name ? "Guardian name is required." : nameLength > 200 ? "Guardian name must be 200 characters or fewer." : "";
    const emailInput = event.currentTarget.elements.namedItem("edit_guardian_email") as HTMLInputElement | null;
    const emailLength = Array.from(editGuardianForm.email_address.trim()).length;
    const emailError = emailLength > 254 ? "Email address must be 254 characters or fewer." :
      editGuardianForm.email_address.trim() && emailInput && !emailInput.checkValidity() ? "Enter a valid email address." : "";
    const fieldErrors: Record<string, string> = {};
    if (Array.from(editGuardianForm.mobile_number.trim()).length > 30) fieldErrors.mobile_number = "Mobile number must be 30 characters or fewer.";
    if (Array.from(editGuardianForm.alternate_mobile_number.trim()).length > 30) fieldErrors.alternate_mobile_number = "Alternate mobile number must be 30 characters or fewer.";
    if (Array.from(editGuardianForm.occupation.trim()).length > 120) fieldErrors.occupation = "Occupation must be 120 characters or fewer.";
    if (Array.from(editGuardianForm.education.trim()).length > 120) fieldErrors.education = "Education must be 120 characters or fewer.";
    if (Array.from(editGuardianForm.workplace_name.trim()).length > 160) fieldErrors.workplace_name = "Workplace must be 160 characters or fewer.";
    setEditGuardianNameError(nameError);
    setEditGuardianEmailError(emailError);
    setEditGuardianFieldErrors(fieldErrors);
    if (nameError || emailError || Object.keys(fieldErrors).length > 0) return;
    const originalGuardian = allLinks.find(link => link.guardian_id === editingGuardianID);
    if (originalGuardian &&
      originalGuardian.guardian_name === editGuardianForm.full_name.trim() &&
      (originalGuardian.mobile_number ?? "") === editGuardianForm.mobile_number.trim() &&
      (originalGuardian.email_address ?? "") === editGuardianForm.email_address.trim() &&
      (originalGuardian.alternate_mobile_number ?? "") === editGuardianForm.alternate_mobile_number.trim() &&
      (originalGuardian.occupation ?? "") === editGuardianForm.occupation.trim() &&
      (originalGuardian.education ?? "") === editGuardianForm.education.trim() &&
      (originalGuardian.workplace_name ?? "") === editGuardianForm.workplace_name.trim() &&
      originalGuardian.guardian_status === editGuardianForm.status) {
      setEditingGuardianID(0);
      setFeedback({ kind: "success", message: "No changes to save." });
      return;
    }
    setConfirmSharedGuardianSave(true);
  }

  async function confirmSaveEditedGuardian() {
    if (!canManage || editingGuardianID <= 0) return;
    try {
      await updateGuardian.mutateAsync({
        guardianID: editingGuardianID,
        input: {
          full_name: editGuardianForm.full_name.trim(),
          mobile_number: editGuardianForm.mobile_number.trim() || null,
          email_address: editGuardianForm.email_address.trim() || null,
          alternate_mobile_number: editGuardianForm.alternate_mobile_number.trim() || null,
          occupation: editGuardianForm.occupation.trim() || null,
          education: editGuardianForm.education.trim() || null,
          workplace_name: editGuardianForm.workplace_name.trim() || null,
          status: editGuardianForm.status,
        },
      });
      setConfirmSharedGuardianSave(false);
      setEditingGuardianID(0);
      setEditGuardianError("");
      setFeedback({ kind: "success", message: "Guardian profile updated. Changes apply to every student linked to this guardian." });
    } catch (error) {
      setConfirmSharedGuardianSave(false);
      const emailFieldError = guardianEmailFieldError(error);
      if (emailFieldError) {
        setEditGuardianEmailError(emailFieldError);
        setEditGuardianError("");
      } else {
        setEditGuardianError(error instanceof Error ? error.message : "Guardian profile could not be updated.");
      }
    }
  }

  async function linkSelectedGuardian(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setGuardianSelectError("");
    setRelationshipError("");
    if (!canManage || linksQuery.isLoading || linksQuery.isError || guardiansQuery.isLoading || guardiansQuery.isError) return;
    if (!selectedGuardianID) setGuardianSelectError("Choose a guardian.");
    const resolvedRelationship = relationshipType === "Other" ? customRelationship.trim() : relationshipType.trim();
    if (!resolvedRelationship) setRelationshipError("Choose or specify the relationship to this student.");
    else if (Array.from(resolvedRelationship).length > 40) setRelationshipError("Relationship must be 40 characters or fewer.");
    if (!selectedGuardianID || !resolvedRelationship || Array.from(resolvedRelationship).length > 40) return;
    try {
      await linkGuardian.mutateAsync({
        studentID: student.id,
        input: {
          guardian_id: selectedGuardianID,
          relationship_type: resolvedRelationship,
          is_primary_contact: primaryContact,
          is_emergency_contact: emergencyContact,
        },
      });
      setSelectedGuardianID(0);
      setRelationshipType("");
      setCustomRelationship("");
      setPrimaryContact(false);
      setEmergencyContact(false);
      setGuardianSelectError("");
      setRelationshipError("");
      setFeedback({ kind: "success", message: "Guardian linked to this student." });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Guardian could not be linked.";
      if (/student guardian relationship conflicts with an existing record/i.test(message)) {
        setGuardianSelectError("This guardian is already linked to this student. Review the existing relationships before adding another link.");
        setFeedback(null);
        return;
      }
      setFeedback({ kind: "error", message });
    }
  }

  function beginEditRelationship(link: (typeof allLinks)[number]) {
    setEditingLinkID(link.id);
    setEditRelationshipForm({
      relationship_type: ["Father","Mother","Step-father","Step-mother","Grandfather","Grandmother","Brother","Sister","Uncle","Aunt","Legal Guardian","Foster Parent"].includes(link.relationship_type) ? link.relationship_type : "Other",
      is_primary_contact: link.is_primary_contact,
      is_emergency_contact: link.is_emergency_contact,
    });
    setEditRelationshipCustom(["Father","Mother","Step-father","Step-mother","Grandfather","Grandmother","Brother","Sister","Uncle","Aunt","Legal Guardian","Foster Parent"].includes(link.relationship_type) ? "" : link.relationship_type);
    setEditRelationshipError("");
    setEditRelationshipFieldError("");
  }

  async function saveEditedRelationship(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setEditRelationshipError("");
    setEditRelationshipFieldError("");
    if (!canManage || editingLinkID <= 0) return;
    const relationship = editRelationshipForm.relationship_type.trim();
    const resolvedRelationship = relationship === "Other" ? editRelationshipCustom.trim() : relationship;
    if (!resolvedRelationship) {
      setEditRelationshipFieldError("Choose or specify the relationship to this student.");
      return;
    }
    if (Array.from(resolvedRelationship).length > 40) {
      setEditRelationshipFieldError("Relationship must be 40 characters or fewer.");
      return;
    }
    const originalLink = allLinks.find(link => link.id === editingLinkID);
    if (originalLink &&
      originalLink.relationship_type === resolvedRelationship &&
      originalLink.is_primary_contact === editRelationshipForm.is_primary_contact &&
      originalLink.is_emergency_contact === editRelationshipForm.is_emergency_contact) {
      setEditingLinkID(0);
      setFeedback({ kind: "success", message: "No changes to save." });
      return;
    }
    try {
      await updateLink.mutateAsync({
        linkID: editingLinkID,
        input: {
          relationship_type: resolvedRelationship,
          is_primary_contact: editRelationshipForm.is_primary_contact,
          is_emergency_contact: editRelationshipForm.is_emergency_contact,
        },
      });
      setEditingLinkID(0);
      setEditRelationshipError("");
      setFeedback({ kind: "success", message: "This student's guardian relationship details were updated. Other students linked to the same guardian were not changed." });
    } catch (error) {
      setEditRelationshipError(error instanceof Error ? error.message : "Guardian relationship could not be updated.");
    }
  }

  async function confirmRevoke() {
    if (!canManage || pendingRevokeID <= 0) return;
    setRevokeReasonError("");
    if (!revokeReason.trim()) {
      setRevokeReasonError("A reason is required to revoke this relationship.");
      return;
    }
    if (Array.from(revokeReason.trim()).length > 500) {
      setRevokeReasonError("Reason must be 500 characters or fewer.");
      return;
    }
    try {
      await revokeLink.mutateAsync({ linkID: pendingRevokeID, reason: revokeReason.trim() });
      setPendingRevokeID(0);
      setRevokeReason("");
      setFeedback({ kind: "success", message: "Guardian relationship revoked. Its history is preserved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Guardian relationship could not be revoked." });
    }
  }

  return (
    <Modal
      open
      title="Parents & Siblings"
      description={student.full_name + " · Staff-managed guardian records only; this does not create a parent login account."}
      onClose={onClose}
      closeDisabled={busy}
    >
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <section className="student-family-section">
        <div className="student-family-heading">
          <div><h3>Parents / guardians</h3><p>Link existing school contacts or create a new guardian contact. Phone and email values may be shared by family members.</p></div>
          <label className="student-address-same"><input type="checkbox" checked={includeHistory} onChange={event => setIncludeHistory(event.target.checked)}/> Show revoked history</label>
        </div>
        {linksQuery.isLoading ? <p className="student-address-note">Loading guardian relationships…</p> :
          linksQuery.isError ? <div className="student-feedback error" role="alert"><p>{linksQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void linksQuery.refetch()} disabled={linksQuery.isFetching}>Retry loading guardian relationships</button></div> :
          visibleLinks.length === 0 ? <p className="student-address-note">No guardian relationships are linked to this student yet.</p> :
          <div className="student-family-list">
            {visibleLinks.map(link => (
              <article className="student-family-card" key={link.id}>
                <div>
                  <strong>{link.guardian_name}</strong>
                  <span>{link.relationship_type}{link.is_primary_contact ? " · Primary contact" : ""}{link.is_emergency_contact ? " · Emergency contact" : ""}</span>
                  <span>{link.mobile_number || "No phone"}{link.alternate_mobile_number ? " · Alt: " + link.alternate_mobile_number : ""}{link.email_address ? " · " + link.email_address : ""}</span>
                  {(link.occupation || link.education || link.workplace_name) && <small>{[link.occupation, link.education, link.workplace_name].filter(Boolean).join(" · ")}</small>}
                  {link.revoked_at && <small>Revoked {new Date(link.revoked_at).toLocaleDateString()}</small>}
                  {link.revoked_at && link.revocation_reason && <small>Reason: {link.revocation_reason}</small>}
                </div>
                {canManage && !link.revoked_at && (
                  <div className="student-family-card-actions">
                    <button type="button" className="student-link-button" onClick={() => beginEditRelationship(link)}>Edit relationship</button>
                    <button type="button" className="student-link-button" onClick={() => beginEditGuardian(link)}>Edit shared guardian details</button>
                    <button type="button" className="student-link-button" onClick={() => { setPendingRevokeID(link.id); setRevokeReason(""); setRevokeReasonError(""); }}>Revoke link</button>
                  </div>
                )}
                {canManage && editingLinkID === link.id && !link.revoked_at && (
                  <form className="student-family-create-form" onSubmit={saveEditedRelationship} noValidate>
                    <h4>Edit relationship for this student</h4>
                    <label><StudentFieldLabel label="Relationship to student" required hint="Choose the guardian's relationship to this student. Select Other to enter a custom relationship." /><Select ariaLabel="Relationship to student" value={editRelationshipForm.relationship_type} disabled={busy} invalid={Boolean(editRelationshipFieldError)} describedBy={editRelationshipFieldError ? "edit-student-guardian-relationship-error" : undefined} onChange={value => { setEditRelationshipForm(current => ({ ...current, relationship_type: value })); setEditRelationshipFieldError(""); }} placeholder="Select relationship" options={[{value:"Father",label:"Father"},{value:"Mother",label:"Mother"},{value:"Step-father",label:"Step-father"},{value:"Step-mother",label:"Step-mother"},{value:"Grandfather",label:"Grandfather"},{value:"Grandmother",label:"Grandmother"},{value:"Brother",label:"Brother"},{value:"Sister",label:"Sister"},{value:"Uncle",label:"Uncle"},{value:"Aunt",label:"Aunt"},{value:"Legal Guardian",label:"Legal Guardian"},{value:"Foster Parent",label:"Foster Parent"},{value:"Other",label:"Other"}]}/>{editRelationshipForm.relationship_type === "Other" && <input disabled={busy} value={editRelationshipCustom} maxLength={40} onChange={event => setEditRelationshipCustom(event.target.value)} placeholder="Specify relationship"/>}{editRelationshipFieldError && <small id="edit-student-guardian-relationship-error" className="app-field-error">{editRelationshipFieldError}</small>}</label>
                    <div className="student-family-checks">
                      <label><input type="checkbox" checked={editRelationshipForm.is_primary_contact} disabled={busy} onChange={event => setEditRelationshipForm(current => ({ ...current, is_primary_contact: event.target.checked }))}/> <StudentFieldLabel label="Primary contact" hint="Only one active guardian can be primary for this student. Selecting this clears the previous primary contact for this student only." /></label>
                      <label><input type="checkbox" checked={editRelationshipForm.is_emergency_contact} disabled={busy} onChange={event => setEditRelationshipForm(current => ({ ...current, is_emergency_contact: event.target.checked }))}/> <StudentFieldLabel label="Emergency contact" hint="Mark this guardian as an emergency contact for this student." /></label>
                    </div>
                    {editRelationshipError && <small className="app-field-error" role="alert">{editRelationshipError}</small>}
                    <div className="student-family-card-actions">
                      <button type="button" className="student-link-button" onClick={() => { setEditingLinkID(0); setEditRelationshipError(""); setEditRelationshipFieldError(""); }} disabled={busy}>Cancel</button>
                      <button type="submit" className="secondary" disabled={busy || linksQuery.isLoading || linksQuery.isError}>{updateLink.isPending ? "Saving…" : "Save relationship"}</button>
                    </div>
                  </form>
                )}
                {canManage && editingGuardianID === link.guardian_id && !link.revoked_at && (
                  <form className="student-family-create-form" onSubmit={saveEditedGuardian} noValidate>
                    <h4>Edit shared guardian profile</h4>
                    <p className="student-address-note">These contact and profile details are shared across all students linked to this guardian.</p>
                    <label><StudentFieldLabel label="Full name" required hint="Enter the guardian’s full name as provided by the family. Maximum 200 characters." /><input disabled={busy} value={editGuardianForm.full_name} aria-invalid={Boolean(editGuardianNameError)} aria-describedby={editGuardianNameError ? "edit-guardian-name-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, full_name: event.target.value })); setEditGuardianNameError(""); }} placeholder="Guardian full name"/>{editGuardianNameError && <small id="edit-guardian-name-error" className="app-field-error">{editGuardianNameError}</small>}</label>
                    <label><StudentFieldLabel label="Mobile number" hint="Optional contact number. Up to 30 characters; household members may share a number." /><input disabled={busy} value={editGuardianForm.mobile_number} aria-invalid={Boolean(editGuardianFieldErrors.mobile_number)} aria-describedby={editGuardianFieldErrors.mobile_number ? "edit-guardian-mobile-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, mobile_number: event.target.value })); setEditGuardianFieldErrors(current => ({...current,mobile_number:""})); }} placeholder="Mobile number"/>{editGuardianFieldErrors.mobile_number && <small id="edit-guardian-mobile-error" className="app-field-error">{editGuardianFieldErrors.mobile_number}</small>}</label>
                    <label><StudentFieldLabel label="Email address" hint="Optional. Enter a valid email address; maximum 254 characters." /><input name="edit_guardian_email" disabled={busy} type="email" value={editGuardianForm.email_address} aria-invalid={Boolean(editGuardianEmailError)} aria-describedby={editGuardianEmailError ? "edit-guardian-email-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, email_address: event.target.value })); setEditGuardianEmailError(""); }} placeholder="guardian@example.com"/>{editGuardianEmailError && <small id="edit-guardian-email-error" className="app-field-error">{editGuardianEmailError}</small>}</label>
                    <label><StudentFieldLabel label="Alternate mobile number" hint="Optional secondary contact number. Maximum 30 characters." /><input disabled={busy} value={editGuardianForm.alternate_mobile_number} aria-invalid={Boolean(editGuardianFieldErrors.alternate_mobile_number)} aria-describedby={editGuardianFieldErrors.alternate_mobile_number ? "edit-guardian-alt-mobile-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, alternate_mobile_number: event.target.value })); setEditGuardianFieldErrors(current => ({...current,alternate_mobile_number:""})); }} placeholder="Optional alternate number"/>{editGuardianFieldErrors.alternate_mobile_number && <small id="edit-guardian-alt-mobile-error" className="app-field-error">{editGuardianFieldErrors.alternate_mobile_number}</small>}</label>
                    <label><StudentFieldLabel label="Occupation" hint="Optional. Enter the guardian’s occupation. Maximum 120 characters." /><input disabled={busy} value={editGuardianForm.occupation} aria-invalid={Boolean(editGuardianFieldErrors.occupation)} aria-describedby={editGuardianFieldErrors.occupation ? "edit-guardian-occupation-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, occupation: event.target.value })); setEditGuardianFieldErrors(current => ({...current,occupation:""})); }} placeholder="e.g. Teacher, Business owner"/>{editGuardianFieldErrors.occupation && <small id="edit-guardian-occupation-error" className="app-field-error">{editGuardianFieldErrors.occupation}</small>}</label>
                    <label><StudentFieldLabel label="Education" hint="Optional highest education information. Maximum 120 characters." /><input disabled={busy} value={editGuardianForm.education} aria-invalid={Boolean(editGuardianFieldErrors.education)} aria-describedby={editGuardianFieldErrors.education ? "edit-guardian-education-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, education: event.target.value })); setEditGuardianFieldErrors(current => ({...current,education:""})); }} placeholder="Highest education (optional)"/>{editGuardianFieldErrors.education && <small id="edit-guardian-education-error" className="app-field-error">{editGuardianFieldErrors.education}</small>}</label>
                    <label><StudentFieldLabel label="Workplace / organization" hint="Optional employer or organization name. Maximum 160 characters." /><input disabled={busy} value={editGuardianForm.workplace_name} aria-invalid={Boolean(editGuardianFieldErrors.workplace_name)} aria-describedby={editGuardianFieldErrors.workplace_name ? "edit-guardian-workplace-error" : undefined} onChange={event => { setEditGuardianForm(current => ({ ...current, workplace_name: event.target.value })); setEditGuardianFieldErrors(current => ({...current,workplace_name:""})); }} placeholder="Organization name (optional)"/>{editGuardianFieldErrors.workplace_name && <small id="edit-guardian-workplace-error" className="app-field-error">{editGuardianFieldErrors.workplace_name}</small>}</label>
                    {editGuardianError && <small className="app-field-error" role="alert">{editGuardianError}</small>}
                    <div className="student-family-card-actions">
                      <button type="button" className="student-link-button" onClick={() => { setEditingGuardianID(0); setEditGuardianError(""); setEditGuardianNameError(""); setEditGuardianEmailError(""); setEditGuardianFieldErrors({}); }} disabled={busy}>Cancel</button>
                      <button type="submit" className="secondary" disabled={busy}>{updateGuardian.isPending ? "Saving…" : "Save guardian details"}</button>
                    </div>
                  </form>
                )}
              </article>
            ))}
          </div>
        }
        {canManage && (
          <>
            <form className="student-family-link-form" onSubmit={linkSelectedGuardian} noValidate>
              <h4>Link a guardian</h4>
              <label><StudentFieldLabel label="Search active guardian contacts" hint="Search existing active guardian contacts by name, phone or email." /><input value={search} onChange={event => { setSearch(event.target.value); setSelectedGuardianID(0); setGuardianSelectError(""); }} placeholder="Search by name, phone or email"/></label>
              <StudentFieldLabel label="Guardian contact" required hint="Choose an existing active guardian contact. If the contact does not exist, create it below first." />
              <Select ariaLabel="Select guardian" value={String(selectedGuardianID || "")} onChange={value => { setSelectedGuardianID(Number(value)); setGuardianSelectError(""); }} placeholder="Choose a guardian" options={guardianOptions} disabled={guardiansQuery.isLoading || guardiansQuery.isError} invalid={Boolean(guardianSelectError)} describedBy={guardianSelectError ? "student-guardian-select-error" : undefined}/>
              {guardianSelectError && <small id="student-guardian-select-error" className="app-field-error">{guardianSelectError}</small>}
              {guardiansQuery.isError && <div className="student-feedback error" role="alert"><p>{guardiansQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void guardiansQuery.refetch()} disabled={guardiansQuery.isFetching}>Retry loading guardian contacts</button></div>}
              {!guardiansQuery.isLoading && !guardiansQuery.isError && guardianOptions.length === 0 && <p className="student-address-note">No unlinked active guardian contacts match this search. You can create a new contact below.</p>}
              <label><StudentFieldLabel label="Relationship to student" required hint="Choose the guardian's relationship to this student. Select Other to enter a custom relationship." /><Select ariaLabel="Relationship to student" value={relationshipType} invalid={Boolean(relationshipError)} describedBy={relationshipError ? "student-guardian-relationship-error" : undefined} onChange={value => { setRelationshipType(value); setRelationshipError(""); }} placeholder="Select relationship" options={[{value:"Father",label:"Father"},{value:"Mother",label:"Mother"},{value:"Step-father",label:"Step-father"},{value:"Step-mother",label:"Step-mother"},{value:"Grandfather",label:"Grandfather"},{value:"Grandmother",label:"Grandmother"},{value:"Brother",label:"Brother"},{value:"Sister",label:"Sister"},{value:"Uncle",label:"Uncle"},{value:"Aunt",label:"Aunt"},{value:"Legal Guardian",label:"Legal Guardian"},{value:"Foster Parent",label:"Foster Parent"},{value:"Other",label:"Other"}]}/>{relationshipError && <small id="student-guardian-relationship-error" className="app-field-error">{relationshipError}</small>}</label>
              {relationshipType === "Other" && <label><StudentFieldLabel label="Specify relationship" required/><input value={customRelationship} onChange={event => { setCustomRelationship(event.target.value); setRelationshipError(""); }} placeholder="Enter relationship"/></label>}
              
              <div className="student-family-checks">
                <label><input type="checkbox" checked={primaryContact} onChange={event => setPrimaryContact(event.target.checked)}/> <StudentFieldLabel label="Primary contact" hint="Optional. Mark this guardian as a primary school contact for the student." /></label>
                <label><input type="checkbox" checked={emergencyContact} onChange={event => setEmergencyContact(event.target.checked)}/> <StudentFieldLabel label="Emergency contact" hint="Optional. Mark this guardian as an emergency contact." /></label>
              </div>
              
              <button type="submit" className="primary" disabled={busy || linksQuery.isLoading || linksQuery.isError || guardiansQuery.isLoading || guardiansQuery.isError}>{linkGuardian.isPending ? "Linking…" : "Link guardian"}</button>
              <button type="button" className="student-link-button" onClick={() => { setShowCreateGuardian(value => !value); setNewGuardianNameError(""); setNewGuardianEmailError(""); setNewGuardianFieldErrors({}); }}>Create a new guardian contact</button>
            </form>
            {showCreateGuardian && (
              <form className="student-family-create-form" onSubmit={saveGuardian} noValidate>
                <h4>New guardian contact</h4>
                <label><StudentFieldLabel label="Full name" required hint="Enter the guardian’s full name as provided by the family. Maximum 200 characters." /><input name="guardian_full_name" disabled={busy} value={newGuardianName} aria-invalid={Boolean(newGuardianNameError)} aria-describedby={newGuardianNameError ? "student-guardian-name-error" : undefined} onChange={event => { setNewGuardianName(event.target.value); setNewGuardianNameError(""); }} placeholder="Guardian full name"/>{newGuardianNameError && <small id="student-guardian-name-error" className="app-field-error">{newGuardianNameError}</small>}</label>
                <label><StudentFieldLabel label="Mobile number" hint="Optional contact number. Up to 30 characters; household members may share a number." /><input disabled={busy} value={newGuardianMobile} aria-invalid={Boolean(newGuardianFieldErrors.mobile_number)} aria-describedby={newGuardianFieldErrors.mobile_number ? "new-guardian-mobile-error" : undefined} onChange={event => { setNewGuardianMobile(event.target.value); setNewGuardianFieldErrors(current => ({...current,mobile_number:""})); }} placeholder="Mobile number"/>{newGuardianFieldErrors.mobile_number && <small id="new-guardian-mobile-error" className="app-field-error">{newGuardianFieldErrors.mobile_number}</small>}</label>
                <label><StudentFieldLabel label="Email address" hint="Optional. Enter a valid email address; maximum 254 characters." /><input name="guardian_email_address" disabled={busy} type="email" value={newGuardianEmail} aria-invalid={Boolean(newGuardianEmailError)} aria-describedby={newGuardianEmailError ? "student-guardian-email-error" : undefined} onChange={event => { setNewGuardianEmail(event.target.value); setNewGuardianEmailError(""); }} placeholder="guardian@example.com"/>{newGuardianEmailError && <small id="student-guardian-email-error" className="app-field-error">{newGuardianEmailError}</small>}</label>
                <label><StudentFieldLabel label="Alternate mobile number" hint="Optional secondary contact number. Maximum 30 characters." /><input disabled={busy} value={newGuardianAlternateMobile} aria-invalid={Boolean(newGuardianFieldErrors.alternate_mobile_number)} aria-describedby={newGuardianFieldErrors.alternate_mobile_number ? "new-guardian-alt-mobile-error" : undefined} onChange={event => { setNewGuardianAlternateMobile(event.target.value); setNewGuardianFieldErrors(current => ({...current,alternate_mobile_number:""})); }} placeholder="Optional alternate number"/>{newGuardianFieldErrors.alternate_mobile_number && <small id="new-guardian-alt-mobile-error" className="app-field-error">{newGuardianFieldErrors.alternate_mobile_number}</small>}</label>
                <label><StudentFieldLabel label="Occupation" hint="Optional. Enter the guardian’s occupation. Maximum 120 characters." /><input disabled={busy} value={newGuardianOccupation} aria-invalid={Boolean(newGuardianFieldErrors.occupation)} aria-describedby={newGuardianFieldErrors.occupation ? "new-guardian-occupation-error" : undefined} onChange={event => { setNewGuardianOccupation(event.target.value); setNewGuardianFieldErrors(current => ({...current,occupation:""})); }} placeholder="e.g. Teacher, Business owner"/>{newGuardianFieldErrors.occupation && <small id="new-guardian-occupation-error" className="app-field-error">{newGuardianFieldErrors.occupation}</small>}</label>
                <label><StudentFieldLabel label="Education" hint="Optional highest education information. Maximum 120 characters." /><input disabled={busy} value={newGuardianEducation} aria-invalid={Boolean(newGuardianFieldErrors.education)} aria-describedby={newGuardianFieldErrors.education ? "new-guardian-education-error" : undefined} onChange={event => { setNewGuardianEducation(event.target.value); setNewGuardianFieldErrors(current => ({...current,education:""})); }} placeholder="Highest education (optional)"/>{newGuardianFieldErrors.education && <small id="new-guardian-education-error" className="app-field-error">{newGuardianFieldErrors.education}</small>}</label>
                <label><StudentFieldLabel label="Workplace / organization" hint="Optional employer or organization name. Maximum 160 characters." /><input disabled={busy} value={newGuardianWorkplace} aria-invalid={Boolean(newGuardianFieldErrors.workplace_name)} aria-describedby={newGuardianFieldErrors.workplace_name ? "new-guardian-workplace-error" : undefined} onChange={event => { setNewGuardianWorkplace(event.target.value); setNewGuardianFieldErrors(current => ({...current,workplace_name:""})); }} placeholder="Organization name (optional)"/>{newGuardianFieldErrors.workplace_name && <small id="new-guardian-workplace-error" className="app-field-error">{newGuardianFieldErrors.workplace_name}</small>}</label>
                
                <button type="submit" className="secondary" disabled={busy}>{createGuardian.isPending ? "Saving…" : "Save guardian contact"}</button>
              </form>
            )}
            {pendingRevokeID > 0 && (
              <div className="student-location-confirm" role="alertdialog" aria-label="Revoke guardian relationship">
                <label><StudentFieldLabel label="Reason for revocation" required hint="Explain why this guardian relationship is being revoked. Maximum 500 characters; the reason is retained in the audit/history." /><input value={revokeReason} aria-invalid={Boolean(revokeReasonError)} aria-describedby={revokeReasonError ? "student-guardian-revoke-reason-error" : undefined} onChange={event => { setRevokeReason(event.target.value); setRevokeReasonError(""); }} placeholder="Explain why this link is being revoked"/>{revokeReasonError && <small id="student-guardian-revoke-reason-error" className="app-field-error">{revokeReasonError}</small>}</label>
                <div><button type="button" className="app-modal-cancel" onClick={() => { setPendingRevokeID(0); setRevokeReasonError(""); }} disabled={busy}>Cancel</button><button type="button" className="secondary" onClick={() => void confirmRevoke()} disabled={busy}>{revokeLink.isPending ? "Revoking…" : "Confirm revoke"}</button></div>
              </div>
            )}
          </>
        )}
        {!canManage && <p className="student-address-note">View-only access. Guardian records can be changed only for active students and require guardian management permission.</p>}
      </section>

      <section className="student-family-section">
        <div className="student-family-heading"><div><h3>Other students sharing a guardian</h3><p>This is derived from active guardian links. It does not assume a biological sibling relationship.</p></div></div>
        {!canReadRelatedStudents ? <p className="student-address-note">Basic student read permission is required to view related student records.</p> : relatedQuery.isLoading ? <p className="student-address-note">Loading related students…</p> :
          relatedQuery.isError ? <div className="student-feedback error" role="alert"><p>{relatedQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void relatedQuery.refetch()} disabled={relatedQuery.isFetching}>Retry loading related students</button></div> :
          relatedStudents.length === 0 ? <p className="student-address-note">No other students share an active guardian with this student.</p> :
          <div className="student-family-list">
            {relatedStudents.map(item => (
              <article className="student-family-card" key={item.student_id}>
                <div>
                  <strong>{item.full_name}</strong>
                  <span>{item.admission_number || "No current-year admission number"}{item.class_name ? " · " + item.class_name : ""}{item.section_name ? " · " + item.section_name : ""}</span>
                  <small>Shared guardian: {item.shared_guardian_names.join(", ")}</small>
                </div>
              </article>
            ))}
          </div>
        }
      </section>
      <div className="app-modal-actions"><button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button></div>
      <ConfirmDialog
        open={confirmSharedGuardianSave}
        title="Update shared guardian profile?"
        description="These changes update the guardian’s shared contact and profile record for every student linked to this guardian, not only the student currently open. Continue only after confirming the family details."
        confirmLabel="Update all linked records"
        pending={updateGuardian.isPending}
        error={editGuardianError}
        onConfirm={() => void confirmSaveEditedGuardian()}
        onCancel={() => { if (!updateGuardian.isPending) setConfirmSharedGuardianSave(false); }}
      />
    </Modal>
  );
}
