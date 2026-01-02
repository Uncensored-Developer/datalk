package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

type ListUsersParam struct {
	Email []string
	ID    []int64
}

type Storage interface {
	UpsertUser(ctx context.Context, user *users.User) error

	ListUsers(ctx context.Context, params ListUsersParam) ([]*users.User, error)
}
