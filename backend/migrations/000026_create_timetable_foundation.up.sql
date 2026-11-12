-- V26: school/year-scoped timetable periods and weekly schedules.
-- A schedule entry is bound to an active teacher-subject assignment so that
-- school, academic year, section, subject, and teacher cannot drift apart.

ALTER TABLE teacher_subject_assignments
    ADD CONSTRAINT teacher_subject_assignments_timetable_scope_unique
    UNIQUE (id, school_id, academic_year_id, section_id, subject_id, teacher_principal_id);

CREATE TABLE timetable_periods (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    period_number INTEGER NOT NULL,
    name VARCHAR(80) NOT NULL,
    starts_at TIME NOT NULL,
    ends_at TIME NOT NULL,
    is_break BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT timetable_periods_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_periods_number_positive CHECK (period_number > 0),
    CONSTRAINT timetable_periods_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 80),
    CONSTRAINT timetable_periods_time_range CHECK (ends_at > starts_at),
    CONSTRAINT timetable_periods_id_scope_unique UNIQUE (id, school_id, academic_year_id),
    CONSTRAINT timetable_periods_number_unique UNIQUE (school_id, academic_year_id, period_number)
);

CREATE INDEX ix_timetable_periods_school_year_order
    ON timetable_periods (school_id, academic_year_id, period_number);

CREATE TABLE timetable_entries (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    section_id BIGINT NOT NULL,
    subject_id BIGINT NOT NULL,
    teacher_principal_id BIGINT NOT NULL,
    teacher_subject_assignment_id BIGINT NOT NULL,
    period_id BIGINT NOT NULL,
    day_of_week SMALLINT NOT NULL,
    room VARCHAR(80),
    created_by_principal_id BIGINT NOT NULL,
    updated_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT timetable_entries_section_scope_fkey
        FOREIGN KEY (section_id, school_id, academic_year_id)
        REFERENCES sections (id, school_id, academic_year_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_subject_school_fkey
        FOREIGN KEY (subject_id, school_id)
        REFERENCES subjects (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_teacher_school_fkey
        FOREIGN KEY (teacher_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_assignment_scope_fkey
        FOREIGN KEY (teacher_subject_assignment_id, school_id, academic_year_id, section_id, subject_id, teacher_principal_id)
        REFERENCES teacher_subject_assignments (id, school_id, academic_year_id, section_id, subject_id, teacher_principal_id)
        ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_period_scope_fkey
        FOREIGN KEY (period_id, school_id, academic_year_id)
        REFERENCES timetable_periods (id, school_id, academic_year_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_creator_school_fkey
        FOREIGN KEY (created_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_updater_school_fkey
        FOREIGN KEY (updated_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT timetable_entries_day_range CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT timetable_entries_room_nonempty CHECK (room IS NULL OR length(btrim(room)) BETWEEN 1 AND 80),
    CONSTRAINT timetable_entries_id_scope_unique UNIQUE (id, school_id)
);

-- A section cannot have two classes in the same weekly slot.
CREATE UNIQUE INDEX ux_timetable_entries_section_slot
    ON timetable_entries (school_id, academic_year_id, section_id, day_of_week, period_id);

-- A teacher cannot be assigned to two sections in the same weekly slot.
CREATE UNIQUE INDEX ux_timetable_entries_teacher_slot
    ON timetable_entries (school_id, academic_year_id, teacher_principal_id, day_of_week, period_id);

CREATE INDEX ix_timetable_entries_school_year_section
    ON timetable_entries (school_id, academic_year_id, section_id, day_of_week, period_id);

CREATE INDEX ix_timetable_entries_teacher_week
    ON timetable_entries (school_id, academic_year_id, teacher_principal_id, day_of_week, period_id);

CREATE OR REPLACE FUNCTION public.guard_v26_teacher_assignment_schedule()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v26$
BEGIN
    IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND EXISTS (
           SELECT 1
             FROM public.timetable_entries te
            WHERE te.teacher_subject_assignment_id = OLD.id
       ) THEN
        RAISE EXCEPTION 'teacher assignment is used by the timetable and cannot be revoked';
    END IF;
    RETURN NEW;
END;
$v26$;

CREATE TRIGGER trg_guard_v26_teacher_assignment_schedule
BEFORE UPDATE OF revoked_at ON public.teacher_subject_assignments
FOR EACH ROW EXECUTE FUNCTION public.guard_v26_teacher_assignment_schedule();

COMMENT ON TABLE timetable_periods IS
'School and academic-year-specific ordered timetable periods; break periods are represented explicitly.';
COMMENT ON TABLE timetable_entries IS
'Weekly section timetable entries bound to an active teacher-subject assignment; unique indexes prevent class and teacher double-booking.';

CREATE OR REPLACE FUNCTION public.validate_v26_timetable_entry()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v26$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM public.teacher_subject_assignments a
          JOIN public.academic_years ay
            ON ay.id = a.academic_year_id AND ay.school_id = a.school_id
          JOIN public.sections s
            ON s.id = a.section_id AND s.school_id = a.school_id
           AND s.academic_year_id = a.academic_year_id
          JOIN public.subjects sub
            ON sub.id = a.subject_id AND sub.school_id = a.school_id
          JOIN public.timetable_periods p
            ON p.id = NEW.period_id AND p.school_id = NEW.school_id
           AND p.academic_year_id = NEW.academic_year_id
          JOIN public.users u
            ON u.principal_id = a.teacher_principal_id AND u.school_id = a.school_id
         WHERE a.id = NEW.teacher_subject_assignment_id
           AND a.school_id = NEW.school_id
           AND a.academic_year_id = NEW.academic_year_id
           AND a.section_id = NEW.section_id
           AND a.subject_id = NEW.subject_id
           AND a.teacher_principal_id = NEW.teacher_principal_id
           AND a.revoked_at IS NULL
           AND ay.status <> 'closed'
           AND s.is_active = TRUE
           AND sub.is_active = TRUE
           AND u.status = 'active'
           AND p.is_break = FALSE
    ) THEN
        RAISE EXCEPTION 'timetable entries require an active teacher assignment, section, subject, teacher, year, and teaching period';
    END IF;
    RETURN NEW;
END;
$v26$;

CREATE TRIGGER trg_validate_v26_timetable_entry
BEFORE INSERT OR UPDATE ON public.timetable_entries
FOR EACH ROW EXECUTE FUNCTION public.validate_v26_timetable_entry();


CREATE OR REPLACE FUNCTION public.prevent_v26_timetable_period_overlap()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v26$
BEGIN
    PERFORM pg_advisory_xact_lock(
        hashtextextended(NEW.school_id::text || ':' || NEW.academic_year_id::text, 0)
    );
    IF EXISTS (
        SELECT 1
          FROM public.timetable_periods p
         WHERE p.school_id = NEW.school_id
           AND p.academic_year_id = NEW.academic_year_id
           AND p.id <> COALESCE(NEW.id, 0)
           AND p.starts_at < NEW.ends_at
           AND p.ends_at > NEW.starts_at
    ) THEN
        RAISE EXCEPTION 'timetable periods cannot overlap within an academic year';
    END IF;
    RETURN NEW;
END;
$v26$;

CREATE TRIGGER trg_prevent_v26_timetable_period_overlap
BEFORE INSERT OR UPDATE OF school_id, academic_year_id, starts_at, ends_at
ON public.timetable_periods
FOR EACH ROW EXECUTE FUNCTION public.prevent_v26_timetable_period_overlap();

