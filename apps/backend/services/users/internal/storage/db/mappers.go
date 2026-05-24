package db

import (
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/aarondl/opt/null"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
)

func userToDB(user *users.User) *models.UserSetter {
	return &models.UserSetter{
		Email:              omit.From(user.Email),
		Name:               omit.From(user.Name),
		PasswordHash:       omit.From(user.PasswordHash),
		Role:               omit.From(string(user.Role)),
		IsActive:           omit.From(user.IsActive),
		LastLoginAt:        omitnull.FromPtr(user.LastLoginAt),
		CreatedAt:          omit.From(user.CreatedAt),
		MustChangePassword: omit.From(user.MustChangePassword),
		UpdatedAt:          omit.From(user.UpdatedAt),
	}
}

func userFromDB(dbUser *models.User) (*users.User, error) {
	user := &users.User{
		ID:                 dbUser.ID,
		Email:              dbUser.Email,
		Name:               dbUser.Name,
		PasswordHash:       dbUser.PasswordHash,
		Role:               users.Role(dbUser.Role),
		IsActive:           dbUser.IsActive,
		CreatedAt:          dbUser.CreatedAt,
		MustChangePassword: dbUser.MustChangePassword,
		UpdatedAt:          dbUser.UpdatedAt,
	}
	if dbUser.LastLoginAt.IsValue() {
		user.LastLoginAt = ptrFromNullTime(dbUser.LastLoginAt)
	}
	return user, nil
}

func usersFromDB(dbUsers []*models.User) ([]*users.User, error) {
	return slices.Map(dbUsers, userFromDB)
}

func refreshTokenToDB(token *userauth.RefreshToken) *models.RefreshTokenSetter {
	return &models.RefreshTokenSetter{
		UserID:    omit.From(token.UserID),
		TokenHash: omit.From(token.TokenHash),
		ExpiresAt: omit.From(token.ExpiresAt),
		RevokedAt: omitnull.FromPtr(token.RevokedAt),
	}
}

func refreshTokenRevokeToDB(at time.Time) *models.RefreshTokenSetter {
	return &models.RefreshTokenSetter{
		RevokedAt: omitnull.From(at),
	}
}

func refreshTokenFromDB(dbToken *models.RefreshToken) (*userauth.RefreshToken, error) {
	return &userauth.RefreshToken{
		ID:        dbToken.ID,
		UserID:    dbToken.UserID,
		TokenHash: dbToken.TokenHash,
		ExpiresAt: dbToken.ExpiresAt,
		RevokedAt: dbToken.RevokedAt.Ptr(),
		CreatedAt: dbToken.CreatedAt,
	}, nil
}

func ptrFromNullTime(value null.Val[time.Time]) *time.Time {
	if value.IsNull() {
		return nil
	}
	t := value.GetOrZero()
	return &t
}
