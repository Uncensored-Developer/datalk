package users

import (
	"errors"
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage/testing"
	hashertesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users/hashers/testing"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_CreateUser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		newUser     NewUser
		modifyFn    func(newUser NewUser, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher)
		expect      *users.User
		expectError error
	}{
		{
			name: "invalid: empty name",
			newUser: NewUser{
				Name:     "",
				Email:    "user@example.com",
				Password: "secret",
				Role:     users.RoleMember,
			},
			expectError: errors.New("name is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: empty email",
			newUser: NewUser{
				Name:     "John Doe",
				Email:    "",
				Password: "secret",
				Role:     users.RoleMember,
			},
			expectError: errors.New("email is required"),
			modifyFn:    nil,
		},
		{
			name: "invalid: empty password",
			newUser: NewUser{
				Name:     "John Doe",
				Email:    "user@example.com",
				Password: "",
				Role:     users.RoleMember,
			},
			expectError: errors.New("password is required"),
			modifyFn:    nil,
		},
		{
			name: "hashing failure",
			newUser: NewUser{
				Name:     "John Doe",
				Email:    "user@example.com",
				Password: "secret",
				Role:     users.RoleMember,
			},
			modifyFn: func(newUser NewUser, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				mockHasher.
					On("Hash", mock.Anything, newUser.Password).
					Return("", errors.New("hash service down"))
			},
			expectError: errors.New("failed to hash password"),
		},
		{
			name: "storage failure",
			newUser: NewUser{
				Name:     "John Doe",
				Email:    "user@example.com",
				Password: "secret",
				Role:     users.RoleAdmin,
			},
			modifyFn: func(newUser NewUser, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				const hashed = "hashed-pass"
				mockHasher.
					On("Hash", mock.Anything, newUser.Password).
					Return(hashed, nil)

				// Ensure UpsertUser receives expected fields
				matcher := mock.MatchedBy(func(u *users.User) bool {
					return u != nil &&
						u.Name == newUser.Name &&
						u.Email == newUser.Email &&
						u.PasswordHash == hashed &&
						u.Role == newUser.Role &&
						u.IsActive
				})
				mockStorage.
					On("UpsertUser", mock.Anything, matcher).
					Return(errors.New("db write failed"))
			},
			expectError: errors.New("failed to insert user"),
		},
		{
			name: "success",
			newUser: NewUser{
				Name:     "Jane Smith",
				Email:    "jane@example.com",
				Password: "strong-pass",
				Role:     users.RoleMember,
			},
			modifyFn: func(newUser NewUser, mockStorage *storagetesting.Storage, mockHasher *hashertesting.Hasher) {
				const hashed = "HASHED_123"
				mockHasher.
					On("Hash", mock.Anything, newUser.Password).
					Return(hashed, nil)

				// Just accept any valid user and succeed
				mockStorage.
					On("UpsertUser", mock.Anything, mock.MatchedBy(func(u *users.User) bool {
						return u != nil &&
							u.Name == newUser.Name &&
							u.Email == newUser.Email &&
							u.PasswordHash == hashed &&
							u.Role == newUser.Role &&
							u.IsActive && !u.CreatedAt.IsZero()
					})).
					Return(nil)
			},
			expect: &users.User{
				Name:         "Jane Smith",
				Email:        "jane@example.com",
				PasswordHash: "HASHED_123",
				Role:         users.RoleMember,
				IsActive:     true,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockHasher := hashertesting.NewHasher(t)

			if tc.modifyFn != nil {
				tc.modifyFn(tc.newUser, mockStorage, mockHasher)
			}

			service := &Service{
				Base:    &base.Base{},
				storage: mockStorage,
				hasher:  mockHasher,
			}

			got, err := service.CreateUser(t.Context(), tc.newUser)
			if tc.expectError != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expectError.Error())
			} else {
				require.NoError(t, err)
				// Validate core fields, allowing dynamic fields like CreatedAt
				if tc.expect != nil {
					assert.Equal(t, tc.expect.Name, got.Name)
					assert.Equal(t, tc.expect.Email, got.Email)
					assert.Equal(t, tc.expect.PasswordHash, got.PasswordHash)
					assert.Equal(t, tc.expect.Role, got.Role)
					assert.True(t, got.IsActive)
					assert.False(t, got.CreatedAt.IsZero())
				}
			}
		})
	}
}
