import { useEffect, useMemo, useState } from "react";
import type { CurriculumSubject } from "../api";
import {
  useSaveSchoolStudentEnrollmentSubjectsMutation,
  useSchoolStudentEnrollmentCurriculumQuery,
} from "../queries";

type Props = {
  enrollmentID: number;
  studentStatus: string;
  enrollmentStatus: string;
  canManage: boolean;
};

export function StudentEnrollmentSubjectsPanel({
  enrollmentID, studentStatus, enrollmentStatus, canManage,
}: Props) {
  const query = useSchoolStudentEnrollmentCurriculumQuery(enrollmentID);
  const save = useSaveSchoolStudentEnrollmentSubjectsMutation();
  const [selected, setSelected] = useState<number[]>([]);
  const data = query.data?.enrollment_curriculum;
  const curriculum = data?.curriculum;
  const canEdit = canManage && studentStatus === "active" && enrollmentStatus === "active";
  const grouped = useMemo(() => {
    const groups = new Map<number, { name: string; min: number; max: number; subjects: CurriculumSubject[] }>();
    for (const group of curriculum?.optional_groups ?? []) {
      groups.set(group.id, { name: group.name, min: group.min_selections, max: group.max_selections, subjects: [] });
    }
    for (const subject of curriculum?.subjects ?? []) {
      if (subject.optional_group_id) groups.get(subject.optional_group_id)?.subjects.push(subject);
    }
    return [...groups.entries()];
  }, [curriculum]);

  useEffect(() => {
    setSelected(data?.selected_optional_subject_ids ?? []);
  }, [data?.enrollment_id, data?.curriculum?.id, data?.selected_optional_subject_ids]);

  function toggle(id: number, checked: boolean) {
    setSelected(current => checked ? [...current, id] : current.filter(value => value !== id));
  }

  if (query.isLoading) return <p className="student-address-note">Loading curriculum and subject choices…</p>;
  if (query.isError) return <div className="student-feedback error" role="alert"><p>{query.error.message}</p><button type="button" className="student-secondary" onClick={() => void query.refetch()}>Retry</button></div>;
  if (!curriculum) return <p className="student-address-note">No curriculum has been configured for this class and academic year yet.</p>;

  const counts = new Map<number, number>();
  for (const id of selected) {
    const subject = curriculum.subjects.find(item => item.id === id);
    if (subject?.optional_group_id) counts.set(subject.optional_group_id, (counts.get(subject.optional_group_id) ?? 0) + 1);
  }
  const invalidGroup = grouped.find(([id, group]) => {
    const count = counts.get(id) ?? 0;
    return count < group.min || count > group.max;
  });

  return (
    <section className="student-enrollment-subjects" aria-label="Curriculum and subject choices">
      <div className="student-profile-enrollment-heading">
        <strong>Curriculum · Version {curriculum.version_number}</strong>
        <span className={"student-status " + (data?.assigned ? "active" : "pending")}>{data?.assigned ? "Assigned" : "Available"}</span>
      </div>
      <p className="student-address-note">Compulsory subjects are included automatically. Optional choices follow the school's selection limits.</p>
      <div className="student-subject-list">
        {curriculum.subjects.filter(subject => subject.is_compulsory).map(subject => (
          <div className="student-subject-row" key={subject.id}>
            <span>{subject.name}</span><span className="student-address-note">Compulsory</span>
          </div>
        ))}
      </div>
      {grouped.map(([groupID, group]) => (
        <fieldset className="student-subject-group" key={groupID} disabled={!canEdit || save.isPending}>
          <legend>{group.name} <span>Choose {group.min === group.max ? group.min : group.min + "–" + group.max}</span></legend>
          {group.subjects.map(subject => (
            <label className="student-subject-option" key={subject.id}>
              <input type="checkbox" checked={selected.includes(subject.id)} onChange={event => toggle(subject.id, event.target.checked)} />
              <span>{subject.name}{subject.code ? " (" + subject.code + ")" : ""}</span>
            </label>
          ))}
          <small>{counts.get(groupID) ?? 0} selected · minimum {group.min}, maximum {group.max}</small>
        </fieldset>
      ))}
      {data?.selection_history && data.selection_history.length > 0 && (
        <details className="student-subject-history">
          <summary>Previous subject choices ({data.selection_history.length})</summary>
          <ul>{data.selection_history.map((item, index) => (
            <li key={item.curriculum_version + "-" + item.subject_name + "-" + index}>
              {item.subject_name} · curriculum v{item.curriculum_version} · {item.revoked_at ? "Replaced" : "Current"}
            </li>
          ))}</ul>
        </details>
      )}
      {canEdit ? (
        <div className="student-subject-actions">
          {invalidGroup && <p className="student-field-error" role="alert">{invalidGroup[1].name}: select between {invalidGroup[1].min} and {invalidGroup[1].max} subjects.</p>}
          {save.isError && <p className="student-field-error" role="alert">{save.error.message}</p>}
          <button type="button" className="student-primary" disabled={Boolean(invalidGroup) || save.isPending} onClick={() => save.mutate({ enrollmentID, curriculum_id: curriculum.id, selected_optional_subject_ids: selected })}>
            {save.isPending ? "Saving…" : "Save subject choices"}
          </button>
        </div>
      ) : <p className="student-address-note">Subject choices are view-only for inactive students or enrollments, or when you do not have update permission.</p>}
    </section>
  );
}
