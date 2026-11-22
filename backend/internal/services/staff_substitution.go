package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"school-management-system/internal/repository"
)

var ErrInvalidStaffSubstitution = errors.New("invalid staff substitution input")

type staffSubstitutionRepository interface {
	ListStaffSubstitutions(context.Context,int64,int64,string,string,int64)([]repository.StaffSubstitution,error)
	CreateStaffSubstitution(context.Context,int64,int64,int64,int64,int64,string,*int64,string)(*repository.StaffSubstitution,error)
	CancelStaffSubstitution(context.Context,int64,int64,int64)error
}

type StaffSubstitutionService struct { repo staffSubstitutionRepository }

func NewStaffSubstitutionService(repo staffSubstitutionRepository)*StaffSubstitutionService {
	return &StaffSubstitutionService{repo:repo}
}

func(s *StaffSubstitutionService) List(ctx context.Context,actor,school int64,from,to string,staffID int64)([]repository.StaffSubstitution,error) {
	if s==nil||s.repo==nil||actor<=0||school<=0||staffID<0{return nil,ErrInvalidStaffSubstitution}
	start,err:=time.Parse("2006-01-02",from);if err!=nil{return nil,ErrInvalidStaffSubstitution}
	end,err:=time.Parse("2006-01-02",to);if err!=nil||end.Before(start)||end.Sub(start)>366*24*time.Hour{return nil,ErrInvalidStaffSubstitution}
	return s.repo.ListStaffSubstitutions(ctx,actor,school,from,to,staffID)
}

func(s *StaffSubstitutionService) Create(ctx context.Context,actor,school,entryID,originalID,substituteID int64,date string,leaveID *int64,reason string)(*repository.StaffSubstitution,error) {
	if s==nil||s.repo==nil||actor<=0||school<=0||entryID<=0||originalID<=0||substituteID<=0||originalID==substituteID{return nil,ErrInvalidStaffSubstitution}
	if _,err:=time.Parse("2006-01-02",date);err!=nil{return nil,ErrInvalidStaffSubstitution}
	reason=strings.TrimSpace(reason);if reason==""||len([]rune(reason))>1000{return nil,ErrInvalidStaffSubstitution}
	if leaveID!=nil&&*leaveID<=0{return nil,ErrInvalidStaffSubstitution}
	return s.repo.CreateStaffSubstitution(ctx,actor,school,entryID,originalID,substituteID,date,leaveID,reason)
}

func(s *StaffSubstitutionService) Cancel(ctx context.Context,actor,school,id int64)error {
	if s==nil||s.repo==nil||actor<=0||school<=0||id<=0{return ErrInvalidStaffSubstitution}
	return s.repo.CancelStaffSubstitution(ctx,actor,school,id)
}
