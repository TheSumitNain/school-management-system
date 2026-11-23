-- V61: serialize substitute assignment and recurring timetable writes before conflict validation.
-- Lock ordering matches the timetable path: timetable entry row first, then
-- the shared teacher/weekday/period advisory lock. This avoids a check/insert
-- race without introducing a row-lock/advisory-lock inversion.

CREATE OR REPLACE FUNCTION public.lock_v61_staff_substitution_slot()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v61$
DECLARE
  v_day_of_week INTEGER;
  v_period_id BIGINT;
BEGIN
  IF NEW.timetable_entry_id IS NULL OR NEW.school_id IS NULL
     OR NEW.substitute_teacher_principal_id IS NULL OR NEW.cover_date IS NULL THEN
    RETURN NEW;
  END IF;

  SELECT te.day_of_week,te.period_id
    INTO v_day_of_week,v_period_id
    FROM public.timetable_entries te
   WHERE te.id=NEW.timetable_entry_id
     AND te.school_id=NEW.school_id
   FOR UPDATE;

  IF NOT FOUND THEN
    RETURN NEW; -- V60 validation reports the authoritative not-found error.
  END IF;

  PERFORM pg_advisory_xact_lock(hashtextextended(
    NEW.school_id::text||':'||NEW.substitute_teacher_principal_id::text||':'||
    v_day_of_week::text||':'||v_period_id::text,0));
  RETURN NEW;
END;
$v61$;

CREATE TRIGGER trg_lock_v61_staff_substitution_slot
BEFORE INSERT OR UPDATE OF school_id,timetable_entry_id,cover_date,substitute_teacher_principal_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.lock_v61_staff_substitution_slot();

COMMENT ON FUNCTION public.lock_v61_staff_substitution_slot() IS
'Acquires the same teacher/weekday/period advisory lock as timetable writes before V60 conflict validation.';
