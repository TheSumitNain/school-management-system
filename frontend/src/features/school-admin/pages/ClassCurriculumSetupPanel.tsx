import { useMemo, useState, type FormEvent } from "react";
import {
  useSaveSchoolClassCurriculumMutation,
  useSchoolClassCurriculaQuery,
} from "../queries";

type Year = { id: number; name: string; status: string };
type SchoolClass = { id: number; name: string; is_active: boolean };
type Subject = { id: number; name: string; subject_code?: string; is_active: boolean };
type Group = { name: string; min_selections: number; max_selections: number };
type Props = { years: Year[]; classes: SchoolClass[]; subjects: Subject[]; canManage: boolean };

export function ClassCurriculumSetupPanel({ years, classes, subjects, canManage }: Props) {
  const [yearID, setYearID] = useState("");
  const [classID, setClassID] = useState("");
  const [groups, setGroups] = useState<Group[]>([]);
  const [included, setIncluded] = useState<number[]>([]);
  const [groupBySubject, setGroupBySubject] = useState<Record<number, string>>({});
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const save = useSaveSchoolClassCurriculumMutation();
  const yearNumber = Number(yearID);
  const classNumber = Number(classID);
  const curriculaQuery = useSchoolClassCurriculaQuery(yearNumber, classNumber);
  const activeSubjects = subjects.filter(subject => subject.is_active);
  const selectedCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const id of included) {
      const name = (groupBySubject[id] ?? "").trim().toLocaleLowerCase();
      if (name) counts.set(name, (counts.get(name) ?? 0) + 1);
    }
    return counts;
  }, [included, groupBySubject]);

  function addGroup() {
    setGroups(current => [...current, { name: "", min_selections: 1, max_selections: 1 }]);
  }
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");
    if (!yearNumber || !classNumber) { setError("Select an academic year and class."); return; }
    if (included.length === 0) { setError("Select at least one subject for this curriculum."); return; }
    const normalized = new Set<string>();
    for (const group of groups) {
      const key = group.name.trim().toLocaleLowerCase();
      if (!key || normalized.has(key) || group.min_selections < 0 || group.max_selections < 1 || group.min_selections > group.max_selections) {
        setError("Each optional group needs a unique name and valid minimum/maximum selection limits."); return;
      }
      normalized.add(key);
      const count = selectedCounts.get(key) ?? 0;
      if (count === 0 || group.min_selections > count || group.max_selections > count) {
        setError(group.name + " must be assigned to enough subjects for its selection limits."); return;
      }
    }
    const groupNames = new Set(groups.map(group => group.name.trim().toLocaleLowerCase()));
    const payloadSubjects = included.map(subjectID => {
      const optionalGroup = (groupBySubject[subjectID] ?? "").trim();
      return { subject_id: subjectID, ...(optionalGroup ? { optional_group: optionalGroup } : {}) };
    });
    if (payloadSubjects.some(subject => subject.optional_group && !groupNames.has(subject.optional_group.toLocaleLowerCase()))) {
      setError("Every optional subject must belong to a configured optional group."); return;
    }
    save.mutate({
      academic_year_id: yearNumber,
      class_id: classNumber,
      optional_groups: groups.map(group => ({ ...group, name: group.name.trim() })),
      subjects: payloadSubjects,
    }, {
      onSuccess: () => { setSuccess("Curriculum version saved. Existing versions and assigned student choices are preserved."); setIncluded([]); setGroupBySubject({}); setGroups([]); },
      onError: mutationError => setError(mutationError.message),
    });
  }

  return (
    <section className="table-panel curriculum-setup-panel">
      <div className="toolbar"><div><h2>Class curriculum</h2><small>Configure compulsory and optional subjects for a class in an academic year.</small></div></div>
      <div className="curriculum-filters">
        <label>Academic year
          <select value={yearID} onChange={event => { setYearID(event.target.value); setIncluded([]); setGroupBySubject({}); setGroups([]); setError(""); setSuccess(""); }} required>
            <option value="">Select academic year</option>
            {years.filter(year => year.status !== "closed").map(year => <option key={year.id} value={year.id}>{year.name}</option>)}
          </select>
        </label>
        <label>Class
          <select value={classID} onChange={event => { setClassID(event.target.value); setIncluded([]); setGroupBySubject({}); setGroups([]); setError(""); setSuccess(""); }} required>
            <option value="">Select class</option>
            {classes.filter(item => item.is_active).map(item => <option key={item.id} value={item.id}>{item.name}</option>)}
          </select>
        </label>
      </div>
      {yearNumber > 0 && classNumber > 0 && (
        <section className="curriculum-existing">
          <h3>Saved curriculum versions</h3>
          {curriculaQuery.isLoading ? <p>Loading saved versions…</p> :
            curriculaQuery.isError ? <div className="student-feedback error" role="alert"><p>{curriculaQuery.error.message}</p><button type="button" className="academic-refresh" onClick={() => void curriculaQuery.refetch()}>Retry</button></div> :
            (curriculaQuery.data?.curricula ?? []).length === 0 ? <p>No curriculum configured for this class and year.</p> :
            <ul>{curriculaQuery.data?.curricula.map(item => <li key={item.id}>Version {item.version_number} · {item.status} · {item.subjects.length} subjects · {item.optional_groups.length} optional groups</li>)}</ul>}
        </section>
      )}
      {canManage ? (
        <form onSubmit={submit} className="curriculum-editor">
          <div className="toolbar"><div><h3>Create next version</h3><small>Saving creates a new immutable version; it never overwrites past student choices.</small></div>
            <button type="button" className="academic-add" onClick={addGroup}>Add optional group</button>
          </div>
          {groups.map((group, index) => (
            <fieldset className="curriculum-group-editor" key={index}>
              <legend>Optional group {index + 1}</legend>
              <label>Group name<input value={group.name} maxLength={100} placeholder="e.g. Second language" onChange={event => setGroups(current => current.map((item, i) => i === index ? { ...item, name: event.target.value } : item))} /></label>
              <label>Minimum choices<input type="number" min={0} max={150} value={group.min_selections} onChange={event => setGroups(current => current.map((item, i) => i === index ? { ...item, min_selections: Number(event.target.value) } : item))} /></label>
              <label>Maximum choices<input type="number" min={1} max={150} value={group.max_selections} onChange={event => setGroups(current => current.map((item, i) => i === index ? { ...item, max_selections: Number(event.target.value) } : item))} /></label>
              <button type="button" className="academic-edit" onClick={() => { const removedName = group.name.trim().toLocaleLowerCase(); setGroups(current => current.filter((_, i) => i !== index)); setGroupBySubject(current => Object.fromEntries(Object.entries(current).map(([id, name]) => [id, name.trim().toLocaleLowerCase() === removedName ? "" : name]))); }}>Remove group</button>
            </fieldset>
          ))}
          <div className="curriculum-subject-picker">
            <h3>Subjects</h3>
            {activeSubjects.length === 0 ? <p>Add active subjects in Academic Structure first.</p> : activeSubjects.map(subject => {
              const checked = included.includes(subject.id);
              const groupValue = groupBySubject[subject.id] ?? "";
              return (
                <div className="curriculum-subject-row" key={subject.id}>
                  <label><input type="checkbox" checked={checked} onChange={event => setIncluded(current => event.target.checked ? [...current, subject.id] : current.filter(id => id !== subject.id))} />
                    <span>{subject.name}{subject.subject_code ? " (" + subject.subject_code + ")" : ""}</span>
                  </label>
                  {checked && <label className="curriculum-subject-group-select">Type
                    <select value={groupValue} onChange={event => setGroupBySubject(current => ({ ...current, [subject.id]: event.target.value }))}>
                      <option value="">Compulsory</option>
                      {groups.filter(group => group.name.trim()).map((group, index) => <option key={index} value={group.name}>{group.name}</option>)}
                    </select>
                  </label>}
                </div>
              );
            })}
          </div>
          {error && <p className="app-field-error" role="alert">{error}</p>}
          {success && <p className="student-success" role="status">{success}</p>}
          <button type="submit" className="primary academic-add" disabled={save.isPending || !yearNumber || !classNumber}>{save.isPending ? "Saving curriculum…" : "Save curriculum version"}</button>
        </form>
      ) : <p className="table-message">You have read-only access to class curricula.</p>}
    </section>
  );
}
