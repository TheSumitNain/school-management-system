package services

import (
	"context"
	"testing"

	"school-management-system/internal/repository"
)

type fakeStaffRecordsRepository struct {
	qualification *repository.StaffQualification
	certification *repository.StaffCertification
	experience *repository.StaffExperience
	calls int
}
func(f *fakeStaffRecordsRepository) ListStaffQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error){return nil,nil}
func(f *fakeStaffRecordsRepository) CreateStaffQualification(_ context.Context,_,_,_ int64,item repository.StaffQualification)(*repository.StaffQualification,error){f.calls++;f.qualification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) ListStaffCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error){return nil,nil}
func(f *fakeStaffRecordsRepository) CreateStaffCertification(_ context.Context,_,_,_ int64,item repository.StaffCertification)(*repository.StaffCertification,error){f.calls++;f.certification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) ListStaffExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error){return nil,nil}
func(f *fakeStaffRecordsRepository) CreateStaffExperience(_ context.Context,_,_,_ int64,item repository.StaffExperience)(*repository.StaffExperience,error){f.calls++;f.experience=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffQualification(_ context.Context,_,_,_,_ int64,item repository.StaffQualification)(*repository.StaffQualification,error){f.calls++;f.qualification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffCertification(_ context.Context,_,_,_,_ int64,item repository.StaffCertification)(*repository.StaffCertification,error){f.calls++;f.certification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffExperience(_ context.Context,_,_,_,_ int64,item repository.StaffExperience)(*repository.StaffExperience,error){f.calls++;f.experience=&item;return &item,nil}

func TestStaffRecordsServiceValidatesAndNormalizesQualification(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
	year:=2020;score:=82.5
	got,err:=svc.CreateQualification(context.Background(),1,2,3,repository.StaffQualification{
		QualificationType:"postgraduate",QualificationName:"  M.Sc. Mathematics  ",Institution:"  State University  ",
		CompletionYear:&year,Score:&score,ScoreScale:"percentage",
	})
	if err!=nil{t.Fatalf("create valid qualification: %v",err)}
	if got.QualificationName!="M.Sc. Mathematics"||got.Institution!="State University"||repo.calls!=1{t.Fatalf("qualification was not normalized or saved: %+v calls=%d",got,repo.calls)}
}
func TestStaffRecordsServiceRejectsInvalidQualificationScorePair(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo);score:=88.0
	_,err:=svc.CreateQualification(context.Background(),1,2,3,repository.StaffQualification{QualificationType:"undergraduate",QualificationName:"B.A.",Institution:"College",Score:&score})
	if err!=ErrInvalidStaffRecord{t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid qualification reached repository")}
}
func TestStaffRecordsServiceValidatesCertificationDates(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
	_,err:=svc.CreateCertification(context.Background(),1,2,3,repository.StaffCertification{CertificateName:"Teacher Training",IssuingOrganization:"Board",IssueDate:"2025-05-01",ExpiryDate:"2024-05-01"})
	if err!=ErrInvalidStaffRecord{t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid certification reached repository")}
}
func TestStaffRecordsServiceValidatesCurrentExperienceEndDate(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
	_,err:=svc.CreateExperience(context.Background(),1,2,3,repository.StaffExperience{EmployerName:"School",Designation:"Teacher",EmploymentType:"permanent",StartDate:"2020-01-01",EndDate:"2024-01-01",IsCurrent:true})
	if err!=ErrInvalidStaffRecord{t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid experience reached repository")}
}

func TestStaffRecordsServiceRejectsInvalidUpdatesBeforeRepository(t *testing.T) {
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo);score:=90.0
	_,err:=svc.UpdateQualification(context.Background(),1,2,3,4,repository.StaffQualification{QualificationType:"postgraduate",QualificationName:"M.Sc.",Institution:"University",Score:&score})
	if err!=ErrInvalidStaffRecord{t.Fatalf("qualification update error=%v want=%v",err,ErrInvalidStaffRecord)}
	_,err=svc.UpdateCertification(context.Background(),1,2,3,5,repository.StaffCertification{CertificateName:"Training",IssuingOrganization:"Board",IssueDate:"2025-01-01",ExpiryDate:"2024-01-01"})
	if err!=ErrInvalidStaffRecord{t.Fatalf("certification update error=%v want=%v",err,ErrInvalidStaffRecord)}
	_,err=svc.UpdateExperience(context.Background(),1,2,3,6,repository.StaffExperience{EmployerName:"School",Designation:"Teacher",EmploymentType:"permanent",StartDate:"2020-01-01",EndDate:"2019-01-01"})
	if err!=ErrInvalidStaffRecord{t.Fatalf("experience update error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatalf("invalid updates reached repository %d times",repo.calls)}
}
