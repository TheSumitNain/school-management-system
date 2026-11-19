-- V52 rollback: restore the V44 group-based optional-selection rule.
CREATE OR REPLACE FUNCTION validate_v44_enrollment_subject_selection()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
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
    SELECT optional_group_id IS NOT NULL INTO v_is_optional
      FROM class_curriculum_subjects
     WHERE id=NEW.curriculum_subject_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR NOT v_is_optional THEN
        RAISE EXCEPTION 'only optional curriculum subjects can be selected';
    END IF;
    RETURN NEW;
END;
$v44$;

ALTER TABLE class_curriculum_subjects
    DROP CONSTRAINT IF EXISTS class_curriculum_subjects_group_requires_optional;
ALTER TABLE class_curriculum_subjects
    DROP COLUMN IF EXISTS is_optional;
