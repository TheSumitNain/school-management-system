-- V43 down: remove only the database-level primary-contact invariant.
DROP INDEX IF EXISTS ux_student_guardians_active_primary_per_student;
