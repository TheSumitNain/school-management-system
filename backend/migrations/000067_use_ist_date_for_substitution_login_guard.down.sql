-- Restore the V60 session-date behavior when rolling back V67.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_login_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF NEW.status='disabled' AND OLD.status IS DISTINCT FROM NEW.status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.substitute_teacher_principal_id=NEW.principal_id
          AND ss.status='active'
          AND ss.cover_date>=CURRENT_DATE
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before disabling this substitute login' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;
