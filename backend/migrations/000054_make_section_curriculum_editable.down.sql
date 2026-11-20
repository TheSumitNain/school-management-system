-- Restore V53 immutable curriculum-child behavior.
DROP INDEX IF EXISTS ix_class_curriculum_subjects_active;

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum_child()
RETURNS trigger
LANGUAGE plpgsql AS $v53$
DECLARE
    v_curriculum_id BIGINT;
    v_school_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_curriculum_id := OLD.curriculum_id;
        v_school_id := OLD.school_id;
    ELSIF TG_OP = 'INSERT' THEN
        v_curriculum_id := NEW.curriculum_id;
        v_school_id := NEW.school_id;
    ELSE
        v_curriculum_id := OLD.curriculum_id;
        v_school_id := OLD.school_id;
        IF NEW.id IS DISTINCT FROM OLD.id
           OR NEW.school_id IS DISTINCT FROM OLD.school_id
           OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
           OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
            RAISE EXCEPTION 'class curriculum subject/group identity is immutable';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM student_enrollment_curricula ec
        WHERE ec.curriculum_id = v_curriculum_id AND ec.school_id = v_school_id
    ) THEN
        RAISE EXCEPTION 'curriculum subjects and optional groups are immutable after enrollment assignment';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$v53$;

CREATE OR REPLACE FUNCTION validate_v44_enrollment_subject_selection()
RETURNS trigger
LANGUAGE plpgsql AS $v52$
DECLARE
    v_assignment_status TEXT;
    v_is_optional BOOLEAN;
BEGIN
    SELECT status INTO v_assignment_status
      FROM student_enrollment_curricula
     WHERE id=NEW.enrollment_curriculum_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_assignment_status <> 'active' THEN
        RAISE EXCEPTION 'subject choices require an active enrollment curriculum';
    END IF;
    SELECT is_optional INTO v_is_optional
      FROM class_curriculum_subjects
     WHERE id=NEW.curriculum_subject_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR NOT v_is_optional THEN
        RAISE EXCEPTION 'only optional curriculum subjects can be selected';
    END IF;
    RETURN NEW;
END;
$v52$;

ALTER TABLE class_curriculum_subjects
    DROP COLUMN is_active;
