-- V51: guardian contact identifiers are optional but unique within a school.
-- Formatting variants of an Indian mobile number resolve to the same 10-digit key.
CREATE OR REPLACE FUNCTION normalize_guardian_mobile(value TEXT)
RETURNS TEXT
LANGUAGE SQL
IMMUTABLE
STRICT
PARALLEL SAFE
AS $v51$
    WITH digits AS (
        SELECT regexp_replace(value, '[^0-9]', '', 'g') AS value
    )
    SELECT CASE
        WHEN length(value) = 12 AND left(value, 2) = '91' THEN right(value, 10)
        WHEN length(value) = 11 AND left(value, 1) = '0' THEN right(value, 10)
        ELSE NULLIF(value, '')
    END
    FROM digits;
$v51$;

-- Never merge, delete, or rewrite existing guardian contacts automatically.
-- Stop the migration with an actionable preflight error if legacy duplicates
-- exist; resolve those records manually before retrying V51.
DO $v51$
DECLARE
    duplicate_mobile_groups INTEGER;
    duplicate_email_groups INTEGER;
BEGIN
    SELECT COUNT(*) INTO duplicate_mobile_groups
    FROM (
        SELECT school_id, normalize_guardian_mobile(mobile_number)
        FROM guardians
        WHERE mobile_number IS NOT NULL
          AND normalize_guardian_mobile(mobile_number) IS NOT NULL
        GROUP BY school_id, normalize_guardian_mobile(mobile_number)
        HAVING COUNT(*) > 1
    ) duplicates;

    SELECT COUNT(*) INTO duplicate_email_groups
    FROM (
        SELECT school_id, lower(btrim(email_address))
        FROM guardians
        WHERE email_address IS NOT NULL AND btrim(email_address) <> ''
        GROUP BY school_id, lower(btrim(email_address))
        HAVING COUNT(*) > 1
    ) duplicates;

    IF duplicate_mobile_groups > 0 OR duplicate_email_groups > 0 THEN
        RAISE EXCEPTION
            'V51 blocked: found % duplicate school/mobile groups and % duplicate school/email groups. Review guardians manually; do not delete or merge automatically.',
            duplicate_mobile_groups, duplicate_email_groups;
    END IF;
END;
$v51$;

CREATE UNIQUE INDEX ux_guardians_school_mobile_unique
    ON guardians (school_id, normalize_guardian_mobile(mobile_number))
    WHERE mobile_number IS NOT NULL
      AND normalize_guardian_mobile(mobile_number) IS NOT NULL;

CREATE UNIQUE INDEX ux_guardians_school_email_unique
    ON guardians (school_id, lower(btrim(email_address)))
    WHERE email_address IS NOT NULL
      AND btrim(email_address) <> '';
