-- Persist short-lived, one-time MFA enrollment and login challenges.
-- Only a SHA-256 digest of the opaque challenge token is stored.
CREATE TABLE auth_mfa_challenges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    challenge_hash CHAR(64) NOT NULL UNIQUE
        CHECK (challenge_hash ~ '^[0-9a-f]{64}$'),
    purpose VARCHAR(20) NOT NULL
        CHECK (purpose IN ('enrollment', 'login')),
    expires_at TIMESTAMPTZ NOT NULL,
    attempts SMALLINT NOT NULL DEFAULT 0
        CHECK (attempts BETWEEN 0 AND 5),
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > created_at)
);

CREATE INDEX ix_auth_mfa_challenges_principal
    ON auth_mfa_challenges (principal_id, purpose, created_at DESC);

CREATE INDEX ix_auth_mfa_challenges_expiry
    ON auth_mfa_challenges (expires_at)
    WHERE consumed_at IS NULL;

CREATE OR REPLACE FUNCTION validate_auth_mfa_challenge_principal() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM auth_principals
         WHERE id = NEW.principal_id
           AND principal_type IN ('staff', 'platform_admin')
    ) THEN
        RAISE EXCEPTION 'MFA challenges require a staff or Platform Admin principal';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_validate_auth_mfa_challenge_principal
BEFORE INSERT OR UPDATE OF principal_id ON auth_mfa_challenges
FOR EACH ROW EXECUTE FUNCTION validate_auth_mfa_challenge_principal();
