import { useEffect, useState, type FormEvent } from "react";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import type { StudentProfile } from "../api";
import { useSaveSchoolStudentBasicDetailsMutation, useSchoolStudentBasicDetailsQuery } from "../queries";

type Props = { student: StudentProfile; canUpdate: boolean; onClose: () => void };
type Values = {
  date_of_birth: string; place_of_birth: string; gender: string; nationality: string;
  religion: string; social_category: string; ews_status: string; school_registration_number: string;
};
function localDateValue() {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
}
const emptyValues: Values = {
  date_of_birth: "", place_of_birth: "", gender: "", nationality: "",
  religion: "", social_category: "", ews_status: "", school_registration_number: "",
};

export function StudentBasicDetailsSection({ student, canUpdate, onClose }: Props) {
  const query = useSchoolStudentBasicDetailsQuery(student.id);
  const save = useSaveSchoolStudentBasicDetailsMutation();
  const [values, setValues] = useState<Values>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof Values, string>>>({});
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const busy = save.isPending;

  useEffect(() => {
    const item = query.data?.basic_details;
    if (!item) return;
    setValues({
      date_of_birth: item.date_of_birth?.slice(0, 10) ?? "",
      place_of_birth: item.place_of_birth ?? "",
      gender: item.gender ?? "",
      nationality: item.nationality ?? "",
      religion: item.religion ?? "",
      social_category: item.social_category ?? "",
      ews_status: item.ews_status == null ? "" : item.ews_status ? "yes" : "no",
      school_registration_number: item.school_registration_number ?? "",
    });
    setErrors({});
  }, [query.data?.basic_details]);

  function update<K extends keyof Values>(key: K, value: Values[K]) {
    setValues(current => ({ ...current, [key]: value }));
    setErrors(current => ({ ...current, [key]: undefined }));
    setFeedback(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canUpdate || query.isLoading || query.isError) return;
    const nextErrors: Partial<Record<keyof Values, string>> = {};
    if (values.place_of_birth.trim().length > 120) nextErrors.place_of_birth = "Place of birth must be 120 characters or fewer.";
    if (values.nationality.trim().length > 80) nextErrors.nationality = "Nationality must be 80 characters or fewer.";
    if (values.religion.trim().length > 80) nextErrors.religion = "Religion must be 80 characters or fewer.";
    if (values.school_registration_number.trim().length > 50) nextErrors.school_registration_number = "Registration number must be 50 characters or fewer.";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;
    try {
      await save.mutateAsync({
        studentID: student.id,
        input: {
          date_of_birth: values.date_of_birth || null,
          place_of_birth: values.place_of_birth.trim() || null,
          gender: values.gender || null,
          nationality: values.nationality.trim() || null,
          religion: values.religion.trim() || null,
          social_category: values.social_category || null,
          ews_status: values.ews_status === "" ? null : values.ews_status === "yes",
          school_registration_number: values.school_registration_number.trim() || null,
        },
      });
      setFeedback({ kind: "success", message: "Basic details saved." });
    } catch (error) {
      setFeedback({ kind: "error", message: error instanceof Error ? error.message : "Basic details could not be saved." });
    }
  }

  const fieldError = (key: keyof Values) => errors[key] && <small className="app-field-error" role="alert">{errors[key]}</small>;

  return (
    <Modal open title="Basic Details" description={student.full_name + " · Optional personal details can be completed later."} onClose={onClose} closeDisabled={busy}>
      {query.isLoading && <p className="student-address-note">Loading basic details…</p>}
      {query.isError && <p className="student-feedback error" role="alert">{query.error.message}</p>}
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <form className="student-basic-details-form" onSubmit={submit} noValidate>
        <label>Date of birth<DatePicker ariaLabel="Date of birth" value={values.date_of_birth} max={localDateValue()} onChange={value => update("date_of_birth", value)} placeholder="Choose date of birth" disabled={!canUpdate}/></label>
        <label>Place of birth<input value={values.place_of_birth} maxLength={120} disabled={!canUpdate} onChange={event => update("place_of_birth", event.target.value)} placeholder="City, district or place"/>{fieldError("place_of_birth")}</label>
        <label>Gender<Select ariaLabel="Gender" value={values.gender} disabled={!canUpdate} onChange={value => update("gender", value)} placeholder="Not specified" options={[{value:"female",label:"Female"},{value:"male",label:"Male"},{value:"other",label:"Other"},{value:"not_specified",label:"Prefer not to say"}]}/></label>
        <label>Nationality<input value={values.nationality} maxLength={80} disabled={!canUpdate} onChange={event => update("nationality", event.target.value)} placeholder="e.g. Indian"/>{fieldError("nationality")}</label>
        <label>Religion<input value={values.religion} maxLength={80} disabled={!canUpdate} onChange={event => update("religion", event.target.value)} placeholder="Optional"/>{fieldError("religion")}</label>
        <label>Social category<Select ariaLabel="Social category" value={values.social_category} disabled={!canUpdate} onChange={value => update("social_category", value)} placeholder="Not specified" options={[{value:"general",label:"General"},{value:"obc",label:"OBC"},{value:"sc",label:"SC"},{value:"st",label:"ST"},{value:"other",label:"Other"},{value:"not_specified",label:"Not specified"}]}/></label>
        <label>EWS status<Select ariaLabel="EWS status" value={values.ews_status} disabled={!canUpdate} onChange={value => update("ews_status", value)} placeholder="Not recorded" options={[{value:"yes",label:"Yes"},{value:"no",label:"No"}]}/></label>
        <label>School registration number<input value={values.school_registration_number} maxLength={50} disabled={!canUpdate} onChange={event => update("school_registration_number", event.target.value)} placeholder="Optional school SR number"/>{fieldError("school_registration_number")}</label>
        {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit these details.</p>}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
          {canUpdate && <button type="submit" className="primary" disabled={busy || query.isLoading || query.isError}>{busy ? "Saving…" : "Save basic details"}</button>}
        </div>
      </form>
    </Modal>
  );
}
