package users

import (
	"context"

	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type ChangePasswordParams struct {
	UserID          int32
	CurrentPassword string
	NewPassword     string
}

func (s *Service) ChangePassword(ctx context.Context, params ChangePasswordParams) (*usertypes.User, error) {
	if params.UserID <= 0 || params.CurrentPassword == "" || params.NewPassword == "" {
		return nil, xerrors.New("user id, current password, and new password are required")
	}

	user, err := s.GetUser(ctx, params.UserID)
	if err != nil {
		return nil, err
	}

	ok, err := s.hasher.Verify(ctx, params.CurrentPassword, user.PasswordHash)
	if err != nil {
		return nil, xerrors.Newf("failed to verify password: %w", err)
	}
	if !ok {
		return nil, serviceerrors.ErrUnauthorized
	}

	hashedPassword, err := s.hasher.Hash(ctx, params.NewPassword)
	if err != nil {
		return nil, xerrors.Newf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword
	user.MustChangePassword = false
	if err := s.storage.UpsertUser(ctx, user); err != nil {
		return nil, xerrors.Newf("failed to update password: %w", err)
	}

	return user, nil
}
