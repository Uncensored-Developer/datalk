package db

import (
	"encoding/json"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/models"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/slices"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob/types"
)

func connectionToDB(connection *connections.Connection) (*models.ConnectionSetter, error) {
	dbConnection := &models.ConnectionSetter{
		Name:      omit.From(connection.Name),
		Kind:      omit.From(string(connection.Database)),
		DSN:       omitnull.From(connection.DSN),
		IsEnabled: omit.From(connection.IsEnabled),
		UserID:    omit.From(connection.UserID),
		CreatedAt: omit.From(connection.CreatedAt),
	}

	metadataBytes, err := json.Marshal(connection.Metadata)
	if err != nil {
		return nil, err
	}
	dbConnection.Metadata = omit.From(types.NewJSON[json.RawMessage](metadataBytes))

	return dbConnection, nil
}

func connectionFromDB(dbConnection *models.Connection) (*connections.Connection, error) {
	connection := &connections.Connection{
		ID:        dbConnection.ID,
		UserID:    dbConnection.UserID,
		Name:      dbConnection.Name,
		Database:  connections.Database(dbConnection.Kind),
		DSN:       dbConnection.DSN.GetOrZero(),
		IsEnabled: dbConnection.IsEnabled,
		CreatedAt: dbConnection.CreatedAt,
	}

	err := json.Unmarshal(dbConnection.Metadata.Val, &connection.Metadata)
	if err != nil {
		return nil, err
	}
	return connection, nil
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
