-- V45: retain a single authoritative audit event for student guardian revocation.
-- The V12 row trigger already records revocations. Extend its metadata with a
-- reason-presence flag; the repository must not insert a duplicate event.

CREATE OR REPLACE FUNCTION audit_v12_parent_relationship_change()
RETURNS trigger
LANGUAGE plpgsql AS $v45$
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
        v_metadata := jsonb_build_object(
            'student_id', NEW.student_id,
            'guardian_id', NEW.guardian_id,
            'has_revocation_reason', NEW.revocation_reason IS NOT NULL AND length(btrim(NEW.revocation_reason)) > 0
        );
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
$v45$;
