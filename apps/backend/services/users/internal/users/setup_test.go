package users

import (
	"testing"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
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

	t.Run("allows setup when no owner or admin exists", func(t *testing.T) {
		t.Parallel()

		mockStorage := storagetesting.NewStorage(t)
		mockHasher := hashertesting.NewHasher(t)
		mockStorage.
			On("ListUsers", mock.Anything, setupStatusListUsersParam()).
			Return([]*usertypes.User{}, nil).
			Once()
		mockHasher.On("Hash", mock.Anything, "secret").Return("hash", nil).Once()
		mockStorage.On("UpsertUser", mock.Anything, mock.MatchedBy(func(user *usertypes.User) bool {
			return assert.NotNil(t, user) &&
				assert.Equal(t, usertypes.RoleOwner, user.Role) &&
				assert.False(t, user.MustChangePassword)
		})).Run(func(args mock.Arguments) {
			user := args.Get(1).(*usertypes.User)
			user.ID = 2
		}).Return(nil).Once()
		mockStorage.On("InsertRefreshToken", mock.Anything, mock.MatchedBy(func(token *userauth.RefreshToken) bool {
			return assert.NotNil(t, token) && assert.Equal(t, int32(2), token.UserID)
		})).Return(nil).Once()

		service := newAuthTestService(mockStorage, mockHasher)
		session, err := service.Setup(t.Context(), NewUser{Name: "Root", Email: "root@example.com", Password: "secret"})

		require.NoError(t, err)
		require.NotNil(t, session)
		assert.Equal(t, int32(2), session.User.ID)
		assert.NotEmpty(t, session.Tokens.AccessToken)
		assert.NotEmpty(t, session.Tokens.RefreshToken)
	})

	testCases := []struct {
		name      string
		modifyFn  func(mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expectErr error
	}{
		{
			name: "rejects when owner or admin users exist",
			modifyFn: func(mockStorage *storagetesting.Storage, _ *hashertesting.Hasher) {
				mockStorage.
					On("ListUsers", mock.Anything, setupStatusListUsersParam()).
					Return([]*usertypes.User{{ID: 1, Role: usertypes.RoleOwner}}, nil).
					Once()
			},
			expectErr: serviceerrors.ErrSetupUnavailable,
		},
		{
			name: "maps concurrent owner insert to unavailable",
			modifyFn: func(mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				mockStorage.
					On("ListUsers", mock.Anything, setupStatusListUsersParam()).
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
