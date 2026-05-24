package storage

import (
	"context"
	"time"

	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
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

	InsertRefreshToken(ctx context.Context, token *userauth.RefreshToken) error

	GetRefreshToken(ctx context.Context, tokenHash string) (*userauth.RefreshToken, error)

	RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error
}
