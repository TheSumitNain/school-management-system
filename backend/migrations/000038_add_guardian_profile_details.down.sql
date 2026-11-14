-- V38 rollback is intentionally blocked once guardian profile data has been entered.
-- This prevents a rollback from silently discarding staff-maintained parent details.
DO $v38_down$
BEGIN
    IF EXISTS (
        SELECT 1 FROM guardians
         WHERE alternate_mobile_number IS NOT NULL
            OR occupation IS NOT NULL
            OR education IS NOT NULL
            OR workplace_name IS NOT NULL
    ) THEN
        RAISE EXCEPTION
            'V38 rollback refused: guardian profile details exist; export or deliberately clear them before rollback';
    END IF;
END;
$v38_down$;

-- V38 rollback removes only the optional guardian profile attributes.
ALTER TABLE guardians
    DROP CONSTRAINT IF EXISTS guardians_workplace_nonempty,
    DROP CONSTRAINT IF EXISTS guardians_education_nonempty,
    DROP CONSTRAINT IF EXISTS guardians_occupation_nonempty,
    DROP CONSTRAINT IF EXISTS guardians_alternate_mobile_nonempty;

ALTER TABLE guardians
    DROP COLUMN IF EXISTS workplace_name,
    DROP COLUMN IF EXISTS education,
    DROP COLUMN IF EXISTS occupation,
    DROP COLUMN IF EXISTS alternate_mobile_number;
