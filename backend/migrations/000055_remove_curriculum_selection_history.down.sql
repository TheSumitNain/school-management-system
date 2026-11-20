-- Restore V54's no-delete curriculum and student-selection behavior.

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum_child()
RETURNS trigger
LANGUAGE plpgsql AS $v54$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'class curriculum subject/group history cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        RETURN NEW;
    END IF;

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
        IF (NEW.optional_group_id IS DISTINCT FROM OLD.optional_group_id
            OR NEW.is_optional IS DISTINCT FROM OLD.is_optional)
           AND EXISTS (
               SELECT 1
                 FROM student_enrollment_subject_selections s
                 JOIN student_enrollment_curricula ec
                   ON ec.id=s.enrollment_curriculum_id
                  AND ec.curriculum_id=s.curriculum_id
                  AND ec.school_id=s.school_id
                WHERE s.curriculum_subject_id=OLD.id
                  AND s.school_id=OLD.school_id
                  AND s.revoked_at IS NULL
                  AND ec.status='active'
           ) THEN
            RAISE EXCEPTION 'subject choice rules cannot change while students have active selections';
        END IF;
        IF OLD.is_active AND NOT NEW.is_active
           AND EXISTS (
               SELECT 1
                 FROM student_enrollment_subject_selections s
                 JOIN student_enrollment_curricula ec
                   ON ec.id=s.enrollment_curriculum_id
                  AND ec.curriculum_id=s.curriculum_id
                  AND ec.school_id=s.school_id
                WHERE s.curriculum_subject_id=OLD.id
                  AND s.school_id=OLD.school_id
                  AND s.revoked_at IS NULL
                  AND ec.status='active'
           ) THEN
            RAISE EXCEPTION 'active student subject selections must be revoked before removing a curriculum subject';
        END IF;
        RETURN NEW;
    END IF;

    IF TG_TABLE_NAME = 'class_curriculum_optional_groups'
       AND (NEW.min_selections IS DISTINCT FROM OLD.min_selections
            OR NEW.max_selections IS DISTINCT FROM OLD.max_selections)
       AND EXISTS (
           SELECT 1
             FROM student_enrollment_curricula ec
             CROSS JOIN LATERAL (
                 SELECT COUNT(*)::integer AS selected_count
                   FROM student_enrollment_subject_selections s
                   JOIN class_curriculum_subjects cs
                     ON cs.id=s.curriculum_subject_id
                    AND cs.curriculum_id=s.curriculum_id
                    AND cs.school_id=s.school_id
                  WHERE s.enrollment_curriculum_id=ec.id
                    AND s.school_id=ec.school_id
                    AND s.revoked_at IS NULL
                    AND cs.is_active=TRUE
                    AND cs.optional_group_id=OLD.id
             ) group_choices
            WHERE ec.curriculum_id=OLD.curriculum_id
              AND ec.school_id=OLD.school_id
              AND ec.status='active'
              AND group_choices.selected_count > 0
              AND group_choices.selected_count NOT BETWEEN NEW.min_selections AND NEW.max_selections
       ) THEN
        RAISE EXCEPTION 'optional group limits would invalidate existing student selections';
    END IF;
    RETURN NEW;
END;
$v54$;

DROP TRIGGER IF EXISTS trg_v55_guard_student_subject_selection_update
    ON student_enrollment_subject_selections;
DROP FUNCTION IF EXISTS guard_v55_student_subject_selection_update();

CREATE OR REPLACE FUNCTION guard_v55_student_subject_selection_no_delete()
RETURNS trigger
LANGUAGE plpgsql AS $v55$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student subject selection history cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.enrollment_curriculum_id IS DISTINCT FROM OLD.enrollment_curriculum_id
       OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
       OR NEW.curriculum_subject_id IS DISTINCT FROM OLD.curriculum_subject_id
       OR NEW.subject_name_snapshot IS DISTINCT FROM OLD.subject_name_snapshot
       OR NEW.subject_code_snapshot IS DISTINCT FROM OLD.subject_code_snapshot
       OR NEW.selected_by_principal_id IS DISTINCT FROM OLD.selected_by_principal_id
       OR NEW.selected_at IS DISTINCT FROM OLD.selected_at THEN
        RAISE EXCEPTION 'student subject selection identity and snapshot are immutable';
    END IF;
    IF OLD.revoked_at IS NOT NULL OR NEW.revoked_at IS NULL THEN
        RAISE EXCEPTION 'student subject selection revocation is one-way';
    END IF;
    RETURN NEW;
END;
$v55$;
CREATE TRIGGER trg_v55_guard_student_subject_selection_update
BEFORE UPDATE OR DELETE ON student_enrollment_subject_selections
FOR EACH ROW EXECUTE FUNCTION guard_v55_student_subject_selection_no_delete();

DROP TRIGGER IF EXISTS trg_v55_validate_student_subject_selection
    ON student_enrollment_subject_selections;
DO $v55$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger
         WHERE tgrelid='student_enrollment_subject_selections'::regclass
           AND NOT tgisinternal AND (tgtype & 4) = 4
    ) THEN
        CREATE TRIGGER trg_v55_validate_student_subject_selection
        BEFORE INSERT ON student_enrollment_subject_selections
        FOR EACH ROW EXECUTE FUNCTION validate_v44_enrollment_subject_selection();
    END IF;
END;
$v55$;

COMMENT ON TABLE class_curriculum_subjects IS
    'Current curriculum subjects; removed subjects are soft-deactivated and remain linked to revoked student-selection history.';
COMMENT ON TABLE student_enrollment_subject_selections IS
    'Append-only optional subject choices tied to a specific enrollment and curriculum version.';
