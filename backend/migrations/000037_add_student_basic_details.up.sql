-- V37: independently maintained basic student identity details.
-- These fields are optional and are not required to create the permanent
-- student identity or its first academic-year enrollment. Aadhaar is
-- intentionally excluded from this profile.

CREATE TABLE student_basic_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    date_of_birth DATE,
    place_of_birth VARCHAR(120),
    gender VARCHAR(24),
    nationality VARCHAR(80),
    religion VARCHAR(80),
    social_category VARCHAR(24),
    ews_status BOOLEAN,
    school_registration_number VARCHAR(50),
    created_by_principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    updated_by_principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_basic_details_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_basic_details_student_unique UNIQUE (student_id),
    CONSTRAINT student_basic_details_birth_date CHECK (
        date_of_birth IS NULL OR date_of_birth <= CURRENT_DATE
    ),
    CONSTRAINT student_basic_details_place_of_birth CHECK (
        place_of_birth IS NULL OR length(BTRIM(place_of_birth)) BETWEEN 1 AND 120
    ),
    CONSTRAINT student_basic_details_gender CHECK (
        gender IS NULL OR gender IN ('female','male','other','not_specified')
    ),
    CONSTRAINT student_basic_details_nationality CHECK (
        nationality IS NULL OR length(BTRIM(nationality)) BETWEEN 1 AND 80
    ),
    CONSTRAINT student_basic_details_religion CHECK (
        religion IS NULL OR length(BTRIM(religion)) BETWEEN 1 AND 80
    ),
    CONSTRAINT student_basic_details_social_category CHECK (
        social_category IS NULL OR social_category IN ('general','obc','sc','st','other','not_specified')
    ),
    CONSTRAINT student_basic_details_registration_number CHECK (
        school_registration_number IS NULL OR length(BTRIM(school_registration_number)) BETWEEN 1 AND 50
    )
);

CREATE UNIQUE INDEX ux_student_basic_details_school_registration_number
    ON student_basic_details (school_id, lower(BTRIM(school_registration_number)))
    WHERE school_registration_number IS NOT NULL;

CREATE INDEX ix_student_basic_details_school_student
    ON student_basic_details (school_id, student_id);

CREATE OR REPLACE FUNCTION guard_student_basic_details()
RETURNS trigger
LANGUAGE plpgsql
AS $v37$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student basic details cannot be deleted; clear optional values instead';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.created_by_principal_id IS DISTINCT FROM OLD.created_by_principal_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.updated_at < OLD.updated_at THEN
        RAISE EXCEPTION 'student basic detail identity and creator are immutable';
    END IF;
    RETURN NEW;
END;
$v37$;

CREATE TRIGGER trg_guard_student_basic_details
BEFORE UPDATE OR DELETE ON student_basic_details
FOR EACH ROW EXECUTE FUNCTION guard_student_basic_details();

COMMENT ON TABLE student_basic_details IS
    'Optional student profile details. Reads require students.read_sensitive; updates require students.update and audit changed field names only.';
