package schemas

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/dummy"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	schematesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/testing"
	schemaerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/errors"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_EmbedSnapshotContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		snapshotID  int32
		cfg         config.Config
		setupFn     func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient)
		assertErrFn func(t *testing.T, err error)
	}{
		{
			name:       "embedding disabled",
			snapshotID: 10,
			cfg:        config.Config{EmbeddingEnabled: false},
			setupFn:    func(*testing.T, context.Context, *storagetesting.Storage, *schematesting.EmbeddingClient) {},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, schemaerrors.ErrEmbeddingDisabled)
			},
		},
		{
			name:       "snapshot not found",
			snapshotID: 10,
			cfg:        config.Config{EmbeddingEnabled: true, EmbeddingBatchSize: 2},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, _ *schematesting.EmbeddingClient) {
				mockStorage.On("ListSnapshots", ctx, storage.SnapshotsFilter{ID: []int32{10}}).Return([]*schematypes.Snapshot{}, nil)
			},
			assertErrFn: func(t *testing.T, err error) {
				require.ErrorIs(t, err, storage.ErrSnapshotNotFound)
			},
		},
		{
			name:       "success",
			snapshotID: 77,
			cfg:        config.Config{EmbeddingEnabled: true, EmbeddingBatchSize: 2},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient) {
				snapshot := &schematypes.Snapshot{
					ID:           77,
					ConnectionID: 42,
					SchemaHash:   "schema-hash",
					SchemaJSON:   []byte(`{"kind":"postgres","namespaces":[{"name":"public","tables":[{"name":"users","comment":"Application users","columns":[{"name":"id","data_type":"uuid","nullable":false,"position":1}]}]}]}`),
				}

				mockStorage.On("ListSnapshots", ctx, storage.SnapshotsFilter{ID: []int32{snapshot.ID}}).Return([]*schematypes.Snapshot{snapshot}, nil)

				mockStorage.On("GetEmbeddingJob", ctx, snapshot.ID).Return((*schematypes.EmbeddingJob)(nil), nil)

				mockStorage.On("UpsertEmbeddingJob", ctx, mock.MatchedBy(func(job *schematypes.EmbeddingJob) bool {
					require.NotNil(t, job)
					if job.Status != schematypes.EmbeddingJobStatusProcessing {
						return false
					}

					assert.Equal(t, snapshot.ID, job.SnapshotID)
					assert.EqualValues(t, 0, job.RetryCount)
					assert.Nil(t, job.ErrorMessage)
					assert.Nil(t, job.CompletedAt)
					return true
				})).Return(nil).Once()

				mockClient.On("EmbedTexts", ctx, mock.MatchedBy(func(inputs []string) bool {
					require.Len(t, inputs, 1)

					assert.NotEmpty(t, inputs[0])
					return true
				})).Return([][]float32{testEmbeddingVector(0.1)}, nil).Once()

				mockStorage.On("ReplaceChunks", ctx, snapshot.ID, mock.MatchedBy(func(chunks []*schematypes.Chunk) bool {
					require.Len(t, chunks, 1)

					chunk := chunks[0]
					assert.Equal(t, "table", chunk.ObjectType)
					assert.Equal(t, "public.users", chunk.ObjectName)
					assert.Equal(t, snapshot.ConnectionID, chunk.ConnectionID)
					assert.Len(t, chunk.Embedding, 768)

					var metadata map[string]any
					if err := json.Unmarshal(chunk.Metadata, &metadata); err != nil {
						assert.NoError(t, err)
						return false
					}

					assert.Equal(t, "public", metadata["namespace"])
					assert.Equal(t, "public.users", metadata["qualified_name"])
					assert.Equal(t, "nomic-embed-text", metadata["model"])

					return true
				})).Return(nil).Once()

				mockStorage.On("UpsertEmbeddingJob", ctx, mock.MatchedBy(func(job *schematypes.EmbeddingJob) bool {
					require.NotNil(t, job)
					if job.Status != schematypes.EmbeddingJobStatusCompleted {
						return false
					}

					assert.Equal(t, snapshot.ID, job.SnapshotID)
					assert.EqualValues(t, 0, job.RetryCount)
					assert.Nil(t, job.ErrorMessage)
					assert.NotNil(t, job.CompletedAt)
					return true
				})).Return(nil).Once()
			},
		},
		{
			name:       "failed embed marks job failed",
			snapshotID: 88,
			cfg:        config.Config{EmbeddingEnabled: true, EmbeddingBatchSize: 2},
			setupFn: func(t *testing.T, ctx context.Context, mockStorage *storagetesting.Storage, mockClient *schematesting.EmbeddingClient) {
				ollamaErr := errors.New("ollama down")
				snapshot := &schematypes.Snapshot{
					ID:           88,
					ConnectionID: 50,
					SchemaHash:   "schema-hash",
					SchemaJSON:   []byte(`{"kind":"postgres","namespaces":[{"name":"public","tables":[{"name":"users","columns":[{"name":"id","data_type":"uuid","nullable":false,"position":1}]}]}]}`),
				}
				existingJob := &schematypes.EmbeddingJob{
					SnapshotID: snapshot.ID,
					Status:     schematypes.EmbeddingJobStatusFailed,
					RetryCount: 2,
					StartedAt:  time.Now().Add(-time.Hour),
				}

				mockStorage.On("ListSnapshots", ctx, storage.SnapshotsFilter{ID: []int32{snapshot.ID}}).Return([]*schematypes.Snapshot{snapshot}, nil)

				mockStorage.On("GetEmbeddingJob", ctx, snapshot.ID).Return(existingJob, nil)

				mockStorage.On("UpsertEmbeddingJob", ctx, mock.MatchedBy(func(job *schematypes.EmbeddingJob) bool {
					if job == nil || job.Status != schematypes.EmbeddingJobStatusProcessing {
						return false
					}
					assert.Equal(t, schematypes.EmbeddingJobStatusProcessing, job.Status)
					assert.Equal(t, existingJob.RetryCount, job.RetryCount)
					return true
				})).Return(nil).Once()

				mockClient.On("EmbedTexts", ctx, mock.Anything).Return(nil, ollamaErr).Once()

				mockStorage.On("UpsertEmbeddingJob", ctx, mock.MatchedBy(func(job *schematypes.EmbeddingJob) bool {
					require.NotNil(t, job)
					if job.Status != schematypes.EmbeddingJobStatusFailed {
						return false
					}

					assert.EqualValues(t, 3, job.RetryCount)
					if job.ErrorMessage == nil {
						assert.NotNil(t, job.ErrorMessage)
						return false
					}
					assert.Equal(t, "failed to embed chunks: ollama down", *job.ErrorMessage)
					return true
				})).Return(nil).Once()
			},
			assertErrFn: func(t *testing.T, err error) {
				require.EqualError(t, err, "failed to embed chunks: ollama down")
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
				locker:          dummy.NewDummyDistributedLocker(),
				storage:         mockStorage,
				embeddingClient: mockClient,
			}

			err := service.EmbedSnapshotContent(ctx, tc.snapshotID)
			if tc.assertErrFn != nil {
				require.Error(t, err)
				tc.assertErrFn(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func newTestBaseWithConfig(cfg config.Config) *base.Base {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return base.NewBase("schemas-core", logger, cfg)
}

func testEmbeddingVector(value float32) []float32 {
	vector := make([]float32, 768)
	for i := range vector {
		vector[i] = value
	}
	return vector
}
