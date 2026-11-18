-- Restore the original V37 database-session date behavior on rollback.
ALTER TABLE student_basic_details
    DROP CONSTRAINT student_basic_details_birth_date;

ALTER TABLE student_basic_details
    ADD CONSTRAINT student_basic_details_birth_date
    CHECK (
        date_of_birth IS NULL
        OR date_of_birth <= CURRENT_DATE
    );
