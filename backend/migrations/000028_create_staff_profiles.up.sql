-- V28: employee profiles are separate from staff login credentials.
-- Existing staff accounts are intentionally left without a profile row. This
-- preserves their access and history while allowing the UI to identify them as
-- legacy/incomplete until a School Admin supplies the required profile data.

CREATE TABLE public.staff_profiles (
    principal_id BIGINT PRIMARY KEY,
    school_id BIGINT NOT NULL,
    employee_code VARCHAR(32) NOT NULL,
    full_name VARCHAR(160) NOT NULL,
    designation VARCHAR(100) NOT NULL,
    mobile_number VARCHAR(16) NOT NULL,
    email VARCHAR(254),
    joining_date DATE,
    employment_status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (employment_status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_profiles_user_school_fkey
        FOREIGN KEY (principal_id, school_id)
        REFERENCES public.users (principal_id, school_id)
        ON DELETE RESTRICT,

    CONSTRAINT staff_profiles_employee_code_check
        CHECK (employee_code ~ '^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$'),
    CONSTRAINT staff_profiles_full_name_check
        CHECK (length(btrim(full_name)) BETWEEN 1 AND 160),
    CONSTRAINT staff_profiles_designation_check
        CHECK (length(btrim(designation)) BETWEEN 1 AND 100),
    CONSTRAINT staff_profiles_mobile_check
        CHECK (mobile_number ~ '^\+?[0-9]{10,15}$'),
    CONSTRAINT staff_profiles_email_check
        CHECK (email IS NULL OR (length(btrim(email)) BETWEEN 3 AND 254 AND position('@' in email) > 1))
);

CREATE UNIQUE INDEX ux_staff_profiles_school_employee_code
    ON public.staff_profiles (school_id, lower(employee_code));

CREATE INDEX ix_staff_profiles_school_status_name
    ON public.staff_profiles (school_id, employment_status, lower(full_name));

-- Inactivating an employee profile disables the canonical principal, bumps
-- token_version and revokes active refresh sessions in the same transaction.
-- Reactivating a profile does not automatically re-enable a login account.
CREATE OR REPLACE FUNCTION public.disable_staff_login_for_inactive_profile()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.employment_status IS DISTINCT FROM NEW.employment_status
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

CREATE TRIGGER trg_staff_profile_inactive_disables_login
AFTER UPDATE OF employment_status ON public.staff_profiles
FOR EACH ROW
EXECUTE FUNCTION public.disable_staff_login_for_inactive_profile();
