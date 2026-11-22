-- V58 rollback must preserve any recorded staff attendance history.
DO $v58_down$
BEGIN
    IF EXISTS (SELECT 1 FROM public.staff_attendance_records) THEN
        RAISE EXCEPTION
            'V58 rollback refused: staff attendance records exist; preserve attendance history and use a forward migration';
    END IF;
END;
$v58_down$;

DROP TRIGGER trg_validate_staff_attendance_date ON public.staff_attendance_records;
DROP FUNCTION public.validate_staff_attendance_date();
DROP TRIGGER trg_z_grant_staff_attendance_permissions ON public.schools;
DROP FUNCTION public.grant_staff_attendance_permissions();

-- V4 protects system-role bundles; disable only that guard within this
-- transactional rollback while removing precisely the V58 permission grants.
ALTER TABLE public.role_permissions DISABLE TRIGGER trg_protect_system_role_permissions;
DELETE FROM public.role_permissions rp
 USING public.roles r, public.permissions p
 WHERE rp.role_id = r.id
   AND rp.permission_id = p.id
   AND r.role_key = 'school_admin'
   AND r.scope_type = 'school'
   AND r.principal_type = 'staff'
   AND r.is_system = TRUE
   AND p.permission_key IN (
       'staff.attendance.read',
       'staff.attendance.mark',
       'staff.attendance.update_same_day',
       'staff.attendance.correct_past'
   );

ALTER TABLE public.role_permissions ENABLE TRIGGER trg_protect_system_role_permissions;

DROP TABLE public.staff_attendance_records;

DELETE FROM public.permissions
 WHERE permission_key IN (
       'staff.attendance.read',
       'staff.attendance.mark',
       'staff.attendance.update_same_day',
       'staff.attendance.correct_past'
 );
