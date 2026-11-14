-- V34: normalized, tenant-aware student address and location foundation.
-- Canonical locations may be shared (school_id IS NULL); schools may add
-- their own missing locations beneath canonical or same-school parents.

CREATE TABLE student_locations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT REFERENCES schools(id) ON DELETE RESTRICT,
    parent_id BIGINT REFERENCES student_locations(id) ON DELETE RESTRICT,
    country_code VARCHAR(2) NOT NULL DEFAULT 'IN',
    level VARCHAR(20) NOT NULL
        CHECK (level IN ('state', 'district', 'subdistrict', 'locality')),
    name VARCHAR(120) NOT NULL,
    normalized_name VARCHAR(120)
        GENERATED ALWAYS AS (LOWER(BTRIM(name))) STORED,
    locality_type VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_locations_name_nonempty
        CHECK (length(BTRIM(name)) BETWEEN 1 AND 120),
    CONSTRAINT student_locations_country_code
        CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT student_locations_locality_type
        CHECK (
            (level = 'locality' AND locality_type IN ('village', 'city', 'town', 'other'))
            OR (level <> 'locality' AND locality_type IS NULL)
        )
);

-- Global canonical and school-owned additions have separate uniqueness
-- scopes. A school may override an unavailable canonical child only by
-- creating its own child under the same parent; duplicates within an owner
-- and parent are rejected after trimming and case folding.
CREATE UNIQUE INDEX ux_student_locations_global_name
    ON student_locations (COALESCE(parent_id, 0), level, normalized_name)
    WHERE school_id IS NULL;
CREATE UNIQUE INDEX ux_student_locations_school_name
    ON student_locations (school_id, COALESCE(parent_id, 0), level, normalized_name)
    WHERE school_id IS NOT NULL;
CREATE INDEX ix_student_locations_parent_active
    ON student_locations (parent_id, is_active, normalized_name);
CREATE INDEX ix_student_locations_school_active
    ON student_locations (school_id, parent_id, is_active, normalized_name);

CREATE OR REPLACE FUNCTION guard_student_location_hierarchy()
RETURNS trigger
LANGUAGE plpgsql AS $v34$
DECLARE
    parent_school_id BIGINT;
    parent_level TEXT;
    parent_country_code TEXT;
    expected_level TEXT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student locations cannot be deleted; deactivate instead';
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF NEW.school_id IS DISTINCT FROM OLD.school_id
           OR NEW.parent_id IS DISTINCT FROM OLD.parent_id
           OR NEW.level IS DISTINCT FROM OLD.level
           OR NEW.country_code IS DISTINCT FROM OLD.country_code THEN
            RAISE EXCEPTION 'student location ownership and hierarchy are immutable';
        END IF;

        IF OLD.is_active = TRUE AND NEW.is_active = FALSE
           AND EXISTS (
               SELECT 1 FROM public.student_locations child
                WHERE child.parent_id = OLD.id AND child.is_active = TRUE
           ) THEN
            RAISE EXCEPTION 'student location with active children cannot be deactivated';
        END IF;
    END IF;

    IF NEW.school_id IS NOT NULL AND EXISTS (
        SELECT 1
          FROM public.student_locations canonical
         WHERE canonical.school_id IS NULL
           AND canonical.parent_id IS NOT DISTINCT FROM NEW.parent_id
           AND canonical.level = NEW.level
           AND canonical.normalized_name = LOWER(BTRIM(NEW.name))
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            CONSTRAINT = 'ux_student_locations_global_school_collision',
            MESSAGE = 'school location duplicates an available canonical location';
    END IF;

    IF NEW.parent_id IS NULL THEN
        IF NEW.level <> 'state' THEN
            RAISE EXCEPTION 'student location hierarchy must start at a state';
        END IF;
        RETURN NEW;
    END IF;

    SELECT school_id, level, country_code
      INTO parent_school_id, parent_level, parent_country_code
      FROM public.student_locations
     WHERE id = NEW.parent_id
       AND is_active = TRUE
     FOR KEY SHARE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'student location parent does not exist or is inactive';
    END IF;

    expected_level := CASE parent_level
        WHEN 'state' THEN 'district'
        WHEN 'district' THEN 'subdistrict'
        WHEN 'subdistrict' THEN 'locality'
        ELSE NULL
    END;

    IF expected_level IS NULL OR NEW.level <> expected_level THEN
        RAISE EXCEPTION 'student location parent level does not match child level';
    END IF;

    IF parent_school_id IS NOT NULL
       AND parent_school_id IS DISTINCT FROM NEW.school_id THEN
        RAISE EXCEPTION 'student location cannot reference another school location';
    END IF;

    IF parent_country_code IS DISTINCT FROM NEW.country_code THEN
        RAISE EXCEPTION 'student location country must match its parent';
    END IF;

    RETURN NEW;
END;
$v34$;

CREATE TRIGGER trg_guard_student_location_hierarchy
BEFORE INSERT OR UPDATE OR DELETE ON student_locations
FOR EACH ROW EXECUTE FUNCTION guard_student_location_hierarchy();

CREATE TABLE student_addresses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    student_id BIGINT NOT NULL,
    address_type VARCHAR(20) NOT NULL
        CHECK (address_type IN ('current', 'permanent')),
    address_line_1 VARCHAR(200),
    address_line_2 VARCHAR(200),
    area_mohalla VARCHAR(120),
    landmark VARCHAR(120),
    postal_code VARCHAR(20),
    location_id BIGINT REFERENCES student_locations(id) ON DELETE RESTRICT,
    same_as_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_addresses_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_addresses_student_type_unique
        UNIQUE (student_id, address_type),
    CONSTRAINT student_addresses_same_as_current
        CHECK (
            NOT same_as_current
            OR (
                address_type = 'permanent'
                AND address_line_1 IS NULL
                AND address_line_2 IS NULL
                AND area_mohalla IS NULL
                AND landmark IS NULL
                AND postal_code IS NULL
                AND location_id IS NULL
            )
        )
);

