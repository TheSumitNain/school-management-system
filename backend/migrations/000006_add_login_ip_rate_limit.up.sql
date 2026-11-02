-- Persist per-source login request limits across API instances.
-- Only a SHA-256 digest of the canonical client IP is stored.
CREATE TABLE auth_login_ip_attempts (
    ip_hash CHAR(64) PRIMARY KEY
        CHECK (ip_hash ~ '^[0-9a-f]{64}$'),
    attempts SMALLINT NOT NULL DEFAULT 1
        CHECK (attempts BETWEEN 1 AND 30),
    window_started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    blocked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_auth_login_ip_attempts_blocked_until
    ON auth_login_ip_attempts (blocked_until)
    WHERE blocked_until IS NOT NULL;
