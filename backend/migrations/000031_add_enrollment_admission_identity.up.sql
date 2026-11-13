-- V31: move the admission-number source of truth toward academic-year enrollment.
-- This is an expand-only migration. Existing student-level admission numbers
-- remain untouched until every reader/writer has been switched and legacy data
-- has been reconciled. Historical enrollments may remain NULL when their
-- original year-specific admission number cannot be proven.
ALTER TABLE student_enrollments
    ADD COLUMN admission_number VARCHAR(50),
    ADD COLUMN admission_date DATE;

ALTER TABLE student_enrollments
    ADD CONSTRAINT student_enrollments_admission_number_nonempty
        CHECK (
            admission_number IS NULL
            OR length(btrim(admission_number)) BETWEEN 1 AND 50
        ),
    ADD CONSTRAINT student_enrollments_admission_identity_pair
        CHECK (
            (admission_number IS NULL) = (admission_date IS NULL)
        );

-- Admission numbers may be reused in a different academic year, but not
-- duplicated within the same school/year after trimming and case folding.
-- NULL legacy rows are intentionally excluded until reconciliation.
CREATE UNIQUE INDEX ux_student_enrollments_school_year_admission_number
    ON student_enrollments (
        school_id,
        academic_year_id,
        lower(btrim(admission_number))
    )
    WHERE admission_number IS NOT NULL;
