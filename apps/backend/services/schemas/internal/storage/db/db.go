package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
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

func (s *Storage) GetSnapshot(ctx context.Context, snapshotID int32) (schemas.Snapshot, error) {
	dbSnapshot, err := models.SchemaSnapshots.Query(
		models.SelectWhere.SchemaSnapshots.ID.EQ(snapshotID),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return schemas.Snapshot{}, err
	}

	fetchedSnapshot, err := snapshotFromDB(dbSnapshot)
	if err != nil {
		return schemas.Snapshot{}, xerrors.Newf("failed to map db snapshot: %w", err)
	}

	return *fetchedSnapshot, nil
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
