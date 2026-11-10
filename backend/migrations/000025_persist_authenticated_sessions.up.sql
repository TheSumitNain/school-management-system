-- V25: authenticated sessions persist until explicit or security-driven revocation.
-- Preserve already-valid sessions; do not resurrect sessions that had expired.
UPDATE refresh_sessions
   SET expires_at = 'infinity'::timestamptz
 WHERE revoked_at IS NULL
   AND expires_at > clock_timestamp();
