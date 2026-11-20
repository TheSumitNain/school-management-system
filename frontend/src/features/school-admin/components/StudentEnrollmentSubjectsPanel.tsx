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
  const [targetCurriculumID, setTargetCurriculumID] = useState(0);
  const data = query.data?.enrollment_curriculum;
  const curriculum = [data?.curriculum, data?.available_curriculum].find(item => item?.id === targetCurriculumID) ?? data?.curriculum ?? data?.available_curriculum;
  const movingToCurrentCurriculum = Boolean(data?.available_curriculum && targetCurriculumID === data.available_curriculum.id);
  const selectedCurriculumIsAssigned = Boolean(data?.assigned && data.curriculum?.id === targetCurriculumID);
  // Query refetches can replace the selected-ID array with a new reference
  // even when its values have not changed. Track the saved values rather than
  // the array identity so background refetches never discard unsaved edits.
  const savedSelectionKey = (data?.selected_optional_subject_ids ?? []).join(",");
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
    setTargetCurriculumID(data?.curriculum?.id ?? data?.available_curriculum?.id ?? 0);
    setSelected(savedSelectionKey ? savedSelectionKey.split(",").map(Number) : []);
  }, [enrollmentID, data?.enrollment_id, data?.curriculum?.id, data?.available_curriculum?.id, savedSelectionKey]);

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
  const invalidGroups = grouped.filter(([id, group]) => {
    const count = counts.get(id) ?? 0;
    return count < group.min || count > group.max;
  });

  return (
    <section className="student-enrollment-subjects" aria-label="Curriculum and subject choices">
      <div className="student-profile-enrollment-heading">
        <strong>Curriculum</strong>
        <span className={"student-status " + (selectedCurriculumIsAssigned ? "active" : "pending")}>{selectedCurriculumIsAssigned ? "Assigned" : "Available"}</span>
        <button type="button" className="student-link-button" onClick={() => void query.refetch()} disabled={query.isFetching || save.isPending} aria-label="Refresh curriculum and subject choices">
          {query.isFetching ? "Refreshing…" : "Refresh"}
        </button>
      </div>
      <p className="student-address-note">Compulsory subjects are included automatically. Optional choices follow the school's selection limits.</p>
      {data?.curriculum && data.available_curriculum && (
        <label className="student-curriculum-version-select">Curriculum assignment
          <select value={targetCurriculumID} onChange={event => { const nextID = Number(event.target.value); setTargetCurriculumID(nextID); setSelected(nextID === data.curriculum?.id ? (data.selected_optional_subject_ids ?? []) : []); }}>
            <option value={data.curriculum.id}>Currently assigned curriculum</option>
            <option value={data.available_curriculum.id}>Current section curriculum</option>
          </select>
        </label>
      )}
      {!data?.assigned && <p className="student-address-note">This curriculum is available for the selected class and academic year. Saving subject choices will assign it to this student’s enrollment.</p>}
      {movingToCurrentCurriculum && <p className="student-address-note">Saving switches this enrollment to the current section curriculum. Its current optional subject choices will be replaced; previous subject choices are not retained.</p>}
      <section className="student-subject-section" aria-label="Compulsory subjects">
        <h4>Compulsory subjects <span>Included automatically</span></h4>
        <div className="student-subject-list">
          {curriculum.subjects.filter(subject => subject.is_compulsory).map(subject => (
            <div className="student-subject-row" key={subject.id}>
              <span>{subject.name}{subject.code ? " (" + subject.code + ")" : ""}</span><span className="student-subject-type-badge compulsory">Compulsory</span>
            </div>
          ))}
        </div>
      </section>
      {curriculum.subjects.some(subject => subject.is_optional && !subject.optional_group_id) && (
        <section className="student-subject-section" aria-label="Individual optional subjects">
          <h4>Optional subjects <span>Choose individually</span></h4>
          <div className="student-subject-list">
            {curriculum.subjects.filter(subject => subject.is_optional && !subject.optional_group_id).map(subject => (
              <label className="student-subject-option student-subject-standalone" key={subject.id}>
                <input type="checkbox" disabled={!canEdit || save.isPending} checked={selected.includes(subject.id)} onChange={event => toggle(subject.id, event.target.checked)} />
                <span>{subject.name}{subject.code ? " (" + subject.code + ")" : ""}</span>
                <span className="student-subject-type-badge optional">Optional</span>
              </label>
            ))}
          </div>
        </section>
      )}
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
          {invalidGroups.some(([invalidID]) => invalidID === groupID) && <p className="student-field-error" role="alert">{group.min === group.max
            ? `Select exactly ${group.min} ${group.min === 1 ? "subject" : "subjects"} in ${group.name}.`
            : `Select between ${group.min} and ${group.max} subjects in ${group.name}.`}</p>}
        </fieldset>
      ))}
      {canEdit ? (
        <div className="student-subject-actions">
          {save.isError && <p className="student-field-error" role="alert">{save.error.message}</p>}
          <button type="button" className="student-secondary" disabled={invalidGroups.length > 0 || save.isPending} onClick={() => save.mutate({ enrollmentID, curriculum_id: targetCurriculumID || curriculum.id, selected_optional_subject_ids: selected })}>
            {save.isPending ? "Saving…" : "Save subject choices"}
          </button>
        </div>
      ) : <p className="student-address-note">Subject choices are view-only for inactive students or enrollments, or when you do not have update permission.</p>}
    </section>
  );
}
