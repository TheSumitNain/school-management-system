-- V49: align the database-enforced date-of-birth boundary with the
-- system-wide school calendar timezone. PostgreSQL CURRENT_DATE follows the
-- session timezone and may disagree with the authoritative Asia/Kolkata date.
ALTER TABLE student_basic_details
    DROP CONSTRAINT student_basic_details_birth_date;

ALTER TABLE student_basic_details
    ADD CONSTRAINT student_basic_details_birth_date
    CHECK (
        date_of_birth IS NULL
        OR date_of_birth <= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
    );
