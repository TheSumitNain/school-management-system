-- V42: keep multiple authoritative source identifiers and snapshot history
-- without overwriting the original IGOD State/UT provenance from V40.

CREATE TABLE student_location_source_references (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    location_id BIGINT NOT NULL
        REFERENCES student_locations(id) ON DELETE RESTRICT,
    source_system VARCHAR(30) NOT NULL,
    source_code VARCHAR(50) NOT NULL,
    source_name VARCHAR(120) NOT NULL,
    snapshot_date DATE NOT NULL,
    is_current BOOLEAN NOT NULL DEFAULT TRUE,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    superseded_at TIMESTAMPTZ,
    CONSTRAINT student_location_source_reference_system
        CHECK (
            source_system = UPPER(BTRIM(source_system))
            AND length(BTRIM(source_system)) BETWEEN 2 AND 30
        ),
    CONSTRAINT student_location_source_reference_code
        CHECK (
            source_code = BTRIM(source_code)
            AND length(BTRIM(source_code)) BETWEEN 1 AND 50
        ),
    CONSTRAINT student_location_source_reference_name
        CHECK (
            source_name = BTRIM(source_name)
            AND length(BTRIM(source_name)) BETWEEN 1 AND 120
        ),
    CONSTRAINT student_location_source_reference_lifecycle
        CHECK (
            (is_current = TRUE AND superseded_at IS NULL)
            OR (is_current = FALSE AND superseded_at IS NOT NULL)
        )
);

CREATE UNIQUE INDEX ux_student_location_source_reference_current_code
    ON student_location_source_references (source_system, source_code)
    WHERE is_current = TRUE;

CREATE UNIQUE INDEX ux_student_location_source_reference_current_location
    ON student_location_source_references (location_id, source_system)
    WHERE is_current = TRUE;

CREATE INDEX ix_student_location_source_reference_history
    ON student_location_source_references (location_id, snapshot_date DESC, id DESC);

CREATE OR REPLACE FUNCTION guard_student_location_source_reference()
RETURNS trigger
LANGUAGE plpgsql AS $v42$
DECLARE
    target_school_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student location source references are append-only; supersede instead';
    END IF;

    IF TG_OP = 'INSERT' THEN
        SELECT school_id INTO target_school_id
          FROM public.student_locations
         WHERE id = NEW.location_id
         FOR KEY SHARE;
        IF NOT FOUND OR target_school_id IS NOT NULL THEN
            RAISE EXCEPTION 'source references may only identify shared canonical locations';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.location_id IS DISTINCT FROM OLD.location_id
       OR NEW.source_system IS DISTINCT FROM OLD.source_system
       OR NEW.source_code IS DISTINCT FROM OLD.source_code
       OR NEW.source_name IS DISTINCT FROM OLD.source_name
       OR NEW.snapshot_date IS DISTINCT FROM OLD.snapshot_date
       OR NEW.imported_at IS DISTINCT FROM OLD.imported_at THEN
        RAISE EXCEPTION 'student location source reference identity and source snapshot are immutable';
    END IF;

    IF OLD.is_current = FALSE OR NEW.is_current = TRUE OR NEW.superseded_at IS NULL THEN
        RAISE EXCEPTION 'a current source reference may only be superseded once';
    END IF;

    RETURN NEW;
END;
$v42$;

CREATE TRIGGER trg_guard_student_location_source_reference
BEFORE INSERT OR UPDATE OR DELETE ON student_location_source_references
FOR EACH ROW EXECUTE FUNCTION guard_student_location_source_reference();

-- Preserve V40's existing IGOD identifiers as the first source-reference
-- snapshot. Later LGD identifiers can be added without replacing that source.
INSERT INTO student_location_source_references
    (location_id, source_system, source_code, source_name, snapshot_date)
SELECT id, source_system, source_code, name, DATE '2026-11-14'
  FROM student_locations
 WHERE school_id IS NULL
   AND source_system = 'IGOD'
   AND source_code IS NOT NULL;
