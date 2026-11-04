-- V18 rollback removes only lifecycle guards and restores the exact V17 section guard.
DROP TRIGGER IF EXISTS trg_guard_v18_class_incharge_assignment_history ON public.class_incharge_assignments;
DROP TRIGGER IF EXISTS trg_guard_v18_teacher_subject_assignment_history ON public.teacher_subject_assignments;
DROP FUNCTION IF EXISTS public.guard_v18_assignment_history();

CREATE OR REPLACE FUNCTION public.validate_v17_section()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v17$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM public.classes c
          JOIN public.academic_years ay
            ON ay.id = NEW.academic_year_id
           AND ay.school_id = NEW.school_id
         WHERE c.id = NEW.class_id
           AND c.school_id = NEW.school_id
           AND c.is_active = TRUE
           AND ay.status <> 'closed'
    ) THEN
        RAISE EXCEPTION 'new sections require an active class and a non-closed academic year in the same school';
    END IF;
    RETURN NEW;
END;
$v17$;

DROP TRIGGER IF EXISTS trg_validate_v17_section ON public.sections;
CREATE TRIGGER trg_validate_v17_section
BEFORE INSERT OR UPDATE OF school_id, academic_year_id, class_id
ON public.sections
FOR EACH ROW EXECUTE FUNCTION public.validate_v17_section();
