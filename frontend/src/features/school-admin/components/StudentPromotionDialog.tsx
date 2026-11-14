import { useEffect, useState, type FormEvent } from "react";
import { DatePicker } from "../../../shared/components/DatePicker";
import { Modal } from "../../../shared/components/Modal";
import { Select } from "../../../shared/components/Select";
import { StudentFieldLabel } from "./StudentFieldLabel";
import type { StudentAcademicYear, StudentEnrollment } from "../api";
import { useSchoolStudentSectionsQuery, usePromoteSchoolStudentEnrollmentMutation } from "../queries";

type Props = {
  enrollment: StudentEnrollment;
  academicYears: StudentAcademicYear[];
  academicYearsError?: string;
  onRetryAcademicYears?: () => void;
  onClose: () => void;
  onPromoted: () => void;
};

function localDateValue() {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
}

export function StudentPromotionDialog({ enrollment, academicYears, academicYearsError, onRetryAcademicYears, onClose, onPromoted }: Props) {
  const availableYears = academicYears.filter(year => year.status === "active" && year.id !== enrollment.academic_year_id);
  const [academicYearID, setAcademicYearID] = useState(0);
  const [sectionID, setSectionID] = useState("");
  const [admissionNumber, setAdmissionNumber] = useState("");
  const [rollNumber, setRollNumber] = useState("");
  const [admissionDate, setAdmissionDate] = useState(localDateValue());
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [feedback, setFeedback] = useState("");
  const promote = usePromoteSchoolStudentEnrollmentMutation();
  const sectionsQuery = useSchoolStudentSectionsQuery(academicYearID, academicYearID > 0);
  const sections = (sectionsQuery.data?.sections ?? []).filter(section => section.is_active);
  const busy = promote.isPending;
  const destinationUnavailable = Boolean(academicYearsError) || sectionsQuery.isLoading || sectionsQuery.isError || sections.length === 0;

  useEffect(() => {
    if (academicYearID === 0 && availableYears.length > 0) {
      setAcademicYearID(availableYears.find(year => year.is_current)?.id ?? availableYears[0].id);
    }
  }, [academicYearID, availableYears]);

  function clearError(key: string) {
    setErrors(current => ({ ...current, [key]: "" }));
    setFeedback("");
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (academicYearsError || sectionsQuery.isLoading || sectionsQuery.isError) return;
    const next: Record<string, string> = {};
    if (!academicYearID) next.academic_year_id = "Select the destination academic year.";
    if (!sectionID) next.section_id = "Select a destination class and section.";
    if (!admissionNumber.trim()) next.admission_number = "Enter the admission number for the destination year.";
    if (!admissionDate) next.admission_date = "Choose the admission date.";
    setErrors(next);
    setFeedback("");
    if (Object.keys(next).length > 0) return;

    try {
      await promote.mutateAsync({
        sourceEnrollmentID: enrollment.id,
        input: {
          academic_year_id: academicYearID,
          section_id: Number(sectionID),
          roll_number: rollNumber.trim() || null,
          admission_number: admissionNumber.trim(),
          admission_date: admissionDate,
          enrolled_at: admissionDate,
        },
      });
      onPromoted();
    } catch (error) {
      setFeedback(error instanceof Error ? error.message : "Student promotion could not be completed.");
    }
  }

  return (
    <Modal open title="Promote student"
      description={`${enrollment.student_name || "Student"} · ${enrollment.class_name_snapshot} / ${enrollment.section_name_snapshot}`}
      onClose={onClose} closeDisabled={busy}>
      <form className="student-promotion-form" onSubmit={submit} noValidate>
        <p className="student-address-note">The current enrollment will be closed as promoted and a new academic-year enrollment will be created. Previous academic history remains unchanged.</p>
        {feedback && <p className="student-feedback error" role="alert">{feedback}</p>}
        {academicYearsError ? (
          <div className="student-feedback error" role="alert"><p>{academicYearsError}</p><button type="button" className="student-secondary" onClick={onRetryAcademicYears} disabled={!onRetryAcademicYears}>Retry loading academic years</button></div>
        ) : availableYears.length === 0 ? (
          <p className="student-feedback error" role="alert">No other active academic year is available. Create or activate the destination academic year first.</p>
        ) : <>
          <label><StudentFieldLabel label="Destination academic year" required hint="Choose an active academic year different from the student’s current enrollment year." />
            <Select ariaLabel="Destination academic year" value={String(academicYearID || "")} invalid={Boolean(errors.academic_year_id)} describedBy={errors.academic_year_id ? "promotion-year-error" : undefined}
              onChange={value => { setAcademicYearID(Number(value)); setSectionID(""); clearError("academic_year_id"); clearError("section_id"); }}
              options={availableYears.map(year => ({ value: String(year.id), label: year.name + (year.is_current ? " · Current" : "") }))} placeholder="Select academic year" disabled={busy || Boolean(academicYearsError) || availableYears.length === 0} />
            {errors.academic_year_id && <small id="promotion-year-error" className="app-field-error">{errors.academic_year_id}</small>}
          </label>
          <label><StudentFieldLabel label="Destination class / section" required hint="Choose a section that belongs to the selected destination academic year." />
            <Select ariaLabel="Destination class and section" value={sectionID} invalid={Boolean(errors.section_id)} describedBy={errors.section_id ? "promotion-section-error" : undefined}
              onChange={value => { setSectionID(value); clearError("section_id"); }} disabled={!academicYearID || sectionsQuery.isLoading || sectionsQuery.isError || sections.length === 0}
              options={sections.map(section => ({ value: String(section.id), label: section.class_name + " · " + section.name }))} placeholder="Select section" />
            {errors.section_id && <small id="promotion-section-error" className="app-field-error">{errors.section_id}</small>}
            {sectionsQuery.isLoading && <small>Loading destination sections…</small>}
            {sectionsQuery.isError && <div className="student-feedback error" role="alert"><p>{sectionsQuery.error.message}</p><button type="button" className="student-secondary" onClick={() => void sectionsQuery.refetch()} disabled={sectionsQuery.isFetching}>Retry loading destination sections</button></div>}
            {academicYearID > 0 && !sectionsQuery.isLoading && !sectionsQuery.isError && sections.length === 0 && <small>No active sections are available in this academic year.</small>}
          </label>
          <label><StudentFieldLabel label="New admission number" required hint="Enter the student’s admission number for the destination academic year. Maximum 50 characters; unique within the school and year." />
            <input disabled={busy || destinationUnavailable} value={admissionNumber} maxLength={50} aria-invalid={Boolean(errors.admission_number)} aria-describedby={errors.admission_number ? "promotion-admission-error" : undefined}
              onChange={event => { setAdmissionNumber(event.target.value); clearError("admission_number"); }} placeholder="e.g. ADM-2027-001" />
            {errors.admission_number && <small id="promotion-admission-error" className="app-field-error">{errors.admission_number}</small>}
          </label>
          <label><StudentFieldLabel label="Roll number" hint="Optional. Leave blank to automatically assign the next available roll number in the destination section. If entering one manually, use the school’s roll-number format (up to 30 characters)." />
            <input disabled={busy || destinationUnavailable} value={rollNumber} maxLength={30} onChange={event => setRollNumber(event.target.value)} placeholder="Leave blank for automatic assignment" />
            <small>Leave blank to assign the next available roll number in the destination section. Manual override is also supported.</small>
          </label>
          <label><StudentFieldLabel label="Admission / promotion date" required hint="Choose a date within the destination academic year and not earlier than the current enrollment." />
            <DatePicker ariaLabel="Admission and promotion date" value={admissionDate} onChange={value => { setAdmissionDate(value); clearError("admission_date"); }}
              placeholder="Choose a date" disabled={busy || destinationUnavailable} invalid={Boolean(errors.admission_date)} describedBy={errors.admission_date ? "promotion-date-error" : undefined} />
            {errors.admission_date && <small id="promotion-date-error" className="app-field-error">{errors.admission_date}</small>}
            <small>Choose a date within the destination academic year and not earlier than the current enrollment.</small>
          </label>
        </>}
        <div className="app-modal-actions">
          <button type="button" className="app-modal-cancel" onClick={onClose} disabled={busy}>Cancel</button>
          <button type="submit" className="primary" disabled={busy || Boolean(academicYearsError) || availableYears.length === 0 || sectionsQuery.isLoading || sectionsQuery.isError || sections.length === 0}>
            {busy ? "Promoting…" : "Promote student"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
