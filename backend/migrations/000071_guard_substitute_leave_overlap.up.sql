-- V71: serialize leave approval and dated substitute coverage for the same staff member.
-- A teacher cannot be scheduled as a substitute on a date covered by approved leave.
-- Both write paths take the same principal advisory lock, closing the concurrent
-- approval-vs-assignment race in either order.

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

CREATE TRIGGER trg_guard_v71_leave_approval_against_substitution
BEFORE INSERT OR UPDATE OF status,start_date,end_date,staff_principal_id,school_id ON public.staff_leave_requests
FOR EACH ROW EXECUTE FUNCTION public.guard_v71_leave_approval_against_substitution();

-- V62 locks both teachers in deterministic principal-ID order. Run this
-- validation after that lock has been acquired so leave approval and coverage
-- creation observe one another without introducing a second lock order.
CREATE OR REPLACE FUNCTION public.guard_v71_substitution_against_approved_leave()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v71$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM public.staff_leave_requests lr
     WHERE lr.school_id=NEW.school_id
       AND lr.staff_principal_id=NEW.substitute_teacher_principal_id
       AND lr.status='approved'
       AND lr.start_date<=NEW.cover_date
       AND lr.end_date>=NEW.cover_date
  ) THEN
    RAISE EXCEPTION 'substitute has approved leave on the proposed cover date'
      USING ERRCODE='23505', CONSTRAINT='staff_substitution_leave_overlap';
  END IF;
  RETURN NEW;
END;
$v71$;

CREATE TRIGGER trg_validate_v71_substitute_leave
BEFORE INSERT OR UPDATE OF school_id,cover_date,substitute_teacher_principal_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.guard_v71_substitution_against_approved_leave();
