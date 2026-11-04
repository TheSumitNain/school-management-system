-- V21: persist per-device activity for administrative idle timeout.
-- Existing sessions inherit their creation time; token rotation must preserve it.
ALTER TABLE refresh_sessions
    ADD COLUMN last_activity_at TIMESTAMPTZ NOT NULL DEFAULT created_at;

CREATE INDEX ix_refresh_sessions_active_family_activity
    ON refresh_sessions (family_id, last_activity_at DESC)
    WHERE revoked_at IS NULL;
