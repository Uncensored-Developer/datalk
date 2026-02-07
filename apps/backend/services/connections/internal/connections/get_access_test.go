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

func TestService_GetAccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	expectedAccess := &connections.Access{
		UserID:       4,
		ConnectionID: 9,
		CanQuery:     true,
		AllowWrites:  false,
		CanManage:    true,
		GrantedAt:    now,
	}

	testCases := []struct {
		name         string
		userID       int32
		connectionID int32
		modifyFn     func(userID int32, connectionID int32, mockStorage *storagetesting.Storage)
		expect       *connections.Access
		expectError  error
	}{
		{
			name:         "storage failure",
			userID:       1,
			connectionID: 2,
			modifyFn: func(userID int32, connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListAccess", mock.Anything, storage.ListAccessParam{UserID: []int32{userID}, ConnectionID: []int32{connectionID}}).
					Return(nil, errors.New("db offline"))
			},
			expectError: errors.New("failed to list access"),
		},
		{
			name:         "access not found",
			userID:       3,
			connectionID: 4,
			modifyFn: func(userID int32, connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListAccess", mock.Anything, storage.ListAccessParam{UserID: []int32{userID}, ConnectionID: []int32{connectionID}}).
					Return([]*connections.Access{}, nil)
			},
			expectError: serviceerrors.ErrAccessNotFound,
		},
		{
			name:         "success: returns first result",
			userID:       expectedAccess.UserID,
			connectionID: expectedAccess.ConnectionID,
			modifyFn: func(userID int32, connectionID int32, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListAccess", mock.Anything, storage.ListAccessParam{UserID: []int32{userID}, ConnectionID: []int32{connectionID}}).
					Return([]*connections.Access{expectedAccess}, nil)
			},
			expect: expectedAccess,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.userID, tc.connectionID, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.GetAccess(t.Context(), tc.userID, tc.connectionID)
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
