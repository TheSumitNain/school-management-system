-- V15 is intentionally forward-only. Removing this state would silently
-- disable mandatory first-login password changes for provisioned staff.
DO $v15_down$
BEGIN
    RAISE EXCEPTION
        'V15 is forward-only: use a reviewed forward migration to change staff password lifecycle enforcement';
END;
$v15_down$;
