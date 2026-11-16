-- Admission identity must not be dated after the enrollment itself.
-- NOT VALID preserves any historical rows while enforcing the invariant for
-- all new and updated enrollment records.
ALTER TABLE student_enrollments
    ADD CONSTRAINT student_enrollments_admission_not_after_enrollment
    CHECK (admission_date IS NULL OR admission_date <= enrolled_at) NOT VALID;
