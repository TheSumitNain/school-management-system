import { useEffect, useState, type FormEvent } from "react";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { StudentFieldLabel } from "./StudentFieldLabel";
import { Select } from "../../../shared/components/Select";
import type { StudentProfile } from "../api";
import { useSaveSchoolStudentBasicDetailsMutation, useSchoolStudentBasicDetailsQuery, useUpdateSchoolStudentMutation } from "../queries";

type Props = { student: StudentProfile; canUpdate: boolean; onStudentNameChange: (fullName: string) => void; onClose: () => void };
type Values = {
  date_of_birth: string; place_of_birth: string; gender: string; nationality: string;
  religion: string; social_category: string; ews_status: string; school_registration_number: string;
};
function localDateValue() {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
}
function earliestDateOfBirth() {
  const now = new Date();
  const year = now.getFullYear() - 130;
  const month = now.getMonth();
  const day = Math.min(now.getDate(), new Date(year, month + 1, 0).getDate());
  return [year, String(month + 1).padStart(2, "0"), String(day).padStart(2, "0")].join("-");
}
const emptyValues: Values = {
  date_of_birth: "", place_of_birth: "", gender: "", nationality: "",
  religion: "", social_category: "", ews_status: "", school_registration_number: "",
};

export function StudentBasicDetailsSection({ student, canUpdate, onStudentNameChange, onClose }: Props) {
  const query = useSchoolStudentBasicDetailsQuery(student.id);
  const save = useSaveSchoolStudentBasicDetailsMutation();
  const updateStudent = useUpdateSchoolStudentMutation();
  const [fullName, setFullName] = useState(student.full_name);
  const [nameError, setNameError] = useState("");
  const [nameFeedback, setNameFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const [values, setValues] = useState<Values>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof Values, string>>>({});
  const [feedback, setFeedback] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const busy = save.isPending || updateStudent.isPending;

  useEffect(() => {
    setFullName(student.full_name);
    setNameError("");
  }, [student.id, student.full_name]);

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

  async function saveStudentName() {
    const trimmed = fullName.trim();
    if (!trimmed) {
      setNameError("Student name is required.");
      return;
    }
    if (trimmed.length > 200) {
      setNameError("Student name must be 200 characters or fewer.");
      return;
    }
    if (trimmed === student.full_name) {
      setNameError("");
      setNameFeedback(null);
      return;
    }
    setNameError("");
    setNameFeedback(null);
    try {
      await updateStudent.mutateAsync({ studentID: student.id, fullName: trimmed });
      setFullName(trimmed);
      onStudentNameChange(trimmed);
      setNameFeedback({ kind: "success", message: "Student name saved." });
    } catch (error) {
      setNameFeedback({ kind: "error", message: error instanceof Error ? error.message : "Student name could not be saved." });
    }
  }

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
      {query.isError && <div className="student-feedback error" role="alert"><p>{query.error.message}</p><button type="button" className="student-secondary" onClick={() => void query.refetch()} disabled={query.isFetching}>Retry loading details</button></div>}
      {nameFeedback && <p className={nameFeedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={nameFeedback.kind === "error" ? "alert" : "status"}>{nameFeedback.message}</p>}
      {feedback && <p className={feedback.kind === "error" ? "student-feedback error" : "student-feedback success"} role={feedback.kind === "error" ? "alert" : "status"}>{feedback.message}</p>}
      <form className="student-basic-details-form" onSubmit={submit} noValidate>
        <label><StudentFieldLabel label="Student name" required hint="Enter the student’s full name as it should appear in school records. Maximum 200 characters." />
          <input value={fullName} maxLength={200} disabled={!canUpdate || busy} aria-invalid={Boolean(nameError)} aria-describedby={nameError ? "student-basic-name-error" : undefined}
            onChange={event => { setFullName(event.target.value); setNameError(""); setNameFeedback(null); }} placeholder="Enter student's full name" />
          {nameError && <small id="student-basic-name-error" className="app-field-error" role="alert">{nameError}</small>}
          {canUpdate && <button type="button" className="student-link-button" onClick={() => void saveStudentName()} disabled={busy || fullName.trim() === student.full_name}>Save student name</button>}
        </label>
        <label><StudentFieldLabel label="Date of birth" hint="Optional. Choose a valid date from the past 130 years; future dates are not allowed." /><DatePicker ariaLabel="Date of birth" value={values.date_of_birth} min={earliestDateOfBirth()} max={localDateValue()} onChange={value => update("date_of_birth", value)} placeholder="Choose date of birth" disabled={!canUpdate || query.isLoading || query.isError || busy}/></label>
        <label><StudentFieldLabel label="Place of birth" hint="Optional. Enter a city, district or place name. Maximum 120 characters." /><input value={values.place_of_birth} maxLength={120} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("place_of_birth", event.target.value)} placeholder="City, district or place"/>{fieldError("place_of_birth")}</label>
        <label><StudentFieldLabel label="Gender" hint="Optional. Choose the value provided by the family; “Prefer not to say” is available." /><Select ariaLabel="Gender" value={values.gender} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("gender", value)} placeholder="Not specified" options={[{value:"female",label:"Female"},{value:"male",label:"Male"},{value:"other",label:"Other"},{value:"not_specified",label:"Prefer not to say"}]}/></label>
        <label><StudentFieldLabel label="Nationality" hint="Optional. Enter a nationality such as Indian. Maximum 80 characters." /><input value={values.nationality} maxLength={80} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("nationality", event.target.value)} placeholder="e.g. Indian"/>{fieldError("nationality")}</label>
        <label><StudentFieldLabel label="Religion" hint="Optional and sensitive. Record only when the school has a legitimate need and the family has provided the information." /><input value={values.religion} maxLength={80} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("religion", event.target.value)} placeholder="Optional"/>{fieldError("religion")}</label>
        <label><StudentFieldLabel label="Social category" hint="Optional and sensitive. Select only when the information is relevant and confirmed by the family." /><Select ariaLabel="Social category" value={values.social_category} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("social_category", value)} placeholder="Not specified" options={[{value:"general",label:"General"},{value:"obc",label:"OBC"},{value:"sc",label:"SC"},{value:"st",label:"ST"},{value:"other",label:"Other"},{value:"not_specified",label:"Not specified"}]}/></label>
        <label><StudentFieldLabel label="EWS status" hint="Optional. Record Yes or No only when the school has verified the applicable status." /><Select ariaLabel="EWS status" value={values.ews_status} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("ews_status", value)} placeholder="Not recorded" options={[{value:"yes",label:"Yes"},{value:"no",label:"No"}]}/></label>
        <label><StudentFieldLabel label="School registration number" hint="Optional school SR/registration identifier. Maximum 50 characters; it must be unique within this school." /><input value={values.school_registration_number} maxLength={50} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("school_registration_number", event.target.value)} placeholder="Optional school SR number"/>{fieldError("school_registration_number")}</label>
        {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit these details.</p>}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
          {canUpdate && <button type="submit" className="primary" disabled={busy || query.isLoading || query.isError}>{save.isPending ? "Saving…" : "Save basic details"}</button>}
        </div>
      </form>
    </Modal>
  );
}