CREATE INDEX ix_student_addresses_school_student
    ON student_addresses (school_id, student_id);

CREATE OR REPLACE FUNCTION guard_student_address_location()
RETURNS trigger
LANGUAGE plpgsql AS $v34$
DECLARE
    location_school_id BIGINT;
    location_level TEXT;
    location_active BOOLEAN;
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student address records cannot be deleted';
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF NEW.school_id IS DISTINCT FROM OLD.school_id
           OR NEW.student_id IS DISTINCT FROM OLD.student_id
           OR NEW.address_type IS DISTINCT FROM OLD.address_type THEN
            RAISE EXCEPTION 'student address ownership and type are immutable';
        END IF;
    END IF;

    IF NEW.location_id IS NOT NULL THEN
        SELECT school_id, level, is_active
          INTO location_school_id, location_level, location_active
          FROM public.student_locations
         WHERE id = NEW.location_id
         FOR KEY SHARE;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'student address location does not exist';
        END IF;
        IF NOT location_active THEN
            IF TG_OP = 'INSERT' THEN
                RAISE EXCEPTION 'student address cannot select an inactive location';
            ELSIF NEW.location_id IS DISTINCT FROM OLD.location_id THEN
                RAISE EXCEPTION 'student address cannot select an inactive location';
            END IF;
        END IF;

        IF location_level <> 'locality' THEN
            RAISE EXCEPTION 'student address must reference a village/city locality';
        END IF;

        IF location_school_id IS NOT NULL
           AND location_school_id IS DISTINCT FROM NEW.school_id THEN
            RAISE EXCEPTION 'student address cannot reference another school location';
        END IF;
    END IF;

    IF NEW.same_as_current AND NOT EXISTS (
        SELECT 1
          FROM public.student_addresses current_address
         WHERE current_address.school_id = NEW.school_id
           AND current_address.student_id = NEW.student_id
           AND current_address.address_type = 'current'
    ) THEN
        RAISE EXCEPTION 'current address must exist before permanent address can be marked same';
    END IF;

    RETURN NEW;
END;
$v34$;

CREATE TRIGGER trg_guard_student_address_location
BEFORE INSERT OR UPDATE OR DELETE ON student_addresses
FOR EACH ROW EXECUTE FUNCTION guard_student_address_location();

COMMENT ON TABLE student_locations IS
    'Hierarchical shared canonical or school-owned locations; referenced rows are deactivated, not deleted.';
COMMENT ON TABLE student_addresses IS
    'Current and permanent student addresses; same-as-current permanent addresses reference the current row without duplicating its fields.';
