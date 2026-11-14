-- V36: a guardian relationship may be ended and later re-established.
-- Keep every revoked row as history while allowing only one active link per pair.

ALTER TABLE student_guardians
    DROP CONSTRAINT IF EXISTS student_guardians_unique;

CREATE UNIQUE INDEX ux_student_guardians_active_pair
    ON student_guardians (student_id, guardian_id)
    WHERE revoked_at IS NULL;

COMMENT ON INDEX ux_student_guardians_active_pair IS
    'Prevents duplicate active student/guardian links while preserving multiple revoked relationship periods.';
