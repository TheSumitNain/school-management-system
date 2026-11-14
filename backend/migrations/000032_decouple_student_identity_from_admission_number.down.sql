-- Refuse rollback if the forward model has been used. Restoring the
-- school-wide unique NOT NULL admission number would either invent identity
-- data or reject valid year-scoped data.
DO $v32_down$
BEGIN
    IF EXISTS (SELECT 1 FROM students WHERE admission_number IS NULL) THEN
        RAISE EXCEPTION
            'V32 rollback refused: students without legacy admission numbers exist';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM students
         GROUP BY school_id, LOWER(BTRIM(admission_number))
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            'V32 rollback refused: duplicate legacy admission numbers exist';
    END IF;
END;
$v32_down$;

DROP INDEX ix_students_school_legacy_admission_number;

ALTER TABLE students
    ALTER COLUMN admission_number SET NOT NULL;

CREATE UNIQUE INDEX ux_students_school_admission_number
    ON students (school_id, LOWER(BTRIM(admission_number)));
