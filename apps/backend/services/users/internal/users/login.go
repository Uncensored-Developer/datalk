package users

import (
	"context"
	stderrors "errors"
	"time"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	"github.com/mdobak/go-xerrors"
)

type LoginParams struct {
	Email    string
	Password string
}

func (s *Service) Login(ctx context.Context, params LoginParams) (*userauth.Session, error) {
	if params.Email == "" || params.Password == "" {
		return nil, serviceerrors.ErrUnauthorized
	}

	user, err := s.getUserByEmail(ctx, params.Email)
	if err != nil {
		if stderrors.Is(err, serviceerrors.ErrUserNotFound) {
			return nil, serviceerrors.ErrUnauthorized
		}
		return nil, err
	}
	if !user.IsActive {
		return nil, serviceerrors.ErrInactiveUser
	}

	ok, err := s.hasher.Verify(ctx, params.Password, user.PasswordHash)
	if err != nil {
		return nil, xerrors.Newf("failed to verify password: %w", err)
	}
	if !ok {
		return nil, serviceerrors.ErrUnauthorized
	}

	now := time.Now().UTC()
	user.LastLoginAt = &now
	if err := s.storage.UpsertUser(ctx, user); err != nil {
		return nil, xerrors.Newf("failed to update last login: %w", err)
	}

	tokens, err := s.issueTokenPair(ctx, user, now)
	if err != nil {
		return nil, err
	}
	return &userauth.Session{User: user, Tokens: tokens}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	if err := s.storage.RevokeRefreshToken(ctx, userauth.HashRefreshToken(refreshToken), time.Now().UTC()); err != nil {
		if stderrors.Is(err, storage.ErrRefreshTokenNotRevoked) {
			return nil
		}
		return xerrors.Newf("failed to revoke refresh token: %w", err)
	}
	return nil
}
