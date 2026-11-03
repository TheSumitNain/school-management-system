-- V16: bind custom-role grants and staff permission overrides to the exact
-- delegation row that authorized them. Revoking and re-issuing a delegation
-- must never reactivate grants linked to the old delegation ID.

-- Existing grants cannot be safely attributed to a particular delegation:
-- V4 recorded only the school/permission pair. Refuse to guess provenance.
DO $v16_preflight$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM public.role_permissions rp
          JOIN public.roles r ON r.id = rp.role_id
         WHERE r.is_system = FALSE
    ) THEN
        RAISE EXCEPTION
            'V16 preflight failed: custom-role permission grants already exist without exact delegation provenance';
    END IF;

    IF EXISTS (SELECT 1 FROM public.user_permission_overrides) THEN
        RAISE EXCEPTION
            'V16 preflight failed: user permission overrides already exist without exact delegation provenance';
    END IF;
END;
$v16_preflight$;

ALTER TABLE public.role_permissions
    ADD COLUMN delegation_id BIGINT
    REFERENCES public.role_permission_delegations(id) ON DELETE RESTRICT;

ALTER TABLE public.user_permission_overrides
    ADD COLUMN delegation_id BIGINT
    REFERENCES public.role_permission_delegations(id) ON DELETE RESTRICT;

CREATE INDEX ix_role_permissions_delegation
    ON public.role_permissions (delegation_id)
    WHERE delegation_id IS NOT NULL;

CREATE INDEX ix_user_permission_overrides_delegation
    ON public.user_permission_overrides (delegation_id)
    WHERE delegation_id IS NOT NULL;

-- System-role bundles are platform-controlled and must never reference a
-- school delegation. Every custom school-role grant must reference the exact,
-- active delegation with can_assign_to_roles=true.
CREATE OR REPLACE FUNCTION public.validate_role_permission_scope()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v16$
DECLARE
    v_role RECORD;
    v_permission RECORD;
    v_delegation RECORD;
BEGIN
    SELECT id, school_id, scope_type, is_system
      INTO v_role
      FROM public.roles
     WHERE id = NEW.role_id;

    SELECT id, scope_type, is_active
      INTO v_permission
      FROM public.permissions
     WHERE id = NEW.permission_id;

    IF v_role.id IS NULL OR v_permission.id IS NULL
       OR v_role.scope_type <> v_permission.scope_type
       OR NOT v_permission.is_active THEN
        RAISE EXCEPTION 'role and active permission scopes must match';
    END IF;

    IF v_role.is_system THEN
        IF NEW.delegation_id IS NOT NULL THEN
            RAISE EXCEPTION 'system-role permission grants cannot reference school delegations';
        END IF;
        RETURN NEW;
    END IF;

    IF v_role.scope_type <> 'school' OR NEW.delegation_id IS NULL THEN
        RAISE EXCEPTION 'custom school-role permission grants require an exact delegation ID';
    END IF;

    SELECT id
      INTO v_delegation
      FROM public.role_permission_delegations
     WHERE id = NEW.delegation_id
       AND school_id = v_role.school_id
       AND permission_id = NEW.permission_id
       AND can_assign_to_roles = TRUE
       AND revoked_at IS NULL;

    IF v_delegation.id IS NULL THEN
        RAISE EXCEPTION 'custom role permission grant does not match its exact active delegation';
    END IF;

    RETURN NEW;
END;
$v16$;

-- Every override remains subject to same-school School Admin authority. The
-- exact delegation ID is now checked instead of merely finding any active
-- delegation for the same school/permission pair.
CREATE OR REPLACE FUNCTION public.validate_user_permission_override()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $v16$
DECLARE
    v_permission_id BIGINT;
    v_grantor_school BIGINT;
    v_authority_permission_id BIGINT;
    v_delegation_id BIGINT;
BEGIN
    SELECT id INTO v_permission_id
      FROM public.permissions
     WHERE id = NEW.permission_id
       AND scope_type = 'school'
       AND is_active = TRUE
       AND is_delegable = TRUE;

    IF v_permission_id IS NULL THEN
        RAISE EXCEPTION 'user permission overrides require an active, delegable school permission';
    END IF;

    SELECT school_id INTO v_grantor_school
      FROM public.users
     WHERE principal_id = NEW.granted_by;

    SELECT id INTO v_authority_permission_id
      FROM public.permissions
     WHERE permission_key = 'permissions.override'
       AND scope_type = 'school'
       AND is_active = TRUE;

    IF v_grantor_school IS DISTINCT FROM NEW.school_id OR NOT EXISTS (
        SELECT 1
          FROM public.auth_principals ap
         WHERE ap.id = NEW.granted_by
           AND ap.principal_type = 'staff'
           AND ap.status = 'active'
    ) OR NOT EXISTS (
        SELECT 1
          FROM public.principal_roles pr
          JOIN public.roles r ON r.id = pr.role_id
          JOIN public.role_permissions rp ON rp.role_id = r.id
         WHERE pr.principal_id = NEW.granted_by
           AND pr.revoked_at IS NULL
           AND r.school_id = NEW.school_id
           AND r.scope_type = 'school'
           AND r.principal_type = 'staff'
           AND r.role_key = 'school_admin'
           AND rp.permission_id = v_authority_permission_id
    ) OR v_authority_permission_id IS NULL OR EXISTS (
        SELECT 1
          FROM public.user_permission_overrides actor_override
         WHERE actor_override.principal_id = NEW.granted_by
           AND actor_override.school_id = NEW.school_id
           AND actor_override.permission_id = v_authority_permission_id
           AND actor_override.effect = 'deny'
    ) THEN
        RAISE EXCEPTION 'override grantor must be an active same-school School Admin with effective permissions.override permission';
    END IF;

    SELECT id INTO v_delegation_id
      FROM public.role_permission_delegations
     WHERE id = NEW.delegation_id
       AND school_id = NEW.school_id
       AND permission_id = NEW.permission_id
       AND can_override_users = TRUE
       AND revoked_at IS NULL;

    IF NEW.delegation_id IS NULL OR v_delegation_id IS NULL THEN
        RAISE EXCEPTION 'user permission override does not match its exact active delegation';
    END IF;

    RETURN NEW;
END;
$v16$;

COMMENT ON COLUMN public.role_permissions.delegation_id IS
'Exact school delegation that authorized this custom-role permission grant. Revoked delegation IDs never reactivate after a later re-grant.';

COMMENT ON COLUMN public.user_permission_overrides.delegation_id IS
'Exact school delegation that authorized this user permission override. Revoked delegation IDs never reactivate after a later re-grant.';
