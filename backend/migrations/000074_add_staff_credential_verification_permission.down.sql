-- V74 rollback removes only the non-delegable permission and its system-role grants.
DROP TRIGGER trg_zz_grant_staff_credential_verification_permission ON public.schools;
DROP FUNCTION public.grant_staff_credential_verification_permission();

DELETE FROM public.role_permissions rp
USING public.permissions p, public.roles r
WHERE rp.permission_id=p.id
  AND rp.role_id=r.id
  AND p.permission_key='staff.credentials.verify'
  AND r.role_key='school_admin'
  AND r.principal_type='staff'
  AND r.scope_type='school'
  AND r.is_system=TRUE;

DELETE FROM public.permissions
WHERE permission_key='staff.credentials.verify';
