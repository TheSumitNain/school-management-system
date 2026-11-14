package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type StudentLocation struct {
	ID           int64     `json:"id"`
	SchoolID     *int64    `json:"school_id,omitempty"`
	ParentID     *int64    `json:"parent_id,omitempty"`
	CountryCode  string    `json:"country_code"`
	Level        string    `json:"level"`
	Name         string    `json:"name"`
	LocalityType *string   `json:"locality_type,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StudentAddress struct {
	ID            int64     `json:"id"`
	SchoolID      int64     `json:"school_id"`
	StudentID     int64     `json:"student_id"`
	AddressType   string    `json:"address_type"`
	AddressLine1  *string   `json:"address_line_1,omitempty"`
	AddressLine2  *string   `json:"address_line_2,omitempty"`
	AreaMohalla   *string   `json:"area_mohalla,omitempty"`
	Landmark      *string   `json:"landmark,omitempty"`
	PostalCode    *string   `json:"postal_code,omitempty"`
	LocationID   *int64    `json:"location_id,omitempty"`
	LocationName *string   `json:"location_name,omitempty"`
	LocationLevel *string   `json:"location_level,omitempty"`
	LocalityType *string   `json:"locality_type,omitempty"`
	StateLocationID *int64 `json:"state_location_id,omitempty"`
	DistrictLocationID *int64 `json:"district_location_id,omitempty"`
	SubdistrictLocationID *int64 `json:"subdistrict_location_id,omitempty"`
	SameAsCurrent bool      `json:"same_as_current"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func nextStudentLocationLevel(parentLevel string) string {
	switch parentLevel {
	case "state":
		return "district"
	case "district":
		return "subdistrict"
	case "subdistrict":
		return "locality"
	default:
		return ""
	}
}

func normalizeAddressText(value *string, max int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, nil
	}
	if len(normalized) > max {
		return nil, ErrInvalidAcademicInput
	}
	return &normalized, nil
}

// ListStudentLocations returns active canonical locations plus active
// locations owned by the authenticated school. A nil parent selects states.
func (r *AcademicRepository) ListStudentLocations(
	ctx context.Context, actorID, schoolID int64, parentID *int64, level, search string,
) ([]StudentLocation, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	level = strings.TrimSpace(strings.ToLower(level))
	search = strings.TrimSpace(search)
	if (level != "" && level != "state" && level != "district" && level != "subdistrict" && level != "locality") ||
		len(search) > 100 || (parentID != nil && *parentID <= 0) {
		return nil, ErrInvalidAcademicInput
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, school_id, parent_id, country_code, level, name,
		       locality_type, is_active, created_at, updated_at
		  FROM student_locations
		 WHERE (school_id IS NULL OR school_id = $1)
		   AND is_active = TRUE
		   AND (($2::bigint IS NULL AND parent_id IS NULL) OR parent_id = $2)
		   AND ($3 = '' OR level = $3)
		   AND ($4 = '' OR name ILIKE '%' || $4 || '%')
		 ORDER BY CASE WHEN school_id IS NULL THEN 0 ELSE 1 END,
		          normalized_name, id
	`, schoolID, parentID, level, search)
	if err != nil {
		return nil, fmt.Errorf("list student locations: %w", err)
	}
	defer rows.Close()

	items := make([]StudentLocation, 0)
	for rows.Next() {
		var item StudentLocation
		if err := rows.Scan(&item.ID, &item.SchoolID, &item.ParentID, &item.CountryCode,
			&item.Level, &item.Name, &item.LocalityType, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan student location: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student locations: %w", err)
	}
	return items, nil
}

// CreateStudentLocation creates a school-owned addition. Shared canonical
// rows are provisioned only through reviewed reference-data migrations.
func (r *AcademicRepository) CreateStudentLocation(
	ctx context.Context, actorID, schoolID int64, parentID *int64, countryCode, level, name string, localityType *string,
) (*StudentLocation, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	level = strings.TrimSpace(strings.ToLower(level))
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 || (parentID != nil && *parentID <= 0) {
		return nil, ErrInvalidAcademicInput
	}
	if level != "state" && level != "district" && level != "subdistrict" && level != "locality" {
		return nil, ErrInvalidAcademicInput
	}
	if level == "locality" {
		if localityType == nil {
			return nil, ErrInvalidAcademicInput
		}
		value := strings.TrimSpace(strings.ToLower(*localityType))
		if value != "village" && value != "city" && value != "town" && value != "other" {
			return nil, ErrInvalidAcademicInput
		}
		localityType = &value
	} else if localityType != nil && strings.TrimSpace(*localityType) != "" {
		return nil, ErrInvalidAcademicInput
	} else {
		localityType = nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student location creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}

	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	if parentID == nil {
		if level != "state" {
			return nil, ErrInvalidAcademicInput
		}
		if countryCode == "" {
			countryCode = "IN"
		}
		if len(countryCode) != 2 {
			return nil, ErrInvalidAcademicInput
		}
	} else {
		var parentLevel, parentCountryCode string
		if err := tx.QueryRow(ctx, `
			SELECT level, country_code
			  FROM student_locations
			 WHERE id = $1 AND (school_id IS NULL OR school_id = $2) AND is_active = TRUE
		`, *parentID, schoolID).Scan(&parentLevel, &parentCountryCode); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrAcademicNotFound
			}
			return nil, fmt.Errorf("load parent student location: %w", err)
		}
		if nextStudentLocationLevel(parentLevel) != level {
			return nil, ErrInvalidAcademicInput
		}
		if countryCode != "" && countryCode != parentCountryCode {
			return nil, ErrInvalidAcademicInput
		}
		countryCode = parentCountryCode
	}

	var item StudentLocation
	err = tx.QueryRow(ctx, `
		INSERT INTO student_locations
			(school_id,parent_id,country_code,level,name,locality_type)
		VALUES($1,$2,$3,$4,$5,$6)
		RETURNING id,school_id,parent_id,country_code,level,name,locality_type,is_active,created_at,updated_at
	`, schoolID, parentID, countryCode, level, name, localityType).Scan(
		&item.ID, &item.SchoolID, &item.ParentID, &item.CountryCode, &item.Level,
		&item.Name, &item.LocalityType, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student_location.created','student_location',$3,'success',
		       jsonb_build_object('level',$4::text,'parent_id',$5::bigint))
	`, actorID, schoolID, fmt.Sprint(item.ID), level, parentID); err != nil {
		return nil, fmt.Errorf("audit student location creation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student location creation: %w", err)
	}
	return &item, nil
}

