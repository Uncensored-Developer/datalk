package db

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ptr"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
)

func userToDB(user *users.User) *models.UserSetter {
	dbLoginAt := ""
	if user.LastLoginAt != nil {
		dbLoginAt = common.TimeToDB(*user.LastLoginAt)
	}

	return &models.UserSetter{
		Email:        omit.From(user.Email),
		Name:         omit.From(user.Name),
		PasswordHash: omit.From(user.PasswordHash),
		Role:         omit.From(string(user.Role)),
		IsActive:     omit.From(common.BoolToDB(user.IsActive)),
		LastLoginAt:  omitnull.From(dbLoginAt),
		CreatedAt:    omit.From(common.TimeToDB(user.CreatedAt)),
	}
}

func userFromDB(dbUser *models.User) (*users.User, error) {
	return &users.User{
		ID:           dbUser.ID,
		Email:        dbUser.Email,
		Name:         dbUser.Name,
		PasswordHash: dbUser.PasswordHash,
		Role:         users.Role(dbUser.Role),
		IsActive:     common.BoolFromDB(dbUser.IsActive),
		LastLoginAt:  ptr.TimePtr(common.TimeFromDB(dbUser.LastLoginAt.GetOrZero())),
		CreatedAt:    common.TimeFromDB(dbUser.CreatedAt),
	}, nil
}

func usersFromDB(dbUsers []*models.User) ([]*users.User, error) {
	return slices.Map(dbUsers, userFromDB)
}
