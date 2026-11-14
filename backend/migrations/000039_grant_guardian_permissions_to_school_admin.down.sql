-- V39 is intentionally forward-only. Removing guardian permissions from
-- School Admin system roles would contradict the approved administrator
-- access policy and would require a separately reviewed forward migration.
DO $v39_down$
BEGIN
    RAISE EXCEPTION
        'V39 is forward-only: use a reviewed forward migration to change School Admin guardian permissions';
END;
$v39_down$;
