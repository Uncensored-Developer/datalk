package schemas

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/pkg/distlock"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	connectiontypes "github.com/Uncensored-Developer/datalk/apps/backend/services/connections/pkg/connections"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/schemas/introspector"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/schemas/internal/storage/db"
)

//go:generate go tool with-modfile mockery --name ConnectionGetter --outpkg testing --output ./testing --filename generated__connection_getter_mocks.go
type ConnectionGetter interface {
	GetConnection(ctx context.Context, connectionID int32) (connectiontypes.Connection, error)
}

//go:generate go tool with-modfile mockery --name IntrospectorFactory --outpkg testing --output ./testing --filename generated__introspector_factory_mocks.go
type IntrospectorFactory interface {
	ForConnection(ctx context.Context, connection connectiontypes.Connection) (introspector.Introspector, error)
}

type IntrospectorRegistry map[introspector.DBKind]introspector.Introspector

type Service struct {
	*base.Base

	locker              distlock.DistributedLocker
	connectionGetter    ConnectionGetter
	storage             storage.Storage
	introspectorFactory IntrospectorFactory
}

func NewService(conn *sql.DB, logger *slog.Logger, connectionGetter ConnectionGetter, locker distlock.DistributedLocker) *Service {
	return &Service{
		Base:             base.NewBase("schemas-core", logger),
		storage:          db.NewStorage(conn),
		connectionGetter: connectionGetter,
		locker:           locker,
	}
}
