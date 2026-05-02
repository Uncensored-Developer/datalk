package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/aarondl/opt/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_InsertSnapshotAndListSnapshots(t *testing.T) {
	t.Parallel()

	createdConn := createConnection(t, "schema-connection")

	introspectedAt := time.Now().UTC().Truncate(time.Second)
	snapshot := &schemas.Snapshot{
		ConnectionID:   createdConn.ID,
		SchemaHash:     "hash-1",
		SchemaJSON:     json.RawMessage(`{"tables":[]}`),
		IntrospectedAt: introspectedAt,
	}

	err := s.InsertSnapshot(t.Context(), snapshot)
	require.NoError(t, err)
	require.NotZero(t, snapshot.ID)
	require.False(t, snapshot.IntrospectedAt.IsZero())

	got, err := s.ListSnapshots(t.Context(), storage.SnapshotsFilter{
		ConnectionID: []int32{createdConn.ID},
	})
	require.NoError(t, err)
	require.Len(t, got, 1)

	assert.Equal(t, snapshot.ID, got[0].ID)
	assert.Equal(t, snapshot.ConnectionID, got[0].ConnectionID)
	assert.Equal(t, snapshot.SchemaHash, got[0].SchemaHash)
	assert.JSONEq(t, string(snapshot.SchemaJSON), string(got[0].SchemaJSON))
	assert.WithinDuration(t, snapshot.IntrospectedAt, got[0].IntrospectedAt, time.Second)
}

