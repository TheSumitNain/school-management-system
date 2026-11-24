package services

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

var ErrInvalidStaffRecord = errors.New("invalid staff record")

type StaffRecordFieldValidationError struct {
	FieldErrors map[string]string
}

func (e *StaffRecordFieldValidationError) Error() string { return ErrInvalidStaffRecord.Error() }
func (e *StaffRecordFieldValidationError) Unwrap() error { return ErrInvalidStaffRecord }

func staffRecordFieldError(field, message string) error {
	return &StaffRecordFieldValidationError{FieldErrors: map[string]string{field: message}}
}

type staffRecordsRepository interface {
	ListStaffQualificationReport(context.Context,int64,int64,repository.StaffQualificationReportFilter)(*repository.StaffQualificationReportPage,error)
	ListStaffExperienceReport(context.Context,int64,int64,repository.StaffExperienceReportFilter)(*repository.StaffExperienceReportPage,error)
	ListStaffQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error)
	CreateStaffQualification(context.Context,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	ListStaffCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error)
	CreateStaffCertification(context.Context,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	ListStaffExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error)
	CreateStaffExperience(context.Context,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	UpdateStaffQualification(context.Context,int64,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	UpdateStaffCertification(context.Context,int64,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	UpdateStaffExperience(context.Context,int64,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	ListStaffAuditEvents(context.Context,int64,int64,int64)([]repository.StaffAuditEvent,error)
}

type StaffRecordsService struct { repository staffRecordsRepository }
func NewStaffRecordsService(repo staffRecordsRepository)*StaffRecordsService{return &StaffRecordsService{repository:repo}}

var staffRecordTextPattern = regexp.MustCompile(`\S`)
func parseStaffRecordDate(value string) (time.Time, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01-02") != value {
		return time.Time{}, false
	}
	return parsed, true
}

func qualificationScoreValidationField(score *float64, scale string) string {
	scales := map[string]bool{"percentage": true, "cgpa": true, "gpa": true, "marks": true, "grade": true}
	if score == nil {
		if scale != "" { return "score_scale" }
		return ""
	}
	if !scales[scale] { return "score_scale" }
	if math.IsNaN(*score) || math.IsInf(*score, 0) || *score < 0 || *score > 99999.999 {
		return "score"
	}
	if scale == "percentage" && *score > 100 {
		return "score"
	}
	return ""
}

func validRecordText(value string,max int)bool{return len([]rune(strings.TrimSpace(value)))>0&&len([]rune(strings.TrimSpace(value)))<=max&&staffRecordTextPattern.MatchString(value)}


func staffRecordValidationError(fieldErrors map[string]string) error {
	if len(fieldErrors) == 0 { return nil }
	return &StaffRecordFieldValidationError{FieldErrors: fieldErrors}
}
func validateQualificationFields(item repository.StaffQualification) map[string]string {
	errs := map[string]string{}; add := func(f,m string){ errs[f]=m }
	types:=map[string]bool{"school":true,"diploma":true,"undergraduate":true,"postgraduate":true,"doctorate":true,"other":true}
	if !types[item.QualificationType]{add("qualification_type","Select a valid qualification type.")}
	if !validRecordText(item.QualificationName,160){add("qualification_name","Enter a qualification name of 160 characters or fewer.")}
	if !validRecordText(item.Institution,200){add("institution","Enter an institution of 200 characters or fewer.")}
	if len([]rune(item.Specialization))>160{add("specialization","Specialization must be 160 characters or fewer.")}
	if len([]rune(item.BoardUniversity))>200{add("board_university","Board / University must be 200 characters or fewer.")}
	if len(item.CredentialReference)>120{add("credential_reference","Credential reference must be 120 characters or fewer.")}
	if item.CompletionYear!=nil&&(*item.CompletionYear<1900||*item.CompletionYear>2200){add("completion_year","Completion year must be between 1900 and 2200.")}
	if field:=qualificationScoreValidationField(item.Score,item.ScoreScale);field!=""{if field=="score_scale"{add(field,"Choose a valid score scale when a score is entered.")}else{add(field,"Enter a valid score.")}}
	return errs
}
func validateCertificationFields(item repository.StaffCertification) map[string]string {
	errs:=map[string]string{};add:=func(f,m string){errs[f]=m}
	if !validRecordText(item.CertificateName,180){add("certificate_name","Enter a certificate name of 180 characters or fewer.")}
	if !validRecordText(item.IssuingOrganization,200){add("issuing_organization","Enter an issuing organization of 200 characters or fewer.")}
	if len(item.CredentialID)>120{add("credential_id","Credential ID must be 120 characters or fewer.")}
	var issue,expiry time.Time
	hasIssue,hasExpiry:=item.IssueDate!="",item.ExpiryDate!=""
	issueOK,expiryOK:=false,false
	if hasIssue{issue,issueOK=parseStaffRecordDate(item.IssueDate);if !issueOK{add("issue_date","Enter a valid issue date.")}}
	if hasExpiry{expiry,expiryOK=parseStaffRecordDate(item.ExpiryDate);if !expiryOK{add("expiry_date","Enter a valid expiry date.")}}
	if issueOK&&expiryOK&&expiry.Before(issue){add("expiry_date","Expiry date cannot be before the issue date.")}
	return errs
}
func validateExperienceFields(item repository.StaffExperience) map[string]string {
	errs:=map[string]string{};add:=func(f,m string){errs[f]=m}
	if !validRecordText(item.EmployerName,200){add("employer_name","Enter an employer name of 200 characters or fewer.")}
	if !validRecordText(item.Designation,120){add("designation","Enter a designation of 120 characters or fewer.")}
	if len([]rune(item.Description))>2000{add("description","Description must be 2,000 characters or fewer.")}
	start,startOK:=parseStaffRecordDate(item.StartDate);if !startOK{add("start_date","Enter a valid start date.")}
	if item.IsCurrent{if item.EndDate!=""{add("end_date","A current role cannot have an end date.")}}else if item.EndDate!=""{end,endOK:=parseStaffRecordDate(item.EndDate);if !endOK||(startOK&&end.Before(start)){add("end_date","End date must be valid and on or after the start date.")}}
	types:=map[string]bool{"permanent":true,"contract":true,"part_time":true,"temporary":true,"internship":true,"other":true}
	if !types[item.EmploymentType]{add("employment_type","Select a valid employment type.")}
	return errs
}

func(s *StaffRecordsService) ListQualifications(ctx context.Context,actor,school,staff int64)([]repository.StaffQualification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffQualifications(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateQualification(ctx context.Context,actor,school,staff int64,item repository.StaffQualification)(*repository.StaffQualification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.QualificationName=strings.TrimSpace(item.QualificationName);item.Specialization=strings.TrimSpace(item.Specialization);item.Institution=strings.TrimSpace(item.Institution);item.BoardUniversity=strings.TrimSpace(item.BoardUniversity);item.CredentialReference=strings.TrimSpace(item.CredentialReference)
	if err := staffRecordValidationError(validateQualificationFields(item)); err != nil { return nil, err }
	return s.repository.CreateStaffQualification(ctx,actor,school,staff,item)
}
func(s *StaffRecordsService) ListCertifications(ctx context.Context,actor,school,staff int64)([]repository.StaffCertification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffCertifications(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateCertification(ctx context.Context,actor,school,staff int64,item repository.StaffCertification)(*repository.StaffCertification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.CertificateName=strings.TrimSpace(item.CertificateName);item.IssuingOrganization=strings.TrimSpace(item.IssuingOrganization);item.CredentialID=strings.TrimSpace(item.CredentialID)
	if err := staffRecordValidationError(validateCertificationFields(item)); err != nil { return nil, err }
	return s.repository.CreateStaffCertification(ctx,actor,school,staff,item)
}
func(s *StaffRecordsService) ListExperience(ctx context.Context,actor,school,staff int64)([]repository.StaffExperience,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffExperience(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateExperience(ctx context.Context,actor,school,staff int64,item repository.StaffExperience)(*repository.StaffExperience,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.EmployerName=strings.TrimSpace(item.EmployerName);item.Designation=strings.TrimSpace(item.Designation);item.Description=strings.TrimSpace(item.Description)
	if err := staffRecordValidationError(validateExperienceFields(item)); err != nil { return nil, err }
	return s.repository.CreateStaffExperience(ctx,actor,school,staff,item)
}


func (s *StaffRecordsService) UpdateQualification(ctx context.Context, actor, school, staff, record int64, item repository.StaffQualification) (*repository.StaffQualification,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.QualificationName=strings.TrimSpace(item.QualificationName);item.Specialization=strings.TrimSpace(item.Specialization);item.Institution=strings.TrimSpace(item.Institution);item.BoardUniversity=strings.TrimSpace(item.BoardUniversity);item.CredentialReference=strings.TrimSpace(item.CredentialReference)
	if err := staffRecordValidationError(validateQualificationFields(item)); err != nil { return nil, err }
	return s.repository.UpdateStaffQualification(ctx,actor,school,staff,record,item)
}

func (s *StaffRecordsService) UpdateCertification(ctx context.Context, actor, school, staff, record int64, item repository.StaffCertification) (*repository.StaffCertification,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.CertificateName=strings.TrimSpace(item.CertificateName);item.IssuingOrganization=strings.TrimSpace(item.IssuingOrganization);item.CredentialID=strings.TrimSpace(item.CredentialID)
	if err := staffRecordValidationError(validateCertificationFields(item)); err != nil { return nil, err }
	return s.repository.UpdateStaffCertification(ctx,actor,school,staff,record,item)
}

func (s *StaffRecordsService) UpdateExperience(ctx context.Context, actor, school, staff, record int64, item repository.StaffExperience) (*repository.StaffExperience,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.EmployerName=strings.TrimSpace(item.EmployerName);item.Designation=strings.TrimSpace(item.Designation);item.Description=strings.TrimSpace(item.Description)
	if err := staffRecordValidationError(validateExperienceFields(item)); err != nil { return nil, err }
	return s.repository.UpdateStaffExperience(ctx,actor,school,staff,record,item)
}

func(s *StaffRecordsService) ListAuditEvents(ctx context.Context,actor,school,staff int64)([]repository.StaffAuditEvent,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffAuditEvents(ctx,actor,school,staff)
}

func (s *StaffRecordsService) ListQualificationsReport(ctx context.Context, actor, school int64, filter repository.StaffQualificationReportFilter) (*repository.StaffQualificationReportPage, error) {
	if s == nil || s.repository == nil || actor <= 0 || school <= 0 {
		return nil, ErrInvalidStaffRecord
	}
	if filter.RecordType == "" { filter.RecordType = "all" }
	if filter.RecordType != "all" && filter.RecordType != "qualification" && filter.RecordType != "certification" {
		return nil, ErrInvalidStaffRecord
	}
	if filter.VerificationStatus != "" && filter.VerificationStatus != "pending" && filter.VerificationStatus != "verified" && filter.VerificationStatus != "rejected" {
		return nil, ErrInvalidStaffRecord
	}
	if filter.Limit == 0 { filter.Limit = 50 }
	if filter.Limit < 1 || filter.Limit > 200 || filter.Offset < 0 || filter.Offset > 1000000 {
		return nil, ErrInvalidStaffRecord
	}
	return s.repository.ListStaffQualificationReport(ctx, actor, school, filter)
}

func (s *StaffRecordsService) ListExperienceReport(ctx context.Context, actor, school int64, filter repository.StaffExperienceReportFilter) (*repository.StaffExperienceReportPage,error) {
    if s==nil || s.repository==nil || actor<=0 || school<=0 { return nil,ErrInvalidStaffRecord }
    allowed:=map[string]bool{"":true,"permanent":true,"contract":true,"part_time":true,"temporary":true,"internship":true,"other":true}
    if !allowed[filter.EmploymentType] { return nil,ErrInvalidStaffRecord }
    if filter.Limit==0 { filter.Limit=50 }
    if filter.Limit<1 || filter.Limit>200 || filter.Offset<0 || filter.Offset>1000000 { return nil,ErrInvalidStaffRecord }
    return s.repository.ListStaffExperienceReport(ctx,actor,school,filter)
}
