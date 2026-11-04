-- V19: allow Platform Admins to explicitly delegate guardian permissions
-- to a school-specific custom staff role. This does not grant access by itself.
-- Every grant still requires an exact, active delegation ID (V16).
UPDATE public.permissions
   SET is_delegable = TRUE
 WHERE scope_type = 'school'
   AND permission_key IN ('guardians.read', 'guardians.manage')
   AND is_active = TRUE;

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
