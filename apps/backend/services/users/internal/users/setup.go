package users

import (
	"context"
	stderrors "errors"
	"time"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) Setup(ctx context.Context, newUser NewUser) (*userauth.Session, error) {
	status, err := s.SetupStatus(ctx)
	if err != nil {
		return nil, xerrors.Newf("failed to get setup status: %w", err)
	}
	if !status.SetupRequired {
		return nil, serviceerrors.ErrSetupUnavailable
	}

	newUser.Role = usertypes.RoleOwner
	newUser.MustChangePassword = false
	user, err := s.CreateUser(ctx, newUser)
	if err != nil {
		if stderrors.Is(err, storage.ErrOwnerAlreadyExists) {
			return nil, serviceerrors.ErrSetupUnavailable
		}
		return nil, err
	}

	tokens, err := s.issueTokenPair(ctx, user, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &userauth.Session{User: user, Tokens: tokens}, nil
}
