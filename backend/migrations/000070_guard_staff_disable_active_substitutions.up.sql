-- V70: do not disable a teacher while future/today substitution coverage is active.
-- Lock both sides of a new substitution in principal-ID order. This makes
-- assignment creation serialize with account disable for either teacher.

DROP TRIGGER IF EXISTS trg_validate_v62_substitute_principal ON public.staff_substitutions;

CREATE OR REPLACE FUNCTION public.validate_v70_substitution_principal_pair()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v70$
DECLARE
    v_active_count INTEGER;
BEGIN
    PERFORM ap.id
      FROM public.auth_principals ap
     WHERE ap.id IN (NEW.original_teacher_principal_id,NEW.substitute_teacher_principal_id)
     ORDER BY ap.id
     FOR UPDATE;

    SELECT COUNT(*)::integer
      INTO v_active_count
      FROM public.auth_principals ap
     WHERE ap.id IN (NEW.original_teacher_principal_id,NEW.substitute_teacher_principal_id)
       AND ap.principal_type='staff'
       AND ap.status='active';

    IF v_active_count <> 2 THEN
        RAISE EXCEPTION 'both substitution teachers must have active staff accounts'
            USING ERRCODE='23514', CONSTRAINT='staff_substitution_active_principals_check';
    END IF;
    RETURN NEW;
END;
$v70$;

CREATE TRIGGER trg_validate_v70_substitution_principal_pair
BEFORE INSERT OR UPDATE OF school_id,original_teacher_principal_id,substitute_teacher_principal_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v70_substitution_principal_pair();

CREATE OR REPLACE FUNCTION public.guard_v70_staff_disable_active_substitutions()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v70$
DECLARE
    v_school_id BIGINT;
BEGIN
    IF OLD.principal_type='staff' AND OLD.status='active' AND NEW.status='disabled' THEN
        SELECT sp.school_id INTO v_school_id
          FROM public.staff_profiles sp
         WHERE sp.principal_id=OLD.id;

        IF v_school_id IS NOT NULL AND EXISTS (
            SELECT 1
              FROM public.staff_substitutions ss
             WHERE ss.school_id=v_school_id
               AND ss.status='active'
               AND ss.cover_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
               AND (ss.original_teacher_principal_id=OLD.id OR ss.substitute_teacher_principal_id=OLD.id)
        ) THEN
            RAISE EXCEPTION 'cancel or reassign active substitutions before disabling this staff account'
                USING ERRCODE='23514', CONSTRAINT='staff_principal_active_substitutions_check';
        END IF;
    END IF;
    RETURN NEW;
END;
$v70$;

CREATE TRIGGER trg_guard_v70_staff_disable_active_substitutions
BEFORE UPDATE OF status ON public.auth_principals
FOR EACH ROW EXECUTE FUNCTION public.guard_v70_staff_disable_active_substitutions();
