package api

import "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"

type NewUserParams struct {
	Name     string
	Email    string
	Password string
	Role     users.Role
}
