-- V64: explicitly classify staff as teaching or non-teaching.
-- NULL is retained for legacy profiles; no historic classification is inferred.
ALTER TABLE public.staff_profiles
    ADD COLUMN staff_type VARCHAR(20);

ALTER TABLE public.staff_profiles
    ADD CONSTRAINT staff_profiles_staff_type_check
    CHECK (staff_type IS NULL OR staff_type IN ('teaching', 'non_teaching'));
