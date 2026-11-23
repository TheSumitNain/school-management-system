-- V63: completed dated coverage remains immutable history but no longer
-- blocks future recurring timetable edits or new weekly assignments.

CREATE OR REPLACE FUNCTION public.guard_v60_timetable_substitution_history()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v63$
DECLARE
  v_today_ist DATE := (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date;
BEGIN
  IF TG_OP='DELETE' AND EXISTS (
    SELECT 1 FROM public.staff_substitutions ss
     WHERE ss.school_id=OLD.school_id
       AND ss.timetable_entry_id=OLD.id
  ) THEN
    RAISE EXCEPTION 'timetable entry has substitution history and cannot be deleted'
      USING ERRCODE='23514';
  END IF;

  IF TG_OP='UPDATE' AND EXISTS (
    SELECT 1 FROM public.staff_substitutions ss
     WHERE ss.school_id=OLD.school_id
       AND ss.timetable_entry_id=OLD.id
       AND ss.status='active'
       AND ss.cover_date>=v_today_ist
  ) THEN
    RAISE EXCEPTION 'cancel current or future dated substitutions before changing the recurring schedule'
      USING ERRCODE='23514';
  END IF;

  IF TG_OP IN ('INSERT','UPDATE') THEN
    PERFORM pg_advisory_xact_lock(hashtextextended(
      NEW.school_id::text||':'||NEW.teacher_principal_id::text||':'||
      NEW.day_of_week::text||':'||NEW.period_id::text,0));
    IF EXISTS (
      SELECT 1 FROM public.staff_substitutions ss
       WHERE ss.school_id=NEW.school_id
         AND ss.substitute_teacher_principal_id=NEW.teacher_principal_id
         AND ss.period_id=NEW.period_id
         AND ss.status='active'
         AND ss.cover_date>=v_today_ist
         AND EXTRACT(ISODOW FROM ss.cover_date)::integer=NEW.day_of_week
    ) THEN
      RAISE EXCEPTION 'teacher has an active current or future substitute assignment in this weekly slot'
        USING ERRCODE='23505', CONSTRAINT='ux_staff_substitutions_substitute_period_active';
    END IF;
  END IF;

  IF TG_OP='DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END;
$v63$;

COMMENT ON FUNCTION public.guard_v60_timetable_substitution_history() IS
'Preserves all referenced timetable history; only current/future active covers block recurring schedule changes or weekly-slot reuse.';
