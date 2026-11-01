-- V4: shared principal identity, RBAC catalogue, account subtypes,
-- principal-bound refresh sessions, MFA/OTP storage, and append-only audit.
-- This migration is intentionally additive. Existing users.id values remain stable.

-- V4_USERNAME_PREFLIGHT
-- Canonicalize existing staff usernames only when the complete dataset is safe.
-- Do not silently rename accounts: invalid values or global collisions must be
-- resolved deliberately before this migration is applied.
DO $v4_username_preflight$
DECLARE
    v_invalid_count BIGINT;
    v_collision_group_count BIGINT;
    v_invalid_sample_ids TEXT;
    v_collision_sample_ids TEXT;
BEGIN
    SELECT COUNT(*)
      INTO v_invalid_count
      FROM users
     WHERE LENGTH(LOWER(BTRIM(username))) NOT BETWEEN 3 AND 32
        OR SUBSTRING(LOWER(BTRIM(username)), 1, 1) !~ '[a-z0-9]'
        OR LOWER(BTRIM(username)) ~ '[^a-z0-9._-]';

    SELECT COUNT(*)
      INTO v_collision_group_count
      FROM (
          SELECT LOWER(BTRIM(username)) COLLATE "C" AS canonical_username
            FROM users
           GROUP BY LOWER(BTRIM(username)) COLLATE "C"
          HAVING COUNT(*) > 1
      ) duplicate_usernames;

    IF v_invalid_count > 0 OR v_collision_group_count > 0 THEN
        SELECT STRING_AGG(id::TEXT, ', ' ORDER BY id)
          INTO v_invalid_sample_ids
          FROM (
              SELECT id
                FROM users
               WHERE LENGTH(LOWER(BTRIM(username))) NOT BETWEEN 3 AND 32
                  OR SUBSTRING(LOWER(BTRIM(username)), 1, 1) !~ '[a-z0-9]'
                  OR LOWER(BTRIM(username)) ~ '[^a-z0-9._-]'
               ORDER BY id
               LIMIT 10
          ) invalid_rows;

        SELECT STRING_AGG(user_ids, '; ' ORDER BY user_ids)
          INTO v_collision_sample_ids
          FROM (
              SELECT STRING_AGG(id::TEXT, ', ' ORDER BY id) AS user_ids
                FROM users
               GROUP BY LOWER(BTRIM(username)) COLLATE "C"
              HAVING COUNT(*) > 1
               ORDER BY LOWER(BTRIM(username)) COLLATE "C"
               LIMIT 5
          ) duplicate_samples;

        RAISE EXCEPTION USING
            MESSAGE = FORMAT(
                'V4 username preflight failed: %s invalid username(s), %s normalized collision group(s). Invalid user IDs (up to 10): %s. Collision user ID groups (up to 5): %s.',
                v_invalid_count,
                v_collision_group_count,
                COALESCE(v_invalid_sample_ids, 'none'),
                COALESCE(v_collision_sample_ids, 'none')
            ),
            HINT = 'Resolve the listed users before applying V4. Usernames must normalize to 3-32 lowercase ASCII letters, digits, dots, underscores, or hyphens, and must be globally unique after LOWER(BTRIM(username)). No accounts are renamed automatically.';
    END IF;
END;
$v4_username_preflight$;

-- The preflight proves that normalization is valid and collision-free.
-- Trim surrounding spaces and lowercase existing usernames before enforcing
-- the canonical check and global unique index below.
UPDATE users
   SET username = LOWER(BTRIM(username))
 WHERE username IS DISTINCT FROM LOWER(BTRIM(username));

CREATE TABLE auth_principals (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_type VARCHAR(30) NOT NULL
        CHECK (principal_type IN ('staff', 'platform_admin', 'parent')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'active', 'disabled')),
    token_version BIGINT NOT NULL DEFAULT 1 CHECK (token_version >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, principal_type)
);

ALTER TABLE users ADD COLUMN principal_id BIGINT;

WITH principal_map AS MATERIALIZED (
    SELECT id AS user_id,
           nextval(pg_get_serial_sequence('auth_principals', 'id')::regclass) AS principal_id,
           CASE WHEN status = 'active' THEN 'active' ELSE 'disabled' END AS principal_status
    FROM users
), inserted_principals AS (
    INSERT INTO auth_principals (id, principal_type, status)
    OVERRIDING SYSTEM VALUE
    SELECT principal_id, 'staff', principal_status FROM principal_map
    RETURNING id
)
UPDATE users u
   SET principal_id = pm.principal_id
  FROM principal_map pm
  JOIN inserted_principals ip ON ip.id = pm.principal_id
 WHERE u.id = pm.user_id;

ALTER TABLE users ALTER COLUMN principal_id SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_principal_id_key UNIQUE (principal_id);
ALTER TABLE users ADD CONSTRAINT users_principal_school_unique UNIQUE (principal_id, school_id);
ALTER TABLE users ADD CONSTRAINT users_id_principal_unique UNIQUE (id, principal_id);
ALTER TABLE users ADD CONSTRAINT users_principal_id_fkey
    FOREIGN KEY (principal_id) REFERENCES auth_principals(id) ON DELETE RESTRICT;
ALTER TABLE users ADD CONSTRAINT users_username_canonical_check
    CHECK (username ~ '^[a-z0-9][a-z0-9._-]{2,31}$');

CREATE OR REPLACE FUNCTION normalize_staff_username() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.username := LOWER(BTRIM(NEW.username));
    IF NEW.username !~ '^[a-z0-9][a-z0-9._-]{2,31}$' THEN
        RAISE EXCEPTION 'username must be 3-32 lowercase ASCII letters, digits, dots, underscores, or hyphens';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_users_normalize_username
BEFORE INSERT OR UPDATE OF username ON users
FOR EACH ROW EXECUTE FUNCTION normalize_staff_username();

DROP INDEX users_school_username_unique;
CREATE UNIQUE INDEX ux_users_username_global ON users (LOWER(username));

CREATE OR REPLACE FUNCTION protect_principal_identity() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id OR
       NEW.principal_type IS DISTINCT FROM OLD.principal_type THEN
        RAISE EXCEPTION 'principal ID and principal type are immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_protect_principal_identity
BEFORE UPDATE OF id, principal_type ON auth_principals
FOR EACH ROW EXECUTE FUNCTION protect_principal_identity();

CREATE OR REPLACE FUNCTION protect_user_identity_scope() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.principal_id IS DISTINCT FROM OLD.principal_id THEN
        RAISE EXCEPTION 'users.principal_id is immutable';
    END IF;
    IF NEW.school_id IS DISTINCT FROM OLD.school_id THEN
        RAISE EXCEPTION 'staff school_id is immutable; provision a separate account for another school';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_protect_user_identity_scope
