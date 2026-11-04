-- V18: close two lifecycle gaps in the V17 academic foundation.
-- V17 remains immutable; this forward migration hardens section reactivation
-- and makes teacher-assignment records append-only with one-way revocation.

CREATE OR REPLACE FUNCTION public.validate_v17_section()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v18$
BEGIN
    -- Deactivation must remain possible after a class/year is archived.
    -- Re-activation, however, requires the class and academic year to be usable.
    IF NEW.is_active AND NOT EXISTS (
        SELECT 1
          FROM public.classes c
          JOIN public.academic_years ay
            ON ay.id = NEW.academic_year_id
           AND ay.school_id = NEW.school_id
         WHERE c.id = NEW.class_id
           AND c.school_id = NEW.school_id
           AND c.is_active = TRUE
           AND ay.status <> 'closed'
    ) THEN
        RAISE EXCEPTION 'active sections require an active class and a non-closed academic year in the same school';
    END IF;
    RETURN NEW;
END;
$v18$;

DROP TRIGGER IF EXISTS trg_validate_v17_section ON public.sections;
CREATE TRIGGER trg_validate_v17_section
BEFORE INSERT OR UPDATE OF school_id, academic_year_id, class_id, is_active
ON public.sections
FOR EACH ROW EXECUTE FUNCTION public.validate_v17_section();

CREATE OR REPLACE FUNCTION public.guard_v18_assignment_history()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v18$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'teacher assignment history cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.revoked_at IS NOT NULL THEN
            RAISE EXCEPTION 'new teacher assignments must start active';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.section_id IS DISTINCT FROM OLD.section_id
       OR (TG_TABLE_NAME = 'teacher_subject_assignments'
           AND NEW.subject_id IS DISTINCT FROM OLD.subject_id)
       OR NEW.teacher_principal_id IS DISTINCT FROM OLD.teacher_principal_id
       OR NEW.assigned_by_principal_id IS DISTINCT FROM OLD.assigned_by_principal_id
       OR NEW.assigned_at IS DISTINCT FROM OLD.assigned_at THEN
        RAISE EXCEPTION 'teacher assignment identity is immutable';
    END IF;

    IF OLD.revoked_at IS NOT NULL THEN
        RAISE EXCEPTION 'revoked teacher assignment history is immutable';
    END IF;

    IF NEW.revoked_at IS NULL THEN
        RAISE EXCEPTION 'active teacher assignments may only transition to revoked';
    END IF;

    RETURN NEW;
END;
$v18$;

CREATE TRIGGER trg_guard_v18_teacher_subject_assignment_history
BEFORE INSERT OR UPDATE OR DELETE ON public.teacher_subject_assignments
FOR EACH ROW EXECUTE FUNCTION public.guard_v18_assignment_history();

CREATE TRIGGER trg_guard_v18_class_incharge_assignment_history
BEFORE INSERT OR UPDATE OR DELETE ON public.class_incharge_assignments
FOR EACH ROW EXECUTE FUNCTION public.guard_v18_assignment_history();

COMMENT ON FUNCTION public.guard_v18_assignment_history() IS
'Teacher assignment rows are append-only; the only permitted update is one-way revocation.';
