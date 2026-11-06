-- Refuse rollback if any purpose value cannot fit the original V7 width.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM auth_mfa_challenges
         WHERE length(purpose) > 20
    ) THEN
        RAISE EXCEPTION
            'cannot shrink MFA challenge purpose while values exceed 20 characters';
    END IF;
END;
$$;

ALTER TABLE auth_mfa_challenges
    ALTER COLUMN purpose TYPE VARCHAR(20);
