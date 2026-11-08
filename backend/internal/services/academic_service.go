package services

import (
	"context"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

type academicRepository interface {
	ListAcademicYears(context.Context,int64,int64)([]repository.AcademicYear,error)
	SaveAcademicYear(context.Context,int64,int64,int64,string,time.Time,time.Time,string,bool)(*repository.AcademicYear,error)
	ListClasses(context.Context,int64,int64)([]repository.AcademicClass,error)
	SaveClass(context.Context,int64,int64,int64,string,int,bool)(*repository.AcademicClass,error)
	ListSections(context.Context,int64,int64,int64)([]repository.AcademicSection,error)
	SaveSection(context.Context,int64,int64,int64,int64,int64,string,int,bool)(*repository.AcademicSection,error)
	ListSubjects(context.Context,int64,int64)([]repository.AcademicSubject,error)
	SaveSubject(context.Context,int64,int64,int64,string,*string,bool)(*repository.AcademicSubject,error)
	ListEligibleTeachers(context.Context,int64,int64)([]repository.EligibleTeacher,error)
	ListTeacherSubjectAssignments(context.Context,int64,int64,int64,int64)([]repository.TeacherSubjectAssignment,error)
	CreateTeacherSubjectAssignment(context.Context,int64,int64,int64,int64,int64,int64)(*repository.TeacherSubjectAssignment,error)
	RevokeTeacherSubjectAssignment(context.Context,int64,int64,int64)error
	ListClassInchargeAssignments(context.Context,int64,int64,int64)([]repository.ClassInchargeAssignment,error)
	CreateClassInchargeAssignment(context.Context,int64,int64,int64,int64,int64)(*repository.ClassInchargeAssignment,error)
	RevokeClassInchargeAssignment(context.Context,int64,int64,int64)error
	ListStudentEnrollments(context.Context,int64,int64,int64,int64)([]repository.StudentEnrollment,error)
	CreateStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,time.Time)(*repository.StudentEnrollment,error)
	PromoteStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,time.Time)(*repository.StudentEnrollment,error)

	ListStudents(context.Context,int64,int64,string,string,int,int)([]repository.StudentProfile,int64,error)
	GetStudent(context.Context,int64,int64,int64)(*repository.StudentProfile,error)
	CreateStudent(context.Context,int64,int64,string,string)(*repository.StudentProfile,error)
	UpdateStudent(context.Context,int64,int64,int64,string)(*repository.StudentProfile,error)
	ArchiveStudent(context.Context,int64,int64,int64)(*repository.StudentProfile,error)
}

type AcademicService struct { repository academicRepository }

func NewAcademicService(repo academicRepository) *AcademicService { return &AcademicService{repository:repo} }

func (s *AcademicService) ListAcademicYears(ctx context.Context,actorID,schoolID int64)([]repository.AcademicYear,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListAcademicYears(ctx,actorID,schoolID)
}
func (s *AcademicService) SaveAcademicYear(ctx context.Context,actorID,schoolID,id int64,name string,startsOn,endsOn time.Time,status string,current bool)(*repository.AcademicYear,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	if strings.TrimSpace(name)==""||!endsOn.After(startsOn){return nil,repository.ErrInvalidAcademicInput}
	return s.repository.SaveAcademicYear(ctx,actorID,schoolID,id,name,startsOn,endsOn,status,current)
}
func (s *AcademicService) ListClasses(ctx context.Context,actorID,schoolID int64)([]repository.AcademicClass,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListClasses(ctx,actorID,schoolID)
}
func (s *AcademicService) SaveClass(ctx context.Context,actorID,schoolID,id int64,name string,order int,active bool)(*repository.AcademicClass,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.SaveClass(ctx,actorID,schoolID,id,name,order,active)
}
func (s *AcademicService) ListSections(ctx context.Context,actorID,schoolID,yearID int64)([]repository.AcademicSection,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListSections(ctx,actorID,schoolID,yearID)
}
func (s *AcademicService) SaveSection(ctx context.Context,actorID,schoolID,id,yearID,classID int64,name string,order int,active bool)(*repository.AcademicSection,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.SaveSection(ctx,actorID,schoolID,id,yearID,classID,name,order,active)
}
func (s *AcademicService) ListSubjects(ctx context.Context,actorID,schoolID int64)([]repository.AcademicSubject,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListSubjects(ctx,actorID,schoolID)
}
func (s *AcademicService) SaveSubject(ctx context.Context,actorID,schoolID,id int64,name string,code *string,active bool)(*repository.AcademicSubject,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.SaveSubject(ctx,actorID,schoolID,id,name,code,active)
}


