package models

import "errors"

var ErrPrincipalNotAuthenticated = errors.New("principal is not eligible for authentication")

// AuthPrincipal is the canonical authentication identity introduced by V4.
// School-specific staff details remain in User; platform administrators and
// parents do not require a legacy users row.
type AuthPrincipal struct {
	ID           int64  `json:"principal_id"`
	Type         string `json:"principal_type"`
	Status       string `json:"status"`
	TokenVersion int64  `json:"-"`

	// Staff-only context. Zero/empty for platform administrators and parents.
	SchoolID int64  `json:"school_id,omitempty"`
	Username string `json:"username,omitempty"`

	// False for a parent until the mobile number is verified.
	MobileVerified bool `json:"-"`
}
