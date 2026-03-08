package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/aarondl/opt/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_InsertAndGetSnapshot(t *testing.T) {
	t.Parallel()

	userTmpl := factory.UserTemplate{}
	createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	connTmpl := factory.ConnectionTemplate{}
	connTmpl.Apply(t.Context(),
		factory.ConnectionMods.Name("schema-connection"),
		factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
		factory.ConnectionMods.DSN(null.From("postgres://schema")),
		factory.ConnectionMods.UserID(createdUser.ID),
	)
	createdConn := connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	namespaceTmpl := factory.ConnectionNamespaceTemplate{}
	namespaceTmpl.Apply(t.Context(),
		factory.ConnectionNamespaceMods.WithExistingConnection(createdConn),
		factory.ConnectionNamespaceMods.Name("public"),
		factory.ConnectionNamespaceMods.NamespaceType(string(connections.NamespaceTypeSchema)),
		factory.ConnectionNamespaceMods.IsEnabled(true),
	)
	createdNamespace := namespaceTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	snapshot := &schemas.Snapshot{
		ConnectionID:   createdConn.ID,
		NamespaceID:    createdNamespace.ID,
		SchemaHash:     "hash-1",
		SchemaJSON:     json.RawMessage(`{"tables":[]}`),
		Status:         schemas.SnapshotStatusStarted,
		ErrorMessage:   nil,
		IntrospectedAt: time.Now().UTC(),
	}

	err := s.InsertSnapshot(t.Context(), snapshot)
	require.NoError(t, err)
	require.NotZero(t, snapshot.ID)
	require.False(t, snapshot.IntrospectedAt.IsZero())

	got, err := s.GetSnapshot(t.Context(), snapshot.ID)
	require.NoError(t, err)
	assert.Equal(t, snapshot.ID, got.ID)
	assert.Equal(t, snapshot.ConnectionID, got.ConnectionID)
	assert.Equal(t, snapshot.NamespaceID, got.NamespaceID)
	assert.Equal(t, snapshot.SchemaHash, got.SchemaHash)
	assert.Equal(t, snapshot.Status, got.Status)
	assert.JSONEq(t, string(snapshot.SchemaJSON), string(got.SchemaJSON))
}

func TestStorage_InsertChunk(t *testing.T) {
	t.Parallel()

	userTmpl := factory.UserTemplate{}
	createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	connTmpl := factory.ConnectionTemplate{}
	connTmpl.Apply(t.Context(),
		factory.ConnectionMods.Name("chunk-connection"),
		factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
		factory.ConnectionMods.DSN(null.From("postgres://chunk")),
		factory.ConnectionMods.UserID(createdUser.ID),
	)
	createdConn := connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	namespaceTmpl := factory.ConnectionNamespaceTemplate{}
	namespaceTmpl.Apply(t.Context(),
		factory.ConnectionNamespaceMods.WithExistingConnection(createdConn),
		factory.ConnectionNamespaceMods.Name("public"),
		factory.ConnectionNamespaceMods.NamespaceType(string(connections.NamespaceTypeSchema)),
		factory.ConnectionNamespaceMods.IsEnabled(true),
	)
	createdNamespace := namespaceTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

	snapshot := &schemas.Snapshot{
		ConnectionID:   createdConn.ID,
		NamespaceID:    createdNamespace.ID,
		SchemaHash:     "hash-2",
		SchemaJSON:     json.RawMessage(`{"tables":[{"name":"orders"}]}`),
		Status:         schemas.SnapshotStatusCompleted,
		ErrorMessage:   nil,
		IntrospectedAt: time.Now().UTC(),
	}
	err := s.InsertSnapshot(t.Context(), snapshot)
	require.NoError(t, err)

	embedding := make([]float32, 1536)
	embedding[0] = 0.1
	embedding[1] = 0.2
	embedding[2] = 0.3

	chunk := &schemas.Chunk{
		SnapshotID:   snapshot.ID,
		ConnectionID: snapshot.ConnectionID,
		NamespaceID:  snapshot.NamespaceID,
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
		models.SelectWhere.SchemaChunks.ID.EQ(int64(chunk.ID)),
	).One(t.Context(), runner.BobConn)
	require.NoError(t, err)

	got, err := chunkFromDB(dbChunk)
	require.NoError(t, err)
	assert.Equal(t, chunk.ID, got.ID)
	assert.Equal(t, chunk.SnapshotID, got.SnapshotID)
	assert.Equal(t, chunk.ConnectionID, got.ConnectionID)
	assert.Equal(t, chunk.NamespaceID, got.NamespaceID)
	assert.Equal(t, chunk.ObjectType, got.ObjectType)
	assert.Equal(t, chunk.ObjectName, got.ObjectName)
	assert.JSONEq(t, string(chunk.SchemaJSON), string(got.SchemaJSON))
	assert.Equal(t, chunk.Content, got.Content)
	assert.Equal(t, chunk.Embedding, got.Embedding)
	assert.JSONEq(t, string(chunk.Metadata), string(got.Metadata))
}
