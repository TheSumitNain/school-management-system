import { useDeferredValue, useState, type FormEvent } from "react";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import type { StudentProfile } from "../api";
import {
  useCreateSchoolGuardianMutation,
  useLinkSchoolStudentGuardianMutation,
  useRevokeSchoolStudentGuardianMutation,
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
  const [pendingRevokeID, setPendingRevokeID] = useState(0);
  const [revokeReason, setRevokeReason] = useState("");
  const [guardianSelectError, setGuardianSelectError] = useState("");
  const [relationshipError, setRelationshipError] = useState("");
  const [newGuardianNameError, setNewGuardianNameError] = useState("");
  const [newGuardianEmailError, setNewGuardianEmailError] = useState("");
  const [revokeReasonError, setRevokeReasonError] = useState("");
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const linksQuery = useSchoolStudentGuardiansQuery(student.id, true);
  const guardiansQuery = useSchoolGuardiansQuery({ q: deferredSearch, status: "active" });
  const relatedQuery = useSchoolStudentRelatedStudentsQuery(student.id, canReadRelatedStudents);
  const createGuardian = useCreateSchoolGuardianMutation();
  const linkGuardian = useLinkSchoolStudentGuardianMutation();
  const revokeLink = useRevokeSchoolStudentGuardianMutation();

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
  const busy = createGuardian.isPending || linkGuardian.isPending || revokeLink.isPending;

  async function saveGuardian(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNewGuardianNameError("");
    setNewGuardianEmailError("");
    if (!canManage) return;
    const nameError = newGuardianName.trim() ? "" : "Guardian name is required.";
    const emailInput = event.currentTarget.elements.namedItem("guardian_email_address") as HTMLInputElement | null;
    const emailError = newGuardianEmail.trim() && emailInput && !emailInput.checkValidity()
      ? "Enter a valid email address." : "";
    setNewGuardianNameError(nameError);
    setNewGuardianEmailError(emailError);
    if (nameError || emailError) return;
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
      setShowCreateGuardian(false);
      setFeedback({ kind: "success", message: "Guardian contact created. Select the relationship details and link it to this student." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Guardian contact could not be saved." });
    }
  }

  async function linkSelectedGuardian(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setGuardianSelectError("");
    setRelationshipError("");
    if (!canManage) return;
    if (!selectedGuardianID) setGuardianSelectError("Choose a guardian.");
    if (!relationshipType.trim()) setRelationshipError("Enter the relationship to this student.");
    if (!selectedGuardianID || !relationshipType.trim()) return;
    try {
      await linkGuardian.mutateAsync({
        studentID: student.id,
        input: {
          guardian_id: selectedGuardianID,
          relationship_type: relationshipType.trim(),
          is_primary_contact: primaryContact,
          is_emergency_contact: emergencyContact,
        },
      });
      setSelectedGuardianID(0);
      setRelationshipType("");
      setPrimaryContact(false);
      setEmergencyContact(false);
      setGuardianSelectError("");
      setRelationshipError("");
      setFeedback({ kind: "success", message: "Guardian linked to this student." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Guardian could not be linked." });
    }
  }

  async function confirmRevoke() {
    if (!canManage || pendingRevokeID <= 0) return;
    setRevokeReasonError("");
    if (!revokeReason.trim()) {
      setRevokeReasonError("A reason is required to revoke this relationship.");
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
          linksQuery.isError ? <p className="student-feedback error" role="alert">{linksQuery.error.message}</p> :
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
                </div>
                {canManage && !link.revoked_at && <button type="button" className="student-link-button" onClick={() => { setPendingRevokeID(link.id); setRevokeReason(""); setRevokeReasonError(""); }}>Revoke link</button>}
              </article>
            ))}
          </div>
        }
        {canManage && (
          <>
            <form className="student-family-link-form" onSubmit={linkSelectedGuardian} noValidate>
              <h4>Link a guardian</h4>
              <label>Search active guardian contacts<input value={search} onChange={event => { setSearch(event.target.value); setSelectedGuardianID(0); setGuardianSelectError(""); }} placeholder="Search by name, phone or email"/></label>
              <Select ariaLabel="Select guardian" value={String(selectedGuardianID || "")} onChange={value => { setSelectedGuardianID(Number(value)); setGuardianSelectError(""); }} placeholder="Choose a guardian" options={guardianOptions} disabled={guardiansQuery.isLoading} invalid={Boolean(guardianSelectError)} describedBy={guardianSelectError ? "student-guardian-select-error" : undefined}/>
              {guardianSelectError && <small id="student-guardian-select-error" className="app-field-error">{guardianSelectError}</small>}
              {guardiansQuery.isError && <small className="app-field-error" role="alert">{guardiansQuery.error.message}</small>}
              {!guardiansQuery.isLoading && !guardiansQuery.isError && guardianOptions.length === 0 && <p className="student-address-note">No unlinked active guardian contacts match this search. You can create a new contact below.</p>}
              <label>Relationship to student<input value={relationshipType} maxLength={40} aria-invalid={Boolean(relationshipError)} aria-describedby={relationshipError ? "student-guardian-relationship-error" : undefined} onChange={event => { setRelationshipType(event.target.value); setRelationshipError(""); }} placeholder="Father, Mother, Guardian…"/></label>
              {relationshipError && <small id="student-guardian-relationship-error" className="app-field-error">{relationshipError}</small>}
              <div className="student-family-checks">
                <label><input type="checkbox" checked={primaryContact} onChange={event => setPrimaryContact(event.target.checked)}/> Primary contact</label>
                <label><input type="checkbox" checked={emergencyContact} onChange={event => setEmergencyContact(event.target.checked)}/> Emergency contact</label>
              </div>
              
              <button type="submit" className="primary" disabled={busy}>{linkGuardian.isPending ? "Linking…" : "Link guardian"}</button>
              <button type="button" className="student-link-button" onClick={() => { setShowCreateGuardian(value => !value); setNewGuardianNameError(""); setNewGuardianEmailError(""); }}>Create a new guardian contact</button>
            </form>
            {showCreateGuardian && (
              <form className="student-family-create-form" onSubmit={saveGuardian} noValidate>
                <h4>New guardian contact</h4>
                <label>Full name<input name="guardian_full_name" disabled={busy} value={newGuardianName} maxLength={200} aria-invalid={Boolean(newGuardianNameError)} aria-describedby={newGuardianNameError ? "student-guardian-name-error" : undefined} onChange={event => { setNewGuardianName(event.target.value); setNewGuardianNameError(""); }} placeholder="Guardian full name"/>{newGuardianNameError && <small id="student-guardian-name-error" className="app-field-error">{newGuardianNameError}</small>}</label>
                <label>Mobile number<input disabled={busy} value={newGuardianMobile} maxLength={30} onChange={event => setNewGuardianMobile(event.target.value)} placeholder="Mobile number"/></label>
                <label>Email address<input name="guardian_email_address" disabled={busy} type="email" value={newGuardianEmail} maxLength={254} aria-invalid={Boolean(newGuardianEmailError)} aria-describedby={newGuardianEmailError ? "student-guardian-email-error" : undefined} onChange={event => { setNewGuardianEmail(event.target.value); setNewGuardianEmailError(""); }} placeholder="guardian@example.com"/>{newGuardianEmailError && <small id="student-guardian-email-error" className="app-field-error">{newGuardianEmailError}</small>}</label>
                <label>Alternate mobile number<input disabled={busy} value={newGuardianAlternateMobile} maxLength={30} onChange={event => setNewGuardianAlternateMobile(event.target.value)} placeholder="Optional alternate number"/></label>
                <label>Occupation<input disabled={busy} value={newGuardianOccupation} maxLength={120} onChange={event => setNewGuardianOccupation(event.target.value)} placeholder="e.g. Teacher, Business owner"/></label>
                <label>Education<input disabled={busy} value={newGuardianEducation} maxLength={120} onChange={event => setNewGuardianEducation(event.target.value)} placeholder="Highest education (optional)"/></label>
                <label>Workplace / organization<input disabled={busy} value={newGuardianWorkplace} maxLength={160} onChange={event => setNewGuardianWorkplace(event.target.value)} placeholder="Organization name (optional)"/></label>
                
                <button type="submit" className="secondary" disabled={busy}>{createGuardian.isPending ? "Saving…" : "Save guardian contact"}</button>
              </form>
            )}
            {pendingRevokeID > 0 && (
              <div className="student-location-confirm" role="alertdialog" aria-label="Revoke guardian relationship">
                <label>Reason for revocation<input value={revokeReason} maxLength={500} aria-invalid={Boolean(revokeReasonError)} aria-describedby={revokeReasonError ? "student-guardian-revoke-reason-error" : undefined} onChange={event => { setRevokeReason(event.target.value); setRevokeReasonError(""); }} placeholder="Explain why this link is being revoked"/>{revokeReasonError && <small id="student-guardian-revoke-reason-error" className="app-field-error">{revokeReasonError}</small>}</label>
                <div><button type="button" className="app-modal-cancel" onClick={() => { setPendingRevokeID(0); setRevokeReasonError(""); }} disabled={busy}>Cancel</button><button type="button" className="secondary" onClick={() => void confirmRevoke()} disabled={busy}>{revokeLink.isPending ? "Revoking…" : "Confirm revoke"}</button></div>
              </div>
            )}
          </>
        )}
        {!canManage && <p className="student-address-note">View-only access. Guardian management permission is required to create, link or revoke guardian records.</p>}
      </section>

      <section className="student-family-section">
        <div className="student-family-heading"><div><h3>Other students sharing a guardian</h3><p>This is derived from active guardian links. It does not assume a biological sibling relationship.</p></div></div>
        {!canReadRelatedStudents ? <p className="student-address-note">Basic student read permission is required to view related student records.</p> : relatedQuery.isLoading ? <p className="student-address-note">Loading related students…</p> :
          relatedQuery.isError ? <p className="student-feedback error" role="alert">{relatedQuery.error.message}</p> :
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
    </Modal>
  );
}
