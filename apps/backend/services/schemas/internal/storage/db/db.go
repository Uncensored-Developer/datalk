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
	pgvector "github.com/pgvector/pgvector-go"
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

// SearchChunks searches for schema chunks based on a query embedding and snapshot ID, with optional similarity threshold and limit.
func (s *Storage) SearchChunks(ctx context.Context, params storage.SearchChunksParams) ([]*schemas.RetrievedChunk, error) {
	if params.SnapshotID == 0 {
		return nil, xerrors.New("snapshot id is required")
	}
	if len(params.QueryEmbedding) == 0 {
		return nil, xerrors.New("query embedding is required")
	}
	if params.Limit <= 0 {
		return nil, xerrors.New("limit must be greater than 0")
	}

	query := `
SELECT
	id,
	object_type,
	object_name,
	content,
	schema_json,
	CAST(1 - (embedding OPERATOR(datalk.<=>) $1::datalk.vector) AS real) AS similarity
FROM schema_chunks
WHERE snapshot_id = $2
  AND embedding IS NOT NULL`

	args := []any{pgvector.NewVector(params.QueryEmbedding), params.SnapshotID}
	if params.SimilarityThreshold != nil {
		query += `
  AND CAST(1 - (embedding OPERATOR(datalk.<=>) $1::datalk.vector) AS real) >= $3`
		args = append(args, *params.SimilarityThreshold)
		query += `
ORDER BY embedding OPERATOR(datalk.<=>) $1::datalk.vector ASC, id ASC
LIMIT $4`
		args = append(args, params.Limit)
	} else {
		query += `
ORDER BY embedding OPERATOR(datalk.<=>) $1::datalk.vector ASC, id ASC
LIMIT $3`
		args = append(args, params.Limit)
	}

	rows, err := s.Executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, xerrors.Newf("failed to search chunks: %w", err)
	}
	defer rows.Close()

	results := make([]*schemas.RetrievedChunk, 0, params.Limit)
	for rows.Next() {
		var chunk schemas.RetrievedChunk
		var schemaJSON []byte
		var similarity float32

		if err := rows.Scan(
			&chunk.ChunkID,
			&chunk.ObjectType,
			&chunk.ObjectName,
			&chunk.Content,
			&schemaJSON,
			&similarity,
		); err != nil {
			return nil, xerrors.Newf("failed to scan retrieved chunk: %w", err)
		}

		chunk.SchemaJSON = schemaJSON
		chunk.Similarity = similarity
		results = append(results, &chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, xerrors.Newf("failed to iterate retrieved chunks: %w", err)
	}

	return results, nil
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
	case storage.SnapshotOrderingIntrospectedAt:
		return models.SchemaSnapshots.Columns.IntrospectedAt, nil
	case storage.SnapshotOrderingID:
		return models.SchemaSnapshots.Columns.ID, nil
	default:
		return nil, xerrors.Newf("unsupported snapshots ordering: %v", field)
	}
}
