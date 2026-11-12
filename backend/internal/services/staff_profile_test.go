package services

import (
    "context"
    "testing"

    "school-management-system/internal/models"
)

type fakeStaffProfileRepository struct {
    *fakeStaffAccountRepository
    profile models.StaffProfile
}

func (f *fakeStaffProfileRepository) CreateStaffAccountWithProfile(_ context.Context, actorID, schoolID int64, username, passwordHash string, profile models.StaffProfile) (*models.User, error) {
    f.calls++
    f.actorID, f.schoolID, f.username, f.passwordHash = actorID, schoolID, username, passwordHash
    f.profile = profile
    return &models.User{ID: 22, PrincipalID: 33, SchoolID: schoolID, Username: username, ProfileComplete: true}, nil
}

func TestStaffAccountServiceCreatesProfileAndAccountTogether(t *testing.T) {
    repo := &fakeStaffProfileRepository{fakeStaffAccountRepository: &fakeStaffAccountRepository{}}
    service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
    user, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.one", "A-Strong-Initial-Pass-123", models.StaffProfile{
        EmployeeCode: "EMP-001",
        FullName: "Rahul Sharma",
        Designation: "Mathematics Teacher",
        MobileNumber: "+919876543210",
        Email: "rahul@example.com",
    })
    if err != nil {
        t.Fatalf("CreateWithProfile returned error: %v", err)
    }
    if user == nil || !user.ProfileComplete || repo.calls != 1 || repo.actorID != 31 || repo.schoolID != 8 ||
        repo.username != "teacher.one" || repo.passwordHash == "" || repo.profile.EmploymentStatus != "active" {
        t.Fatalf("profile-aware account creation did not call the atomic repository operation: user=%+v repo=%+v", user, repo)
    }
    if CheckPassword("A-Strong-Initial-Pass-123", repo.passwordHash) != true {
        t.Fatal("profile-aware creation did not pass a valid password hash")
    }
}

func TestStaffAccountServiceRejectsInvalidEmployeeProfile(t *testing.T) {
    repo := &fakeStaffProfileRepository{fakeStaffAccountRepository: &fakeStaffAccountRepository{}}
    service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
    profile := models.StaffProfile{EmployeeCode: "EMP-001", FullName: "Teacher", Designation: "Teacher", MobileNumber: "9876543210"}
    profile.MobileNumber = "12345"
    if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.one", "A-Strong-Initial-Pass-123", profile); err != ErrInvalidStaffAccount {
        t.Fatalf("invalid mobile error = %v, want ErrInvalidStaffAccount", err)
    }
    if repo.calls != 0 {
        t.Fatalf("invalid profile reached repository %d times", repo.calls)
    }
}
