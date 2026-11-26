-- Restore the original V62 substitute-only canonical principal guard.
CREATE OR REPLACE FUNCTION public.validate_v62_substitute_principal()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v62$
DECLARE
  v_principal_type VARCHAR(32);
  v_status VARCHAR(20);
BEGIN
  SELECT ap.principal_type,ap.status
    INTO v_principal_type,v_status
    FROM public.auth_principals ap
   WHERE ap.id=NEW.substitute_teacher_principal_id
   FOR UPDATE;
  IF NOT FOUND OR v_principal_type<>'staff' OR v_status<>'active' THEN
    RAISE EXCEPTION 'substitute must have an active canonical staff principal'
      USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v62$;

DROP TRIGGER IF EXISTS trg_validate_v62_substitute_principal ON public.staff_substitutions;
CREATE TRIGGER trg_validate_v62_substitute_principal
BEFORE INSERT OR UPDATE OF substitute_teacher_principal_id,school_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v62_substitute_principal();

-- Restore the original V60 substitute-only account-disable guard.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_principal_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF NEW.principal_type='staff' AND NEW.status='disabled' AND OLD.status IS DISTINCT FROM NEW.status
     AND EXISTS (
       SELECT 1
         FROM public.staff_substitutions ss
         JOIN public.users u ON u.principal_id=NEW.id AND u.school_id=ss.school_id
        WHERE ss.substitute_teacher_principal_id=NEW.id
          AND ss.status='active'
          AND ss.cover_date>=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before disabling this substitute principal' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;
