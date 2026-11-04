-- V17: configurable school academics with year-scoped sections, immutable
-- enrollment history, and school-bound teacher assignments.
-- Class labels are data, not an application enum or a fixed grade catalogue.

CREATE TABLE academic_years (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    name VARCHAR(80) NOT NULL,
    starts_on DATE NOT NULL,
    ends_on DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'planned'
        CHECK (status IN ('planned', 'active', 'closed')),
    is_current BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT academic_years_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 80),
    CONSTRAINT academic_years_date_range CHECK (ends_on > starts_on),
    CONSTRAINT academic_years_current_active CHECK (NOT is_current OR status = 'active'),
    CONSTRAINT academic_years_id_school_unique UNIQUE (id, school_id)
);
CREATE UNIQUE INDEX ux_academic_years_school_name
    ON academic_years (school_id, lower(btrim(name)));
CREATE UNIQUE INDEX ux_academic_years_one_current
    ON academic_years (school_id) WHERE is_current;
CREATE INDEX ix_academic_years_school_dates
    ON academic_years (school_id, starts_on DESC, ends_on DESC);

CREATE TABLE classes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    name VARCHAR(100) NOT NULL,
    normalized_name VARCHAR(100) GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT classes_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 100),
    CONSTRAINT classes_display_order_nonnegative CHECK (display_order >= 0),
    CONSTRAINT classes_id_school_unique UNIQUE (id, school_id)
);
CREATE UNIQUE INDEX ux_classes_school_name ON classes (school_id, normalized_name);
CREATE INDEX ix_classes_school_order ON classes (school_id, is_active, display_order, id);

CREATE TABLE sections (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    class_id BIGINT NOT NULL,
    name VARCHAR(40) NOT NULL,
    normalized_name VARCHAR(40) GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    display_order INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sections_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 40),
    CONSTRAINT sections_display_order_nonnegative CHECK (display_order >= 0),
    CONSTRAINT sections_academic_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT sections_class_school_fkey
        FOREIGN KEY (class_id, school_id)
        REFERENCES classes (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT sections_id_scope_unique UNIQUE (id, school_id, academic_year_id, class_id),
    CONSTRAINT sections_id_year_school_unique UNIQUE (id, school_id, academic_year_id)
);
CREATE UNIQUE INDEX ux_sections_class_year_name
    ON sections (school_id, academic_year_id, class_id, normalized_name);
CREATE INDEX ix_sections_school_year_class
    ON sections (school_id, academic_year_id, class_id, is_active, display_order);

CREATE TABLE subjects (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    name VARCHAR(120) NOT NULL,
    normalized_name VARCHAR(120) GENERATED ALWAYS AS (lower(btrim(name))) STORED,
    subject_code VARCHAR(40),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subjects_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    CONSTRAINT subjects_code_nonempty CHECK (subject_code IS NULL OR length(btrim(subject_code)) BETWEEN 1 AND 40),
    CONSTRAINT subjects_id_school_unique UNIQUE (id, school_id)
);
CREATE UNIQUE INDEX ux_subjects_school_name ON subjects (school_id, normalized_name);
CREATE UNIQUE INDEX ux_subjects_school_code ON subjects (school_id, lower(btrim(subject_code)))
    WHERE subject_code IS NOT NULL;
CREATE INDEX ix_subjects_school_active ON subjects (school_id, is_active, normalized_name);

