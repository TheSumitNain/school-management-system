-- V27: optional section medium metadata.
-- Section names remain fully school-defined; medium is independent and optional.
ALTER TABLE public.sections
    ADD COLUMN medium VARCHAR(20);

ALTER TABLE public.sections
    ADD CONSTRAINT sections_medium_allowed
    CHECK (medium IS NULL OR medium IN ('hindi', 'english'));

COMMENT ON COLUMN public.sections.medium IS
'Optional medium of instruction for this section: hindi or english. The section name remains independently customizable.';
