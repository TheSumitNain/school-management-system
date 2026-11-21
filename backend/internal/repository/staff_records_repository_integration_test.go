package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"school-management-system/internal/models"

	"github.com/jackc/pgx/v5"
)

func TestStaffRecordsRepositoryPostgresIntegration(t *testing.T) {
	_, pool, actorUserID := newPostgresIntegrationRepository(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tx, err := pool.Begin(ctx)
	if err != nil { t.Fatalf("begin rollback-scoped staff records test: %v", err) }
	defer func() { _ = tx.Rollback(ctx) }()

	var actorID, schoolID int64
	if err := tx.QueryRow(ctx, `SELECT principal_id,school_id FROM users WHERE id=$1`, actorUserID).Scan(&actorID,&schoolID); err != nil {
		t.Fatalf("load School Admin fixture: %v",err)
	}

	var platformAdminID int64
	err = tx.QueryRow(ctx,`
		SELECT ap.id
		FROM platform_admin_accounts pa
		JOIN auth_principals ap ON ap.id=pa.principal_id
		JOIN principal_roles pr ON pr.principal_id=ap.id AND pr.revoked_at IS NULL
		JOIN roles r ON r.id=pr.role_id
		JOIN role_permissions rp ON rp.role_id=r.id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE ap.principal_type='platform_admin' AND ap.status='active'
		  AND r.role_key='platform_admin' AND r.scope_type='platform' AND r.is_system=TRUE
		  AND p.permission_key='platform.permissions.delegate' AND p.scope_type='platform' AND p.is_active=TRUE
		LIMIT 1
	`).Scan(&platformAdminID)
	if errors.Is(err,pgx.ErrNoRows) {
		var anyBootstrap bool
		if checkErr:=tx.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM platform_admin_accounts)`).Scan(&anyBootstrap);checkErr!=nil{t.Fatalf("check Platform Admin bootstrap: %v",checkErr)}
		if anyBootstrap { t.Skip("Platform Admin exists but no active administrator can delegate permissions") }
		platformAdminID,err=NewPlatformAdminRepository(pool).bootstrapFirstInTx(ctx,tx,fmt.Sprintf("staff-records-%d@example.invalid",time.Now().UnixNano()),"integration-test-password-hash")
	}
	if err!=nil { t.Fatalf("resolve Platform Admin fixture: %v",err) }

	var schoolAdminRoleID int64
	if err:=tx.QueryRow(ctx,`SELECT id FROM roles WHERE school_id=$1 AND principal_type='staff' AND scope_type='school' AND role_key='school_admin' AND is_system=TRUE`,schoolID).Scan(&schoolAdminRoleID);err!=nil{
		t.Fatalf("load School Admin role: %v",err)
	}
	if _,err:=tx.Exec(ctx,`INSERT INTO principal_roles(principal_id,role_id,assigned_by) VALUES($1,$2,$3)`,actorID,schoolAdminRoleID,platformAdminID);err!=nil{
		t.Fatalf("assign School Admin fixture role: %v",err)
	}

	repo:=NewUserRepository(tx)
	suffix:=fmt.Sprintf("%d",time.Now().UnixNano())
	staff,err:=repo.CreateStaffAccountWithProfile(ctx,actorID,schoolID,"staffrecord-"+suffix,"integration-test-password-hash",models.StaffProfile{
		EmployeeCode:"SR-"+suffix,FullName:"Staff Record Integration",Designation:"Teacher",MobileNumber:"9876543210",EmploymentStatus:"active",
	})
	if err!=nil{t.Fatalf("create staff profile fixture: %v",err)}

	year:=2021
	score:=86.25
	qualification,err:=repo.CreateStaffQualification(ctx,actorID,schoolID,staff.PrincipalID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"M.Sc. Mathematics",Institution:"State University",CompletionYear:&year,Score:&score,ScoreScale:"percentage",
	})
	if err!=nil{t.Fatalf("create staff qualification: %v",err)}
	if qualification.ID<=0||qualification.VerificationStatus!="pending"{t.Fatalf("unexpected qualification response: %+v",qualification)}

	certification,err:=repo.CreateStaffCertification(ctx,actorID,schoolID,staff.PrincipalID,StaffCertification{
		CertificateName:"Teacher Training",IssuingOrganization:"State Board",IssueDate:"2022-01-10",ExpiryDate:"2027-01-10",
	})
	if err!=nil{t.Fatalf("create staff certification: %v",err)}
	if certification.ID<=0||certification.VerificationStatus!="pending"{t.Fatalf("unexpected certification response: %+v",certification)}

	experience,err:=repo.CreateStaffExperience(ctx,actorID,schoolID,staff.PrincipalID,StaffExperience{
		EmployerName:"Previous School",Designation:"Mathematics Teacher",EmploymentType:"permanent",StartDate:"2018-04-01",EndDate:"2021-03-31",
	})
	if err!=nil{t.Fatalf("create staff experience: %v",err)}
	if experience.ID<=0{t.Fatalf("unexpected experience response: %+v",experience)}

	updatedQualification,err:=repo.UpdateStaffQualification(ctx,actorID,schoolID,staff.PrincipalID,qualification.ID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"M.Sc. Applied Mathematics",Institution:"State University",CompletionYear:&year,Score:&score,ScoreScale:"percentage",
	})
	if err!=nil||updatedQualification.QualificationName!="M.Sc. Applied Mathematics"{t.Fatalf("update qualification: item=%+v err=%v",updatedQualification,err)}
	updatedCertification,err:=repo.UpdateStaffCertification(ctx,actorID,schoolID,staff.PrincipalID,certification.ID,StaffCertification{
		CertificateName:"Advanced Teacher Training",IssuingOrganization:"State Board",IssueDate:"2022-01-10",ExpiryDate:"2028-01-10",
	})
	if err!=nil||updatedCertification.CertificateName!="Advanced Teacher Training"{t.Fatalf("update certification: item=%+v err=%v",updatedCertification,err)}
	updatedExperience,err:=repo.UpdateStaffExperience(ctx,actorID,schoolID,staff.PrincipalID,experience.ID,StaffExperience{
		EmployerName:"Previous School",Designation:"Senior Mathematics Teacher",EmploymentType:"permanent",StartDate:"2018-04-01",EndDate:"2021-03-31",
	})
	if err!=nil||updatedExperience.Designation!="Senior Mathematics Teacher"{t.Fatalf("update experience: item=%+v err=%v",updatedExperience,err)}
	if _,err:=repo.UpdateStaffQualification(ctx,actorID,schoolID+1,staff.PrincipalID,qualification.ID,StaffQualification{
		QualificationType:"postgraduate",QualificationName:"Cross tenant",Institution:"Denied",
	});!errors.Is(err,ErrStaffRecordAccessDenied){t.Fatalf("cross-school qualification update error=%v",err)}

	qualifications,err:=repo.ListStaffQualifications(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(qualifications)!=1||qualifications[0].ID!=qualification.ID{t.Fatalf("list qualifications: items=%+v err=%v",qualifications,err)}
	certifications,err:=repo.ListStaffCertifications(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(certifications)!=1||certifications[0].ID!=certification.ID{t.Fatalf("list certifications: items=%+v err=%v",certifications,err)}
	experiences,err:=repo.ListStaffExperience(ctx,actorID,schoolID,staff.PrincipalID)
	if err!=nil||len(experiences)!=1||experiences[0].ID!=experience.ID{t.Fatalf("list experience: items=%+v err=%v",experiences,err)}

	if _,err:=repo.ListStaffQualifications(ctx,actorID,schoolID+1,staff.PrincipalID);!errors.Is(err,ErrStaffRecordAccessDenied){
		t.Fatalf("cross-school record read error=%v want ErrStaffRecordAccessDenied",err)
	}
	var auditCount int
	if err:=tx.QueryRow(ctx,`SELECT COUNT(*) FROM audit_events WHERE actor_principal_id=$1 AND school_id=$2 AND action IN ('staff.qualification.created','staff.certification.created','staff.experience.created','staff.qualification.updated','staff.certification.updated','staff.experience.updated')`,actorID,schoolID).Scan(&auditCount);err!=nil{
		t.Fatalf("count staff record audit events: %v",err)
	}
	if auditCount!=6{t.Fatalf("audit count=%d want=6 (three creates and three updates)",auditCount)}
}
