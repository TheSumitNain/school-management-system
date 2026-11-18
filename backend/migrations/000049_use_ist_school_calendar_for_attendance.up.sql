-- V49: use the product's authoritative Asia/Kolkata school calendar date
-- for attendance future-date constraints, independent of the DB session timezone.
CREATE OR REPLACE FUNCTION public.validate_v20_attendance_record()
RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, public
AS $v49$
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
    IF NEW.attendance_date > ((CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date) THEN
        RAISE EXCEPTION 'future attendance cannot be recorded';
    END IF;
    RETURN NEW;
END;
$v49$;

-- Keep the database-enforced date-of-birth boundary on the same school-local day.
ALTER TABLE student_basic_details
    DROP CONSTRAINT student_basic_details_birth_date;

ALTER TABLE student_basic_details
    ADD CONSTRAINT student_basic_details_birth_date
    CHECK (
        date_of_birth IS NULL
        OR date_of_birth <= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date
    );

-- Keep physical measurement dates on the same school-local calendar day.
ALTER TABLE student_physical_details
    ADD CONSTRAINT student_physical_details_measured_on_not_future
    CHECK (measured_on <= (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Kolkata')::date);