CREATE TABLE student_enrollments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    student_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    class_id BIGINT NOT NULL,
    section_id BIGINT NOT NULL,
    roll_number VARCHAR(30),
    class_name_snapshot VARCHAR(100) NOT NULL,
    section_name_snapshot VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'promoted', 'transferred', 'withdrawn')),
    enrolled_at DATE NOT NULL DEFAULT CURRENT_DATE,
    ended_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT student_enrollments_end_date CHECK (ended_at IS NULL OR ended_at >= enrolled_at),
    CONSTRAINT student_enrollments_active_end CHECK (
        (status = 'active' AND ended_at IS NULL) OR
        (status <> 'active' AND ended_at IS NOT NULL)
    ),
    CONSTRAINT student_enrollments_student_school_fkey
        FOREIGN KEY (student_id, school_id)
        REFERENCES students (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollments_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollments_section_scope_fkey
        FOREIGN KEY (section_id, school_id, academic_year_id, class_id)
        REFERENCES sections (id, school_id, academic_year_id, class_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollments_id_school_unique UNIQUE (id, school_id)
);
CREATE UNIQUE INDEX ux_student_enrollments_student_year
    ON student_enrollments (student_id, academic_year_id);
CREATE UNIQUE INDEX ux_student_enrollments_active_roll
    ON student_enrollments (school_id, section_id, lower(btrim(roll_number)))
    WHERE status = 'active' AND roll_number IS NOT NULL;
CREATE INDEX ix_student_enrollments_school_section
    ON student_enrollments (school_id, academic_year_id, section_id, status);
CREATE INDEX ix_student_enrollments_student_history
    ON student_enrollments (student_id, enrolled_at DESC, id DESC);

CREATE TABLE teacher_subject_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    section_id BIGINT NOT NULL,
    subject_id BIGINT NOT NULL,
    teacher_principal_id BIGINT NOT NULL,
    assigned_by_principal_id BIGINT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT teacher_subject_assignments_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT teacher_subject_assignments_section_scope_fkey
        FOREIGN KEY (section_id, school_id, academic_year_id)
        REFERENCES sections (id, school_id, academic_year_id) ON DELETE RESTRICT,
    CONSTRAINT teacher_subject_assignments_subject_school_fkey
        FOREIGN KEY (subject_id, school_id)
        REFERENCES subjects (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT teacher_subject_assignments_teacher_school_fkey
        FOREIGN KEY (teacher_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT teacher_subject_assignments_actor_school_fkey
        FOREIGN KEY (assigned_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT teacher_subject_assignments_revocation_check
        CHECK (revoked_at IS NULL OR revoked_at >= assigned_at)
);
CREATE UNIQUE INDEX ux_teacher_subject_assignment_active
    ON teacher_subject_assignments (school_id, academic_year_id, section_id, subject_id, teacher_principal_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_teacher_subject_assignments_teacher
    ON teacher_subject_assignments (teacher_principal_id, school_id, academic_year_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_teacher_subject_assignments_section
    ON teacher_subject_assignments (school_id, academic_year_id, section_id)
    WHERE revoked_at IS NULL;

CREATE TABLE class_incharge_assignments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    section_id BIGINT NOT NULL,
    teacher_principal_id BIGINT NOT NULL,
    assigned_by_principal_id BIGINT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT class_incharge_assignments_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_incharge_assignments_section_scope_fkey
        FOREIGN KEY (section_id, school_id, academic_year_id)
        REFERENCES sections (id, school_id, academic_year_id) ON DELETE RESTRICT,
    CONSTRAINT class_incharge_assignments_teacher_school_fkey
        FOREIGN KEY (teacher_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_incharge_assignments_actor_school_fkey
        FOREIGN KEY (assigned_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_incharge_assignments_revocation_check
        CHECK (revoked_at IS NULL OR revoked_at >= assigned_at)
);
CREATE UNIQUE INDEX ux_class_incharge_one_active_per_section
    ON class_incharge_assignments (section_id) WHERE revoked_at IS NULL;
CREATE INDEX ix_class_incharge_teacher
    ON class_incharge_assignments (teacher_principal_id, school_id, academic_year_id)
    WHERE revoked_at IS NULL;

-- Enforce active staff identities at the database boundary, not merely in UI code.
CREATE OR REPLACE FUNCTION validate_v17_teacher_assignment()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v17$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM public.auth_principals ap
          JOIN public.users u ON u.principal_id = ap.id
         WHERE ap.id = NEW.teacher_principal_id
           AND ap.principal_type = 'staff'
           AND ap.status = 'active'
           AND u.status = 'active'
           AND u.school_id = NEW.school_id
           AND EXISTS (
               SELECT 1
                 FROM public.principal_roles pr
                 JOIN public.roles r ON r.id = pr.role_id
                WHERE pr.principal_id = ap.id
                  AND pr.revoked_at IS NULL
                  AND r.school_id = NEW.school_id
                  AND r.principal_type = 'staff'
                  AND r.scope_type = 'school'
                  AND r.role_key = 'teacher'
                  AND r.is_system = TRUE
           )
    ) THEN
        RAISE EXCEPTION 'teacher assignment requires an active same-school staff principal with the Teacher role';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM public.auth_principals ap
          JOIN public.users u ON u.principal_id = ap.id
         WHERE ap.id = NEW.assigned_by_principal_id
           AND ap.principal_type = 'staff'
           AND ap.status = 'active'
           AND u.status = 'active'
           AND u.school_id = NEW.school_id
    ) THEN
        RAISE EXCEPTION 'assignment actor must be active staff from the same school';
    END IF;

    RETURN NEW;
END;
$v17$;

CREATE TRIGGER trg_validate_v17_teacher_subject_assignment
BEFORE INSERT OR UPDATE OF school_id, teacher_principal_id, assigned_by_principal_id
ON teacher_subject_assignments
FOR EACH ROW EXECUTE FUNCTION validate_v17_teacher_assignment();

CREATE TRIGGER trg_validate_v17_class_incharge_assignment
BEFORE INSERT OR UPDATE OF school_id, teacher_principal_id, assigned_by_principal_id
ON class_incharge_assignments
FOR EACH ROW EXECUTE FUNCTION validate_v17_teacher_assignment();

-- Capture the display labels at enrollment time so later class/section
-- renames never rewrite the student's historical academic record.
CREATE OR REPLACE FUNCTION snapshot_v17_enrollment_labels()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v17$
BEGIN
    SELECT c.name, s.name
      INTO NEW.class_name_snapshot, NEW.section_name_snapshot
      FROM public.classes c
      JOIN public.sections s
        ON s.class_id = c.id
       AND s.school_id = c.school_id
     WHERE c.id = NEW.class_id
       AND c.school_id = NEW.school_id
       AND s.id = NEW.section_id
       AND s.academic_year_id = NEW.academic_year_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'enrollment class and section must belong to the same school and academic year';
    END IF;
    RETURN NEW;
END;
$v17$;

CREATE TRIGGER trg_snapshot_v17_enrollment_labels
BEFORE INSERT ON student_enrollments
FOR EACH ROW EXECUTE FUNCTION snapshot_v17_enrollment_labels();

-- School ownership and academic identity are immutable. Historical enrollment
-- rows may only transition once from active to a terminal status; they cannot
-- be deleted, re-homed, or rewritten into another year/class/section.
CREATE OR REPLACE FUNCTION guard_v17_enrollment_history()
RETURNS trigger
LANGUAGE plpgsql AS $v17$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student enrollment history cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.class_id IS DISTINCT FROM OLD.class_id
       OR NEW.section_id IS DISTINCT FROM OLD.section_id
       OR NEW.roll_number IS DISTINCT FROM OLD.roll_number
       OR NEW.class_name_snapshot IS DISTINCT FROM OLD.class_name_snapshot
       OR NEW.section_name_snapshot IS DISTINCT FROM OLD.section_name_snapshot
       OR NEW.enrolled_at IS DISTINCT FROM OLD.enrolled_at
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'student enrollment identity is immutable';
    END IF;
    IF OLD.status <> 'active' THEN
        RAISE EXCEPTION 'closed student enrollment history is immutable';
    END IF;
    IF NEW.status = 'active' OR NEW.ended_at IS NULL THEN
        RAISE EXCEPTION 'active enrollment can only transition to a terminal status with ended_at';
    END IF;
    RETURN NEW;
END;
$v17$;

CREATE TRIGGER trg_guard_v17_enrollment_history
BEFORE UPDATE OR DELETE ON student_enrollments
FOR EACH ROW EXECUTE FUNCTION guard_v17_enrollment_history();

-- School ownership of all academic catalogue records cannot be changed.
CREATE OR REPLACE FUNCTION guard_v17_school_scope_immutable()
RETURNS trigger
LANGUAGE plpgsql AS $v17$
BEGIN
    IF NEW.school_id IS DISTINCT FROM OLD.school_id THEN
        RAISE EXCEPTION '% school_id is immutable', TG_TABLE_NAME;
    END IF;
    RETURN NEW;
END;
$v17$;

CREATE TRIGGER trg_academic_years_school_scope_immutable
BEFORE UPDATE OF school_id ON academic_years
FOR EACH ROW EXECUTE FUNCTION guard_v17_school_scope_immutable();
CREATE TRIGGER trg_classes_school_scope_immutable
BEFORE UPDATE OF school_id ON classes
FOR EACH ROW EXECUTE FUNCTION guard_v17_school_scope_immutable();
CREATE TRIGGER trg_sections_school_scope_immutable
BEFORE UPDATE OF school_id ON sections
FOR EACH ROW EXECUTE FUNCTION guard_v17_school_scope_immutable();
CREATE TRIGGER trg_subjects_school_scope_immutable
BEFORE UPDATE OF school_id ON subjects
FOR EACH ROW EXECUTE FUNCTION guard_v17_school_scope_immutable();

COMMENT ON TABLE academic_years IS 'School-owned academic periods; only one may be marked current per school.';
COMMENT ON TABLE classes IS 'School-configurable grade labels; no fixed class/grade enumeration is imposed.';
COMMENT ON TABLE sections IS 'Year-specific sections belonging to one school class and academic year.';
COMMENT ON TABLE student_enrollments IS 'Append-only student placement history; promotion creates a new academic-year row.';
COMMENT ON TABLE teacher_subject_assignments IS 'School/year/section/subject-scoped active teacher assignments.';
COMMENT ON TABLE class_incharge_assignments IS 'At most one active class-in-charge teacher per year-specific section.';
