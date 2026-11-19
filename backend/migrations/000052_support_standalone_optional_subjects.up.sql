-- V52: distinguish standalone optional subjects from grouped optional choices.
-- Existing grouped subjects remain optional; subjects without a group remain compulsory.
ALTER TABLE class_curriculum_subjects
    ADD COLUMN is_optional BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE class_curriculum_subjects
   SET is_optional = TRUE
 WHERE optional_group_id IS NOT NULL;

ALTER TABLE class_curriculum_subjects
    ADD CONSTRAINT class_curriculum_subjects_group_requires_optional
    CHECK (optional_group_id IS NULL OR is_optional = TRUE);

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

COMMENT ON COLUMN class_curriculum_subjects.is_optional IS
    'True for standalone optional subjects and subjects belonging to an optional group; false for compulsory subjects.';

-- Keep legacy/direct SQL writers compatible: a subject placed in a group is optional.
CREATE OR REPLACE FUNCTION set_v52_group_subject_optional()
RETURNS trigger
LANGUAGE plpgsql AS $v52$
BEGIN
    IF NEW.optional_group_id IS NOT NULL THEN
        NEW.is_optional := TRUE;
    END IF;
    RETURN NEW;
END;
$v52$;

CREATE TRIGGER trg_set_v52_group_subject_optional
BEFORE INSERT OR UPDATE OF optional_group_id, is_optional
ON class_curriculum_subjects
FOR EACH ROW EXECUTE FUNCTION set_v52_group_subject_optional();