BEFORE UPDATE OF principal_id, school_id ON users
FOR EACH ROW EXECUTE FUNCTION protect_user_identity_scope();

CREATE TABLE platform_admin_accounts (
    principal_id BIGINT PRIMARY KEY REFERENCES auth_principals(id) ON DELETE RESTRICT,
    email VARCHAR(254) NOT NULL,
    password_hash TEXT NOT NULL CHECK (length(password_hash) > 0),
    must_change_password BOOLEAN NOT NULL DEFAULT TRUE,
    mfa_enrolled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ux_platform_admin_email ON platform_admin_accounts (LOWER(BTRIM(email)));

CREATE TABLE parent_accounts (
    principal_id BIGINT PRIMARY KEY REFERENCES auth_principals(id) ON DELETE RESTRICT,
    mobile_number VARCHAR(20) NOT NULL,
    mobile_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT parent_accounts_mobile_e164_check
        CHECK (mobile_number ~ '^\+[1-9][0-9]{7,14}$')
);
CREATE UNIQUE INDEX ux_parent_accounts_mobile ON parent_accounts (mobile_number);

-- Keep legacy status reads/writes compatible during the application cutover.
CREATE OR REPLACE FUNCTION sync_user_principal_status() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE auth_principals
       SET status = CASE WHEN NEW.status = 'active' THEN 'active' ELSE 'disabled' END,
           updated_at = NOW()
     WHERE id = NEW.principal_id;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_users_sync_principal_status
AFTER UPDATE OF status ON users
FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION sync_user_principal_status();

-- Keep the legacy users.status column aligned when the canonical principal
-- status is changed by newer principal-aware application code.
CREATE OR REPLACE FUNCTION sync_principal_user_status() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE users
       SET status = CASE WHEN NEW.status = 'active' THEN 'active' ELSE 'disabled' END,
           updated_at = NOW()
     WHERE principal_id = NEW.id
       AND status IS DISTINCT FROM CASE WHEN NEW.status = 'active' THEN 'active' ELSE 'disabled' END;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_principal_sync_user_status
AFTER UPDATE OF status ON auth_principals
FOR EACH ROW WHEN (OLD.status IS DISTINCT FROM NEW.status)
EXECUTE FUNCTION sync_principal_user_status();

-- Transitional compatibility: the current staff creation path does not yet
-- supply principal_id. Create its staff principal in the same transaction.
CREATE OR REPLACE FUNCTION assign_staff_principal_on_user_insert() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.principal_id IS NULL THEN
        INSERT INTO auth_principals (principal_type, status)
        VALUES ('staff', CASE WHEN NEW.status = 'active' THEN 'active' ELSE 'disabled' END)
        RETURNING id INTO NEW.principal_id;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_users_assign_principal
BEFORE INSERT ON users
FOR EACH ROW EXECUTE FUNCTION assign_staff_principal_on_user_insert();

CREATE TABLE permissions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    permission_key VARCHAR(120) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    scope_type VARCHAR(20) NOT NULL CHECK (scope_type IN ('platform', 'school', 'global')),
    is_delegable BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO permissions (permission_key, description, scope_type, is_delegable) VALUES
('platform.schools.read','View platform school records','platform',FALSE),
('platform.schools.create','Create a school','platform',FALSE),
('platform.schools.update','Update platform-level school metadata','platform',FALSE),
('platform.schools.suspend','Suspend or reactivate a school','platform',FALSE),
('platform.school_admins.create','Provision or invite a School Admin','platform',FALSE),
('platform.school_admins.disable','Disable a School Admin account','platform',FALSE),
('platform.audit.read','Read platform-level audit events','platform',FALSE),
('platform.permissions.delegate','Grant school permission delegation ceilings','platform',FALSE),
('school.settings.read','View school settings','school',TRUE),
('school.settings.update','Update school settings','school',FALSE),
('school.audit.read','Read authorized school audit events','school',FALSE),
('academic_years.read','Read configured academic years','school',TRUE),
('academic_years.manage','Create, update, or close academic years','school',FALSE),
('classes.read','Read classes and sections','school',TRUE),
('classes.manage','Create, update, or archive classes and sections','school',FALSE),
('subjects.read','Read school subjects','school',TRUE),
('subjects.manage','Create or update school subjects','school',FALSE),
('teacher_assignments.read','Read teacher assignments','school',FALSE),
('teacher_assignments.manage','Manage teacher assignments','school',FALSE),
('class_incharge_assignments.read','Read class-in-charge assignments','school',FALSE),
('class_incharge_assignments.manage','Manage class-in-charge assignments','school',FALSE),
('guardians.read','Read guardian records within authorized scope','school',FALSE),
('guardians.manage','Manage guardian relationship records','school',FALSE),
('students.read_basic','Read approved basic student fields','school',TRUE),
('students.read_sensitive','Read restricted student profile fields','school',FALSE),
('students.create','Create or admit students','school',FALSE),
('students.update','Update student records','school',FALSE),
('students.archive','Archive student records','school',FALSE),
('attendance.read','Read attendance within authorized scope','school',TRUE),
('attendance.mark','Mark attendance within authorized scope','school',TRUE),
('attendance.update_same_day','Correct same-day attendance within cutoff','school',TRUE),
('attendance.correct_past','Correct past attendance','school',FALSE),
('timetable.read','Read timetable within authorized scope','school',TRUE),
('timetable.manage','Manage school timetable','school',FALSE),
('homework.read','Read homework within authorized scope','school',TRUE),
('homework.create','Create homework within authorized scope','school',TRUE),
('homework.update','Update homework within authorized scope','school',TRUE),
('homework.delete','Delete homework','school',FALSE),
('exams.read','Read exam information within authorized scope','school',TRUE),
('exams.manage','Manage school exam configuration','school',FALSE),
('results.read','Read results within authorized scope','school',TRUE),
('results.enter','Enter marks within explicitly assigned scope','school',TRUE),
('results.update','Update marks within explicitly assigned scope','school',TRUE),
('results.publish','Publish results','school',FALSE),
('notices.read','Read applicable notices','school',TRUE),
('notices.create','Create notices','school',TRUE),
('notices.update','Update notices','school',TRUE),
('notices.publish','Publish notices','school',FALSE),
('notices.delete','Delete notices','school',FALSE),
('users.read','Read staff accounts within school','school',FALSE),
('users.create','Create staff accounts','school',FALSE),
('users.update','Update staff account details','school',FALSE),
('users.disable','Disable staff accounts','school',FALSE),
('users.password_reset','Reset same-school staff passwords','school',FALSE),
('roles.read','Read roles visible within school','school',TRUE),
('roles.create_custom','Create school-scoped custom roles','school',FALSE),
('roles.update_custom','Update school-scoped custom roles','school',FALSE),
('roles.assign','Assign permitted roles to staff','school',FALSE),
('permissions.read','Read platform permission catalogue','school',TRUE),
('permissions.assign','Assign permissions within delegation ceiling','school',FALSE),
('permissions.override','Set user permission overrides within delegation ceiling','school',FALSE),
('children.list','List verified linked children','global',FALSE),
('children.profile.read','Read approved profile of linked child','global',FALSE),
('children.attendance.read','Read attendance of linked child','global',FALSE),
('children.timetable.read','Read timetable of linked child','global',FALSE),
('children.homework.read','Read homework of linked child','global',FALSE),
('children.results.read','Read published results of linked child','global',FALSE),
('children.notices.read','Read applicable notices for linked child','global',FALSE);

CREATE OR REPLACE FUNCTION protect_permission_identity_scope() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.permission_key IS DISTINCT FROM OLD.permission_key OR
       NEW.scope_type IS DISTINCT FROM OLD.scope_type OR
       NEW.is_delegable IS DISTINCT FROM OLD.is_delegable THEN
        RAISE EXCEPTION
            'permission identity, scope, and delegability are immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_protect_permission_identity_scope
BEFORE UPDATE OF permission_key, scope_type, is_delegable ON permissions
FOR EACH ROW EXECUTE FUNCTION protect_permission_identity_scope();

CREATE TABLE roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT REFERENCES schools(id) ON DELETE RESTRICT,
    principal_type VARCHAR(30) NOT NULL CHECK (principal_type IN ('staff','platform_admin','parent')),
    scope_type VARCHAR(20) NOT NULL CHECK (scope_type IN ('platform','school','global')),
    role_key VARCHAR(80) NOT NULL,
    name VARCHAR(120) NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_by BIGINT REFERENCES auth_principals(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
      (scope_type = 'platform' AND school_id IS NULL AND principal_type = 'platform_admin') OR
      (scope_type = 'school' AND school_id IS NOT NULL AND principal_type = 'staff') OR
      (scope_type = 'global' AND school_id IS NULL AND principal_type = 'parent')
    )
);
CREATE UNIQUE INDEX ux_roles_school_key ON roles (school_id, LOWER(role_key)) WHERE scope_type = 'school';
CREATE UNIQUE INDEX ux_roles_global_key ON roles (scope_type, principal_type, LOWER(role_key)) WHERE scope_type IN ('platform','global');

