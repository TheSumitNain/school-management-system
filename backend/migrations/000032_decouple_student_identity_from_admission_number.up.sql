-- V32: the permanent student identity is no longer identified by a
-- school-wide admission number. Admission identity belongs to a specific
-- academic-year enrollment (V31). Keep legacy values for compatibility and
-- audit/history, but allow new student identities to have no legacy value and
-- allow the same legacy value to exist on multiple student rows during the
-- transition. The V31 enrollment index remains the authoritative uniqueness
-- rule: school + academic year + normalized admission number.

ALTER TABLE students
    ALTER COLUMN admission_number DROP NOT NULL;

DROP INDEX ux_students_school_admission_number;

-- Retain an efficient lookup path for legacy values while readers are migrated.
CREATE INDEX ix_students_school_legacy_admission_number
    ON students (school_id, LOWER(BTRIM(admission_number)))
    WHERE admission_number IS NOT NULL;
