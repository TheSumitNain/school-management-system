-- V24: controlled Platform Admin lifecycle database safeguards.
-- The first administrator remains a NULL-actor, one-time CLI bootstrap.
-- Every later Platform Admin assignment requires a different active Platform
-- Admin with platform.admins.manage. Deactivation of the last active admin is
-- serialized and rejected at the database boundary.

INSERT INTO public.permissions (permission_key, description, scope_type, is_delegable)
VALUES ('platform.admins.manage', 'Create and deactivate Platform Admin accounts', 'platform', FALSE);

-- System-role permission bundles are immutable at runtime. This migration is
-- the controlled seed point for the new lifecycle permission.
ALTER TABLE public.role_permissions DISABLE TRIGGER trg_protect_system_role_permissions;
INSERT INTO public.role_permissions (role_id, permission_id)
SELECT r.id, p.id
  FROM public.roles r
  CROSS JOIN public.permissions p
 WHERE r.principal_type = 'platform_admin'
   AND r.scope_type = 'platform'
   AND r.role_key = 'platform_admin'
   AND r.is_system = TRUE
   AND p.permission_key = 'platform.admins.manage';
ALTER TABLE public.role_permissions ENABLE TRIGGER trg_protect_system_role_permissions;

-- V14: allow exactly one controlled first Platform Admin bootstrap.
--
-- The initial Platform Admin cannot be assigned by an authenticated actor because
-- no Platform Admin exists yet. This narrowly scoped exception permits a NULL
-- assigned_by only for the platform_admin system role, only while no Platform
-- Admin account or role-assignment history exists. A transaction advisory lock
-- serializes concurrent bootstrap attempts. Subsequent assignments continue to
-- require the existing authenticated actor rules.

