package users

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
	"github.com/mdobak/go-xerrors"
)

type NewUser struct {
	Name     string
	Email    string
	Password string
	Role     users.Role
}

func (n *NewUser) Validate() error {
	if n.Name == "" {
		return errors.New("name is required")
	}
	if n.Email == "" {
		return errors.New("email is required")
	}
	if n.Password == "" {
		return errors.New("password is required")
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
		Name:         newUser.Name,
		Email:        newUser.Email,
		PasswordHash: hashedPassword,
		Role:         newUser.Role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
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
