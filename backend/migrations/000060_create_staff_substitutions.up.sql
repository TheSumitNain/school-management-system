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
  v_class_name VARCHAR(100);
  v_section_name VARCHAR(40);
  v_subject_name VARCHAR(120);
  v_original_teacher_name VARCHAR(160);
  v_substitute_teacher_name VARCHAR(160);
  v_period_number INTEGER;
  v_period_name VARCHAR(80);
  v_period_starts_at TIME;
  v_period_ends_at TIME;
  v_room VARCHAR(80);
BEGIN
  SELECT te.* INTO v_entry
    FROM public.timetable_entries te
   WHERE te.id=NEW.timetable_entry_id AND te.school_id=NEW.school_id
   FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'substitution timetable entry not found' USING ERRCODE='23503';
  END IF;

  v_period_id:=v_entry.period_id;
  v_academic_year_id:=v_entry.academic_year_id;
  IF NOT EXISTS (SELECT 1 FROM public.academic_years ay WHERE ay.id=v_academic_year_id AND ay.school_id=NEW.school_id AND ay.status<>'closed' AND NEW.cover_date BETWEEN ay.starts_on AND ay.ends_on) THEN
    RAISE EXCEPTION 'cover date must fall inside the timetable academic year' USING ERRCODE='23514';
  END IF;
  IF NEW.period_id<>v_period_id THEN
    RAISE EXCEPTION 'substitution period must match the recurring timetable entry' USING ERRCODE='23514';
  END IF;

  IF NEW.original_teacher_principal_id<>v_entry.teacher_principal_id
     OR EXTRACT(ISODOW FROM NEW.cover_date)::integer<>v_entry.day_of_week THEN
    RAISE EXCEPTION 'substitution must match the original teacher and timetable weekday' USING ERRCODE='23514';
  END IF;

  PERFORM 1 FROM public.staff_profiles original_sp
   WHERE original_sp.principal_id=NEW.original_teacher_principal_id
     AND original_sp.school_id=NEW.school_id
     AND original_sp.employment_status='active'
   FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'original teacher must have active employment' USING ERRCODE='23514';
  END IF;

  PERFORM 1
    FROM public.staff_profiles sp
    JOIN public.users u ON u.principal_id=sp.principal_id AND u.school_id=sp.school_id
   WHERE sp.principal_id=NEW.substitute_teacher_principal_id
     AND sp.school_id=NEW.school_id
     AND sp.employment_status='active'
     AND u.status='active'
   FOR UPDATE OF sp,u;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'substitute must be active staff with an enabled login' USING ERRCODE='23514';
  END IF;

  PERFORM 1 FROM public.teacher_subject_assignments a
   WHERE a.school_id=NEW.school_id
     AND a.academic_year_id=v_academic_year_id
     AND a.section_id=v_entry.section_id
     AND a.subject_id=v_entry.subject_id
     AND a.teacher_principal_id=NEW.substitute_teacher_principal_id
     AND a.revoked_at IS NULL
   FOR UPDATE;
  IF NOT FOUND THEN
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
       AND lr.duration_type='full_day'
       AND lr.start_date<=NEW.cover_date
       AND lr.end_date>=NEW.cover_date
  ) THEN
    RAISE EXCEPTION 'leave reference must be approved for the original teacher and cover date' USING ERRCODE='23514';
  END IF;

  SELECT cl.name,sec.name,subj.name,orig.full_name,subsp.full_name,
         p.period_number,p.name,p.starts_at,p.ends_at,te.room
    INTO v_class_name,v_section_name,v_subject_name,v_original_teacher_name,v_substitute_teacher_name,
         v_period_number,v_period_name,v_period_starts_at,v_period_ends_at,v_room
    FROM public.timetable_entries te
    JOIN public.sections sec ON sec.id=te.section_id AND sec.school_id=te.school_id
    JOIN public.classes cl ON cl.id=sec.class_id AND cl.school_id=sec.school_id
    JOIN public.subjects subj ON subj.id=te.subject_id AND subj.school_id=te.school_id
    JOIN public.staff_profiles orig ON orig.principal_id=NEW.original_teacher_principal_id AND orig.school_id=NEW.school_id
    JOIN public.staff_profiles subsp ON subsp.principal_id=NEW.substitute_teacher_principal_id AND subsp.school_id=NEW.school_id
    JOIN public.timetable_periods p ON p.id=te.period_id AND p.school_id=te.school_id
   WHERE te.id=NEW.timetable_entry_id AND te.school_id=NEW.school_id;
  NEW.class_name_snapshot:=v_class_name;
  NEW.section_name_snapshot:=v_section_name;
  NEW.subject_name_snapshot:=v_subject_name;
  NEW.original_teacher_name_snapshot:=v_original_teacher_name;
  NEW.substitute_teacher_name_snapshot:=v_substitute_teacher_name;
  NEW.period_number_snapshot:=v_period_number;
  NEW.period_name_snapshot:=v_period_name;
  NEW.period_starts_at_snapshot:=v_period_starts_at;
  NEW.period_ends_at_snapshot:=v_period_ends_at;
  NEW.room_snapshot:=v_room;

  PERFORM pg_advisory_xact_lock(hashtextextended(
    NEW.school_id::text||':'||NEW.substitute_teacher_principal_id::text||':'||
    EXTRACT(ISODOW FROM NEW.cover_date)::integer::text||':'||v_period_id::text,0));
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_validate_v60_staff_substitution
BEFORE INSERT OR UPDATE OF school_id,timetable_entry_id,period_id,cover_date,original_teacher_principal_id,
  substitute_teacher_principal_id,approved_leave_request_id
ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.validate_v60_staff_substitution();

