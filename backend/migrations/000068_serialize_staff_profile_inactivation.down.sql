-- Restore the pre-V68 profile trigger; rollback reintroduces the
-- refresh-rotation race and should only be used as an explicit rollback.
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
AS $v29$
BEGIN
    IF NEW.employment_status = 'inactive' THEN
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
$v29$;
