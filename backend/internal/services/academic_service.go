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
	ListStudentEnrollmentDirectory(context.Context,int64,int64,int64,int,int)([]repository.StudentEnrollment,int64,error)
	ListSectionClassCurricula(context.Context,int64,int64,int64,int64,int64)([]repository.ClassCurriculum,error)
	SaveSectionClassCurriculum(context.Context,int64,int64,int64,int64,int64,[]repository.CurriculumOptionalGroupInput,[]repository.CurriculumSubjectInput)(*repository.ClassCurriculum,error)
	SaveSectionClassCurriculumWithConfirmations(context.Context,int64,int64,int64,int64,int64,[]repository.CurriculumOptionalGroupInput,[]repository.CurriculumSubjectInput,[]repository.CurriculumRemovalConfirmation)(*repository.ClassCurriculum,error)
	PreviewSectionCurriculumRemoval(context.Context,int64,int64,int64,int64,int64,[]int64)([]repository.CurriculumRemovalImpact,error)
	GetStudentEnrollmentCurriculum(context.Context,int64,int64,int64)(*repository.StudentEnrollmentCurriculum,error)
	SaveStudentEnrollmentSubjects(context.Context,int64,int64,int64,int64,[]int64)(*repository.StudentEnrollmentCurriculum,error)
	CreateStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,string,time.Time,time.Time)(*repository.StudentEnrollment,error)
	PromoteStudentEnrollment(context.Context,int64,int64,int64,int64,int64,*string,string,time.Time,time.Time)(*repository.StudentEnrollment,error)
	CreateStudentAdmission(context.Context,int64,int64,string,string,int64,int64,*string,time.Time)(*repository.StudentAdmissionResult,error)
	ListStudentLocations(context.Context,int64,int64,*int64,string,string)([]repository.StudentLocation,error)
	CreateStudentLocation(context.Context,int64,int64,*int64,string,string,string,*string)(*repository.StudentLocation,error)
	DeactivateStudentLocation(context.Context,int64,int64,int64)(*repository.StudentLocation,error)
	ListStudentAddresses(context.Context,int64,int64,int64)([]repository.StudentAddress,error)
	SaveStudentAddress(context.Context,int64,int64,int64,string,*string,*string,*string,*string,*string,*int64,bool)(*repository.StudentAddress,error)
	GetStudentContactDetails(context.Context,int64,int64,int64)(*repository.StudentContactDetails,error)
	SaveStudentContactDetails(context.Context,int64,int64,int64,*string,*string,*string)(*repository.StudentContactDetails,error)
	GetStudentBasicDetails(context.Context,int64,int64,int64)(*repository.StudentBasicDetails,error)
	SaveStudentBasicDetails(context.Context,int64,int64,int64,*time.Time,*string,*string,*string,*string,*string,*bool,*string)(*repository.StudentBasicDetails,error)
	GetStudentMedicalDetails(context.Context,int64,int64,int64)(*repository.StudentMedicalDetails,error)
	SaveStudentMedicalDetails(context.Context,int64,int64,int64,*string,*string,*string,*string,*string)(*repository.StudentMedicalDetails,error)
	GetStudentPhysicalDetails(context.Context,int64,int64,int64)(*repository.StudentPhysicalDetails,error)
	SaveStudentPhysicalDetails(context.Context,int64,int64,int64,*float64,*float64,*time.Time,*string,*string)(*repository.StudentPhysicalDetails,error)

	ListStudents(context.Context,int64,int64,int64,string,string,int,int)([]repository.StudentProfile,int64,error)
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
func (s *AcademicService) PreviewClassDeactivation(ctx context.Context, actorID, schoolID, classID int64) (*repository.ClassDeactivationImpact, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	type previewRepository interface { PreviewClassDeactivation(context.Context,int64,int64,int64)(*repository.ClassDeactivationImpact,error) }
	repo,ok:=s.repository.(previewRepository);if !ok{return nil,repository.ErrAcademicAccessDenied}
	return repo.PreviewClassDeactivation(ctx,actorID,schoolID,classID)
}
func (s *AcademicService) SaveClassWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, classID int64, name string, order int, fingerprint string) (*repository.AcademicClass,error) {
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	type confirmedRepository interface { SaveClassWithDeactivationConfirmation(context.Context,int64,int64,int64,string,int,string)(*repository.AcademicClass,error) }
	repo,ok:=s.repository.(confirmedRepository);if !ok{return nil,repository.ErrAcademicAccessDenied}
	return repo.SaveClassWithDeactivationConfirmation(ctx,actorID,schoolID,classID,name,order,fingerprint)
}

