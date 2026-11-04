DROP INDEX IF EXISTS ix_refresh_sessions_active_family_activity;
ALTER TABLE refresh_sessions DROP COLUMN IF EXISTS last_activity_at;
