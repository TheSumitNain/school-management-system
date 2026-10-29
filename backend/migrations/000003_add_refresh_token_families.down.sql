DROP INDEX IF EXISTS refresh_sessions_family_id_idx;

ALTER TABLE refresh_sessions
    DROP COLUMN IF EXISTS family_id;