func (s *AcademicService) SaveClass(ctx context.Context,actorID,schoolID,id int64,name string,order int,active bool)(*repository.AcademicClass,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.SaveClass(ctx,actorID,schoolID,id,name,order,active)
}
func (s *AcademicService) ListSections(ctx context.Context,actorID,schoolID,yearID int64)([]repository.AcademicSection,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListSections(ctx,actorID,schoolID,yearID)
}
func (s *AcademicService) PreviewSectionDeactivation(ctx context.Context, actorID, schoolID, sectionID int64) (*repository.SectionDeactivationImpact, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	type previewRepository interface {
		PreviewSectionDeactivation(context.Context, int64, int64, int64) (*repository.SectionDeactivationImpact, error)
	}
	repo, ok := s.repository.(previewRepository)
	if !ok { return nil, repository.ErrAcademicAccessDenied }
	return repo.PreviewSectionDeactivation(ctx, actorID, schoolID, sectionID)
}

func (s *AcademicService) SaveSectionWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, id, yearID, classID int64, name string, medium *string, order int, fingerprint string) (*repository.AcademicSection, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	type confirmedRepository interface {
		SaveSectionWithDeactivationConfirmation(context.Context, int64, int64, int64, int64, int64, string, *string, int, string) (*repository.AcademicSection, error)
	}
	repo, ok := s.repository.(confirmedRepository)
	if !ok { return nil, repository.ErrAcademicAccessDenied }
	return repo.SaveSectionWithDeactivationConfirmation(ctx, actorID, schoolID, id, yearID, classID, name, medium, order, fingerprint)
}

