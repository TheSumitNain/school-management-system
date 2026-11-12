DROP TRIGGER IF EXISTS trg_prevent_inactive_staff_principal_activation ON public.auth_principals;
DROP FUNCTION IF EXISTS public.prevent_inactive_staff_principal_activation();

-- Restore the V28 update-only profile transition trigger behavior.
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE'
       AND OLD.employment_status IS DISTINCT FROM NEW.employment_status
       AND NEW.employment_status = 'inactive' THEN
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
$$;
DROP TRIGGER IF EXISTS trg_staff_profile_inactive_disables_login ON public.staff_profiles;
CREATE TRIGGER trg_staff_profile_inactive_disables_login
AFTER UPDATE OF employment_status ON public.staff_profiles
FOR EACH ROW
EXECUTE FUNCTION public.disable_staff_login_for_inactive_profile();
