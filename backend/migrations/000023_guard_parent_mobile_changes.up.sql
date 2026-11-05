-- V23: keep the school-visible guardian mobile aligned with active Parent links.
-- A guardian's display contact and canonical E.164 portal mobile are one
-- identity boundary while an authenticated Parent link remains active.
CREATE OR REPLACE FUNCTION guard_guardian_portal_eligibility()
RETURNS trigger
LANGUAGE plpgsql AS $v23$
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
       AND (
           NEW.mobile_number IS DISTINCT FROM OLD.mobile_number
           OR NEW.mobile_number_e164 IS DISTINCT FROM OLD.mobile_number_e164
       )
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
$v23$;
