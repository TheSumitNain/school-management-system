-- Add a bounded, one-time step-up challenge for recovery-code regeneration.
ALTER TABLE auth_mfa_challenges
    DROP CONSTRAINT auth_mfa_challenges_purpose_check;

ALTER TABLE auth_mfa_challenges
    ADD CONSTRAINT auth_mfa_challenges_purpose_check
    CHECK (purpose IN ('enrollment', 'login', 'recovery_regeneration'));
