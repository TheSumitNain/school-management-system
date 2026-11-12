-- V27 rollback: remove optional section medium metadata.
ALTER TABLE public.sections
    DROP CONSTRAINT IF EXISTS sections_medium_allowed;
ALTER TABLE public.sections
    DROP COLUMN IF EXISTS medium;
