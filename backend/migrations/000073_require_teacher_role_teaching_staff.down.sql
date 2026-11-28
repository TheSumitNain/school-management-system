DROP TRIGGER IF EXISTS trg_v73_guard_teacher_profile_type_transition ON public.staff_profiles;
DROP FUNCTION IF EXISTS public.guard_v73_teacher_profile_type_transition();

DROP TRIGGER IF EXISTS trg_v73_guard_teacher_role_assignment ON public.principal_roles;
DROP FUNCTION IF EXISTS public.guard_v73_teacher_role_assignment_staff_type();
