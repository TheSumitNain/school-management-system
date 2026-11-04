-- V21: persist per-device activity for administrative idle timeout.
-- Existing sessions inherit their creation time; token rotation must preserve it.
ALTER TABLE refresh_sessions
    ADD COLUMN last_activity_at TIMESTAMPTZ;

UPDATE refresh_sessions
   SET last_activity_at = created_at
 WHERE last_activity_at IS NULL;

ALTER TABLE refresh_sessions
    ALTER COLUMN last_activity_at SET NOT NULL;

CREATE INDEX ix_refresh_sessions_active_family_activity
    ON refresh_sessions (family_id, last_activity_at DESC)
    WHERE revoked_at IS NULL;
