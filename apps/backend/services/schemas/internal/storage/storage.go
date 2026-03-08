package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

//go:generate go tool with-modfile mockery --name Storage --outpkg testing --output ./testing --filename generated__storage_mocks.go
type Storage interface {
	InsertSnapshot(ctx context.Context, snapshot *schemas.Snapshot) error

	GetSnapshot(ctx context.Context, snapshotID int32) (schemas.Snapshot, error)

	InsertChunk(ctx context.Context, snapshot *schemas.Chunk) error
}
