package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"school-management-system/internal/models"
)

type fakeStaffAccountRepository struct {
	user *models.User
	actorID int64
	schoolID int64
	username string
	passwordHash string
	calls int
	updateCalls int
	enableCalls int
	enablePasswordHash string
	profileCalls int
	profile models.StaffProfile
}

func (f *fakeStaffAccountRepository) CreateStaffAccount(_ context.Context, actorID, schoolID int64, username, passwordHash string) (*models.User, error) {
	f.calls++
	f.actorID, f.schoolID, f.username, f.passwordHash = actorID, schoolID, username, passwordHash
	return f.user, nil
}

func (f *fakeStaffAccountRepository) CreateStaffAccountWithProfile(_ context.Context, actorID, schoolID int64, username, passwordHash string, profile models.StaffProfile) (*models.User, error) {
	f.profileCalls++
	f.actorID, f.schoolID, f.username, f.passwordHash = actorID, schoolID, username, passwordHash
	f.profile = profile
	return f.user, nil
}

func (f *fakeStaffAccountRepository) ListStaffAccounts(context.Context, int64, int64) ([]models.User, error) {
	return []models.User{}, nil
}

func (f *fakeStaffAccountRepository) ListStaffDirectoryPage(_ context.Context, _, _ int64, filter models.StaffDirectoryFilter) (*models.StaffDirectoryPage, error) {
	return &models.StaffDirectoryPage{Staff: []models.User{}, Total: 0, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (f *fakeStaffAccountRepository) GetStaffDirectorySummary(context.Context, int64, int64) (*models.StaffDirectorySummary, error) {
	return &models.StaffDirectorySummary{Total: 12, ActiveEmployment: 10, LoginEnabled: 9, ProfileIncomplete: 2, ActiveEmploymentDisabledLogin: 1, MustChangePassword: 3, InactiveEmployment: 2}, nil
}

func (f *fakeStaffAccountRepository) DisableStaffAccount(context.Context, int64, int64, int64) error {
	return nil
}

func (f *fakeStaffAccountRepository) UpdateStaffAccount(context.Context, int64, int64, int64, string) (*models.User, error) {
	f.updateCalls++
	return f.user, nil
}

func (f *fakeStaffAccountRepository) UpdateStaffProfile(_ context.Context, _, _, _ int64, profile models.StaffProfile) (*models.User, error) {
	f.profileCalls++
	f.profile = profile
	return f.user, nil
}

func (f *fakeStaffAccountRepository) EnableStaffAccount(_ context.Context, _, _, _ int64, passwordHash string) error {
	f.enableCalls++
	f.enablePasswordHash = passwordHash
	return nil
}

func TestStaffAccountServiceValidatesDirectoryFiltersAndPagination(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	valid := models.StaffDirectoryFilter{Search: "  teacher  ", EmploymentStatus: "active", AccountStatus: "enabled", ProfileStatus: "complete", Limit: 50, Offset: 100}
	page, err := service.ListDirectoryPage(context.Background(), 31, 8, valid)
	if err != nil || page == nil || page.Limit != 50 || page.Offset != 100 {
		t.Fatalf("valid directory query failed: page=%+v err=%v", page, err)
	}
	for _, invalid := range []models.StaffDirectoryFilter{
		{Limit: 101}, {Limit: -1}, {Limit: 50, Offset: -1}, {Limit: 50, Offset: 1_000_001},
		{Limit: 50, EmploymentStatus: "terminated"}, {Limit: 50, AccountStatus: "pending"},
		{Limit: 50, ProfileStatus: "unknown"}, {Limit: 50, Search: strings.Repeat("x", 101)},
	} {
		if _, err := service.ListDirectoryPage(context.Background(), 31, 8, invalid); err != ErrInvalidStaffAccount {
			t.Fatalf("invalid directory filter %+v error=%v, want ErrInvalidStaffAccount", invalid, err)
		}
	}
}

func TestStaffAccountServiceReturnsDirectorySummaryForSchoolActor(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	summary, err := service.GetDirectorySummary(context.Background(), 31, 8)
	if err != nil || summary == nil || summary.Total != 12 || summary.ActiveEmployment != 10 || summary.MustChangePassword != 3 {
		t.Fatalf("directory summary mismatch: summary=%+v err=%v", summary, err)
	}
	for _, invalid := range [][2]int64{{0, 8}, {31, 0}} {
		if _, err := service.GetDirectorySummary(context.Background(), invalid[0], invalid[1]); err != ErrInvalidStaffAccount {
			t.Fatalf("invalid actor/school %v error=%v, want ErrInvalidStaffAccount", invalid, err)
		}
	}
}

func TestStaffAccountServiceCreatesHashedTemporaryAccount(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{
		ID: 15, PrincipalID: 27, SchoolID: 8, Username: "teacher.one",
		Status: "active", MustChangePassword: true,
	}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	user, err := service.Create(context.Background(), 31, 8, " Teacher.One ", "A-Strong-Initial-Pass-123")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if user != repo.user || repo.calls != 1 || repo.actorID != 31 || repo.schoolID != 8 ||
		repo.username != "teacher.one" || repo.passwordHash == "" || repo.passwordHash == "A-Strong-Initial-Pass-123" {
		t.Fatalf("unexpected staff account service behavior: user=%+v repo=%+v", user, repo)
	}
	if CheckPassword("A-Strong-Initial-Pass-123", repo.passwordHash) != true {
		t.Fatal("repository did not receive a valid hash for the initial password")
	}
}

func TestStaffAccountServiceRejectsInvalidUsernameOrPassword(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	for _, username := range []string{"ab", "bad name", "-teacher", "नमस्ते", strings.Repeat("a", 33)} {
		if _, err := service.Create(context.Background(), 31, 8, username, "A-Strong-Initial-Pass-123"); err == nil {
			t.Errorf("Create(%q) expected username validation error", username)
		}
	}
	if _, err := service.Create(context.Background(), 31, 8, "teacher.one", "short"); err != ErrInvalidPassword {
		t.Fatalf("invalid password error = %v, want ErrInvalidPassword", err)
	}
	if repo.calls != 0 {
		t.Fatalf("invalid requests reached repository %d times", repo.calls)
	}
}


func TestStaffAccountServiceValidatesUsernameUpdate(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8, Username: "new.teacher"}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	user, err := service.Update(context.Background(), 31, 8, 27, " New.Teacher ")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if user != repo.user || repo.updateCalls != 1 {
		t.Fatalf("Update did not call repository exactly once: user=%+v calls=%d", user, repo.updateCalls)
	}
	if _, err := service.Update(context.Background(), 31, 8, 27, "bad name"); err != ErrInvalidStaffAccount {
		t.Fatalf("invalid username update error = %v, want ErrInvalidStaffAccount", err)
	}
}

func TestStaffAccountServiceHashesTemporaryPasswordBeforeReEnable(t *testing.T) {
	repo := &fakeStaffAccountRepository{}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	if err := service.Enable(context.Background(), 31, 8, 27, "A-Strong-Temporary-Pass-123"); err != nil {
		t.Fatalf("Enable returned error: %v", err)
	}
	if repo.enableCalls != 1 || repo.enablePasswordHash == "" ||
		repo.enablePasswordHash == "A-Strong-Temporary-Pass-123" ||
		!CheckPassword("A-Strong-Temporary-Pass-123", repo.enablePasswordHash) {
		t.Fatalf("Enable did not pass a valid password hash: calls=%d hash_set=%v",
			repo.enableCalls, repo.enablePasswordHash != "")
	}
	if err := service.Enable(context.Background(), 31, 8, 27, "short"); err != ErrInvalidPassword {
		t.Fatalf("invalid temporary password error = %v, want ErrInvalidPassword", err)
	}
}

func TestValidStaffJoiningDateUsesAsiaKolkataCalendarBoundary(t *testing.T) {
	now := time.Date(2026, time.November, 24, 20, 0, 0, 0, time.UTC) // 2026-11-25 in Asia/Kolkata
	within := time.Date(2027, time.November, 25, 0, 0, 0, 0, time.UTC)
	beyond := time.Date(2027, time.November, 26, 0, 0, 0, 0, time.UTC)
	if !validStaffJoiningDate(nil, now) {
		t.Fatal("nil joining date should be valid")
	}
	if !validStaffJoiningDate(&within, now) {
		t.Fatal("joining date on the school-local one-year boundary should be valid")
	}
	if validStaffJoiningDate(&beyond, now) {
		t.Fatal("joining date after the school-local one-year boundary should be invalid")
	}
	leapNow := time.Date(2024, time.February, 29, 12, 0, 0, 0, time.FixedZone("IST", 5*60*60+30*60))
	leapBoundary := time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)
	leapBeyond := time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC)
	if !validStaffJoiningDate(&leapBoundary, leapNow) || validStaffJoiningDate(&leapBeyond, leapNow) {
		t.Fatal("leap-day anniversary should clamp to the final valid day of February")
	}
}

