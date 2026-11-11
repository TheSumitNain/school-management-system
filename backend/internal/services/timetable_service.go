package services

import (
	"context"

	"school-management-system/internal/repository"
)

type timetableRepository interface {
	GetCurrentTimetableYear(context.Context,int64,int64)(*repository.AcademicYear,error)
	ListTimetablePeriods(context.Context,int64,int64,int64)([]repository.TimetablePeriod,error)
	SaveTimetablePeriod(context.Context,int64,int64,int64,int64,int,string,string,string,bool)(*repository.TimetablePeriod,error)
	ListTimetableEntries(context.Context,int64,int64,int64,int64)([]repository.TimetableEntry,error)
	SaveTimetableEntry(context.Context,int64,int64,int64,int64,int64,int64,int64,int64,int64,int,*string)(*repository.TimetableEntry,error)
	DeleteTimetableEntry(context.Context,int64,int64,int64,int64)error
}

type TimetableService struct { repository timetableRepository }

func NewTimetableService(repo timetableRepository)*TimetableService{return &TimetableService{repository:repo}}

func(s *TimetableService) CurrentYear(ctx context.Context,actorID,schoolID int64)(*repository.AcademicYear,error){
	if s==nil||s.repository==nil{return nil,repository.ErrTimetableAccessDenied}
	return s.repository.GetCurrentTimetableYear(ctx,actorID,schoolID)
}
func(s *TimetableService) ListPeriods(ctx context.Context,actorID,schoolID,yearID int64)([]repository.TimetablePeriod,error){
	if s==nil||s.repository==nil{return nil,repository.ErrTimetableAccessDenied}
	return s.repository.ListTimetablePeriods(ctx,actorID,schoolID,yearID)
}
func(s *TimetableService) SavePeriod(ctx context.Context,actorID,schoolID,id,yearID int64,number int,name,startsAt,endsAt string,isBreak bool)(*repository.TimetablePeriod,error){
	if s==nil||s.repository==nil{return nil,repository.ErrTimetableAccessDenied}
	return s.repository.SaveTimetablePeriod(ctx,actorID,schoolID,id,yearID,number,name,startsAt,endsAt,isBreak)
}
func(s *TimetableService) ListEntries(ctx context.Context,actorID,schoolID,yearID,sectionID int64)([]repository.TimetableEntry,error){
	if s==nil||s.repository==nil{return nil,repository.ErrTimetableAccessDenied}
	return s.repository.ListTimetableEntries(ctx,actorID,schoolID,yearID,sectionID)
}
func(s *TimetableService) SaveEntry(ctx context.Context,actorID,schoolID,id,yearID,sectionID,subjectID,teacherID,assignmentID,periodID int64,day int,room *string)(*repository.TimetableEntry,error){
	if s==nil||s.repository==nil{return nil,repository.ErrTimetableAccessDenied}
	return s.repository.SaveTimetableEntry(ctx,actorID,schoolID,id,yearID,sectionID,subjectID,teacherID,assignmentID,periodID,day,room)
}
func(s *TimetableService) DeleteEntry(ctx context.Context,actorID,schoolID,yearID,id int64)error{
	if s==nil||s.repository==nil{return repository.ErrTimetableAccessDenied}
	return s.repository.DeleteTimetableEntry(ctx,actorID,schoolID,yearID,id)
}
