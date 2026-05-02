package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
)

var ErrSnapshotNotFound = xerrors.New("snapshot not found")

type SnapshotOrdering int

const (
	SnapshotOrderingIntrospectedAt SnapshotOrdering = iota
	SnapshotOrderingID
)

type SnapshotsFilter struct {
	ConnectionID []int32
	ID           []int32
	Pagination   pagination.LimitOffsetPagination
	Ordering     ordering.Orderings[SnapshotOrdering]
}

//go:generate go tool with-modfile mockery --name Storage --outpkg testing --output ./testing --filename generated__storage_mocks.go
type Storage interface {
	InsertSnapshot(ctx context.Context, snapshot *schemas.Snapshot) error

	ListSnapshots(ctx context.Context, filter SnapshotsFilter) ([]*schemas.Snapshot, error)

	InsertChunk(ctx context.Context, snapshot *schemas.Chunk) error

	ReplaceChunks(ctx context.Context, snapshotID int32, chunks []*schemas.Chunk) error

	GetEmbeddingJob(ctx context.Context, snapshotID int32) (*schemas.EmbeddingJob, error)

	UpsertEmbeddingJob(ctx context.Context, job *schemas.EmbeddingJob) error
}