CREATE OR REPLACE FUNCTION public.validate_principal_role_assignment()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v14$
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
      FROM public.auth_principals
     WHERE id = NEW.principal_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'role assignment target principal does not exist';
    END IF;
    IF v_target_status <> 'active' THEN
        RAISE EXCEPTION 'role assignment target principal must be active';
    END IF;

    SELECT id, school_id, principal_type, scope_type, role_key, is_system
      INTO v_role FROM public.roles WHERE id = NEW.role_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'role assignment target role does not exist';
    END IF;
    IF v_type IS DISTINCT FROM v_role.principal_type THEN
        RAISE EXCEPTION 'principal type does not match role type';
    END IF;
    IF v_type = 'staff' THEN
        SELECT school_id INTO v_school_id
          FROM public.users WHERE principal_id = NEW.principal_id;
        IF v_school_id IS DISTINCT FROM v_role.school_id THEN
            RAISE EXCEPTION 'staff principal and role must belong to the same school';
        END IF;
    END IF;
    IF v_type = 'platform_admin' AND v_role.scope_type <> 'platform' THEN
        RAISE EXCEPTION 'Platform Admin may only receive platform roles';
    ELSIF v_type = 'parent' AND v_role.scope_type <> 'global' THEN
        RAISE EXCEPTION 'Parent may only receive global roles';
    END IF;

    -- One-time bootstrap exception. It is intentionally limited to the
    -- platform_admin system role and NULL assigned_by. The advisory lock
    -- prevents two concurrent transactions from both becoming the first admin.
    IF v_type = 'platform_admin'
       AND v_role.role_key = 'platform_admin'
       AND v_role.scope_type = 'platform'
       AND v_role.is_system IS TRUE
       AND NEW.assigned_by IS NULL THEN
        PERFORM pg_catalog.pg_advisory_xact_lock(7414, 1);

        IF EXISTS (
            SELECT 1
              FROM public.platform_admin_accounts pa
             WHERE pa.principal_id <> NEW.principal_id
        ) OR EXISTS (
            SELECT 1
              FROM public.principal_roles pr
              JOIN public.roles existing_role ON existing_role.id = pr.role_id
             WHERE existing_role.principal_type = 'platform_admin'
               AND existing_role.scope_type = 'platform'
               AND existing_role.role_key = 'platform_admin'
        ) THEN
            RAISE EXCEPTION 'the first Platform Admin has already been bootstrapped';
        END IF;

        RETURN NEW;
    END IF;

    -- Parent role assignment is created only by the verified-parent OTP flow.
    -- The parent subtype must already be verified; assigned_by remains NULL
    -- because this is a system-mediated identity activation, not staff delegation.
    IF v_type = 'parent' AND v_role.role_key = 'parent' AND NEW.assigned_by IS NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM public.parent_accounts
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
      FROM public.auth_principals
     WHERE id = NEW.assigned_by AND status = 'active';
    IF NOT FOUND THEN
        RAISE EXCEPTION 'assigning principal must exist and be active';
    END IF;

    IF v_type = 'platform_admin'
       AND v_role.role_key = 'platform_admin'
       AND v_role.scope_type = 'platform'
       AND v_role.is_system IS TRUE THEN
        IF NEW.assigned_by IS NULL OR NEW.assigned_by = NEW.principal_id THEN
            RAISE EXCEPTION 'Platform Admin creation requires a different active Platform Admin actor';
        END IF;

        SELECT principal_type INTO v_actor_type
          FROM public.auth_principals
         WHERE id = NEW.assigned_by AND status = 'active';
        IF NOT FOUND OR v_actor_type <> 'platform_admin' OR NOT EXISTS (
            SELECT 1
              FROM public.principal_roles apr
              JOIN public.roles ar ON ar.id = apr.role_id
              JOIN public.role_permissions arp ON arp.role_id = ar.id
              JOIN public.permissions ap ON ap.id = arp.permission_id
             WHERE apr.principal_id = NEW.assigned_by
               AND apr.revoked_at IS NULL
               AND ar.scope_type = 'platform'
               AND ar.principal_type = 'platform_admin'
               AND ar.role_key = 'platform_admin'
               AND ap.permission_key = 'platform.admins.manage'
               AND ap.scope_type = 'platform'
               AND ap.is_active = TRUE
        ) THEN
            RAISE EXCEPTION 'Platform Admin assignment requires platform.admins.manage';
        END IF;
    ELSIF v_role.role_key = 'school_admin' THEN
        IF v_actor_type <> 'platform_admin' OR NOT EXISTS (
            SELECT 1
              FROM public.principal_roles apr
              JOIN public.roles ar ON ar.id = apr.role_id
              JOIN public.role_permissions arp ON arp.role_id = ar.id
              JOIN public.permissions ap ON ap.id = arp.permission_id
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
        SELECT school_id INTO v_actor_school_id
          FROM public.users WHERE principal_id = NEW.assigned_by;
        SELECT id INTO v_permission_id
          FROM public.permissions
         WHERE permission_key = 'roles.assign'
           AND scope_type = 'school' AND is_active = TRUE;
        IF v_actor_school_id IS DISTINCT FROM v_role.school_id OR NOT EXISTS (
            SELECT 1
              FROM public.principal_roles pr
              JOIN public.roles ar ON ar.id = pr.role_id
              JOIN public.role_permissions arp ON arp.role_id = ar.id
             WHERE pr.principal_id = NEW.assigned_by
               AND pr.revoked_at IS NULL
               AND ar.scope_type = 'school'
               AND ar.school_id = v_role.school_id
               AND ar.principal_type = 'staff'
               AND ar.role_key = 'school_admin'
               AND arp.permission_id = v_permission_id
        ) OR v_permission_id IS NULL OR EXISTS (
            SELECT 1
              FROM public.user_permission_overrides upo
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
$v14$;

COMMENT ON FUNCTION public.validate_principal_role_assignment() IS
'Validates principal role assignments. Allows a single serialized first Platform Admin bootstrap with NULL assigned_by; all later assignments require authenticated authority.';


CREATE OR REPLACE FUNCTION public.guard_last_active_platform_admin()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v24$
BEGIN
    IF OLD.principal_type = 'platform_admin'
       AND OLD.status = 'active'
       AND NEW.status <> 'active' THEN
        PERFORM pg_catalog.pg_advisory_xact_lock(7414, 2);

        IF NOT EXISTS (
            SELECT 1
              FROM public.auth_principals ap
              JOIN public.platform_admin_accounts pa ON pa.principal_id = ap.id
              JOIN public.principal_roles pr ON pr.principal_id = ap.id
                                               AND pr.revoked_at IS NULL
              JOIN public.roles r ON r.id = pr.role_id
                                 AND r.principal_type = 'platform_admin'
                                 AND r.scope_type = 'platform'
                                 AND r.role_key = 'platform_admin'
             WHERE ap.id <> OLD.id
               AND ap.principal_type = 'platform_admin'
               AND ap.status = 'active'
        ) THEN
            RAISE EXCEPTION 'cannot deactivate the last active Platform Admin';
        END IF;
    END IF;
    RETURN NEW;
END;
$v24$;

CREATE TRIGGER trg_guard_last_active_platform_admin
BEFORE UPDATE OF status ON public.auth_principals
FOR EACH ROW
EXECUTE FUNCTION public.guard_last_active_platform_admin();

COMMENT ON FUNCTION public.guard_last_active_platform_admin() IS
'Serializes Platform Admin deactivation and prevents disabling the last active Platform Admin.';
