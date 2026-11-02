-- V5: database-backed staff login lockout and append-only security events.
-- Username identifiers are stored only as SHA-256 digests, never as raw values.
CREATE TABLE auth_login_attempts (
    identifier_hash CHAR(64) PRIMARY KEY
        CHECK (identifier_hash ~ '^[0-9a-f]{64}$'),
    failed_attempts SMALLINT NOT NULL DEFAULT 0
        CHECK (failed_attempts BETWEEN 0 AND 5),
    window_started_at TIMESTAMPTZ NOT NULL,
    last_attempt_at TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (locked_until IS NULL OR locked_until >= window_started_at)
);
CREATE INDEX ix_auth_login_attempts_locked_until
    ON auth_login_attempts (locked_until)
    WHERE locked_until IS NOT NULL;

COMMENT ON TABLE auth_login_attempts IS
    'Rolling staff-login failure counters keyed by a SHA-256 digest of the normalized username.';

