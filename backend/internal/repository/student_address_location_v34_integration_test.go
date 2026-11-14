package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestStudentAddressLocationV34PostgresIntegration verifies hierarchical
// ownership, parent-level rules, duplicate normalization and address tenant
// isolation. All fixtures are rollback-scoped.
func TestStudentAddressLocationV34PostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	var schemaReady bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass('public.student_locations') IS NOT NULL
		   AND to_regclass('public.student_addresses') IS NOT NULL
	`).Scan(&schemaReady); err != nil {
		t.Fatalf("inspect V34 address/location schema: %v", err)
	}
	if !schemaReady {
		t.Fatal("V34 student address/location tables are missing")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped V34 test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, actorPrincipalID int64
	if err := tx.QueryRow(ctx, `SELECT school_id,principal_id FROM users WHERE id=$1`, userID).Scan(&schoolID, &actorPrincipalID); err != nil {
		t.Fatalf("load integration school actor: %v", err)
	}
	platformAdminPrincipalID := ensurePlatformAdminLifecycleActor(t, pool)
	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff'
		  AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles(principal_id,role_id,assigned_by)
		VALUES($1,$2,$3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign School Admin fixture role: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var otherSchoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools(name,code,status)
		VALUES($1,$2,'active') RETURNING id
	`, "V34 Other School "+suffix, "V34-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second school fixture: %v", err)
	}

	var stateID, districtID, subdistrictID, localityID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,level,name)
		VALUES($1,'state',$2) RETURNING id
	`, schoolID, "V34 State "+suffix).Scan(&stateID); err != nil {
		t.Fatalf("create school state: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,parent_id,level,name)
		VALUES($1,$2,'district',$3) RETURNING id
	`, schoolID, stateID, "V34 District "+suffix).Scan(&districtID); err != nil {
		t.Fatalf("create school district: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,parent_id,level,name)
		VALUES($1,$2,'subdistrict',$3) RETURNING id
	`, schoolID, districtID, "V34 Block "+suffix).Scan(&subdistrictID); err != nil {
		t.Fatalf("create school block: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,parent_id,level,name,locality_type)
		VALUES($1,$2,'locality',$3,'village') RETURNING id
	`, schoolID, subdistrictID, "V34 Village "+suffix).Scan(&localityID); err != nil {
		t.Fatalf("create school village: %v", err)
	}

	// A school may add a district under a shared canonical state.
	var canonicalStateID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,level,name)
		VALUES(NULL,'state',$1) RETURNING id
	`, "V34 Canonical State "+suffix).Scan(&canonicalStateID); err != nil {
		t.Fatalf("create canonical state fixture: %v", err)
	}
	var schoolDistrictID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO student_locations(school_id,parent_id,level,name)
		VALUES($1,$2,'district',$3) RETURNING id
	`, schoolID, canonicalStateID, "V34 School District "+suffix).Scan(&schoolDistrictID); err != nil {
		t.Fatalf("create school-owned district under canonical state: %v", err)
	}


	assertRejected := func(label, query string, args ...any) *pgconn.PgError {
		t.Helper()
		if _, err := tx.Exec(ctx, `SAVEPOINT v34_rejected_write`); err != nil {
			t.Fatalf("%s: create savepoint: %v", label, err)
		}
		_, writeErr := tx.Exec(ctx, query, args...)
		if _, err := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT v34_rejected_write`); err != nil {
			t.Fatalf("%s: rollback savepoint: %v", label, err)
		}
		if _, err := tx.Exec(ctx, `RELEASE SAVEPOINT v34_rejected_write`); err != nil {
			t.Fatalf("%s: release savepoint: %v", label, err)
		}
		var pgErr *pgconn.PgError
		if !errors.As(writeErr, &pgErr) {
			t.Fatalf("%s error = %v, want PostgreSQL constraint/trigger error", label, writeErr)
		}
		return pgErr
	}

	assertRejected("wrong hierarchy level",
		`INSERT INTO student_locations(school_id,parent_id,level,name,locality_type)
		 VALUES($1,$2,'locality','V34 Wrong Level','village')`, schoolID, stateID)
	assertRejected("cross-school location parent",
		`INSERT INTO student_locations(school_id,parent_id,level,name)
		 VALUES($1,$2,'district','V34 Foreign Child')`, otherSchoolID, stateID)
	assertRejected("global child under school-owned parent",
		`INSERT INTO student_locations(school_id,parent_id,level,name)
		 VALUES(NULL,$1,'district','V34 Invalid Global Child')`, stateID)
	if pgErr := assertRejected("school-owned location duplicates canonical",
		`INSERT INTO student_locations(school_id,level,name)
		 VALUES($1,'state',$2)`, schoolID, "  v34 canonical STATE "+suffix+"  "); pgErr.Code != "23505" {
		t.Fatalf("school/canonical duplicate SQLSTATE = %s, want 23505", pgErr.Code)
	}
	if pgErr := assertRejected("canonical location duplicates active school location",
		`INSERT INTO student_locations(school_id,level,name)
		 VALUES(NULL,'state',$1)`, "V34 State "+suffix); pgErr.Code != "23505" {
		t.Fatalf("canonical/school duplicate SQLSTATE = %s, want 23505", pgErr.Code)
	}
	assertRejected("canonical location duplicates active school location",
		`INSERT INTO student_locations(school_id,level,name)
		 VALUES(NULL,'state',$1)`, "V34 State "+suffix)
	assertRejected("duplicate normalized location",
		`INSERT INTO student_locations(school_id,parent_id,level,name)
		 VALUES($1,$2,'district',$3)`, schoolID, stateID, "  v34 DISTRICT "+suffix+"  ")

	var studentID, otherStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING id
	`, schoolID, "V34 Address Student "+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create address student: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,NULL,$2) RETURNING id
	`, otherSchoolID, "V34 Foreign Address Student "+suffix).Scan(&otherStudentID); err != nil {
		t.Fatalf("create foreign address student: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO student_addresses
			(school_id,student_id,address_type,address_line_1,area_mohalla,postal_code,location_id)
		VALUES($1,$2,'current','House 12','Main Mohalla','134001',$3)
	`, schoolID, studentID, localityID); err != nil {
		t.Fatalf("create current student address: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_addresses(school_id,student_id,address_type,same_as_current)
		VALUES($1,$2,'permanent',TRUE)
	`, schoolID, studentID); err != nil {
		t.Fatalf("create same-as-current permanent address: %v", err)
	}
	assertRejected("same-as-current without current address",
		`INSERT INTO student_addresses(school_id,student_id,address_type,same_as_current)
		 VALUES($1,$2,'permanent',TRUE)`, otherSchoolID, otherStudentID)
	assertRejected("cross-school address location",
		`INSERT INTO student_addresses(school_id,student_id,address_type,location_id)
		 VALUES($1,$2,'current',$3)`, otherSchoolID, otherStudentID, localityID)

	// Parent locations with active descendants cannot be deactivated; leaf
	// locations can be deactivated, but referenced address history is retained.
	assertRejected("deactivate parent with active children",
		`UPDATE student_locations SET is_active=FALSE WHERE id=$1`, stateID)
	if _, err := tx.Exec(ctx, `UPDATE student_locations SET is_active=FALSE WHERE id=$1`, localityID); err != nil {
		t.Fatalf("deactivate referenced leaf location: %v", err)
	}
	assertRejected("reuse inactive location",
		`INSERT INTO student_addresses(school_id,student_id,address_type,location_id)
		 VALUES($1,$2,'permanent',$3)`, schoolID, studentID, localityID)
	assertRejected("delete address history",
		`DELETE FROM student_addresses WHERE school_id=$1 AND student_id=$2 AND address_type='current'`,
		schoolID, studentID)
	// Referenced locations cannot be hard-deleted; deactivation is the supported lifecycle.
	assertRejected("delete referenced location",
		`DELETE FROM student_locations WHERE id=$1`, localityID)

	var addressCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_addresses
		WHERE school_id=$1 AND student_id=$2
	`, schoolID, studentID).Scan(&addressCount); err != nil {
		t.Fatalf("count V34 student addresses: %v", err)
	}
	repository := NewAcademicRepository(tx)
	addresses, err := repository.ListStudentAddresses(ctx, actorPrincipalID, schoolID, studentID)
	if err != nil || len(addresses) != 2 {
		t.Fatalf("read student addresses through repository: count=%d err=%v", len(addresses), err)
	}
	var foundCurrent bool
	for _, address := range addresses {
		if address.AddressType == "current" {
			foundCurrent = address.AddressLine1 != nil && *address.AddressLine1 == "House 12" &&
				address.AreaMohalla != nil && *address.AreaMohalla == "Main Mohalla" &&
				address.LocationName != nil && *address.LocationName == "V34 Village "+suffix
		}
	}
	if !foundCurrent {
		t.Fatalf("current address did not preserve saved address and inactive locality: %+v", addresses)
	}
	var readAuditCount int
	var readAuditMetadata string
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),COALESCE(string_agg(metadata::text,' '),'')
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.address.read'
		   AND resource_id=$3
	`, schoolID, actorPrincipalID, fmt.Sprint(studentID)).Scan(&readAuditCount, &readAuditMetadata); err != nil {
		t.Fatalf("inspect address read audit: %v", err)
	}
	if readAuditCount != 1 || !strings.Contains(readAuditMetadata, "address") ||
		strings.Contains(readAuditMetadata, "House 12") || strings.Contains(readAuditMetadata, "Main Mohalla") {
		t.Fatalf("address read audit must contain section only: count=%d metadata=%s", readAuditCount, readAuditMetadata)
	}
	if addressCount != 2 {
		t.Fatalf("student address count = %d, want current + permanent", addressCount)
	}
}
