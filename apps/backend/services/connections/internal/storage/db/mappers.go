package db

import (
	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
)

func connectionToDB(connection *connections.Connection) *models.ConnectionSetter {
	return &models.ConnectionSetter{
		Name:      omit.From(connection.Name),
		Kind:      omit.From(string(connection.Database)),
		DSN:       omitnull.From(connection.DSN),
		IsEnabled: omit.From(connection.IsEnabled),
		UserID:    omit.From(connection.UserID),
		CreatedAt: omit.From(connection.CreatedAt),
	}
}

func connectionFromDB(dbConnection *models.Connection) (*connections.Connection, error) {
	return &connections.Connection{
		ID:        dbConnection.ID,
		UserID:    dbConnection.UserID,
		Name:      dbConnection.Name,
		Database:  connections.Database(dbConnection.Kind),
		DSN:       dbConnection.DSN.GetOrZero(),
		IsEnabled: dbConnection.IsEnabled,
		CreatedAt: dbConnection.CreatedAt,
	}, nil
}

func connectionsFromDB(dbConnections []*models.Connection) ([]*connections.Connection, error) {
	return slices.Map(dbConnections, connectionFromDB)
}

func accessToDB(access *connections.Access) *models.ConnectionAccessSetter {
	return &models.ConnectionAccessSetter{
		UserID:       omit.From(access.UserID),
		ConnectionID: omit.From(access.ConnectionID),
		CanQuery:     omit.From(access.CanQuery),
		AllowWrites:  omit.From(access.AllowWrites),
		CanManage:    omit.From(access.CanManage),
		GrantedAt:    omit.From(access.GrantedAt),
	}
}

func accessFromDB(dbAccess *models.ConnectionAccess) (*connections.Access, error) {
	return &connections.Access{
		UserID:       dbAccess.UserID,
		ConnectionID: dbAccess.ConnectionID,
		CanQuery:     dbAccess.CanQuery,
		AllowWrites:  dbAccess.AllowWrites,
		CanManage:    dbAccess.CanManage,
		GrantedAt:    dbAccess.GrantedAt,
	}, nil
}

func accessListFromDB(dbAccess []*models.ConnectionAccess) ([]*connections.Access, error) {
	return slices.Map(dbAccess, accessFromDB)
}

func namespaceToDB(namespace *connections.Namespace) *models.ConnectionNamespaceSetter {
	return &models.ConnectionNamespaceSetter{
		ConnectionID:  omit.From(namespace.ConnectionID),
		Name:          omit.From(namespace.Name),
		NamespaceType: omit.From(string(namespace.NamespaceType)),
		IsEnabled:     omit.From(namespace.IsEnabled),
		CreatedAt:     omit.From(namespace.CreatedAt),
	}
}

func namespaceFromDB(dbNamespace *models.ConnectionNamespace) (*connections.Namespace, error) {
	return &connections.Namespace{
		ID:            dbNamespace.ID,
		ConnectionID:  dbNamespace.ConnectionID,
		Name:          dbNamespace.Name,
		NamespaceType: connections.NamespaceType(dbNamespace.NamespaceType),
		IsEnabled:     dbNamespace.IsEnabled,
		CreatedAt:     dbNamespace.CreatedAt,
	}, nil
}

func namespacesFromDB(dbNamespaces []*models.ConnectionNamespace) ([]*connections.Namespace, error) {
	return slices.Map(dbNamespaces, namespaceFromDB)
}
