-- V40 down: refuse to remove canonical states once dependent hierarchy
-- rows exist, and refuse to drop source metadata after another source is added.
DO $v40$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM student_locations parent
          JOIN student_locations child ON child.parent_id = parent.id
         WHERE parent.source_system = 'IGOD'
           AND parent.level = 'state'
    ) THEN
        RAISE EXCEPTION 'cannot roll back V40 while dependent location rows exist';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM student_locations
         WHERE source_system IS NOT NULL
           AND NOT (source_system = 'IGOD' AND level = 'state')
    ) THEN
        RAISE EXCEPTION 'cannot roll back V40 after additional reference sources have been imported';
    END IF;
END
$v40$;

DELETE FROM student_locations
 WHERE source_system = 'IGOD'
   AND level = 'state';

DROP INDEX ux_student_locations_source_code;
ALTER TABLE student_locations
    DROP CONSTRAINT student_locations_source_pair,
    DROP COLUMN source_code,
    DROP COLUMN source_system;
