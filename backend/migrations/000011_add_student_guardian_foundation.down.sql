-- V11 is reversible only before school data has been created.
-- Refuse destructive rollback once any student, guardian, or link exists.

DO $v11_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_guardians)
       OR EXISTS (SELECT 1 FROM guardians)
       OR EXISTS (SELECT 1 FROM students) THEN
        RAISE EXCEPTION
            'V11 rollback refused: student or guardian data exists; use a reviewed forward migration instead';
    END IF;
END;
$v11_down$;

DROP TABLE student_guardians;
DROP TABLE guardians;
DROP TABLE students;

DROP FUNCTION guard_v11_school_scope_immutable();
