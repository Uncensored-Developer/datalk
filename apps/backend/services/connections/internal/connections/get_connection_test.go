package connections

import (
	"errors"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetConnection(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	expectedConnection := &connections.Connection{
		ID:        42,
		UserID:    7,
		Name:      "warehouse",
		Database:  connections.DatabasePostgres,
		DSN:       "postgres://warehouse",
		IsEnabled: true,
		CreatedAt: now,
	}

	testCases := []struct {
		name         string
		connectionID int32
		modifyFn     func(connectionID int32, mockStorage *storagetesting.Storage)
		expect       *connections.Connection
		expectError  error
	}{
		{
			name:         "storage failure",
			connectionID: 7,
			modifyFn: func(connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListConnections", mock.Anything, storage.ListConnectionsParam{ID: []int32{connectionID}}).
					Return(nil, errors.New("db offline"))
			},
			expectError: errors.New("failed to list connections"),
		},
		{
			name:         "connection not found",
			connectionID: 99,
			modifyFn: func(connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListConnections", mock.Anything, storage.ListConnectionsParam{ID: []int32{connectionID}}).
					Return([]*connections.Connection{}, nil)
			},
			expectError: serviceerrors.ErrConnectionNotFound,
		},
		{
			name:         "success: returns first result",
			connectionID: expectedConnection.ID,
			modifyFn: func(connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListConnections", mock.Anything, storage.ListConnectionsParam{ID: []int32{connectionID}}).
					Return([]*connections.Connection{expectedConnection}, nil)
			},
			expect: expectedConnection,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.connectionID, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.GetConnection(t.Context(), tc.connectionID)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tc.expect, got)
		})
	}
}
