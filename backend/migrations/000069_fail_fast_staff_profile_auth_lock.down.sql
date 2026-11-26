-- Restore V68's blocking advisory lock. This rollback also restores
-- the potential lock-order wait when callers bypass the staff repository.
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v68$
BEGIN
    IF NEW.employment_status = 'inactive' THEN
        PERFORM pg_advisory_xact_lock(
            hashtextextended('refresh-principal:' || NEW.principal_id::bigint::text, 0)
        );

        UPDATE public.auth_principals
           SET status = 'disabled',
               token_version = token_version + 1,
               updated_at = NOW()
         WHERE id = NEW.principal_id
           AND principal_type = 'staff'
           AND status <> 'disabled';

        UPDATE public.refresh_sessions
           SET revoked_at = COALESCE(revoked_at, NOW())
         WHERE principal_id = NEW.principal_id
           AND revoked_at IS NULL;
    END IF;
    RETURN NEW;
END;
$v68$;
