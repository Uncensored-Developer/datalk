package schemas

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	schemadb "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/db"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_RetrieveRelevantSchemaContextIntegration_SearchesSeededChunks(t *testing.T) {
	runner, cfg := requireIntegrationRunner(t, "schema_retrieval")
	ctx := t.Context()
	connectionID := seedSchemaConnection(t, runner.Conn)

	storage := schemadb.NewStorage(runner.Conn)
	oldSnapshot := &schematypes.Snapshot{
		ConnectionID:   connectionID,
		SchemaHash:     fmt.Sprintf("old-%d", time.Now().UnixNano()),
		SchemaJSON:     json.RawMessage(`{"tables":["old"]}`),
		IntrospectedAt: time.Now().UTC().Add(-time.Hour),
	}
	latestSnapshot := &schematypes.Snapshot{
		ConnectionID:   connectionID,
		SchemaHash:     fmt.Sprintf("latest-%d", time.Now().UnixNano()),
		SchemaJSON:     json.RawMessage(`{"tables":["latest"]}`),
		IntrospectedAt: time.Now().UTC(),
	}
	require.NoError(t, storage.InsertSnapshot(ctx, oldSnapshot))
	require.NoError(t, storage.InsertSnapshot(ctx, latestSnapshot))
	require.NoError(t, storage.UpsertEmbeddingJob(ctx, &schematypes.EmbeddingJob{
		SnapshotID: oldSnapshot.ID,
		Status:     schematypes.EmbeddingJobStatusCompleted,
		StartedAt:  time.Now().UTC(),
	}))
	require.NoError(t, storage.UpsertEmbeddingJob(ctx, &schematypes.EmbeddingJob{
		SnapshotID: latestSnapshot.ID,
		Status:     schematypes.EmbeddingJobStatusProcessing,
		StartedAt:  time.Now().UTC(),
	}))

	usersChunk := &schematypes.Chunk{
		SnapshotID:   oldSnapshot.ID,
		ConnectionID: connectionID,
		ObjectType:   "table",
		ObjectName:   "public.users",
		SchemaJSON:   json.RawMessage(`{"table":"users"}`),
		Content:      "users table with subscription relationship",
		Embedding:    integrationEmbeddingAt(0),
		Metadata:     json.RawMessage(`{"qualified_name":"public.users"}`),
	}
	subscriptionsChunk := &schematypes.Chunk{
		SnapshotID:   oldSnapshot.ID,
		ConnectionID: connectionID,
		ObjectType:   "table",
		ObjectName:   "public.subscriptions",
		SchemaJSON:   json.RawMessage(`{"table":"subscriptions"}`),
		Content:      "subscriptions table with subscribed_at",
		Embedding:    integrationEmbeddingAt(1),
		Metadata:     json.RawMessage(`{"qualified_name":"public.subscriptions"}`),
	}
	ignoredLatestChunk := &schematypes.Chunk{
		SnapshotID:   latestSnapshot.ID,
		ConnectionID: connectionID,
		ObjectType:   "table",
		ObjectName:   "public.processing_snapshot",
		SchemaJSON:   json.RawMessage(`{"table":"processing_snapshot"}`),
		Content:      "processing snapshot should not be used",
		Embedding:    integrationEmbeddingAt(0),
		Metadata:     json.RawMessage(`{"qualified_name":"public.processing_snapshot"}`),
	}
	require.NoError(t, storage.InsertChunk(ctx, usersChunk))
	require.NoError(t, storage.InsertChunk(ctx, subscriptionsChunk))
	require.NoError(t, storage.InsertChunk(ctx, ignoredLatestChunk))

	service := NewService(
		runner.Conn,
		config.Config{EmbeddingEnabled: true, EmbeddingBatchSize: cfg.EmbeddingBatchSize},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		nil,
		nil,
		nil,
		staticEmbeddingClient{embedding: integrationEmbeddingAt(0)},
	)

	got, err := service.RetrieveRelevantSchemaContext(ctx, schematypes.RetrieveRelevantSchemaContextParams{
		ConnectionID: connectionID,
		QueryText:    "how many users subscribed this month",
		Limit:        2,
	})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, oldSnapshot.ID, got.SnapshotID)
	require.Len(t, got.Chunks, 2)
	assert.Equal(t, usersChunk.ID, got.Chunks[0].ChunkID)
	assert.Equal(t, "public.users", got.Chunks[0].ObjectName)
	assert.Equal(t, subscriptionsChunk.ID, got.Chunks[1].ChunkID)
	assert.NotContains(t, []string{got.Chunks[0].ObjectName, got.Chunks[1].ObjectName}, "public.processing_snapshot")
}

type staticEmbeddingClient struct {
	embedding []float32
}

func (c staticEmbeddingClient) EmbedTexts(_ context.Context, inputs []string) ([][]float32, error) {
	embeddings := make([][]float32, 0, len(inputs))
	for range inputs {
		embeddings = append(embeddings, c.embedding)
	}
	return embeddings, nil
}

func seedSchemaConnection(t *testing.T, db *sql.DB) int32 {
	t.Helper()

	var userID int32
	err := db.QueryRowContext(
		t.Context(),
		`INSERT INTO users (email, name, password_hash, role) VALUES ($1, 'Schema User', 'hash', 'member') RETURNING id`,
		fmt.Sprintf("schema-%d@example.com", time.Now().UnixNano()),
	).Scan(&userID)
	require.NoError(t, err)

	var connectionID int32
	err = db.QueryRowContext(
		t.Context(),
		`INSERT INTO connections (name, kind, dsn, user_id) VALUES ($1, 'postgres', 'postgres://schema-retrieval', $2) RETURNING id`,
		fmt.Sprintf("schema-integration-%d", time.Now().UnixNano()),
		userID,
	).Scan(&connectionID)
	require.NoError(t, err)

	return connectionID
}

func integrationEmbeddingAt(index int) []float32 {
	vector := make([]float32, 768)
	vector[index] = 1
	return vector
}
