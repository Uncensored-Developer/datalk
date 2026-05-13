package schemas

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	schematesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/testing"
	schemaerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_RetrieveRelevantSchemaContext(t *testing.T) {
	t.Parallel()

	const connectionID int32 = 42

	testCases := []struct {
		name        string
		params      schematypes.RetrieveRelevantSchemaContextParams
		cfg         config.Config
		setupFn     func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient)
		assertErrFn func(t *testing.T, err error)
		assertFn    func(t *testing.T, got *schematypes.RetrievedSchemaContext)
	}{
		{
			name:   "empty query text",
			params: schematypes.RetrieveRelevantSchemaContextParams{ConnectionID: connectionID, QueryText: "   "},
			cfg:    config.Config{EmbeddingEnabled: true},
			setupFn: func(*testing.T, context.Context, *storagetesting.Storage, *schematesting.EmbeddingClient) {
			},
			assertErrFn: func(t *testing.T, err error) {
				require.EqualError(t, err, "query text cannot be empty")
			},
		},
		{
			name:   "embedding disabled",
			params: schematypes.RetrieveRelevantSchemaContextParams{ConnectionID: connectionID, QueryText: "how many users"},
			cfg:    config.Config{EmbeddingEnabled: false},
			setupFn: func(*testing.T, context.Context, *storagetesting.Storage, *schematesting.EmbeddingClient) {
			},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, schemaerrors.ErrEmbeddingDisabled)
			},
		},
		{
			name:   "snapshot not found",
			params: schematypes.RetrieveRelevantSchemaContextParams{ConnectionID: connectionID, QueryText: "how many users"},
			cfg:    config.Config{EmbeddingEnabled: true},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, _ *schematesting.EmbeddingClient) {
				mockStorage.On("ListSnapshots", ctx, expectedRetrievalSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{}, nil)
			},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, schemaerrors.ErrSnapshotNotFound)
			},
		},
		{
			name:   "embedded snapshot not ready",
			params: schematypes.RetrieveRelevantSchemaContextParams{ConnectionID: connectionID, QueryText: "how many users"},
			cfg:    config.Config{EmbeddingEnabled: true},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, _ *schematesting.EmbeddingClient) {
				snapshots := []*schematypes.Snapshot{
					{ID: 11, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC()},
					{ID: 10, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC().Add(-time.Hour)},
				}
				mockStorage.On("ListSnapshots", ctx, expectedRetrievalSnapshotsFilter(connectionID)).Return(snapshots, nil)
				mockStorage.On("GetEmbeddingJob", ctx, int32(11)).Return(&schematypes.EmbeddingJob{SnapshotID: 11, Status: schematypes.EmbeddingJobStatusProcessing}, nil).Once()
				mockStorage.On("GetEmbeddingJob", ctx, int32(10)).Return((*schematypes.EmbeddingJob)(nil), nil).Once()
			},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, schemaerrors.ErrEmbeddedSnapshotNotReady)
			},
		},
		{
			name: "success uses latest completed snapshot and dedupes chunks",
			params: schematypes.RetrieveRelevantSchemaContextParams{
				ConnectionID: connectionID,
				QueryText:    "how many users subscribed this month",
				Limit:        2,
			},
			cfg: config.Config{EmbeddingEnabled: true},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient) {
				snapshots := []*schematypes.Snapshot{
					{ID: 12, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC()},
					{ID: 11, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC().Add(-time.Hour)},
					{ID: 10, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC().Add(-2 * time.Hour)},
				}
				mockStorage.On("ListSnapshots", ctx, expectedRetrievalSnapshotsFilter(connectionID)).Return(snapshots, nil)
				mockStorage.On("GetEmbeddingJob", ctx, int32(12)).Return(&schematypes.EmbeddingJob{SnapshotID: 12, Status: schematypes.EmbeddingJobStatusProcessing}, nil).Once()
				mockStorage.On("GetEmbeddingJob", ctx, int32(11)).Return(&schematypes.EmbeddingJob{SnapshotID: 11, Status: schematypes.EmbeddingJobStatusCompleted}, nil).Once()
				mockClient.On("EmbedTexts", ctx, []string{"how many users subscribed this month"}).Return([][]float32{testEmbeddingVector(0.1)}, nil).Once()
				mockStorage.On("SearchChunks", ctx, storage.SearchChunksParams{
					SnapshotID:     11,
					QueryEmbedding: testEmbeddingVector(0.1),
					Limit:          4,
				}).Return([]*schematypes.RetrievedChunk{
					{
						ChunkID:    1,
						ObjectType: "table",
						ObjectName: "public.users",
						Content:    "users table",
						Similarity: 0.95,
					},
					{
						ChunkID:    2,
						ObjectType: "table",
						ObjectName: "public.users",
						Content:    "users table duplicate",
						Similarity: 0.94,
					},
					{
						ChunkID:    3,
						ObjectType: "table",
						ObjectName: "public.subscriptions",
						Content:    "subscriptions table",
						Similarity: 0.91,
					},
				}, nil).Once()
			},
			assertFn: func(t *testing.T, got *schematypes.RetrievedSchemaContext) {
				require.NotNil(t, got)
				assert.Equal(t, connectionID, got.ConnectionID)
				assert.Equal(t, int32(11), got.SnapshotID)
				assert.Equal(t, "nomic-embed-text", got.EmbeddingModel)
				assert.Equal(t, "how many users subscribed this month", got.QueryText)
				require.Len(t, got.Chunks, 2)
				assert.Equal(t, int64(1), got.Chunks[0].ChunkID)
				assert.Equal(t, "public.users", got.Chunks[0].ObjectName)
				assert.Equal(t, int64(3), got.Chunks[1].ChunkID)
				assert.Equal(t, "public.subscriptions", got.Chunks[1].ObjectName)
				assert.False(t, got.RetrievedAt.IsZero())
			},
		},
		{
			name: "empty retrieval result returns embedded snapshot not ready",
			params: schematypes.RetrieveRelevantSchemaContextParams{
				ConnectionID: connectionID,
				QueryText:    "group by week",
				Limit:        2,
			},
			cfg: config.Config{EmbeddingEnabled: true},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient) {
				snapshots := []*schematypes.Snapshot{
					{ID: 20, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC()},
				}
				mockStorage.On("ListSnapshots", ctx, expectedRetrievalSnapshotsFilter(connectionID)).Return(snapshots, nil)
				mockStorage.On("GetEmbeddingJob", ctx, int32(20)).Return(&schematypes.EmbeddingJob{SnapshotID: 20, Status: schematypes.EmbeddingJobStatusCompleted}, nil).Once()
				mockClient.On("EmbedTexts", ctx, []string{"group by week"}).Return([][]float32{testEmbeddingVector(0.2)}, nil).Once()
				mockStorage.On("SearchChunks", ctx, storage.SearchChunksParams{
					SnapshotID:     20,
					QueryEmbedding: testEmbeddingVector(0.2),
					Limit:          4,
				}).Return([]*schematypes.RetrievedChunk{}, nil).Once()
			},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, schemaerrors.ErrEmbeddedSnapshotNotReady)
			},
		},
		{
			name: "embedding query failure",
			params: schematypes.RetrieveRelevantSchemaContextParams{
				ConnectionID: connectionID,
				QueryText:    "how many users",
			},
			cfg: config.Config{EmbeddingEnabled: true},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient) {
				ollamaErr := errors.New("ollama down")
				snapshots := []*schematypes.Snapshot{
					{ID: 30, ConnectionID: connectionID, IntrospectedAt: time.Now().UTC()},
				}
				mockStorage.On("ListSnapshots", ctx, expectedRetrievalSnapshotsFilter(connectionID)).Return(snapshots, nil)
				mockStorage.On("GetEmbeddingJob", ctx, int32(30)).Return(&schematypes.EmbeddingJob{SnapshotID: 30, Status: schematypes.EmbeddingJobStatusCompleted}, nil).Once()
				mockClient.On("EmbedTexts", ctx, []string{"how many users"}).Return(nil, ollamaErr).Once()
			},
			assertErrFn: func(t *testing.T, err error) {
				require.EqualError(t, err, "failed to embed retrieval query: ollama down")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockStorage := storagetesting.NewStorage(t)
			mockClient := schematesting.NewEmbeddingClient(t)

			tc.setupFn(t, ctx, mockStorage, mockClient)

			service := &Service{
				Base:            newTestBaseWithConfig(tc.cfg),
				storage:         mockStorage,
				embeddingClient: mockClient,
			}

			got, err := service.RetrieveRelevantSchemaContext(ctx, tc.params)
			if tc.assertErrFn != nil {
				require.Error(t, err)
				tc.assertErrFn(t, err)
				return
			}

			require.NoError(t, err)
			if tc.assertFn != nil {
				tc.assertFn(t, got)
			}
		})
	}
}

func expectedRetrievalSnapshotsFilter(connectionID int32) storage.SnapshotsFilter {
	return storage.SnapshotsFilter{
		ConnectionID: []int32{connectionID},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByDesc(storage.SnapshotOrderingIntrospectedAt),
			ordering.OrderByDesc(storage.SnapshotOrderingID),
		},
	}
}

func TestDedupeRetrievedChunks(t *testing.T) {
	t.Parallel()

	got := dedupeRetrievedChunks([]*schematypes.RetrievedChunk{
		nil,
		{ChunkID: 1, ObjectName: "public.users"},
		{ChunkID: 2, ObjectName: "public.users"},
		{ChunkID: 3, ObjectName: "public.subscriptions"},
	}, 2)

	require.Len(t, got, 2)
	assert.Equal(t, int64(1), got[0].ChunkID)
	assert.Equal(t, int64(3), got[1].ChunkID)
}
