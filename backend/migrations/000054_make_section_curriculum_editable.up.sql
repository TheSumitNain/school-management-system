-- V54: make the section curriculum editable in place while preserving student selection history.
-- Removed curriculum subjects are soft-deactivated; active student selections are
-- revoked in the same transaction and remain visible in selection history.

ALTER TABLE class_curriculum_subjects
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX ix_class_curriculum_subjects_active
    ON class_curriculum_subjects (curriculum_id, subject_id)
    WHERE is_active = TRUE;

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
            WHERE ec.curriculum_id=OLD.curriculum_id
              AND ec.school_id=OLD.school_id
              AND ec.status='active'
              AND (
                  SELECT COUNT(*)
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
              ) > 0
                AND (
                  SELECT COUNT(*)
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
                ) NOT BETWEEN NEW.min_selections AND NEW.max_selections
       ) THEN
        RAISE EXCEPTION 'optional group limits would invalidate existing student selections';
    END IF;

    RETURN NEW;
END;
$v54$;

CREATE OR REPLACE FUNCTION validate_v44_enrollment_subject_selection()
RETURNS trigger
LANGUAGE plpgsql AS $v54$
DECLARE
    v_assignment_status TEXT;
    v_is_optional BOOLEAN;
    v_subject_active BOOLEAN;
BEGIN
    SELECT status INTO v_assignment_status
      FROM student_enrollment_curricula
     WHERE id=NEW.enrollment_curriculum_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_assignment_status <> 'active' THEN
        RAISE EXCEPTION 'subject choices require an active enrollment curriculum';
    END IF;

    SELECT is_optional, is_active
      INTO v_is_optional, v_subject_active
      FROM class_curriculum_subjects
     WHERE id=NEW.curriculum_subject_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR NOT v_is_optional OR NOT v_subject_active THEN
        RAISE EXCEPTION 'only active optional curriculum subjects can be selected';
    END IF;
    RETURN NEW;
END;
$v54$;

COMMENT ON COLUMN class_curriculum_subjects.is_active IS
    'False means the subject was removed from the editable curriculum; historical student selections remain linked and revoked.';
