package services

import (
	"context"
	"errors"
	"strings"

	"school-management-system/internal/repository"
)

var ErrInvalidGuardian = errors.New("invalid guardian request")

type guardianRepository interface {
	ListGuardians(context.Context,int64,int64,string,string,int,int)([]repository.Guardian,int64,error)
	GetGuardian(context.Context,int64,int64,int64)(*repository.Guardian,error)
	SaveGuardian(context.Context,int64,int64,int64,string,*string,*string,string)(*repository.Guardian,error)
	ListStudentGuardians(context.Context,int64,int64,int64,bool)([]repository.StudentGuardian,error)
	LinkStudentGuardian(context.Context,int64,int64,int64,int64,string,bool,bool)(*repository.StudentGuardian,error)
	RevokeStudentGuardianLink(context.Context,int64,int64,int64,string)error
}

type GuardianService struct { repository guardianRepository }

func NewGuardianService(repo guardianRepository)*GuardianService{return &GuardianService{repository:repo}}

func(s *GuardianService) ListGuardians(ctx context.Context,actor,school int64,q,status string,limit,offset int)([]repository.Guardian,int64,error){
	if s==nil||s.repository==nil{return nil,0,repository.ErrGuardianAccessDenied}
	return s.repository.ListGuardians(ctx,actor,school,strings.TrimSpace(q),status,limit,offset)
}
func(s *GuardianService) GetGuardian(ctx context.Context,actor,school,id int64)(*repository.Guardian,error){
	if s==nil||s.repository==nil{return nil,repository.ErrGuardianAccessDenied}
	return s.repository.GetGuardian(ctx,actor,school,id)
}
func(s *GuardianService) SaveGuardian(ctx context.Context,actor,school,id int64,name string,mobile,email *string,status string)(*repository.Guardian,error){
	if s==nil||s.repository==nil{return nil,repository.ErrGuardianAccessDenied}
	return s.repository.SaveGuardian(ctx,actor,school,id,name,mobile,email,status)
}
func(s *GuardianService) ListStudentGuardians(ctx context.Context,actor,school,student int64,history bool)([]repository.StudentGuardian,error){
	if s==nil||s.repository==nil{return nil,repository.ErrGuardianAccessDenied}
	return s.repository.ListStudentGuardians(ctx,actor,school,student,history)
}
func(s *GuardianService) LinkStudentGuardian(ctx context.Context,actor,school,student,guardian int64,relation string,primary,emergency bool)(*repository.StudentGuardian,error){
	if s==nil||s.repository==nil{return nil,repository.ErrGuardianAccessDenied}
	return s.repository.LinkStudentGuardian(ctx,actor,school,student,guardian,strings.TrimSpace(relation),primary,emergency)
}
func(s *GuardianService) RevokeStudentGuardianLink(ctx context.Context,actor,school,link int64,reason string)error{
	if s==nil||s.repository==nil{return repository.ErrGuardianAccessDenied}
	return s.repository.RevokeStudentGuardianLink(ctx,actor,school,link,strings.TrimSpace(reason))
}
