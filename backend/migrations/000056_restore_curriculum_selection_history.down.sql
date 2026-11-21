-- Restore V55 current-state-only curriculum and student selection behavior.

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum_child()
RETURNS trigger
LANGUAGE plpgsql AS $v55$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF TG_TABLE_NAME = 'class_curriculum_subjects' THEN
            RETURN OLD;
        END IF;
        IF TG_TABLE_NAME = 'class_curriculum_optional_groups'
           AND NOT EXISTS (
               SELECT 1 FROM class_curriculum_subjects cs
                WHERE cs.optional_group_id=OLD.id
                  AND cs.curriculum_id=OLD.curriculum_id
                  AND cs.school_id=OLD.school_id
           ) THEN
            RETURN OLD;
        END IF;
        RAISE EXCEPTION 'optional groups can only be deleted after all subjects are ungrouped or removed';
    END IF;
    IF TG_OP = 'INSERT' THEN RETURN NEW; END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'class curriculum subject/group identity is immutable';
    END IF;
    IF TG_TABLE_NAME = 'class_curriculum_subjects' THEN
        IF NEW.subject_id IS DISTINCT FROM OLD.subject_id
           OR NEW.subject_name_snapshot IS DISTINCT FROM OLD.subject_name_snapshot
           OR NEW.subject_code_snapshot IS DISTINCT FROM OLD.subject_code_snapshot THEN
            RAISE EXCEPTION 'curriculum subject source and snapshots are immutable';
        END IF;
        IF (NEW.is_optional IS DISTINCT FROM OLD.is_optional
            OR (NEW.optional_group_id IS DISTINCT FROM OLD.optional_group_id
                AND NEW.optional_group_id IS NOT NULL))
           AND EXISTS (
               SELECT 1 FROM student_enrollment_subject_selections s
               JOIN student_enrollment_curricula ec
                 ON ec.id=s.enrollment_curriculum_id AND ec.curriculum_id=s.curriculum_id AND ec.school_id=s.school_id
              WHERE s.curriculum_subject_id=OLD.id AND s.school_id=OLD.school_id
                AND s.revoked_at IS NULL AND ec.status='active'
           ) THEN
            RAISE EXCEPTION 'subject choice rules cannot change while students have active selections';
        END IF;
        RETURN NEW;
    END IF;
    IF TG_TABLE_NAME = 'class_curriculum_optional_groups'
       AND (NEW.min_selections IS DISTINCT FROM OLD.min_selections
            OR NEW.max_selections IS DISTINCT FROM OLD.max_selections)
       AND EXISTS (
           SELECT 1 FROM student_enrollment_curricula ec
           CROSS JOIN LATERAL (
               SELECT COUNT(*)::integer AS selected_count
                 FROM student_enrollment_subject_selections s
                 JOIN class_curriculum_subjects cs
                   ON cs.id=s.curriculum_subject_id AND cs.curriculum_id=s.curriculum_id AND cs.school_id=s.school_id
                WHERE s.enrollment_curriculum_id=ec.id AND s.school_id=ec.school_id
                  AND s.revoked_at IS NULL AND cs.is_active=TRUE AND cs.optional_group_id=OLD.id
           ) group_choices
          WHERE ec.curriculum_id=OLD.curriculum_id AND ec.school_id=OLD.school_id
            AND ec.status='active' AND group_choices.selected_count > 0
            AND group_choices.selected_count NOT BETWEEN NEW.min_selections AND NEW.max_selections
       ) THEN
        RAISE EXCEPTION 'optional group limits would invalidate existing student selections';
    END IF;
    RETURN NEW;
END;
$v55$;

DROP TRIGGER IF EXISTS trg_guard_v44_enrollment_subject_history
    ON student_enrollment_subject_selections;
DROP INDEX IF EXISTS ix_student_enrollment_subject_selection_history;

CREATE OR REPLACE FUNCTION guard_v55_student_subject_selection_update()
RETURNS trigger
LANGUAGE plpgsql AS $v55$
BEGIN
    RAISE EXCEPTION 'student subject choices are current-state rows; update by replacing the row instead of retaining history';
END;
$v55$;
CREATE TRIGGER trg_v55_guard_student_subject_selection_update
BEFORE UPDATE ON student_enrollment_subject_selections
FOR EACH ROW EXECUTE FUNCTION guard_v55_student_subject_selection_update();

COMMENT ON COLUMN class_curriculum_subjects.is_active IS
    'Compatibility flag for legacy V54 rows. Current edits permanently delete removed subjects; newly saved current subjects remain active.';
COMMENT ON TABLE class_curriculum_subjects IS
    'Current editable curriculum subjects. Confirmed removals delete the subject and its current student choices; no curriculum-selection history is retained.';
COMMENT ON TABLE student_enrollment_subject_selections IS
    'Current optional subject choices for active enrollment curricula. Saving changes replaces current rows; no selection history is retained.';
