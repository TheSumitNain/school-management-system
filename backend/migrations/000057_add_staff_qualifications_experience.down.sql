-- V57 rollback: refuse to discard staff education or employment history.
DO $v57_down$
BEGIN
    IF EXISTS (SELECT 1 FROM public.staff_qualifications)
       OR EXISTS (SELECT 1 FROM public.staff_certifications)
       OR EXISTS (SELECT 1 FROM public.staff_experience) THEN
        RAISE EXCEPTION
            'V57 rollback refused: staff qualification, certification or experience records exist; preserve the history and use a forward migration';
    END IF;
END;
$v57_down$;

DROP TABLE public.staff_experience;
DROP TABLE public.staff_certifications;
DROP TABLE public.staff_qualifications;
ALTER TABLE public.staff_profiles
    DROP CONSTRAINT uq_staff_profiles_principal_school;
