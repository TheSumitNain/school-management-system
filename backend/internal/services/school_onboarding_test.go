package services

import (
	"context"
	"errors"
	"testing"

	"school-management-system/internal/repository"
)

type fakeSchoolOnboardingRepository struct {
	result *repository.SchoolOnboardingResult
	err error
	called bool
	name, code, username, passwordHash string
	actorID int64
	schools []repository.PlatformSchoolRecord
	listErr error
}

func (f *fakeSchoolOnboardingRepository) ListPlatformSchools(_ context.Context, actorID int64) ([]repository.PlatformSchoolRecord, error) { f.actorID = actorID; return f.schools, f.listErr }

func (f *fakeSchoolOnboardingRepository) CreateSchoolWithAdmin(_ context.Context, actorID int64, name, code, username, passwordHash string) (*repository.SchoolOnboardingResult, error) {
	f.called = true
	f.actorID = actorID
	f.name, f.code, f.username, f.passwordHash = name, code, username, passwordHash
	return f.result, f.err
}

func TestSchoolOnboardingServiceNormalizesAndHashesCredentials(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{result: &repository.SchoolOnboardingResult{
		SchoolID: 3, SchoolName: "Example School", SchoolCode: "EX-3",
		SchoolStatus: "active", AdminPrincipalID: 8, AdminUsername: "admin.user",
	}}
	service := NewSchoolOnboardingService(repo, alwaysClearPasswordChecker{})
	result, err := service.CreateSchoolWithAdmin(context.Background(), 9, SchoolOnboardingInput{
		Name: " Example School ", Code: " ex-3 ", AdminUsername: " Admin.User ",
		AdminInitialPassword: "StrongUnique!Pass123",
	})
	if err != nil {
		t.Fatalf("CreateSchoolWithAdmin returned error: %v", err)
	}
	if !repo.called || repo.actorID != 9 || repo.name != "Example School" ||
		repo.code != "EX-3" || repo.username != "admin.user" {
		t.Fatalf("unexpected normalized repository input: %+v", repo)
	}
	if repo.passwordHash == "" || repo.passwordHash == "StrongUnique!Pass123" {
		t.Fatal("initial password was not securely hashed")
	}
	if result == nil || result.SchoolID != 3 || result.AdminPrincipalID != 8 {
		t.Fatalf("unexpected onboarding result: %+v", result)
	}
	if CheckPassword("StrongUnique!Pass123", repo.passwordHash) == false {
		t.Fatal("stored hash does not verify the supplied password")
	}
}

func TestSchoolOnboardingServiceRejectsInvalidInputBeforeRepository(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{}
	service := NewSchoolOnboardingService(repo)
	_, err := service.CreateSchoolWithAdmin(context.Background(), 9, SchoolOnboardingInput{
		Name: "School", Code: "bad code", AdminUsername: "admin.user",
		AdminInitialPassword: "StrongUnique!Pass123",
	})
	if !errors.Is(err, ErrInvalidSchoolOnboarding) || repo.called {
		t.Fatalf("expected validation failure without repository call, err=%v called=%v", err, repo.called)
	}
}

func TestSchoolOnboardingServiceRejectsWeakPasswordBeforeRepository(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{}
	service := NewSchoolOnboardingService(repo)
	_, err := service.CreateSchoolWithAdmin(context.Background(), 9, SchoolOnboardingInput{
		Name: "School", Code: "SCH-1", AdminUsername: "admin.user",
		AdminInitialPassword: "password123",
	})
	if !errors.Is(err, ErrInvalidPassword) || repo.called {
		t.Fatalf("expected password-policy failure without repository call, err=%v called=%v", err, repo.called)
	}
}


func TestSchoolOnboardingRejectsCompromisedAdminPassword(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{}
	service := NewSchoolOnboardingService(repo, fixedPasswordChecker{compromised: true})
	_, err := service.CreateSchoolWithAdmin(context.Background(), 9, SchoolOnboardingInput{
		Name: "School", Code: "SCH-2", AdminUsername: "admin.user",
		AdminInitialPassword: "StrongUnique!Pass123",
	})
	if !errors.Is(err, ErrCompromisedPassword) || repo.called {
		t.Fatalf("compromised School Admin password must be rejected before repository call, err=%v called=%v", err, repo.called)
	}
}

func TestSchoolOnboardingFailsClosedWhenBreachCheckUnavailable(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{}
	service := NewSchoolOnboardingService(repo, fixedPasswordChecker{err: errors.New("offline")})
	_, err := service.CreateSchoolWithAdmin(context.Background(), 9, SchoolOnboardingInput{
		Name: "School", Code: "SCH-3", AdminUsername: "admin.user",
		AdminInitialPassword: "StrongUnique!Pass123",
	})
	if !errors.Is(err, ErrPasswordBreachCheckUnavailable) || repo.called {
		t.Fatalf("School Admin onboarding must fail closed before repository call, err=%v called=%v", err, repo.called)
	}
}


func TestSchoolOnboardingServiceListsPlatformSchoolsForActor(t *testing.T) {
	repo := &fakeSchoolOnboardingRepository{schools: []repository.PlatformSchoolRecord{{ID: 4, Name: "North School", Code: "NORTH", Status: "active"}}}
	service := NewSchoolOnboardingService(repo, alwaysClearPasswordChecker{})
	schools, err := service.ListPlatformSchools(context.Background(), 19)
	if err != nil { t.Fatalf("ListPlatformSchools returned error: %v", err) }
	if repo.actorID != 19 || len(schools) != 1 || schools[0].ID != 4 { t.Fatalf("unexpected school listing: actor=%d schools=%+v", repo.actorID, schools) }
}
