package storage

import (
	"context"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

type ListUsersParam struct {
	Email []string
	ID    []int32
}

//go:generate go tool with-modfile mockery --name Storage --outpkg testing --output ./testing --filename generated__storage_mocks.go
type Storage interface {
	UpsertUser(ctx context.Context, user *users.User) error

	ListUsers(ctx context.Context, params ListUsersParam) ([]*users.User, error)
}
