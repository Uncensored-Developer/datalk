package users

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage/testing"
	hashertesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users/hashers/testing"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_Setup(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		modifyFn  func(mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expectErr error
	}{
		{
			name: "rejects when users exist",
			modifyFn: func(mockStorage *storagetesting.Storage, _ *hashertesting.Hasher) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{}).
					Return([]*usertypes.User{{ID: 1}}, nil).
					Once()
			},
			expectErr: serviceerrors.ErrSetupUnavailable,
		},
		{
			name: "maps concurrent owner insert to unavailable",
			modifyFn: func(mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{}).
					Return([]*usertypes.User{}, nil).
					Once()
				mockHasher.On("Hash", mock.Anything, "secret").Return("hash", nil).Once()
				mockStorage.On("UpsertUser", mock.Anything, mock.MatchedBy(func(user *usertypes.User) bool {
					return user != nil && user.Role == usertypes.RoleOwner && !user.MustChangePassword
				})).Return(storage.ErrOwnerAlreadyExists).Once()
			},
			expectErr: serviceerrors.ErrSetupUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockHasher := hashertesting.NewHasher(t)
			tc.modifyFn(mockStorage, mockHasher)

			service := newAuthTestService(mockStorage, mockHasher)
			session, err := service.Setup(t.Context(), NewUser{Name: "Root", Email: "root@example.com", Password: "secret"})

			require.ErrorIs(t, err, tc.expectErr)
			assert.Nil(t, session)
			mockStorage.AssertNotCalled(t, "InsertRefreshToken", mock.Anything, mock.Anything)
		})
	}
}
