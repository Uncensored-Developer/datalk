package users

import (
	"testing"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	storagetesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage/testing"
	hashertesting "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users/hashers/testing"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_SetupStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		users         []*usertypes.User
		setupRequired bool
	}{
		{
			name:          "required when admin query returns no users",
			users:         []*usertypes.User{},
			setupRequired: true,
		},
		{
			name:          "not required when setup status query returns an admin",
			users:         []*usertypes.User{{ID: 1, Role: usertypes.RoleAdmin}},
			setupRequired: false,
		},
		{
			name:          "not required when setup status query returns an owner",
			users:         []*usertypes.User{{ID: 1, Role: usertypes.RoleOwner}},
			setupRequired: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockStorage := storagetesting.NewStorage(t)
			mockStorage.
				On("ListUsers", mock.Anything, setupStatusListUsersParam()).
				Return(tc.users, nil).
				Once()
			service := newAuthTestService(mockStorage, hashertesting.NewHasher(t))

			status, err := service.SetupStatus(t.Context())

			require.NoError(t, err)
			assert.Equal(t, tc.setupRequired, status.SetupRequired)
		})
	}
}

func setupStatusListUsersParam() storage.ListUsersParam {
	return storage.ListUsersParam{Roles: []usertypes.Role{usertypes.RoleOwner, usertypes.RoleAdmin}}
}
