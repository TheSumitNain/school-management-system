-- Restore the V62 substitute-only validation trigger before removing V70.
DROP TRIGGER IF EXISTS trg_guard_v70_staff_disable_active_substitutions ON public.auth_principals;
DROP FUNCTION IF EXISTS public.guard_v70_staff_disable_active_substitutions();
DROP TRIGGER IF EXISTS trg_validate_v70_substitution_principal_pair ON public.staff_substitutions;
DROP FUNCTION IF EXISTS public.validate_v70_substitution_principal_pair();

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

CREATE TRIGGER trg_validate_v62_substitute_principal
BEFORE INSERT OR UPDATE OF substitute_teacher_principal_id,school_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v62_substitute_principal();
