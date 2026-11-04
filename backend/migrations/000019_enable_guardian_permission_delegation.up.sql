-- V19: allow Platform Admins to explicitly delegate guardian permissions
-- to a school-specific custom staff role. This does not grant access by itself.
-- V4 protects permission delegability from runtime UPDATE statements. The
-- migration temporarily removes only that catalogue guard for this reviewed
-- schema change, then restores the exact trigger immediately.
DROP TRIGGER trg_protect_permission_identity_scope ON public.permissions;

UPDATE public.permissions
   SET is_delegable = TRUE
 WHERE scope_type = 'school'
   AND permission_key IN ('guardians.read', 'guardians.manage')
   AND is_active = TRUE;

CREATE TRIGGER trg_protect_permission_identity_scope
BEFORE UPDATE OF permission_key, scope_type, is_delegable ON public.permissions
FOR EACH ROW EXECUTE FUNCTION public.protect_permission_identity_scope();

DO $v19_verify$
BEGIN
    IF (SELECT COUNT(*) FROM public.permissions
         WHERE scope_type = 'school' AND is_active = TRUE
           AND is_delegable = TRUE
           AND permission_key IN ('guardians.read', 'guardians.manage')) <> 2 THEN
        RAISE EXCEPTION 'V19 requires both active guardian permissions';
    END IF;
END;
$v19_verify$;
