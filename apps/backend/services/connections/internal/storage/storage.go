package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
)

type ListConnectionsParam struct {
	ID     []int32
	Name   []string
	UserID []int32
}

type ListAccessParam struct {
	UserID       []int32
	ConnectionID []int32
}

type ListNamespaceParam struct {
	ID           []int32
	ConnectionID []int32
}

//go:generate go tool with-modfile mockery --name Storage --outpkg testing --output ./testing --filename generated__storage_mocks.go
type Storage interface {
	UpsertConnection(ctx context.Context, connection *connections.Connection) error

	ListConnections(ctx context.Context, params ListConnectionsParam) ([]*connections.Connection, error)

	UpsertAccess(ctx context.Context, access *connections.Access) error

	ListAccess(ctx context.Context, params ListAccessParam) ([]*connections.Access, error)

	UpsertNamespace(ctx context.Context, namespace *connections.Namespace) error

	ListNamespace(ctx context.Context, params ListNamespaceParam) ([]*connections.Namespace, error)
}
