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
	"github.com/mdobak/go-xerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_Login(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                   string
		params                 LoginParams
		modifyFn               func(params LoginParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expectError            error
		expectErrorContains    string
		assertNoRefreshTokenFn bool
	}{
		{
			name:   "issues token pair",
			params: LoginParams{Email: "admin@example.com", Password: "secret"},
			modifyFn: func(params LoginParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				user := &usertypes.User{
					ID:           7,
					Email:        params.Email,
					Name:         "Admin",
					PasswordHash: "hash",
					Role:         usertypes.RoleAdmin,
					IsActive:     true,
				}
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{Email: []string{user.Email}}).
					Return([]*usertypes.User{user}, nil).
					Once()
				mockHasher.On("Verify", mock.Anything, params.Password, "hash").Return(true, nil).Once()
				mockStorage.On("UpsertUser", mock.Anything, mock.MatchedBy(func(updatedUser *usertypes.User) bool {
					return updatedUser != nil && updatedUser.ID == user.ID && updatedUser.LastLoginAt != nil
				})).Return(nil).Once()
				mockStorage.
					On("InsertRefreshToken", mock.Anything, mock.MatchedBy(func(token *userauth.RefreshToken) bool {
						return token != nil &&
							token.UserID == user.ID &&
							token.TokenHash != "" &&
							token.ExpiresAt.After(time.Now().UTC())
					})).
					Return(nil).
					Once()
			},
		},
		{
			name:   "rejects bad password",
			params: LoginParams{Email: "user@example.com", Password: "wrong"},
			modifyFn: func(params LoginParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				user := &usertypes.User{ID: 7, Email: params.Email, PasswordHash: "hash", IsActive: true}
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{Email: []string{user.Email}}).
					Return([]*usertypes.User{user}, nil).
					Once()
				mockHasher.On("Verify", mock.Anything, params.Password, "hash").Return(false, nil).Once()
			},
			expectError:            serviceerrors.ErrUnauthorized,
			assertNoRefreshTokenFn: true,
		},
		{
			name:   "maps storage not found to unauthorized",
			params: LoginParams{Email: "missing@example.com", Password: "secret"},
			modifyFn: func(params LoginParams, mockStorage *storagetesting.Storage, _ *hashertesting.Hasher) {
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{Email: []string{params.Email}}).
					Return([]*usertypes.User{}, nil).
					Once()
			},
			expectError: serviceerrors.ErrUnauthorized,
		},
		{
			name:   "returns verify errors",
			params: LoginParams{Email: "user@example.com", Password: "secret"},
			modifyFn: func(params LoginParams, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				user := &usertypes.User{ID: 7, Email: params.Email, PasswordHash: "hash", IsActive: true}
				mockStorage.
					On("ListUsers", mock.Anything, storage.ListUsersParam{Email: []string{user.Email}}).
					Return([]*usertypes.User{user}, nil).
					Once()
				mockHasher.On("Verify", mock.Anything, params.Password, "hash").Return(false, xerrors.New("argon2 failure")).Once()
			},
			expectErrorContains: "failed to verify password",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockHasher := hashertesting.NewHasher(t)
			tc.modifyFn(tc.params, mockStorage, mockHasher)

			service := newAuthTestService(mockStorage, mockHasher)
			session, err := service.Login(t.Context(), tc.params)

			switch {
			case tc.expectError != nil:
				require.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, session)
			case tc.expectErrorContains != "":
				require.ErrorContains(t, err, tc.expectErrorContains)
				assert.Nil(t, session)
			default:
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.NotEmpty(t, session.Tokens.AccessToken)
				assert.NotEmpty(t, session.Tokens.RefreshToken)
				assert.False(t, session.Tokens.MustChangePassword)
			}

			if tc.assertNoRefreshTokenFn {
				mockStorage.AssertNotCalled(t, "InsertRefreshToken", mock.Anything, mock.Anything)
			}
		})
	}
}
