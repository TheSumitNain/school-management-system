-- V53: support section-specific curricula while preserving legacy class-level
-- curriculum versions and all enrollment/subject-selection history.
--
-- Legacy rows intentionally retain NULL section_id. New application writes
-- must supply a section_id; the composite FK binds it to the same school,
-- academic year and class.

ALTER TABLE class_curricula
    ADD COLUMN section_id BIGINT;

ALTER TABLE class_curricula
    ADD CONSTRAINT class_curricula_section_scope_fkey
    FOREIGN KEY (section_id, school_id, academic_year_id, class_id)
    REFERENCES sections (id, school_id, academic_year_id, class_id)
    ON DELETE RESTRICT;

ALTER TABLE class_curricula
    DROP CONSTRAINT class_curricula_school_year_class_version_unique;

-- Treat NULL as the legacy class-level scope (0 is not a valid section ID).
-- This preserves uniqueness for old rows while allowing independent version
-- sequences for each real section.
CREATE UNIQUE INDEX ux_class_curricula_scope_version
    ON class_curricula (school_id, academic_year_id, class_id, COALESCE(section_id, 0), version_number);

DROP INDEX ux_class_curricula_one_active;
CREATE UNIQUE INDEX ux_class_curricula_one_active
    ON class_curricula (school_id, academic_year_id, class_id, COALESCE(section_id, 0))
    WHERE status = 'active';

CREATE INDEX ix_class_curricula_section_versions
    ON class_curricula (school_id, academic_year_id, class_id, section_id, version_number DESC);

-- Curriculum identity remains immutable after creation.
CREATE OR REPLACE FUNCTION guard_v44_class_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v53$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'class curriculum versions cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.class_id IS DISTINCT FROM OLD.class_id
       OR NEW.section_id IS DISTINCT FROM OLD.section_id
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
$v53$;

-- New enrollment assignments must match the student's exact section.
-- Existing legacy assignments are retained; this trigger only validates
-- future INSERTs and never rewrites historical assignments.
CREATE OR REPLACE FUNCTION validate_v44_enrollment_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v53$
DECLARE
    v_enrollment_year BIGINT;
    v_enrollment_class BIGINT;
    v_enrollment_section BIGINT;
    v_enrollment_status TEXT;
    v_student_status TEXT;
    v_curriculum_year BIGINT;
    v_curriculum_class BIGINT;
    v_curriculum_section BIGINT;
    v_curriculum_status TEXT;
BEGIN
    SELECT e.academic_year_id, e.class_id, e.section_id, e.status, st.status
      INTO v_enrollment_year, v_enrollment_class, v_enrollment_section,
           v_enrollment_status, v_student_status
      FROM student_enrollments e
      JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
     WHERE e.id=NEW.enrollment_id AND e.school_id=NEW.school_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'student enrollment not found for curriculum assignment';
    END IF;

    SELECT academic_year_id, class_id, section_id, status
      INTO v_curriculum_year, v_curriculum_class, v_curriculum_section, v_curriculum_status
      FROM class_curricula
     WHERE id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_curriculum_status <> 'active'
       OR v_curriculum_year <> v_enrollment_year
       OR v_curriculum_class <> v_enrollment_class
       OR v_curriculum_section IS DISTINCT FROM v_enrollment_section THEN
        RAISE EXCEPTION 'curriculum does not match the active enrollment section, class and year';
    END IF;

    IF v_enrollment_status <> 'active' OR v_student_status <> 'active' THEN
        RAISE EXCEPTION 'only active students and enrollments can receive a curriculum';
    END IF;
    RETURN NEW;
END;
$v53$;

COMMENT ON COLUMN class_curricula.section_id IS
    'Section-specific curriculum scope. NULL is reserved for immutable legacy class-level curriculum versions; new writes must specify a section.';
