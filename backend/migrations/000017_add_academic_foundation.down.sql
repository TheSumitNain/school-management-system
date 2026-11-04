-- V17 rollback is permitted only while the academic foundation is empty.
DO $v17_down$
BEGIN
    IF EXISTS (SELECT 1 FROM class_incharge_assignments)
       OR EXISTS (SELECT 1 FROM teacher_subject_assignments)
       OR EXISTS (SELECT 1 FROM student_enrollments)
       OR EXISTS (SELECT 1 FROM sections)
       OR EXISTS (SELECT 1 FROM subjects)
       OR EXISTS (SELECT 1 FROM classes)
       OR EXISTS (SELECT 1 FROM academic_years) THEN
        RAISE EXCEPTION
            'V17 rollback refused: academic configuration, assignments, or enrollment history exists; use a reviewed forward migration instead';
    END IF;
END;
$v17_down$;

DROP TRIGGER trg_guard_v17_enrollment_history ON student_enrollments;
DROP TRIGGER trg_snapshot_v17_enrollment_labels ON student_enrollments;
DROP TRIGGER trg_validate_v17_class_incharge_assignment ON class_incharge_assignments;
DROP TRIGGER trg_validate_v17_teacher_subject_assignment ON teacher_subject_assignments;
DROP TRIGGER trg_subjects_school_scope_immutable ON subjects;
DROP TRIGGER trg_sections_school_scope_immutable ON sections;
DROP TRIGGER trg_classes_school_scope_immutable ON classes;
DROP TRIGGER trg_academic_years_school_scope_immutable ON academic_years;

DROP TABLE class_incharge_assignments;
DROP TABLE teacher_subject_assignments;
DROP TABLE student_enrollments;
DROP TABLE sections;
DROP TABLE subjects;
DROP TABLE classes;
DROP TABLE academic_years;

DROP FUNCTION guard_v17_enrollment_history();
DROP FUNCTION snapshot_v17_enrollment_labels();
DROP FUNCTION validate_v17_teacher_assignment();
DROP FUNCTION guard_v17_school_scope_immutable();
