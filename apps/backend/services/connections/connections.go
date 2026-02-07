package connections

import (
	"database/sql"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	pkglogger "github.com/Uncensored-Developer/datalk/apps/backend/pkg/logger"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/api"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/connections"
)

type Connections struct {
	API *api.Api
}

func New(cfg config.Config, conn *sql.DB) Connections {
	logger := pkglogger.SetupLogger(cfg)
	connectionsService := connections.NewService(conn, logger)
	return Connections{
		API: api.New(logger, connectionsService),
	}
}
