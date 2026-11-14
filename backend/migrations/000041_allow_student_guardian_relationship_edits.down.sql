-- Restore V12 relationship immutability when rolling V41 back.
CREATE OR REPLACE FUNCTION guard_v12_relationship_history()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION '% relationship history cannot be deleted', TG_TABLE_NAME;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.guardian_id IS DISTINCT FROM OLD.guardian_id
       OR NEW.relationship_type IS DISTINCT FROM OLD.relationship_type
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION '% relationship identity is immutable', TG_TABLE_NAME;
    END IF;

    IF OLD.revoked_at IS NOT NULL
       AND (NEW.revoked_at IS DISTINCT FROM OLD.revoked_at
            OR NEW.revoked_by_principal_id IS DISTINCT FROM OLD.revoked_by_principal_id
            OR NEW.revocation_reason IS DISTINCT FROM OLD.revocation_reason) THEN
        RAISE EXCEPTION '% relationship revocation is immutable', TG_TABLE_NAME;
    END IF;

    IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND (NEW.revoked_by_principal_id IS NULL
            OR NEW.revocation_reason IS NULL
            OR length(btrim(NEW.revocation_reason)) = 0) THEN
        RAISE EXCEPTION '% revocation requires an actor and reason', TG_TABLE_NAME;
    END IF;

    IF TG_TABLE_NAME = 'student_guardians'
       AND OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND NOT EXISTS (
           SELECT 1
             FROM auth_principals ap
             JOIN users u ON u.principal_id = ap.id
            WHERE ap.id = NEW.revoked_by_principal_id
              AND ap.principal_type = 'staff'
              AND ap.status = 'active'
              AND u.status = 'active'
              AND u.school_id = NEW.school_id
       ) THEN
        RAISE EXCEPTION 'student-guardian revocation requires active staff from the same school';
    END IF;

    RETURN NEW;
END;
$v12$;
