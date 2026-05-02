package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/info"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

var schemaEmbeddingJobsTable = psql.NewTablex[*models.SchemaEmbeddingJob, models.SchemaEmbeddingJobSlice, *models.SchemaEmbeddingJobSetter](
	"",
	info.SchemaEmbeddingJobs.Name,
	models.SchemaEmbeddingJobs.Columns,
)

type Storage struct {
	*common.Storage
}

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		Storage: common.NewStorage("schemas", conn),
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

	if len(filter.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.SchemaSnapshots.ID.In(filter.ID...))
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

func (s *Storage) ReplaceChunks(ctx context.Context, snapshotID int32, chunks []*schemas.Chunk) error {
	return s.InTransaction(ctx, func(ctx context.Context) error {
		existingChunks, err := models.SchemaChunks.Query(
			models.SelectWhere.SchemaChunks.SnapshotID.EQ(snapshotID),
		).All(ctx, s.Executor(ctx))
		if err := common.Err.HandleIgnoreNoRows(err); err != nil {
			return xerrors.Newf("failed to list chunks for replacement: %w", err)
		}

		if len(existingChunks) > 0 {
			if err := existingChunks.DeleteAll(ctx, s.Executor(ctx)); err != nil {
				return xerrors.Newf("failed to delete chunks: %w", err)
			}
		}

		for _, chunk := range chunks {
			if err := s.InsertChunk(ctx, chunk); err != nil {
				return xerrors.Newf("failed to insert chunk: %w", err)
			}
		}

		return nil
	})
}

func (s *Storage) GetEmbeddingJob(ctx context.Context, snapshotID int32) (*schemas.EmbeddingJob, error) {
	dbJob, err := models.SchemaEmbeddingJobs.Query(
		models.SelectWhere.SchemaEmbeddingJobs.SnapshotID.EQ(snapshotID),
	).One(ctx, s.Executor(ctx))
	if err := common.Err.HandleIgnoreNoRows(err); err != nil {
		return nil, xerrors.Newf("failed to fetch embedding job: %w", err)
	}
	if dbJob == nil {
		return nil, nil
	}

	job, err := embeddingJobFromDB(dbJob)
	if err != nil {
		return nil, xerrors.Newf("failed to map embedding job: %w", err)
	}

	return job, nil
}

func (s *Storage) UpsertEmbeddingJob(ctx context.Context, job *schemas.EmbeddingJob) error {
	if job == nil {
		return xerrors.New("embedding job cannot be nil")
	}

	jobSetter := embeddingJobToDB(job)
	dbJob, err := schemaEmbeddingJobsTable.Insert(
		jobSetter,
		im.OnConflict(info.SchemaEmbeddingJobs.Columns.SnapshotID.Name).DoUpdate(
			im.SetExcluded(
				info.SchemaEmbeddingJobs.Columns.Status.Name,
				info.SchemaEmbeddingJobs.Columns.ErrorMessage.Name,
				info.SchemaEmbeddingJobs.Columns.RetryCount.Name,
				info.SchemaEmbeddingJobs.Columns.StartedAt.Name,
				info.SchemaEmbeddingJobs.Columns.CompletedAt.Name,
			),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return xerrors.Newf("failed to upsert embedding job: %w", err)
	}

	upsertedJob, err := embeddingJobFromDB(dbJob)
	if err != nil {
		return xerrors.Newf("failed to map db embedding job: %w", err)
	}

	*job = *upsertedJob
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
