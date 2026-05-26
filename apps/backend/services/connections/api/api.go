package api

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	connectionsservice "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
)

//go:generate go tool with-modfile mockery --name Service --outpkg testing --output ./testing --filename generated__connections_service_mocks.go
type Service interface {
	CreateConnection(ctx context.Context, params connectionsservice.NewConnection) (*connections.Connection, error)
	GetConnection(ctx context.Context, ID int32) (*connections.Connection, error)
	ListConnections(ctx context.Context, params connectionsservice.ListConnections) ([]*connections.Connection, error)
	UpdateConnection(ctx context.Context, params connectionsservice.UpdateConnection) (*connections.Connection, error)
	DeleteConnection(ctx context.Context, ID int32) error
	CreateAccess(ctx context.Context, params connectionsservice.NewAccess) (*connections.Access, error)
	GetAccess(ctx context.Context, userID int32, connectionID int32) (*connections.Access, error)
}

type Api struct {
	*base.Base
	service Service
}

func New(logger *slog.Logger, cfg config.Config, service Service) *Api {
	return &Api{
		Base:    base.NewBase("connections", logger, cfg),
		service: service,
	}
}

func (a *Api) CreateConnection(ctx context.Context, params NewConnectionParams) (*connections.Connection, error) {
	return a.service.CreateConnection(ctx, connectionsservice.NewConnection{
		Name:     params.Name,
		Database: params.Database,
		DSN:      params.DSN,
		UserID:   params.UserID,
		Metadata: params.Metadata,
	})
}

func (a *Api) GetConnection(ctx context.Context, connectionID int32) (*connections.Connection, error) {
	return a.service.GetConnection(ctx, connectionID)
}

func (a *Api) ListConnections(ctx context.Context, params ListConnectionsParams) ([]*connections.Connection, error) {
	return a.service.ListConnections(ctx, connectionsservice.ListConnections{
		UserID:  params.UserID,
		IsAdmin: params.IsAdmin,
	})
}

func (a *Api) UpdateConnection(ctx context.Context, params UpdateConnectionParams) (*connections.Connection, error) {
	return a.service.UpdateConnection(ctx, connectionsservice.UpdateConnection{
		ID:        params.ID,
		Name:      params.Name,
		Database:  params.Database,
		DSN:       params.DSN,
		IsEnabled: params.IsEnabled,
		Metadata:  params.Metadata,
	})
}

func (a *Api) DeleteConnection(ctx context.Context, connectionID int32) error {
	return a.service.DeleteConnection(ctx, connectionID)
}

func (a *Api) CreateAccess(ctx context.Context, params NewAccessParams) (*connections.Access, error) {
	return a.service.CreateAccess(ctx, connectionsservice.NewAccess{
		UserID:       params.UserID,
		ConnectionID: params.ConnectionID,
		CanQuery:     params.CanQuery,
		AllowWrites:  params.AllowWrites,
		CanManage:    params.CanManage,
	})
}

func (a *Api) GetAccess(ctx context.Context, userID int32, connectionID int32) (*connections.Access, error) {
	return a.service.GetAccess(ctx, userID, connectionID)
}
