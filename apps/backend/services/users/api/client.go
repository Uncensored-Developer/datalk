package api

import (
	"context"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	usersservice "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

// Client is the client interface to the users API contract that other services can depend on.
//
//go:generate go tool with-modfile mockery --name Client --structname API --outpkg testing --output ./testing --filename generated__users_api_mocks.go
type Client interface {
	Setup(ctx context.Context, params NewUserParams) (*userauth.Session, error)
	Login(ctx context.Context, params LoginParams) (*userauth.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*userauth.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, params ChangePasswordParams) (*users.User, error)
	CreateUser(ctx context.Context, params NewUserParams) (*users.User, error)
	RegisterUser(ctx context.Context, newUser usersservice.NewUser) (*users.User, error)
	GetUser(ctx context.Context, ID int32) (*users.User, error)
}