// DeactivateStudentLocation is the only supported location lifecycle change;
// referenced locations remain available to existing address records.
func (r *AcademicRepository) DeactivateStudentLocation(
	ctx context.Context, actorID, schoolID, locationID int64,
) (*StudentLocation, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if locationID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student location deactivation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	var item StudentLocation
	err = tx.QueryRow(ctx, `
		UPDATE student_locations SET is_active=FALSE,updated_at=NOW()
		 WHERE id=$1 AND school_id=$2 AND is_active=TRUE
		RETURNING id,school_id,parent_id,country_code,level,name,locality_type,is_active,created_at,updated_at
	`, locationID, schoolID).Scan(
		&item.ID, &item.SchoolID, &item.ParentID, &item.CountryCode, &item.Level,
		&item.Name, &item.LocalityType, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAcademicNotFound
	}
	if err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		VALUES($1,$2,'school.student_location.deactivated','student_location',$3,'success','{}'::jsonb)
	`, actorID, schoolID, fmt.Sprint(item.ID)); err != nil {
		return nil, fmt.Errorf("audit student location deactivation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student location deactivation: %w", err)
	}
	return &item, nil
}

const studentAddressSelect = `
	SELECT sa.id,sa.school_id,sa.student_id,sa.address_type,
	       sa.address_line_1,sa.address_line_2,sa.area_mohalla,sa.landmark,
	       sa.postal_code,sa.location_id,l.name,l.level,l.locality_type,
	       state_location.id,district_location.id,subdistrict_location.id,
	       sa.same_as_current,sa.created_at,sa.updated_at
	  FROM student_addresses sa
	  LEFT JOIN student_locations l ON l.id=sa.location_id
	  LEFT JOIN student_locations subdistrict_location ON subdistrict_location.id=l.parent_id
	  LEFT JOIN student_locations district_location ON district_location.id=subdistrict_location.parent_id
	  LEFT JOIN student_locations state_location ON state_location.id=district_location.parent_id
	 WHERE sa.school_id=$1 AND sa.student_id=$2
`

func scanStudentAddress(row interface{ Scan(...any) error }) (*StudentAddress, error) {
	var item StudentAddress
	if err := row.Scan(&item.ID,&item.SchoolID,&item.StudentID,&item.AddressType,
		&item.AddressLine1,&item.AddressLine2,&item.AreaMohalla,&item.Landmark,
		&item.PostalCode,&item.LocationID,&item.LocationName,&item.LocationLevel,
		&item.LocalityType,&item.StateLocationID,&item.DistrictLocationID,&item.SubdistrictLocationID,
		&item.SameAsCurrent,&item.CreatedAt,&item.UpdatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *AcademicRepository) ListStudentAddresses(
	ctx context.Context, actorID, schoolID, studentID int64,
) ([]StudentAddress, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	if studentID <= 0 {
		return nil, ErrInvalidAcademicInput
	}
	if err := authorizeAcademic(ctx, r.db, actorID, schoolID, "students.read_sensitive"); err != nil {
		return nil, err
	}
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM students WHERE id=$1 AND school_id=$2)`,
		studentID, schoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check student for address list: %w", err)
	}
	if !exists {
		return nil, ErrAcademicNotFound
	}
	rows, err := r.db.Query(ctx, studentAddressSelect+` ORDER BY CASE address_type WHEN 'current' THEN 0 ELSE 1 END`, schoolID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student addresses: %w", err)
	}
	defer rows.Close()
	items := make([]StudentAddress, 0, 2)
	for rows.Next() {
		item, scanErr := scanStudentAddress(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan student address: %w", scanErr)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate student addresses: %w", err)
	}
	return items, nil
}

