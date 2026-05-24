package authenticator

import (
	"context"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	usersapi "github.com/Uncensored-Developer/datalk/apps/backend/services/users/api"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

type JWTAuthenticator struct {
	usersAPI usersapi.Client
	tokens   *userauth.TokenManager
}

func NewJWTAuthenticator(cfg config.Config, usersAPI usersapi.Client) *JWTAuthenticator {
	return &JWTAuthenticator{
		usersAPI: usersAPI,
		tokens:   userauth.NewTokenManager(cfg),
	}
}

func (a *JWTAuthenticator) Authenticate(ctx context.Context, accessToken string) (*users.User, error) {
	userID, err := a.tokens.VerifyAccessToken(accessToken, time.Now().UTC())
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := a.usersAPI.GetUser(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if !user.IsActive {
		return nil, ErrUnauthorized
	}
	return user, nil
}
