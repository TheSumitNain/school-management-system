DO $v24_down$
BEGIN
    RAISE EXCEPTION
        'V24 is forward-only: Platform Admin lifecycle assignments and deactivation history must not be rolled back on a live database';
END;
$v24_down$;
