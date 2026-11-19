-- V53 rollback. Existing NULL-scoped legacy curricula and historical
-- enrollment assignments remain unchanged.
DROP INDEX IF EXISTS ix_class_curricula_section_versions;
DROP INDEX IF EXISTS ux_class_curricula_one_active;
DROP INDEX IF EXISTS ux_class_curricula_scope_version;

ALTER TABLE class_curricula
    ADD CONSTRAINT class_curricula_school_year_class_version_unique
    UNIQUE (school_id, academic_year_id, class_id, version_number);

CREATE UNIQUE INDEX ux_class_curricula_one_active
    ON class_curricula (school_id, academic_year_id, class_id)
    WHERE status = 'active';

CREATE INDEX ix_class_curricula_school_year_class
    ON class_curricula (school_id, academic_year_id, class_id, version_number DESC);

ALTER TABLE class_curricula
    DROP CONSTRAINT IF EXISTS class_curricula_section_scope_fkey,
    DROP COLUMN IF EXISTS section_id;

-- Restore the V44 assignment validation semantics.
CREATE OR REPLACE FUNCTION validate_v44_enrollment_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
DECLARE
    v_enrollment_year BIGINT;
    v_enrollment_class BIGINT;
    v_enrollment_status TEXT;
    v_student_status TEXT;
    v_curriculum_year BIGINT;
    v_curriculum_class BIGINT;
    v_curriculum_status TEXT;
BEGIN
    SELECT e.academic_year_id, e.class_id, e.status, st.status
      INTO v_enrollment_year, v_enrollment_class, v_enrollment_status, v_student_status
      FROM student_enrollments e
      JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
     WHERE e.id=NEW.enrollment_id AND e.school_id=NEW.school_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'student enrollment not found for curriculum assignment';
    END IF;
    SELECT academic_year_id, class_id, status
      INTO v_curriculum_year, v_curriculum_class, v_curriculum_status
      FROM class_curricula
     WHERE id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_curriculum_status <> 'active'
       OR v_curriculum_year <> v_enrollment_year
       OR v_curriculum_class <> v_enrollment_class THEN
        RAISE EXCEPTION 'curriculum does not match the active enrollment class and year';
    END IF;
    IF v_enrollment_status <> 'active' OR v_student_status <> 'active' THEN
        RAISE EXCEPTION 'only active students and enrollments can receive a curriculum';
    END IF;
    RETURN NEW;
END;
$v44$;

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'class curriculum versions cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.class_id IS DISTINCT FROM OLD.class_id
       OR NEW.version_number IS DISTINCT FROM OLD.version_number
       OR NEW.created_by_principal_id IS DISTINCT FROM OLD.created_by_principal_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'class curriculum identity is immutable';
    END IF;
    IF OLD.status = 'retired' AND NEW.status <> OLD.status THEN
        RAISE EXCEPTION 'retired class curriculum cannot be reactivated';
    END IF;
    RETURN NEW;
END;
$v44$;
