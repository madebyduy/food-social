package auth

import "time"

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusSuspended Status = "SUSPENDED"
	StatusBanned    Status = "BANNED"
	StatusDeleted   Status = "DELETED"
)

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         Role
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
