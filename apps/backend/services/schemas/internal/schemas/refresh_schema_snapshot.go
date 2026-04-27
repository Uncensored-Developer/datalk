package schemas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/ordering"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/pagination"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/events"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
	"github.com/mdobak/go-xerrors"
)

var databaseToDBKind = map[connectiontypes.Database]introspector.DBKind{
	connectiontypes.DatabasePostgres: introspector.DBPostgres,
	connectiontypes.DatabaseMySQL:    introspector.DBMySQL,
	connectiontypes.DatabaseCQL:      introspector.DBCQL,
}

func (s *Service) RefreshSchemaSnapshot(ctx context.Context, connectionID int32) error {
	key := fmt.Sprintf("schemas-core:refresh-schema-snapshot:%d", connectionID)
	lock, err := s.locker.Lock(ctx, []string{key}, distlock.LockOptions{Wait: true})
	if err != nil {
		return xerrors.Newf("failed to acquire lock: %w", err)
	}
	defer lock.Unlock()

	connection, err := s.connectionGetter.GetConnection(ctx, connectionID)
	if err != nil {
		return xerrors.Newf("failed to get connection: %w", err)
	}

	snapshots, err := s.storage.ListSnapshots(ctx, storage.SnapshotsFilter{
		ConnectionID: []int32{connectionID},
		Pagination: pagination.LimitOffsetPagination{
			Limit: 1,
		},
		Ordering: ordering.Orderings[storage.SnapshotOrdering]{
			ordering.OrderByDesc(storage.SnapshotOrderingID),
		},
	})
	if err != nil {
		return xerrors.Newf("failed to list snapshots: %w", err)
	}

	var snapshot schematypes.Snapshot

	if len(snapshots) > 0 {
		snapshot = *snapshots[0]
	}

	dbIntrospector, err := s.introspectorFactory.ForConnection(ctx, connection)
	if err != nil {
		return xerrors.Newf("failed to introspector: %w", err)
	}

	catalog, err := dbIntrospector.Introspect(ctx, toIntrospectOptions(connection.Metadata))
	if err != nil {
		return xerrors.Newf("failed to introspect: %w", err)
	}

	schemaJson, err := json.Marshal(catalog)
	if err != nil {
		return xerrors.Newf("failed to marshal catalog: %w", err)
	}
	schemaHashBytes := sha256.Sum256(schemaJson)
	schemaHash := hex.EncodeToString(schemaHashBytes[:])

	if schemaHash != snapshot.SchemaHash {
		s.Logger().Info("Schema changed, updating schema")
		// Schema has changed or snapshot not found
		newSnapshot := &schematypes.Snapshot{
			ConnectionID:   connectionID,
			SchemaHash:     schemaHash,
			SchemaJSON:     schemaJson,
			IntrospectedAt: time.Now(),
		}
		err = s.storage.InsertSnapshot(ctx, newSnapshot)
		if err != nil {
			return xerrors.Newf("failed to insert snapshot: %w", err)
		}

		err = events.SendSnapshotCreated(ctx, events.SnapshotCreatedContent{
			SnapshotID: newSnapshot.ID,
		})
		if err != nil {
			return xerrors.Newf("failed to send snapshot created event: %w", err)
		}
	}

	return nil
}

func toIntrospectOptions(metadata connectiontypes.Metadata) introspector.IntrospectOptions {
	return introspector.IntrospectOptions{
		IncludeNamespaces:        metadata.IncludeNamespaces,
		ExcludeNamespaces:        metadata.ExcludeNamespaces,
		IncludeTablesByNamespace: metadata.IncludeTablesByNamespace,
		ExcludeTablesByNamespace: metadata.ExcludeTablesByNamespace,
		IncludeViews:             metadata.IncludeViews,
		IncludeIndexes:           metadata.IncludeIndexes,
		IncludeForeignKeys:       metadata.IncludeForeignKeys,
		IncludeComments:          metadata.IncludeComments,
	}
}
