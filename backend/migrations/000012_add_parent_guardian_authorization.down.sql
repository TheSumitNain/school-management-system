-- V12 rollback is intentionally guarded. Do not erase parent access,
-- relationship revocation, or OTP/rate-limit history after it has been used.

DO $v12_rollback_guard$
BEGIN
    IF EXISTS (SELECT 1 FROM parent_guardian_links)
       OR EXISTS (SELECT 1 FROM parent_otp_rate_limits)
       OR EXISTS (SELECT 1 FROM parent_otp_challenges WHERE mobile_lookup_hash IS NOT NULL)
       OR EXISTS (SELECT 1 FROM student_guardians WHERE revoked_at IS NOT NULL)
       OR EXISTS (
           SELECT 1 FROM guardians
            WHERE mobile_number_e164 IS NOT NULL
               OR parent_portal_enabled_at IS NOT NULL
               OR parent_portal_enabled_by IS NOT NULL
       ) THEN
        RAISE EXCEPTION
            'V12 rollback refused: parent links, OTP/rate-limit history, guardian portal settings, or relationship revocations exist';
    END IF;
END;
$v12_rollback_guard$;

DROP TRIGGER IF EXISTS trg_audit_student_guardian_revocation ON student_guardians;
DROP TRIGGER IF EXISTS trg_audit_parent_guardian_links ON parent_guardian_links;
DROP TRIGGER IF EXISTS trg_guardian_portal_eligibility ON guardians;
DROP TRIGGER IF EXISTS trg_validate_parent_guardian_link ON parent_guardian_links;
DROP TRIGGER IF EXISTS trg_parent_account_link_identity ON parent_accounts;
DROP TRIGGER IF EXISTS trg_parent_guardian_links_history ON parent_guardian_links;
DROP TRIGGER IF EXISTS trg_student_guardians_v12_history ON student_guardians;

DROP FUNCTION IF EXISTS audit_v12_parent_relationship_change();
DROP FUNCTION IF EXISTS guard_guardian_portal_eligibility();
DROP FUNCTION IF EXISTS validate_parent_guardian_link();
DROP FUNCTION IF EXISTS guard_parent_account_link_identity();
DROP FUNCTION IF EXISTS guard_parent_guardian_link_history();
DROP FUNCTION IF EXISTS guard_v12_relationship_history();

DROP TABLE parent_guardian_links;
DROP TABLE parent_otp_rate_limits;

DROP INDEX IF EXISTS ix_parent_otp_mobile_recent;
DROP INDEX IF EXISTS ux_parent_otp_unconsumed_mobile;
DROP INDEX IF EXISTS ux_parent_otp_unconsumed_principal;
ALTER TABLE parent_otp_challenges
    DROP CONSTRAINT parent_otp_mobile_hash_format,
    DROP CONSTRAINT parent_otp_challenge_identity_check,
    DROP COLUMN mobile_lookup_hash,
    ALTER COLUMN principal_id SET NOT NULL;
CREATE UNIQUE INDEX ux_parent_otp_unconsumed
    ON parent_otp_challenges (principal_id)
    WHERE consumed_at IS NULL;

-- Restore the original V4 principal-bound OTP guards after dropping V12's
-- pre-account mobile-HMAC columns.
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
        RETURN NEW;
    END IF;

    IF NEW.id IS DISTINCT FROM OLD.id OR
       NEW.principal_id IS DISTINCT FROM OLD.principal_id OR
       NEW.otp_hash IS DISTINCT FROM OLD.otp_hash OR
       NEW.expires_at IS DISTINCT FROM OLD.expires_at OR
       NEW.created_at IS DISTINCT FROM OLD.created_at THEN
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
    IF NOT EXISTS (
        SELECT 1 FROM auth_principals
         WHERE id = NEW.principal_id AND principal_type = 'parent'
    ) THEN
        RAISE EXCEPTION 'OTP challenges require a parent principal';
    END IF;
    RETURN NEW;
END;
$v12$;

DROP INDEX IF EXISTS ix_student_guardians_active_guardian;
DROP INDEX IF EXISTS ux_student_guardians_active_pair;
ALTER TABLE student_guardians
    DROP CONSTRAINT student_guardians_revocation_consistency,
    DROP COLUMN revoked_at,
    DROP COLUMN revoked_by_principal_id,
    DROP COLUMN revocation_reason,
    ADD CONSTRAINT student_guardians_student_guardian_unique
        UNIQUE (student_id, guardian_id);

DROP INDEX IF EXISTS ix_guardians_portal_mobile;
ALTER TABLE guardians
    DROP CONSTRAINT guardians_mobile_e164_check,
    DROP CONSTRAINT guardians_portal_enablement_actor_check,
    DROP CONSTRAINT guardians_portal_requires_verified_format_mobile,
    DROP COLUMN mobile_number_e164,
    DROP COLUMN parent_portal_enabled_at,
    DROP COLUMN parent_portal_enabled_by;
