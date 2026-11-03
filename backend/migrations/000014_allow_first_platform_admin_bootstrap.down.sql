DO $v14_down$
BEGIN
    RAISE EXCEPTION
        'V14 is forward-only: do not remove the first-Platform-Admin bootstrap guard from a live database';
END;
$v14_down$;
