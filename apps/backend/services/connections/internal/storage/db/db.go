package db

import (
	"context"
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/info"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/mdobak/go-xerrors"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/im"
)

type Storage struct {
	*common.Storage
}

func NewStorage(conn *sql.DB) *Storage {
	return &Storage{
		common.NewStorage("connections", conn),
	}
}

func (s *Storage) UpsertConnection(ctx context.Context, connection *connections.Connection) error {
	connectionSetter := connectionToDB(connection)
	dbConnection, err := models.Connections.Insert(
		connectionSetter,
		im.OnConflict(info.Connections.Columns.Name.Name).DoUpdate(
			im.SetExcluded(
				info.Connections.Columns.Kind.Name,
				info.Connections.Columns.DSN.Name,
				info.Connections.Columns.IsEnabled.Name,
				info.Connections.Columns.UserID.Name,
			),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	upsertedConnection, err := connectionFromDB(dbConnection)
	if err != nil {
		return xerrors.Newf("failed to map db connection: %w", err)
	}

	connection.ID = upsertedConnection.ID
	connection.UserID = upsertedConnection.UserID
	connection.Name = upsertedConnection.Name
	connection.Database = upsertedConnection.Database
	connection.DSN = upsertedConnection.DSN
	connection.IsEnabled = upsertedConnection.IsEnabled
	connection.CreatedAt = upsertedConnection.CreatedAt
	return nil
}

func (s *Storage) ListConnections(ctx context.Context, params storage.ListConnectionsParam) ([]*connections.Connection, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(params.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.Connections.ID.In(params.ID...))
	}

	if len(params.Name) > 0 {
		queryMods = append(queryMods, models.SelectWhere.Connections.Name.In(params.Name...))
	}

	if len(params.UserID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.Connections.UserID.In(params.UserID...))
	}

	fetchedConnections, err := models.Connections.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err != nil {
		return nil, err
	}

	return connectionsFromDB(fetchedConnections)
}

func (s *Storage) UpsertAccess(ctx context.Context, access *connections.Access) error {
	accessSetter := accessToDB(access)
	dbAccess, err := models.ConnectionAccesses.Insert(
		accessSetter,
		im.OnConflict(
			info.ConnectionAccesses.Columns.UserID.Name,
			info.ConnectionAccesses.Columns.ConnectionID.Name,
		).DoUpdate(
			im.SetExcluded(
				info.ConnectionAccesses.Columns.CanQuery.Name,
				info.ConnectionAccesses.Columns.AllowWrites.Name,
				info.ConnectionAccesses.Columns.CanManage.Name,
			),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	upsertedAccess, err := accessFromDB(dbAccess)
	if err != nil {
		return xerrors.Newf("failed to map db access: %w", err)
	}

	access.UserID = upsertedAccess.UserID
	access.ConnectionID = upsertedAccess.ConnectionID
	access.CanQuery = upsertedAccess.CanQuery
	access.AllowWrites = upsertedAccess.AllowWrites
	access.CanManage = upsertedAccess.CanManage
	access.GrantedAt = upsertedAccess.GrantedAt
	return nil
}

func (s *Storage) ListAccess(ctx context.Context, params storage.ListAccessParam) ([]*connections.Access, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(params.UserID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ConnectionAccesses.UserID.In(params.UserID...))
	}

	if len(params.ConnectionID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ConnectionAccesses.ConnectionID.In(params.ConnectionID...))
	}

	fetchedAccess, err := models.ConnectionAccesses.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err != nil {
		return nil, err
	}

	return accessListFromDB(fetchedAccess)
}

func (s *Storage) UpsertNamespace(ctx context.Context, namespace *connections.Namespace) error {
	namespaceSetter := namespaceToDB(namespace)
	dbNamespace, err := models.ConnectionNamespaces.Insert(
		namespaceSetter,
		im.OnConflict(
			info.ConnectionNamespaces.Columns.ConnectionID.Name,
			info.ConnectionNamespaces.Columns.NamespaceType.Name,
			info.ConnectionNamespaces.Columns.Name.Name,
		).DoUpdate(
			im.SetExcluded(
				info.ConnectionNamespaces.Columns.IsEnabled.Name,
			),
		),
	).One(ctx, s.Executor(ctx))
	if err != nil {
		return err
	}

	upsertedNamespace, err := namespaceFromDB(dbNamespace)
	if err != nil {
		return xerrors.Newf("failed to map db namespace: %w", err)
	}

	namespace.ID = upsertedNamespace.ID
	namespace.ConnectionID = upsertedNamespace.ConnectionID
	namespace.Name = upsertedNamespace.Name
	namespace.NamespaceType = upsertedNamespace.NamespaceType
	namespace.IsEnabled = upsertedNamespace.IsEnabled
	namespace.CreatedAt = upsertedNamespace.CreatedAt
	return nil
}

func (s *Storage) ListNamespace(ctx context.Context, params storage.ListNamespaceParam) ([]*connections.Namespace, error) {
	var queryMods []bob.Mod[*dialect.SelectQuery]

	if len(params.ID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ConnectionNamespaces.ID.In(params.ID...))
	}

	if len(params.ConnectionID) > 0 {
		queryMods = append(queryMods, models.SelectWhere.ConnectionNamespaces.ConnectionID.In(params.ConnectionID...))
	}

	fetchedNamespaces, err := models.ConnectionNamespaces.Query(queryMods...).All(ctx, s.Executor(ctx))
	if err != nil {
		return nil, err
	}

	return namespacesFromDB(fetchedNamespaces)
}
