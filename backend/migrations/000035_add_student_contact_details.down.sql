DO $v35_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_contact_details) THEN
        RAISE EXCEPTION
            'V35 rollback refused: student contact detail records exist';
    END IF;
END;
$v35_down$;

DROP TRIGGER IF EXISTS trg_guard_student_contact_details ON student_contact_details;
DROP FUNCTION IF EXISTS guard_student_contact_details();
DROP TABLE student_contact_details;
