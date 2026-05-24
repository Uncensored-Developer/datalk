package api

import "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"

type NewUserParams struct {
	Name     string
	Email    string
	Password string
	Role     users.Role
}

type LoginParams struct {
	Email    string
	Password string
}

type ChangePasswordParams struct {
	UserID          int32
	CurrentPassword string
	NewPassword     string
}
