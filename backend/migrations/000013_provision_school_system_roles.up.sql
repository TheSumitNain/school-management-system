-- V13: provision the approved school-scoped system roles for every school
-- created after V4. V4 only seeded roles for schools existing at migration time.
--
-- System-role creation remains blocked for ordinary SQL statements. The only
-- runtime path is the AFTER INSERT trigger on schools, which inserts the two
-- fixed staff system roles and their exact permission bundles atomically.

CREATE OR REPLACE FUNCTION public.guard_role_insert_and_system_classification()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v13$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_system THEN
            -- The nested trigger depth is required: only the school-insert
            -- provisioning trigger may create the reserved school system roles.
            IF pg_trigger_depth() < 2
               OR NEW.scope_type <> 'school'
               OR NEW.principal_type <> 'staff'
               OR NEW.school_id IS NULL
               OR NEW.role_key NOT IN ('school_admin', 'teacher') THEN
                RAISE EXCEPTION
                    'school system roles may only be provisioned by the school creation trigger';
            END IF;
            RETURN NEW;
        END IF;

        IF NEW.scope_type <> 'school'
           OR NEW.principal_type <> 'staff'
           OR NEW.school_id IS NULL THEN
            RAISE EXCEPTION
                'runtime custom roles must be school-scoped staff roles';
        END IF;

        IF NEW.created_by IS NULL OR NOT EXISTS (
            SELECT 1
              FROM public.auth_principals ap
              JOIN public.users u ON u.principal_id = ap.id
             WHERE ap.id = NEW.created_by
               AND ap.principal_type = 'staff'
               AND ap.status = 'active'
               AND u.school_id = NEW.school_id
        ) THEN
            RAISE EXCEPTION
                'custom role creator must be an active staff principal in the same school';
        END IF;
    ELSE
        IF NEW.is_system IS DISTINCT FROM OLD.is_system THEN
            RAISE EXCEPTION 'role system classification is immutable';
        END IF;
    END IF;

    RETURN NEW;
END;
$v13$;

CREATE OR REPLACE FUNCTION public.protect_system_roles()
RETURNS trigger
LANGUAGE plpgsql
AS $v13$
DECLARE
    v_old_role_id BIGINT;
    v_new_role_id BIGINT;
BEGIN
    -- Permit only the nested school-creation trigger to add permission rows
    -- to the two reserved school system roles. Updates and deletes remain
    -- forbidden, as do direct INSERT statements at trigger depth 1.
    IF TG_TABLE_NAME = 'role_permissions'
       AND TG_OP = 'INSERT'
       AND pg_trigger_depth() >= 2
       AND EXISTS (
            SELECT 1
              FROM public.roles r
             WHERE r.id = NEW.role_id
               AND r.is_system = TRUE
               AND r.scope_type = 'school'
               AND r.principal_type = 'staff'
               AND r.role_key IN ('school_admin', 'teacher')
       ) THEN
        RETURN NEW;
    END IF;

    IF TG_TABLE_NAME = 'roles' THEN
        IF TG_OP <> 'INSERT' THEN v_old_role_id := OLD.id; END IF;
        IF TG_OP <> 'DELETE' THEN v_new_role_id := NEW.id; END IF;
    ELSE
        IF TG_OP <> 'INSERT' THEN v_old_role_id := OLD.role_id; END IF;
        IF TG_OP <> 'DELETE' THEN v_new_role_id := NEW.role_id; END IF;
    END IF;

    IF (v_old_role_id IS NOT NULL AND EXISTS
        (SELECT 1 FROM public.roles WHERE id = v_old_role_id AND is_system)) OR
       (v_new_role_id IS NOT NULL AND EXISTS
        (SELECT 1 FROM public.roles WHERE id = v_new_role_id AND is_system)) THEN
        RAISE EXCEPTION 'system roles and their permission bundles are immutable';
    END IF;

    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$v13$;

CREATE OR REPLACE FUNCTION public.provision_school_system_roles()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $v13$
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
    IF v_granted_count <> 51 THEN
        RAISE EXCEPTION
            'School Admin system role for school % requires exactly 51 active permissions; found %',
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
$v13$;

CREATE TRIGGER trg_provision_school_system_roles
AFTER INSERT ON public.schools
FOR EACH ROW
EXECUTE FUNCTION public.provision_school_system_roles();

COMMENT ON FUNCTION public.provision_school_system_roles() IS
'Atomically creates the reserved School Admin and Teacher system roles for a newly created school; callable only as a schools AFTER INSERT trigger.';
