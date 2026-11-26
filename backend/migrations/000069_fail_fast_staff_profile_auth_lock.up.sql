-- V69: avoid lock-order deadlocks when profile rows are updated directly.
-- The repository path takes the principal advisory lock before touching the
-- profile. A direct SQL update has already locked the profile row by the time
-- this AFTER trigger runs; never wait while holding that row lock.
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v69$
BEGIN
    IF NEW.employment_status = 'inactive' THEN
        IF NOT pg_try_advisory_xact_lock(
            hashtextextended('refresh-principal:' || NEW.principal_id::bigint::text, 0)
        ) THEN
            RAISE EXCEPTION 'staff profile update conflicts with an in-flight authentication change; retry'
                USING ERRCODE = '40001',
                      CONSTRAINT = 'staff_profile_auth_lifecycle_serialization';
        END IF;

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
$v69$;
