DO $v36_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_medical_details)
       OR EXISTS (SELECT 1 FROM student_physical_details) THEN
        RAISE EXCEPTION
            'V36 rollback refused: student medical or physical profile records exist';
    END IF;
END;
$v36_down$;

DROP TRIGGER IF EXISTS trg_guard_student_medical_details ON student_medical_details;
DROP TRIGGER IF EXISTS trg_guard_student_physical_details ON student_physical_details;
DROP FUNCTION IF EXISTS guard_student_health_profile();
DROP TABLE student_physical_details;
DROP TABLE student_medical_details;
