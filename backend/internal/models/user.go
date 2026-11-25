package models

import "time"

type StaffProfile struct {
    EmployeeCode string
    FullName string
    Designation string
    MobileNumber string
    Email string
    JoiningDate *time.Time
    EmploymentStatus string
}

type User struct {
    EmployeeCode string `json:"employee_code"`
    FullName string `json:"full_name"`
    Designation string `json:"designation"`
    MobileNumber string `json:"mobile_number"`
    Email string `json:"email"`
    JoiningDate *time.Time `json:"joining_date"`
    EmploymentStatus string `json:"employment_status"`
    ProfileComplete bool `json:"profile_complete"`
	ID           int64     `json:"id"`
	PrincipalID  int64     `json:"-"`
	SchoolID     int64     `json:"school_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	MustChangePassword bool `json:"must_change_password"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}


type StaffDirectoryFilter struct {
	Search string
	EmploymentStatus string
	AccountStatus string
	ProfileStatus string
	Limit int
	Offset int
}

type StaffDirectoryPage struct {
	Staff []User
	Total int64
	Limit int
	Offset int
}

type StaffDirectorySummary struct {
	Total int64
	ActiveEmployment int64
	LoginEnabled int64
	ProfileIncomplete int64
	ActiveEmploymentDisabledLogin int64
	MustChangePassword int64
	InactiveEmployment int64
}
