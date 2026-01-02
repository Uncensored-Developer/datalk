package db

import (
	"testing"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/db/common"
	"github.com/Uncensored-Developer/datalk/apps/backend/db/factory"
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
		newUser := &users.User{
			Email:        "test1.user@datalk.app",
			Name:         "Edward Newgate",
			PasswordHash: "fake_password_hash",
			Role:         users.RoleMember,
			IsActive:     true,
			CreatedAt:    time.Now().UTC(),
		}
		err := s.UpsertUser(t.Context(), newUser)
		require.NoError(t, err)
		assert.NotZero(t, newUser.ID)
		assert.NotEmpty(t, newUser.CreatedAt)
		assert.Empty(t, newUser.LastLoginAt)

		gotUsers, err := s.ListUsers(t.Context(), storage.ListUsersParam{Email: []string{newUser.Email}})
		require.NoError(t, err)
		require.Len(t, gotUsers, 1)
		assert.Equal(t, newUser.ID, gotUsers[0].ID)
	})

	t.Run("Updating existing user", func(t *testing.T) {
		userTmpl := factory.UserTemplate{}
		createdUser := userTmpl.CreateOrFail(t.Context(), t, runner.BobConn)
		createdUser.Name = "Edward Newgate (updated)"
		createdUser.IsActive = 0
		createdUser.LastLoginAt = null.From(common.TimeToDB(time.Now().UTC()))

		user, err := userFromDB(createdUser)
		require.NoError(t, err)

		err = s.UpsertUser(t.Context(), user)
		require.NoError(t, err)

		assert.Equal(t, "Edward Newgate (updated)", user.Name)
		assert.False(t, user.IsActive)
		assert.NotEmpty(t, user.LastLoginAt)
	})
}
