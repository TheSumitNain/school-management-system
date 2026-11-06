-- V11: establish school-scoped student identity and guardian contact records.
-- Academic-year enrollment/class placement and authenticated parent links are
-- deliberately deferred to later, separately reviewed migrations.

CREATE TABLE students (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL
        REFERENCES schools(id) ON DELETE RESTRICT,
    admission_number VARCHAR(50) NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT students_admission_number_nonempty
        CHECK (length(btrim(admission_number)) > 0),
    CONSTRAINT students_full_name_nonempty
        CHECK (length(btrim(full_name)) > 0),
    CONSTRAINT students_id_school_unique UNIQUE (id, school_id)
);

-- Admission numbers are unique within a school after trimming and case-folding.
CREATE UNIQUE INDEX ux_students_school_admission_number
    ON students (school_id, LOWER(BTRIM(admission_number)));

CREATE TABLE guardians (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL
        REFERENCES schools(id) ON DELETE RESTRICT,
    full_name VARCHAR(200) NOT NULL,
    mobile_number VARCHAR(30),
    email_address VARCHAR(254),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT guardians_full_name_nonempty
        CHECK (length(btrim(full_name)) > 0),
    CONSTRAINT guardians_mobile_nonempty
        CHECK (mobile_number IS NULL OR length(btrim(mobile_number)) > 0),
    CONSTRAINT guardians_email_nonempty
        CHECK (email_address IS NULL OR length(btrim(email_address)) > 0),
    CONSTRAINT guardians_id_school_unique UNIQUE (id, school_id)
);

-- Do not make guardian phone/email globally unique: households may share
-- contact details, and contact records are not authenticated parent accounts.
CREATE INDEX ix_guardians_school_name
    ON guardians (school_id, LOWER(BTRIM(full_name)));

CREATE TABLE student_guardians (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL
        REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    guardian_id BIGINT NOT NULL,
    relationship_type VARCHAR(40) NOT NULL,
    is_primary_contact BOOLEAN NOT NULL DEFAULT FALSE,
    is_emergency_contact BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT student_guardians_relationship_nonempty
        CHECK (length(btrim(relationship_type)) BETWEEN 1 AND 40),
    CONSTRAINT student_guardians_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_guardians_guardian_school_fkey
        FOREIGN KEY (guardian_id, school_id)
        REFERENCES guardians (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_guardians_student_guardian_unique
        UNIQUE (student_id, guardian_id)
);

CREATE INDEX ix_student_guardians_school_student
    ON student_guardians (school_id, student_id);

CREATE INDEX ix_student_guardians_school_guardian
    ON student_guardians (school_id, guardian_id);

-- A student or guardian is permanently owned by its creating school.
-- Moving an identity across tenants is not an update operation.
CREATE OR REPLACE FUNCTION guard_v11_school_scope_immutable()
RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.school_id IS DISTINCT FROM OLD.school_id THEN
        RAISE EXCEPTION '% school_id is immutable; create a separately authorized record in the destination school',
            TG_TABLE_NAME;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_students_school_scope_immutable
BEFORE UPDATE OF school_id ON students
FOR EACH ROW EXECUTE FUNCTION guard_v11_school_scope_immutable();

CREATE TRIGGER trg_guardians_school_scope_immutable
BEFORE UPDATE OF school_id ON guardians
FOR EACH ROW EXECUTE FUNCTION guard_v11_school_scope_immutable();

CREATE TRIGGER trg_student_guardians_school_scope_immutable
BEFORE UPDATE OF school_id ON student_guardians
FOR EACH ROW EXECUTE FUNCTION guard_v11_school_scope_immutable();
