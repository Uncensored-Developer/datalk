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

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*userauth.Session, error) {
	if refreshToken == "" {
		return nil, serviceerrors.ErrRefreshTokenInvalid
	}

	tokenHash := userauth.HashRefreshToken(refreshToken)
	token, err := s.storage.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if stderrors.Is(err, storage.ErrRefreshTokenNotFound) {
			return nil, serviceerrors.ErrRefreshTokenInvalid
		}
		return nil, xerrors.Newf("failed to get refresh token: %w", err)
	}

	now := time.Now().UTC()
	if token.RevokedAt != nil || !now.Before(token.ExpiresAt) {
		return nil, serviceerrors.ErrRefreshTokenInvalid
	}
	if err := s.storage.RevokeRefreshToken(ctx, tokenHash, now); err != nil {
		if stderrors.Is(err, storage.ErrRefreshTokenNotRevoked) {
			return nil, serviceerrors.ErrRefreshTokenInvalid
		}
		return nil, xerrors.Newf("failed to revoke refresh token: %w", err)
	}

	user, err := s.GetUser(ctx, token.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, serviceerrors.ErrInactiveUser
	}

	tokens, err := s.issueTokenPair(ctx, user, now)
	if err != nil {
		return nil, err
	}
	return &userauth.Session{User: user, Tokens: tokens}, nil
}
