-- V64: persist the staff category used by directory filters and staff workflows.
-- Keep the column nullable for existing profiles: their category must be
-- explicitly classified by a School Admin rather than inferred from names
-- or designations. New profile writes already require a valid staff_type.

ALTER TABLE public.staff_profiles
    ADD COLUMN staff_type VARCHAR(20);

ALTER TABLE public.staff_profiles
    ADD CONSTRAINT staff_profiles_staff_type_check
    CHECK (staff_type IS NULL OR staff_type IN ('teaching', 'non_teaching'));

COMMENT ON COLUMN public.staff_profiles.staff_type IS
'Employee category: teaching or non_teaching. NULL is reserved for legacy profiles awaiting explicit classification.';
