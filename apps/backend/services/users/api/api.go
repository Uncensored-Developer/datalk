package api

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	usersservice "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

//go:generate go tool with-modfile mockery --name Service --outpkg testing --output ./testing --filename generated__users_service_mocks.go
type Service interface {
	CreateUser(ctx context.Context, params usersservice.NewUser) (*users.User, error)
	GetUser(ctx context.Context, ID int32) (*users.User, error)
}

type Api struct {
	*base.Base
	service Service
}

func New(logger *slog.Logger, cfg config.Config, service Service) *Api {
	return &Api{
		Base:    base.NewBase("users", logger, cfg),
		service: service,
	}
}

func (a *Api) CreateUser(ctx context.Context, params NewUserParams) (*users.User, error) {
	return a.service.CreateUser(ctx, usersservice.NewUser{
		Email:    params.Email,
		Password: params.Password,
		Name:     params.Name,
		Role:     params.Role,
	})
}

func (a *Api) GetUser(ctx context.Context, userID int32) (*users.User, error) {
	return a.service.GetUser(ctx, userID)
}
