package api

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	userauth "github.com/Uncensored-Developer/datalk/apps/backend/pkg/auth"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	usersservice "github.com/Uncensored-Developer/datalk/apps/backend/services/users/internal/users"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/users/pkg/users"
)

//go:generate go tool with-modfile mockery --name Service --outpkg testing --output ./testing --filename generated__users_service_mocks.go
type Service interface {
	CreateUser(ctx context.Context, params usersservice.NewUser) (*users.User, error)
	GetUser(ctx context.Context, ID int32) (*users.User, error)
	Setup(ctx context.Context, params usersservice.NewUser) (*userauth.Session, error)
	Login(ctx context.Context, params usersservice.LoginParams) (*userauth.Session, error)
	Refresh(ctx context.Context, refreshToken string) (*userauth.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, params usersservice.ChangePasswordParams) (*users.User, error)
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
		Email:              params.Email,
		Password:           params.Password,
		Name:               params.Name,
		Role:               params.Role,
		MustChangePassword: true,
	})
}

func (a *Api) RegisterUser(ctx context.Context, newUser usersservice.NewUser) (*users.User, error) {
	return a.service.CreateUser(ctx, newUser)
}

func (a *Api) GetUser(ctx context.Context, userID int32) (*users.User, error) {
	return a.service.GetUser(ctx, userID)
}

func (a *Api) Setup(ctx context.Context, params NewUserParams) (*userauth.Session, error) {
	return a.service.Setup(ctx, usersservice.NewUser{
		Email:    params.Email,
		Password: params.Password,
		Name:     params.Name,
		Role:     users.RoleOwner,
	})
}

func (a *Api) Login(ctx context.Context, params LoginParams) (*userauth.Session, error) {
	return a.service.Login(ctx, usersservice.LoginParams(params))
}

func (a *Api) Refresh(ctx context.Context, refreshToken string) (*userauth.Session, error) {
	return a.service.Refresh(ctx, refreshToken)
}

func (a *Api) Logout(ctx context.Context, refreshToken string) error {
	return a.service.Logout(ctx, refreshToken)
}

func (a *Api) ChangePassword(ctx context.Context, params ChangePasswordParams) (*users.User, error) {
	return a.service.ChangePassword(ctx, usersservice.ChangePasswordParams(params))
}
