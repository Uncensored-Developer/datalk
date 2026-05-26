package users

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type SetupStatus struct {
	SetupRequired bool
}

func (s *Service) SetupStatus(ctx context.Context) (*SetupStatus, error) {
	existingUsers, err := s.storage.ListUsers(ctx, storage.ListUsersParam{
		Roles: []usertypes.Role{usertypes.RoleOwner, usertypes.RoleAdmin},
	})
	if err != nil {
		return nil, xerrors.Newf("failed to list users: %w", err)
	}

	return &SetupStatus{SetupRequired: len(existingUsers) == 0}, nil
}
