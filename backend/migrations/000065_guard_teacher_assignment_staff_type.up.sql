-- V65: enforce Staff Type and employment eligibility at the teacher-assignment boundary.
-- A login role alone is not enough: only an explicitly classified, active Teaching
-- employee may receive a new subject or class-in-charge assignment.

CREATE OR REPLACE FUNCTION public.validate_v65_teacher_assignment_staff_profile()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v65$
DECLARE
    v_staff_type VARCHAR(20);
    v_employment_status VARCHAR(20);
BEGIN
    SELECT sp.staff_type, sp.employment_status
      INTO v_staff_type, v_employment_status
      FROM public.staff_profiles sp
     WHERE sp.principal_id = NEW.teacher_principal_id
       AND sp.school_id = NEW.school_id
     FOR SHARE;

    IF NOT FOUND
       OR v_staff_type IS DISTINCT FROM 'teaching'
       OR v_employment_status IS DISTINCT FROM 'active' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            CONSTRAINT = 'teacher_assignment_staff_profile_check',
            MESSAGE = 'teacher assignment requires an active employee profile classified as Teaching';
    END IF;

    RETURN NEW;
END;
$v65$;

CREATE TRIGGER trg_validate_v65_teacher_subject_staff_profile
BEFORE INSERT OR UPDATE OF school_id, teacher_principal_id
ON public.teacher_subject_assignments
FOR EACH ROW EXECUTE FUNCTION public.validate_v65_teacher_assignment_staff_profile();

CREATE TRIGGER trg_validate_v65_class_incharge_staff_profile
BEFORE INSERT OR UPDATE OF school_id, teacher_principal_id
ON public.class_incharge_assignments
FOR EACH ROW EXECUTE FUNCTION public.validate_v65_teacher_assignment_staff_profile();

-- Keep an employee from becoming Non-teaching or Inactive while active academic
-- assignments still depend on that employee as a teacher. The row lock taken by
-- the assignment validator serializes concurrent profile/assignment changes.
CREATE OR REPLACE FUNCTION public.guard_v65_staff_profile_teacher_assignments()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v65$
BEGIN
    IF (OLD.staff_type IS DISTINCT FROM NEW.staff_type
        OR OLD.employment_status IS DISTINCT FROM NEW.employment_status)
       AND (NEW.staff_type IS DISTINCT FROM 'teaching'
            OR NEW.employment_status IS DISTINCT FROM 'active')
       AND (
            EXISTS (
                SELECT 1
                  FROM public.teacher_subject_assignments tsa
                 WHERE tsa.teacher_principal_id = OLD.principal_id
                   AND tsa.school_id = OLD.school_id
                   AND tsa.revoked_at IS NULL
            )
            OR EXISTS (
                SELECT 1
                  FROM public.class_incharge_assignments cia
                 WHERE cia.teacher_principal_id = OLD.principal_id
                   AND cia.school_id = OLD.school_id
                   AND cia.revoked_at IS NULL
            )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            CONSTRAINT = 'staff_profiles_teacher_assignments_check',
            MESSAGE = 'active teaching and class in-charge assignments must be reassigned or removed before changing Staff Type or Employment Status';
    END IF;

    RETURN NEW;
END;
$v65$;

CREATE TRIGGER trg_guard_v65_staff_profile_teacher_assignments
BEFORE UPDATE OF staff_type, employment_status
ON public.staff_profiles
FOR EACH ROW EXECUTE FUNCTION public.guard_v65_staff_profile_teacher_assignments();
