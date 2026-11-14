-- V38: extend staff-managed guardian contact/profile details.
-- Optional fields preserve existing guardian records and do not create parent accounts.
ALTER TABLE guardians
    ADD COLUMN alternate_mobile_number VARCHAR(30),
    ADD COLUMN occupation VARCHAR(120),
    ADD COLUMN education VARCHAR(120),
    ADD COLUMN workplace_name VARCHAR(160);

ALTER TABLE guardians
    ADD CONSTRAINT guardians_alternate_mobile_nonempty
        CHECK (alternate_mobile_number IS NULL OR length(btrim(alternate_mobile_number)) > 0),
    ADD CONSTRAINT guardians_occupation_nonempty
        CHECK (occupation IS NULL OR length(btrim(occupation)) > 0),
    ADD CONSTRAINT guardians_education_nonempty
        CHECK (education IS NULL OR length(btrim(education)) > 0),
    ADD CONSTRAINT guardians_workplace_nonempty
        CHECK (workplace_name IS NULL OR length(btrim(workplace_name)) > 0);