func (s *AcademicService) ListEligibleTeachers(ctx context.Context, actorID, schoolID int64) ([]repository.EligibleTeacher, error) {
	if s == nil || s.repository == nil {
		return nil, repository.ErrAcademicAccessDenied
	}
	return s.repository.ListEligibleTeachers(ctx, actorID, schoolID)
}

func (s *AcademicService) ListTeacherSubjectAssignments(ctx context.Context,actorID,schoolID,yearID,sectionID int64)([]repository.TeacherSubjectAssignment,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListTeacherSubjectAssignments(ctx,actorID,schoolID,yearID,sectionID)
}
func (s *AcademicService) CreateTeacherSubjectAssignment(ctx context.Context,actorID,schoolID,yearID,sectionID,subjectID,teacherID int64)(*repository.TeacherSubjectAssignment,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.CreateTeacherSubjectAssignment(ctx,actorID,schoolID,yearID,sectionID,subjectID,teacherID)
}
func (s *AcademicService) RevokeTeacherSubjectAssignment(ctx context.Context,actorID,schoolID,assignmentID int64)error{
	if s==nil||s.repository==nil{return repository.ErrAcademicAccessDenied}
	return s.repository.RevokeTeacherSubjectAssignment(ctx,actorID,schoolID,assignmentID)
}
func (s *AcademicService) ListClassInchargeAssignments(ctx context.Context,actorID,schoolID,yearID int64)([]repository.ClassInchargeAssignment,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListClassInchargeAssignments(ctx,actorID,schoolID,yearID)
}
func (s *AcademicService) CreateClassInchargeAssignment(ctx context.Context,actorID,schoolID,yearID,sectionID,teacherID int64)(*repository.ClassInchargeAssignment,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.CreateClassInchargeAssignment(ctx,actorID,schoolID,yearID,sectionID,teacherID)
}
func (s *AcademicService) RevokeClassInchargeAssignment(ctx context.Context,actorID,schoolID,assignmentID int64)error{
	if s==nil||s.repository==nil{return repository.ErrAcademicAccessDenied}
	return s.repository.RevokeClassInchargeAssignment(ctx,actorID,schoolID,assignmentID)
}

func (s *AcademicService) ListStudentEnrollments(ctx context.Context, actorID, schoolID, yearID, studentID int64) ([]repository.StudentEnrollment, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudentEnrollments(ctx, actorID, schoolID, yearID, studentID)
}
func (s *AcademicService) CreateStudentEnrollment(ctx context.Context, actorID, schoolID, studentID, yearID, sectionID int64, rollNumber *string, enrolledAt time.Time) (*repository.StudentEnrollment, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudentEnrollment(ctx, actorID, schoolID, studentID, yearID, sectionID, rollNumber, enrolledAt)
}
func (s *AcademicService) PromoteStudentEnrollment(ctx context.Context, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID int64, rollNumber *string, enrolledAt time.Time) (*repository.StudentEnrollment, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.PromoteStudentEnrollment(ctx, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID, rollNumber, enrolledAt)
}

func (s *AcademicService) ListStudents(ctx context.Context, actorID, schoolID int64, search, status string, limit, offset int) ([]repository.StudentProfile, int64, error) {
	if s == nil || s.repository == nil { return nil, 0, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudents(ctx, actorID, schoolID, search, status, limit, offset)
}
func (s *AcademicService) GetStudent(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudent(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) CreateStudent(ctx context.Context, actorID, schoolID int64, admissionNumber, fullName string) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudent(ctx, actorID, schoolID, admissionNumber, fullName)
}
func (s *AcademicService) UpdateStudent(ctx context.Context, actorID, schoolID, studentID int64, fullName string) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.UpdateStudent(ctx, actorID, schoolID, studentID, fullName)
}
func (s *AcademicService) ArchiveStudent(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ArchiveStudent(ctx, actorID, schoolID, studentID)
}
