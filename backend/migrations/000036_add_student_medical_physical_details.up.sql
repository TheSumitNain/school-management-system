-- V36: keep optional medical and physical profile data separate from
-- permanent student identity and expose it only through restricted endpoints.

CREATE TABLE student_medical_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    blood_group VARCHAR(3),
    allergies VARCHAR(2000),
    known_conditions VARCHAR(2000),
    ongoing_medications VARCHAR(2000),
    emergency_medical_notes VARCHAR(2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_medical_details_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_medical_details_student_unique UNIQUE (student_id),
    CONSTRAINT student_medical_details_blood_group
        CHECK (blood_group IS NULL OR blood_group IN ('A+','A-','B+','B-','AB+','AB-','O+','O-')),
    CONSTRAINT student_medical_details_allergies
        CHECK (allergies IS NULL OR length(BTRIM(allergies)) BETWEEN 1 AND 2000),
    CONSTRAINT student_medical_details_known_conditions
        CHECK (known_conditions IS NULL OR length(BTRIM(known_conditions)) BETWEEN 1 AND 2000),
    CONSTRAINT student_medical_details_medications
        CHECK (ongoing_medications IS NULL OR length(BTRIM(ongoing_medications)) BETWEEN 1 AND 2000),
    CONSTRAINT student_medical_details_emergency_notes
        CHECK (emergency_medical_notes IS NULL OR length(BTRIM(emergency_medical_notes)) BETWEEN 1 AND 2000)
);

CREATE INDEX ix_student_medical_details_school_student
    ON student_medical_details (school_id, student_id);

CREATE TABLE student_physical_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    height_cm NUMERIC(5,2),
    weight_kg NUMERIC(5,2),
    measured_on DATE,
    vision_notes VARCHAR(1000),
    physical_notes VARCHAR(2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_physical_details_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_physical_details_student_unique UNIQUE (student_id),
    CONSTRAINT student_physical_details_height_range
        CHECK (height_cm IS NULL OR height_cm BETWEEN 20 AND 250),
    CONSTRAINT student_physical_details_weight_range
        CHECK (weight_kg IS NULL OR weight_kg BETWEEN 1 AND 300),
    CONSTRAINT student_physical_details_vision_notes
        CHECK (vision_notes IS NULL OR length(BTRIM(vision_notes)) BETWEEN 1 AND 1000),
    CONSTRAINT student_physical_details_physical_notes
        CHECK (physical_notes IS NULL OR length(BTRIM(physical_notes)) BETWEEN 1 AND 2000)
);

CREATE INDEX ix_student_physical_details_school_student
    ON student_physical_details (school_id, student_id);

CREATE OR REPLACE FUNCTION guard_student_health_profile()
RETURNS trigger
LANGUAGE plpgsql AS $v36$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION '% records cannot be deleted; clear or update the section instead', TG_TABLE_NAME;
    END IF;

    IF TG_OP = 'UPDATE' AND (
        NEW.id IS DISTINCT FROM OLD.id
        OR NEW.school_id IS DISTINCT FROM OLD.school_id
        OR NEW.student_id IS DISTINCT FROM OLD.student_id
        OR NEW.created_at IS DISTINCT FROM OLD.created_at
    ) THEN
        RAISE EXCEPTION '% ownership and identity are immutable', TG_TABLE_NAME;
    END IF;

    RETURN NEW;
END;
$v36$;

CREATE TRIGGER trg_guard_student_medical_details
BEFORE UPDATE OR DELETE ON student_medical_details
FOR EACH ROW EXECUTE FUNCTION guard_student_health_profile();

CREATE TRIGGER trg_guard_student_physical_details
BEFORE UPDATE OR DELETE ON student_physical_details
FOR EACH ROW EXECUTE FUNCTION guard_student_health_profile();

COMMENT ON TABLE student_medical_details IS
    'Optional student health information, readable only through students.read_sensitive and audited at the repository boundary.';

COMMENT ON TABLE student_physical_details IS
    'Optional dated physical measurements and notes, readable only through students.read_sensitive and audited at the repository boundary.';
