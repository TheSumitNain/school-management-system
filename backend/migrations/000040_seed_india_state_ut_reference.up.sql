-- V40: seed India's current State/Union Territory directory.
-- Source: National Portal of India, State/UT web directory (IGOD):
-- https://www.india.gov.in/directory/web-directory/state-uts
-- The portal reports 36 State/UT entries and attributes the directory to IGOD.
-- Checked 2026-11-14. District/subdistrict/locality data is deliberately not guessed.

ALTER TABLE student_locations
    ADD COLUMN source_system VARCHAR(30),
    ADD COLUMN source_code VARCHAR(50),
    ADD CONSTRAINT student_locations_source_pair
        CHECK (
            (source_system IS NULL AND source_code IS NULL)
            OR (
                source_system IS NOT NULL
                AND source_code IS NOT NULL
                AND school_id IS NULL
                AND source_system ~ '^[A-Z0-9_]{2,30}
        );

CREATE UNIQUE INDEX ux_student_locations_source_code
    ON student_locations (source_system, source_code)
    WHERE source_system IS NOT NULL;

WITH reference_states(source_code, name) AS (
    VALUES
        ('AN', 'Andaman and Nicobar Islands'),
        ('AP', 'Andhra Pradesh'),
        ('AR', 'Arunachal Pradesh'),
        ('AS', 'Assam'),
        ('BR', 'Bihar'),
        ('CH', 'Chandigarh'),
        ('CG', 'Chhattisgarh'),
        ('ND', 'Dadra and Nagar Haveli and Daman and Diu'),
        ('DL', 'Delhi'),
        ('GA', 'Goa'),
        ('GJ', 'Gujarat'),
        ('HR', 'Haryana'),
        ('HP', 'Himachal Pradesh'),
        ('JK', 'Jammu and Kashmir'),
        ('JH', 'Jharkhand'),
        ('KA', 'Karnataka'),
        ('KL', 'Kerala'),
        ('LA', 'Ladakh'),
        ('LD', 'Lakshadweep'),
        ('MP', 'Madhya Pradesh'),
        ('MH', 'Maharashtra'),
        ('MN', 'Manipur'),
        ('ML', 'Meghalaya'),
        ('MZ', 'Mizoram'),
        ('NL', 'Nagaland'),
        ('OD', 'Odisha'),
        ('PY', 'Puducherry'),
        ('PB', 'Punjab'),
        ('RJ', 'Rajasthan'),
        ('SK', 'Sikkim'),
        ('TN', 'Tamil Nadu'),
        ('TS', 'Telangana'),
        ('TR', 'Tripura'),
        ('UP', 'Uttar Pradesh'),
        ('UK', 'Uttarakhand'),
        ('WB', 'West Bengal')
)
INSERT INTO student_locations
    (school_id, parent_id, country_code, level, name, source_system, source_code)
SELECT NULL, NULL, 'IN', 'state', reference_states.name, 'IGOD', reference_states.source_code
  FROM reference_states
 WHERE NOT EXISTS (
     SELECT 1
       FROM student_locations existing
      WHERE existing.country_code = 'IN'
        AND existing.level = 'state'
        AND existing.parent_id IS NULL
        AND existing.normalized_name = LOWER(BTRIM(reference_states.name))
 )
ON CONFLICT DO NOTHING;

COMMENT ON COLUMN student_locations.source_system IS
    'Authoritative reference-data source identifier for shared canonical locations; NULL for school-owned/manual rows.';
COMMENT ON COLUMN student_locations.source_code IS
    'Stable code assigned by source_system; NULL for school-owned/manual rows.';

                AND length(BTRIM(source_code)) BETWEEN 1 AND 50
            )
        );

CREATE UNIQUE INDEX ux_student_locations_source_code
    ON student_locations (source_system, source_code)
    WHERE source_system IS NOT NULL;

WITH reference_states(source_code, name) AS (
    VALUES
        ('AN', 'Andaman and Nicobar Islands'),
        ('AP', 'Andhra Pradesh'),
        ('AR', 'Arunachal Pradesh'),
        ('AS', 'Assam'),
        ('BR', 'Bihar'),
        ('CH', 'Chandigarh'),
        ('CG', 'Chhattisgarh'),
        ('ND', 'Dadra and Nagar Haveli and Daman and Diu'),
        ('DL', 'Delhi'),
        ('GA', 'Goa'),
        ('GJ', 'Gujarat'),
        ('HR', 'Haryana'),
        ('HP', 'Himachal Pradesh'),
        ('JK', 'Jammu and Kashmir'),
        ('JH', 'Jharkhand'),
        ('KA', 'Karnataka'),
        ('KL', 'Kerala'),
        ('LA', 'Ladakh'),
        ('LD', 'Lakshadweep'),
        ('MP', 'Madhya Pradesh'),
        ('MH', 'Maharashtra'),
        ('MN', 'Manipur'),
        ('ML', 'Meghalaya'),
        ('MZ', 'Mizoram'),
        ('NL', 'Nagaland'),
        ('OD', 'Odisha'),
        ('PY', 'Puducherry'),
        ('PB', 'Punjab'),
        ('RJ', 'Rajasthan'),
        ('SK', 'Sikkim'),
        ('TN', 'Tamil Nadu'),
        ('TS', 'Telangana'),
        ('TR', 'Tripura'),
        ('UP', 'Uttar Pradesh'),
        ('UK', 'Uttarakhand'),
        ('WB', 'West Bengal')
)
INSERT INTO student_locations
    (school_id, parent_id, country_code, level, name, source_system, source_code)
SELECT NULL, NULL, 'IN', 'state', reference_states.name, 'IGOD', reference_states.source_code
  FROM reference_states
 WHERE NOT EXISTS (
     SELECT 1
       FROM student_locations existing
      WHERE existing.country_code = 'IN'
        AND existing.level = 'state'
        AND existing.parent_id IS NULL
        AND existing.normalized_name = LOWER(BTRIM(reference_states.name))
 )
ON CONFLICT DO NOTHING;

COMMENT ON COLUMN student_locations.source_system IS
    'Authoritative reference-data source identifier for shared canonical locations; NULL for school-owned/manual rows.';
COMMENT ON COLUMN student_locations.source_code IS
    'Stable code assigned by source_system; NULL for school-owned/manual rows.';
