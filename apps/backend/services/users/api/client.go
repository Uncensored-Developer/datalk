package api

import (
	"context"

	usersservice "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

// Client is the client interface to the users API contract that other services can depend on.
//
//go:generate go tool with-modfile mockery --name Client --structname API --outpkg testing --output ./testing --filename generated__users_api_mocks.go
type Client interface {
	RegisterUser(ctx context.Context, newUser usersservice.NewUser) (*users.User, error)
	GetUser(ctx context.Context, ID int64) (*users.User, error)
}
