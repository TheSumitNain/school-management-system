-- V30: a staff principal can only be reactivated when a complete employee
-- profile exists and its employment status is Active. Legacy profile-less
-- accounts remain usable while already active, but must be completed before
-- an administrator can re-enable a disabled login.
CREATE OR REPLACE FUNCTION public.prevent_inactive_staff_principal_activation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.principal_type = 'staff'
       AND NEW.status = 'active'
       AND OLD.status IS DISTINCT FROM NEW.status
       AND NOT EXISTS (
           SELECT 1
             FROM public.staff_profiles profile
            WHERE profile.principal_id = NEW.id
              AND profile.employment_status = 'active'
       ) THEN
        RAISE EXCEPTION 'staff employee profile must be complete and active before login activation'
            USING ERRCODE = '23514',
                  CONSTRAINT = 'staff_profile_must_be_active_before_login';
    END IF;
    RETURN NEW;
END;
$$;
