-- V35: student contact channels are independently editable and optional.
-- Contact values are deliberately not unique: households may share phones
-- and email addresses, and these are not authenticated parent identities.

CREATE TABLE student_contact_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    primary_phone VARCHAR(30),
    alternate_phone VARCHAR(30),
    email_address VARCHAR(254),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_contact_details_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_contact_details_student_unique UNIQUE (student_id),
    CONSTRAINT student_contact_details_primary_phone
        CHECK (primary_phone IS NULL OR length(BTRIM(primary_phone)) BETWEEN 1 AND 30),
    CONSTRAINT student_contact_details_alternate_phone
        CHECK (alternate_phone IS NULL OR length(BTRIM(alternate_phone)) BETWEEN 1 AND 30),
    CONSTRAINT student_contact_details_email
        CHECK (
            email_address IS NULL
            OR (
                length(BTRIM(email_address)) BETWEEN 3 AND 254
                AND position('@' IN email_address) > 1
            )
        )
);

CREATE INDEX ix_student_contact_details_school_student
    ON student_contact_details (school_id, student_id);

CREATE OR REPLACE FUNCTION guard_student_contact_details()
RETURNS trigger
LANGUAGE plpgsql AS $v35$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student contact details cannot be deleted; update the section instead';
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF NEW.school_id IS DISTINCT FROM OLD.school_id
           OR NEW.student_id IS DISTINCT FROM OLD.student_id THEN
            RAISE EXCEPTION 'student contact ownership is immutable';
        END IF;
    END IF;

    RETURN NEW;
END;
$v35$;

CREATE TRIGGER trg_guard_student_contact_details
BEFORE INSERT OR UPDATE OR DELETE ON student_contact_details
FOR EACH ROW EXECUTE FUNCTION guard_student_contact_details();

COMMENT ON TABLE student_contact_details IS
    'Optional student phone/email contact channels; shared household contacts are allowed and values are protected by student-profile RBAC.';
