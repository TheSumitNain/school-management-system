-- V12: connect authenticated Parent principals to explicitly portal-enabled
-- school guardian records, preserve relationship revocation history, and support
-- pre-account OTP challenges without creating/reserving principals prematurely.
--
-- This migration does not create parent accounts or send OTPs. Runtime services
-- must verify OTPs, apply persistent rate limits, and create links atomically.

ALTER TABLE guardians
    ADD COLUMN mobile_number_e164 VARCHAR(16),
    ADD COLUMN parent_portal_enabled_at TIMESTAMPTZ,
    ADD COLUMN parent_portal_enabled_by BIGINT
        REFERENCES auth_principals(id) ON DELETE RESTRICT,
    ADD CONSTRAINT guardians_mobile_e164_check
        CHECK (mobile_number_e164 IS NULL OR mobile_number_e164 ~ '^\+[1-9][0-9]{7,14}$'),
    ADD CONSTRAINT guardians_portal_enablement_actor_check
        CHECK (
            (parent_portal_enabled_at IS NULL AND parent_portal_enabled_by IS NULL)
            OR
            (parent_portal_enabled_at IS NOT NULL AND parent_portal_enabled_by IS NOT NULL)
        ),
    ADD CONSTRAINT guardians_portal_requires_verified_format_mobile
        CHECK (parent_portal_enabled_at IS NULL OR mobile_number_e164 IS NOT NULL);

CREATE INDEX ix_guardians_portal_mobile
    ON guardians (mobile_number_e164)
    WHERE parent_portal_enabled_at IS NOT NULL AND status = 'active';

-- A guardian contact record can be actively claimed by only one authenticated
-- Parent principal. A principal may link multiple guardian records across schools.
CREATE TABLE parent_guardian_links (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    principal_id BIGINT NOT NULL
        REFERENCES auth_principals(id) ON DELETE RESTRICT,
    guardian_id BIGINT NOT NULL,
    school_id BIGINT NOT NULL,
    link_method VARCHAR(24) NOT NULL
        CHECK (link_method IN ('otp_verified', 'school_approved')),
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by_principal_id BIGINT
        REFERENCES auth_principals(id) ON DELETE RESTRICT,
    revocation_reason VARCHAR(500),
    CONSTRAINT parent_guardian_links_guardian_school_fkey
        FOREIGN KEY (guardian_id, school_id)
        REFERENCES guardians (id, school_id) ON DELETE RESTRICT,
    CONSTRAINT parent_guardian_links_revocation_consistency
        CHECK (
            (revoked_at IS NULL AND revoked_by_principal_id IS NULL AND revocation_reason IS NULL)
            OR
            (revoked_at IS NOT NULL AND revoked_by_principal_id IS NOT NULL
             AND length(btrim(revocation_reason)) BETWEEN 1 AND 500)
        )
);

