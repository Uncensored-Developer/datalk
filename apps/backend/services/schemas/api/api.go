package api

import (
	"context"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	schematypes "github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/pkg/schemas"
)

//go:generate go tool with-modfile mockery --name Service --outpkg testing --output ./testing --filename generated__schemas_service_mocks.go
type Service interface {
	RefreshSchemaSnapshot(ctx context.Context, connectionID int32) error
	RetrieveRelevantSchemaContext(ctx context.Context, params schematypes.RetrieveRelevantSchemaContextParams) (*schematypes.RetrievedSchemaContext, error)
}

type Api struct {
	*base.Base
	service Service
}

func New(logger *slog.Logger, cfg config.Config, service Service) *Api {
	return &Api{
		Base:    base.NewBase("schemas", logger, cfg),
		service: service,
	}
}

func (a *Api) SnapshotDatabase(ctx context.Context, connectionID int32) error {
	return a.service.RefreshSchemaSnapshot(ctx, connectionID)
}

func (a *Api) RetrieveRelevantSchemaContext(ctx context.Context, params schematypes.RetrieveRelevantSchemaContextParams) (*schematypes.RetrievedSchemaContext, error) {
	return a.service.RetrieveRelevantSchemaContext(ctx, params)
}
