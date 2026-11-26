-- V70: serialize both sides of substitution creation with canonical account disable.
-- Keep the V62 trigger name for compatibility, but extend it to lock and
-- validate original + substitute principals in deterministic ID order.

CREATE OR REPLACE FUNCTION public.validate_v62_substitute_principal()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v62$
DECLARE
    v_active_count INTEGER;
BEGIN
    -- Match the repository's refresh/profile lifecycle lock order: advisory
    -- principal locks first, then canonical principal rows, then V60 profile
    -- row locks. Direct profile SQL uses pg_try_advisory_xact_lock in V69,
    -- so it fails fast instead of deadlocking while holding a profile row.
    PERFORM pg_advisory_xact_lock(hashtextextended('refresh-principal:' || ids.principal_id::text,0))
      FROM (
        SELECT DISTINCT unnest(ARRAY[NEW.original_teacher_principal_id,NEW.substitute_teacher_principal_id]) AS principal_id
         ORDER BY principal_id
      ) ids;

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
$v62$;

DROP TRIGGER IF EXISTS trg_validate_v62_substitute_principal ON public.staff_substitutions;
CREATE TRIGGER trg_validate_v62_substitute_principal
BEFORE INSERT OR UPDATE OF school_id,original_teacher_principal_id,substitute_teacher_principal_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v62_substitute_principal();

-- V60 originally guarded only the substitute principal. Extend the canonical
-- disable boundary to the original teacher as well, and use a stable
-- constraint name so the repository can return a conflict rather than 500.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_principal_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v70$
DECLARE
    v_school_id BIGINT;
BEGIN
    IF NEW.principal_type='staff' AND OLD.status='active' AND NEW.status='disabled' THEN
        SELECT sp.school_id INTO v_school_id
          FROM public.staff_profiles sp
         WHERE sp.principal_id=NEW.id;

        IF v_school_id IS NOT NULL AND EXISTS (
            SELECT 1
              FROM public.staff_substitutions ss
             WHERE ss.school_id=v_school_id
               AND ss.status='active'
               AND ss.cover_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
               AND (ss.original_teacher_principal_id=NEW.id OR ss.substitute_teacher_principal_id=NEW.id)
        ) THEN
            RAISE EXCEPTION 'cancel or reassign active substitutions before disabling this staff account'
                USING ERRCODE='23514', CONSTRAINT='staff_principal_active_substitutions_check';
        END IF;
    END IF;
    RETURN NEW;
END;
$v70$;

-- Give profile-inactivation conflicts the same stable repository error as
-- account disable. The V60 rule already covers both original and substitute.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_staff_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v70$
BEGIN
  IF NEW.employment_status='inactive' AND OLD.employment_status IS DISTINCT FROM NEW.employment_status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.status='active'
          AND ss.cover_date>=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
          AND (ss.original_teacher_principal_id=NEW.principal_id OR ss.substitute_teacher_principal_id=NEW.principal_id)
     ) THEN
    RAISE EXCEPTION 'cancel or reassign active substitutions before making this staff member inactive'
      USING ERRCODE='23514', CONSTRAINT='staff_principal_active_substitutions_check';
  END IF;
  RETURN NEW;
END;
$v70$;