CREATE OR REPLACE FUNCTION protect_role_identity_scope() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.school_id IS DISTINCT FROM OLD.school_id OR
       NEW.principal_type IS DISTINCT FROM OLD.principal_type OR
       NEW.scope_type IS DISTINCT FROM OLD.scope_type OR
       NEW.role_key IS DISTINCT FROM OLD.role_key OR
       NEW.is_system IS DISTINCT FROM OLD.is_system OR
       NEW.created_by IS DISTINCT FROM OLD.created_by THEN
        RAISE EXCEPTION 'role identity, scope, system classification, and creator are immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_protect_role_identity_scope
BEFORE UPDATE OF school_id, principal_type, scope_type, role_key, is_system, created_by ON roles
FOR EACH ROW EXECUTE FUNCTION protect_role_identity_scope();

INSERT INTO roles (school_id, principal_type, scope_type, role_key, name, is_system)
VALUES (NULL, 'platform_admin', 'platform', 'platform_admin', 'Platform Admin', TRUE),
       (NULL, 'parent', 'global', 'parent', 'Parent', TRUE);
INSERT INTO roles (school_id, principal_type, scope_type, role_key, name, is_system)
SELECT id, 'staff', 'school', 'school_admin', 'School Admin', TRUE FROM schools;
INSERT INTO roles (school_id, principal_type, scope_type, role_key, name, is_system)
SELECT id, 'staff', 'school', 'teacher', 'Teacher', TRUE FROM schools;

-- Only platform-controlled migration seeds may create system/platform/global roles.
-- Runtime role creation is restricted to school-scoped staff custom roles.
CREATE OR REPLACE FUNCTION guard_role_insert_and_system_classification() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.is_system THEN
            RAISE EXCEPTION 'system roles may only be created by controlled migrations';
        END IF;

        IF NEW.scope_type <> 'school'
           OR NEW.principal_type <> 'staff'
           OR NEW.school_id IS NULL THEN
            RAISE EXCEPTION 'runtime custom roles must be school-scoped staff roles';
        END IF;

        IF NEW.created_by IS NULL OR NOT EXISTS (
            SELECT 1
              FROM auth_principals ap
              JOIN users u ON u.principal_id = ap.id
             WHERE ap.id = NEW.created_by
               AND ap.principal_type = 'staff'
               AND ap.status = 'active'
               AND u.school_id = NEW.school_id
        ) THEN
            RAISE EXCEPTION 'custom role creator must be an active staff principal in the same school';
        END IF;
    ELSE
        IF NEW.is_system IS DISTINCT FROM OLD.is_system THEN
            RAISE EXCEPTION 'role system classification is immutable';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_guard_role_insert_and_system_classification
BEFORE INSERT OR UPDATE ON roles
FOR EACH ROW EXECUTE FUNCTION guard_role_insert_and_system_classification();

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);
-- Conservative built-in bundles. Resource/assignment constraints remain mandatory in the API.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.role_key = 'platform_admin' AND r.scope_type = 'platform' AND p.scope_type = 'platform';

-- School Admin receives only the explicitly approved Phase 1 permission bundle.
-- Never grant every school-scoped permission automatically.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p
  ON p.scope_type = 'school'
   AND p.permission_key IN (
    'school.settings.read',
    'school.settings.update',
    'school.audit.read',
    'academic_years.read',
    'academic_years.manage',
    'classes.read',
    'classes.manage',
    'subjects.read',
    'subjects.manage',
    'teacher_assignments.read',
    'teacher_assignments.manage',
    'class_incharge_assignments.read',
    'class_incharge_assignments.manage',
    'students.read_basic',
    'students.read_sensitive',
    'students.create',
    'students.update',
    'students.archive',
    'attendance.read',
    'attendance.mark',
    'attendance.update_same_day',
    'attendance.correct_past',
    'timetable.read',
    'timetable.manage',
    'homework.read',
    'homework.create',
    'homework.update',
    'homework.delete',
    'exams.read',
    'exams.manage',
    'results.read',
    'results.enter',
    'results.update',
    'results.publish',
    'notices.read',
    'notices.create',
    'notices.update',
    'notices.publish',
    'notices.delete',
    'users.read',
    'users.create',
    'users.update',
    'users.disable',
    'users.password_reset',
    'roles.read',
    'roles.create_custom',
    'roles.update_custom',
    'roles.assign',
    'permissions.read',
    'permissions.assign',
    'permissions.override'
  )
