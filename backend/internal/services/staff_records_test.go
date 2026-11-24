package services

import (
	"context"
	"errors"
	"testing"
	"math"
	"strings"

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
func(f *fakeStaffRecordsRepository) ListStaffAuditEvents(context.Context,int64,int64,int64,repository.StaffAuditEventFilter)(*repository.StaffAuditEventPage,error){return &repository.StaffAuditEventPage{Events:[]repository.StaffAuditEvent{}},nil}

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
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid qualification reached repository")}
}
func TestStaffRecordsServiceValidatesCertificationDates(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
	_,err:=svc.CreateCertification(context.Background(),1,2,3,repository.StaffCertification{CertificateName:"Teacher Training",IssuingOrganization:"Board",IssueDate:"2025-05-01",ExpiryDate:"2024-05-01"})
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid certification reached repository")}
}
func TestStaffRecordsServiceValidatesCurrentExperienceEndDate(t *testing.T){
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo)
	_,err:=svc.CreateExperience(context.Background(),1,2,3,repository.StaffExperience{EmployerName:"School",Designation:"Teacher",EmploymentType:"permanent",StartDate:"2020-01-01",EndDate:"2024-01-01",IsCurrent:true})
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("error=%v want=%v",err,ErrInvalidStaffRecord)}
	if repo.calls!=0{t.Fatal("invalid experience reached repository")}
}

