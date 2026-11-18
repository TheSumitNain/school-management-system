-- V50 down: remove the active relationship-type guard without changing data.
DROP TRIGGER IF EXISTS trg_single_active_guardian_relationship_type ON student_guardians;
DROP FUNCTION IF EXISTS enforce_single_active_guardian_relationship_type();
