package users

import (
	"testing"
	"time"

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

func TestService_Refresh(t *testing.T) {
	t.Parallel()

	const refreshToken = "refresh-token"
	tokenHash := userauth.HashRefreshToken(refreshToken)
	user := &usertypes.User{ID: 7, Email: "user@example.com", IsActive: true}

	testCases := []struct {
		name                   string
		modifyFn               func(mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expectError            error
		assertNoUserLookup     bool
		assertNoRefreshTokenFn bool
	}{
		{
			name: "rotates token",
			modifyFn: func(mockStorage *storagetesting.Storage, _ *hashertesting.Hasher) {
				mockStorage.
					On("GetRefreshToken", mock.Anything, tokenHash).
					Return(&userauth.RefreshToken{
						UserID:    user.ID,
						TokenHash: tokenHash,
						ExpiresAt: time.Now().UTC().Add(time.Hour),
					}, nil).
					Once()
				mockStorage.On("RevokeRefreshToken", mock.Anything, tokenHash, mock.AnythingOfType("time.Time")).Return(nil).Once()
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{ID: []int32{user.ID}}).
					Return([]*usertypes.User{user}, nil).
					Once()
				mockStorage.
					On("InsertRefreshToken", mock.Anything, mock.MatchedBy(func(token *userauth.RefreshToken) bool {
						return token != nil && token.UserID == user.ID && token.TokenHash != tokenHash
					})).
					Return(nil).
					Once()
			},
		},
		{
			name: "rejects already consumed token",
			modifyFn: func(mockStorage *storagetesting.Storage, _ *hashertesting.Hasher) {
				mockStorage.
					On("GetRefreshToken", mock.Anything, tokenHash).
					Return(&userauth.RefreshToken{
						UserID:    user.ID,
						TokenHash: tokenHash,
						ExpiresAt: time.Now().UTC().Add(time.Hour),
					}, nil).
					Once()
				mockStorage.
					On("RevokeRefreshToken", mock.Anything, tokenHash, mock.AnythingOfType("time.Time")).
					Return(storage.ErrRefreshTokenNotRevoked).
					Once()
			},
			expectError:            serviceerrors.ErrRefreshTokenInvalid,
			assertNoUserLookup:     true,
			assertNoRefreshTokenFn: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockHasher := hashertesting.NewHasher(t)
			tc.modifyFn(mockStorage, mockHasher)

			service := newAuthTestService(mockStorage, mockHasher)
			session, err := service.Refresh(t.Context(), refreshToken)

			if tc.expectError != nil {
				require.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, session)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.NotEmpty(t, session.Tokens.AccessToken)
				assert.NotEmpty(t, session.Tokens.RefreshToken)
			}

			if tc.assertNoUserLookup {
				mockStorage.AssertNotCalled(t, "ListUsers", mock.Anything, mock.Anything)
			}
			if tc.assertNoRefreshTokenFn {
				mockStorage.AssertNotCalled(t, "InsertRefreshToken", mock.Anything, mock.Anything)
			}
		})
	}
}