func (s *AcademicService) SaveSection(ctx context.Context,actorID,schoolID,id,yearID,classID int64,name string,order int,active bool)(*repository.AcademicSection,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.SaveSection(ctx,actorID,schoolID,id,yearID,classID,name,order,active)
}
func (s *AcademicService) SaveSectionWithMedium(ctx context.Context,actorID,schoolID,id,yearID,classID int64,name string,medium *string,order int,active bool)(*repository.AcademicSection,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	type sectionMediumRepository interface {
		SaveSectionWithMedium(context.Context,int64,int64,int64,int64,int64,string,*string,int,bool)(*repository.AcademicSection,error)
	}
	if repo,ok:=s.repository.(sectionMediumRepository);ok{return repo.SaveSectionWithMedium(ctx,actorID,schoolID,id,yearID,classID,name,medium,order,active)}
	if medium!=nil&&strings.TrimSpace(*medium)!=""{return nil,repository.ErrInvalidAcademicInput}
	return s.repository.SaveSection(ctx,actorID,schoolID,id,yearID,classID,name,order,active)
}
func (s *AcademicService) ListSubjects(ctx context.Context,actorID,schoolID int64)([]repository.AcademicSubject,error){
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	return s.repository.ListSubjects(ctx,actorID,schoolID)
}
func (s *AcademicService) PreviewSubjectDeactivation(ctx context.Context, actorID, schoolID, subjectID int64) (*repository.SubjectDeactivationImpact,error) {
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	type previewRepository interface{PreviewSubjectDeactivation(context.Context,int64,int64,int64)(*repository.SubjectDeactivationImpact,error)}
	repo,ok:=s.repository.(previewRepository);if !ok{return nil,repository.ErrAcademicAccessDenied}
	return repo.PreviewSubjectDeactivation(ctx,actorID,schoolID,subjectID)
}
func (s *AcademicService) SaveSubjectWithDeactivationConfirmation(ctx context.Context, actorID, schoolID, subjectID int64, name string, code *string, fingerprint string) (*repository.AcademicSubject,error) {
	if s==nil||s.repository==nil{return nil,repository.ErrAcademicAccessDenied}
	type confirmedRepository interface{SaveSubjectWithDeactivationConfirmation(context.Context,int64,int64,int64,string,*string,string)(*repository.AcademicSubject,error)}
	repo,ok:=s.repository.(confirmedRepository);if !ok{return nil,repository.ErrAcademicAccessDenied}
	return repo.SaveSubjectWithDeactivationConfirmation(ctx,actorID,schoolID,subjectID,name,code,fingerprint)
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
func (s *AcademicService) ListStudentEnrollmentDirectory(ctx context.Context, actorID, schoolID, yearID int64, limit, offset int) ([]repository.StudentEnrollment, int64, error) {
	if s == nil || s.repository == nil { return nil, 0, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudentEnrollmentDirectory(ctx, actorID, schoolID, yearID, limit, offset)
}
func (s *AcademicService) CreateStudentEnrollment(ctx context.Context, actorID, schoolID, studentID, yearID, sectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*repository.StudentEnrollment, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudentEnrollment(ctx, actorID, schoolID, studentID, yearID, sectionID, rollNumber, admissionNumber, admissionDate, enrolledAt)
}
func (s *AcademicService) PromoteStudentEnrollment(ctx context.Context, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID int64, rollNumber *string, admissionNumber string, admissionDate, enrolledAt time.Time) (*repository.StudentEnrollment, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.PromoteStudentEnrollment(ctx, actorID, schoolID, sourceEnrollmentID, destinationYearID, destinationSectionID, rollNumber, admissionNumber, admissionDate, enrolledAt)
}
func (s *AcademicService) CreateStudentAdmission(ctx context.Context, actorID, schoolID int64, fullName, admissionNumber string, yearID, sectionID int64, rollNumber *string, admissionDate time.Time) (*repository.StudentAdmissionResult, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudentAdmission(ctx, actorID, schoolID, fullName, admissionNumber, yearID, sectionID, rollNumber, admissionDate)
}

func (s *AcademicService) ListStudents(ctx context.Context, actorID, schoolID, academicYearID int64, search, status string, limit, offset int) ([]repository.StudentProfile, int64, error) {
	if s == nil || s.repository == nil { return nil, 0, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudents(ctx, actorID, schoolID, academicYearID, search, status, limit, offset)
}
func (s *AcademicService) GetStudent(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudent(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) CreateStudent(ctx context.Context, actorID, schoolID int64, admissionNumber, fullName string) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudent(ctx, actorID, schoolID, admissionNumber, fullName)
}
func (s *AcademicService) ListStudentLocations(ctx context.Context, actorID, schoolID int64, parentID *int64, level, search string) ([]repository.StudentLocation, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudentLocations(ctx, actorID, schoolID, parentID, level, search)
}
func (s *AcademicService) CreateStudentLocation(ctx context.Context, actorID, schoolID int64, parentID *int64, countryCode, level, name string, localityType *string) (*repository.StudentLocation, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.CreateStudentLocation(ctx, actorID, schoolID, parentID, countryCode, level, name, localityType)
}
func (s *AcademicService) DeactivateStudentLocation(ctx context.Context, actorID, schoolID, locationID int64) (*repository.StudentLocation, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.DeactivateStudentLocation(ctx, actorID, schoolID, locationID)
}
func (s *AcademicService) ListStudentAddresses(ctx context.Context, actorID, schoolID, studentID int64) ([]repository.StudentAddress, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ListStudentAddresses(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) SaveStudentAddress(ctx context.Context, actorID, schoolID, studentID int64, addressType string, line1, line2, areaMohalla, landmark, postalCode *string, locationID *int64, sameAsCurrent bool) (*repository.StudentAddress, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentAddress(ctx, actorID, schoolID, studentID, addressType, line1, line2, areaMohalla, landmark, postalCode, locationID, sameAsCurrent)
}

func (s *AcademicService) GetStudentBasicDetails(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentBasicDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudentBasicDetails(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) SaveStudentBasicDetails(ctx context.Context, actorID, schoolID, studentID int64, dateOfBirth *time.Time, placeOfBirth, gender, nationality, religion, socialCategory *string, ewsStatus *bool, registrationNumber *string) (*repository.StudentBasicDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentBasicDetails(ctx, actorID, schoolID, studentID, dateOfBirth, placeOfBirth, gender, nationality, religion, socialCategory, ewsStatus, registrationNumber)
}

func (s *AcademicService) GetStudentContactDetails(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentContactDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudentContactDetails(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) SaveStudentContactDetails(ctx context.Context, actorID, schoolID, studentID int64, primaryPhone, alternatePhone, emailAddress *string) (*repository.StudentContactDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentContactDetails(ctx, actorID, schoolID, studentID, primaryPhone, alternatePhone, emailAddress)
}

func (s *AcademicService) GetStudentMedicalDetails(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentMedicalDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudentMedicalDetails(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) SaveStudentMedicalDetails(ctx context.Context, actorID, schoolID, studentID int64, bloodGroup, allergies, knownConditions, ongoingMedications, emergencyNotes *string) (*repository.StudentMedicalDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentMedicalDetails(ctx, actorID, schoolID, studentID, bloodGroup, allergies, knownConditions, ongoingMedications, emergencyNotes)
}
func (s *AcademicService) GetStudentPhysicalDetails(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentPhysicalDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudentPhysicalDetails(ctx, actorID, schoolID, studentID)
}
func (s *AcademicService) SaveStudentPhysicalDetails(ctx context.Context, actorID, schoolID, studentID int64, heightCM, weightKG *float64, measuredOn *time.Time, visionNotes, physicalNotes *string) (*repository.StudentPhysicalDetails, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentPhysicalDetails(ctx, actorID, schoolID, studentID, heightCM, weightKG, measuredOn, visionNotes, physicalNotes)
}

func (s *AcademicService) UpdateStudent(ctx context.Context, actorID, schoolID, studentID int64, fullName string) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.UpdateStudent(ctx, actorID, schoolID, studentID, fullName)
}
func (s *AcademicService) ArchiveStudent(ctx context.Context, actorID, schoolID, studentID int64) (*repository.StudentProfile, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ArchiveStudent(ctx, actorID, schoolID, studentID)
}

func (s *AcademicService) ListSectionClassCurricula(ctx context.Context, actorID, schoolID, yearID, classID, sectionID int64) ([]repository.ClassCurriculum, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.ListSectionClassCurricula(ctx, actorID, schoolID, yearID, classID, sectionID)
}
func (s *AcademicService) SaveSectionClassCurriculum(ctx context.Context, actorID, schoolID, yearID, classID, sectionID int64, groups []repository.CurriculumOptionalGroupInput, subjects []repository.CurriculumSubjectInput) (*repository.ClassCurriculum, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveSectionClassCurriculum(ctx, actorID, schoolID, yearID, classID, sectionID, groups, subjects)
}
func (s *AcademicService) SaveSectionClassCurriculumWithConfirmations(ctx context.Context, actorID, schoolID, yearID, classID, sectionID int64, groups []repository.CurriculumOptionalGroupInput, subjects []repository.CurriculumSubjectInput, confirmations []repository.CurriculumRemovalConfirmation) (*repository.ClassCurriculum, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveSectionClassCurriculumWithConfirmations(ctx, actorID, schoolID, yearID, classID, sectionID, groups, subjects, confirmations)
}
func (s *AcademicService) PreviewSectionCurriculumRemoval(ctx context.Context, actorID, schoolID, yearID, classID, sectionID int64, subjectIDs []int64) ([]repository.CurriculumRemovalImpact, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.PreviewSectionCurriculumRemoval(ctx, actorID, schoolID, yearID, classID, sectionID, subjectIDs)
}
func (s *AcademicService) GetStudentEnrollmentCurriculum(ctx context.Context, actorID, schoolID, enrollmentID int64) (*repository.StudentEnrollmentCurriculum, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.GetStudentEnrollmentCurriculum(ctx, actorID, schoolID, enrollmentID)
}
func (s *AcademicService) SaveStudentEnrollmentSubjects(ctx context.Context, actorID, schoolID, enrollmentID, curriculumID int64, selected []int64) (*repository.StudentEnrollmentCurriculum, error) {
	if s == nil || s.repository == nil { return nil, repository.ErrAcademicAccessDenied }
	return s.repository.SaveStudentEnrollmentSubjects(ctx, actorID, schoolID, enrollmentID, curriculumID, selected)
}
