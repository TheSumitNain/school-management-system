import { useEffect, useState, type FormEvent } from "react";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
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
  const [medicalErrors, setMedicalErrors] = useState<Record<string, string>>({});
  const [physicalErrors, setPhysicalErrors] = useState<Record<string, string>>({});
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
    setMedicalErrors({});
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
    if (!canUpdate || medicalQuery.isLoading) return;
    setMedicalErrors({});
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
    if (!canUpdate || physicalQuery.isLoading) return;
    const errors: Record<string, string> = {};
    const height = heightCM.trim() ? Number(heightCM) : null;
    const weight = weightKG.trim() ? Number(weightKG) : null;
    if (height !== null && (!Number.isFinite(height) || height < 20 || height > 250)) {
      errors.heightCM = "Height must be between 20 and 250 cm.";
    }
    if (weight !== null && (!Number.isFinite(weight) || weight < 1 || weight > 300)) {
      errors.weightKG = "Weight must be between 1 and 300 kg.";
    }
    setPhysicalErrors(errors);
    if (Object.keys(errors).length > 0) return;
    try {
      await savePhysical.mutateAsync({
        studentID: student.id,
        input: {
          height_cm: height,
          weight_kg: weight,
          measured_on: measuredOn || null,
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
        <button type="button" className={section === "medical" ? "active" : ""} aria-pressed={section === "medical"} onClick={() => { setSection("medical"); setFeedback(null); setMedicalErrors({}); }}>Medical details</button>
        <button type="button" className={section === "physical" ? "active" : ""} aria-pressed={section === "physical"} onClick={() => { setSection("physical"); setFeedback(null); setPhysicalErrors({}); }}>Physical details</button>
      </div>
      {section === "medical" ? (
        <form className="student-address-form" onSubmit={submitMedical} noValidate>
          {medicalQuery.isLoading ? <p className="student-address-note">Loading restricted medical details…</p> :
            medicalQuery.isError ? <p className="student-feedback error" role="alert">{medicalQuery.error.message}</p> :
            <>
              <div className="student-health-grid">
                <label>Blood group<Select ariaLabel="Blood group" value={bloodGroup} disabled={!canUpdate || busy} onChange={setBloodGroup} placeholder="Select blood group" options={["A+","A-","B+","B-","AB+","AB-","O+","O-"].map(value => ({value,label:value}))}/></label>
                <label>Allergies<textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={allergies} onChange={event => setAllergies(event.target.value)} placeholder="Record known allergies, if any"/></label>
                <label>Known medical conditions<textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={knownConditions} onChange={event => setKnownConditions(event.target.value)} placeholder="Record school-relevant conditions, if any"/></label>
                <label>Ongoing medications<textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={ongoingMedications} onChange={event => setOngoingMedications(event.target.value)} placeholder="Record medication information provided by the family"/></label>
                <label className="wide">Emergency medical notes<textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={emergencyNotes} onChange={event => setEmergencyNotes(event.target.value)} placeholder="Only essential emergency information"/></label>
              </div>
              {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit these details.</p>}
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
            physicalQuery.isError ? <p className="student-feedback error" role="alert">{physicalQuery.error.message}</p> :
            <>
              <div className="student-health-grid">
                <label>Height (cm)<input type="number" min={20} max={250} step="0.01" disabled={!canUpdate || busy} value={heightCM} aria-invalid={Boolean(physicalErrors.heightCM)} aria-describedby={physicalErrors.heightCM ? "student-height-error" : undefined} onChange={event => { setHeightCM(event.target.value); setPhysicalErrors(current => ({...current,heightCM:undefined})); }} placeholder="e.g. 132.5"/>{physicalErrors.heightCM && <small id="student-height-error" className="app-field-error">{physicalErrors.heightCM}</small>}</label>
                <label>Weight (kg)<input type="number" min={1} max={300} step="0.01" disabled={!canUpdate || busy} value={weightKG} aria-invalid={Boolean(physicalErrors.weightKG)} aria-describedby={physicalErrors.weightKG ? "student-weight-error" : undefined} onChange={event => { setWeightKG(event.target.value); setPhysicalErrors(current => ({...current,weightKG:undefined})); }} placeholder="e.g. 31.25"/>{physicalErrors.weightKG && <small id="student-weight-error" className="app-field-error">{physicalErrors.weightKG}</small>}</label>
                <label>Measurement date<DatePicker value={measuredOn} onChange={setMeasuredOn} ariaLabel="Measurement date" disabled={!canUpdate || busy} placeholder="Choose measurement date"/></label>
                <label>Vision notes<textarea disabled={!canUpdate || busy} rows={3} maxLength={1000} value={visionNotes} onChange={event => setVisionNotes(event.target.value)} placeholder="Optional vision-related notes"/></label>
                <label className="wide">Physical notes<textarea disabled={!canUpdate || busy} rows={3} maxLength={2000} value={physicalNotes} onChange={event => setPhysicalNotes(event.target.value)} placeholder="Optional physical details provided by the family"/></label>
              </div>
              {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit these details.</p>}
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
