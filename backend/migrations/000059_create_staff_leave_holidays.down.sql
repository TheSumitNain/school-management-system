-- Preserve approved/pending/rejected leave and holiday history on rollback.
DO $v59_down$
BEGIN
 IF EXISTS(SELECT 1 FROM public.staff_leave_requests) OR
    EXISTS(SELECT 1 FROM public.staff_leave_types) OR
    EXISTS(SELECT 1 FROM public.school_holidays) THEN
  RAISE EXCEPTION 'V59 rollback refused: leave or holiday records exist; preserve history and use a forward migration';
 END IF;
END;
$v59_down$;
DROP TRIGGER trg_zz_grant_staff_leave_holiday_permissions ON public.schools;
DROP FUNCTION public.grant_staff_leave_holiday_permissions();
ALTER TABLE public.role_permissions DISABLE TRIGGER trg_protect_system_role_permissions;
DELETE FROM public.role_permissions rp USING public.roles r,public.permissions p
WHERE rp.role_id=r.id AND rp.permission_id=p.id
 AND r.role_key='school_admin' AND r.scope_type='school' AND r.principal_type='staff' AND r.is_system=TRUE
 AND p.permission_key IN ('staff.leave.read','staff.leave.manage_types','staff.leave.manage_requests',
 'staff.leave.decide','staff.holidays.read','staff.holidays.manage');
ALTER TABLE public.role_permissions ENABLE TRIGGER trg_protect_system_role_permissions;
DROP TABLE public.school_holidays;
DROP TABLE public.staff_leave_requests;
DROP TABLE public.staff_leave_types;
DELETE FROM public.permissions WHERE permission_key IN ('staff.leave.read','staff.leave.manage_types',
 'staff.leave.manage_requests','staff.leave.decide','staff.holidays.read','staff.holidays.manage');
