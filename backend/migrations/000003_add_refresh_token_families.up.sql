ALTER TABLE refresh_sessions
    ADD COLUMN family_id UUID NOT NULL DEFAULT gen_random_uuid();

CREATE INDEX refresh_sessions_family_id_idx
    ON refresh_sessions (family_id);
