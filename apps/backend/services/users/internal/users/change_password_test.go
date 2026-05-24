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

func TestService_ChangePassword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                   string
		params                 ChangePasswordParams
		modifyFn               func(params ChangePasswordParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expectPasswordHash     string
		expectMustChange       bool
		expectError            error
		assertNoPasswordUpdate bool
	}{
		{
			name: "clears password change flag",
			params: ChangePasswordParams{
				UserID:          7,
				CurrentPassword: "old",
				NewPassword:     "new",
			},
			modifyFn: func(params ChangePasswordParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				user := &usertypes.User{ID: params.UserID, Email: "user@example.com", PasswordHash: "old-hash", MustChangePassword: true}
				mockStorage.On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int32{user.ID}}).Return([]*usertypes.User{user}, nil).Once()
				mockHasher.On("Verify", mock.Anything, params.CurrentPassword, "old-hash").Return(true, nil).Once()
				mockHasher.On("Hash", mock.Anything, params.NewPassword).Return("new-hash", nil).Once()
				mockStorage.On("UpsertUser", mock.Anything, mock.MatchedBy(func(updatedUser *usertypes.User) bool {
					return updatedUser != nil &&
						updatedUser.ID == user.ID &&
						updatedUser.PasswordHash == "new-hash" &&
						!updatedUser.MustChangePassword
				})).Return(nil).Once()
			},
			expectPasswordHash: "new-hash",
			expectMustChange:   false,
		},
		{
			name: "rejects bad current password",
			params: ChangePasswordParams{
				UserID:          7,
				CurrentPassword: "bad",
				NewPassword:     "new",
			},
			modifyFn: func(params ChangePasswordParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				user := &usertypes.User{ID: params.UserID, Email: "user@example.com", PasswordHash: "old-hash"}
				mockStorage.On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int32{user.ID}}).Return([]*usertypes.User{user}, nil).Once()
				mockHasher.On("Verify", mock.Anything, params.CurrentPassword, "old-hash").Return(false, nil).Once()
			},
			expectError:            serviceerrors.ErrUnauthorized,
			assertNoPasswordUpdate: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockHasher := hashertesting.NewHasher(t)
			tc.modifyFn(tc.params, mockStorage, mockHasher)

			service := newAuthTestService(mockStorage, mockHasher)
			updatedUser, err := service.ChangePassword(t.Context(), tc.params)

			if tc.expectError != nil {
				require.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, updatedUser)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectPasswordHash, updatedUser.PasswordHash)
				assert.Equal(t, tc.expectMustChange, updatedUser.MustChangePassword)
			}

			if tc.assertNoPasswordUpdate {
				mockHasher.AssertNotCalled(t, "Hash", mock.Anything, mock.Anything)
				mockStorage.AssertNotCalled(t, "UpsertUser", mock.Anything, mock.Anything)
			}
		})
	}
}
