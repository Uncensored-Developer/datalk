package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

type Storage struct {
	*common.Storage
}

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		common.NewStorage("schemas", conn),
	}
}

func (s *Storage) InsertSnapshot(ctx context.Context, snapshot *schemas.Snapshot) error {
	snapshotSetter := snapshotToDB(snapshot)
	dbSnapshot, err := models.SchemaSnapshots.Insert(snapshotSetter).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	insertedSnapshot, err := snapshotFromDB(dbSnapshot)
	if err != nil {
		return xerrors.Newf("failed to map db snapshot: %w", err)
	}

	*snapshot = *insertedSnapshot
	return nil
}

func (s *Storage) ListSnapshots(ctx context.Context, filter storage.SnapshotsFilter) ([]*schemas.Snapshot, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(filter.ConnectionID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.SchemaSnapshots.ConnectionID.In(filter.ConnectionID...))
	}

	queryMods = append(
		queryMods,
		common.PaginationToBobMods(filter.Pagination)...,
	)

	orderingMods, err := common.OrderingToBobMods(
		filter.Ordering,
		listSnapshotsOrderingExpr,
	)
	if err != nil {
		return nil, xerrors.Newf("invalid list snapshots filter: %w", err)
	}
	queryMods = append(queryMods, orderingMods...)

	dbSnapshots, err := models.SchemaSnapshots.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch snapshots: %w", err)
	}

	return snapshotsFromDB(dbSnapshots)
}

func (s *Storage) InsertChunk(ctx context.Context, chunk *schemas.Chunk) error {
	chunkSetter := chunkToDB(chunk)
	dbChunk, err := models.SchemaChunks.Insert(chunkSetter).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	insertedChunk, err := chunkFromDB(dbChunk)
	if err != nil {
		return xerrors.Newf("failed to map db chunk: %w", err)
	}

	*chunk = *insertedChunk
	return nil
}

func listSnapshotsOrderingExpr(field storage.SnapshotOrdering) (bob.Expression, error) {
	switch field {
	case storage.SnapshotOrderingID:
		return models.SchemaSnapshots.Columns.ID, nil
	default:
		return nil, xerrors.Newf("unsupported snapshots ordering: %v", field)
	}
}
