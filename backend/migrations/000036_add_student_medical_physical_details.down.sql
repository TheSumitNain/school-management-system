DO $v36_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_medical_details)
       OR EXISTS (SELECT 1 FROM student_medical_details_history)
       OR EXISTS (SELECT 1 FROM student_physical_details) THEN
        RAISE EXCEPTION
            'V36 rollback refused: student medical versions or physical measurements exist';
    END IF;
END;
$v36_down$;

DROP TRIGGER IF EXISTS trg_guard_student_medical_details ON student_medical_details;
DROP TRIGGER IF EXISTS trg_guard_student_medical_details_history ON student_medical_details_history;
DROP TRIGGER IF EXISTS trg_guard_student_physical_details ON student_physical_details;
DROP FUNCTION IF EXISTS guard_student_medical_details();
DROP FUNCTION IF EXISTS guard_student_health_history();
DROP TABLE student_physical_details;
DROP TABLE student_medical_details_history;
DROP TABLE student_medical_details;