WHERE r.role_key = 'school_admin'
  AND r.scope_type = 'school';

-- Fail migration unless every School Admin system role receives exactly the approved 51-key set.
-- The count plus bidirectional EXCEPT checks catch missing, duplicate, and unexpected grants.
DO $school_admin_assert$
DECLARE
    v_role RECORD;
    v_permission_count INTEGER;
    v_expected_permissions TEXT[] := ARRAY[
        'school.settings.read',
        'school.settings.update',
        'school.audit.read',
        'academic_years.read',
        'academic_years.manage',
        'classes.read',
        'classes.manage',
        'subjects.read',
        'subjects.manage',
        'teacher_assignments.read',
        'teacher_assignments.manage',
        'class_incharge_assignments.read',
        'class_incharge_assignments.manage',
        'students.read_basic',
        'students.read_sensitive',
        'students.create',
        'students.update',
        'students.archive',
        'attendance.read',
        'attendance.mark',
        'attendance.update_same_day',
        'attendance.correct_past',
        'timetable.read',
        'timetable.manage',
        'homework.read',
        'homework.create',
        'homework.update',
        'homework.delete',
        'exams.read',
        'exams.manage',
        'results.read',
        'results.enter',
        'results.update',
        'results.publish',
        'notices.read',
        'notices.create',
        'notices.update',
        'notices.publish',
        'notices.delete',
        'users.read',
        'users.create',
        'users.update',
        'users.disable',
        'users.password_reset',
        'roles.read',
        'roles.create_custom',
        'roles.update_custom',
        'roles.assign',
        'permissions.read',
        'permissions.assign',
        'permissions.override'
    ];
BEGIN
    FOR v_role IN
        SELECT id
          FROM roles
         WHERE role_key = 'school_admin'
           AND principal_type = 'staff'
           AND scope_type = 'school'
           AND school_id IS NOT NULL
           AND is_system = TRUE
    LOOP
        SELECT COUNT(*)
          INTO v_permission_count
          FROM role_permissions
         WHERE role_id = v_role.id;

        IF v_permission_count <> 51
           OR EXISTS (
                SELECT permission_key
                  FROM unnest(v_expected_permissions) AS expected(permission_key)
                EXCEPT
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
           )
           OR EXISTS (
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
                EXCEPT
                SELECT permission_key
                  FROM unnest(v_expected_permissions) AS expected(permission_key)
           ) THEN
            RAISE EXCEPTION
                'School Admin role % does not have exactly the 51 approved permissions',
                v_role.id;
        END IF;
    END LOOP;
END;
$school_admin_assert$;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p
  ON p.scope_type = 'school'
 AND p.permission_key IN (
    'students.read_basic',
    'attendance.read',
    'attendance.mark',
    'attendance.update_same_day',
    'timetable.read',
    'homework.read',
    'notices.read'
 )
WHERE r.role_key = 'teacher'
  AND r.principal_type = 'staff'
  AND r.scope_type = 'school'
  AND r.school_id IS NOT NULL
  AND r.is_system = TRUE;

-- Fail migration unless every school has exactly one Teacher system role
-- and every such role has exactly the approved seven-key default bundle.
DO $teacher_assert$
DECLARE
    v_school_count INTEGER;
    v_teacher_role_count INTEGER;
    v_role RECORD;
    v_permission_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_school_count FROM schools;

    SELECT COUNT(*)
      INTO v_teacher_role_count
      FROM roles
     WHERE role_key = 'teacher'
       AND principal_type = 'staff'
       AND scope_type = 'school'
       AND school_id IS NOT NULL
       AND is_system = TRUE;

    IF v_teacher_role_count <> v_school_count THEN
        RAISE EXCEPTION
            'Expected one Teacher system role per school (%); found %',
            v_school_count, v_teacher_role_count;
    END IF;

    FOR v_role IN
        SELECT id
          FROM roles
         WHERE role_key = 'teacher'
           AND principal_type = 'staff'
           AND scope_type = 'school'
           AND school_id IS NOT NULL
           AND is_system = TRUE
    LOOP
        SELECT COUNT(*)
          INTO v_permission_count
          FROM role_permissions
         WHERE role_id = v_role.id;

        IF v_permission_count <> 7
           OR EXISTS (
                SELECT permission_key
                  FROM (VALUES
                    ('students.read_basic'),
                    ('attendance.read'),
                    ('attendance.mark'),
                    ('attendance.update_same_day'),
                    ('timetable.read'),
                    ('homework.read'),
                    ('notices.read')
                  ) AS expected(permission_key)
                EXCEPT
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
           )
           OR EXISTS (
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
                EXCEPT
                SELECT permission_key
                  FROM (VALUES
                    ('students.read_basic'),
                    ('attendance.read'),
                    ('attendance.mark'),
                    ('attendance.update_same_day'),
                    ('timetable.read'),
                    ('homework.read'),
                    ('notices.read')
                  ) AS expected(permission_key)
           ) THEN
            RAISE EXCEPTION
                'Teacher role % does not have exactly the seven approved permissions',
                v_role.id;
        END IF;
    END LOOP;
END;
$teacher_assert$;
-- Parent receives only the seven explicitly approved read-only child/dashboard permissions.
-- Do not CROSS JOIN the global catalogue: future global permissions must not be inherited automatically.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p
  ON p.scope_type = 'global'
 AND p.permission_key IN (
    'children.list',
    'children.profile.read',
    'children.attendance.read',
    'children.timetable.read',
    'children.homework.read',
    'children.results.read',
    'children.notices.read'
 )
WHERE r.role_key = 'parent'
  AND r.scope_type = 'global';

-- Fail migration unless exactly one Parent system role has exactly the approved permission set.
-- Bidirectional EXCEPT checks catch both missing and unexpected grants, not just count drift.
DO $parent_assert$
DECLARE
    v_parent_role_count INTEGER;
    v_role RECORD;
    v_permission_count INTEGER;
