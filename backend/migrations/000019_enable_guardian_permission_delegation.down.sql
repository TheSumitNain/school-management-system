-- V19 down: refuse to remove delegability after any guardian authority
-- has been delegated. V4/V16 delegation history is append-only.
DO $v19_preflight$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.role_permission_delegations d
          JOIN public.permissions p ON p.id = d.permission_id
         WHERE p.permission_key IN ('guardians.read', 'guardians.manage')
    ) THEN
        RAISE EXCEPTION
            'cannot roll back V19 after guardian permission delegation history exists; revoke grants and preserve the audit history';
    END IF;
END;
$v19_preflight$;

DROP TRIGGER trg_protect_permission_identity_scope ON public.permissions;

UPDATE public.permissions
   SET is_delegable = FALSE
 WHERE scope_type = 'school'
   AND permission_key IN ('guardians.read', 'guardians.manage');

CREATE TRIGGER trg_protect_permission_identity_scope
BEFORE UPDATE OF permission_key, scope_type, is_delegable ON public.permissions
FOR EACH ROW EXECUTE FUNCTION public.protect_permission_identity_scope();
