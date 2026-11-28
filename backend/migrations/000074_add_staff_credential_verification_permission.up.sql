-- V74: reserve credential verification for the School Admin system role.
-- Verification is intentionally separate from users.update and cannot be delegated.
INSERT INTO public.permissions(permission_key, description, scope_type, is_delegable)
VALUES (
  'staff.credentials.verify',
  'Verify or reject staff education qualifications and professional certifications',
  'school',
  FALSE
);

CREATE OR REPLACE FUNCTION public.grant_staff_credential_verification_permission()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path=pg_catalog,public
AS $v74_grant$
DECLARE
  v_role_id BIGINT;
  v_permission_id BIGINT;
  v_count BIGINT;
BEGIN
  SELECT id INTO v_role_id
    FROM public.roles
   WHERE school_id=NEW.id
     AND principal_type='staff'
     AND scope_type='school'
     AND role_key='school_admin'
     AND is_system=TRUE;
  IF v_role_id IS NULL THEN
    RAISE EXCEPTION 'School Admin system role missing for school %', NEW.id;
  END IF;

  SELECT id INTO v_permission_id
    FROM public.permissions
   WHERE permission_key='staff.credentials.verify'
     AND scope_type='school'
     AND is_active=TRUE;
  IF v_permission_id IS NULL THEN
    RAISE EXCEPTION 'Staff credential verification permission is unavailable';
  END IF;

  INSERT INTO public.role_permissions(role_id,permission_id)
  VALUES(v_role_id,v_permission_id)
  ON CONFLICT DO NOTHING;
  GET DIAGNOSTICS v_count=ROW_COUNT;
  IF v_count<>1 THEN
    RAISE EXCEPTION 'Expected one staff credential verification grant for school %, granted %', NEW.id, v_count;
  END IF;
  RETURN NEW;
END;
$v74_grant$;

REVOKE ALL ON FUNCTION public.grant_staff_credential_verification_permission() FROM PUBLIC;

CREATE TRIGGER trg_zz_grant_staff_credential_verification_permission
AFTER INSERT ON public.schools
FOR EACH ROW
EXECUTE FUNCTION public.grant_staff_credential_verification_permission();

-- Backfill every school that existed before this migration using the same
-- fail-closed grant path used for schools created after V74.
CREATE TEMP TABLE v74_existing_school_ids(id BIGINT PRIMARY KEY) ON COMMIT DROP;
CREATE TRIGGER trg_v74_existing_school_grants
AFTER INSERT ON pg_temp.v74_existing_school_ids
FOR EACH ROW
EXECUTE FUNCTION public.grant_staff_credential_verification_permission();

INSERT INTO pg_temp.v74_existing_school_ids(id)
SELECT id FROM public.schools ORDER BY id;

DROP TABLE pg_temp.v74_existing_school_ids;
