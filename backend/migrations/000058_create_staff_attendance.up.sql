-- V58: school-scoped staff attendance and dedicated School Admin permissions.
-- This is independent of student attendance_records and never changes student attendance.

INSERT INTO public.permissions (permission_key, description, scope_type, is_delegable)
VALUES
 ('staff.attendance.read', 'Read staff attendance and monthly register', 'school', FALSE),
 ('staff.attendance.mark', 'Mark staff attendance for the current school day', 'school', FALSE),
 ('staff.attendance.update_same_day', 'Update staff attendance on the same school day', 'school', FALSE),
 ('staff.attendance.correct_past', 'Correct historical staff attendance records', 'school', FALSE);

CREATE TABLE public.staff_attendance_records (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    staff_principal_id BIGINT NOT NULL,
    attendance_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL
        CHECK (status IN ('present', 'absent', 'late', 'half_day')),
    remarks VARCHAR(500),
    marked_by_principal_id BIGINT NOT NULL,
    updated_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT staff_attendance_staff_school_fkey
        FOREIGN KEY (staff_principal_id, school_id)
        REFERENCES public.staff_profiles (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_attendance_marker_school_fkey
        FOREIGN KEY (marked_by_principal_id, school_id)
        REFERENCES public.users (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT staff_attendance_updater_school_fkey
        FOREIGN KEY (updated_by_principal_id, school_id)
        REFERENCES public.users (principal_id, school_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_staff_attendance_school_staff_date
        UNIQUE (school_id, staff_principal_id, attendance_date)
);

CREATE OR REPLACE FUNCTION public.validate_staff_attendance_date()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $
DECLARE
    v_school_today DATE := (NOW() AT TIME ZONE 'Asia/Kolkata')::date;
BEGIN
    IF NEW.attendance_date > v_school_today THEN
        RAISE EXCEPTION 'staff attendance cannot be recorded for a future school date'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$;

CREATE TRIGGER trg_validate_staff_attendance_date
BEFORE INSERT OR UPDATE OF attendance_date ON public.staff_attendance_records
FOR EACH ROW
EXECUTE FUNCTION public.validate_staff_attendance_date();

CREATE INDEX ix_staff_attendance_school_date
    ON public.staff_attendance_records (school_id, attendance_date DESC, staff_principal_id);
CREATE INDEX ix_staff_attendance_school_staff_date
    ON public.staff_attendance_records (school_id, staff_principal_id, attendance_date DESC);

-- Existing schools receive the new permissions through a nested trigger call.
-- V4 protects immutable system-role permission bundles from direct writes;
-- invoking the grant function as a trigger preserves that invariant.
-- The V13 provisioning trigger creates the School Admin role for future schools.
-- PostgreSQL runs same-event triggers in name order; z_ ensures this runs after it.
CREATE OR REPLACE FUNCTION public.grant_staff_attendance_permissions()
RETURNS TRIGGER
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
DECLARE
    v_school_admin_role_id BIGINT;
    v_granted_count BIGINT;
BEGIN
    SELECT id INTO v_school_admin_role_id
      FROM public.roles
     WHERE school_id = NEW.id
       AND principal_type = 'staff'
       AND scope_type = 'school'
       AND role_key = 'school_admin'
       AND is_system = TRUE;

    IF v_school_admin_role_id IS NULL THEN
        RAISE EXCEPTION 'School Admin system role missing for new school %', NEW.id;
    END IF;

    INSERT INTO public.role_permissions (role_id, permission_id)
    SELECT v_school_admin_role_id, p.id
      FROM public.permissions p
     WHERE p.permission_key IN (
       'staff.attendance.read',
       'staff.attendance.mark',
       'staff.attendance.update_same_day',
       'staff.attendance.correct_past'
     )
       AND p.scope_type = 'school'
       AND p.is_active = TRUE
    ON CONFLICT DO NOTHING;

    GET DIAGNOSTICS v_granted_count = ROW_COUNT;
    IF v_granted_count <> 4 THEN
        RAISE EXCEPTION 'Expected 4 staff attendance permissions for school %, granted %', NEW.id, v_granted_count;
    END IF;

    RETURN NEW;
END;
$$;

REVOKE ALL ON FUNCTION public.grant_staff_attendance_permissions() FROM PUBLIC;

CREATE TRIGGER trg_z_grant_staff_attendance_permissions
AFTER INSERT ON public.schools
FOR EACH ROW
EXECUTE FUNCTION public.grant_staff_attendance_permissions();

-- Apply the same grant path to schools that already exist at migration time.
CREATE TEMP TABLE v58_existing_school_ids (id BIGINT PRIMARY KEY) ON COMMIT DROP;
CREATE TRIGGER trg_v58_existing_school_grants
AFTER INSERT ON pg_temp.v58_existing_school_ids
FOR EACH ROW EXECUTE FUNCTION public.grant_staff_attendance_permissions();

INSERT INTO pg_temp.v58_existing_school_ids (id)
SELECT id FROM public.schools ORDER BY id;
DROP TABLE pg_temp.v58_existing_school_ids;

COMMENT ON TABLE public.staff_attendance_records IS
    'Daily school-scoped staff attendance. Independent from student attendance_records; one record per staff member per school date.';
