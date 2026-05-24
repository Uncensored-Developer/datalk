package users

import (
	"context"
	"time"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/storage"
	serviceerrors "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/errors"
	usertypes "github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

func (s *Service) issueTokenPair(ctx context.Context, user *usertypes.User, now time.Time) (userauth.TokenPair, error) {
	accessToken, expiresAt, err := s.tokens.SignAccessToken(user.ID, now)
	if err != nil {
		return userauth.TokenPair{}, xerrors.Newf("failed to sign access token: %w", err)
	}

	refreshToken, refreshHash, err := s.tokens.NewRefreshToken()
	if err != nil {
		return userauth.TokenPair{}, err
	}
	if err := s.storage.InsertRefreshToken(ctx, &userauth.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(s.tokens.RefreshTTL()),
	}); err != nil {
		return userauth.TokenPair{}, xerrors.Newf("failed to store refresh token: %w", err)
	}

	return userauth.TokenPair{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresAt:          expiresAt,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

func (s *Service) getUserByEmail(ctx context.Context, email string) (*usertypes.User, error) {
	fetchedUsers, err := s.storage.ListUsers(ctx, storage.ListUsersParam{Email: []string{email}})
	if err != nil {
		return nil, xerrors.Newf("failed to list users: %w", err)
	}
	if len(fetchedUsers) == 0 {
		return nil, serviceerrors.ErrUserNotFound
	}
	return fetchedUsers[0], nil
}
