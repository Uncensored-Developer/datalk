package schemas

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/embedding"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/db"
)

//go:generate go tool with-modfile mockery --name ConnectionGetter --outpkg testing --output ./testing --filename generated__connection_getter_mocks.go
type ConnectionGetter interface {
	GetConnection(ctx context.Context, ID int32) (connectiontypes.Connection, error)
}

//go:generate go tool with-modfile mockery --name IntrospectorFactory --outpkg testing --output ./testing --filename generated__introspector_factory_mocks.go
type IntrospectorFactory interface {
	ForConnection(ctx context.Context, connection connectiontypes.Connection) (introspector.Introspector, error)
}

type Service struct {
	*base.Base

	locker              distlock.DistributedLocker
	connectionGetter    ConnectionGetter
	storage             storage.Storage
	introspectorFactory IntrospectorFactory
	embeddingClient     embedding.Client
}

func NewService(conn *sql.DB, cfg config.Config, logger *slog.Logger, connectionGetter ConnectionGetter, introspectorFactory IntrospectorFactory, locker distlock.DistributedLocker, embeddingClient embedding.Client) *Service {
	return &Service{
		Base:                base.NewBase("schemas-core", logger, cfg),
		storage:             db.NewStorage(conn),
		connectionGetter:    connectionGetter,
		locker:              locker,
		introspectorFactory: introspectorFactory,
		embeddingClient:     embeddingClient,
	}
}
