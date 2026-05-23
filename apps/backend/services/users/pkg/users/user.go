package users

import (
	"time"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleOwner  Role = "owner"
	RoleMember Role = "member"
)

type User struct {
	ID                 int32
	Email              string
	Name               string
	PasswordHash       string
	Role               Role
	IsActive           bool
	MustChangePassword bool
	LastLoginAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (u *User) IsAdmin() bool {
	if u == nil {
		return false
	}
	return u.Role == RoleOwner || u.Role == RoleAdmin
}
