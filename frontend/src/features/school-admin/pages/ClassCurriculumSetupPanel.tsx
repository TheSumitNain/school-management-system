import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { Select } from "../../../shared/components/Select";
import { notifyToast } from "../../../shared/notifications";
import { Modal } from "../../../shared/components/Modal";
import {
  usePreviewSchoolClassCurriculumRemovalMutation,
  useSaveSchoolClassCurriculumMutation,
  useSchoolClassCurriculaQuery,
} from "../queries";
import type { CurriculumRemovalImpact, CurriculumSaveInput } from "../api";

type Year = { id: number; name: string; status: string };
type SchoolClass = { id: number; name: string; is_active: boolean };
type Section = { id: number; academic_year_id: number; class_id: number; name: string; medium?: "hindi" | "english" | null; is_active: boolean };
type Subject = { id: number; name: string; subject_code?: string; is_active: boolean };
type Group = { name: string; min_selections: number; max_selections: number };
type GroupFieldErrors = { name?: string; min?: string; max?: string; subjects?: string };
type Props = { years: Year[]; classes: SchoolClass[]; sections: Section[]; subjects: Subject[]; canManage: boolean };

export function ClassCurriculumSetupPanel({ years, classes, sections, subjects, canManage }: Props) {
  const [yearID, setYearID] = useState("");
  const [classID, setClassID] = useState("");
  const [sectionID, setSectionID] = useState("");
  const [groups, setGroups] = useState<Group[]>([]);
  const [included, setIncluded] = useState<number[]>([]);
  const [groupBySubject, setGroupBySubject] = useState<Record<number, string>>({});
  const [loadedCurriculumID, setLoadedCurriculumID] = useState<number | null>(null);
  const [pendingSave, setPendingSave] = useState<CurriculumSaveInput | null>(null);
  const [pendingRemovalImpacts, setPendingRemovalImpacts] = useState<CurriculumRemovalImpact[] | null>(null);
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{ year?: string; class?: string; section?: string; subjects?: string; groups: Record<number, GroupFieldErrors> }>({ groups: {} });
  const curriculumSummaryRef = useRef<HTMLElement>(null);
  const groupEditorRefs = useRef<Array<HTMLFieldSetElement | null>>([]);
  const [newGroupIndex, setNewGroupIndex] = useState<number | null>(null);
  const save = useSaveSchoolClassCurriculumMutation();
  const preview = usePreviewSchoolClassCurriculumRemovalMutation();
  const yearNumber = Number(yearID);
  const classNumber = Number(classID);
  const sectionNumber = Number(sectionID);
  const availableSections = sections.filter(section => section.is_active && section.academic_year_id === yearNumber && section.class_id === classNumber);
  const curriculaQuery = useSchoolClassCurriculaQuery(yearNumber, classNumber, sectionNumber);
  const currentCurriculum = (curriculaQuery.data?.curricula ?? []).find(item => item.status === "active") ?? null;
  const activeSubjects = subjects.filter(subject => subject.is_active);
  const busy = save.isPending || preview.isPending;
  const compulsoryCount = included.filter(id => !(groupBySubject[id] ?? "").trim()).length;
  const individualOptionalCount = included.filter(id => groupBySubject[id] === "__standalone_optional__").length;
  const groupedOptionalCount = included.length - compulsoryCount - individualOptionalCount;
  const selectedCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const id of included) {
      const name = (groupBySubject[id] ?? "").trim().toLowerCase();
      if (name && name !== "__standalone_optional__") counts.set(name, (counts.get(name) ?? 0) + 1);
    }
    return counts;
  }, [included, groupBySubject]);

  useEffect(() => {
    if (curriculaQuery.isLoading || curriculaQuery.isError || loadedCurriculumID === (currentCurriculum?.id ?? 0)) return;
    const curriculum = currentCurriculum;
    setIncluded(curriculum?.subjects.map(subject => subject.subject_id) ?? []);
    setGroups(curriculum?.optional_groups.map(group => ({
      name: group.name,
      min_selections: group.min_selections,
      max_selections: group.max_selections,
    })) ?? []);
    setGroupBySubject(Object.fromEntries((curriculum?.subjects ?? []).map(subject => [
      subject.subject_id,
      subject.optional_group_name ?? (subject.is_optional ? "__standalone_optional__" : ""),
    ])));
    setLoadedCurriculumID(curriculum?.id ?? 0);
    setFieldErrors({ groups: {} });
  }, [curriculaQuery.isLoading, curriculaQuery.isError, currentCurriculum, loadedCurriculumID]);

  function clearEditorForScopeChange() {
    setLoadedCurriculumID(null);
    setIncluded([]);
    setGroupBySubject({});
    setGroups([]);
    setPendingSave(null);
    setPendingRemovalImpacts(null);
    setError("");
    setFieldErrors({ groups: {} });
  }

  function addGroup() {
    setGroups(current => {
      const nextIndex = current.length;
      setNewGroupIndex(nextIndex);
      return [...current, { name: "", min_selections: 1, max_selections: 1 }];
    });
    setFieldErrors(current => ({ ...current, groups: {} }));
  }

  useEffect(() => {
    if (newGroupIndex === null) return;
    const frame = window.requestAnimationFrame(() => {
      const card = groupEditorRefs.current[newGroupIndex];
      card?.scrollIntoView({ behavior: "smooth", block: "center" });
      card?.querySelector<HTMLInputElement>('input:not([type="hidden"])')?.focus({ preventScroll: true });
    });
    return () => window.cancelAnimationFrame(frame);
  }, [newGroupIndex, groups.length]);

  function persist(input: CurriculumSaveInput, confirmations: CurriculumRemovalImpact[] = []) {
    save.mutate({
      ...input,
      removal_confirmations: confirmations.map(item => ({
        subject_id: item.subject_id,
        affected_students: item.affected_students,
      })),
    }, {
      onSuccess: () => {
        const message = confirmations.length > 0
          ? "Curriculum updated. Confirmed subject removals and affected student choices have been applied."
          : currentCurriculum
            ? "Curriculum changes saved successfully."
            : "New curriculum created successfully.";
        notifyToast("success", message);
        setPendingSave(null);
        setPendingRemovalImpacts(null);
        window.requestAnimationFrame(() => {
          window.scrollTo({ top: 0, behavior: "smooth" });
          curriculumSummaryRef.current?.focus({ preventScroll: true });
        });
      },
      onError: mutationError => {
        if (mutationError.message.toLowerCase().includes("affected-student count changed")) {
          setPendingSave(null);
          setPendingRemovalImpacts(null);
        }
        setError(mutationError.message);
        notifyToast("error", mutationError.message);
      },
    });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const nextErrors: { year?: string; class?: string; section?: string; subjects?: string; groups: Record<number, GroupFieldErrors> } = { groups: {} };
    if (!yearNumber) nextErrors.year = "Select an academic year.";
    if (!classNumber) nextErrors.class = "Select a class.";
    if (!sectionNumber) nextErrors.section = "Select a section.";
    if (included.length === 0) nextErrors.subjects = "Select at least one subject for this curriculum.";
    const normalized = new Set<string>();
    for (const [index, group] of groups.entries()) {
      const key = group.name.trim().toLowerCase();
      const errors: GroupFieldErrors = {};
      if (!key) errors.name = "Enter a name for this optional group.";
      else if (normalized.has(key)) errors.name = "Optional group names must be unique.";
      else normalized.add(key);
      if (group.min_selections < 0) errors.min = "Minimum choices cannot be negative.";
      if (group.max_selections < 1) errors.max = "Maximum choices must be at least 1.";
      if (group.min_selections > group.max_selections) errors.min = "Minimum choices cannot exceed maximum choices.";
      const count = selectedCounts.get(key) ?? 0;
      if (count === 0) errors.subjects = "Assign at least one subject to this optional group.";
      else {
        if (group.min_selections > count) errors.min = "Minimum choices cannot exceed the number of subjects in this group.";
        if (group.max_selections > count) errors.max = "Maximum choices cannot exceed the number of subjects in this group.";
      }
      if (Object.keys(errors).length > 0) nextErrors.groups[index] = errors;
    }
    const groupNames = new Set(groups.map(group => group.name.trim().toLowerCase()));
    const payloadSubjects = included.map(subjectID => {
      const optionalGroup = (groupBySubject[subjectID] ?? "").trim();
      return {
        subject_id: subjectID,
        is_optional: Boolean(optionalGroup),
        ...(optionalGroup && optionalGroup !== "__standalone_optional__" ? { optional_group: optionalGroup } : {}),
      };
    });
    if (payloadSubjects.some(subject => subject.optional_group && !groupNames.has(subject.optional_group.toLowerCase()))) {
      nextErrors.subjects = "Every optional subject must belong to a configured optional group.";
    }
    setFieldErrors(nextErrors);
    if (nextErrors.year || nextErrors.class || nextErrors.section || nextErrors.subjects || Object.keys(nextErrors.groups).length > 0) {
      window.requestAnimationFrame(() => {
        const firstInvalid = document.querySelector('[aria-invalid="true"], [data-validation-error="true"]') as HTMLElement | null;
        firstInvalid?.scrollIntoView({ behavior: "smooth", block: "center" });
        firstInvalid?.focus();
      });
      return;
    }

    const input: CurriculumSaveInput = {
      academic_year_id: yearNumber,
      class_id: classNumber,
      section_id: sectionNumber,
      optional_groups: groups.map(group => ({ ...group, name: group.name.trim() })),
      subjects: payloadSubjects,
    };

    if (currentCurriculum) {
      const normalizeGroups = (items: Array<{ name: string; min_selections: number; max_selections: number }>) =>
        items.map(item => ({
          name: item.name.trim(),
          min_selections: item.min_selections,
          max_selections: item.max_selections,
        })).sort((a, b) => a.name.localeCompare(b.name));
      const normalizeSubjects = (items: Array<{ subject_id: number; is_optional?: boolean; optional_group?: string }>) =>
        items.map(item => ({
          subject_id: item.subject_id,
          is_optional: Boolean(item.is_optional),
          optional_group: item.optional_group ?? "",
        })).sort((a, b) => a.subject_id - b.subject_id);
      const existingSubjects = currentCurriculum.subjects.map(subject => ({
        subject_id: subject.subject_id,
        is_optional: !subject.is_compulsory,
        optional_group: subject.optional_group_name ?? "",
      }));
      const existingGroups = currentCurriculum.optional_groups.map(group => ({
        name: group.name,
        min_selections: group.min_selections,
        max_selections: group.max_selections,
      }));
      if (
        JSON.stringify(normalizeGroups(input.optional_groups)) === JSON.stringify(normalizeGroups(existingGroups)) &&
        JSON.stringify(normalizeSubjects(input.subjects)) === JSON.stringify(normalizeSubjects(existingSubjects))
      ) {
        notifyToast("info", "No changes to save.");
        return;
      }
    }

    const removedSubjectIDs = (currentCurriculum?.subjects ?? [])
      .filter(subject => !included.includes(subject.subject_id))
      .map(subject => subject.subject_id);
    if (removedSubjectIDs.length === 0) {
      persist(input);
      return;
    }
    setPendingSave(input);
    preview.mutate({
      academic_year_id: yearNumber,
      class_id: classNumber,
      section_id: sectionNumber,
      subject_ids: removedSubjectIDs,
    }, {
      onSuccess: result => setPendingRemovalImpacts(result.impacts),
      onError: mutationError => {
        setPendingSave(null);
        setError(mutationError.message);
        notifyToast("error", mutationError.message);
      },
    });
  }

  return (
    <section className="table-panel curriculum-setup-panel">
      <div className="toolbar"><div><h2>Section-wise curriculum</h2><small>One editable curriculum for each academic year, class and section.</small></div></div>
      <div className="curriculum-filters">
        <label>Academic year
          <Select ariaLabel="Curriculum academic year" value={yearID} disabled={busy} invalid={Boolean(fieldErrors.year)} describedBy={fieldErrors.year ? "curriculum-year-error" : undefined} placeholder="Select academic year" options={years.filter(year => year.status !== "closed").map(year => ({ value: String(year.id), label: year.name }))} onChange={value => { setYearID(value); setClassID(""); setSectionID(""); clearEditorForScopeChange(); }} />
          {fieldErrors.year && <small id="curriculum-year-error" className="app-field-error" role="alert">{fieldErrors.year}</small>}
        </label>
        <label>Class
          <Select ariaLabel="Curriculum class" value={classID} disabled={busy} invalid={Boolean(fieldErrors.class)} describedBy={fieldErrors.class ? "curriculum-class-error" : undefined} placeholder="Select class" options={classes.filter(item => item.is_active).map(item => ({ value: String(item.id), label: item.name }))} onChange={value => { setClassID(value); setSectionID(""); clearEditorForScopeChange(); }} />
          {fieldErrors.class && <small id="curriculum-class-error" className="app-field-error" role="alert">{fieldErrors.class}</small>}
        </label>
        <label>Section
          <Select ariaLabel="Curriculum section" value={sectionID} disabled={busy || !yearNumber || !classNumber} invalid={Boolean(fieldErrors.section)} describedBy={fieldErrors.section ? "curriculum-section-error" : undefined} placeholder="Select section" options={availableSections.map(item => ({ value: String(item.id), label: item.name + (item.medium ? " · " + item.medium : "") }))} onChange={value => { setSectionID(value); clearEditorForScopeChange(); }} />
          {fieldErrors.section && <small id="curriculum-section-error" className="app-field-error" role="alert">{fieldErrors.section}</small>}
          {yearNumber > 0 && classNumber > 0 && availableSections.length === 0 && <small className="app-field-error">No active sections are configured for this class and academic year.</small>}
        </label>
      </div>
      {yearNumber > 0 && classNumber > 0 && sectionNumber > 0 && (
        <section ref={curriculumSummaryRef} className="curriculum-existing" tabIndex={-1}>
          <h3>Current curriculum</h3>
          {curriculaQuery.isLoading ? <p>Loading curriculum…</p> :
            curriculaQuery.isError ? <div className="student-feedback error" role="alert"><p>{curriculaQuery.error.message}</p><button type="button" className="academic-refresh" onClick={() => void curriculaQuery.refetch()}>Retry</button></div> :
            !currentCurriculum ? <p>No curriculum configured for this section yet.</p> :
            <div>
              <p><strong>{currentCurriculum.subjects.length} subjects</strong> · {currentCurriculum.optional_groups.length} optional groups</p>
              <div className="curriculum-version-subjects">
                {currentCurriculum.subjects.map(subject => (
                  <span key={subject.id} className={"curriculum-version-subject " + (subject.is_compulsory ? "compulsory" : subject.optional_group_id ? "grouped" : "optional")}>
                    {subject.name} · {subject.is_compulsory ? "Compulsory" : subject.optional_group_name ?? "Optional"}
                  </span>
                ))}
              </div>
            </div>}
        </section>
      )}
      {canManage ? (
        <form onSubmit={submit} className="curriculum-editor">
          <div className="toolbar"><div><h3>{currentCurriculum ? "Edit curriculum" : "Create curriculum"}</h3><small>Changes update this section's existing curriculum. No new version is created.</small></div>
            <button type="button" className="academic-refresh academic-add" onClick={addGroup} disabled={busy} aria-controls="curriculum-optional-groups">Add optional group</button>
          </div>
          {groups.map((group, index) => (
            <fieldset className="curriculum-group-editor" key={index} tabIndex={-1} data-validation-error={fieldErrors.groups[index]?.subjects ? "true" : undefined} aria-invalid={Boolean(fieldErrors.groups[index]?.subjects)} aria-describedby={fieldErrors.groups[index]?.subjects ? `curriculum-group-${index}-subjects-error` : undefined}>
              <legend>Optional group {index + 1} {newGroupIndex === index && <span className="curriculum-new-group-badge">New group</span>}</legend>
              <label>Group name<input disabled={busy} value={group.name} maxLength={100} placeholder="e.g. Skill Subject" aria-invalid={Boolean(fieldErrors.groups[index]?.name)} aria-describedby={fieldErrors.groups[index]?.name ? `curriculum-group-${index}-name-error` : undefined} onChange={event => {
                const nextName = event.target.value;
                setGroups(current => current.map((item, i) => i === index ? { ...item, name: nextName } : item));
                setGroupBySubject(current => {
                  const previousName = group.name.trim();
                  if (!previousName) return current;
                  return Object.fromEntries(Object.entries(current).map(([id, name]) => [id, name === previousName ? (nextName.trim() ? nextName : "__standalone_optional__") : name]));
                });
                setFieldErrors(current => ({ ...current, subjects: undefined, groups: { ...current.groups, [index]: { ...current.groups[index], name: undefined, min: undefined, max: undefined, subjects: undefined } } }));
              }} />{fieldErrors.groups[index]?.name && <small id={`curriculum-group-${index}-name-error`} className="app-field-error" role="alert">{fieldErrors.groups[index].name}</small>}</label>
              <label>Minimum choices<input disabled={busy} type="number" min={0} max={150} value={group.min_selections} aria-invalid={Boolean(fieldErrors.groups[index]?.min)} aria-describedby={fieldErrors.groups[index]?.min ? `curriculum-group-${index}-min-error` : undefined} onChange={event => { setGroups(current => current.map((item, i) => i === index ? { ...item, min_selections: Number(event.target.value) } : item)); setFieldErrors(current => ({ ...current, groups: { ...current.groups, [index]: { ...current.groups[index], min: undefined } } })); }} />{fieldErrors.groups[index]?.min && <small id={`curriculum-group-${index}-min-error`} className="app-field-error" role="alert">{fieldErrors.groups[index].min}</small>}</label>
              <label>Maximum choices<input disabled={busy} type="number" min={1} max={150} value={group.max_selections} aria-invalid={Boolean(fieldErrors.groups[index]?.max)} aria-describedby={fieldErrors.groups[index]?.max ? `curriculum-group-${index}-max-error` : undefined} onChange={event => { setGroups(current => current.map((item, i) => i === index ? { ...item, max_selections: Number(event.target.value) } : item)); setFieldErrors(current => ({ ...current, groups: { ...current.groups, [index]: { ...current.groups[index], min: undefined, max: undefined } } })); }} />{fieldErrors.groups[index]?.max && <small id={`curriculum-group-${index}-max-error`} className="app-field-error" role="alert">{fieldErrors.groups[index].max}</small>}</label>
              <div className="curriculum-group-subject-assignment" aria-label={`Subject assignment for optional group ${index + 1}`}>
                <span>Subject assignment</span>
                {fieldErrors.groups[index]?.subjects && <small id={`curriculum-group-${index}-subjects-error`} className="app-field-error" role="alert">{fieldErrors.groups[index].subjects}</small>}
              </div>
              <button type="button" className="academic-edit" disabled={busy} onClick={() => {
                const removedName = group.name.trim().toLowerCase();
                setGroups(current => current.filter((_, i) => i !== index));
                setGroupBySubject(current => {
                  if (!removedName) return current;
                  return Object.fromEntries(Object.entries(current).map(([id, name]) => [id, name.trim().toLowerCase() === removedName ? "__standalone_optional__" : name]));
                });
                setFieldErrors(current => ({ ...current, groups: {} }));
              }}>Remove group</button>
            </fieldset>
          ))}
          <div className="curriculum-subject-picker">
            <div className="curriculum-subject-picker-heading">
              <div><h3>Subjects</h3><p>Select the subjects for this class, then choose how each one is assigned.</p></div>
              <span>{included.length} of {activeSubjects.length} selected</span>
            </div>
            <div className="curriculum-subject-summary" aria-live="polite">
              <span className="compulsory"><strong>{compulsoryCount}</strong> Compulsory</span>
              <span className="individual"><strong>{individualOptionalCount}</strong> Optional</span>
              <span className="grouped"><strong>{groupedOptionalCount}</strong> In groups</span>
            </div>
            {activeSubjects.length === 0 ? <p>Add active subjects in Academic Structure first.</p> : activeSubjects.map(subject => {
              const checked = included.includes(subject.id);
              const groupValue = groupBySubject[subject.id] ?? "";
              return (
                <div className="curriculum-subject-row" key={subject.id}>
                  <label><input type="checkbox" disabled={busy} checked={checked} onChange={event => {
                    setIncluded(current => event.target.checked ? [...current, subject.id] : current.filter(id => id !== subject.id));
                    setFieldErrors(current => ({ ...current, subjects: undefined, groups: Object.fromEntries(Object.entries(current.groups).map(([key, errors]) => [key, { ...errors, min: undefined, max: undefined, subjects: undefined }])) }));
                  }} />
                    <span>{subject.name}{subject.subject_code ? " (" + subject.subject_code + ")" : ""}</span>
                  </label>
                  {checked && <label className="curriculum-subject-group-select">Type
                    <select disabled={busy} value={groupValue} onChange={event => { setGroupBySubject(current => ({ ...current, [subject.id]: event.target.value })); setFieldErrors(current => ({ ...current, subjects: undefined })); }}>
                      <option value="">Compulsory</option>
                      <option value="__standalone_optional__">Optional (individual choice)</option>
                      {groups.filter(group => group.name.trim()).map((group, index) => <option key={index} value={group.name}>Optional group: {group.name}</option>)}
                    </select>
                  </label>}
                </div>
              );
            })}
          </div>
          {fieldErrors.subjects && <p className="app-field-error" role="alert">{fieldErrors.subjects}</p>}
          {error && <p className="app-field-error" role="alert">{error}</p>}
          <button type="submit" className="primary academic-add" disabled={busy || !yearNumber || !classNumber || !sectionNumber || curriculaQuery.isLoading || curriculaQuery.isError}>
            {busy ? "Saving curriculum…" : currentCurriculum ? "Save curriculum changes" : "Save curriculum"}
          </button>
        </form>
      ) : <p className="table-message">You have read-only access to class curricula.</p>}
      <Modal
        open={Boolean(pendingSave && pendingRemovalImpacts !== null)}
        title="Confirm subject removal"
        description="These subjects will be removed from the curriculum and from affected students' active selections."
        onClose={() => { if (!busy) { setPendingSave(null); setPendingRemovalImpacts(null); } }}
        closeDisabled={busy}
      >
        {pendingRemovalImpacts && pendingSave && <>
          {pendingRemovalImpacts.length === 0 ? <p>No currently assigned students were found for the removed subjects.</p> : (
            <ul className="curriculum-version-list">
              {pendingRemovalImpacts.map(item => <li key={item.subject_id}>
                <strong>{item.subject_name}</strong>
                <p><strong>{item.affected_students}</strong> affected students</p>
              </li>)}
            </ul>
          )}
          <p>Confirming permanently removes these subjects and the affected students’ current choices. No curriculum or subject-selection history will be kept.</p>
          <div className="app-modal-actions">
            <button type="button" className="app-modal-cancel" disabled={busy} onClick={() => { setPendingSave(null); setPendingRemovalImpacts(null); }}>Cancel</button>
            <button type="button" className="primary app-modal-confirm" disabled={busy} onClick={() => persist(pendingSave, pendingRemovalImpacts)}>
              {save.isPending ? "Applying changes…" : "Confirm and save"}
            </button>
          </div>
        </>}
      </Modal>
    </section>
  );
}
