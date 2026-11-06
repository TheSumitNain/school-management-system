DROP TRIGGER IF EXISTS trg_prevent_self_user_permission_override ON public.user_permission_overrides;
DROP TRIGGER IF EXISTS trg_prevent_self_principal_role_assignment ON public.principal_roles;
DROP FUNCTION IF EXISTS public.prevent_self_permission_escalation();