func TestStaffAccountServiceRejectsJoiningDateBeyondOneYearOnProfileUpdate(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	base := models.StaffProfile{
		EmployeeCode: "EMP-27", FullName: "Teacher One", Designation: "Teacher",
		MobileNumber: "9876543210", EmploymentStatus: "active",
	}
	yearZero := time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)
	base.JoiningDate = &yearZero
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != ErrInvalidStaffAccount {
		t.Fatalf("year-zero joining date update error = %v, want ErrInvalidStaffAccount", err)
	}
	if repo.profileCalls != 0 {
		t.Fatalf("year-zero joining date reached repository %d times", repo.profileCalls)
	}

	future := time.Now().AddDate(1, 0, 1)
	base.JoiningDate = &future
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != ErrInvalidStaffAccount {
		t.Fatalf("future joining date update error = %v, want ErrInvalidStaffAccount", err)
	}
	if repo.profileCalls != 0 {
		t.Fatalf("invalid future joining date reached repository %d times", repo.profileCalls)
	}

	valid := time.Now().AddDate(0, 6, 0)
	base.JoiningDate = &valid
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != nil {
		t.Fatalf("valid upcoming joining date update returned error: %v", err)
	}
	if repo.profileCalls != 1 {
		t.Fatalf("valid profile update calls=%d want=1", repo.profileCalls)
	}
}