-- A recurring slot cannot be rewritten or removed while dated cover records
-- still refer to it; cancellation retains the history and releases the slot.
CREATE OR REPLACE FUNCTION public.guard_v60_timetable_substitution_history()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF TG_OP='DELETE' AND EXISTS (
    SELECT 1 FROM public.staff_substitutions ss
     WHERE ss.school_id=OLD.school_id AND ss.timetable_entry_id=OLD.id
  ) THEN
    RAISE EXCEPTION 'timetable entry has substitution history and cannot be deleted'
      USING ERRCODE='23514';
  END IF;
  IF TG_OP='UPDATE' AND EXISTS (
    SELECT 1 FROM public.staff_substitutions ss
     WHERE ss.school_id=OLD.school_id
       AND ss.timetable_entry_id=OLD.id
       AND ss.status='active'
  ) THEN
    RAISE EXCEPTION 'cancel active dated substitutions before changing the recurring schedule'
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
         AND EXTRACT(ISODOW FROM ss.cover_date)::integer=NEW.day_of_week
    ) THEN
      RAISE EXCEPTION 'teacher has an active dated substitute assignment in this weekly slot'
        USING ERRCODE='23505', CONSTRAINT='ux_staff_substitutions_substitute_period_active';
    END IF;
  END IF;
  IF TG_OP='DELETE' THEN RETURN OLD; END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_timetable_substitution_update
BEFORE INSERT OR UPDATE OR DELETE ON public.timetable_entries
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_timetable_substitution_history();

-- Substitution records are append-only: only active -> cancelled is allowed.
CREATE OR REPLACE FUNCTION public.guard_v60_staff_substitution_history()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF TG_OP='DELETE' THEN
    RAISE EXCEPTION 'substitution history cannot be deleted; cancel the record instead' USING ERRCODE='23514';
  END IF;
  IF OLD.status<>'active'
     OR NEW.school_id IS DISTINCT FROM OLD.school_id
     OR NEW.timetable_entry_id IS DISTINCT FROM OLD.timetable_entry_id
     OR NEW.period_id IS DISTINCT FROM OLD.period_id
     OR NEW.cover_date IS DISTINCT FROM OLD.cover_date
     OR NEW.original_teacher_principal_id IS DISTINCT FROM OLD.original_teacher_principal_id
     OR NEW.substitute_teacher_principal_id IS DISTINCT FROM OLD.substitute_teacher_principal_id
     OR NEW.approved_leave_request_id IS DISTINCT FROM OLD.approved_leave_request_id
     OR NEW.reason IS DISTINCT FROM OLD.reason
     OR NEW.class_name_snapshot IS DISTINCT FROM OLD.class_name_snapshot
     OR NEW.section_name_snapshot IS DISTINCT FROM OLD.section_name_snapshot
     OR NEW.subject_name_snapshot IS DISTINCT FROM OLD.subject_name_snapshot
     OR NEW.original_teacher_name_snapshot IS DISTINCT FROM OLD.original_teacher_name_snapshot
     OR NEW.substitute_teacher_name_snapshot IS DISTINCT FROM OLD.substitute_teacher_name_snapshot
     OR NEW.period_number_snapshot IS DISTINCT FROM OLD.period_number_snapshot
     OR NEW.period_name_snapshot IS DISTINCT FROM OLD.period_name_snapshot
     OR NEW.period_starts_at_snapshot IS DISTINCT FROM OLD.period_starts_at_snapshot
     OR NEW.period_ends_at_snapshot IS DISTINCT FROM OLD.period_ends_at_snapshot
     OR NEW.room_snapshot IS DISTINCT FROM OLD.room_snapshot
     OR NEW.created_by_principal_id IS DISTINCT FROM OLD.created_by_principal_id
     OR NEW.created_at IS DISTINCT FROM OLD.created_at
     OR NEW.status<>'cancelled' THEN
    RAISE EXCEPTION 'substitution identity and reason are immutable; only cancellation is allowed' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_staff_substitution_history
