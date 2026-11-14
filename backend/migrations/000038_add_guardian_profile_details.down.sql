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
