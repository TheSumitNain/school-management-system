import { useEffect, useState, type FormEvent } from "react";
import { ApiError } from "../../../lib/api";
import { DatePicker } from "../../../shared/components/DatePicker";
import { schoolDateValue, schoolDateValueYearsAgo } from "../../../shared/date/schoolDate";
import { Modal } from "../../../shared/components/Modal";
import { StudentFieldLabel } from "./StudentFieldLabel";
import { notifyToast } from "../../../shared/notifications";
import { Select } from "../../../shared/components/Select";
import type { StudentProfile } from "../api";
import { useSaveSchoolStudentBasicDetailsMutation, useSchoolStudentBasicDetailsQuery, useUpdateSchoolStudentMutation } from "../queries";

type Props = { student: StudentProfile; canUpdate: boolean; onStudentNameChange: (fullName: string) => void; onClose: () => void };
type Values = {
  date_of_birth: string; place_of_birth: string; gender: string; nationality: string;
  religion: string; social_category: string; ews_status: string; school_registration_number: string;
};
function localDateValue() {
  return schoolDateValue();
}
function earliestDateOfBirth() {
  return schoolDateValueYearsAgo(130);
}
const emptyValues: Values = {
  date_of_birth: "", place_of_birth: "", gender: "", nationality: "",
  religion: "", social_category: "", ews_status: "", school_registration_number: "",
};

