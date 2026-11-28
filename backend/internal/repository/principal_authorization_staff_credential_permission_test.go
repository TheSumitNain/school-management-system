package repository

import "testing"

func TestSchoolAdminCredentialVerificationPermissionIsApproved(t *testing.T) {
	if !approvedSchoolAdminPermissions["staff.credentials.verify"] {
		t.Fatal("School Admin runtime allowlist must include staff credential verification")
	}
}
