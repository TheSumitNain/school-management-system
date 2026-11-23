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
func(f *fakeStaffRecordsRepository) ListStaffQualificationReport(_ context.Context,_,_ int64,filter repository.StaffQualificationReportFilter)(*repository.StaffQualificationReportPage,error){return &repository.StaffQualificationReportPage{Rows:[]repository.StaffQualificationReportRow{},Counts:[]repository.StaffQualificationReportCount{},Limit:filter.Limit,Offset:filter.Offset},nil}
func(f *fakeStaffRecordsRepository) ListStaffExperienceReport(_ context.Context,_,_ int64,filter repository.StaffExperienceReportFilter)(*repository.StaffExperienceReportPage,error){return &repository.StaffExperienceReportPage{Rows:[]repository.StaffExperienceReportRow{},Counts:[]repository.StaffExperienceReportCount{},Limit:filter.Limit,Offset:filter.Offset},nil}
func(f *fakeStaffRecordsRepository) CreateStaffQualification(_ context.Context,_,_,_ int64,item repository.StaffQualification)(*repository.StaffQualification,error){f.calls++;f.qualification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) ListStaffCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error){return nil,nil}
func(f *fakeStaffRecordsRepository) CreateStaffCertification(_ context.Context,_,_,_ int64,item repository.StaffCertification)(*repository.StaffCertification,error){f.calls++;f.certification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) ListStaffExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error){return nil,nil}
func(f *fakeStaffRecordsRepository) CreateStaffExperience(_ context.Context,_,_,_ int64,item repository.StaffExperience)(*repository.StaffExperience,error){f.calls++;f.experience=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffQualification(_ context.Context,_,_,_,_ int64,item repository.StaffQualification)(*repository.StaffQualification,error){f.calls++;f.qualification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffCertification(_ context.Context,_,_,_,_ int64,item repository.StaffCertification)(*repository.StaffCertification,error){f.calls++;f.certification=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) UpdateStaffExperience(_ context.Context,_,_,_,_ int64,item repository.StaffExperience)(*repository.StaffExperience,error){f.calls++;f.experience=&item;return &item,nil}
func(f *fakeStaffRecordsRepository) ListStaffAuditEvents(context.Context,int64,int64,int64)([]repository.StaffAuditEvent,error){return nil,nil}

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

func TestStaffRecordsServiceValidatesQualificationReportFilters(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	page, err := svc.ListQualificationsReport(context.Background(),1,2,repository.StaffQualificationReportFilter{})
	if err != nil { t.Fatalf("default report filter: %v",err) }
	if page.Limit != 50 || page.Offset != 0 { t.Fatalf("unexpected defaults: %+v",page) }
	for _, filter := range []repository.StaffQualificationReportFilter{
		{RecordType:"documents"},
		{VerificationStatus:"unknown"},
		{Limit:201},
		{Offset:-1},
		{Offset:1000001},
	} {
		if _, err := svc.ListQualificationsReport(context.Background(),1,2,filter); err != ErrInvalidStaffRecord {
			t.Fatalf("filter %+v error=%v want=%v",filter,err,ErrInvalidStaffRecord)
		}
	}
}

func TestStaffRecordsServiceValidatesExperienceReportFilters(t *testing.T) {
    repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
    page,err:=svc.ListExperienceReport(context.Background(),1,2,repository.StaffExperienceReportFilter{})
    if err!=nil {t.Fatalf("default experience report filter: %v",err)}
    if page.Limit!=50 || page.Offset!=0 {t.Fatalf("unexpected defaults: %+v",page)}
    for _,filter:=range []repository.StaffExperienceReportFilter{{EmploymentType:"unknown"},{Limit:201},{Offset:-1},{Offset:1000001}} {
        if _,err:=svc.ListExperienceReport(context.Background(),1,2,filter);err!=ErrInvalidStaffRecord {t.Fatalf("filter %+v error=%v want=%v",filter,err,ErrInvalidStaffRecord)}
    }
}

func TestStaffRecordsServiceRejectsInvalidCertificationCalendarDates(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	for _, value := range []string{"0000-01-01", "2026-2-01", "2026-02-30"} {
		for _, update := range []bool{false, true} {
			item := repository.StaffCertification{CertificateName: "Training", IssuingOrganization: "Board", IssueDate: value}
			var err error
			if update {
				_, err = svc.UpdateCertification(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateCertification(context.Background(), 1, 2, 3, item)
			}
			if err != ErrInvalidStaffRecord {
				t.Fatalf("certification date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
			}
		}
	}
	if repo.calls != 0 {
		t.Fatalf("invalid certification dates reached repository %d times", repo.calls)
	}
}

func TestStaffRecordsServiceComparesYearOneCertificationDates(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	item := repository.StaffCertification{
		CertificateName: "Training", IssuingOrganization: "Board",
		IssueDate: "0001-01-02", ExpiryDate: "0001-01-01",
	}
	if _, err := svc.CreateCertification(context.Background(), 1, 2, 3, item); err != ErrInvalidStaffRecord {
		t.Fatalf("reversed year-one certification dates error=%v want ErrInvalidStaffRecord", err)
	}
	if repo.calls != 0 {
		t.Fatalf("reversed year-one certification reached repository %d times", repo.calls)
	}
	item.ExpiryDate = "0001-01-03"
	if _, err := svc.CreateCertification(context.Background(), 1, 2, 3, item); err != nil {
		t.Fatalf("valid year-one certification dates: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("valid certification repository calls=%d want=1", repo.calls)
	}
}

func TestStaffRecordsServiceRejectsInvalidExperienceCalendarDates(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	for _, value := range []string{"0000-01-01", "2026-2-01", "2026-02-30"} {
		for _, update := range []bool{false, true} {
			item := repository.StaffExperience{
				EmployerName: "School", Designation: "Teacher", EmploymentType: "permanent",
				StartDate: value,
			}
			var err error
			if update {
				_, err = svc.UpdateExperience(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateExperience(context.Background(), 1, 2, 3, item)
			}
			if err != ErrInvalidStaffRecord {
				t.Fatalf("experience date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
			}
		}
	}
	if repo.calls != 0 {
		t.Fatalf("invalid experience dates reached repository %d times", repo.calls)
	}
}
