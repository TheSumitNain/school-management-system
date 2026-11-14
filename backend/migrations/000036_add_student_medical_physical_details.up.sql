-- V36: restricted, auditable student health profiles with preserved history.
-- Medical profile edits snapshot the previous version; physical measurements
-- are append-only dated records. Neither section is part of permanent identity.

CREATE TABLE student_medical_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    version_number INTEGER NOT NULL DEFAULT 1,
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
    CONSTRAINT student_medical_details_version_positive CHECK (version_number >= 1),
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

CREATE TABLE student_medical_details_history (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    version_number INTEGER NOT NULL,
    blood_group VARCHAR(3),
    allergies VARCHAR(2000),
    known_conditions VARCHAR(2000),
    ongoing_medications VARCHAR(2000),
    emergency_medical_notes VARCHAR(2000),
    changed_by_principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_medical_details_history_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_medical_details_history_version_unique UNIQUE (student_id, version_number),
    CONSTRAINT student_medical_details_history_blood_group
        CHECK (blood_group IS NULL OR blood_group IN ('A+','A-','B+','B-','AB+','AB-','O+','O-'))
);

CREATE INDEX ix_student_medical_details_history_school_student
    ON student_medical_details_history (school_id, student_id, version_number DESC);

CREATE TABLE student_physical_details (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    height_cm NUMERIC(5,2),
    weight_kg NUMERIC(5,2),
    measured_on DATE,
    vision_notes VARCHAR(1000),
    physical_notes VARCHAR(2000),
    recorded_by_principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_physical_details_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_physical_details_height_range
        CHECK (height_cm IS NULL OR height_cm BETWEEN 20 AND 250),
    CONSTRAINT student_physical_details_weight_range
        CHECK (weight_kg IS NULL OR weight_kg BETWEEN 1 AND 300),
    CONSTRAINT student_physical_details_measurement_date_required
        CHECK ((height_cm IS NULL AND weight_kg IS NULL) OR measured_on IS NOT NULL),
    CONSTRAINT student_physical_details_nonempty
        CHECK (height_cm IS NOT NULL OR weight_kg IS NOT NULL OR vision_notes IS NOT NULL OR physical_notes IS NOT NULL),
    CONSTRAINT student_physical_details_vision_notes
        CHECK (vision_notes IS NULL OR length(BTRIM(vision_notes)) BETWEEN 1 AND 1000),
    CONSTRAINT student_physical_details_physical_notes
        CHECK (physical_notes IS NULL OR length(BTRIM(physical_notes)) BETWEEN 1 AND 2000)
);

CREATE INDEX ix_student_physical_details_school_student_history
    ON student_physical_details (school_id, student_id, measured_on DESC, created_at DESC, id DESC);

CREATE OR REPLACE FUNCTION guard_student_medical_details()
RETURNS trigger
LANGUAGE plpgsql AS $v36$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student medical profile cannot be deleted; clear its values instead';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.version_number <> OLD.version_number + 1 THEN
        RAISE EXCEPTION 'student medical profile identity is immutable and versions must advance exactly once';
    END IF;
    RETURN NEW;
END;
$v36$;

CREATE OR REPLACE FUNCTION guard_student_health_history()
RETURNS trigger
LANGUAGE plpgsql AS $v36$
BEGIN
    RAISE EXCEPTION '% history records are append-only', TG_TABLE_NAME;
END;
$v36$;

CREATE TRIGGER trg_guard_student_medical_details
BEFORE UPDATE OR DELETE ON student_medical_details
FOR EACH ROW EXECUTE FUNCTION guard_student_medical_details();

CREATE TRIGGER trg_guard_student_medical_details_history
BEFORE UPDATE OR DELETE ON student_medical_details_history
FOR EACH ROW EXECUTE FUNCTION guard_student_health_history();

CREATE TRIGGER trg_guard_student_physical_details
BEFORE UPDATE OR DELETE ON student_physical_details
FOR EACH ROW EXECUTE FUNCTION guard_student_health_history();

COMMENT ON TABLE student_medical_details IS
    'Current optional student health information; reads require students.read_sensitive and are audited.';

COMMENT ON TABLE student_medical_details_history IS
    'Append-only prior medical profile versions; restricted reads are audited and values are excluded from audit metadata.';

COMMENT ON TABLE student_physical_details IS
    'Append-only dated physical measurements and notes; reads require students.read_sensitive and are audited.';
