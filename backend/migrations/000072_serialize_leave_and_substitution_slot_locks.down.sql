-- Restore the V71 implementation when rolling V72 back.
CREATE OR REPLACE FUNCTION public.guard_v71_leave_approval_against_substitution()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v71$
BEGIN
  IF NEW.status='approved' AND (
       TG_OP='INSERT'
       OR OLD.status IS DISTINCT FROM NEW.status
       OR OLD.start_date IS DISTINCT FROM NEW.start_date
       OR OLD.end_date IS DISTINCT FROM NEW.end_date
       OR OLD.staff_principal_id IS DISTINCT FROM NEW.staff_principal_id
       OR OLD.school_id IS DISTINCT FROM NEW.school_id
     ) THEN
    PERFORM pg_advisory_xact_lock(
      hashtextextended('refresh-principal:' || NEW.staff_principal_id::bigint::text,0)
    );

    IF EXISTS (
      SELECT 1
        FROM public.staff_substitutions ss
       WHERE ss.school_id=NEW.school_id
         AND ss.substitute_teacher_principal_id=NEW.staff_principal_id
         AND ss.status='active'
         AND ss.cover_date >= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
         AND ss.cover_date BETWEEN NEW.start_date AND NEW.end_date
    ) THEN
      RAISE EXCEPTION 'cancel or reassign active substitute coverage before approving overlapping leave'
        USING ERRCODE='23505', CONSTRAINT='staff_substitution_leave_overlap';
    END IF;
  END IF;
  RETURN NEW;
END;
$v71$;
