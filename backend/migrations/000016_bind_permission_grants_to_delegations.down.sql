-- V16 is intentionally forward-only. Removing exact delegation provenance
-- would make stale custom-role grants and ALLOW overrides eligible again.
DO $v16_down$
BEGIN
    RAISE EXCEPTION
        'V16 is forward-only: use a reviewed forward migration to change exact delegation linkage';
END;
$v16_down$;
