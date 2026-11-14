-- V33: year-scoped admission identity is part of immutable enrollment
-- identity, just like the student, year, class, section and enrollment date.
-- Keep the existing one-way active -> terminal status transition unchanged.

CREATE OR REPLACE FUNCTION guard_v17_enrollment_history()
RETURNS trigger
LANGUAGE plpgsql AS $v33$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student enrollment history cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.class_id IS DISTINCT FROM OLD.class_id
       OR NEW.section_id IS DISTINCT FROM OLD.section_id
       OR NEW.class_name_snapshot IS DISTINCT FROM OLD.class_name_snapshot
       OR NEW.section_name_snapshot IS DISTINCT FROM OLD.section_name_snapshot
       OR NEW.admission_number IS DISTINCT FROM OLD.admission_number
       OR NEW.admission_date IS DISTINCT FROM OLD.admission_date
       OR NEW.enrolled_at IS DISTINCT FROM OLD.enrolled_at
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'student enrollment identity is immutable';
    END IF;
    IF OLD.status <> 'active' THEN
        RAISE EXCEPTION 'closed student enrollment history is immutable';
    END IF;
    IF NEW.status = 'active' THEN
        IF NEW.ended_at IS NOT NULL THEN
            RAISE EXCEPTION 'active enrollment cannot have ended_at';
        END IF;
        RETURN NEW;
    END IF;
    IF NEW.ended_at IS NULL THEN
        RAISE EXCEPTION 'active enrollment can only transition to a terminal status with ended_at';
    END IF;
    RETURN NEW;
END;
$v33$;
