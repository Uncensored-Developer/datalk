package api

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
)

// Client is the client interface to the connections API contract that other services can depend on.
//
//go:generate go tool with-modfile mockery --name Client --structname API --outpkg testing --output ./testing --filename generated__connections_api_mocks.go
type Client interface {
	CreateConnection(ctx context.Context, params NewConnectionParams) (*connections.Connection, error)
	GetConnection(ctx context.Context, ID int32) (*connections.Connection, error)
	ListConnections(ctx context.Context, params ListConnectionsParams) ([]*connections.Connection, error)
	UpdateConnection(ctx context.Context, params UpdateConnectionParams) (*connections.Connection, error)
	DeleteConnection(ctx context.Context, connectionID int32) error
	CreateAccess(ctx context.Context, params NewAccessParams) (*connections.Access, error)
	ListAccess(ctx context.Context, params ListAccessParams) ([]*connections.Access, error)
	GetAccess(ctx context.Context, userID int32, connectionID int32) (*connections.Access, error)
}
