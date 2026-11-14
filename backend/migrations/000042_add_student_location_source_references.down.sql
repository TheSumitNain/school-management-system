-- V42 down: refuse to discard any imported or superseded source history.
DO $v42_down$
BEGIN
    IF (SELECT COUNT(*) FROM student_location_source_references) <> 36
       OR EXISTS (
           SELECT 1
             FROM student_location_source_references ref
             LEFT JOIN student_locations loc ON loc.id = ref.location_id
            WHERE ref.source_system <> 'IGOD'
               OR ref.is_current = FALSE
               OR loc.id IS NULL
               OR loc.source_system IS DISTINCT FROM ref.source_system
               OR loc.source_code IS DISTINCT FROM ref.source_code
               OR loc.name IS DISTINCT FROM ref.source_name
       ) THEN
        RAISE EXCEPTION 'V42 rollback would discard location source-reference history; use a forward recovery migration';
    END IF;
END;
$v42_down$;

DROP TRIGGER trg_guard_student_location_source_reference ON student_location_source_references;
DROP FUNCTION guard_student_location_source_reference();
DROP TABLE student_location_source_references;