func (r *AcademicRepository) SaveStudentAddress(
	ctx context.Context, actorID, schoolID, studentID int64, addressType string,
	line1, line2, areaMohalla, landmark, postalCode *string,
	locationID *int64, sameAsCurrent bool,
) (*StudentAddress, error) {
	if r == nil || r.db == nil {
		return nil, ErrAcademicAccessDenied
	}
	addressType = strings.TrimSpace(strings.ToLower(addressType))
	if studentID <= 0 || (addressType != "current" && addressType != "permanent") ||
		(locationID != nil && *locationID <= 0) || (sameAsCurrent && addressType != "permanent") {
		return nil, ErrInvalidAcademicInput
	}
	var err error
	if line1, err = normalizeAddressText(line1, 200); err != nil { return nil, err }
	if line2, err = normalizeAddressText(line2, 200); err != nil { return nil, err }
	if areaMohalla, err = normalizeAddressText(areaMohalla, 120); err != nil { return nil, err }
	if landmark, err = normalizeAddressText(landmark, 120); err != nil { return nil, err }
	if postalCode, err = normalizeAddressText(postalCode, 20); err != nil { return nil, err }
	if sameAsCurrent && (line1 != nil || line2 != nil || areaMohalla != nil ||
		landmark != nil || postalCode != nil || locationID != nil) {
		return nil, ErrInvalidAcademicInput
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin student address save: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := authorizeAcademic(ctx, tx, actorID, schoolID, "students.update"); err != nil {
		return nil, err
	}
	var studentActive bool
	if err := tx.QueryRow(ctx, `SELECT status='active' FROM students WHERE id=$1 AND school_id=$2 FOR UPDATE`,
		studentID, schoolID).Scan(&studentActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { return nil, ErrAcademicNotFound }
		return nil, fmt.Errorf("load student for address save: %w", err)
	}
	if !studentActive {
		return nil, ErrInvalidAcademicInput
	}
	if sameAsCurrent {
		var hasCurrent bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM student_addresses WHERE school_id=$1 AND student_id=$2 AND address_type='current'
		)`, schoolID, studentID).Scan(&hasCurrent); err != nil {
			return nil, fmt.Errorf("check current address: %w", err)
		}
		if !hasCurrent {
			return nil, ErrInvalidAcademicInput
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO student_addresses
			(school_id,student_id,address_type,address_line_1,address_line_2,area_mohalla,landmark,postal_code,location_id,same_as_current)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT(student_id,address_type) DO UPDATE SET
			address_line_1=EXCLUDED.address_line_1,
			address_line_2=EXCLUDED.address_line_2,
			area_mohalla=EXCLUDED.area_mohalla,
			landmark=EXCLUDED.landmark,
			postal_code=EXCLUDED.postal_code,
			location_id=EXCLUDED.location_id,
			same_as_current=EXCLUDED.same_as_current,
			updated_at=NOW()
	`, schoolID, studentID, addressType, line1, line2, areaMohalla, landmark, postalCode, locationID, sameAsCurrent); err != nil {
		return nil, academicWriteError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events(actor_principal_id,school_id,action,resource_type,resource_id,outcome,metadata)
		SELECT $1,$2,'school.student.address.updated','student_address',sa.id::text,'success',
		       jsonb_build_object('address_type',$3::text,'same_as_current',$4::boolean)
		  FROM student_addresses sa
		 WHERE sa.school_id=$2 AND sa.student_id=$5 AND sa.address_type=$3
	`, actorID, schoolID, addressType, sameAsCurrent, studentID); err != nil {
		return nil, fmt.Errorf("audit student address update: %w", err)
	}
	item, err := scanStudentAddress(tx.QueryRow(ctx, studentAddressSelect+` AND sa.address_type=$3`, schoolID, studentID, addressType))
	if err != nil {
		return nil, fmt.Errorf("read saved student address: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit student address save: %w", err)
	}
	return item, nil
}
