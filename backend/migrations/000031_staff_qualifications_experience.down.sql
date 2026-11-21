-- V31 rollback: remove school-scoped staff qualifications and experience.
ALTER TABLE public.role_permissions DISABLE TRIGGER trg_protect_system_role_permissions;
DELETE FROM public.role_permissions rp
 USING public.permissions p
 WHERE rp.permission_id = p.id
   AND p.permission_key IN ('staff.qualifications.read','staff.qualifications.manage','staff.qualifications.verify');
ALTER TABLE public.role_permissions ENABLE TRIGGER trg_protect_system_role_permissions;

DELETE FROM public.permissions
 WHERE permission_key IN ('staff.qualifications.read','staff.qualifications.manage','staff.qualifications.verify');

DROP TABLE IF EXISTS public.staff_experience;
DROP TABLE IF EXISTS public.staff_certifications;
DROP TABLE IF EXISTS public.staff_qualifications;
