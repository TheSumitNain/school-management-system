-- V20: date-scoped attendance tied to immutable student enrollment history.
CREATE TABLE attendance_records (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    enrollment_id BIGINT NOT NULL,
    attendance_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('present','absent','late','excused')),
    remarks VARCHAR(500),
    marked_by_principal_id BIGINT NOT NULL,
    updated_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT attendance_records_enrollment_school_fkey FOREIGN KEY (enrollment_id, school_id)
        REFERENCES student_enrollments(id, school_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_marker_school_fkey FOREIGN KEY (marked_by_principal_id, school_id)
        REFERENCES users(principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_updater_school_fkey FOREIGN KEY (updated_by_principal_id, school_id)
        REFERENCES users(principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT attendance_records_id_school_unique UNIQUE (id, school_id)
);
CREATE UNIQUE INDEX ux_attendance_records_enrollment_date ON attendance_records(enrollment_id, attendance_date);
CREATE INDEX ix_attendance_records_school_date ON attendance_records(school_id, attendance_date DESC, enrollment_id);
CREATE INDEX ix_attendance_records_school_enrollment_date ON attendance_records(school_id, enrollment_id, attendance_date DESC);

CREATE OR REPLACE FUNCTION public.validate_v20_attendance_record()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, public
AS $v20$
DECLARE v_year_start DATE; v_year_end DATE; v_enrolled_at DATE; v_ended_at DATE;
BEGIN
    IF TG_OP = 'UPDATE' AND (
        NEW.school_id IS DISTINCT FROM OLD.school_id OR
        NEW.enrollment_id IS DISTINCT FROM OLD.enrollment_id OR
        NEW.attendance_date IS DISTINCT FROM OLD.attendance_date OR
        NEW.marked_by_principal_id IS DISTINCT FROM OLD.marked_by_principal_id OR
        NEW.created_at IS DISTINCT FROM OLD.created_at
    ) THEN RAISE EXCEPTION 'attendance identity is immutable'; END IF;

    SELECT ay.starts_on, ay.ends_on, e.enrolled_at, e.ended_at
      INTO v_year_start, v_year_end, v_enrolled_at, v_ended_at
      FROM public.student_enrollments e
      JOIN public.academic_years ay ON ay.id=e.academic_year_id AND ay.school_id=e.school_id
     WHERE e.id=NEW.enrollment_id AND e.school_id=NEW.school_id;
    IF NOT FOUND THEN RAISE EXCEPTION 'attendance enrollment does not exist in this school'; END IF;
    IF NEW.attendance_date < v_year_start OR NEW.attendance_date > v_year_end
       OR NEW.attendance_date < v_enrolled_at
       OR (v_ended_at IS NOT NULL AND NEW.attendance_date > v_ended_at) THEN
        RAISE EXCEPTION 'attendance date is outside the student enrollment period';
    END IF;
    IF NEW.attendance_date > CURRENT_DATE THEN RAISE EXCEPTION 'future attendance cannot be recorded'; END IF;
    RETURN NEW;
END;
$v20$;
CREATE TRIGGER trg_validate_v20_attendance_record BEFORE INSERT OR UPDATE ON attendance_records
FOR EACH ROW EXECUTE FUNCTION public.validate_v20_attendance_record();
COMMENT ON TABLE attendance_records IS 'One attendance status per student enrollment and date; immutable school/enrollment identity and audited writes.';
