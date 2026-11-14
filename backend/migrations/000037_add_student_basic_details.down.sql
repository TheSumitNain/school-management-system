DO $v37_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_basic_details) THEN
        RAISE EXCEPTION
            'V37 rollback refused: student basic profile details exist';
    END IF;
END;
$v37_down$;

DROP TRIGGER IF EXISTS trg_guard_student_basic_details ON student_basic_details;
DROP FUNCTION IF EXISTS guard_student_basic_details();
DROP TABLE student_basic_details;
