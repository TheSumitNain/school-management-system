package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
)

// TestGuardianRepositoryPostgresIntegration exercises guardian contact CRUD,
// student relationship history, live delegated RBAC, audit events, and tenant
// isolation. All application writes are contained by an outer transaction.
func TestGuardianRepositoryPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback-scoped guardian repository test: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var actorPrincipalID, schoolID int64
	if err := tx.QueryRow(ctx, `
		SELECT principal_id, school_id FROM users WHERE id = $1
	`, actorUserID).Scan(&actorPrincipalID, &schoolID); err != nil {
		t.Fatalf("load School Admin fixture identity: %v", err)
	}

	// Resolve a Platform Admin capable of granting school permission
	// delegations. A bootstrap created here exists only inside this transaction.
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
	if errors.Is(err, pgx.ErrNoRows) {
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
		if bootstrapExists {
			t.Skip("Platform Admin bootstrap exists but no active administrator has platform.permissions.delegate")
		}
		platformAdminPrincipalID, err = NewPlatformAdminRepository(pool).bootstrapFirstInTx(
			ctx, tx,
			fmt.Sprintf("guardian-repository-%d@example.invalid", time.Now().UnixNano()),
			"integration-test-password-hash",
		)
	}
	if err != nil {
		t.Fatalf("resolve rollback-scoped Platform Admin actor: %v", err)
	}

	var schoolAdminRoleID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id = $1 AND principal_type = 'staff'
		   AND scope_type = 'school' AND role_key = 'school_admin'
		   AND is_system = TRUE
	`, schoolID).Scan(&schoolAdminRoleID); err != nil {
		t.Fatalf("load School Admin system role: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_roles (principal_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
	`, actorPrincipalID, schoolAdminRoleID, platformAdminPrincipalID); err != nil {
		t.Fatalf("assign School Admin fixture role: %v", err)
	}

	roleRepo := NewRoleRepository(tx)
	readDelegation, err := roleRepo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "guardians.read", true, false,
	)
	if err != nil {
		t.Fatalf("delegate guardians.read to school roles: %v", err)
	}
	studentReadDelegation, err := roleRepo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "students.read_basic", true, false,
	)
	if err != nil {
		t.Fatalf("delegate students.read_basic to school roles: %v", err)
	}
	manageDelegation, err := roleRepo.GrantSchoolPermissionDelegation(
		ctx, platformAdminPrincipalID, schoolID, "guardians.manage", true, false,
	)
	if err != nil {
		t.Fatalf("delegate guardians.manage to school roles: %v", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	customRole, err := roleRepo.CreateCustomRole(
		ctx, actorPrincipalID, schoolID, "guardian_test_"+suffix, "Guardian Integration Role",
	)
	if err != nil {
		t.Fatalf("create guardian integration role: %v", err)
	}
	if err := roleRepo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, readDelegation.ID, "guardians.read",
	); err != nil {
		t.Fatalf("grant guardians.read to custom role: %v", err)
	}
	if err := roleRepo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, manageDelegation.ID, "guardians.manage",
	); err != nil {
		t.Fatalf("grant guardians.manage to custom role: %v", err)
	}
	if err := roleRepo.GrantCustomRolePermission(
		ctx, actorPrincipalID, schoolID, customRole.ID, studentReadDelegation.ID, "students.read_basic",
	); err != nil {
		t.Fatalf("grant students.read_basic to custom role: %v", err)
	}
	var targetPrincipalID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, schoolID, "gtrg-"+suffix).Scan(&targetPrincipalID); err != nil {
		t.Fatalf("create guardian permission target staff: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, targetPrincipalID, customRole.ID); err != nil {
		t.Fatalf("assign guardian integration role to target staff: %v", err)
	}

	repo := NewAcademicRepository(tx)

	// Even if a Platform Admin delegates guardian permissions to a custom
	// role, a Teacher-only principal must not gain family/contact access.
	var teacherRoleID, teacherPrincipalID int64
	if err := tx.QueryRow(ctx, `
		SELECT id FROM roles
		 WHERE school_id=$1 AND principal_type='staff' AND scope_type='school'
		   AND role_key='teacher' AND is_system=TRUE
	`, schoolID).Scan(&teacherRoleID); err != nil {
		t.Fatalf("load Teacher system role: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (school_id, username, password_hash, status)
		VALUES ($1, $2, 'integration-password-hash', 'active')
		RETURNING principal_id
	`, schoolID, "tguard-"+suffix).Scan(&teacherPrincipalID); err != nil {
		t.Fatalf("create Teacher boundary fixture: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, teacherPrincipalID, teacherRoleID); err != nil {
		t.Fatalf("assign Teacher system role: %v", err)
	}
	if err := roleRepo.AssignStaffRole(ctx, actorPrincipalID, schoolID, teacherPrincipalID, customRole.ID); err != nil {
		t.Fatalf("assign guardian custom role to Teacher: %v", err)
	}
	if _, _, err := repo.ListGuardians(ctx, teacherPrincipalID, schoolID, "", "active", 10, 0); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("Teacher with delegated guardians.read must be denied; got %v", err)
	}
	if err := authorizeAcademic(ctx, tx, teacherPrincipalID, schoolID, "students.read_sensitive"); !errors.Is(err, ErrAcademicAccessDenied) {
		t.Fatalf("Teacher with a custom role must not read sensitive student profiles; got %v", err)
	}

	mobile := "+911234567890"
	email := "guardian-" + suffix + "@example.invalid"
	occupation := "Teacher"
	education := "Graduate"
	workplace := "Public School"
	alternateMobile := "+919999999999"
	guardian, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, 0,
		"  Guardian One  ", &mobile, &email, &alternateMobile, &occupation, &education, &workplace, "active")
	if err != nil {
		t.Fatalf("create guardian contact: %v", err)
	}
	if guardian.ID <= 0 || guardian.FullName != "Guardian One" ||
		guardian.MobileNumber == nil || *guardian.MobileNumber != mobile ||
		guardian.EmailAddress == nil || *guardian.EmailAddress != email ||
		guardian.Status != "active" || guardian.Occupation == nil || *guardian.Occupation != occupation || guardian.Education == nil || *guardian.Education != education || guardian.WorkplaceName == nil || *guardian.WorkplaceName != workplace || guardian.AlternateMobileNumber == nil || *guardian.AlternateMobileNumber != alternateMobile {
		t.Fatalf("unexpected created guardian: %+v", guardian)
	}

	tooLongUnicodeMobile := strings.Repeat("९", 31)
	if _, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One", &tooLongUnicodeMobile, &email, &alternateMobile,
		&occupation, &education, &workplace, "active"); !errors.Is(err, ErrInvalidGuardianInput) {
		t.Fatalf("save guardian with 31-character Unicode mobile error = %v, want invalid input", err)
	}
	tooLongUnicodeEmail := strings.Repeat("अ", 255) + "@example.invalid"
	if _, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One", &mobile, &tooLongUnicodeEmail, &alternateMobile,
		&occupation, &education, &workplace, "active"); !errors.Is(err, ErrInvalidGuardianEmail) {
		t.Fatalf("save guardian with over-limit Unicode email error = %v, want invalid email", err)
	}

	for _, invalidEmail := range []string{"not-an-email", "Guardian <guardian@example.invalid>", "guardian@example.invalid,other@example.invalid"} {
		for _, existingGuardianID := range []int64{0, guardian.ID} {
			if _, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, existingGuardianID,
				"Guardian One", &mobile, &invalidEmail, &alternateMobile,
				&occupation, &education, &workplace, "active"); !errors.Is(err, ErrInvalidGuardianEmail) {
				t.Errorf("save guardian (id=%d) with invalid email %q error = %v, want invalid email", existingGuardianID, invalidEmail, err)
			}
		}
	}

	updated, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, &alternateMobile, &occupation, &education, &workplace, "inactive")
	if err != nil {
		t.Fatalf("update guardian contact: %v", err)
	}
	if updated.ID != guardian.ID || updated.FullName != "Guardian One Updated" || updated.Status != "inactive" {
		t.Fatalf("unexpected updated guardian: %+v", updated)
	}
	updated, err = repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, &alternateMobile, &occupation, &education, &workplace, "active")
	if err != nil {
		t.Fatalf("reactivate guardian contact: %v", err)
	}

	got, err := repo.GetGuardian(ctx, targetPrincipalID, schoolID, guardian.ID)
	if err != nil || got.ID != guardian.ID || got.FullName != "Guardian One Updated" {
		t.Fatalf("get guardian: guardian=%+v err=%v", got, err)
	}
	guardians, total, err := repo.ListGuardians(ctx, targetPrincipalID, schoolID,
		"Guardian One Updated", "active", 10, 0)
	if err != nil || total != 1 || len(guardians) != 1 || guardians[0].ID != guardian.ID {
		t.Fatalf("search guardians: total=%d items=%+v err=%v", total, guardians, err)
	}

	var studentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, 'Guardian Integration Student') RETURNING id
	`, schoolID, "GUARDIAN-"+suffix).Scan(&studentID); err != nil {
		t.Fatalf("create same-school student fixture: %v", err)
	}
	// An inactive guardian must not be linked, even if the caller had
	// previously loaded the active guardian profile.
	if _, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, &alternateMobile, &occupation, &education, &workplace, "inactive"); err != nil {
		t.Fatalf("deactivate guardian before link rejection check: %v", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, guardian.ID, "father", false, false); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("link inactive guardian error = %v, want ErrGuardianNotFound", err)
	}
	if _, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Guardian One Updated", &mobile, &email, &alternateMobile, &occupation, &education, &workplace, "active"); err != nil {
		t.Fatalf("reactivate guardian after link rejection check: %v", err)
	}

	var otherSchoolID, foreignStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO schools (name, code, status)
		VALUES ($1, $2, 'active') RETURNING id
	`, "Guardian Isolation "+suffix, "GI-"+suffix).Scan(&otherSchoolID); err != nil {
		t.Fatalf("create second-school fixture: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, 'Foreign Guardian Student') RETURNING id
	`, otherSchoolID, "FOREIGN-"+suffix).Scan(&foreignStudentID); err != nil {
		t.Fatalf("create foreign-school student fixture: %v", err)
	}
	if _, err := repo.GetGuardian(ctx, targetPrincipalID, otherSchoolID, guardian.ID); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("cross-school guardian read error = %v, want access denied", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		foreignStudentID, guardian.ID, "father", true, true); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("cross-school student link error = %v, want not found", err)
	}

	if links, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, foreignStudentID, true); !errors.Is(err, ErrGuardianNotFound) || len(links) != 0 {
		t.Fatalf("cross-school student guardian list error = %v links=%d; want not found and no data", err, len(links))
	}
	if related, err := repo.ListStudentRelatedStudents(ctx, targetPrincipalID, schoolID, foreignStudentID); !errors.Is(err, ErrGuardianNotFound) || len(related) != 0 {
		t.Fatalf("cross-school related-student list error = %v items=%d; want not found and no data", err, len(related))
	}
	if guardians, total, err := repo.ListGuardians(ctx, targetPrincipalID, otherSchoolID, "", "", 10, 0); !errors.Is(err, ErrGuardianAccessDenied) || len(guardians) != 0 || total != 0 {
		t.Fatalf("cross-school guardian directory error = %v items=%d total=%d; want access denied and no data", err, len(guardians), total)
	}

	firstLink, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, guardian.ID, "father", true, true)
	if err != nil {
		t.Fatalf("link guardian to student: %v", err)
	}
	if firstLink.ID <= 0 || !firstLink.IsPrimaryContact || !firstLink.IsEmergencyContact ||
		firstLink.RelationshipType != "Father" || firstLink.GuardianName != "Guardian One Updated" ||
		firstLink.GuardianStatus != "active" ||
		firstLink.AlternateMobileNumber == nil || *firstLink.AlternateMobileNumber != alternateMobile ||
		firstLink.Occupation == nil || *firstLink.Occupation != occupation ||
		firstLink.Education == nil || *firstLink.Education != education ||
		firstLink.WorkplaceName == nil || *firstLink.WorkplaceName != workplace {
		t.Fatalf("unexpected first student-guardian link: %+v", firstLink)
	}

	var linkUpdatedAt time.Time
	var linkCreatedAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM student_guardians WHERE id=$1 AND school_id=$2`, firstLink.ID, schoolID).Scan(&linkUpdatedAt); err != nil {
		t.Fatalf("read newly linked guardian timestamp: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student_guardian_link_created' AND resource_id=$3`,
		schoolID,targetPrincipalID,fmt.Sprint(firstLink.ID)).Scan(&linkCreatedAuditCount); err != nil {
		t.Fatalf("count guardian link creation audit events: %v", err)
	}
	retriedLink, err := repo.LinkStudentGuardian(ctx,targetPrincipalID,schoolID,studentID,guardian.ID,"father",true,true)
	if err != nil || retriedLink == nil || retriedLink.ID != firstLink.ID {
		t.Fatalf("retry identical student guardian link: link=%+v err=%v",retriedLink,err)
	}
	var retryLinkUpdatedAt time.Time
	var retryLinkAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM student_guardians WHERE id=$1 AND school_id=$2`, firstLink.ID, schoolID).Scan(&retryLinkUpdatedAt); err != nil {
		t.Fatalf("read guardian timestamp after identical link retry: %v",err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student_guardian_link_created' AND resource_id=$3`,
		schoolID,targetPrincipalID,fmt.Sprint(firstLink.ID)).Scan(&retryLinkAuditCount); err != nil {
		t.Fatalf("count guardian link creation audit events after retry: %v",err)
	}
	if !retryLinkUpdatedAt.Equal(linkUpdatedAt) || retryLinkAuditCount != linkCreatedAuditCount || linkCreatedAuditCount != 1 {
		t.Fatalf("identical guardian link retry changed timestamp/audit count: before=%s after=%s audits before=%d after=%d",
			linkUpdatedAt,retryLinkUpdatedAt,linkCreatedAuditCount,retryLinkAuditCount)
	}

	if _, err := repo.UpdateStudentGuardianLink(ctx, teacherPrincipalID, schoolID, firstLink.ID, "father", false, false); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("Teacher must not update guardian relationship: %v", err)
	}
	if _, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, otherSchoolID, firstLink.ID, "father", false, false); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("cross-school guardian relationship update error = %v, want access denied", err)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, otherSchoolID, firstLink.ID, "cross-school attempt"); !errors.Is(err, ErrGuardianAccessDenied) {
		t.Fatalf("cross-school guardian relationship revoke error = %v, want access denied", err)
	}

	var relatedStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, NULL, 'Student Sharing Guardian') RETURNING id
	`, schoolID).Scan(&relatedStudentID); err != nil {
		t.Fatalf("create related-student fixture: %v", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		relatedStudentID, guardian.ID, "father", true, false); err != nil {
		t.Fatalf("link shared guardian to related student: %v", err)
	}

	// Shared guardian profile edits must flow to every linked student, while
	// relationship type and per-student contact flags remain independent.
	sharedMobile := "+919812345678"
	sharedOccupation := "Updated shared occupation"
	sharedUpdate, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Shared Guardian Profile Updated", &sharedMobile, &email, &alternateMobile,
		&sharedOccupation, &education, &workplace, "active")
	if err != nil {
		t.Fatalf("update shared guardian profile after linking multiple students: %v", err)
	}
	if sharedUpdate.ID != guardian.ID {
		t.Fatalf("shared guardian update changed identity: %+v", sharedUpdate)
	}

	var sharedGuardianUpdatedAt time.Time
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM guardians WHERE id=$1`, guardian.ID).Scan(&sharedGuardianUpdatedAt); err != nil {
		t.Fatalf("read shared guardian timestamp: %v", err)
	}
	// The same shared-profile payload may be submitted from either linked
	// student's edit flow; it must not create a second update/audit event.
	sharedRetry, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, guardian.ID,
		"Shared Guardian Profile Updated", &sharedMobile, &email, &alternateMobile,
		&sharedOccupation, &education, &workplace, "active")
	if err != nil || sharedRetry.ID != guardian.ID {
		t.Fatalf("retry identical shared guardian profile save: guardian=%+v err=%v", sharedRetry, err)
	}
	var sharedRetryUpdatedAt time.Time
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM guardians WHERE id=$1`, guardian.ID).Scan(&sharedRetryUpdatedAt); err != nil {
		t.Fatalf("read retried shared guardian timestamp: %v", err)
	}
	if !sharedRetryUpdatedAt.Equal(sharedGuardianUpdatedAt) {
		t.Fatalf("identical shared guardian save changed updated_at: before=%s after=%s", sharedGuardianUpdatedAt, sharedRetryUpdatedAt)
	}
	studentLinksAfterSharedEdit, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, false)
	if err != nil || len(studentLinksAfterSharedEdit) != 1 ||
		studentLinksAfterSharedEdit[0].GuardianName != "Shared Guardian Profile Updated" ||
		studentLinksAfterSharedEdit[0].MobileNumber == nil || *studentLinksAfterSharedEdit[0].MobileNumber != sharedMobile ||
		studentLinksAfterSharedEdit[0].RelationshipType != "Father" ||
		!studentLinksAfterSharedEdit[0].IsPrimaryContact || !studentLinksAfterSharedEdit[0].IsEmergencyContact {
		t.Fatalf("shared profile update was not reflected without changing first student's relationship: links=%+v err=%v", studentLinksAfterSharedEdit, err)
	}
	relatedLinksAfterSharedEdit, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, relatedStudentID, false)
	if err != nil || len(relatedLinksAfterSharedEdit) != 1 ||
		relatedLinksAfterSharedEdit[0].GuardianName != "Shared Guardian Profile Updated" ||
		relatedLinksAfterSharedEdit[0].RelationshipType != "Father" ||
		!relatedLinksAfterSharedEdit[0].IsPrimaryContact || relatedLinksAfterSharedEdit[0].IsEmergencyContact {
		t.Fatalf("shared profile update changed or failed to propagate to related student's relationship: links=%+v err=%v", relatedLinksAfterSharedEdit, err)
	}
	relatedStudents, err := repo.ListStudentRelatedStudents(ctx, targetPrincipalID, schoolID, studentID)
	if err != nil || len(relatedStudents) != 1 || relatedStudents[0].StudentID != relatedStudentID ||
		len(relatedStudents[0].SharedGuardianNames) != 1 || relatedStudents[0].SharedGuardianNames[0] != "Shared Guardian Profile Updated" {
		t.Fatalf("list students sharing active guardian: want student_id=%d guardian=%q; items=%+v err=%v", relatedStudentID, firstLink.GuardianName, relatedStudents, err)
	}

	secondMobile := "+919876543210"
	secondGuardian, err := repo.SaveGuardian(ctx, targetPrincipalID, schoolID, 0,
		"Guardian Two", &secondMobile, nil, nil, nil, nil, nil, "active")
	if err != nil {
		t.Fatalf("create second guardian contact: %v", err)
	}
	secondLink, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, secondGuardian.ID, "mother", true, false)
	if err != nil {
		t.Fatalf("link second primary guardian: %v", err)
	}
	if !secondLink.IsPrimaryContact {
		t.Fatalf("second link should be primary: %+v", secondLink)
	}

	// The database itself must reject a second active primary contact even
	// when a writer bypasses the repository's per-student serialization.
	var thirdGuardianID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO guardians (school_id, full_name)
		VALUES ($1, 'Guardian Three') RETURNING id
	`, schoolID).Scan(&thirdGuardianID); err != nil {
		t.Fatalf("create third guardian for primary-contact constraint test: %v", err)
	}
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin primary-contact constraint savepoint: %v", err)
	}
	_, duplicatePrimaryErr := savepoint.Exec(ctx, `
		INSERT INTO student_guardians (
			school_id, student_id, guardian_id, relationship_type,
			is_primary_contact, is_emergency_contact
		) VALUES ($1, $2, $3, 'Aunt', TRUE, FALSE)
	`, schoolID, studentID, thirdGuardianID)
	if duplicatePrimaryErr == nil || !strings.Contains(duplicatePrimaryErr.Error(), "ux_student_guardians_active_primary_per_student") {
		_ = savepoint.Rollback(ctx)
		t.Fatalf("duplicate active primary contact error = %v, want unique index violation", duplicatePrimaryErr)
	}
	if err := savepoint.Rollback(ctx); err != nil {
		t.Fatalf("rollback primary-contact constraint savepoint: %v", err)
	}
	if _, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID, studentID, thirdGuardianID, "Father", false, false); !errors.Is(err, ErrGuardianRelationshipConflict) {
		t.Fatalf("second active Father relationship error = %v, want ErrGuardianRelationshipConflict", err)
	}
	if _, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, schoolID, firstLink.ID, "Mother", false, false); !errors.Is(err, ErrGuardianRelationshipConflict) {
		t.Fatalf("editing Father link to an already-active Mother relationship error = %v, want ErrGuardianRelationshipConflict", err)
	}
	duplicateRelationshipSavepoint, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin duplicate relationship constraint savepoint: %v", err)
	}
	_, duplicateRelationshipErr := duplicateRelationshipSavepoint.Exec(ctx, `
		INSERT INTO student_guardians (school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'father')
	`, schoolID, studentID, thirdGuardianID)
	if duplicateRelationshipErr == nil || !strings.Contains(duplicateRelationshipErr.Error(), "active guardian relationship type already exists") {
		_ = duplicateRelationshipSavepoint.Rollback(ctx)
		t.Fatalf("duplicate active relationship error = %v, want relationship-type trigger rejection", duplicateRelationshipErr)
	}
	if err := duplicateRelationshipSavepoint.Rollback(ctx); err != nil {
		t.Fatalf("rollback duplicate relationship constraint savepoint: %v", err)
	}
	invalidRelationshipSavepoint, err := tx.Begin(ctx)
	if err != nil {
		t.Fatalf("begin invalid relationship constraint savepoint: %v", err)
	}
	_, invalidRelationshipErr := invalidRelationshipSavepoint.Exec(ctx, `
		INSERT INTO student_guardians (school_id, student_id, guardian_id, relationship_type)
		VALUES ($1, $2, $3, 'Step-father')
	`, schoolID, studentID, thirdGuardianID)
	if invalidRelationshipErr == nil || !strings.Contains(invalidRelationshipErr.Error(), "guardian relationship type is not allowed") {
		_ = invalidRelationshipSavepoint.Rollback(ctx)
		t.Fatalf("unsupported guardian relationship error = %v, want database allowlist rejection", invalidRelationshipErr)
	}
	if err := invalidRelationshipSavepoint.Rollback(ctx); err != nil {
		t.Fatalf("rollback invalid relationship constraint savepoint: %v", err)
	}

	editedLink, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "grandfather", true, false)
	if err != nil {
		t.Fatalf("update student-specific guardian relationship: %v", err)
	}
	if editedLink.ID != firstLink.ID || editedLink.RelationshipType != "Grandfather" ||
		!editedLink.IsPrimaryContact || editedLink.IsEmergencyContact {
		t.Fatalf("unexpected edited guardian relationship: %+v", editedLink)
	}

	var relationshipUpdatedAt time.Time
	var relationshipAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM student_guardians WHERE id=$1 AND school_id=$2`, firstLink.ID, schoolID).Scan(&relationshipUpdatedAt); err != nil {
		t.Fatalf("read guardian relationship updated_at: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student_guardian_link_updated' AND resource_id=$3`,
		schoolID, targetPrincipalID, fmt.Sprint(firstLink.ID)).Scan(&relationshipAuditCount); err != nil {
		t.Fatalf("count guardian relationship update audit events: %v", err)
	}
	unchangedLink, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "grandfather", true, false)
	if err != nil || unchangedLink == nil || unchangedLink.ID != firstLink.ID ||
		unchangedLink.RelationshipType != "Grandfather" || !unchangedLink.IsPrimaryContact || unchangedLink.IsEmergencyContact {
		t.Fatalf("retry identical student guardian relationship update: link=%+v err=%v", unchangedLink, err)
	}
	var retryRelationshipUpdatedAt time.Time
	var retryRelationshipAuditCount int
	if err := tx.QueryRow(ctx, `SELECT updated_at FROM student_guardians WHERE id=$1 AND school_id=$2`, firstLink.ID, schoolID).Scan(&retryRelationshipUpdatedAt); err != nil {
		t.Fatalf("read guardian relationship updated_at after retry: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE school_id=$1 AND actor_principal_id=$2 AND action='school.student_guardian_link_updated' AND resource_id=$3`,
		schoolID, targetPrincipalID, fmt.Sprint(firstLink.ID)).Scan(&retryRelationshipAuditCount); err != nil {
		t.Fatalf("count guardian relationship update audit events after retry: %v", err)
	}
	if !retryRelationshipUpdatedAt.Equal(relationshipUpdatedAt) || retryRelationshipAuditCount != relationshipAuditCount || relationshipAuditCount != 1 {
		t.Fatalf("identical guardian relationship retry changed timestamp/audit count: timestamp before=%s after=%s audits before=%d after=%d",
			relationshipUpdatedAt, retryRelationshipUpdatedAt, relationshipAuditCount, retryRelationshipAuditCount)
	}
	active, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, false)
	if err != nil || len(active) != 2 {
		t.Fatalf("list active links after relationship update: items=%+v err=%v", active, err)
	}
	for _, link := range active {
		if link.ID == secondLink.ID && link.IsPrimaryContact {
			t.Fatal("updating primary guardian did not clear previous primary contact")
		}
	}
	relatedLink, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, relatedStudentID, false)
	if err != nil || len(relatedLink) != 1 || relatedLink[0].RelationshipType != "Father" ||
		!relatedLink[0].IsPrimaryContact || relatedLink[0].IsEmergencyContact {
		t.Fatalf("editing one student's relationship changed another student's shared guardian link: links=%+v err=%v", relatedLink, err)
	}

	history, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(history) != 2 {
		t.Fatalf("list complete student guardian history: count=%d err=%v", len(history), err)
	}
	// Primary-contact state is asserted against the active-link query above.
	// includeHistory broadens the result to revoked links; it is not a separate
	// historical snapshot of every prior field value.
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "relationship ended"); err != nil {
		t.Fatalf("revoke student guardian link: %v", err)
	}
	if _, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, schoolID, firstLink.ID, "mother", false, false); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("revoked guardian relationship update error = %v, want not found", err)
	}
	active, err = repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, false)
	if err != nil || len(active) != 1 || active[0].ID != secondLink.ID {
		t.Fatalf("active links after revocation: items=%+v err=%v", active, err)
	}
	relatedStudents, err = repo.ListStudentRelatedStudents(ctx, targetPrincipalID, schoolID, studentID)
	if err != nil || len(relatedStudents) != 0 {
		t.Fatalf("related students after shared guardian revocation: items=%+v err=%v", relatedStudents, err)
	}
	history, err = repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(history) != 2 {
		t.Fatalf("historical links after revocation: count=%d err=%v", len(history), err)
	}
	var revokedFound bool
	for _, link := range history {
		if link.ID == firstLink.ID {
			revokedFound = link.RevokedAt != nil && link.RevocationReason != nil &&
				*link.RevocationReason == "relationship ended"
		}
	}
	if !revokedFound {
		t.Fatalf("revoked link history must retain its timestamp and reason: %+v", history)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		firstLink.ID, "duplicate revoke"); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("duplicate link revocation error = %v, want not found", err)
	}

	// Relationship changes are allowed only while the owning student is active.
	if _, err := tx.Exec(ctx, "UPDATE students SET status='inactive' WHERE id=$1 AND school_id=$2", studentID, schoolID); err != nil {
		t.Fatalf("deactivate student for relationship edit boundary: %v", err)
	}
	if _, err := repo.UpdateStudentGuardianLink(ctx, targetPrincipalID, schoolID, secondLink.ID, "mother", false, false); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("inactive student's guardian relationship update error = %v, want not found", err)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID, secondLink.ID, "inactive student boundary"); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("inactive student's guardian relationship revoke error = %v, want not found", err)
	}
	inactiveHistory, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(inactiveHistory) != 2 {
		t.Fatalf("inactive student guardian history should retain both relationships: links=%+v err=%v", inactiveHistory, err)
	}
	var originalLinkStillRevoked, remainingLinkStillActive bool
	for _, link := range inactiveHistory {
		if link.ID == firstLink.ID { originalLinkStillRevoked = link.RevokedAt != nil }
		if link.ID == secondLink.ID { remainingLinkStillActive = link.RevokedAt == nil }
	}
	if !originalLinkStillRevoked || !remainingLinkStillActive {
		t.Fatalf("inactive student guardian history changed unexpectedly: links=%+v", inactiveHistory)
	}
	if _, err := tx.Exec(ctx, "UPDATE students SET status='active' WHERE id=$1 AND school_id=$2", studentID, schoolID); err != nil {
		t.Fatalf("restore active student after relationship edit boundary: %v", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE students SET status='archived' WHERE id=$1 AND school_id=$2", studentID, schoolID); err != nil {
		t.Fatalf("archive student for guardian revocation boundary: %v", err)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID, secondLink.ID, "archived student boundary"); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("archived student's guardian relationship revoke error = %v, want not found", err)
	}
	archivedHistory, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(archivedHistory) != 2 {
		t.Fatalf("archived student guardian history should retain both relationships: links=%+v err=%v", archivedHistory, err)
	}
	var archivedOriginalLinkStillRevoked, archivedRemainingLinkStillActive bool
	for _, link := range archivedHistory {
		if link.ID == firstLink.ID { archivedOriginalLinkStillRevoked = link.RevokedAt != nil }
		if link.ID == secondLink.ID { archivedRemainingLinkStillActive = link.RevokedAt == nil }
	}
	if !archivedOriginalLinkStillRevoked || !archivedRemainingLinkStillActive {
		t.Fatalf("archived student guardian history changed unexpectedly: links=%+v", archivedHistory)
	}
	// Restore the main fixture to active before testing a new relationship.
	// The archived-state assertions above intentionally verify that mutations
	// are blocked while the student is archived.
	if _, err := tx.Exec(ctx, "UPDATE students SET status='active' WHERE id=$1 AND school_id=$2", studentID, schoolID); err != nil {
		t.Fatalf("restore active student before guardian re-link coverage: %v", err)
	}

	var guardianAuditCount, linkCreateAuditCount, linkUpdateAuditCount, linkRevokeAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE action IN ('school.guardian.created', 'school.guardian.updated')),
		  COUNT(*) FILTER (WHERE action = 'school.student_guardian_link_created'),
		  COUNT(*) FILTER (WHERE action = 'school.student_guardian_link_updated'),
		  COUNT(*) FILTER (WHERE action = 'school.student_guardian_link_revoked')
		FROM audit_events
		WHERE school_id = $1
		  AND actor_principal_id = $2
		  AND (
		    (resource_type = 'guardian' AND resource_id IN ($3, $4))
		    OR resource_type = 'student_guardian_link'
		  )
	`, schoolID, targetPrincipalID, fmt.Sprint(guardian.ID),
		fmt.Sprint(secondGuardian.ID)).Scan(
		&guardianAuditCount, &linkCreateAuditCount, &linkUpdateAuditCount, &linkRevokeAuditCount,
	); err != nil {
		t.Fatalf("count guardian audit events: %v", err)
	}
	if guardianAuditCount != 7 || linkCreateAuditCount != 3 || linkUpdateAuditCount != 1 || linkRevokeAuditCount != 1 {
		t.Fatalf("guardian audit counts = contacts %d, link creates %d, link updates %d, link revokes %d; want 7, 3, 1, 1",
			guardianAuditCount, linkCreateAuditCount, linkUpdateAuditCount, linkRevokeAuditCount)
	}

	var revocationAuditMetadataValid bool
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) = 1
		   AND BOOL_AND(metadata->>'student_id' IS NOT NULL)
	   AND BOOL_AND(metadata->>'has_revocation_reason' = 'true')
	   AND BOOL_AND(metadata::text NOT ILIKE '%relationship ended%')
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action='school.student_guardian_link_revoked'
		   AND resource_type='student_guardian_link'
		   AND resource_id=$3
	`, schoolID, targetPrincipalID, fmt.Sprint(firstLink.ID)).Scan(&revocationAuditMetadataValid); err != nil {
		t.Fatalf("validate guardian revocation audit metadata: %v", err)
	}
	if !revocationAuditMetadataValid {
		t.Fatal("guardian revocation audit must identify the student and reason presence without storing the private reason")
	}

	var guardianReadAuditCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		  FROM audit_events
		 WHERE school_id=$1 AND actor_principal_id=$2
		   AND action IN (
		       'school.guardian.list.read',
		       'school.guardian.read',
		       'school.student.guardians.read',
		       'school.student.related_students.read'
		   )
		   AND metadata ? 'section'
		   AND metadata ? 'result_count'
		   AND metadata::text NOT ILIKE '%'||$3||'%'
		   AND metadata::text NOT ILIKE '%'||$4||'%'
	`, schoolID, targetPrincipalID, mobile, email).Scan(&guardianReadAuditCount); err != nil {
		t.Fatalf("count guardian sensitive-read audit events: %v", err)
	}
	if guardianReadAuditCount < 8 {
		t.Fatalf("guardian sensitive-read audit count = %d, want at least 8 read events with metadata free of contact details", guardianReadAuditCount)
	}

	// Inactive students remain read-only: guardian relationship history is
	// preserved and cannot be changed through the staff family-management flow.
	var inactiveStudentID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO students (school_id, admission_number, full_name)
		VALUES ($1, $2, 'Inactive Guardian Student') RETURNING id
	`, schoolID, "INACTIVE-GUARDIAN-"+suffix).Scan(&inactiveStudentID); err != nil {
		t.Fatalf("create inactive-student revocation fixture: %v", err)
	}
	inactiveLink, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		inactiveStudentID, guardian.ID, "mother", false, false)
	if err != nil {
		t.Fatalf("link guardian before student deactivation: %v", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE students SET status='inactive' WHERE id=$1 AND school_id=$2", inactiveStudentID, schoolID); err != nil {
		t.Fatalf("deactivate guardian relationship fixture student: %v", err)
	}
	if err := repo.RevokeStudentGuardianLink(ctx, targetPrincipalID, schoolID,
		inactiveLink.ID, "student record inactive"); !errors.Is(err, ErrGuardianNotFound) {
		t.Fatalf("revoke guardian relationship for inactive student: got %v, want ErrGuardianNotFound", err)
	}
	inactiveGuardianHistory, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, inactiveStudentID, true)
	if err != nil || len(inactiveGuardianHistory) != 1 || inactiveGuardianHistory[0].RevokedAt != nil {
		t.Fatalf("inactive student guardian history must remain unchanged: links=%+v err=%v", inactiveGuardianHistory, err)
	}

	// Re-linking after revocation creates a new active relationship while
	// retaining the original revoked row as immutable history.
	relinked, err := repo.LinkStudentGuardian(ctx, targetPrincipalID, schoolID,
		studentID, guardian.ID, "father", false, true)
	if err != nil || relinked == nil {
		t.Fatalf("re-link guardian after revocation: link=%+v err=%v", relinked, err)
	}
	if relinked.ID == firstLink.ID || relinked.RelationshipType != "Father" ||
		relinked.IsPrimaryContact || !relinked.IsEmergencyContact {
		t.Fatalf("unexpected replacement guardian relationship: %+v", relinked)
	}
	relinkedHistory, err := repo.ListStudentGuardians(ctx, targetPrincipalID, schoolID, studentID, true)
	if err != nil || len(relinkedHistory) != 3 {
		t.Fatalf("guardian history after re-link: links=%+v err=%v", relinkedHistory, err)
	}
	var oldRevocationPreserved, newLinkActive bool
	for _, link := range relinkedHistory {
		if link.ID == firstLink.ID {
			oldRevocationPreserved = link.RevokedAt != nil
		}
		if link.ID == relinked.ID {
			newLinkActive = link.RevokedAt == nil
		}
	}
	if !oldRevocationPreserved || !newLinkActive {
		t.Fatalf("re-link changed old relationship history or failed to create active link: %+v", relinkedHistory)
	}

	target := &models.AuthPrincipal{
		ID: targetPrincipalID, Type: "staff", Status: "active",
		TokenVersion: 1, SchoolID: schoolID,
	}
	effective, err := loadEffectivePermissions(ctx, tx, target)
	if err != nil {
		t.Fatalf("resolve guardian actor live permissions: %v", err)
	}
	for _, key := range []string{"guardians.read", "guardians.manage", "students.read_basic"} {
		found := false
		for _, permission := range effective {
			if permission == key {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected live delegated permission %q, got %v", key, effective)
		}
	}


}
