-- Support the enrollment directory's two ordering modes without a sort over
-- every historical enrollment in a school: all years, and one selected year.
CREATE INDEX ix_student_enrollments_school_enrolled_at
    ON student_enrollments (school_id, enrolled_at DESC, id DESC);

CREATE INDEX ix_student_enrollments_school_year_enrolled_at
    ON student_enrollments (school_id, academic_year_id, enrolled_at DESC, id DESC);