func TestStaffRecordsServiceRejectsInvalidUpdatesBeforeRepository(t *testing.T) {
	repo:=&fakeStaffRecordsRepository{};svc:=NewStaffRecordsService(repo);score:=90.0
	_,err:=svc.UpdateQualification(context.Background(),1,2,3,4,repository.StaffQualification{QualificationType:"postgraduate",QualificationName:"M.Sc.",Institution:"University",Score:&score})
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("qualification update error=%v want=%v",err,ErrInvalidStaffRecord)}
	_,err=svc.UpdateCertification(context.Background(),1,2,3,5,repository.StaffCertification{CertificateName:"Training",IssuingOrganization:"Board",IssueDate:"2025-01-01",ExpiryDate:"2024-01-01"})
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("certification update error=%v want=%v",err,ErrInvalidStaffRecord)}
	_,err=svc.UpdateExperience(context.Background(),1,2,3,6,repository.StaffExperience{EmployerName:"School",Designation:"Teacher",EmploymentType:"permanent",StartDate:"2020-01-01",EndDate:"2019-01-01"})
	if !errors.Is(err,ErrInvalidStaffRecord){t.Fatalf("experience update error=%v want=%v",err,ErrInvalidStaffRecord)}
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
		if _, err := svc.ListQualificationsReport(context.Background(),1,2,filter); !errors.Is(err, ErrInvalidStaffRecord) {
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
        if _,err:=svc.ListExperienceReport(context.Background(),1,2,filter);!errors.Is(err,ErrInvalidStaffRecord) {t.Fatalf("filter %+v error=%v want=%v",filter,err,ErrInvalidStaffRecord)}
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
			if !errors.Is(err, ErrInvalidStaffRecord) {
				t.Fatalf("certification issue date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
			}
			item = repository.StaffCertification{CertificateName: "Training", IssuingOrganization: "Board", ExpiryDate: value}
			if update {
				_, err = svc.UpdateCertification(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateCertification(context.Background(), 1, 2, 3, item)
			}
			if !errors.Is(err, ErrInvalidStaffRecord) {
				t.Fatalf("certification expiry date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
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
	if _, err := svc.CreateCertification(context.Background(), 1, 2, 3, item); !errors.Is(err, ErrInvalidStaffRecord) {
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
			if !errors.Is(err, ErrInvalidStaffRecord) {
				t.Fatalf("experience start date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
			}
			item = repository.StaffExperience{
				EmployerName: "School", Designation: "Teacher", EmploymentType: "permanent",
				StartDate: "2020-01-01", EndDate: value,
			}
			if update {
				_, err = svc.UpdateExperience(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateExperience(context.Background(), 1, 2, 3, item)
			}
			if !errors.Is(err, ErrInvalidStaffRecord) {
				t.Fatalf("experience end date %q update=%v error=%v want ErrInvalidStaffRecord", value, update, err)
			}
		}
	}
	if repo.calls != 0 {
		t.Fatalf("invalid experience dates reached repository %d times", repo.calls)
	}
}


func TestStaffRecordsRejectsNonFiniteAndOutOfRangeQualificationScores(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	for _, score := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 100.01, 100000.0} {
		for _, update := range []bool{false, true} {
			value := score
			item := repository.StaffQualification{
				QualificationType: "undergraduate",
				QualificationName: "B.Sc.",
				Institution: "University",
				Score: &value,
				ScoreScale: "percentage",
			}
			var err error
			if update {
				_, err = svc.UpdateQualification(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateQualification(context.Background(), 1, 2, 3, item)
			}
			if !errors.Is(err, ErrInvalidStaffRecord) {
				t.Fatalf("score=%v update=%v error=%v want ErrInvalidStaffRecord", score, update, err)
			}
		}
	}
	if repo.calls != 0 {
		t.Fatalf("invalid qualification scores reached repository %d times", repo.calls)
	}
	valid := 100.0
	item := repository.StaffQualification{
		QualificationType: "undergraduate",
		QualificationName: "B.Sc.",
		Institution: "University",
		Score: &valid,
		ScoreScale: "percentage",
	}
	if _, err := svc.CreateQualification(context.Background(), 1, 2, 3, item); err != nil {
		t.Fatalf("valid 100%% qualification score rejected: %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("valid qualification repository calls=%d want=1", repo.calls)
	}

	maxPrecision := 99999.999
	maxItem := repository.StaffQualification{
		QualificationType: "undergraduate",
		QualificationName: "B.Sc.",
		Institution: "University",
		Score: &maxPrecision,
		ScoreScale: "marks",
	}
	if _, err := svc.CreateQualification(context.Background(), 1, 2, 3, maxItem); err != nil {
		t.Fatalf("maximum NUMERIC(8,3) qualification score rejected: %v", err)
	}
	if repo.calls != 2 {
		t.Fatalf("valid qualification repository calls=%d want=2", repo.calls)
	}
}

func TestStaffRecordsServiceReturnsFieldSpecificValidationErrors(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	_, err := svc.CreateQualification(context.Background(), 1, 2, 3, repository.StaffQualification{
		QualificationType: "undergraduate",
		QualificationName: "",
		Institution: "University",
	})
	var validation *StaffRecordFieldValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error type=%T want StaffRecordFieldValidationError", err)
	}
	if !errors.Is(err, ErrInvalidStaffRecord) {
		t.Fatalf("field validation error must remain compatible with ErrInvalidStaffRecord: %v", err)
	}
	if got := validation.FieldErrors["qualification_name"]; got == "" {
		t.Fatalf("qualification_name field error missing: %+v", validation.FieldErrors)
	}
	if repo.calls != 0 {
		t.Fatalf("invalid qualification reached repository %d times", repo.calls)
	}
}

func TestQualificationScoreValidationFieldTargetsScoreOrScale(t *testing.T) {
	valid := 95.0
	invalid := 101.0
	cases := []struct {
		name string
		score *float64
		scale string
		want string
	}{
		{name: "no score and no scale", want: ""},
		{name: "scale without score", scale: "percentage", want: "score_scale"},
		{name: "score without scale", score: &valid, want: "score_scale"},
		{name: "unknown scale", score: &valid, scale: "unknown", want: "score_scale"},
		{name: "out of range percentage", score: &invalid, scale: "percentage", want: "score"},
		{name: "valid percentage", score: &valid, scale: "percentage", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := qualificationScoreValidationField(tc.score, tc.scale); got != tc.want {
				t.Fatalf("validation field=%q want=%q", got, tc.want)
			}
		})
	}
}


func TestStaffRecordsServiceReturnsAllQualificationFieldValidationErrors(t *testing.T) {
	repo := &fakeStaffRecordsRepository{}
	svc := NewStaffRecordsService(repo)
	_, err := svc.CreateQualification(context.Background(), 1, 2, 3, repository.StaffQualification{
		QualificationType: "invalid",
		QualificationName: "",
		Institution: "",
	})
	var validation *StaffRecordFieldValidationError
	if !errors.As(err, &validation) { t.Fatalf("error type=%T want StaffRecordFieldValidationError", err) }
	for _, field := range []string{"qualification_type", "qualification_name", "institution"} {
		if validation.FieldErrors[field] == "" { t.Errorf("missing %q error: %+v", field, validation.FieldErrors) }
	}
	if len(validation.FieldErrors) != 3 { t.Fatalf("got %d errors, want 3: %+v", len(validation.FieldErrors), validation.FieldErrors) }
	if repo.calls != 0 { t.Fatalf("invalid qualification reached repository %d times", repo.calls) }
}


func TestStaffRecordsServiceReturnsAllCertificationFieldValidationErrors(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(map[bool]string{false:"create", true:"update"}[update], func(t *testing.T) {
			repo := &fakeStaffRecordsRepository{}
			svc := NewStaffRecordsService(repo)
			item := repository.StaffCertification{
				CertificateName: "", IssuingOrganization: "",
				IssueDate: "2026-02-30", ExpiryDate: "2025-01-01",
			}
			var err error
			if update {
				_, err = svc.UpdateCertification(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateCertification(context.Background(), 1, 2, 3, item)
			}
			var validation *StaffRecordFieldValidationError
			if !errors.As(err, &validation) { t.Fatalf("error type=%T want StaffRecordFieldValidationError", err) }
			for _, field := range []string{"certificate_name", "issuing_organization", "issue_date"} {
				if validation.FieldErrors[field] == "" { t.Errorf("missing %q error: %+v", field, validation.FieldErrors) }
			}
			if len(validation.FieldErrors) != 3 { t.Fatalf("got %d errors, want 3: %+v", len(validation.FieldErrors), validation.FieldErrors) }
			if repo.calls != 0 { t.Fatalf("invalid certification reached repository %d times", repo.calls) }
		})
	}
}

func TestStaffRecordsServiceReturnsAllExperienceFieldValidationErrors(t *testing.T) {
	for _, update := range []bool{false, true} {
		t.Run(map[bool]string{false:"create", true:"update"}[update], func(t *testing.T) {
			repo := &fakeStaffRecordsRepository{}
			svc := NewStaffRecordsService(repo)
			item := repository.StaffExperience{
				EmployerName: "", Designation: "", EmploymentType: "unknown",
				StartDate: "2026-02-30", EndDate: "2025-02-30",
			}
			var err error
			if update {
				_, err = svc.UpdateExperience(context.Background(), 1, 2, 3, 4, item)
			} else {
				_, err = svc.CreateExperience(context.Background(), 1, 2, 3, item)
			}
			var validation *StaffRecordFieldValidationError
			if !errors.As(err, &validation) { t.Fatalf("error type=%T want StaffRecordFieldValidationError", err) }
			for _, field := range []string{"employer_name", "designation", "employment_type", "start_date", "end_date"} {
				if validation.FieldErrors[field] == "" { t.Errorf("missing %q error: %+v", field, validation.FieldErrors) }
			}
			if len(validation.FieldErrors) != 5 { t.Fatalf("got %d errors, want 5: %+v", len(validation.FieldErrors), validation.FieldErrors) }
			if repo.calls != 0 { t.Fatalf("invalid experience reached repository %d times", repo.calls) }
		})
	}
}


func TestStaffRecordCredentialReferenceLimitsCountUnicodeCharacters(t *testing.T) {
	valid := strings.Repeat("界", 120)
	invalid := strings.Repeat("界", 121)
	for _, update := range []bool{false, true} {
		for _, value := range []struct{name, reference string; wantErr bool}{
			{name: "120 characters accepted", reference: valid},
			{name: "121 characters rejected", reference: invalid, wantErr: true},
		} {
			t.Run(map[bool]string{false:"create/", true:"update/"}[update]+value.name, func(t *testing.T) {
				repo := &fakeStaffRecordsRepository{}
				svc := NewStaffRecordsService(repo)
				item := repository.StaffQualification{QualificationType:"undergraduate", QualificationName:"B.Sc.", Institution:"University", CredentialReference:value.reference}
				var err error
				if update { _, err = svc.UpdateQualification(context.Background(), 1, 2, 3, 4, item) } else { _, err = svc.CreateQualification(context.Background(), 1, 2, 3, item) }
				if value.wantErr {
					var validation *StaffRecordFieldValidationError
					if !errors.As(err, &validation) || validation.FieldErrors["credential_reference"] == "" { t.Fatalf("error=%v want credential_reference validation", err) }
					if repo.calls != 0 { t.Fatalf("invalid qualification reached repository %d times", repo.calls) }
				} else if err != nil || repo.calls != 1 { t.Fatalf("valid Unicode credential reference: error=%v calls=%d", err, repo.calls) }
			})
		}
		for _, value := range []struct{name, id string; wantErr bool}{
			{name:"120 characters accepted", id:valid},
			{name:"121 characters rejected", id:invalid, wantErr:true},
		} {
			t.Run(map[bool]string{false:"certification-create/", true:"certification-update/"}[update]+value.name, func(t *testing.T) {
				repo := &fakeStaffRecordsRepository{}
				svc := NewStaffRecordsService(repo)
				item := repository.StaffCertification{CertificateName:"Training", IssuingOrganization:"Board", CredentialID:value.id}
				var err error
				if update { _, err = svc.UpdateCertification(context.Background(), 1, 2, 3, 4, item) } else { _, err = svc.CreateCertification(context.Background(), 1, 2, 3, item) }
				if value.wantErr {
					var validation *StaffRecordFieldValidationError
					if !errors.As(err, &validation) || validation.FieldErrors["credential_id"] == "" { t.Fatalf("error=%v want credential_id validation", err) }
					if repo.calls != 0 { t.Fatalf("invalid certification reached repository %d times", repo.calls) }
				} else if err != nil || repo.calls != 1 { t.Fatalf("valid Unicode credential ID: error=%v calls=%d", err, repo.calls) }
			})
		}
	}
}