BEGIN
    SELECT COUNT(*)
      INTO v_parent_role_count
      FROM roles
     WHERE role_key = 'parent'
       AND principal_type = 'parent'
       AND scope_type = 'global'
       AND school_id IS NULL
       AND is_system = TRUE;

    IF v_parent_role_count <> 1 THEN
        RAISE EXCEPTION
            'Expected exactly one global Parent system role; found %',
            v_parent_role_count;
    END IF;

    FOR v_role IN
        SELECT id
          FROM roles
         WHERE role_key = 'parent'
           AND principal_type = 'parent'
           AND scope_type = 'global'
           AND school_id IS NULL
           AND is_system = TRUE
    LOOP
        SELECT COUNT(*)
          INTO v_permission_count
          FROM role_permissions
         WHERE role_id = v_role.id;

        IF v_permission_count <> 7
           OR EXISTS (
                SELECT permission_key
                  FROM (VALUES
                    ('children.list'),
                    ('children.profile.read'),
                    ('children.attendance.read'),
                    ('children.timetable.read'),
                    ('children.homework.read'),
                    ('children.results.read'),
                    ('children.notices.read')
                  ) AS expected(permission_key)
                EXCEPT
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
           )
           OR EXISTS (
                SELECT p.permission_key
                  FROM role_permissions rp
                  JOIN permissions p ON p.id = rp.permission_id
                 WHERE rp.role_id = v_role.id
                EXCEPT
                SELECT permission_key
                  FROM (VALUES
                    ('children.list'),
                    ('children.profile.read'),
                    ('children.attendance.read'),
                    ('children.timetable.read'),
                    ('children.homework.read'),
                    ('children.results.read'),
                    ('children.notices.read')
                  ) AS expected(permission_key)
           ) THEN
            RAISE EXCEPTION
                'Parent role % does not have exactly the seven approved permissions',
                v_role.id;
        END IF;
    END LOOP;
END;
$parent_assert$;

CREATE TABLE principal_roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    assigned_by BIGINT REFERENCES auth_principals(id) ON DELETE RESTRICT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX ux_principal_roles_active ON principal_roles (principal_id, role_id) WHERE revoked_at IS NULL;
CREATE INDEX ix_principal_roles_principal_active ON principal_roles (principal_id) WHERE revoked_at IS NULL;
CREATE INDEX ix_principal_roles_role_active ON principal_roles (role_id) WHERE revoked_at IS NULL;

-- Preserve role-assignment history. Revocation is one-way; a later grant must
-- create a new principal_roles row rather than reactivating historical data.
CREATE OR REPLACE FUNCTION guard_principal_role_history() RETURNS trigger
LANGUAGE plpgsql AS $
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'principal role assignment history cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.revoked_at IS NOT NULL THEN
            RAISE EXCEPTION 'new principal role assignments must start active';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.principal_id IS DISTINCT FROM OLD.principal_id OR
       NEW.role_id IS DISTINCT FROM OLD.role_id OR
       NEW.assigned_by IS DISTINCT FROM OLD.assigned_by OR
       NEW.assigned_at IS DISTINCT FROM OLD.assigned_at THEN
        RAISE EXCEPTION 'principal role assignment identity and provenance are immutable';
    END IF;

    IF OLD.revoked_at IS NOT NULL AND
       NEW.revoked_at IS DISTINCT FROM OLD.revoked_at THEN
        RAISE EXCEPTION 'principal role assignment revocation is irreversible';
    END IF;

    RETURN NEW;
END;
$;
CREATE TRIGGER trg_guard_principal_role_history
BEFORE INSERT OR UPDATE OR DELETE ON principal_roles
FOR EACH ROW EXECUTE FUNCTION guard_principal_role_history();

CREATE TABLE user_permission_overrides (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    effect VARCHAR(10) NOT NULL CHECK (effect IN ('allow','deny')),
    reason TEXT NOT NULL,
    granted_by BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (principal_id, school_id, permission_id),
    FOREIGN KEY (principal_id, school_id)
      REFERENCES users(principal_id, school_id) ON DELETE RESTRICT
);

CREATE TABLE role_permission_delegations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    school_id BIGINT NOT NULL REFERENCES schools(id) ON DELETE RESTRICT,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    can_assign_to_roles BOOLEAN NOT NULL DEFAULT FALSE,
    can_override_users BOOLEAN NOT NULL DEFAULT FALSE,
    granted_by BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX ux_role_permission_delegations_active
    ON role_permission_delegations (school_id, permission_id) WHERE revoked_at IS NULL;

-- Move refresh-session ownership from a staff-only user ID to a principal ID.
ALTER TABLE refresh_sessions ADD COLUMN principal_id BIGINT;
UPDATE refresh_sessions rs SET principal_id = u.principal_id
FROM users u WHERE u.id = rs.user_id;
ALTER TABLE refresh_sessions ALTER COLUMN principal_id SET NOT NULL;
ALTER TABLE refresh_sessions DROP CONSTRAINT refresh_sessions_user_id_fkey;
ALTER TABLE refresh_sessions ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE refresh_sessions ADD CONSTRAINT refresh_sessions_user_principal_fkey
    FOREIGN KEY (user_id, principal_id) REFERENCES users(id, principal_id) ON DELETE RESTRICT;
ALTER TABLE refresh_sessions ADD CONSTRAINT refresh_sessions_principal_id_fkey
    FOREIGN KEY (principal_id) REFERENCES auth_principals(id) ON DELETE RESTRICT;
CREATE INDEX ix_refresh_sessions_principal_id ON refresh_sessions (principal_id);
CREATE INDEX ix_refresh_sessions_principal_active
    ON refresh_sessions (principal_id, expires_at) WHERE revoked_at IS NULL;

-- Transitional compatibility: the current staff session service inserts user_id
-- only. Resolve principal_id for staff; future platform/parent services supply it.
-- Keep the legacy user_id and canonical principal_id columns synchronized
-- for staff sessions, regardless of which identity the caller supplies.
CREATE OR REPLACE FUNCTION assign_refresh_session_principal() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_user_principal_id BIGINT;
    v_staff_user_id BIGINT;
BEGIN
    IF NEW.user_id IS NOT NULL THEN
        SELECT principal_id INTO v_user_principal_id
          FROM users
         WHERE id = NEW.user_id;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'refresh session user does not exist';
        END IF;

        IF NEW.principal_id IS NULL THEN
            NEW.principal_id := v_user_principal_id;
        ELSIF NEW.principal_id IS DISTINCT FROM v_user_principal_id THEN
            RAISE EXCEPTION 'refresh session user_id and principal_id do not match';
        END IF;
    ELSIF NEW.principal_id IS NOT NULL THEN
        SELECT id INTO v_staff_user_id
          FROM users
         WHERE principal_id = NEW.principal_id;

        IF FOUND THEN
            NEW.user_id := v_staff_user_id;
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_refresh_sessions_assign_principal
BEFORE INSERT OR UPDATE OF user_id, principal_id ON refresh_sessions
FOR EACH ROW EXECUTE FUNCTION assign_refresh_session_principal();

