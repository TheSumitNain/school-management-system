-- V39: grant guardian management to every School Admin system role.
-- The system-role protection trigger is removed only for this reviewed,
-- migration-owned bundle update, then restored before the migration completes.

DROP TRIGGER trg_protect_system_role_permissions ON public.role_permissions;

INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM public.roles r
  JOIN public.permissions p
    ON p.permission_key IN ('guardians.read', 'guardians.manage')
   AND p.scope_type = 'school'
   AND p.is_active = TRUE
 WHERE r.is_system = TRUE
   AND r.scope_type = 'school'
   AND r.principal_type = 'staff'
   AND r.role_key = 'school_admin'
ON CONFLICT (role_id, permission_id) DO NOTHING;

CREATE TRIGGER trg_protect_system_role_permissions
BEFORE INSERT OR UPDATE OR DELETE ON public.role_permissions
FOR EACH ROW EXECUTE FUNCTION public.protect_system_roles();

-- Keep future school provisioning aligned with the migrated system-role
-- bundle. The teacher permission bundle remains unchanged.
CREATE OR REPLACE FUNCTION public.provision_school_system_roles()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $v39$
DECLARE
    v_school_admin_role_id BIGINT;
    v_teacher_role_id BIGINT;
    v_granted_count INTEGER;
BEGIN
    INSERT INTO public.roles
        (school_id, principal_type, scope_type, role_key, name, is_system)
    VALUES
        (NEW.id, 'staff', 'school', 'school_admin', 'School Admin', TRUE)
    RETURNING id INTO v_school_admin_role_id;

    INSERT INTO public.role_permissions (role_id, permission_id)
    SELECT v_school_admin_role_id, p.id
      FROM public.permissions p
     WHERE p.scope_type = 'school'
       AND p.is_active = TRUE
       AND p.permission_key = ANY (ARRAY[
            'school.settings.read',
            'school.settings.update',
            'school.audit.read',
            'academic_years.read',
            'academic_years.manage',
            'classes.read',
            'classes.manage',
            'subjects.read',
            'subjects.manage',
            'teacher_assignments.read',
            'teacher_assignments.manage',
            'class_incharge_assignments.read',
            'class_incharge_assignments.manage',
            'students.read_basic',
            'students.read_sensitive',
            'students.create',
            'students.update',
            'students.archive',
            'guardians.read',
            'guardians.manage',
            'attendance.read',
            'attendance.mark',
            'attendance.update_same_day',
            'attendance.correct_past',
            'timetable.read',
            'timetable.manage',
            'homework.read',
            'homework.create',
            'homework.update',
            'homework.delete',
            'exams.read',
            'exams.manage',
            'results.read',
            'results.enter',
            'results.update',
            'results.publish',
            'notices.read',
            'notices.create',
            'notices.update',
            'notices.publish',
            'notices.delete',
            'users.read',
            'users.create',
            'users.update',
            'users.disable',
            'users.password_reset',
            'roles.read',
            'roles.create_custom',
            'roles.update_custom',
            'roles.assign',
            'permissions.read',
            'permissions.assign',
            'permissions.override'
       ]::TEXT[]);
    GET DIAGNOSTICS v_granted_count = ROW_COUNT;
    IF v_granted_count <> 53 THEN
        RAISE EXCEPTION
            'School Admin system role for school % requires exactly 53 active permissions; found %',
            NEW.id, v_granted_count;
    END IF;

    INSERT INTO public.roles
        (school_id, principal_type, scope_type, role_key, name, is_system)
    VALUES
        (NEW.id, 'staff', 'school', 'teacher', 'Teacher', TRUE)
    RETURNING id INTO v_teacher_role_id;

    INSERT INTO public.role_permissions (role_id, permission_id)
    SELECT v_teacher_role_id, p.id
      FROM public.permissions p
     WHERE p.scope_type = 'school'
       AND p.is_active = TRUE
       AND p.permission_key = ANY (ARRAY[
            'students.read_basic',
            'attendance.read',
            'attendance.mark',
            'attendance.update_same_day',
            'timetable.read',
            'homework.read',
            'notices.read'
       ]::TEXT[]);
    GET DIAGNOSTICS v_granted_count = ROW_COUNT;
    IF v_granted_count <> 7 THEN
        RAISE EXCEPTION
            'Teacher system role for school % requires exactly 7 active permissions; found %',
            NEW.id, v_granted_count;
    END IF;

    RETURN NEW;
END;
$v39$;

COMMENT ON FUNCTION public.provision_school_system_roles() IS
'Atomically creates the reserved School Admin (53 permissions) and Teacher (7 permissions) system roles for a newly created school; callable only as a schools AFTER INSERT trigger.';

DO $v39_verify$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.roles r
          CROSS JOIN (VALUES ('guardians.read'), ('guardians.manage')) required(permission_key)
          LEFT JOIN public.permissions p
            ON p.permission_key = required.permission_key
           AND p.scope_type = 'school'
           AND p.is_active = TRUE
          LEFT JOIN public.role_permissions rp
            ON rp.role_id = r.id
           AND rp.permission_id = p.id
         WHERE r.is_system = TRUE
           AND r.scope_type = 'school'
           AND r.principal_type = 'staff'
           AND r.role_key = 'school_admin'
           AND (p.id IS NULL OR rp.role_id IS NULL)
    ) THEN
        RAISE EXCEPTION 'V39 failed to grant both guardian permissions to every School Admin system role';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM public.roles r
          JOIN public.role_permissions rp ON rp.role_id = r.id
          JOIN public.permissions p ON p.id = rp.permission_id
         WHERE r.is_system = TRUE
           AND r.scope_type = 'school'
           AND r.principal_type = 'staff'
           AND r.role_key = 'teacher'
           AND p.permission_key IN ('guardians.read', 'guardians.manage')
    ) THEN
        RAISE EXCEPTION 'V39 must not grant guardian permissions to Teacher system roles';
    END IF;
END;
$v39_verify$;
