package users

import (
	"context"
	"strings"

	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type UpdateUserParams struct {
	ID       int32
	Name     *string
	Role     *usertypes.Role
	IsActive *bool
}

func (u *UpdateUserParams) Validate() error {
	if u.ID <= 0 {
		return xerrors.New("user id is required")
	}
	if u.Name != nil && strings.TrimSpace(*u.Name) == "" {
		return xerrors.New("name is required")
	}
	if u.Role != nil {
		switch *u.Role {
		case usertypes.RoleOwner, usertypes.RoleAdmin, usertypes.RoleMember:
			// valid
		default:
			return xerrors.New("role is invalid")
		}
	}
	return nil
}

func (s *Service) UpdateUser(ctx context.Context, params UpdateUserParams) (*usertypes.User, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	user, err := s.GetUser(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != nil {
		user.Name = strings.TrimSpace(*params.Name)
	}
	if params.Role != nil {
		user.Role = *params.Role
	}
	if params.IsActive != nil {
		user.IsActive = *params.IsActive
	}

	if err := s.storage.UpsertUser(ctx, user); err != nil {
		return nil, xerrors.Newf("failed to update user: %w", err)
	}

	return user, nil
}
