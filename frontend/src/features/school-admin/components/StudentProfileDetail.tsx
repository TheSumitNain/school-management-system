import { Eye, HeartPulse, MapPin, Pencil, Users } from "lucide-react";
import { Modal } from "../../../shared/components/Modal";
import type { StudentAcademicYear, StudentEnrollment, StudentProfile } from "../api";
import { useSchoolStudentEnrollmentHistoryQuery } from "../queries";

type Props = {
  student: StudentProfile;
  canReadHistory: boolean;
  canUpdateBasic: boolean;
  canReadFamily: boolean;
  academicYears: StudentAcademicYear[];
  onEditBasic: () => void;
  onOpenAddress: () => void;
  onOpenFamily: () => void;
  onOpenHealth: () => void;
  onClose: () => void;
};

function dateLabel(value?: string | null) {
  if (!value) return "—";
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime())
    ? value
    : new Intl.DateTimeFormat("en-IN", { dateStyle: "medium" }).format(parsed);
}

function enrollmentYear(enrollment: StudentEnrollment, years: StudentAcademicYear[]) {
  return years.find(year => year.id === enrollment.academic_year_id)?.name
    ?? (enrollment.academic_year_id ? "Academic year #" + enrollment.academic_year_id : "Academic year unavailable");
}

export function StudentProfileDetail({
  student, canReadHistory, canUpdateBasic, canReadFamily, academicYears,
  onEditBasic, onOpenAddress, onOpenFamily, onOpenHealth, onClose,
}: Props) {
  const historyQuery = useSchoolStudentEnrollmentHistoryQuery(student.id, canReadHistory);
  const history = [...(historyQuery.data?.enrollments ?? [])].sort(
    (a, b) => new Date(b.enrolled_at).getTime() - new Date(a.enrolled_at).getTime(),
  );
  const currentAcademicYear = academicYears.find(year => year.is_current);
  const hasActiveCurrentEnrollment = Boolean(currentAcademicYear && history.some(
    enrollment => enrollment.academic_year_id === currentAcademicYear.id && enrollment.status === "active",
  ));

  return (
    <Modal
      open
      title="Student profile"
      description={student.full_name + " · Permanent student identity and year-wise academic history"}
      onClose={onClose}
    >
      <div className="student-profile-detail">
        <section className="student-profile-summary">
          <div className="student-profile-heading">
            <div>
              <span className="student-profile-eyebrow">Student ID #{student.id}</span>
              <h3>{student.full_name}</h3>
            </div>
            <span className={"student-status " + student.status}>{student.status}</span>
          </div>
          <dl className="student-profile-facts">
            <div><dt>Admission number shown in directory</dt><dd>{student.admission_number || "Not assigned"}</dd></div>
            <div><dt>Student record created</dt><dd>{dateLabel(student.created_at)}</dd></div>
          </dl>
          <div className="student-profile-actions">
            {canUpdateBasic && student.status !== "archived" && <button type="button" className="secondary" onClick={onEditBasic}><Pencil size={14}/> Edit basic details</button>}
            <button type="button" className="secondary" onClick={onOpenAddress}><MapPin size={14}/> Address & communication</button>
            <button type="button" className="secondary" onClick={onOpenHealth}><HeartPulse size={14}/> Medical & physical</button>
            {canReadFamily && <button type="button" className="secondary" onClick={onOpenFamily}><Users size={14}/> Parents & siblings</button>}
          </div>
        </section>

        <section className="student-profile-history">
          <div className="student-profile-section-heading">
            <div><h3>Academic history</h3><p>Each academic year keeps its own admission identity and enrollment record.</p></div>
            <Eye size={17} aria-hidden="true"/>
          </div>
          {!canReadHistory ? <p className="student-address-note">Basic student read permission is required to view academic history.</p> :
            historyQuery.isLoading ? <p className="student-address-note">Loading academic history…</p> :
            historyQuery.isError ? <p className="student-feedback error" role="alert">{historyQuery.error.message}</p> :
            <>
              {currentAcademicYear && <p className={hasActiveCurrentEnrollment ? "student-profile-current-enrollment active" : "student-profile-current-enrollment"}>
                {currentAcademicYear.name}: {hasActiveCurrentEnrollment ? "Active enrollment" : "No active enrollment"}
              </p>}
              {history.length === 0 ? <p className="student-address-note">No academic enrollment history is available for this student.</p> :
              <div className="student-profile-history-list">
              {history.map(enrollment => (
                <article className="student-profile-enrollment" key={enrollment.id}>
                  <div className="student-profile-enrollment-heading">
                    <strong>{enrollmentYear(enrollment, academicYears)}</strong>
                    <span className={"student-status " + enrollment.status}>{enrollment.status}</span>
                  </div>
                  <dl className="student-profile-facts">
                    <div><dt>Admission number</dt><dd>{enrollment.admission_number || "Not recorded"}</dd></div>
                    <div><dt>Admission date</dt><dd>{dateLabel(enrollment.admission_date)}</dd></div>
                    <div><dt>Class / section</dt><dd>{enrollment.class_name_snapshot} · {enrollment.section_name_snapshot}</dd></div>
                    <div><dt>Roll number</dt><dd>{enrollment.roll_number || "—"}</dd></div>
                    <div><dt>Enrolled on</dt><dd>{dateLabel(enrollment.enrolled_at)}</dd></div>
                    <div><dt>Ended on</dt><dd>{dateLabel(enrollment.ended_at)}</dd></div>
                  </dl>
                </article>
              ))}
              </div>}
            </>
          }
        </section>
      </div>
    </Modal>
  );
}
