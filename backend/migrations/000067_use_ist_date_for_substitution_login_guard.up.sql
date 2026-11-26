-- V67: use the school calendar date consistently when guarding staff login disablement.
-- V60 used the database session's CURRENT_DATE, which can differ from India time
-- around midnight. Keep the guard aligned with the application's school calendar.

CREATE OR REPLACE FUNCTION public.guard_v60_substitution_login_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v67$
BEGIN
  IF NEW.status='disabled' AND OLD.status IS DISTINCT FROM NEW.status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.substitute_teacher_principal_id=NEW.principal_id
          AND ss.status='active'
          AND ss.cover_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before disabling this substitute login' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v67$;
