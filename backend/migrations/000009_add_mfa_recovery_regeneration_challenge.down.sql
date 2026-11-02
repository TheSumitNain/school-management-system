-- Do not discard active recovery-regeneration challenge history on rollback.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM auth_mfa_challenges
         WHERE purpose = 'recovery_regeneration'
    ) THEN
        RAISE EXCEPTION
            'cannot roll back V9 while recovery-regeneration challenges exist';
    END IF;
END;
$$;

ALTER TABLE auth_mfa_challenges
    DROP CONSTRAINT auth_mfa_challenges_purpose_check;

ALTER TABLE auth_mfa_challenges
    ADD CONSTRAINT auth_mfa_challenges_purpose_check
    CHECK (purpose IN ('enrollment', 'login'));
