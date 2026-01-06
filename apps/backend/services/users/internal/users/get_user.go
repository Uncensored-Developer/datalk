package users

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) GetUser(ctx context.Context, ID int64) (*users.User, error) {
	fetchedUsers, err := s.storage.ListUsers(ctx, storage.ListUsersParam{ID: []int64{ID}})
	if err != nil {
		return nil, xerrors.Newf("failed to list users: %w", err)
	}
	if len(fetchedUsers) == 0 {
		return nil, errors.ErrUserNotFound
	}
	return fetchedUsers[0], nil
}
