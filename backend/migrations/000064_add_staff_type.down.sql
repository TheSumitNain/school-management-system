ALTER TABLE public.staff_profiles
    DROP CONSTRAINT IF EXISTS staff_profiles_staff_type_check;
ALTER TABLE public.staff_profiles
    DROP COLUMN IF EXISTS staff_type;
