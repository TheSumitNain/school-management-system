-- V13 is intentionally forward-only. System roles provisioned for schools
-- created after this migration are security-critical and must not be silently
-- removed or leave schools without their baseline authorization roles.
DO $v13_down$
BEGIN
    RAISE EXCEPTION
        'V13 is forward-only: use a reviewed forward migration to change school system-role provisioning';
END;
$v13_down$;
