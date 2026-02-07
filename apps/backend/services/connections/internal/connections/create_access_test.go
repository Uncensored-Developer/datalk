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

func TestService_CreateAccess(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		newAccess   NewAccess
		modifyFn    func(newAccess NewAccess, mockStorage *storagetesting.Storage)
		expect      *connections.Access
		expectError error
	}{
		{
			name: "invalid: missing user id",
			newAccess: NewAccess{
				UserID:       0,
				ConnectionID: 10,
				CanQuery:     true,
			},
			expectError: errors.New("user id is required"),
		},
		{
			name: "invalid: missing connection id",
			newAccess: NewAccess{
				UserID:       2,
				ConnectionID: 0,
				CanQuery:     true,
			},
			expectError: errors.New("connection id is required"),
		},
		{
			name: "storage failure",
			newAccess: NewAccess{
				UserID:       2,
				ConnectionID: 3,
				CanQuery:     true,
				AllowWrites:  false,
				CanManage:    false,
			},
			modifyFn: func(newAccess NewAccess, mockStorage *storagetesting.Storage) {
				matcher := mock.MatchedBy(func(a *connections.Access) bool {
					return a != nil &&
						a.UserID == newAccess.UserID &&
						a.ConnectionID == newAccess.ConnectionID &&
						a.CanQuery == newAccess.CanQuery &&
						a.AllowWrites == newAccess.AllowWrites &&
						a.CanManage == newAccess.CanManage
				})
				mockStorage.
					On("UpsertAccess", mock.Anything, matcher).
					Return(errors.New("db write failed"))
			},
			expectError: errors.New("failed to insert access"),
		},
		{
			name: "success",
			newAccess: NewAccess{
				UserID:       5,
				ConnectionID: 7,
				CanQuery:     true,
				AllowWrites:  true,
				CanManage:    false,
			},
			modifyFn: func(newAccess NewAccess, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("UpsertAccess", mock.Anything, mock.MatchedBy(func(a *connections.Access) bool {
						return a != nil &&
							a.UserID == newAccess.UserID &&
							a.ConnectionID == newAccess.ConnectionID &&
							a.CanQuery == newAccess.CanQuery &&
							a.AllowWrites == newAccess.AllowWrites &&
							a.CanManage == newAccess.CanManage && !a.GrantedAt.IsZero()
					})).
					Return(nil)
			},
			expect: &connections.Access{
				UserID:       5,
				ConnectionID: 7,
				CanQuery:     true,
				AllowWrites:  true,
				CanManage:    false,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.newAccess, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.CreateAccess(t.Context(), tc.newAccess)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
				return
			}

			require.NoError(t, err)
			if tc.expect != nil {
				assert.Equal(t, tc.expect.UserID, got.UserID)
				assert.Equal(t, tc.expect.ConnectionID, got.ConnectionID)
				assert.Equal(t, tc.expect.CanQuery, got.CanQuery)
				assert.Equal(t, tc.expect.AllowWrites, got.AllowWrites)
				assert.Equal(t, tc.expect.CanManage, got.CanManage)
				assert.False(t, got.GrantedAt.IsZero())
			}
		})
	}
}
