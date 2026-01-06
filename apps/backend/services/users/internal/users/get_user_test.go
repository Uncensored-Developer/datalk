package users

import (
	"errors"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage/testing"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetUser(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	expectedUser := &users.User{
		ID:        42,
		Email:     "jane@example.com",
		Name:      "Jane Doe",
		Role:      users.RoleMember,
		IsActive:  true,
		CreatedAt: now,
	}

	testCases := []struct {
		name        string
		userID      int64
		modifyFn    func(userID int64, mockStorage *storagetesting.Storage)
		expect      *users.User
		expectError error
	}{
		{
			name:   "storage failure",
			userID: 7,
			modifyFn: func(userID int64, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int64{userID}}).
					Return(nil, errors.New("db offline"))
			},
			expectError: errors.New("failed to list users"),
		},
		{
			name:   "user not found",
			userID: 99,
			modifyFn: func(userID int64, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int64{userID}}).
					Return([]*users.User{}, nil)
			},
			expectError: serviceerrors.ErrUserNotFound,
		},
		{
			name:   "success: returns first result",
			userID: expectedUser.ID,
			modifyFn: func(userID int64, mockStorage *storagetesting.Storage) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int64{userID}}).
					Return([]*users.User{expectedUser}, nil)
			},
			expect: expectedUser,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			if tc.modifyFn != nil {
				tc.modifyFn(tc.userID, mockStorage)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
			}

			got, err := service.GetUser(t.Context(), tc.userID)
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
