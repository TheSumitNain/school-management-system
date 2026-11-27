DROP TRIGGER IF EXISTS trg_validate_v71_substitute_leave ON public.staff_substitutions;
DROP FUNCTION IF EXISTS public.guard_v71_substitution_against_approved_leave();

DROP TRIGGER IF EXISTS trg_guard_v71_leave_approval_against_substitution ON public.staff_leave_requests;
DROP FUNCTION IF EXISTS public.guard_v71_leave_approval_against_substitution();