export function StudentBasicDetailsSection({ student, canUpdate, onStudentNameChange, onClose }: Props) {
  const query = useSchoolStudentBasicDetailsQuery(student.id);
  const basicDetailsSnapshot = JSON.stringify(query.data?.basic_details ? {
    date_of_birth: query.data.basic_details.date_of_birth,
    place_of_birth: query.data.basic_details.place_of_birth,
    gender: query.data.basic_details.gender,
    nationality: query.data.basic_details.nationality,
    religion: query.data.basic_details.religion,
    social_category: query.data.basic_details.social_category,
    ews_status: query.data.basic_details.ews_status,
    school_registration_number: query.data.basic_details.school_registration_number,
  } : null);
  const save = useSaveSchoolStudentBasicDetailsMutation();
  const updateStudent = useUpdateSchoolStudentMutation();
  const [fullName, setFullName] = useState(student.full_name);
  const [nameError, setNameError] = useState("");
  const [values, setValues] = useState<Values>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof Values, string>>>({});
  const [feedback, setFeedbackState] = useState<{ kind: "success" | "error"; message: string } | null>(null);
  const setFeedback = (next: typeof feedback) => {
    setFeedbackState(next);
    if (next) notifyToast(next.kind === "success" && next.message.startsWith("No changes to save") ? "info" : next.kind, next.message);
  };
  const busy = save.isPending || updateStudent.isPending;

  useEffect(() => {
    setFullName(student.full_name);
    setNameError("");
    setFeedback(null);
  }, [student.id]);

  useEffect(() => {
    const item = query.data?.basic_details;
    if (!item) {
      // A student may have no basic-details row yet. Clear the previous
      // student's draft instead of leaving it visible in the new profile.
      setValues(emptyValues);
      setErrors({});
      return;
    }
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
  }, [student.id, basicDetailsSnapshot]);


  function update<K extends keyof Values>(key: K, value: Values[K]) {
    setValues(current => ({ ...current, [key]: value }));
    setErrors(current => ({ ...current, [key]: undefined }));
    setFeedback(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canUpdate || query.isLoading || query.isError) return;
    const trimmedName = fullName.trim();
    if (!trimmedName) { setNameError("Student name is required."); return; }
    if (Array.from(trimmedName).length > 200) { setNameError("Student name must be 200 characters or fewer."); return; }
    setNameError("");
    const nextErrors: Partial<Record<keyof Values, string>> = {};
    if (Array.from(values.place_of_birth.trim()).length > 120) nextErrors.place_of_birth = "Place of birth must be 120 characters or fewer.";
    if (Array.from(values.nationality.trim()).length > 80) nextErrors.nationality = "Nationality must be 80 characters or fewer.";
    if (Array.from(values.religion.trim()).length > 80) nextErrors.religion = "Religion must be 80 characters or fewer.";
    if (Array.from(values.school_registration_number.trim()).length > 50) nextErrors.school_registration_number = "Registration number must be 50 characters or fewer.";
    setErrors(nextErrors);
    if (Object.keys(nextErrors).length) return;
    const input = {
      date_of_birth: values.date_of_birth || null,
      place_of_birth: values.place_of_birth.trim() || null,
      gender: values.gender || null,
      nationality: values.nationality.trim() || null,
      religion: values.religion.trim() || null,
      social_category: values.social_category || null,
      ews_status: values.ews_status === "" ? null : values.ews_status === "yes",
      school_registration_number: values.school_registration_number.trim() || null,
    };
    const current = query.data?.basic_details;
    const unchanged = (current?.date_of_birth?.slice(0, 10) ?? null) === input.date_of_birth &&
      (current?.place_of_birth ?? null) === input.place_of_birth &&
      (current?.gender ?? null) === input.gender &&
      (current?.nationality ?? null) === input.nationality &&
      (current?.religion ?? null) === input.religion &&
      (current?.social_category ?? null) === input.social_category &&
      (current?.ews_status ?? null) === input.ews_status &&
      (current?.school_registration_number ?? null) === input.school_registration_number;
    const nameChanged = trimmedName !== student.full_name;
    if (nameChanged) {
      try {
        await updateStudent.mutateAsync({ studentID: student.id, fullName: trimmedName });
        setFullName(trimmedName);
        onStudentNameChange(trimmedName);
      } catch (error) {
        notifyToast("error", error instanceof Error ? error.message : "Student name could not be saved.");
        return;
      }
    }
    if (unchanged) {
      if (nameChanged) notifyToast("success", "Student name saved.");
      else setFeedback({ kind: "success", message: "No changes to save." });
      return;
    }
    try {
      await save.mutateAsync({ studentID: student.id, input });
      setFeedback({ kind: "success", message: nameChanged ? "Student name and basic details saved." : "Basic details saved." });
    } catch (error) {
      const message = error instanceof Error ? error.message : "Basic details could not be saved.";
      const serverFieldErrors = error instanceof ApiError ? error.fieldErrors : {};
      const mappedErrors: Partial<Record<keyof Values, string>> = {};
      const supportedFields: (keyof Values)[] = [
        "date_of_birth", "place_of_birth", "gender", "nationality",
        "religion", "social_category", "ews_status", "school_registration_number",
      ];
      for (const key of supportedFields) {
        const fieldMessage = serverFieldErrors[key];
        if (fieldMessage) mappedErrors[key] = fieldMessage;
      }
      if (Object.keys(mappedErrors).length > 0) {
        setErrors(current => ({ ...current, ...mappedErrors }));
        setFeedback(null);
        return;
      }
      setFeedback({ kind: "error", message });
    }
  }

  const fieldError = (key: keyof Values) => errors[key] && <small id={"student-basic-" + key + "-error"} className="app-field-error" role="alert">{errors[key]}</small>;

  return (
    <Modal open title="Basic Details" description={student.full_name + " · Optional personal details can be completed later."} onClose={onClose} closeDisabled={busy}>
      {query.isLoading && <p className="student-address-note">Loading basic details…</p>}
      {query.isError && <div className="student-feedback error" role="alert"><p>{query.error.message}</p><button type="button" className="student-secondary" onClick={() => void query.refetch()} disabled={query.isFetching}>Retry loading details</button></div>}
      <form className="student-basic-details-form" onSubmit={submit} noValidate>
        <label><StudentFieldLabel label="Student name" required hint="Enter the student’s full name as it should appear in school records. Maximum 200 characters." />
          <input value={fullName} disabled={!canUpdate || busy} aria-invalid={Boolean(nameError)} aria-describedby={nameError ? "student-basic-name-error" : undefined}
            onChange={event => { setFullName(event.target.value); setNameError(""); }} placeholder="Enter student's full name" />
          {nameError && <small id="student-basic-name-error" className="app-field-error" role="alert">{nameError}</small>}
        </label>
        <div className="student-basic-date-field"><StudentFieldLabel label="Date of birth" hint="Optional. Choose a valid date from the past 130 years; future dates are not allowed." /><DatePicker ariaLabel="Date of birth" value={values.date_of_birth} min={earliestDateOfBirth()} max={localDateValue()} invalid={Boolean(errors.date_of_birth)} describedBy={errors.date_of_birth ? "student-basic-date_of_birth-error" : undefined} onChange={value => update("date_of_birth", value)} placeholder="Choose date of birth" disabled={!canUpdate || query.isLoading || query.isError || busy}/>{fieldError("date_of_birth")}</div>
        <label><StudentFieldLabel label="Place of birth" hint="Optional. Enter a city, district or place name. Maximum 120 characters." /><input value={values.place_of_birth} aria-invalid={Boolean(errors.place_of_birth)} aria-describedby={errors.place_of_birth ? "student-basic-place_of_birth-error" : undefined} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("place_of_birth", event.target.value)} placeholder="City, district or place"/>{fieldError("place_of_birth")}</label>
        <div className="student-basic-select-field"><StudentFieldLabel label="Gender" hint="Optional. Choose the value provided by the family; “Prefer not to say” is available." /><Select ariaLabel="Gender" value={values.gender} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("gender", value)} placeholder="Not specified" options={[{value:"female",label:"Female"},{value:"male",label:"Male"},{value:"other",label:"Other"},{value:"not_specified",label:"Prefer not to say"}]}/></div>
        <label><StudentFieldLabel label="Nationality" hint="Optional. Enter a nationality such as Indian. Maximum 80 characters." /><input value={values.nationality} aria-invalid={Boolean(errors.nationality)} aria-describedby={errors.nationality ? "student-basic-nationality-error" : undefined} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("nationality", event.target.value)} placeholder="e.g. Indian"/>{fieldError("nationality")}</label>
        <label><StudentFieldLabel label="Religion" hint="Optional and sensitive. Record only when the school has a legitimate need and the family has provided the information." /><input value={values.religion} aria-invalid={Boolean(errors.religion)} aria-describedby={errors.religion ? "student-basic-religion-error" : undefined} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("religion", event.target.value)} placeholder="Optional"/>{fieldError("religion")}</label>
        <div className="student-basic-select-field"><StudentFieldLabel label="Social category" hint="Optional and sensitive. Select only when the information is relevant and confirmed by the family." /><Select ariaLabel="Social category" value={values.social_category} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("social_category", value)} placeholder="Not specified" options={[{value:"general",label:"General"},{value:"obc",label:"OBC"},{value:"sc",label:"SC"},{value:"st",label:"ST"},{value:"other",label:"Other"},{value:"not_specified",label:"Not specified"}]}/></div>
        <div className="student-basic-select-field"><StudentFieldLabel label="EWS status" hint="Optional. Record Yes or No only when the school has verified the applicable status." /><Select ariaLabel="EWS status" value={values.ews_status} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={value => update("ews_status", value)} placeholder="Not recorded" options={[{value:"yes",label:"Yes"},{value:"no",label:"No"}]}/></div>
        <label className="student-basic-registration-field"><StudentFieldLabel label="School registration number" hint="Optional school SR/registration identifier. Maximum 50 characters; it must be unique within this school." /><input value={values.school_registration_number} aria-invalid={Boolean(errors.school_registration_number)} aria-describedby={errors.school_registration_number ? "student-basic-school_registration_number-error" : undefined} disabled={!canUpdate || query.isLoading || query.isError || busy} onChange={event => update("school_registration_number", event.target.value)} placeholder="Optional school SR number"/>{fieldError("school_registration_number")}</label>
        {!canUpdate && <p className="student-address-note">View-only access. Student update permission is required to edit these details.</p>}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Close</button>
          {canUpdate && <button type="submit" className="primary" disabled={busy || query.isLoading || query.isError}>{save.isPending ? "Saving…" : "Save basic details"}</button>}
        </div>
      </form>
    </Modal>
  );
}
