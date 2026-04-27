package schemas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock/dummy"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	introspectortesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector/testing"
	schematesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/testing"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_RefreshSchemaSnapshot(t *testing.T) {
	connectionID := int32(42)
	connection := connectiontypes.Connection{
		ID:       connectionID,
		Database: connectiontypes.DatabasePostgres,
		Metadata: connectiontypes.Metadata{
			IncludeNamespaces: []string{"public"},
			IncludeViews:      true,
		},
	}
	catalog := &introspector.Catalog{
		Kind:         introspector.DBPostgres,
		DiscoveredAt: time.Unix(1700000000, 0).UTC(),
		Namespaces: []introspector.Namespace{
			{
				Name: "public",
				Tables: []introspector.Table{
					{Name: "users"},
				},
			},
		},
	}
	schemaJSON := mustMarshalCatalog(t, catalog)
	schemaHash := hashSchema(schemaJSON)

	testCases := []struct {
		name        string
		modifyFn    func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector)
		expectError string
	}{
		{
			name: "get connection failure",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connectiontypes.Connection{}, errors.New("connection missing"))
			},
			expectError: "failed to get connection",
		},
		{
			name: "list snapshots failure",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return(nil, errors.New("storage down"))
			},
			expectError: "failed to list snapshots",
		},
		{
			name: "introspector factory failure",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{}, nil)
				mockFactory.On("ForConnection", ctx, connection).Return(nil, errors.New("factory down"))
			},
			expectError: "failed to introspector",
		},
		{
			name: "introspect failure",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{}, nil)
				mockFactory.On("ForConnection", ctx, connection).Return(mockIntrospector, nil)
				mockIntrospector.On("Introspect", ctx, toIntrospectOptions(connection.Metadata)).Return((*introspector.Catalog)(nil), errors.New("introspect failed"))
			},
			expectError: "failed to introspect",
		},
		{
			name: "insert snapshot failure",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{}, nil)
				mockFactory.On("ForConnection", ctx, connection).Return(mockIntrospector, nil)
				mockIntrospector.On("Introspect", ctx, toIntrospectOptions(connection.Metadata)).Return(catalog, nil)
				mockStorage.On("InsertSnapshot", ctx, mock.MatchedBy(func(snapshot *schematypes.Snapshot) bool {
					return snapshot != nil &&
						snapshot.ConnectionID == connectionID &&
						snapshot.SchemaHash == schemaHash &&
						string(snapshot.SchemaJSON) == string(schemaJSON) &&
						!snapshot.IntrospectedAt.IsZero()
				})).Return(errors.New("insert failed"))
			},
			expectError: "failed to insert snapshot",
		},
		{
			name: "success inserts when schema changed",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{
					{ID: 7, ConnectionID: connectionID, SchemaHash: "old-hash"},
				}, nil)
				mockFactory.On("ForConnection", ctx, connection).Return(mockIntrospector, nil)
				mockIntrospector.On("Introspect", ctx, toIntrospectOptions(connection.Metadata)).Return(catalog, nil)
				mockStorage.On("InsertSnapshot", ctx, mock.MatchedBy(func(snapshot *schematypes.Snapshot) bool {
					if snapshot == nil {
						return false
					}
					snapshot.ID = 99
					return snapshot.ConnectionID == connectionID &&
						snapshot.SchemaHash == schemaHash &&
						string(snapshot.SchemaJSON) == string(schemaJSON) &&
						!snapshot.IntrospectedAt.IsZero()
				})).Return(nil)
			},
		},
		{
			name: "success skips insert when schema unchanged",
			modifyFn: func(ctx context.Context, mockGetter *schematesting.ConnectionGetter, mockStorage *storagetesting.Storage, mockFactory *schematesting.IntrospectorFactory, mockIntrospector *introspectortesting.Introspector) {
				mockGetter.On("GetConnection", ctx, connectionID).Return(connection, nil)
				mockStorage.On("ListSnapshots", ctx, expectedSnapshotsFilter(connectionID)).Return([]*schematypes.Snapshot{
					{ID: 7, ConnectionID: connectionID, SchemaHash: schemaHash},
				}, nil)
				mockFactory.On("ForConnection", ctx, connection).Return(mockIntrospector, nil)
				mockIntrospector.On("Introspect", ctx, toIntrospectOptions(connection.Metadata)).Return(catalog, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()

			mockGetter := schematesting.NewConnectionGetter(t)
			mockStorage := storagetesting.NewStorage(t)
			mockFactory := schematesting.NewIntrospectorFactory(t)
			mockIntrospector := introspectortesting.NewIntrospector(t)

			tc.modifyFn(ctx, mockGetter, mockStorage, mockFactory, mockIntrospector)

			service := &Service{
				Base:                newTestBase(),
				locker:              dummy.NewDummyDistributedLocker(),
				connectionGetter:    mockGetter,
				storage:             mockStorage,
				introspectorFactory: mockFactory,
			}

			err := service.RefreshSchemaSnapshot(ctx, connectionID)
			if tc.expectError != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError)
				return
			}

			require.NoError(t, err)
		})
	}
}

func expectedSnapshotsFilter(connectionID int32) storage.SnapshotsFilter {
	return storage.SnapshotsFilter{
		ConnectionID: []int32{connectionID},
		Pagination: pagination.LimitOffsetPagination{
			Limit: 1,
		},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByDesc(storage.SnapshotOrderingID),
		},
	}
}

func mustMarshalCatalog(t *testing.T, catalog *introspector.Catalog) []byte {
	t.Helper()

	payload, err := json.Marshal(catalog)
	require.NoError(t, err)
	return payload
}

func hashSchema(payload []byte) string {
	hashBytes := sha256.Sum256(payload)
	return hex.EncodeToString(hashBytes[:])
}

func newTestBase() *base.Base {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return base.NewBase("schemas-core", logger)
}