func TestStorage_ListSnapshots_FiltersByConnection(t *testing.T) {
	t.Parallel()

	primaryConn := createConnection(t, "primary-connection")
	secondaryConn := createConnection(t, "secondary-connection")

	primarySnapshot1 := insertSnapshot(t, primaryConn.ID, "hash-started")
	primarySnapshot2 := insertSnapshot(t, primaryConn.ID, "hash-completed")
	secondarySnapshot := insertSnapshot(t, secondaryConn.ID, "hash-secondary")

	got, err := s.ListSnapshots(t.Context(), storage.SnapshotsFilter{
		ConnectionID: []int32{primaryConn.ID},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByAsc(storage.SnapshotOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, []int32{primarySnapshot1.ID, primarySnapshot2.ID}, snapshotIDs(got))
	assert.NotContains(t, snapshotIDs(got), secondarySnapshot.ID)
}

func TestStorage_ListSnapshots_LimitAndPagination(t *testing.T) {
	t.Parallel()

	createdConn := createConnection(t, "pagination-connection")

	firstSnapshot := insertSnapshot(t, createdConn.ID, "hash-page-1")
	secondSnapshot := insertSnapshot(t, createdConn.ID, "hash-page-2")
	thirdSnapshot := insertSnapshot(t, createdConn.ID, "hash-page-3")

	firstPage, err := s.ListSnapshots(t.Context(), storage.SnapshotsFilter{
		ConnectionID: []int32{createdConn.ID},
		Pagination: pagination.LimitOffsetPagination{
			Limit:  2,
			Offset: 0,
		},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByAsc(storage.SnapshotOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, firstPage, 2)
	assert.Equal(t, []int32{firstSnapshot.ID, secondSnapshot.ID}, snapshotIDs(firstPage))

	secondPage, err := s.ListSnapshots(t.Context(), storage.SnapshotsFilter{
		ConnectionID: []int32{createdConn.ID},
		Pagination: pagination.LimitOffsetPagination{
			Limit:  2,
			Offset: 2,
		},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByAsc(storage.SnapshotOrderingID),
		},
	})
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	assert.Equal(t, []int32{thirdSnapshot.ID}, snapshotIDs(secondPage))
}

func TestStorage_InsertChunk(t *testing.T) {
	t.Parallel()

	createdConn := createConnection(t, "chunk-connection")

	snapshot := &schemas.Snapshot{
		ConnectionID:   createdConn.ID,
		SchemaHash:     "hash-2",
		SchemaJSON:     json.RawMessage(`{"tables":[{"name":"orders"}]}`),
		IntrospectedAt: time.Now().UTC(),
	}
	err := s.InsertSnapshot(t.Context(), snapshot)
	require.NoError(t, err)

	embedding := make([]float32, 768)
	embedding[0] = 0.1
	embedding[1] = 0.2
	embedding[2] = 0.3

	chunk := &schemas.Chunk{
		SnapshotID:   snapshot.ID,
		ConnectionID: snapshot.ConnectionID,
		ObjectType:   "table",
		ObjectName:   "orders",
		SchemaJSON:   json.RawMessage(`{"table":"orders"}`),
		Content:      "orders table",
		Embedding:    embedding,
		Metadata:     []byte(`{"source":"introspector"}`),
		CreatedAt:    time.Now().UTC(),
	}

	err = s.InsertChunk(t.Context(), chunk)
	require.NoError(t, err)
	require.NotZero(t, chunk.ID)
	require.False(t, chunk.CreatedAt.IsZero())

	dbChunk, err := models.SchemaChunks.Query(
		models.SelectWhere.SchemaChunks.ID.EQ(chunk.ID),
	).One(t.Context(), runner.BobConn)
	require.NoError(t, err)

	got, err := chunkFromDB(dbChunk)
	require.NoError(t, err)
	assert.Equal(t, chunk.ID, got.ID)
	assert.Equal(t, chunk.SnapshotID, got.SnapshotID)
	assert.Equal(t, chunk.ConnectionID, got.ConnectionID)
	assert.Equal(t, chunk.ObjectType, got.ObjectType)
	assert.Equal(t, chunk.ObjectName, got.ObjectName)
	assert.JSONEq(t, string(chunk.SchemaJSON), string(got.SchemaJSON))
	assert.Equal(t, chunk.Content, got.Content)
	assert.Equal(t, chunk.Embedding, got.Embedding)
	assert.JSONEq(t, string(chunk.Metadata), string(got.Metadata))
}

func TestStorage_UpsertAndGetEmbeddingJob(t *testing.T) {
	t.Parallel()

	createdConn := createConnection(t, "embedding-job-connection")
	snapshot := insertSnapshot(t, createdConn.ID, "hash-job")

	job := &schemas.EmbeddingJob{
		SnapshotID: snapshot.ID,
		Status:     schemas.EmbeddingJobStatusProcessing,
		RetryCount: 1,
		StartedAt:  time.Now().UTC().Truncate(time.Second),
	}

	err := s.UpsertEmbeddingJob(t.Context(), job)
	require.NoError(t, err)

	got, err := s.GetEmbeddingJob(t.Context(), snapshot.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, job.SnapshotID, got.SnapshotID)
	assert.Equal(t, job.Status, got.Status)
	assert.Equal(t, job.RetryCount, got.RetryCount)

	now := time.Now().UTC().Truncate(time.Second)
	message := "embed failed"
	job = &schemas.EmbeddingJob{
		SnapshotID:   snapshot.ID,
		Status:       schemas.EmbeddingJobStatusFailed,
		RetryCount:   2,
		StartedAt:    job.StartedAt,
		CompletedAt:  &now,
		ErrorMessage: &message,
	}

	err = s.UpsertEmbeddingJob(t.Context(), job)
	require.NoError(t, err)

	got, err = s.GetEmbeddingJob(t.Context(), snapshot.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, schemas.EmbeddingJobStatusFailed, got.Status)
	assert.Equal(t, int32(2), got.RetryCount)
	require.NotNil(t, got.ErrorMessage)
	assert.Equal(t, message, *got.ErrorMessage)
	require.NotNil(t, got.CompletedAt)
}

func TestStorage_ReplaceChunks(t *testing.T) {
	t.Parallel()

	createdConn := createConnection(t, "replace-chunks-connection")
	snapshot := insertSnapshot(t, createdConn.ID, "hash-replace")

	initialChunk := &schemas.Chunk{
		SnapshotID:   snapshot.ID,
		ConnectionID: snapshot.ConnectionID,
		ObjectType:   "table",
		ObjectName:   "public.users",
		SchemaJSON:   json.RawMessage(`{"table":"users"}`),
		Content:      "users table",
		Embedding:    testEmbeddingVector(0.1),
		Metadata:     []byte(`{"qualified_name":"public.users"}`),
	}
	require.NoError(t, s.InsertChunk(t.Context(), initialChunk))

	replacementChunks := []*schemas.Chunk{
		{
			SnapshotID:   snapshot.ID,
			ConnectionID: snapshot.ConnectionID,
			ObjectType:   "table",
			ObjectName:   "public.orders",
			SchemaJSON:   json.RawMessage(`{"table":"orders"}`),
			Content:      "orders table",
			Embedding:    testEmbeddingVector(0.2),
			Metadata:     []byte(`{"qualified_name":"public.orders"}`),
		},
	}

	require.NoError(t, s.ReplaceChunks(t.Context(), snapshot.ID, replacementChunks))

	dbChunks, err := models.SchemaChunks.Query(
		models.SelectWhere.SchemaChunks.SnapshotID.EQ(snapshot.ID),
	).All(t.Context(), runner.BobConn)
	require.NoError(t, err)
	require.Len(t, dbChunks, 1)
	assert.Equal(t, "public.orders", dbChunks[0].ObjectName)
}

func createConnection(t *testing.T, name string) *models.Connection {
	t.Helper()

	userTmpl := factory.UserTemplate{}
	createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	connTmpl := factory.ConnectionTemplate{}
	connTmpl.Apply(t.Context(),
		factory.ConnectionMods.Name(name),
		factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
		factory.ConnectionMods.DSN(null.From("postgres://"+name)),
		factory.ConnectionMods.UserID(createdUser.ID),
	)

	return connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
}

func insertSnapshot(t *testing.T, connectionID int32, schemaHash string) *schemas.Snapshot {
	t.Helper()

	snapshot := &schemas.Snapshot{
		ConnectionID:   connectionID,
		SchemaHash:     schemaHash,
		SchemaJSON:     json.RawMessage(`{"tables":[]}`),
		IntrospectedAt: time.Now().UTC(),
	}

	err := s.InsertSnapshot(t.Context(), snapshot)
	require.NoError(t, err)

	return snapshot
}

func snapshotIDs(snapshots []*schemas.Snapshot) []int32 {
	ids := make([]int32, len(snapshots))
	for index, snapshot := range snapshots {
		ids[index] = snapshot.ID
	}

	return ids
}

func testEmbeddingVector(value float32) []float32 {
	vector := make([]float32, 768)
	for i := range vector {
		vector[i] = value
	}
	return vector
}
