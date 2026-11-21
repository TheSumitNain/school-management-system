package models

import "time"

type StaffQualification struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	QualificationType string `json:"qualification_type"`
	QualificationName string `json:"qualification_name"`
	Specialization string `json:"specialization,omitempty"`
	Institution string `json:"institution"`
	BoardUniversity string `json:"board_university,omitempty"`
	CompletionYear *int `json:"completion_year,omitempty"`
	Score *float64 `json:"score,omitempty"`
	ScoreScale string `json:"score_scale,omitempty"`
	CredentialReference string `json:"credential_reference,omitempty"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StaffCertification struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	CertificateName string `json:"certificate_name"`
	IssuingOrganization string `json:"issuing_organization"`
	IssueDate string `json:"issue_date,omitempty"`
	ExpiryDate string `json:"expiry_date,omitempty"`
	CredentialID string `json:"credential_id,omitempty"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type StaffExperience struct {
	ID int64 `json:"id"`
	StaffPrincipalID int64 `json:"staff_principal_id"`
	EmployerName string `json:"employer_name"`
	Designation string `json:"designation"`
	EmploymentType string `json:"employment_type"`
	StartDate string `json:"start_date"`
	EndDate string `json:"end_date,omitempty"`
	IsCurrent bool `json:"is_current"`
	Description string `json:"description,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
