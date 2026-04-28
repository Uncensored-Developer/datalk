package connections

import (
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/config"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/db"
)

type Service struct {
	*base.Base

	storage storage.Storage
}

func NewService(conn *sql.DB, cfg config.Config, logger *slog.Logger) *Service {
	return &Service{
		Base:    base.NewBase("connections-core", logger, cfg),
		storage: db.NewStorage(conn),
	}
}
