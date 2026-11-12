-- V28 rollback: remove employee profile metadata and its login-safety trigger.
DROP TRIGGER IF EXISTS trg_staff_profile_inactive_disables_login ON public.staff_profiles;
DROP FUNCTION IF EXISTS public.disable_staff_login_for_inactive_profile();
DROP TABLE IF EXISTS public.staff_profiles;
