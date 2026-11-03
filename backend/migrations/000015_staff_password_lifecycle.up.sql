-- V15: establish the staff temporary-password lifecycle.
--
-- Existing staff retain their current login state. Accounts created after this
-- migration default to requiring a password change, including the initial
-- School Admin created during school onboarding.
ALTER TABLE public.users
    ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE public.users
    ALTER COLUMN must_change_password SET DEFAULT TRUE;

COMMENT ON COLUMN public.users.must_change_password IS
'When true, the staff principal may only complete the authenticated password-change flow until a new password is set.';