BEFORE UPDATE OR DELETE ON public.staff_substitutions
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_staff_substitution_history();

-- A substitute's matching assignment cannot be revoked while future cover is active.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_assignment_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
     AND EXISTS (
       SELECT 1
         FROM public.staff_substitutions ss
         JOIN public.timetable_entries te ON te.id=ss.timetable_entry_id AND te.school_id=ss.school_id
        WHERE ss.school_id=OLD.school_id
          AND ss.substitute_teacher_principal_id=OLD.teacher_principal_id
          AND ss.status='active'
          AND ss.cover_date>=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
          AND te.academic_year_id=OLD.academic_year_id
          AND te.section_id=OLD.section_id
          AND te.subject_id=OLD.subject_id
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before revoking the substitute class/subject assignment' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_substitution_assignment_lifecycle
BEFORE UPDATE OF revoked_at ON public.teacher_subject_assignments
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_substitution_assignment_lifecycle();

-- A future cover cannot silently outlive the substitute's employment or login.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_staff_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF NEW.employment_status='inactive' AND OLD.employment_status IS DISTINCT FROM NEW.employment_status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.status='active'
          AND ss.cover_date>=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
          AND (ss.original_teacher_principal_id=NEW.principal_id OR ss.substitute_teacher_principal_id=NEW.principal_id)
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before making this staff member inactive' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_substitution_staff_lifecycle
BEFORE UPDATE OF employment_status ON public.staff_profiles
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_substitution_staff_lifecycle();

CREATE OR REPLACE FUNCTION public.guard_v60_substitution_login_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF NEW.status='disabled' AND OLD.status IS DISTINCT FROM NEW.status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.substitute_teacher_principal_id=NEW.principal_id
          AND ss.status='active'
          AND ss.cover_date>=CURRENT_DATE
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before disabling this substitute login' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_substitution_login_lifecycle
BEFORE UPDATE OF status ON public.users
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_substitution_login_lifecycle();

-- V4 auth_principals is canonical; protect the actual login-disable path.
CREATE OR REPLACE FUNCTION public.guard_v60_substitution_principal_lifecycle()
RETURNS trigger
LANGUAGE plpgsql
SET search_path=pg_catalog,public
AS $v60$
BEGIN
  IF NEW.principal_type='staff' AND NEW.status='disabled' AND OLD.status IS DISTINCT FROM NEW.status
     AND EXISTS (
       SELECT 1 FROM public.staff_substitutions ss
        WHERE ss.school_id=NEW.school_id
          AND ss.substitute_teacher_principal_id=NEW.id
          AND ss.status='active'
          AND ss.cover_date>=(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
     ) THEN
    RAISE EXCEPTION 'cancel future substitutions before disabling this substitute principal' USING ERRCODE='23514';
  END IF;
  RETURN NEW;
END;
$v60$;

CREATE TRIGGER trg_guard_v60_substitution_principal_lifecycle
BEFORE UPDATE OF status ON public.auth_principals
FOR EACH ROW EXECUTE FUNCTION public.guard_v60_substitution_principal_lifecycle();

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