func TestStaffAccountServiceRejectsMissingEmploymentStatusOnProfileUpdate(t *testing.T) {
    repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8}}
    service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
    profile := models.StaffProfile{
        EmployeeCode: "EMP-27", FullName: "Teacher One", Designation: "Teacher",
        MobileNumber: "9876543210", EmploymentStatus: "",
    }
    if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, profile); err != ErrInvalidStaffAccount {
        t.Fatalf("missing employment status update error = %v, want ErrInvalidStaffAccount", err)
    }
    if repo.profileCalls != 0 {
        t.Fatalf("profile update without employment status reached repository %d times", repo.profileCalls)
    }
}

func TestStaffProfileTextLimitsCountUnicodeCharacters(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	base := models.StaffProfile{
		EmployeeCode: "EMP-27", FullName: strings.Repeat("🔤", 160), Designation: strings.Repeat("📚", 100),
		MobileNumber: "9876543210", EmploymentStatus: "active",
	}
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != nil {
		t.Fatalf("valid Unicode profile update returned error: %v", err)
	}
	if repo.profileCalls != 1 {
		t.Fatalf("valid Unicode profile update calls=%d want=1", repo.profileCalls)
	}

	base.FullName = strings.Repeat("🔤", 161)
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != ErrInvalidStaffAccount {
		t.Fatalf("161-character full name error = %v, want ErrInvalidStaffAccount", err)
	}
	base.FullName = strings.Repeat("🔤", 160)
	base.Designation = strings.Repeat("📚", 101)
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, base); err != ErrInvalidStaffAccount {
		t.Fatalf("101-character designation error = %v, want ErrInvalidStaffAccount", err)
	}
	if repo.profileCalls != 1 {
		t.Fatalf("invalid Unicode profile updates reached repository; calls=%d want=1", repo.profileCalls)
	}
}

