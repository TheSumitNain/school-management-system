-- V66: dated substitute coverage is a teacher-only workflow. Both the
-- original timetable owner and the substitute must remain explicitly active
-- Teaching employees; a legacy assignment or active login is not sufficient.

CREATE OR REPLACE FUNCTION public.validate_v66_substitution_staff_type()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v66$
DECLARE
  v_original_eligible BOOLEAN;
  v_substitute_eligible BOOLEAN;
BEGIN
  SELECT EXISTS (
    SELECT 1 FROM public.staff_profiles sp
    JOIN public.auth_principals ap ON ap.id=sp.principal_id AND ap.principal_type='staff'
    JOIN public.users u ON u.principal_id=sp.principal_id AND u.school_id=sp.school_id
    WHERE sp.principal_id=NEW.original_teacher_principal_id
      AND sp.school_id=NEW.school_id
      AND sp.staff_type='teaching' AND sp.employment_status='active'
      AND ap.status='active' AND u.status='active'
  ) INTO v_original_eligible;

  SELECT EXISTS (
    SELECT 1 FROM public.staff_profiles sp
    JOIN public.auth_principals ap ON ap.id=sp.principal_id AND ap.principal_type='staff'
    JOIN public.users u ON u.principal_id=sp.principal_id AND u.school_id=sp.school_id
    WHERE sp.principal_id=NEW.substitute_teacher_principal_id
      AND sp.school_id=NEW.school_id
      AND sp.staff_type='teaching' AND sp.employment_status='active'
      AND ap.status='active' AND u.status='active'
  ) INTO v_substitute_eligible;

  IF NOT v_original_eligible OR NOT v_substitute_eligible THEN
    RAISE EXCEPTION USING
      ERRCODE='23514',
      CONSTRAINT='staff_substitution_teaching_profile_check',
      MESSAGE='substitution requires both original and substitute staff to have active Teaching profiles';
  END IF;
  RETURN NEW;
END;
$v66$;

CREATE TRIGGER trg_validate_v66_substitution_staff_type
BEFORE INSERT OR UPDATE OF school_id,timetable_entry_id,original_teacher_principal_id,substitute_teacher_principal_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v66_substitution_staff_type();
