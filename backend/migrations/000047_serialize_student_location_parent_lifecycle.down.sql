-- Restore the V34 parent-lock mode when rolling back V47.
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
           AND canonical.country_code = NEW.country_code
           AND canonical.is_active = TRUE
           AND canonical.normalized_name = LOWER(BTRIM(NEW.name))
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            CONSTRAINT = 'ux_student_locations_global_school_collision',
            MESSAGE = 'school location duplicates an available canonical location';
    END IF;

    IF NEW.school_id IS NULL AND EXISTS (
        SELECT 1
          FROM public.student_locations school_owned
         WHERE school_owned.school_id IS NOT NULL
           AND school_owned.parent_id IS NOT DISTINCT FROM NEW.parent_id
           AND school_owned.level = NEW.level
           AND school_owned.country_code = NEW.country_code
           AND school_owned.is_active = TRUE
           AND school_owned.normalized_name = LOWER(BTRIM(NEW.name))
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            CONSTRAINT = 'ux_student_locations_school_global_collision',
            MESSAGE = 'canonical location duplicates an active school-owned location';
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
$v47$;
