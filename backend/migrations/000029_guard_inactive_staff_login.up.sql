-- V29: enforce the inactive employee login guard at the identity boundary.
-- This protects every code path that attempts to reactivate a staff principal,
-- not only the Staff Management HTTP handler.

CREATE OR REPLACE FUNCTION public.prevent_inactive_staff_principal_activation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.principal_type = 'staff'
       AND NEW.status = 'active'
       AND OLD.status IS DISTINCT FROM NEW.status
       AND EXISTS (
           SELECT 1
             FROM public.staff_profiles profile
            WHERE profile.principal_id = NEW.id
              AND profile.employment_status = 'inactive'
       ) THEN
        RAISE EXCEPTION 'inactive staff profile cannot have an active login account'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'staff_profile_must_be_active_before_login';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_prevent_inactive_staff_principal_activation
BEFORE UPDATE OF status ON public.auth_principals
FOR EACH ROW
EXECUTE FUNCTION public.prevent_inactive_staff_principal_activation();

-- Also apply the same login disable/revocation behavior when a profile is
-- inserted directly as inactive (rather than transitioned by an UPDATE).
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
AS $$
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
$$;

DROP TRIGGER IF EXISTS trg_staff_profile_inactive_disables_login ON public.staff_profiles;
CREATE TRIGGER trg_staff_profile_inactive_disables_login
AFTER INSERT OR UPDATE OF employment_status ON public.staff_profiles
FOR EACH ROW
EXECUTE FUNCTION public.disable_staff_login_for_inactive_profile();
