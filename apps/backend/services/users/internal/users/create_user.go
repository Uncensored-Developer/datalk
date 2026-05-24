package users

import (
	"context"
	"log/slog"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type NewUser struct {
	Name               string
	Email              string
	Password           string
	Role               users.Role
	MustChangePassword bool
}

func (n *NewUser) Validate() error {
	if n.Name == "" {
		return xerrors.New("name is required")
	}
	if n.Email == "" {
		return xerrors.New("email is required")
	}
	if n.Password == "" {
		return xerrors.New("password is required")
	}
	switch n.Role {
	case "", users.RoleOwner, users.RoleAdmin, users.RoleMember:
	default:
		return xerrors.New("role is invalid")
	}
	return nil
}

func (s *Service) CreateUser(ctx context.Context, newUser NewUser) (*users.User, error) {
	if err := newUser.Validate(); err != nil {
		return nil, err
	}

	hashedPassword, err := s.hasher.Hash(ctx, newUser.Password)
	if err != nil {
		return nil, xerrors.Newf("failed to hash password: %w", err)
	}

	user := users.User{
		Name:               newUser.Name,
		Email:              newUser.Email,
		PasswordHash:       hashedPassword,
		Role:               newUser.Role,
		IsActive:           true,
		MustChangePassword: newUser.MustChangePassword,
		CreatedAt:          time.Now().UTC(),
	}
	if user.Role == "" {
		user.Role = users.RoleMember
	}
	err = s.storage.UpsertUser(ctx, &user)
	if err != nil {
		return nil, xerrors.Newf("failed to insert user: %w", err)
	}

	s.Logger().Info(
		"user created",
		slog.Int("user_id", int(user.ID)),
		slog.String("role", string(user.Role)),
	)

	return &user, nil
}
