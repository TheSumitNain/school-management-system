-- V64 rollback: remove the staff category introduced by this migration.
ALTER TABLE public.staff_profiles
    DROP CONSTRAINT IF EXISTS staff_profiles_staff_type_check,
    DROP COLUMN IF EXISTS staff_type;
