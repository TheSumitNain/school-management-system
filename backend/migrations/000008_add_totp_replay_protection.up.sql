-- V8: persist the last accepted TOTP counter to prevent authenticator-code replay.
ALTER TABLE principal_mfa_credentials
    ADD COLUMN last_accepted_step BIGINT
    CHECK (last_accepted_step IS NULL OR last_accepted_step >= 0);
