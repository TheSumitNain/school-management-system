-- V60: dated substitute-teacher coverage records; recurring timetable entries remain unchanged.
INSERT INTO public.permissions(permission_key,description,scope_type,is_delegable)
VALUES
 ('staff.substitutions.read','Read dated substitute-teacher coverage', 'school', FALSE),
 ('staff.substitutions.manage','Assign and cancel dated substitute-teacher coverage', 'school', FALSE);

ALTER TABLE public.timetable_entries
  ADD CONSTRAINT timetable_entries_id_school_period_unique UNIQUE(id,school_id,period_id);

CREATE TABLE public.staff_substitutions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    timetable_entry_id BIGINT NOT NULL,
    period_id BIGINT NOT NULL,
    cover_date DATE NOT NULL,
    original_teacher_principal_id BIGINT NOT NULL,
    substitute_teacher_principal_id BIGINT NOT NULL,
    approved_leave_request_id BIGINT,
    reason VARCHAR(1000) NOT NULL,
    status VARCHAR(12) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','cancelled')),
    created_by_principal_id BIGINT NOT NULL,
    cancelled_by_principal_id BIGINT,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_substitutions_entry_school_period_fkey
      FOREIGN KEY(timetable_entry_id,school_id,period_id)
      REFERENCES public.timetable_entries(id,school_id,period_id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_original_staff_fkey
      FOREIGN KEY(original_teacher_principal_id,school_id)
      REFERENCES public.staff_profiles(principal_id,school_id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_substitute_staff_fkey
      FOREIGN KEY(substitute_teacher_principal_id,school_id)
      REFERENCES public.staff_profiles(principal_id,school_id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_leave_request_fkey
      FOREIGN KEY(approved_leave_request_id)
      REFERENCES public.staff_leave_requests(id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_creator_fkey
      FOREIGN KEY(created_by_principal_id,school_id)
      REFERENCES public.users(principal_id,school_id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_canceller_fkey
      FOREIGN KEY(cancelled_by_principal_id,school_id)
      REFERENCES public.users(principal_id,school_id) ON DELETE RESTRICT,
    CONSTRAINT staff_substitutions_reason_check
      CHECK(length(btrim(reason)) BETWEEN 1 AND 1000),
    CONSTRAINT staff_substitutions_different_teacher
      CHECK(original_teacher_principal_id <> substitute_teacher_principal_id),
    CONSTRAINT staff_substitutions_cancel_state
      CHECK((status='active' AND cancelled_by_principal_id IS NULL AND cancelled_at IS NULL)
         OR (status='cancelled' AND cancelled_by_principal_id IS NOT NULL AND cancelled_at IS NOT NULL))
);

CREATE UNIQUE INDEX ux_staff_substitutions_entry_date_active
  ON public.staff_substitutions(school_id,timetable_entry_id,cover_date)
  WHERE status='active';
CREATE UNIQUE INDEX ux_staff_substitutions_substitute_period_active
  ON public.staff_substitutions(school_id,substitute_teacher_principal_id,cover_date,period_id)
  WHERE status='active';
CREATE INDEX ix_staff_substitutions_school_date
  ON public.staff_substitutions(school_id,cover_date,status,id);
CREATE INDEX ix_staff_substitutions_original_teacher
  ON public.staff_substitutions(school_id,original_teacher_principal_id,cover_date DESC);

-- The authoritative guard validates date/weekday, active employment and login,
-- matching teaching assignment, existing recurring timetable conflicts, and
-- any optional approved leave reference in the same transaction as the write.
CREATE OR REPLACE FUNCTION public.validate_v60_staff_substitution()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
DECLARE
  v_entry public.timetable_entries%ROWTYPE;
  v_period_id BIGINT;
  v_academic_year_id BIGINT;
BEGIN
  SELECT te.* INTO v_entry
    FROM public.timetable_entries te
   WHERE te.id=NEW.timetable_entry_id AND te.school_id=NEW.school_id;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'substitution timetable entry not found' USING ERRCODE='23503';
  END IF;

  v_period_id:=v_entry.period_id;
  v_academic_year_id:=v_entry.academic_year_id;
  IF NEW.period_id<>v_period_id THEN
    RAISE EXCEPTION 'substitution period must match the recurring timetable entry' USING ERRCODE='23514';
  END IF;

  IF NEW.original_teacher_principal_id<>v_entry.teacher_principal_id
     OR EXTRACT(ISODOW FROM NEW.cover_date)::integer<>v_entry.day_of_week THEN
    RAISE EXCEPTION 'substitution must match the original teacher and timetable weekday' USING ERRCODE='23514';
  END IF;

  IF NOT EXISTS (
    SELECT 1
      FROM public.staff_profiles sp
      JOIN public.users u ON u.principal_id=sp.principal_id AND u.school_id=sp.school_id
     WHERE sp.principal_id=NEW.substitute_teacher_principal_id
       AND sp.school_id=NEW.school_id
       AND sp.employment_status='active'
       AND u.status='active'
  ) THEN
    RAISE EXCEPTION 'substitute must be active staff with an enabled login' USING ERRCODE='23514';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM public.teacher_subject_assignments a
     WHERE a.school_id=NEW.school_id
       AND a.academic_year_id=v_academic_year_id
       AND a.section_id=v_entry.section_id
       AND a.subject_id=v_entry.subject_id
       AND a.teacher_principal_id=NEW.substitute_teacher_principal_id
       AND a.revoked_at IS NULL
  ) THEN
    RAISE EXCEPTION 'substitute requires an active matching class and subject assignment' USING ERRCODE='23514';
  END IF;

  IF EXISTS (
    SELECT 1 FROM public.timetable_entries te
     WHERE te.school_id=NEW.school_id
       AND te.academic_year_id=v_academic_year_id
       AND te.teacher_principal_id=NEW.substitute_teacher_principal_id
       AND te.day_of_week=v_entry.day_of_week
       AND te.period_id=v_period_id
  ) THEN
    RAISE EXCEPTION 'substitute already has a recurring timetable class in this period' USING ERRCODE='23505',
      CONSTRAINT='ux_timetable_entries_teacher_slot';
  END IF;

  IF NEW.approved_leave_request_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM public.staff_leave_requests lr
     WHERE lr.id=NEW.approved_leave_request_id
       AND lr.school_id=NEW.school_id
       AND lr.staff_principal_id=NEW.original_teacher_principal_id
       AND lr.status='approved'
       AND lr.start_date<=NEW.cover_date
       AND lr.end_date>=NEW.cover_date
  ) THEN
    RAISE EXCEPTION 'leave reference must be approved for the original teacher and cover date' USING ERRCODE='23514';
  END IF;

  PERFORM pg_advisory_xact_lock(hashtextextended(
    NEW.school_id::text||':'||NEW.substitute_teacher_principal_id::text||':'||
    NEW.cover_date::text||':'||v_period_id::text,0));
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_validate_v60_staff_substitution
BEFORE INSERT OR UPDATE ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v60_staff_substitution();

-- A recurring slot cannot be rewritten or removed while dated cover records
-- still refer to it; cancellation retains the history and releases the slot.
CREATE OR REPLACE FUNCTION public.guard_v60_timetable_substitution_history()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF EXISTS (
    SELECT 1 FROM public.staff_substitutions ss
     WHERE ss.school_id=OLD.school_id
       AND ss.timetable_entry_id=OLD.id
       AND ss.status='active'
  ) THEN
    RAISE EXCEPTION 'timetable entry has active dated substitutions; cancel them before changing the recurring schedule'
      USING ERRCODE='23514';
  END IF;
  RETURN CASE WHEN TG_OP='DELETE' THEN OLD ELSE NEW END;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_timetable_substitution_update
BEFORE UPDATE OR DELETE ON public.timetable_entries
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_timetable_substitution_history();

CREATE OR REPLACE FUNCTION public.grant_staff_substitution_permissions()
RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public
AS $v60_grant$
DECLARE v_role_id BIGINT; v_count BIGINT;
BEGIN
 SELECT id INTO v_role_id FROM public.roles
  WHERE school_id=NEW.id AND principal_type='staff' AND scope_type='school'
    AND role_key='school_admin' AND is_system=TRUE;
 IF v_role_id IS NULL THEN RAISE EXCEPTION 'School Admin system role missing for school %',NEW.id; END IF;
 INSERT INTO public.role_permissions(role_id,permission_id)
 SELECT v_role_id,p.id FROM public.permissions p
  WHERE p.permission_key IN ('staff.substitutions.read','staff.substitutions.manage')
    AND p.scope_type='school' AND p.is_active=TRUE
 ON CONFLICT DO NOTHING;
 GET DIAGNOSTICS v_count=ROW_COUNT;
 IF v_count<>2 THEN RAISE EXCEPTION 'Expected 2 substitution permissions for school %, granted %',NEW.id,v_count; END IF;
 RETURN NEW;
END;
$v60_grant$;
REVOKE ALL ON FUNCTION public.grant_staff_substitution_permissions() FROM PUBLIC;
CREATE TRIGGER trg_zz_grant_staff_substitution_permissions
AFTER INSERT ON public.schools FOR EACH ROW
EXECUTE FUNCTION public.grant_staff_substitution_permissions();

CREATE TEMP TABLE v60_existing_school_ids(id BIGINT PRIMARY KEY) ON COMMIT DROP;
CREATE TRIGGER trg_v60_existing_school_grants AFTER INSERT ON pg_temp.v60_existing_school_ids
FOR EACH ROW EXECUTE FUNCTION public.grant_staff_substitution_permissions();
INSERT INTO pg_temp.v60_existing_school_ids(id) SELECT id FROM public.schools ORDER BY id;
DROP TABLE pg_temp.v60_existing_school_ids;

COMMENT ON TABLE public.staff_substitutions IS
'Dated temporary coverage for one recurring timetable entry; original weekly timetable teacher is never rewritten.';
