DO $v36_down$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_guardians
         GROUP BY student_id, guardian_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION
            'V36 rollback refused: repeated guardian relationship periods exist';
    END IF;
END;
$v36_down$;

DROP INDEX IF EXISTS ux_student_guardians_active_pair;

ALTER TABLE student_guardians
    ADD CONSTRAINT student_guardians_unique UNIQUE (student_id, guardian_id);
