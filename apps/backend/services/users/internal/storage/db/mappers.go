package db

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/gotidy/ptr"
)

func userToDB(user *users.User) *models.UserSetter {
	return &models.UserSetter{
		Email:        omit.From(user.Email),
		Name:         omit.From(user.Name),
		PasswordHash: omit.From(user.PasswordHash),
		Role:         omit.From(string(user.Role)),
		IsActive:     omit.From(user.IsActive),
		LastLoginAt:  omitnull.FromPtr(user.LastLoginAt),
		CreatedAt:    omit.From(user.CreatedAt),
	}
}

func userFromDB(dbUser *models.User) (*users.User, error) {
	return &users.User{
		ID:           dbUser.ID,
		Email:        dbUser.Email,
		Name:         dbUser.Name,
		PasswordHash: dbUser.PasswordHash,
		Role:         users.Role(dbUser.Role),
		IsActive:     dbUser.IsActive,
		LastLoginAt:  ptr.Of(dbUser.LastLoginAt.GetOrZero()),
		CreatedAt:    dbUser.CreatedAt,
	}, nil
}

func usersFromDB(dbUsers []*models.User) ([]*users.User, error) {
	return slices.Map(dbUsers, userFromDB)
}
