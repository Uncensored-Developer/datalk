package api

import (
	"context"

	connectionsservice "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
)

// Client is the client interface to the connections API contract that other services can depend on.
//
//go:generate go tool with-modfile mockery --name Client --structname API --outpkg testing --output ./testing --filename generated__connections_api_mocks.go
type Client interface {
	RegisterConnection(ctx context.Context, newConnection connectionsservice.NewConnection) (*connections.Connection, error)
	GetConnection(ctx context.Context, ID int32) (*connections.Connection, error)
	RegisterAccess(ctx context.Context, newAccess connectionsservice.NewAccess) (*connections.Access, error)
	GetAccess(ctx context.Context, userID int32, connectionID int32) (*connections.Access, error)
	RegisterNamespace(ctx context.Context, newNamespace connectionsservice.NewNamespace) (*connections.Namespace, error)
	GetNamespace(ctx context.Context, ID int32) (*connections.Namespace, error)
}
