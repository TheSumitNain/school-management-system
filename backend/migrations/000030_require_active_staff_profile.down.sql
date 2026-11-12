-- Restore the V29 guard, which blocks inactive profiles but permits
-- reactivation of legacy accounts without a profile.
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
