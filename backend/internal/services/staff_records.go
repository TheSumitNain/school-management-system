package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

var ErrInvalidStaffRecord = errors.New("invalid staff record")

type staffRecordsRepository interface {
	ListStaffQualifications(context.Context,int64,int64,int64)([]repository.StaffQualification,error)
	CreateStaffQualification(context.Context,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	ListStaffCertifications(context.Context,int64,int64,int64)([]repository.StaffCertification,error)
	CreateStaffCertification(context.Context,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	ListStaffExperience(context.Context,int64,int64,int64)([]repository.StaffExperience,error)
	CreateStaffExperience(context.Context,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
	UpdateStaffQualification(context.Context,int64,int64,int64,int64,repository.StaffQualification)(*repository.StaffQualification,error)
	UpdateStaffCertification(context.Context,int64,int64,int64,int64,repository.StaffCertification)(*repository.StaffCertification,error)
	UpdateStaffExperience(context.Context,int64,int64,int64,int64,repository.StaffExperience)(*repository.StaffExperience,error)
}

type StaffRecordsService struct { repository staffRecordsRepository }
func NewStaffRecordsService(repo staffRecordsRepository)*StaffRecordsService{return &StaffRecordsService{repository:repo}}

var staffRecordTextPattern = regexp.MustCompile(`\S`)
func validRecordText(value string,max int)bool{return len([]rune(strings.TrimSpace(value)))>0&&len([]rune(strings.TrimSpace(value)))<=max&&staffRecordTextPattern.MatchString(value)}

func(s *StaffRecordsService) ListQualifications(ctx context.Context,actor,school,staff int64)([]repository.StaffQualification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffQualifications(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateQualification(ctx context.Context,actor,school,staff int64,item repository.StaffQualification)(*repository.StaffQualification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.QualificationName=strings.TrimSpace(item.QualificationName);item.Specialization=strings.TrimSpace(item.Specialization);item.Institution=strings.TrimSpace(item.Institution);item.BoardUniversity=strings.TrimSpace(item.BoardUniversity);item.CredentialReference=strings.TrimSpace(item.CredentialReference)
	types:=map[string]bool{"school":true,"diploma":true,"undergraduate":true,"postgraduate":true,"doctorate":true,"other":true}
	scales:=map[string]bool{"percentage":true,"cgpa":true,"gpa":true,"marks":true,"grade":true}
	if !types[item.QualificationType]||!validRecordText(item.QualificationName,160)||!validRecordText(item.Institution,200)||len([]rune(item.Specialization))>160||len([]rune(item.BoardUniversity))>200||len(item.CredentialReference)>120{return nil,ErrInvalidStaffRecord}
	if item.CompletionYear!=nil&&(*item.CompletionYear<1900||*item.CompletionYear>2200){return nil,ErrInvalidStaffRecord}
	if (item.Score==nil)!=(item.ScoreScale=="") || (item.Score!=nil&&(*item.Score<0||!scales[item.ScoreScale])){return nil,ErrInvalidStaffRecord}
	return s.repository.CreateStaffQualification(ctx,actor,school,staff,item)
}
func(s *StaffRecordsService) ListCertifications(ctx context.Context,actor,school,staff int64)([]repository.StaffCertification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffCertifications(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateCertification(ctx context.Context,actor,school,staff int64,item repository.StaffCertification)(*repository.StaffCertification,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.CertificateName=strings.TrimSpace(item.CertificateName);item.IssuingOrganization=strings.TrimSpace(item.IssuingOrganization);item.CredentialID=strings.TrimSpace(item.CredentialID)
	if !validRecordText(item.CertificateName,180)||!validRecordText(item.IssuingOrganization,200)||len(item.CredentialID)>120{return nil,ErrInvalidStaffRecord}
	var issue,expiry time.Time;var err error
	if item.IssueDate!=""{issue,err=time.Parse("2006-01-02",item.IssueDate);if err!=nil{return nil,ErrInvalidStaffRecord}}
	if item.ExpiryDate!=""{expiry,err=time.Parse("2006-01-02",item.ExpiryDate);if err!=nil{return nil,ErrInvalidStaffRecord}}
	if !issue.IsZero()&&!expiry.IsZero()&&expiry.Before(issue){return nil,ErrInvalidStaffRecord}
	return s.repository.CreateStaffCertification(ctx,actor,school,staff,item)
}
func(s *StaffRecordsService) ListExperience(ctx context.Context,actor,school,staff int64)([]repository.StaffExperience,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	return s.repository.ListStaffExperience(ctx,actor,school,staff)
}
func(s *StaffRecordsService) CreateExperience(ctx context.Context,actor,school,staff int64,item repository.StaffExperience)(*repository.StaffExperience,error){
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0{return nil,ErrInvalidStaffRecord}
	item.EmployerName=strings.TrimSpace(item.EmployerName);item.Designation=strings.TrimSpace(item.Designation);item.Description=strings.TrimSpace(item.Description)
	if !validRecordText(item.EmployerName,200)||!validRecordText(item.Designation,120)||len([]rune(item.Description))>2000{return nil,ErrInvalidStaffRecord}
	start,err:=time.Parse("2006-01-02",item.StartDate);if err!=nil{return nil,ErrInvalidStaffRecord}
	if item.IsCurrent {if item.EndDate!=""{return nil,ErrInvalidStaffRecord}} else if item.EndDate!="" {end,e:=time.Parse("2006-01-02",item.EndDate);if e!=nil||end.Before(start){return nil,ErrInvalidStaffRecord}}
	types:=map[string]bool{"permanent":true,"contract":true,"part_time":true,"temporary":true,"internship":true,"other":true}
	if !types[item.EmploymentType]{return nil,ErrInvalidStaffRecord}
	return s.repository.CreateStaffExperience(ctx,actor,school,staff,item)
}


func (s *StaffRecordsService) UpdateQualification(ctx context.Context, actor, school, staff, record int64, item repository.StaffQualification) (*repository.StaffQualification,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.QualificationName=strings.TrimSpace(item.QualificationName);item.Specialization=strings.TrimSpace(item.Specialization);item.Institution=strings.TrimSpace(item.Institution);item.BoardUniversity=strings.TrimSpace(item.BoardUniversity);item.CredentialReference=strings.TrimSpace(item.CredentialReference)
	types:=map[string]bool{"school":true,"diploma":true,"undergraduate":true,"postgraduate":true,"doctorate":true,"other":true}
	scales:=map[string]bool{"percentage":true,"cgpa":true,"gpa":true,"marks":true,"grade":true}
	if !types[item.QualificationType]||!validRecordText(item.QualificationName,160)||!validRecordText(item.Institution,200)||len([]rune(item.Specialization))>160||len([]rune(item.BoardUniversity))>200||len(item.CredentialReference)>120{return nil,ErrInvalidStaffRecord}
	if item.CompletionYear!=nil&&(*item.CompletionYear<1900||*item.CompletionYear>2200){return nil,ErrInvalidStaffRecord}
	if (item.Score==nil)!=(item.ScoreScale=="")||(item.Score!=nil&&(*item.Score<0||!scales[item.ScoreScale])){return nil,ErrInvalidStaffRecord}
	return s.repository.UpdateStaffQualification(ctx,actor,school,staff,record,item)
}

func (s *StaffRecordsService) UpdateCertification(ctx context.Context, actor, school, staff, record int64, item repository.StaffCertification) (*repository.StaffCertification,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.CertificateName=strings.TrimSpace(item.CertificateName);item.IssuingOrganization=strings.TrimSpace(item.IssuingOrganization);item.CredentialID=strings.TrimSpace(item.CredentialID)
	if !validRecordText(item.CertificateName,180)||!validRecordText(item.IssuingOrganization,200)||len(item.CredentialID)>120{return nil,ErrInvalidStaffRecord}
	var issue,expiry time.Time;var err error
	if item.IssueDate!=""{issue,err=time.Parse("2006-01-02",item.IssueDate);if err!=nil{return nil,ErrInvalidStaffRecord}}
	if item.ExpiryDate!=""{expiry,err=time.Parse("2006-01-02",item.ExpiryDate);if err!=nil{return nil,ErrInvalidStaffRecord}}
	if !issue.IsZero()&&!expiry.IsZero()&&expiry.Before(issue){return nil,ErrInvalidStaffRecord}
	return s.repository.UpdateStaffCertification(ctx,actor,school,staff,record,item)
}

func (s *StaffRecordsService) UpdateExperience(ctx context.Context, actor, school, staff, record int64, item repository.StaffExperience) (*repository.StaffExperience,error) {
	if s==nil||s.repository==nil||actor<=0||school<=0||staff<=0||record<=0{return nil,ErrInvalidStaffRecord}
	item.EmployerName=strings.TrimSpace(item.EmployerName);item.Designation=strings.TrimSpace(item.Designation);item.Description=strings.TrimSpace(item.Description)
	if !validRecordText(item.EmployerName,200)||!validRecordText(item.Designation,120)||len([]rune(item.Description))>2000{return nil,ErrInvalidStaffRecord}
	start,err:=time.Parse("2006-01-02",item.StartDate);if err!=nil{return nil,ErrInvalidStaffRecord}
	if item.IsCurrent {if item.EndDate!=""{return nil,ErrInvalidStaffRecord}} else if item.EndDate!="" {end,e:=time.Parse("2006-01-02",item.EndDate);if e!=nil||end.Before(start){return nil,ErrInvalidStaffRecord}}
	types:=map[string]bool{"permanent":true,"contract":true,"part_time":true,"temporary":true,"internship":true,"other":true}
	if !types[item.EmploymentType]{return nil,ErrInvalidStaffRecord}
	return s.repository.UpdateStaffExperience(ctx,actor,school,staff,record,item)
}
