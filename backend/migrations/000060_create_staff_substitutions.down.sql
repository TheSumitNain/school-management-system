-- V60 rollback is intentionally refused once substitution records exist.
DO $v60_down$
BEGIN
 IF EXISTS(SELECT 1 FROM public.staff_substitutions) THEN
  RAISE EXCEPTION 'V60 rollback refused: substitution history exists; preserve it and use a forward migration';
 END IF;
END;
$v60_down$;
DROP TRIGGER trg_zz_grant_staff_substitution_permissions ON public.schools;
DROP FUNCTION public.grant_staff_substitution_permissions();
ALTER TABLE public.role_permissions DISABLE TRIGGER trg_protect_system_role_permissions;
DELETE FROM public.role_permissions rp
 USING public.roles r,public.permissions p
 WHERE rp.role_id=r.id AND rp.permission_id=p.id
   AND r.role_key='school_admin' AND r.scope_type='school'
   AND r.principal_type='staff' AND r.is_system=TRUE
   AND p.permission_key IN ('staff.substitutions.read','staff.substitutions.manage');
ALTER TABLE public.role_permissions ENABLE TRIGGER trg_protect_system_role_permissions;
DROP TRIGGER trg_guard_v60_substitution_principal_lifecycle ON public.auth_principals;
DROP FUNCTION public.guard_v60_substitution_principal_lifecycle();
DROP TRIGGER trg_guard_v60_substitution_assignment_lifecycle ON public.teacher_subject_assignments;
DROP FUNCTION public.guard_v60_substitution_assignment_lifecycle();
DROP TRIGGER trg_guard_v60_staff_substitution_history ON public.staff_substitutions;
DROP FUNCTION public.guard_v60_staff_substitution_history();
DROP TRIGGER trg_guard_v60_substitution_staff_lifecycle ON public.staff_profiles;
DROP FUNCTION public.guard_v60_substitution_staff_lifecycle();
DROP TRIGGER trg_guard_v60_substitution_login_lifecycle ON public.users;
DROP FUNCTION public.guard_v60_substitution_login_lifecycle();
DROP TRIGGER trg_guard_v60_timetable_substitution_update ON public.timetable_entries;
DROP FUNCTION public.guard_v60_timetable_substitution_history();
DROP TRIGGER trg_validate_v60_staff_substitution ON public.staff_substitutions;
DROP FUNCTION public.validate_v60_staff_substitution();
DROP TABLE public.staff_substitutions;
DELETE FROM public.permissions
 WHERE permission_key IN ('staff.substitutions.read','staff.substitutions.manage');
