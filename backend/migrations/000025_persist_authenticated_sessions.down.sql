-- Rollback safety: invalidate persistent sessions before older application
-- code (which enforces finite expiries) is redeployed. Users must sign in again.
UPDATE refresh_sessions
   SET expires_at = clock_timestamp()
 WHERE revoked_at IS NULL
   AND expires_at = 'infinity'::timestamptz;
