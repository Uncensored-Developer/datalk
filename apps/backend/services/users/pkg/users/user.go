package users

import "time"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type User struct {
	ID           int64
	Email        string
	Name         string
	PasswordHash string
	Role         Role
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}
