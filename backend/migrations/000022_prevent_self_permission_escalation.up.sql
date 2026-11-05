-- V22: prevent a school administrator from elevating their own principal
-- through role assignment or direct permission overrides. These restrictions
-- are enforced in the database as well as the repository layer.

CREATE OR REPLACE FUNCTION public.prevent_self_permission_escalation()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = pg_catalog, public
AS $$
BEGIN
    IF TG_TABLE_NAME = 'principal_roles'
       AND NEW.assigned_by IS NOT NULL
       AND NEW.assigned_by = NEW.principal_id THEN
        RAISE EXCEPTION 'principals cannot assign roles to themselves';
    END IF;

    IF TG_TABLE_NAME = 'user_permission_overrides'
       AND NEW.granted_by = NEW.principal_id THEN
        RAISE EXCEPTION 'principals cannot grant permission overrides to themselves';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_prevent_self_principal_role_assignment
BEFORE INSERT OR UPDATE OF principal_id, assigned_by ON public.principal_roles
FOR EACH ROW EXECUTE FUNCTION public.prevent_self_permission_escalation();

CREATE TRIGGER trg_prevent_self_user_permission_override
BEFORE INSERT OR UPDATE OF principal_id, granted_by ON public.user_permission_overrides
FOR EACH ROW EXECUTE FUNCTION public.prevent_self_permission_escalation();
