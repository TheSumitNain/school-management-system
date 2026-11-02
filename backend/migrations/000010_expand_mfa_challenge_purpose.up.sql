-- Allow the descriptive recovery_regeneration challenge purpose.
-- Keep a bounded varchar rather than an unbounded text column.
ALTER TABLE auth_mfa_challenges
    ALTER COLUMN purpose TYPE VARCHAR(32);
