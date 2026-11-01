DO $$
BEGIN
    RAISE EXCEPTION 'V4 is intentionally forward-only. Restore a verified pre-migration backup only after explicitly accepting loss of all post-backup writes.';
END;
$$;
