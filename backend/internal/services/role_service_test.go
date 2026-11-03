package services

import (
	"context"
	"testing"

	"school-management-system/internal/repository"
)

type fakeRoleRepository struct {
	roles []repository.SchoolRole
	permissions []repository.SchoolPermission
	created *repository.SchoolRole
	actorID int64
	schoolID int64
	roleKey string
	name string
}

func (f *fakeRoleRepository) ListSchoolRoles(context.Context, int64, int64) ([]repository.SchoolRole, error) {
	return f.roles, nil
}

func (f *fakeRoleRepository) ListSchoolPermissions(context.Context) ([]repository.SchoolPermission, error) {
	return f.permissions, nil
}

func (f *fakeRoleRepository) CreateCustomRole(_ context.Context, actorID, schoolID int64, roleKey, name string) (*repository.SchoolRole, error) {
	f.actorID, f.schoolID, f.roleKey, f.name = actorID, schoolID, roleKey, name
	return f.created, nil
}

func (f *fakeRoleRepository) GrantSchoolPermissionDelegation(context.Context, int64, int64, string, bool, bool) (*repository.SchoolPermissionDelegation, error) {
	return nil, nil
}

func (f *fakeRoleRepository) RevokeSchoolPermissionDelegation(context.Context, int64, int64) error {
	return nil
}

func (f *fakeRoleRepository) ListSchoolPermissionDelegations(context.Context, int64, int64) ([]repository.SchoolPermissionDelegation, error) {
	return nil, nil
}
func (f *fakeRoleRepository) GrantCustomRolePermission(context.Context, int64, int64, int64, int64, string) error {
	return nil
}
func (f *fakeRoleRepository) RevokeCustomRolePermission(context.Context, int64, int64, int64, string) error {
	return nil
}

func (f *fakeRoleRepository) ListStaffPermissionOverrides(context.Context, int64, int64, int64) ([]repository.StaffPermissionOverride, error) {
	return nil, nil
}
func (f *fakeRoleRepository) SetStaffPermissionOverride(context.Context, int64, int64, int64, int64, string, string, string) error {
	return nil
}
func (f *fakeRoleRepository) DeleteStaffPermissionOverride(context.Context, int64, int64, int64, string) error {
	return nil
}

func TestCustomRoleKey(t *testing.T) {
	tests := []struct {
		name string
		input string
		want string
	}{
		{name: "spaces and punctuation", input: "Sports & Arts", want: "custom_sports_arts"},
		{name: "mixed case", input: "Library Manager", want: "custom_library_manager"},
		{name: "trims punctuation", input: "  --Science Lab!! ", want: "custom_science_lab"},
		{name: "non ASCII only", input: "विद्यालय", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := customRoleKey(test.input); got != test.want {
				t.Fatalf("customRoleKey(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestRoleServiceCreateCustomRoleNormalizesNameAndKey(t *testing.T) {
	created := &repository.SchoolRole{ID: 9, RoleKey: "custom_sports_arts", Name: "Sports & Arts"}
	repo := &fakeRoleRepository{created: created}
	service := NewRoleService(repo)

	got, err := service.CreateCustomRole(context.Background(), 21, 8, "  Sports & Arts  ")
	if err != nil {
		t.Fatalf("CreateCustomRole returned error: %v", err)
	}
	if got != created || repo.actorID != 21 || repo.schoolID != 8 ||
		repo.roleKey != "custom_sports_arts" || repo.name != "Sports & Arts" {
		t.Fatalf("unexpected custom role creation: result=%+v repo=%+v", got, repo)
	}
}

func TestRoleServiceRejectsInvalidCustomRoleNames(t *testing.T) {
	repo := &fakeRoleRepository{}
	service := NewRoleService(repo)
	for _, name := range []string{"", " ", "A", "विद्यालय"} {
		if _, err := service.CreateCustomRole(context.Background(), 21, 8, name); err == nil {
			t.Errorf("CreateCustomRole(%q) expected validation error", name)
		}
	}
	if repo.created != nil || repo.roleKey != "" {
		t.Fatal("invalid role name reached the repository")
	}
}