CREATE TABLE principal_mfa_credentials (
    principal_id BIGINT PRIMARY KEY REFERENCES auth_principals(id) ON DELETE RESTRICT,
    method VARCHAR(20) NOT NULL CHECK (method = 'totp'),
    secret_ciphertext BYTEA NOT NULL,
    encryption_key_version INTEGER NOT NULL CHECK (encryption_key_version >= 1),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE principal_mfa_recovery_codes (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    code_hash CHAR(64) NOT NULL UNIQUE,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_mfa_recovery_unused ON principal_mfa_recovery_codes (principal_id) WHERE used_at IS NULL;

CREATE TABLE parent_otp_challenges (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL REFERENCES auth_principals(id) ON DELETE RESTRICT,
    otp_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    attempt_count SMALLINT NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 5),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX ux_parent_otp_unconsumed ON parent_otp_challenges (principal_id) WHERE consumed_at IS NULL;
CREATE INDEX ix_parent_otp_recent ON parent_otp_challenges (principal_id, created_at DESC);

CREATE TABLE audit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor_principal_id BIGINT REFERENCES auth_principals(id) ON DELETE RESTRICT,
    school_id BIGINT REFERENCES schools(id) ON DELETE RESTRICT,
    action VARCHAR(120) NOT NULL,
    resource_type VARCHAR(120) NOT NULL,
    resource_id VARCHAR(120),
    outcome VARCHAR(10) NOT NULL CHECK (outcome IN ('success','failure')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_audit_events_created ON audit_events (created_at DESC);
CREATE INDEX ix_audit_events_school_created ON audit_events (school_id, created_at DESC);
CREATE INDEX ix_audit_events_actor_created ON audit_events (actor_principal_id, created_at DESC);
CREATE INDEX ix_audit_events_action_created ON audit_events (action, created_at DESC);

-- Exactly one account subtype must exist for every principal, and it must match its type.
CREATE OR REPLACE FUNCTION assert_principal_subtype() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_ids BIGINT[];
    v_id BIGINT;
    v_type VARCHAR(30);
    v_count INTEGER;
    v_staff BOOLEAN;
    v_platform BOOLEAN;
    v_parent BOOLEAN;
BEGIN
    -- Never dereference an unassigned NEW/OLD record. On UPDATE validate both
    -- identities so changing a subtype's principal_id cannot orphan the old one.
    IF TG_TABLE_NAME = 'auth_principals' THEN
        IF TG_OP = 'INSERT' THEN
            v_ids := ARRAY[NEW.id];
        ELSIF TG_OP = 'DELETE' THEN
            v_ids := ARRAY[OLD.id];
        ELSE
            v_ids := ARRAY[OLD.id, NEW.id];
        END IF;
    ELSE
        IF TG_OP = 'INSERT' THEN
            v_ids := ARRAY[NEW.principal_id];
        ELSIF TG_OP = 'DELETE' THEN
            v_ids := ARRAY[OLD.principal_id];
        ELSE
            v_ids := ARRAY[OLD.principal_id, NEW.principal_id];
        END IF;
    END IF;

    FOREACH v_id IN ARRAY v_ids LOOP
        SELECT principal_type INTO v_type
          FROM auth_principals WHERE id = v_id;
        IF NOT FOUND THEN
            CONTINUE;
        END IF;

        SELECT EXISTS(SELECT 1 FROM users WHERE principal_id = v_id),
               EXISTS(SELECT 1 FROM platform_admin_accounts WHERE principal_id = v_id),
               EXISTS(SELECT 1 FROM parent_accounts WHERE principal_id = v_id)
          INTO v_staff, v_platform, v_parent;
        v_count := (CASE WHEN v_staff THEN 1 ELSE 0 END) +
                   (CASE WHEN v_platform THEN 1 ELSE 0 END) +
                   (CASE WHEN v_parent THEN 1 ELSE 0 END);
        IF v_count <> 1 OR
           (v_type = 'staff' AND NOT v_staff) OR
           (v_type = 'platform_admin' AND NOT v_platform) OR
           (v_type = 'parent' AND NOT v_parent) THEN
            RAISE EXCEPTION 'principal % must have exactly one matching account subtype', v_id;
        END IF;
    END LOOP;
    RETURN NULL;
END;
$$;
CREATE CONSTRAINT TRIGGER ct_principal_subtype_principal
AFTER INSERT OR UPDATE ON auth_principals DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION assert_principal_subtype();
CREATE CONSTRAINT TRIGGER ct_principal_subtype_users
AFTER INSERT OR UPDATE OR DELETE ON users DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION assert_principal_subtype();
CREATE CONSTRAINT TRIGGER ct_principal_subtype_platform
AFTER INSERT OR UPDATE OR DELETE ON platform_admin_accounts DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION assert_principal_subtype();
CREATE CONSTRAINT TRIGGER ct_principal_subtype_parent
AFTER INSERT OR UPDATE OR DELETE ON parent_accounts DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION assert_principal_subtype();

CREATE OR REPLACE FUNCTION validate_role_permission_scope() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_role RECORD;
    v_permission RECORD;
BEGIN
    SELECT id, school_id, scope_type, is_system INTO v_role FROM roles WHERE id = NEW.role_id;
    SELECT id, scope_type, is_active, is_delegable INTO v_permission FROM permissions WHERE id = NEW.permission_id;
    IF v_role.id IS NULL OR v_permission.id IS NULL OR
       v_role.scope_type <> v_permission.scope_type OR NOT v_permission.is_active THEN
        RAISE EXCEPTION 'role and active permission scopes must match';
    END IF;
    IF v_role.scope_type = 'school' AND NOT v_role.is_system AND NOT EXISTS (
        SELECT 1 FROM role_permission_delegations d
         WHERE d.school_id = v_role.school_id
           AND d.permission_id = v_permission.id
           AND d.can_assign_to_roles = TRUE
           AND d.revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'custom role permission exceeds the active school delegation ceiling';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_role_permission_scope BEFORE INSERT OR UPDATE ON role_permissions
FOR EACH ROW EXECUTE FUNCTION validate_role_permission_scope();

CREATE OR REPLACE FUNCTION protect_system_roles() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_old_role_id BIGINT;
    v_new_role_id BIGINT;
BEGIN
    IF TG_TABLE_NAME = 'roles' THEN
        IF TG_OP <> 'INSERT' THEN v_old_role_id := OLD.id; END IF;
        IF TG_OP <> 'DELETE' THEN v_new_role_id := NEW.id; END IF;
    ELSE
        IF TG_OP <> 'INSERT' THEN v_old_role_id := OLD.role_id; END IF;
        IF TG_OP <> 'DELETE' THEN v_new_role_id := NEW.role_id; END IF;
    END IF;

    IF (v_old_role_id IS NOT NULL AND EXISTS
        (SELECT 1 FROM roles WHERE id = v_old_role_id AND is_system)) OR
       (v_new_role_id IS NOT NULL AND EXISTS
        (SELECT 1 FROM roles WHERE id = v_new_role_id AND is_system)) THEN
        RAISE EXCEPTION 'system roles and their permission bundles are immutable';
    END IF;

    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_protect_system_roles BEFORE UPDATE OR DELETE ON roles
FOR EACH ROW EXECUTE FUNCTION protect_system_roles();
CREATE TRIGGER trg_protect_system_role_permissions BEFORE INSERT OR UPDATE OR DELETE ON role_permissions
FOR EACH ROW EXECUTE FUNCTION protect_system_roles();

CREATE OR REPLACE FUNCTION validate_principal_role_assignment() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_type VARCHAR(30);
    v_target_status VARCHAR(20);
    v_role RECORD;
    v_school_id BIGINT;
    v_actor_type VARCHAR(30);
    v_actor_school_id BIGINT;
    v_permission_id BIGINT;
BEGIN
    SELECT principal_type, status
      INTO v_type, v_target_status
      FROM auth_principals
     WHERE id = NEW.principal_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'role assignment target principal does not exist';
    END IF;
    IF v_target_status <> 'active' THEN
        RAISE EXCEPTION 'role assignment target principal must be active';
    END IF;

    SELECT id, school_id, principal_type, scope_type, role_key
      INTO v_role FROM roles WHERE id = NEW.role_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'role assignment target role does not exist';
    END IF;
    IF v_type IS DISTINCT FROM v_role.principal_type THEN
        RAISE EXCEPTION 'principal type does not match role type';
    END IF;
    IF v_type = 'staff' THEN
        SELECT school_id INTO v_school_id FROM users WHERE principal_id = NEW.principal_id;
        IF v_school_id IS DISTINCT FROM v_role.school_id THEN
            RAISE EXCEPTION 'staff principal and role must belong to the same school';
        END IF;
    END IF;
    IF v_type = 'platform_admin' AND v_role.scope_type <> 'platform' THEN
        RAISE EXCEPTION 'Platform Admin may only receive platform roles';
    ELSIF v_type = 'parent' AND v_role.scope_type <> 'global' THEN
        RAISE EXCEPTION 'Parent may only receive global roles';
    END IF;

    -- Parent role assignment is created only by the verified-parent OTP flow.
    -- The parent subtype must already be verified; assigned_by remains NULL
    -- because this is a system-mediated identity activation, not staff delegation.
    IF v_type = 'parent' AND v_role.role_key = 'parent' AND NEW.assigned_by IS NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM parent_accounts
             WHERE principal_id = NEW.principal_id AND mobile_verified_at IS NOT NULL
        ) THEN
            RAISE EXCEPTION 'parent role requires a verified mobile account';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.assigned_by IS NULL THEN
        RAISE EXCEPTION 'role assignment requires an authenticated assigning principal';
    END IF;
    SELECT principal_type INTO v_actor_type
      FROM auth_principals
     WHERE id = NEW.assigned_by AND status = 'active';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'assigning principal must exist and be active';
    END IF;

    IF v_role.role_key = 'school_admin' THEN
        IF v_actor_type <> 'platform_admin' OR NOT EXISTS (
            SELECT 1
              FROM principal_roles apr
              JOIN roles ar ON ar.id = apr.role_id
              JOIN role_permissions arp ON arp.role_id = ar.id
              JOIN permissions ap ON ap.id = arp.permission_id
             WHERE apr.principal_id = NEW.assigned_by
               AND apr.revoked_at IS NULL
               AND ar.scope_type = 'platform'
               AND ar.principal_type = 'platform_admin'
               AND ar.role_key = 'platform_admin'
               AND ap.permission_key = 'platform.school_admins.create'
               AND ap.scope_type = 'platform'
               AND ap.is_active = TRUE
        ) THEN
            RAISE EXCEPTION 'School Admin assignment requires an active Platform Admin with platform.school_admins.create';
        END IF;
    ELSE
        -- Routine staff role assignment requires an active same-school School Admin
        -- whose effective system-role grant includes roles.assign and is not denied.
        IF v_actor_type <> 'staff' THEN
            RAISE EXCEPTION 'only a same-school School Admin may assign routine staff roles';
        END IF;
        SELECT u.school_id INTO v_actor_school_id
          FROM users u
         WHERE u.principal_id = NEW.assigned_by;
        SELECT id INTO v_permission_id
          FROM permissions
         WHERE permission_key = 'roles.assign'
           AND scope_type = 'school' AND is_active = TRUE;
        IF v_actor_school_id IS DISTINCT FROM v_role.school_id OR NOT EXISTS (
            SELECT 1
              FROM principal_roles pr
              JOIN roles ar ON ar.id = pr.role_id
              JOIN role_permissions arp ON arp.role_id = ar.id
             WHERE pr.principal_id = NEW.assigned_by
               AND pr.revoked_at IS NULL
               AND ar.scope_type = 'school'
               AND ar.school_id = v_role.school_id
               AND ar.principal_type = 'staff'
               AND ar.role_key = 'school_admin'
               AND arp.permission_id = v_permission_id
        ) OR v_permission_id IS NULL OR EXISTS (
            SELECT 1
              FROM user_permission_overrides upo
             WHERE upo.principal_id = NEW.assigned_by
               AND upo.school_id = v_role.school_id
               AND upo.permission_id = v_permission_id
               AND upo.effect = 'deny'
        ) THEN
            RAISE EXCEPTION 'role assignment requires an active same-school School Admin with effective roles.assign permission';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_principal_role BEFORE INSERT OR UPDATE OF principal_id, role_id, assigned_by ON principal_roles
FOR EACH ROW EXECUTE FUNCTION validate_principal_role_assignment();

CREATE OR REPLACE FUNCTION validate_user_permission_override() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_permission_id BIGINT;
    v_grantor_school BIGINT;
    v_authority_permission_id BIGINT;
BEGIN
    SELECT id INTO v_permission_id FROM permissions
     WHERE id = NEW.permission_id AND scope_type = 'school' AND is_active AND is_delegable;
    IF v_permission_id IS NULL THEN
        RAISE EXCEPTION 'user permission overrides require an active, delegable school permission';
    END IF;
    SELECT school_id INTO v_grantor_school FROM users
     WHERE principal_id = NEW.granted_by;
    SELECT id INTO v_authority_permission_id FROM permissions
     WHERE permission_key = 'permissions.override'
       AND scope_type = 'school' AND is_active = TRUE;
    IF v_grantor_school IS DISTINCT FROM NEW.school_id OR NOT EXISTS (
        SELECT 1 FROM auth_principals ap
         WHERE ap.id = NEW.granted_by AND ap.principal_type = 'staff' AND ap.status = 'active'
    ) OR NOT EXISTS (
        SELECT 1
          FROM principal_roles pr
          JOIN roles r ON r.id = pr.role_id
          JOIN role_permissions rp ON rp.role_id = r.id
         WHERE pr.principal_id = NEW.granted_by
           AND pr.revoked_at IS NULL
           AND r.school_id = NEW.school_id
           AND r.scope_type = 'school'
           AND r.principal_type = 'staff'
           AND r.role_key = 'school_admin'
           AND rp.permission_id = v_authority_permission_id
    ) OR v_authority_permission_id IS NULL OR EXISTS (
        SELECT 1
          FROM user_permission_overrides actor_override
         WHERE actor_override.principal_id = NEW.granted_by
           AND actor_override.school_id = NEW.school_id
           AND actor_override.permission_id = v_authority_permission_id
           AND actor_override.effect = 'deny'
    ) THEN
        RAISE EXCEPTION 'override grantor must be an active same-school School Admin with effective permissions.override permission';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM role_permission_delegations d
         WHERE d.school_id = NEW.school_id AND d.permission_id = NEW.permission_id
           AND d.can_override_users = TRUE AND d.revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'user permission override exceeds the active delegation ceiling';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_user_permission_override BEFORE INSERT OR UPDATE ON user_permission_overrides
FOR EACH ROW EXECUTE FUNCTION validate_user_permission_override();

CREATE OR REPLACE FUNCTION validate_permission_delegation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM permissions
         WHERE id = NEW.permission_id
           AND scope_type = 'school'
           AND is_delegable = TRUE
           AND is_active = TRUE
    ) THEN
        RAISE EXCEPTION
            'only active, explicitly delegable school permissions may be delegated';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM auth_principals ap
          JOIN principal_roles pr
            ON pr.principal_id = ap.id
           AND pr.revoked_at IS NULL
          JOIN roles r
            ON r.id = pr.role_id
           AND r.principal_type = 'platform_admin'
           AND r.scope_type = 'platform'
           AND r.role_key = 'platform_admin'
           AND r.is_system = TRUE
          JOIN role_permissions rp
            ON rp.role_id = r.id
          JOIN permissions dp
            ON dp.id = rp.permission_id
           AND dp.permission_key = 'platform.permissions.delegate'
           AND dp.scope_type = 'platform'
           AND dp.is_active = TRUE
         WHERE ap.id = NEW.granted_by
           AND ap.principal_type = 'platform_admin'
           AND ap.status = 'active'
    ) THEN
        RAISE EXCEPTION
            'grantor must be an active Platform Admin with platform.permissions.delegate';
    END IF;

    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_permission_delegation BEFORE INSERT OR UPDATE OF school_id, permission_id, can_assign_to_roles, can_override_users, granted_by ON role_permission_delegations
FOR EACH ROW EXECUTE FUNCTION validate_permission_delegation();

-- Revocation is one-way: once revoked, a delegation cannot be reactivated
-- or have its historical revocation timestamp rewritten.
CREATE OR REPLACE FUNCTION prevent_delegation_reactivation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.revoked_at IS NOT NULL AND NEW.revoked_at IS DISTINCT FROM OLD.revoked_at THEN
        RAISE EXCEPTION 'delegation revocation is immutable and cannot be cleared or rewritten';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_prevent_delegation_reactivation
BEFORE UPDATE OF revoked_at ON role_permission_delegations
FOR EACH ROW EXECUTE FUNCTION prevent_delegation_reactivation();

CREATE OR REPLACE FUNCTION validate_refresh_session_principal() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE v_type VARCHAR(30);
BEGIN
    SELECT principal_type INTO v_type FROM auth_principals WHERE id = NEW.principal_id;
    IF v_type = 'staff' THEN
        IF NEW.user_id IS NULL OR NOT EXISTS (
            SELECT 1 FROM users WHERE id = NEW.user_id AND principal_id = NEW.principal_id
        ) THEN RAISE EXCEPTION 'staff refresh session requires matching user_id'; END IF;
    ELSIF v_type IN ('platform_admin','parent') THEN
        IF NEW.user_id IS NOT NULL THEN RAISE EXCEPTION 'non-staff refresh session must not have user_id'; END IF;
    ELSE
        RAISE EXCEPTION 'refresh session principal does not exist';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_refresh_session_principal BEFORE INSERT OR UPDATE ON refresh_sessions
FOR EACH ROW EXECUTE FUNCTION validate_refresh_session_principal();

CREATE OR REPLACE FUNCTION validate_mfa_principal() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM auth_principals WHERE id = NEW.principal_id AND principal_type IN ('staff','platform_admin')) THEN
        RAISE EXCEPTION 'MFA credentials are only supported for staff and Platform Admin principals';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_mfa_principal BEFORE INSERT OR UPDATE ON principal_mfa_credentials
FOR EACH ROW EXECUTE FUNCTION validate_mfa_principal();

CREATE OR REPLACE FUNCTION validate_parent_otp_principal() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM auth_principals WHERE id = NEW.principal_id AND principal_type = 'parent') THEN
        RAISE EXCEPTION 'OTP challenges require a parent principal';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_validate_parent_otp_principal BEFORE INSERT OR UPDATE ON parent_otp_challenges
FOR EACH ROW EXECUTE FUNCTION validate_parent_otp_principal();

CREATE OR REPLACE FUNCTION reject_audit_event_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events are append-only';
END;
$$;
CREATE TRIGGER trg_audit_events_immutable BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

-- TRUNCATE is a statement-level operation and does not fire row-level triggers.
-- Block it explicitly so the append-only rule also covers bulk table truncation.
CREATE TRIGGER trg_audit_events_no_truncate BEFORE TRUNCATE ON audit_events
FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_event_mutation();

-- NOTE: deployment must grant the application role only the required INSERT/SELECT
-- privileges, deny UPDATE/DELETE/TRUNCATE, and ensure it does not own this table.
-- Keep ownership with a separate migration/administrative role; table owners can
-- alter or disable triggers and therefore are outside this trigger's protection.
