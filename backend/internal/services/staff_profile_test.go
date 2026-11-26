package services

import (
    "context"
    "testing"\n    "strings"\n    "time"

    "school-management-system/internal/models"
)

type fakeStaffProfileRepository struct {
    *fakeStaffAccountRepository
    profile models.StaffProfile
    updateCalls int
    updateActorID int64
    updateSchoolID int64
    updateTargetID int64
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

func (f *fakeStaffProfileRepository) UpdateStaffProfile(_ context.Context, actorID, schoolID, targetID int64, profile models.StaffProfile) (*models.User, error) {
    f.updateCalls++
    f.updateActorID, f.updateSchoolID, f.updateTargetID = actorID, schoolID, targetID
    f.profile = profile
    return &models.User{ID: 22, PrincipalID: targetID, SchoolID: schoolID, Username: "teacher.one", ProfileComplete: true}, nil
}

func validStaffProfileForUpdate() models.StaffProfile {
    joiningDate := time.Date(2024, time.April, 1, 0, 0, 0, 0, time.UTC)
    return models.StaffProfile{
        EmployeeCode: "EMP-001", FullName: "Teacher One", Designation: "Class Teacher",
        MobileNumber: "+919876543210", Email: "teacher@example.com",
        JoiningDate: &joiningDate, EmploymentStatus: "active",
    }
}

func TestStaffAccountServiceUpdateProfileNormalizesAndPassesAuthenticatedScope(t *testing.T) {
    repo := &fakeStaffProfileRepository{fakeStaffAccountRepository: &fakeStaffAccountRepository{}}
    service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
    profile := validStaffProfileForUpdate()
    profile.EmployeeCode = " EMP-001 "
    profile.FullName = " Teacher One "
    profile.Designation = " Class Teacher "
    profile.MobileNumber = " +919876543210 "
    profile.Email = " teacher@example.com "
    user, err := service.UpdateProfile(context.Background(), 31, 8, 33, profile)
    if err != nil {
        t.Fatalf("UpdateProfile returned error: %v", err)
    }
    if user == nil || repo.updateCalls != 1 || repo.updateActorID != 31 || repo.updateSchoolID != 8 || repo.updateTargetID != 33 {
        t.Fatalf("profile update did not preserve authenticated scope: user=%+v repo=%+v", user, repo)
    }
    if repo.profile.EmployeeCode != "EMP-001" || repo.profile.FullName != "Teacher One" ||
        repo.profile.Designation != "Class Teacher" || repo.profile.MobileNumber != "+919876543210" ||
        repo.profile.Email != "teacher@example.com" {
        t.Fatalf("profile update did not normalize fields: %+v", repo.profile)
    }
}

func TestStaffAccountServiceUpdateProfileRejectsInvalidFieldsBeforeRepository(t *testing.T) {
    tests := []struct {
        name string
        mutate func(*models.StaffProfile)
    }{
        {"employee code", func(p *models.StaffProfile) { p.EmployeeCode = "invalid code" }},
        {"empty full name", func(p *models.StaffProfile) { p.FullName = "  " }},
        {"overlong full name", func(p *models.StaffProfile) { p.FullName = strings.Repeat("名", 161) }},
        {"empty designation", func(p *models.StaffProfile) { p.Designation = "" }},
        {"overlong designation", func(p *models.StaffProfile) { p.Designation = strings.Repeat("教", 101) }},
        {"invalid mobile", func(p *models.StaffProfile) { p.MobileNumber = "12345" }},
        {"invalid email", func(p *models.StaffProfile) { p.Email = "teacher@@example.com" }},
        {"invalid employment status", func(p *models.StaffProfile) { p.EmploymentStatus = "terminated" }},
        {"future joining date", func(p *models.StaffProfile) { d := time.Now().AddDate(2, 0, 0); p.JoiningDate = &d }},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            repo := &fakeStaffProfileRepository{fakeStaffAccountRepository: &fakeStaffAccountRepository{}}
            service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
            profile := validStaffProfileForUpdate()
            tc.mutate(&profile)
            if _, err := service.UpdateProfile(context.Background(), 31, 8, 33, profile); err != ErrInvalidStaffAccount {
                t.Fatalf("UpdateProfile error = %v, want ErrInvalidStaffAccount", err)
            }
            if repo.updateCalls != 0 {
                t.Fatalf("invalid profile reached repository %d times", repo.updateCalls)
            }
        })
    }
}

func TestStaffAccountServiceUpdateProfileRejectsInvalidActorAndTargetScope(t *testing.T) {
    cases := []struct{ name string; actor, school, target int64 }{
        {"missing actor", 0, 8, 33}, {"missing school", 31, 0, 33}, {"missing target", 31, 8, 0},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            repo := &fakeStaffProfileRepository{fakeStaffAccountRepository: &fakeStaffAccountRepository{}}
            service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
            if _, err := service.UpdateProfile(context.Background(), tc.actor, tc.school, tc.target, validStaffProfileForUpdate()); err != ErrInvalidStaffAccount {
                t.Fatalf("UpdateProfile error = %v, want ErrInvalidStaffAccount", err)
            }
            if repo.updateCalls != 0 {
                t.Fatalf("invalid scope reached repository %d times", repo.updateCalls)
            }
        })
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
