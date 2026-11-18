-- V50: allow only one active guardian link per relationship type for each student.
-- Preserve existing duplicate rows; enforce this invariant for future writes.
CREATE OR REPLACE FUNCTION enforce_single_active_guardian_relationship_type()
RETURNS TRIGGER LANGUAGE plpgsql AS $v50$
BEGIN
    IF TG_OP = 'UPDATE'
       AND LOWER(BTRIM(NEW.relationship_type)) IS NOT DISTINCT FROM LOWER(BTRIM(OLD.relationship_type))
       AND NEW.revoked_at IS NOT DISTINCT FROM OLD.revoked_at THEN
        RETURN NEW;
    END IF;
    IF NEW.revoked_at IS NOT NULL THEN
        RETURN NEW;
    END IF;
    IF LOWER(BTRIM(NEW.relationship_type)) NOT IN (
        'father', 'mother', 'grandfather', 'grandmother',
        'brother', 'sister', 'uncle', 'aunt'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514',
            MESSAGE = 'guardian relationship type is not allowed';
    END IF;
    NEW.relationship_type := CASE LOWER(BTRIM(NEW.relationship_type))
        WHEN 'father' THEN 'Father'
        WHEN 'mother' THEN 'Mother'
        WHEN 'grandfather' THEN 'Grandfather'
        WHEN 'grandmother' THEN 'Grandmother'
        WHEN 'brother' THEN 'Brother'
        WHEN 'sister' THEN 'Sister'
        WHEN 'uncle' THEN 'Uncle'
        WHEN 'aunt' THEN 'Aunt'
    END;
    PERFORM 1 FROM students
     WHERE id = NEW.student_id AND school_id = NEW.school_id
     FOR UPDATE;
    IF EXISTS (
        SELECT 1 FROM student_guardians existing
         WHERE existing.school_id = NEW.school_id
           AND existing.student_id = NEW.student_id
           AND LOWER(BTRIM(existing.relationship_type)) = LOWER(BTRIM(NEW.relationship_type))
           AND existing.revoked_at IS NULL
           AND existing.id IS DISTINCT FROM NEW.id
    ) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0001',
            MESSAGE = 'active guardian relationship type already exists for this student';
    END IF;
    RETURN NEW;
END;
$v50$;
CREATE TRIGGER trg_single_active_guardian_relationship_type
BEFORE INSERT OR UPDATE OF relationship_type, revoked_at ON student_guardians
FOR EACH ROW EXECUTE FUNCTION enforce_single_active_guardian_relationship_type();
