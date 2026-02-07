package db

import (
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/aarondl/opt/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_UpsertConnection(t *testing.T) {
	t.Parallel()

	t.Run("Inserting and Listing new connection", func(t *testing.T) {
		t.Parallel()

		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		newConnection := &connections.Connection{
			Name:      "test-connection",
			Database:  connections.DatabasePostgres,
			DSN:       "postgres://test-connection",
			UserID:    createdUser.ID,
			IsEnabled: true,
			CreatedAt: time.Now().UTC(),
		}
		err := s.UpsertConnection(t.Context(), newConnection)
		require.NoError(t, err)
		assert.NotZero(t, newConnection.ID)
		assert.NotEmpty(t, newConnection.CreatedAt)

		gotConnections, err := s.ListConnections(t.Context(), storage.ListConnectionsParam{ID: []int32{newConnection.ID}})
		require.NoError(t, err)
		require.Len(t, gotConnections, 1)
		assert.Equal(t, newConnection.ID, gotConnections[0].ID)
		assert.Equal(t, newConnection.UserID, gotConnections[0].UserID)
	})

	t.Run("Updating existing connection", func(t *testing.T) {
		t.Parallel()

		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		connTmpl := factory.ConnectionTemplate{}
		connTmpl.Apply(t.Context(),
			factory.ConnectionMods.Name("primary-connection"),
			factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
			factory.ConnectionMods.DSN(null.From("postgres://seed")),
		)
		connTmpl.Apply(t.Context(), factory.ConnectionMods.UserID(createdUser.ID))
		createdConn := connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		connection, err := connectionFromDB(createdConn)
		require.NoError(t, err)
		connection.Name = "primary-connection-updated"
		connection.IsEnabled = false
		connection.DSN = "postgres://updated"
		connection.UserID = createdUser.ID

		err = s.UpsertConnection(t.Context(), connection)
		require.NoError(t, err)

		assert.Equal(t, "primary-connection-updated", connection.Name)
		assert.False(t, connection.IsEnabled)
		assert.Equal(t, "postgres://updated", connection.DSN)
		assert.Equal(t, createdUser.ID, connection.UserID)
	})
}

func TestStorage_UpsertAccess(t *testing.T) {
	t.Parallel()

	t.Run("Inserting and Listing new access", func(t *testing.T) {
		t.Parallel()

		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		connTmpl := factory.ConnectionTemplate{}
		connTmpl.Apply(t.Context(),
			factory.ConnectionMods.Name("test-access-connection"),
			factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
			factory.ConnectionMods.DSN(null.From("postgres://access")),
			factory.ConnectionMods.UserID(createdUser.ID),
		)
		createdConn := connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		newAccess := &connections.Access{
			UserID:       createdUser.ID,
			ConnectionID: createdConn.ID,
			CanQuery:     true,
			AllowWrites:  false,
			CanManage:    false,
			GrantedAt:    time.Now().UTC(),
		}
		err := s.UpsertAccess(t.Context(), newAccess)
		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, newAccess.UserID)
		assert.Equal(t, createdConn.ID, newAccess.ConnectionID)
		assert.False(t, newAccess.GrantedAt.IsZero())

		gotAccess, err := s.ListAccess(t.Context(), storage.ListAccessParam{UserID: []int32{createdUser.ID}})
		require.NoError(t, err)
		require.Len(t, gotAccess, 1)
		assert.Equal(t, newAccess.UserID, gotAccess[0].UserID)
		assert.Equal(t, newAccess.ConnectionID, gotAccess[0].ConnectionID)
	})

	t.Run("Updating existing access", func(t *testing.T) {
		t.Parallel()

		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		connTmpl := factory.ConnectionTemplate{}
		connTmpl.Apply(t.Context(),
			factory.ConnectionMods.Name("update-access-connection"),
			factory.ConnectionMods.Kind(string(connections.DatabasePostgres)),
			factory.ConnectionMods.DSN(null.From("postgres://access-update")),
			factory.ConnectionMods.UserID(createdUser.ID),
		)
		createdConn := connTmpl.CreateOrFail(t.Context(), t, runner.BobConn)

		access := &connections.Access{
			UserID:       createdUser.ID,
			ConnectionID: createdConn.ID,
			CanQuery:     true,
			AllowWrites:  false,
			CanManage:    false,
			GrantedAt:    time.Now().UTC(),
		}
		err := s.UpsertAccess(t.Context(), access)
		require.NoError(t, err)
		originalGrantedAt := access.GrantedAt

		access.CanQuery = false
		access.AllowWrites = true
		access.CanManage = true
		err = s.UpsertAccess(t.Context(), access)
		require.NoError(t, err)

		assert.False(t, access.CanQuery)
		assert.True(t, access.AllowWrites)
		assert.True(t, access.CanManage)
		assert.Equal(t, originalGrantedAt, access.GrantedAt)
	})
}
