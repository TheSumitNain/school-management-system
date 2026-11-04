-- V18 rollback removes only lifecycle guards and restores the V17 section trigger.
DROP TRIGGER IF EXISTS trg_guard_v18_class_incharge_assignment_history ON public.class_incharge_assignments;
DROP TRIGGER IF EXISTS trg_guard_v18_teacher_subject_assignment_history ON public.teacher_subject_assignments;
DROP FUNCTION IF EXISTS public.guard_v18_assignment_history();

DROP TRIGGER IF EXISTS trg_validate_v17_section ON public.sections;
CREATE TRIGGER trg_validate_v17_section
BEFORE INSERT OR UPDATE OF school_id, academic_year_id, class_id
ON public.sections
FOR EACH ROW EXECUTE FUNCTION public.validate_v17_section();
