package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStudentProfileRepositoryPostgresIntegration(t *testing.T) {
	_, pool, userID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin student profile integration transaction: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var schoolID, actorID int64
	if err := tx.QueryRow(ctx, `
		SELECT school_id, principal_id FROM users WHERE id = $1
	`, userID).Scan(&schoolID, &actorID); err != nil {
		t.Fatalf("load school admin fixture: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'school_admin' AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load system School Admin role: %v", err)
	}
	var platformAdminPrincipalID int64
	err = tx.QueryRow(ctx, `
		SELECT ap.id
		  FROM platform_admin_accounts pa
		  JOIN auth_principals ap ON ap.id = pa.principal_id
		  JOIN principal_roles pr ON pr.principal_id = ap.id AND pr.revoked_at IS NULL
		  JOIN roles r ON r.id = pr.role_id
		  JOIN role_permissions rp ON rp.role_id = r.id
		  JOIN permissions p ON p.id = rp.permission_id
		 WHERE ap.principal_type = 'platform_admin'
		   AND ap.status = 'active'
		   AND r.role_key = 'platform_admin'
		   AND r.scope_type = 'platform'
		   AND r.is_system = TRUE
		   AND p.permission_key = 'platform.permissions.delegate'
		   AND p.scope_type = 'platform'
		   AND p.is_active = TRUE
		 LIMIT 1
	`).Scan(&platformAdminPrincipalID)
	if err != nil {
		var bootstrapExists bool
		if checkErr := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM platform_admin_accounts)
			    OR EXISTS (
			        SELECT 1 FROM principal_roles pr
			        JOIN roles r ON r.id = pr.role_id
			        WHERE r.principal_type = 'platform_admin'
			          AND r.scope_type = 'platform'
			          AND r.role_key = 'platform_admin'
			    )
		`).Scan(&bootstrapExists); checkErr != nil {
			t.Fatalf("check Platform Admin bootstrap state: %v", checkErr)
		}
		if bootstrapExists { t.Skip("no active Platform Admin has platform.permissions.delegate") }
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx, fmt.Sprintf("student-profile-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
		if err != nil { t.Fatalf("create rollback-scoped Platform Admin fixture: %v", err) }
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("grant School Admin role to fixture actor: %v", err)
	}

	repo := NewAcademicRepository(tx)
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	student, err := repo.CreateStudent(ctx, actorID, schoolID, " ADM-"+suffix+" ", "  Student One  ")
	if err != nil { t.Fatalf("create student admission: %v", err) }
	if student.AdmissionNumber != "ADM-"+suffix || student.FullName != "Student One" || student.Status != "active" {
		t.Fatalf("unexpected created student: %+v", student)
	}

	// PostgreSQL VARCHAR(200) counts characters, not UTF-8 bytes. Names in
	// Indian scripts must therefore be validated by Unicode code point count.
	unicodeName := strings.Repeat("अ", 200)
	unicodeStudent, err := repo.CreateStudent(ctx, actorID, schoolID, "ADM-UNICODE-"+suffix, unicodeName)
	if err != nil || unicodeStudent == nil || unicodeStudent.FullName != unicodeName {
		t.Fatalf("create 200-character Unicode student name: student=%+v err=%v", unicodeStudent, err)
	}
	if _, err := repo.CreateStudent(ctx, actorID, schoolID, "ADM-UNICODE-TOO-LONG-"+suffix, strings.Repeat("अ", 201)); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("create 201-character Unicode student name error = %v, want invalid input", err)
	}
	updatedUnicodeName := strings.Repeat("म", 200)
	updatedUnicode, err := repo.UpdateStudent(ctx, actorID, schoolID, unicodeStudent.ID, updatedUnicodeName)
	if err != nil || updatedUnicode == nil || updatedUnicode.FullName != updatedUnicodeName {
		t.Fatalf("update to 200-character Unicode student name: student=%+v err=%v", updatedUnicode, err)
	}
	if _, err := repo.UpdateStudent(ctx, actorID, schoolID, unicodeStudent.ID, strings.Repeat("म", 201)); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("update to 201-character Unicode student name error = %v, want invalid input", err)
	}

	// The combined admission path must apply the same character-based limits
	// to the student name, year-scoped admission number and optional roll number.
	var admissionYearID, admissionClassID, admissionSectionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active',FALSE) RETURNING id
	`, schoolID, "Unicode Admission Year "+suffix).Scan(&admissionYearID); err != nil {
		t.Fatalf("create Unicode admission year: %v", err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id`,
		schoolID, "Unicode Admission Class "+suffix).Scan(&admissionClassID); err != nil {
		t.Fatalf("create Unicode admission class: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,$4) RETURNING id
	`, schoolID, admissionYearID, admissionClassID, "Unicode "+suffix).Scan(&admissionSectionID); err != nil {
		t.Fatalf("create Unicode admission section: %v", err)
	}
	admissionDate := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	unicodeAdmissionName := strings.Repeat("अ", 200)
	unicodeAdmissionNumber := strings.Repeat("क", 50)
	unicodeRollNumber := strings.Repeat("र", 30)
	admission, err := repo.CreateStudentAdmission(ctx, actorID, schoolID, unicodeAdmissionName,
		unicodeAdmissionNumber, admissionYearID, admissionSectionID, &unicodeRollNumber, admissionDate)
	if err != nil || admission == nil || admission.Student.FullName != unicodeAdmissionName ||
		admission.Enrollment.AdmissionNumber == nil || *admission.Enrollment.AdmissionNumber != unicodeAdmissionNumber ||
		admission.Enrollment.RollNumber == nil || *admission.Enrollment.RollNumber != unicodeRollNumber {
		t.Fatalf("create combined admission with maximum Unicode field lengths: result=%+v err=%v", admission, err)
	}

	// Promotion must not shorten an enrollment past attendance already recorded
	// for it. The destination year deliberately overlaps the source year so
	// the date is otherwise valid and this specifically exercises the history
	// boundary rather than ordinary academic-year date validation.
	attendanceDate := time.Now().UTC().Truncate(24 * time.Hour)
	if _, err := repo.MarkAttendance(ctx, actorID, schoolID, admission.Enrollment.ID, attendanceDate, "present", nil); err != nil {
		t.Fatalf("mark attendance before promotion boundary test: %v", err)
	}
	destinationStartsOn := attendanceDate.AddDate(0, 0, -1)
	destinationEndsOn := destinationStartsOn.AddDate(1, 0, -1)
	var destinationYearID, destinationSectionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
		VALUES($1,$2,$3,$4,'active',FALSE) RETURNING id
	`, schoolID, "Promotion Attendance Boundary "+suffix, destinationStartsOn, destinationEndsOn).Scan(&destinationYearID); err != nil {
		t.Fatalf("create overlapping destination academic year: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,$4) RETURNING id
	`, schoolID, destinationYearID, admissionClassID, "Promotion "+suffix).Scan(&destinationSectionID); err != nil {
		t.Fatalf("create overlapping destination section: %v", err)
	}
	if _, err := repo.PromoteStudentEnrollment(ctx, actorID, schoolID, admission.Enrollment.ID,
		destinationYearID, destinationSectionID, nil, "PROMOTED-"+suffix,
		destinationStartsOn, destinationStartsOn); !errors.Is(err, ErrPromotionBeforeAttendance) {
		t.Fatalf("promote before recorded attendance error = %v, want promotion-before-attendance", err)
	}
	var sourceEnrollmentStatus string
	var destinationEnrollmentCount int
	if err := tx.QueryRow(ctx, `
		SELECT status FROM student_enrollments WHERE id=$1
	`, admission.Enrollment.ID).Scan(&sourceEnrollmentStatus); err != nil {
		t.Fatalf("read source enrollment after rejected promotion: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_enrollments WHERE student_id=$1 AND academic_year_id=$2
	`, admission.Student.ID, destinationYearID).Scan(&destinationEnrollmentCount); err != nil {
		t.Fatalf("count destination enrollments after rejected promotion: %v", err)
	}
	if sourceEnrollmentStatus != "active" || destinationEnrollmentCount != 0 {
		t.Fatalf("rejected promotion changed history: source status=%q destination enrollments=%d",
			sourceEnrollmentStatus, destinationEnrollmentCount)
	}

	// A school may admit a student into a future active academic year before
	// that year becomes current. The directory and profile must still resolve
	// the year-scoped admission identity instead of showing an empty number.
	futureYearItems, futureYearTotal, err := repo.ListStudents(ctx, actorID, schoolID, unicodeAdmissionNumber, "active", 10, 0)
	if err != nil || futureYearTotal != 1 || len(futureYearItems) != 1 ||
		futureYearItems[0].ID != admission.Student.ID || futureYearItems[0].AdmissionNumber != unicodeAdmissionNumber {
		t.Fatalf("directory lookup for non-current-year admission: items=%+v total=%d err=%v",
			futureYearItems, futureYearTotal, err)
	}
	futureYearProfile, err := repo.GetStudent(ctx, actorID, schoolID, admission.Student.ID)
	if err != nil || futureYearProfile.AdmissionNumber != unicodeAdmissionNumber {
		t.Fatalf("profile lookup for non-current-year admission: profile=%+v err=%v", futureYearProfile, err)
	}
	if _, _, err := repo.ListStudents(ctx, actorID, schoolID, strings.Repeat("अ", 101), "active", 10, 0); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("directory search over 100 Unicode characters error = %v, want invalid input", err)
	}
	if _, err := repo.CreateStudentAdmission(ctx, actorID, schoolID, strings.Repeat("अ", 201),
		"ADM-UNICODE-LONG-"+suffix, admissionYearID, admissionSectionID, nil, admissionDate); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("combined admission with 201-character student name error = %v, want invalid input", err)
	}
	if _, err := repo.CreateStudentAdmission(ctx, actorID, schoolID, "Unicode Admission",
		strings.Repeat("क", 51), admissionYearID, admissionSectionID, nil, admissionDate); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("combined admission with 51-character admission number error = %v, want invalid input", err)
	}
	tooLongRollNumber := strings.Repeat("र", 31)
	if _, err := repo.CreateStudentAdmission(ctx, actorID, schoolID, "Unicode Admission",
		"ADM-UNICODE-ROLL-"+suffix, admissionYearID, admissionSectionID, &tooLongRollNumber, admissionDate); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("combined admission with 31-character roll number error = %v, want invalid input", err)
	}
	var createdAuditCount int
	var createdAuditHasAdmissionFlag bool
	var createdAuditLeaksStudentData bool
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*),
		       COALESCE(BOOL_OR(metadata->>'has_admission_number'='true'),FALSE),
		       COALESCE(BOOL_OR(metadata::text ILIKE '%'||$3||'%' OR metadata::text ILIKE '%'||$4||'%'),FALSE)
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.created' AND resource_id=$5
	`, schoolID, actorID, student.FullName, student.AdmissionNumber, fmt.Sprint(student.ID)).Scan(
		&createdAuditCount, &createdAuditHasAdmissionFlag, &createdAuditLeaksStudentData,
	); err != nil {
		t.Fatalf("inspect student-created audit metadata: %v", err)
	}
	if createdAuditCount != 1 || !createdAuditHasAdmissionFlag || createdAuditLeaksStudentData {
		t.Fatalf("student-created audit count=%d has_admission_number=%v leaks_student_data=%v",
			createdAuditCount, createdAuditHasAdmissionFlag, createdAuditLeaksStudentData)
	}
	duplicateLegacy, err := repo.CreateStudent(ctx, actorID, schoolID, "adm-"+suffix, "Duplicate Legacy Admission")
	if err != nil || duplicateLegacy == nil || duplicateLegacy.AdmissionNumber != "adm-"+suffix {
		t.Fatalf("duplicate legacy student-level admission value should be retained without a conflict: student=%+v err=%v", duplicateLegacy, err)
	}
	legacyUnicodeAdmission := strings.Repeat("न", 50)
	legacyUnicodeStudent, err := repo.CreateStudent(ctx, actorID, schoolID, legacyUnicodeAdmission, "Legacy Unicode Admission")
	if err != nil || legacyUnicodeStudent == nil || legacyUnicodeStudent.AdmissionNumber != legacyUnicodeAdmission {
		t.Fatalf("create legacy student with 50-character Unicode admission number: student=%+v err=%v", legacyUnicodeStudent, err)
	}
	if _, err := repo.CreateStudent(ctx, actorID, schoolID, strings.Repeat("न", 51), "Too Long Unicode Admission"); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("create legacy student with 51-character Unicode admission number error = %v, want invalid input", err)
	}
	got, err := repo.GetStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || got.ID != student.ID { t.Fatalf("get student: student=%+v err=%v", got, err) }

	// Address reads must return every saved hierarchy level, including names
	// needed to display a previously selected location after deactivation.
	stateName := strings.Repeat("ह", 120)
	if _, err := repo.CreateStudentLocation(ctx, actorID, schoolID, nil, "é", "state", "Invalid Unicode Country Code", nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("create state with non-ASCII two-byte country code error = %v, want invalid input", err)
	}
	if _, err := repo.CreateStudentLocation(ctx, actorID, schoolID, nil, "1!", "state", "Invalid Country Code", nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("create state with non-letter country code error = %v, want invalid input", err)
	}
	state, err := repo.CreateStudentLocation(ctx, actorID, schoolID, nil, "IN", "state", stateName, nil)
	if err != nil { t.Fatalf("create address state: %v", err) }
	if _, err := repo.CreateStudentLocation(ctx, actorID, schoolID, nil, "IN", "state", strings.Repeat("ह", 121), nil); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("create 121-character Unicode location name error = %v, want invalid input", err)
	}
	district, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &state.ID, "", "district", "District "+suffix, nil)
	if err != nil { t.Fatalf("create address district: %v", err) }
	unicodeLocationName := strings.Repeat("अ", 50)
	if _, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &state.ID, "", "district", unicodeLocationName, nil); err != nil {
		t.Fatalf("create Unicode location for search: %v", err)
	}
	unicodeLocations, err := repo.ListStudentLocations(ctx, actorID, schoolID, &state.ID, "district", unicodeLocationName)
	if err != nil || len(unicodeLocations) != 1 || unicodeLocations[0].Name != unicodeLocationName {
		t.Fatalf("search locations by 50 Unicode characters: locations=%+v err=%v", unicodeLocations, err)
	}
	if _, err := repo.ListStudentLocations(ctx, actorID, schoolID, &state.ID, "district", strings.Repeat("अ", 101)); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("location search over 100 Unicode characters error = %v, want invalid input", err)
	}
	subdistrict, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &district.ID, "", "subdistrict", "Block "+suffix, nil)
	if err != nil { t.Fatalf("create address block: %v", err) }
	localityType := "village"
	locality, err := repo.CreateStudentLocation(ctx, actorID, schoolID, &subdistrict.ID, "", "locality", "Village "+suffix, &localityType)
	if err != nil { t.Fatalf("create address locality: %v", err) }
	line1 := strings.Repeat("ग", 200)
	if _, err := repo.SaveStudentAddress(ctx, actorID, schoolID, student.ID, "current",
		&line1, nil, nil, nil, nil, &locality.ID, false); err != nil {
		t.Fatalf("save student address: %v", err)
	}
	tooLongAddressLine := strings.Repeat("ग", 201)
	if _, err := repo.SaveStudentAddress(ctx, actorID, schoolID, student.ID, "current",
		&tooLongAddressLine, nil, nil, nil, nil, &locality.ID, false); !errors.Is(err, ErrInvalidAcademicInput) {
		t.Fatalf("save 201-character Unicode address line error = %v, want invalid input", err)
	}
	if _, err := repo.DeactivateStudentLocation(ctx, actorID, schoolID, locality.ID); err != nil {
		t.Fatalf("deactivate saved locality: %v", err)
	}
	addresses, err := repo.ListStudentAddresses(ctx, actorID, schoolID, student.ID)
	if err != nil || len(addresses) != 1 {
		t.Fatalf("list saved address: addresses=%+v err=%v", addresses, err)
	}
	address := addresses[0]
	if address.StateLocationID == nil || *address.StateLocationID != state.ID ||
		address.DistrictLocationID == nil || *address.DistrictLocationID != district.ID ||
		address.SubdistrictLocationID == nil || *address.SubdistrictLocationID != subdistrict.ID ||
		address.StateLocationName == nil || *address.StateLocationName != state.Name ||
		address.DistrictLocationName == nil || *address.DistrictLocationName != district.Name ||
		address.SubdistrictLocationName == nil || *address.SubdistrictLocationName != subdistrict.Name ||
		address.LocationName == nil || *address.LocationName != locality.Name {
		t.Fatalf("saved address hierarchy was not fully returned: %+v", address)
	}

	updated, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Student One Updated")
	if err != nil || updated.FullName != "Student One Updated" {
		t.Fatalf("update student: student=%+v err=%v", updated, err)
	}

	var studentUpdatedAt time.Time
	var studentUpdateAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM students WHERE id=$1 AND school_id=$2`, student.ID, schoolID).Scan(&studentUpdatedAt); err != nil {
		t.Fatalf("read student updated_at after actual update: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student.updated' AND resource_id=$3`,
		schoolID, actorID, fmt.Sprint(student.ID)).Scan(&studentUpdateAuditCount); err != nil {
		t.Fatalf("count student update audit events: %v", err)
	}
	unchangedRetry, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Student One Updated")
	if err != nil || unchangedRetry == nil || unchangedRetry.ID != student.ID || unchangedRetry.FullName != "Student One Updated" {
		t.Fatalf("retry identical student basic-details update: student=%+v err=%v", unchangedRetry, err)
	}
	var retryUpdatedAt time.Time
	var retryAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM students WHERE id=$1 AND school_id=$2`, student.ID, schoolID).Scan(&retryUpdatedAt); err != nil {
		t.Fatalf("read student updated_at after identical retry: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student.updated' AND resource_id=$3`,
		schoolID, actorID, fmt.Sprint(student.ID)).Scan(&retryAuditCount); err != nil {
		t.Fatalf("count student update audit events after identical retry: %v", err)
	}
	if !retryUpdatedAt.Equal(studentUpdatedAt) || retryAuditCount != studentUpdateAuditCount || studentUpdateAuditCount != 1 {
		t.Fatalf("identical student update retry changed timestamp/audit count: timestamp before=%s after=%s audits before=%d after=%d",
			studentUpdatedAt, retryUpdatedAt, studentUpdateAuditCount, retryAuditCount)
	}
	items, total, err := repo.ListStudents(ctx, actorID, schoolID, "Updated", "active", 10, 0)
	if err != nil || total < 1 { t.Fatalf("search students: total=%d err=%v", total, err) }
	found := false
	for _, item := range items { if item.ID == student.ID { found = true } }
	if !found { t.Fatalf("student %d missing from school-scoped search: %+v", student.ID, items) }

	var profileReadAuditCount, directoryReadAuditCount int
	var directoryResultCount int
	var readAuditContainsStudentData bool
	if err := tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE action='school.student.profile.read'),
			COUNT(*) FILTER (WHERE action='school.student.directory.read'),
			COALESCE(MAX((metadata->>'result_count')::integer) FILTER (WHERE action='school.student.directory.read'),0),
			COALESCE(BOOL_OR(metadata::text ILIKE '%'||$3||'%' OR metadata::text ILIKE '%'||$4||'%')
				FILTER (WHERE action IN ('school.student.profile.read','school.student.directory.read')),FALSE)
		FROM audit_events
		WHERE school_id=$1 AND actor_principal_id=$2
	`, schoolID, actorID, student.FullName, student.AdmissionNumber).Scan(
		&profileReadAuditCount, &directoryReadAuditCount, &directoryResultCount, &readAuditContainsStudentData,
	); err != nil {
		t.Fatalf("inspect student profile/directory read audits: %v", err)
	}
	if profileReadAuditCount != 2 || directoryReadAuditCount != 2 || directoryResultCount != 1 {
		t.Fatalf("student read audit counts profile=%d directory=%d directory result_count=%d; want 2,2,1",
			profileReadAuditCount, directoryReadAuditCount, directoryResultCount)
	}
	if readAuditContainsStudentData {
		t.Fatal("student profile/directory read audit metadata must not contain student name or admission number")
	}

	// Direct API writes must obey the same active-only edit boundary as the
	// Student Management UI. Inactive records remain readable but immutable.
	if _, err := tx.Exec(ctx, `UPDATE students SET status='inactive' WHERE id=$1 AND school_id=$2`, student.ID, schoolID); err != nil {
		t.Fatalf("mark student inactive for write-boundary test: %v", err)
	}
	if _, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Inactive Name Must Not Change"); !errors.Is(err, ErrAcademicNotFound) {
		t.Fatalf("update inactive student error = %v, want not found", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE students SET status='active' WHERE id=$1 AND school_id=$2`, student.ID, schoolID); err != nil {
		t.Fatalf("restore student active after write-boundary test: %v", err)
	}

	archiveAdmissionNumber := "ARCHIVE-" + suffix
	archiveEnrollment, err := repo.CreateStudentEnrollment(ctx, actorID, schoolID, student.ID,
		admissionYearID, admissionSectionID, nil, archiveAdmissionNumber, admissionDate, admissionDate)
	if err != nil || archiveEnrollment == nil || archiveEnrollment.Status != "active" {
		t.Fatalf("create active enrollment for archive test: enrollment=%+v err=%v", archiveEnrollment, err)
	}
	var archiveAttendanceDay time.Time
	if err := tx.QueryRow(ctx, "SELECT CURRENT_DATE").Scan(&archiveAttendanceDay); err != nil {
		t.Fatalf("read database date for archive attendance history: %v", err)
	}
	if _, err := repo.MarkAttendance(ctx, actorID, schoolID, archiveEnrollment.ID, archiveAttendanceDay, "present", nil); err != nil {
		t.Fatalf("mark attendance before archive: %v", err)
	}
	archived, err := repo.ArchiveStudent(ctx, actorID, schoolID, student.ID)
	if err != nil || archived.Status != "archived" { t.Fatalf("archive student: student=%+v err=%v", archived, err) }
	var archivedEnrollmentStatus string
	var archivedEnrollmentEnd time.Time
	if err := tx.QueryRow(ctx, `
		SELECT status, ended_at FROM student_enrollments WHERE id=$1 AND school_id=$2
	`, archiveEnrollment.ID, schoolID).Scan(&archivedEnrollmentStatus, &archivedEnrollmentEnd); err != nil {
		t.Fatalf("read enrollment after student archive: %v", err)
	}
	if archivedEnrollmentStatus != "withdrawn" || archivedEnrollmentEnd.Before(admissionDate) {
		t.Fatalf("archived student's enrollment status/end date = %q/%s; want withdrawn and end date >= %s",
			archivedEnrollmentStatus, archivedEnrollmentEnd.Format("2006-01-02"), admissionDate.Format("2006-01-02"))
	}
	// A retry after a successful archive must return the archived record and
	// must not duplicate the archive audit event or alter enrollment history.
	retriedArchive, retryErr := repo.ArchiveStudent(ctx, actorID, schoolID, student.ID)
	if retryErr != nil || retriedArchive == nil || retriedArchive.ID != student.ID || retriedArchive.Status != "archived" {
		t.Fatalf("retry archived student operation: student=%+v err=%v", retriedArchive, retryErr)
	}
	var archiveAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student.archived' AND resource_id=$3
	`, schoolID, actorID, fmt.Sprint(student.ID)).Scan(&archiveAuditCount); err != nil {
		t.Fatalf("inspect student archive audit after retry: %v", err)
	}
	if archiveAuditCount != 1 {
		t.Fatalf("student archive audit count after retry = %d, want exactly one", archiveAuditCount)
	}
	var preservedEnrollmentCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM student_enrollments WHERE id=$1 AND student_id=$2
	`, archiveEnrollment.ID, student.ID).Scan(&preservedEnrollmentCount); err != nil || preservedEnrollmentCount != 1 {
		t.Fatalf("archived enrollment history count=%d err=%v; want preserved row", preservedEnrollmentCount, err)
	}
	var preservedAttendanceCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM attendance_records WHERE school_id=$1 AND enrollment_id=$2
	`, schoolID, archiveEnrollment.ID).Scan(&preservedAttendanceCount); err != nil || preservedAttendanceCount != 1 {
		t.Fatalf("archived attendance history count=%d err=%v; want exactly one preserved row", preservedAttendanceCount, err)
	}
	if _, err := repo.UpdateStudent(ctx, actorID, schoolID, student.ID, "Should Not Update"); !errors.Is(err, ErrAcademicNotFound) {
		t.Fatalf("update archived student error = %v, want not found", err)
	}

	// Attendance marking must reject an archived student's retained enrollment.
	// MarkAttendance also locks the permanent student row before inserting, matching
	// ArchiveStudent's lock order so a concurrent archive cannot race a new mark.
	if _, err := repo.MarkAttendance(ctx, actorID, schoolID, archiveEnrollment.ID, time.Now(), "present", nil); !errors.Is(err, ErrAttendanceNotFound) {
		t.Fatalf("mark attendance for archived student error = %v, want not found", err)
	}

	var otherSchoolID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status) VALUES ($1, $2, 'active') RETURNING id
	`, "Student Profile Isolation "+suffix, "SPI-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create isolation school: %v", err)
	}
	// Seed a complete foreign-school enrollment so the history API is tested
	// against real data, not only an empty foreign-school scope.
	var foreignStudentID, foreignYearID, foreignClassID, foreignSectionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students(school_id,admission_number,full_name)
		VALUES($1,$2,'Foreign Enrollment Student') RETURNING id
	`, otherSchoolID, "FOREIGN-ENROLL-"+suffix).Scan(&foreignStudentID); err != nil {
		t.Fatalf("create foreign enrollment student: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO academic_years(school_id,name,starts_on,ends_on,status,is_current)
		VALUES($1,$2,DATE '2026-04-01',DATE '2027-03-31','active',FALSE) RETURNING id
	`, otherSchoolID, "Foreign Enrollment Year "+suffix).Scan(&foreignYearID); err != nil {
		t.Fatalf("create foreign enrollment year: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO classes(school_id,name) VALUES($1,$2) RETURNING id
	`, otherSchoolID, "Foreign Enrollment Class "+suffix).Scan(&foreignClassID); err != nil {
		t.Fatalf("create foreign enrollment class: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO sections(school_id,academic_year_id,class_id,name)
		VALUES($1,$2,$3,'A') RETURNING id
	`, otherSchoolID, foreignYearID, foreignClassID).Scan(&foreignSectionID); err != nil {
		t.Fatalf("create foreign enrollment section: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO student_enrollments(school_id,student_id,academic_year_id,class_id,section_id,admission_number,admission_date,enrolled_at)
		VALUES($1,$2,$3,$4,$5,$6,DATE '2026-04-01',DATE '2026-04-01')
	`, otherSchoolID, foreignStudentID, foreignYearID, foreignClassID, foreignSectionID, "FOREIGN-ENROLL-"+suffix); err != nil {
		t.Fatalf("create foreign enrollment record: %v", err)
	}
	if items, err := repo.ListStudentEnrollments(ctx, actorID, schoolID, 0, foreignStudentID); !errors.Is(err, ErrAcademicNotFound) || len(items) != 0 {
		t.Fatalf("foreign student enrollment history error = %v items=%d; want not found and no data", err, len(items))
	}
	if items, err := repo.ListStudentEnrollments(ctx, actorID, otherSchoolID, 0, 0); !errors.Is(err, ErrAcademicAccessDenied) || len(items) != 0 {
		t.Fatalf("cross-school enrollment directory error = %v items=%d; want access denied and no data", err, len(items))
	}
	if _, err := repo.GetStudent(ctx, actorID, otherSchoolID, student.ID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school student lookup error = %v, want access denied", err)
	}

	if items, total, err := repo.ListStudents(ctx, actorID, otherSchoolID, "", "", 20, 0); !errors.Is(err, ErrAcademicAccessDenied) || len(items) != 0 || total != 0 {
		t.Fatalf("cross-school student directory error = %v items=%d total=%d; want access denied and no data", err, len(items), total)
	}

	if _, err := repo.UpdateStudent(ctx, actorID, otherSchoolID, student.ID, "Cross School Update Must Fail"); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school student update error = %v, want access denied", err)
	}
	if _, err := repo.ArchiveStudent(ctx, actorID, otherSchoolID, student.ID); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("cross-school student archive error = %v, want access denied", err)
	}
}
