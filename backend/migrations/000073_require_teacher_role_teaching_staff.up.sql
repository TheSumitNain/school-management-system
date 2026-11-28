-- V73: a Teacher role belongs only to an active Teaching employee profile.
-- Keep the role assignment and staff-type transition rules enforced in PostgreSQL,
-- not only in the School Admin UI.

CREATE OR REPLACE FUNCTION public.guard_v73_teacher_role_assignment_staff_type()
RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v73$
DECLARE
  v_role_key TEXT;
  v_role_school_id BIGINT;
  v_staff_type TEXT;
  v_employment_status TEXT;
BEGIN
  SELECT role_key,school_id
    INTO v_role_key,v_role_school_id
    FROM public.roles
   WHERE id=NEW.role_id
     AND principal_type='staff'
     AND scope_type='school';

  IF FOUND AND v_role_key='teacher' THEN
    SELECT staff_type,employment_status
      INTO v_staff_type,v_employment_status
      FROM public.staff_profiles
     WHERE principal_id=NEW.principal_id
       AND school_id=v_role_school_id
     FOR SHARE;

    IF NOT FOUND
       OR v_staff_type IS DISTINCT FROM 'teaching'
       OR v_employment_status IS DISTINCT FROM 'active' THEN
      RAISE EXCEPTION 'Teacher role can only be assigned to an active Teaching employee'
        USING ERRCODE='23514',
              CONSTRAINT='teacher_role_requires_active_teaching_profile';
    END IF;
  END IF;

  RETURN NEW;
END;
$v73$;

CREATE TRIGGER trg_v73_guard_teacher_role_assignment
BEFORE INSERT OR UPDATE OF principal_id,role_id
ON public.principal_roles
FOR EACH ROW
EXECUTE FUNCTION public.guard_v73_teacher_role_assignment_staff_type();

CREATE OR REPLACE FUNCTION public.guard_v73_teacher_profile_type_transition()
RETURNS TRIGGER
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v73$
BEGIN
  IF NEW.staff_type IS DISTINCT FROM 'teaching'
     AND EXISTS (
       SELECT 1
         FROM public.principal_roles pr
         JOIN public.roles r ON r.id=pr.role_id
        WHERE pr.principal_id=NEW.principal_id
          AND pr.revoked_at IS NULL
          AND r.school_id=NEW.school_id
          AND r.principal_type='staff'
          AND r.scope_type='school'
          AND r.role_key='teacher'
     ) THEN
    RAISE EXCEPTION 'Remove the Teacher role before changing this employee from Teaching to another staff type'
      USING ERRCODE='23514',
            CONSTRAINT='teacher_role_requires_teaching_profile';
  END IF;

  RETURN NEW;
END;
$v73$;

CREATE TRIGGER trg_v73_guard_teacher_profile_type_transition
BEFORE UPDATE OF staff_type
ON public.staff_profiles
FOR EACH ROW
EXECUTE FUNCTION public.guard_v73_teacher_profile_type_transition();