CREATE UNIQUE INDEX ux_parent_guardian_links_active_guardian
    ON parent_guardian_links (guardian_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_parent_guardian_links_active_principal
    ON parent_guardian_links (principal_id, school_id, guardian_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_parent_guardian_links_history
    ON parent_guardian_links (guardian_id, linked_at DESC);

-- Retain revoked school-record relationships instead of deleting/recreating
-- the same student/guardian pair and losing its history.
ALTER TABLE student_guardians
    ADD COLUMN revoked_at TIMESTAMPTZ,
    ADD COLUMN revoked_by_principal_id BIGINT
        REFERENCES auth_principals(id) ON DELETE RESTRICT,
    ADD COLUMN revocation_reason VARCHAR(500),
    ADD CONSTRAINT student_guardians_revocation_consistency
        CHECK (
            (revoked_at IS NULL AND revoked_by_principal_id IS NULL AND revocation_reason IS NULL)
            OR
            (revoked_at IS NOT NULL AND revoked_by_principal_id IS NOT NULL
             AND length(btrim(revocation_reason)) BETWEEN 1 AND 500)
        );

ALTER TABLE student_guardians
    DROP CONSTRAINT student_guardians_student_guardian_unique;
CREATE UNIQUE INDEX ux_student_guardians_active_pair
    ON student_guardians (student_id, guardian_id)
    WHERE revoked_at IS NULL;
CREATE INDEX ix_student_guardians_active_guardian
    ON student_guardians (school_id, guardian_id, student_id)
    WHERE revoked_at IS NULL;

-- Existing V4 challenges remain principal-bound. V12 additionally permits a
-- mobile-HMAC-bound challenge before an account exists; no raw mobile is stored.
ALTER TABLE parent_otp_challenges
    ALTER COLUMN principal_id DROP NOT NULL,
    ADD COLUMN mobile_lookup_hash CHAR(64),
    ADD CONSTRAINT parent_otp_challenge_identity_check
        CHECK (
            principal_id IS NOT NULL
            OR mobile_lookup_hash IS NOT NULL
        ),
    ADD CONSTRAINT parent_otp_mobile_hash_format
        CHECK (
            mobile_lookup_hash IS NULL
            OR mobile_lookup_hash ~ '^[0-9a-f]{64}$'
        );

DROP INDEX ux_parent_otp_unconsumed;
CREATE UNIQUE INDEX ux_parent_otp_unconsumed_principal
    ON parent_otp_challenges (principal_id)
    WHERE consumed_at IS NULL AND principal_id IS NOT NULL;
CREATE UNIQUE INDEX ux_parent_otp_unconsumed_mobile
    ON parent_otp_challenges (mobile_lookup_hash)
    WHERE consumed_at IS NULL AND mobile_lookup_hash IS NOT NULL;
CREATE INDEX ix_parent_otp_mobile_recent
    ON parent_otp_challenges (mobile_lookup_hash, created_at DESC)
    WHERE mobile_lookup_hash IS NOT NULL;

 
-- Extend the existing V4 OTP guards without weakening principal-bound history.
-- A pre-account challenge must carry a mobile HMAC; a bound challenge must
-- reference an actual Parent principal. The lookup hash is immutable.
CREATE OR REPLACE FUNCTION guard_parent_otp_challenge_history() RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'parent OTP challenge history cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.consumed_at IS NOT NULL OR NEW.attempt_count <> 0 THEN
            RAISE EXCEPTION 'new parent OTP challenges must be unconsumed with zero attempts';
        END IF;
        IF NEW.principal_id IS NULL AND NEW.mobile_lookup_hash IS NULL THEN
            RAISE EXCEPTION 'pre-account OTP challenge requires a mobile lookup HMAC';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.principal_id IS DISTINCT FROM OLD.principal_id
       OR NEW.mobile_lookup_hash IS DISTINCT FROM OLD.mobile_lookup_hash
       OR NEW.otp_hash IS DISTINCT FROM OLD.otp_hash
       OR NEW.expires_at IS DISTINCT FROM OLD.expires_at
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION 'parent OTP challenge identity and expiry are immutable';
    END IF;

    IF NEW.attempt_count < OLD.attempt_count THEN
        RAISE EXCEPTION 'parent OTP challenge attempt count cannot decrease';
    END IF;

    IF OLD.consumed_at IS NOT NULL AND NEW.consumed_at IS DISTINCT FROM OLD.consumed_at THEN
        RAISE EXCEPTION 'consumed parent OTP challenge cannot be reactivated or rewritten';
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE OR REPLACE FUNCTION validate_parent_otp_principal() RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF NEW.principal_id IS NULL THEN
        IF NEW.mobile_lookup_hash IS NULL THEN
            RAISE EXCEPTION 'pre-account OTP challenges require a mobile lookup HMAC';
        END IF;
    ELSIF NOT EXISTS (
        SELECT 1 FROM auth_principals
         WHERE id = NEW.principal_id AND principal_type = 'parent'
    ) THEN
        RAISE EXCEPTION 'OTP challenges require a parent principal';
    END IF;
    RETURN NEW;
END;
$v12$;

CREATE TABLE parent_otp_rate_limits (
    subject_type VARCHAR(8) NOT NULL
        CHECK (subject_type IN ('mobile', 'ip')),
    subject_hash CHAR(64) NOT NULL
        CHECK (subject_hash ~ '^[0-9a-f]{64}$'),
    window_started_at TIMESTAMPTZ NOT NULL,
    send_count SMALLINT NOT NULL DEFAULT 0 CHECK (send_count BETWEEN 0 AND 100),
    last_sent_at TIMESTAMPTZ,
    blocked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (subject_type, subject_hash, window_started_at),
    CHECK (last_sent_at IS NULL OR last_sent_at >= window_started_at),
    CHECK (blocked_until IS NULL OR blocked_until >= window_started_at)
);
CREATE INDEX ix_parent_otp_rate_limits_window
    ON parent_otp_rate_limits (window_started_at);

CREATE OR REPLACE FUNCTION guard_v12_relationship_history()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION '% relationship history cannot be deleted', TG_TABLE_NAME;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.student_id IS DISTINCT FROM OLD.student_id
       OR NEW.guardian_id IS DISTINCT FROM OLD.guardian_id
       OR NEW.relationship_type IS DISTINCT FROM OLD.relationship_type
       OR NEW.created_at IS DISTINCT FROM OLD.created_at THEN
        RAISE EXCEPTION '% relationship identity is immutable', TG_TABLE_NAME;
    END IF;

    IF OLD.revoked_at IS NOT NULL
       AND (NEW.revoked_at IS DISTINCT FROM OLD.revoked_at
            OR NEW.revoked_by_principal_id IS DISTINCT FROM OLD.revoked_by_principal_id
            OR NEW.revocation_reason IS DISTINCT FROM OLD.revocation_reason) THEN
        RAISE EXCEPTION '% relationship revocation is immutable', TG_TABLE_NAME;
    END IF;

    IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND (NEW.revoked_by_principal_id IS NULL
            OR NEW.revocation_reason IS NULL
            OR length(btrim(NEW.revocation_reason)) = 0) THEN
        RAISE EXCEPTION '% revocation requires an actor and reason', TG_TABLE_NAME;
    END IF;

    IF TG_TABLE_NAME = 'student_guardians'
       AND OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND NOT EXISTS (
           SELECT 1
             FROM auth_principals ap
             JOIN users u ON u.principal_id = ap.id
            WHERE ap.id = NEW.revoked_by_principal_id
              AND ap.principal_type = 'staff'
              AND ap.status = 'active'
              AND u.status = 'active'
              AND u.school_id = NEW.school_id
       ) THEN
        RAISE EXCEPTION 'student-guardian revocation requires active staff from the same school';
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE TRIGGER trg_student_guardians_v12_history
BEFORE UPDATE OR DELETE ON student_guardians
FOR EACH ROW EXECUTE FUNCTION guard_v12_relationship_history();

CREATE OR REPLACE FUNCTION guard_parent_guardian_link_history()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'parent guardian link history cannot be deleted';
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.revoked_at IS NOT NULL
           OR NEW.revoked_by_principal_id IS NOT NULL
           OR NEW.revocation_reason IS NOT NULL THEN
            RAISE EXCEPTION 'new parent guardian links must be active';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.principal_id IS DISTINCT FROM OLD.principal_id
       OR NEW.guardian_id IS DISTINCT FROM OLD.guardian_id
       OR NEW.school_id IS DISTINCT FROM OLD.school_id
       OR NEW.link_method IS DISTINCT FROM OLD.link_method
       OR NEW.linked_at IS DISTINCT FROM OLD.linked_at THEN
        RAISE EXCEPTION 'parent guardian link identity is immutable';
    END IF;

    IF OLD.revoked_at IS NOT NULL
       AND (NEW.revoked_at IS DISTINCT FROM OLD.revoked_at
            OR NEW.revoked_by_principal_id IS DISTINCT FROM OLD.revoked_by_principal_id
            OR NEW.revocation_reason IS DISTINCT FROM OLD.revocation_reason) THEN
        RAISE EXCEPTION 'parent guardian link revocation is immutable';
    END IF;

    IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND (NEW.revoked_by_principal_id IS NULL
            OR NEW.revocation_reason IS NULL
            OR length(btrim(NEW.revocation_reason)) = 0) THEN
        RAISE EXCEPTION 'parent guardian link revocation requires an actor and reason';
    END IF;

    IF OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL
       AND NEW.revoked_by_principal_id <> NEW.principal_id
       AND NOT EXISTS (
           SELECT 1
             FROM auth_principals ap
             JOIN users u ON u.principal_id = ap.id
            WHERE ap.id = NEW.revoked_by_principal_id
              AND ap.principal_type = 'staff'
              AND ap.status = 'active'
              AND u.status = 'active'
              AND u.school_id = NEW.school_id
       ) THEN
        RAISE EXCEPTION 'parent link revocation requires the linked Parent or active same-school staff';
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE TRIGGER trg_parent_guardian_links_history
BEFORE INSERT OR UPDATE OR DELETE ON parent_guardian_links
FOR EACH ROW EXECUTE FUNCTION guard_parent_guardian_link_history();

-- A link is valid only when the parent account has verified the exact E.164
-- mobile explicitly enabled on the active school guardian record.
CREATE OR REPLACE FUNCTION validate_parent_guardian_link()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
DECLARE
    v_principal_status VARCHAR(20);
    v_mobile VARCHAR(20);
    v_mobile_verified_at TIMESTAMPTZ;
    v_guardian_mobile VARCHAR(16);
    v_guardian_status VARCHAR(20);
    v_portal_enabled_at TIMESTAMPTZ;
BEGIN
    SELECT ap.status, pa.mobile_number, pa.mobile_verified_at
      INTO v_principal_status, v_mobile, v_mobile_verified_at
      FROM auth_principals ap
      JOIN parent_accounts pa ON pa.principal_id = ap.id
     WHERE ap.id = NEW.principal_id
       AND ap.principal_type = 'parent'
     FOR KEY SHARE OF ap, pa;

    IF NOT FOUND OR v_principal_status <> 'active' OR v_mobile_verified_at IS NULL THEN
        RAISE EXCEPTION 'parent guardian link requires an active, mobile-verified Parent principal';
    END IF;

    SELECT g.mobile_number_e164, g.status, g.parent_portal_enabled_at
      INTO v_guardian_mobile, v_guardian_status, v_portal_enabled_at
      FROM guardians g
     WHERE g.id = NEW.guardian_id
       AND g.school_id = NEW.school_id
     FOR KEY SHARE;

    IF NOT FOUND OR v_guardian_status <> 'active'
       OR v_portal_enabled_at IS NULL
       OR v_guardian_mobile IS DISTINCT FROM v_mobile THEN
        RAISE EXCEPTION 'guardian is not active, portal-enabled, or verified for this Parent mobile';
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE TRIGGER trg_validate_parent_guardian_link
BEFORE INSERT ON parent_guardian_links
FOR EACH ROW EXECUTE FUNCTION validate_parent_guardian_link();

-- Portal eligibility can be revoked immediately. It cannot be enabled unless
-- the guardian record is active and has a canonical mobile contact.
CREATE OR REPLACE FUNCTION guard_guardian_portal_eligibility()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
BEGIN
    IF NEW.parent_portal_enabled_at IS NOT NULL THEN
        IF NEW.status <> 'active' OR NEW.mobile_number_e164 IS NULL THEN
            RAISE EXCEPTION 'parent portal eligibility requires an active guardian with an E.164 mobile';
        END IF;

        IF NOT EXISTS (
            SELECT 1
              FROM auth_principals ap
              JOIN users u ON u.principal_id = ap.id
             WHERE ap.id = NEW.parent_portal_enabled_by
               AND ap.principal_type = 'staff'
               AND ap.status = 'active'
               AND u.status = 'active'
               AND u.school_id = NEW.school_id
        ) THEN
            RAISE EXCEPTION 'parent portal eligibility must be enabled by active staff from the guardian school';
        END IF;
    END IF;

    IF TG_OP = 'UPDATE'
       AND NEW.mobile_number_e164 IS DISTINCT FROM OLD.mobile_number_e164
       AND EXISTS (
           SELECT 1 FROM parent_guardian_links pgl
            WHERE pgl.guardian_id = NEW.id AND pgl.revoked_at IS NULL
       ) THEN
        RAISE EXCEPTION 'guardian mobile cannot change while an active Parent link exists; revoke the link first';
    END IF;

    IF TG_OP = 'UPDATE'
       AND (NEW.status <> 'active' OR NEW.parent_portal_enabled_at IS NULL)
       AND EXISTS (
           SELECT 1 FROM parent_guardian_links
            WHERE guardian_id = NEW.id AND revoked_at IS NULL
       ) THEN
        RAISE EXCEPTION 'revoke active Parent links with an authorized actor before disabling guardian portal eligibility';
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE TRIGGER trg_guardian_portal_eligibility
BEFORE INSERT OR UPDATE
ON guardians
FOR EACH ROW EXECUTE FUNCTION guard_guardian_portal_eligibility();

-- Keep relationship changes visible in the append-only security audit trail.
CREATE OR REPLACE FUNCTION audit_v12_parent_relationship_change()
RETURNS trigger
LANGUAGE plpgsql AS $v12$
DECLARE
    v_actor BIGINT;
    v_school BIGINT;
    v_resource_id BIGINT;
    v_action VARCHAR(120);
    v_metadata JSONB;
BEGIN
    IF TG_TABLE_NAME = 'parent_guardian_links' THEN
        v_actor := CASE WHEN TG_OP = 'INSERT' THEN NEW.principal_id ELSE NEW.revoked_by_principal_id END;
        v_school := NEW.school_id;
        v_resource_id := NEW.id;
        v_action := CASE WHEN TG_OP = 'INSERT' THEN 'parent.guardian_link_created' ELSE 'parent.guardian_link_revoked' END;
        v_metadata := jsonb_build_object('guardian_id', NEW.guardian_id, 'link_method', NEW.link_method);
    ELSE
        v_actor := NEW.revoked_by_principal_id;
        v_school := NEW.school_id;
        v_resource_id := NEW.id;
        v_action := 'school.student_guardian_link_revoked';
        v_metadata := jsonb_build_object('student_id', NEW.student_id, 'guardian_id', NEW.guardian_id);
    END IF;

    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL) THEN
        INSERT INTO audit_events (
            actor_principal_id, school_id, action, resource_type, resource_id, outcome, metadata
        ) VALUES (
            v_actor, v_school, v_action,
            CASE WHEN TG_TABLE_NAME = 'parent_guardian_links' THEN 'parent_guardian_link' ELSE 'student_guardian_link' END,
            v_resource_id::TEXT, 'success', v_metadata
        );
    END IF;

    RETURN NEW;
END;
$v12$;

CREATE TRIGGER trg_audit_parent_guardian_links
AFTER INSERT OR UPDATE ON parent_guardian_links
FOR EACH ROW EXECUTE FUNCTION audit_v12_parent_relationship_change();

CREATE TRIGGER trg_audit_student_guardian_revocation
AFTER UPDATE ON student_guardians
FOR EACH ROW EXECUTE FUNCTION audit_v12_parent_relationship_change();
