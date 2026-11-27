-- V72: make leave approval serialize on the exact slot locks used by
-- dated substitute coverage. V60 hashes school:teacher:ISO-weekday:period.
-- Lock every school period for every weekday in the leave range in a stable
-- order before checking existing active cover. The V60 substitution trigger
-- takes the same per-slot key before V71 checks approved leave.

CREATE OR REPLACE FUNCTION public.guard_v71_leave_approval_against_substitution()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v72$
DECLARE
  v_slot RECORD;
BEGIN
  IF NEW.status='approved' AND (
       TG_OP='INSERT'
       OR OLD.status IS DISTINCT FROM NEW.status
       OR OLD.start_date IS DISTINCT FROM NEW.start_date
       OR OLD.end_date IS DISTINCT FROM NEW.end_date
       OR OLD.staff_principal_id IS DISTINCT FROM NEW.staff_principal_id
       OR OLD.school_id IS DISTINCT FROM NEW.school_id
     ) THEN
    FOR v_slot IN
      SELECT DISTINCT
             EXTRACT(ISODOW FROM calendar.cover_day)::integer AS day_of_week,
             tp.id AS period_id
        FROM generate_series(
               NEW.start_date::timestamp,
               NEW.end_date::timestamp,
               interval '1 day'
             ) AS calendar(cover_day)
        CROSS JOIN public.timetable_periods tp
       WHERE tp.school_id=NEW.school_id
       ORDER BY day_of_week,tp.id
    LOOP
      PERFORM pg_advisory_xact_lock(hashtextextended(
        NEW.school_id::text||':'||NEW.staff_principal_id::text||':'||
        v_slot.day_of_week::text||':'||v_slot.period_id::text,0));
    END LOOP;

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
$v72$;
