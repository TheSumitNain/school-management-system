-- V62: enforce the canonical staff principal status when assigning a substitute.
-- users.status is mirrored for compatibility, but auth_principals is the
-- authoritative identity/login status in the current authentication model.

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
