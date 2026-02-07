package connections

import (
	"errors"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateConnection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		newConnection NewConnection
		modifyFn      func(newConnection NewConnection, mockStorage *storagetesting.Storage)
		expect        *connections.Connection
		expectError   error
	}{
		{
			name: "invalid: empty name",
			newConnection: NewConnection{
				Name:     "",
				Database: connections.DatabasePostgres,
				DSN:      "postgres://test",
				UserID:   1,
			},
			expectError: errors.New("name is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: empty database",
			newConnection: NewConnection{
				Name:     "analytics",
				Database: "",
				DSN:      "postgres://test",
				UserID:   2,
			},
			expectError: errors.New("database is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: unknown database",
			newConnection: NewConnection{
				Name:     "analytics",
				Database: connections.Database("oracle"),
				DSN:      "postgres://test",
				UserID:   2,
			},
			expectError: errors.New("database is invalid"),
			modifyFn:    nil,
		},
		{
			name: "invalid: missing user id",
			newConnection: NewConnection{
				Name:     "analytics",
				Database: connections.DatabasePostgres,
				DSN:      "postgres://test",
				UserID:   0,
			},
			expectError: errors.New("user id is required"),
			modifyFn:    nil,
		},
		{
			name: "storage failure",
			newConnection: NewConnection{
				Name:     "analytics",
				Database: connections.DatabaseMySQL,
				DSN:      "mysql://test",
				UserID:   10,
			},
			modifyFn: func(newConnection NewConnection, mockStorage *storagetesting.Storage) {
				matcher := mock.MatchedBy(func(c *connections.Connection) bool {
					return c != nil &&
						c.Name == newConnection.Name &&
						c.Database == newConnection.Database &&
						c.DSN == newConnection.DSN &&
						c.UserID == newConnection.UserID &&
						c.IsEnabled
				})
				mockStorage.
					On("UpsertConnection", mock.Anything, matcher).
					Return(errors.New("db write failed"))
			},
			expectError: errors.New("failed to insert connection"),
		},
		{
			name: "success",
			newConnection: NewConnection{
				Name:     "warehouse",
				Database: connections.DatabasePostgres,
				DSN:      "postgres://warehouse",
				UserID:   7,
			},
			modifyFn: func(newConnection NewConnection, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("UpsertConnection", mock.Anything, mock.MatchedBy(func(c *connections.Connection) bool {
						return c != nil &&
							c.Name == newConnection.Name &&
							c.Database == newConnection.Database &&
							c.DSN == newConnection.DSN &&
							c.UserID == newConnection.UserID &&
							c.IsEnabled && !c.CreatedAt.IsZero()
					})).
					Return(nil)
			},
			expect: &connections.Connection{
				Name:      "warehouse",
				Database:  connections.DatabasePostgres,
				DSN:       "postgres://warehouse",
				UserID:    7,
				IsEnabled: true,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.newConnection, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.CreateConnection(t.Context(), tc.newConnection)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
				return
			}

			require.NoError(t, err)
			if tc.expect != nil {
				assert.Equal(t, tc.expect.Name, got.Name)
				assert.Equal(t, tc.expect.Database, got.Database)
				assert.Equal(t, tc.expect.DSN, got.DSN)
				assert.Equal(t, tc.expect.UserID, got.UserID)
				assert.True(t, got.IsEnabled)
				assert.False(t, got.CreatedAt.IsZero())
			}
		})
	}
}
