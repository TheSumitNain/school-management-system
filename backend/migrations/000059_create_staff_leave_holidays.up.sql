-- V59: staff leave requests, leave types and school holiday calendar.
-- Leave and holidays remain separate from staff attendance records.
INSERT INTO public.permissions (permission_key, description, scope_type, is_delegable)
VALUES
 ('staff.leave.read', 'Read staff leave types and requests', 'school', FALSE),
 ('staff.leave.manage_types', 'Create and update staff leave types', 'school', FALSE),
 ('staff.leave.manage_requests', 'Create and cancel staff leave requests', 'school', FALSE),
 ('staff.leave.decide', 'Approve or reject staff leave requests', 'school', FALSE),
 ('staff.holidays.read', 'Read school holiday calendar', 'school', FALSE),
 ('staff.holidays.manage', 'Create and update school holidays', 'school', FALSE);

CREATE TABLE public.staff_leave_types (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    is_paid BOOLEAN NOT NULL,
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_principal_id BIGINT NOT NULL,
    updated_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_staff_leave_type_school_name UNIQUE (school_id, name),
    CONSTRAINT uq_staff_leave_type_id_school UNIQUE (id, school_id),
    CONSTRAINT staff_leave_type_creator_school_fkey
      FOREIGN KEY (created_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT staff_leave_type_updater_school_fkey
      FOREIGN KEY (updated_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT ck_staff_leave_type_name CHECK (length(btrim(name)) BETWEEN 2 AND 100)
);

CREATE TABLE public.staff_leave_requests (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    staff_principal_id BIGINT NOT NULL,
    leave_type_id BIGINT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    duration_type VARCHAR(10) NOT NULL CHECK (duration_type IN ('full_day','half_day')),
    half_day_session VARCHAR(10),
    reason VARCHAR(1000) NOT NULL,
    status VARCHAR(12) NOT NULL DEFAULT 'pending'
      CHECK (status IN ('pending','approved','rejected','cancelled')),
    decision_comment VARCHAR(1000),
    decided_by_principal_id BIGINT,
    decided_at TIMESTAMPTZ,
    created_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_staff_leave_request_staff_school
      FOREIGN KEY (staff_principal_id, school_id) REFERENCES public.staff_profiles(principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT fk_staff_leave_request_type_school
      FOREIGN KEY (leave_type_id, school_id) REFERENCES public.staff_leave_types(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT fk_staff_leave_request_creator_school
      FOREIGN KEY (created_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT fk_staff_leave_request_decider_school
      FOREIGN KEY (decided_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT ck_staff_leave_request_dates CHECK (start_date <= end_date),
    CONSTRAINT ck_staff_leave_request_half_day CHECK (
      (duration_type = 'full_day' AND half_day_session IS NULL)
      OR (duration_type = 'half_day' AND start_date = end_date AND half_day_session IN ('first_half','second_half'))
    ),
    CONSTRAINT ck_staff_leave_request_reason CHECK (length(btrim(reason)) BETWEEN 1 AND 1000),
    CONSTRAINT ck_staff_leave_request_decision CHECK (
      (status IN ('pending','cancelled') AND decided_by_principal_id IS NULL AND decided_at IS NULL)
      OR (status IN ('approved','rejected') AND decided_by_principal_id IS NOT NULL AND decided_at IS NOT NULL)
    ),
    CONSTRAINT ck_staff_leave_rejection_comment CHECK (status <> 'rejected' OR length(btrim(COALESCE(decision_comment,''))) > 0)
);
CREATE INDEX ix_staff_leave_requests_school_status_dates
  ON public.staff_leave_requests(school_id,status,start_date,end_date);
CREATE INDEX ix_staff_leave_requests_staff_dates
  ON public.staff_leave_requests(school_id,staff_principal_id,start_date,end_date);

CREATE TABLE public.school_holidays (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES public.schools(id) ON DELETE RESTRICT,
    holiday_date DATE NOT NULL,
    name VARCHAR(160) NOT NULL,
    description VARCHAR(1000),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by_principal_id BIGINT NOT NULL,
    updated_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_school_holiday_creator_school
      FOREIGN KEY (created_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT fk_school_holiday_updater_school
      FOREIGN KEY (updated_by_principal_id, school_id) REFERENCES public.users(principal_id, school_id),
    CONSTRAINT ck_school_holiday_name CHECK (length(btrim(name)) BETWEEN 2 AND 160),
    CONSTRAINT uq_school_holiday_date_name UNIQUE (school_id,holiday_date,name)
);
CREATE INDEX ix_school_holidays_school_date ON public.school_holidays(school_id,holiday_date) WHERE is_active=TRUE;

CREATE OR REPLACE FUNCTION public.grant_staff_leave_holiday_permissions()
RETURNS TRIGGER LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public
AS $v59_grant_staff_leave_holiday_permissions$
DECLARE v_role_id BIGINT; v_count BIGINT;
BEGIN
 SELECT id INTO v_role_id FROM public.roles
 WHERE school_id=NEW.id AND principal_type='staff' AND scope_type='school'
   AND role_key='school_admin' AND is_system=TRUE;
 IF v_role_id IS NULL THEN RAISE EXCEPTION 'School Admin system role missing for school %',NEW.id; END IF;
 INSERT INTO public.role_permissions(role_id,permission_id)
 SELECT v_role_id,p.id FROM public.permissions p
 WHERE p.permission_key IN ('staff.leave.read','staff.leave.manage_types','staff.leave.manage_requests',
  'staff.leave.decide','staff.holidays.read','staff.holidays.manage')
 AND p.scope_type='school' AND p.is_active=TRUE ON CONFLICT DO NOTHING;
 GET DIAGNOSTICS v_count=ROW_COUNT;
 IF v_count<>6 THEN RAISE EXCEPTION 'Expected 6 leave/holiday permissions for school %, granted %',NEW.id,v_count; END IF;
 RETURN NEW;
END;
$v59_grant_staff_leave_holiday_permissions$;
REVOKE ALL ON FUNCTION public.grant_staff_leave_holiday_permissions() FROM PUBLIC;
CREATE TRIGGER trg_zz_grant_staff_leave_holiday_permissions
AFTER INSERT ON public.schools FOR EACH ROW
EXECUTE FUNCTION public.grant_staff_leave_holiday_permissions();

CREATE TEMP TABLE v59_existing_school_ids(id BIGINT PRIMARY KEY) ON COMMIT DROP;
CREATE TRIGGER trg_v59_existing_school_grants AFTER INSERT ON pg_temp.v59_existing_school_ids
FOR EACH ROW EXECUTE FUNCTION public.grant_staff_leave_holiday_permissions();
INSERT INTO pg_temp.v59_existing_school_ids(id) SELECT id FROM public.schools ORDER BY id;
DROP TABLE pg_temp.v59_existing_school_ids;
