-- Address/location records may contain school data and must not be
-- silently discarded by a schema rollback.
DO $v34_down$
BEGIN
    IF EXISTS (SELECT 1 FROM student_addresses)
       OR EXISTS (SELECT 1 FROM student_locations) THEN
        RAISE EXCEPTION
            'V34 rollback refused: student address or location records exist';
    END IF;
END;
$v34_down$;

DROP TRIGGER IF EXISTS trg_guard_student_address_location ON student_addresses;
DROP FUNCTION IF EXISTS guard_student_address_location();
DROP TRIGGER IF EXISTS trg_guard_student_location_hierarchy ON student_locations;
DROP FUNCTION IF EXISTS guard_student_location_hierarchy();
DROP TABLE student_addresses;
DROP TABLE student_locations;
