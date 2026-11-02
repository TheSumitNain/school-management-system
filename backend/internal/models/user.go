package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	PrincipalID  int64     `json:"-"`
	SchoolID     int64     `json:"school_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
