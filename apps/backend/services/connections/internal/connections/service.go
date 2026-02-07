package connections

import (
	"database/sql"
	"log/slog"

	"github.com/Uncensored-Developer/datalk/apps/backend/services/base"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage"
	"github.com/Uncensored-Developer/datalk/apps/backend/services/connections/internal/storage/db"
)

type Service struct {
	*base.Base

	storage storage.Storage
}

func NewService(conn *sql.DB, logger *slog.Logger) *Service {
	return &Service{
		Base:    base.NewBase("connections-core", logger),
		storage: db.NewStorage(conn),
	}
}
