-- V44: versioned class/year curriculum and enrollment-specific subject choices.
-- Compulsory subjects are inherited from the curriculum; optional choices are
-- recorded per enrollment and remain append-only when choices are replaced.

CREATE TABLE class_curricula (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    academic_year_id BIGINT NOT NULL,
    class_id BIGINT NOT NULL,
    version_number INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'retired')),
    created_by_principal_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT class_curricula_version_positive CHECK (version_number > 0),
    CONSTRAINT class_curricula_school_year_class_version_unique
        UNIQUE (school_id, academic_year_id, class_id, version_number),
    CONSTRAINT class_curricula_id_school_unique UNIQUE (id, school_id),
    CONSTRAINT class_curricula_year_school_fkey
        FOREIGN KEY (academic_year_id, school_id)
        REFERENCES academic_years (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_curricula_class_school_fkey
        FOREIGN KEY (class_id, school_id)
        REFERENCES classes (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_curricula_creator_school_fkey
        FOREIGN KEY (created_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX ux_class_curricula_one_active
    ON class_curricula (school_id, academic_year_id, class_id)
    WHERE status = 'active';
CREATE INDEX ix_class_curricula_school_year_class
    ON class_curricula (school_id, academic_year_id, class_id, version_number DESC);

CREATE TABLE class_curriculum_optional_groups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    curriculum_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    min_selections SMALLINT NOT NULL DEFAULT 1,
    max_selections SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT class_curriculum_groups_name_nonempty CHECK (length(btrim(name)) BETWEEN 1 AND 100),
    CONSTRAINT class_curriculum_groups_selection_bounds
        CHECK (min_selections >= 0 AND max_selections >= 1 AND min_selections <= max_selections),
    CONSTRAINT class_curriculum_groups_id_curriculum_school_unique
        UNIQUE (id, curriculum_id, school_id),
    CONSTRAINT class_curriculum_groups_curriculum_fkey
        FOREIGN KEY (curriculum_id, school_id)
        REFERENCES class_curricula (id, school_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX ux_class_curriculum_groups_name
    ON class_curriculum_optional_groups (curriculum_id, lower(btrim(name)));
CREATE INDEX ix_class_curriculum_groups_curriculum
    ON class_curriculum_optional_groups (curriculum_id, school_id);

CREATE TABLE class_curriculum_subjects (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    curriculum_id BIGINT NOT NULL,
    subject_id BIGINT NOT NULL,
    optional_group_id BIGINT,
    subject_name_snapshot VARCHAR(120) NOT NULL,
    subject_code_snapshot VARCHAR(40),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT class_curriculum_subjects_name_nonempty
        CHECK (length(btrim(subject_name_snapshot)) BETWEEN 1 AND 120),
    CONSTRAINT class_curriculum_subjects_code_nonempty
        CHECK (subject_code_snapshot IS NULL OR length(btrim(subject_code_snapshot)) BETWEEN 1 AND 40),
    CONSTRAINT class_curriculum_subjects_id_curriculum_school_unique
        UNIQUE (id, curriculum_id, school_id),
    CONSTRAINT class_curriculum_subjects_curriculum_fkey
        FOREIGN KEY (curriculum_id, school_id)
        REFERENCES class_curricula (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_curriculum_subjects_subject_fkey
        FOREIGN KEY (subject_id, school_id)
        REFERENCES subjects (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT class_curriculum_subjects_group_fkey
        FOREIGN KEY (optional_group_id, curriculum_id, school_id)
        REFERENCES class_curriculum_optional_groups (id, curriculum_id, school_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX ux_class_curriculum_subject_once
    ON class_curriculum_subjects (curriculum_id, subject_id);
CREATE INDEX ix_class_curriculum_subjects_group
    ON class_curriculum_subjects (curriculum_id, optional_group_id);

CREATE TABLE student_enrollment_curricula (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    enrollment_id BIGINT NOT NULL,
    curriculum_id BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'replaced')),
    assigned_by_principal_id BIGINT NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT student_enrollment_curricula_status_revocation
        CHECK ((status = 'active' AND revoked_at IS NULL) OR (status = 'replaced' AND revoked_at IS NOT NULL)),
    CONSTRAINT student_enrollment_curricula_id_curriculum_school_unique
        UNIQUE (id, curriculum_id, school_id),
    CONSTRAINT student_enrollment_curricula_enrollment_fkey
        FOREIGN KEY (enrollment_id, school_id)
        REFERENCES student_enrollments (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollment_curricula_curriculum_fkey
        FOREIGN KEY (curriculum_id, school_id)
        REFERENCES class_curricula (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollment_curricula_actor_fkey
        FOREIGN KEY (assigned_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX ux_student_enrollment_curricula_active
    ON student_enrollment_curricula (enrollment_id)
    WHERE status = 'active';
CREATE INDEX ix_student_enrollment_curricula_history
    ON student_enrollment_curricula (enrollment_id, assigned_at DESC, id DESC);

CREATE TABLE student_enrollment_subject_selections (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL,
    enrollment_curriculum_id BIGINT NOT NULL,
    curriculum_id BIGINT NOT NULL,
    curriculum_subject_id BIGINT NOT NULL,
    subject_name_snapshot VARCHAR(120) NOT NULL,
    subject_code_snapshot VARCHAR(40),
    selected_by_principal_id BIGINT NOT NULL,
    selected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    CONSTRAINT student_enrollment_subject_selection_name_nonempty
        CHECK (length(btrim(subject_name_snapshot)) BETWEEN 1 AND 120),
    CONSTRAINT student_enrollment_subject_selection_code_nonempty
        CHECK (subject_code_snapshot IS NULL OR length(btrim(subject_code_snapshot)) BETWEEN 1 AND 40),
    CONSTRAINT student_enrollment_subject_selection_revocation
        CHECK (revoked_at IS NULL OR revoked_at >= selected_at),
    CONSTRAINT student_enrollment_subject_selection_assignment_fkey
        FOREIGN KEY (enrollment_curriculum_id, curriculum_id, school_id)
        REFERENCES student_enrollment_curricula (id, curriculum_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollment_subject_selection_subject_fkey
        FOREIGN KEY (curriculum_subject_id, curriculum_id, school_id)
        REFERENCES class_curriculum_subjects (id, curriculum_id, school_id) ON DELETE RESTRICT,
    CONSTRAINT student_enrollment_subject_selection_actor_fkey
        FOREIGN KEY (selected_by_principal_id, school_id)
        REFERENCES users (principal_id, school_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX ux_student_enrollment_subject_selection_active
    ON student_enrollment_subject_selections (enrollment_curriculum_id, curriculum_subject_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_student_enrollment_subject_selection_history
    ON student_enrollment_subject_selections (enrollment_curriculum_id, selected_at DESC, id DESC);

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'class curriculum versions cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.academic_year_id IS DISTINCT FROM OLD.academic_year_id
       OR NEW.class_id IS DISTINCT FROM OLD.class_id
       OR NEW.version_number IS DISTINCT FROM OLD.version_number
       OR NEW.created_by_principal_id IS DISTINCT FROM OLD.created_by_principal_id
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'class curriculum identity is immutable';
    END IF;
    IF OLD.status = 'retired' AND NEW.status <> OLD.status THEN
        RAISE EXCEPTION 'retired class curriculum cannot be reactivated';
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_guard_v44_class_curriculum
BEFORE UPDATE OR DELETE ON class_curricula
FOR EACH ROW EXECUTE FUNCTION guard_v44_class_curriculum();

CREATE OR REPLACE FUNCTION guard_v44_class_curriculum_child()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
DECLARE
    v_curriculum_id BIGINT;
    v_school_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_curriculum_id := OLD.curriculum_id;
        v_school_id := OLD.school_id;
    ELSIF TG_OP = 'INSERT' THEN
        v_curriculum_id := NEW.curriculum_id;
        v_school_id := NEW.school_id;
    ELSE
        v_curriculum_id := OLD.curriculum_id;
        v_school_id := OLD.school_id;
        IF NEW.id IS DISTINCT FROM OLD.id
           OR NEW.school_id IS DISTINCT FROM OLD.school_id
           OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
           OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
            RAISE EXCEPTION 'class curriculum subject/group identity is immutable';
        END IF;
    END IF;
    IF EXISTS (
        SELECT 1 FROM student_enrollment_curricula ec
        WHERE ec.curriculum_id = v_curriculum_id AND ec.school_id = v_school_id
    ) THEN
        RAISE EXCEPTION 'curriculum subjects and optional groups are immutable after enrollment assignment';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_guard_v44_curriculum_groups
BEFORE INSERT OR UPDATE OR DELETE ON class_curriculum_optional_groups
FOR EACH ROW EXECUTE FUNCTION guard_v44_class_curriculum_child();
CREATE TRIGGER trg_guard_v44_curriculum_subjects
BEFORE INSERT OR UPDATE OR DELETE ON class_curriculum_subjects
FOR EACH ROW EXECUTE FUNCTION guard_v44_class_curriculum_child();

CREATE OR REPLACE FUNCTION validate_v44_enrollment_curriculum()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
DECLARE
    v_enrollment_year BIGINT;
    v_enrollment_class BIGINT;
    v_enrollment_status TEXT;
    v_student_status TEXT;
    v_curriculum_year BIGINT;
    v_curriculum_class BIGINT;
    v_curriculum_status TEXT;
BEGIN
    SELECT e.academic_year_id, e.class_id, e.status, st.status
      INTO v_enrollment_year, v_enrollment_class, v_enrollment_status, v_student_status
      FROM student_enrollments e
      JOIN students st ON st.id=e.student_id AND st.school_id=e.school_id
     WHERE e.id=NEW.enrollment_id AND e.school_id=NEW.school_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'student enrollment not found for curriculum assignment';
    END IF;
    SELECT academic_year_id, class_id, status
      INTO v_curriculum_year, v_curriculum_class, v_curriculum_status
      FROM class_curricula
     WHERE id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_curriculum_status <> 'active'
       OR v_curriculum_year <> v_enrollment_year
       OR v_curriculum_class <> v_enrollment_class THEN
        RAISE EXCEPTION 'curriculum does not match the active enrollment class and year';
    END IF;
    IF v_enrollment_status <> 'active' OR v_student_status <> 'active' THEN
        RAISE EXCEPTION 'only active students and enrollments can receive a curriculum';
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_validate_v44_enrollment_curriculum
BEFORE INSERT ON student_enrollment_curricula
FOR EACH ROW EXECUTE FUNCTION validate_v44_enrollment_curriculum();

CREATE OR REPLACE FUNCTION validate_v44_enrollment_subject_selection()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
DECLARE
    v_assignment_status TEXT;
    v_is_optional BOOLEAN;
BEGIN
    SELECT status INTO v_assignment_status
      FROM student_enrollment_curricula
     WHERE id=NEW.enrollment_curriculum_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR v_assignment_status <> 'active' THEN
        RAISE EXCEPTION 'subject choices require an active enrollment curriculum';
    END IF;
    SELECT optional_group_id IS NOT NULL INTO v_is_optional
      FROM class_curriculum_subjects
     WHERE id=NEW.curriculum_subject_id
       AND curriculum_id=NEW.curriculum_id AND school_id=NEW.school_id;
    IF NOT FOUND OR NOT v_is_optional THEN
        RAISE EXCEPTION 'only optional curriculum subjects can be selected';
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_validate_v44_enrollment_subject_selection
BEFORE INSERT ON student_enrollment_subject_selections
FOR EACH ROW EXECUTE FUNCTION validate_v44_enrollment_subject_selection();

CREATE OR REPLACE FUNCTION guard_v44_enrollment_curriculum_history()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student enrollment curriculum history cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.enrollment_id IS DISTINCT FROM OLD.enrollment_id
       OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
       OR NEW.assigned_by_principal_id IS DISTINCT FROM OLD.assigned_by_principal_id
       OR NEW.assigned_at IS DISTINCT FROM OLD.assigned_at THEN
        RAISE EXCEPTION 'student enrollment curriculum identity is immutable';
    END IF;
    IF OLD.status <> 'active' OR NEW.status <> 'replaced' OR NEW.revoked_at IS NULL THEN
        RAISE EXCEPTION 'active enrollment curriculum can only be replaced once';
    END IF;
    IF EXISTS (
        SELECT 1 FROM student_enrollment_subject_selections s
        WHERE s.enrollment_curriculum_id = OLD.id AND s.revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'optional subject selections must be revoked before replacing an enrollment curriculum';
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_guard_v44_enrollment_curriculum_history
BEFORE UPDATE OR DELETE ON student_enrollment_curricula
FOR EACH ROW EXECUTE FUNCTION guard_v44_enrollment_curriculum_history();

CREATE OR REPLACE FUNCTION guard_v44_enrollment_subject_history()
RETURNS trigger
LANGUAGE plpgsql AS $v44$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'student enrollment subject history cannot be deleted';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.enrollment_curriculum_id IS DISTINCT FROM OLD.enrollment_curriculum_id
       OR NEW.curriculum_id IS DISTINCT FROM OLD.curriculum_id
       OR NEW.curriculum_subject_id IS DISTINCT FROM OLD.curriculum_subject_id
       OR NEW.subject_name_snapshot IS DISTINCT FROM OLD.subject_name_snapshot
       OR NEW.subject_code_snapshot IS DISTINCT FROM OLD.subject_code_snapshot
       OR NEW.selected_by_principal_id IS DISTINCT FROM OLD.selected_by_principal_id
       OR NEW.selected_at IS DISTINCT FROM OLD.selected_at THEN
        RAISE EXCEPTION 'student enrollment subject selection identity is immutable';
    END IF;
    IF OLD.revoked_at IS NOT NULL OR NEW.revoked_at IS NULL THEN
        RAISE EXCEPTION 'student enrollment subject selection can only be revoked once';
    END IF;
    RETURN NEW;
END;
$v44$;
CREATE TRIGGER trg_guard_v44_enrollment_subject_history
BEFORE UPDATE OR DELETE ON student_enrollment_subject_selections
FOR EACH ROW EXECUTE FUNCTION guard_v44_enrollment_subject_history();

COMMENT ON TABLE class_curricula IS 'Immutable, versioned school/year/class subject curricula; a new version supersedes the previous active version.';
COMMENT ON TABLE class_curriculum_optional_groups IS 'Configurable optional-subject groups with explicit minimum and maximum student choices.';
COMMENT ON TABLE class_curriculum_subjects IS 'Compulsory subjects (no optional group) and optional subject choices for one curriculum version.';
COMMENT ON TABLE student_enrollment_curricula IS 'Append-only curriculum assignment history for one academic-year student enrollment.';
COMMENT ON TABLE student_enrollment_subject_selections IS 'Append-only optional subject choices tied to a specific enrollment and curriculum version.';
