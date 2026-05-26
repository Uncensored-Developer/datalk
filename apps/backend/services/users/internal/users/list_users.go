package users

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) ListUsers(ctx context.Context) ([]*usertypes.User, error) {
	users, err := s.storage.ListUsers(ctx, storage.ListUsersParam{})
	if err != nil {
		return nil, xerrors.Newf("failed to list users: %w", err)
	}
	return users, nil
}
