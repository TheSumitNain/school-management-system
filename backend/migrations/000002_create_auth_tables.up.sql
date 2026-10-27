
CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    school_id BIGINT NOT NULL
        REFERENCES schools(id) ON DELETE RESTRICT,

    username VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (length(btrim(username)) > 0),
    CHECK (length(password_hash) > 0)
);

CREATE UNIQUE INDEX users_school_username_unique
    ON users (school_id, LOWER(username));

CREATE TABLE refresh_sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,

    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (expires_at > created_at)
);

CREATE INDEX refresh_sessions_user_id_idx
    ON refresh_sessions (user_id);

CREATE INDEX refresh_sessions_expires_at_idx
    ON refresh_sessions (expires_at);
