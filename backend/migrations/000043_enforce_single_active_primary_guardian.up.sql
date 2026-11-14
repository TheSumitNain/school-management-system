-- V43: enforce at most one active primary guardian contact per student.
-- Application transactions already serialize primary selection; this index
-- protects the invariant against alternate writers and future code paths.
DO $v43$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_guardians
         WHERE revoked_at IS NULL
           AND is_primary_contact = TRUE
         GROUP BY school_id, student_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'cannot enforce one active primary guardian: duplicate active primary contacts exist';
    END IF;
END;
$v43$;

CREATE UNIQUE INDEX ux_student_guardians_active_primary_per_student
    ON student_guardians (school_id, student_id)
    WHERE revoked_at IS NULL AND is_primary_contact = TRUE;