func TestStaffProfileTextLimitsCountUnicodeCharactersOnCreate(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	profile := models.StaffProfile{
		EmployeeCode: "EMP-27", FullName: strings.Repeat("🔤", 160), Designation: strings.Repeat("📚", 100),
		MobileNumber: "9876543210", EmploymentStatus: "active",
	}
	user, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.one", "A-Strong-Initial-Pass-123", profile)
	if err != nil {
		t.Fatalf("valid Unicode profile creation returned error: %v", err)
	}
	if user != repo.user || repo.profileCalls != 1 || repo.passwordHash == "" ||
		repo.passwordHash == "A-Strong-Initial-Pass-123" || len([]rune(repo.profile.FullName)) != 160 ||
		len([]rune(repo.profile.Designation)) != 100 {
		t.Fatalf("valid Unicode profile creation did not reach repository correctly: user=%+v calls=%d", user, repo.profileCalls)
	}

	profile.FullName = strings.Repeat("🔤", 161)
	if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.two", "A-Strong-Initial-Pass-123", profile); err != ErrInvalidStaffAccount {
		t.Fatalf("161-character full name creation error = %v, want ErrInvalidStaffAccount", err)
	}
	profile.FullName = strings.Repeat("🔤", 160)
	profile.Designation = strings.Repeat("📚", 101)
	if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.three", "A-Strong-Initial-Pass-123", profile); err != ErrInvalidStaffAccount {
		t.Fatalf("101-character designation creation error = %v, want ErrInvalidStaffAccount", err)
	}
	if repo.profileCalls != 1 {
		t.Fatalf("invalid Unicode profile creations reached repository; calls=%d want=1", repo.profileCalls)
	}
}


func TestStaffProfileEmailValidationMatchesDatabaseCharacterConstraints(t *testing.T) {
	repo := &fakeStaffAccountRepository{user: &models.User{ID: 15, PrincipalID: 27, SchoolID: 8}}
	service := NewStaffAccountService(repo, alwaysClearPasswordChecker{})
	profile := models.StaffProfile{
		EmployeeCode: "EMP-27", FullName: "Teacher One", Designation: "Teacher",
		MobileNumber: "9876543210", EmploymentStatus: "active",
	}
	validUnicodeEmail := strings.Repeat("é", 250) + "@x.y" // 254 Unicode characters, despite >254 UTF-8 bytes.
	profile.Email = validUnicodeEmail
	if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, profile); err != nil {
		t.Fatalf("254-character Unicode email update returned error: %v", err)
	}
	if repo.profileCalls != 1 {
		t.Fatalf("valid Unicode email update calls=%d want=1", repo.profileCalls)
	}

	for _, email := range []string{"a@", "@domain.com", "teacher@domain", "teacher@@domain.com", "teacher @domain.com", strings.Repeat("a", 251) + "@x.y"} {
		profile.Email = email
		if _, err := service.UpdateProfile(context.Background(), 31, 8, 27, profile); err != ErrInvalidStaffAccount {
			t.Errorf("UpdateProfile email %q error = %v, want ErrInvalidStaffAccount", email, err)
		}
	}
	if repo.profileCalls != 1 {
		t.Fatalf("invalid email updates reached repository; calls=%d want=1", repo.profileCalls)
	}

	profile.Email = "a@"
	if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.one", "A-Strong-Initial-Pass-123", profile); err != ErrInvalidStaffAccount {
		t.Fatalf("CreateWithProfile short email error = %v, want ErrInvalidStaffAccount", err)
	}
	profile.Email = validUnicodeEmail
	if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.two", "A-Strong-Initial-Pass-123", profile); err != nil {
		t.Fatalf("254-character Unicode email creation returned error: %v", err)
	}
	if repo.profileCalls != 2 {
		t.Fatalf("profile creation calls=%d want=2 (one valid update, one valid create)", repo.profileCalls)
	}

	profile.Email = "a@b.c"
	if _, err := service.CreateWithProfile(context.Background(), 31, 8, "teacher.three", "A-Strong-Initial-Pass-123", profile); err != nil {
		t.Fatalf("shortest frontend-valid email creation returned error: %v", err)
	}
	if repo.profileCalls != 3 {
		t.Fatalf("shortest valid email creation calls=%d want=3", repo.profileCalls)
	}

	for _, email := range []string{strings.Repeat("a", 252) + "@x.y", "teacher@@domain.com", "teacher @domain.com", "teacher@domain", "teacher@domain..com", "teacher@domain.com.", ".teacher@domain.com", "teacher..name@domain.com", "teacher@-domain.com", "teacher@domain-.com", "teacher@domain,com"} {
		profile.Email = email
		if _, err := service.CreateWithProfile(context.Background(), 31, 8, "invalid.teacher", "A-Strong-Initial-Pass-123", profile); err != ErrInvalidStaffAccount {
			t.Errorf("CreateWithProfile email %q error = %v, want ErrInvalidStaffAccount", email, err)
		}
	}
	if repo.profileCalls != 3 {
		t.Fatalf("invalid email creations reached repository; calls=%d want=3", repo.profileCalls)
	}
}
