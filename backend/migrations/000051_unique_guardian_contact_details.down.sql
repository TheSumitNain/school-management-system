-- V51 down: remove school-scoped guardian contact uniqueness.
DROP INDEX IF EXISTS ux_guardians_school_email_unique;
DROP INDEX IF EXISTS ux_guardians_school_mobile_unique;
DROP FUNCTION IF EXISTS normalize_guardian_mobile(TEXT);
