import { useEffect, useState, type FormEvent } from "react";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import { StudentFieldLabel } from "./StudentFieldLabel";
import type { StudentProfile } from "../api";
import {
  useSaveSchoolStudentMedicalDetailsMutation,
  useSaveSchoolStudentPhysicalDetailsMutation,
  useSchoolStudentMedicalDetailsQuery,
  useSchoolStudentPhysicalDetailsQuery,
} from "../queries";

type Props = {
  student: StudentProfile;
  canUpdate: boolean;
  onClose: () => void;
};

type Section = "medical" | "physical";

function textValue(value?: string | null) {
  return value ?? "";
}

function numberValue(value?: number | null) {
  return value == null ? "" : String(value);
}
function dateLabel(value?: string | null) {
  if (!value) return "—";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime())
    ? value
    : new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(parsed);
}

export function StudentHealthSection({ student, canUpdate, onClose }: Props) {
  const [section, setSection] = useState<Section>("medical");
  const [bloodGroup, setBloodGroup] = useState("");
  const [allergies, setAllergies] = useState("");
  const [knownConditions, setKnownConditions] = useState("");
  const [ongoingMedications, setOngoingMedications] = useState("");
  const [emergencyNotes, setEmergencyNotes] = useState("");
  const [heightCM, setHeightCM] = useState("");
  const [weightKG, setWeightKG] = useState("");
  const [measuredOn, setMeasuredOn] = useState("");
  const [visionNotes, setVisionNotes] = useState("");
  const [physicalNotes, setPhysicalNotes] = useState("");
  const [physicalErrors, setPhysicalErrors] = useState<{ heightCM?: string; weightKG?: string; measuredOn?: string }>({});
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);

  const medicalQuery = useSchoolStudentMedicalDetailsQuery(student.id, section === "medical");
  const physicalQuery = useSchoolStudentPhysicalDetailsQuery(student.id, section === "physical");
  const saveMedical = useSaveSchoolStudentMedicalDetailsMutation();
  const savePhysical = useSaveSchoolStudentPhysicalDetailsMutation();
  const busy = saveMedical.isPending || savePhysical.isPending;

  useEffect(() => {
    if (section !== "medical") return;
    const details = medicalQuery.data?.medical_details;
    setBloodGroup(textValue(details?.blood_group));
    setAllergies(textValue(details?.allergies));
    setKnownConditions(textValue(details?.known_conditions));
    setOngoingMedications(textValue(details?.ongoing_medications));
    setEmergencyNotes(textValue(details?.emergency_medical_notes));
  }, [section, medicalQuery.data?.medical_details]);

  useEffect(() => {
    if (section !== "physical") return;
    const details = physicalQuery.data?.physical_details;
    setHeightCM(numberValue(details?.height_cm));
    setWeightKG(numberValue(details?.weight_kg));
    setMeasuredOn(textValue(details?.measured_on).slice(0, 10));
    setVisionNotes(textValue(details?.vision_notes));
    setPhysicalNotes(textValue(details?.physical_notes));
    setPhysicalErrors({});
  }, [section, physicalQuery.data?.physical_details]);

  async function submitMedical(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canUpdate || medicalQuery.isLoading || medicalQuery.isError) return;
    const current = medicalQuery.data?.medical_details;
    if (current && (current.blood_group ?? "") === bloodGroup &&
      (current.allergies ?? "") === allergies.trim() &&
      (current.known_conditions ?? "") === knownConditions.trim() &&
      (current.ongoing_medications ?? "") === ongoingMedications.trim() &&
      (current.emergency_medical_notes ?? "") === emergencyNotes.trim()) {
      setFeedback({ kind: "success", message: "No changes to save." });
      return;
    }
    try {
      await saveMedical.mutateAsync({
        studentID: student.id,
        input: {
          blood_group: bloodGroup || null,
          allergies: allergies.trim() || null,
          known_conditions: knownConditions.trim() || null,
          ongoing_medications: ongoingMedications.trim() || null,
          emergency_medical_notes: emergencyNotes.trim() || null,
        },
      });
      setFeedback({ kind: "success", message: "Medical details saved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Medical details could not be saved." });
    }
  }

  async function submitPhysical(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canUpdate || physicalQuery.isLoading || physicalQuery.isError) return;
    const errors: { heightCM?: string; weightKG?: string; measuredOn?: string } = {};
    const height = heightCM.trim() ? Number(heightCM) : null;
    const weight = weightKG.trim() ? Number(weightKG) : null;
    if (height !== null && (!Number.isFinite(height) || height < 20 || height > 250)) {
      errors.heightCM = "Height must be between 20 and 250 cm.";
    } else if (height !== null && Math.abs(height * 100 - Math.round(height * 100)) > 1e-8) {
      errors.heightCM = "Use no more than 2 decimal places.";
    }
    if (weight !== null && (!Number.isFinite(weight) || weight < 1 || weight > 300)) {
      errors.weightKG = "Weight must be between 1 and 300 kg.";
    } else if (weight !== null && Math.abs(weight * 100 - Math.round(weight * 100)) > 1e-8) {
      errors.weightKG = "Use no more than 2 decimal places.";
    }
    const hasPhysicalDetails = height !== null || weight !== null || Boolean(visionNotes.trim() || physicalNotes.trim());
    if (!hasPhysicalDetails) errors.measuredOn = "Enter at least one physical detail to add a measurement record.";
    else if (!measuredOn) errors.measuredOn = "Measurement date is required.";
    setPhysicalErrors(errors);
    if (Object.keys(errors).length > 0) return;
    const current = physicalQuery.data?.physical_details;
    if (current && current.id > 0 &&
      (current.height_cm ?? null) === height &&
      (current.weight_kg ?? null) === weight &&
      (current.measured_on ?? "").slice(0, 10) === measuredOn &&
      (current.vision_notes ?? "") === visionNotes.trim() &&
      (current.physical_notes ?? "") === physicalNotes.trim()) {
      setFeedback({ kind: "success", message: "No changes to save. A new measurement is recorded only when details or date change." });
      return;
    }
    try {
      await savePhysical.mutateAsync({
        studentID: student.id,
        input: {
          height_cm: height,
          weight_kg: weight,
          measured_on: measuredOn,
          vision_notes: visionNotes.trim() || null,
          physical_notes: physicalNotes.trim() || null,
        },
      });
      setFeedback({ kind: "success", message: "Physical details saved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Physical details could not be saved." });
    }
  }

  return (
    <Modal
      open
      title="Medical & Physical Details"
      description={student.full_name + " · Restricted student information; access is audited."}
      onClose={onClose}
      closeDisabled={busy}
    >
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <p className="student-health-privacy-note">Optional school-entered information only. This section does not create medical diagnoses or automated health assessments. Teachers do not have access to these records.</p>
      <div className="student-address-tabs" role="group" aria-label="Health profile section">
        <button type="button" className={section === "medical" ? "active" : ""} aria-pressed={section === "medical"} onClick={() => { setSection("medical"); setFeedback(null); }}>Medical details</button>
        <button type="button" className={section === "physical" ? "active" : ""} aria-pressed={section === "physical"} onClick={() => { setSection("physical"); setFeedback(null); setPhysicalErrors({}); }}>Physical details</button>
      </div>
      {section === "medical" ? (
        <form className="student-address-form" onSubmit={submitMedical} noValidate>
          {medicalQuery.isLoading ? <p className="student-address-note">Loading restricted medical details…</p> :
            medicalQuery.isError ? <div className="student-feedback error" role="alert"><p>{medicalQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void medicalQuery.refetch()} disabled={medicalQuery.isFetching}>Retry loading medical details</button></div> :
            <>
              <div className="student-health-grid">
                <label><StudentFieldLabel label="Blood group" hint="Optional. Choose the blood group only when it has been confirmed by the family or a reliable record." /><Select ariaLabel="Blood group" value={bloodGroup} disabled={!canUpdate || busy} onChange={setBloodGroup} placeholder="Select blood group" options={["A+","A-","B+","B-","AB+","AB-","O+","O-"].map(value => ({value,label:value}))}/></label>
                <label><StudentFieldLabel label="Allergies" hint="Optional. Record only information provided by the family; maximum 2,000 characters." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={allergies} onChange={event => setAllergies(event.target.value)} placeholder="Record known allergies, if any"/></label>
                <label><StudentFieldLabel label="Known medical conditions" hint="Optional. Record school-relevant information provided by the family; do not enter an unverified diagnosis. Maximum 2,000 characters." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={knownConditions} onChange={event => setKnownConditions(event.target.value)} placeholder="Record school-relevant conditions, if any"/></label>
                <label><StudentFieldLabel label="Ongoing medications" hint="Optional. Record medication information provided by the family. Maximum 2,000 characters." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={ongoingMedications} onChange={event => setOngoingMedications(event.target.value)} placeholder="Record medication information provided by the family"/></label>
                <label className="wide"><StudentFieldLabel label="Emergency medical notes" hint="Optional. Keep notes concise and limited to information needed for an emergency. Maximum 2,000 characters." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={emergencyNotes} onChange={event => setEmergencyNotes(event.target.value)} placeholder="Only essential emergency information"/></label>
              </div>
              {medicalQuery.data?.medical_details.history?.length ? <details className="student-health-history">
                <summary>Previous medical versions ({medicalQuery.data.medical_details.history.length})</summary>
                <div className="student-health-history-list">
                  {medicalQuery.data.medical_details.history.map(version => (
                    <article key={version.id}>
                      <strong>Version {version.version_number} · {dateLabel(version.changed_at)}</strong>
                      <dl>
                        <div><dt>Blood group</dt><dd>{version.blood_group || "—"}</dd></div>
                        <div><dt>Allergies</dt><dd>{version.allergies || "—"}</dd></div>
                        <div><dt>Known conditions</dt><dd>{version.known_conditions || "—"}</dd></div>
                        <div><dt>Ongoing medications</dt><dd>{version.ongoing_medications || "—"}</dd></div>
                        <div><dt>Emergency notes</dt><dd>{version.emergency_medical_notes || "—"}</dd></div>
                      </dl>
                    </article>
                  ))}
                </div>
              </details> : null}
              {!canUpdate && <p className="student-address-note">View-only access. An active student record and student update permission are required to edit these details.</p>}
              <div className="app-modal-actions">
                <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
                {canUpdate && <button className="primary" disabled={busy || medicalQuery.isLoading || medicalQuery.isError}>{saveMedical.isPending ? "Saving…" : "Save medical details"}</button>}
              </div>
            </>
          }
        </form>
      ) : (
        <form className="student-address-form" onSubmit={submitPhysical} noValidate>
          {physicalQuery.isLoading ? <p className="student-address-note">Loading restricted physical details…</p> :
            physicalQuery.isError ? <div className="student-feedback error" role="alert"><p>{physicalQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void physicalQuery.refetch()} disabled={physicalQuery.isFetching}>Retry loading physical details</button></div> :
            <>
              <div className="student-health-grid">
                <label><StudentFieldLabel label="Height (cm)" hint="Optional. Enter a measurement from 20 to 250 cm, using up to 2 decimal places." /><input type="number" min={20} max={250} step="0.01" disabled={!canUpdate || busy} value={heightCM} aria-invalid={Boolean(physicalErrors.heightCM)} aria-describedby={physicalErrors.heightCM ? "student-height-error" : undefined} onChange={event => { setHeightCM(event.target.value); setPhysicalErrors(current => ({...current,heightCM:undefined})); }} placeholder="e.g. 132.5"/>{physicalErrors.heightCM && <small id="student-height-error" className="app-field-error">{physicalErrors.heightCM}</small>}</label>
                <label><StudentFieldLabel label="Weight (kg)" hint="Optional. Enter a measurement from 1 to 300 kg, using up to 2 decimal places." /><input type="number" min={1} max={300} step="0.01" disabled={!canUpdate || busy} value={weightKG} aria-invalid={Boolean(physicalErrors.weightKG)} aria-describedby={physicalErrors.weightKG ? "student-weight-error" : undefined} onChange={event => { setWeightKG(event.target.value); setPhysicalErrors(current => ({...current,weightKG:undefined})); }} placeholder="e.g. 31.25"/>{physicalErrors.weightKG && <small id="student-weight-error" className="app-field-error">{physicalErrors.weightKG}</small>}</label>
                <label><StudentFieldLabel label="Measurement date" conditional hint="Required when recording any height, weight, vision note or physical note. Choose the date the measurement/details were recorded." /><DatePicker value={measuredOn} onChange={value => { setMeasuredOn(value); setPhysicalErrors(current => ({...current,measuredOn:undefined})); }} ariaLabel="Measurement date" disabled={!canUpdate || busy} invalid={Boolean(physicalErrors.measuredOn)} describedBy={physicalErrors.measuredOn ? "student-measured-on-error" : undefined} placeholder="Choose measurement date"/>{physicalErrors.measuredOn && <small id="student-measured-on-error" className="app-field-error">{physicalErrors.measuredOn}</small>}</label>
                <label><StudentFieldLabel label="Vision notes" hint="Optional school-entered note; maximum 1,000 characters. Do not enter a diagnosis." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={1000} value={visionNotes} onChange={event => setVisionNotes(event.target.value)} placeholder="Optional vision-related notes"/></label>
                <label className="wide"><StudentFieldLabel label="Physical notes" hint="Optional school-entered information provided by the family; maximum 2,000 characters." /><textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={physicalNotes} onChange={event => setPhysicalNotes(event.target.value)} placeholder="Optional physical details provided by the family"/></label>
              </div>
              {physicalQuery.data?.physical_details.history && physicalQuery.data.physical_details.history.filter(record => record.id !== physicalQuery.data?.physical_details.id).length > 0 && <details className="student-health-history">
                <summary>Previous physical measurements ({physicalQuery.data.physical_details.history.filter(record => record.id !== physicalQuery.data?.physical_details.id).length})</summary>
                <div className="student-health-history-list">
                  {physicalQuery.data.physical_details.history.filter(record => record.id !== physicalQuery.data?.physical_details.id).map(record => (
                    <article key={record.id}>
                      <strong>Measured {dateLabel(record.measured_on)}</strong>
                      <dl>
                        <div><dt>Height</dt><dd>{record.height_cm == null ? "—" : record.height_cm + " cm"}</dd></div>
                        <div><dt>Weight</dt><dd>{record.weight_kg == null ? "—" : record.weight_kg + " kg"}</dd></div>
                        <div><dt>Vision notes</dt><dd>{record.vision_notes || "—"}</dd></div>
                        <div><dt>Physical notes</dt><dd>{record.physical_notes || "—"}</dd></div>
                      </dl>
                    </article>
                  ))}
                </div>
              </details>}
              {!canUpdate && <p className="student-address-note">View-only access. An active student record and student update permission are required to edit these details.</p>}
              <div className="app-modal-actions">
                <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
                {canUpdate && <button className="primary" disabled={busy || physicalQuery.isLoading || physicalQuery.isError}>{savePhysical.isPending ? "Saving…" : "Save physical details"}</button>}
              </div>
            </>
          }
        </form>
      )}
    </Modal>
  );
}
