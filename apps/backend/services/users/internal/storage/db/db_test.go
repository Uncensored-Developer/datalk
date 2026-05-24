package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/aarondl/opt/null"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorage_UpsertUser(t *testing.T) {
	t.Parallel()

	t.Run("Inserting and Listing new user", func(t *testing.T) {
		t.Parallel()
		createdAt := time.Now().UTC().Add(-time.Hour)
		updatedAt := time.Now().UTC().Add(-time.Minute)
		newUser := &users.User{
			Email:              uniqueEmail("insert"),
			Name:               "Edward Newgate",
			PasswordHash:       "fake_password_hash",
			Role:               users.RoleMember,
			IsActive:           true,
			CreatedAt:          createdAt,
			MustChangePassword: true,
			UpdatedAt:          updatedAt,
		}
		err := s.UpsertUser(t.Context(), newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)
		assert.NotEmpty(t, newUser.CreatedAt)
		assert.Empty(t, newUser.LastLoginAt)
		assert.True(t, newUser.MustChangePassword)
		assert.WithinDuration(t, updatedAt, newUser.UpdatedAt, time.Second)

		gotUsers, err := s.ListUsers(t.Context(), storage.ListUsersParam{Email: []string{newUser.Email}})
		require.NoError(t, err)
		require.Len(t, gotUsers, 1)
		gotUser := gotUsers[0]
		assert.Equal(t, newUser.ID, gotUser.ID)
		assert.Equal(t, newUser.Email, gotUser.Email)
		assert.Equal(t, newUser.PasswordHash, gotUser.PasswordHash)
		assert.Equal(t, users.RoleMember, gotUser.Role)
		assert.True(t, gotUser.IsActive)
		assert.True(t, gotUser.MustChangePassword)
		assert.WithinDuration(t, createdAt, gotUser.CreatedAt, time.Second)
		assert.WithinDuration(t, updatedAt, gotUser.UpdatedAt, time.Second)
	})

	t.Run("Updating existing user", func(t *testing.T) {
		t.Parallel()
		createdAt := time.Now().UTC().Add(-2 * time.Hour)
		originalUpdatedAt := time.Now().UTC().Add(-time.Hour)
		userTmpl := factory.UserTemplate{}
		userTmpl.Apply(
			t.Context(),
			factory.UserMods.Email(uniqueEmail("update")),
			factory.UserMods.MustChangePassword(false),
			factory.UserMods.CreatedAt(createdAt),
			factory.UserMods.UpdatedAt(originalUpdatedAt),
		)
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
		createdUser.Name = "Edward Newgate (updated)"
		createdUser.IsActive = false
		createdUser.PasswordHash = "updated_password_hash"
		createdUser.MustChangePassword = true
		lastLoginAt := time.Now().UTC().Add(-30 * time.Minute)
		updatedAt := time.Now().UTC()
		createdUser.LastLoginAt = null.From(lastLoginAt)
		createdUser.UpdatedAt = updatedAt

		user, err := userFromDB(createdUser)
		require.NoError(t, err)

		err = s.UpsertUser(t.Context(), user)
		require.NoError(t, err)

		assert.Equal(t, "Edward Newgate (updated)", user.Name)
		assert.False(t, user.IsActive)
		assert.NotEmpty(t, user.LastLoginAt)
		assert.Equal(t, "updated_password_hash", user.PasswordHash)
		assert.True(t, user.MustChangePassword)
		require.NotNil(t, user.LastLoginAt)
		assert.WithinDuration(t, lastLoginAt, *user.LastLoginAt, time.Second)
		assert.WithinDuration(t, updatedAt, user.UpdatedAt, time.Second)

		gotUsers, err := s.ListUsers(t.Context(), storage.ListUsersParam{ID: []int32{user.ID}})
		require.NoError(t, err)
		require.Len(t, gotUsers, 1)
		gotUser := gotUsers[0]
		assert.Equal(t, user.ID, gotUser.ID)
		assert.Equal(t, "Edward Newgate (updated)", gotUser.Name)
		assert.False(t, gotUser.IsActive)
		assert.Equal(t, "updated_password_hash", gotUser.PasswordHash)
		assert.True(t, gotUser.MustChangePassword)
		require.NotNil(t, gotUser.LastLoginAt)
		assert.WithinDuration(t, lastLoginAt, *gotUser.LastLoginAt, time.Second)
		assert.WithinDuration(t, updatedAt, gotUser.UpdatedAt, time.Second)
	})
}

func TestStorage_UpsertUserRejectsSecondOwner(t *testing.T) {
	t.Parallel()

	firstOwner := &users.User{
		Email:        uniqueEmail("owner1"),
		Name:         "Owner One",
		PasswordHash: "fake_password_hash",
		Role:         users.RoleOwner,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}
	err := s.UpsertUser(t.Context(), firstOwner)
	require.NoError(t, err)

	secondOwner := &users.User{
		Email:        uniqueEmail("owner2"),
		Name:         "Owner Two",
		PasswordHash: "fake_password_hash",
		Role:         users.RoleOwner,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}
	err = s.UpsertUser(t.Context(), secondOwner)
	require.ErrorIs(t, err, storage.ErrOwnerAlreadyExists)
}

func TestStorage_RefreshTokens(t *testing.T) {
	t.Parallel()

	t.Run("Inserting and getting refresh token", func(t *testing.T) {
		t.Parallel()
		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
		expiresAt := time.Now().UTC().Add(time.Hour)
		tokenHash := uniqueTokenHash("insert")
		token := &userauth.RefreshToken{
			UserID:    createdUser.ID,
			TokenHash: tokenHash,
			ExpiresAt: expiresAt,
		}

		err := s.InsertRefreshToken(t.Context(), token)
		require.NoError(t, err)
		assert.NotZero(t, token.ID)
		assert.Equal(t, createdUser.ID, token.UserID)
		assert.Equal(t, tokenHash, token.TokenHash)
		assert.WithinDuration(t, expiresAt, token.ExpiresAt, time.Second)
		assert.Nil(t, token.RevokedAt)
		assert.False(t, token.CreatedAt.IsZero())

		gotToken, err := s.GetRefreshToken(t.Context(), tokenHash)
		require.NoError(t, err)
		assert.Equal(t, token.ID, gotToken.ID)
		assert.Equal(t, createdUser.ID, gotToken.UserID)
		assert.Equal(t, tokenHash, gotToken.TokenHash)
		assert.WithinDuration(t, expiresAt, gotToken.ExpiresAt, time.Second)
		assert.Nil(t, gotToken.RevokedAt)
		assert.False(t, gotToken.CreatedAt.IsZero())
	})

	t.Run("Revoking refresh token", func(t *testing.T) {
		t.Parallel()
		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
		tokenHash := uniqueTokenHash("revoke")
		token := &userauth.RefreshToken{
			UserID:    createdUser.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		}

		err := s.InsertRefreshToken(t.Context(), token)
		require.NoError(t, err)

		revokedAt := time.Now().UTC()
		err = s.RevokeRefreshToken(t.Context(), tokenHash, revokedAt)
		require.NoError(t, err)

		gotToken, err := s.GetRefreshToken(t.Context(), tokenHash)
		require.NoError(t, err)
		require.NotNil(t, gotToken.RevokedAt)
		assert.WithinDuration(t, revokedAt, *gotToken.RevokedAt, time.Second)

		err = s.RevokeRefreshToken(t.Context(), tokenHash, time.Now().UTC())
		require.ErrorIs(t, err, storage.ErrRefreshTokenNotRevoked)
	})
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s.%d@datalk.app", prefix, time.Now().UnixNano())
}

func uniqueTokenHash(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
