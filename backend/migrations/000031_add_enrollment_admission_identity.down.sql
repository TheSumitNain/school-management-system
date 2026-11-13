-- Refuse rollback after enrollment-level admission data has been written.
-- This keeps an accidental migration rollback from silently deleting it.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_enrollments
         WHERE admission_number IS NOT NULL
            OR admission_date IS NOT NULL
    ) THEN
        RAISE EXCEPTION
            'cannot roll back V31 while enrollment admission data exists; reconcile/export data first';
    END IF;
END;
$$;

DROP INDEX IF EXISTS ux_student_enrollments_school_year_admission_number;

ALTER TABLE student_enrollments
    DROP CONSTRAINT IF EXISTS student_enrollments_admission_identity_pair,
    DROP CONSTRAINT IF EXISTS student_enrollments_admission_number_nonempty,
    DROP COLUMN IF EXISTS admission_date,
    DROP COLUMN IF EXISTS admission_number;
